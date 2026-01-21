package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Organization methods

// GetOrCreateDefaultOrg returns the default organization, creating it if necessary.
func (db *DB) GetOrCreateDefaultOrg(ctx context.Context) (*models.Organization, error) {
	var org models.Organization
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE slug = 'default'
	`).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)

	if err == nil {
		return &org, nil
	}

	// Create default organization
	org = *models.NewOrganization("Default", "default")
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create default organization: %w", err)
	}

	db.logger.Info().Str("org_id", org.ID.String()).Msg("created default organization")
	return &org, nil
}

// GetAllOrganizations returns all organizations.
func (db *DB) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		var org models.Organization
		err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, &org)
	}

	return orgs, nil
}

// User methods

// GetUserByOIDCSubject returns a user by their OIDC subject.
func (db *DB) GetUserByOIDCSubject(ctx context.Context, subject string) (*models.User, error) {
	var user models.User
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, created_at, updated_at
		FROM users
		WHERE oidc_subject = $1
	`, subject).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by OIDC subject: %w", err)
	}
	user.Role = models.UserRole(roleStr)
	return &user, nil
}

// GetUserByID returns a user by their ID.
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}
	user.Role = models.UserRole(roleStr)
	return &user, nil
}

// CreateUser creates a new user.
func (db *DB) CreateUser(ctx context.Context, user *models.User) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, org_id, oidc_subject, email, name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.OrgID, user.OIDCSubject, user.Email, user.Name, string(user.Role), user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// Agent methods

// GetAgentsByOrgID returns all agents for an organization.
func (db *DB) GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at, created_at, updated_at
		FROM agents
		WHERE org_id = $1
		ORDER BY hostname
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var a models.Agent
		var osInfoBytes []byte
		var networkMountsBytes []byte
		var healthMetricsBytes []byte
		var statusStr string
		var healthStatusStr *string
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
			&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
			&a.HealthCheckedAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
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
		agents = append(agents, &a)
	}

	return agents, nil
}

// GetAgentByID returns an agent by ID.
func (db *DB) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
	var networkMountsBytes []byte
	var healthMetricsBytes []byte
	var statusStr string
	var healthStatusStr *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at, created_at, updated_at
		FROM agents
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
		&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
		&a.HealthCheckedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
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

// GetAgentByAPIKeyHash returns an agent by API key hash.
func (db *DB) GetAgentByAPIKeyHash(ctx context.Context, hash string) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
	var networkMountsBytes []byte
	var healthMetricsBytes []byte
	var statusStr string
	var healthStatusStr *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at, created_at, updated_at
		FROM agents
		WHERE api_key_hash = $1
	`, hash).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
		&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
		&a.HealthCheckedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent by API key: %w", err)
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

// CreateAgent creates a new agent.
func (db *DB) CreateAgent(ctx context.Context, agent *models.Agent) error {
	osInfoBytes, err := agent.OSInfoJSON()
	if err != nil {
		return fmt.Errorf("marshal OS info: %w", err)
	}
	networkMountsBytes, err := agent.NetworkMountsJSON()
	if err != nil {
		return fmt.Errorf("marshal network mounts: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agents (id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, agent.ID, agent.OrgID, agent.Hostname, agent.APIKeyHash, osInfoBytes, networkMountsBytes,
		agent.LastSeen, string(agent.Status), agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

// UpdateAgent updates an existing agent.
func (db *DB) UpdateAgent(ctx context.Context, agent *models.Agent) error {
	osInfoBytes, err := agent.OSInfoJSON()
	if err != nil {
		return fmt.Errorf("marshal OS info: %w", err)
	}
	networkMountsBytes, err := agent.NetworkMountsJSON()
	if err != nil {
		return fmt.Errorf("marshal network mounts: %w", err)
	}

	healthMetricsBytes, err := agent.HealthMetricsJSON()
	if err != nil {
		return fmt.Errorf("marshal health metrics: %w", err)
	}

	agent.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE agents
		SET hostname = $2, os_info = $3, network_mounts = $4, last_seen = $5, status = $6, updated_at = $7,
		    health_status = $8, health_metrics = $9, health_checked_at = $10
		WHERE id = $1
	`, agent.ID, agent.Hostname, osInfoBytes, networkMountsBytes, agent.LastSeen, string(agent.Status), agent.UpdatedAt,
		string(agent.HealthStatus), healthMetricsBytes, agent.HealthCheckedAt)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	return nil
}

// DeleteAgent deletes an agent by ID.
func (db *DB) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM agents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}

// UpdateAgentAPIKeyHash updates an agent's API key hash.
func (db *DB) UpdateAgentAPIKeyHash(ctx context.Context, id uuid.UUID, apiKeyHash string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET api_key_hash = $2, updated_at = $3
		WHERE id = $1
	`, id, apiKeyHash, time.Now())
	if err != nil {
		return fmt.Errorf("update agent API key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// RevokeAgentAPIKey clears an agent's API key hash (effectively disabling API access).
func (db *DB) RevokeAgentAPIKey(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET api_key_hash = '', status = $2, updated_at = $3
		WHERE id = $1
	`, id, string(models.AgentStatusPending), time.Now())
	if err != nil {
		return fmt.Errorf("revoke agent API key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// CreateAgentHealthHistory creates a new health history record.
func (db *DB) CreateAgentHealthHistory(ctx context.Context, history *models.AgentHealthHistory) error {
	issuesBytes, err := json.Marshal(history.Issues)
	if err != nil {
		return fmt.Errorf("marshal issues: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agent_health_history (
			id, agent_id, org_id, health_status, cpu_usage, memory_usage, disk_usage,
			disk_free_bytes, disk_total_bytes, network_up, restic_version, restic_available,
			issues, recorded_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, history.ID, history.AgentID, history.OrgID, string(history.HealthStatus),
		history.CPUUsage, history.MemoryUsage, history.DiskUsage,
		history.DiskFreeBytes, history.DiskTotalBytes, history.NetworkUp,
		history.ResticVersion, history.ResticAvailable, issuesBytes,
		history.RecordedAt, history.CreatedAt)
	if err != nil {
		return fmt.Errorf("create health history: %w", err)
	}
	return nil
}

// GetAgentHealthHistory returns health history for an agent.
func (db *DB) GetAgentHealthHistory(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.AgentHealthHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, org_id, health_status, cpu_usage, memory_usage, disk_usage,
		       disk_free_bytes, disk_total_bytes, network_up, restic_version, restic_available,
		       issues, recorded_at, created_at
		FROM agent_health_history
		WHERE agent_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("get health history: %w", err)
	}
	defer rows.Close()

	var history []*models.AgentHealthHistory
	for rows.Next() {
		var h models.AgentHealthHistory
		var healthStatusStr string
		var issuesBytes []byte
		err := rows.Scan(
			&h.ID, &h.AgentID, &h.OrgID, &healthStatusStr,
			&h.CPUUsage, &h.MemoryUsage, &h.DiskUsage,
			&h.DiskFreeBytes, &h.DiskTotalBytes, &h.NetworkUp,
			&h.ResticVersion, &h.ResticAvailable,
			&issuesBytes, &h.RecordedAt, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan health history: %w", err)
		}
		h.HealthStatus = models.HealthStatus(healthStatusStr)
		if len(issuesBytes) > 0 {
			if err := json.Unmarshal(issuesBytes, &h.Issues); err != nil {
				db.logger.Warn().Err(err).Str("history_id", h.ID.String()).Msg("failed to parse health issues")
			}
		}
		history = append(history, &h)
	}

	return history, nil
}

// GetFleetHealthSummary returns aggregated health stats for all agents in an org.
func (db *DB) GetFleetHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.FleetHealthSummary, error) {
	summary := &models.FleetHealthSummary{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_agents,
			COUNT(*) FILTER (WHERE health_status = 'healthy') as healthy_count,
			COUNT(*) FILTER (WHERE health_status = 'warning') as warning_count,
			COUNT(*) FILTER (WHERE health_status = 'critical') as critical_count,
			COUNT(*) FILTER (WHERE health_status = 'unknown' OR health_status IS NULL) as unknown_count,
			COUNT(*) FILTER (WHERE status = 'active') as active_count,
			COUNT(*) FILTER (WHERE status = 'offline') as offline_count,
			COALESCE(AVG(CASE WHEN health_metrics->>'cpu_usage' IS NOT NULL
				THEN (health_metrics->>'cpu_usage')::DECIMAL ELSE NULL END), 0) as avg_cpu_usage,
			COALESCE(AVG(CASE WHEN health_metrics->>'memory_usage' IS NOT NULL
				THEN (health_metrics->>'memory_usage')::DECIMAL ELSE NULL END), 0) as avg_memory_usage,
			COALESCE(AVG(CASE WHEN health_metrics->>'disk_usage' IS NOT NULL
				THEN (health_metrics->>'disk_usage')::DECIMAL ELSE NULL END), 0) as avg_disk_usage
		FROM agents
		WHERE org_id = $1
	`, orgID).Scan(
		&summary.TotalAgents,
		&summary.HealthyCount,
		&summary.WarningCount,
		&summary.CriticalCount,
		&summary.UnknownCount,
		&summary.ActiveCount,
		&summary.OfflineCount,
		&summary.AvgCPUUsage,
		&summary.AvgMemoryUsage,
		&summary.AvgDiskUsage,
	)
	if err != nil {
		return nil, fmt.Errorf("get fleet health summary: %w", err)
	}

	return summary, nil
}

// GetAgentStats returns aggregated statistics for an agent.
func (db *DB) GetAgentStats(ctx context.Context, agentID uuid.UUID) (*models.AgentStats, error) {
	stats := &models.AgentStats{AgentID: agentID}

	// Get backup statistics
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as successful,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(SUM(size_bytes) FILTER (WHERE status = 'completed'), 0) as total_size,
			MAX(completed_at) FILTER (WHERE status = 'completed') as last_backup
		FROM backups
		WHERE agent_id = $1
	`, agentID).Scan(
		&stats.TotalBackups,
		&stats.SuccessfulBackups,
		&stats.FailedBackups,
		&stats.TotalSizeBytes,
		&stats.LastBackupAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup stats: %w", err)
	}

	// Calculate success rate
	if stats.TotalBackups > 0 {
		stats.SuccessRate = float64(stats.SuccessfulBackups) / float64(stats.TotalBackups) * 100
	}

	// Get schedule count
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM schedules WHERE agent_id = $1 AND enabled = true
	`, agentID).Scan(&stats.ScheduleCount)
	if err != nil {
		return nil, fmt.Errorf("get schedule count: %w", err)
	}

	return stats, nil
}

// Repository methods

// GetRepositoriesByOrgID returns all repositories for an organization.
func (db *DB) GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, config_encrypted, created_at, updated_at
		FROM repositories
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer rows.Close()

	var repos []*models.Repository
	for rows.Next() {
		var r models.Repository
		var typeStr string
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.Name, &typeStr, &r.ConfigEncrypted,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		r.Type = models.RepositoryType(typeStr)
		repos = append(repos, &r)
	}

	return repos, nil
}

// GetRepositoryByID returns a repository by ID.
func (db *DB) GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	var r models.Repository
	var typeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, config_encrypted, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.ConfigEncrypted,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}
	r.Type = models.RepositoryType(typeStr)
	return &r, nil
}

// CreateRepository creates a new repository.
func (db *DB) CreateRepository(ctx context.Context, repo *models.Repository) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repositories (id, org_id, name, type, config_encrypted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, repo.ID, repo.OrgID, repo.Name, string(repo.Type), repo.ConfigEncrypted,
		repo.CreatedAt, repo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create repository: %w", err)
	}
	return nil
}

// UpdateRepository updates an existing repository.
func (db *DB) UpdateRepository(ctx context.Context, repo *models.Repository) error {
	repo.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE repositories
		SET name = $2, config_encrypted = $3, updated_at = $4
		WHERE id = $1
	`, repo.ID, repo.Name, repo.ConfigEncrypted, repo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update repository: %w", err)
	}
	return nil
}

// DeleteRepository deletes a repository by ID.
func (db *DB) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM repositories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete repository: %w", err)
	}
	return nil
}

// Schedule methods

// GetSchedulesByAgentID returns all schedules for an agent.
func (db *DB) GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		FROM schedules
		WHERE agent_id = $1
		ORDER BY name
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	// Load repositories for all schedules
	for _, s := range schedules {
		repos, err := db.GetScheduleRepositories(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("get schedule repositories: %w", err)
		}
		s.Repositories = repos
	}

	return schedules, nil
}

// GetScheduleByID returns a schedule by ID.
func (db *DB) GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		FROM schedules
		WHERE id = $1
	`, id)

	s, err := scanScheduleRow(row)
	if err != nil {
		return nil, err
	}

	// Load repositories
	repos, err := db.GetScheduleRepositories(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("get schedule repositories: %w", err)
	}
	s.Repositories = repos

	return s, nil
}

// CreateSchedule creates a new schedule.
func (db *DB) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	pathsBytes, err := schedule.PathsJSON()
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := schedule.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	retentionBytes, err := schedule.RetentionPolicyJSON()
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	excludedHoursBytes, err := schedule.ExcludedHoursJSON()
	if err != nil {
		return fmt.Errorf("marshal excluded hours: %w", err)
	}

	// Extract backup window times
	var windowStart, windowEnd *string
	if schedule.BackupWindow != nil {
		if schedule.BackupWindow.Start != "" {
			windowStart = &schedule.BackupWindow.Start
		}
		if schedule.BackupWindow.End != "" {
			windowEnd = &schedule.BackupWindow.End
		}
	}

	// Default mount behavior to fail if not set
	mountBehavior := string(schedule.OnMountUnavailable)
	if mountBehavior == "" {
		mountBehavior = string(models.MountBehaviorFail)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO schedules (id, agent_id, agent_group_id, policy_id, name, cron_expression, paths,
		                       excludes, retention_policy, bandwidth_limit_kbps,
		                       backup_window_start, backup_window_end, excluded_hours,
		                       compression_level, on_mount_unavailable, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, schedule.ID, schedule.AgentID, schedule.AgentGroupID, schedule.PolicyID, schedule.Name,
		schedule.CronExpression, pathsBytes, excludesBytes, retentionBytes,
		schedule.BandwidthLimitKB, windowStart, windowEnd, excludedHoursBytes,
		schedule.CompressionLevel, mountBehavior, schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}

	// Create schedule-repository associations
	for _, sr := range schedule.Repositories {
		sr.ScheduleID = schedule.ID
		if err := db.CreateScheduleRepository(ctx, &sr); err != nil {
			return fmt.Errorf("create schedule repository: %w", err)
		}
	}

	return nil
}

// UpdateSchedule updates an existing schedule.
func (db *DB) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	pathsBytes, err := schedule.PathsJSON()
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := schedule.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	retentionBytes, err := schedule.RetentionPolicyJSON()
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	excludedHoursBytes, err := schedule.ExcludedHoursJSON()
	if err != nil {
		return fmt.Errorf("marshal excluded hours: %w", err)
	}

	// Extract backup window times
	var windowStart, windowEnd *string
	if schedule.BackupWindow != nil {
		if schedule.BackupWindow.Start != "" {
			windowStart = &schedule.BackupWindow.Start
		}
		if schedule.BackupWindow.End != "" {
			windowEnd = &schedule.BackupWindow.End
		}
	}

	// Default mount behavior to fail if not set
	mountBehavior := string(schedule.OnMountUnavailable)
	if mountBehavior == "" {
		mountBehavior = string(models.MountBehaviorFail)
	}

	schedule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE schedules
		SET policy_id = $2, name = $3, cron_expression = $4, paths = $5, excludes = $6,
		    retention_policy = $7, bandwidth_limit_kbps = $8, backup_window_start = $9,
		    backup_window_end = $10, excluded_hours = $11, compression_level = $12,
		    on_mount_unavailable = $13, enabled = $14, updated_at = $15
		WHERE id = $1
	`, schedule.ID, schedule.PolicyID, schedule.Name, schedule.CronExpression, pathsBytes,
		excludesBytes, retentionBytes, schedule.BandwidthLimitKB, windowStart, windowEnd,
		excludedHoursBytes, schedule.CompressionLevel, mountBehavior, schedule.Enabled, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	return nil
}

// DeleteSchedule deletes a schedule by ID.
func (db *DB) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

// Backup Script methods

// GetBackupScriptsByScheduleID returns all backup scripts for a schedule.
func (db *DB) GetBackupScriptsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupScript, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, type, script, timeout_seconds, fail_on_error, enabled, created_at, updated_at
		FROM backup_scripts
		WHERE schedule_id = $1
		ORDER BY type
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list backup scripts by schedule: %w", err)
	}
	defer rows.Close()

	return scanBackupScripts(rows)
}

// GetBackupScriptByID returns a backup script by ID.
func (db *DB) GetBackupScriptByID(ctx context.Context, id uuid.UUID) (*models.BackupScript, error) {
	var s models.BackupScript
	var typeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, type, script, timeout_seconds, fail_on_error, enabled, created_at, updated_at
		FROM backup_scripts
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.ScheduleID, &typeStr, &s.Script, &s.TimeoutSeconds,
		&s.FailOnError, &s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup script: %w", err)
	}
	s.Type = models.BackupScriptType(typeStr)
	return &s, nil
}

// GetBackupScriptByScheduleAndType returns a backup script by schedule ID and type.
func (db *DB) GetBackupScriptByScheduleAndType(ctx context.Context, scheduleID uuid.UUID, scriptType models.BackupScriptType) (*models.BackupScript, error) {
	var s models.BackupScript
	var typeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, type, script, timeout_seconds, fail_on_error, enabled, created_at, updated_at
		FROM backup_scripts
		WHERE schedule_id = $1 AND type = $2
	`, scheduleID, string(scriptType)).Scan(
		&s.ID, &s.ScheduleID, &typeStr, &s.Script, &s.TimeoutSeconds,
		&s.FailOnError, &s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup script by type: %w", err)
	}
	s.Type = models.BackupScriptType(typeStr)
	return &s, nil
}

// GetEnabledBackupScriptsByScheduleID returns all enabled backup scripts for a schedule.
func (db *DB) GetEnabledBackupScriptsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupScript, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, type, script, timeout_seconds, fail_on_error, enabled, created_at, updated_at
		FROM backup_scripts
		WHERE schedule_id = $1 AND enabled = true
		ORDER BY type
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list enabled backup scripts by schedule: %w", err)
	}
	defer rows.Close()

	return scanBackupScripts(rows)
}

// CreateBackupScript creates a new backup script.
func (db *DB) CreateBackupScript(ctx context.Context, script *models.BackupScript) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backup_scripts (id, schedule_id, type, script, timeout_seconds, fail_on_error, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, script.ID, script.ScheduleID, string(script.Type), script.Script, script.TimeoutSeconds,
		script.FailOnError, script.Enabled, script.CreatedAt, script.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create backup script: %w", err)
	}
	return nil
}

// UpdateBackupScript updates an existing backup script.
func (db *DB) UpdateBackupScript(ctx context.Context, script *models.BackupScript) error {
	script.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE backup_scripts
		SET script = $2, timeout_seconds = $3, fail_on_error = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`, script.ID, script.Script, script.TimeoutSeconds, script.FailOnError, script.Enabled, script.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update backup script: %w", err)
	}
	return nil
}

