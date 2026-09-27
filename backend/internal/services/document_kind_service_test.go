package services

import (
	"errors"
	"fmt"
	"testing"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// seedDocumentKinds creates the kinds seeded by migrations 00056 and 00057 so
// tests that create documents through the services have kinds to reference.
func seedDocumentKinds(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&models.DocumentKind{}); err != nil {
		t.Fatal(err)
	}
	for index, slug := range []string{
		"guideline", "sop", "case_definition", "ipc_protocol", "laboratory_protocol",
		"surveillance_protocol", "contact_tracing_guide", "treatment_protocol", "referral_protocol",
		"training_material", "checklist", "communication_material", "form", "policy",
		"situation_report_attachment", "other",
	} {
		kind := models.DocumentKind{Name: slug, Slug: slug, Status: "active", SortOrder: (index + 1) * 10}
		if err := db.Create(&kind).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func documentKindTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.New())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.DocumentKind{}, &models.GuidelineCategory{}, &models.GuidelineDocument{}, &models.GuidelineVersion{}, &models.Outbreak{}, &models.OutbreakResource{}, &models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDocumentKindCRUD(t *testing.T) {
	service := GuidelineContentService{DB: documentKindTestDB(t)}
	order := 20
	form, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Reporting Form"), Description: stringPtr("  HMIS forms  "), SortOrder: &order})
	if err != nil {
		t.Fatal(err)
	}
	if form.Slug != "reporting_form" || form.Status != "active" || form.Description == nil || *form.Description != "HMIS forms" {
		t.Fatalf("created kind = %#v", form)
	}
	if _, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Reporting form")}); !errors.Is(err, ErrGuidelineContentConflict) {
		t.Fatalf("duplicate slug err = %v", err)
	}
	if _, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("  ")}); !errors.Is(err, ErrGuidelineContentInvalid) {
		t.Fatalf("blank name err = %v", err)
	}
	if _, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Bad"), Slug: stringPtr("Bad Slug")}); !errors.Is(err, ErrGuidelineContentInvalid) {
		t.Fatalf("invalid slug err = %v", err)
	}

	renamed, err := service.SaveDocumentKind(&form.ID, DocumentKindInput{Name: stringPtr("Form"), Slug: stringPtr("reporting_form")})
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Form" || renamed.Slug != "reporting_form" || renamed.SortOrder != 20 {
		t.Fatalf("rename changed more than the name: %#v", renamed)
	}
	if _, err := service.SaveDocumentKind(&form.ID, DocumentKindInput{Slug: stringPtr("form")}); !errors.Is(err, ErrDocumentKindSlugImmutable) {
		t.Fatalf("slug change err = %v", err)
	}

	guideline, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Guideline"), Status: stringPtr("inactive")})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := service.ListDocumentKinds(false, GuidelineContentQuery{})
	if err != nil || reader.TotalItems != 1 || reader.Items[0].ID != form.ID {
		t.Fatalf("reader kinds = %#v err=%v", reader, err)
	}
	editor, err := service.ListDocumentKinds(true, GuidelineContentQuery{})
	if err != nil || editor.TotalItems != 2 {
		t.Fatalf("editor kinds = %#v err=%v", editor, err)
	}

	// Form is the only active kind, so it can be neither deactivated nor deleted.
	if _, err := service.SaveDocumentKind(&form.ID, DocumentKindInput{Status: stringPtr("inactive")}); !errors.Is(err, ErrDocumentKindRequired) {
		t.Fatalf("deactivate last active err = %v", err)
	}
	if err := service.DeleteDocumentKind(form.ID); !errors.Is(err, ErrDocumentKindRequired) {
		t.Fatalf("delete last active err = %v", err)
	}

	// An archived kind keeps its slug reserved.
	if err := service.DeleteDocumentKind(guideline.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Guideline")}); !errors.Is(err, ErrGuidelineContentConflict) {
		t.Fatalf("reused archived slug err = %v", err)
	}
}

