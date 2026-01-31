package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BackupValidationStatus represents the current status of a backup validation.
type BackupValidationStatus string

const (
	// BackupValidationStatusPending indicates the validation is queued.
	BackupValidationStatusPending BackupValidationStatus = "pending"
	// BackupValidationStatusRunning indicates the validation is in progress.
	BackupValidationStatusRunning BackupValidationStatus = "running"
	// BackupValidationStatusPassed indicates the validation passed.
	BackupValidationStatusPassed BackupValidationStatus = "passed"
	// BackupValidationStatusFailed indicates the validation failed.
	BackupValidationStatusFailed BackupValidationStatus = "failed"
	// BackupValidationStatusSkipped indicates the validation was skipped.
	BackupValidationStatusSkipped BackupValidationStatus = "skipped"
)

// BackupValidation represents a validation run for a backup.
type BackupValidation struct {
	ID           uuid.UUID              `json:"id"`
	BackupID     uuid.UUID              `json:"backup_id"`
	RepositoryID uuid.UUID              `json:"repository_id"`
	SnapshotID   string                 `json:"snapshot_id"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Status       BackupValidationStatus `json:"status"`
	DurationMs   *int64                 `json:"duration_ms,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Details      *BackupValidationDetails `json:"details,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// BackupValidationDetails contains detailed results of the backup validation.
type BackupValidationDetails struct {
	// Snapshot verification
	SnapshotExists       bool   `json:"snapshot_exists"`
	SnapshotVerified     bool   `json:"snapshot_verified"`
	SnapshotErrorMessage string `json:"snapshot_error_message,omitempty"`

	// Metadata validation
	MetadataValid        bool     `json:"metadata_valid"`
	MetadataErrors       []string `json:"metadata_errors,omitempty"`

	// File count comparison
	FileCountValid       bool  `json:"file_count_valid"`
	ExpectedFileCount    int   `json:"expected_file_count"`
	ActualFileCount      int   `json:"actual_file_count"`
	FileCountDifference  int   `json:"file_count_difference"`
	FileCountErrorMargin float64 `json:"file_count_error_margin"` // percentage

	// Spot check results
	SpotCheckPerformed   bool                   `json:"spot_check_performed"`
	SpotCheckFilesTotal  int                    `json:"spot_check_files_total"`
	SpotCheckFilesPassed int                    `json:"spot_check_files_passed"`
	SpotCheckFilesFailed int                    `json:"spot_check_files_failed"`
	SpotCheckResults     []SpotCheckFileResult  `json:"spot_check_results,omitempty"`

	// Repository integrity check (optional, runs after validation)
	IntegrityCheckRun    bool   `json:"integrity_check_run"`
	IntegrityCheckPassed bool   `json:"integrity_check_passed"`
	IntegrityCheckError  string `json:"integrity_check_error,omitempty"`
}

// SpotCheckFileResult represents the result of validating a single file.
type SpotCheckFileResult struct {
	Path          string `json:"path"`
	ExpectedSize  int64  `json:"expected_size"`
	ActualSize    int64  `json:"actual_size"`
	SizeMatch     bool   `json:"size_match"`
	ExistsInSnap  bool   `json:"exists_in_snapshot"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// NewBackupValidation creates a new BackupValidation record.
func NewBackupValidation(backupID, repositoryID uuid.UUID, snapshotID string) *BackupValidation {
	now := time.Now()
	return &BackupValidation{
		ID:           uuid.New(),
		BackupID:     backupID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		StartedAt:    now,
		Status:       BackupValidationStatusRunning,
		CreatedAt:    now,
	}
}

// Pass marks the validation as passed.
func (bv *BackupValidation) Pass(details *BackupValidationDetails) {
	now := time.Now()
	bv.CompletedAt = &now
	bv.Status = BackupValidationStatusPassed
	durationMs := now.Sub(bv.StartedAt).Milliseconds()
	bv.DurationMs = &durationMs
	bv.Details = details
}

// Fail marks the validation as failed with the given error message.
func (bv *BackupValidation) Fail(errMsg string, details *BackupValidationDetails) {
	now := time.Now()
	bv.CompletedAt = &now
	bv.Status = BackupValidationStatusFailed
	durationMs := now.Sub(bv.StartedAt).Milliseconds()
	bv.DurationMs = &durationMs
	bv.ErrorMessage = errMsg
	bv.Details = details
}

// Skip marks the validation as skipped with the given reason.
func (bv *BackupValidation) Skip(reason string) {
	now := time.Now()
	bv.CompletedAt = &now
	bv.Status = BackupValidationStatusSkipped
	durationMs := now.Sub(bv.StartedAt).Milliseconds()
	bv.DurationMs = &durationMs
	bv.ErrorMessage = reason
}

// Duration returns the duration of the validation, or zero if not completed.
func (bv *BackupValidation) Duration() time.Duration {
	if bv.CompletedAt == nil {
		return 0
	}
	return bv.CompletedAt.Sub(bv.StartedAt)
}

// IsComplete returns true if the validation has finished.
func (bv *BackupValidation) IsComplete() bool {
	return bv.Status == BackupValidationStatusPassed ||
		bv.Status == BackupValidationStatusFailed ||
		bv.Status == BackupValidationStatusSkipped
}

// SetDetails sets the details from JSON bytes.
func (bv *BackupValidation) SetDetails(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var details BackupValidationDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return err
	}
	bv.Details = &details
	return nil
}

// DetailsJSON returns the details as JSON bytes for database storage.
func (bv *BackupValidation) DetailsJSON() ([]byte, error) {
	if bv.Details == nil {
		return nil, nil
	}
	return json.Marshal(bv.Details)
}

// Summary returns a brief summary of the validation result.
func (bv *BackupValidation) Summary() string {
	if bv.Details == nil {
		return bv.Status.String()
	}

	d := bv.Details
	if !d.SnapshotExists {
		return "Snapshot not found in repository"
	}
	if !d.MetadataValid {
		return "Snapshot metadata validation failed"
	}
	if !d.FileCountValid {
		return "File count mismatch detected"
	}
	if d.SpotCheckPerformed && d.SpotCheckFilesFailed > 0 {
		return "Spot check found file integrity issues"
	}
	if d.IntegrityCheckRun && !d.IntegrityCheckPassed {
		return "Repository integrity check failed"
	}
	return "All validation checks passed"
}

// String returns the string representation of the status.
func (s BackupValidationStatus) String() string {
	return string(s)
}
