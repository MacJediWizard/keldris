package models

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
)

// JobType defines the type of job in the queue.
type JobType string

const (
	// JobTypeBackup is a backup job.
	JobTypeBackup JobType = "backup"
	// JobTypeRestore is a restore job.
	JobTypeRestore JobType = "restore"
	// JobTypeVerification is a verification job.
	JobTypeVerification JobType = "verification"
)

// JobStatus defines the status of a job.
type JobStatus string

const (
	// JobStatusPending indicates the job is waiting to be processed.
	JobStatusPending JobStatus = "pending"
	// JobStatusRunning indicates the job is currently being processed.
	JobStatusRunning JobStatus = "running"
	// JobStatusCompleted indicates the job completed successfully.
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job failed and may be retried.
	JobStatusFailed JobStatus = "failed"
	// JobStatusDeadLetter indicates the job has exhausted all retries.
	JobStatusDeadLetter JobStatus = "dead_letter"
)

// DefaultMaxRetries is the default number of retry attempts.
const DefaultMaxRetries = 3

// Job represents a job in the queue.
type Job struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	JobType      JobType    `json:"job_type"`
	Priority     int        `json:"priority"`
	Status       JobStatus  `json:"status"`
	Payload      JobPayload `json:"payload"`
	RetryCount   int        `json:"retry_count"`
	MaxRetries   int        `json:"max_retries"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	LastErrorAt  *time.Time `json:"last_error_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	// Optional references for quick lookups
	AgentID      *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`
	ScheduleID   *uuid.UUID `json:"schedule_id,omitempty"`
}

// JobPayload contains job-specific data stored as JSONB.
type JobPayload struct {
	// Common fields
	Description string `json:"description,omitempty"`

	// Backup job fields
	ScheduleID   *uuid.UUID `json:"schedule_id,omitempty"`
	AgentID      *uuid.UUID `json:"agent_id,omitempty"`
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`

	// Restore job fields
	SnapshotID string   `json:"snapshot_id,omitempty"`
	TargetPath string   `json:"target_path,omitempty"`
	Include    []string `json:"include,omitempty"`
	Exclude    []string `json:"exclude,omitempty"`

	// Verification job fields
	VerificationType string `json:"verification_type,omitempty"`
	ReadDataSubset   string `json:"read_data_subset,omitempty"`

	// Result data (populated on completion)
	Result map[string]interface{} `json:"result,omitempty"`
}

// NewJob creates a new job with the given parameters.
func NewJob(orgID uuid.UUID, jobType JobType, priority int, payload JobPayload) *Job {
	now := time.Now()
	job := &Job{
		ID:         uuid.New(),
		OrgID:      orgID,
		JobType:    jobType,
		Priority:   priority,
		Status:     JobStatusPending,
		Payload:    payload,
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries,
		CreatedAt:  now,
	}

	// Set optional references from payload
	if payload.AgentID != nil {
		job.AgentID = payload.AgentID
	}
	if payload.RepositoryID != nil {
		job.RepositoryID = payload.RepositoryID
	}
	if payload.ScheduleID != nil {
		job.ScheduleID = payload.ScheduleID
	}

	return job
}

// NewBackupJob creates a new backup job.
func NewBackupJob(orgID, agentID, repositoryID, scheduleID uuid.UUID, priority int) *Job {
	payload := JobPayload{
		AgentID:      &agentID,
		RepositoryID: &repositoryID,
		ScheduleID:   &scheduleID,
		Description:  "Scheduled backup",
	}
	return NewJob(orgID, JobTypeBackup, priority, payload)
}

// NewRestoreJob creates a new restore job.
func NewRestoreJob(orgID, agentID, repositoryID uuid.UUID, snapshotID, targetPath string) *Job {
	payload := JobPayload{
		AgentID:      &agentID,
		RepositoryID: &repositoryID,
		SnapshotID:   snapshotID,
		TargetPath:   targetPath,
		Description:  "Restore operation",
	}
	return NewJob(orgID, JobTypeRestore, 0, payload)
}

// NewVerificationJob creates a new verification job.
func NewVerificationJob(orgID, repositoryID uuid.UUID, verificationType string, priority int) *Job {
	payload := JobPayload{
		RepositoryID:     &repositoryID,
		VerificationType: verificationType,
		Description:      "Repository verification",
	}
	return NewJob(orgID, JobTypeVerification, priority, payload)
}

