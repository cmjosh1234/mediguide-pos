package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// restoreWithdrawnGuideline restores the retained publication, rather than
// ingesting or approving it again. In particular, a structured publication may
// deliberately omit raw Markdown containing unreviewed text. Neither its manifest
// nor its chunk approvals should change when visibility is restored.
func restoreWithdrawnGuideline(tx *gorm.DB, document *models.GuidelineDocument, version *models.GuidelineVersion, actorID uuid.UUID, ip string) error {
	if document.CurrentVersionID != nil {
		if *document.CurrentVersionID == version.ID && document.WithdrawnAt == nil {
			return nil // A retried successful publication is already visible.
		}
		return ErrGuidelinePublicationConflict
	}
	if document.WithdrawnAt == nil || document.WithdrawnVersionID == nil || *document.WithdrawnVersionID != version.ID {
		return ErrGuidelinePublicationConflict
	}
	if version.ApprovedBy == nil || version.ApprovedAt == nil {
		return fmt.Errorf("%w: the retained version has no publication approval; create and review a new version", ErrGuidelineValidationFailed)
	}
	if version.CurrentMarkdownRevisionID != nil && (version.PublishedMarkdownRevisionID == nil ||
		*version.CurrentMarkdownRevisionID != *version.PublishedMarkdownRevisionID ||
		version.StructuredMarkdownRevisionID == nil || *version.CurrentMarkdownRevisionID != *version.StructuredMarkdownRevisionID) {
		return fmt.Errorf("%w: the retained projection differs from the published Markdown revision; create and review a new version", ErrGuidelineValidationFailed)
	}
	var manifest models.GuidelineVersionManifest
	if err := tx.Where("version_id = ? AND guideline_id = ?", version.ID, document.ID).First(&manifest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: the retained publication manifest is missing; restore it before republishing", ErrGuidelineValidationFailed)
		}
		return err
	}
	// Recheck the clinical validation inside the publication transaction too.
	// Missing intermediate extraction files are warnings for structured output;
	// missing sources, invalid content and unreviewed high-risk blocks still fail.
	validation, err := validateGuidelinePublication(tx, version)
	if err != nil {
		return err
	}
	if !validation.Valid {
		messages := make([]string, 0, len(validation.Errors))
		for _, issue := range validation.Errors {
			messages = append(messages, issue.Message)
		}
		return fmt.Errorf("%w: %s", ErrGuidelineValidationFailed, strings.Join(messages, "; "))
	}
	if err := writeGuidelineAudit(tx, actorID, "guideline.document.republished", "guideline_document", document.ID, ip, map[string]any{
		"version_id": version.ID, "withdrawn_version_id": document.WithdrawnVersionID,
		"previous_withdrawal_reason": document.WithdrawalReason,
	}); err != nil {
		return err
	}
	return tx.Model(document).Updates(map[string]any{
		"current_version_id": version.ID, "withdrawn_version_id": nil,
		"withdrawn_at": nil, "withdrawn_by": nil, "withdrawal_reason": "",
	}).Error
}

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
