package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Downtime Event methods

// CreateDowntimeEvent creates a new downtime event.
func (db *DB) CreateDowntimeEvent(ctx context.Context, event *models.DowntimeEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO downtime_events (
			id, org_id, component_type, component_id, component_name,
			started_at, ended_at, duration_seconds, severity, cause, notes,
			resolved_by, auto_detected, alert_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, event.ID, event.OrgID, string(event.ComponentType), event.ComponentID, event.ComponentName,
		event.StartedAt, event.EndedAt, event.DurationSeconds, string(event.Severity), event.Cause, event.Notes,
		event.ResolvedBy, event.AutoDetected, event.AlertID, event.CreatedAt, event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create downtime event: %w", err)
	}
	return nil
}

// UpdateDowntimeEvent updates an existing downtime event.
func (db *DB) UpdateDowntimeEvent(ctx context.Context, event *models.DowntimeEvent) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE downtime_events SET
			ended_at = $2,
			duration_seconds = $3,
			severity = $4,
			cause = $5,
			notes = $6,
			resolved_by = $7,
			alert_id = $8,
			updated_at = $9
		WHERE id = $1
	`, event.ID, event.EndedAt, event.DurationSeconds, string(event.Severity), event.Cause, event.Notes,
		event.ResolvedBy, event.AlertID, event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update downtime event: %w", err)
	}
	return nil
}

// GetDowntimeEventByID returns a downtime event by ID.
func (db *DB) GetDowntimeEventByID(ctx context.Context, id uuid.UUID) (*models.DowntimeEvent, error) {
	var event models.DowntimeEvent
	var componentTypeStr string
	var severityStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       started_at, ended_at, duration_seconds, severity, cause, notes,
		       resolved_by, auto_detected, alert_id, created_at, updated_at
		FROM downtime_events
		WHERE id = $1
	`, id).Scan(
		&event.ID, &event.OrgID, &componentTypeStr, &event.ComponentID, &event.ComponentName,
		&event.StartedAt, &event.EndedAt, &event.DurationSeconds, &severityStr, &event.Cause, &event.Notes,
		&event.ResolvedBy, &event.AutoDetected, &event.AlertID, &event.CreatedAt, &event.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get downtime event: %w", err)
	}
	event.ComponentType = models.ComponentType(componentTypeStr)
	event.Severity = models.DowntimeSeverity(severityStr)
	return &event, nil
}

// GetDowntimeEventsByOrgID returns downtime events for an organization with pagination.
func (db *DB) GetDowntimeEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.DowntimeEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       started_at, ended_at, duration_seconds, severity, cause, notes,
		       resolved_by, auto_detected, alert_id, created_at, updated_at
		FROM downtime_events
		WHERE org_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list downtime events: %w", err)
	}
	defer rows.Close()

	return scanDowntimeEvents(rows)
}

// GetActiveDowntimeEventsByOrgID returns active (ongoing) downtime events for an organization.
func (db *DB) GetActiveDowntimeEventsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       started_at, ended_at, duration_seconds, severity, cause, notes,
		       resolved_by, auto_detected, alert_id, created_at, updated_at
		FROM downtime_events
		WHERE org_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active downtime events: %w", err)
	}
	defer rows.Close()

	return scanDowntimeEvents(rows)
}

