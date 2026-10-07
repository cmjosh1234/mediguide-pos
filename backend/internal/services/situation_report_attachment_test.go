package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// reportAttachmentTestService has the guideline library tables, so attachment
// checks and public visibility run against real published documents.
// addDocument returns the /public/guidelines/{id} link of a new document.
func reportAttachmentTestService(t *testing.T) (OutbreakAdminService, func(status string) string) {
	t.Helper()
	db := publicGuidelineTestDB(t)
	if err := db.AutoMigrate(&models.Outbreak{}, &models.SituationReport{}, &models.SituationReportAttachment{}, &models.SituationReportAsset{}, &models.NotificationTopicJob{}, &models.ContentHub{}, &models.ContentHubOutbreak{}, &models.ContentPillar{}, &models.ContentPillarItem{}); err != nil {
		t.Fatal(err)
	}
	kind := models.DocumentKind{Name: "Situation Report Attachment", Slug: "situation_report_attachment", Status: "active", PublishAsUploaded: true}
	if err := db.Create(&kind).Error; err != nil {
		t.Fatal(err)
	}
	addDocument := func(status string) string {
		document := models.GuidelineDocument{Title: "Week 12 report", DocumentKindID: &kind.ID}
		if err := db.Create(&document).Error; err != nil {
			t.Fatal(err)
		}
		version := models.GuidelineVersion{DocumentID: document.ID, Version: "1", Status: status, OriginalFileKey: "reports/week-12.pdf"}
		if err := db.Create(&version).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&document).Update("current_version_id", version.ID).Error; err != nil {
			t.Fatal(err)
		}
		return "/public/guidelines/" + document.ID.String()
	}
	return OutbreakAdminService{DB: db}, addDocument
}

func publishableReportInput(now time.Time) SituationReportInput {
	return SituationReportInput{Title: ptr("Week 12 situation report"), StandaloneAllowed: ptr(true), PublicationDate: &now, GeographicArea: ptr("Gulu"), SourceOrganization: ptr("Ministry of Health"), SourceReference: ptr("MOH-SITREP-12"), EffectiveAt: &now, LastVerifiedAt: &now}
}

func attachmentInput(url, kind string) SituationReportAttachmentInput {
	return SituationReportAttachmentInput{Title: ptr("Full report"), DocumentKind: &kind, URL: &url}
}

// submitAndApproveReport takes a draft report to approved and returns the
// lock version publishing needs.
func submitAndApproveReport(t *testing.T, service OutbreakAdminService, author OutbreakActor, id uuid.UUID, lock int) int {
	t.Helper()
	if _, err := service.TransitionReport(author, id, "submit", TransitionInput{LockVersion: lock}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, id, "approve", TransitionInput{LockVersion: lock + 1}); err != nil {
		t.Fatal(err)
	}
	return lock + 2
}

func TestReportAttachmentsComeFromTheLibraryAndPublishWithTheReport(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(addDocument("draft"), "situation_report_attachment")); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("unpublished document attached: %v", err)
	}
	document := addDocument("published")
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(document, "form")); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("document attached under the wrong document type: %v", err)
	}
	attachment, err := service.CreateReportAttachment(author, report.ID, attachmentInput(document, "situation_report_attachment"))
	if err != nil || attachment.SortOrder != 1 || attachment.LockVersion != 1 {
		t.Fatalf("attach a published library document: %#v %v", attachment, err)
	}
	stale := attachment.LockVersion
	edited, err := service.UpdateReportAttachment(author, report.ID, attachment.ID, SituationReportAttachmentInput{Title: ptr("Full week 12 report"), LockVersion: &stale})
	if err != nil || edited.Title != "Full week 12 report" || edited.LockVersion != 2 {
		t.Fatalf("edit attachment: %#v %v", edited, err)
	}
	if _, err := service.UpdateReportAttachment(author, report.ID, attachment.ID, SituationReportAttachmentInput{Title: ptr("Lost edit"), LockVersion: &stale}); !errors.Is(err, ErrOutbreakConflict) {
		t.Fatalf("stale attachment edit accepted: %v", err)
	}

	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	published, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(document, "situation_report_attachment")); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("attachment added to a published report: %v", err)
	}
	if err := service.DeleteReportAttachment(author, report.ID, attachment.ID, edited.LockVersion); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("attachment removed from a published report: %v", err)
	}

	public, err := (OutbreakService{DB: service.DB}).GetReport(report.ID)
	if err != nil {
		t.Fatal(err)
	}
	documentID := strings.TrimPrefix(document, "/public/guidelines/")
	if len(public.Attachments) != 1 || public.Attachments[0].URL != document || public.Attachments[0].ResourceType != "guideline" || public.Attachments[0].Title != "Full week 12 report" {
		t.Fatalf("public report attachments: %#v", public.Attachments)
	}
	if public.ReportAssetURL != "/api/public/guidelines/"+documentID+"/original/download" {
		t.Fatalf("older apps should open the first attachment as the full report, got %q", public.ReportAssetURL)
	}

	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Wrong week"})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := service.ListReportAttachments(correction.ID)
	if err != nil || len(copied) != 1 || copied[0].ID == attachment.ID || copied[0].URL != document || copied[0].Title != "Full week 12 report" {
		t.Fatalf("correction should start with the report's attachments: %#v %v", copied, err)
	}
	if err := service.DeleteReportAttachment(author, correction.ID, copied[0].ID, copied[0].LockVersion); err != nil {
		t.Fatalf("remove an attachment from the correction draft: %v", err)
	}
	if original, err := service.ListReportAttachments(report.ID); err != nil || len(original) != 1 {
		t.Fatalf("published report lost its attachment: %#v %v", original, err)
	}
}