// DeleteBackupScript deletes a backup script by ID.
func (db *DB) DeleteBackupScript(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM backup_scripts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup script: %w", err)
	}
	return nil
}

// scanBackupScripts scans multiple backup script rows.
func scanBackupScripts(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.BackupScript, error) {
	var scripts []*models.BackupScript
	for rows.Next() {
		var s models.BackupScript
		var typeStr string
		err := rows.Scan(
			&s.ID, &s.ScheduleID, &typeStr, &s.Script, &s.TimeoutSeconds,
			&s.FailOnError, &s.Enabled, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup script: %w", err)
		}
		s.Type = models.BackupScriptType(typeStr)
		scripts = append(scripts, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup scripts: %w", err)
	}

	return scripts, nil
}

// scanSchedule scans a schedule from a row iterator.
func scanSchedule(rows interface {
	Scan(dest ...any) error
}) (*models.Schedule, error) {
	var s models.Schedule
	var pathsBytes, excludesBytes, retentionBytes, excludedHoursBytes []byte
	var agentGroupID *uuid.UUID
	var windowStart, windowEnd, compressionLevel, mountBehavior *string
	err := rows.Scan(
		&s.ID, &s.AgentID, &agentGroupID, &s.PolicyID, &s.Name, &s.CronExpression,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.BandwidthLimitKB,
		&windowStart, &windowEnd, &excludedHoursBytes, &compressionLevel, &mountBehavior,
		&s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	s.AgentGroupID = agentGroupID

	if err := s.SetPaths(pathsBytes); err != nil {
		return nil, fmt.Errorf("parse paths: %w", err)
	}
	if err := s.SetExcludes(excludesBytes); err != nil {
		return nil, fmt.Errorf("parse excludes: %w", err)
	}
	if err := s.SetRetentionPolicy(retentionBytes); err != nil {
		return nil, fmt.Errorf("parse retention policy: %w", err)
	}
	if err := s.SetExcludedHours(excludedHoursBytes); err != nil {
		return nil, fmt.Errorf("parse excluded hours: %w", err)
	}
	s.SetBackupWindow(windowStart, windowEnd)
	s.CompressionLevel = compressionLevel

	// Set mount behavior with default
	if mountBehavior != nil && *mountBehavior != "" {
		s.OnMountUnavailable = models.MountBehavior(*mountBehavior)
	} else {
		s.OnMountUnavailable = models.MountBehaviorFail
	}

	return &s, nil
}

// scanScheduleRow scans a schedule from a single row.
func scanScheduleRow(row interface {
	Scan(dest ...any) error
}) (*models.Schedule, error) {
	return scanSchedule(row)
}

// Policy methods

// GetPoliciesByOrgID returns all policies for an organization.
func (db *DB) GetPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Policy, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, paths, excludes, retention_policy,
		       bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, cron_expression, created_at, updated_at
		FROM backup_policies
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*models.Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}

	return policies, nil
}

// GetPolicyByID returns a policy by ID.
func (db *DB) GetPolicyByID(ctx context.Context, id uuid.UUID) (*models.Policy, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, paths, excludes, retention_policy,
		       bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, cron_expression, created_at, updated_at
		FROM backup_policies
		WHERE id = $1
	`, id)

	return scanPolicy(row)
}

// CreatePolicy creates a new policy.
func (db *DB) CreatePolicy(ctx context.Context, policy *models.Policy) error {
	pathsBytes, err := policy.PathsJSON()
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := policy.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	retentionBytes, err := policy.RetentionPolicyJSON()
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	excludedHoursBytes, err := policy.ExcludedHoursJSON()
	if err != nil {
		return fmt.Errorf("marshal excluded hours: %w", err)
	}

	var windowStart, windowEnd *string
	if policy.BackupWindow != nil {
		if policy.BackupWindow.Start != "" {
			windowStart = &policy.BackupWindow.Start
		}
		if policy.BackupWindow.End != "" {
			windowEnd = &policy.BackupWindow.End
		}
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO backup_policies (id, org_id, name, description, paths, excludes,
		                             retention_policy, bandwidth_limit_kbps,
		                             backup_window_start, backup_window_end,
		                             excluded_hours, cron_expression, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, policy.ID, policy.OrgID, policy.Name, policy.Description, pathsBytes, excludesBytes,
		retentionBytes, policy.BandwidthLimitKB, windowStart, windowEnd,
		excludedHoursBytes, policy.CronExpression, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create policy: %w", err)
	}
	return nil
}

// UpdatePolicy updates an existing policy.
func (db *DB) UpdatePolicy(ctx context.Context, policy *models.Policy) error {
	pathsBytes, err := policy.PathsJSON()
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := policy.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	retentionBytes, err := policy.RetentionPolicyJSON()
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	excludedHoursBytes, err := policy.ExcludedHoursJSON()
	if err != nil {
		return fmt.Errorf("marshal excluded hours: %w", err)
	}

	var windowStart, windowEnd *string
	if policy.BackupWindow != nil {
		if policy.BackupWindow.Start != "" {
			windowStart = &policy.BackupWindow.Start
		}
		if policy.BackupWindow.End != "" {
			windowEnd = &policy.BackupWindow.End
		}
	}

	policy.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE backup_policies
		SET name = $2, description = $3, paths = $4, excludes = $5, retention_policy = $6,
		    bandwidth_limit_kbps = $7, backup_window_start = $8, backup_window_end = $9,
		    excluded_hours = $10, cron_expression = $11, updated_at = $12
		WHERE id = $1
	`, policy.ID, policy.Name, policy.Description, pathsBytes, excludesBytes, retentionBytes,
		policy.BandwidthLimitKB, windowStart, windowEnd, excludedHoursBytes,
		policy.CronExpression, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}
	return nil
}

// DeletePolicy deletes a policy by ID.
func (db *DB) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM backup_policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	return nil
}

// GetSchedulesByPolicyID returns all schedules using a policy.
func (db *DB) GetSchedulesByPolicyID(ctx context.Context, policyID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		FROM schedules
		WHERE policy_id = $1
		ORDER BY name
	`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list schedules by policy: %w", err)
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}

// scanPolicy scans a policy from a row iterator.
func scanPolicy(rows interface {
	Scan(dest ...any) error
}) (*models.Policy, error) {
	var p models.Policy
	var pathsBytes, excludesBytes, retentionBytes, excludedHoursBytes []byte
	var windowStart, windowEnd *string
	var description *string
	var cronExpression *string
	err := rows.Scan(
		&p.ID, &p.OrgID, &p.Name, &description, &pathsBytes, &excludesBytes,
		&retentionBytes, &p.BandwidthLimitKB, &windowStart, &windowEnd,
		&excludedHoursBytes, &cronExpression, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan policy: %w", err)
	}

	if description != nil {
		p.Description = *description
	}
	if cronExpression != nil {
		p.CronExpression = *cronExpression
	}

	if err := p.SetPaths(pathsBytes); err != nil {
		return nil, fmt.Errorf("parse paths: %w", err)
	}
	if err := p.SetExcludes(excludesBytes); err != nil {
		return nil, fmt.Errorf("parse excludes: %w", err)
	}
	if err := p.SetRetentionPolicy(retentionBytes); err != nil {
		return nil, fmt.Errorf("parse retention policy: %w", err)
	}
	if err := p.SetExcludedHours(excludedHoursBytes); err != nil {
		return nil, fmt.Errorf("parse excluded hours: %w", err)
	}
	p.SetBackupWindow(windowStart, windowEnd)

	return &p, nil
}

// Backup methods

// GetBackupsByScheduleID returns all backups for a schedule.
func (db *DB) GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error, created_at
		FROM backups
		WHERE schedule_id = $1
		ORDER BY started_at DESC
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list backups by schedule: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupsByAgentID returns all backups for an agent.
func (db *DB) GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error, created_at
		FROM backups
		WHERE agent_id = $1
		ORDER BY started_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list backups by agent: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupByID returns a backup by ID.
func (db *DB) GetBackupByID(ctx context.Context, id uuid.UUID) (*models.Backup, error) {
	var b models.Backup
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error, created_at
		FROM backups
		WHERE id = $1
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
		&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	b.Status = models.BackupStatus(statusStr)
	return &b, nil
}

// CreateBackup creates a new backup record.
func (db *DB) CreateBackup(ctx context.Context, backup *models.Backup) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backups (id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		                     status, size_bytes, files_new, files_changed, error_message,
		                     retention_applied, snapshots_removed, snapshots_kept, retention_error,
		                     pre_script_output, pre_script_error, post_script_output, post_script_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, backup.ID, backup.ScheduleID, backup.AgentID, backup.RepositoryID, backup.SnapshotID,
		backup.StartedAt, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError,
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError, backup.CreatedAt)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	return nil
}

// UpdateBackup updates an existing backup record.
func (db *DB) UpdateBackup(ctx context.Context, backup *models.Backup) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE backups
		SET snapshot_id = $2, completed_at = $3, status = $4, size_bytes = $5,
		    files_new = $6, files_changed = $7, error_message = $8,
		    retention_applied = $9, snapshots_removed = $10, snapshots_kept = $11, retention_error = $12,
		    pre_script_output = $13, pre_script_error = $14, post_script_output = $15, post_script_error = $16
		WHERE id = $1
	`, backup.ID, backup.SnapshotID, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError,
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError)
	if err != nil {
		return fmt.Errorf("update backup: %w", err)
	}
	return nil
}

// scanBackups scans multiple backup rows.
func scanBackups(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Backup, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var backups []*models.Backup
	for r.Next() {
		var b models.Backup
		var statusStr string
		err := r.Scan(
			&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
			&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
			&b.FilesChanged, &b.ErrorMessage,
			&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
			&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		b.Status = models.BackupStatus(statusStr)
		backups = append(backups, &b)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate backups: %w", err)
	}

	return backups, nil
}

// ScheduleRepository methods

// GetScheduleRepositories returns all repositories for a schedule, ordered by priority.
func (db *DB) GetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID) ([]models.ScheduleRepository, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, repository_id, priority, enabled, created_at
		FROM schedule_repositories
		WHERE schedule_id = $1
		ORDER BY priority ASC
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list schedule repositories: %w", err)
	}
	defer rows.Close()

	var repos []models.ScheduleRepository
	for rows.Next() {
		var sr models.ScheduleRepository
		err := rows.Scan(&sr.ID, &sr.ScheduleID, &sr.RepositoryID, &sr.Priority, &sr.Enabled, &sr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan schedule repository: %w", err)
		}
		repos = append(repos, sr)
	}

	return repos, nil
}

// CreateScheduleRepository creates a schedule-repository association.
func (db *DB) CreateScheduleRepository(ctx context.Context, sr *models.ScheduleRepository) error {
	if sr.ID == uuid.Nil {
		sr.ID = uuid.New()
	}
	if sr.CreatedAt.IsZero() {
		sr.CreatedAt = time.Now()
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO schedule_repositories (id, schedule_id, repository_id, priority, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sr.ID, sr.ScheduleID, sr.RepositoryID, sr.Priority, sr.Enabled, sr.CreatedAt)
	if err != nil {
		return fmt.Errorf("create schedule repository: %w", err)
	}
	return nil
}

// DeleteScheduleRepositories deletes all repository associations for a schedule.
func (db *DB) DeleteScheduleRepositories(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM schedule_repositories WHERE schedule_id = $1`, scheduleID)
	if err != nil {
		return fmt.Errorf("delete schedule repositories: %w", err)
	}
	return nil
}

// SetScheduleRepositories replaces all repository associations for a schedule.
func (db *DB) SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error {
	// Delete existing associations
	if err := db.DeleteScheduleRepositories(ctx, scheduleID); err != nil {
		return err
	}

	// Create new associations
	for _, sr := range repos {
		sr.ScheduleID = scheduleID
		if err := db.CreateScheduleRepository(ctx, &sr); err != nil {
			return err
		}
	}

	return nil
}

// ReplicationStatus methods

// GetReplicationStatusBySchedule returns all replication status records for a schedule.
func (db *DB) GetReplicationStatusBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]*models.ReplicationStatus, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, source_repository_id, target_repository_id,
		       last_snapshot_id, last_sync_at, status, error_message, created_at, updated_at
		FROM replication_status
		WHERE schedule_id = $1
		ORDER BY created_at ASC
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list replication status: %w", err)
	}
	defer rows.Close()

	var statuses []*models.ReplicationStatus
	for rows.Next() {
		var rs models.ReplicationStatus
		var statusStr string
		err := rows.Scan(
			&rs.ID, &rs.ScheduleID, &rs.SourceRepositoryID, &rs.TargetRepositoryID,
			&rs.LastSnapshotID, &rs.LastSyncAt, &statusStr, &rs.ErrorMessage,
			&rs.CreatedAt, &rs.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan replication status: %w", err)
		}
		rs.Status = models.ReplicationStatusType(statusStr)
		statuses = append(statuses, &rs)
	}

	return statuses, nil
}

// GetOrCreateReplicationStatus gets or creates a replication status record.
func (db *DB) GetOrCreateReplicationStatus(ctx context.Context, scheduleID, sourceRepoID, targetRepoID uuid.UUID) (*models.ReplicationStatus, error) {
	var rs models.ReplicationStatus
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, source_repository_id, target_repository_id,
		       last_snapshot_id, last_sync_at, status, error_message, created_at, updated_at
		FROM replication_status
		WHERE schedule_id = $1 AND source_repository_id = $2 AND target_repository_id = $3
	`, scheduleID, sourceRepoID, targetRepoID).Scan(
		&rs.ID, &rs.ScheduleID, &rs.SourceRepositoryID, &rs.TargetRepositoryID,
		&rs.LastSnapshotID, &rs.LastSyncAt, &statusStr, &rs.ErrorMessage,
		&rs.CreatedAt, &rs.UpdatedAt,
	)
	if err == nil {
		rs.Status = models.ReplicationStatusType(statusStr)
		return &rs, nil
	}

	// Create new record
	newRS := models.NewReplicationStatus(scheduleID, sourceRepoID, targetRepoID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO replication_status (id, schedule_id, source_repository_id, target_repository_id,
		                                status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newRS.ID, newRS.ScheduleID, newRS.SourceRepositoryID, newRS.TargetRepositoryID,
		string(newRS.Status), newRS.CreatedAt, newRS.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create replication status: %w", err)
	}

	return newRS, nil
}

// UpdateReplicationStatus updates a replication status record.
func (db *DB) UpdateReplicationStatus(ctx context.Context, rs *models.ReplicationStatus) error {
	rs.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE replication_status
		SET last_snapshot_id = $2, last_sync_at = $3, status = $4, error_message = $5, updated_at = $6
		WHERE id = $1
	`, rs.ID, rs.LastSnapshotID, rs.LastSyncAt, string(rs.Status), rs.ErrorMessage, rs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update replication status: %w", err)
	}
	return nil
}

// GetBackupBySnapshotID returns a backup by its snapshot ID.
func (db *DB) GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error) {
	var b models.Backup
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE snapshot_id = $1
	`, snapshotID).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup by snapshot ID: %w", err)
	}
	b.Status = models.BackupStatus(statusStr)
	return &b, nil
}

// Alert methods

// GetAlertsByOrgID returns all alerts for an organization.
func (db *DB) GetAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, rule_id, type, severity, status, title, message,
		       resource_type, resource_id, acknowledged_by, acknowledged_at,
		       resolved_at, metadata, created_at, updated_at
		FROM alerts
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	return db.scanAlerts(rows)
}

// GetActiveAlertsByOrgID returns active (non-resolved) alerts for an organization.
func (db *DB) GetActiveAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, rule_id, type, severity, status, title, message,
		       resource_type, resource_id, acknowledged_by, acknowledged_at,
		       resolved_at, metadata, created_at, updated_at
		FROM alerts
		WHERE org_id = $1 AND status != 'resolved'
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'warning' THEN 2
				ELSE 3
			END,
			created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	defer rows.Close()

	return db.scanAlerts(rows)
}

// GetActiveAlertCountByOrgID returns the count of active alerts for an organization.
func (db *DB) GetActiveAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE org_id = $1 AND status != 'resolved'
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active alerts: %w", err)
	}
	return count, nil
}

// GetAlertByID returns an alert by ID.
func (db *DB) GetAlertByID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	var a models.Alert
	var typeStr, severityStr, statusStr string
	var resourceTypeStr *string
	var metadataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, rule_id, type, severity, status, title, message,
		       resource_type, resource_id, acknowledged_by, acknowledged_at,
		       resolved_at, metadata, created_at, updated_at
		FROM alerts
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.RuleID, &typeStr, &severityStr, &statusStr,
		&a.Title, &a.Message, &resourceTypeStr, &a.ResourceID,
		&a.AcknowledgedBy, &a.AcknowledgedAt, &a.ResolvedAt,
		&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}
	a.Type = models.AlertType(typeStr)
	a.Severity = models.AlertSeverity(severityStr)
	a.Status = models.AlertStatus(statusStr)
	if resourceTypeStr != nil {
		rt := models.ResourceType(*resourceTypeStr)
		a.ResourceType = &rt
	}
	if err := a.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("alert_id", a.ID.String()).Msg("failed to parse alert metadata")
	}
	return &a, nil
}

