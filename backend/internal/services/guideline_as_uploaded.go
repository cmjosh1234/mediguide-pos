package services

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrGuidelinePublishedAsUploaded rejects content editing for documents whose
	// kind publishes the uploaded file itself (for example forms).
	ErrGuidelinePublishedAsUploaded = errors.New("this document is published as its uploaded file and has no editable content")
	// ErrGuidelineDocumentKindModeLocked stops a document with an uploaded
	// source from switching between as-uploaded and editable kinds.
	ErrGuidelineDocumentKindModeLocked = errors.New("a document with an uploaded file cannot switch between publish-as-uploaded and editable document kinds")
)

// VersionPublishesAsUploaded reports whether the version's document kind keeps
// the uploaded file as the published document.
func (s GuidelineService) VersionPublishesAsUploaded(versionID uuid.UUID) (bool, error) {
	return versionPublishesAsUploaded(s.DB, versionID)
}

func versionPublishesAsUploaded(tx *gorm.DB, versionID uuid.UUID) (bool, error) {
	var version models.GuidelineVersion
	if err := tx.Select("id", "document_id").First(&version, "id = ?", versionID).Error; err != nil {
		return false, err
	}
	return documentPublishesAsUploaded(tx, version.DocumentID)
}

func documentPublishesAsUploaded(tx *gorm.DB, documentID uuid.UUID) (bool, error) {
	var document models.GuidelineDocument
	if err := tx.Select("id", "document_kind_id").First(&document, "id = ?", documentID).Error; err != nil {
		return false, err
	}
	return documentKindPublishesAsUploaded(tx, document.DocumentKindID)
}

func documentKindPublishesAsUploaded(tx *gorm.DB, kindID *uuid.UUID) (bool, error) {
	if kindID == nil {
		return false, nil
	}
	var kind models.DocumentKind
	// Archived kinds still describe the documents that use them.
	if err := tx.Unscoped().Select("id", "publish_as_uploaded").First(&kind, "id = ?", *kindID).Error; err != nil {
		return false, err
	}
	return kind.PublishAsUploaded, nil
}

// ensureDocumentKindModeChange allows a kind change that keeps the publishing
// mode, or any change before a file or Markdown has been added to the document.
func ensureDocumentKindModeChange(tx *gorm.DB, document models.GuidelineDocument, kindID uuid.UUID) error {
	current, err := documentKindPublishesAsUploaded(tx, document.DocumentKindID)
	if err != nil {
		return err
	}
	next, err := documentKindPublishesAsUploaded(tx, &kindID)
	if err != nil {
		return err
	}
	if current == next {
		return nil
	}
	var withContent int64
	if err := tx.Model(&models.GuidelineVersion{}).
		Where("document_id = ? AND (original_file_key <> '' OR markdown_file_key <> '' OR current_markdown_revision_id IS NOT NULL)", document.ID).
		Count(&withContent).Error; err != nil {
		return err
	}
	if withContent > 0 {
		return ErrGuidelineDocumentKindModeLocked
	}
	return nil
}

// OriginalTextIndexJobType asks the AI worker to index the text of an original
// file for search and RAG without producing editable content.
const OriginalTextIndexJobType = "original_text_index"

const docxMIMEType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// ErrUnsupportedAsUploadedSource rejects files that cannot be kept as the
// published document.
var ErrUnsupportedAsUploadedSource = errors.New("documents published as uploaded must be a PDF or Word (.docx) file; save .doc files as .docx or PDF first")

// originalFileType returns the download extension and MIME type of an original
// file. Original files are PDFs unless they were uploaded as Word documents.
func originalFileType(key string) (extension, mimeType string) {
	if strings.EqualFold(filepath.Ext(key), ".docx") {
		return "docx", docxMIMEType
	}
	return "pdf", "application/pdf"
}

// detectAsUploadedSource checks the file contents as well as its name, so a
// renamed file cannot be published as a form.
func detectAsUploadedSource(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size <= 0 {
		return "", ErrUnsupportedAsUploadedSource
	}
	defer file.Seek(0, io.SeekStart) //nolint:errcheck // rewound again before storage
	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case ".pdf":
		head := make([]byte, 5)
		if _, err := io.ReadFull(file, head); err != nil || string(head) != "%PDF-" {
			return "", ErrUnsupportedAsUploadedSource
		}
		return "pdf", nil
	case ".docx":
		archive, err := zip.NewReader(file, header.Size)
		if err != nil {
			return "", ErrUnsupportedAsUploadedSource
		}
		for _, entry := range archive.File {
			if entry.Name == "word/document.xml" {
				return "docx", nil
			}
		}
	}
	return "", ErrUnsupportedAsUploadedSource
}

// UploadAsUploadedSource stores a PDF or Word file unchanged as the version's
// original and queues text indexing for search. No Markdown, sections or
// blocks are produced, so the file itself is what readers see.
func (s GuidelineService) UploadAsUploadedSource(ctx context.Context, versionID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*models.IngestionJob, error) {
	var target models.GuidelineVersion
	if err := s.DB.First(&target, "id = ?", versionID).Error; err != nil {
		return nil, err
	}
	if err := validateVersionAllowsIngestion(&target); err != nil {
		return nil, err
	}
	asUploaded, err := versionPublishesAsUploaded(s.DB, versionID)
	if err != nil {
		return nil, err
	}
	if !asUploaded {
		return nil, ErrUnsupportedGuidelineSource
	}
	format, err := detectAsUploadedSource(file, header)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	key := fmt.Sprintf("guidelines/%s/original/%d_%s", versionID.String(), time.Now().Unix(), filepath.Base(header.Filename))
	_, contentType := originalFileType(key)
	if err := s.Store.Put(ctx, key, file, header.Size, contentType); err != nil {
		return nil, err
	}
	var job models.IngestionJob
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.GuidelineVersion{}).Where("id = ?", versionID).Updates(map[string]any{
			"original_file_key": key, "checksum": checksum,
		}).Error; err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]string{"file_key": key, "source_format": format, "checksum": checksum})
		if err != nil {
			return err
		}
		job = models.IngestionJob{VersionID: versionID, JobType: OriginalTextIndexJobType, Status: "queued", PayloadJSON: string(payload)}
		return tx.Create(&job).Error
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// ensureAsUploadedVersionReadyForPublish requires the stored original and a
// finished text index, so published forms are searchable from the start.
func ensureAsUploadedVersionReadyForPublish(tx *gorm.DB, version *models.GuidelineVersion) error {
	if strings.TrimSpace(version.OriginalFileKey) == "" {
		return fmt.Errorf("%w: upload the PDF or Word file before publishing", ErrGuidelineIngestionIncomplete)
	}
	return ensureLatestIngestionJobCompleted(tx, version.ID)
}
