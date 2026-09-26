package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"mediguide/internal/models"
	"mediguide/internal/storage"
	"testing"
	"time"
)

type multipartTestStore struct {
	fakePublicStore
	source     []byte
	completed  bool
	incomplete bool
}

func (s *multipartTestStore) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *multipartTestStore) BeginUpload(context.Context, string, string) (string, error) {
	return "upload", nil
}
func (s *multipartTestStore) SignUploadPart(context.Context, string, string, int) (string, error) {
	return "https://storage.test/signed", nil
}
func (s *multipartTestStore) UploadParts(context.Context, string, string) ([]storage.UploadPart, error) {
	if s.completed {
		return nil, errors.New("upload already completed")
	}
	if s.incomplete {
		return []storage.UploadPart{}, nil
	}
	return []storage.UploadPart{{Number: 1, ETag: "part", Size: int64(len(s.source))}}, nil
}
func (s *multipartTestStore) CompleteUpload(_ context.Context, key, _ string, _ []storage.UploadPart) error {
	s.objects[key] = s.source
	s.completed = true
	return nil
}
func (s *multipartTestStore) AbortUpload(context.Context, string, string) error { return nil }
func (s *multipartTestStore) ObjectSize(_ context.Context, key string) (int64, error) {
	if b, ok := s.objects[key]; ok {
		return int64(len(b)), nil
	}
	return 0, errors.New("not found")
}

