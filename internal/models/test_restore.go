package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TestRestoreStatus represents the current status of a test restore.
type TestRestoreResultStatus string

const (
	// TestRestoreStatusPending indicates the test restore is queued.
	TestRestoreStatusPending TestRestoreResultStatus = "pending"
	// TestRestoreStatusRunning indicates the test restore is in progress.
	TestRestoreStatusRunning TestRestoreResultStatus = "running"
	// TestRestoreStatusPassed indicates the test restore passed all verifications.
	TestRestoreStatusPassed TestRestoreResultStatus = "passed"
	// TestRestoreStatusFailed indicates the test restore failed.
	TestRestoreStatusFailed TestRestoreResultStatus = "failed"
)

// TestRestoreFrequency defines preset frequencies for test restores.
type TestRestoreFrequency string

const (
	// TestRestoreFrequencyWeekly runs test restores weekly.
	TestRestoreFrequencyWeekly TestRestoreFrequency = "weekly"
	// TestRestoreFrequencyMonthly runs test restores monthly.
	TestRestoreFrequencyMonthly TestRestoreFrequency = "monthly"
	// TestRestoreFrequencyCustom uses a custom cron expression.
	TestRestoreFrequencyCustom TestRestoreFrequency = "custom"
)

// TestRestoreSettings defines settings for automated test restores per repository.
type TestRestoreSettings struct {
	ID               uuid.UUID            `json:"id"`
	RepositoryID     uuid.UUID            `json:"repository_id"`
	Enabled          bool                 `json:"enabled"`
	Frequency        TestRestoreFrequency `json:"frequency"`
	CronExpression   string               `json:"cron_expression"`
	SamplePercentage int                  `json:"sample_percentage"` // Percentage of files to restore (1-100)
	LastRunAt        *time.Time           `json:"last_run_at,omitempty"`
	LastRunStatus    *string              `json:"last_run_status,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// NewTestRestoreSettings creates new TestRestoreSettings with default values.
func NewTestRestoreSettings(repositoryID uuid.UUID) *TestRestoreSettings {
	now := time.Now()
	return &TestRestoreSettings{
		ID:               uuid.New(),
		RepositoryID:     repositoryID,
		Enabled:          true,
		Frequency:        TestRestoreFrequencyWeekly,
		CronExpression:   "0 0 3 * * 0", // Default: 3 AM on Sundays (cron with seconds)
		SamplePercentage: 10,            // Default: 10% of files
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// SetFrequency sets the frequency and updates the cron expression accordingly.
func (s *TestRestoreSettings) SetFrequency(freq TestRestoreFrequency, customCron string) {
	s.Frequency = freq
	s.UpdatedAt = time.Now()

	switch freq {
	case TestRestoreFrequencyWeekly:
		s.CronExpression = "0 0 3 * * 0" // 3 AM on Sundays
	case TestRestoreFrequencyMonthly:
		s.CronExpression = "0 0 3 1 * *" // 3 AM on the 1st of each month
	case TestRestoreFrequencyCustom:
		if customCron != "" {
			s.CronExpression = customCron
		}
	}
}

// RecordLastRun updates the last run timestamp and status.
func (s *TestRestoreSettings) RecordLastRun(success bool) {
	now := time.Now()
	s.LastRunAt = &now
	s.UpdatedAt = now
	if success {
		status := string(TestRestoreStatusPassed)
		s.LastRunStatus = &status
	} else {
		status := string(TestRestoreStatusFailed)
		s.LastRunStatus = &status
	}
}

// FileChecksum represents a verified file checksum.
type FileChecksum struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"` // SHA256 hex string
	Size     int64  `json:"size"`
}

// TestRestoreDetails contains detailed information about a test restore execution.
type TestRestoreDetails struct {
	// SnapshotID is the ID of the snapshot that was restored.
	SnapshotID string `json:"snapshot_id,omitempty"`
	// TotalFilesInSnapshot is the total number of files in the snapshot.
	TotalFilesInSnapshot int `json:"total_files_in_snapshot,omitempty"`
	// FilesRestored is the number of files successfully restored.
	FilesRestored int `json:"files_restored,omitempty"`
	// FilesVerified is the number of files that passed checksum verification.
	FilesVerified int `json:"files_verified,omitempty"`
	// BytesRestored is the total size of restored data in bytes.
	BytesRestored int64 `json:"bytes_restored,omitempty"`
	// TempDirectory is the temporary directory used for the restore.
	TempDirectory string `json:"temp_directory,omitempty"`
	// VerifiedChecksums contains checksums of verified files.
	VerifiedChecksums []FileChecksum `json:"verified_checksums,omitempty"`
	// VerificationErrors contains any errors encountered during verification.
	VerificationErrors []string `json:"verification_errors,omitempty"`
}

