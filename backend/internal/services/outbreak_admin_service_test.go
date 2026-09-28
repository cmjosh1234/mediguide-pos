package services

import (
	"errors"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func outbreakAdminTestService(t *testing.T) OutbreakAdminService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}, &models.Region{}, &models.HealthSubRegion{}, &models.District{}, &models.Disease{}, &models.Outbreak{}, &models.OutbreakUpdate{}, &models.OutbreakResource{}, &models.SituationReport{}, &models.SituationReportAsset{}, &models.NotificationTopicJob{}); err != nil {
		t.Fatal(err)
	}
	seedDocumentKinds(t, db)
	return OutbreakAdminService{DB: db}
}

func TestOutbreakTypedMetricsRejectDuplicatesAndInvalidFreshness(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	metric := OutbreakMetric{Key: "confirmed_cases", Label: "Confirmed cases", Value: "20", NumericValue: ptr(20.0), Unit: "cases", AsOf: now, SourceReference: "WHO report 11", SortOrder: 1}
	input := validOutbreakDraftInput(now)
	input.Metrics = &[]OutbreakMetric{metric, metric}
	if _, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, input); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("duplicate metric key accepted: %v", err)
	}
	input.Metrics = &[]OutbreakMetric{metric}
	input.LastVerifiedAt = ptr(now.Add(-time.Hour))
	input.DataAsOf = &now
	if _, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, input); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("verification older than data accepted: %v", err)
	}
}

func TestOutbreakMetricsCanBeUpdatedRegardlessOfStatus(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New()}
	reviewer := OutbreakActor{ID: uuid.New()}
	publisher := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionOutbreak(author, item.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if item, err = service.TransitionOutbreak(reviewer, item.ID, "approve", TransitionInput{LockVersion: 2}); err != nil {
		t.Fatal(err)
	}
	if item, err = service.TransitionOutbreak(publisher, item.ID, "publish", TransitionInput{LockVersion: 3, OperationalStatus: "active"}); err != nil {
		t.Fatal(err)
	}

	// The main update path is still immutable once published.
	if _, err := service.UpdateOutbreak(author, item.ID, OutbreakInput{Title: ptr("New title"), LockVersion: &item.LockVersion}); !errors.Is(err, ErrOutbreakImmutable) {
		t.Fatalf("expected published outbreak to remain immutable for general edits: %v", err)
	}

	metric := OutbreakMetric{Key: "confirmed_cases", Label: "Confirmed cases", Value: "34", NumericValue: ptr(34.0), Unit: "cases", AsOf: now, SourceReference: "WHO report 12", SortOrder: 1}
	updated, err := service.UpdateMetrics(publisher, item.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric}, LockVersion: &item.LockVersion})
	if err != nil {
		t.Fatalf("metrics update should be allowed on a published outbreak: %v", err)
	}
	if updated.Status != "active" || len(updated.Metrics) != 1 || updated.Metrics[0].Key != "confirmed_cases" || updated.Metrics[0].Value != "34" {
		t.Fatalf("metrics were not applied without changing status: %#v", updated)
	}

	// The stale lock version from before the metrics update is now rejected.
	if _, err := service.UpdateMetrics(publisher, item.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric}, LockVersion: &item.LockVersion}); !errors.Is(err, ErrOutbreakConflict) {
		t.Fatalf("expected a stale lock version to conflict: %v", err)
	}

	// Invalid metrics are still rejected.
	if _, err := service.UpdateMetrics(publisher, item.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric, metric}, LockVersion: &updated.LockVersion}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("duplicate metric key accepted: %v", err)
	}
}