// GetAlertByResourceAndType returns an active alert for a specific resource and type.
func (db *DB) GetAlertByResourceAndType(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (*models.Alert, error) {
	var a models.Alert
	var typeStr, severityStr, statusStr string
	var resourceTypeStr *string
	var metadataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, rule_id, type, severity, status, title, message,
		       resource_type, resource_id, acknowledged_by, acknowledged_at,
		       resolved_at, metadata, created_at, updated_at
		FROM alerts
		WHERE org_id = $1 AND resource_type = $2 AND resource_id = $3 AND type = $4 AND status != 'resolved'
		LIMIT 1
	`, orgID, string(resourceType), resourceID, string(alertType)).Scan(
		&a.ID, &a.OrgID, &a.RuleID, &typeStr, &severityStr, &statusStr,
		&a.Title, &a.Message, &resourceTypeStr, &a.ResourceID,
		&a.AcknowledgedBy, &a.AcknowledgedAt, &a.ResolvedAt,
		&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get alert by resource: %w", err)
	}
	a.Type = models.AlertType(typeStr)
	a.Severity = models.AlertSeverity(severityStr)
	a.Status = models.AlertStatus(statusStr)
	if resourceTypeStr != nil {
		rt := models.ResourceType(*resourceTypeStr)
		a.ResourceType = &rt
	}
	if err := a.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("alert_id", a.ID.String()).Msg("failed to parse alert metadata")
	}
	return &a, nil
}

// CreateAlert creates a new alert.
func (db *DB) CreateAlert(ctx context.Context, alert *models.Alert) error {
	metadataBytes, err := alert.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var resourceType *string
	if alert.ResourceType != nil {
		rt := string(*alert.ResourceType)
		resourceType = &rt
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO alerts (id, org_id, rule_id, type, severity, status, title, message,
		                    resource_type, resource_id, acknowledged_by, acknowledged_at,
		                    resolved_at, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, alert.ID, alert.OrgID, alert.RuleID, string(alert.Type), string(alert.Severity),
		string(alert.Status), alert.Title, alert.Message, resourceType, alert.ResourceID,
		alert.AcknowledgedBy, alert.AcknowledgedAt, alert.ResolvedAt,
		metadataBytes, alert.CreatedAt, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create alert: %w", err)
	}
	return nil
}

// UpdateAlert updates an existing alert.
func (db *DB) UpdateAlert(ctx context.Context, alert *models.Alert) error {
	metadataBytes, err := alert.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	alert.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE alerts
		SET status = $2, acknowledged_by = $3, acknowledged_at = $4,
		    resolved_at = $5, metadata = $6, updated_at = $7
		WHERE id = $1
	`, alert.ID, string(alert.Status), alert.AcknowledgedBy, alert.AcknowledgedAt,
		alert.ResolvedAt, metadataBytes, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update alert: %w", err)
	}
	return nil
}

// ResolveAlertsByResource resolves all active alerts for a specific resource.
func (db *DB) ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE alerts
		SET status = 'resolved', resolved_at = $3, updated_at = $3
		WHERE resource_type = $1 AND resource_id = $2 AND status != 'resolved'
	`, string(resourceType), resourceID, now)
	if err != nil {
		return fmt.Errorf("resolve alerts by resource: %w", err)
	}
	return nil
}

// Storage Stats methods

// CreateStorageStats creates a new storage stats record.
func (db *DB) CreateStorageStats(ctx context.Context, stats *models.StorageStats) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO storage_stats (id, repository_id, total_size, total_file_count, raw_data_size,
		                           restore_size, dedup_ratio, space_saved, space_saved_pct,
		                           snapshot_count, collected_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, stats.ID, stats.RepositoryID, stats.TotalSize, stats.TotalFileCount, stats.RawDataSize,
		stats.RestoreSize, stats.DedupRatio, stats.SpaceSaved, stats.SpaceSavedPct,
		stats.SnapshotCount, stats.CollectedAt, stats.CreatedAt)
	if err != nil {
		return fmt.Errorf("create storage stats: %w", err)
	}
	return nil
}

// scanAlerts scans multiple alert rows.
func (db *DB) scanAlerts(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Alert, error) {
	var alerts []*models.Alert
	for rows.Next() {
		var a models.Alert
		var typeStr, severityStr, statusStr string
		var resourceTypeStr *string
		var metadataBytes []byte
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.RuleID, &typeStr, &severityStr, &statusStr,
			&a.Title, &a.Message, &resourceTypeStr, &a.ResourceID,
			&a.AcknowledgedBy, &a.AcknowledgedAt, &a.ResolvedAt,
			&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Type = models.AlertType(typeStr)
		a.Severity = models.AlertSeverity(severityStr)
		a.Status = models.AlertStatus(statusStr)
		if resourceTypeStr != nil {
			rt := models.ResourceType(*resourceTypeStr)
			a.ResourceType = &rt
		}
		if err := a.SetMetadata(metadataBytes); err != nil {
			db.logger.Warn().Err(err).Str("alert_id", a.ID.String()).Msg("failed to parse alert metadata")
		}
		alerts = append(alerts, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alerts: %w", err)
	}

	return alerts, nil
}

// Alert Rule methods

// GetAlertRulesByOrgID returns all alert rules for an organization.
func (db *DB) GetAlertRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, enabled, config, created_at, updated_at
		FROM alert_rules
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.AlertRule
	for rows.Next() {
		var r models.AlertRule
		var typeStr string
		var configBytes []byte
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.Name, &typeStr, &r.Enabled,
			&configBytes, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert rule: %w", err)
		}
		r.Type = models.AlertType(typeStr)
		if err := r.SetConfig(configBytes); err != nil {
			db.logger.Warn().Err(err).Str("rule_id", r.ID.String()).Msg("failed to parse alert rule config")
		}
		rules = append(rules, &r)
	}

	return rules, nil
}

// GetEnabledAlertRulesByOrgID returns enabled alert rules for an organization.
func (db *DB) GetEnabledAlertRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, enabled, config, created_at, updated_at
		FROM alert_rules
		WHERE org_id = $1 AND enabled = true
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list enabled alert rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.AlertRule
	for rows.Next() {
		var r models.AlertRule
		var typeStr string
		var configBytes []byte
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.Name, &typeStr, &r.Enabled,
			&configBytes, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert rule: %w", err)
		}
		r.Type = models.AlertType(typeStr)
		if err := r.SetConfig(configBytes); err != nil {
			db.logger.Warn().Err(err).Str("rule_id", r.ID.String()).Msg("failed to parse alert rule config")
		}
		rules = append(rules, &r)
	}

	return rules, nil
}

// GetAlertRuleByID returns an alert rule by ID.
func (db *DB) GetAlertRuleByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error) {
	var r models.AlertRule
	var typeStr string
	var configBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, enabled, config, created_at, updated_at
		FROM alert_rules
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.Enabled,
		&configBytes, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get alert rule: %w", err)
	}
	r.Type = models.AlertType(typeStr)
	if err := r.SetConfig(configBytes); err != nil {
		db.logger.Warn().Err(err).Str("rule_id", r.ID.String()).Msg("failed to parse alert rule config")
	}
	return &r, nil
}

// CreateAlertRule creates a new alert rule.
func (db *DB) CreateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	configBytes, err := rule.ConfigJSON()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO alert_rules (id, org_id, name, type, enabled, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rule.ID, rule.OrgID, rule.Name, string(rule.Type), rule.Enabled,
		configBytes, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create alert rule: %w", err)
	}
	return nil
}

// UpdateAlertRule updates an existing alert rule.
func (db *DB) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	configBytes, err := rule.ConfigJSON()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	rule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE alert_rules
		SET name = $2, enabled = $3, config = $4, updated_at = $5
		WHERE id = $1
	`, rule.ID, rule.Name, rule.Enabled, configBytes, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update alert rule: %w", err)
	}
	return nil
}

// DeleteAlertRule deletes an alert rule by ID.
func (db *DB) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	return nil
}

// GetAllAgents returns all agents across all organizations (for monitoring).
func (db *DB) GetAllAgents(ctx context.Context) ([]*models.Agent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, last_seen, status, created_at, updated_at
		FROM agents
		WHERE status != 'disabled'
		ORDER BY org_id, hostname
	`)
	if err != nil {
		return nil, fmt.Errorf("list all agents: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var a models.Agent
		var osInfoBytes []byte
		var statusStr string
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes,
			&a.LastSeen, &statusStr, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		a.Status = models.AgentStatus(statusStr)
		if err := a.SetOSInfo(osInfoBytes); err != nil {
			db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
		}
		agents = append(agents, &a)
	}

	return agents, nil
}

// GetAllSchedules returns all enabled schedules across all organizations (for monitoring).
func (db *DB) GetAllSchedules(ctx context.Context) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression, s.paths, s.excludes,
		       s.retention_policy, s.bandwidth_limit_kbps, s.backup_window_start, s.backup_window_end,
		       s.excluded_hours, s.compression_level, s.enabled, s.created_at, s.updated_at
		FROM schedules s
		WHERE s.enabled = true
		ORDER BY s.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list all schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}

// GetEnabledSchedules returns all enabled schedules.
func (db *DB) GetEnabledSchedules(ctx context.Context) ([]models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		FROM schedules
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, *s)
	}

	return schedules, nil
}

// GetLatestBackupByScheduleID returns the most recent backup for a schedule.
func (db *DB) GetLatestBackupByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.Backup, error) {
	var b models.Backup
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message, created_at
		FROM backups
		WHERE schedule_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, scheduleID).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest backup: %w", err)
	}
	b.Status = models.BackupStatus(statusStr)
	return &b, nil
}

// Restore methods

// GetRestoresByAgentID returns all restores for an agent.
func (db *DB) GetRestoresByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Restore, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, repository_id, snapshot_id, target_path, include_paths,
		       exclude_paths, status, started_at, completed_at, error_message, created_at, updated_at
		FROM restores
		WHERE agent_id = $1
		ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list restores by agent: %w", err)
	}
	defer rows.Close()

	return scanRestores(rows)
}

// GetRestoreByID returns a restore by ID.
func (db *DB) GetRestoreByID(ctx context.Context, id uuid.UUID) (*models.Restore, error) {
	var r models.Restore
	var statusStr string
	var includePathsBytes, excludePathsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, repository_id, snapshot_id, target_path, include_paths,
		       exclude_paths, status, started_at, completed_at, error_message, created_at, updated_at
		FROM restores
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.AgentID, &r.RepositoryID, &r.SnapshotID, &r.TargetPath,
		&includePathsBytes, &excludePathsBytes, &statusStr, &r.StartedAt,
		&r.CompletedAt, &r.ErrorMessage, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get restore: %w", err)
	}
	r.Status = models.RestoreStatus(statusStr)
	if err := parseStringSlice(includePathsBytes, &r.IncludePaths); err != nil {
		return nil, fmt.Errorf("parse include paths: %w", err)
	}
	if err := parseStringSlice(excludePathsBytes, &r.ExcludePaths); err != nil {
		return nil, fmt.Errorf("parse exclude paths: %w", err)
	}
	return &r, nil
}

// CreateRestore creates a new restore record.
func (db *DB) CreateRestore(ctx context.Context, restore *models.Restore) error {
	includePathsBytes, err := toJSONBytes(restore.IncludePaths)
	if err != nil {
		return fmt.Errorf("marshal include paths: %w", err)
	}

	excludePathsBytes, err := toJSONBytes(restore.ExcludePaths)
	if err != nil {
		return fmt.Errorf("marshal exclude paths: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO restores (id, agent_id, repository_id, snapshot_id, target_path, include_paths,
		                      exclude_paths, status, started_at, completed_at, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, restore.ID, restore.AgentID, restore.RepositoryID, restore.SnapshotID,
		restore.TargetPath, includePathsBytes, excludePathsBytes,
		string(restore.Status), restore.StartedAt, restore.CompletedAt,
		restore.ErrorMessage, restore.CreatedAt, restore.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create restore: %w", err)
	}
	return nil
}

// UpdateRestore updates an existing restore record.
func (db *DB) UpdateRestore(ctx context.Context, restore *models.Restore) error {
	restore.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE restores
		SET status = $2, started_at = $3, completed_at = $4, error_message = $5, updated_at = $6
		WHERE id = $1
	`, restore.ID, string(restore.Status), restore.StartedAt, restore.CompletedAt,
		restore.ErrorMessage, restore.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update restore: %w", err)
	}
	return nil
}

// GetOrgIDByAgentID returns the org ID for an agent.
func (db *DB) GetOrgIDByAgentID(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		SELECT org_id FROM agents WHERE id = $1
	`, agentID).Scan(&orgID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get org by agent: %w", err)
	}
	return orgID, nil
}

// GetOrgIDByScheduleID returns the org ID for a schedule (via its agent).
func (db *DB) GetOrgIDByScheduleID(ctx context.Context, scheduleID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		SELECT a.org_id
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE s.id = $1
	`, scheduleID).Scan(&orgID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get org by schedule: %w", err)
	}
	return orgID, nil
}

// GetRepository returns a repository by ID (alias for GetRepositoryByID).
func (db *DB) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	return db.GetRepositoryByID(ctx, id)
}

// Organization methods

// GetOrganizationByID returns an organization by ID.
func (db *DB) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &org, nil
}

// GetOrganizationBySlug returns an organization by slug.
func (db *DB) GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	var org models.Organization
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get organization by slug: %w", err)
	}
	return &org, nil
}

// CreateOrganization creates a new organization.
func (db *DB) CreateOrganization(ctx context.Context, org *models.Organization) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

// UpdateOrganization updates an existing organization.
func (db *DB) UpdateOrganization(ctx context.Context, org *models.Organization) error {
	org.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations
		SET name = $2, slug = $3, updated_at = $4
		WHERE id = $1
	`, org.ID, org.Name, org.Slug, org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}
	return nil
}

// DeleteOrganization deletes an organization by ID.
func (db *DB) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	return nil
}

// Membership methods

// GetMembershipByUserAndOrg returns a user's membership in an organization.
func (db *DB) GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	var m models.OrgMembership
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, org_id, role, created_at, updated_at
		FROM org_memberships
		WHERE user_id = $1 AND org_id = $2
	`, userID, orgID).Scan(&m.ID, &m.UserID, &m.OrgID, &roleStr, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get membership: %w", err)
	}
	m.Role = models.OrgRole(roleStr)
	return &m, nil
}

// GetMembershipsByUserID returns all memberships for a user.
func (db *DB) GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, org_id, role, created_at, updated_at
		FROM org_memberships
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*models.OrgMembership
	for rows.Next() {
		var m models.OrgMembership
		var roleStr string
		if err := rows.Scan(&m.ID, &m.UserID, &m.OrgID, &roleStr, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		memberships = append(memberships, &m)
	}
	return memberships, nil
}

// GetMembershipsByOrgID returns all memberships for an organization.
func (db *DB) GetMembershipsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMembershipWithUser, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT m.id, m.user_id, m.org_id, m.role, u.email, u.name, m.created_at, m.updated_at
		FROM org_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.org_id = $1
		ORDER BY m.created_at
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*models.OrgMembershipWithUser
	for rows.Next() {
		var m models.OrgMembershipWithUser
		var roleStr string
		if err := rows.Scan(&m.ID, &m.UserID, &m.OrgID, &roleStr, &m.Email, &m.Name, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		memberships = append(memberships, &m)
	}
	return memberships, nil
}

// CreateMembership creates a new organization membership.
func (db *DB) CreateMembership(ctx context.Context, m *models.OrgMembership) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_memberships (id, user_id, org_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.UserID, m.OrgID, string(m.Role), m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

// UpdateMembership updates an existing membership.
func (db *DB) UpdateMembership(ctx context.Context, m *models.OrgMembership) error {
	m.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_memberships
		SET role = $3, updated_at = $4
		WHERE user_id = $1 AND org_id = $2
	`, m.UserID, m.OrgID, string(m.Role), m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update membership: %w", err)
	}
	return nil
}

// DeleteMembership removes a user from an organization.
func (db *DB) DeleteMembership(ctx context.Context, userID, orgID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM org_memberships
		WHERE user_id = $1 AND org_id = $2
	`, userID, orgID)
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	return nil
}

// GetUserOrganizations returns all organizations a user belongs to.
func (db *DB) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM organizations o
		JOIN org_memberships m ON m.org_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, &org)
	}
	return orgs, nil
}

// Invitation methods

// CreateInvitation creates a new organization invitation.
func (db *DB) CreateInvitation(ctx context.Context, inv *models.OrgInvitation) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_invitations (id, org_id, email, role, token, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, inv.ID, inv.OrgID, inv.Email, string(inv.Role), inv.Token, inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt)
	if err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

// GetInvitationByToken returns an invitation by its token.
func (db *DB) GetInvitationByToken(ctx context.Context, token string) (*models.OrgInvitation, error) {
	var inv models.OrgInvitation
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		FROM org_invitations
		WHERE token = $1
	`, token).Scan(&inv.ID, &inv.OrgID, &inv.Email, &roleStr, &inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get invitation: %w", err)
	}
	inv.Role = models.OrgRole(roleStr)
	return &inv, nil
}

// GetPendingInvitationsByOrgID returns all pending invitations for an organization.
func (db *DB) GetPendingInvitationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.OrgInvitationWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT i.id, i.org_id, o.name, i.email, i.role, i.invited_by, u.name, i.expires_at, i.accepted_at, i.created_at
		FROM org_invitations i
		JOIN organizations o ON o.id = i.org_id
		JOIN users u ON u.id = i.invited_by
		WHERE i.org_id = $1 AND i.accepted_at IS NULL AND i.expires_at > NOW()
		ORDER BY i.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	var invitations []*models.OrgInvitationWithDetails
	for rows.Next() {
		var inv models.OrgInvitationWithDetails
		var roleStr string
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.OrgName, &inv.Email, &roleStr, &inv.InvitedBy, &inv.InviterName, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		inv.Role = models.OrgRole(roleStr)
		invitations = append(invitations, &inv)
	}
	return invitations, nil
}

// GetPendingInvitationsByEmail returns pending invitations for an email address.
func (db *DB) GetPendingInvitationsByEmail(ctx context.Context, email string) ([]*models.OrgInvitationWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT i.id, i.org_id, o.name, i.email, i.role, i.invited_by, u.name, i.expires_at, i.accepted_at, i.created_at
		FROM org_invitations i
		JOIN organizations o ON o.id = i.org_id
		JOIN users u ON u.id = i.invited_by
		WHERE i.email = $1 AND i.accepted_at IS NULL AND i.expires_at > NOW()
		ORDER BY i.created_at DESC
	`, email)
	if err != nil {
		return nil, fmt.Errorf("list invitations by email: %w", err)
	}
	defer rows.Close()

	var invitations []*models.OrgInvitationWithDetails
	for rows.Next() {
		var inv models.OrgInvitationWithDetails
		var roleStr string
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.OrgName, &inv.Email, &roleStr, &inv.InvitedBy, &inv.InviterName, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		inv.Role = models.OrgRole(roleStr)
		invitations = append(invitations, &inv)
	}
	return invitations, nil
}

// AcceptInvitation marks an invitation as accepted.
func (db *DB) AcceptInvitation(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_invitations
		SET accepted_at = $2
		WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}
	return nil
}

// DeleteInvitation deletes an invitation.
func (db *DB) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM org_invitations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}
	return nil
}

// Notification Channel methods

// GetNotificationChannelsByOrgID returns all notification channels for an organization.
func (db *DB) GetNotificationChannelsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationChannel, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, config_encrypted, enabled, created_at, updated_at
		FROM notification_channels
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []*models.NotificationChannel
	for rows.Next() {
		var c models.NotificationChannel
		var typeStr string
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.Name, &typeStr, &c.ConfigEncrypted,
			&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		c.Type = models.NotificationChannelType(typeStr)
		channels = append(channels, &c)
	}

	return channels, nil
}

