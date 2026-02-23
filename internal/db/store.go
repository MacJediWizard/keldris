package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// scanner is an interface for row scanning (pgx.Rows, etc.)
type scanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

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
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		FROM users
		WHERE oidc_subject = $1
	`, subject).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
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
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
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
		INSERT INTO users (id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.OrgID, user.OIDCSubject, user.Email, user.Name, string(user.Role), user.IsSuperuser, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// ListUsers returns all users for an organization with their membership roles.
func (db *DB) ListUsers(ctx context.Context, orgID uuid.UUID) ([]*models.User, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT u.id, u.org_id, u.oidc_subject, u.email, u.name, u.role,
		       COALESCE(m.role, u.role) AS effective_role,
		       u.created_at, u.updated_at
		FROM users u
		LEFT JOIN org_memberships m ON m.user_id = u.id AND m.org_id = u.org_id
		WHERE u.org_id = $1
		ORDER BY u.name, u.email
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		var roleStr string
		var effectiveRoleStr string
		if err := rows.Scan(&u.ID, &u.OrgID, &u.OIDCSubject, &u.Email, &u.Name, &roleStr, &effectiveRoleStr, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.Role = models.UserRole(effectiveRoleStr)
		users = append(users, &u)
	}
	return users, nil
}

// UpdateUser updates a user's name, email, and role.
func (db *DB) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET name = $2, email = $3, role = $4, updated_at = $5
		WHERE id = $1
	`, user.ID, user.Name, user.Email, string(user.Role), user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Also update the membership role if one exists
	_, err = db.Pool.Exec(ctx, `
		UPDATE org_memberships
		SET role = $3, updated_at = $4
		WHERE user_id = $1 AND org_id = $2
	`, user.ID, user.OrgID, string(user.Role), user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update user membership: %w", err)
	}

	return nil
}

// DeleteUser deletes a user by ID. Returns an error if the user is the last owner of their organization.
func (db *DB) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Check if the user is the last owner of any organization
	var ownerCount int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM org_memberships
		WHERE org_id = (SELECT org_id FROM org_memberships WHERE user_id = $1 AND role = 'owner' LIMIT 1)
		  AND role = 'owner'
	`, id).Scan(&ownerCount)
	if err != nil {
		return fmt.Errorf("check owner count: %w", err)
	}

	// If user is an owner and is the only one, block deletion
	var isOwner bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM org_memberships WHERE user_id = $1 AND role = 'owner')
	`, id).Scan(&isOwner)
	if err != nil {
		return fmt.Errorf("check if user is owner: %w", err)
	}

	if isOwner && ownerCount <= 1 {
		return fmt.Errorf("cannot delete user: last owner of organization")
	}

	_, err = db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// Agent methods

// GetAgentsByOrgID returns all agents for an organization.
func (db *DB) GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at,
		       debug_mode, debug_mode_expires_at, debug_mode_enabled_at, debug_mode_enabled_by,
		       metadata, created_at, updated_at
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
		var metadataBytes []byte
		var statusStr string
		var healthStatusStr *string
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
			&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
			&a.HealthCheckedAt,
			&a.DebugMode, &a.DebugModeExpiresAt, &a.DebugModeEnabledAt, &a.DebugModeEnabledBy,
			&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
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
		if err := a.SetMetadata(metadataBytes); err != nil {
			db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse metadata")
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
	var metadataBytes []byte
	var statusStr string
	var healthStatusStr *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at,
		       debug_mode, debug_mode_expires_at, debug_mode_enabled_at, debug_mode_enabled_by,
		       metadata, created_at, updated_at
		FROM agents
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
		&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
		&a.HealthCheckedAt,
		&a.DebugMode, &a.DebugModeExpiresAt, &a.DebugModeEnabledAt, &a.DebugModeEnabledBy,
		&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
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
	if err := a.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse metadata")
	}
	return &a, nil
}

