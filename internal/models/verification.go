package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// VerificationStatus represents the current status of a verification.
type VerificationStatus string

const (
	// VerificationStatusPending indicates the verification is queued.
	VerificationStatusPending VerificationStatus = "pending"
	// VerificationStatusRunning indicates the verification is in progress.
	VerificationStatusRunning VerificationStatus = "running"
	// VerificationStatusPassed indicates the verification passed.
	VerificationStatusPassed VerificationStatus = "passed"
	// VerificationStatusFailed indicates the verification failed.
	VerificationStatusFailed VerificationStatus = "failed"
)

// VerificationType defines the type of verification performed.
type VerificationType string

const (
	// VerificationTypeCheck runs restic check on the repository.
	VerificationTypeCheck VerificationType = "check"
	// VerificationTypeCheckReadData runs restic check with --read-data.
	VerificationTypeCheckReadData VerificationType = "check_read_data"
	// VerificationTypeTestRestore performs a test restore to a temp location.
	VerificationTypeTestRestore VerificationType = "test_restore"
)

// Verification represents a single verification execution record.
type Verification struct {
	ID           uuid.UUID          `json:"id"`
	RepositoryID uuid.UUID          `json:"repository_id"`
	Type         VerificationType   `json:"type"`
	SnapshotID   string             `json:"snapshot_id,omitempty"` // For test restore
	StartedAt    time.Time          `json:"started_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	Status       VerificationStatus `json:"status"`
	DurationMs   *int64             `json:"duration_ms,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
	Details      *VerificationDetails `json:"details,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
}

// VerificationDetails contains additional information about the verification.
type VerificationDetails struct {
	// For check verifications
	ErrorsFound []string `json:"errors_found,omitempty"`
	// For test restore
	FilesRestored int   `json:"files_restored,omitempty"`
	BytesRestored int64 `json:"bytes_restored,omitempty"`
	// Read data subset used (if applicable)
	ReadDataSubset string `json:"read_data_subset,omitempty"`
}

// NewVerification creates a new Verification record for the given repository.
func NewVerification(repositoryID uuid.UUID, verType VerificationType) *Verification {
	now := time.Now()
	return &Verification{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		Type:         verType,
		StartedAt:    now,
		Status:       VerificationStatusRunning,
		CreatedAt:    now,
	}
}

// Pass marks the verification as passed.
func (v *Verification) Pass(details *VerificationDetails) {
	now := time.Now()
	v.CompletedAt = &now
	v.Status = VerificationStatusPassed
	durationMs := now.Sub(v.StartedAt).Milliseconds()
	v.DurationMs = &durationMs
	v.Details = details
}

// Fail marks the verification as failed with the given error message.
func (v *Verification) Fail(errMsg string, details *VerificationDetails) {
	now := time.Now()
	v.CompletedAt = &now
	v.Status = VerificationStatusFailed
	durationMs := now.Sub(v.StartedAt).Milliseconds()
	v.DurationMs = &durationMs
	v.ErrorMessage = errMsg
	v.Details = details
}

// Duration returns the duration of the verification, or zero if not completed.
func (v *Verification) Duration() time.Duration {
	if v.CompletedAt == nil {
		return 0
	}
	return v.CompletedAt.Sub(v.StartedAt)
}

// IsComplete returns true if the verification has finished.
func (v *Verification) IsComplete() bool {
	return v.Status == VerificationStatusPassed || v.Status == VerificationStatusFailed
}

// SetDetails sets the details from JSON bytes.
func (v *Verification) SetDetails(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var details VerificationDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return err
	}
	v.Details = &details
	return nil
}

// DetailsJSON returns the details as JSON bytes for database storage.
func (v *Verification) DetailsJSON() ([]byte, error) {
	if v.Details == nil {
		return nil, nil
	}
	return json.Marshal(v.Details)
}

// VerificationSchedule defines when verifications should run for a repository.
type VerificationSchedule struct {
	ID           uuid.UUID        `json:"id"`
	RepositoryID uuid.UUID        `json:"repository_id"`
	Type         VerificationType `json:"type"`
	CronExpression string         `json:"cron_expression"`
	Enabled      bool             `json:"enabled"`
	// ReadDataSubset for check_read_data type (e.g., "2.5%" or "5G")
	ReadDataSubset string         `json:"read_data_subset,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// NewVerificationSchedule creates a new VerificationSchedule.
func NewVerificationSchedule(repositoryID uuid.UUID, verType VerificationType, cronExpr string) *VerificationSchedule {
	now := time.Now()
	return &VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repositoryID,
		Type:           verType,
		CronExpression: cronExpr,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// RepositoryVerificationStatus summarizes verification status for a repository.
type RepositoryVerificationStatus struct {
	RepositoryID      uuid.UUID          `json:"repository_id"`
	LastVerification  *Verification      `json:"last_verification,omitempty"`
	NextScheduledAt   *time.Time         `json:"next_scheduled_at,omitempty"`
	ConsecutiveFails  int                `json:"consecutive_fails"`
}