func TestReportNeedsAnAttachmentToPublish(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	document := addDocument("published")
	attachment, err := service.CreateReportAttachment(author, report.ID, attachmentInput(document, "situation_report_attachment"))
	if err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)

	// The attached document is unpublished after the report was approved.
	documentID := strings.TrimPrefix(document, "/public/guidelines/")
	if err := service.DB.Model(&models.GuidelineVersion{}).Where("document_id = ?", documentID).Update("status", "draft").Error; err != nil {
		t.Fatal(err)
	}
	publisher := OutbreakActor{ID: uuid.New()}
	_, err = service.TransitionReport(publisher, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err == nil || !strings.Contains(err.Error(), `"Full report" no longer points to a published document`) {
		t.Fatalf("expected the stale attachment to be named, got %v", err)
	}

	if err := service.DeleteReportAttachment(author, report.ID, attachment.ID, attachment.LockVersion); err != nil {
		t.Fatal(err)
	}
	_, err = service.TransitionReport(publisher, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err == nil || err.Error() != "Attach at least one document before publishing." {
		t.Fatalf("expected a report without attachments to be refused, got %v", err)
	}
}

func TestPublicReportHidesAttachmentsThatAreNoLongerPublic(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	document := addDocument("published")
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(document, "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	if _, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock}); err != nil {
		t.Fatal(err)
	}

	documentID := strings.TrimPrefix(document, "/public/guidelines/")
	if err := service.DB.Model(&models.GuidelineVersion{}).Where("document_id = ?", documentID).Update("status", "draft").Error; err != nil {
		t.Fatal(err)
	}
	page, err := (OutbreakService{DB: service.DB}).ListReports(SituationReportQuery{Page: PageInput{Page: 1, PerPage: 10}})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list reports: %#v %v", page, err)
	}
	if got := page.Items[0]; len(got.Attachments) != 0 || got.ReportAssetURL != "" {
		t.Fatalf("an unpublished document is still offered: %#v %q", got.Attachments, got.ReportAssetURL)
	}
}

