package models

import (
	"time"

	"github.com/google/uuid"
)

// DryRunFile represents a file that would be backed up in a dry run.
type DryRunFile struct {
	Path   string `json:"path"`
	Type   string `json:"type"` // "file" or "dir"
	Size   int64  `json:"size"`
	Action string `json:"action"` // "new", "changed", or "unchanged"
}

// DryRunExcluded represents a file that was excluded from backup.
type DryRunExcluded struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// DryRunResult contains the results of a dry run backup operation.
type DryRunResult struct {
	ID              uuid.UUID        `json:"id"`
	ScheduleID      uuid.UUID        `json:"schedule_id"`
	AgentID         uuid.UUID        `json:"agent_id"`
	FilesToBackup   []DryRunFile     `json:"files_to_backup"`
	ExcludedFiles   []DryRunExcluded `json:"excluded_files"`
	TotalFiles      int              `json:"total_files"`
	TotalSize       int64            `json:"total_size"`
	NewFiles        int              `json:"new_files"`
	ChangedFiles    int              `json:"changed_files"`
	UnchangedFiles  int              `json:"unchanged_files"`
	Duration        time.Duration    `json:"duration"`
	CreatedAt       time.Time        `json:"created_at"`
}

// NewDryRunResult creates a new DryRunResult with the given schedule and agent IDs.
func NewDryRunResult(scheduleID, agentID uuid.UUID) *DryRunResult {
	return &DryRunResult{
		ID:            uuid.New(),
		ScheduleID:    scheduleID,
		AgentID:       agentID,
		FilesToBackup: make([]DryRunFile, 0),
		ExcludedFiles: make([]DryRunExcluded, 0),
		CreatedAt:     time.Now(),
	}
}
