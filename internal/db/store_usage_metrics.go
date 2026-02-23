package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Usage Metrics methods

// CreateUsageMetrics creates a new usage metrics snapshot.
func (db *DB) CreateUsageMetrics(ctx context.Context, metrics *models.UsageMetrics) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO usage_metrics (
			id, org_id, snapshot_date, agent_count, active_agent_count,
			user_count, active_user_count, total_storage_bytes, backup_storage_bytes,
			backups_completed, backups_failed, backups_total, repository_count,
			schedule_count, snapshot_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, metrics.ID, metrics.OrgID, metrics.SnapshotDate, metrics.AgentCount, metrics.ActiveAgentCount,
		metrics.UserCount, metrics.ActiveUserCount, metrics.TotalStorageBytes, metrics.BackupStorageBytes,
		metrics.BackupsCompleted, metrics.BackupsFailed, metrics.BackupsTotal, metrics.RepositoryCount,
		metrics.ScheduleCount, metrics.SnapshotCount, metrics.CreatedAt, metrics.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create usage metrics: %w", err)
	}
	return nil
}

// UpsertUsageMetrics creates or updates a usage metrics snapshot.
func (db *DB) UpsertUsageMetrics(ctx context.Context, metrics *models.UsageMetrics) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO usage_metrics (
			id, org_id, snapshot_date, agent_count, active_agent_count,
			user_count, active_user_count, total_storage_bytes, backup_storage_bytes,
			backups_completed, backups_failed, backups_total, repository_count,
			schedule_count, snapshot_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (org_id, snapshot_date) DO UPDATE SET
			agent_count = $4,
			active_agent_count = $5,
			user_count = $6,
			active_user_count = $7,
			total_storage_bytes = $8,
			backup_storage_bytes = $9,
			backups_completed = usage_metrics.backups_completed + $10,
			backups_failed = usage_metrics.backups_failed + $11,
			backups_total = usage_metrics.backups_total + $12,
			repository_count = $13,
			schedule_count = $14,
			snapshot_count = $15,
			updated_at = $17
	`, metrics.ID, metrics.OrgID, metrics.SnapshotDate, metrics.AgentCount, metrics.ActiveAgentCount,
		metrics.UserCount, metrics.ActiveUserCount, metrics.TotalStorageBytes, metrics.BackupStorageBytes,
		metrics.BackupsCompleted, metrics.BackupsFailed, metrics.BackupsTotal, metrics.RepositoryCount,
		metrics.ScheduleCount, metrics.SnapshotCount, metrics.CreatedAt, metrics.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert usage metrics: %w", err)
	}
	return nil
}

// GetUsageMetricsByOrgID returns usage metrics for an organization within a date range.
func (db *DB) GetUsageMetricsByOrgID(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]*models.UsageMetrics, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, snapshot_date, agent_count, active_agent_count,
		       user_count, active_user_count, total_storage_bytes, backup_storage_bytes,
		       backups_completed, backups_failed, backups_total, repository_count,
		       schedule_count, snapshot_count, created_at, updated_at
		FROM usage_metrics
		WHERE org_id = $1 AND snapshot_date >= $2 AND snapshot_date < $3
		ORDER BY snapshot_date ASC
	`, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("list usage metrics: %w", err)
	}
	defer rows.Close()

	return scanUsageMetrics(rows)
}

// GetLatestUsageMetrics returns the most recent usage metrics for an organization.
func (db *DB) GetLatestUsageMetrics(ctx context.Context, orgID uuid.UUID) (*models.UsageMetrics, error) {
	var m models.UsageMetrics
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_date, agent_count, active_agent_count,
		       user_count, active_user_count, total_storage_bytes, backup_storage_bytes,
		       backups_completed, backups_failed, backups_total, repository_count,
		       schedule_count, snapshot_count, created_at, updated_at
		FROM usage_metrics
		WHERE org_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, orgID).Scan(
		&m.ID, &m.OrgID, &m.SnapshotDate, &m.AgentCount, &m.ActiveAgentCount,
		&m.UserCount, &m.ActiveUserCount, &m.TotalStorageBytes, &m.BackupStorageBytes,
		&m.BackupsCompleted, &m.BackupsFailed, &m.BackupsTotal, &m.RepositoryCount,
		&m.ScheduleCount, &m.SnapshotCount, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest usage metrics: %w", err)
	}
	return &m, nil
}

