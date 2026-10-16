package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrGuidelineWithdrawalReason    = errors.New("a withdrawal reason of 1 to 2000 characters is required")
	ErrGuidelinePublicationConflict = errors.New("the document publication has changed; refresh and try again")
)

type WithdrawGuidelineInput struct {
	Reason           string    `json:"reason" binding:"required"`
	CurrentVersionID uuid.UUID `json:"current_version_id" binding:"required"`
}

// WithdrawDocument removes the public publication link, leaving the approved
// version immutable. Every public reader, search and RAG query requires that link.
func (s GuidelineService) WithdrawDocument(id, actorID uuid.UUID, in WithdrawGuidelineInput, ip string) error {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 {
		return ErrGuidelineWithdrawalReason
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var document models.GuidelineDocument
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&document, "id = ?", id).Error; err != nil {
			return err
		}
		if document.CurrentVersionID == nil || *document.CurrentVersionID != in.CurrentVersionID {
			return ErrGuidelinePublicationConflict
		}
		var version models.GuidelineVersion
		if err := tx.First(&version, "id = ? AND document_id = ? AND status = ?", in.CurrentVersionID, id, "published").Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGuidelinePublicationConflict
			}
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&document).Updates(map[string]any{
			"current_version_id": nil, "withdrawn_version_id": version.ID,
			"withdrawn_at": now, "withdrawn_by": actorID, "withdrawal_reason": reason,
		}).Error; err != nil {
			return err
		}
		return writeGuidelineAudit(tx, actorID, "guideline.document.withdrawn", "guideline_document", id, ip, map[string]any{
			"version_id": version.ID, "reason": reason, "withdrawn_at": now,
		})
	})
	if err == nil {
		s.invalidatePublishedCaches(context.Background())
	}
	return err
}
