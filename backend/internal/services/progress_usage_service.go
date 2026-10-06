package services

import (
	"errors"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrProgressUsageInvalid = errors.New("invalid progress or usage payload")

type ProgressUsageService struct{ DB *gorm.DB }
type ReadingProgressQuery struct {
	Page        PageInput
	GuidelineID string
	Bookmarked  *bool
	ProgressMin *float64
	ProgressMax *float64
	Sort, Order string
}
type ReadingProgressInput struct {
	ProgressPercentage *float64 `json:"progress_percentage"`
	CurrentSection     *string  `json:"current_section"`
	LastReadAt         *string  `json:"last_read_at"`
	IsBookmarked       *bool    `json:"is_bookmarked"`
	ReadingTimeSeconds *int64   `json:"reading_time_seconds"`
	TotalSections      *int64   `json:"total_sections"`
	IsCompleted        *bool    `json:"is_completed"`
	Notes              *string  `json:"notes"`
}
type UsageEventInput struct {
	ResourceType   string  `json:"resource_type,omitempty"`
	Feature        string  `json:"feature,omitempty"`
	ResourceID     *string `json:"resource_id"`
	IdempotencyKey string  `json:"idempotency_key"`
}
type UsageAggregate struct {
	EventType string `json:"event_type"`
	Count     int64  `json:"count"`
}

func (s ProgressUsageService) ListProgress(userID uuid.UUID, in ReadingProgressQuery) (*PageResult[models.ReadingProgress], error) {
	p := in.Page.Normalize(20, 100)
	q := s.DB.Model(&models.ReadingProgress{}).Where("user_id=?", userID)
	if in.GuidelineID != "" {
		id, err := uuid.Parse(in.GuidelineID)
		if err != nil {
			return nil, ErrProgressUsageInvalid
		}
		q = q.Where("guideline_document_id=?", id)
	}
	if in.Bookmarked != nil {
		q = q.Where("is_bookmarked=?", *in.Bookmarked)
	}
	if in.ProgressMin != nil {
		q = q.Where("progress_percentage>=?", *in.ProgressMin)
	}
	if in.ProgressMax != nil {
		q = q.Where("progress_percentage<=?", *in.ProgressMax)
	}
	return pageHelp[models.ReadingProgress](q, p, map[string]string{"last_read_at": "last_read_at", "progress_percentage": "progress_percentage", "updated_at": "updated_at"}, in.Sort, in.Order, "updated_at DESC")
}

func (s ProgressUsageService) GetProgress(userID, guidelineID uuid.UUID) (*models.ReadingProgress, error) {
	var value models.ReadingProgress
	err := s.DB.Where("user_id=? AND guideline_document_id=?", userID, guidelineID).First(&value).Error
	return &value, err
}

func (s ProgressUsageService) UpsertProgress(userID, guidelineID uuid.UUID, in ReadingProgressInput) (*models.ReadingProgress, error) {
	if err := s.requireGuidelineDocument(guidelineID); err != nil {
		return nil, err
	}
	var value models.ReadingProgress
	err := s.DB.Where("user_id=? AND guideline_document_id=?", userID, guidelineID).First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		value = models.ReadingProgress{UserID: userID, GuidelineDocumentID: guidelineID}
	} else if err != nil {
		return nil, err
	}
	if in.ProgressPercentage != nil {
		value.ProgressPercentage = *in.ProgressPercentage
	}
	if in.CurrentSection != nil {
		v := strings.TrimSpace(*in.CurrentSection)
		value.CurrentSection = &v
	}
	if in.LastReadAt != nil {
		v := strings.TrimSpace(*in.LastReadAt)
		if v != "" {
			if _, e := time.Parse(time.RFC3339, v); e != nil {
				return nil, ErrProgressUsageInvalid
			}
		}
		value.LastReadAt = &v
	}
	if in.IsBookmarked != nil {
		value.IsBookmarked = *in.IsBookmarked
	}
	if in.ReadingTimeSeconds != nil {
		value.ReadingTimeSeconds = in.ReadingTimeSeconds
	}
	if in.TotalSections != nil {
		value.TotalSections = in.TotalSections
	}
	if in.IsCompleted != nil {
		value.IsCompleted = *in.IsCompleted
	}
	if in.Notes != nil {
		v := strings.TrimSpace(*in.Notes)
		value.Notes = &v
	}
	if value.ProgressPercentage >= 1 {
		value.IsCompleted = true
	}
	if value.ProgressPercentage < 0 || value.ProgressPercentage > 1 || (value.ReadingTimeSeconds != nil && *value.ReadingTimeSeconds < 0) || (value.TotalSections != nil && *value.TotalSections < 0) {
		return nil, ErrProgressUsageInvalid
	}
	if err := s.DB.Save(&value).Error; err != nil {
		return nil, err
	}
	return &value, nil
}