// scanUsageMetrics is a helper to scan multiple usage metrics.
func scanUsageMetrics(rows pgx.Rows) ([]*models.UsageMetrics, error) {
	var metrics []*models.UsageMetrics
	for rows.Next() {
		var m models.UsageMetrics
		err := rows.Scan(
			&m.ID, &m.OrgID, &m.SnapshotDate, &m.AgentCount, &m.ActiveAgentCount,
			&m.UserCount, &m.ActiveUserCount, &m.TotalStorageBytes, &m.BackupStorageBytes,
			&m.BackupsCompleted, &m.BackupsFailed, &m.BackupsTotal, &m.RepositoryCount,
			&m.ScheduleCount, &m.SnapshotCount, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan usage metrics: %w", err)
		}
		metrics = append(metrics, &m)
	}
	return metrics, nil
}

// Organization Usage Limits methods

// GetOrgUsageLimits returns usage limits for an organization.
func (db *DB) GetOrgUsageLimits(ctx context.Context, orgID uuid.UUID) (*models.OrgUsageLimits, error) {
	var l models.OrgUsageLimits
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, max_agents, max_users, max_storage_bytes,
		       max_backups_per_month, max_repositories, warning_threshold,
		       critical_threshold, billing_tier, billing_period_start,
		       billing_period_end, created_at, updated_at
		FROM org_usage_limits
		WHERE org_id = $1
	`, orgID).Scan(
		&l.ID, &l.OrgID, &l.MaxAgents, &l.MaxUsers, &l.MaxStorageBytes,
		&l.MaxBackupsPerMonth, &l.MaxRepositories, &l.WarningThreshold,
		&l.CriticalThreshold, &l.BillingTier, &l.BillingPeriodStart,
		&l.BillingPeriodEnd, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get org usage limits: %w", err)
	}
	return &l, nil
}

// CreateOrgUsageLimits creates new usage limits for an organization.
func (db *DB) CreateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_usage_limits (
			id, org_id, max_agents, max_users, max_storage_bytes,
			max_backups_per_month, max_repositories, warning_threshold,
			critical_threshold, billing_tier, billing_period_start,
			billing_period_end, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, limits.ID, limits.OrgID, limits.MaxAgents, limits.MaxUsers, limits.MaxStorageBytes,
		limits.MaxBackupsPerMonth, limits.MaxRepositories, limits.WarningThreshold,
		limits.CriticalThreshold, limits.BillingTier, limits.BillingPeriodStart,
		limits.BillingPeriodEnd, limits.CreatedAt, limits.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create org usage limits: %w", err)
	}
	return nil
}

// UpdateOrgUsageLimits updates usage limits for an organization.
func (db *DB) UpdateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error {
	limits.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_usage_limits SET
			max_agents = $2,
			max_users = $3,
			max_storage_bytes = $4,
			max_backups_per_month = $5,
			max_repositories = $6,
			warning_threshold = $7,
			critical_threshold = $8,
			billing_tier = $9,
			billing_period_start = $10,
			billing_period_end = $11,
			updated_at = $12
		WHERE id = $1
	`, limits.ID, limits.MaxAgents, limits.MaxUsers, limits.MaxStorageBytes,
		limits.MaxBackupsPerMonth, limits.MaxRepositories, limits.WarningThreshold,
		limits.CriticalThreshold, limits.BillingTier, limits.BillingPeriodStart,
		limits.BillingPeriodEnd, limits.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update org usage limits: %w", err)
	}
	return nil
}

