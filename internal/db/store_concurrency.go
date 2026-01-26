package db

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Backup Queue Methods

// CreateBackupQueueEntry creates a new backup queue entry.
func (db *DB) CreateBackupQueueEntry(ctx context.Context, entry *models.BackupQueueEntry) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backup_queue (id, org_id, agent_id, schedule_id, priority, queued_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.ID, entry.OrgID, entry.AgentID, entry.ScheduleID, entry.Priority, entry.QueuedAt, entry.Status, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("create backup queue entry: %w", err)
	}
	return nil
}

// GetQueuedBackupsByOrg returns all queued backups for an organization.
func (db *DB) GetQueuedBackupsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.BackupQueueEntry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, schedule_id, priority, queued_at, started_at, status, created_at
		FROM backup_queue
		WHERE org_id = $1 AND status = 'queued'
		ORDER BY priority DESC, queued_at ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get queued backups by org: %w", err)
	}
	defer rows.Close()

	var entries []*models.BackupQueueEntry
	position := 1
	for rows.Next() {
		var e models.BackupQueueEntry
		var statusStr string
		err := rows.Scan(&e.ID, &e.OrgID, &e.AgentID, &e.ScheduleID, &e.Priority, &e.QueuedAt, &e.StartedAt, &statusStr, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan backup queue entry: %w", err)
		}
		e.Status = models.ConcurrencyQueueStatus(statusStr)
		e.QueuePosition = position
		position++
		entries = append(entries, &e)
	}

	return entries, nil
}

// GetQueuedBackupsByAgent returns all queued backups for an agent.
func (db *DB) GetQueuedBackupsByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupQueueEntry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, schedule_id, priority, queued_at, started_at, status, created_at
		FROM backup_queue
		WHERE agent_id = $1 AND status = 'queued'
		ORDER BY priority DESC, queued_at ASC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("get queued backups by agent: %w", err)
	}
	defer rows.Close()

	var entries []*models.BackupQueueEntry
	position := 1
	for rows.Next() {
		var e models.BackupQueueEntry
		var statusStr string
		err := rows.Scan(&e.ID, &e.OrgID, &e.AgentID, &e.ScheduleID, &e.Priority, &e.QueuedAt, &e.StartedAt, &statusStr, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan backup queue entry: %w", err)
		}
		e.Status = models.ConcurrencyQueueStatus(statusStr)
		e.QueuePosition = position
		position++
		entries = append(entries, &e)
	}

	return entries, nil
}

// GetOldestQueuedBackup returns the oldest queued backup for an organization.
func (db *DB) GetOldestQueuedBackup(ctx context.Context, orgID uuid.UUID) (*models.BackupQueueEntry, error) {
	var e models.BackupQueueEntry
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, schedule_id, priority, queued_at, started_at, status, created_at
		FROM backup_queue
		WHERE org_id = $1 AND status = 'queued'
		ORDER BY priority DESC, queued_at ASC
		LIMIT 1
	`, orgID).Scan(&e.ID, &e.OrgID, &e.AgentID, &e.ScheduleID, &e.Priority, &e.QueuedAt, &e.StartedAt, &statusStr, &e.CreatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get oldest queued backup: %w", err)
	}
	e.Status = models.ConcurrencyQueueStatus(statusStr)
	return &e, nil
}

// UpdateBackupQueueEntry updates a backup queue entry.
func (db *DB) UpdateBackupQueueEntry(ctx context.Context, entry *models.BackupQueueEntry) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE backup_queue
		SET status = $2, started_at = $3
		WHERE id = $1
	`, entry.ID, entry.Status, entry.StartedAt)
	if err != nil {
		return fmt.Errorf("update backup queue entry: %w", err)
	}
	return nil
}

// DeleteBackupQueueEntry deletes a backup queue entry.
func (db *DB) DeleteBackupQueueEntry(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM backup_queue WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup queue entry: %w", err)
	}
	return nil
}

// GetQueuePosition returns the queue position for a given entry.
func (db *DB) GetQueuePosition(ctx context.Context, orgID, entryID uuid.UUID) (int, error) {
	var position int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) + 1
		FROM backup_queue
		WHERE org_id = $1
		  AND status = 'queued'
		  AND (priority > (SELECT priority FROM backup_queue WHERE id = $2)
		       OR (priority = (SELECT priority FROM backup_queue WHERE id = $2)
		           AND queued_at < (SELECT queued_at FROM backup_queue WHERE id = $2)))
	`, orgID, entryID).Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("get queue position: %w", err)
	}
	return position, nil
}

// GetRunningBackupsCountByOrg returns the count of running backups for an organization.
func (db *DB) GetRunningBackupsCountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.status = 'running'
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get running backups count by org: %w", err)
	}
	return count, nil
}

// GetRunningBackupsCountByAgent returns the count of running backups for an agent.
func (db *DB) GetRunningBackupsCountByAgent(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups
		WHERE agent_id = $1 AND status = 'running'
	`, agentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get running backups count by agent: %w", err)
	}
	return count, nil
}

