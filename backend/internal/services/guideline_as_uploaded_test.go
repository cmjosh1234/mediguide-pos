package services

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime/multipart"
	"strings"
	"testing"

	"mediguide/internal/models"

	"github.com/google/uuid"
)

// testDocx builds a minimal Word document whose body contains text.
func testDocx(t *testing.T, text string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func asUploadedTestFile(content []byte) multipart.File {
	return outbreakDocumentTestFile{bytes.NewReader(content)}
}

func TestAsUploadedFormPublishesOriginalFileAndIndexedText(t *testing.T) {
	ctx := context.Background()
	db := publicGuidelineTestDB(t)
	seedDocumentKinds(t, db)
	if err := db.Model(&models.DocumentKind{}).Where("slug = ?", "form").Update("publish_as_uploaded", true).Error; err != nil {
		t.Fatal(err)
	}
	var form models.DocumentKind
	if err := db.First(&form, "slug = ?", "form").Error; err != nil {
		t.Fatal(err)
	}
	store := &fakePublicStore{objects: map[string][]byte{}}
	service := GuidelineService{DB: db, Store: store}
	document, err := service.CreateDocument(CreateGuidelineInput{Title: "HMIS 105 Outpatient Register", DocumentKindID: &form.ID})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(document.ID, CreateVersionInput{Version: "2024"})
	if err != nil {
		t.Fatal(err)
	}

	// Only real PDF or .docx files are accepted, and never Markdown or the structured PDF path.
	markdown := []byte("# Not a form")
	if _, err := service.UploadMarkdown(ctx, version.ID, asUploadedTestFile(markdown), &multipart.FileHeader{Filename: "form.md", Size: int64(len(markdown))}); !errors.Is(err, ErrGuidelinePublishedAsUploaded) {
		t.Fatalf("markdown upload err = %v", err)
	}
	pdf := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF")
	if _, err := service.UploadPDF(ctx, version.ID, asUploadedTestFile(pdf), &multipart.FileHeader{Filename: "form.pdf", Size: int64(len(pdf))}); !errors.Is(err, ErrGuidelinePublishedAsUploaded) {
		t.Fatalf("structured pdf upload err = %v", err)
	}
	for name, content := range map[string][]byte{"renamed.pdf": []byte("plain text"), "legacy.doc": pdf, "not-word.docx": pdf} {
		if _, err := service.UploadAsUploadedSource(ctx, version.ID, asUploadedTestFile(content), &multipart.FileHeader{Filename: name, Size: int64(len(content))}); !errors.Is(err, ErrUnsupportedAsUploadedSource) {
			t.Fatalf("%s upload err = %v", name, err)
		}
	}

	docx := testDocx(t, "Patient name and date of visit")
	job, err := service.UploadAsUploadedSource(ctx, version.ID, asUploadedTestFile(docx), &multipart.FileHeader{Filename: "HMIS 105.docx", Size: int64(len(docx))})
	if err != nil {
		t.Fatal(err)
	}
	if job.JobType != OriginalTextIndexJobType || job.Status != "queued" || !strings.Contains(job.PayloadJSON, `"source_format":"docx"`) {
		t.Fatalf("job = %#v", job)
	}
	var stored models.GuidelineVersion
	if err := db.First(&stored, "id = ?", version.ID).Error; err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(docx)
	if !strings.HasSuffix(stored.OriginalFileKey, ".docx") || stored.Checksum != hex.EncodeToString(sum[:]) || stored.MarkdownFileKey != "" {
		t.Fatalf("stored version = %#v", stored)
	}
	if !bytes.Equal(store.objects[stored.OriginalFileKey], docx) {
		t.Fatal("original file was not stored unchanged")
	}
	var protocols int64
	if err := db.Model(&models.ClinicalProtocol{}).Count(&protocols).Error; err != nil || protocols != 0 {
		t.Fatalf("a form must not create a clinical protocol draft: %d err=%v", protocols, err)
	}

	// Publishing waits for the text index.
	if err := service.PublishVersion(version.ID, uuid.New()); !errors.Is(err, ErrGuidelineIngestionIncomplete) {
		t.Fatalf("publish before indexing err = %v", err)
	}
	if err := db.Model(&models.IngestionJob{}).Where("id = ?", job.ID).Update("status", "completed").Error; err != nil {
		t.Fatal(err)
	}
	chunk := models.GuidelineChunk{DocumentID: document.ID, VersionID: version.ID, Title: document.Title, Content: "Patient name and date of visit", ReviewStatus: "draft"}
	if err := db.Create(&chunk).Error; err != nil {
		t.Fatal(err)
	}
	validation, err := service.ValidateVersionForPublication(version.ID)
	if err != nil || !validation.Valid || len(validation.Warnings) != 0 {
		t.Fatalf("validation = %#v err=%v", validation, err)
	}
	if err := service.PublishVersion(version.ID, uuid.New()); err != nil {
		t.Fatalf("publish form: %v", err)
	}
	if err := db.First(&chunk, "id = ?", chunk.ID).Error; err != nil || chunk.ReviewStatus != "approved" {
		t.Fatalf("form text chunk = %q err=%v", chunk.ReviewStatus, err)
	}

	public := PublicGuidelineService{DB: db, Store: store}
	detail, err := public.Get(ctx, document.ID)
	if err != nil {
		t.Fatalf("public detail: %v", err)
	}
	if detail.DocumentKind == nil || detail.DocumentKind.Slug != "form" || !detail.DocumentKind.PublishAsUploaded {
		t.Fatalf("public document kind = %#v", detail.DocumentKind)
	}
	original, err := public.Original(ctx, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if original.MIMEType != docxMIMEType || !strings.HasSuffix(original.OriginalFilename, ".docx") {
		t.Fatalf("original link = %#v", original)
	}
}

func TestAsUploadedEditingIsRejectedForPublishedAsUploadedVersions(t *testing.T) {
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
	pdf := []byte("%PDF-1.7\n%%EOF")
	if _, err := service.UploadAsUploadedSource(context.Background(), version.ID, asUploadedTestFile(pdf), &multipart.FileHeader{Filename: "malaria.pdf", Size: int64(len(pdf))}); !errors.Is(err, ErrUnsupportedGuidelineSource) {
		t.Fatalf("as-uploaded upload for an editable kind err = %v", err)
	}
}
