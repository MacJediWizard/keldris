package models

import (
	"time"

	"github.com/google/uuid"
)

// BackupScriptType represents when the script should run.
type BackupScriptType string

const (
	// BackupScriptTypePreBackup runs before the backup starts.
	BackupScriptTypePreBackup BackupScriptType = "pre_backup"
	// BackupScriptTypePostSuccess runs after a successful backup.
	BackupScriptTypePostSuccess BackupScriptType = "post_success"
	// BackupScriptTypePostFailure runs after a failed backup.
	BackupScriptTypePostFailure BackupScriptType = "post_failure"
	// BackupScriptTypePostAlways runs after backup regardless of outcome.
	BackupScriptTypePostAlways BackupScriptType = "post_always"
)

// ValidBackupScriptTypes returns all valid script types.
func ValidBackupScriptTypes() []BackupScriptType {
	return []BackupScriptType{
		BackupScriptTypePreBackup,
		BackupScriptTypePostSuccess,
		BackupScriptTypePostFailure,
		BackupScriptTypePostAlways,
	}
}

// IsValid checks if the script type is valid.
func (t BackupScriptType) IsValid() bool {
	for _, valid := range ValidBackupScriptTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// BackupScript represents a script to run before or after a backup.
type BackupScript struct {
	ID             uuid.UUID        `json:"id"`
	ScheduleID     uuid.UUID        `json:"schedule_id"`
	Type           BackupScriptType `json:"type"`
	Script         string           `json:"script"`
	TimeoutSeconds int              `json:"timeout_seconds"`
	FailOnError    bool             `json:"fail_on_error"`
	Enabled        bool             `json:"enabled"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// NewBackupScript creates a new BackupScript with the given details.
func NewBackupScript(scheduleID uuid.UUID, scriptType BackupScriptType, script string) *BackupScript {
	now := time.Now()
	return &BackupScript{
		ID:             uuid.New(),
		ScheduleID:     scheduleID,
		Type:           scriptType,
		Script:         script,
		TimeoutSeconds: 300, // Default 5 minutes
		FailOnError:    false,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsPreBackup returns true if this is a pre-backup script.
func (s *BackupScript) IsPreBackup() bool {
	return s.Type == BackupScriptTypePreBackup
}

// IsPostScript returns true if this is any post-backup script type.
func (s *BackupScript) IsPostScript() bool {
	return s.Type == BackupScriptTypePostSuccess ||
		s.Type == BackupScriptTypePostFailure ||
		s.Type == BackupScriptTypePostAlways
}

// ShouldRunOnSuccess returns true if this script should run after successful backup.
func (s *BackupScript) ShouldRunOnSuccess() bool {
	return s.Type == BackupScriptTypePostSuccess || s.Type == BackupScriptTypePostAlways
}

// ShouldRunOnFailure returns true if this script should run after failed backup.
func (s *BackupScript) ShouldRunOnFailure() bool {
	return s.Type == BackupScriptTypePostFailure || s.Type == BackupScriptTypePostAlways
}

// CreateBackupScriptRequest is the request body for creating a backup script.
type CreateBackupScriptRequest struct {
	Type           BackupScriptType `json:"type" binding:"required"`
	Script         string           `json:"script" binding:"required"`
	TimeoutSeconds *int             `json:"timeout_seconds,omitempty"`
	FailOnError    *bool            `json:"fail_on_error,omitempty"`
	Enabled        *bool            `json:"enabled,omitempty"`
}

// UpdateBackupScriptRequest is the request body for updating a backup script.
type UpdateBackupScriptRequest struct {
	Script         *string `json:"script,omitempty"`
	TimeoutSeconds *int    `json:"timeout_seconds,omitempty"`
	FailOnError    *bool   `json:"fail_on_error,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// ScriptExecution represents the result of running a script.
type ScriptExecution struct {
	Output   string
	Error    error
	Duration time.Duration
	ExitCode int
}