func (s ProgressUsageService) DeleteProgress(userID, guidelineID uuid.UUID) error {
	result := s.DB.Where("user_id=? AND guideline_document_id=?", userID, guidelineID).Delete(&models.ReadingProgress{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s ProgressUsageService) RecordUsage(userID uuid.UUID, eventType string, in UsageEventInput) (any, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" || len(key) > 128 {
		return nil, ErrProgressUsageInvalid
	}
	if eventType == "feature" {
		allowed := map[string]bool{"home": true, "guidelines": true, "drugs": true, "facilities": true, "ministry_directory": true, "abbreviations": true, "ai": true, "tools": true, "documents": true, "downloads": true, "library": true, "outbreaks": true, "search": true, "notifications": true, "conversations": true, "profile": true, "support": true, "discovery": true, "more": true}
		if !allowed[in.Feature] {
			return nil, ErrProgressUsageInvalid
		}
		return createUsage(s.DB, models.FeatureUsageLog{UserID: userID, Feature: in.Feature, IdempotencyKey: &key}, userID, key, "feature", in.Feature, nil, uuid.Nil)
	}
	resourceID := uuid.Nil
	if eventType != "ai" {
		if in.ResourceID == nil {
			return nil, ErrProgressUsageInvalid
		}
		id, err := uuid.Parse(*in.ResourceID)
		if err != nil {
			return nil, ErrProgressUsageInvalid
		}
		resourceID = id
	}
	switch eventType {
	case "guideline":
		if in.ResourceType != "" && in.ResourceType != "guideline_document" {
			return nil, ErrProgressUsageInvalid
		}
		if err := s.requireGuidelineDocument(resourceID); err != nil {
			return nil, err
		}
		return createUsage(s.DB, models.GuidelineUsageLog{UserID: userID, GuidelineDocumentID: resourceID, IdempotencyKey: &key}, userID, key, "guideline_document_id", resourceID, nil, uuid.Nil)
	case "abbreviation":
		if err := requireUsageResource(s.DB, &models.Abbreviation{}, resourceID); err != nil {
			return nil, err
		}
		return createUsage(s.DB, models.AbbreviationUsageLog{UserID: userID, AbbreviationID: resourceID, IdempotencyKey: &key}, userID, key, "abbreviation_id", resourceID, &models.Abbreviation{}, resourceID)
	case "ai":
		return createUsage(s.DB, models.AIUsageLog{UserID: userID, IdempotencyKey: &key}, userID, key, "", nil, nil, uuid.Nil)
	default:
		return nil, ErrProgressUsageInvalid
	}
}

// Insert and counter update are atomic; database uniqueness handles concurrent retries.
func createUsage[T any](db *gorm.DB, value T, userID uuid.UUID, key, resourceColumn string, resource any, counter any, counterID uuid.UUID) (*T, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&value)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			lookup := tx.Where("user_id = ? AND idempotency_key = ?", userID, key)
			if resourceColumn != "" {
				lookup = lookup.Where(resourceColumn+" = ?", resource)
			}
			var existing T
			if err := lookup.First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrProgressUsageInvalid
				}
				return err
			}
			value = existing
			return nil
		}
		if counter != nil {
			return tx.Model(counter).Where("id = ?", counterID).UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func requireUsageResource(db *gorm.DB, model any, id uuid.UUID) error {
	var count int64
	if err := db.Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s ProgressUsageService) requireGuidelineDocument(id uuid.UUID) error {
	var count int64
	if err := s.DB.Model(&models.GuidelineDocument{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s ProgressUsageService) UsageAggregates(since *time.Time) ([]UsageAggregate, error) {
	rows := []UsageAggregate{}
	query := `SELECT event_type, COUNT(*) AS count FROM (
	SELECT 'guideline' event_type, created_at FROM guideline_usage_logs WHERE deleted_at IS NULL UNION ALL
 SELECT 'guideline', created_at FROM historical_guideline_usage_logs WHERE deleted_at IS NULL UNION ALL
 SELECT 'drug', created_at FROM drug_usage_logs WHERE deleted_at IS NULL UNION ALL
 SELECT 'facility', created_at FROM facility_usage_logs WHERE deleted_at IS NULL UNION ALL
 SELECT 'feature_' || feature, created_at FROM feature_usage_logs WHERE deleted_at IS NULL UNION ALL
	SELECT 'abbreviation', created_at FROM abbreviation_usage_logs WHERE deleted_at IS NULL UNION ALL
	SELECT 'ai', created_at FROM ai_usage_logs WHERE deleted_at IS NULL UNION ALL
 SELECT calculator_type, created_at FROM calculator_usage_logs WHERE deleted_at IS NULL) usage_events`
	args := []any{}
	if since != nil {
		query += " WHERE created_at >= ?"
		args = append(args, *since)
	}
	query += " GROUP BY event_type ORDER BY event_type"
	return rows, s.DB.Raw(query, args...).Scan(&rows).Error
}