// GetConcurrencyQueueSummary returns queue statistics for an organization.
func (db *DB) GetConcurrencyQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.ConcurrencyQueueSummary, error) {
	summary := &models.ConcurrencyQueueSummary{
		ByOrg:   make(map[uuid.UUID]int),
		ByAgent: make(map[uuid.UUID]int),
	}

	// Get total queued
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), MIN(queued_at)
		FROM backup_queue
		WHERE org_id = $1 AND status = 'queued'
	`, orgID).Scan(&summary.TotalQueued, &summary.OldestQueued)
	if err != nil {
		return nil, fmt.Errorf("get queue summary: %w", err)
	}

	// Get counts by agent
	rows, err := db.Pool.Query(ctx, `
		SELECT agent_id, COUNT(*)
		FROM backup_queue
		WHERE org_id = $1 AND status = 'queued'
		GROUP BY agent_id
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get queue summary by agent: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agentID uuid.UUID
		var count int
		if err := rows.Scan(&agentID, &count); err != nil {
			return nil, fmt.Errorf("scan agent queue count: %w", err)
		}
		summary.ByAgent[agentID] = count
	}

	// Calculate average wait time from started entries
	var avgWait *float64
	err = db.Pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (started_at - queued_at)) / 60)
		FROM backup_queue
		WHERE org_id = $1 AND status = 'started' AND started_at IS NOT NULL
		AND started_at > NOW() - INTERVAL '24 hours'
	`, orgID).Scan(&avgWait)
	if err == nil && avgWait != nil {
		summary.AvgWaitMinutes = *avgWait
	}

	return summary, nil
}

// GetQueuedBackupsWithDetails returns queued backups with schedule and agent details.
func (db *DB) GetQueuedBackupsWithDetails(ctx context.Context, orgID uuid.UUID) ([]*models.BackupQueueEntryWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT bq.id, bq.org_id, bq.agent_id, bq.schedule_id, bq.priority, bq.queued_at,
		       bq.started_at, bq.status, bq.created_at,
		       s.name as schedule_name, a.hostname as agent_hostname
		FROM backup_queue bq
		JOIN schedules s ON bq.schedule_id = s.id
		JOIN agents a ON bq.agent_id = a.id
		WHERE bq.org_id = $1 AND bq.status = 'queued'
		ORDER BY bq.priority DESC, bq.queued_at ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get queued backups with details: %w", err)
	}
	defer rows.Close()

	var entries []*models.BackupQueueEntryWithDetails
	position := 1
	for rows.Next() {
		var e models.BackupQueueEntryWithDetails
		var statusStr string
		err := rows.Scan(&e.ID, &e.OrgID, &e.AgentID, &e.ScheduleID, &e.Priority, &e.QueuedAt,
			&e.StartedAt, &statusStr, &e.CreatedAt,
			&e.ScheduleName, &e.AgentName)
		if err != nil {
			return nil, fmt.Errorf("scan backup queue entry with details: %w", err)
		}
		e.Status = models.ConcurrencyQueueStatus(statusStr)
		e.QueuePosition = position
		position++
		entries = append(entries, &e)
	}

	return entries, nil
}

// Concurrency Limit Update Methods

// UpdateOrganizationConcurrencyLimit updates the max concurrent backups limit for an organization.
func (db *DB) UpdateOrganizationConcurrencyLimit(ctx context.Context, orgID uuid.UUID, limit *int) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations
		SET max_concurrent_backups = $2, updated_at = NOW()
		WHERE id = $1
	`, orgID, limit)
	if err != nil {
		return fmt.Errorf("update organization concurrency limit: %w", err)
	}
	return nil
}

// UpdateAgentConcurrencyLimit updates the max concurrent backups limit for an agent.
func (db *DB) UpdateAgentConcurrencyLimit(ctx context.Context, agentID uuid.UUID, limit *int) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET max_concurrent_backups = $2, updated_at = NOW()
		WHERE id = $1
	`, agentID, limit)
	if err != nil {
		return fmt.Errorf("update agent concurrency limit: %w", err)
	}
	return nil
}

// GetOrganizationByIDWithConcurrency returns an organization by ID including concurrency settings.
func (db *DB) GetOrganizationByIDWithConcurrency(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, max_concurrent_backups, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.MaxConcurrentBackups, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization by ID: %w", err)
	}
	return &org, nil
}

// GetAgentByIDWithConcurrency returns an agent by ID including concurrency settings.
func (db *DB) GetAgentByIDWithConcurrency(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
	var networkMountsBytes []byte
	var healthMetricsBytes []byte
	var statusStr string
	var healthStatusStr *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at,
		       debug_mode, debug_mode_expires_at, debug_mode_enabled_at, debug_mode_enabled_by,
		       max_concurrent_backups, created_at, updated_at
		FROM agents
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
		&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
		&a.HealthCheckedAt,
		&a.DebugMode, &a.DebugModeExpiresAt, &a.DebugModeEnabledAt, &a.DebugModeEnabledBy,
		&a.MaxConcurrentBackups, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent by ID: %w", err)
	}
	a.Status = models.AgentStatus(statusStr)
	if healthStatusStr != nil {
		a.HealthStatus = models.HealthStatus(*healthStatusStr)
	} else {
		a.HealthStatus = models.HealthStatusUnknown
	}
	if err := a.SetOSInfo(osInfoBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
	}
	if err := a.SetNetworkMounts(networkMountsBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse network mounts")
	}
	if err := a.SetHealthMetrics(healthMetricsBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse health metrics")
	}
	return &a, nil
}

// CleanupOldQueueEntries removes queue entries older than a specified duration.
func (db *DB) CleanupOldQueueEntries(ctx context.Context) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM backup_queue
		WHERE (status = 'started' AND started_at < NOW() - INTERVAL '24 hours')
		   OR (status = 'canceled')
		   OR (status = 'queued' AND queued_at < NOW() - INTERVAL '7 days')
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup old queue entries: %w", err)
	}
	return result.RowsAffected(), nil
}
