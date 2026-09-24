package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PublicOutbreakTopic is the Firebase topic the mobile app subscribes to when
// a signed-in user turns on outbreak alerts.
const PublicOutbreakTopic = "public-outbreaks"

// An alert that could not be sent within a day is no longer news, and the
// outbreak hub already shows the current picture.
const outbreakTopicAlertTTL = 24 * time.Hour

// NotificationTopicService sends queued public topic broadcasts. Jobs are
// written in the same transaction as the content they announce, so a publish
// that commits is always announced and one that rolls back never is.
type NotificationTopicService struct {
	DB            *gorm.DB
	Firebase      *FirebaseService
	WorkerID      string
	BatchSize     int
	LeaseDuration time.Duration
	Now           func() time.Time
}

type outbreakTopicAlert struct {
	SourceType string
	SourceID   uuid.UUID
	OutbreakID uuid.UUID
	Title      string
	Body       string
	Urgent     bool
}

// outbreakPublishAnnounced reports whether publishing an outbreak should alert
// subscribers. A correction republishes content that was already announced,
// and an outbreak published as contained or closed is a record, not news.
func outbreakPublishAnnounced(row models.Outbreak, status string) bool {
	return row.SupersedesID == nil && status != "contained" && status != "closed"
}

func enqueueOutbreakTopicAlertTx(tx *gorm.DB, alert outbreakTopicAlert, now time.Time) error {
	resourceID := alert.OutbreakID.String()
	priority := "normal"
	if alert.Urgent {
		priority = "urgent"
	}
	body := strings.TrimSpace(alert.Body)
	if body == "" {
		body = "Open MediGuide for guidance and updates."
	}
	payload, err := json.Marshal(NotificationDeliveryPayload{
		Title:          truncateRunes(strings.TrimSpace(alert.Title), 120),
		Body:           truncateRunes(body, 240),
		Action:         NotificationAction{Type: NotificationActionOutbreak, ResourceID: &resourceID, Parameters: map[string]string{}},
		Priority:       priority,
		CollapseKey:    "outbreak-" + resourceID,
		TTLSeconds:     int(outbreakTopicAlertTTL.Seconds()),
		AndroidChannel: notificationAndroidChannel("", priority),
		PublicContent:  true,
	})
	if err != nil {
		return err
	}
	now = now.UTC()
	expires := now.Add(outbreakTopicAlertTTL)
	job := models.NotificationTopicJob{
		Topic: PublicOutbreakTopic, SourceType: alert.SourceType, SourceID: alert.SourceID,
		Status: "pending", IdempotencyKey: alert.SourceType + ":" + alert.SourceID.String() + ":published",
		PayloadJSON: payload, MaxAttempts: 8, NextAttemptAt: now, ExpiresAt: &expires,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&job).Error
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

func (s NotificationTopicService) ProcessBatch(ctx context.Context) (*NotificationWorkerBatchResult, error) {
	jobs, err := s.claimBatch()
	if err != nil {
		return nil, err
	}
	result := &NotificationWorkerBatchResult{Claimed: len(jobs)}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		outcome := s.deliver(ctx, job)
		status, err := s.recordResult(job, outcome)
		if err != nil {
			return result, err
		}
		switch status {
		case "accepted":
			result.Accepted++
		case "retry":
			result.Retried++
		default:
			result.Failed++
		}
	}
	return result, nil
}

func (s NotificationTopicService) claimBatch() ([]models.NotificationTopicJob, error) {
	now := s.now()
	batch := s.BatchSize
	if batch <= 0 || batch > 100 {
		batch = 20
	}
	lease := s.LeaseDuration
	if lease <= 0 {
		lease = 2 * time.Minute
	}
	worker := strings.TrimSpace(s.WorkerID)
	if worker == "" {
		worker = uuid.NewString()
	}
	claimed := []models.NotificationTopicJob{}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Where("((status IN ('pending','retry') AND next_attempt_at <= ?) OR (status = 'processing' AND locked_at < ?))", now, now.Add(-lease)).Order("next_attempt_at, created_at").Limit(batch)
		if tx.Dialector.Name() == "postgres" {
			query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		}
		if err := query.Find(&claimed).Error; err != nil {
			return err
		}
		for i := range claimed {
			if err := tx.Model(&models.NotificationTopicJob{}).Where("id = ?", claimed[i].ID).Updates(map[string]any{"status": "processing", "locked_at": now, "locked_by": worker, "attempt_count": gorm.Expr("attempt_count + 1")}).Error; err != nil {
				return err
			}
			claimed[i].Status = "processing"
			claimed[i].LockedAt = &now
			claimed[i].LockedBy = &worker
			claimed[i].AttemptCount++
		}
		return nil
	})
	return claimed, err
}

func (s NotificationTopicService) deliver(ctx context.Context, job models.NotificationTopicJob) FirebaseDeliveryOutcome {
	if s.expired(job, s.now()) {
		return FirebaseDeliveryOutcome{Code: "expired", Message: "topic broadcast expired"}
	}
	if s.Firebase == nil {
		return FirebaseDeliveryOutcome{Code: "firebase_disabled", Message: "Firebase messaging is not configured"}
	}
	var payload NotificationDeliveryPayload
	if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
		return FirebaseDeliveryOutcome{Code: "invalid_payload", Message: "notification payload is invalid"}
	}
	return s.Firebase.DeliverPublicTopic(ctx, job.Topic, payload, false)
}

func (s NotificationTopicService) recordResult(job models.NotificationTopicJob, outcome FirebaseDeliveryOutcome) (string, error) {
	now := s.now()
	status := "accepted"
	updates := map[string]any{"locked_at": nil, "locked_by": nil, "last_error_code": cleanOptional(&outcome.Code), "last_error_message": cleanOptional(&outcome.Message)}
	switch {
	case outcome.Accepted || outcome.Validated:
		updates["completed_at"] = now
		updates["provider_message_id"] = cleanOptional(&outcome.MessageID)
	case outcome.Retryable && job.AttemptCount < job.MaxAttempts && !s.expired(job, now):
		status = "retry"
		delay := outcome.RetryAfter
		if delay <= 0 {
			delay = retryBackoff(job.AttemptCount)
		}
		updates["next_attempt_at"] = now.Add(delay)
	default:
		status = "failed"
		updates["completed_at"] = now
	}
	updates["status"] = status
	result := s.DB.Model(&models.NotificationTopicJob{}).Where("id = ? AND status = 'processing' AND locked_by = ?", job.ID, notificationStringValue(job.LockedBy)).Updates(updates)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected != 1 {
		return "", ErrNotificationConflict
	}
	return status, nil
}

func (s NotificationTopicService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s NotificationTopicService) expired(job models.NotificationTopicJob, now time.Time) bool {
	return job.ExpiresAt != nil && !job.ExpiresAt.After(now)
}
