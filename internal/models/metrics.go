package models

import (
	"time"

	"github.com/google/uuid"
)

// MetricsHistory represents a point-in-time snapshot of system metrics.
type MetricsHistory struct {
	ID uuid.UUID `json:"id"`
	OrgID uuid.UUID `json:"org_id"`

	// Backup metrics
	BackupCount         int   `json:"backup_count"`
	BackupSuccessCount  int   `json:"backup_success_count"`
	BackupFailedCount   int   `json:"backup_failed_count"`
	BackupTotalSize     int64 `json:"backup_total_size"`
	BackupTotalDuration int64 `json:"backup_total_duration_ms"`

	// Agent metrics
	AgentTotalCount   int `json:"agent_total_count"`
	AgentOnlineCount  int `json:"agent_online_count"`
	AgentOfflineCount int `json:"agent_offline_count"`

	// Storage metrics
	StorageUsedBytes  int64 `json:"storage_used_bytes"`
	StorageRawBytes   int64 `json:"storage_raw_bytes"`
	StorageSpaceSaved int64 `json:"storage_space_saved"`

	// Repository metrics
	RepositoryCount int `json:"repository_count"`
	TotalSnapshots  int `json:"total_snapshots"`

	CollectedAt time.Time `json:"collected_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewMetricsHistory creates a new MetricsHistory record.
func NewMetricsHistory(orgID uuid.UUID) *MetricsHistory {
	now := time.Now()
	return &MetricsHistory{
		ID:          uuid.New(),
		OrgID:       orgID,
		CollectedAt: now,
		CreatedAt:   now,
	}
}

// MetricsDailySummary represents aggregated daily metrics.
type MetricsDailySummary struct {
	ID    uuid.UUID `json:"id"`
	OrgID uuid.UUID `json:"org_id"`
	Date  time.Time `json:"date"`

	TotalBackups      int   `json:"total_backups"`
	SuccessfulBackups int   `json:"successful_backups"`
	FailedBackups     int   `json:"failed_backups"`
	TotalSizeBytes    int64 `json:"total_size_bytes"`
	TotalDurationSecs int64 `json:"total_duration_secs"`
	AgentsActive      int   `json:"agents_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}


// DashboardStats represents the main dashboard statistics.
type DashboardStats struct {
	// Agent counts
	AgentTotal   int `json:"agent_total"`
	AgentOnline  int `json:"agent_online"`
	AgentOffline int `json:"agent_offline"`

	// Backup counts
	BackupTotal     int `json:"backup_total"`
	BackupRunning   int `json:"backup_running"`
	BackupFailed24h int `json:"backup_failed_24h"`

	// Repository and schedule counts
	RepositoryCount  int `json:"repository_count"`
	ScheduleCount    int `json:"schedule_count"`
	ScheduleEnabled  int `json:"schedule_enabled"`

	// Storage stats
	TotalBackupSize int64   `json:"total_backup_size"`
	TotalRawSize    int64   `json:"total_raw_size"`
	TotalSpaceSaved int64   `json:"total_space_saved"`
	AvgDedupRatio   float64 `json:"avg_dedup_ratio"`

	// Success rates
	SuccessRate7d  float64 `json:"success_rate_7d"`
	SuccessRate30d float64 `json:"success_rate_30d"`
}

// BackupSuccessRate represents backup success rate for a time period.
type BackupSuccessRate struct {
	Period         string  `json:"period"`
	Total          int     `json:"total"`
	Successful     int     `json:"successful"`
	Failed         int     `json:"failed"`
	SuccessPercent float64 `json:"success_percent"`
}

// StorageGrowthTrend represents storage growth data point.
type StorageGrowthTrend struct {
	Date           time.Time `json:"date"`
	TotalSize      int64     `json:"total_size"`
	RawSize        int64     `json:"raw_size"`
	SnapshotCount  int       `json:"snapshot_count"`
}

// BackupDurationTrend represents backup duration over time.
type BackupDurationTrend struct {
	Date           time.Time `json:"date"`
	AvgDurationMs  int64     `json:"avg_duration_ms"`
	MaxDurationMs  int64     `json:"max_duration_ms"`
	MinDurationMs  int64     `json:"min_duration_ms"`
	BackupCount    int       `json:"backup_count"`
}

// DailyBackupStats represents daily backup statistics.
type DailyBackupStats struct {
	Date       time.Time `json:"date"`
	Total      int       `json:"total"`
	Successful int       `json:"successful"`
	Failed     int       `json:"failed"`
	TotalSize  int64     `json:"total_size"`
}
