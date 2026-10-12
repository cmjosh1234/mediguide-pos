package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProcessDigests leases one durable batch per device/window. Its member jobs
// retain individual preference, expiry and campaign delivery lifecycles.
func (s NotificationOutboxService) ProcessDigests(ctx context.Context) (*NotificationWorkerBatchResult, error) {
	now := s.now()
	worker := s.WorkerID
	if worker == "" {
		worker = uuid.NewString()
	}
	lease := s.LeaseDuration
	if lease <= 0 {
		lease = 2 * time.Minute
	}
	var digests []models.NotificationDigest
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("((status IN ('pending','retry') AND next_attempt_at <= ?) OR (status = 'processing' AND locked_at < ?))", now, now.Add(-lease)).
			Where("EXISTS (SELECT 1 FROM notification_outbox_jobs jobs WHERE jobs.digest_id = notification_digests.id AND jobs.deleted_at IS NULL AND ((jobs.status IN ('pending','retry') AND jobs.next_attempt_at <= ?) OR (jobs.status = 'processing' AND jobs.locked_at < ?) ))", now, now.Add(-lease)).
			Order("next_attempt_at, id").Limit(20)
		if tx.Dialector.Name() == "postgres" {
			q = q.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		}
		if err := q.Find(&digests).Error; err != nil {
			return err
		}
		for i := range digests {
			attempt := digests[i].AttemptCount + 1
			if digests[i].Status == "accepted" {
				attempt = 1
			}
			if err := tx.Model(&digests[i]).Updates(map[string]any{"status": "processing", "locked_at": now, "locked_by": worker, "attempt_count": attempt}).Error; err != nil {
				return err
			}
			digests[i].AttemptCount = attempt
			digests[i].LockedBy = &worker
		}
		return nil
	})
	result := &NotificationWorkerBatchResult{Claimed: len(digests)}
	if err != nil {
		return result, err
	}
	for _, digest := range digests {
		if err := s.processDigest(ctx, digest, worker, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s NotificationOutboxService) processDigest(ctx context.Context, digest models.NotificationDigest, worker string, result *NotificationWorkerBatchResult) error {
	var jobs []models.NotificationOutboxJob
	now := s.now()
	// Freeze the member set before sending; new content gets the next window.
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("digest_id = ? AND ((status IN ('pending','retry') AND next_attempt_at <= ?) OR (status = 'processing' AND locked_at < ?))", digest.ID, now, now.Add(-2*time.Minute)).Find(&jobs).Error; err != nil {
			return err
		}
		for i := range jobs {
			if err := tx.Model(&jobs[i]).Updates(map[string]any{"status": "processing", "locked_at": now, "locked_by": worker, "attempt_count": gorm.Expr("attempt_count + 1")}).Error; err != nil {
				return err
			}
			jobs[i].LockedBy = &worker
			jobs[i].AttemptCount++
		}
		return nil
	})
	if err != nil {
		return err
	}
	campaigns := map[uuid.UUID]bool{}
	var eligible []models.NotificationOutboxJob
	ttl := 3600
	for _, job := range jobs {
		campaigns[job.CampaignID] = true
		var payload NotificationDeliveryPayload
		var blocked *FirebaseDeliveryOutcome
		if s.expired(job, now) {
			blocked = &FirebaseDeliveryOutcome{Code: "expired", Message: "digest item expired"}
		} else if json.Unmarshal(job.PayloadJSON, &payload) != nil {
			blocked = &FirebaseDeliveryOutcome{Code: "invalid_payload", Message: "invalid digest item"}
		} else {
			blocked = s.deliveryPreferenceOutcome(job, payload, now)
		}
		if blocked != nil {
			if err := s.RecordResult(job, *blocked, now); err != nil {
				return err
			}
			continue
		}
		if job.ExpiresAt != nil {
			remaining := int(job.ExpiresAt.Sub(now).Seconds())
			if remaining <= 0 {
				if err := s.RecordResult(job, FirebaseDeliveryOutcome{Code: "expired", Message: "digest item expired"}, now); err != nil {
					return err
				}
				continue
			}
			if remaining < ttl {
				ttl = remaining
			}
		}
		eligible = append(eligible, job)
	}
	outcome := FirebaseDeliveryOutcome{Accepted: true}
	if len(eligible) > 0 {
		var device models.FirebaseDevice
		if s.Firebase == nil || !s.Firebase.Enabled() {
			outcome = FirebaseDeliveryOutcome{Code: "firebase_unavailable", Message: "Firebase delivery unavailable", Retryable: true}
		} else if err := s.DB.First(&device, "id = ? AND user_id = ? AND notifications_enabled = ?", digest.FirebaseDeviceID, eligible[0].UserID, true).Error; err != nil {
			outcome = FirebaseDeliveryOutcome{Code: "device_unavailable", Message: "digest device unavailable"}
		} else {
			route := "/notifications"
			members := make([]string, 0, len(eligible))
			for _, job := range eligible {
				members = append(members, job.ID.String())
			}
			sort.Strings(members)
			membershipHash := sha256.Sum256([]byte(strings.Join(members, ":")))
			messageID := "digest-" + digest.ID.String() + "-" + hex.EncodeToString(membershipHash[:8])
			payload := NotificationDeliveryPayload{Title: "Your MediGuide updates", Body: fmt.Sprintf("%d routine updates are ready in your inbox.", len(eligible)),
				Action:   NotificationAction{Type: NotificationActionInternalRoute, Route: &route, Parameters: map[string]string{}},
				Priority: "normal", AndroidChannel: "mediguide_updates", PublicContent: true, TTLSeconds: ttl, CollapseKey: "digest-" + digest.ID.String(), MessageID: messageID}
			outcome = s.Firebase.DeliverToDevice(ctx, device, payload, nil, false)
		}
		for _, job := range eligible {
			if err := s.RecordResult(job, outcome, now); err != nil {
				return err
			}
		}
	}
	state := "accepted"
	next := now
	if !outcome.Accepted {
		state = "failed"
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var current models.NotificationDigest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", digest.ID).Error; err != nil {
			return err
		}
		if current.Status != "processing" || notificationStringValue(current.LockedBy) != worker {
			return ErrNotificationConflict
		}
		var remaining []models.NotificationOutboxJob
		if err := tx.Select("next_attempt_at").Where("digest_id = ? AND status IN ?", digest.ID, []string{"pending", "retry"}).Order("next_attempt_at").Limit(1).Find(&remaining).Error; err != nil {
			return err
		}
		if len(remaining) > 0 {
			state = "retry"
			next = remaining[0].NextAttemptAt
			if next.Before(now) {
				next = now
			}
		}
		return tx.Model(&current).Updates(map[string]any{"status": state, "next_attempt_at": next, "locked_at": nil, "locked_by": nil}).Error
	})
	if err != nil {
		return err
	}
	for id := range campaigns {
		if err := s.refreshCampaignOutcome(id); err != nil {
			return err
		}
	}
	switch state {
	case "accepted":
		result.Accepted++
	case "retry", "pending":
		result.Retried++
	default:
		result.Failed++
	}
	return nil
}

// Scheduling/resuming a held campaign may add members to a completed window.
// Reopen only those referenced batches, keeping completed batches out of the
// worker's hot claim query as delivery history grows.
func reactivateNotificationDigests(tx *gorm.DB, campaignID uuid.UUID, now time.Time) error {
	ids := tx.Model(&models.NotificationOutboxJob{}).Select("digest_id").Where("campaign_id = ? AND digest_id IS NOT NULL AND status IN ?", campaignID, []string{"pending", "retry"})
	return tx.Model(&models.NotificationDigest{}).Where("id IN (?) AND status IN ?", ids, []string{"accepted", "failed"}).
		Updates(map[string]any{"status": "pending", "attempt_count": 0, "next_attempt_at": gorm.Expr("CASE WHEN window_at > ? THEN window_at ELSE ? END", now, now), "locked_at": nil, "locked_by": nil}).Error
}