// GetAgentByAPIKeyHash returns an agent by API key hash.
func (db *DB) GetAgentByAPIKeyHash(ctx context.Context, hash string) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
	var networkMountsBytes []byte
	var healthMetricsBytes []byte
	var metadataBytes []byte
	var statusStr string
	var healthStatusStr *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, network_mounts, last_seen, status,
		       health_status, health_metrics, health_checked_at,
		       debug_mode, debug_mode_expires_at, debug_mode_enabled_at, debug_mode_enabled_by,
		       metadata, created_at, updated_at
		FROM agents
		WHERE api_key_hash = $1
	`, hash).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
		&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
		&a.HealthCheckedAt,
		&a.DebugMode, &a.DebugModeExpiresAt, &a.DebugModeEnabledAt, &a.DebugModeEnabledBy,
		&metadataBytes, &a.CreatedAt, &a.UpdatedAt,
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
	if err := a.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse metadata")
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

	metadataBytes, err := agent.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	agent.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE agents
		SET hostname = $2, os_info = $3, network_mounts = $4, last_seen = $5, status = $6, updated_at = $7,
		    health_status = $8, health_metrics = $9, health_checked_at = $10,
		    debug_mode = $11, debug_mode_expires_at = $12, debug_mode_enabled_at = $13,
		    debug_mode_enabled_by = $14, metadata = $15
		WHERE id = $1
	`, agent.ID, agent.Hostname, osInfoBytes, networkMountsBytes, agent.LastSeen, string(agent.Status), agent.UpdatedAt,
		string(agent.HealthStatus), healthMetricsBytes, agent.HealthCheckedAt,
		agent.DebugMode, agent.DebugModeExpiresAt, agent.DebugModeEnabledAt,
		agent.DebugModeEnabledBy, metadataBytes)
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

