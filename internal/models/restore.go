package models

import (
	"time"

	"github.com/google/uuid"
)

// RestoreStatus represents the current status of a restore operation.
type RestoreStatus string

const (
	// RestoreStatusPending indicates the restore is queued.
	RestoreStatusPending RestoreStatus = "pending"
	// RestoreStatusRunning indicates the restore is in progress.
	RestoreStatusRunning RestoreStatus = "running"
	// RestoreStatusCompleted indicates the restore completed successfully.
	RestoreStatusCompleted RestoreStatus = "completed"
	// RestoreStatusFailed indicates the restore failed.
	RestoreStatusFailed RestoreStatus = "failed"
	// RestoreStatusCanceled indicates the restore was canceled.
	RestoreStatusCanceled RestoreStatus = "canceled"
)

// Restore represents a restore job execution record.
type Restore struct {
	ID           uuid.UUID     `json:"id"`
	AgentID      uuid.UUID     `json:"agent_id"`
	RepositoryID uuid.UUID     `json:"repository_id"`
	SnapshotID   string        `json:"snapshot_id"`
	TargetPath   string        `json:"target_path"`
	IncludePaths []string      `json:"include_paths,omitempty"`
	ExcludePaths []string      `json:"exclude_paths,omitempty"`
	Status       RestoreStatus `json:"status"`
	StartedAt    *time.Time    `json:"started_at,omitempty"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	DeletedAt    *time.Time    `json:"deleted_at,omitempty"`
}

// NewRestore creates a new Restore job.
func NewRestore(agentID, repositoryID uuid.UUID, snapshotID, targetPath string, includePaths, excludePaths []string) *Restore {
	now := time.Now()
	return &Restore{
		ID:           uuid.New(),
		AgentID:      agentID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		TargetPath:   targetPath,
		IncludePaths: includePaths,
		ExcludePaths: excludePaths,
		Status:       RestoreStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Start marks the restore as running.
func (r *Restore) Start() {
	now := time.Now()
	r.StartedAt = &now
	r.Status = RestoreStatusRunning
	r.UpdatedAt = now
}

// Complete marks the restore as completed successfully.
func (r *Restore) Complete() {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusCompleted
	r.UpdatedAt = now
}

// Fail marks the restore as failed with the given error message.
func (r *Restore) Fail(errMsg string) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusFailed
	r.ErrorMessage = errMsg
	r.UpdatedAt = now
}

// Cancel marks the restore as canceled.
func (r *Restore) Cancel() {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusCanceled
	r.UpdatedAt = now
}

// Duration returns the duration of the restore, or zero if not started/completed.
func (r *Restore) Duration() time.Duration {
	if r.StartedAt == nil || r.CompletedAt == nil {
		return 0
	}
	return r.CompletedAt.Sub(*r.StartedAt)
}

// IsComplete returns true if the restore has finished.
func (r *Restore) IsComplete() bool {
	return r.Status == RestoreStatusCompleted ||
		r.Status == RestoreStatusFailed ||
		r.Status == RestoreStatusCanceled
}