func TestDocumentKindGuidelineAssignment(t *testing.T) {
	db := documentKindTestDB(t)
	content := GuidelineContentService{DB: db}
	guidelines := GuidelineService{DB: db}
	first, second := 10, 20
	guideline, err := content.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Guideline"), SortOrder: &first})
	if err != nil {
		t.Fatal(err)
	}
	form, err := content.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Form"), SortOrder: &second})
	if err != nil {
		t.Fatal(err)
	}
	retired, err := content.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Retired"), Status: stringPtr("inactive")})
	if err != nil {
		t.Fatal(err)
	}

	defaulted, err := guidelines.CreateDocument(CreateGuidelineInput{Title: "Malaria"})
	if err != nil {
		t.Fatal(err)
	}
	if defaulted.DocumentKindID == nil || *defaulted.DocumentKindID != guideline.ID || defaulted.DocumentKind == nil || defaulted.DocumentKind.Name != "Guideline" {
		t.Fatalf("default kind = %#v", defaulted.DocumentKind)
	}
	explicit, err := guidelines.CreateDocument(CreateGuidelineInput{Title: "HMIS 105", DocumentKindID: &form.ID})
	if err != nil {
		t.Fatal(err)
	}
	if *explicit.DocumentKindID != form.ID {
		t.Fatalf("explicit kind = %v", explicit.DocumentKindID)
	}
	if _, err := guidelines.CreateDocument(CreateGuidelineInput{Title: "Old", DocumentKindID: &retired.ID}); !errors.Is(err, ErrGuidelineDocumentKindInvalid) {
		t.Fatalf("inactive kind err = %v", err)
	}
	missing := uuid.New()
	if _, err := guidelines.UpdateDocument(defaulted.ID, UpdateGuidelineInput{DocumentKindID: &missing}); !errors.Is(err, ErrGuidelineDocumentKindInvalid) {
		t.Fatalf("unknown kind err = %v", err)
	}

	forms, err := guidelines.ListDocuments(GuidelineDocumentFilter{DocumentKindID: &form.ID})
	if err != nil || forms.TotalItems != 1 || forms.Items[0].ID != explicit.ID || forms.Items[0].DocumentKind == nil {
		t.Fatalf("forms filter = %#v err=%v", forms, err)
	}

	moved, err := guidelines.UpdateDocument(defaulted.ID, UpdateGuidelineInput{DocumentKindID: &form.ID})
	if err != nil || *moved.DocumentKindID != form.ID {
		t.Fatalf("moved = %#v err=%v", moved, err)
	}
	counted, err := content.GetDocumentKind(form.ID, true)
	if err != nil || counted.GuidelineDocumentCount != 2 || counted.OutbreakResourceCount != 0 {
		t.Fatalf("form count = %#v err=%v", counted, err)
	}
	if err := content.DeleteDocumentKind(form.ID); !errors.Is(err, ErrDocumentKindInUse) {
		t.Fatalf("delete in-use err = %v", err)
	}

	// Soft-deleted documents no longer hold the kind in use.
	for _, id := range []uuid.UUID{defaulted.ID, explicit.ID} {
		if err := guidelines.DeleteDocument(id, uuid.New(), "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
	}
	if err := content.DeleteDocumentKind(form.ID); err != nil {
		t.Fatalf("delete unused kind: %v", err)
	}
	if _, err := content.GetDocumentKind(form.ID, true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted kind still readable: %v", err)
	}
}

func TestDocumentKindPublishAsUploadedMode(t *testing.T) {
	db := documentKindTestDB(t)
	content := GuidelineContentService{DB: db}
	guidelines := GuidelineService{DB: db}
	first, second := 10, 20
	guideline, err := content.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Guideline"), SortOrder: &first})
	if err != nil {
		t.Fatal(err)
	}
	asUploaded := true
	form, err := content.SaveDocumentKind(nil, DocumentKindInput{Name: stringPtr("Form"), SortOrder: &second, PublishAsUploaded: &asUploaded})
	if err != nil || !form.PublishAsUploaded {
		t.Fatalf("form kind = %#v err=%v", form, err)
	}

	document, err := guidelines.CreateDocument(CreateGuidelineInput{Title: "HMIS 105", DocumentKindID: &form.ID})
	if err != nil {
		t.Fatal(err)
	}
	version := models.GuidelineVersion{DocumentID: document.ID, Version: "1", Status: "draft"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	if yes, err := guidelines.VersionPublishesAsUploaded(version.ID); err != nil || !yes {
		t.Fatalf("form version publishes as uploaded = %v err=%v", yes, err)
	}

	// A used kind cannot change mode; an unused one can.
	editable := false
	if _, err := content.SaveDocumentKind(&form.ID, DocumentKindInput{PublishAsUploaded: &editable}); !errors.Is(err, ErrDocumentKindModeInUse) {
		t.Fatalf("mode change on used kind err = %v", err)
	}
	if updated, err := content.SaveDocumentKind(&guideline.ID, DocumentKindInput{PublishAsUploaded: &asUploaded}); err != nil || !updated.PublishAsUploaded {
		t.Fatalf("mode change on unused kind = %#v err=%v", updated, err)
	}
	if _, err := content.SaveDocumentKind(&guideline.ID, DocumentKindInput{PublishAsUploaded: &editable}); err != nil {
		t.Fatal(err)
	}

	// Before any upload the document may move to an editable kind and back.
	if _, err := guidelines.UpdateDocument(document.ID, UpdateGuidelineInput{DocumentKindID: &guideline.ID}); err != nil {
		t.Fatalf("switch before upload: %v", err)
	}
	if _, err := guidelines.UpdateDocument(document.ID, UpdateGuidelineInput{DocumentKindID: &form.ID}); err != nil {
		t.Fatalf("switch back before upload: %v", err)
	}
	if err := db.Model(&version).Update("original_file_key", "guidelines/v/original/form.pdf").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := guidelines.UpdateDocument(document.ID, UpdateGuidelineInput{DocumentKindID: &guideline.ID}); !errors.Is(err, ErrGuidelineDocumentKindModeLocked) {
		t.Fatalf("switch after upload err = %v", err)
	}
}
