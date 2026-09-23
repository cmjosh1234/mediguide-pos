package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mediguide/internal/models"
	"mediguide/internal/storage"
)

var ErrUploadInvalid = errors.New("invalid upload")
var ErrUploadConflict = errors.New("upload is no longer available")

type BeginGuidelineUpload struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	Checksum  string `json:"checksum"`
}

type GuidelineUploadState struct {
	models.GuidelineUpload
	Parts          []storage.UploadPart `json:"parts"`
	Job            *models.IngestionJob `json:"job,omitempty"`
	ObjectComplete bool                 `json:"object_complete"`
}

func (s GuidelineService) BeginSourceUpload(ctx context.Context, versionID, userID uuid.UUID, in BeginGuidelineUpload, maxBytes int64) (*models.GuidelineUpload, error) {
	ext := strings.ToLower(filepath.Ext(in.Filename))
	checksum, err := hex.DecodeString(in.Checksum)
	if in.SizeBytes <= 0 || in.SizeBytes > maxBytes || err != nil || len(checksum) != 32 || (ext != ".pdf" && ext != ".md" && ext != ".markdown") || len(in.Filename) > 255 {
		return nil, fmt.Errorf("%w: choose a PDF or UTF-8 Markdown file within the upload limit and supply its SHA-256", ErrUploadInvalid)
	}
	if ext != ".pdf" && in.SizeBytes > maxMarkdownDraftBytes {
		return nil, fmt.Errorf("%w: Markdown exceeds the authoring limit", ErrUploadInvalid)
	}
	var version models.GuidelineVersion
	if err = s.DB.WithContext(ctx).First(&version, "id = ?", versionID).Error; err != nil {
		return nil, err
	}
	if err = validateVersionAllowsIngestion(&version); err != nil {
		return nil, err
	}
	asUploaded, err := versionPublishesAsUploaded(s.DB.WithContext(ctx), versionID)
	if err != nil {
		return nil, err
	}
	if asUploaded {
		return nil, fmt.Errorf("%w: this document kind is published as uploaded; use the standard upload", ErrUploadConflict)
	}
	store, ok := s.Store.(storage.MultipartStore)
	if !ok {
		return nil, ErrUploadConflict
	}
	session := models.GuidelineUpload{VersionID: versionID, UserID: userID, Filename: filepath.Base(in.Filename), SizeBytes: in.SizeBytes, Checksum: strings.ToLower(in.Checksum), PartSize: 8 << 20, Status: "uploading", ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	session.ID = uuid.New()
	session.ObjectKey = fmt.Sprintf("guidelines/%s/uploads/%s/source%s", versionID, session.ID, ext)
	mime := "text/markdown; charset=utf-8"
	if ext == ".pdf" {
		mime = "application/pdf"
	}
	session.MultipartID, err = store.BeginUpload(ctx, session.ObjectKey, mime)
	if err != nil {
		return nil, err
	}
	if err = s.DB.WithContext(ctx).Create(&session).Error; err != nil {
		_ = store.AbortUpload(ctx, session.ObjectKey, session.MultipartID)
		return nil, err
	}
	return &session, nil
}

func (s GuidelineService) UploadSession(ctx context.Context, versionID, userID, id uuid.UUID) (*GuidelineUploadState, error) {
	var row models.GuidelineUpload
	if err := s.DB.WithContext(ctx).First(&row, "id=? AND version_id=? AND user_id=?", id, versionID, userID).Error; err != nil {
		return nil, err
	}
	state := &GuidelineUploadState{GuidelineUpload: row, Parts: []storage.UploadPart{}}
	if row.Status == "uploading" && time.Now().Before(row.ExpiresAt) {
		var err error
		state.Parts, err = s.Store.(storage.MultipartStore).UploadParts(ctx, row.ObjectKey, row.MultipartID)
		// Completion may have succeeded in storage before a process restart. The
		// complete endpoint verifies the durable object and finishes the transaction.
		if err != nil {
			if size, e := s.Store.(storage.MultipartStore).ObjectSize(ctx, row.ObjectKey); e != nil || size != row.SizeBytes {
				return nil, err
			}
			state.ObjectComplete = true
			state.Parts = []storage.UploadPart{}
		}
	}
	if row.JobID != nil {
		var job models.IngestionJob
		if err := s.DB.WithContext(ctx).First(&job, "id=?", *row.JobID).Error; err != nil {
			return nil, err
		}
		job.PayloadJSON = ""
		if job.Error != "" {
			job.Error = "Document processing failed. Open the review workspace or ask an administrator to inspect the job logs."
		}
		state.Job = &job
	}
	return state, nil
}

func (s GuidelineService) SignSourcePart(ctx context.Context, versionID, userID, id uuid.UUID, number int) (string, error) {
	var row models.GuidelineUpload
	if err := s.DB.WithContext(ctx).First(&row, "id=? AND version_id=? AND user_id=?", id, versionID, userID).Error; err != nil {
		return "", err
	}
	if row.Status != "uploading" || time.Now().After(row.ExpiresAt) {
		return "", ErrUploadConflict
	}
	if number < 1 || int64(number) > (row.SizeBytes+row.PartSize-1)/row.PartSize {
		return "", ErrUploadInvalid
	}
	return s.Store.(storage.MultipartStore).SignUploadPart(ctx, row.ObjectKey, row.MultipartID, number)
}

func (s GuidelineService) CompleteSourceUpload(ctx context.Context, versionID, userID, id uuid.UUID) (*models.IngestionJob, error) {
	verificationStarted := time.Now()
	var job models.IngestionJob
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.GuidelineUpload
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id=? AND version_id=? AND user_id=?", id, versionID, userID).Error; err != nil {
			return err
		}
		if row.JobID != nil {
			return tx.First(&job, "id=?", *row.JobID).Error
		}
		if row.Status != "uploading" || time.Now().After(row.ExpiresAt) {
			return ErrUploadConflict
		}
		var version models.GuidelineVersion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, "id=?", versionID).Error; err != nil {
			return err
		}
		if err := validateVersionAllowsIngestion(&version); err != nil {
			return err
		}
		store := s.Store.(storage.MultipartStore)
		// A durable final object also allows recovery after storage completion but
		// before the database commit. Neither ETags nor client claims are trusted as
		// the file checksum.
		if _, err := store.ObjectSize(ctx, row.ObjectKey); err != nil {
			parts, err := store.UploadParts(ctx, row.ObjectKey, row.MultipartID)
			if err != nil {
				return err
			}
			sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
			expected := int((row.SizeBytes + row.PartSize - 1) / row.PartSize)
			if len(parts) != expected {
				return fmt.Errorf("%w: upload parts are incomplete", ErrUploadInvalid)
			}
			for i, p := range parts {
				size := row.PartSize
				if i == expected-1 {
					size = row.SizeBytes - int64(i)*row.PartSize
				}
				if p.Number != i+1 || p.Size != size {
					return fmt.Errorf("%w: unexpected part size or order", ErrUploadInvalid)
				}
			}
			if err = store.CompleteUpload(ctx, row.ObjectKey, row.MultipartID, parts); err != nil {
				return err
			}
		}
		size, err := store.ObjectSize(ctx, row.ObjectKey)
		if err != nil {
			return err
		}
		if size != row.SizeBytes {
			return ErrUploadInvalid
		}
		reader, err := s.Store.Get(ctx, row.ObjectKey)
		if err != nil {
			return err
		}
		hash := sha256.New()
		prefix := make([]byte, 512)
		n, readErr := io.ReadFull(reader, prefix)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			reader.Close()
			return readErr
		}
		hash.Write(prefix[:n])
		copied, err := io.Copy(hash, io.LimitReader(reader, row.SizeBytes+1))
		reader.Close()
		if err != nil {
			return err
		}
		if int64(n)+copied != row.SizeBytes || hex.EncodeToString(hash.Sum(nil)) != row.Checksum {
			return fmt.Errorf("%w: file checksum does not match", ErrUploadInvalid)
		}
		ext := strings.ToLower(filepath.Ext(row.Filename))
		if ext == ".pdf" && !strings.HasPrefix(string(prefix[:n]), "%PDF-") {
			return fmt.Errorf("%w: file is not a PDF", ErrUploadInvalid)
		}
		scoped := s
		scoped.DB = tx
		if ext != ".pdf" {
			r, err := s.Store.Get(ctx, row.ObjectKey)
			if err != nil {
				return err
			}
			content, err := io.ReadAll(io.LimitReader(r, maxMarkdownDraftBytes+1))
			r.Close()
			if err != nil {
				return err
			}
			if !utf8.Valid(content) || strings.ContainsRune(string(content), 0) || strings.TrimSpace(string(content)) == "" {
				return fmt.Errorf("%w: file must contain UTF-8 Markdown", ErrUploadInvalid)
			}
			queued, err := scoped.queueMarkdownSource(ctx, versionID, strings.NewReader(string(content)), int64(len(content)), row.Filename, "uploaded_markdown")
			if err != nil {
				return err
			}
			job = *queued
		} else {
			v, document, err := scoped.loadVersionDocument(tx, versionID)
			if err != nil {
				return err
			}
			if err = ensureDraftProtocol(tx, document, v); err != nil {
				return err
			}
			payload, _ := json.Marshal(map[string]any{"file_key": row.ObjectKey, "source_format": "pdf", "upload_session_id": row.ID})
			job = models.IngestionJob{VersionID: versionID, JobType: "pdf_ingestion", Status: "queued", PayloadJSON: string(payload)}
			if err = tx.Model(&version).Update("original_file_key", row.ObjectKey).Error; err != nil {
				return err
			}
			if err = tx.Create(&job).Error; err != nil {
				return err
			}
		}
		metrics, err := json.Marshal(map[string]any{"upload_session_seconds": time.Since(row.CreatedAt).Seconds(), "verification_and_queue_seconds": time.Since(verificationStarted).Seconds(), "upload_source_bytes": row.SizeBytes})
		if err != nil {
			return err
		}
		if err = tx.Model(&job).Update("metrics_json", string(metrics)).Error; err != nil {
			return err
		}
		return tx.Model(&row).Updates(map[string]any{"status": "completed", "job_id": job.ID}).Error
	})
	return &job, err
}

func (s GuidelineService) AbortSourceUpload(ctx context.Context, versionID, userID, id uuid.UUID) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row models.GuidelineUpload
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id=? AND version_id=? AND user_id=?", id, versionID, userID).Error; err != nil {
			return err
		}
		if row.Status == "aborted" {
			return nil
		}
		if row.JobID != nil {
			return ErrUploadConflict
		}
		// MinIO may already have completed the object before validation failed.
		store := s.Store.(storage.MultipartStore)
		if err := store.AbortUpload(ctx, row.ObjectKey, row.MultipartID); err != nil {
			if _, e := store.ObjectSize(ctx, row.ObjectKey); e != nil {
				return err
			}
		}
		if err := s.Store.Delete(ctx, row.ObjectKey); err != nil {
			return err
		}
		return tx.Model(&row).Update("status", "aborted").Error
	})
}
