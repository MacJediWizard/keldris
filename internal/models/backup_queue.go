package models

import (
	"time"

	"github.com/google/uuid"
)

// BackupQueueStatus represents the status of a queued backup.
type BackupQueueStatus string

const (
	// BackupQueueStatusQueued indicates the backup is waiting in queue.
	BackupQueueStatusQueued BackupQueueStatus = "queued"
	// BackupQueueStatusStarted indicates the backup has been started.
	BackupQueueStatusStarted BackupQueueStatus = "started"
	// BackupQueueStatusCanceled indicates the queued backup was canceled.
	BackupQueueStatusCanceled BackupQueueStatus = "canceled"
)

// BackupQueueEntry represents a backup waiting in the queue.
type BackupQueueEntry struct {
	ID            uuid.UUID         `json:"id"`
	OrgID         uuid.UUID         `json:"org_id"`
	AgentID       uuid.UUID         `json:"agent_id"`
	ScheduleID    uuid.UUID         `json:"schedule_id"`
	Priority      int               `json:"priority"` // Higher values = higher priority
	QueuedAt      time.Time         `json:"queued_at"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	Status        BackupQueueStatus `json:"status"`
	QueuePosition int               `json:"queue_position,omitempty"` // Calculated position
	CreatedAt     time.Time         `json:"created_at"`
}

// BackupQueueEntryWithDetails extends BackupQueueEntry with related info.
type BackupQueueEntryWithDetails struct {
	BackupQueueEntry
	ScheduleName string `json:"schedule_name"`
	AgentName    string `json:"agent_hostname"`
}

// NewBackupQueueEntry creates a new backup queue entry.
func NewBackupQueueEntry(orgID, agentID, scheduleID uuid.UUID, priority int) *BackupQueueEntry {
	now := time.Now()
	return &BackupQueueEntry{
		ID:         uuid.New(),
		OrgID:      orgID,
		AgentID:    agentID,
		ScheduleID: scheduleID,
		Priority:   priority,
		QueuedAt:   now,
		Status:     BackupQueueStatusQueued,
		CreatedAt:  now,
	}
}

// MarkStarted marks the queue entry as started.
func (e *BackupQueueEntry) MarkStarted() {
	now := time.Now()
	e.StartedAt = &now
	e.Status = BackupQueueStatusStarted
}

// Cancel marks the queue entry as canceled.
func (e *BackupQueueEntry) Cancel() {
	e.Status = BackupQueueStatusCanceled
}

// ConcurrencyStatus represents the current backup concurrency state.
type ConcurrencyStatus struct {
	OrgID                uuid.UUID `json:"org_id"`
	OrgLimit             *int      `json:"org_limit,omitempty"`           // nil means unlimited
	OrgRunningCount      int       `json:"org_running_count"`             // Backups running for org
	OrgQueuedCount       int       `json:"org_queued_count"`              // Backups queued for org
	AgentID              uuid.UUID `json:"agent_id,omitempty"`
	AgentLimit           *int      `json:"agent_limit,omitempty"`         // nil means use org limit
	AgentRunningCount    int       `json:"agent_running_count"`           // Backups running for agent
	AgentQueuedCount     int       `json:"agent_queued_count"`            // Backups queued for agent
	CanStartNow          bool      `json:"can_start_now"`                 // Whether a new backup can start
	QueuePosition        int       `json:"queue_position,omitempty"`      // Position in queue if queued
	EstimatedWaitMinutes int       `json:"estimated_wait_minutes,omitempty"` // Estimated wait time
}

// BackupQueueSummary provides queue statistics.
type BackupQueueSummary struct {
	TotalQueued   int `json:"total_queued"`
	ByOrg         map[uuid.UUID]int `json:"by_org,omitempty"`
	ByAgent       map[uuid.UUID]int `json:"by_agent,omitempty"`
	OldestQueued  *time.Time `json:"oldest_queued,omitempty"`
	AvgWaitMinutes float64 `json:"avg_wait_minutes"`
}