// UpsertOrgUsageLimits creates or updates usage limits for an organization.
func (db *DB) UpsertOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_usage_limits (
			id, org_id, max_agents, max_users, max_storage_bytes,
			max_backups_per_month, max_repositories, warning_threshold,
			critical_threshold, billing_tier, billing_period_start,
			billing_period_end, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (org_id) DO UPDATE SET
			max_agents = $3,
			max_users = $4,
			max_storage_bytes = $5,
			max_backups_per_month = $6,
			max_repositories = $7,
			warning_threshold = $8,
			critical_threshold = $9,
			billing_tier = $10,
			billing_period_start = $11,
			billing_period_end = $12,
			updated_at = $14
	`, limits.ID, limits.OrgID, limits.MaxAgents, limits.MaxUsers, limits.MaxStorageBytes,
		limits.MaxBackupsPerMonth, limits.MaxRepositories, limits.WarningThreshold,
		limits.CriticalThreshold, limits.BillingTier, limits.BillingPeriodStart,
		limits.BillingPeriodEnd, limits.CreatedAt, limits.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert org usage limits: %w", err)
	}
	return nil
}

// Usage Alerts methods

// CreateUsageAlert creates a new usage alert.
func (db *DB) CreateUsageAlert(ctx context.Context, alert *models.UsageAlert) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO usage_alerts (
			id, org_id, alert_type, severity, current_value, limit_value,
			percentage_used, message, acknowledged, acknowledged_by,
			acknowledged_at, resolved, resolved_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, alert.ID, alert.OrgID, string(alert.AlertType), string(alert.Severity),
		alert.CurrentValue, alert.LimitValue, alert.PercentageUsed, alert.Message,
		alert.Acknowledged, alert.AcknowledgedBy, alert.AcknowledgedAt,
		alert.Resolved, alert.ResolvedAt, alert.CreatedAt)
	if err != nil {
		return fmt.Errorf("create usage alert: %w", err)
	}
	return nil
}

// GetActiveUsageAlertsByOrgID returns active (unresolved) usage alerts for an organization.
func (db *DB) GetActiveUsageAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UsageAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, alert_type, severity, current_value, limit_value,
		       percentage_used, message, acknowledged, acknowledged_by,
		       acknowledged_at, resolved, resolved_at, created_at
		FROM usage_alerts
		WHERE org_id = $1 AND resolved = false
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active usage alerts: %w", err)
	}
	defer rows.Close()

	return scanUsageAlerts(rows)
}

// GetActiveUsageAlertByType returns the active alert of a specific type for an organization.
func (db *DB) GetActiveUsageAlertByType(ctx context.Context, orgID uuid.UUID, alertType models.UsageAlertType) (*models.UsageAlert, error) {
	var a models.UsageAlert
	var alertTypeStr, severityStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, alert_type, severity, current_value, limit_value,
		       percentage_used, message, acknowledged, acknowledged_by,
		       acknowledged_at, resolved, resolved_at, created_at
		FROM usage_alerts
		WHERE org_id = $1 AND alert_type = $2 AND resolved = false
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID, string(alertType)).Scan(
		&a.ID, &a.OrgID, &alertTypeStr, &severityStr, &a.CurrentValue, &a.LimitValue,
		&a.PercentageUsed, &a.Message, &a.Acknowledged, &a.AcknowledgedBy,
		&a.AcknowledgedAt, &a.Resolved, &a.ResolvedAt, &a.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active usage alert by type: %w", err)
	}
	a.AlertType = models.UsageAlertType(alertTypeStr)
	a.Severity = models.UsageAlertSeverity(severityStr)
	return &a, nil
}

// UpdateUsageAlert updates an existing usage alert.
func (db *DB) UpdateUsageAlert(ctx context.Context, alert *models.UsageAlert) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE usage_alerts SET
			severity = $2,
			current_value = $3,
			limit_value = $4,
			percentage_used = $5,
			message = $6,
			acknowledged = $7,
			acknowledged_by = $8,
			acknowledged_at = $9,
			resolved = $10,
			resolved_at = $11
		WHERE id = $1
	`, alert.ID, string(alert.Severity), alert.CurrentValue, alert.LimitValue,
		alert.PercentageUsed, alert.Message, alert.Acknowledged, alert.AcknowledgedBy,
		alert.AcknowledgedAt, alert.Resolved, alert.ResolvedAt)
	if err != nil {
		return fmt.Errorf("update usage alert: %w", err)
	}
	return nil
}