// GetDowntimeEventsByComponent returns downtime events for a specific component.
func (db *DB) GetDowntimeEventsByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID) ([]*models.DowntimeEvent, error) {
	var rows pgx.Rows
	var err error

	if componentID != nil {
		rows, err = db.Pool.Query(ctx, `
			SELECT id, org_id, component_type, component_id, component_name,
			       started_at, ended_at, duration_seconds, severity, cause, notes,
			       resolved_by, auto_detected, alert_id, created_at, updated_at
			FROM downtime_events
			WHERE org_id = $1 AND component_type = $2 AND component_id = $3
			ORDER BY started_at DESC
		`, orgID, string(componentType), componentID)
	} else {
		rows, err = db.Pool.Query(ctx, `
			SELECT id, org_id, component_type, component_id, component_name,
			       started_at, ended_at, duration_seconds, severity, cause, notes,
			       resolved_by, auto_detected, alert_id, created_at, updated_at
			FROM downtime_events
			WHERE org_id = $1 AND component_type = $2
			ORDER BY started_at DESC
		`, orgID, string(componentType))
	}

	if err != nil {
		return nil, fmt.Errorf("list downtime events by component: %w", err)
	}
	defer rows.Close()

	return scanDowntimeEvents(rows)
}

// GetDowntimeEventsByTimeRange returns downtime events within a time range.
func (db *DB) GetDowntimeEventsByTimeRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.DowntimeEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       started_at, ended_at, duration_seconds, severity, cause, notes,
		       resolved_by, auto_detected, alert_id, created_at, updated_at
		FROM downtime_events
		WHERE org_id = $1 AND started_at <= $3 AND (ended_at IS NULL OR ended_at >= $2)
		ORDER BY started_at DESC
	`, orgID, start, end)
	if err != nil {
		return nil, fmt.Errorf("list downtime events by time range: %w", err)
	}
	defer rows.Close()

	return scanDowntimeEvents(rows)
}

// GetActiveDowntimeByComponent returns active downtime for a specific component.
func (db *DB) GetActiveDowntimeByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID uuid.UUID) (*models.DowntimeEvent, error) {
	var event models.DowntimeEvent
	var componentTypeStr string
	var severityStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       started_at, ended_at, duration_seconds, severity, cause, notes,
		       resolved_by, auto_detected, alert_id, created_at, updated_at
		FROM downtime_events
		WHERE org_id = $1 AND component_type = $2 AND component_id = $3 AND ended_at IS NULL
		LIMIT 1
	`, orgID, string(componentType), componentID).Scan(
		&event.ID, &event.OrgID, &componentTypeStr, &event.ComponentID, &event.ComponentName,
		&event.StartedAt, &event.EndedAt, &event.DurationSeconds, &severityStr, &event.Cause, &event.Notes,
		&event.ResolvedBy, &event.AutoDetected, &event.AlertID, &event.CreatedAt, &event.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active downtime by component: %w", err)
	}
	event.ComponentType = models.ComponentType(componentTypeStr)
	event.Severity = models.DowntimeSeverity(severityStr)
	return &event, nil
}

// DeleteDowntimeEvent deletes a downtime event.
func (db *DB) DeleteDowntimeEvent(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM downtime_events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete downtime event: %w", err)
	}
	return nil
}

// scanDowntimeEvents is a helper to scan multiple downtime events.
func scanDowntimeEvents(rows pgx.Rows) ([]*models.DowntimeEvent, error) {
	var events []*models.DowntimeEvent
	for rows.Next() {
		var event models.DowntimeEvent
		var componentTypeStr string
		var severityStr string
		err := rows.Scan(
			&event.ID, &event.OrgID, &componentTypeStr, &event.ComponentID, &event.ComponentName,
			&event.StartedAt, &event.EndedAt, &event.DurationSeconds, &severityStr, &event.Cause, &event.Notes,
			&event.ResolvedBy, &event.AutoDetected, &event.AlertID, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan downtime event: %w", err)
		}
		event.ComponentType = models.ComponentType(componentTypeStr)
		event.Severity = models.DowntimeSeverity(severityStr)
		events = append(events, &event)
	}
	return events, nil
}

// Uptime Stats methods