// Start marks the job as running.
func (j *Job) Start() {
	now := time.Now()
	j.Status = JobStatusRunning
	j.StartedAt = &now
}

// Complete marks the job as completed successfully.
func (j *Job) Complete(result map[string]interface{}) {
	now := time.Now()
	j.Status = JobStatusCompleted
	j.CompletedAt = &now
	j.Payload.Result = result
}

// Fail marks the job as failed with the given error message.
// Returns true if the job should be retried, false if it should be moved to dead letter.
func (j *Job) Fail(errMsg string) bool {
	now := time.Now()
	j.Status = JobStatusFailed
	j.ErrorMessage = errMsg
	j.LastErrorAt = &now
	j.RetryCount++

	if j.RetryCount >= j.MaxRetries {
		j.Status = JobStatusDeadLetter
		j.CompletedAt = &now
		return false
	}

	// Calculate next retry time with exponential backoff
	// Base delay: 30 seconds, max delay: 30 minutes
	backoffSeconds := math.Min(30*math.Pow(2, float64(j.RetryCount-1)), 1800)
	nextRetry := now.Add(time.Duration(backoffSeconds) * time.Second)
	j.NextRetryAt = &nextRetry

	return true
}

// Cancel cancels a pending job.
func (j *Job) Cancel() bool {
	if j.Status != JobStatusPending {
		return false
	}
	j.Status = JobStatusDeadLetter
	now := time.Now()
	j.CompletedAt = &now
	j.ErrorMessage = "Job canceled by user"
	return true
}

// Retry resets a failed job for retry.
func (j *Job) Retry() bool {
	if j.Status != JobStatusFailed && j.Status != JobStatusDeadLetter {
		return false
	}
	j.Status = JobStatusPending
	j.RetryCount = 0
	j.NextRetryAt = nil
	j.ErrorMessage = ""
	j.LastErrorAt = nil
	j.StartedAt = nil
	j.CompletedAt = nil
	return true
}

// IsTerminal returns true if the job is in a terminal state.
func (j *Job) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusDeadLetter
}

// CanRetry returns true if the job can be retried.
func (j *Job) CanRetry() bool {
	return j.Status == JobStatusFailed || j.Status == JobStatusDeadLetter
}

// ReadyForRetry returns true if the job is ready to be retried based on NextRetryAt.
func (j *Job) ReadyForRetry() bool {
	if j.Status != JobStatusFailed {
		return false
	}
	if j.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*j.NextRetryAt)
}

// Duration returns the duration of the job, or zero if not completed.
func (j *Job) Duration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}
	endTime := time.Now()
	if j.CompletedAt != nil {
		endTime = *j.CompletedAt
	}
	return endTime.Sub(*j.StartedAt)
}

// WaitTime returns the time the job spent waiting in the queue.
func (j *Job) WaitTime() time.Duration {
	if j.StartedAt == nil {
		return time.Since(j.CreatedAt)
	}
	return j.StartedAt.Sub(j.CreatedAt)
}

// PayloadJSON returns the payload as JSON bytes for database storage.
func (j *Job) PayloadJSON() ([]byte, error) {
	return json.Marshal(j.Payload)
}

// SetPayload sets the payload from JSON bytes.
func (j *Job) SetPayload(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &j.Payload)
}

// JobQueueSummary provides queue statistics.
type JobQueueSummary struct {
	TotalPending    int               `json:"total_pending"`
	TotalRunning    int               `json:"total_running"`
	TotalCompleted  int               `json:"total_completed"`
	TotalFailed     int               `json:"total_failed"`
	TotalDeadLetter int               `json:"total_dead_letter"`
	ByType          map[JobType]int   `json:"by_type,omitempty"`
	AvgWaitMinutes  float64           `json:"avg_wait_minutes"`
	OldestPending   *time.Time        `json:"oldest_pending,omitempty"`
}

// JobWithDetails extends Job with related entity names for display.
type JobWithDetails struct {
	Job
	AgentHostname   string `json:"agent_hostname,omitempty"`
	RepositoryName  string `json:"repository_name,omitempty"`
	ScheduleName    string `json:"schedule_name,omitempty"`
	QueuePosition   int    `json:"queue_position,omitempty"`
}