// GetNotificationChannelByID returns a notification channel by ID.
func (db *DB) GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error) {
	var c models.NotificationChannel
	var typeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, config_encrypted, enabled, created_at, updated_at
		FROM notification_channels
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.Name, &typeStr, &c.ConfigEncrypted,
		&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification channel: %w", err)
	}
	c.Type = models.NotificationChannelType(typeStr)
	return &c, nil
}

// GetEnabledEmailChannelsByOrgID returns all enabled email channels for an organization.
func (db *DB) GetEnabledEmailChannelsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationChannel, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, config_encrypted, enabled, created_at, updated_at
		FROM notification_channels
		WHERE org_id = $1 AND type = 'email' AND enabled = true
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list enabled email channels: %w", err)
	}
	defer rows.Close()

	var channels []*models.NotificationChannel
	for rows.Next() {
		var c models.NotificationChannel
		var typeStr string
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.Name, &typeStr, &c.ConfigEncrypted,
			&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		c.Type = models.NotificationChannelType(typeStr)
		channels = append(channels, &c)
	}

	return channels, nil
}

// CreateNotificationChannel creates a new notification channel.
func (db *DB) CreateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO notification_channels (id, org_id, name, type, config_encrypted, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, channel.ID, channel.OrgID, channel.Name, string(channel.Type), channel.ConfigEncrypted,
		channel.Enabled, channel.CreatedAt, channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification channel: %w", err)
	}
	return nil
}

// scanRestores scans multiple restore rows.
func scanRestores(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Restore, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var restores []*models.Restore
	for r.Next() {
		var restore models.Restore
		var statusStr string
		var includePathsBytes, excludePathsBytes []byte
		if err := r.Scan(
			&restore.ID, &restore.AgentID, &restore.RepositoryID, &restore.SnapshotID,
			&restore.TargetPath, &includePathsBytes, &excludePathsBytes, &statusStr,
			&restore.StartedAt, &restore.CompletedAt, &restore.ErrorMessage,
			&restore.CreatedAt, &restore.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan restore: %w", err)
		}
		restore.Status = models.RestoreStatus(statusStr)
		if err := parseStringSlice(includePathsBytes, &restore.IncludePaths); err != nil {
			return nil, fmt.Errorf("parse include paths: %w", err)
		}
		if err := parseStringSlice(excludePathsBytes, &restore.ExcludePaths); err != nil {
			return nil, fmt.Errorf("parse exclude paths: %w", err)
		}
		restores = append(restores, &restore)
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate restores: %w", err)
	}
	return restores, nil
}

// toJSONBytes converts a slice to JSON bytes for database storage.
func toJSONBytes(slice []string) ([]byte, error) {
	if len(slice) == 0 {
		return nil, nil
	}
	return json.Marshal(slice)
}

// parseStringSlice parses JSON bytes into a string slice.
func parseStringSlice(data []byte, dest *[]string) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dest)
}

// UpdateNotificationChannel updates an existing notification channel.
func (db *DB) UpdateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error {
	channel.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE notification_channels
		SET name = $2, config_encrypted = $3, enabled = $4, updated_at = $5
		WHERE id = $1
	`, channel.ID, channel.Name, channel.ConfigEncrypted, channel.Enabled, channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update notification channel: %w", err)
	}
	return nil
}

// DeleteNotificationChannel deletes a notification channel by ID.
func (db *DB) DeleteNotificationChannel(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM notification_channels WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete notification channel: %w", err)
	}
	return nil
}

// Notification Preference methods

// GetNotificationPreferencesByOrgID returns all notification preferences for an organization.
func (db *DB) GetNotificationPreferencesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationPreference, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, channel_id, event_type, enabled, created_at, updated_at
		FROM notification_preferences
		WHERE org_id = $1
		ORDER BY event_type
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list notification preferences: %w", err)
	}
	defer rows.Close()

	var prefs []*models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		var eventTypeStr string
		err := rows.Scan(
			&p.ID, &p.OrgID, &p.ChannelID, &eventTypeStr, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification preference: %w", err)
		}
		p.EventType = models.NotificationEventType(eventTypeStr)
		prefs = append(prefs, &p)
	}

	return prefs, nil
}

// GetNotificationPreferencesByChannelID returns all preferences for a channel.
func (db *DB) GetNotificationPreferencesByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.NotificationPreference, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, channel_id, event_type, enabled, created_at, updated_at
		FROM notification_preferences
		WHERE channel_id = $1
		ORDER BY event_type
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list channel preferences: %w", err)
	}
	defer rows.Close()

	var prefs []*models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		var eventTypeStr string
		err := rows.Scan(
			&p.ID, &p.OrgID, &p.ChannelID, &eventTypeStr, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification preference: %w", err)
		}
		p.EventType = models.NotificationEventType(eventTypeStr)
		prefs = append(prefs, &p)
	}

	return prefs, nil
}

// GetEnabledPreferencesForEvent returns all enabled preferences for a specific event type in an org.
func (db *DB) GetEnabledPreferencesForEvent(ctx context.Context, orgID uuid.UUID, eventType models.NotificationEventType) ([]*models.NotificationPreference, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT np.id, np.org_id, np.channel_id, np.event_type, np.enabled, np.created_at, np.updated_at
		FROM notification_preferences np
		JOIN notification_channels nc ON np.channel_id = nc.id
		WHERE np.org_id = $1 AND np.event_type = $2 AND np.enabled = true AND nc.enabled = true
	`, orgID, string(eventType))
	if err != nil {
		return nil, fmt.Errorf("list enabled preferences for event: %w", err)
	}
	defer rows.Close()

	var prefs []*models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		var eventTypeStr string
		err := rows.Scan(
			&p.ID, &p.OrgID, &p.ChannelID, &eventTypeStr, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification preference: %w", err)
		}
		p.EventType = models.NotificationEventType(eventTypeStr)
		prefs = append(prefs, &p)
	}

	return prefs, nil
}

// CreateNotificationPreference creates a new notification preference.
func (db *DB) CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO notification_preferences (id, org_id, channel_id, event_type, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pref.ID, pref.OrgID, pref.ChannelID, string(pref.EventType), pref.Enabled,
		pref.CreatedAt, pref.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification preference: %w", err)
	}
	return nil
}

// UpdateNotificationPreference updates an existing notification preference.
func (db *DB) UpdateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	pref.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE notification_preferences
		SET enabled = $2, updated_at = $3
		WHERE id = $1
	`, pref.ID, pref.Enabled, pref.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update notification preference: %w", err)
	}
	return nil
}

// DeleteNotificationPreference deletes a notification preference by ID.
func (db *DB) DeleteNotificationPreference(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM notification_preferences WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete notification preference: %w", err)
	}
	return nil
}

// Notification Log methods

// GetNotificationLogsByOrgID returns notification logs for an organization.
func (db *DB) GetNotificationLogsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.NotificationLog, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, channel_id, event_type, recipient, subject, status, error_message, sent_at, created_at
		FROM notification_logs
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notification logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.NotificationLog
	for rows.Next() {
		var l models.NotificationLog
		var statusStr string
		err := rows.Scan(
			&l.ID, &l.OrgID, &l.ChannelID, &l.EventType, &l.Recipient,
			&l.Subject, &statusStr, &l.ErrorMessage, &l.SentAt, &l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification log: %w", err)
		}
		l.Status = models.NotificationStatus(statusStr)
		logs = append(logs, &l)
	}

	return logs, nil
}

// CreateNotificationLog creates a new notification log entry.
func (db *DB) CreateNotificationLog(ctx context.Context, log *models.NotificationLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO notification_logs (id, org_id, channel_id, event_type, recipient, subject, status, error_message, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, log.ID, log.OrgID, log.ChannelID, log.EventType, log.Recipient,
		log.Subject, string(log.Status), log.ErrorMessage, log.SentAt, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("create notification log: %w", err)
	}
	return nil
}

// UpdateNotificationLog updates an existing notification log entry.
func (db *DB) UpdateNotificationLog(ctx context.Context, log *models.NotificationLog) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE notification_logs
		SET status = $2, error_message = $3, sent_at = $4
		WHERE id = $1
	`, log.ID, string(log.Status), log.ErrorMessage, log.SentAt)
	if err != nil {
		return fmt.Errorf("update notification log: %w", err)
	}
	return nil
}

// Repository Key methods

// GetRepositoryKeyByRepositoryID returns a repository key by repository ID.
func (db *DB) GetRepositoryKeyByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryKey, error) {
	var rk models.RepositoryKey
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, encrypted_key, escrow_enabled, escrow_encrypted_key, created_at, updated_at
		FROM repository_keys
		WHERE repository_id = $1
	`, repositoryID).Scan(
		&rk.ID, &rk.RepositoryID, &rk.EncryptedKey, &rk.EscrowEnabled,
		&rk.EscrowEncryptedKey, &rk.CreatedAt, &rk.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get repository key: %w", err)
	}
	return &rk, nil
}

// CreateRepositoryKey creates a new repository key.
func (db *DB) CreateRepositoryKey(ctx context.Context, rk *models.RepositoryKey) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repository_keys (id, repository_id, encrypted_key, escrow_enabled, escrow_encrypted_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, rk.ID, rk.RepositoryID, rk.EncryptedKey, rk.EscrowEnabled,
		rk.EscrowEncryptedKey, rk.CreatedAt, rk.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create repository key: %w", err)
	}
	return nil
}

// UpdateRepositoryKeyEscrow updates the escrow settings for a repository key.
func (db *DB) UpdateRepositoryKeyEscrow(ctx context.Context, repositoryID uuid.UUID, escrowEnabled bool, escrowEncryptedKey []byte) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE repository_keys
		SET escrow_enabled = $2, escrow_encrypted_key = $3, updated_at = NOW()
		WHERE repository_id = $1
	`, repositoryID, escrowEnabled, escrowEncryptedKey)
	if err != nil {
		return fmt.Errorf("update repository key escrow: %w", err)
	}
	return nil
}

// DeleteRepositoryKey deletes a repository key by repository ID.
func (db *DB) DeleteRepositoryKey(ctx context.Context, repositoryID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM repository_keys WHERE repository_id = $1`, repositoryID)
	if err != nil {
		return fmt.Errorf("delete repository key: %w", err)
	}
	return nil
}

// GetRepositoryKeysWithEscrowByOrgID returns all repository keys with escrow enabled for an organization.
// This is used by admins for key recovery.
func (db *DB) GetRepositoryKeysWithEscrowByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RepositoryKey, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT rk.id, rk.repository_id, rk.encrypted_key, rk.escrow_enabled, rk.escrow_encrypted_key, rk.created_at, rk.updated_at
		FROM repository_keys rk
		INNER JOIN repositories r ON rk.repository_id = r.id
		WHERE r.org_id = $1 AND rk.escrow_enabled = true
		ORDER BY rk.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list repository keys with escrow: %w", err)
	}
	defer rows.Close()

	var keys []*models.RepositoryKey
	for rows.Next() {
		var rk models.RepositoryKey
		err := rows.Scan(
			&rk.ID, &rk.RepositoryID, &rk.EncryptedKey, &rk.EscrowEnabled,
			&rk.EscrowEncryptedKey, &rk.CreatedAt, &rk.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan repository key: %w", err)
		}
		keys = append(keys, &rk)
	}

	return keys, nil
}

// Audit log methods

// AuditLogFilter holds filter parameters for querying audit logs.
type AuditLogFilter struct {
	Action       string
	ResourceType string
	Result       string
	StartDate    *time.Time
	EndDate      *time.Time
	Search       string
	Limit        int
	Offset       int
}

// GetAuditLogsByOrgID returns audit logs for an organization with optional filters.
func (db *DB) GetAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter AuditLogFilter) ([]*models.AuditLog, error) {
	query := `
		SELECT id, org_id, user_id, agent_id, action, resource_type, resource_id,
		       result, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE org_id = $1
	`
	args := []any{orgID}
	argNum := 2

	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, filter.Action)
		argNum++
	}
	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argNum)
		args = append(args, filter.ResourceType)
		argNum++
	}
	if filter.Result != "" {
		query += fmt.Sprintf(" AND result = $%d", argNum)
		args = append(args, filter.Result)
		argNum++
	}
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, filter.StartDate)
		argNum++
	}
	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, filter.EndDate)
		argNum++
	}
	if filter.Search != "" {
		query += fmt.Sprintf(" AND (details ILIKE $%d OR resource_type ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// GetAuditLogByID returns an audit log by ID.
func (db *DB) GetAuditLogByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var a models.AuditLog
	var actionStr, resultStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, agent_id, action, resource_type, resource_id,
		       result, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.UserID, &a.AgentID, &actionStr, &a.ResourceType,
		&a.ResourceID, &resultStr, &a.IPAddress, &a.UserAgent, &a.Details, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	a.Action = models.AuditAction(actionStr)
	a.Result = models.AuditResult(resultStr)
	return &a, nil
}

// CreateAuditLog creates a new audit log entry.
func (db *DB) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (id, org_id, user_id, agent_id, action, resource_type,
		                        resource_id, result, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, log.ID, log.OrgID, log.UserID, log.AgentID, string(log.Action), log.ResourceType,
		log.ResourceID, string(log.Result), log.IPAddress, log.UserAgent, log.Details, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// CountAuditLogsByOrgID returns the total count of audit logs for an organization with filters.
func (db *DB) CountAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter AuditLogFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE org_id = $1`
	args := []any{orgID}
	argNum := 2

	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, filter.Action)
		argNum++
	}
	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argNum)
		args = append(args, filter.ResourceType)
		argNum++
	}
	if filter.Result != "" {
		query += fmt.Sprintf(" AND result = $%d", argNum)
		args = append(args, filter.Result)
		argNum++
	}
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, filter.StartDate)
		argNum++
	}
	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, filter.EndDate)
		argNum++
	}
	if filter.Search != "" {
		query += fmt.Sprintf(" AND (details ILIKE $%d OR resource_type ILIKE $%d)", argNum, argNum)
		args = append(args, "%"+filter.Search+"%")
	}

	var count int64
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}
	return count, nil
}

// scanAuditLogs scans multiple audit log rows.
func scanAuditLogs(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.AuditLog, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var logs []*models.AuditLog
	for r.Next() {
		var a models.AuditLog
		var actionStr, resultStr string
		err := r.Scan(
			&a.ID, &a.OrgID, &a.UserID, &a.AgentID, &actionStr, &a.ResourceType,
			&a.ResourceID, &resultStr, &a.IPAddress, &a.UserAgent, &a.Details, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		a.Action = models.AuditAction(actionStr)
		a.Result = models.AuditResult(resultStr)
		logs = append(logs, &a)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}

	return logs, nil
}

// Storage Stats query methods

// GetLatestStorageStats returns the most recent storage stats for a repository.
func (db *DB) GetLatestStorageStats(ctx context.Context, repositoryID uuid.UUID) (*models.StorageStats, error) {
	var s models.StorageStats
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, total_size, total_file_count, raw_data_size, restore_size,
		       dedup_ratio, space_saved, space_saved_pct, snapshot_count, collected_at, created_at
		FROM storage_stats
		WHERE repository_id = $1
		ORDER BY collected_at DESC
		LIMIT 1
	`, repositoryID).Scan(
		&s.ID, &s.RepositoryID, &s.TotalSize, &s.TotalFileCount, &s.RawDataSize,
		&s.RestoreSize, &s.DedupRatio, &s.SpaceSaved, &s.SpaceSavedPct,
		&s.SnapshotCount, &s.CollectedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest storage stats: %w", err)
	}
	return &s, nil
}

// GetStorageStatsByRepositoryID returns all storage stats for a repository ordered by date.
func (db *DB) GetStorageStatsByRepositoryID(ctx context.Context, repositoryID uuid.UUID, limit int) ([]*models.StorageStats, error) {
	query := `
		SELECT id, repository_id, total_size, total_file_count, raw_data_size, restore_size,
		       dedup_ratio, space_saved, space_saved_pct, snapshot_count, collected_at, created_at
		FROM storage_stats
		WHERE repository_id = $1
		ORDER BY collected_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Pool.Query(ctx, query, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list storage stats: %w", err)
	}
	defer rows.Close()

	return scanStorageStats(rows)
}

// GetStorageStatsSummary returns aggregated storage statistics for all repositories in an org.
func (db *DB) GetStorageStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageStatsSummary, error) {
	var summary models.StorageStatsSummary
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(latest.raw_data_size), 0) as total_raw_size,
			COALESCE(SUM(latest.restore_size), 0) as total_restore_size,
			COALESCE(SUM(latest.space_saved), 0) as total_space_saved,
			COALESCE(AVG(latest.dedup_ratio), 0) as avg_dedup_ratio,
			COUNT(DISTINCT latest.repository_id) as repository_count,
			COALESCE(SUM(latest.snapshot_count), 0) as total_snapshots
		FROM (
			SELECT DISTINCT ON (s.repository_id)
				s.repository_id, s.raw_data_size, s.restore_size, s.space_saved,
				s.dedup_ratio, s.snapshot_count
			FROM storage_stats s
			JOIN repositories r ON s.repository_id = r.id
			WHERE r.org_id = $1
			ORDER BY s.repository_id, s.collected_at DESC
		) as latest
	`, orgID).Scan(
		&summary.TotalRawSize, &summary.TotalRestoreSize, &summary.TotalSpaceSaved,
		&summary.AvgDedupRatio, &summary.RepositoryCount, &summary.TotalSnapshots,
	)
	if err != nil {
		return nil, fmt.Errorf("get storage stats summary: %w", err)
	}
	return &summary, nil
}

// GetStorageGrowth returns storage growth data points for a repository over a time period.
func (db *DB) GetStorageGrowth(ctx context.Context, repositoryID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DATE(collected_at) as date,
		       MAX(raw_data_size) as raw_data_size,
		       MAX(restore_size) as restore_size
		FROM storage_stats
		WHERE repository_id = $1
		  AND collected_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY DATE(collected_at)
		ORDER BY date ASC
	`, repositoryID, days)
	if err != nil {
		return nil, fmt.Errorf("get storage growth: %w", err)
	}
	defer rows.Close()

	var points []*models.StorageGrowthPoint
	for rows.Next() {
		var p models.StorageGrowthPoint
		err := rows.Scan(&p.Date, &p.RawDataSize, &p.RestoreSize)
		if err != nil {
			return nil, fmt.Errorf("scan storage growth: %w", err)
		}
		points = append(points, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage growth: %w", err)
	}

	return points, nil
}

