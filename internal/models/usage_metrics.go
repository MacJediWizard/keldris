// Package models defines the domain models for Keldris.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UsageMetrics represents a daily snapshot of organization usage.
type UsageMetrics struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	SnapshotDate time.Time `json:"snapshot_date"`

	// Agent counts
	AgentCount       int `json:"agent_count"`
	ActiveAgentCount int `json:"active_agent_count"`

	// User counts
	UserCount       int `json:"user_count"`
	ActiveUserCount int `json:"active_user_count"`

	// Storage metrics (in bytes)
	TotalStorageBytes  int64 `json:"total_storage_bytes"`
	BackupStorageBytes int64 `json:"backup_storage_bytes"`

	// Backup counts for the period
	BackupsCompleted int `json:"backups_completed"`
	BackupsFailed    int `json:"backups_failed"`
	BackupsTotal     int `json:"backups_total"`

	// Repository counts
	RepositoryCount int `json:"repository_count"`

	// Schedule counts
	ScheduleCount int `json:"schedule_count"`

	// Snapshot counts
	SnapshotCount int `json:"snapshot_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUsageMetrics creates a new UsageMetrics snapshot.
func NewUsageMetrics(orgID uuid.UUID, snapshotDate time.Time) *UsageMetrics {
	now := time.Now()
	return &UsageMetrics{
		ID:           uuid.New(),
		OrgID:        orgID,
		SnapshotDate: snapshotDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// OrgUsageLimits represents usage limits for an organization's billing tier.
type OrgUsageLimits struct {
	ID    uuid.UUID `json:"id"`
	OrgID uuid.UUID `json:"org_id"`

	// Agent limits
	MaxAgents *int `json:"max_agents,omitempty"` // nil means unlimited

	// User limits
	MaxUsers *int `json:"max_users,omitempty"` // nil means unlimited

	// Storage limits (in bytes)
	MaxStorageBytes *int64 `json:"max_storage_bytes,omitempty"` // nil means unlimited

	// Backup limits
	MaxBackupsPerMonth *int `json:"max_backups_per_month,omitempty"` // nil means unlimited

	// Repository limits
	MaxRepositories *int `json:"max_repositories,omitempty"` // nil means unlimited

	// Alert thresholds (percentage 0-100)
	WarningThreshold  int `json:"warning_threshold"`
	CriticalThreshold int `json:"critical_threshold"`

	// Billing tier info
	BillingTier        string     `json:"billing_tier"`
	BillingPeriodStart *time.Time `json:"billing_period_start,omitempty"`
	BillingPeriodEnd   *time.Time `json:"billing_period_end,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewOrgUsageLimits creates a new OrgUsageLimits with default values.
