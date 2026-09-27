package services

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"testing"

	"mediguide/internal/models"

	"github.com/google/uuid"
)

func TestLinkDocumentPublishesItsExternalURL(t *testing.T) {
	ctx := context.Background()
	db := publicGuidelineTestDB(t)
	seedDocumentKinds(t, db)
	link := models.DocumentKind{Name: "Link", Slug: "link", Status: "active", SortOrder: 155, PublishAsLink: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	store := &fakePublicStore{objects: map[string][]byte{}}
	service := GuidelineService{DB: db, Store: store}
	document, err := service.CreateDocument(CreateGuidelineInput{Title: "WHO malaria guidelines", DocumentKindID: &link.ID})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(document.ID, CreateVersionInput{Version: "2024"})
	if err != nil {
		t.Fatal(err)
	}

	// Link documents take no files.
	pdf := []byte("%PDF-1.7\n%%EOF")
	header := &multipart.FileHeader{Filename: "who.pdf", Size: int64(len(pdf))}
	if _, err := service.UploadPDF(ctx, version.ID, asUploadedTestFile(pdf), header); !errors.Is(err, ErrGuidelinePublishedAsLink) {
		t.Fatalf("pdf upload err = %v", err)
	}
	markdown := []byte("# Not a link")
	if _, err := service.UploadMarkdown(ctx, version.ID, asUploadedTestFile(markdown), &multipart.FileHeader{Filename: "who.md", Size: int64(len(markdown))}); !errors.Is(err, ErrGuidelinePublishedAsLink) {
		t.Fatalf("markdown upload err = %v", err)
	}
	if _, err := service.UploadAsUploadedSource(ctx, version.ID, asUploadedTestFile(pdf), header); !errors.Is(err, ErrGuidelinePublishedAsLink) {
		t.Fatalf("as-uploaded upload err = %v", err)
	}

	// Publishing waits for a link.
	validation, err := service.ValidateVersionForPublication(version.ID)
	if err != nil || validation.Valid || validation.Errors[0].Code != "missing_external_url" {
		t.Fatalf("validation without link = %#v err=%v", validation, err)
	}
	if err := service.PublishVersion(version.ID, uuid.New()); !errors.Is(err, ErrGuidelineValidationFailed) {
		t.Fatalf("publish without link err = %v", err)
	}

	for _, invalid := range []string{"", "www.who.int", "http://www.who.int", "https://", "https://user:secret@www.who.int", "javascript:alert(1)", "https://www.who.int/" + strings.Repeat("a", 2048)} {
		if _, err := service.SetVersionLink(version.ID, uuid.New(), invalid, ""); !errors.Is(err, ErrGuidelineLinkInvalid) {
			t.Fatalf("link %q err = %v", invalid, err)
		}
	}
	saved, err := service.SetVersionLink(version.ID, uuid.New(), "  https://www.who.int/publications/malaria  ", "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.ExternalURL != "https://www.who.int/publications/malaria" {
		t.Fatalf("saved link = %q", saved.ExternalURL)
	}

	if err := service.PublishVersion(version.ID, uuid.New()); err != nil {
		t.Fatalf("publish link: %v", err)
	}
	if _, err := service.SetVersionLink(version.ID, uuid.New(), "https://www.who.int/other", ""); !errors.Is(err, ErrPublishedVersionImmutable) {
		t.Fatalf("link change after publish err = %v", err)
	}

	public := PublicGuidelineService{DB: db, Store: store}
	detail, err := public.Get(ctx, document.ID)
	if err != nil {
		t.Fatalf("public detail: %v", err)
	}
	if detail.DocumentKind == nil || !detail.DocumentKind.PublishAsLink || detail.ExternalURL != "https://www.who.int/publications/malaria" {
		t.Fatalf("public link = %#v kind=%#v", detail.ExternalURL, detail.DocumentKind)
	}
}

func TestLinkIsRejectedForOtherDocumentKinds(t *testing.T) {
	db := publicGuidelineTestDB(t)
	seedDocumentKinds(t, db)
	service := GuidelineService{DB: db}
	document, err := service.CreateDocument(CreateGuidelineInput{Title: "Malaria"})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(document.ID, CreateVersionInput{Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetVersionLink(version.ID, uuid.New(), "https://www.who.int", ""); !errors.Is(err, ErrGuidelineLinkNotSupported) {
		t.Fatalf("link for an editable kind err = %v", err)
	}
}

func TestLinkDocumentCannotSwitchKindModeOnceLinked(t *testing.T) {
	db := publicGuidelineTestDB(t)
	seedDocumentKinds(t, db)
	link := models.DocumentKind{Name: "Link", Slug: "link", Status: "active", SortOrder: 155, PublishAsLink: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	var guideline models.DocumentKind
	if err := db.First(&guideline, "slug = ?", "guideline").Error; err != nil {
		t.Fatal(err)
	}
	service := GuidelineService{DB: db}
	document, err := service.CreateDocument(CreateGuidelineInput{Title: "WHO", DocumentKindID: &link.ID})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(document.ID, CreateVersionInput{Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDocumentKindModeChange(db, *document, guideline.ID); err != nil {
		t.Fatalf("switch before a link is set: %v", err)
	}
	if _, err := service.SetVersionLink(version.ID, uuid.New(), "https://www.who.int", ""); err != nil {
		t.Fatal(err)
	}
	if err := ensureDocumentKindModeChange(db, *document, guideline.ID); !errors.Is(err, ErrGuidelineDocumentKindModeLocked) {
		t.Fatalf("switch after a link is set err = %v", err)
	}
}
