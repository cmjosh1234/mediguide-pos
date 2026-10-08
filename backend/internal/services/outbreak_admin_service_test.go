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
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}, &models.Region{}, &models.HealthSubRegion{}, &models.District{}, &models.Disease{}, &models.Outbreak{}, &models.OutbreakUpdate{}, &models.OutbreakResource{}, &models.SituationReport{}, &models.SituationReportAttachment{}, &models.SituationReportAsset{}, &models.NotificationTopicJob{}, &models.ContentHub{}, &models.ContentHubOutbreak{}, &models.ContentDiseaseAssignment{}); err != nil {
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

func TestOutbreakValidationNamesTheFieldsToFix(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	fieldsOf := func(err error) map[string]string {
		t.Helper()
		var validation *OutbreakValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("expected a validation error, got %v", err)
		}
		fields := map[string]string{}
		for _, field := range validation.Fields {
			fields[field.Field] = field.Message
		}
		return fields
	}

	input := validOutbreakDraftInput(now)
	// Only Effective at (still "now") falls before the new start date.
	input.StartDate = ptr(now.Add(30 * time.Minute))
	input.LastUpdate = ptr(now.Add(50 * time.Minute))
	input.DataAsOf = ptr(now.Add(50 * time.Minute))
	_, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, input)
	if fields := fieldsOf(err); len(fields) != 2 || fields["effective_at"] != "Effective at can't be earlier than the start date." || fields["start_date"] == "" {
		t.Fatalf("date order error should name effective_at and start_date: %#v", fields)
	}

	input = validOutbreakDraftInput(now)
	input.GeographicArea, input.SourceReference = ptr(""), ptr("")
	item, err := service.CreateOutbreak(OutbreakActor{ID: uuid.New()}, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(OutbreakActor{ID: uuid.New()}, item.ID, "submit", TransitionInput{LockVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(OutbreakActor{ID: uuid.New()}, item.ID, "approve", TransitionInput{LockVersion: 2}); err != nil {
		t.Fatal(err)
	}
	_, err = service.TransitionOutbreak(OutbreakActor{ID: uuid.New()}, item.ID, "publish", TransitionInput{LockVersion: 3})
	if fields := fieldsOf(err); len(fields) != 2 || fields["geographic_area"] != "Geographic coverage is needed to publish." || fields["source_reference"] != "Source reference is needed to publish." {
		t.Fatalf("each missing field should get its own reason: %#v", fields)
	}
}

func TestApprovedOutbreakCorrectionOverwritesOnlyCoreFieldsOfLiveOutbreak(t *testing.T) {
	service := outbreakAdminTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author, corrector, reviewer := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(now))
	if err != nil {
		t.Fatal(err)
	}
	live := publishOutbreakForTest(t, service, item.ID, 1, "monitoring")
	update, err := service.CreateUpdate(author, item.ID, ChildContentInput{Title: ptr("Week 1 update")})
	if err != nil {
		t.Fatal(err)
	}

	correction, err := service.CorrectOutbreak(corrector, item.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Wrong source reference"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CorrectOutbreak(corrector, item.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Another fix"}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("second open correction allowed: %v", err)
	}
	if got, err := service.GetOutbreak(item.ID); err != nil || got.OpenCorrectionID == nil || *got.OpenCorrectionID != correction.ID {
		t.Fatalf("open correction not reported on the live outbreak: %#v %v", got, err)
	}
	if _, err := service.UpdateMetrics(corrector, correction.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{}, LockVersion: &correction.LockVersion}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("metrics changed on a correction: %v", err)
	}
	if _, err := service.CreateUpdate(corrector, correction.ID, ChildContentInput{Title: ptr("Misplaced update")}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("update added to a correction: %v", err)
	}

	metric := OutbreakMetric{Key: "confirmed_cases", Label: "Confirmed cases", Value: "20", Unit: "cases", AsOf: now, SourceReference: "WHO report 11", SortOrder: 1}
	staleMetrics := []OutbreakMetric{metric}
	edited, err := service.UpdateOutbreak(corrector, correction.ID, OutbreakInput{Title: ptr("Ebola virus disease response"), SourceReference: ptr("MOH-2026-02"), Metrics: &staleMetrics, LockVersion: &correction.LockVersion})
	if err != nil || len(edited.Metrics) != 0 {
		t.Fatalf("correction edit: %#v %v", edited, err)
	}
	if _, err := service.TransitionOutbreak(corrector, correction.ID, "submit", TransitionInput{LockVersion: edited.LockVersion}); err != nil {
		t.Fatal(err)
	}

	// Metrics keep moving on the live outbreak while the correction is in review.
	current, err := service.GetOutbreak(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	metric.Value = "25"
	if _, err := service.UpdateMetrics(author, item.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric}, LockVersion: &current.LockVersion}); err != nil {
		t.Fatal(err)
	}

	inReview := TransitionInput{LockVersion: edited.LockVersion + 1}
	if _, err := service.TransitionOutbreak(corrector, correction.ID, "approve", inReview); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("corrector approved their own correction: %v", err)
	}
	if _, err := service.TransitionOutbreak(reviewer, correction.ID, "publish", inReview); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("correction published on its own: %v", err)
	}
	applied, err := service.TransitionOutbreak(reviewer, correction.ID, "approve", inReview)
	if err != nil {
		t.Fatal(err)
	}
	if applied.ID != item.ID || applied.Title != "Ebola virus disease response" || applied.SourceReference != "MOH-2026-02" {
		t.Fatalf("correction not applied to the live outbreak: %#v", applied)
	}
	if applied.Status != "monitoring" || applied.PublishedAt == nil || !applied.PublishedAt.Equal(*live.PublishedAt) || applied.AuthorID == nil || *applied.AuthorID != author.ID || applied.OpenCorrectionID != nil {
		t.Fatalf("live outbreak lost its own properties: %#v", applied)
	}
	if len(applied.Metrics) != 1 || applied.Metrics[0].Value != "25" {
		t.Fatalf("metrics changed during review were overwritten: %#v", applied.Metrics)
	}
	if _, err := service.GetOutbreak(correction.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("applied correction still exists: %v", err)
	}
	if got, err := service.GetUpdate(item.ID, update.ID); err != nil || got.OutbreakID != item.ID {
		t.Fatalf("update no longer on the live outbreak: %#v %v", got, err)
	}

	history, err := service.ListAudit("outbreak", item.ID, PageInput{})
	if err != nil {
		t.Fatal(err)
	}
	var entry *OutbreakAuditDTO
	for i := range history.Items {
		if history.Items[i].Action == "outbreak.correction_applied" {
			entry = &history.Items[i]
		}
	}
	if entry == nil || entry.ActorID != reviewer.ID.String() || entry.Metadata["reason"] != "Wrong source reference" {
		t.Fatalf("correction not audited on the live outbreak: %#v", entry)
	}
	changes, _ := entry.Metadata["changes"].(map[string]any)
	if _, ok := changes["title"]; !ok || len(changes) != 2 || changes["source_reference"] == nil {
		t.Fatalf("audited changes should be exactly title and source_reference: %#v", changes)
	}

	// Once applied, the outbreak can be corrected again.
	if _, err := service.CorrectOutbreak(corrector, item.ID, TransitionInput{LockVersion: applied.LockVersion, Reason: "Later fix"}); err != nil {
		t.Fatal(err)
	}
}