func TestOutbreakGeographyAndResourceSafetyValidation(t *testing.T) {
	service := outbreakAdminTestService(t)
	regionOne := models.Region{Name: "Central"}
	regionTwo := models.Region{Name: "Northern"}
	if err := service.DB.Create(&regionOne).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&regionTwo).Error; err != nil {
		t.Fatal(err)
	}
	district := models.District{Name: "Kampala", RegionID: regionOne.ID, HealthSubRegionID: uuid.New()}
	if err := service.DB.Create(&district).Error; err != nil {
		t.Fatal(err)
	}
	input := validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour))
	input.RegionID = &regionTwo.ID
	input.DistrictID = &district.ID
	if _, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, input); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("cross-region district accepted: %v", err)
	}

	parent := models.Outbreak{Title: "Response", Status: "draft", LastUpdate: time.Now(), VisualTone: "warning"}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	title, kind, hostile := "Unsafe resource", "approved_external_url", "javascript:alert(1)"
	if _, err := service.CreateResource(OutbreakActor{ID: uuid.New()}, parent.ID, ChildContentInput{Title: &title, ResourceType: &kind, URL: &hostile}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("hostile resource URL accepted: %v", err)
	}
	redirect := "https://health.go.ug/open?redirect=https://evil.example"
	if _, err := service.CreateResource(OutbreakActor{ID: uuid.New()}, parent.ID, ChildContentInput{Title: &title, ResourceType: &kind, URL: &redirect}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("redirect-style resource URL accepted: %v", err)
	}
	managed, rawStorageKey := "managed_document", "situation-reports/private/report.pdf"
	if _, err := service.CreateResource(OutbreakActor{ID: uuid.New()}, parent.ID, ChildContentInput{Title: &title, ResourceType: &managed, AssetURL: &rawStorageKey}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("raw managed storage key accepted: %v", err)
	}
}