// AcknowledgeUsageAlert marks a usage alert as acknowledged.
func (db *DB) AcknowledgeUsageAlert(ctx context.Context, id, userID uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE usage_alerts SET
			acknowledged = true,
			acknowledged_by = $2,
			acknowledged_at = $3
		WHERE id = $1
	`, id, userID, now)
	if err != nil {
		return fmt.Errorf("acknowledge usage alert: %w", err)
	}
	return nil
}

// ResolveUsageAlert marks a usage alert as resolved.
func (db *DB) ResolveUsageAlert(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE usage_alerts SET
			resolved = true,
			resolved_at = $2
		WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("resolve usage alert: %w", err)
	}
	return nil
}

// scanUsageAlerts is a helper to scan multiple usage alerts.
func scanUsageAlerts(rows pgx.Rows) ([]*models.UsageAlert, error) {
	var alerts []*models.UsageAlert
	for rows.Next() {
		var a models.UsageAlert
		var alertTypeStr, severityStr string
		err := rows.Scan(
			&a.ID, &a.OrgID, &alertTypeStr, &severityStr, &a.CurrentValue, &a.LimitValue,
			&a.PercentageUsed, &a.Message, &a.Acknowledged, &a.AcknowledgedBy,
			&a.AcknowledgedAt, &a.Resolved, &a.ResolvedAt, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan usage alert: %w", err)
		}
		a.AlertType = models.UsageAlertType(alertTypeStr)
		a.Severity = models.UsageAlertSeverity(severityStr)
		alerts = append(alerts, &a)
	}
	return alerts, nil
}

// Monthly Usage Summary methods

// UpsertMonthlyUsageSummary creates or updates a monthly usage summary.
func (db *DB) UpsertMonthlyUsageSummary(ctx context.Context, summary *models.MonthlyUsageSummary) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO monthly_usage_summary (
			id, org_id, year_month, peak_agent_count, peak_user_count,
			peak_storage_bytes, total_backups_completed, total_backups_failed,
			total_data_backed_up_bytes, avg_agent_count, avg_storage_bytes,
			billable_agent_hours, billable_storage_gb_hours, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (org_id, year_month) DO UPDATE SET
			peak_agent_count = GREATEST(monthly_usage_summary.peak_agent_count, $4),
			peak_user_count = GREATEST(monthly_usage_summary.peak_user_count, $5),
			peak_storage_bytes = GREATEST(monthly_usage_summary.peak_storage_bytes, $6),
			total_backups_completed = $7,
			total_backups_failed = $8,
			total_data_backed_up_bytes = $9,
			avg_agent_count = $10,
			avg_storage_bytes = $11,
			billable_agent_hours = $12,
			billable_storage_gb_hours = $13,
			updated_at = $15
	`, summary.ID, summary.OrgID, summary.YearMonth, summary.PeakAgentCount, summary.PeakUserCount,
		summary.PeakStorageBytes, summary.TotalBackupsCompleted, summary.TotalBackupsFailed,
		summary.TotalDataBackedUpBytes, summary.AvgAgentCount, summary.AvgStorageBytes,
		summary.BillableAgentHours, summary.BillableStorageGBHours, summary.CreatedAt, summary.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert monthly usage summary: %w", err)
	}
	return nil
}

// GetMonthlyUsageSummary returns the monthly usage summary for an organization.
func (db *DB) GetMonthlyUsageSummary(ctx context.Context, orgID uuid.UUID, yearMonth string) (*models.MonthlyUsageSummary, error) {
	var s models.MonthlyUsageSummary
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, year_month, peak_agent_count, peak_user_count,
		       peak_storage_bytes, total_backups_completed, total_backups_failed,
		       total_data_backed_up_bytes, avg_agent_count, avg_storage_bytes,
		       billable_agent_hours, billable_storage_gb_hours, created_at, updated_at
		FROM monthly_usage_summary
		WHERE org_id = $1 AND year_month = $2
	`, orgID, yearMonth).Scan(
		&s.ID, &s.OrgID, &s.YearMonth, &s.PeakAgentCount, &s.PeakUserCount,
		&s.PeakStorageBytes, &s.TotalBackupsCompleted, &s.TotalBackupsFailed,
		&s.TotalDataBackedUpBytes, &s.AvgAgentCount, &s.AvgStorageBytes,
		&s.BillableAgentHours, &s.BillableStorageGBHours, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get monthly usage summary: %w", err)
	}
	return &s, nil
}

// GetMonthlyUsageSummariesByOrgID returns monthly usage summaries for an organization.
func (db *DB) GetMonthlyUsageSummariesByOrgID(ctx context.Context, orgID uuid.UUID, months int) ([]*models.MonthlyUsageSummary, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, year_month, peak_agent_count, peak_user_count,
		       peak_storage_bytes, total_backups_completed, total_backups_failed,
		       total_data_backed_up_bytes, avg_agent_count, avg_storage_bytes,
		       billable_agent_hours, billable_storage_gb_hours, created_at, updated_at
		FROM monthly_usage_summary
		WHERE org_id = $1
		ORDER BY year_month DESC
		LIMIT $2
	`, orgID, months)
	if err != nil {
		return nil, fmt.Errorf("list monthly usage summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*models.MonthlyUsageSummary
	for rows.Next() {
		var s models.MonthlyUsageSummary
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.YearMonth, &s.PeakAgentCount, &s.PeakUserCount,
			&s.PeakStorageBytes, &s.TotalBackupsCompleted, &s.TotalBackupsFailed,
			&s.TotalDataBackedUpBytes, &s.AvgAgentCount, &s.AvgStorageBytes,
			&s.BillableAgentHours, &s.BillableStorageGBHours, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan monthly usage summary: %w", err)
		}
		summaries = append(summaries, &s)
	}
	return summaries, nil
}

