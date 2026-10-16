package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mediguide/internal/models"

	"github.com/google/uuid"
)

func TestGuidelineWithdrawalRetainsHistoryAndCanRepublish(t *testing.T) {
	db := publicGuidelineTestDB(t)
	seedDocumentKinds(t, db)
	link := models.DocumentKind{Name: "Link", Slug: "link", Status: "active", PublishAsLink: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	admin := GuidelineService{DB: db}
	doc, err := admin.CreateDocument(CreateGuidelineInput{Title: "Malaria guidance", DocumentKindID: &link.ID})
	if err != nil {
		t.Fatal(err)
	}
	version, err := admin.CreateVersion(doc.ID, CreateVersionInput{Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	if _, err := admin.SetVersionLink(version.ID, actor, "https://www.who.int/malaria", ""); err != nil {
		t.Fatal(err)
	}
	if err := admin.PublishVersion(version.ID, actor); err != nil {
		t.Fatal(err)
	}
	public := PublicGuidelineService{DB: db}
	ctx := context.Background()
	if _, err := public.Get(ctx, doc.ID); err != nil {
		t.Fatal(err)
	}
	chunk := models.GuidelineChunk{DocumentID: doc.ID, VersionID: version.ID, Title: "Malaria guidance", Content: "Malaria assessment", ReviewStatus: "approved"}
	if err := db.Create(&chunk).Error; err != nil {
		t.Fatal(err)
	}
	if results, err := (SearchService{DB: db}).SearchPublishedGuidelineContext(ctx, doc.ID, "malaria", 5); err != nil || len(results) != 1 {
		t.Fatalf("published search = %v, %v", results, err)
	}
	if err := admin.WithdrawDocument(doc.ID, actor, WithdrawGuidelineInput{Reason: "  Source under review  ", CurrentVersionID: version.ID}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	retained, err := admin.GetDocument(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retained.CurrentVersionID != nil || retained.WithdrawnVersionID == nil || *retained.WithdrawnVersionID != version.ID || retained.WithdrawnAt == nil || retained.WithdrawnBy == nil || *retained.WithdrawnBy != actor || retained.WithdrawalReason != "Source under review" {
		t.Fatalf("withdrawal metadata = %#v", retained)
	}
	if len(retained.Versions) != 1 || retained.Versions[0].Status != "published" || retained.Versions[0].ExternalURL == "" {
		t.Fatal("approved version was changed or removed")
	}
	if _, err := admin.SetVersionLink(version.ID, actor, "https://www.who.int/changed", ""); !errors.Is(err, ErrPublishedVersionImmutable) {
		t.Fatalf("withdrawn approved version became editable: %v", err)
	}
	if _, err := public.Get(ctx, doc.ID); !errors.Is(err, ErrPublicGuidelineNotFound) {
		t.Fatalf("withdrawn detail exposed: %v", err)
	}
	if _, err := public.Manifest(ctx, doc.ID); !errors.Is(err, ErrPublicGuidelineNotFound) {
		t.Fatalf("withdrawn manifest exposed: %v", err)
	}
	if _, err := public.Original(ctx, doc.ID); !errors.Is(err, ErrPublicGuidelineNotFound) {
		t.Fatalf("withdrawn original exposed: %v", err)
	}
	if _, err := public.OfflinePackage(ctx, doc.ID); !errors.Is(err, ErrPublicGuidelineNotFound) {
		t.Fatalf("withdrawn offline package exposed: %v", err)
	}
	if results, err := (SearchService{DB: db}).SearchPublishedGuidelineContext(ctx, doc.ID, "malaria", 5); err != nil || len(results) != 0 {
		t.Fatalf("withdrawn search = %v, %v", results, err)
	}
	if citations := (RAGService{DB: db}).enrichCitations([]Citation{{ChunkID: chunk.ID.String()}}); len(citations) != 0 {
		t.Fatalf("withdrawn worker citation exposed: %v", citations)
	}
	page, err := public.List(ctx, PublicGuidelineFilter{})
	if err != nil || page.TotalItems != 0 {
		t.Fatalf("withdrawn library = %#v, %v", page, err)
	}
	var audit models.AuditLog
	if err := db.Where("action = ? AND entity_id = ?", "guideline.document.withdrawn", doc.ID.String()).First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.ActorID != actor.String() || !strings.Contains(audit.MetadataJSON, "Source under review") {
		t.Fatalf("withdrawal audit = %#v", audit)
	}
	if err := admin.WithdrawDocument(doc.ID, actor, WithdrawGuidelineInput{Reason: "Again", CurrentVersionID: version.ID}, ""); !errors.Is(err, ErrGuidelinePublicationConflict) {
		t.Fatalf("duplicate withdrawal: %v", err)
	}
	if err := admin.PublishVersion(version.ID, actor); err != nil {
		t.Fatalf("republish: %v", err)
	}
	restored, err := admin.GetDocument(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.CurrentVersionID == nil || *restored.CurrentVersionID != version.ID || restored.WithdrawnAt != nil || restored.WithdrawnVersionID != nil || restored.WithdrawalReason != "" {
		t.Fatalf("republication metadata = %#v", restored)
	}
	if _, err := public.Get(ctx, doc.ID); err != nil {
		t.Fatalf("republished detail: %v", err)
	}
	var republished models.AuditLog
	if err := db.Where("action = ? AND entity_id = ?", "guideline.document.republished", doc.ID.String()).First(&republished).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&audit, "id = ?", audit.ID).Error; err != nil {
		t.Fatal("withdrawal audit lost on republish")
	}
}

func TestGuidelineWithdrawalRejectsInvalidReasonsAndStalePublication(t *testing.T) {
	db := publicGuidelineTestDB(t)
	doc := models.GuidelineDocument{Title: "Private draft"}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatal(err)
	}
	service := GuidelineService{DB: db}
	for _, reason := range []string{"", "   ", strings.Repeat("x", 2001)} {
		if err := service.WithdrawDocument(doc.ID, uuid.New(), WithdrawGuidelineInput{Reason: reason}, ""); !errors.Is(err, ErrGuidelineWithdrawalReason) {
			t.Fatalf("reason accepted: %v", err)
		}
	}
	otherVersion := uuid.New()
	for _, current := range []*uuid.UUID{nil, &otherVersion} {
		if err := db.Model(&doc).Update("current_version_id", current).Error; err != nil {
			t.Fatal(err)
		}
		if err := service.WithdrawDocument(doc.ID, uuid.New(), WithdrawGuidelineInput{Reason: "Review", CurrentVersionID: uuid.New()}, ""); !errors.Is(err, ErrGuidelinePublicationConflict) {
			t.Fatalf("stale/draft withdrawal: %v", err)
		}
	}
}