// GetAllStorageGrowth returns aggregated storage growth for all repositories in an org.
func (db *DB) GetAllStorageGrowth(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DATE(s.collected_at) as date,
		       SUM(latest.raw_data_size) as raw_data_size,
		       SUM(latest.restore_size) as restore_size
		FROM (
			SELECT DISTINCT ON (repository_id, DATE(collected_at))
				repository_id, collected_at, raw_data_size, restore_size
			FROM storage_stats
			WHERE repository_id IN (SELECT id FROM repositories WHERE org_id = $1)
			  AND collected_at >= NOW() - INTERVAL '1 day' * $2
			ORDER BY repository_id, DATE(collected_at), collected_at DESC
		) as latest
		JOIN storage_stats s ON s.id = (
			SELECT id FROM storage_stats
			WHERE repository_id = latest.repository_id
			  AND DATE(collected_at) = DATE(latest.collected_at)
			ORDER BY collected_at DESC
			LIMIT 1
		)
		GROUP BY DATE(s.collected_at)
		ORDER BY date ASC
	`, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("get all storage growth: %w", err)
	}
	defer rows.Close()

	var points []*models.StorageGrowthPoint
	for rows.Next() {
		var p models.StorageGrowthPoint
		err := rows.Scan(&p.Date, &p.RawDataSize, &p.RestoreSize)
		if err != nil {
			return nil, fmt.Errorf("scan all storage growth: %w", err)
		}
		points = append(points, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all storage growth: %w", err)
	}

	return points, nil
}

// GetLatestStatsForAllRepos returns the latest storage stats for all repositories in an org.
func (db *DB) GetLatestStatsForAllRepos(ctx context.Context, orgID uuid.UUID) ([]*models.StorageStats, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT ON (s.repository_id)
			s.id, s.repository_id, s.total_size, s.total_file_count, s.raw_data_size,
			s.restore_size, s.dedup_ratio, s.space_saved, s.space_saved_pct,
			s.snapshot_count, s.collected_at, s.created_at
		FROM storage_stats s
		JOIN repositories r ON s.repository_id = r.id
		WHERE r.org_id = $1
		ORDER BY s.repository_id, s.collected_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get latest stats for all repos: %w", err)
	}
	defer rows.Close()

	return scanStorageStats(rows)
}

// scanStorageStats scans multiple storage stats rows.
func scanStorageStats(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.StorageStats, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var stats []*models.StorageStats
	for r.Next() {
		var s models.StorageStats
		err := r.Scan(
			&s.ID, &s.RepositoryID, &s.TotalSize, &s.TotalFileCount, &s.RawDataSize,
			&s.RestoreSize, &s.DedupRatio, &s.SpaceSaved, &s.SpaceSavedPct,
			&s.SnapshotCount, &s.CollectedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan storage stats: %w", err)
		}
		stats = append(stats, &s)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage stats: %w", err)
	}

	return stats, nil
}
// Verification Schedule methods

// GetEnabledVerificationSchedules returns all enabled verification schedules.
func (db *DB) GetEnabledVerificationSchedules(ctx context.Context) ([]*models.VerificationSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, type, cron_expression, enabled, read_data_subset, created_at, updated_at
		FROM verification_schedules
		WHERE enabled = true
		ORDER BY repository_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled verification schedules: %w", err)
	}
	defer rows.Close()

	return scanVerificationSchedules(rows)
}

// GetVerificationSchedulesByRepoID returns verification schedules for a repository.
func (db *DB) GetVerificationSchedulesByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.VerificationSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, type, cron_expression, enabled, read_data_subset, created_at, updated_at
		FROM verification_schedules
		WHERE repository_id = $1
		ORDER BY type
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("list verification schedules by repo: %w", err)
	}
	defer rows.Close()

	return scanVerificationSchedules(rows)
}

// GetVerificationScheduleByID returns a verification schedule by ID.
func (db *DB) GetVerificationScheduleByID(ctx context.Context, id uuid.UUID) (*models.VerificationSchedule, error) {
	var vs models.VerificationSchedule
	var typeStr string
	var readDataSubset *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, type, cron_expression, enabled, read_data_subset, created_at, updated_at
		FROM verification_schedules
		WHERE id = $1
	`, id).Scan(
		&vs.ID, &vs.RepositoryID, &typeStr, &vs.CronExpression,
		&vs.Enabled, &readDataSubset, &vs.CreatedAt, &vs.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get verification schedule: %w", err)
	}
	vs.Type = models.VerificationType(typeStr)
	if readDataSubset != nil {
		vs.ReadDataSubset = *readDataSubset
	}
	return &vs, nil
}

// CreateVerificationSchedule creates a new verification schedule.
func (db *DB) CreateVerificationSchedule(ctx context.Context, vs *models.VerificationSchedule) error {
	var readDataSubset *string
	if vs.ReadDataSubset != "" {
		readDataSubset = &vs.ReadDataSubset
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO verification_schedules (id, repository_id, type, cron_expression, enabled, read_data_subset, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, vs.ID, vs.RepositoryID, string(vs.Type), vs.CronExpression,
		vs.Enabled, readDataSubset, vs.CreatedAt, vs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create verification schedule: %w", err)
	}
	return nil
}

// UpdateVerificationSchedule updates an existing verification schedule.
func (db *DB) UpdateVerificationSchedule(ctx context.Context, vs *models.VerificationSchedule) error {
	var readDataSubset *string
	if vs.ReadDataSubset != "" {
		readDataSubset = &vs.ReadDataSubset
	}

	vs.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE verification_schedules
		SET cron_expression = $2, enabled = $3, read_data_subset = $4, updated_at = $5
		WHERE id = $1
	`, vs.ID, vs.CronExpression, vs.Enabled, readDataSubset, vs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update verification schedule: %w", err)
	}
	return nil
}

// DeleteVerificationSchedule deletes a verification schedule by ID.
func (db *DB) DeleteVerificationSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM verification_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete verification schedule: %w", err)
	}
	return nil
}

// scanVerificationSchedules scans multiple verification schedule rows.
func scanVerificationSchedules(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.VerificationSchedule, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var schedules []*models.VerificationSchedule
	for r.Next() {
		var vs models.VerificationSchedule
		var typeStr string
		var readDataSubset *string
		err := r.Scan(
			&vs.ID, &vs.RepositoryID, &typeStr, &vs.CronExpression,
			&vs.Enabled, &readDataSubset, &vs.CreatedAt, &vs.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan verification schedule: %w", err)
		}
		vs.Type = models.VerificationType(typeStr)
		if readDataSubset != nil {
			vs.ReadDataSubset = *readDataSubset
		}
		schedules = append(schedules, &vs)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate verification schedules: %w", err)
	}

	return schedules, nil
}

// Verification methods

// GetVerificationsByRepoID returns all verifications for a repository.
func (db *DB) GetVerificationsByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.Verification, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, type, snapshot_id, started_at, completed_at,
		       status, duration_ms, error_message, details, created_at
		FROM verifications
		WHERE repository_id = $1
		ORDER BY started_at DESC
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("list verifications by repo: %w", err)
	}
	defer rows.Close()

	return scanVerifications(rows)
}

// GetLatestVerificationByRepoID returns the most recent verification for a repository.
func (db *DB) GetLatestVerificationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.Verification, error) {
	var v models.Verification
	var typeStr, statusStr string
	var snapshotID *string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, type, snapshot_id, started_at, completed_at,
		       status, duration_ms, error_message, details, created_at
		FROM verifications
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, repoID).Scan(
		&v.ID, &v.RepositoryID, &typeStr, &snapshotID, &v.StartedAt,
		&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage,
		&detailsBytes, &v.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest verification: %w", err)
	}
	v.Type = models.VerificationType(typeStr)
	v.Status = models.VerificationStatus(statusStr)
	if snapshotID != nil {
		v.SnapshotID = *snapshotID
	}
	if err := v.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("verification_id", v.ID.String()).Msg("failed to parse verification details")
	}
	return &v, nil
}

// GetVerificationByID returns a verification by ID.
func (db *DB) GetVerificationByID(ctx context.Context, id uuid.UUID) (*models.Verification, error) {
	var v models.Verification
	var typeStr, statusStr string
	var snapshotID *string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, type, snapshot_id, started_at, completed_at,
		       status, duration_ms, error_message, details, created_at
		FROM verifications
		WHERE id = $1
	`, id).Scan(
		&v.ID, &v.RepositoryID, &typeStr, &snapshotID, &v.StartedAt,
		&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage,
		&detailsBytes, &v.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get verification: %w", err)
	}
	v.Type = models.VerificationType(typeStr)
	v.Status = models.VerificationStatus(statusStr)
	if snapshotID != nil {
		v.SnapshotID = *snapshotID
	}
	if err := v.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("verification_id", v.ID.String()).Msg("failed to parse verification details")
	}
	return &v, nil
}

// CreateVerification creates a new verification record.
func (db *DB) CreateVerification(ctx context.Context, v *models.Verification) error {
	detailsBytes, err := v.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal verification details: %w", err)
	}

	var snapshotID *string
	if v.SnapshotID != "" {
		snapshotID = &v.SnapshotID
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO verifications (id, repository_id, type, snapshot_id, started_at, completed_at,
		                           status, duration_ms, error_message, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, v.ID, v.RepositoryID, string(v.Type), snapshotID, v.StartedAt,
		v.CompletedAt, string(v.Status), v.DurationMs, v.ErrorMessage,
		detailsBytes, v.CreatedAt)
	if err != nil {
		return fmt.Errorf("create verification: %w", err)
	}
	return nil
}

// UpdateVerification updates an existing verification record.
func (db *DB) UpdateVerification(ctx context.Context, v *models.Verification) error {
	detailsBytes, err := v.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal verification details: %w", err)
	}

	var snapshotID *string
	if v.SnapshotID != "" {
		snapshotID = &v.SnapshotID
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE verifications
		SET snapshot_id = $2, completed_at = $3, status = $4, duration_ms = $5,
		    error_message = $6, details = $7
		WHERE id = $1
	`, v.ID, snapshotID, v.CompletedAt, string(v.Status), v.DurationMs,
		v.ErrorMessage, detailsBytes)
	if err != nil {
		return fmt.Errorf("update verification: %w", err)
	}
	return nil
}

// GetConsecutiveFailedVerifications returns the count of consecutive failed verifications for a repository.
func (db *DB) GetConsecutiveFailedVerifications(ctx context.Context, repoID uuid.UUID) (int, error) {
	// Count consecutive failures from the most recent verification backwards
	rows, err := db.Pool.Query(ctx, `
		SELECT status
		FROM verifications
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT 10
	`, repoID)
	if err != nil {
		return 0, fmt.Errorf("get recent verifications: %w", err)
	}
	defer rows.Close()

	var consecutiveFails int
	for rows.Next() {
		var statusStr string
		if err := rows.Scan(&statusStr); err != nil {
			return 0, fmt.Errorf("scan verification status: %w", err)
		}
		if models.VerificationStatus(statusStr) == models.VerificationStatusFailed {
			consecutiveFails++
		} else {
			break
		}
	}

	return consecutiveFails, nil
}

// scanVerifications scans multiple verification rows.
func scanVerifications(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Verification, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var verifications []*models.Verification
	for r.Next() {
		var v models.Verification
		var typeStr, statusStr string
		var snapshotID *string
		var detailsBytes []byte
		err := r.Scan(
			&v.ID, &v.RepositoryID, &typeStr, &snapshotID, &v.StartedAt,
			&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage,
			&detailsBytes, &v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan verification: %w", err)
		}
		v.Type = models.VerificationType(typeStr)
		v.Status = models.VerificationStatus(statusStr)
		if snapshotID != nil {
			v.SnapshotID = *snapshotID
		}
		// Skip setting details to avoid warning spam
		verifications = append(verifications, &v)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate verifications: %w", err)
	}

	return verifications, nil
}

// Maintenance Window methods

// GetMaintenanceWindowByID returns a maintenance window by ID.
func (db *DB) GetMaintenanceWindowByID(ctx context.Context, id uuid.UUID) (*models.MaintenanceWindow, error) {
	var m models.MaintenanceWindow
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE id = $1
	`, id).Scan(
		&m.ID, &m.OrgID, &m.Title, &m.Message, &m.StartsAt, &m.EndsAt,
		&m.NotifyBeforeMinutes, &m.NotificationSent, &m.CreatedBy,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get maintenance window: %w", err)
	}
	return &m, nil
}

// ListMaintenanceWindowsByOrg returns all maintenance windows for an organization.
func (db *DB) ListMaintenanceWindowsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.MaintenanceWindow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE org_id = $1
		ORDER BY starts_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list maintenance windows: %w", err)
	}
	defer rows.Close()

	return scanMaintenanceWindows(rows)
}

// ListActiveMaintenanceWindows returns maintenance windows that are currently active.
func (db *DB) ListActiveMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time) ([]*models.MaintenanceWindow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE org_id = $1
		  AND starts_at <= $2
		  AND ends_at > $2
		ORDER BY ends_at ASC
	`, orgID, now)
	if err != nil {
		return nil, fmt.Errorf("list active maintenance windows: %w", err)
	}
	defer rows.Close()

	return scanMaintenanceWindows(rows)
}

// ListUpcomingMaintenanceWindows returns maintenance windows starting within the given minutes.
func (db *DB) ListUpcomingMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time, withinMinutes int) ([]*models.MaintenanceWindow, error) {
	notifyTime := now.Add(time.Duration(withinMinutes) * time.Minute)
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE org_id = $1
		  AND starts_at > $2
		  AND starts_at <= $3
		ORDER BY starts_at ASC
	`, orgID, now, notifyTime)
	if err != nil {
		return nil, fmt.Errorf("list upcoming maintenance windows: %w", err)
	}
	defer rows.Close()

	return scanMaintenanceWindows(rows)
}

// ListPendingMaintenanceNotifications returns windows that need notifications sent.
func (db *DB) ListPendingMaintenanceNotifications(ctx context.Context) ([]*models.MaintenanceWindow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE notification_sent = false
		  AND starts_at > NOW()
		  AND starts_at <= NOW() + (notify_before_minutes * INTERVAL '1 minute')
		ORDER BY starts_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list pending maintenance notifications: %w", err)
	}
	defer rows.Close()

	return scanMaintenanceWindows(rows)
}

// CreateMaintenanceWindow creates a new maintenance window.
func (db *DB) CreateMaintenanceWindow(ctx context.Context, m *models.MaintenanceWindow) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO maintenance_windows (id, org_id, title, message, starts_at, ends_at,
		            notify_before_minutes, notification_sent, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, m.ID, m.OrgID, m.Title, m.Message, m.StartsAt, m.EndsAt,
		m.NotifyBeforeMinutes, m.NotificationSent, m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create maintenance window: %w", err)
	}
	return nil
}

// Exclude Pattern methods

// GetExcludePatternsByOrgID returns all exclude patterns for an organization (including built-in patterns).
func (db *DB) GetExcludePatternsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ExcludePattern, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at
		FROM exclude_patterns
		WHERE org_id = $1 OR is_builtin = true
		ORDER BY is_builtin DESC, category, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list exclude patterns: %w", err)
	}
	defer rows.Close()

	return scanExcludePatterns(rows)
}

// GetBuiltinExcludePatterns returns only the built-in exclude patterns.
func (db *DB) GetBuiltinExcludePatterns(ctx context.Context) ([]*models.ExcludePattern, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at
		FROM exclude_patterns
		WHERE is_builtin = true
		ORDER BY category, name
	`)
	if err != nil {
		return nil, fmt.Errorf("list built-in exclude patterns: %w", err)
	}
	defer rows.Close()

	return scanExcludePatterns(rows)
}

// GetExcludePatternByID returns an exclude pattern by ID.
func (db *DB) GetExcludePatternByID(ctx context.Context, id uuid.UUID) (*models.ExcludePattern, error) {
	var ep models.ExcludePattern
	var patternsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at
		FROM exclude_patterns
		WHERE id = $1
	`, id).Scan(
		&ep.ID, &ep.OrgID, &ep.Name, &ep.Description, &patternsBytes,
		&ep.Category, &ep.IsBuiltin, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get exclude pattern: %w", err)
	}
	if err := ep.SetPatterns(patternsBytes); err != nil {
		return nil, fmt.Errorf("parse exclude patterns: %w", err)
	}
	return &ep, nil
}

// GetExcludePatternsByCategory returns all exclude patterns for a given category.
func (db *DB) GetExcludePatternsByCategory(ctx context.Context, orgID uuid.UUID, category string) ([]*models.ExcludePattern, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at
		FROM exclude_patterns
		WHERE (org_id = $1 OR is_builtin = true) AND category = $2
		ORDER BY is_builtin DESC, name
	`, orgID, category)
	if err != nil {
		return nil, fmt.Errorf("list exclude patterns by category: %w", err)
	}
	defer rows.Close()

	return scanExcludePatterns(rows)
}

// CreateExcludePattern creates a new exclude pattern.
func (db *DB) CreateExcludePattern(ctx context.Context, ep *models.ExcludePattern) error {
	patternsBytes, err := ep.PatternsJSON()
	if err != nil {
		return fmt.Errorf("marshal patterns: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO exclude_patterns (id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, ep.ID, ep.OrgID, ep.Name, ep.Description, patternsBytes, ep.Category, ep.IsBuiltin, ep.CreatedAt, ep.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create exclude pattern: %w", err)
	}
	return nil
}

// UpdateMaintenanceWindow updates an existing maintenance window.
func (db *DB) UpdateMaintenanceWindow(ctx context.Context, m *models.MaintenanceWindow) error {
	m.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE maintenance_windows
		SET title = $2, message = $3, starts_at = $4, ends_at = $5,
		    notify_before_minutes = $6, notification_sent = $7, updated_at = $8
		WHERE id = $1
	`, m.ID, m.Title, m.Message, m.StartsAt, m.EndsAt,
		m.NotifyBeforeMinutes, m.NotificationSent, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update maintenance window: %w", err)
	}
	return nil
}

// UpdateExcludePattern updates an exclude pattern.
func (db *DB) UpdateExcludePattern(ctx context.Context, ep *models.ExcludePattern) error {
	patternsBytes, err := ep.PatternsJSON()
	if err != nil {
		return fmt.Errorf("marshal patterns: %w", err)
	}

	ep.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE exclude_patterns
		SET name = $2, description = $3, patterns = $4, category = $5, updated_at = $6
		WHERE id = $1 AND is_builtin = false
	`, ep.ID, ep.Name, ep.Description, patternsBytes, ep.Category, ep.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update exclude pattern: %w", err)
	}
	return nil
}

// DeleteMaintenanceWindow deletes a maintenance window.
func (db *DB) DeleteMaintenanceWindow(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM maintenance_windows
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete maintenance window: %w", err)
	}
	return nil
}

// DeleteExcludePattern deletes an exclude pattern by ID.
func (db *DB) DeleteExcludePattern(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM exclude_patterns
		WHERE id = $1 AND is_builtin = false
	`, id)
	if err != nil {
		return fmt.Errorf("delete exclude pattern: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("exclude pattern not found or is built-in")
	}
	return nil
}

// MarkMaintenanceNotificationSent marks a maintenance window's notification as sent.
func (db *DB) MarkMaintenanceNotificationSent(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE maintenance_windows
		SET notification_sent = true, updated_at = $2
		WHERE id = $1
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("mark maintenance notification sent: %w", err)
	}
	return nil
}

