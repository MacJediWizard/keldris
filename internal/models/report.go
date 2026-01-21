package models

import (
	"time"

	"github.com/google/uuid"
)

// ReportFrequency represents how often a report is sent.
type ReportFrequency string

const (
	ReportFrequencyDaily   ReportFrequency = "daily"
	ReportFrequencyWeekly  ReportFrequency = "weekly"
	ReportFrequencyMonthly ReportFrequency = "monthly"
)

// ReportStatus represents the status of a sent report.
type ReportStatus string

const (
	ReportStatusSent    ReportStatus = "sent"
	ReportStatusFailed  ReportStatus = "failed"
	ReportStatusPreview ReportStatus = "preview"
)

// ReportSchedule represents a scheduled report configuration.
type ReportSchedule struct {
	ID         uuid.UUID       `json:"id"`
	OrgID      uuid.UUID       `json:"org_id"`
	Name       string          `json:"name"`
	Frequency  ReportFrequency `json:"frequency"`
	Recipients []string        `json:"recipients"`
	ChannelID  *uuid.UUID      `json:"channel_id,omitempty"`
	Timezone   string          `json:"timezone"`
	Enabled    bool            `json:"enabled"`
	LastSentAt *time.Time      `json:"last_sent_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// NewReportSchedule creates a new report schedule.
func NewReportSchedule(orgID uuid.UUID, name string, frequency ReportFrequency, recipients []string) *ReportSchedule {
	now := time.Now()
	return &ReportSchedule{
		ID:         uuid.New(),
		OrgID:      orgID,
		Name:       name,
		Frequency:  frequency,
		Recipients: recipients,
		Timezone:   "UTC",
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// ReportHistory represents a historical record of a sent report.
type ReportHistory struct {
	ID           uuid.UUID    `json:"id"`
	OrgID        uuid.UUID    `json:"org_id"`
	ScheduleID   *uuid.UUID   `json:"schedule_id,omitempty"`
	ReportType   string       `json:"report_type"`
	PeriodStart  time.Time    `json:"period_start"`
	PeriodEnd    time.Time    `json:"period_end"`
	Recipients   []string     `json:"recipients"`
	Status       ReportStatus `json:"status"`
	ErrorMessage string       `json:"error_message,omitempty"`
	ReportData   *ReportData  `json:"report_data,omitempty"`
	SentAt       *time.Time   `json:"sent_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// NewReportHistory creates a new report history entry.
func NewReportHistory(orgID uuid.UUID, scheduleID *uuid.UUID, reportType string, periodStart, periodEnd time.Time, recipients []string) *ReportHistory {
	return &ReportHistory{
		ID:          uuid.New(),
		OrgID:       orgID,
		ScheduleID:  scheduleID,
		ReportType:  reportType,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Recipients:  recipients,
		Status:      ReportStatusSent,
		CreatedAt:   time.Now(),
	}
}

// MarkSent marks the report as sent.
func (r *ReportHistory) MarkSent() {
	now := time.Now()
	r.Status = ReportStatusSent
	r.SentAt = &now
}

// MarkFailed marks the report as failed.
func (r *ReportHistory) MarkFailed(errMsg string) {
	r.Status = ReportStatusFailed
	r.ErrorMessage = errMsg
}

// ReportData contains the aggregated data for a report.
type ReportData struct {
	BackupSummary  BackupSummary  `json:"backup_summary"`
	StorageSummary StorageSummary `json:"storage_summary"`
	AgentSummary   AgentSummary   `json:"agent_summary"`
	AlertSummary   AlertSummary   `json:"alert_summary"`
	TopIssues      []ReportIssue  `json:"top_issues,omitempty"`
}

// BackupSummary contains backup statistics for the report period.
type BackupSummary struct {
	TotalBackups      int     `json:"total_backups"`
	SuccessfulBackups int     `json:"successful_backups"`
	FailedBackups     int     `json:"failed_backups"`
	SuccessRate       float64 `json:"success_rate"`
	TotalDataBacked   int64   `json:"total_data_backed"`
	SchedulesActive   int     `json:"schedules_active"`
}

// StorageSummary contains storage statistics.
type StorageSummary struct {
	TotalRawSize     int64   `json:"total_raw_size"`
	TotalRestoreSize int64   `json:"total_restore_size"`
	SpaceSaved       int64   `json:"space_saved"`
	SpaceSavedPct    float64 `json:"space_saved_pct"`
	RepositoryCount  int     `json:"repository_count"`
	TotalSnapshots   int     `json:"total_snapshots"`
}

// AgentSummary contains agent health statistics.
type AgentSummary struct {
	TotalAgents   int `json:"total_agents"`
	ActiveAgents  int `json:"active_agents"`
	OfflineAgents int `json:"offline_agents"`
	PendingAgents int `json:"pending_agents"`
}

// AlertSummary contains alert statistics.
type AlertSummary struct {
	TotalAlerts        int `json:"total_alerts"`
	CriticalAlerts     int `json:"critical_alerts"`
	WarningAlerts      int `json:"warning_alerts"`
	AcknowledgedAlerts int `json:"acknowledged_alerts"`
	ResolvedAlerts     int `json:"resolved_alerts"`
}

// ReportIssue represents a notable issue to highlight in the report.
type ReportIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// CreateReportScheduleRequest represents a request to create a report schedule.
type CreateReportScheduleRequest struct {
	Name       string   `json:"name" binding:"required,min=1,max=255"`
	Frequency  string   `json:"frequency" binding:"required"`
	Recipients []string `json:"recipients" binding:"required,min=1"`
	ChannelID  *string  `json:"channel_id,omitempty"`
	Timezone   string   `json:"timezone"`
	Enabled    *bool    `json:"enabled,omitempty"`
}

// UpdateReportScheduleRequest represents a request to update a report schedule.
type UpdateReportScheduleRequest struct {
	Name       *string  `json:"name,omitempty"`
	Frequency  *string  `json:"frequency,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	ChannelID  *string  `json:"channel_id,omitempty"`
	Timezone   *string  `json:"timezone,omitempty"`
	Enabled    *bool    `json:"enabled,omitempty"`
}

// SendReportRequest represents a request to manually send a report.
type SendReportRequest struct {
	Recipients []string `json:"recipients,omitempty"`
	Preview    bool     `json:"preview"`
}

// PreviewReportRequest represents a request to preview a report.
type PreviewReportRequest struct {
	Frequency string `json:"frequency" binding:"required"`
	Timezone  string `json:"timezone"`
}