// SetAgentDebugMode enables or disables debug mode on an agent.
func (db *DB) SetAgentDebugMode(ctx context.Context, id uuid.UUID, enabled bool, expiresAt *time.Time, enabledBy *uuid.UUID) error {
	now := time.Now()
	var enabledAt *time.Time
	if enabled {
		enabledAt = &now
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET debug_mode = $2,
		    debug_mode_expires_at = $3,
		    debug_mode_enabled_at = $4,
		    debug_mode_enabled_by = $5,
		    updated_at = $6
		WHERE id = $1
	`, id, enabled, expiresAt, enabledAt, enabledBy, now)
	if err != nil {
		return fmt.Errorf("set agent debug mode: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// DisableExpiredDebugModes disables debug mode on agents where the expiration has passed.
func (db *DB) DisableExpiredDebugModes(ctx context.Context) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET debug_mode = false,
		    debug_mode_expires_at = NULL,
		    debug_mode_enabled_at = NULL,
		    debug_mode_enabled_by = NULL,
		    updated_at = $1
		WHERE debug_mode = true
		  AND debug_mode_expires_at IS NOT NULL
		  AND debug_mode_expires_at < $1
	`, time.Now())
	if err != nil {
		return 0, fmt.Errorf("disable expired debug modes: %w", err)
	}
	return result.RowsAffected(), nil
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
		WHERE agent_id = $1 AND deleted_at IS NULL
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
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
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
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
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

	// Serialize backup type specific options
	dockerOptionsBytes, err := schedule.DockerOptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal docker options: %w", err)
	}

	piholeConfigBytes, err := schedule.PiholeConfigJSON()
	if err != nil {
		return fmt.Errorf("marshal pihole config: %w", err)
	}

	proxmoxOptionsBytes, err := schedule.ProxmoxOptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal proxmox options: %w", err)
	}

	classificationDataTypesBytes, err := schedule.ClassificationDataTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal classification data types: %w", err)
	}

	// Default backup type to file if not set
	backupType := string(schedule.BackupType)
	if backupType == "" {
		backupType = string(models.BackupTypeFile)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO schedules (id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths,
		                       excludes, retention_policy, bandwidth_limit_kbps,
		                       backup_window_start, backup_window_end, excluded_hours,
		                       compression_level, max_file_size_mb, on_mount_unavailable,
		                       priority, preemptible, classification_level, classification_data_types,
		                       docker_options, pihole_config, proxmox_options,
		                       enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
	`, schedule.ID, schedule.AgentID, schedule.AgentGroupID, schedule.PolicyID, schedule.Name,
		backupType, schedule.CronExpression, pathsBytes, excludesBytes, retentionBytes,
		schedule.BandwidthLimitKB, windowStart, windowEnd, excludedHoursBytes,
		schedule.CompressionLevel, schedule.MaxFileSizeMB, mountBehavior,
		schedule.Priority, schedule.Preemptible, schedule.ClassificationLevel, classificationDataTypesBytes,
		dockerOptionsBytes, piholeConfigBytes, proxmoxOptionsBytes,
		schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
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

	// Serialize backup type specific options
	dockerOptionsBytes, err := schedule.DockerOptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal docker options: %w", err)
	}

	piholeConfigBytes, err := schedule.PiholeConfigJSON()
	if err != nil {
		return fmt.Errorf("marshal pihole config: %w", err)
	}

	proxmoxOptionsBytes, err := schedule.ProxmoxOptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal proxmox options: %w", err)
	}

	classificationDataTypesBytes, err := schedule.ClassificationDataTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal classification data types: %w", err)
	}

	// Default backup type to file if not set
	backupType := string(schedule.BackupType)
	if backupType == "" {
		backupType = string(models.BackupTypeFile)
	}

	schedule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE schedules
		SET policy_id = $2, name = $3, backup_type = $4, cron_expression = $5, paths = $6, excludes = $7,
		    retention_policy = $8, bandwidth_limit_kbps = $9, backup_window_start = $10,
		    backup_window_end = $11, excluded_hours = $12, compression_level = $13,
		    max_file_size_mb = $14, on_mount_unavailable = $15,
		    priority = $16, preemptible = $17, classification_level = $18, classification_data_types = $19,
		    docker_options = $20, pihole_config = $21, proxmox_options = $22,
		    enabled = $23, updated_at = $24
		WHERE id = $1
	`, schedule.ID, schedule.PolicyID, schedule.Name, backupType, schedule.CronExpression, pathsBytes,
		excludesBytes, retentionBytes, schedule.BandwidthLimitKB, windowStart, windowEnd,
		excludedHoursBytes, schedule.CompressionLevel, schedule.MaxFileSizeMB, mountBehavior,
		schedule.Priority, schedule.Preemptible, schedule.ClassificationLevel, classificationDataTypesBytes,
		dockerOptionsBytes, piholeConfigBytes, proxmoxOptionsBytes,
		schedule.Enabled, schedule.UpdatedAt)
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
	var classificationDataTypesBytes, dockerOptionsBytes, piholeConfigBytes, proxmoxOptionsBytes []byte
	var agentGroupID *uuid.UUID
	var backupType, windowStart, windowEnd, compressionLevel, mountBehavior, classificationLevel *string
	err := rows.Scan(
		&s.ID, &s.AgentID, &agentGroupID, &s.PolicyID, &s.Name, &backupType, &s.CronExpression,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.BandwidthLimitKB,
		&windowStart, &windowEnd, &excludedHoursBytes, &compressionLevel, &s.MaxFileSizeMB,
		&mountBehavior,
		&s.Priority, &s.Preemptible, &classificationLevel, &classificationDataTypesBytes,
		&dockerOptionsBytes, &piholeConfigBytes, &proxmoxOptionsBytes,
		&s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	s.AgentGroupID = agentGroupID

	// Set backup type with default
	if backupType != nil && *backupType != "" {
		s.BackupType = models.BackupType(*backupType)
	} else {
		s.BackupType = models.BackupTypeFile
	}

	// Set backup window
	if windowStart != nil || windowEnd != nil {
		s.BackupWindow = &models.BackupWindow{}
		if windowStart != nil {
			s.BackupWindow.Start = *windowStart
		}
		if windowEnd != nil {
			s.BackupWindow.End = *windowEnd
		}
	}

	// Set compression level
	if compressionLevel != nil {
		s.CompressionLevel = compressionLevel
	}

	// Set mount behavior
	if mountBehavior != nil {
		s.OnMountUnavailable = models.MountBehavior(*mountBehavior)
	}

	// Set classification level
	if classificationLevel != nil {
		s.ClassificationLevel = *classificationLevel
	}

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
	if err := s.SetClassificationDataTypes(classificationDataTypesBytes); err != nil {
		return nil, fmt.Errorf("parse classification data types: %w", err)
	}
	if err := s.SetDockerOptions(dockerOptionsBytes); err != nil {
		return nil, fmt.Errorf("parse docker options: %w", err)
	}
	if err := s.SetPiholeConfig(piholeConfigBytes); err != nil {
		return nil, fmt.Errorf("parse pihole config: %w", err)
	}
	if err := s.SetProxmoxOptions(proxmoxOptionsBytes); err != nil {
		return nil, fmt.Errorf("parse proxmox options: %w", err)
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
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression,
		       paths, excludes, retention_policy, bandwidth_limit_kbps,
		       backup_window_start, backup_window_end, excluded_hours, compression_level,
		       max_file_size_mb, mount_behavior,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
		FROM backup_schedules
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

// Backup methods

// GetBackupsByScheduleID returns all backups for a schedule.
func (db *DB) GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE schedule_id = $1 AND deleted_at IS NULL
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
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE agent_id = $1 AND deleted_at IS NULL
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
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
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
		                     retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, backup.ID, backup.ScheduleID, backup.AgentID, backup.RepositoryID, backup.SnapshotID,
		backup.StartedAt, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError, backup.CreatedAt)
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
		    retention_applied = $9, snapshots_removed = $10, snapshots_kept = $11, retention_error = $12
		WHERE id = $1
	`, backup.ID, backup.SnapshotID, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError)
	if err != nil {
		return fmt.Errorf("update backup: %w", err)
	}
	return nil
}

// DeleteBackup soft-deletes a backup by setting deleted_at.
func (db *DB) DeleteBackup(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, `
		UPDATE backups SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("backup not found")
	}
	return nil
}