func NewOrgUsageLimits(orgID uuid.UUID) *OrgUsageLimits {
	now := time.Now()
	return &OrgUsageLimits{
		ID:                uuid.New(),
		OrgID:             orgID,
		WarningThreshold:  80,
		CriticalThreshold: 95,
		BillingTier:       "free",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// UsageAlertType represents the type of usage being alerted.
type UsageAlertType string

const (
	UsageAlertTypeAgents       UsageAlertType = "agents"
	UsageAlertTypeUsers        UsageAlertType = "users"
	UsageAlertTypeStorage      UsageAlertType = "storage"
	UsageAlertTypeBackups      UsageAlertType = "backups"
	UsageAlertTypeRepositories UsageAlertType = "repositories"
)

// UsageAlertSeverity represents the severity of a usage alert.
type UsageAlertSeverity string

const (
	UsageAlertSeverityWarning  UsageAlertSeverity = "warning"
	UsageAlertSeverityCritical UsageAlertSeverity = "critical"
	UsageAlertSeverityExceeded UsageAlertSeverity = "exceeded"
)

// UsageAlert represents an alert for approaching or exceeding usage limits.
type UsageAlert struct {
	ID             uuid.UUID          `json:"id"`
	OrgID          uuid.UUID          `json:"org_id"`
	AlertType      UsageAlertType     `json:"alert_type"`
	Severity       UsageAlertSeverity `json:"severity"`
	CurrentValue   int64              `json:"current_value"`
	LimitValue     int64              `json:"limit_value"`
	PercentageUsed float64            `json:"percentage_used"`
	Message        string             `json:"message"`
	Acknowledged   bool               `json:"acknowledged"`
	AcknowledgedBy *uuid.UUID         `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time         `json:"acknowledged_at,omitempty"`
	Resolved       bool               `json:"resolved"`
	ResolvedAt     *time.Time         `json:"resolved_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// NewUsageAlert creates a new UsageAlert.
func NewUsageAlert(orgID uuid.UUID, alertType UsageAlertType, severity UsageAlertSeverity, current, limit int64, message string) *UsageAlert {
	percentUsed := 0.0
	if limit > 0 {
		percentUsed = float64(current) / float64(limit) * 100
	}
	return &UsageAlert{
		ID:             uuid.New(),
		OrgID:          orgID,
		AlertType:      alertType,
		Severity:       severity,
		CurrentValue:   current,
		LimitValue:     limit,
		PercentageUsed: percentUsed,
		Message:        message,
		CreatedAt:      time.Now(),
	}
}

// Acknowledge marks the alert as acknowledged.
func (a *UsageAlert) Acknowledge(userID uuid.UUID) {
	now := time.Now()
	a.Acknowledged = true
	a.AcknowledgedBy = &userID
	a.AcknowledgedAt = &now
}

// Resolve marks the alert as resolved.
func (a *UsageAlert) Resolve() {
	now := time.Now()
	a.Resolved = true
	a.ResolvedAt = &now
}

// MonthlyUsageSummary represents aggregated monthly usage for billing.
type MonthlyUsageSummary struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	YearMonth string    `json:"year_month"` // Format: YYYY-MM

	// Peak values during the month
	PeakAgentCount   int   `json:"peak_agent_count"`
	PeakUserCount    int   `json:"peak_user_count"`
	PeakStorageBytes int64 `json:"peak_storage_bytes"`

	// Totals for the month
	TotalBackupsCompleted   int   `json:"total_backups_completed"`
	TotalBackupsFailed      int   `json:"total_backups_failed"`
	TotalDataBackedUpBytes  int64 `json:"total_data_backed_up_bytes"`

	// Average values
	AvgAgentCount   float64 `json:"avg_agent_count"`
	AvgStorageBytes int64   `json:"avg_storage_bytes"`

	// For billing calculations
	BillableAgentHours     int   `json:"billable_agent_hours"`
	BillableStorageGBHours int64 `json:"billable_storage_gb_hours"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewMonthlyUsageSummary creates a new MonthlyUsageSummary.
func NewMonthlyUsageSummary(orgID uuid.UUID, yearMonth string) *MonthlyUsageSummary {
	now := time.Now()
	return &MonthlyUsageSummary{
		ID:        uuid.New(),
		OrgID:     orgID,
		YearMonth: yearMonth,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CurrentUsage represents the current usage state for an organization.
type CurrentUsage struct {
	OrgID uuid.UUID `json:"org_id"`

	// Current counts
	AgentCount       int   `json:"agent_count"`
	ActiveAgentCount int   `json:"active_agent_count"`
	UserCount        int   `json:"user_count"`
	StorageBytes     int64 `json:"storage_bytes"`
	RepositoryCount  int   `json:"repository_count"`
	BackupsThisMonth int   `json:"backups_this_month"`

	// Limits (nil means unlimited)
	AgentLimit      *int   `json:"agent_limit,omitempty"`
	UserLimit       *int   `json:"user_limit,omitempty"`
	StorageLimit    *int64 `json:"storage_limit,omitempty"`
	RepositoryLimit *int   `json:"repository_limit,omitempty"`
	BackupLimit     *int   `json:"backup_limit,omitempty"`

	// Usage percentages
	AgentUsagePercent      *float64 `json:"agent_usage_percent,omitempty"`
	UserUsagePercent       *float64 `json:"user_usage_percent,omitempty"`
	StorageUsagePercent    *float64 `json:"storage_usage_percent,omitempty"`
	RepositoryUsagePercent *float64 `json:"repository_usage_percent,omitempty"`
	BackupUsagePercent     *float64 `json:"backup_usage_percent,omitempty"`

	// Billing info
	BillingTier        string     `json:"billing_tier"`
	BillingPeriodStart *time.Time `json:"billing_period_start,omitempty"`
	BillingPeriodEnd   *time.Time `json:"billing_period_end,omitempty"`

	// Active alerts
	ActiveAlerts []UsageAlert `json:"active_alerts,omitempty"`
}

// UsageHistoryPoint represents a point in usage history for charts.
type UsageHistoryPoint struct {
	Date             time.Time `json:"date"`
	AgentCount       int       `json:"agent_count"`
	UserCount        int       `json:"user_count"`
	StorageBytes     int64     `json:"storage_bytes"`
	BackupsCompleted int       `json:"backups_completed"`
	BackupsFailed    int       `json:"backups_failed"`
}

// BillingUsageReport represents usage data formatted for billing integration.
type BillingUsageReport struct {
	OrgID       uuid.UUID `json:"org_id"`
	OrgName     string    `json:"org_name"`
	BillingTier string    `json:"billing_tier"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	// Usage metrics
	PeakAgentCount   int   `json:"peak_agent_count"`
	PeakUserCount    int   `json:"peak_user_count"`
	PeakStorageBytes int64 `json:"peak_storage_bytes"`
	TotalBackups     int   `json:"total_backups"`
	AvgAgentCount    float64 `json:"avg_agent_count"`
	AvgStorageBytes  int64   `json:"avg_storage_bytes"`

	// Billable units
	BillableAgentHours     int   `json:"billable_agent_hours"`
	BillableStorageGBHours int64 `json:"billable_storage_gb_hours"`
}
