package models

import (
	"time"

	"github.com/google/uuid"
)

// DatabaseBackupStatus represents the status of a database backup.
type DatabaseBackupStatus string

const (
	// DatabaseBackupStatusPending indicates the backup is scheduled but not started.
	DatabaseBackupStatusPending DatabaseBackupStatus = "pending"
	// DatabaseBackupStatusRunning indicates the backup is in progress.
	DatabaseBackupStatusRunning DatabaseBackupStatus = "running"
	// DatabaseBackupStatusCompleted indicates the backup completed successfully.
	DatabaseBackupStatusCompleted DatabaseBackupStatus = "completed"
	// DatabaseBackupStatusFailed indicates the backup failed.
	DatabaseBackupStatusFailed DatabaseBackupStatus = "failed"
)

// DatabaseBackup represents a backup of the Keldris PostgreSQL database.
type DatabaseBackup struct {
	ID           uuid.UUID            `json:"id" db:"id"`
	Status       DatabaseBackupStatus `json:"status" db:"status"`
	FilePath     string               `json:"file_path,omitempty" db:"file_path"`
	SizeBytes    *int64               `json:"size_bytes,omitempty" db:"size_bytes"`
	Checksum     string               `json:"checksum,omitempty" db:"checksum"`
	StartedAt    time.Time            `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time           `json:"completed_at,omitempty" db:"completed_at"`
	Duration     *int64               `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage string               `json:"error_message,omitempty" db:"error_message"`
	TriggeredBy  *uuid.UUID           `json:"triggered_by,omitempty" db:"triggered_by"`
	IsScheduled  bool                 `json:"is_scheduled" db:"is_scheduled"`
	Verified     bool                 `json:"verified" db:"verified"`
	VerifiedAt   *time.Time           `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt    time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at" db:"updated_at"`
}

// NewDatabaseBackup creates a new database backup record.
func NewDatabaseBackup() *DatabaseBackup {
	now := time.Now()
	return &DatabaseBackup{
		ID:          uuid.New(),
		Status:      DatabaseBackupStatusPending,
		IsScheduled: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewManualDatabaseBackup creates a new manually-triggered database backup record.
func NewManualDatabaseBackup(triggeredBy uuid.UUID) *DatabaseBackup {
	backup := NewDatabaseBackup()
	backup.TriggeredBy = &triggeredBy
	backup.IsScheduled = false
	return backup
}

// Start marks the backup as running.
func (b *DatabaseBackup) Start() {
	b.Status = DatabaseBackupStatusRunning
	b.StartedAt = time.Now()
	b.UpdatedAt = time.Now()
}

// Complete marks the backup as completed successfully.
func (b *DatabaseBackup) Complete(filePath string, sizeBytes int64, checksum string) {
	now := time.Now()
	b.Status = DatabaseBackupStatusCompleted
	b.FilePath = filePath
	b.SizeBytes = &sizeBytes
	b.Checksum = checksum
	b.CompletedAt = &now
	durationMs := now.Sub(b.StartedAt).Milliseconds()
	b.Duration = &durationMs
	b.UpdatedAt = now
}

// Fail marks the backup as failed.
func (b *DatabaseBackup) Fail(errorMessage string) {
	now := time.Now()
	b.Status = DatabaseBackupStatusFailed
	b.ErrorMessage = errorMessage
	b.CompletedAt = &now
	durationMs := now.Sub(b.StartedAt).Milliseconds()
	b.Duration = &durationMs
	b.UpdatedAt = now
}

// MarkVerified marks the backup as verified.
func (b *DatabaseBackup) MarkVerified() {
	now := time.Now()
	b.Verified = true
	b.VerifiedAt = &now
	b.UpdatedAt = now
}

// IsSuccessful returns true if the backup completed successfully.
func (b *DatabaseBackup) IsSuccessful() bool {
	return b.Status == DatabaseBackupStatusCompleted
}

// IsRecent returns true if the backup was created within the given duration.
func (b *DatabaseBackup) IsRecent(within time.Duration) bool {
	return time.Since(b.CreatedAt) <= within
}

// DatabaseBackupSummary provides a summary of backup status for dashboards.
type DatabaseBackupSummary struct {
	TotalBackups       int       `json:"total_backups"`
	SuccessfulBackups  int       `json:"successful_backups"`
	FailedBackups      int       `json:"failed_backups"`
	TotalSizeBytes     int64     `json:"total_size_bytes"`
	LastBackupAt       *time.Time `json:"last_backup_at,omitempty"`
	LastBackupStatus   string    `json:"last_backup_status,omitempty"`
	NextScheduledAt    *time.Time `json:"next_scheduled_at,omitempty"`
	OldestBackupAt     *time.Time `json:"oldest_backup_at,omitempty"`
	BackupServiceUp    bool      `json:"backup_service_up"`
}
