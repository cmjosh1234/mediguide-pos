package services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mediguide/internal/models"
)

// Import health is separate from delivery health: a Firebase export outage must
// never stop pushes or cause accepted sends to be counted as delivered.
type NotificationImportHealth struct {
	mu          sync.RWMutex
	LastSuccess time.Time
	Failures    uint64
}

func (h *NotificationImportHealth) Record(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err == nil {
		h.LastSuccess = time.Now().UTC()
	} else {
		h.Failures++
	}
}
func (s NotificationOutboxService) MetricsHandler(importEnabled bool, health *NotificationImportHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		var pending, failed int64
		var oldest struct {
			NextAttemptAt       *time.Time
			DigestNextAttemptAt *time.Time
		}
		db := s.DB.WithContext(ctx)
		due := db.Model(&models.NotificationOutboxJob{}).
			Joins("LEFT JOIN notification_digests d ON d.id = notification_outbox_jobs.digest_id").
			Where("notification_outbox_jobs.status IN ('pending','retry') AND notification_outbox_jobs.next_attempt_at <= ?", s.now()).
			Where("d.id IS NULL OR d.next_attempt_at <= ?", s.now())
		if err := due.Count(&pending).Error; err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		if err := due.Select("notification_outbox_jobs.next_attempt_at, d.next_attempt_at AS digest_next_attempt_at").
			Order("CASE WHEN d.next_attempt_at > notification_outbox_jobs.next_attempt_at THEN d.next_attempt_at ELSE notification_outbox_jobs.next_attempt_at END").Limit(1).Find(&oldest).Error; err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		if err := db.Model(&models.NotificationOutboxJob{}).Where("status = 'failed' AND completed_at >= ?", s.now().Add(-24*time.Hour)).
			Where("last_error_code NOT IN ?", []string{"preference_disabled", "interest_disabled", "device_unavailable", "expired", "campaign_cancelled"}).Count(&failed).Error; err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		if oldest.DigestNextAttemptAt != nil && (oldest.NextAttemptAt == nil || oldest.DigestNextAttemptAt.After(*oldest.NextAttemptAt)) {
			oldest.NextAttemptAt = oldest.DigestNextAttemptAt
		}
		var publicationDue int64
		var publication struct{ NextAttemptAt *time.Time }
		publicationQuery := db.Model(&models.NotificationTopicJob{}).Where("status IN ('pending','retry') AND next_attempt_at <= ?", s.now())
		if err := publicationQuery.Count(&publicationDue).Error; err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		if err := publicationQuery.Select("next_attempt_at").Order("next_attempt_at").Limit(1).Find(&publication).Error; err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		pending += publicationDue
		if publication.NextAttemptAt != nil && (oldest.NextAttemptAt == nil || publication.NextAttemptAt.Before(*oldest.NextAttemptAt)) {
			oldest.NextAttemptAt = publication.NextAttemptAt
		}
		firebaseConfigured := 0
		if s.Firebase != nil && s.Firebase.Enabled() {
			firebaseConfigured = 1
		}
		age := float64(0)
		if oldest.NextAttemptAt != nil {
			age = s.now().Sub(*oldest.NextAttemptAt).Seconds()
			if age < 0 {
				age = 0
			}
		}
		enabled := 0
		if importEnabled {
			enabled = 1
		}
		health.mu.RLock()
		last := health.LastSuccess
		failures := health.Failures
		health.mu.RUnlock()
		timestamp := int64(0)
		if !last.IsZero() {
			timestamp = last.Unix()
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# TYPE mediguide_notification_due_jobs gauge\nmediguide_notification_due_jobs %d\n# TYPE mediguide_notification_oldest_due_seconds gauge\nmediguide_notification_oldest_due_seconds %.0f\n# TYPE mediguide_notification_failed_24h gauge\nmediguide_notification_failed_24h %d\n# TYPE mediguide_notification_firebase_configured gauge\nmediguide_notification_firebase_configured %d\n# TYPE mediguide_notification_delivery_import_enabled gauge\nmediguide_notification_delivery_import_enabled %d\n# TYPE mediguide_notification_delivery_import_last_success_seconds gauge\nmediguide_notification_delivery_import_last_success_seconds %d\n# TYPE mediguide_notification_delivery_import_failures_total counter\nmediguide_notification_delivery_import_failures_total %d\n", pending, age, failed, firebaseConfigured, enabled, timestamp, failures)
	}
}