// SeedBuiltinExcludePatterns inserts or updates the built-in exclude patterns from the library.
func (db *DB) SeedBuiltinExcludePatterns(ctx context.Context, patterns []*models.ExcludePattern) error {
	for _, ep := range patterns {
		patternsBytes, err := ep.PatternsJSON()
		if err != nil {
			return fmt.Errorf("marshal patterns for %s: %w", ep.Name, err)
		}

		_, err = db.Pool.Exec(ctx, `
			INSERT INTO exclude_patterns (id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at)
			VALUES ($1, NULL, $2, $3, $4, $5, true, $6, $7)
			ON CONFLICT (name) WHERE is_builtin = true
			DO UPDATE SET description = $3, patterns = $4, category = $5, updated_at = $7
		`, ep.ID, ep.Name, ep.Description, patternsBytes, ep.Category, ep.CreatedAt, ep.UpdatedAt)
		if err != nil {
			return fmt.Errorf("seed exclude pattern %s: %w", ep.Name, err)
		}
	}
	return nil
}

// scanMaintenanceWindows scans rows into maintenance window structs.
func scanMaintenanceWindows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.MaintenanceWindow, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var windows []*models.MaintenanceWindow
	for r.Next() {
		var m models.MaintenanceWindow
		err := r.Scan(
			&m.ID, &m.OrgID, &m.Title, &m.Message, &m.StartsAt, &m.EndsAt,
			&m.NotifyBeforeMinutes, &m.NotificationSent, &m.CreatedBy,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance window: %w", err)
		}
		windows = append(windows, &m)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate maintenance windows: %w", err)
	}

	return windows, nil
}

// scanExcludePatterns scans multiple exclude pattern rows.
func scanExcludePatterns(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.ExcludePattern, error) {
	var patterns []*models.ExcludePattern
	for rows.Next() {
		var ep models.ExcludePattern
		var patternsBytes []byte
		err := rows.Scan(
			&ep.ID, &ep.OrgID, &ep.Name, &ep.Description, &patternsBytes,
			&ep.Category, &ep.IsBuiltin, &ep.CreatedAt, &ep.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan exclude pattern: %w", err)
		}
		if err := ep.SetPatterns(patternsBytes); err != nil {
			// Log warning but continue
			ep.Patterns = []string{}
		}
		patterns = append(patterns, &ep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exclude patterns: %w", err)
	}

	return patterns, nil
}

//
// DR Runbook methods

// GetDRRunbooksByOrgID returns all DR runbooks for an organization.
func (db *DB) GetDRRunbooksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DRRunbook, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, schedule_id, name, description, steps, contacts,
		       credentials_location, recovery_time_objective_minutes,
		       recovery_point_objective_minutes, status, created_at, updated_at
		FROM dr_runbooks
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list DR runbooks: %w", err)
	}
	defer rows.Close()

	var runbooks []*models.DRRunbook
	for rows.Next() {
		r, err := scanDRRunbook(rows)
		if err != nil {
			return nil, err
		}
		runbooks = append(runbooks, r)
	}

	return runbooks, nil
}

// GetDRRunbookByID returns a DR runbook by ID.
func (db *DB) GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, schedule_id, name, description, steps, contacts,
		       credentials_location, recovery_time_objective_minutes,
		       recovery_point_objective_minutes, status, created_at, updated_at
		FROM dr_runbooks
		WHERE id = $1
	`, id)

	return scanDRRunbook(row)
}

// GetDRRunbookByScheduleID returns a DR runbook for a schedule.
func (db *DB) GetDRRunbookByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.DRRunbook, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, schedule_id, name, description, steps, contacts,
		       credentials_location, recovery_time_objective_minutes,
		       recovery_point_objective_minutes, status, created_at, updated_at
		FROM dr_runbooks
		WHERE schedule_id = $1
	`, scheduleID)

	return scanDRRunbook(row)
}

// CreateDRRunbook creates a new DR runbook.
func (db *DB) CreateDRRunbook(ctx context.Context, runbook *models.DRRunbook) error {
	stepsBytes, err := runbook.StepsJSON()
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}

	contactsBytes, err := runbook.ContactsJSON()
	if err != nil {
		return fmt.Errorf("marshal contacts: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dr_runbooks (id, org_id, schedule_id, name, description, steps, contacts,
		                         credentials_location, recovery_time_objective_minutes,
		                         recovery_point_objective_minutes, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, runbook.ID, runbook.OrgID, runbook.ScheduleID, runbook.Name, runbook.Description,
		stepsBytes, contactsBytes, runbook.CredentialsLocation,
		runbook.RecoveryTimeObjectiveMins, runbook.RecoveryPointObjectiveMins,
		string(runbook.Status), runbook.CreatedAt, runbook.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create DR runbook: %w", err)
	}
	return nil
}

// UpdateDRRunbook updates an existing DR runbook.
func (db *DB) UpdateDRRunbook(ctx context.Context, runbook *models.DRRunbook) error {
	stepsBytes, err := runbook.StepsJSON()
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}

	contactsBytes, err := runbook.ContactsJSON()
	if err != nil {
		return fmt.Errorf("marshal contacts: %w", err)
	}

	runbook.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE dr_runbooks
		SET name = $2, description = $3, steps = $4, contacts = $5,
		    credentials_location = $6, recovery_time_objective_minutes = $7,
		    recovery_point_objective_minutes = $8, status = $9, updated_at = $10,
		    schedule_id = $11
		WHERE id = $1
	`, runbook.ID, runbook.Name, runbook.Description, stepsBytes, contactsBytes,
		runbook.CredentialsLocation, runbook.RecoveryTimeObjectiveMins,
		runbook.RecoveryPointObjectiveMins, string(runbook.Status), runbook.UpdatedAt,
		runbook.ScheduleID)
	if err != nil {
		return fmt.Errorf("update DR runbook: %w", err)
	}
	return nil
}

// DeleteDRRunbook deletes a DR runbook by ID.
func (db *DB) DeleteDRRunbook(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM dr_runbooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete DR runbook: %w", err)
	}
	return nil
}

// scanDRRunbook scans a DR runbook from a row.
func scanDRRunbook(row interface{ Scan(dest ...any) error }) (*models.DRRunbook, error) {
	var r models.DRRunbook
	var stepsBytes, contactsBytes []byte
	var statusStr string
	err := row.Scan(
		&r.ID, &r.OrgID, &r.ScheduleID, &r.Name, &r.Description,
		&stepsBytes, &contactsBytes, &r.CredentialsLocation,
		&r.RecoveryTimeObjectiveMins, &r.RecoveryPointObjectiveMins,
		&statusStr, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan DR runbook: %w", err)
	}

	r.Status = models.DRRunbookStatus(statusStr)
	if err := r.SetSteps(stepsBytes); err != nil {
		return nil, fmt.Errorf("parse steps: %w", err)
	}
	if err := r.SetContacts(contactsBytes); err != nil {
		return nil, fmt.Errorf("parse contacts: %w", err)
	}

	return &r, nil
}

// DR Test methods

// GetDRTestsByRunbookID returns all DR tests for a runbook.
func (db *DB) GetDRTestsByRunbookID(ctx context.Context, runbookID uuid.UUID) ([]*models.DRTest, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, runbook_id, schedule_id, agent_id, snapshot_id, status,
		       started_at, completed_at, restore_size_bytes, restore_duration_seconds,
		       verification_passed, notes, error_message, created_at
		FROM dr_tests
		WHERE runbook_id = $1
		ORDER BY created_at DESC
	`, runbookID)
	if err != nil {
		return nil, fmt.Errorf("list DR tests: %w", err)
	}
	defer rows.Close()

	return scanDRTests(rows)
}

// GetDRTestsByOrgID returns all DR tests for an organization.
func (db *DB) GetDRTestsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DRTest, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT t.id, t.runbook_id, t.schedule_id, t.agent_id, t.snapshot_id, t.status,
		       t.started_at, t.completed_at, t.restore_size_bytes, t.restore_duration_seconds,
		       t.verification_passed, t.notes, t.error_message, t.created_at
		FROM dr_tests t
		JOIN dr_runbooks r ON t.runbook_id = r.id
		WHERE r.org_id = $1
		ORDER BY t.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list DR tests by org: %w", err)
	}
	defer rows.Close()

	return scanDRTests(rows)
}

// GetDRTestByID returns a DR test by ID.
func (db *DB) GetDRTestByID(ctx context.Context, id uuid.UUID) (*models.DRTest, error) {
	var t models.DRTest
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, runbook_id, schedule_id, agent_id, snapshot_id, status,
		       started_at, completed_at, restore_size_bytes, restore_duration_seconds,
		       verification_passed, notes, error_message, created_at
		FROM dr_tests
		WHERE id = $1
	`, id).Scan(
		&t.ID, &t.RunbookID, &t.ScheduleID, &t.AgentID, &t.SnapshotID, &statusStr,
		&t.StartedAt, &t.CompletedAt, &t.RestoreSizeBytes, &t.RestoreDurationSeconds,
		&t.VerificationPassed, &t.Notes, &t.ErrorMessage, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get DR test: %w", err)
	}
	t.Status = models.DRTestStatus(statusStr)
	return &t, nil
}

// GetLatestDRTestByRunbookID returns the most recent DR test for a runbook.
func (db *DB) GetLatestDRTestByRunbookID(ctx context.Context, runbookID uuid.UUID) (*models.DRTest, error) {
	var t models.DRTest
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, runbook_id, schedule_id, agent_id, snapshot_id, status,
		       started_at, completed_at, restore_size_bytes, restore_duration_seconds,
		       verification_passed, notes, error_message, created_at
		FROM dr_tests
		WHERE runbook_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, runbookID).Scan(
		&t.ID, &t.RunbookID, &t.ScheduleID, &t.AgentID, &t.SnapshotID, &statusStr,
		&t.StartedAt, &t.CompletedAt, &t.RestoreSizeBytes, &t.RestoreDurationSeconds,
		&t.VerificationPassed, &t.Notes, &t.ErrorMessage, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest DR test: %w", err)
	}
	t.Status = models.DRTestStatus(statusStr)
	return &t, nil
}

// CreateDRTest creates a new DR test record.
func (db *DB) CreateDRTest(ctx context.Context, test *models.DRTest) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO dr_tests (id, runbook_id, schedule_id, agent_id, snapshot_id, status,
		                      started_at, completed_at, restore_size_bytes, restore_duration_seconds,
		                      verification_passed, notes, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, test.ID, test.RunbookID, test.ScheduleID, test.AgentID, test.SnapshotID,
		string(test.Status), test.StartedAt, test.CompletedAt, test.RestoreSizeBytes,
		test.RestoreDurationSeconds, test.VerificationPassed, test.Notes,
		test.ErrorMessage, test.CreatedAt)
	if err != nil {
		return fmt.Errorf("create DR test: %w", err)
	}
	return nil
}

// UpdateDRTest updates an existing DR test record.
func (db *DB) UpdateDRTest(ctx context.Context, test *models.DRTest) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE dr_tests
		SET status = $2, started_at = $3, completed_at = $4, snapshot_id = $5,
		    restore_size_bytes = $6, restore_duration_seconds = $7,
		    verification_passed = $8, notes = $9, error_message = $10
		WHERE id = $1
	`, test.ID, string(test.Status), test.StartedAt, test.CompletedAt, test.SnapshotID,
		test.RestoreSizeBytes, test.RestoreDurationSeconds, test.VerificationPassed,
		test.Notes, test.ErrorMessage)
	if err != nil {
		return fmt.Errorf("update DR test: %w", err)
	}
	return nil
}

// scanDRTests scans multiple DR test rows.
func scanDRTests(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.DRTest, error) {
	type scanner interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r := rows.(scanner)

	var tests []*models.DRTest
	for r.Next() {
		var t models.DRTest
		var statusStr string
		err := r.Scan(
			&t.ID, &t.RunbookID, &t.ScheduleID, &t.AgentID, &t.SnapshotID, &statusStr,
			&t.StartedAt, &t.CompletedAt, &t.RestoreSizeBytes, &t.RestoreDurationSeconds,
			&t.VerificationPassed, &t.Notes, &t.ErrorMessage, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan DR test: %w", err)
		}
		t.Status = models.DRTestStatus(statusStr)
		tests = append(tests, &t)
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate DR tests: %w", err)
	}

	return tests, nil
}

// DR Test Schedule methods

// GetDRTestSchedulesByRunbookID returns all test schedules for a runbook.
func (db *DB) GetDRTestSchedulesByRunbookID(ctx context.Context, runbookID uuid.UUID) ([]*models.DRTestSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, runbook_id, cron_expression, enabled, last_run_at, next_run_at, created_at, updated_at
		FROM dr_test_schedules
		WHERE runbook_id = $1
		ORDER BY created_at
	`, runbookID)
	if err != nil {
		return nil, fmt.Errorf("list DR test schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*models.DRTestSchedule
	for rows.Next() {
		var s models.DRTestSchedule
		err := rows.Scan(
			&s.ID, &s.RunbookID, &s.CronExpression, &s.Enabled,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan DR test schedule: %w", err)
		}
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

// GetEnabledDRTestSchedules returns all enabled DR test schedules.
func (db *DB) GetEnabledDRTestSchedules(ctx context.Context) ([]*models.DRTestSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, runbook_id, cron_expression, enabled, last_run_at, next_run_at, created_at, updated_at
		FROM dr_test_schedules
		WHERE enabled = true
		ORDER BY next_run_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled DR test schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*models.DRTestSchedule
	for rows.Next() {
		var s models.DRTestSchedule
		err := rows.Scan(
			&s.ID, &s.RunbookID, &s.CronExpression, &s.Enabled,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan DR test schedule: %w", err)
		}
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

// CreateDRTestSchedule creates a new DR test schedule.
func (db *DB) CreateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO dr_test_schedules (id, runbook_id, cron_expression, enabled, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, schedule.ID, schedule.RunbookID, schedule.CronExpression, schedule.Enabled,
		schedule.LastRunAt, schedule.NextRunAt, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create DR test schedule: %w", err)
	}
	return nil
}

// UpdateDRTestSchedule updates an existing DR test schedule.
func (db *DB) UpdateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error {
	schedule.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE dr_test_schedules
		SET cron_expression = $2, enabled = $3, last_run_at = $4, next_run_at = $5, updated_at = $6
		WHERE id = $1
	`, schedule.ID, schedule.CronExpression, schedule.Enabled, schedule.LastRunAt, schedule.NextRunAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update DR test schedule: %w", err)
	}
	return nil
}

// DeleteDRTestSchedule deletes a DR test schedule by ID.
func (db *DB) DeleteDRTestSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM dr_test_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete DR test schedule: %w", err)
	}
	return nil
}

// GetDRStatus returns the overall DR status for an organization.
func (db *DB) GetDRStatus(ctx context.Context, orgID uuid.UUID) (*models.DRStatus, error) {
	status := &models.DRStatus{}

	// Get runbook counts
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active')
		FROM dr_runbooks
		WHERE org_id = $1
	`, orgID).Scan(&status.TotalRunbooks, &status.ActiveRunbooks)
	if err != nil {
		return nil, fmt.Errorf("get runbook counts: %w", err)
	}

	// Get test statistics from last 30 days
	err = db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COALESCE(AVG(CASE WHEN verification_passed = true THEN 1.0 ELSE 0.0 END) * 100, 0)
		FROM dr_tests t
		JOIN dr_runbooks r ON t.runbook_id = r.id
		WHERE r.org_id = $1 AND t.created_at > NOW() - INTERVAL '30 days'
	`, orgID).Scan(&status.TestsLast30Days, &status.PassRate)
	if err != nil {
		return nil, fmt.Errorf("get test statistics: %w", err)
	}

	// Get last test date
	err = db.Pool.QueryRow(ctx, `
		SELECT MAX(t.completed_at)
		FROM dr_tests t
		JOIN dr_runbooks r ON t.runbook_id = r.id
		WHERE r.org_id = $1
	`, orgID).Scan(&status.LastTestAt)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, fmt.Errorf("get last test date: %w", err)
	}

	// Get next scheduled test
	err = db.Pool.QueryRow(ctx, `
		SELECT MIN(s.next_run_at)
		FROM dr_test_schedules s
		JOIN dr_runbooks r ON s.runbook_id = r.id
		WHERE r.org_id = $1 AND s.enabled = true AND s.next_run_at IS NOT NULL
	`, orgID).Scan(&status.NextTestAt)
	if err != nil && err.Error() != "no rows in result set" {
		return nil, fmt.Errorf("get next test date: %w", err)
	}

	return status, nil
}


// GetTagsByOrgID returns all tags for an organization.
func (db *DB) GetTagsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Tag, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, color, created_at, updated_at
		FROM tags
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var t models.Tag
		err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Color, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

// GetTagByID returns a tag by ID.
func (db *DB) GetTagByID(ctx context.Context, id uuid.UUID) (*models.Tag, error) {
	var t models.Tag
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, color, created_at, updated_at
		FROM tags
		WHERE id = $1
	`, id).Scan(&t.ID, &t.OrgID, &t.Name, &t.Color, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &t, nil
}

// GetTagByNameAndOrgID returns a tag by name and organization ID.
func (db *DB) GetTagByNameAndOrgID(ctx context.Context, name string, orgID uuid.UUID) (*models.Tag, error) {
	var t models.Tag
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, color, created_at, updated_at
		FROM tags
		WHERE name = $1 AND org_id = $2
	`, name, orgID).Scan(&t.ID, &t.OrgID, &t.Name, &t.Color, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get tag by name: %w", err)
	}
	return &t, nil
}