// TestRestoreResult represents a single test restore execution record.
type TestRestoreResult struct {
	ID               uuid.UUID               `json:"id"`
	RepositoryID     uuid.UUID               `json:"repository_id"`
	SnapshotID       string                  `json:"snapshot_id,omitempty"`
	SamplePercentage int                     `json:"sample_percentage"`
	StartedAt        time.Time               `json:"started_at"`
	CompletedAt      *time.Time              `json:"completed_at,omitempty"`
	Status           TestRestoreResultStatus `json:"status"`
	DurationMs       *int64                  `json:"duration_ms,omitempty"`
	FilesRestored    int                     `json:"files_restored"`
	FilesVerified    int                     `json:"files_verified"`
	BytesRestored    int64                   `json:"bytes_restored"`
	ErrorMessage     string                  `json:"error_message,omitempty"`
	Details          *TestRestoreDetails     `json:"details,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
}

// NewTestRestoreResult creates a new TestRestoreResult record.
func NewTestRestoreResult(repositoryID uuid.UUID) *TestRestoreResult {
	now := time.Now()
	return &TestRestoreResult{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		StartedAt:    now,
		Status:       TestRestoreStatusRunning,
		CreatedAt:    now,
	}
}

// Pass marks the test restore as passed.
func (r *TestRestoreResult) Pass(details *TestRestoreDetails) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = TestRestoreStatusPassed
	durationMs := now.Sub(r.StartedAt).Milliseconds()
	r.DurationMs = &durationMs
	if details != nil {
		r.SnapshotID = details.SnapshotID
		r.FilesRestored = details.FilesRestored
		r.FilesVerified = details.FilesVerified
		r.BytesRestored = details.BytesRestored
		r.Details = details
	}
}

// Fail marks the test restore as failed with the given error message.
func (r *TestRestoreResult) Fail(errMsg string, details *TestRestoreDetails) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = TestRestoreStatusFailed
	durationMs := now.Sub(r.StartedAt).Milliseconds()
	r.DurationMs = &durationMs
	r.ErrorMessage = errMsg
	if details != nil {
		r.SnapshotID = details.SnapshotID
		r.FilesRestored = details.FilesRestored
		r.FilesVerified = details.FilesVerified
		r.BytesRestored = details.BytesRestored
		r.Details = details
	}
}

// Duration returns the duration of the test restore, or zero if not completed.
func (r *TestRestoreResult) Duration() time.Duration {
	if r.CompletedAt == nil {
		return 0
	}
	return r.CompletedAt.Sub(r.StartedAt)
}

// IsComplete returns true if the test restore has finished.
func (r *TestRestoreResult) IsComplete() bool {
	return r.Status == TestRestoreStatusPassed || r.Status == TestRestoreStatusFailed
}

// SetDetails sets the details from JSON bytes.
func (r *TestRestoreResult) SetDetails(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var details TestRestoreDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return err
	}
	r.Details = &details
	return nil
}

// DetailsJSON returns the details as JSON bytes for database storage.
func (r *TestRestoreResult) DetailsJSON() ([]byte, error) {
	if r.Details == nil {
		return nil, nil
	}
	return json.Marshal(r.Details)
}

// TestRestoreStatus summarizes test restore status for a repository.
type TestRestoreStatus struct {
	RepositoryID     uuid.UUID            `json:"repository_id"`
	Settings         *TestRestoreSettings `json:"settings,omitempty"`
	LastResult       *TestRestoreResult   `json:"last_result,omitempty"`
	NextScheduledAt  *time.Time           `json:"next_scheduled_at,omitempty"`
	ConsecutiveFails int                  `json:"consecutive_fails"`
}