// scanBackups scans multiple backup rows.
func scanBackups(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Backup, error) {
	r := rows.(scanner)

	var backups []*models.Backup
	for r.Next() {
		var b models.Backup
		var statusStr string
		err := r.Scan(
			&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
			&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
			&b.FilesChanged, &b.ErrorMessage,
			&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
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
		SELECT s.id, s.agent_id, s.repository_id, s.name, s.cron_expression, s.paths, s.excludes,
		       s.retention_policy, s.enabled, s.created_at, s.updated_at
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
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
		FROM backup_schedules
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
		return nil, fmt.Errorf("parse policy paths: %w", err)
	}
	if err := p.SetExcludes(excludesBytes); err != nil {
		return nil, fmt.Errorf("parse policy excludes: %w", err)
	}
	if err := p.SetRetentionPolicy(retentionBytes); err != nil {
		return nil, fmt.Errorf("parse policy retention: %w", err)
	}
	if err := p.SetExcludedHours(excludedHoursBytes); err != nil {
		return nil, fmt.Errorf("parse policy excluded hours: %w", err)
	}
	if windowStart != nil || windowEnd != nil {
		p.BackupWindow = &models.BackupWindow{}
		if windowStart != nil {
			p.BackupWindow.Start = *windowStart
		}
		if windowEnd != nil {
			p.BackupWindow.End = *windowEnd
		}
	}

	return &p, nil
}

// GetLatestBackupByScheduleID returns the most recent backup for a schedule.
func (db *DB) GetLatestBackupByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.Backup, error) {
	var b models.Backup
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE schedule_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, scheduleID).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
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
		WHERE id = $1 AND deleted_at IS NULL
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

// DeleteRestore soft-deletes a restore by setting deleted_at.
func (db *DB) DeleteRestore(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, `
		UPDATE restores SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete restore: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("restore not found")
	}
	return nil
}

// scanRestores scans multiple restore rows.
func scanRestores(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.Restore, error) {
	r := rows.(scanner)

	var restores []*models.Restore
	for r.Next() {
		var restore models.Restore
		var statusStr string
		var includePathsBytes, excludePathsBytes []byte
		err := r.Scan(
			&restore.ID, &restore.AgentID, &restore.RepositoryID, &restore.SnapshotID,
			&restore.TargetPath, &includePathsBytes, &excludePathsBytes,
			&statusStr, &restore.StartedAt, &restore.CompletedAt,
			&restore.ErrorMessage, &restore.CreatedAt, &restore.UpdatedAt,
		)
		if err != nil {
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

// parseStringSlice parses a JSON byte slice into a string slice.
func parseStringSlice(data []byte, dest *[]string) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dest)
}

// toJSONBytes marshals a value to JSON bytes.
func toJSONBytes(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
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