// validationFields maps each field a validation error names to its reason.
func validationFields(t *testing.T, err error) map[string]string {
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

func TestReportValidationExplainsWhatToFix(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}

	_, err := service.CreateReport(author, SituationReportInput{Title: ptr("Week 12"), PublicationDate: &now})
	if fields := validationFields(t, err); fields["outbreak_id"] == "" || fields["standalone_allowed"] == "" {
		t.Fatalf("a report with no outbreak should point at the outbreak and standalone fields: %#v", fields)
	}

	input := publishableReportInput(now)
	input.GeographicArea, input.SourceReference, input.EffectiveAt = ptr(""), ptr(""), nil
	report, err := service.CreateReport(author, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(addDocument("published"), "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	if _, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "approve", TransitionInput{LockVersion: lock}); err == nil || err.Error() != "This report is already approved and is waiting to be published." {
		t.Fatalf("approving twice should be refused, got %v", err)
	}
	_, err = service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err == nil || err.Error() != "Before publishing, fill in and save: Geographic area, Source reference, Effective at." {
		t.Fatalf("expected the missing fields to be listed, got %v", err)
	}
	if fields := validationFields(t, err); len(fields) != 3 || fields["effective_at"] != "Effective at is needed to publish." {
		t.Fatalf("each missing field should be named: %#v", fields)
	}
}

func TestCorrectingAnOlderReportKeepsItsUploadedPDF(t *testing.T) {
	service, _ := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	asset := models.SituationReportAsset{SituationReportID: report.ID, StorageKey: "situation-reports/older.pdf", FileName: "older.pdf", ContentType: "application/pdf", SizeBytes: 9, ChecksumSHA256: "older"}
	if err := service.DB.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Model(&models.SituationReport{}).Where("id = ?", report.ID).Update("report_asset_id", asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	published, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err != nil {
		t.Fatalf("an older report with its own PDF should publish without attachments: %v", err)
	}

	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Wrong district"})
	if err != nil {
		t.Fatal(err)
	}
	// Approving the correction applies it to the published report, which keeps
	// serving the PDF uploaded before attachments.
	submitAndApproveReport(t, service, author, correction.ID, correction.LockVersion)
	store := &fakePublicStore{objects: map[string][]byte{}}
	target, err := (OutbreakService{DB: service.DB, Store: store}).PresignReportAsset(context.Background(), report.ID)
	if err != nil || !strings.HasSuffix(target.String(), "situation-reports/older.pdf") {
		t.Fatalf("the corrected report lost its uploaded PDF: %v %v", target, err)
	}
}

func TestPublishedReportsJoinTheirOutbreakHub(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	outbreak := models.Outbreak{Title: "Ebola response", Status: "active", PublishedAt: &now, LastUpdate: now, LockVersion: 1}
	hub := models.ContentHub{Name: "Ebola response hub", Slug: "ebola-response", Status: "active", LockVersion: 1}
	for _, row := range []any{&outbreak, &hub} {
		if err := service.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	pillar := models.ContentPillar{HubID: hub.ID, Name: "Situation reports", Slug: situationReportsPillar}
	if err := service.DB.Create(&pillar).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&models.ContentHubOutbreak{ContentHubID: hub.ID, OutbreakID: outbreak.ID}).Error; err != nil {
		t.Fatal(err)
	}
	earlierID := uuid.New()
	if err := service.DB.Create(&models.ContentPillarItem{PillarID: pillar.ID, ContentType: models.ContentDiseaseSituationReport, ContentID: &earlierID, SortOrder: 10, Status: "active", LockVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	items := func() []models.ContentPillarItem {
		t.Helper()
		var rows []models.ContentPillarItem
		if err := service.DB.Where("pillar_id = ?", pillar.ID).Order("sort_order").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		return rows
	}
	publish := func(id uuid.UUID, lock int) *SituationReportAdminDTO {
		t.Helper()
		if _, err := service.CreateReportAttachment(author, id, attachmentInput(addDocument("published"), "situation_report_attachment")); err != nil {
			t.Fatal(err)
		}
		lock = submitAndApproveReport(t, service, author, id, lock)
		published, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, id, "publish", TransitionInput{LockVersion: lock})
		if err != nil {
			t.Fatal(err)
		}
		return published
	}

	input := publishableReportInput(now)
	input.OutbreakID, input.StandaloneAllowed = &outbreak.ID, ptr(false)
	report, err := service.CreateReport(author, input)
	if err != nil {
		t.Fatal(err)
	}
	published := publish(report.ID, report.LockVersion)
	listed := items()
	if len(listed) != 2 || *listed[0].ContentID != report.ID || listed[0].SortOrder != 0 || listed[0].LabelOverride != "" {
		t.Fatalf("a newly published report should open the hub's Situation reports section: %#v", listed)
	}

	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Wrong week"})
	if err != nil {
		t.Fatal(err)
	}
	submitAndApproveReport(t, service, author, correction.ID, correction.LockVersion)
	listed = items()
	if len(listed) != 2 || *listed[0].ContentID != report.ID || *listed[1].ContentID != earlierID {
		t.Fatalf("an applied correction keeps the report's single card: %#v", listed)
	}

	standalone, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	publish(standalone.ID, standalone.LockVersion)
	if got := len(items()); got != 2 {
		t.Fatalf("a standalone report has no hub, but %d items are listed", got)
	}
}

func TestReportIsApprovedBySomeoneOtherThanItsSubmitter(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author, submitter, reviewer := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(addDocument("published"), "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	submitted, err := service.TransitionReport(submitter, report.ID, "submit", TransitionInput{LockVersion: report.LockVersion})
	if err != nil || submitted.SubmittedBy == nil || *submitted.SubmittedBy != submitter.ID || submitted.SubmittedAt == nil {
		t.Fatalf("the submitter should be recorded: %#v %v", submitted, err)
	}
	inReview := TransitionInput{LockVersion: submitted.LockVersion}
	if _, err := service.TransitionReport(submitter, report.ID, "approve", inReview); err == nil || err.Error() != "You submitted this report for review, so a different reviewer has to approve it." {
		t.Fatalf("the submitter approved their own submission: %v", err)
	}
	if _, err := service.TransitionReport(author, report.ID, "approve", inReview); err == nil || err.Error() != "You created this report, so a different reviewer has to approve it." {
		t.Fatalf("the author approved their own report: %v", err)
	}
	approved, err := service.TransitionReport(reviewer, report.ID, "approve", inReview)
	if err != nil || approved.ApprovedBy == nil || *approved.ApprovedBy != reviewer.ID {
		t.Fatalf("an independent reviewer should approve: %#v %v", approved, err)
	}
	// Anyone allowed to publish can, including the submitter.
	published, err := service.TransitionReport(submitter, report.ID, "publish", TransitionInput{LockVersion: approved.LockVersion})
	if err != nil {
		t.Fatal(err)
	}

	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Wrong week"})
	if err != nil || correction.SubmittedBy != nil || correction.SubmittedAt != nil {
		t.Fatalf("a correction starts unsubmitted: %#v %v", correction, err)
	}
}

func TestApprovedReportCorrectionReplacesThePublishedReport(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author, corrector, reviewer := OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}, OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	firstDocument := addDocument("published")
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(firstDocument, "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	live, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err != nil {
		t.Fatal(err)
	}

	correction, err := service.CorrectReport(corrector, report.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Wrong week"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CorrectReport(corrector, report.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Another fix"}); err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("a second open correction was allowed: %v", err)
	}
	if got, err := service.GetReport(report.ID); err != nil || got.OpenCorrectionID == nil || *got.OpenCorrectionID != correction.ID {
		t.Fatalf("the open correction should be shown on the published report: %#v %v", got, err)
	}

	// A correction can't move the report to another outbreak.
	otherOutbreak := uuid.New()
	if _, err := service.UpdateReport(corrector, correction.ID, SituationReportInput{OutbreakID: &otherOutbreak, LockVersion: &correction.LockVersion}); err == nil || !strings.Contains(err.Error(), "keeps the report's related outbreak") {
		t.Fatalf("a correction moved the report: %v", err)
	}
	edited, err := service.UpdateReport(corrector, correction.ID, SituationReportInput{Title: ptr("Week 12 situation report (corrected)"), LockVersion: &correction.LockVersion})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := service.ListReportAttachments(correction.ID)
	if err != nil || len(copied) != 1 {
		t.Fatalf("the correction should start with the report's attachment: %#v %v", copied, err)
	}
	if err := service.DeleteReportAttachment(corrector, correction.ID, copied[0].ID, copied[0].LockVersion); err != nil {
		t.Fatal(err)
	}
	secondDocument := addDocument("published")
	if _, err := service.CreateReportAttachment(corrector, correction.ID, attachmentInput(secondDocument, "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}

	submitted, err := service.TransitionReport(corrector, correction.ID, "submit", TransitionInput{LockVersion: edited.LockVersion})
	if err != nil {
		t.Fatal(err)
	}
	inReview := TransitionInput{LockVersion: submitted.LockVersion}
	if _, err := service.TransitionReport(reviewer, correction.ID, "publish", inReview); err == nil || !strings.Contains(err.Error(), "isn't published on its own") {
		t.Fatalf("a correction was published on its own: %v", err)
	}
	if _, err := service.TransitionReport(corrector, correction.ID, "approve", inReview); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("the corrector approved their own correction: %v", err)
	}
	applied, err := service.TransitionReport(reviewer, correction.ID, "approve", inReview)
	if err != nil {
		t.Fatal(err)
	}
	if applied.ID != report.ID || applied.Title != "Week 12 situation report (corrected)" || applied.Status != "published" || applied.OpenCorrectionID != nil {
		t.Fatalf("the correction should be applied to the published report: %#v", applied)
	}
	if applied.PublishedAt == nil || !applied.PublishedAt.Equal(*live.PublishedAt) || applied.AuthorID == nil || *applied.AuthorID != author.ID {
		t.Fatalf("the published report lost its own history: %#v", applied)
	}
	attachments, err := service.ListReportAttachments(report.ID)
	if err != nil || len(attachments) != 1 || attachments[0].URL != secondDocument {
		t.Fatalf("the report should now have the correction's attachments: %#v %v", attachments, err)
	}
	if _, err := service.GetReport(correction.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("the applied correction still exists: %v", err)
	}

	history, err := service.ListAudit("situation_report", report.ID, PageInput{})
	if err != nil {
		t.Fatal(err)
	}
	var entry *OutbreakAuditDTO
	for i := range history.Items {
		if history.Items[i].Action == "situation_report.correction_applied" {
			entry = &history.Items[i]
		}
	}
	if entry == nil {
		t.Fatal("the correction should be audited on the report")
	}
	changes, _ := entry.Metadata["changes"].(map[string]any)
	if entry.Metadata["reason"] != "Wrong week" || changes["title"] == nil || changes["attachments"] == nil || len(changes) != 2 {
		t.Fatalf("the correction should be audited on the report with its changes: %#v", entry)
	}

	// Once applied, the report can be corrected again.
	if _, err := service.CorrectReport(corrector, report.ID, TransitionInput{LockVersion: applied.LockVersion, Reason: "Later fix"}); err != nil {
		t.Fatal(err)
	}
}

func TestReportCorrectionIsNotAppliedToAWithdrawnReport(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(addDocument("published"), "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	lock := submitAndApproveReport(t, service, author, report.ID, report.LockVersion)
	live, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err != nil {
		t.Fatal(err)
	}
	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: live.LockVersion, Reason: "Wrong week"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionReport(author, correction.ID, "submit", TransitionInput{LockVersion: correction.LockVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionReport(author, report.ID, "withdraw", TransitionInput{LockVersion: live.LockVersion, Reason: "Duplicate"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, correction.ID, "approve", TransitionInput{LockVersion: correction.LockVersion + 1}); err == nil || !strings.Contains(err.Error(), "report it corrects is withdrawn") {
		t.Fatalf("a correction was applied to a withdrawn report: %v", err)
	}
}

func TestReportMetricsSaveOnTheirOwnUntilPublished(t *testing.T) {
	service, addDocument := reportAttachmentTestService(t)
	now := time.Now().UTC().Add(-time.Hour)
	author := OutbreakActor{ID: uuid.New()}
	report, err := service.CreateReport(author, publishableReportInput(now))
	if err != nil {
		t.Fatal(err)
	}
	metric := OutbreakMetric{Key: "confirmed_cases", Label: "Confirmed cases", Value: "20", Unit: "cases", AsOf: now, SourceReference: "WHO report 11", SortOrder: 1}

	lock := report.LockVersion
	saved, err := service.UpdateReportMetrics(author, report.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric}, LockVersion: &lock})
	if err != nil || len(saved.Metrics) != 1 || saved.LockVersion != lock+1 {
		t.Fatalf("save a metric on a draft: %#v %v", saved, err)
	}
	if _, err := service.UpdateReportMetrics(author, report.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{}, LockVersion: &lock}); !errors.Is(err, ErrOutbreakConflict) {
		t.Fatalf("a stale metrics save was accepted: %v", err)
	}
	if _, err := service.UpdateReportMetrics(author, report.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric, metric}, LockVersion: &saved.LockVersion}); !errors.Is(err, ErrOutbreakInvalid) {
		t.Fatalf("duplicate metric keys were accepted: %v", err)
	}

	if _, err := service.CreateReportAttachment(author, report.ID, attachmentInput(addDocument("published"), "situation_report_attachment")); err != nil {
		t.Fatal(err)
	}
	lock = submitAndApproveReport(t, service, author, report.ID, saved.LockVersion)
	published, err := service.TransitionReport(OutbreakActor{ID: uuid.New()}, report.ID, "publish", TransitionInput{LockVersion: lock})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateReportMetrics(author, report.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{}, LockVersion: &published.LockVersion}); err == nil || !strings.Contains(err.Error(), "locked once the report is published") {
		t.Fatalf("a published report's metrics changed outside a correction: %v", err)
	}

	correction, err := service.CorrectReport(author, report.ID, TransitionInput{LockVersion: published.LockVersion, Reason: "Late cases"})
	if err != nil || len(correction.Metrics) != 1 {
		t.Fatalf("a correction should start with the report's metrics: %#v %v", correction, err)
	}
	metric.Value = "25"
	if fixed, err := service.UpdateReportMetrics(author, correction.ID, OutbreakMetricsInput{Metrics: []OutbreakMetric{metric}, LockVersion: &correction.LockVersion}); err != nil || fixed.Metrics[0].Value != "25" {
		t.Fatalf("a correction's metrics should be editable: %#v %v", fixed, err)
	}
}