func TestOutbreakAdministrationFiltersAreTypedAndApplied(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	old := now.Add(-72 * time.Hour)
	region := models.Region{Name: "Central"}
	if err := service.DB.Create(&region).Error; err != nil {
		t.Fatal(err)
	}
	ebolaDiseaseID, malariaDiseaseID := uuid.New(), uuid.New()
	first := models.Outbreak{Base: models.Base{UpdatedAt: now}, Title: "Ebola response", DiseaseID: &ebolaDiseaseID, GeographicArea: "Kampala", RegionID: &region.ID, Status: "active", VisualTone: "critical", LastUpdate: now, EffectiveAt: &now, LockVersion: 1}
	second := models.Outbreak{Base: models.Base{UpdatedAt: old}, Title: "Malaria update", DiseaseID: &malariaDiseaseID, GeographicArea: "Gulu", Status: "monitoring", VisualTone: "info", LastUpdate: old, EffectiveAt: &old, LockVersion: 1}
	if err := service.DB.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	page, err := service.ListOutbreaks(OutbreakAdminQuery{Page: PageInput{Page: 1, PerPage: 20}, Disease: ebolaDiseaseID.String(), RegionID: region.ID.String(), VisualTone: "critical", UpdatedFrom: now.Add(-time.Hour).Format(time.RFC3339), Sort: "effective_at", Order: "desc"})
	if err != nil || page.TotalItems != 1 || page.Items[0].ID != first.ID {
		t.Fatalf("typed filters returned %#v err=%v", page, err)
	}
	if _, err := service.ListOutbreaks(OutbreakAdminQuery{UpdatedFrom: "yesterday"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("malformed date accepted: %v", err)
	}
}

func TestSituationReportHighlightsAreBoundedAndUnique(t *testing.T) {
	service := outbreakAdminTestService(t)
	title := "Weekly report"
	highlights := []string{"New cases investigated", " new cases investigated "}
	if _, err := service.CreateReport(OutbreakActor{ID: uuid.New()}, SituationReportInput{Title: &title, StandaloneAllowed: ptr(true), KeyHighlights: &highlights}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("duplicate highlight accepted: %v", err)
	}
}

func ptr[T any](value T) *T { return &value }

func validOutbreakDraftInput(now time.Time) OutbreakInput {
	metrics := []OutbreakMetric{}
	return OutbreakInput{Title: ptr("Ebola response"), DiseaseID: ptr(uuid.New()), GeographicArea: ptr("Uganda"), Summary: ptr("Public health response"), LastUpdate: &now, VisualTone: ptr("critical"), SourceOrganization: ptr("Ministry of Health"), SourceReference: ptr("MOH-2026-01"), EffectiveAt: &now, DataAsOf: &now, LastVerifiedAt: &now, Metrics: &metrics}
}

func TestOutbreakLifecycleRequiresIndependentReviewerAndOptimisticLock(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New(), IP: "127.0.0.1"}
	reviewer := OutbreakActor{ID: uuid.New(), IP: "127.0.0.2"}
	publisher := OutbreakActor{ID: uuid.New(), IP: "127.0.0.3"}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil || item.Status != "draft" || item.LockVersion != 1 {
		t.Fatalf("create: %#v %v", item, err)
	}
	if _, err = service.TransitionOutbreak(author, item.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionOutbreak(author, item.ID, "approve", TransitionInput{LockVersion: 2}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("self approval allowed: %v", err)
	}
	approved, err := service.TransitionOutbreak(reviewer, item.ID, "approve", TransitionInput{LockVersion: 2})
	if err != nil || approved.ApprovedBy == nil || *approved.ApprovedBy != reviewer.ID {
		t.Fatalf("approve: %#v %v", approved, err)
	}
	if _, err = service.TransitionOutbreak(reviewer, item.ID, "publish", TransitionInput{LockVersion: 2}); !errors.Is(err, ErrOutbreakConflict) {
		t.Fatalf("stale lock accepted: %v", err)
	}
	if _, err = service.TransitionOutbreak(reviewer, item.ID, "publish", TransitionInput{LockVersion: 3, OperationalStatus: "active"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("critical outbreak reviewer also published: %v", err)
	}
	published, err := service.TransitionOutbreak(publisher, item.ID, "publish", TransitionInput{LockVersion: 3, OperationalStatus: "active"})
	if err != nil || published.Status != "active" || published.PublishedAt == nil {
		t.Fatalf("publish: %#v %v", published, err)
	}
	if _, err = service.UpdateOutbreak(author, item.ID, OutbreakInput{Title: ptr("silent edit"), LockVersion: &published.LockVersion}); !errors.Is(err, ErrOutbreakImmutable) {
		t.Fatalf("published edit allowed: %v", err)
	}
	corrected, err := service.CorrectOutbreak(author, item.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Correct case definition"})
	if err != nil || corrected.Status != "draft" || corrected.SupersedesID == nil || *corrected.SupersedesID != item.ID {
		t.Fatalf("correction: %#v %v", corrected, err)
	}
	var auditCount int64
	if err := service.DB.Model(&models.AuditLog{}).Where("entity_type = ?", "outbreak").Count(&auditCount).Error; err != nil || auditCount < 5 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}
}

func TestOutbreakReviewCommentIsValidatedAndAudited(t *testing.T) {
	service := outbreakAdminTestService(t)
	actor := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(actor, validOutbreakDraftInput(time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.AddReviewComment(actor, "outbreak", item.ID, "  Confirm source date before publishing.  "); err != nil {
		t.Fatal(err)
	}
	if err := service.AddReviewComment(actor, "outbreak", item.ID, "  "); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("blank comment accepted: %v", err)
	}
	history, err := service.ListAudit("outbreak", item.ID, PageInput{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range history.Items {
		if event.Action == "outbreak.review_comment" && event.Metadata["comment"] == "Confirm source date before publishing." {
			found = true
		}
	}
	if !found {
		t.Fatalf("review comment missing from audit history: %#v", history.Items)
	}
}

func TestOutbreakChildrenRequireOwnPublication(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	parent := models.Outbreak{Title: "Published parent", Status: "active", PublishedAt: &now, LastUpdate: now, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	draft := models.OutbreakUpdate{OutbreakID: parent.ID, Title: "Draft child", Status: "draft", LockVersion: 1}
	published := models.OutbreakUpdate{OutbreakID: parent.ID, Title: "Published child", Status: "published", PublishedAt: &now, LockVersion: 1}
	if err := service.DB.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&published).Error; err != nil {
		t.Fatal(err)
	}
	page, err := (OutbreakService{DB: service.DB}).Updates(parent.ID, PageInput{})
	if err != nil || page.TotalItems != 1 || page.Items[0].ID != published.ID {
		t.Fatalf("public children: %#v %v", page, err)
	}
}

func TestSituationReportPublicationValidatesStandaloneAndSource(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC()
	author := OutbreakActor{ID: uuid.New()}
	if _, err := service.CreateReport(author, SituationReportInput{Title: ptr("Unscoped")}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("unapproved standalone report accepted: %v", err)
	}
	report, err := service.CreateReport(author, SituationReportInput{Title: ptr("Standalone"), StandaloneAllowed: ptr(true), PublicationDate: &now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionReport(author, report.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	reviewer := OutbreakActor{ID: uuid.New()}
	if _, err = service.TransitionReport(reviewer, report.ID, "approve", TransitionInput{LockVersion: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.TransitionReport(reviewer, report.ID, "publish", TransitionInput{LockVersion: 3}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("report without source/asset published: %v", err)
	}
}

func TestOutbreakResourceLinksPublishedDocumentOfChosenKind(t *testing.T) {
	service := outbreakAdminTestService(t)
	for _, statement := range []string{
		`CREATE TABLE guideline_documents (id text PRIMARY KEY, current_version_id text, document_kind_id text, deleted_at datetime)`,
		`CREATE TABLE guideline_versions (id text PRIMARY KEY, document_id text, status text, deleted_at datetime)`,
	} {
		if err := service.DB.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	var sop models.DocumentKind
	if err := service.DB.First(&sop, "slug = ?", "sop").Error; err != nil {
		t.Fatal(err)
	}
	addDocument := func(status string) string {
		documentID, versionID := uuid.NewString(), uuid.NewString()
		service.DB.Exec(`INSERT INTO guideline_documents (id, current_version_id, document_kind_id) VALUES (?, ?, ?)`, documentID, versionID, sop.ID.String())
		service.DB.Exec(`INSERT INTO guideline_versions (id, document_id, status) VALUES (?, ?, ?)`, versionID, documentID, status)
		return "/public/guidelines/" + documentID
	}
	parent := models.Outbreak{Title: "Parent", Status: "draft", LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	create := func(url, kind string) (*OutbreakResourceAdminDTO, error) {
		title, resourceType := "IPC SOP", "guideline"
		return service.CreateResource(OutbreakActor{ID: uuid.New()}, parent.ID, ChildContentInput{Title: &title, ResourceType: &resourceType, DocumentKind: &kind, URL: &url})
	}

	published := addDocument("published")
	created, err := create(published, "sop")
	if err != nil || created.DocumentKind != "sop" {
		t.Fatalf("expected a published SOP to be linked as an SOP: %#v %v", created, err)
	}
	if _, err := create(published, "form"); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected a kind mismatch to be rejected, got %v", err)
	}
	if _, err := create(addDocument("draft"), "sop"); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected an unpublished document to be rejected, got %v", err)
	}
}

func TestOutbreakChildPublishExplainsUnpublishedParent(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now()
	author, reviewer := uuid.New(), uuid.New()
	parent := models.Outbreak{Title: "Draft parent", Status: "draft", LastUpdate: now, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.OutbreakUpdate{OutbreakID: parent.ID, Title: "Update", Status: "pending_review", AuthorID: &author, ApprovedBy: &reviewer, ApprovedAt: &now, LockVersion: 1}
	if err := service.DB.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	_, err := service.TransitionUpdate(OutbreakActor{ID: reviewer}, parent.ID, child.ID, "publish", TransitionInput{LockVersion: 1})
	want := "This update can't be published yet because its outbreak isn't published (the outbreak is draft). Publish the outbreak first, then publish this update."
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestEditingPublishedResourceReplacesItAfterReview(t *testing.T) {
	service := outbreakAdminTestService(t)
	published := time.Now().Add(-time.Hour)
	author, reviewer, editor := uuid.New(), uuid.New(), uuid.New()
	parent := models.Outbreak{Title: "Live outbreak", Status: "active", PublishedAt: &published, LastUpdate: published, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	original := models.OutbreakResource{OutbreakID: parent.ID, Title: "Hub", ResourceType: "internal_route", DocumentKind: "other", URL: "/outbreak-hub", Status: "published", PublishedAt: &published, AuthorID: &author, LockVersion: 1}
	if err := service.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	newTitle := "Outbreak hub"
	edit := ResourceCorrectionInput{LockVersion: 1, Reason: "Clearer title", Changes: &ChildContentInput{Title: &newTitle}}

	if _, err := service.EditPublishedResource(OutbreakActor{ID: editor}, parent.ID, original.ID, ResourceCorrectionInput{LockVersion: 1, Changes: edit.Changes}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected a missing reason to be rejected, got %v", err)
	}
	pending, err := service.EditPublishedResource(OutbreakActor{ID: editor}, parent.ID, original.ID, edit)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != "pending_review" || pending.Title != newTitle || pending.SupersedesID == nil || *pending.SupersedesID != original.ID {
		t.Fatalf("expected an edited correction in review: %#v", pending)
	}
	if _, err := service.EditPublishedResource(OutbreakActor{ID: editor}, parent.ID, original.ID, edit); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected a second pending edit to be rejected, got %v", err)
	}

	approved, err := service.TransitionResource(OutbreakActor{ID: reviewer}, parent.ID, pending.ID, "approve", TransitionInput{LockVersion: pending.LockVersion})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionResource(OutbreakActor{ID: reviewer}, parent.ID, pending.ID, "publish", TransitionInput{LockVersion: approved.LockVersion}); err != nil {
		t.Fatal(err)
	}
	var retired models.OutbreakResource
	if err := service.DB.First(&retired, "id = ?", original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retired.Status != "withdrawn" || retired.WithdrawalReason != "superseded by approved correction: Clearer title" {
		t.Fatalf("expected the original to be retired with the edit reason: %s %q", retired.Status, retired.WithdrawalReason)
	}
	page, err := (OutbreakService{DB: service.DB}).Resources(parent.ID, PageInput{})
	if err != nil || page.TotalItems != 1 || page.Items[0].Title != newTitle {
		t.Fatalf("expected only the corrected resource to be public: %#v %v", page, err)
	}
}

func TestEditingResourceInReviewClearsApproval(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now()
	reviewer := uuid.New()
	parent := models.Outbreak{Title: "Parent", Status: "draft", LastUpdate: now, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	row := models.OutbreakResource{OutbreakID: parent.ID, Title: "Hub", ResourceType: "internal_route", DocumentKind: "other", URL: "/outbreak-hub", Status: "pending_review", ApprovedBy: &reviewer, ApprovedAt: &now, LockVersion: 1}
	if err := service.DB.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	title, lock := "Edited hub", 1
	updated, err := service.UpdateResource(OutbreakActor{ID: uuid.New()}, parent.ID, row.ID, ChildContentInput{Title: &title, LockVersion: &lock})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "draft" || updated.ApprovedAt != nil || updated.Title != title {
		t.Fatalf("expected edited resource back in draft without approval: %#v", updated)
	}
}

func TestEditingPublishedUpdateReplacesItAfterReview(t *testing.T) {
	service := outbreakAdminTestService(t)
	published := time.Now().Add(-time.Hour)
	author, reviewer, editor := uuid.New(), uuid.New(), uuid.New()
	parent := models.Outbreak{Title: "Live outbreak", Status: "active", PublishedAt: &published, LastUpdate: published, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	original := models.OutbreakUpdate{OutbreakID: parent.ID, Title: "Cases rising", Summary: "Twelve cases.", Status: "published", PublishedAt: &published, AuthorID: &author, LockVersion: 1}
	if err := service.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	newSummary := "Fourteen cases."
	edit := ResourceCorrectionInput{LockVersion: 1, Reason: "Late reports", Changes: &ChildContentInput{Summary: &newSummary}}

	if _, err := service.EditPublishedUpdate(OutbreakActor{ID: editor}, parent.ID, original.ID, ResourceCorrectionInput{LockVersion: 1, Changes: edit.Changes}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected a missing reason to be rejected, got %v", err)
	}
	pending, err := service.EditPublishedUpdate(OutbreakActor{ID: editor}, parent.ID, original.ID, edit)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != "pending_review" || pending.Summary != newSummary || pending.Title != original.Title || pending.SupersedesID == nil || *pending.SupersedesID != original.ID {
		t.Fatalf("expected an edited correction in review: %#v", pending)
	}
	if _, err := service.EditPublishedUpdate(OutbreakActor{ID: editor}, parent.ID, original.ID, edit); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("expected a second pending edit to be rejected, got %v", err)
	}

	approved, err := service.TransitionUpdate(OutbreakActor{ID: reviewer}, parent.ID, pending.ID, "approve", TransitionInput{LockVersion: pending.LockVersion})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionUpdate(OutbreakActor{ID: reviewer}, parent.ID, pending.ID, "publish", TransitionInput{LockVersion: approved.LockVersion}); err != nil {
		t.Fatal(err)
	}
	var retired models.OutbreakUpdate
	if err := service.DB.First(&retired, "id = ?", original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retired.Status != "withdrawn" || retired.WithdrawalReason != "superseded by approved correction: Late reports" {
		t.Fatalf("expected the original to be retired with the edit reason: %s %q", retired.Status, retired.WithdrawalReason)
	}
	page, err := (OutbreakService{DB: service.DB}).Updates(parent.ID, PageInput{})
	if err != nil || page.TotalItems != 1 || page.Items[0].Summary != newSummary {
		t.Fatalf("expected only the corrected update to be public: %#v %v", page, err)
	}
}

func TestEditingUpdateInReviewClearsApproval(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now()
	reviewer := uuid.New()
	parent := models.Outbreak{Title: "Parent", Status: "draft", LastUpdate: now, LockVersion: 1}
	if err := service.DB.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	row := models.OutbreakUpdate{OutbreakID: parent.ID, Title: "Update", Status: "pending_review", ApprovedBy: &reviewer, ApprovedAt: &now, LockVersion: 1}
	if err := service.DB.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	title, lock := "Edited update", 1
	updated, err := service.UpdateUpdate(OutbreakActor{ID: uuid.New()}, parent.ID, row.ID, ChildContentInput{Title: &title, LockVersion: &lock})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "draft" || updated.ApprovedAt != nil || updated.Title != title {
		t.Fatalf("expected edited update back in draft without approval: %#v", updated)
	}
}

func TestResourcesCanBeAddedUntilOutbreakIsClosedOrWithdrawn(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now()
	title, resourceType, route := "Hub", "internal_route", "/outbreak-hub"
	input := ChildContentInput{Title: &title, ResourceType: &resourceType, URL: &route}
	for status, allowed := range map[string]bool{"draft": true, "active": true, "published": true, "closed": false, "withdrawn": false} {
		parent := models.Outbreak{Title: status, Status: status, LastUpdate: now, LockVersion: 1}
		if err := service.DB.Create(&parent).Error; err != nil {
			t.Fatal(err)
		}
		_, err := service.CreateResource(OutbreakActor{ID: uuid.New()}, parent.ID, input)
		if allowed && err != nil {
			t.Fatalf("%s outbreak: expected a resource to be added, got %v", status, err)
		}
		if !allowed && (err == nil || err.Error() != "Resources can't be added to an outbreak that is "+status+".") {
			t.Fatalf("%s outbreak: expected the resource to be refused, got %v", status, err)
		}
	}
}