// CreateTag creates a new tag.
func (db *DB) CreateTag(ctx context.Context, tag *models.Tag) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO tags (id, org_id, name, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tag.ID, tag.OrgID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

// UpdateTag updates an existing tag.
func (db *DB) UpdateTag(ctx context.Context, tag *models.Tag) error {
	tag.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE tags
		SET name = $2, color = $3, updated_at = $4
		WHERE id = $1
	`, tag.ID, tag.Name, tag.Color, tag.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update tag: %w", err)
	}
	return nil
}

// DeleteTag deletes a tag by ID.
func (db *DB) DeleteTag(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

// Backup-Tag association methods

// GetTagsByBackupID returns all tags for a backup.
func (db *DB) GetTagsByBackupID(ctx context.Context, backupID uuid.UUID) ([]*models.Tag, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT t.id, t.org_id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN backup_tags bt ON t.id = bt.tag_id
		WHERE bt.backup_id = $1
		ORDER BY t.name
	`, backupID)
	if err != nil {
		return nil, fmt.Errorf("list tags for backup: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var t models.Tag
		err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Color, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

// GetBackupIDsByTagID returns all backup IDs that have a specific tag.
func (db *DB) GetBackupIDsByTagID(ctx context.Context, tagID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT backup_id FROM backup_tags WHERE tag_id = $1
	`, tagID)
	if err != nil {
		return nil, fmt.Errorf("list backup IDs for tag: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan backup ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup IDs: %w", err)
	}

	return ids, nil
}

// AssignTagToBackup assigns a tag to a backup.
func (db *DB) AssignTagToBackup(ctx context.Context, backupID, tagID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backup_tags (id, backup_id, tag_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (backup_id, tag_id) DO NOTHING
	`, uuid.New(), backupID, tagID, time.Now())
	if err != nil {
		return fmt.Errorf("assign tag to backup: %w", err)
	}
	return nil
}

// RemoveTagFromBackup removes a tag from a backup.
func (db *DB) RemoveTagFromBackup(ctx context.Context, backupID, tagID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM backup_tags WHERE backup_id = $1 AND tag_id = $2
	`, backupID, tagID)
	if err != nil {
		return fmt.Errorf("remove tag from backup: %w", err)
	}
	return nil
}

// SetBackupTags replaces all tags for a backup with the given tags.
func (db *DB) SetBackupTags(ctx context.Context, backupID uuid.UUID, tagIDs []uuid.UUID) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// Remove all existing tags
		_, err := tx.Exec(ctx, `DELETE FROM backup_tags WHERE backup_id = $1`, backupID)
		if err != nil {
			return fmt.Errorf("clear backup tags: %w", err)
		}

		// Add new tags
		for _, tagID := range tagIDs {
			_, err := tx.Exec(ctx, `
				INSERT INTO backup_tags (id, backup_id, tag_id, created_at)
				VALUES ($1, $2, $3, $4)
			`, uuid.New(), backupID, tagID, time.Now())
			if err != nil {
				return fmt.Errorf("assign tag to backup: %w", err)
			}
		}
		return nil
	})
}

// Snapshot-Tag association methods

// GetTagsBySnapshotID returns all tags for a snapshot.
func (db *DB) GetTagsBySnapshotID(ctx context.Context, snapshotID string) ([]*models.Tag, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT t.id, t.org_id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN snapshot_tags st ON t.id = st.tag_id
		WHERE st.snapshot_id = $1
		ORDER BY t.name
	`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("list tags for snapshot: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var t models.Tag
		err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Color, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

// AssignTagToSnapshot assigns a tag to a snapshot.
func (db *DB) AssignTagToSnapshot(ctx context.Context, snapshotID string, tagID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_tags (id, snapshot_id, tag_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (snapshot_id, tag_id) DO NOTHING
	`, uuid.New(), snapshotID, tagID, time.Now())
	if err != nil {
		return fmt.Errorf("assign tag to snapshot: %w", err)
	}
	return nil
}

// RemoveTagFromSnapshot removes a tag from a snapshot.
func (db *DB) RemoveTagFromSnapshot(ctx context.Context, snapshotID string, tagID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM snapshot_tags WHERE snapshot_id = $1 AND tag_id = $2
	`, snapshotID, tagID)
	if err != nil {
		return fmt.Errorf("remove tag from snapshot: %w", err)
	}
	return nil
}

// SetSnapshotTags replaces all tags for a snapshot with the given tags.
func (db *DB) SetSnapshotTags(ctx context.Context, snapshotID string, tagIDs []uuid.UUID) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// Remove all existing tags
		_, err := tx.Exec(ctx, `DELETE FROM snapshot_tags WHERE snapshot_id = $1`, snapshotID)
		if err != nil {
			return fmt.Errorf("clear snapshot tags: %w", err)
		}

		// Add new tags
		for _, tagID := range tagIDs {
			_, err := tx.Exec(ctx, `
				INSERT INTO snapshot_tags (id, snapshot_id, tag_id, created_at)
				VALUES ($1, $2, $3, $4)
			`, uuid.New(), snapshotID, tagID, time.Now())
			if err != nil {
				return fmt.Errorf("assign tag to snapshot: %w", err)
			}
		}
		return nil
	})
}

// GetBackupsByTagIDs returns all backups that have any of the specified tags.
func (db *DB) GetBackupsByTagIDs(ctx context.Context, tagIDs []uuid.UUID) ([]*models.Backup, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT b.id, b.schedule_id, b.agent_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error, b.created_at
		FROM backups b
		JOIN backup_tags bt ON b.id = bt.backup_id
		WHERE bt.tag_id = ANY($1)
		ORDER BY b.started_at DESC
	`, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("list backups by tags: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// Search methods

// SearchResult represents a single search result.
type SearchResult struct {
	Type        string    `json:"type"` // agent, backup, snapshot, schedule, repository
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SearchFilter contains filters for search queries.
type SearchFilter struct {
	Query    string      `json:"q"`
	Types    []string    `json:"types,omitempty"`     // agent, backup, snapshot, schedule, repository
	Status   string      `json:"status,omitempty"`    // filter by status
	TagIDs   []uuid.UUID `json:"tag_ids,omitempty"`   // filter by tags
	DateFrom *time.Time  `json:"date_from,omitempty"` // filter by date range
	DateTo   *time.Time  `json:"date_to,omitempty"`   // filter by date range
	SizeMin  *int64      `json:"size_min,omitempty"`  // filter by size
	SizeMax  *int64      `json:"size_max,omitempty"`  // filter by size
	Limit    int         `json:"limit,omitempty"`     // max results per type
}

// Search performs a global search across agents, backups, snapshots, schedules, and repositories.
func (db *DB) Search(ctx context.Context, orgID uuid.UUID, filter SearchFilter) ([]SearchResult, error) {
	var results []SearchResult
	query := "%" + filter.Query + "%"

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	// Determine which types to search
	searchAll := len(filter.Types) == 0
	typeSet := make(map[string]bool)
	for _, t := range filter.Types {
		typeSet[t] = true
	}

	// Search agents
	if searchAll || typeSet["agent"] {
		agentResults, err := db.searchAgents(ctx, orgID, query, filter)
		if err != nil {
			return nil, fmt.Errorf("search agents: %w", err)
		}
		results = append(results, agentResults...)
	}

	// Search backups
	if searchAll || typeSet["backup"] {
		backupResults, err := db.searchBackups(ctx, orgID, query, filter)
		if err != nil {
			return nil, fmt.Errorf("search backups: %w", err)
		}
		results = append(results, backupResults...)
	}

	// Search schedules
	if searchAll || typeSet["schedule"] {
		scheduleResults, err := db.searchSchedules(ctx, orgID, query, filter)
		if err != nil {
			return nil, fmt.Errorf("search schedules: %w", err)
		}
		results = append(results, scheduleResults...)
	}

	// Search repositories
	if searchAll || typeSet["repository"] {
		repoResults, err := db.searchRepositories(ctx, orgID, query, filter)
		if err != nil {
			return nil, fmt.Errorf("search repositories: %w", err)
		}
		results = append(results, repoResults...)
	}

	return results, nil
}

func (db *DB) searchAgents(ctx context.Context, orgID uuid.UUID, query string, filter SearchFilter) ([]SearchResult, error) {
	sqlQuery := `
		SELECT id, hostname, status, created_at
		FROM agents
		WHERE org_id = $1 AND hostname ILIKE $2
	`
	args := []any{orgID, query}

	if filter.Status != "" {
		sqlQuery += " AND status = $3"
		args = append(args, filter.Status)
	}

	sqlQuery += fmt.Sprintf(" ORDER BY hostname LIMIT %d", filter.Limit)

	rows, err := db.Pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id uuid.UUID
		var hostname, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &hostname, &status, &createdAt); err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			Type:      "agent",
			ID:        id.String(),
			Name:      hostname,
			Status:    status,
			CreatedAt: createdAt,
		})
	}
	return results, rows.Err()
}

func (db *DB) searchBackups(ctx context.Context, orgID uuid.UUID, query string, filter SearchFilter) ([]SearchResult, error) {
	sqlQuery := `
		SELECT b.id, b.snapshot_id, b.status, b.size_bytes, b.started_at
		FROM backups b
		JOIN agents a ON b.agent_id = a.id
		WHERE a.org_id = $1 AND (b.snapshot_id ILIKE $2 OR b.id::text ILIKE $2)
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.Status != "" {
		sqlQuery += fmt.Sprintf(" AND b.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	if filter.SizeMin != nil {
		sqlQuery += fmt.Sprintf(" AND b.size_bytes >= $%d", argNum)
		args = append(args, *filter.SizeMin)
		argNum++
	}

	if filter.SizeMax != nil {
		sqlQuery += fmt.Sprintf(" AND b.size_bytes <= $%d", argNum)
		args = append(args, *filter.SizeMax)
		argNum++
	}

	// Filter by tags
	if len(filter.TagIDs) > 0 {
		sqlQuery += fmt.Sprintf(" AND b.id IN (SELECT backup_id FROM backup_tags WHERE tag_id = ANY($%d))", argNum)
		args = append(args, filter.TagIDs)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY b.started_at DESC LIMIT %d", filter.Limit)

	rows, err := db.Pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id uuid.UUID
		var snapshotID, status string
		var sizeBytes *int64
		var startedAt time.Time
		if err := rows.Scan(&id, &snapshotID, &status, &sizeBytes, &startedAt); err != nil {
			return nil, err
		}
		desc := ""
		if sizeBytes != nil {
			desc = fmt.Sprintf("%d bytes", *sizeBytes)
		}
		name := snapshotID
		if name == "" {
			name = id.String()[:8]
		}
		results = append(results, SearchResult{
			Type:        "backup",
			ID:          id.String(),
			Name:        name,
			Description: desc,
			Status:      status,
			CreatedAt:   startedAt,
		})
	}
	return results, rows.Err()
}

func (db *DB) searchSchedules(ctx context.Context, orgID uuid.UUID, query string, filter SearchFilter) ([]SearchResult, error) {
	sqlQuery := `
		SELECT s.id, s.name, s.enabled, s.created_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.name ILIKE $2
	`
	args := []any{orgID, query}

	sqlQuery += fmt.Sprintf(" ORDER BY s.name LIMIT %d", filter.Limit)

	rows, err := db.Pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id uuid.UUID
		var name string
		var enabled bool
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &enabled, &createdAt); err != nil {
			return nil, err
		}
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		results = append(results, SearchResult{
			Type:      "schedule",
			ID:        id.String(),
			Name:      name,
			Status:    status,
			CreatedAt: createdAt,
		})
	}
	return results, rows.Err()
}

func (db *DB) searchRepositories(ctx context.Context, orgID uuid.UUID, query string, filter SearchFilter) ([]SearchResult, error) {
	sqlQuery := `
		SELECT id, name, type, created_at
		FROM repositories
		WHERE org_id = $1 AND name ILIKE $2
	`
	args := []any{orgID, query}

	sqlQuery += fmt.Sprintf(" ORDER BY name LIMIT %d", filter.Limit)

	rows, err := db.Pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id uuid.UUID
		var name, repoType string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &repoType, &createdAt); err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			Type:        "repository",
			ID:          id.String(),
			Name:        name,
			Description: repoType,
			CreatedAt:   createdAt,
		})
	}
	return results, rows.Err()
}

// Metrics methods

// GetBackupsByOrgIDSince returns backups for an organization since a given time.
func (db *DB) GetBackupsByOrgIDSince(ctx context.Context, orgID uuid.UUID, since time.Time) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2
		ORDER BY b.started_at DESC
	`, orgID, since)
	if err != nil {
		return nil, fmt.Errorf("get backups since: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupCountsByOrgID returns backup counts for an organization.
func (db *DB) GetBackupCountsByOrgID(ctx context.Context, orgID uuid.UUID) (total, running, failed24h int, err error) {
	// Total backups
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
	`, orgID).Scan(&total)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get total backups: %w", err)
	}

	// Running backups
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.status = 'running'
	`, orgID).Scan(&running)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get running backups: %w", err)
	}

	// Failed in last 24 hours
	since24h := time.Now().Add(-24 * time.Hour)
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.status = 'failed' AND b.started_at >= $2
	`, orgID, since24h).Scan(&failed24h)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get failed backups: %w", err)
	}

	return total, running, failed24h, nil
}

// CreateMetricsHistory creates a new metrics history record.
func (db *DB) CreateMetricsHistory(ctx context.Context, metrics *models.MetricsHistory) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO metrics_history (
			id, org_id, backup_count, backup_success_count, backup_failed_count,
			backup_total_size, backup_total_duration_ms, agent_total_count,
			agent_online_count, agent_offline_count, storage_used_bytes,
			storage_raw_bytes, storage_space_saved, repository_count,
			total_snapshots, collected_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, metrics.ID, metrics.OrgID, metrics.BackupCount, metrics.BackupSuccessCount,
		metrics.BackupFailedCount, metrics.BackupTotalSize, metrics.BackupTotalDuration,
		metrics.AgentTotalCount, metrics.AgentOnlineCount, metrics.AgentOfflineCount,
		metrics.StorageUsedBytes, metrics.StorageRawBytes, metrics.StorageSpaceSaved,
		metrics.RepositoryCount, metrics.TotalSnapshots, metrics.CollectedAt, metrics.CreatedAt)
	if err != nil {
		return fmt.Errorf("create metrics history: %w", err)
	}
	return nil
}

// GetDashboardStats returns aggregated dashboard statistics for an organization.
func (db *DB) GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Get agent counts
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active') AS online,
			COUNT(*) FILTER (WHERE status = 'offline') AS offline
		FROM agents
		WHERE org_id = $1
	`, orgID).Scan(&stats.AgentTotal, &stats.AgentOnline, &stats.AgentOffline)
	if err != nil {
		return nil, fmt.Errorf("get agent counts: %w", err)
	}

	// Get backup counts
	since24h := time.Now().Add(-24 * time.Hour)
	err = db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE b.status = 'running') AS running,
			COUNT(*) FILTER (WHERE b.status = 'failed' AND b.started_at >= $2) AS failed_24h
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
	`, orgID, since24h).Scan(&stats.BackupTotal, &stats.BackupRunning, &stats.BackupFailed24h)
	if err != nil {
		return nil, fmt.Errorf("get backup counts: %w", err)
	}

	// Get repository count
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM repositories WHERE org_id = $1
	`, orgID).Scan(&stats.RepositoryCount)
	if err != nil {
		return nil, fmt.Errorf("get repository count: %w", err)
	}

	// Get schedule counts
	err = db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE enabled = true) AS enabled
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
	`, orgID).Scan(&stats.ScheduleCount, &stats.ScheduleEnabled)
	if err != nil {
		return nil, fmt.Errorf("get schedule counts: %w", err)
	}

	// Get storage stats summary
	summary, err := db.GetStorageStatsSummary(ctx, orgID)
	if err == nil && summary != nil {
		stats.TotalRawSize = summary.TotalRawSize
		stats.TotalBackupSize = summary.TotalRestoreSize
		stats.TotalSpaceSaved = summary.TotalSpaceSaved
		stats.AvgDedupRatio = summary.AvgDedupRatio
	}

	return stats, nil
}

// GetBackupSuccessRates returns backup success rates for 7-day and 30-day periods.
func (db *DB) GetBackupSuccessRates(ctx context.Context, orgID uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error) {
	rate7d := &models.BackupSuccessRate{Period: "7d"}
	rate30d := &models.BackupSuccessRate{Period: "30d"}

	// 7-day success rate
	since7d := time.Now().Add(-7 * 24 * time.Hour)
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE b.status = 'completed') AS successful,
			COUNT(*) FILTER (WHERE b.status = 'failed') AS failed
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2
	`, orgID, since7d).Scan(&rate7d.Total, &rate7d.Successful, &rate7d.Failed)
	if err != nil {
		return nil, nil, fmt.Errorf("get 7d success rate: %w", err)
	}
	if rate7d.Total > 0 {
		rate7d.SuccessPercent = float64(rate7d.Successful) / float64(rate7d.Total) * 100
	}

	// 30-day success rate
	since30d := time.Now().Add(-30 * 24 * time.Hour)
	err = db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE b.status = 'completed') AS successful,
			COUNT(*) FILTER (WHERE b.status = 'failed') AS failed
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2
	`, orgID, since30d).Scan(&rate30d.Total, &rate30d.Successful, &rate30d.Failed)
	if err != nil {
		return nil, nil, fmt.Errorf("get 30d success rate: %w", err)
	}
	if rate30d.Total > 0 {
		rate30d.SuccessPercent = float64(rate30d.Successful) / float64(rate30d.Total) * 100
	}

	return rate7d, rate30d, nil
}

// GetStorageGrowthTrend returns storage growth over time for an organization.
func (db *DB) GetStorageGrowthTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthTrend, error) {
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := db.Pool.Query(ctx, `
		SELECT
			DATE(ss.collected_at) AS date,
			SUM(ss.total_size) AS total_size,
			SUM(ss.raw_data_size) AS raw_size,
			SUM(ss.snapshot_count) AS snapshot_count
		FROM storage_stats ss
		JOIN repositories r ON ss.repository_id = r.id
		WHERE r.org_id = $1 AND ss.collected_at >= $2
		GROUP BY DATE(ss.collected_at)
		ORDER BY date ASC
	`, orgID, since)
	if err != nil {
		return nil, fmt.Errorf("get storage growth trend: %w", err)
	}
	defer rows.Close()

	var trends []*models.StorageGrowthTrend
	for rows.Next() {
		var t models.StorageGrowthTrend
		err := rows.Scan(&t.Date, &t.TotalSize, &t.RawSize, &t.SnapshotCount)
		if err != nil {
			return nil, fmt.Errorf("scan storage growth: %w", err)
		}
		trends = append(trends, &t)
	}

	return trends, nil
}

// GetBackupDurationTrend returns backup duration trends over time.
func (db *DB) GetBackupDurationTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.BackupDurationTrend, error) {
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := db.Pool.Query(ctx, `
		SELECT
			DATE(b.started_at) AS date,
			AVG(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS avg_duration_ms,
			MAX(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS max_duration_ms,
			MIN(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS min_duration_ms,
			COUNT(*) AS backup_count
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
		  AND b.started_at >= $2
		  AND b.completed_at IS NOT NULL
		  AND b.status = 'completed'
		GROUP BY DATE(b.started_at)
		ORDER BY date ASC
	`, orgID, since)
	if err != nil {
		return nil, fmt.Errorf("get backup duration trend: %w", err)
	}
	defer rows.Close()

	var trends []*models.BackupDurationTrend
	for rows.Next() {
		var t models.BackupDurationTrend
		err := rows.Scan(&t.Date, &t.AvgDurationMs, &t.MaxDurationMs, &t.MinDurationMs, &t.BackupCount)
		if err != nil {
			return nil, fmt.Errorf("scan backup duration: %w", err)
		}
		trends = append(trends, &t)
	}

	return trends, nil
}

