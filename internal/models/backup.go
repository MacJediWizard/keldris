package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BackupStatus represents the current status of a backup.
type BackupStatus string

const (
	// BackupStatusRunning indicates the backup is in progress.
	BackupStatusRunning BackupStatus = "running"
	// BackupStatusCompleted indicates the backup completed successfully.
	BackupStatusCompleted BackupStatus = "completed"
	// BackupStatusFailed indicates the backup failed.
	BackupStatusFailed BackupStatus = "failed"
	// BackupStatusCanceled indicates the backup was canceled.
	BackupStatusCanceled BackupStatus = "canceled"
)

// Backup represents a single backup execution record.
type Backup struct {
	ID                      uuid.UUID    `json:"id"`
	ScheduleID              uuid.UUID    `json:"schedule_id"`
	AgentID                 uuid.UUID    `json:"agent_id"`
	RepositoryID            *uuid.UUID   `json:"repository_id,omitempty"`
	SnapshotID              string       `json:"snapshot_id,omitempty"`
	StartedAt               time.Time    `json:"started_at"`
	CompletedAt             *time.Time   `json:"completed_at,omitempty"`
	Status                  BackupStatus `json:"status"`
	SizeBytes               *int64       `json:"size_bytes,omitempty"`
	FilesNew                *int         `json:"files_new,omitempty"`
	FilesChanged            *int         `json:"files_changed,omitempty"`
	ErrorMessage            string       `json:"error_message,omitempty"`
	RetentionApplied        bool         `json:"retention_applied"`
	SnapshotsRemoved        *int         `json:"snapshots_removed,omitempty"`
	SnapshotsKept           *int         `json:"snapshots_kept,omitempty"`
	RetentionError          string       `json:"retention_error,omitempty"`
	PreScriptOutput         string       `json:"pre_script_output,omitempty"`
	PreScriptError          string       `json:"pre_script_error,omitempty"`
	PostScriptOutput        string       `json:"post_script_output,omitempty"`
	PostScriptError         string       `json:"post_script_error,omitempty"`
	ClassificationLevel     string       `json:"classification_level,omitempty"`      // Data classification level
	ClassificationDataTypes []string     `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	CreatedAt               time.Time    `json:"created_at"`
}

// NewBackup creates a new Backup record for the given schedule, agent, and repository.
func NewBackup(scheduleID, agentID uuid.UUID, repositoryID *uuid.UUID) *Backup {
	now := time.Now()
	return &Backup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: repositoryID,
		StartedAt:    now,
		Status:       BackupStatusRunning,
		CreatedAt:    now,
	}
}

// Complete marks the backup as completed with the given stats.
func (b *Backup) Complete(snapshotID string, sizeBytes int64, filesNew, filesChanged int) {
	now := time.Now()
	b.CompletedAt = &now
	b.Status = BackupStatusCompleted
	b.SnapshotID = snapshotID
	b.SizeBytes = &sizeBytes
	b.FilesNew = &filesNew
	b.FilesChanged = &filesChanged
}

// Fail marks the backup as failed with the given error message.
func (b *Backup) Fail(errMsg string) {
	now := time.Now()
	b.CompletedAt = &now
	b.Status = BackupStatusFailed
	b.ErrorMessage = errMsg
}

// Cancel marks the backup as canceled.
func (b *Backup) Cancel() {
	now := time.Now()
	b.CompletedAt = &now
	b.Status = BackupStatusCanceled
}

// RecordRetention records the results of retention policy enforcement.
func (b *Backup) RecordRetention(removed, kept int, err error) {
	b.RetentionApplied = true
	b.SnapshotsRemoved = &removed
	b.SnapshotsKept = &kept
	if err != nil {
		b.RetentionError = err.Error()
	}
}

// Duration returns the duration of the backup, or zero if not completed.
func (b *Backup) Duration() time.Duration {
	if b.CompletedAt == nil {
		return 0
	}
	return b.CompletedAt.Sub(b.StartedAt)
}

// IsComplete returns true if the backup has finished (success, failure, or canceled).
func (b *Backup) IsComplete() bool {
	return b.Status == BackupStatusCompleted ||
		b.Status == BackupStatusFailed ||
		b.Status == BackupStatusCanceled
}

// RecordPreScript records the results of running a pre-backup script.
func (b *Backup) RecordPreScript(output string, err error) {
	b.PreScriptOutput = output
	if err != nil {
		b.PreScriptError = err.Error()
	}
}

// RecordPostScript records the results of running a post-backup script.
func (b *Backup) RecordPostScript(output string, err error) {
	b.PostScriptOutput = output
	if err != nil {
		b.PostScriptError = err.Error()
	}
}

// SetClassificationDataTypes sets the classification data types from JSON bytes.
func (b *Backup) SetClassificationDataTypes(data []byte) error {
	if len(data) == 0 {
		b.ClassificationDataTypes = []string{"general"}
		return nil
	}
	return json.Unmarshal(data, &b.ClassificationDataTypes)
}

// ClassificationDataTypesJSON returns the classification data types as JSON bytes.
func (b *Backup) ClassificationDataTypesJSON() ([]byte, error) {
	if b.ClassificationDataTypes == nil || len(b.ClassificationDataTypes) == 0 {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(b.ClassificationDataTypes)
}
