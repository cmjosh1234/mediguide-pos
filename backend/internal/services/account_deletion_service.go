package services

import (
	"errors"
	"strings"

	"mediguide/internal/models"
	"mediguide/internal/security"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const AccountDeletionCategory = "account_deletion"

var ErrDeletionCredentials = errors.New("unable to verify account credentials")

// AccountDeletionResult acknowledges a request, never completed erasure.
type AccountDeletionResult struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
}

// RequestAccountDeletion requires password reauthentication. A non-nil actor
// binds the request to the authenticated principal, never an input user ID.
func (s AuthService) RequestAccountDeletion(actor *uuid.UUID, email, password string) (*AccountDeletionResult, error) {
	var result AccountDeletionResult
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"})
		if actor != nil {
			query = query.Where("id = ?", *actor)
		} else {
			query = query.Where("lower(email) = ?", strings.ToLower(strings.TrimSpace(email)))
		}
		if err := query.First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeletionCredentials
			}
			return err
		}
		if !user.IsActive || !security.CheckPassword(user.PasswordHash, password) {
			return ErrDeletionCredentials
		}
		var existing models.SupportTicket
		err := tx.Where("user_id = ? AND category = ? AND status IN ?", user.ID,
			AccountDeletionCategory, []string{"open", "in_progress"}).
			Order("created_at DESC").First(&existing).Error
		if err == nil {
			result = AccountDeletionResult{RequestID: existing.ID, Status: "requested"}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		category := AccountDeletionCategory
		ticket := models.SupportTicket{
			UserID: &user.ID, Category: &category, Priority: "high", Status: "open",
			Subject:     "Account and associated data deletion request",
			Description: "The account holder confirmed this request using their current password. Please process account and associated data deletion under the approved retention procedure. This ticket records a request, not completed deletion.",
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.AuditLog{
			ActorID: user.ID.String(), Action: "user.deletion_requested",
			EntityType: "support_ticket", EntityID: ticket.ID.String(), MetadataJSON: "{}",
		}).Error; err != nil {
			return err
		}
		result = AccountDeletionResult{RequestID: ticket.ID, Status: "requested"}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
