package models

import (
	"github.com/google/uuid"
	"time"
)

// GuidelineUpload retains the source identity across browser refreshes and retries.
type GuidelineUpload struct {
	Base
	VersionID      uuid.UUID  `gorm:"type:uuid" json:"version_id"`
	UserID         uuid.UUID  `gorm:"type:uuid" json:"-"`
	Filename       string     `json:"filename"`
	SizeBytes      int64      `json:"size_bytes"`
	Checksum       string     `json:"checksum"`
	ObjectKey      string     `json:"-"`
	MultipartID    string     `json:"-"`
	PartSize       int64      `json:"part_size"`
	Status         string     `json:"status"`
	NotifiedStatus string     `json:"-"`
	JobID          *uuid.UUID `gorm:"type:uuid" json:"job_id,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
}
