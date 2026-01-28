package models

import (
	"time"

	"github.com/google/uuid"
)

// DockerLogBackupStatus represents the status of a docker log backup.
type DockerLogBackupStatus string

const (
	// DockerLogBackupStatusPending indicates the backup is pending.
	DockerLogBackupStatusPending DockerLogBackupStatus = "pending"
	// DockerLogBackupStatusRunning indicates the backup is in progress.
	DockerLogBackupStatusRunning DockerLogBackupStatus = "running"
	// DockerLogBackupStatusCompleted indicates the backup completed successfully.
	DockerLogBackupStatusCompleted DockerLogBackupStatus = "completed"
	// DockerLogBackupStatusFailed indicates the backup failed.
	DockerLogBackupStatusFailed DockerLogBackupStatus = "failed"
)

// DockerLogRetentionPolicy defines how long to keep container logs.
type DockerLogRetentionPolicy struct {
	MaxAgeDays      int   `json:"max_age_days"`       // Max age of logs to keep in days
	MaxSizeBytes    int64 `json:"max_size_bytes"`     // Max total size of logs per container
	MaxFilesPerDay  int   `json:"max_files_per_day"`  // Max backup files per day
	CompressEnabled bool  `json:"compress_enabled"`   // Whether to compress logs
	CompressLevel   int   `json:"compress_level"`     // Compression level (1-9)
}

// DefaultDockerLogRetentionPolicy returns sensible defaults.
func DefaultDockerLogRetentionPolicy() DockerLogRetentionPolicy {
	return DockerLogRetentionPolicy{
		MaxAgeDays:      30,
		MaxSizeBytes:    1024 * 1024 * 1024, // 1GB
		MaxFilesPerDay:  24,                 // One per hour
		CompressEnabled: true,
		CompressLevel:   6, // Default gzip level
	}
}

// DockerLogBackup represents a backed up container log file.
type DockerLogBackup struct {
	ID               uuid.UUID             `json:"id"`
	AgentID          uuid.UUID             `json:"agent_id"`
	ContainerID      string                `json:"container_id"`
	ContainerName    string                `json:"container_name"`
	ImageName        string                `json:"image_name,omitempty"`
	LogPath          string                `json:"log_path"`              // Path to the backup file
	OriginalSize     int64                 `json:"original_size"`         // Size before compression
	CompressedSize   int64                 `json:"compressed_size"`       // Size after compression (0 if not compressed)
	Compressed       bool                  `json:"compressed"`            // Whether the log is compressed
	StartTime        time.Time             `json:"start_time"`            // Start of log time range
	EndTime          time.Time             `json:"end_time"`              // End of log time range
	LineCount        int64                 `json:"line_count"`            // Number of log lines
	Status           DockerLogBackupStatus `json:"status"`
	ErrorMessage     string                `json:"error_message,omitempty"`
	BackupScheduleID *uuid.UUID            `json:"backup_schedule_id,omitempty"` // Associated backup schedule
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

// NewDockerLogBackup creates a new DockerLogBackup.
func NewDockerLogBackup(agentID uuid.UUID, containerID, containerName string) *DockerLogBackup {
	now := time.Now()
	return &DockerLogBackup{
		ID:            uuid.New(),
		AgentID:       agentID,
		ContainerID:   containerID,
		ContainerName: containerName,
		Status:        DockerLogBackupStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// MarkRunning marks the backup as running.
func (d *DockerLogBackup) MarkRunning() {
	d.Status = DockerLogBackupStatusRunning
	d.UpdatedAt = time.Now()
}

// Complete marks the backup as completed with statistics.
func (d *DockerLogBackup) Complete(logPath string, originalSize, compressedSize, lineCount int64, startTime, endTime time.Time, compressed bool) {
	d.Status = DockerLogBackupStatusCompleted
	d.LogPath = logPath
	d.OriginalSize = originalSize
	d.CompressedSize = compressedSize
	d.Compressed = compressed
	d.LineCount = lineCount
	d.StartTime = startTime
	d.EndTime = endTime
	d.UpdatedAt = time.Now()
}

// Fail marks the backup as failed with an error message.
func (d *DockerLogBackup) Fail(errMsg string) {
	d.Status = DockerLogBackupStatusFailed
	d.ErrorMessage = errMsg
	d.UpdatedAt = time.Now()
}

// CompressionRatio returns the compression ratio (0-1), or 0 if not compressed.
func (d *DockerLogBackup) CompressionRatio() float64 {
	if !d.Compressed || d.OriginalSize == 0 {
		return 0
	}
	return 1 - (float64(d.CompressedSize) / float64(d.OriginalSize))
}

// DockerLogSettings represents the log backup settings for an agent.
type DockerLogSettings struct {
	ID                uuid.UUID                `json:"id"`
	AgentID           uuid.UUID                `json:"agent_id"`
	Enabled           bool                     `json:"enabled"`
	CronExpression    string                   `json:"cron_expression"` // Schedule for backups
	RetentionPolicy   DockerLogRetentionPolicy `json:"retention_policy"`
	IncludeContainers []string                 `json:"include_containers,omitempty"` // Container names/IDs to include (empty = all)
	ExcludeContainers []string                 `json:"exclude_containers,omitempty"` // Container names/IDs to exclude
	IncludeLabels     map[string]string        `json:"include_labels,omitempty"`     // Container labels to include
	ExcludeLabels     map[string]string        `json:"exclude_labels,omitempty"`     // Container labels to exclude
	Timestamps        bool                     `json:"timestamps"`                   // Include timestamps in logs
	Tail              int                      `json:"tail"`                         // Number of lines to tail (0 = all)
	Since             string                   `json:"since,omitempty"`              // Only logs since this time (duration or timestamp)
	Until             string                   `json:"until,omitempty"`              // Only logs until this time
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

// NewDockerLogSettings creates new Docker log settings for an agent.
func NewDockerLogSettings(agentID uuid.UUID) *DockerLogSettings {
	now := time.Now()
	return &DockerLogSettings{
		ID:              uuid.New(),
		AgentID:         agentID,
		Enabled:         false,
		CronExpression:  "0 * * * *", // Every hour by default
		RetentionPolicy: DefaultDockerLogRetentionPolicy(),
		Timestamps:      true,
		Tail:            0, // All lines
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// DockerLogEntry represents a single log line for display.
type DockerLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"` // stdout or stderr
	Message   string    `json:"message"`
	LineNum   int64     `json:"line_num"`
}

// DockerLogViewResponse is the response for viewing backed up logs.
type DockerLogViewResponse struct {
	BackupID      uuid.UUID        `json:"backup_id"`
	ContainerID   string           `json:"container_id"`
	ContainerName string           `json:"container_name"`
	Entries       []DockerLogEntry `json:"entries"`
	TotalLines    int64            `json:"total_lines"`
	Offset        int64            `json:"offset"`
	Limit         int64            `json:"limit"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
}

// DockerContainerInfo represents information about a container.
type DockerContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    string            `json:"status"`
	State     string            `json:"state"` // running, exited, etc.
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}
