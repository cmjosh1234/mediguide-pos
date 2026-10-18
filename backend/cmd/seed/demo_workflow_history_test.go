package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"mediguide/internal/clinicaltools"
	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDemoWorkflowHistoryIsLinkedIdempotentAndInert(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&models.Subcounty{}, &models.Parish{}, &models.User{},
		&models.AccountActionToken{}, &models.AccountEmailDelivery{},
		&models.Calculator{}, &models.CalculatorVersion{}, &models.CalculatorCitation{},
		&models.ClinicalProtocol{}, &models.ProtocolRun{}, &models.RetrievalLog{},
		&models.GuidelineVersion{}, &models.GuidelineAsset{}, &models.GuidelineUpload{},
		&models.SyncPackage{}, &models.DeviceRegistration{}, &models.DeviceSyncLog{},
		&models.FirebaseDevice{}, &models.Notification{}, &models.NotificationCampaign{},
		&models.NotificationCampaignRecipient{}, &models.NotificationDigest{},
		&models.NotificationOutboxJob{}, &models.NotificationDeliveryAttempt{},
		&models.NotificationDelivery{}, &models.NotificationDeliveryEvent{},
		&models.NotificationTopicJob{}, &models.SituationReportAttachment{},
	); err != nil {
		t.Fatal(err)
	}
	adminID, clinicianID := uuid.New(), uuid.New()
	subcounty := models.Subcounty{Base: models.Base{ID: uuid.New()}, Name: "Existing subcounty"}
	if err := database.Create(&subcounty).Error; err != nil {
		t.Fatal(err)
	}
	source := models.GuidelineVersion{Base: models.Base{ID: demoID("guideline-version", "reader-test-form-1.0")}, DocumentID: demoID("guideline", "reader-test-form"), Version: "1.0", Status: "published", OriginalFileKey: "demo/form.pdf"}
	if err := database.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	asset := models.GuidelineAsset{VersionID: source.ID, Type: models.GuidelineAssetOriginalPDF, MIMEType: "application/pdf", StorageKey: source.OriginalFileKey, SizeBytes: 54321, Checksum: demoWorkflowChecksum([]byte("source-pdf"))}
	if err := database.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	store := &readerSeedStore{objects: map[string][]byte{}}
	for range 2 {
		if err := seedDemoWorkflowHistory(context.Background(), database, store, adminID, clinicianID); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range demoWorkflowHistoryRows(adminID, clinicianID, subcounty.ID, source, asset, "", 0, "") {
		var count int64
		if err := database.Table(row.table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		want := int64(1)
		if count != want {
			t.Fatalf("%s has %d rows after reseeding, want %d", row.table, count, want)
		}
	}
	var token models.AccountActionToken
	if err := database.First(&token).Error; err != nil {
		t.Fatal(err)
	}
	if token.ConsumedAt == nil {
		t.Fatal("seed created a usable account action token")
	}
	var email models.AccountEmailDelivery
	if err := database.First(&email).Error; err != nil {
		t.Fatal(err)
	}
	if email.Status != "cancelled" || email.EncryptedToken != "" || email.TokenID != token.ID || email.UserID != clinicianID {
		t.Fatalf("email fixture is not an inert, linked history: %#v", email)
	}
	for table, status := range map[string]string{"notification_outbox_jobs": "accepted", "notification_topic_jobs": "failed", "notification_digests": "accepted", "notification_campaigns": "completed"} {
		var count int64
		if err := database.Table(table).Where("status <> ?", status).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s contains dispatchable work", table)
		}
	}
	var device models.FirebaseDevice
	if err := database.First(&device).Error; err != nil {
		t.Fatal(err)
	}
	if device.NotificationsEnabled {
		t.Fatal("synthetic device opted into notifications")
	}
	var upload models.GuidelineUpload
	if err := database.First(&upload).Error; err != nil {
		t.Fatal(err)
	}
	if upload.VersionID != source.ID || upload.ObjectKey != asset.StorageKey || upload.SizeBytes != asset.SizeBytes || upload.Checksum != asset.Checksum || upload.Status != "completed" {
		t.Fatalf("upload source metadata drifted: %#v", upload)
	}
	var tool models.CalculatorVersion
	if err := database.First(&tool).Error; err != nil {
		t.Fatal(err)
	}
	if tool.Status != "draft" {
		t.Fatal("synthetic calculator was published")
	}
	if _, validation := clinicaltools.ParseAndValidate(tool.DefinitionJSON); !validation.Valid {
		t.Fatalf("invalid sample tool: %#v", validation)
	}
	var pkg models.SyncPackage
	if err := database.First(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	payload := store.objects[pkg.FileKey]
	if int64(len(payload)) != pkg.SizeBytes || demoWorkflowChecksum(payload) != pkg.Checksum {
		t.Fatal("sync package does not match stored size and checksum")
	}
	archive, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != "manifest.json" {
		t.Fatal("missing package manifest")
	}
	reader, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	manifest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Demo bool `json:"demo"`
	}
	if err := json.Unmarshal(manifest, &metadata); err != nil || !metadata.Demo {
		t.Fatal("package is not marked as synthetic")
	}
	var attachment models.SituationReportAttachment
	if err := database.First(&attachment).Error; err != nil {
		t.Fatal(err)
	}
	if attachment.URL != "/public/guidelines/"+source.DocumentID.String() {
		t.Fatal("attachment does not link to its seeded library document")
	}
}