func uploadFixture(t *testing.T, filename, content string) (GuidelineService, *multipartTestStore, *models.GuidelineUpload) {
	t.Helper()
	db := publicGuidelineTestDB(t)
	if err := db.AutoMigrate(&models.GuidelineUpload{}); err != nil {
		t.Fatal(err)
	}
	document := models.GuidelineDocument{Title: "Upload test", Language: "en"}
	if err := db.Create(&document).Error; err != nil {
		t.Fatal(err)
	}
	version := models.GuidelineVersion{DocumentID: document.ID, Version: "test", Status: "draft"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	store := &multipartTestStore{fakePublicStore: fakePublicStore{objects: map[string][]byte{}}, source: []byte(content)}
	service := GuidelineService{DB: db, Store: store}
	row, err := service.BeginSourceUpload(context.Background(), version.ID, uuid.New(), BeginGuidelineUpload{Filename: filename, SizeBytes: int64(len(content)), Checksum: fmt.Sprintf("%x", sha256.Sum256([]byte(content)))}, 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	return service, store, row
}

func TestSourceUploadCompletionIsIdempotentAndRecoversStorageCommit(t *testing.T) {
	s, store, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	ctx := context.Background()
	// Simulate a crash between storage completion and database commit.
	store.objects[row.ObjectKey] = store.source
	store.completed = true
	state, err := s.UploadSession(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil || !state.ObjectComplete {
		t.Fatalf("recovery state: %+v %v", state, err)
	}
	job, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil || again.ID != job.ID {
		t.Fatalf("duplicate completion: %v", err)
	}
	var count int64
	s.DB.Model(&models.IngestionJob{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected one job, got %d", count)
	}
	if err = s.AbortSourceUpload(ctx, row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("completed source must be retained: %v", err)
	}
}

func TestSourceUploadRejectsWrongOwnerPartsChecksumAndPublishedVersion(t *testing.T) {
	s, store, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	ctx := context.Background()
	if _, err := s.UploadSession(ctx, row.VersionID, uuid.New(), row.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("ownership: %v", err)
	}
	if _, err := s.SignSourcePart(ctx, row.VersionID, row.UserID, row.ID, 2); !errors.Is(err, ErrUploadInvalid) {
		t.Fatalf("part bounds: %v", err)
	}
	store.incomplete = true
	if _, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrUploadInvalid) {
		t.Fatalf("missing parts: %v", err)
	}
	store.incomplete = false
	store.source = []byte("%PDF-1.7\nFake document")
	if _, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrUploadInvalid) {
		t.Fatalf("checksum: %v", err)
	}
	s.DB.Model(&models.GuidelineVersion{}).Where("id=?", row.VersionID).Update("status", "published")
	if _, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrPublishedVersionImmutable) {
		t.Fatalf("published: %v", err)
	}
}

func TestSourceUploadMarkdownAndExpiredCleanup(t *testing.T) {
	s, _, row := uploadFixture(t, "source.md", "# Guideline\n\n## Care\nReviewed source text.")
	ctx := context.Background()
	job, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.JobType != "markdown_ingestion" {
		t.Fatalf("wrong job: %s", job.JobType)
	}
	var revisions int64
	s.DB.Model(&models.GuidelineMarkdownRevision{}).Count(&revisions)
	if revisions != 1 {
		t.Fatalf("expected immutable revision, got %d", revisions)
	}
	s, store, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	s.DB.Model(row).Update("expires_at", time.Now().UTC().Add(-time.Hour))
	if _, err = s.SignSourcePart(ctx, row.VersionID, row.UserID, row.ID, 1); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("expired: %v", err)
	}
	if err = s.MaintainSourceUploads(ctx); err != nil {
		t.Fatal(err)
	}
	s.DB.First(row, "id=?", row.ID)
	if row.Status != "aborted" || len(store.objects) != 0 {
		t.Fatal("expired data not cleaned up")
	}
	if err = s.AbortSourceUpload(ctx, row.VersionID, row.UserID, row.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSourceUploadCompletionNotificationIsDeduplicated(t *testing.T) {
	s, _, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	if err := s.DB.AutoMigrate(&models.Notification{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	job, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Model(job).Update("status", "completed")
	for i := 0; i < 2; i++ {
		if err = s.MaintainSourceUploads(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var notices []models.Notification
	s.DB.Find(&notices)
	if len(notices) != 1 || notices[0].UserID == nil || *notices[0].UserID != row.UserID {
		t.Fatalf("expected one owner notification: %+v", notices)
	}
}

func TestSourceUploadRejectsInvalidContentAndCanceledSessions(t *testing.T) {
	for _, test := range []struct{ filename, content string }{
		{"source.pdf", "This is not a PDF"},
		{"source.md", "# Invalid UTF8\n\xff"},
	} {
		t.Run(test.filename, func(t *testing.T) {
			s, _, row := uploadFixture(t, test.filename, test.content)
			if _, err := s.CompleteSourceUpload(context.Background(), row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrUploadInvalid) {
				t.Fatalf("invalid content accepted: %v", err)
			}
			var count int64
			s.DB.Model(&models.IngestionJob{}).Count(&count)
			if count != 0 {
				t.Fatalf("invalid upload queued %d jobs", count)
			}
		})
	}
	s, _, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	ctx := context.Background()
	if err := s.AbortSourceUpload(ctx, row.VersionID, row.UserID, row.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("canceled completion: %v", err)
	}
	if _, err := s.SignSourcePart(ctx, row.VersionID, row.UserID, row.ID, 1); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("canceled signing: %v", err)
	}
}

func TestRetrySourceIngestionJobResetsRuntimeState(t *testing.T) {
	s, _, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	ctx := context.Background()
	job, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err = s.DB.Model(job).Updates(map[string]any{
		"status": "failed", "progress_stage": "embeddings", "progress_percent": 72,
		"attempt_count": 1, "worker_id": "dead-worker", "claimed_at": now,
		"heartbeat_at": now, "lease_expires_at": now, "next_attempt_at": now.Add(time.Minute),
		"completed_at": now, "error": "temporary failure",
	}).Error; err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	if err = s.DB.Create(&models.User{Base: models.Base{ID: actor}, Email: "retry@test.invalid", Name: "Retry User"}).Error; err != nil {
		t.Fatal(err)
	}
	retried, err := s.RetrySourceIngestionJob(ctx, row.VersionID, job.ID, actor)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != "queued" || retried.ProgressStage != "queued" || retried.ProgressPercent != 0 || retried.WorkerID != "" || retried.LeaseExpiresAt != nil || retried.NextAttemptAt != nil || retried.CompletedAt != nil {
		t.Fatalf("job was not reset for retry: %+v", retried)
	}
}

func TestUpdateSourceIngestionPriorityOnlyAllowsWaitingWork(t *testing.T) {
	s, _, row := uploadFixture(t, "source.pdf", "%PDF-1.7\nTest document")
	ctx := context.Background()
	job, err := s.CompleteSourceUpload(ctx, row.VersionID, row.UserID, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	if err = s.DB.Create(&models.User{Base: models.Base{ID: actor}, Email: "priority@test.invalid", Name: "Priority User"}).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := s.UpdateSourceIngestionPriority(ctx, row.VersionID, job.ID, actor, 100)
	if err != nil || updated.Priority != 100 {
		t.Fatalf("priority update failed: %+v %v", updated, err)
	}
	if _, err = s.UpdateSourceIngestionPriority(ctx, row.VersionID, job.ID, actor, 101); !errors.Is(err, ErrUploadInvalid) {
		t.Fatalf("invalid priority accepted: %v", err)
	}
	if err = s.DB.Model(job).Update("status", "running").Error; err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateSourceIngestionPriority(ctx, row.VersionID, job.ID, actor, 75); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("running job priority changed: %v", err)
	}
}
