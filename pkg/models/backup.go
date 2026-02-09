package models

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

// BackupRequest is the request body for initiating a backup from the agent.
type BackupRequest struct {
	ScheduleID string   `json:"schedule_id" binding:"required"`
	Paths      []string `json:"paths" binding:"required"`
	Tags       []string `json:"tags,omitempty"`
	Exclude    []string `json:"exclude,omitempty"`
}

// BackupResponse is the server response to a backup request.
type BackupResponse struct {
	BackupID   string       `json:"backup_id"`
	Status     BackupStatus `json:"status"`
	Message    string       `json:"message,omitempty"`
	SnapshotID string       `json:"snapshot_id,omitempty"`
}
