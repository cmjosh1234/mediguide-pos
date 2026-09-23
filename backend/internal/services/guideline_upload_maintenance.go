package services

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mediguide/internal/models"
	"time"
)

// MaintainSourceUploads is safe across replicas: aborts and notification
// creation serialize on the session row. It never removes attached sources.
func (s GuidelineService) MaintainSourceUploads(ctx context.Context) error {
	var failures []error
	var expired []models.GuidelineUpload
	if err := s.DB.WithContext(ctx).Where("status='uploading' AND expires_at < ?", time.Now().UTC()).Limit(100).Find(&expired).Error; err != nil {
		return err
	}
	for _, row := range expired {
		if err := s.AbortSourceUpload(ctx, row.VersionID, row.UserID, row.ID); err != nil {
			failures = append(failures, err)
		}
	}
	var completed []models.GuidelineUpload
	if err := s.DB.WithContext(ctx).Where("status='completed' AND job_id IS NOT NULL").Where("EXISTS (SELECT 1 FROM ingestion_jobs j WHERE j.id=guideline_uploads.job_id AND j.status IN ('completed','failed','canceled','superseded') AND j.status<>guideline_uploads.notified_status)").Limit(100).Find(&completed).Error; err != nil {
		return err
	}
	for _, candidate := range completed {
		if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var row models.GuidelineUpload
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id=?", candidate.ID).Error; err != nil {
				return err
			}
			var job models.IngestionJob
			if err := tx.First(&job, "id=?", row.JobID).Error; err != nil {
				return err
			}
			if row.NotifiedStatus == job.Status {
				return nil
			}
			if job.Status != "completed" && job.Status != "failed" && job.Status != "canceled" && job.Status != "superseded" {
				return nil
			}
			title := "Document processing finished"
			kind := "info"
			message := fmt.Sprintf("%s: processing %s. Open its guideline version for details.", row.Filename, job.Status)
			if job.Status == "completed" {
				title = "Document ready for review"
				kind = "success"
				message = row.Filename + " is ready for editorial review."
			}
			if job.Status == "failed" {
				title = "Document processing failed"
				kind = "error"
			}
			user := row.UserID.String()
			key := "guideline-upload:" + row.ID.String() + ":" + job.Status
			if _, err := (NotificationService{DB: tx}).Create(NotificationInput{UserID: &user, Title: title, Message: message, Type: kind, Priority: "normal", DeduplicationKey: &key}); err != nil {
				return err
			}
			return tx.Model(&row).Update("notified_status", job.Status).Error
		}); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
