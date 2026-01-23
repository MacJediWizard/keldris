package models

import (
	"time"

	"github.com/google/uuid"
)

// CheckpointStatus represents the status of a backup checkpoint.
type CheckpointStatus string

const (
	// CheckpointStatusActive indicates the checkpoint is from an active or interrupted backup.
	CheckpointStatusActive CheckpointStatus = "active"
	// CheckpointStatusCompleted indicates the backup associated with this checkpoint completed.
	CheckpointStatusCompleted CheckpointStatus = "completed"
	// CheckpointStatusCanceled indicates the checkpoint was canceled by the user.
	CheckpointStatusCanceled CheckpointStatus = "canceled"
	// CheckpointStatusExpired indicates the checkpoint expired and is no longer valid.
	CheckpointStatusExpired CheckpointStatus = "expired"
)

// BackupCheckpoint represents the state of an interrupted backup that can be resumed.
type BackupCheckpoint struct {
	ID                uuid.UUID        `json:"id"`
	ScheduleID        uuid.UUID        `json:"schedule_id"`
	AgentID           uuid.UUID        `json:"agent_id"`
	RepositoryID      uuid.UUID        `json:"repository_id"`
	BackupID          *uuid.UUID       `json:"backup_id,omitempty"` // Associated backup record if started
	Status            CheckpointStatus `json:"status"`
	FilesProcessed    int64            `json:"files_processed"`
	BytesProcessed    int64            `json:"bytes_processed"`
	TotalFiles        *int64           `json:"total_files,omitempty"`        // Estimated total files if known
	TotalBytes        *int64           `json:"total_bytes,omitempty"`        // Estimated total bytes if known
	LastProcessedPath string           `json:"last_processed_path,omitempty"` // Last file/directory processed
	ResticState       []byte           `json:"restic_state,omitempty"`        // Serialized restic internal state
	ErrorMessage      string           `json:"error_message,omitempty"`       // Error that caused interruption
	ResumeCount       int              `json:"resume_count"`                  // Number of times this backup has been resumed
	ExpiresAt         *time.Time       `json:"expires_at,omitempty"`          // When checkpoint becomes invalid
	StartedAt         time.Time        `json:"started_at"`                    // When original backup started
	LastUpdatedAt     time.Time        `json:"last_updated_at"`               // When checkpoint was last saved
	CreatedAt         time.Time        `json:"created_at"`
}

// NewBackupCheckpoint creates a new checkpoint for tracking backup progress.
func NewBackupCheckpoint(scheduleID, agentID, repositoryID uuid.UUID) *BackupCheckpoint {
	now := time.Now()
	// Default expiration of 7 days
	expiresAt := now.Add(7 * 24 * time.Hour)
	return &BackupCheckpoint{
		ID:            uuid.New(),
		ScheduleID:    scheduleID,
		AgentID:       agentID,
		RepositoryID:  repositoryID,
		Status:        CheckpointStatusActive,
		ResumeCount:   0,
		StartedAt:     now,
		LastUpdatedAt: now,
		CreatedAt:     now,
		ExpiresAt:     &expiresAt,
	}
}

// UpdateProgress updates the checkpoint with current backup progress.
func (c *BackupCheckpoint) UpdateProgress(filesProcessed, bytesProcessed int64, lastPath string) {
	c.FilesProcessed = filesProcessed
	c.BytesProcessed = bytesProcessed
	c.LastProcessedPath = lastPath
	c.LastUpdatedAt = time.Now()
}

// SetTotals sets the estimated totals for the backup.
func (c *BackupCheckpoint) SetTotals(totalFiles, totalBytes int64) {
	c.TotalFiles = &totalFiles
	c.TotalBytes = &totalBytes
	c.LastUpdatedAt = time.Now()
}

// SetResticState stores the serialized restic internal state.
func (c *BackupCheckpoint) SetResticState(state []byte) {
	c.ResticState = state
	c.LastUpdatedAt = time.Now()
}

// SetBackupID associates a backup record with this checkpoint.
func (c *BackupCheckpoint) SetBackupID(backupID uuid.UUID) {
	c.BackupID = &backupID
	c.LastUpdatedAt = time.Now()
}

// MarkInterrupted records that the backup was interrupted with an error.
func (c *BackupCheckpoint) MarkInterrupted(errMsg string) {
	c.ErrorMessage = errMsg
	c.LastUpdatedAt = time.Now()
}

// MarkCompleted marks the checkpoint as completed (backup finished successfully).
func (c *BackupCheckpoint) MarkCompleted() {
	c.Status = CheckpointStatusCompleted
	c.LastUpdatedAt = time.Now()
}

// MarkCanceled marks the checkpoint as canceled by the user.
func (c *BackupCheckpoint) MarkCanceled() {
	c.Status = CheckpointStatusCanceled
	c.LastUpdatedAt = time.Now()
}

// MarkExpired marks the checkpoint as expired and no longer valid for resume.
func (c *BackupCheckpoint) MarkExpired() {
	c.Status = CheckpointStatusExpired
	c.LastUpdatedAt = time.Now()
}

// IncrementResumeCount increments the resume counter and updates timestamp.
func (c *BackupCheckpoint) IncrementResumeCount() {
	c.ResumeCount++
	c.LastUpdatedAt = time.Now()
}

// IsResumable returns true if this checkpoint can be used to resume a backup.
func (c *BackupCheckpoint) IsResumable() bool {
	if c.Status != CheckpointStatusActive {
		return false
	}
	if c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt) {
		return false
	}
	return true
}

// ProgressPercent returns the completion percentage if totals are known.
func (c *BackupCheckpoint) ProgressPercent() *float64 {
	if c.TotalBytes == nil || *c.TotalBytes == 0 {
		return nil
	}
	pct := float64(c.BytesProcessed) / float64(*c.TotalBytes) * 100
	return &pct
}