// GetDailyBackupStats returns daily backup statistics.
func (db *DB) GetDailyBackupStats(ctx context.Context, orgID uuid.UUID, days int) ([]*models.DailyBackupStats, error) {
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := db.Pool.Query(ctx, `
		SELECT
			DATE(b.started_at) AS date,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE b.status = 'completed') AS successful,
			COUNT(*) FILTER (WHERE b.status = 'failed') AS failed,
			COALESCE(SUM(b.size_bytes) FILTER (WHERE b.status = 'completed'), 0) AS total_size
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2
		GROUP BY DATE(b.started_at)
		ORDER BY date ASC
	`, orgID, since)
	if err != nil {
		return nil, fmt.Errorf("get daily backup stats: %w", err)
	}
	defer rows.Close()

	var stats []*models.DailyBackupStats
	for rows.Next() {
		var s models.DailyBackupStats
		err := rows.Scan(&s.Date, &s.Total, &s.Successful, &s.Failed, &s.TotalSize)
		if err != nil {
			return nil, fmt.Errorf("scan daily backup stats: %w", err)
		}
		stats = append(stats, &s)
	}

	return stats, nil
}

// Report Schedules

// GetEnabledReportSchedules returns all enabled report schedules.
func (db *DB) GetEnabledReportSchedules(ctx context.Context) ([]*models.ReportSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, frequency, recipients, channel_id, timezone,
		       enabled, last_sent_at, created_at, updated_at
		FROM report_schedules
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("get enabled report schedules: %w", err)
	}
	defer rows.Close()
	return scanReportSchedules(rows)
}

// GetReportSchedulesByOrgID returns all report schedules for an organization.
func (db *DB) GetReportSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ReportSchedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, frequency, recipients, channel_id, timezone,
		       enabled, last_sent_at, created_at, updated_at
		FROM report_schedules
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get report schedules: %w", err)
	}
	defer rows.Close()
	return scanReportSchedules(rows)
}

// GetReportScheduleByID returns a report schedule by ID.
func (db *DB) GetReportScheduleByID(ctx context.Context, id uuid.UUID) (*models.ReportSchedule, error) {
	var s models.ReportSchedule
	var frequencyStr string
	var recipientsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, frequency, recipients, channel_id, timezone,
		       enabled, last_sent_at, created_at, updated_at
		FROM report_schedules
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.OrgID, &s.Name, &frequencyStr, &recipientsBytes, &s.ChannelID,
		&s.Timezone, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get report schedule: %w", err)
	}
	s.Frequency = models.ReportFrequency(frequencyStr)
	if err := parseStringSlice(recipientsBytes, &s.Recipients); err != nil {
		db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse recipients")
	}
	return &s, nil
}

// CreateReportSchedule creates a new report schedule.
func (db *DB) CreateReportSchedule(ctx context.Context, schedule *models.ReportSchedule) error {
	recipientsBytes, err := toJSONBytes(schedule.Recipients)
	if err != nil {
		return fmt.Errorf("marshal recipients: %w", err)
	}
	if recipientsBytes == nil {
		recipientsBytes = []byte("[]")
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO report_schedules (id, org_id, name, frequency, recipients, channel_id,
		                               timezone, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, schedule.ID, schedule.OrgID, schedule.Name, string(schedule.Frequency),
		recipientsBytes, schedule.ChannelID, schedule.Timezone, schedule.Enabled,
		schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create report schedule: %w", err)
	}
	return nil
}

// UpdateReportSchedule updates an existing report schedule.
func (db *DB) UpdateReportSchedule(ctx context.Context, schedule *models.ReportSchedule) error {
	recipientsBytes, err := toJSONBytes(schedule.Recipients)
	if err != nil {
		return fmt.Errorf("marshal recipients: %w", err)
	}
	if recipientsBytes == nil {
		recipientsBytes = []byte("[]")
	}

	schedule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE report_schedules
		SET name = $2, frequency = $3, recipients = $4, channel_id = $5,
		    timezone = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, schedule.ID, schedule.Name, string(schedule.Frequency), recipientsBytes,
		schedule.ChannelID, schedule.Timezone, schedule.Enabled, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update report schedule: %w", err)
	}
	return nil
}

// UpdateReportScheduleLastSent updates the last_sent_at timestamp.
func (db *DB) UpdateReportScheduleLastSent(ctx context.Context, id uuid.UUID, lastSentAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE report_schedules
		SET last_sent_at = $2, updated_at = $3
		WHERE id = $1
	`, id, lastSentAt, time.Now())
	if err != nil {
		return fmt.Errorf("update report schedule last sent: %w", err)
	}
	return nil
}

// DeleteReportSchedule deletes a report schedule by ID.
func (db *DB) DeleteReportSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM report_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete report schedule: %w", err)
	}
	return nil
}

// scanReportSchedules scans multiple report schedule rows.
func scanReportSchedules(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]*models.ReportSchedule, error) {
	var schedules []*models.ReportSchedule
	for rows.Next() {
		var s models.ReportSchedule
		var frequencyStr string
		var recipientsBytes []byte
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Name, &frequencyStr, &recipientsBytes, &s.ChannelID,
			&s.Timezone, &s.Enabled, &s.LastSentAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan report schedule: %w", err)
		}
		s.Frequency = models.ReportFrequency(frequencyStr)
		_ = json.Unmarshal(recipientsBytes, &s.Recipients)
		schedules = append(schedules, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report schedules: %w", err)
	}
	return schedules, nil
}

// Report History

// CreateReportHistory creates a new report history entry.
func (db *DB) CreateReportHistory(ctx context.Context, history *models.ReportHistory) error {
	recipientsBytes, err := toJSONBytes(history.Recipients)
	if err != nil {
		return fmt.Errorf("marshal recipients: %w", err)
	}
	if recipientsBytes == nil {
		recipientsBytes = []byte("[]")
	}

	var reportDataBytes []byte
	if history.ReportData != nil {
		reportDataBytes, err = json.Marshal(history.ReportData)
		if err != nil {
			return fmt.Errorf("marshal report data: %w", err)
		}
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO report_history (id, org_id, schedule_id, report_type, period_start,
		                            period_end, recipients, status, error_message,
		                            report_data, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, history.ID, history.OrgID, history.ScheduleID, history.ReportType,
		history.PeriodStart, history.PeriodEnd, recipientsBytes, string(history.Status),
		history.ErrorMessage, reportDataBytes, history.SentAt, history.CreatedAt)
	if err != nil {
		return fmt.Errorf("create report history: %w", err)
	}
	return nil
}

// GetReportHistoryByOrgID returns report history for an organization.
func (db *DB) GetReportHistoryByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ReportHistory, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, schedule_id, report_type, period_start, period_end,
		       recipients, status, error_message, report_data, sent_at, created_at
		FROM report_history
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("get report history: %w", err)
	}
	defer rows.Close()
	return scanReportHistory(rows)
}

// GetReportHistoryByID returns a report history entry by ID.
func (db *DB) GetReportHistoryByID(ctx context.Context, id uuid.UUID) (*models.ReportHistory, error) {
	var h models.ReportHistory
	var statusStr string
	var recipientsBytes, reportDataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, schedule_id, report_type, period_start, period_end,
		       recipients, status, error_message, report_data, sent_at, created_at
		FROM report_history
		WHERE id = $1
	`, id).Scan(
		&h.ID, &h.OrgID, &h.ScheduleID, &h.ReportType, &h.PeriodStart, &h.PeriodEnd,
		&recipientsBytes, &statusStr, &h.ErrorMessage, &reportDataBytes, &h.SentAt, &h.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get report history: %w", err)
	}
	h.Status = models.ReportStatus(statusStr)
	_ = json.Unmarshal(recipientsBytes, &h.Recipients)
	if len(reportDataBytes) > 0 {
		h.ReportData = &models.ReportData{}
		_ = json.Unmarshal(reportDataBytes, h.ReportData)
	}
	return &h, nil
}

// scanReportHistory scans multiple report history rows.
func scanReportHistory(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.ReportHistory, error) {
	var history []*models.ReportHistory
	for rows.Next() {
		var h models.ReportHistory
		var statusStr string
		var recipientsBytes, reportDataBytes []byte
		err := rows.Scan(
			&h.ID, &h.OrgID, &h.ScheduleID, &h.ReportType, &h.PeriodStart, &h.PeriodEnd,
			&recipientsBytes, &statusStr, &h.ErrorMessage, &reportDataBytes, &h.SentAt, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan report history: %w", err)
		}
		h.Status = models.ReportStatus(statusStr)
		_ = json.Unmarshal(recipientsBytes, &h.Recipients)
		if len(reportDataBytes) > 0 {
			h.ReportData = &models.ReportData{}
			_ = json.Unmarshal(reportDataBytes, h.ReportData)
		}
		history = append(history, &h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report history: %w", err)
	}
	return history, nil
}

// Report Data Queries

// GetBackupsByOrgIDAndDateRange returns backups for an org within a date range.
func (db *DB) GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.snapshot_id, b.started_at,
		       b.completed_at, b.status, b.size_bytes, b.files_new,
		       b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2 AND b.started_at <= $3
		ORDER BY b.started_at DESC
	`, orgID, start, end)
	if err != nil {
		return nil, fmt.Errorf("get backups by date range: %w", err)
	}
	defer rows.Close()
	return scanBackups(rows)
}

// GetAlertsByOrgIDAndDateRange returns alerts for an org within a date range.
func (db *DB) GetAlertsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Alert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, rule_id, type, severity, status, title, message,
		       resource_type, resource_id, acknowledged_by, acknowledged_at,
		       resolved_at, metadata, created_at, updated_at
		FROM alerts
		WHERE org_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`, orgID, start, end)
	if err != nil {
		return nil, fmt.Errorf("get alerts by date range: %w", err)
	}
	defer rows.Close()
	return db.scanAlerts(rows)
}

// GetEnabledSchedulesByOrgID returns all enabled backup schedules for an org.
func (db *DB) GetEnabledSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.enabled,
		       s.created_at, s.updated_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.enabled = true
		ORDER BY s.name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get enabled schedules by org: %w", err)
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// Agent Group methods

// GetAgentGroupsByOrgID returns all agent groups for an organization with agent counts.
func (db *DB) GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT g.id, g.org_id, g.name, g.description, g.color, g.created_at, g.updated_at,
		       COUNT(m.id) as agent_count
		FROM agent_groups g
		LEFT JOIN agent_group_members m ON g.id = m.group_id
		WHERE g.org_id = $1
		GROUP BY g.id
		ORDER BY g.name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.AgentGroup
	for rows.Next() {
		var g models.AgentGroup
		var description, color *string
		err := rows.Scan(
			&g.ID, &g.OrgID, &g.Name, &description, &color,
			&g.CreatedAt, &g.UpdatedAt, &g.AgentCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent group: %w", err)
		}
		if description != nil {
			g.Description = *description
		}
		if color != nil {
			g.Color = *color
		}
		groups = append(groups, &g)
	}

	return groups, nil
}

// GetAgentGroupByID returns an agent group by ID.
func (db *DB) GetAgentGroupByID(ctx context.Context, id uuid.UUID) (*models.AgentGroup, error) {
	var g models.AgentGroup
	var description, color *string
	err := db.Pool.QueryRow(ctx, `
		SELECT g.id, g.org_id, g.name, g.description, g.color, g.created_at, g.updated_at,
		       COUNT(m.id) as agent_count
		FROM agent_groups g
		LEFT JOIN agent_group_members m ON g.id = m.group_id
		WHERE g.id = $1
		GROUP BY g.id
	`, id).Scan(
		&g.ID, &g.OrgID, &g.Name, &description, &color,
		&g.CreatedAt, &g.UpdatedAt, &g.AgentCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}
	if description != nil {
		g.Description = *description
	}
	if color != nil {
		g.Color = *color
	}
	return &g, nil
}

// CreateAgentGroup creates a new agent group.
func (db *DB) CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_groups (id, org_id, name, description, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, group.ID, group.OrgID, group.Name, group.Description, group.Color,
		group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent group: %w", err)
	}
	return nil
}

// UpdateAgentGroup updates an existing agent group.
func (db *DB) UpdateAgentGroup(ctx context.Context, group *models.AgentGroup) error {
	group.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE agent_groups
		SET name = $2, description = $3, color = $4, updated_at = $5
		WHERE id = $1
	`, group.ID, group.Name, group.Description, group.Color, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update agent group: %w", err)
	}
	return nil
}

// DeleteAgentGroup deletes an agent group.
func (db *DB) DeleteAgentGroup(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM agent_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete agent group: %w", err)
	}
	return nil
}

// GetAgentGroupMembers returns all agents in a group.
func (db *DB) GetAgentGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*models.Agent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT a.id, a.org_id, a.hostname, a.api_key_hash, a.os_info, a.last_seen, a.status, a.created_at, a.updated_at
		FROM agents a
		INNER JOIN agent_group_members m ON a.id = m.agent_id
		WHERE m.group_id = $1
		ORDER BY a.hostname
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list agent group members: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		var a models.Agent
		var osInfoBytes []byte
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes,
			&a.LastSeen, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		if osInfoBytes != nil {
			if err := json.Unmarshal(osInfoBytes, &a.OSInfo); err != nil {
				db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
			}
		}
		agents = append(agents, &a)
	}

	return agents, nil
}

// AddAgentToGroup adds an agent to a group.
func (db *DB) AddAgentToGroup(ctx context.Context, groupID, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_group_members (id, group_id, agent_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (group_id, agent_id) DO NOTHING
	`, uuid.New(), groupID, agentID, time.Now())
	if err != nil {
		return fmt.Errorf("add agent to group: %w", err)
	}
	return nil
}

// RemoveAgentFromGroup removes an agent from a group.
func (db *DB) RemoveAgentFromGroup(ctx context.Context, groupID, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_group_members
		WHERE group_id = $1 AND agent_id = $2
	`, groupID, agentID)
	if err != nil {
		return fmt.Errorf("remove agent from group: %w", err)
	}
	return nil
}

// GetGroupsByAgentID returns all groups an agent belongs to.
func (db *DB) GetGroupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.AgentGroup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT g.id, g.org_id, g.name, g.description, g.color, g.created_at, g.updated_at
		FROM agent_groups g
		INNER JOIN agent_group_members m ON g.id = m.group_id
		WHERE m.agent_id = $1
		ORDER BY g.name
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list groups by agent: %w", err)
	}
	defer rows.Close()

	var groups []*models.AgentGroup
	for rows.Next() {
		var g models.AgentGroup
		var description, color *string
		err := rows.Scan(
			&g.ID, &g.OrgID, &g.Name, &description, &color,
			&g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent group: %w", err)
		}
		if description != nil {
			g.Description = *description
		}
		if color != nil {
			g.Color = *color
		}
		groups = append(groups, &g)
	}

	return groups, nil
}

// GetAgentsWithGroupsByOrgID returns all agents with their groups for an organization.
func (db *DB) GetAgentsWithGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentWithGroups, error) {
	// First get all agents
	agents, err := db.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Build result with groups
	result := make([]*models.AgentWithGroups, len(agents))
	for i, agent := range agents {
		groupPtrs, err := db.GetGroupsByAgentID(ctx, agent.ID)
		if err != nil {
			return nil, fmt.Errorf("get groups for agent %s: %w", agent.ID, err)
		}
		// Convert []*AgentGroup to []AgentGroup
		groups := make([]models.AgentGroup, len(groupPtrs))
		for j, g := range groupPtrs {
			groups[j] = *g
		}
		result[i] = &models.AgentWithGroups{
			Agent:  *agent,
			Groups: groups,
		}
	}

	return result, nil
}

// GetAgentsByGroupID returns all agent IDs in a group.
func (db *DB) GetAgentsByGroupID(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT agent_id FROM agent_group_members WHERE group_id = $1
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list agents by group: %w", err)
	}
	defer rows.Close()

	var agentIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan agent id: %w", err)
		}
		agentIDs = append(agentIDs, id)
	}

	return agentIDs, nil
}

// GetSchedulesByAgentGroupID returns all schedules for an agent group.
func (db *DB) GetSchedulesByAgentGroupID(ctx context.Context, agentGroupID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		FROM schedules
		WHERE agent_group_id = $1
		ORDER BY name
	`, agentGroupID)
	if err != nil {
		return nil, fmt.Errorf("list schedules by group: %w", err)
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}
// Onboarding methods

// GetOnboardingProgress returns the onboarding progress for an organization.
func (db *DB) GetOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	var p models.OnboardingProgress
	var currentStepStr string
	var completedStepsArr []string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, current_step, completed_steps, skipped, completed_at, created_at, updated_at
		FROM onboarding_progress
		WHERE org_id = $1
	`, orgID).Scan(
		&p.ID, &p.OrgID, &currentStepStr, &completedStepsArr,
		&p.Skipped, &p.CompletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get onboarding progress: %w", err)
	}
	p.CurrentStep = models.OnboardingStep(currentStepStr)
	p.CompletedSteps = make([]models.OnboardingStep, len(completedStepsArr))
	for i, s := range completedStepsArr {
		p.CompletedSteps[i] = models.OnboardingStep(s)
	}
	return &p, nil
}

// GetOrCreateOnboardingProgress returns existing progress or creates new progress for an organization.
func (db *DB) GetOrCreateOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	// Try to get existing progress
	p, err := db.GetOnboardingProgress(ctx, orgID)
	if err == nil {
		return p, nil
	}

	// Create new progress
	progress := models.NewOnboardingProgress(orgID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO onboarding_progress (id, org_id, current_step, completed_steps, skipped, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, progress.ID, progress.OrgID, string(progress.CurrentStep), []string{}, progress.Skipped, progress.CompletedAt, progress.CreatedAt, progress.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create onboarding progress: %w", err)
	}

	db.logger.Info().Str("org_id", orgID.String()).Msg("created onboarding progress")
	return progress, nil
}

// UpdateOnboardingProgress updates the onboarding progress for an organization.
func (db *DB) UpdateOnboardingProgress(ctx context.Context, progress *models.OnboardingProgress) error {
	completedStepsArr := make([]string, len(progress.CompletedSteps))
	for i, s := range progress.CompletedSteps {
		completedStepsArr[i] = string(s)
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE onboarding_progress
		SET current_step = $2, completed_steps = $3, skipped = $4, completed_at = $5, updated_at = $6
		WHERE org_id = $1
	`, progress.OrgID, string(progress.CurrentStep), completedStepsArr, progress.Skipped, progress.CompletedAt, time.Now())
	if err != nil {
		return fmt.Errorf("update onboarding progress: %w", err)
	}
	return nil
}

// SkipOnboarding marks the onboarding as skipped for an organization.
func (db *DB) SkipOnboarding(ctx context.Context, orgID uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE onboarding_progress
		SET skipped = true, completed_at = $2, updated_at = $2
		WHERE org_id = $1
	`, orgID, now)
	if err != nil {
		return fmt.Errorf("skip onboarding: %w", err)
	}
	return nil
}