// CreateOrUpdateUptimeStats creates or updates uptime statistics.
func (db *DB) CreateOrUpdateUptimeStats(ctx context.Context, stats *models.UptimeStats) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO uptime_stats (
			id, org_id, component_type, component_id, component_name,
			period_start, period_end, total_seconds, downtime_seconds, uptime_percent, incident_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (org_id, component_type, component_id, period_start, period_end)
		DO UPDATE SET
			downtime_seconds = $9,
			uptime_percent = $10,
			incident_count = $11
	`, stats.ID, stats.OrgID, string(stats.ComponentType), stats.ComponentID, stats.ComponentName,
		stats.PeriodStart, stats.PeriodEnd, stats.TotalSeconds, stats.DowntimeSeconds, stats.UptimePercent, stats.IncidentCount, stats.CreatedAt)
	if err != nil {
		return fmt.Errorf("create/update uptime stats: %w", err)
	}
	return nil
}

// GetUptimeStatsByOrgID returns uptime statistics for an organization within a period.
func (db *DB) GetUptimeStatsByOrgID(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.UptimeStats, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       period_start, period_end, total_seconds, downtime_seconds, uptime_percent, incident_count, created_at
		FROM uptime_stats
		WHERE org_id = $1 AND period_start >= $2 AND period_end <= $3
		ORDER BY component_type, component_name
	`, orgID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("list uptime stats: %w", err)
	}
	defer rows.Close()

	var stats []*models.UptimeStats
	for rows.Next() {
		var s models.UptimeStats
		var componentTypeStr string
		err := rows.Scan(
			&s.ID, &s.OrgID, &componentTypeStr, &s.ComponentID, &s.ComponentName,
			&s.PeriodStart, &s.PeriodEnd, &s.TotalSeconds, &s.DowntimeSeconds, &s.UptimePercent, &s.IncidentCount, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan uptime stats: %w", err)
		}
		s.ComponentType = models.ComponentType(componentTypeStr)
		stats = append(stats, &s)
	}
	return stats, nil
}

