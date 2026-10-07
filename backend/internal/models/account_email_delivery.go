package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// AccountEmailDelivery retains only an encrypted token, never a plaintext link.
type AccountEmailDelivery struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Purpose        string    `gorm:"not null"`
	EncryptedToken string    `gorm:"not null"`
	Status         string    `gorm:"not null;index"`
	Attempts       int
	NextAttemptAt  time.Time `gorm:"index"`
	LeaseUntil     *time.Time
	SentAt         *time.Time
	LastErrorCode  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (d *AccountEmailDelivery) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
