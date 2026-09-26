package models

import (
	"gorm.io/datatypes"
	"time"

	"github.com/google/uuid"
)

type IngestionJob struct {
	Base
	MetricsJSON       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"metrics" swaggertype:"object"`
	VersionID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"version_id"`
	JobType           string         `gorm:"default:'pdf_ingestion'" json:"job_type"`
	Status            string         `gorm:"default:'queued';index" json:"status"`
	Error             string         `gorm:"type:text" json:"error,omitempty"`
	PayloadJSON       string         `gorm:"type:jsonb" json:"payload_json,omitempty"`
	AttemptCount      int            `gorm:"default:0" json:"attempt_count"`
	Priority          int            `gorm:"not null;default:50" json:"priority"`
	WorkerID          string         `json:"worker_id,omitempty"`
	ClaimedAt         *time.Time     `json:"claimed_at,omitempty"`
	HeartbeatAt       *time.Time     `json:"heartbeat_at,omitempty"`
	LeaseExpiresAt    *time.Time     `json:"lease_expires_at,omitempty"`
	NextAttemptAt     *time.Time     `json:"next_attempt_at,omitempty"`
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	ProgressStage     string         `gorm:"not null;default:'queued'" json:"progress_stage"`
	ProgressPercent   int            `gorm:"not null;default:0" json:"progress_percent"`
	CancelRequestedAt *time.Time     `json:"cancel_requested_at,omitempty"`
	CanceledAt        *time.Time     `json:"canceled_at,omitempty"`
	Stages            []IngestionTask `gorm:"-" json:"stages,omitempty"`
}

type IngestionTask struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	JobID           uuid.UUID  `gorm:"type:uuid;index;not null" json:"job_id"`
	Stage           string     `json:"stage"`
	Status          string     `json:"status"`
	ProgressPercent int        `json:"progress_percent"`
	AttemptCount    int        `json:"attempt_count"`
	Error           string     `json:"-"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (IngestionTask) TableName() string { return "ingestion_tasks" }
