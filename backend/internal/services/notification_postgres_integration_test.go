package services

import (
	"os"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func notificationPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("MEDIGUIDE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MEDIGUIDE_TEST_DATABASE_URL is not configured")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func TestNotificationAudiencePostgresFilteringAndOutboxClaimIsolation(t *testing.T) {
	database := notificationPostgresTestDB(t)
	country, language := "Uganda", "English"
	user := models.User{Name: "Postgres audience", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", IsActive: true, Status: "active", Country: &country, PreferredLanguage: &language}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&models.User{}, "id = ?", user.ID) })
	device := models.FirebaseDevice{UserID: user.ID, InstallationID: uuid.NewString(), RegistrationToken: uuid.NewString(), Platform: "android", NotificationsEnabled: true, LastSeenAt: time.Now().UTC()}
	if err := database.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	estimate, err := (NotificationService{DB: database}).EstimateAudience(NotificationAudienceDefinition{UserIDs: []string{user.ID.String()}, Countries: []string{"uganda"}, Languages: []string{"ENGLISH"}, Platforms: []string{"android"}})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.EligibleUsers != 1 || estimate.ActiveDevices != 1 {
		t.Fatalf("unexpected PostgreSQL audience result: %#v", estimate)
	}

	campaign := models.NotificationCampaign{CampaignVariablesJSON: datatypes.JSON([]byte(`{}`)), DispatchSnapshotJSON: datatypes.JSON([]byte(`{}`)), Name: "Claim isolation", Type: "update", Status: "queued", RenderedTitle: "Title", RenderedBody: "Body", ActionSnapshotJSON: datatypes.JSON(`{"type":"none","parameters":{}}`), AudienceDefinitionJSON: datatypes.JSON(`{"all_eligible":false}`), RequestedChannelsJSON: datatypes.JSON(`["in-app"]`), ChannelsJSON: datatypes.JSON(`["in-app"]`), Priority: "normal", Timezone: "UTC", IdempotencyKey: uuid.NewString(), LockVersion: 1}
	if err := database.Create(&campaign).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(&models.NotificationCampaign{}, "id = ?", campaign.ID) })
	recipient := models.NotificationCampaignRecipient{CampaignID: campaign.ID, UserID: user.ID, Status: "pending"}
	if err := database.Create(&recipient).Error; err != nil {
		t.Fatal(err)
	}
	job := models.NotificationOutboxJob{CampaignID: campaign.ID, RecipientID: recipient.ID, UserID: user.ID, Channel: "in-app", Status: "pending", IdempotencyKey: uuid.NewString(), PayloadJSON: datatypes.JSON(`{}`), MaxAttempts: 1, NextAttemptAt: time.Now().UTC()}
	if err := database.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	first, err := (NotificationOutboxService{DB: database, WorkerID: "worker-one", BatchSize: 1}).ClaimBatch()
	if err != nil {
		t.Fatal(err)
	}
	second, err := (NotificationOutboxService{DB: database, WorkerID: "worker-two", BatchSize: 1}).ClaimBatch()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("job was not isolated across workers: first=%d second=%d", len(first), len(second))
	}
}

func TestNotificationDigestPostgresClaimsAreIsolated(t *testing.T) {
	database := notificationPostgresTestDB(t)
	user := models.User{Name: "Digest PostgreSQL", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", IsActive: true, Verified: true, Status: "active"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	device := models.FirebaseDevice{UserID: user.ID, InstallationID: uuid.NewString(), RegistrationToken: uuid.NewString(), Platform: "android", NotificationsEnabled: true, LastSeenAt: time.Now().UTC()}
	if err := database.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	settings := models.NotificationPreferenceSettings{UserID: user.ID, RoutineDigest: true, DigestTime: "18:00", QuietHoursTimezone: "UTC", PushEnabled: true, InAppEnabled: true, PreferredLanguage: "en", InterestsJSON: datatypes.JSON([]byte(`{}`))}
	if err := database.Create(&settings).Error; err != nil {
		t.Fatal(err)
	}
	service := NotificationService{DB: database}
	// Select this fixture alone so shared integration fixtures are not notified.
	action := datatypes.JSON([]byte(`{"type":"internal_route","route":"/notifications","parameters":{}}`))
	audience := datatypes.JSON([]byte(`{"all_eligible":false,"user_ids":["` + user.ID.String() + `"]}`))
	campaign := models.NotificationCampaign{Name: "Digest", Type: "update", Status: "queued", RenderedTitle: "Title", RenderedBody: "Body", Priority: "normal", Timezone: "UTC", IdempotencyKey: uuid.NewString(), LockVersion: 1,
		CampaignVariablesJSON: datatypes.JSON([]byte(`{}`)), DispatchSnapshotJSON: datatypes.JSON([]byte(`{}`)), ActionSnapshotJSON: action, AudienceDefinitionJSON: audience,
		RequestedChannelsJSON: datatypes.JSON([]byte(`["push","in-app"]`)), ChannelsJSON: datatypes.JSON([]byte(`["push","in-app"]`)), AudienceCountriesJSON: datatypes.JSON([]byte(`[]`)), AudienceRolesJSON: datatypes.JSON([]byte(`[]`))}
	if err := database.Create(&campaign).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.Unscoped().Delete(&models.NotificationCampaign{}, "id = ?", campaign.ID)
		database.Unscoped().Delete(&models.NotificationDigest{}, "firebase_device_id = ?", device.ID)
		database.Unscoped().Delete(&models.FirebaseDevice{}, "id = ?", device.ID)
		database.Unscoped().Delete(&models.NotificationPreferenceSettings{}, "user_id = ?", user.ID)
		database.Unscoped().Delete(&models.User{}, "id = ?", user.ID)
	})
	if _, err := service.prepareCampaignDispatch(database, &campaign); err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&models.NotificationOutboxJob{}).Where("campaign_id = ?", campaign.ID).Update("status", "pending").Error; err != nil {
		t.Fatal(err)
	}
	// The ordinary claim path must never steal digest members.
	regular, err := (NotificationOutboxService{DB: database, WorkerID: "regular"}).ClaimBatch()
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range regular {
		if job.DigestID != nil {
			t.Fatal("ordinary worker claimed a digest member")
		}
	}
	mock := &firebaseMockMessenger{results: []firebaseMockResult{{id: "projects/test-project/messages/pg-digest"}}}
	now := time.Now().UTC().Add(25 * time.Hour)
	worker := NotificationOutboxService{DB: database, WorkerID: "digest-one", Firebase: &FirebaseService{DB: database, Project: "test-project", Messenger: mock}, Now: func() time.Time { return now }}
	first, err := worker.ProcessDigests(t.Context())
	if err != nil || first.Accepted != 1 {
		t.Fatalf("first digest: %#v %v", first, err)
	}
	worker.WorkerID = "digest-two"
	second, err := worker.ProcessDigests(t.Context())
	if err != nil || second.Claimed != 0 || len(mock.messages) != 1 {
		t.Fatalf("duplicate digest: %#v %v", second, err)
	}
	// Exercise the provider-event row lock and migration's delivered event constraint.
	count, err := worker.ImportFirebaseDeliveryEvent("test-project", FirebaseDeliveryImportEvent{MessageID: "pg-digest", SDKPlatform: "ANDROID", EventTimestamp: now, Event: "MESSAGE_DELIVERED"})
	if err != nil || count != 1 {
		t.Fatalf("provider import: %d %v", count, err)
	}
}