// Additional helper methods for metering service

// GetAgentCountByOrgID returns the total number of agents for an organization.
func (db *DB) GetAgentCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM agents WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents: %w", err)
	}
	return count, nil
}

// GetActiveAgentCountByOrgID returns the number of active agents for an organization.
func (db *DB) GetActiveAgentCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM agents
		WHERE org_id = $1 AND last_seen_at > NOW() - INTERVAL '24 hours'
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active agents: %w", err)
	}
	return count, nil
}

// GetUserCountByOrgID returns the total number of users for an organization.
func (db *DB) GetUserCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

// GetActiveUserCountByOrgID returns the number of active users for an organization.
func (db *DB) GetActiveUserCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE org_id = $1 AND last_login_at > NOW() - INTERVAL '30 days'
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active users: %w", err)
	}
	return count, nil
}

// GetTotalStorageByOrgID returns the total storage used by an organization in bytes.
func (db *DB) GetTotalStorageByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var storage int64
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(ss.total_size), 0)
		FROM storage_stats ss
		JOIN repositories r ON r.id = ss.repository_id
		WHERE r.org_id = $1
	`, orgID).Scan(&storage)
	if err != nil {
		return 0, fmt.Errorf("sum storage: %w", err)
	}
	return storage, nil
}

// GetBackupCountByOrgIDForPeriod returns backup counts for an organization within a time period.
func (db *DB) GetBackupCountByOrgIDForPeriod(ctx context.Context, orgID uuid.UUID, start, end time.Time) (completed, failed int, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE b.status = 'completed'),
			COUNT(*) FILTER (WHERE b.status = 'failed')
		FROM backups b
		JOIN agents a ON a.id = b.agent_id
		WHERE a.org_id = $1 AND b.started_at >= $2 AND b.started_at < $3
	`, orgID, start, end).Scan(&completed, &failed)
	if err != nil {
		return 0, 0, fmt.Errorf("count backups for period: %w", err)
	}
	return completed, failed, nil
}

// GetBackupsThisMonthByOrgID returns the number of backups this month for an organization.
func (db *DB) GetBackupsThisMonthByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN agents a ON a.id = b.agent_id
		WHERE a.org_id = $1
		AND b.started_at >= DATE_TRUNC('month', CURRENT_DATE)
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count backups this month: %w", err)
	}
	return count, nil
}

// GetRepositoryCountByOrgID returns the number of repositories for an organization.
func (db *DB) GetRepositoryCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM repositories WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count repositories: %w", err)
	}
	return count, nil
}

// GetScheduleCountByOrgID returns the number of schedules for an organization.
func (db *DB) GetScheduleCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM schedules s JOIN agents a ON a.id = s.agent_id WHERE a.org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count schedules: %w", err)
	}
	return count, nil
}

// GetSnapshotCountByOrgID returns the total number of snapshots for an organization.
func (db *DB) GetSnapshotCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(ss.snapshot_count), 0)
		FROM storage_stats ss
		JOIN repositories r ON r.id = ss.repository_id
		WHERE r.org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count snapshots: %w", err)
	}
	return count, nil
}