// GetUptimeStatsByComponent returns uptime statistics for a specific component.
func (db *DB) GetUptimeStatsByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID, periodStart, periodEnd time.Time) (*models.UptimeStats, error) {
	var s models.UptimeStats
	var componentTypeStr string

	var err error
	if componentID != nil {
		err = db.Pool.QueryRow(ctx, `
			SELECT id, org_id, component_type, component_id, component_name,
			       period_start, period_end, total_seconds, downtime_seconds, uptime_percent, incident_count, created_at
			FROM uptime_stats
			WHERE org_id = $1 AND component_type = $2 AND component_id = $3 AND period_start = $4 AND period_end = $5
		`, orgID, string(componentType), componentID, periodStart, periodEnd).Scan(
			&s.ID, &s.OrgID, &componentTypeStr, &s.ComponentID, &s.ComponentName,
			&s.PeriodStart, &s.PeriodEnd, &s.TotalSeconds, &s.DowntimeSeconds, &s.UptimePercent, &s.IncidentCount, &s.CreatedAt,
		)
	} else {
		err = db.Pool.QueryRow(ctx, `
			SELECT id, org_id, component_type, component_id, component_name,
			       period_start, period_end, total_seconds, downtime_seconds, uptime_percent, incident_count, created_at
			FROM uptime_stats
			WHERE org_id = $1 AND component_type = $2 AND component_id IS NULL AND period_start = $3 AND period_end = $4
		`, orgID, string(componentType), periodStart, periodEnd).Scan(
			&s.ID, &s.OrgID, &componentTypeStr, &s.ComponentID, &s.ComponentName,
			&s.PeriodStart, &s.PeriodEnd, &s.TotalSeconds, &s.DowntimeSeconds, &s.UptimePercent, &s.IncidentCount, &s.CreatedAt,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("get uptime stats: %w", err)
	}
	s.ComponentType = models.ComponentType(componentTypeStr)
	return &s, nil
}

// Uptime Badge methods

// CreateOrUpdateUptimeBadge creates or updates an uptime badge.
func (db *DB) CreateOrUpdateUptimeBadge(ctx context.Context, badge *models.UptimeBadge) error {
	var componentTypeStr *string
	if badge.ComponentType != nil {
		str := string(*badge.ComponentType)
		componentTypeStr = &str
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO uptime_badges (
			id, org_id, component_type, component_id, component_name,
			badge_type, uptime_percent, last_updated, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (org_id, component_type, component_id, badge_type)
		DO UPDATE SET
			uptime_percent = $7,
			last_updated = $8
	`, badge.ID, badge.OrgID, componentTypeStr, badge.ComponentID, badge.ComponentName,
		string(badge.BadgeType), badge.UptimePercent, badge.LastUpdated, badge.CreatedAt)
	if err != nil {
		return fmt.Errorf("create/update uptime badge: %w", err)
	}
	return nil
}

// GetUptimeBadgesByOrgID returns all uptime badges for an organization.
func (db *DB) GetUptimeBadgesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UptimeBadge, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       badge_type, uptime_percent, last_updated, created_at
		FROM uptime_badges
		WHERE org_id = $1
		ORDER BY badge_type, component_type, component_name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list uptime badges: %w", err)
	}
	defer rows.Close()

	var badges []*models.UptimeBadge
	for rows.Next() {
		var b models.UptimeBadge
		var componentTypeStr *string
		var badgeTypeStr string
		err := rows.Scan(
			&b.ID, &b.OrgID, &componentTypeStr, &b.ComponentID, &b.ComponentName,
			&badgeTypeStr, &b.UptimePercent, &b.LastUpdated, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan uptime badge: %w", err)
		}
		if componentTypeStr != nil {
			ct := models.ComponentType(*componentTypeStr)
			b.ComponentType = &ct
		}
		b.BadgeType = models.BadgeType(badgeTypeStr)
		badges = append(badges, &b)
	}
	return badges, nil
}

// GetUptimeBadge returns a specific uptime badge.
func (db *DB) GetUptimeBadge(ctx context.Context, orgID uuid.UUID, componentType *models.ComponentType, componentID *uuid.UUID, badgeType models.BadgeType) (*models.UptimeBadge, error) {
	var b models.UptimeBadge
	var componentTypeStr *string
	var badgeTypeStr string

	var componentTypeQuery *string
	if componentType != nil {
		str := string(*componentType)
		componentTypeQuery = &str
	}

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, component_type, component_id, component_name,
		       badge_type, uptime_percent, last_updated, created_at
		FROM uptime_badges
		WHERE org_id = $1 AND
		      (component_type = $2 OR ($2 IS NULL AND component_type IS NULL)) AND
		      (component_id = $3 OR ($3 IS NULL AND component_id IS NULL)) AND
		      badge_type = $4
	`, orgID, componentTypeQuery, componentID, string(badgeType)).Scan(
		&b.ID, &b.OrgID, &componentTypeStr, &b.ComponentID, &b.ComponentName,
		&badgeTypeStr, &b.UptimePercent, &b.LastUpdated, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get uptime badge: %w", err)
	}
	if componentTypeStr != nil {
		ct := models.ComponentType(*componentTypeStr)
		b.ComponentType = &ct
	}
	b.BadgeType = models.BadgeType(badgeTypeStr)
	return &b, nil
}

// Downtime Alert methods

// CreateDowntimeAlert creates a new downtime alert.
func (db *DB) CreateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error {
	var componentTypeStr *string
	if alert.ComponentType != nil {
		str := string(*alert.ComponentType)
		componentTypeStr = &str
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO downtime_alerts (
			id, org_id, name, enabled, uptime_threshold, evaluation_period,
			component_type, notify_on_breach, notify_on_recovery, last_triggered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, alert.ID, alert.OrgID, alert.Name, alert.Enabled, alert.UptimeThreshold, alert.EvaluationPeriod,
		componentTypeStr, alert.NotifyOnBreach, alert.NotifyOnRecovery, alert.LastTriggeredAt, alert.CreatedAt, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create downtime alert: %w", err)
	}
	return nil
}

// UpdateDowntimeAlert updates an existing downtime alert.
func (db *DB) UpdateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error {
	var componentTypeStr *string
	if alert.ComponentType != nil {
		str := string(*alert.ComponentType)
		componentTypeStr = &str
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE downtime_alerts SET
			name = $2,
			enabled = $3,
			uptime_threshold = $4,
			evaluation_period = $5,
			component_type = $6,
			notify_on_breach = $7,
			notify_on_recovery = $8,
			last_triggered_at = $9,
			updated_at = $10
		WHERE id = $1
	`, alert.ID, alert.Name, alert.Enabled, alert.UptimeThreshold, alert.EvaluationPeriod,
		componentTypeStr, alert.NotifyOnBreach, alert.NotifyOnRecovery, alert.LastTriggeredAt, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update downtime alert: %w", err)
	}
	return nil
}

// GetDowntimeAlertByID returns a downtime alert by ID.
func (db *DB) GetDowntimeAlertByID(ctx context.Context, id uuid.UUID) (*models.DowntimeAlert, error) {
	var alert models.DowntimeAlert
	var componentTypeStr *string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, enabled, uptime_threshold, evaluation_period,
		       component_type, notify_on_breach, notify_on_recovery, last_triggered_at, created_at, updated_at
		FROM downtime_alerts
		WHERE id = $1
	`, id).Scan(
		&alert.ID, &alert.OrgID, &alert.Name, &alert.Enabled, &alert.UptimeThreshold, &alert.EvaluationPeriod,
		&componentTypeStr, &alert.NotifyOnBreach, &alert.NotifyOnRecovery, &alert.LastTriggeredAt, &alert.CreatedAt, &alert.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get downtime alert: %w", err)
	}
	if componentTypeStr != nil {
		ct := models.ComponentType(*componentTypeStr)
		alert.ComponentType = &ct
	}
	return &alert, nil
}

// GetDowntimeAlertsByOrgID returns all downtime alerts for an organization.
func (db *DB) GetDowntimeAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, enabled, uptime_threshold, evaluation_period,
		       component_type, notify_on_breach, notify_on_recovery, last_triggered_at, created_at, updated_at
		FROM downtime_alerts
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list downtime alerts: %w", err)
	}
	defer rows.Close()

	return scanDowntimeAlerts(rows)
}

// GetEnabledDowntimeAlertsByOrgID returns enabled downtime alerts for an organization.
func (db *DB) GetEnabledDowntimeAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, enabled, uptime_threshold, evaluation_period,
		       component_type, notify_on_breach, notify_on_recovery, last_triggered_at, created_at, updated_at
		FROM downtime_alerts
		WHERE org_id = $1 AND enabled = true
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list enabled downtime alerts: %w", err)
	}
	defer rows.Close()

	return scanDowntimeAlerts(rows)
}

// DeleteDowntimeAlert deletes a downtime alert.
func (db *DB) DeleteDowntimeAlert(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM downtime_alerts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete downtime alert: %w", err)
	}
	return nil
}

// scanDowntimeAlerts is a helper to scan multiple downtime alerts.
func scanDowntimeAlerts(rows pgx.Rows) ([]*models.DowntimeAlert, error) {
	var alerts []*models.DowntimeAlert
	for rows.Next() {
		var alert models.DowntimeAlert
		var componentTypeStr *string
		err := rows.Scan(
			&alert.ID, &alert.OrgID, &alert.Name, &alert.Enabled, &alert.UptimeThreshold, &alert.EvaluationPeriod,
			&componentTypeStr, &alert.NotifyOnBreach, &alert.NotifyOnRecovery, &alert.LastTriggeredAt, &alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan downtime alert: %w", err)
		}
		if componentTypeStr != nil {
			ct := models.ComponentType(*componentTypeStr)
			alert.ComponentType = &ct
		}
		alerts = append(alerts, &alert)
	}
	return alerts, nil
}
