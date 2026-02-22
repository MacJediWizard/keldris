package models

import (
	"encoding/json"
	"time"

	pkgmodels "github.com/MacJediWizard/keldris/pkg/models"
	"github.com/google/uuid"
)

// BackupStatus is a type alias for the shared BackupStatus type in pkg/models.
type BackupStatus = pkgmodels.BackupStatus

const (
	BackupStatusRunning   = pkgmodels.BackupStatusRunning
	BackupStatusCompleted = pkgmodels.BackupStatusCompleted
	BackupStatusFailed    = pkgmodels.BackupStatusFailed
	BackupStatusCanceled  = pkgmodels.BackupStatusCanceled
)

// Backup represents a single backup execution record.
type Backup struct {
	ID               uuid.UUID    `json:"id"`
	ScheduleID       uuid.UUID    `json:"schedule_id"`
	AgentID          uuid.UUID    `json:"agent_id"`
	RepositoryID     *uuid.UUID   `json:"repository_id,omitempty"`
	SnapshotID       string       `json:"snapshot_id,omitempty"`
	StartedAt        time.Time    `json:"started_at"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	Status           BackupStatus `json:"status"`
	SizeBytes        *int64       `json:"size_bytes,omitempty"`
	FilesNew         *int         `json:"files_new,omitempty"`
	FilesChanged     *int         `json:"files_changed,omitempty"`
	ErrorMessage     string       `json:"error_message,omitempty"`
	RetentionApplied bool         `json:"retention_applied"`
	SnapshotsRemoved *int         `json:"snapshots_removed,omitempty"`
	SnapshotsKept    *int         `json:"snapshots_kept,omitempty"`
	RetentionError   string       `json:"retention_error,omitempty"`
	PreScriptOutput  string       `json:"pre_script_output,omitempty"`
	PreScriptError   string       `json:"pre_script_error,omitempty"`
	PostScriptOutput   string              `json:"post_script_output,omitempty"`
	PostScriptError    string              `json:"post_script_error,omitempty"`
	ExcludedLargeFiles []ExcludedLargeFile `json:"excluded_large_files,omitempty"`
	Resumed            bool                `json:"resumed"`
	CheckpointID       *uuid.UUID          `json:"checkpoint_id,omitempty"`
	OriginalBackupID        *uuid.UUID          `json:"original_backup_id,omitempty"`
	ClassificationLevel     string              `json:"classification_level,omitempty"`
	ClassificationDataTypes []string            `json:"classification_data_types,omitempty"`
	ContainerPreHookOutput  string              `json:"container_pre_hook_output,omitempty"`
	ContainerPreHookError   string              `json:"container_pre_hook_error,omitempty"`
	ContainerPostHookOutput string              `json:"container_post_hook_output,omitempty"`
	ContainerPostHookError  string              `json:"container_post_hook_error,omitempty"`
	CreatedAt               time.Time           `json:"created_at"`
	DeletedAt               *time.Time          `json:"deleted_at,omitempty"`
}

// ExcludedLargeFile represents a file excluded from backup due to size.
type ExcludedLargeFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeMB    int64  `json:"size_mb"`
}

// Backup represents a single backup execution record.
type Backup struct {
	ID                      uuid.UUID           `json:"id"`
	ScheduleID              uuid.UUID           `json:"schedule_id"`
	AgentID                 uuid.UUID           `json:"agent_id"`
	RepositoryID            *uuid.UUID          `json:"repository_id,omitempty"`
	SnapshotID              string              `json:"snapshot_id,omitempty"`
	StartedAt               time.Time           `json:"started_at"`
	CompletedAt             *time.Time          `json:"completed_at,omitempty"`
	Status                  BackupStatus        `json:"status"`
	BackupType              BackupType          `json:"backup_type"`                    // Type of backup: files, pihole
	SizeBytes               *int64              `json:"size_bytes,omitempty"`
	FilesNew                *int                `json:"files_new,omitempty"`
	FilesChanged            *int                `json:"files_changed,omitempty"`
	ErrorMessage            string              `json:"error_message,omitempty"`
	RetentionApplied        bool                `json:"retention_applied"`
	SnapshotsRemoved        *int                `json:"snapshots_removed,omitempty"`
	SnapshotsKept           *int                `json:"snapshots_kept,omitempty"`
	RetentionError          string              `json:"retention_error,omitempty"`
	PreScriptOutput         string              `json:"pre_script_output,omitempty"`
	PreScriptError          string              `json:"pre_script_error,omitempty"`
	PostScriptOutput        string              `json:"post_script_output,omitempty"`
	PostScriptError         string              `json:"post_script_error,omitempty"`
	ExcludedLargeFiles      []ExcludedLargeFile `json:"excluded_large_files,omitempty"` // Files excluded due to size limit
	Resumed                 bool                `json:"resumed"`                        // True if this backup was resumed from a checkpoint
	CheckpointID            *uuid.UUID          `json:"checkpoint_id,omitempty"`        // Associated checkpoint if resumed
	OriginalBackupID        *uuid.UUID          `json:"original_backup_id,omitempty"`   // Original backup that was interrupted
	ClassificationLevel       string              `json:"classification_level,omitempty"` // Data classification level
	ClassificationDataTypes   []string            `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	ContainerPreHookOutput    string              `json:"container_pre_hook_output,omitempty"`
	ContainerPreHookError     string              `json:"container_pre_hook_error,omitempty"`
	ContainerPostHookOutput   string              `json:"container_post_hook_output,omitempty"`
	ContainerPostHookError    string              `json:"container_post_hook_error,omitempty"`
	// Validation fields
	ValidationID              *uuid.UUID          `json:"validation_id,omitempty"`     // ID of the associated validation record
	ValidationStatus          string              `json:"validation_status,omitempty"` // Status: passed, failed, skipped, pending
	ValidationError           string              `json:"validation_error,omitempty"`  // Error message if validation failed
	PiholeVersion             string              `json:"pihole_version,omitempty"`    // Pi-hole version at time of backup
	CreatedAt                 time.Time           `json:"created_at"`
	PostScriptOutput string       `json:"post_script_output,omitempty"`
	PostScriptError  string       `json:"post_script_error,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
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
		BackupType:   BackupTypeFiles,
		Resumed:      false,
		CreatedAt:    now,
	}
}

// NewPiholeBackup creates a new Backup record for a Pi-hole backup.
func NewPiholeBackup(scheduleID, agentID uuid.UUID, repositoryID *uuid.UUID, piholeVersion string) *Backup {
	now := time.Now()
	return &Backup{
		ID:            uuid.New(),
		ScheduleID:   scheduleID,
		AgentID:      agentID,
		RepositoryID: repositoryID,
		StartedAt:    now,
		Status:       BackupStatusRunning,
		BackupType:   BackupTypePihole,
		PiholeVersion: piholeVersion,
		Resumed:      false,
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

// SetPiholeBackup marks this backup as a Pi-hole backup.
func (b *Backup) SetPiholeBackup(isPihole bool) {
	if isPihole {
		b.BackupType = BackupTypePihole
	} else {
		b.BackupType = BackupTypeFiles
	}
}

// IsPiholeBackup returns true if this is a Pi-hole backup.
func (b *Backup) IsPiholeBackup() bool {
	return b.BackupType == BackupTypePihole
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

// Cancel marks the backup as canceled.
func (b *Backup) Cancel() {
	now := time.Now()
	b.CompletedAt = &now
	b.Status = BackupStatusCanceled
}


// RecordExcludedLargeFiles records files that were excluded due to size limits.
func (b *Backup) RecordExcludedLargeFiles(files []ExcludedLargeFile) {
	b.ExcludedLargeFiles = files
}


// NewResumedBackup creates a new Backup record that is resuming from a checkpoint.
func NewResumedBackup(scheduleID, agentID uuid.UUID, repositoryID *uuid.UUID, checkpointID uuid.UUID, originalBackupID *uuid.UUID) *Backup {
	now := time.Now()
	return &Backup{
		ID:               uuid.New(),
		ScheduleID:       scheduleID,
		AgentID:          agentID,
		RepositoryID:     repositoryID,
		StartedAt:        now,
		Status:           BackupStatusRunning,
		Resumed:          true,
		CheckpointID:     &checkpointID,
		OriginalBackupID: originalBackupID,
		CreatedAt:        now,
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
	if len(b.ClassificationDataTypes) == 0 {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(b.ClassificationDataTypes)
}

// RecordContainerPreHook records the results of running a container pre-backup hook.
func (b *Backup) RecordContainerPreHook(output string, err error) {
	b.ContainerPreHookOutput = output
	if err != nil {
		b.ContainerPreHookError = err.Error()
	}
}

// RecordContainerPostHook records the results of running a container post-backup hook.
func (b *Backup) RecordContainerPostHook(output string, err error) {
	b.ContainerPostHookOutput = output
	if err != nil {
		b.ContainerPostHookError = err.Error()
	}
}

// RecordValidation records the results of backup validation.
func (b *Backup) RecordValidation(validationID uuid.UUID, status string, errMsg string) {
	b.ValidationID = &validationID
	b.ValidationStatus = status
	b.ValidationError = errMsg
}

// IsValidated returns true if the backup has been validated.
func (b *Backup) IsValidated() bool {
	return b.ValidationID != nil
}

// ValidationPassed returns true if the backup validation passed.
func (b *Backup) ValidationPassed() bool {
	return b.ValidationStatus == "passed"
}