func TestOutbreakCorrectionIsNotAppliedToAWithdrawnOutbreak(t *testing.T) {
	service := outbreakAdminTestService(t)
	author, corrector, reviewer := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	live := publishOutbreakForTest(t, service, item.ID, 1, "active")
	correction, err := service.CorrectOutbreak(corrector, item.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Fix the title"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(corrector, correction.ID, "submit", TransitionInput{LockVersion: correction.LockVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(author, item.ID, "withdraw", TransitionInput{LockVersion: live.LockVersion, Reason: "Duplicate entry"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionOutbreak(reviewer, correction.ID, "approve", TransitionInput{LockVersion: correction.LockVersion + 1}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("correction applied to a withdrawn outbreak: %v", err)
	}
	if got, err := service.GetOutbreak(correction.ID); err != nil || got.Status != "pending_review" {
		t.Fatalf("refused correction should be left in review: %#v %v", got, err)
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

func linkedReportForTest(t *testing.T, db *gorm.DB, outbreakID uuid.UUID) models.SituationReport {
	t.Helper()
	past := time.Now().UTC().Add(-time.Hour)
	report := models.SituationReport{OutbreakID: &outbreakID, Title: "Weekly situation report 1", Status: "published", PublicationDate: past, PublishedAt: &past, ApprovedAt: &past, LockVersion: 1}
	if err := db.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	return report
}

func TestDeletingOutbreakRemovesItsOwnContentAndKeepsLinkedContent(t *testing.T) {
	service := outbreakAdminTestService(t)
	public := OutbreakService{DB: service.DB}
	author, corrector := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	live := publishOutbreakForTest(t, service, item.ID, 1, "active")
	update, err := service.CreateUpdate(author, item.ID, ChildContentInput{Title: ptr("Week 1 update")})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&models.OutbreakResource{OutbreakID: item.ID, Title: "Case definition", Status: "draft", LockVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	correction, err := service.CorrectOutbreak(corrector, item.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Fix the title"})
	if err != nil {
		t.Fatal(err)
	}
	// A correction published on its own before corrections were applied in place.
	past := time.Now().UTC().Add(-time.Hour)
	legacy := models.Outbreak{Title: item.Title, Status: "active", PublishedAt: &past, SupersedesID: &item.ID, LockVersion: 1}
	if err := service.DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	report := linkedReportForTest(t, service.DB, item.ID)
	hub := models.ContentHub{Name: "Ebola", Slug: "ebola", Status: models.ContentHubStatusActive, LockVersion: 1}
	if err := service.DB.Create(&hub).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&models.ContentHubOutbreak{ContentHubID: hub.ID, OutbreakID: item.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := public.GetReport(report.ID); err != nil {
		t.Fatalf("report should be public under its live outbreak: %v", err)
	}

	current, err := service.GetOutbreak(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	reason := OutbreakDeleteInput{Reason: "Entered twice"}
	if _, err := service.DeleteOutbreak(author, item.ID, current.LockVersion, OutbreakDeleteInput{Reason: "  "}, true); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("deleted without a reason: %v", err)
	}
	if _, err := service.DeleteOutbreak(author, item.ID, current.LockVersion, reason, false); !errors.Is(err, ErrOutbreakForbidden) {
		t.Fatalf("live outbreak deleted without the withdraw permission: %v", err)
	}
	if _, err := service.DeleteOutbreak(author, item.ID, current.LockVersion-1, reason, true); !errors.Is(err, ErrOutbreakConflict) {
		t.Fatalf("deleted with a stale lock version: %v", err)
	}
	result, err := service.DeleteOutbreak(author, item.ID, current.LockVersion, reason, true)
	if err != nil {
		t.Fatal(err)
	}
	want := OutbreakDeleteResult{DeletedUpdates: 1, DeletedResources: 1, DeletedCorrections: 1, CancelledAlerts: 1, UnlinkedReports: 1, UnlinkedHubs: 1}
	if *result != want {
		t.Fatalf("delete result = %#v, want %#v", *result, want)
	}

	for name, id := range map[string]uuid.UUID{"outbreak": item.ID, "open correction": correction.ID} {
		if _, err := service.GetOutbreak(id); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("%s still exists: %v", name, err)
		}
	}
	if _, err := service.GetUpdate(item.ID, update.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("update still exists: %v", err)
	}
	if jobs := topicJobsForTest(t, service.DB); len(jobs) != 0 {
		t.Fatalf("queued alert for the deleted outbreak was kept: %#v", jobs)
	}
	if got, err := service.GetOutbreak(legacy.ID); err != nil || got.SupersedesID != nil {
		t.Fatalf("published correction should remain as an ordinary outbreak: %#v %v", got, err)
	}

	var kept models.SituationReport
	if err := service.DB.First(&kept, "id = ?", report.ID).Error; err != nil {
		t.Fatalf("linked report was deleted: %v", err)
	}
	if kept.OutbreakID != nil || !kept.StandaloneAllowed || kept.LockVersion != report.LockVersion+1 {
		t.Fatalf("linked report not unlinked: %#v", kept)
	}
	if _, err := public.GetReport(report.ID); err != nil {
		t.Fatalf("report that was public should stay public: %v", err)
	}
	var hubs, mappings int64
	service.DB.Model(&models.ContentHub{}).Where("id = ?", hub.ID).Count(&hubs)
	service.DB.Model(&models.ContentHubOutbreak{}).Where("outbreak_id = ?", item.ID).Count(&mappings)
	if hubs != 1 || mappings != 0 {
		t.Fatalf("hub should be kept and only unmapped: hubs=%d mappings=%d", hubs, mappings)
	}

	history, err := service.ListAudit("outbreak", item.ID, PageInput{})
	if err != nil {
		t.Fatal(err)
	}
	var audited bool
	for _, entry := range history.Items {
		audited = audited || (entry.Action == "outbreak.deleted" && entry.Metadata["reason"] == "Entered twice")
	}
	if !audited {
		t.Fatalf("deletion not audited with its reason: %#v", history.Items)
	}
}

func TestDeletingWithdrawnOutbreakKeepsItsReportsHidden(t *testing.T) {
	service := outbreakAdminTestService(t)
	public := OutbreakService{DB: service.DB}
	author := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	live := publishOutbreakForTest(t, service, item.ID, 1, "active")
	report := linkedReportForTest(t, service.DB, item.ID)
	withdrawn, err := service.TransitionOutbreak(author, item.ID, "withdraw", TransitionInput{LockVersion: live.LockVersion, Reason: "Not confirmed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := public.GetReport(report.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("report public under a withdrawn outbreak: %v", err)
	}
	if _, err := service.DeleteOutbreak(author, item.ID, withdrawn.LockVersion, OutbreakDeleteInput{Reason: "Not confirmed"}, true); err != nil {
		t.Fatal(err)
	}
	var kept models.SituationReport
	if err := service.DB.First(&kept, "id = ?", report.ID).Error; err != nil || kept.OutbreakID != nil || kept.StandaloneAllowed {
		t.Fatalf("report should be kept, unlinked and not standalone: %#v %v", kept, err)
	}
	if _, err := public.GetReport(report.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleting the outbreak published its hidden report: %v", err)
	}
	reports, err := public.ListReports(SituationReportQuery{})
	if err != nil || len(reports.Items) != 0 {
		t.Fatalf("hidden report listed publicly: %#v %v", reports, err)
	}
}

func TestDraftOutbreakCanBeDeletedWithoutWithdrawPermission(t *testing.T) {
	service := outbreakAdminTestService(t)
	author := OutbreakActor{ID: uuid.New()}
	item, err := service.CreateOutbreak(author, validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteOutbreak(author, item.ID, item.LockVersion, OutbreakDeleteInput{Reason: "Started by mistake"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetOutbreak(item.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("draft still exists: %v", err)
	}
}

func TestAuditHistoryNamesWhoActedAndWhatChanged(t *testing.T) {
	service := outbreakAdminTestService(t)
	author := models.User{Name: "Amina Okello", Email: "amina@example.test", Status: "active"}
	corrector := models.User{Name: "Brian Mugisha", Email: "brian@example.test", Status: "active"}
	ebola := models.Disease{Name: "Ebola virus disease", Slug: "ebola", NormalizedName: "ebola virus disease", Status: models.DiseaseStatusActive}
	marburg := models.Disease{Name: "Marburg virus disease", Slug: "marburg", NormalizedName: "marburg virus disease", Status: models.DiseaseStatusActive}
	for _, row := range []any{&author, &corrector, &ebola, &marburg} {
		if err := service.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	actor := OutbreakActor{ID: author.ID}
	item, err := service.CreateOutbreak(actor, validOutbreakDraftInput(time.Now().UTC().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	live := publishOutbreakForTest(t, service, item.ID, 1, "active")
	if _, err := service.TransitionOutbreak(actor, item.ID, "update_status", TransitionInput{LockVersion: live.LockVersion, OperationalStatus: "contained", Reason: "No new cases for 42 days"}); err != nil {
		t.Fatal(err)
	}
	applied := map[string]any{"reason": "Wrong disease", "corrected_by": corrector.ID.String(), "changes": map[string]any{"disease_id": map[string]any{"from": ebola.ID.String(), "to": marburg.ID.String()}}}
	if err := auditOutbreak(service.DB, actor, "outbreak.correction_applied", "outbreak", item.ID, applied); err != nil {
		t.Fatal(err)
	}
	// People who have since been removed are still named in the history.
	if err := service.DB.Delete(&author).Error; err != nil {
		t.Fatal(err)
	}

	history, err := service.ListAudit("outbreak", item.ID, PageInput{Page: 1, PerPage: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 2 || history.TotalItems <= 2 || history.TotalPages < 2 {
		t.Fatalf("history not paged: %d items, %d total, %d pages", len(history.Items), history.TotalItems, history.TotalPages)
	}
	entries := map[string]OutbreakAuditDTO{}
	for _, entry := range history.Items {
		entries[entry.Action] = entry
	}
	status, correction := entries["outbreak.update_status"], entries["outbreak.correction_applied"]
	if status.ActorName != "Amina Okello" || status.ActorEmail != "amina@example.test" {
		t.Fatalf("status change doesn't name who made it: %#v", status)
	}
	if status.Metadata["from_status"] != "active" || status.Metadata["to_status"] != "contained" || status.Metadata["reason"] != "No new cases for 42 days" {
		t.Fatalf("status change doesn't record what changed: %#v", status.Metadata)
	}
	want := map[string]string{corrector.ID.String(): "Brian Mugisha", ebola.ID.String(): "Ebola virus disease", marburg.ID.String(): "Marburg virus disease"}
	if correction.ActorName != "Amina Okello" || len(correction.Labels) != len(want) {
		t.Fatalf("applied correction labels = %#v (actor %q)", correction.Labels, correction.ActorName)
	}
	for id, name := range want {
		if correction.Labels[id] != name {
			t.Fatalf("label for %s = %q, want %q", id, correction.Labels[id], name)
		}
	}
}
