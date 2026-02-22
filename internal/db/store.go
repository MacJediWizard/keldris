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

// scanner is an interface for row scanning (pgx.Rows, etc.)
type scanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

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
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		SELECT id, org_id, oidc_subject, email, name, role, created_at, updated_at
		FROM users
		WHERE oidc_subject = $1
	`, subject).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
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
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		SELECT id, org_id, oidc_subject, email, name, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
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
		INSERT INTO users (id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.OrgID, user.OIDCSubject, user.Email, user.Name, string(user.Role), user.IsSuperuser, user.CreatedAt, user.UpdatedAt)
		INSERT INTO users (id, org_id, oidc_subject, email, name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, user.ID, user.OrgID, user.OIDCSubject, user.Email, user.Name, string(user.Role), user.CreatedAt, user.UpdatedAt)
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
		SELECT id, org_id, hostname, api_key_hash, os_info, last_seen, status,
		       health_status, health_metrics, health_checked_at, created_at, updated_at
		SELECT id, org_id, hostname, api_key_hash, os_info, last_seen, status, created_at, updated_at
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
		var healthMetricsBytes []byte
		var statusStr string
		var healthStatusStr *string
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes, &networkMountsBytes,
			&a.LastSeen, &statusStr, &healthStatusStr, &healthMetricsBytes,
			&a.HealthCheckedAt, &a.CreatedAt, &a.UpdatedAt,
		var statusStr string
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes,
			&a.LastSeen, &statusStr, &a.CreatedAt, &a.UpdatedAt,
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
		if err := a.SetHealthMetrics(healthMetricsBytes); err != nil {
			db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse health metrics")
		}
		if err := a.SetOSInfo(osInfoBytes); err != nil {
			db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
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
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, last_seen, status, created_at, updated_at
		FROM agents
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes,
		&a.LastSeen, &statusStr, &a.CreatedAt, &a.UpdatedAt,
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
	if err := a.SetHealthMetrics(healthMetricsBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse health metrics")
	}
	if err := a.SetOSInfo(osInfoBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
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
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, api_key_hash, os_info, last_seen, status, created_at, updated_at
		FROM agents
		WHERE api_key_hash = $1
	`, hash).Scan(
		&a.ID, &a.OrgID, &a.Hostname, &a.APIKeyHash, &osInfoBytes,
		&a.LastSeen, &statusStr, &a.CreatedAt, &a.UpdatedAt,
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
	if err := a.SetHealthMetrics(healthMetricsBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse health metrics")
	}
	if err := a.SetOSInfo(osInfoBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
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

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agents (id, org_id, hostname, api_key_hash, os_info, last_seen, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, agent.ID, agent.OrgID, agent.Hostname, agent.APIKeyHash, osInfoBytes,
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
		SET hostname = $2, os_info = $3, last_seen = $4, status = $5, updated_at = $6,
		    health_status = $7, health_metrics = $8, health_checked_at = $9
		WHERE id = $1
	`, agent.ID, agent.Hostname, osInfoBytes, networkMountsBytes, agent.LastSeen, string(agent.Status), agent.UpdatedAt,
		string(agent.HealthStatus), healthMetricsBytes, agent.HealthCheckedAt)
		SET hostname = $2, os_info = $3, last_seen = $4, status = $5, updated_at = $6
		WHERE id = $1
	`, agent.ID, agent.Hostname, osInfoBytes, agent.LastSeen, string(agent.Status), agent.UpdatedAt)
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
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at
		SELECT id, agent_id, policy_id, name, cron_expression, paths, excludes,
		SELECT id, agent_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		       excluded_hours, compression_level, enabled, created_at, updated_at
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		SELECT id, agent_id, repository_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
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
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at
		SELECT id, agent_id, policy_id, name, cron_expression, paths, excludes,
		SELECT id, agent_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		       excluded_hours, compression_level, enabled, created_at, updated_at
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		SELECT id, agent_id, repository_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
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
	return scanScheduleRow(row)
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
		                       compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		                       excludes, retention_policy, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		                       compression_level, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		                       compression_level, on_mount_unavailable, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, schedule.ID, schedule.AgentID, schedule.AgentGroupID, schedule.PolicyID, schedule.Name,
		INSERT INTO schedules (id, agent_id, policy_id, name, cron_expression, paths,
		                       excludes, retention_policy, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, schedule.ID, schedule.AgentID, schedule.PolicyID, schedule.Name,
		INSERT INTO schedules (id, agent_id, name, cron_expression, paths,
		                       excludes, retention_policy, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, schedule.ID, schedule.AgentID, schedule.Name,
		schedule.CronExpression, pathsBytes, excludesBytes, retentionBytes,
		schedule.BandwidthLimitKB, windowStart, windowEnd, excludedHoursBytes,
		schedule.CompressionLevel, schedule.MaxFileSizeMB, mountBehavior, schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
		schedule.CompressionLevel, schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
		                       compression_level, max_file_size_mb, on_mount_unavailable,
		                       priority, preemptible, classification_level, classification_data_types,
		                       docker_options, pihole_config, proxmox_options,
		                       enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
	`, schedule.ID, schedule.AgentID, schedule.AgentGroupID, schedule.PolicyID, schedule.Name,
		backupType, schedule.CronExpression, pathsBytes, excludesBytes, retentionBytes,
		schedule.BandwidthLimitKB, windowStart, windowEnd, excludedHoursBytes,
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

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO schedules (id, agent_id, repository_id, name, cron_expression, paths,
		                       excludes, retention_policy, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, schedule.ID, schedule.AgentID, schedule.RepositoryID, schedule.Name,
		schedule.CronExpression, pathsBytes, excludesBytes, retentionBytes,
		schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create schedule: %w", err)
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
		excludedHoursBytes, schedule.CompressionLevel, schedule.MaxFileSizeMB, mountBehavior, schedule.Enabled, schedule.UpdatedAt)
		    retention_policy = $7, enabled = $8, updated_at = $9
		    enabled = $13, updated_at = $14
		WHERE id = $1
	`, schedule.ID, schedule.PolicyID, schedule.Name, schedule.CronExpression, pathsBytes,
		excludesBytes, retentionBytes, schedule.BandwidthLimitKB, windowStart,
		windowEnd, excludedHoursBytes, schedule.CompressionLevel,
		excludedHoursBytes, schedule.CompressionLevel, schedule.MaxFileSizeMB, mountBehavior,
		schedule.Priority, schedule.Preemptible, schedule.ClassificationLevel, classificationDataTypesBytes,
		dockerOptionsBytes, piholeConfigBytes, proxmoxOptionsBytes,
		schedule.Enabled, schedule.UpdatedAt)
	schedule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE schedules
		SET policy_id = $2, name = $3, cron_expression = $4, paths = $5, excludes = $6,
		    retention_policy = $7, bandwidth_limit_kbps = $8, backup_window_start = $9,
		    backup_window_end = $10, excluded_hours = $11, compression_level = $12,
		    on_mount_unavailable = $13, enabled = $14, updated_at = $15
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
		excludedHoursBytes, schedule.CompressionLevel, mountBehavior, schedule.Enabled, schedule.UpdatedAt)
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
		&mountBehavior, &s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	var pathsBytes, excludesBytes, retentionBytes []byte
	var agentGroupID *uuid.UUID
	var windowStart, windowEnd, compressionLevel, mountBehavior *string
	err := rows.Scan(
		&s.ID, &s.AgentID, &agentGroupID, &s.PolicyID, &s.Name, &s.CronExpression,
		&s.ID, &s.AgentID, &s.PolicyID, &s.Name, &s.CronExpression,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.Enabled,
		&s.CreatedAt, &s.UpdatedAt,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.BandwidthLimitKB,
		&windowStart, &windowEnd, &excludedHoursBytes, &compressionLevel,
		&mountBehavior, &s.Priority, &s.Preemptible, &classificationLevel, &classificationDataTypesBytes,
		&dockerOptionsBytes, &piholeConfigBytes, &proxmoxOptionsBytes,
		&windowStart, &windowEnd, &excludedHoursBytes, &compressionLevel, &mountBehavior,
		&s.Enabled, &s.CreatedAt, &s.UpdatedAt,
	var pathsBytes, excludesBytes, retentionBytes []byte
	err := rows.Scan(
		&s.ID, &s.AgentID, &s.Name, &s.CronExpression,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.Enabled,
		&s.CreatedAt, &s.UpdatedAt,
		&mountBehavior, &s.Priority, &s.Preemptible, &classificationLevel, &classificationDataTypesBytes,
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
	schedule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE schedules
		SET name = $2, cron_expression = $3, paths = $4, excludes = $5,
		    retention_policy = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, schedule.ID, schedule.Name, schedule.CronExpression, pathsBytes,
		excludesBytes, retentionBytes, schedule.Enabled, schedule.UpdatedAt)
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

// scanSchedule scans a schedule from a row iterator.
func scanSchedule(rows interface {
	Scan(dest ...any) error
}) (*models.Schedule, error) {
	var s models.Schedule
	var pathsBytes, excludesBytes, retentionBytes []byte
	err := rows.Scan(
		&s.ID, &s.AgentID, &s.RepositoryID, &s.Name, &s.CronExpression,
		&pathsBytes, &excludesBytes, &retentionBytes, &s.Enabled,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan schedule: %w", err)
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
	s.SetBackupWindow(windowStart, windowEnd)
	s.CompressionLevel = compressionLevel

	// Set classification
	if classificationLevel != nil {
		s.ClassificationLevel = *classificationLevel
	}
	if err := s.SetClassificationDataTypes(classificationDataTypesBytes); err != nil {
		return nil, fmt.Errorf("parse classification data types: %w", err)
	}

	// Set backup type specific options
	if err := s.SetDockerOptions(dockerOptionsBytes); err != nil {
		return nil, fmt.Errorf("parse docker options: %w", err)
	}
	if err := s.SetPiholeConfig(piholeConfigBytes); err != nil {
		return nil, fmt.Errorf("parse pihole config: %w", err)
	}
	if err := s.SetProxmoxOptions(proxmoxOptionsBytes); err != nil {
		return nil, fmt.Errorf("parse proxmox options: %w", err)
	}

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
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
// Backup methods

// GetBackupsByScheduleID returns all backups for a schedule.
func (db *DB) GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE schedule_id = $1
		ORDER BY started_at DESC
	`, scheduleID)
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
// GetBackupsByAgentID returns all backups for an agent.
func (db *DB) GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		return nil, fmt.Errorf("list backups by schedule: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupsByAgentID returns all backups for an agent.
func (db *DB) GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE agent_id = $1
		ORDER BY started_at DESC
	`, agentID)
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := policy.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	retentionBytes, err := policy.RetentionPolicyJSON()
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
		WHERE id = $1
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	excludedHoursBytes, err := policy.ExcludedHoursJSON()
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
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE id = $1
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
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
		INSERT INTO backups (id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		                     status, size_bytes, files_new, files_changed, error_message,
		                     retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, backup.ID, backup.ScheduleID, backup.AgentID, backup.SnapshotID,
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

// UpdatePolicy updates an existing policy.
func (db *DB) UpdatePolicy(ctx context.Context, policy *models.Policy) error {
	pathsBytes, err := policy.PathsJSON()
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	excludesBytes, err := policy.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
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
			&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError, &b.CreatedAt,
			&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
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
		return fmt.Errorf("delete policy: %w", err)
	}
	return nil
}

// GetSchedulesByPolicyID returns all schedules using a policy.
func (db *DB) GetSchedulesByPolicyID(ctx context.Context, policyID uuid.UUID) ([]*models.Schedule, error) {
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message, created_at
		FROM backups
		WHERE snapshot_id = $1
	`, snapshotID).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage, &b.CreatedAt,
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
		SELECT id, agent_id, repository_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
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
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error, created_at
		FROM backups
		WHERE schedule_id = $1
		ORDER BY started_at DESC
	`, scheduleID)
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
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at
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
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
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
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		WHERE agent_id = $1 AND deleted_at IS NULL
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
	var excludedLargeFilesJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		       pre_script_output, pre_script_error, post_script_output, post_script_error, created_at
		FROM backups
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
		&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError,
		&excludedLargeFilesJSON, &b.Resumed, &b.CheckpointID, &b.OriginalBackupID, &b.CreatedAt,
		&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	b.Status = models.BackupStatus(statusStr)
	if excludedLargeFilesJSON != nil {
		if err := json.Unmarshal(excludedLargeFilesJSON, &b.ExcludedLargeFiles); err != nil {
			return nil, fmt.Errorf("unmarshal excluded large files: %w", err)
		}
	}
	return &b, nil
}

// CreateBackup creates a new backup record.
func (db *DB) CreateBackup(ctx context.Context, backup *models.Backup) error {
	var excludedLargeFilesJSON []byte
	if backup.ExcludedLargeFiles != nil {
		var err error
		excludedLargeFilesJSON, err = json.Marshal(backup.ExcludedLargeFiles)
		if err != nil {
			return fmt.Errorf("marshal excluded large files: %w", err)
		}
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backups (id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		                     status, size_bytes, files_new, files_changed, error_message,
		                     retention_applied, snapshots_removed, snapshots_kept, retention_error,
		                     pre_script_output, pre_script_error, post_script_output, post_script_error,
		                     excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		                     pre_script_output, pre_script_error, post_script_output, post_script_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, backup.ID, backup.ScheduleID, backup.AgentID, backup.RepositoryID, backup.SnapshotID,
		backup.StartedAt, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError,
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError,
		excludedLargeFilesJSON, backup.Resumed, backup.CheckpointID, backup.OriginalBackupID, backup.CreatedAt)
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError, backup.CreatedAt)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	return nil
}

// UpdateBackup updates an existing backup record.
func (db *DB) UpdateBackup(ctx context.Context, backup *models.Backup) error {
	var excludedLargeFilesJSON []byte
	if backup.ExcludedLargeFiles != nil {
		var err error
		excludedLargeFilesJSON, err = json.Marshal(backup.ExcludedLargeFiles)
		if err != nil {
			return fmt.Errorf("marshal excluded large files: %w", err)
		}
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
		UPDATE backups
		SET snapshot_id = $2, completed_at = $3, status = $4, size_bytes = $5,
		    files_new = $6, files_changed = $7, error_message = $8,
		    retention_applied = $9, snapshots_removed = $10, snapshots_kept = $11, retention_error = $12,
		    pre_script_output = $13, pre_script_error = $14, post_script_output = $15, post_script_error = $16,
		    excluded_large_files = $17
		    pre_script_output = $13, pre_script_error = $14, post_script_output = $15, post_script_error = $16
		WHERE id = $1
	`, backup.ID, backup.SnapshotID, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage,
		backup.RetentionApplied, backup.SnapshotsRemoved, backup.SnapshotsKept, backup.RetentionError,
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError,
		excludedLargeFilesJSON)
		backup.PreScriptOutput, backup.PreScriptError, backup.PostScriptOutput, backup.PostScriptError)
	if err != nil {
		return fmt.Errorf("update backup: %w", err)
	}
	return nil
}

// DeleteBackup soft-deletes a backup record by setting deleted_at.
func (db *DB) DeleteBackup(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE backups SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete backup: %w", err)
	}
	if result.RowsAffected() == 0 {
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
		var excludedLargeFilesJSON []byte
		err := r.Scan(
			&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
			&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
			&b.FilesChanged, &b.ErrorMessage,
			&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
			&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError,
			&excludedLargeFilesJSON, &b.Resumed, &b.CheckpointID, &b.OriginalBackupID, &b.CreatedAt,
			&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		b.Status = models.BackupStatus(statusStr)
		if excludedLargeFilesJSON != nil {
			if err := json.Unmarshal(excludedLargeFilesJSON, &b.ExcludedLargeFiles); err != nil {
				return nil, fmt.Errorf("unmarshal excluded large files: %w", err)
			}
		}
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
// GetAllSchedules returns all enabled schedules across all organizations (for monitoring).
func (db *DB) GetAllSchedules(ctx context.Context) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression, s.paths, s.excludes,
		       s.retention_policy, s.bandwidth_limit_kbps, s.backup_window_start, s.backup_window_end,
		       s.excluded_hours, s.compression_level, s.on_mount_unavailable, s.enabled, s.created_at, s.updated_at
		FROM schedules s
		WHERE s.enabled = true
		ORDER BY s.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list all schedules: %w", err)
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
// GetEnabledSchedules returns all enabled schedules.
func (db *DB) GetEnabledSchedules(ctx context.Context) ([]models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, backup_type, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable,
		       priority, preemptible, classification_level, classification_data_types,
		       docker_options, pihole_config, proxmox_options,
		       enabled, created_at, updated_at
		FROM schedules
		WHERE enabled = true
		ORDER BY name
	`)
	if err != nil {
		return fmt.Errorf("update replication status: %w", err)
	}
	return nil
}

// GetBackupBySnapshotID returns a backup by its snapshot ID.
func (db *DB) GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error) {
	var b models.Backup
	var statusStr string
	var excludedLargeFilesJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		WHERE snapshot_id = $1 AND deleted_at IS NULL
	`, snapshotID).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage,
		&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
		&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError,
		&excludedLargeFilesJSON, &b.Resumed, &b.CheckpointID, &b.OriginalBackupID, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup by snapshot ID: %w", err)
	}
	b.Status = models.BackupStatus(statusStr)
	if excludedLargeFilesJSON != nil {
		if err := json.Unmarshal(excludedLargeFilesJSON, &b.ExcludedLargeFiles); err != nil {
			return nil, fmt.Errorf("unmarshal excluded large files: %w", err)
		}
	}
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
		       s.excluded_hours, s.compression_level, s.max_file_size_mb, s.on_mount_unavailable, s.enabled, s.created_at, s.updated_at
		       s.retention_policy, s.enabled, s.created_at, s.updated_at
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
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at
		       retention_policy, enabled, created_at, updated_at
		       excluded_hours, compression_level, enabled, created_at, updated_at
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
		WHERE schedule_id = $1 AND deleted_at IS NULL
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
		WHERE agent_id = $1 AND deleted_at IS NULL
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

// DeleteRestore soft-deletes a restore record by setting deleted_at.
func (db *DB) DeleteRestore(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE restores SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete restore: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("restore not found")
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

// GetInvitationByID returns an invitation by its ID.
func (db *DB) GetInvitationByID(ctx context.Context, id uuid.UUID) (*models.OrgInvitation, error) {
	var inv models.OrgInvitation
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		FROM org_invitations WHERE id = $1
	`, id).Scan(&inv.ID, &inv.OrgID, &inv.Email, &roleStr, &inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get invitation by id: %w", err)
	}
	inv.Role = models.OrgRole(roleStr)
	return &inv, nil
}

// UpdateInvitationResent updates the resent timestamp and count for an invitation.
func (db *DB) UpdateInvitationResent(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_invitations
		SET resent_at = NOW(), resent_count = COALESCE(resent_count, 0) + 1
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("update invitation resent: %w", err)
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
	args := []any{repositoryID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
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
		WHERE repository_id = $1 AND deleted_at IS NULL
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
		WHERE repository_id = $1 AND deleted_at IS NULL
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
		WHERE id = $1 AND deleted_at IS NULL
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

// DeleteVerification soft-deletes a verification record by setting deleted_at.
func (db *DB) DeleteVerification(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE verifications SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete verification: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("verification not found")
	}
	return nil
}

// GetConsecutiveFailedVerifications returns the count of consecutive failed verifications for a repository.
func (db *DB) GetConsecutiveFailedVerifications(ctx context.Context, repoID uuid.UUID) (int, error) {
	// Count consecutive failures from the most recent verification backwards
	rows, err := db.Pool.Query(ctx, `
		SELECT status
		FROM verifications
		WHERE repository_id = $1 AND deleted_at IS NULL
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
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE id = $1
	`, id).Scan(
		&m.ID, &m.OrgID, &m.Title, &m.Message, &m.StartsAt, &m.EndsAt,
		&m.NotifyBeforeMinutes, &m.NotificationSent, &m.ReadOnly, &m.CountdownStartMinutes,
		&m.EmergencyOverride, &m.OverriddenBy, &m.OverriddenAt, &m.CreatedBy,
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
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
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
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
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
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
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
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
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
		            notify_before_minutes, notification_sent, read_only, countdown_start_minutes,
		            emergency_override, overridden_by, overridden_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, m.ID, m.OrgID, m.Title, m.Message, m.StartsAt, m.EndsAt,
		m.NotifyBeforeMinutes, m.NotificationSent, m.ReadOnly, m.CountdownStartMinutes,
		m.EmergencyOverride, m.OverriddenBy, m.OverriddenAt, m.CreatedBy, m.CreatedAt, m.UpdatedAt)
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
		    notify_before_minutes = $6, notification_sent = $7, read_only = $8,
		    countdown_start_minutes = $9, emergency_override = $10,
		    overridden_by = $11, overridden_at = $12, updated_at = $13
		WHERE id = $1
	`, m.ID, m.Title, m.Message, m.StartsAt, m.EndsAt,
		m.NotifyBeforeMinutes, m.NotificationSent, m.ReadOnly, m.CountdownStartMinutes,
		m.EmergencyOverride, m.OverriddenBy, m.OverriddenAt, m.UpdatedAt)
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

// Snapshot Comment methods

// CreateSnapshotComment creates a new snapshot comment.
func (db *DB) CreateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_comments (id, org_id, snapshot_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, comment.ID, comment.OrgID, comment.SnapshotID, comment.UserID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot comment: %w", err)
	}
	return nil
}

// GetSnapshotCommentsBySnapshotID returns all comments for a snapshot within an organization.
func (db *DB) GetSnapshotCommentsBySnapshotID(ctx context.Context, snapshotID string, orgID uuid.UUID) ([]*models.SnapshotComment, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, snapshot_id, user_id, content, created_at, updated_at
		FROM snapshot_comments
		WHERE snapshot_id = $1 AND org_id = $2
		ORDER BY created_at DESC
	`, snapshotID, orgID)
	if err != nil {
		return nil, fmt.Errorf("list snapshot comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.SnapshotComment
	for rows.Next() {
		var c models.SnapshotComment
		err := rows.Scan(&c.ID, &c.OrgID, &c.SnapshotID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot comment: %w", err)
		}
		comments = append(comments, &c)
	}

	return comments, nil
}

// GetSnapshotCommentByID returns a snapshot comment by ID.
func (db *DB) GetSnapshotCommentByID(ctx context.Context, id uuid.UUID) (*models.SnapshotComment, error) {
	var c models.SnapshotComment
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_id, user_id, content, created_at, updated_at
		FROM snapshot_comments
		WHERE id = $1
	`, id).Scan(&c.ID, &c.OrgID, &c.SnapshotID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get snapshot comment: %w", err)
	}
	return &c, nil
}

// DeleteSnapshotComment deletes a snapshot comment by ID.
func (db *DB) DeleteSnapshotComment(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM snapshot_comments
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete snapshot comment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot comment not found")
	}
	return nil
}

// UpdateSnapshotComment updates a snapshot comment's content.
func (db *DB) UpdateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error {
	comment.UpdatedAt = time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_comments
		SET content = $2, updated_at = $3
		WHERE id = $1
	`, comment.ID, comment.Content, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update snapshot comment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot comment not found")
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

// SetMaintenanceEmergencyOverride sets or clears the emergency override for a maintenance window.
func (db *DB) SetMaintenanceEmergencyOverride(ctx context.Context, id uuid.UUID, override bool, userID uuid.UUID) error {
	now := time.Now()
	var overriddenBy *uuid.UUID
	var overriddenAt *time.Time
	if override {
		overriddenBy = &userID
		overriddenAt = &now
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE maintenance_windows
		SET emergency_override = $2, overridden_by = $3, overridden_at = $4, updated_at = $5
		WHERE id = $1
	`, id, override, overriddenBy, overriddenAt, now)
	if err != nil {
		return fmt.Errorf("set maintenance emergency override: %w", err)
	}
	return nil
}

// GetActiveReadOnlyWindow returns the currently active read-only maintenance window for an org.
func (db *DB) GetActiveReadOnlyWindow(ctx context.Context, orgID uuid.UUID, now time.Time) (*models.MaintenanceWindow, error) {
	var m models.MaintenanceWindow
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, title, message, starts_at, ends_at, notify_before_minutes,
		       notification_sent, read_only, countdown_start_minutes, emergency_override,
		       overridden_by, overridden_at, created_by, created_at, updated_at
		FROM maintenance_windows
		WHERE org_id = $1
		  AND read_only = true
		  AND emergency_override = false
		  AND starts_at <= $2
		  AND ends_at > $2
		ORDER BY ends_at ASC
		LIMIT 1
	`, orgID, now).Scan(
		&m.ID, &m.OrgID, &m.Title, &m.Message, &m.StartsAt, &m.EndsAt,
		&m.NotifyBeforeMinutes, &m.NotificationSent, &m.ReadOnly, &m.CountdownStartMinutes,
		&m.EmergencyOverride, &m.OverriddenBy, &m.OverriddenAt, &m.CreatedBy,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get active read-only maintenance window: %w", err)
	}
	return &m, nil
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
	r := rows.(scanner)

	var windows []*models.MaintenanceWindow
	for r.Next() {
		var m models.MaintenanceWindow
		err := r.Scan(
			&m.ID, &m.OrgID, &m.Title, &m.Message, &m.StartsAt, &m.EndsAt,
			&m.NotifyBeforeMinutes, &m.NotificationSent, &m.ReadOnly, &m.CountdownStartMinutes,
			&m.EmergencyOverride, &m.OverriddenBy, &m.OverriddenAt, &m.CreatedBy,
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
		WHERE runbook_id = $1 AND deleted_at IS NULL
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
		WHERE r.org_id = $1 AND t.deleted_at IS NULL
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
		WHERE id = $1 AND deleted_at IS NULL
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
		WHERE runbook_id = $1 AND deleted_at IS NULL
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

// DeleteDRTest soft-deletes a DR test record by setting deleted_at.
func (db *DB) DeleteDRTest(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE dr_tests SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete DR test: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("DR test not found")
	}
	return nil
}

// scanDRTests scans multiple DR test rows.
func scanDRTests(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.DRTest, error) {
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
		WHERE r.org_id = $1 AND t.deleted_at IS NULL AND t.created_at > NOW() - INTERVAL '30 days'
	`, orgID).Scan(&status.TestsLast30Days, &status.PassRate)
	if err != nil {
		return nil, fmt.Errorf("get test statistics: %w", err)
	}

	// Get last test date
	err = db.Pool.QueryRow(ctx, `
		SELECT MAX(t.completed_at)
		FROM dr_tests t
		JOIN dr_runbooks r ON t.runbook_id = r.id
		WHERE r.org_id = $1 AND t.deleted_at IS NULL
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
		SELECT DISTINCT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error,
		       b.excluded_large_files, b.resumed, b.checkpoint_id, b.original_backup_id, b.created_at
		FROM backups b
		JOIN backup_tags bt ON b.id = bt.backup_id
		WHERE bt.tag_id = ANY($1) AND b.deleted_at IS NULL
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
	argNum := 3

	if filter.Status != "" {
		sqlQuery += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY hostname LIMIT $%d", argNum)
	args = append(args, filter.Limit)

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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND (b.snapshot_id ILIKE $2 OR b.id::text ILIKE $2)
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

	sqlQuery += fmt.Sprintf(" ORDER BY b.started_at DESC LIMIT $%d", argNum)
	args = append(args, filter.Limit)

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

	sqlQuery += " ORDER BY s.name LIMIT $3"
	args = append(args, filter.Limit)

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

	sqlQuery += " ORDER BY name LIMIT $3"
	args = append(args, filter.Limit)

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
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error,
		       b.excluded_large_files, b.resumed, b.checkpoint_id, b.original_backup_id, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.status = 'running'
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.status = 'failed' AND b.started_at >= $2
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL
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
// GetEnabledSchedulesByOrgID returns all enabled backup schedules for an org.
func (db *DB) GetEnabledSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.backup_type, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps, s.backup_window_start, s.backup_window_end,
		       s.excluded_hours, s.compression_level, s.max_file_size_mb, s.on_mount_unavailable,
		       s.priority, s.preemptible, s.classification_level, s.classification_data_types,
		       s.docker_options, s.pihole_config, s.proxmox_options,
		       s.enabled, s.created_at, s.updated_at
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL
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
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2
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
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at,
		       b.completed_at, b.status, b.size_bytes, b.files_new,
		       b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error,
		       b.excluded_large_files, b.resumed, b.checkpoint_id, b.original_backup_id, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2 AND b.started_at <= $3
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
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps, s.backup_window_start, s.backup_window_end,
		       s.excluded_hours, s.compression_level, s.max_file_size_mb, s.on_mount_unavailable, s.enabled,
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
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, max_file_size_mb, on_mount_unavailable, enabled, created_at, updated_at
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

// GetSnapshotCommentCounts returns comment counts for multiple snapshots within an organization.
func (db *DB) GetSnapshotCommentCounts(ctx context.Context, snapshotIDs []string, orgID uuid.UUID) (map[string]int, error) {
	if len(snapshotIDs) == 0 {
		return make(map[string]int), nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT snapshot_id, COUNT(*) as count
		FROM snapshot_comments
		WHERE snapshot_id = ANY($1) AND org_id = $2
		GROUP BY snapshot_id
	`, snapshotIDs, orgID)
	if err != nil {
		return nil, fmt.Errorf("count snapshot comments: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var snapshotID string
		var count int
		if err := rows.Scan(&snapshotID, &count); err != nil {
			return nil, fmt.Errorf("scan comment count: %w", err)
		}
		counts[snapshotID] = count
	}

	return counts, nil
}
// Cost Estimation methods

// GetStoragePricingByOrgID returns all custom storage pricing for an organization.
func (db *DB) GetStoragePricingByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.StoragePricing, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_type, storage_per_gb_month, egress_per_gb,
		       operations_per_k, provider_name, provider_description, created_at, updated_at
		FROM storage_pricing
		WHERE org_id = $1
		ORDER BY repository_type
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list storage pricing: %w", err)
	}
	defer rows.Close()

	var pricing []*models.StoragePricing
	for rows.Next() {
		var p models.StoragePricing
		var providerName, providerDesc *string
		err := rows.Scan(
			&p.ID, &p.OrgID, &p.RepositoryType, &p.StoragePerGBMonth,
			&p.EgressPerGB, &p.OperationsPerK, &providerName, &providerDesc,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan storage pricing: %w", err)
		}
		if providerName != nil {
			p.ProviderName = *providerName
		}
		if providerDesc != nil {
			p.ProviderDescription = *providerDesc
		}
		pricing = append(pricing, &p)
	}

	return pricing, nil
}

// GetStoragePricingByType returns custom pricing for a specific repository type.
func (db *DB) GetStoragePricingByType(ctx context.Context, orgID uuid.UUID, repoType string) (*models.StoragePricing, error) {
	var p models.StoragePricing
	var providerName, providerDesc *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, repository_type, storage_per_gb_month, egress_per_gb,
		       operations_per_k, provider_name, provider_description, created_at, updated_at
		FROM storage_pricing
		WHERE org_id = $1 AND repository_type = $2
	`, orgID, repoType).Scan(
		&p.ID, &p.OrgID, &p.RepositoryType, &p.StoragePerGBMonth,
		&p.EgressPerGB, &p.OperationsPerK, &providerName, &providerDesc,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get storage pricing: %w", err)
	}
	if providerName != nil {
		p.ProviderName = *providerName
	}
	if providerDesc != nil {
		p.ProviderDescription = *providerDesc
	}
	return &p, nil
}

// CreateStoragePricing creates a new custom storage pricing record.
func (db *DB) CreateStoragePricing(ctx context.Context, p *models.StoragePricing) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO storage_pricing (id, org_id, repository_type, storage_per_gb_month,
		             egress_per_gb, operations_per_k, provider_name, provider_description,
		             created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, p.ID, p.OrgID, p.RepositoryType, p.StoragePerGBMonth,
		p.EgressPerGB, p.OperationsPerK, p.ProviderName, p.ProviderDescription,
		p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create storage pricing: %w", err)
	}
	return nil
}

// UpdateStoragePricing updates an existing storage pricing record.
func (db *DB) UpdateStoragePricing(ctx context.Context, p *models.StoragePricing) error {
	p.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE storage_pricing
		SET storage_per_gb_month = $2, egress_per_gb = $3, operations_per_k = $4,
		    provider_name = $5, provider_description = $6, updated_at = $7
		WHERE id = $1
	`, p.ID, p.StoragePerGBMonth, p.EgressPerGB, p.OperationsPerK,
		p.ProviderName, p.ProviderDescription, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update storage pricing: %w", err)
	}
	return nil
}

// DeleteStoragePricing deletes a storage pricing record.
func (db *DB) DeleteStoragePricing(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM storage_pricing WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete storage pricing: %w", err)
	}
	return nil
}

// CreateCostEstimate creates a new cost estimate record.
func (db *DB) CreateCostEstimate(ctx context.Context, e *models.CostEstimateRecord) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO cost_estimates (id, org_id, repository_id, storage_size_bytes,
		             monthly_cost, yearly_cost, cost_per_gb, estimated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, e.ID, e.OrgID, e.RepositoryID, e.StorageSizeBytes,
		e.MonthlyCost, e.YearlyCost, e.CostPerGB, e.EstimatedAt, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("create cost estimate: %w", err)
	}
	return nil
}

// GetLatestCostEstimates returns the latest cost estimate for each repository in an organization.
func (db *DB) GetLatestCostEstimates(ctx context.Context, orgID uuid.UUID) ([]*models.CostEstimateRecord, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT ON (repository_id)
		       id, org_id, repository_id, storage_size_bytes, monthly_cost,
		       yearly_cost, cost_per_gb, estimated_at, created_at
		FROM cost_estimates
		WHERE org_id = $1
		ORDER BY repository_id, estimated_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get latest cost estimates: %w", err)
	}
	defer rows.Close()

	var estimates []*models.CostEstimateRecord
	for rows.Next() {
		var e models.CostEstimateRecord
		err := rows.Scan(
			&e.ID, &e.OrgID, &e.RepositoryID, &e.StorageSizeBytes,
			&e.MonthlyCost, &e.YearlyCost, &e.CostPerGB, &e.EstimatedAt, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cost estimate: %w", err)
		}
		estimates = append(estimates, &e)
	}

	return estimates, nil
}

// GetCostEstimateHistory returns historical cost estimates for a repository.
func (db *DB) GetCostEstimateHistory(ctx context.Context, repoID uuid.UUID, days int) ([]*models.CostEstimateRecord, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, storage_size_bytes, monthly_cost,
		       yearly_cost, cost_per_gb, estimated_at, created_at
		FROM cost_estimates
		WHERE repository_id = $1
		AND estimated_at >= CURRENT_DATE - INTERVAL '1 day' * $2
		ORDER BY estimated_at DESC
	`, repoID, days)
	if err != nil {
		return nil, fmt.Errorf("get cost estimate history: %w", err)
	}
	defer rows.Close()

	var estimates []*models.CostEstimateRecord
	for rows.Next() {
		var e models.CostEstimateRecord
		err := rows.Scan(
			&e.ID, &e.OrgID, &e.RepositoryID, &e.StorageSizeBytes,
			&e.MonthlyCost, &e.YearlyCost, &e.CostPerGB, &e.EstimatedAt, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cost estimate: %w", err)
		}
		estimates = append(estimates, &e)
	}

	return estimates, nil
}

// GetCostAlertsByOrgID returns all cost alerts for an organization.
func (db *DB) GetCostAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.CostAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, monthly_threshold, enabled, notify_on_exceed,
		       notify_on_forecast, forecast_months, last_triggered_at, created_at, updated_at
		FROM cost_alerts
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list cost alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.CostAlert
	for rows.Next() {
		var a models.CostAlert
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Name, &a.MonthlyThreshold, &a.Enabled,
			&a.NotifyOnExceed, &a.NotifyOnForecast, &a.ForecastMonths,
			&a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cost alert: %w", err)
		}
		alerts = append(alerts, &a)
	}

	return alerts, nil
}

// GetCostAlertByID returns a cost alert by ID.
func (db *DB) GetCostAlertByID(ctx context.Context, id uuid.UUID) (*models.CostAlert, error) {
	var a models.CostAlert
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, monthly_threshold, enabled, notify_on_exceed,
		       notify_on_forecast, forecast_months, last_triggered_at, created_at, updated_at
		FROM cost_alerts
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Name, &a.MonthlyThreshold, &a.Enabled,
		&a.NotifyOnExceed, &a.NotifyOnForecast, &a.ForecastMonths,
		&a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get cost alert: %w", err)
	}
	return &a, nil
}

// CreateCostAlert creates a new cost alert.
func (db *DB) CreateCostAlert(ctx context.Context, a *models.CostAlert) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO cost_alerts (id, org_id, name, monthly_threshold, enabled,
		             notify_on_exceed, notify_on_forecast, forecast_months,
		             created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.OrgID, a.Name, a.MonthlyThreshold, a.Enabled,
		a.NotifyOnExceed, a.NotifyOnForecast, a.ForecastMonths,
		a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create cost alert: %w", err)
	}
	return nil
}

// UpdateCostAlert updates an existing cost alert.
func (db *DB) UpdateCostAlert(ctx context.Context, a *models.CostAlert) error {
	a.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE cost_alerts
		SET name = $2, monthly_threshold = $3, enabled = $4, notify_on_exceed = $5,
		    notify_on_forecast = $6, forecast_months = $7, updated_at = $8
		WHERE id = $1
	`, a.ID, a.Name, a.MonthlyThreshold, a.Enabled,
		a.NotifyOnExceed, a.NotifyOnForecast, a.ForecastMonths, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update cost alert: %w", err)
	}
	return nil
}

// DeleteCostAlert deletes a cost alert.
func (db *DB) DeleteCostAlert(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM cost_alerts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete cost alert: %w", err)
	}
	return nil
}

// UpdateCostAlertTriggered updates the last triggered timestamp for a cost alert.
func (db *DB) UpdateCostAlertTriggered(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE cost_alerts
		SET last_triggered_at = $2, updated_at = $2
		WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("update cost alert triggered: %w", err)
	}
	return nil
}

// GetEnabledCostAlerts returns all enabled cost alerts for an organization.
func (db *DB) GetEnabledCostAlerts(ctx context.Context, orgID uuid.UUID) ([]*models.CostAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, monthly_threshold, enabled, notify_on_exceed,
		       notify_on_forecast, forecast_months, last_triggered_at, created_at, updated_at
		FROM cost_alerts
		WHERE org_id = $1 AND enabled = true
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list enabled cost alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.CostAlert
	for rows.Next() {
		var a models.CostAlert
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Name, &a.MonthlyThreshold, &a.Enabled,
			&a.NotifyOnExceed, &a.NotifyOnForecast, &a.ForecastMonths,
			&a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cost alert: %w", err)
		}
		alerts = append(alerts, &a)
	}

	return alerts, nil
}

// SSO Group Mapping methods

// GetSSOGroupMappingsByOrgID returns all SSO group mappings for an organization.
func (db *DB) GetSSOGroupMappingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE org_id = $1
		ORDER BY oidc_group_name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list SSO group mappings: %w", err)
	}
	defer rows.Close()

	var mappings []*models.SSOGroupMapping
	for rows.Next() {
		var m models.SSOGroupMapping
		var roleStr string
		err := rows.Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan SSO group mapping: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		mappings = append(mappings, &m)
	}

	return mappings, nil
}

// GetSSOGroupMappingsByGroupNames returns all SSO group mappings matching the given group names.
func (db *DB) GetSSOGroupMappingsByGroupNames(ctx context.Context, groupNames []string) ([]*models.SSOGroupMapping, error) {
	if len(groupNames) == 0 {
		return []*models.SSOGroupMapping{}, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE oidc_group_name = ANY($1)
		ORDER BY oidc_group_name
	`, groupNames)
	if err != nil {
		return nil, fmt.Errorf("get SSO group mappings by names: %w", err)
	}
	defer rows.Close()

	var mappings []*models.SSOGroupMapping
	for rows.Next() {
		var m models.SSOGroupMapping
		var roleStr string
		err := rows.Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan SSO group mapping: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		mappings = append(mappings, &m)
	}

	return mappings, nil
}

// GetSSOGroupMappingByID returns an SSO group mapping by ID.
func (db *DB) GetSSOGroupMappingByID(ctx context.Context, id uuid.UUID) (*models.SSOGroupMapping, error) {
	var m models.SSOGroupMapping
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE id = $1
	`, id).Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get SSO group mapping: %w", err)
	}
	m.Role = models.OrgRole(roleStr)
	return &m, nil
}

// CreateSSOGroupMapping creates a new SSO group mapping.
func (db *DB) CreateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sso_group_mappings (id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.OrgID, m.OIDCGroupName, string(m.Role), m.AutoCreateOrg, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create SSO group mapping: %w", err)
	}
	return nil
}

// UpdateSSOGroupMapping updates an existing SSO group mapping.
func (db *DB) UpdateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error {
	m.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE sso_group_mappings
		SET role = $2, auto_create_org = $3, updated_at = $4
		WHERE id = $1
	`, m.ID, string(m.Role), m.AutoCreateOrg, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update SSO group mapping: %w", err)
	}
	return nil
}

// DeleteSSOGroupMapping deletes an SSO group mapping.
func (db *DB) DeleteSSOGroupMapping(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sso_group_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete SSO group mapping: %w", err)
	}
	return nil
}

// User SSO Groups methods

// GetUserSSOGroups returns a user's SSO groups.
func (db *DB) GetUserSSOGroups(ctx context.Context, userID uuid.UUID) (*models.UserSSOGroups, error) {
	var u models.UserSSOGroups
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, oidc_groups, synced_at
		FROM user_sso_groups
		WHERE user_id = $1
	`, userID).Scan(&u.ID, &u.UserID, &u.OIDCGroups, &u.SyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get user SSO groups: %w", err)
	}
	return &u, nil
}

// UpsertUserSSOGroups creates or updates a user's SSO groups.
func (db *DB) UpsertUserSSOGroups(ctx context.Context, userID uuid.UUID, groups []string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_sso_groups (id, user_id, oidc_groups, synced_at)
		VALUES (gen_random_uuid(), $1, $2, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET oidc_groups = $2, synced_at = NOW()
	`, userID, groups)
	if err != nil {
		return fmt.Errorf("upsert user SSO groups: %w", err)
	}
	return nil
}

// Organization SSO Settings methods

// GetOrganizationSSOSettings returns an organization's SSO settings.
func (db *DB) GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT sso_default_role, sso_auto_create_orgs
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&defaultRole, &autoCreateOrgs)
	if err != nil {
		return nil, false, fmt.Errorf("get org SSO settings: %w", err)
	}
	return defaultRole, autoCreateOrgs, nil
}

// UpdateOrganizationSSOSettings updates an organization's SSO settings.
func (db *DB) UpdateOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID, defaultRole *string, autoCreateOrgs bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations
		SET sso_default_role = $2, sso_auto_create_orgs = $3, updated_at = NOW()
		WHERE id = $1
	`, orgID, defaultRole, autoCreateOrgs)
	if err != nil {
		return fmt.Errorf("update org SSO settings: %w", err)
	}
	return nil
}

// UpdateMembershipRole updates a membership's role.
func (db *DB) UpdateMembershipRole(ctx context.Context, membershipID uuid.UUID, role models.OrgRole) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_memberships
		SET role = $2, updated_at = NOW()
		WHERE id = $1
	`, membershipID, string(role))
	if err != nil {
		return fmt.Errorf("update membership role: %w", err)
	}
	return nil
}

// Agent Registration Code methods

// CreateRegistrationCode creates a new agent registration code.
func (db *DB) CreateRegistrationCode(ctx context.Context, code *models.RegistrationCode) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_registration_codes (id, org_id, created_by, code, hostname, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, code.ID, code.OrgID, code.CreatedBy, code.Code, code.Hostname, code.ExpiresAt, code.CreatedAt)
	if err != nil {
		return fmt.Errorf("create registration code: %w", err)
	}
	return nil
}

// GetRegistrationCodeByCode returns a registration code by its code value.
func (db *DB) GetRegistrationCodeByCode(ctx context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error) {
	var r models.RegistrationCode
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, created_by, code, hostname, expires_at, used_at, used_by_agent_id, created_at
		FROM agent_registration_codes
		WHERE org_id = $1 AND code = $2
	`, orgID, code).Scan(
		&r.ID, &r.OrgID, &r.CreatedBy, &r.Code, &r.Hostname,
		&r.ExpiresAt, &r.UsedAt, &r.UsedByAgentID, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get registration code: %w", err)
	}
	return &r, nil
}

// GetPendingRegistrationCodes returns all pending (unused, unexpired) registration codes for an organization.
func (db *DB) GetPendingRegistrationCodes(ctx context.Context, orgID uuid.UUID) ([]*models.RegistrationCode, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by, code, hostname, expires_at, used_at, used_by_agent_id, created_at
		FROM agent_registration_codes
		WHERE org_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list pending registration codes: %w", err)
	}
	defer rows.Close()

	var codes []*models.RegistrationCode
	for rows.Next() {
		var r models.RegistrationCode
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.CreatedBy, &r.Code, &r.Hostname,
			&r.ExpiresAt, &r.UsedAt, &r.UsedByAgentID, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan registration code: %w", err)
		}
		codes = append(codes, &r)
	}

	return codes, nil
}

// GetPendingRegistrationsWithCreator returns pending codes with creator email for display.
func (db *DB) GetPendingRegistrationsWithCreator(ctx context.Context, orgID uuid.UUID) ([]*models.PendingRegistration, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT rc.id, rc.hostname, rc.code, rc.expires_at, rc.created_at, u.email
		FROM agent_registration_codes rc
		JOIN users u ON rc.created_by = u.id
		WHERE rc.org_id = $1 AND rc.used_at IS NULL AND rc.expires_at > NOW()
		ORDER BY rc.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list pending registrations: %w", err)
	}
	defer rows.Close()

	var registrations []*models.PendingRegistration
	for rows.Next() {
		var r models.PendingRegistration
		err := rows.Scan(&r.ID, &r.Hostname, &r.Code, &r.ExpiresAt, &r.CreatedAt, &r.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("scan pending registration: %w", err)
		}
		registrations = append(registrations, &r)
	}

	return registrations, nil
}

// MarkRegistrationCodeUsed marks a registration code as used by an agent.
func (db *DB) MarkRegistrationCodeUsed(ctx context.Context, codeID, agentID uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE agent_registration_codes
		SET used_at = NOW(), used_by_agent_id = $2
		WHERE id = $1 AND used_at IS NULL
	`, codeID, agentID)
	if err != nil {
		return fmt.Errorf("mark registration code used: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("registration code not found or already used")
	}
	return nil
}

// DeleteExpiredRegistrationCodes removes expired registration codes.
func (db *DB) DeleteExpiredRegistrationCodes(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_registration_codes
		WHERE expires_at < NOW() AND used_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("delete expired registration codes: %w", err)
	}
	return nil
}

// DeleteRegistrationCode deletes a registration code by ID.
func (db *DB) DeleteRegistrationCode(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_registration_codes
		WHERE id = $1 AND used_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete registration code: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("registration code not found or already used")
	}
	return nil
}

// Imported Snapshot methods

// CreateImportedSnapshot creates a new imported snapshot record.
func (db *DB) CreateImportedSnapshot(ctx context.Context, snap *models.ImportedSnapshot) error {
	pathsJSON, err := json.Marshal(snap.Paths)
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	tagsJSON, err := json.Marshal(snap.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO imported_snapshots (
			id, repository_id, agent_id, restic_snapshot_id, short_id,
			hostname, username, snapshot_time, paths, tags, imported_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, snap.ID, snap.RepositoryID, snap.AgentID, snap.ResticSnapshotID, snap.ShortID,
		snap.Hostname, snap.Username, snap.SnapshotTime, pathsJSON, tagsJSON,
		snap.ImportedAt, snap.CreatedAt)
	if err != nil {
		return fmt.Errorf("create imported snapshot: %w", err)
	}
	return nil
}

// CreateImportedSnapshots creates multiple imported snapshot records in a batch.
func (db *DB) CreateImportedSnapshots(ctx context.Context, snapshots []*models.ImportedSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	for _, snap := range snapshots {
		if err := db.CreateImportedSnapshot(ctx, snap); err != nil {
			return err
		}
	}
	return nil
}

// GetImportedSnapshotsByRepositoryID returns all imported snapshots for a repository.
func (db *DB) GetImportedSnapshotsByRepositoryID(ctx context.Context, repositoryID uuid.UUID) ([]*models.ImportedSnapshot, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, agent_id, restic_snapshot_id, short_id,
		       hostname, username, snapshot_time, paths, tags, imported_at, created_at
		FROM imported_snapshots
		WHERE repository_id = $1
		ORDER BY snapshot_time DESC
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list imported snapshots: %w", err)
	}
	defer rows.Close()

	return scanImportedSnapshots(rows)
}

// GetImportedSnapshotsByAgentID returns all imported snapshots for an agent.
func (db *DB) GetImportedSnapshotsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.ImportedSnapshot, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, agent_id, restic_snapshot_id, short_id,
		       hostname, username, snapshot_time, paths, tags, imported_at, created_at
		FROM imported_snapshots
		WHERE agent_id = $1
		ORDER BY snapshot_time DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list imported snapshots by agent: %w", err)
	}
	defer rows.Close()

	return scanImportedSnapshots(rows)
}

// GetImportedSnapshotByID returns an imported snapshot by ID.
func (db *DB) GetImportedSnapshotByID(ctx context.Context, id uuid.UUID) (*models.ImportedSnapshot, error) {
	var snap models.ImportedSnapshot
	var pathsBytes, tagsBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, agent_id, restic_snapshot_id, short_id,
		       hostname, username, snapshot_time, paths, tags, imported_at, created_at
		FROM imported_snapshots
		WHERE id = $1
	`, id).Scan(
		&snap.ID, &snap.RepositoryID, &snap.AgentID, &snap.ResticSnapshotID,
		&snap.ShortID, &snap.Hostname, &snap.Username, &snap.SnapshotTime,
		&pathsBytes, &tagsBytes, &snap.ImportedAt, &snap.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get imported snapshot: %w", err)
	}

	if err := json.Unmarshal(pathsBytes, &snap.Paths); err != nil {
		return nil, fmt.Errorf("unmarshal paths: %w", err)
	}
	if err := json.Unmarshal(tagsBytes, &snap.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	return &snap, nil
}

// DeleteImportedSnapshotsByRepositoryID deletes all imported snapshots for a repository.
func (db *DB) DeleteImportedSnapshotsByRepositoryID(ctx context.Context, repositoryID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM imported_snapshots WHERE repository_id = $1`, repositoryID)
	if err != nil {
		return fmt.Errorf("delete imported snapshots: %w", err)
	}
	return nil
}

// UpdateImportedSnapshotAgent updates the agent association for imported snapshots.
func (db *DB) UpdateImportedSnapshotAgent(ctx context.Context, snapshotID uuid.UUID, agentID *uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE imported_snapshots
		SET agent_id = $2
		WHERE id = $1
	`, snapshotID, agentID)
	if err != nil {
		return fmt.Errorf("update imported snapshot agent: %w", err)
	}
	return nil
}

// MarkRepositoryAsImported marks a repository as imported.
func (db *DB) MarkRepositoryAsImported(ctx context.Context, repositoryID uuid.UUID, snapshotCount int) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE repositories
		SET imported = true, imported_at = NOW(), original_snapshot_count = $2, updated_at = NOW()
		WHERE id = $1
	`, repositoryID, snapshotCount)
	if err != nil {
		return fmt.Errorf("mark repository as imported: %w", err)
	}
	return nil
}

// GetImportedSnapshotsByHostname returns imported snapshots for a repository filtered by hostname.
func (db *DB) GetImportedSnapshotsByHostname(ctx context.Context, repositoryID uuid.UUID, hostname string) ([]*models.ImportedSnapshot, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, agent_id, restic_snapshot_id, short_id,
		       hostname, username, snapshot_time, paths, tags, imported_at, created_at
		FROM imported_snapshots
		WHERE repository_id = $1 AND hostname = $2
		ORDER BY snapshot_time DESC
	`, repositoryID, hostname)
	if err != nil {
		return nil, fmt.Errorf("list imported snapshots by hostname: %w", err)
	}
	defer rows.Close()

	return scanImportedSnapshots(rows)
}

// CountImportedSnapshotsByRepositoryID returns the count of imported snapshots for a repository.
func (db *DB) CountImportedSnapshotsByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM imported_snapshots WHERE repository_id = $1
	`, repositoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count imported snapshots: %w", err)
	}
	return count, nil
}

// scanImportedSnapshots scans rows into imported snapshots.
func scanImportedSnapshots(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.ImportedSnapshot, error) {
	var snapshots []*models.ImportedSnapshot
	for rows.Next() {
		var snap models.ImportedSnapshot
		var pathsBytes, tagsBytes []byte
		err := rows.Scan(
			&snap.ID, &snap.RepositoryID, &snap.AgentID, &snap.ResticSnapshotID,
			&snap.ShortID, &snap.Hostname, &snap.Username, &snap.SnapshotTime,
			&pathsBytes, &tagsBytes, &snap.ImportedAt, &snap.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan imported snapshot: %w", err)
		}

		if err := json.Unmarshal(pathsBytes, &snap.Paths); err != nil {
			return nil, fmt.Errorf("unmarshal paths: %w", err)
		}
		if err := json.Unmarshal(tagsBytes, &snap.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		snapshots = append(snapshots, &snap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate imported snapshots: %w", err)
	}

	return snapshots, nil
}

// CreateAgentLogs inserts multiple log entries for an agent.
func (db *DB) CreateAgentLogs(ctx context.Context, logs []*models.AgentLog) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, log := range logs {
		metadataBytes, err := log.MetadataJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		batch.Queue(`
			INSERT INTO agent_logs (id, agent_id, org_id, level, message, component, metadata, timestamp, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, log.ID, log.AgentID, log.OrgID, string(log.Level), log.Message, log.Component, metadataBytes, log.Timestamp, log.CreatedAt)
	}

	results := db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for range logs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("create agent log: %w", err)
		}
	}

	return nil
}

// GetAgentLogs returns logs for an agent with optional filtering.
func (db *DB) GetAgentLogs(ctx context.Context, agentID uuid.UUID, filter *models.AgentLogFilter) ([]*models.AgentLog, int, error) {
	if filter == nil {
		filter = &models.AgentLogFilter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	// Build the query with optional filters
	var args []interface{}
	args = append(args, agentID)
	argIndex := 2

	whereClause := "WHERE agent_id = $1"

	if filter.Level != "" {
		whereClause += fmt.Sprintf(" AND level = $%d", argIndex)
		args = append(args, string(filter.Level))
		argIndex++
	}

	if filter.Component != "" {
		whereClause += fmt.Sprintf(" AND component = $%d", argIndex)
		args = append(args, filter.Component)
		argIndex++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND to_tsvector('english', message) @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, filter.Search)
		argIndex++
	}

	if !filter.Since.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filter.Since)
		argIndex++
	}

	if !filter.Until.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filter.Until)
		argIndex++
	}

	// Count total matching logs
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_logs %s", whereClause)
	var totalCount int
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count agent logs: %w", err)
	}

	// Fetch logs with pagination
	args = append(args, filter.Limit, filter.Offset)
	query := fmt.Sprintf(`
		SELECT id, agent_id, org_id, level, message, component, metadata, timestamp, created_at
		FROM agent_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("get agent logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AgentLog
	for rows.Next() {
		var log models.AgentLog
		var levelStr string
		var metadataBytes []byte
		err := rows.Scan(
			&log.ID, &log.AgentID, &log.OrgID, &levelStr,
			&log.Message, &log.Component, &metadataBytes,
			&log.Timestamp, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan agent log: %w", err)
		}
		log.Level = models.LogLevel(levelStr)
		if len(metadataBytes) > 0 {
			if err := log.SetMetadata(metadataBytes); err != nil {
				db.logger.Warn().Err(err).Str("log_id", log.ID.String()).Msg("failed to parse log metadata")
			}
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate agent logs: %w", err)
	}

	return logs, totalCount, nil
}

// DeleteAgentLogsBefore deletes agent logs older than the given timestamp.
func (db *DB) DeleteAgentLogsBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_logs WHERE timestamp < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("delete agent logs: %w", err)
	}
	return result.RowsAffected(), nil
}

// Backup Checkpoint methods

// CreateCheckpoint creates a new backup checkpoint.
func (db *DB) CreateCheckpoint(ctx context.Context, checkpoint *models.BackupCheckpoint) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO backup_checkpoints (id, schedule_id, agent_id, repository_id, backup_id,
		                                status, files_processed, bytes_processed, total_files, total_bytes,
		                                last_processed_path, restic_state, error_message, resume_count,
		                                expires_at, started_at, last_updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, checkpoint.ID, checkpoint.ScheduleID, checkpoint.AgentID, checkpoint.RepositoryID, checkpoint.BackupID,
		string(checkpoint.Status), checkpoint.FilesProcessed, checkpoint.BytesProcessed,
		checkpoint.TotalFiles, checkpoint.TotalBytes, checkpoint.LastProcessedPath,
		checkpoint.ResticState, checkpoint.ErrorMessage, checkpoint.ResumeCount,
		checkpoint.ExpiresAt, checkpoint.StartedAt, checkpoint.LastUpdatedAt, checkpoint.CreatedAt)
	if err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	return nil
}

// UpdateCheckpoint updates an existing checkpoint.
func (db *DB) UpdateCheckpoint(ctx context.Context, checkpoint *models.BackupCheckpoint) error {
	checkpoint.LastUpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE backup_checkpoints
		SET backup_id = $2, status = $3, files_processed = $4, bytes_processed = $5,
		    total_files = $6, total_bytes = $7, last_processed_path = $8, restic_state = $9,
		    error_message = $10, resume_count = $11, expires_at = $12, last_updated_at = $13
		WHERE id = $1
	`, checkpoint.ID, checkpoint.BackupID, string(checkpoint.Status),
		checkpoint.FilesProcessed, checkpoint.BytesProcessed, checkpoint.TotalFiles,
		checkpoint.TotalBytes, checkpoint.LastProcessedPath, checkpoint.ResticState,
		checkpoint.ErrorMessage, checkpoint.ResumeCount, checkpoint.ExpiresAt, checkpoint.LastUpdatedAt)
	if err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}
	return nil
}

// GetCheckpointByID returns a checkpoint by ID.
func (db *DB) GetCheckpointByID(ctx context.Context, id uuid.UUID) (*models.BackupCheckpoint, error) {
	var c models.BackupCheckpoint
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, backup_id, status,
		       files_processed, bytes_processed, total_files, total_bytes,
		       last_processed_path, restic_state, error_message, resume_count,
		       expires_at, started_at, last_updated_at, created_at
		FROM backup_checkpoints
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.ScheduleID, &c.AgentID, &c.RepositoryID, &c.BackupID, &statusStr,
		&c.FilesProcessed, &c.BytesProcessed, &c.TotalFiles, &c.TotalBytes,
		&c.LastProcessedPath, &c.ResticState, &c.ErrorMessage, &c.ResumeCount,
		&c.ExpiresAt, &c.StartedAt, &c.LastUpdatedAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get checkpoint: %w", err)
	}
	c.Status = models.CheckpointStatus(statusStr)
	return &c, nil
}

// GetActiveCheckpointForSchedule returns the active checkpoint for a schedule if one exists.
func (db *DB) GetActiveCheckpointForSchedule(ctx context.Context, scheduleID uuid.UUID) (*models.BackupCheckpoint, error) {
	var c models.BackupCheckpoint
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, backup_id, status,
		       files_processed, bytes_processed, total_files, total_bytes,
		       last_processed_path, restic_state, error_message, resume_count,
		       expires_at, started_at, last_updated_at, created_at
		FROM backup_checkpoints
		WHERE schedule_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`, scheduleID).Scan(
		&c.ID, &c.ScheduleID, &c.AgentID, &c.RepositoryID, &c.BackupID, &statusStr,
		&c.FilesProcessed, &c.BytesProcessed, &c.TotalFiles, &c.TotalBytes,
		&c.LastProcessedPath, &c.ResticState, &c.ErrorMessage, &c.ResumeCount,
		&c.ExpiresAt, &c.StartedAt, &c.LastUpdatedAt, &c.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active checkpoint for schedule: %w", err)
	}
	c.Status = models.CheckpointStatus(statusStr)
	return &c, nil
}

// GetActiveCheckpointsForAgent returns all active checkpoints for an agent.
func (db *DB) GetActiveCheckpointsForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupCheckpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, backup_id, status,
		       files_processed, bytes_processed, total_files, total_bytes,
		       last_processed_path, restic_state, error_message, resume_count,
		       expires_at, started_at, last_updated_at, created_at
		FROM backup_checkpoints
		WHERE agent_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list active checkpoints for agent: %w", err)
	}
	defer rows.Close()

	return scanCheckpoints(rows)
}

// GetExpiredCheckpoints returns all checkpoints that have expired.
func (db *DB) GetExpiredCheckpoints(ctx context.Context) ([]*models.BackupCheckpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, backup_id, status,
		       files_processed, bytes_processed, total_files, total_bytes,
		       last_processed_path, restic_state, error_message, resume_count,
		       expires_at, started_at, last_updated_at, created_at
		FROM backup_checkpoints
		WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < NOW()
	`)
	if err != nil {
		return nil, fmt.Errorf("list expired checkpoints: %w", err)
	}
	defer rows.Close()

	return scanCheckpoints(rows)
}

// DeleteCheckpoint deletes a checkpoint by ID.
func (db *DB) DeleteCheckpoint(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM backup_checkpoints WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete checkpoint: %w", err)
	}
	return nil
}

// GetCheckpointsByScheduleID returns all checkpoints for a schedule.
func (db *DB) GetCheckpointsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupCheckpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, backup_id, status,
		       files_processed, bytes_processed, total_files, total_bytes,
		       last_processed_path, restic_state, error_message, resume_count,
		       expires_at, started_at, last_updated_at, created_at
		FROM backup_checkpoints
		WHERE schedule_id = $1
		ORDER BY created_at DESC
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list checkpoints for schedule: %w", err)
	}
	defer rows.Close()

	return scanCheckpoints(rows)
}

// scanCheckpoints scans rows into backup checkpoints.
func scanCheckpoints(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.BackupCheckpoint, error) {
	var checkpoints []*models.BackupCheckpoint
	for rows.Next() {
		var c models.BackupCheckpoint
		var statusStr string
		err := rows.Scan(
			&c.ID, &c.ScheduleID, &c.AgentID, &c.RepositoryID, &c.BackupID, &statusStr,
			&c.FilesProcessed, &c.BytesProcessed, &c.TotalFiles, &c.TotalBytes,
			&c.LastProcessedPath, &c.ResticState, &c.ErrorMessage, &c.ResumeCount,
			&c.ExpiresAt, &c.StartedAt, &c.LastUpdatedAt, &c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan checkpoint: %w", err)
		}
		c.Status = models.CheckpointStatus(statusStr)
		checkpoints = append(checkpoints, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkpoints: %w", err)
	}

	return checkpoints, nil
}

// ============================================================================
// Ransomware Detection Methods
// ============================================================================

// GetRansomwareSettingsByScheduleID returns ransomware settings for a schedule.
func (db *DB) GetRansomwareSettingsByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.RansomwareSettings, error) {
	var settings models.RansomwareSettings
	var extensionsBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, enabled, change_threshold_percent,
		       extensions_to_detect, entropy_detection_enabled, entropy_threshold,
		       auto_pause_on_alert, created_at, updated_at
		FROM ransomware_settings
		WHERE schedule_id = $1
	`, scheduleID).Scan(
		&settings.ID, &settings.ScheduleID, &settings.Enabled,
		&settings.ChangeThresholdPercent, &extensionsBytes,
		&settings.EntropyDetectionEnabled, &settings.EntropyThreshold,
		&settings.AutoPauseOnAlert, &settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ransomware settings: %w", err)
	}

	if err := settings.SetExtensions(extensionsBytes); err != nil {
		return nil, fmt.Errorf("unmarshal extensions: %w", err)
	}

	return &settings, nil
}

// GetRansomwareSettingsByOrgID returns all ransomware settings for an organization.
func (db *DB) GetRansomwareSettingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareSettings, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT rs.id, rs.schedule_id, rs.enabled, rs.change_threshold_percent,
		       rs.extensions_to_detect, rs.entropy_detection_enabled, rs.entropy_threshold,
		       rs.auto_pause_on_alert, rs.created_at, rs.updated_at
		FROM ransomware_settings rs
		JOIN schedules s ON rs.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
		ORDER BY rs.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list ransomware settings: %w", err)
	}
	defer rows.Close()

	return db.scanRansomwareSettings(rows)
}

// CreateRansomwareSettings creates new ransomware settings for a schedule.
func (db *DB) CreateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error {
	extensionsBytes, err := settings.ExtensionsJSON()
	if err != nil {
		return fmt.Errorf("marshal extensions: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO ransomware_settings (id, schedule_id, enabled, change_threshold_percent,
		                                  extensions_to_detect, entropy_detection_enabled,
		                                  entropy_threshold, auto_pause_on_alert,
		                                  created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, settings.ID, settings.ScheduleID, settings.Enabled, settings.ChangeThresholdPercent,
		extensionsBytes, settings.EntropyDetectionEnabled, settings.EntropyThreshold,
		settings.AutoPauseOnAlert, settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create ransomware settings: %w", err)
	}
	return nil
}

// UpdateRansomwareSettings updates existing ransomware settings.
func (db *DB) UpdateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error {
	extensionsBytes, err := settings.ExtensionsJSON()
	if err != nil {
		return fmt.Errorf("marshal extensions: %w", err)
	}

	settings.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE ransomware_settings
		SET enabled = $2, change_threshold_percent = $3, extensions_to_detect = $4,
		    entropy_detection_enabled = $5, entropy_threshold = $6,
		    auto_pause_on_alert = $7, updated_at = $8
		WHERE id = $1
	`, settings.ID, settings.Enabled, settings.ChangeThresholdPercent,
		extensionsBytes, settings.EntropyDetectionEnabled, settings.EntropyThreshold,
		settings.AutoPauseOnAlert, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update ransomware settings: %w", err)
	}
	return nil
}

// DeleteRansomwareSettings deletes ransomware settings by ID.
func (db *DB) DeleteRansomwareSettings(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ransomware_settings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ransomware settings: %w", err)
	}
	return nil
}

// scanRansomwareSettings scans rows into ransomware settings.
func (db *DB) scanRansomwareSettings(rows pgx.Rows) ([]*models.RansomwareSettings, error) {
	var settingsList []*models.RansomwareSettings
	for rows.Next() {
		var settings models.RansomwareSettings
		var extensionsBytes []byte

		err := rows.Scan(
			&settings.ID, &settings.ScheduleID, &settings.Enabled,
			&settings.ChangeThresholdPercent, &extensionsBytes,
			&settings.EntropyDetectionEnabled, &settings.EntropyThreshold,
			&settings.AutoPauseOnAlert, &settings.CreatedAt, &settings.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ransomware settings: %w", err)
		}

		if err := settings.SetExtensions(extensionsBytes); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}

		settingsList = append(settingsList, &settings)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ransomware settings: %w", err)
	}

	return settingsList, nil
}

// ============================================================================
// Ransomware Alert Methods
// ============================================================================

// CreateRansomwareAlert creates a new ransomware alert.
func (db *DB) CreateRansomwareAlert(ctx context.Context, alert *models.RansomwareAlert) error {
	indicatorsBytes, err := alert.IndicatorsJSON()
	if err != nil {
		return fmt.Errorf("marshal indicators: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO ransomware_alerts (id, org_id, schedule_id, agent_id, backup_id,
		                               schedule_name, agent_hostname, status, risk_score,
		                               indicators, files_changed, files_new, total_files,
		                               backups_paused, paused_at, resumed_at, resolved_by,
		                               resolved_at, resolution, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, alert.ID, alert.OrgID, alert.ScheduleID, alert.AgentID, alert.BackupID,
		alert.ScheduleName, alert.AgentHostname, string(alert.Status), alert.RiskScore,
		indicatorsBytes, alert.FilesChanged, alert.FilesNew, alert.TotalFiles,
		alert.BackupsPaused, alert.PausedAt, alert.ResumedAt, alert.ResolvedBy,
		alert.ResolvedAt, alert.Resolution, alert.CreatedAt, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create ransomware alert: %w", err)
	}
	return nil
}

// GetRansomwareAlertsByOrgID returns all ransomware alerts for an organization.
func (db *DB) GetRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, schedule_id, agent_id, backup_id, schedule_name, agent_hostname,
		       status, risk_score, indicators, files_changed, files_new, total_files,
		       backups_paused, paused_at, resumed_at, resolved_by, resolved_at, resolution,
		       created_at, updated_at
		FROM ransomware_alerts
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list ransomware alerts: %w", err)
	}
	defer rows.Close()

	return db.scanRansomwareAlerts(rows)
}

// GetActiveRansomwareAlertsByOrgID returns active ransomware alerts for an organization.
func (db *DB) GetActiveRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, schedule_id, agent_id, backup_id, schedule_name, agent_hostname,
		       status, risk_score, indicators, files_changed, files_new, total_files,
		       backups_paused, paused_at, resumed_at, resolved_by, resolved_at, resolution,
		       created_at, updated_at
		FROM ransomware_alerts
		WHERE org_id = $1 AND status IN ('active', 'investigating')
		ORDER BY risk_score DESC, created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active ransomware alerts: %w", err)
	}
	defer rows.Close()

	return db.scanRansomwareAlerts(rows)
}

// GetActiveRansomwareAlertCountByOrgID returns the count of active ransomware alerts.
func (db *DB) GetActiveRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ransomware_alerts
		WHERE org_id = $1 AND status IN ('active', 'investigating')
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active ransomware alerts: %w", err)
	}
	return count, nil
}

// GetCriticalRansomwareAlertCountByOrgID returns the count of critical ransomware alerts (risk score >= 80).
func (db *DB) GetCriticalRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ransomware_alerts
		WHERE org_id = $1 AND status IN ('active', 'investigating') AND risk_score >= 80
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count critical ransomware alerts: %w", err)
	}
	return count, nil
}

// GetRansomwareAlertByID returns a ransomware alert by ID.
func (db *DB) GetRansomwareAlertByID(ctx context.Context, id uuid.UUID) (*models.RansomwareAlert, error) {
	var alert models.RansomwareAlert
	var indicatorsBytes []byte
	var status string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, schedule_id, agent_id, backup_id, schedule_name, agent_hostname,
		       status, risk_score, indicators, files_changed, files_new, total_files,
		       backups_paused, paused_at, resumed_at, resolved_by, resolved_at, resolution,
		       created_at, updated_at
		FROM ransomware_alerts
		WHERE id = $1
	`, id).Scan(
		&alert.ID, &alert.OrgID, &alert.ScheduleID, &alert.AgentID, &alert.BackupID,
		&alert.ScheduleName, &alert.AgentHostname, &status, &alert.RiskScore,
		&indicatorsBytes, &alert.FilesChanged, &alert.FilesNew, &alert.TotalFiles,
		&alert.BackupsPaused, &alert.PausedAt, &alert.ResumedAt, &alert.ResolvedBy,
		&alert.ResolvedAt, &alert.Resolution, &alert.CreatedAt, &alert.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ransomware alert: %w", err)
	}

	alert.Status = models.RansomwareAlertStatus(status)
	if err := alert.SetIndicatorsFromBytes(indicatorsBytes); err != nil {
		return nil, fmt.Errorf("unmarshal indicators: %w", err)
	}

	return &alert, nil
}

// UpdateRansomwareAlert updates an existing ransomware alert.
func (db *DB) UpdateRansomwareAlert(ctx context.Context, alert *models.RansomwareAlert) error {
	indicatorsBytes, err := alert.IndicatorsJSON()
	if err != nil {
		return fmt.Errorf("marshal indicators: %w", err)
	}

	alert.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE ransomware_alerts
		SET status = $2, risk_score = $3, indicators = $4, files_changed = $5,
		    files_new = $6, total_files = $7, backups_paused = $8, paused_at = $9,
		    resumed_at = $10, resolved_by = $11, resolved_at = $12, resolution = $13,
		    updated_at = $14
		WHERE id = $1
	`, alert.ID, string(alert.Status), alert.RiskScore, indicatorsBytes,
		alert.FilesChanged, alert.FilesNew, alert.TotalFiles, alert.BackupsPaused,
		alert.PausedAt, alert.ResumedAt, alert.ResolvedBy, alert.ResolvedAt,
		alert.Resolution, alert.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update ransomware alert: %w", err)
	}
	return nil
}

// PauseSchedule pauses a schedule by setting enabled to false.
func (db *DB) PauseSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE schedules SET enabled = false, updated_at = NOW() WHERE id = $1
	`, scheduleID)
	if err != nil {
		return fmt.Errorf("pause schedule: %w", err)
	}
	return nil
}

// ResumeSchedule resumes a paused schedule by setting enabled to true.
func (db *DB) ResumeSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE schedules SET enabled = true, updated_at = NOW() WHERE id = $1
	`, scheduleID)
	if err != nil {
		return fmt.Errorf("resume schedule: %w", err)
	}
	return nil
}

// scanRansomwareAlerts scans rows into ransomware alerts.
func (db *DB) scanRansomwareAlerts(rows pgx.Rows) ([]*models.RansomwareAlert, error) {
	var alerts []*models.RansomwareAlert
	for rows.Next() {
		var alert models.RansomwareAlert
		var indicatorsBytes []byte
		var status string

		err := rows.Scan(
			&alert.ID, &alert.OrgID, &alert.ScheduleID, &alert.AgentID, &alert.BackupID,
			&alert.ScheduleName, &alert.AgentHostname, &status, &alert.RiskScore,
			&indicatorsBytes, &alert.FilesChanged, &alert.FilesNew, &alert.TotalFiles,
			&alert.BackupsPaused, &alert.PausedAt, &alert.ResumedAt, &alert.ResolvedBy,
			&alert.ResolvedAt, &alert.Resolution, &alert.CreatedAt, &alert.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ransomware alert: %w", err)
		}

		alert.Status = models.RansomwareAlertStatus(status)
		if err := alert.SetIndicatorsFromBytes(indicatorsBytes); err != nil {
			return nil, fmt.Errorf("unmarshal indicators: %w", err)
		}

		alerts = append(alerts, &alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ransomware alerts: %w", err)
	}

	return alerts, nil
}

// ============================================================================
// Snapshot Immutability Methods
// ============================================================================

// CreateSnapshotImmutability creates a new immutability lock for a snapshot.
func (db *DB) CreateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_immutability (
			id, org_id, repository_id, snapshot_id, short_id,
			locked_at, locked_until, locked_by, reason,
			s3_object_lock_enabled, s3_object_lock_mode,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (repository_id, snapshot_id) DO UPDATE SET
			locked_until = EXCLUDED.locked_until,
			reason = EXCLUDED.reason,
			updated_at = NOW()
	`, lock.ID, lock.OrgID, lock.RepositoryID, lock.SnapshotID, lock.ShortID,
		lock.LockedAt, lock.LockedUntil, lock.LockedBy, lock.Reason,
		lock.S3ObjectLockEnabled, lock.S3ObjectLockMode,
		lock.CreatedAt, lock.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot immutability: %w", err)
	}
	return nil
}

// GetSnapshotImmutability returns the immutability lock for a snapshot.
func (db *DB) GetSnapshotImmutability(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (*models.SnapshotImmutability, error) {
	var lock models.SnapshotImmutability
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, repository_id, snapshot_id, short_id,
		       locked_at, locked_until, locked_by, reason,
		       s3_object_lock_enabled, s3_object_lock_mode,
		       created_at, updated_at
		FROM snapshot_immutability
		WHERE repository_id = $1 AND snapshot_id = $2
	`, repositoryID, snapshotID).Scan(
		&lock.ID, &lock.OrgID, &lock.RepositoryID, &lock.SnapshotID, &lock.ShortID,
		&lock.LockedAt, &lock.LockedUntil, &lock.LockedBy, &lock.Reason,
		&lock.S3ObjectLockEnabled, &lock.S3ObjectLockMode,
		&lock.CreatedAt, &lock.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get snapshot immutability: %w", err)
	}
	return &lock, nil
}

// GetSnapshotImmutabilityByID returns an immutability lock by ID.
func (db *DB) GetSnapshotImmutabilityByID(ctx context.Context, id uuid.UUID) (*models.SnapshotImmutability, error) {
	var lock models.SnapshotImmutability
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, repository_id, snapshot_id, short_id,
		       locked_at, locked_until, locked_by, reason,
		       s3_object_lock_enabled, s3_object_lock_mode,
		       created_at, updated_at
		FROM snapshot_immutability
		WHERE id = $1
	`, id).Scan(
		&lock.ID, &lock.OrgID, &lock.RepositoryID, &lock.SnapshotID, &lock.ShortID,
		&lock.LockedAt, &lock.LockedUntil, &lock.LockedBy, &lock.Reason,
		&lock.S3ObjectLockEnabled, &lock.S3ObjectLockMode,
		&lock.CreatedAt, &lock.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get snapshot immutability by id: %w", err)
	}
	return &lock, nil
}

// UpdateSnapshotImmutability updates an existing immutability lock.
func (db *DB) UpdateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_immutability
		SET locked_until = $2, reason = $3, updated_at = NOW()
		WHERE id = $1
	`, lock.ID, lock.LockedUntil, lock.Reason)
	if err != nil {
		return fmt.Errorf("update snapshot immutability: %w", err)
	}
	return nil
}

// DeleteExpiredImmutabilityLocks removes expired locks and returns the count deleted.
func (db *DB) DeleteExpiredImmutabilityLocks(ctx context.Context) (int, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM snapshot_immutability
		WHERE locked_until < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("delete expired immutability locks: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// GetActiveImmutabilityLocksByRepositoryID returns all active locks for a repository.
func (db *DB) GetActiveImmutabilityLocksByRepositoryID(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotImmutability, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, snapshot_id, short_id,
		       locked_at, locked_until, locked_by, reason,
		       s3_object_lock_enabled, s3_object_lock_mode,
		       created_at, updated_at
		FROM snapshot_immutability
		WHERE repository_id = $1 AND locked_until > NOW()
		ORDER BY locked_until DESC
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list active immutability locks: %w", err)
	}
	defer rows.Close()

	return scanSnapshotImmutabilityRows(rows)
}

// GetActiveImmutabilityLocksByOrgID returns all active locks for an organization.
func (db *DB) GetActiveImmutabilityLocksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotImmutability, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, snapshot_id, short_id,
		       locked_at, locked_until, locked_by, reason,
		       s3_object_lock_enabled, s3_object_lock_mode,
		       created_at, updated_at
		FROM snapshot_immutability
		WHERE org_id = $1 AND locked_until > NOW()
		ORDER BY locked_until DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active immutability locks by org: %w", err)
	}
	defer rows.Close()

	return scanSnapshotImmutabilityRows(rows)
}

// IsSnapshotLocked checks if a snapshot has an active immutability lock.
func (db *DB) IsSnapshotLocked(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM snapshot_immutability
		WHERE repository_id = $1 AND snapshot_id = $2 AND locked_until > NOW()
	`, repositoryID, snapshotID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check snapshot locked: %w", err)
	}
	return count > 0, nil
}

// GetImmutabilityLocksBySnapshotIDs returns locks for multiple snapshot IDs (for batch lookup).
func (db *DB) GetImmutabilityLocksBySnapshotIDs(ctx context.Context, repositoryID uuid.UUID, snapshotIDs []string) (map[string]*models.SnapshotImmutability, error) {
	if len(snapshotIDs) == 0 {
		return make(map[string]*models.SnapshotImmutability), nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, snapshot_id, short_id,
		       locked_at, locked_until, locked_by, reason,
		       s3_object_lock_enabled, s3_object_lock_mode,
		       created_at, updated_at
		FROM snapshot_immutability
		WHERE repository_id = $1 AND snapshot_id = ANY($2) AND locked_until > NOW()
	`, repositoryID, snapshotIDs)
	if err != nil {
		return nil, fmt.Errorf("get immutability locks by snapshot ids: %w", err)
	}
	defer rows.Close()

	locks, err := scanSnapshotImmutabilityRows(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*models.SnapshotImmutability)
	for _, lock := range locks {
		result[lock.SnapshotID] = lock
	}
	return result, nil
}

// GetRepositoryImmutabilitySettings returns the immutability settings for a repository.
func (db *DB) GetRepositoryImmutabilitySettings(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryImmutabilitySettings, error) {
	var settings models.RepositoryImmutabilitySettings
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(immutability_enabled, false), default_immutability_days
		FROM repositories
		WHERE id = $1
	`, repositoryID).Scan(&settings.Enabled, &settings.DefaultDays)
	if err != nil {
		return nil, fmt.Errorf("get repository immutability settings: %w", err)
	}
	return &settings, nil
}

// UpdateRepositoryImmutabilitySettings updates the immutability settings for a repository.
func (db *DB) UpdateRepositoryImmutabilitySettings(ctx context.Context, repositoryID uuid.UUID, settings *models.RepositoryImmutabilitySettings) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE repositories
		SET immutability_enabled = $2, default_immutability_days = $3, updated_at = NOW()
		WHERE id = $1
	`, repositoryID, settings.Enabled, settings.DefaultDays)
	if err != nil {
		return fmt.Errorf("update repository immutability settings: %w", err)
	}
	return nil
}

// scanSnapshotImmutabilityRows scans rows into snapshot immutability locks.
func scanSnapshotImmutabilityRows(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.SnapshotImmutability, error) {
	var locks []*models.SnapshotImmutability
	for rows.Next() {
		var lock models.SnapshotImmutability
		err := rows.Scan(
			&lock.ID, &lock.OrgID, &lock.RepositoryID, &lock.SnapshotID, &lock.ShortID,
			&lock.LockedAt, &lock.LockedUntil, &lock.LockedBy, &lock.Reason,
			&lock.S3ObjectLockEnabled, &lock.S3ObjectLockMode,
			&lock.CreatedAt, &lock.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot immutability: %w", err)
		}
		locks = append(locks, &lock)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot immutability: %w", err)
	}

	return locks, nil
}

// Legal Hold methods

// CreateLegalHold creates a new legal hold on a snapshot.
func (db *DB) CreateLegalHold(ctx context.Context, hold *models.LegalHold) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO legal_holds (id, org_id, snapshot_id, reason, placed_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, hold.ID, hold.OrgID, hold.SnapshotID, hold.Reason, hold.PlacedBy, hold.CreatedAt, hold.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create legal hold: %w", err)
	}
	return nil
}

// GetLegalHoldByID returns a legal hold by ID.
func (db *DB) GetLegalHoldByID(ctx context.Context, id uuid.UUID) (*models.LegalHold, error) {
	var h models.LegalHold
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_id, reason, placed_by, created_at, updated_at
		FROM legal_holds
		WHERE id = $1
	`, id).Scan(&h.ID, &h.OrgID, &h.SnapshotID, &h.Reason, &h.PlacedBy, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get legal hold: %w", err)
	}
	return &h, nil
}

// GetLegalHoldBySnapshotID returns the legal hold for a specific snapshot within an organization.
func (db *DB) GetLegalHoldBySnapshotID(ctx context.Context, snapshotID string, orgID uuid.UUID) (*models.LegalHold, error) {
	var h models.LegalHold
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_id, reason, placed_by, created_at, updated_at
		FROM legal_holds
		WHERE snapshot_id = $1 AND org_id = $2
	`, snapshotID, orgID).Scan(&h.ID, &h.OrgID, &h.SnapshotID, &h.Reason, &h.PlacedBy, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get legal hold by snapshot: %w", err)
	}
	return &h, nil
}

// GetLegalHoldsByOrgID returns all legal holds for an organization.
func (db *DB) GetLegalHoldsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LegalHold, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, snapshot_id, reason, placed_by, created_at, updated_at
		FROM legal_holds
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list legal holds: %w", err)
	}
	defer rows.Close()

	var holds []*models.LegalHold
	for rows.Next() {
		var h models.LegalHold
		err := rows.Scan(&h.ID, &h.OrgID, &h.SnapshotID, &h.Reason, &h.PlacedBy, &h.CreatedAt, &h.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan legal hold: %w", err)
		}
		holds = append(holds, &h)
	}

	return holds, nil
}

// DeleteLegalHold removes a legal hold by ID.
func (db *DB) DeleteLegalHold(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM legal_holds WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete legal hold: %w", err)
	}
	return nil
}

// IsSnapshotOnHold checks if a snapshot has a legal hold.
func (db *DB) IsSnapshotOnHold(ctx context.Context, snapshotID string, orgID uuid.UUID) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM legal_holds WHERE snapshot_id = $1 AND org_id = $2
	`, snapshotID, orgID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check legal hold: %w", err)
	}
	return count > 0, nil
}

// GetSnapshotHoldStatus returns a map of snapshot IDs to their hold status for the given snapshots.
func (db *DB) GetSnapshotHoldStatus(ctx context.Context, snapshotIDs []string, orgID uuid.UUID) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(snapshotIDs) == 0 {
		return result, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT snapshot_id FROM legal_holds
		WHERE snapshot_id = ANY($1) AND org_id = $2
	`, snapshotIDs, orgID)
	if err != nil {
		return nil, fmt.Errorf("get snapshot hold status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var snapshotID string
		if err := rows.Scan(&snapshotID); err != nil {
			return nil, fmt.Errorf("scan snapshot hold: %w", err)
		}
		result[snapshotID] = true
	}

	return result, nil
}

// Geo-Replication methods

// CreateGeoReplicationConfig creates a new geo-replication configuration.
func (db *DB) CreateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO geo_replication_configs (
			id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, config.ID, config.OrgID, config.SourceRepositoryID, config.TargetRepositoryID,
		config.SourceRegion, config.TargetRegion, config.Enabled, config.Status,
		config.LastSnapshotID, config.LastSyncAt, config.LastError,
		config.MaxLagSnapshots, config.MaxLagDurationHours, config.AlertOnLag,
		config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create geo-replication config: %w", err)
	}
	return nil
}

// GetGeoReplicationConfig returns a geo-replication configuration by ID.
func (db *DB) GetGeoReplicationConfig(ctx context.Context, id uuid.UUID) (*models.GeoReplicationConfig, error) {
	var config models.GeoReplicationConfig
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		FROM geo_replication_configs
		WHERE id = $1
	`, id).Scan(
		&config.ID, &config.OrgID, &config.SourceRepositoryID, &config.TargetRepositoryID,
		&config.SourceRegion, &config.TargetRegion, &config.Enabled, &config.Status,
		&config.LastSnapshotID, &config.LastSyncAt, &config.LastError,
		&config.MaxLagSnapshots, &config.MaxLagDurationHours, &config.AlertOnLag,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get geo-replication config: %w", err)
	}
	return &config, nil
}

// GetGeoReplicationConfigByRepository returns the geo-replication config for a source repository.
func (db *DB) GetGeoReplicationConfigByRepository(ctx context.Context, repositoryID uuid.UUID) (*models.GeoReplicationConfig, error) {
	var config models.GeoReplicationConfig
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		FROM geo_replication_configs
		WHERE source_repository_id = $1
	`, repositoryID).Scan(
		&config.ID, &config.OrgID, &config.SourceRepositoryID, &config.TargetRepositoryID,
		&config.SourceRegion, &config.TargetRegion, &config.Enabled, &config.Status,
		&config.LastSnapshotID, &config.LastSyncAt, &config.LastError,
		&config.MaxLagSnapshots, &config.MaxLagDurationHours, &config.AlertOnLag,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get geo-replication config by repository: %w", err)
	}
	return &config, nil
}

// UpdateGeoReplicationConfig updates a geo-replication configuration.
func (db *DB) UpdateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error {
	config.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE geo_replication_configs
		SET enabled = $2, status = $3, last_snapshot_id = $4, last_sync_at = $5,
			last_error = $6, max_lag_snapshots = $7, max_lag_duration_hours = $8,
			alert_on_lag = $9, updated_at = $10
		WHERE id = $1
	`, config.ID, config.Enabled, config.Status, config.LastSnapshotID, config.LastSyncAt,
		config.LastError, config.MaxLagSnapshots, config.MaxLagDurationHours,
		config.AlertOnLag, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update geo-replication config: %w", err)
	}
	return nil
}

// DeleteGeoReplicationConfig deletes a geo-replication configuration.
func (db *DB) DeleteGeoReplicationConfig(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM geo_replication_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete geo-replication config: %w", err)
	}
	return nil
}

// ListGeoReplicationConfigsByOrg returns all geo-replication configs for an organization.
func (db *DB) ListGeoReplicationConfigsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.GeoReplicationConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		FROM geo_replication_configs
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list geo-replication configs: %w", err)
	}
	defer rows.Close()

	return scanGeoReplicationConfigs(rows)
}

// ListPendingReplications returns all enabled configs that may need replication.
func (db *DB) ListPendingReplications(ctx context.Context) ([]*models.GeoReplicationConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		FROM geo_replication_configs
		WHERE enabled = true AND status != 'syncing'
		ORDER BY last_sync_at ASC NULLS FIRST
	`)
	if err != nil {
		return nil, fmt.Errorf("list pending replications: %w", err)
	}
	defer rows.Close()

	return scanGeoReplicationConfigs(rows)
}

// RecordReplicationEvent records a replication event.
func (db *DB) RecordReplicationEvent(ctx context.Context, event *models.ReplicationEvent) error {
	durationMs := event.Duration.Milliseconds()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO replication_events (
			id, config_id, snapshot_id, status,
			started_at, completed_at, duration_ms, bytes_copied,
			error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, event.ID, event.ConfigID, event.SnapshotID, event.Status,
		event.StartedAt, event.CompletedAt, durationMs, event.BytesCopied,
		event.ErrorMessage, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("record replication event: %w", err)
	}
	return nil
}

// GetReplicationEvents returns recent replication events for a config.
func (db *DB) GetReplicationEvents(ctx context.Context, configID uuid.UUID, limit int) ([]*models.ReplicationEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, config_id, snapshot_id, status,
			started_at, completed_at, duration_ms, bytes_copied,
			error_message, created_at
		FROM replication_events
		WHERE config_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, configID, limit)
	if err != nil {
		return nil, fmt.Errorf("get replication events: %w", err)
	}
	defer rows.Close()

	var events []*models.ReplicationEvent
	for rows.Next() {
		var event models.ReplicationEvent
		var durationMs int64
		err := rows.Scan(
			&event.ID, &event.ConfigID, &event.SnapshotID, &event.Status,
			&event.StartedAt, &event.CompletedAt, &durationMs, &event.BytesCopied,
			&event.ErrorMessage, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan replication event: %w", err)
		}
		event.Duration = time.Duration(durationMs) * time.Millisecond
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate replication events: %w", err)
	}

	return events, nil
}

// GetReplicationLagForConfig calculates the current replication lag for a config.
func (db *DB) GetReplicationLagForConfig(ctx context.Context, configID uuid.UUID) (snapshotsBehind int, lastSyncAt *time.Time, err error) {
	// Get the config's last sync info
	var config models.GeoReplicationConfig
	err = db.Pool.QueryRow(ctx, `
		SELECT last_snapshot_id, last_sync_at
		FROM geo_replication_configs
		WHERE id = $1
	`, configID).Scan(&config.LastSnapshotID, &config.LastSyncAt)
	if err != nil {
		return 0, nil, fmt.Errorf("get config for lag calculation: %w", err)
	}

	// Count snapshots not yet replicated (simplified - actual implementation would
	// need to query the source repository for newer snapshots)
	// For now, we estimate based on the last successful replication event
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM replication_events
		WHERE config_id = $1 AND status = 'failed'
		AND started_at > COALESCE($2, '1970-01-01'::timestamptz)
	`, configID, config.LastSyncAt).Scan(&snapshotsBehind)
	if err != nil {
		return 0, nil, fmt.Errorf("count failed replications: %w", err)
	}

	return snapshotsBehind, config.LastSyncAt, nil
}

// UpdateRepositoryRegion updates the region for a repository.
func (db *DB) UpdateRepositoryRegion(ctx context.Context, repositoryID uuid.UUID, region string) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE repositories SET region = $2, updated_at = NOW() WHERE id = $1
	`, repositoryID, region)
	if err != nil {
		return fmt.Errorf("update repository region: %w", err)
	}
	return nil
}

// GetRepositoryRegion returns the region for a repository.
func (db *DB) GetRepositoryRegion(ctx context.Context, repositoryID uuid.UUID) (string, error) {
	var region *string
	err := db.Pool.QueryRow(ctx, `
		SELECT region FROM repositories WHERE id = $1
	`, repositoryID).Scan(&region)
	if err != nil {
		return "", fmt.Errorf("get repository region: %w", err)
	}
	if region == nil {
		return "", nil
	}
	return *region, nil
}

// scanGeoReplicationConfigs scans rows into geo-replication configs.
func scanGeoReplicationConfigs(rows pgx.Rows) ([]*models.GeoReplicationConfig, error) {
	var configs []*models.GeoReplicationConfig
	for rows.Next() {
		var config models.GeoReplicationConfig
		err := rows.Scan(
			&config.ID, &config.OrgID, &config.SourceRepositoryID, &config.TargetRepositoryID,
			&config.SourceRegion, &config.TargetRegion, &config.Enabled, &config.Status,
			&config.LastSnapshotID, &config.LastSyncAt, &config.LastError,
			&config.MaxLagSnapshots, &config.MaxLagDurationHours, &config.AlertOnLag,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan geo-replication config: %w", err)
		}
		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geo-replication configs: %w", err)
	}

	return configs, nil
}

// Agent Command methods

// CreateAgentCommand creates a new agent command.
func (db *DB) CreateAgentCommand(ctx context.Context, cmd *models.AgentCommand) error {
	payloadBytes, err := cmd.PayloadJSON()
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agent_commands (id, agent_id, org_id, type, status, payload, created_by,
		                            timeout_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, cmd.ID, cmd.AgentID, cmd.OrgID, string(cmd.Type), string(cmd.Status),
		payloadBytes, cmd.CreatedBy, cmd.TimeoutAt, cmd.CreatedAt, cmd.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent command: %w", err)
	}
	return nil
}

// GetAgentCommandByID returns a command by ID.
func (db *DB) GetAgentCommandByID(ctx context.Context, id uuid.UUID) (*models.AgentCommand, error) {
	var cmd models.AgentCommand
	var typeStr, statusStr string
	var payloadBytes, resultBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT c.id, c.agent_id, c.org_id, c.type, c.status, c.payload, c.result,
		       c.created_by, c.acknowledged_at, c.started_at, c.completed_at,
		       c.timeout_at, c.created_at, c.updated_at,
		       COALESCE(u.name, '')
		FROM agent_commands c
		LEFT JOIN users u ON c.created_by = u.id
		WHERE c.id = $1
	`, id).Scan(
		&cmd.ID, &cmd.AgentID, &cmd.OrgID, &typeStr, &statusStr,
		&payloadBytes, &resultBytes, &cmd.CreatedBy,
		&cmd.AcknowledgedAt, &cmd.StartedAt, &cmd.CompletedAt,
		&cmd.TimeoutAt, &cmd.CreatedAt, &cmd.UpdatedAt,
		&cmd.CreatedByName,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent command: %w", err)
	}

	cmd.Type = models.CommandType(typeStr)
	cmd.Status = models.CommandStatus(statusStr)
	if err := cmd.SetPayload(payloadBytes); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if err := cmd.SetResult(resultBytes); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	return &cmd, nil
}

// GetPendingCommandsForAgent returns pending commands for an agent.
func (db *DB) GetPendingCommandsForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.AgentCommand, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id, c.agent_id, c.org_id, c.type, c.status, c.payload, c.result,
		       c.created_by, c.acknowledged_at, c.started_at, c.completed_at,
		       c.timeout_at, c.created_at, c.updated_at,
		       COALESCE(u.name, '')
		FROM agent_commands c
		LEFT JOIN users u ON c.created_by = u.id
		WHERE c.agent_id = $1 AND c.status = 'pending'
		ORDER BY c.created_at ASC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list pending commands: %w", err)
	}
	defer rows.Close()

	return scanAgentCommands(rows)
}

// GetCommandsByAgentID returns all commands for an agent with optional limit.
func (db *DB) GetCommandsByAgentID(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.AgentCommand, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.id, c.agent_id, c.org_id, c.type, c.status, c.payload, c.result,
		       c.created_by, c.acknowledged_at, c.started_at, c.completed_at,
		       c.timeout_at, c.created_at, c.updated_at,
		       COALESCE(u.name, '')
		FROM agent_commands c
		LEFT JOIN users u ON c.created_by = u.id
		WHERE c.agent_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent commands: %w", err)
	}
	defer rows.Close()

	return scanAgentCommands(rows)
}

// UpdateAgentCommand updates a command's status and result.
func (db *DB) UpdateAgentCommand(ctx context.Context, cmd *models.AgentCommand) error {
	resultBytes, err := cmd.ResultJSON()
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	cmd.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE agent_commands
		SET status = $2, result = $3, acknowledged_at = $4, started_at = $5,
		    completed_at = $6, updated_at = $7
		WHERE id = $1
	`, cmd.ID, string(cmd.Status), resultBytes, cmd.AcknowledgedAt,
		cmd.StartedAt, cmd.CompletedAt, cmd.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update agent command: %w", err)
	}
	return nil
}

// CancelAgentCommand cancels a pending or running command.
func (db *DB) CancelAgentCommand(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE agent_commands
		SET status = 'canceled', completed_at = $2, updated_at = $2
		WHERE id = $1 AND status IN ('pending', 'acknowledged', 'running')
	`, id, now)
	if err != nil {
		return fmt.Errorf("cancel agent command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("command not found or already completed")
	}
	return nil
}

// MarkTimedOutCommands marks commands as timed out if they've exceeded their timeout.
func (db *DB) MarkTimedOutCommands(ctx context.Context) (int64, error) {
	now := time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE agent_commands
		SET status = 'timed_out',
		    result = '{"error": "command timed out waiting for agent response"}'::jsonb,
		    completed_at = $1,
		    updated_at = $1
		WHERE status IN ('pending', 'acknowledged', 'running')
		  AND timeout_at < $1
	`, now)
	if err != nil {
		return 0, fmt.Errorf("mark timed out commands: %w", err)
	}
	return result.RowsAffected(), nil
}

// scanAgentCommands scans rows into agent commands.
func scanAgentCommands(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.AgentCommand, error) {
	var commands []*models.AgentCommand
	for rows.Next() {
		var cmd models.AgentCommand
		var typeStr, statusStr string
		var payloadBytes, resultBytes []byte

		err := rows.Scan(
			&cmd.ID, &cmd.AgentID, &cmd.OrgID, &typeStr, &statusStr,
			&payloadBytes, &resultBytes, &cmd.CreatedBy,
			&cmd.AcknowledgedAt, &cmd.StartedAt, &cmd.CompletedAt,
			&cmd.TimeoutAt, &cmd.CreatedAt, &cmd.UpdatedAt,
			&cmd.CreatedByName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent command: %w", err)
		}

		cmd.Type = models.CommandType(typeStr)
		cmd.Status = models.CommandStatus(statusStr)
		if err := cmd.SetPayload(payloadBytes); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
		if err := cmd.SetResult(resultBytes); err != nil {
			return nil, fmt.Errorf("parse result: %w", err)
		}

		commands = append(commands, &cmd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent commands: %w", err)
	}

	return commands, nil
}

// Snapshot Mount methods

// CreateSnapshotMount creates a new snapshot mount record.
func (db *DB) CreateSnapshotMount(ctx context.Context, mount *models.SnapshotMount) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_mounts (
			id, org_id, agent_id, repository_id, snapshot_id, mount_path,
			status, mounted_at, expires_at, unmounted_at, error_message,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, mount.ID, mount.OrgID, mount.AgentID, mount.RepositoryID, mount.SnapshotID,
		mount.MountPath, string(mount.Status), mount.MountedAt, mount.ExpiresAt,
		mount.UnmountedAt, mount.ErrorMessage, mount.CreatedAt, mount.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot mount: %w", err)
	}
	return nil
}

// UpdateSnapshotMount updates a snapshot mount record.
func (db *DB) UpdateSnapshotMount(ctx context.Context, mount *models.SnapshotMount) error {
	mount.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_mounts
		SET status = $2, mount_path = $3, mounted_at = $4, expires_at = $5,
		    unmounted_at = $6, error_message = $7, updated_at = $8
		WHERE id = $1
	`, mount.ID, string(mount.Status), mount.MountPath, mount.MountedAt,
		mount.ExpiresAt, mount.UnmountedAt, mount.ErrorMessage, mount.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update snapshot mount: %w", err)
	}
	return nil
}

// GetSnapshotMountByID returns a snapshot mount by ID.
func (db *DB) GetSnapshotMountByID(ctx context.Context, id uuid.UUID) (*models.SnapshotMount, error) {
	var mount models.SnapshotMount
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id, mount_path,
		       status, mounted_at, expires_at, unmounted_at, error_message,
		       created_at, updated_at
		FROM snapshot_mounts
		WHERE id = $1
	`, id).Scan(
		&mount.ID, &mount.OrgID, &mount.AgentID, &mount.RepositoryID,
		&mount.SnapshotID, &mount.MountPath, &statusStr, &mount.MountedAt,
		&mount.ExpiresAt, &mount.UnmountedAt, &mount.ErrorMessage,
		&mount.CreatedAt, &mount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get snapshot mount: %w", err)
	}
	mount.Status = models.SnapshotMountStatus(statusStr)
	return &mount, nil
}

// GetActiveSnapshotMountBySnapshotID returns the active mount for a snapshot if one exists.
func (db *DB) GetActiveSnapshotMountBySnapshotID(ctx context.Context, agentID uuid.UUID, snapshotID string) (*models.SnapshotMount, error) {
	var mount models.SnapshotMount
	var statusStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id, mount_path,
		       status, mounted_at, expires_at, unmounted_at, error_message,
		       created_at, updated_at
		FROM snapshot_mounts
		WHERE agent_id = $1 AND snapshot_id = $2
		  AND status IN ('pending', 'mounting', 'mounted')
	`, agentID, snapshotID).Scan(
		&mount.ID, &mount.OrgID, &mount.AgentID, &mount.RepositoryID,
		&mount.SnapshotID, &mount.MountPath, &statusStr, &mount.MountedAt,
		&mount.ExpiresAt, &mount.UnmountedAt, &mount.ErrorMessage,
		&mount.CreatedAt, &mount.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get active snapshot mount: %w", err)
	}
	mount.Status = models.SnapshotMountStatus(statusStr)
	return &mount, nil
}

// GetSnapshotMountsByOrgID returns all snapshot mounts for an organization.
func (db *DB) GetSnapshotMountsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotMount, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id, mount_path,
		       status, mounted_at, expires_at, unmounted_at, error_message,
		       created_at, updated_at
		FROM snapshot_mounts
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list snapshot mounts: %w", err)
	}
	defer rows.Close()

	return scanSnapshotMounts(rows)
}

// GetActiveSnapshotMountsByAgentID returns active mounts for an agent.
func (db *DB) GetActiveSnapshotMountsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.SnapshotMount, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id, mount_path,
		       status, mounted_at, expires_at, unmounted_at, error_message,
		       created_at, updated_at
		FROM snapshot_mounts
		WHERE agent_id = $1 AND status IN ('pending', 'mounting', 'mounted')
		ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list active snapshot mounts: %w", err)
	}
	defer rows.Close()

	return scanSnapshotMounts(rows)
}

// GetExpiredSnapshotMounts returns mounts that have expired but are still marked as mounted.
func (db *DB) GetExpiredSnapshotMounts(ctx context.Context) ([]*models.SnapshotMount, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id, mount_path,
		       status, mounted_at, expires_at, unmounted_at, error_message,
		       created_at, updated_at
		FROM snapshot_mounts
		WHERE status = 'mounted' AND expires_at < NOW()
		ORDER BY expires_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list expired snapshot mounts: %w", err)
	}
	defer rows.Close()

	return scanSnapshotMounts(rows)
}

// DeleteSnapshotMount deletes a snapshot mount record.
func (db *DB) DeleteSnapshotMount(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM snapshot_mounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete snapshot mount: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot mount not found")
	}
	return nil
}

// scanSnapshotMounts scans rows into snapshot mount structs.
func scanSnapshotMounts(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.SnapshotMount, error) {
	var mounts []*models.SnapshotMount
	for rows.Next() {
		var mount models.SnapshotMount
		var statusStr string
		err := rows.Scan(
			&mount.ID, &mount.OrgID, &mount.AgentID, &mount.RepositoryID,
			&mount.SnapshotID, &mount.MountPath, &statusStr, &mount.MountedAt,
			&mount.ExpiresAt, &mount.UnmountedAt, &mount.ErrorMessage,
			&mount.CreatedAt, &mount.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot mount: %w", err)
		}
		mount.Status = models.SnapshotMountStatus(statusStr)
		mounts = append(mounts, &mount)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot mounts: %w", err)
	}

	return mounts, nil
}

// Config Template methods

// CreateConfigTemplate creates a new config template.
func (db *DB) CreateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO config_templates (
			id, org_id, created_by_id, name, description, type, visibility,
			tags, config, usage_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		template.ID, template.OrgID, template.CreatedByID, template.Name,
		template.Description, template.Type, template.Visibility,
		tagsJSON, template.Config, template.UsageCount, template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create config template: %w", err)
	}
	return nil
}

// GetConfigTemplateByID returns a config template by ID.
func (db *DB) GetConfigTemplateByID(ctx context.Context, id uuid.UUID) (*models.ConfigTemplate, error) {
	var template models.ConfigTemplate
	var tagsBytes []byte
	var typeStr, visibilityStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE id = $1
	`, id).Scan(
		&template.ID, &template.OrgID, &template.CreatedByID, &template.Name,
		&template.Description, &typeStr, &visibilityStr,
		&tagsBytes, &template.Config, &template.UsageCount, &template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get config template: %w", err)
	}

	template.Type = models.TemplateType(typeStr)
	template.Visibility = models.TemplateVisibility(visibilityStr)

	if err := template.SetTags(tagsBytes); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	return &template, nil
}

// GetConfigTemplatesByOrgID returns all config templates for an organization.
func (db *DB) GetConfigTemplatesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list config templates: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// GetPublicConfigTemplates returns all public config templates.
func (db *DB) GetPublicConfigTemplates(ctx context.Context) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE visibility = 'public'
		ORDER BY usage_count DESC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list public config templates: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// GetConfigTemplatesByType returns config templates filtered by type.
func (db *DB) GetConfigTemplatesByType(ctx context.Context, orgID uuid.UUID, templateType models.TemplateType) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE (org_id = $1 OR visibility = 'public') AND type = $2
		ORDER BY usage_count DESC, created_at DESC
	`, orgID, templateType)
	if err != nil {
		return nil, fmt.Errorf("list config templates by type: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// UpdateConfigTemplate updates a config template.
func (db *DB) UpdateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE config_templates
		SET name = $2, description = $3, visibility = $4, tags = $5, updated_at = $6
		WHERE id = $1
	`, template.ID, template.Name, template.Description, template.Visibility, tagsJSON, template.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update config template: %w", err)
	}
	return nil
}

// DeleteConfigTemplate deletes a config template.
func (db *DB) DeleteConfigTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM config_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete config template: %w", err)
	}
	return nil
}

// IncrementTemplateUsageCount increments the usage count for a template.
func (db *DB) IncrementTemplateUsageCount(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE config_templates
		SET usage_count = usage_count + 1, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("increment template usage count: %w", err)
	}
	return nil
}

// scanConfigTemplates scans rows into config templates.
func scanConfigTemplates(rows pgx.Rows) ([]*models.ConfigTemplate, error) {
	var templates []*models.ConfigTemplate
	for rows.Next() {
		var template models.ConfigTemplate
		var tagsBytes []byte
		var typeStr, visibilityStr string

		err := rows.Scan(
			&template.ID, &template.OrgID, &template.CreatedByID, &template.Name,
			&template.Description, &typeStr, &visibilityStr,
			&tagsBytes, &template.Config, &template.UsageCount, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan config template: %w", err)
		}

		template.Type = models.TemplateType(typeStr)
		template.Visibility = models.TemplateVisibility(visibilityStr)

		if err := template.SetTags(tagsBytes); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate config templates: %w", err)
	}

	return templates, nil
}

// Announcement methods

// GetAnnouncementByID returns an announcement by ID.
func (db *DB) GetAnnouncementByID(ctx context.Context, id uuid.UUID) (*models.Announcement, error) {
	var a models.Announcement
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, title, message, type, dismissible, starts_at, ends_at,
		       active, created_by, created_at, updated_at
		FROM announcements
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.Title, &a.Message, &a.Type, &a.Dismissible,
		&a.StartsAt, &a.EndsAt, &a.Active, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return &a, nil
}

// ListAnnouncementsByOrg returns all announcements for an organization.
func (db *DB) ListAnnouncementsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Announcement, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, title, message, type, dismissible, starts_at, ends_at,
		       active, created_by, created_at, updated_at
		FROM announcements
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	return scanAnnouncements(rows)
}

// ListActiveAnnouncements returns active announcements for an organization that the user hasn't dismissed.
// It respects scheduled start/end times.
func (db *DB) ListActiveAnnouncements(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, now time.Time) ([]*models.Announcement, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT a.id, a.org_id, a.title, a.message, a.type, a.dismissible, a.starts_at, a.ends_at,
		       a.active, a.created_by, a.created_at, a.updated_at
		FROM announcements a
		LEFT JOIN announcement_dismissals d ON a.id = d.announcement_id AND d.user_id = $3
		WHERE a.org_id = $1
		  AND a.active = true
		  AND d.id IS NULL
		  AND (a.starts_at IS NULL OR a.starts_at <= $2)
		  AND (a.ends_at IS NULL OR a.ends_at > $2)
		ORDER BY
		  CASE a.type
		    WHEN 'critical' THEN 1
		    WHEN 'warning' THEN 2
		    ELSE 3
		  END,
		  a.created_at DESC
	`, orgID, now, userID)
	if err != nil {
		return nil, fmt.Errorf("list active announcements: %w", err)
	}
	defer rows.Close()

	return scanAnnouncements(rows)
}

// CreateAnnouncement creates a new announcement.
func (db *DB) CreateAnnouncement(ctx context.Context, a *models.Announcement) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO announcements (id, org_id, title, message, type, dismissible, starts_at, ends_at,
		            active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, a.ID, a.OrgID, a.Title, a.Message, a.Type, a.Dismissible, a.StartsAt, a.EndsAt,
		a.Active, a.CreatedBy, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create announcement: %w", err)
	}
	return nil
}

// UpdateAnnouncement updates an existing announcement.
func (db *DB) UpdateAnnouncement(ctx context.Context, a *models.Announcement) error {
	a.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE announcements
		SET title = $2, message = $3, type = $4, dismissible = $5, starts_at = $6, ends_at = $7,
		    active = $8, updated_at = $9
		WHERE id = $1
	`, a.ID, a.Title, a.Message, a.Type, a.Dismissible, a.StartsAt, a.EndsAt, a.Active, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update announcement: %w", err)
	}
	return nil
}

// DeleteAnnouncement deletes an announcement and its dismissals.
func (db *DB) DeleteAnnouncement(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

// CreateAnnouncementDismissal records a user's dismissal of an announcement.
func (db *DB) CreateAnnouncementDismissal(ctx context.Context, d *models.AnnouncementDismissal) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO announcement_dismissals (id, org_id, announcement_id, user_id, dismissed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (announcement_id, user_id) DO NOTHING
	`, d.ID, d.OrgID, d.AnnouncementID, d.UserID, d.DismissedAt)
	if err != nil {
		return fmt.Errorf("create announcement dismissal: %w", err)
	}
	return nil
}

// scanAnnouncements scans rows into announcements.
func scanAnnouncements(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.Announcement, error) {
	var announcements []*models.Announcement
	for rows.Next() {
		var a models.Announcement
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.Title, &a.Message, &a.Type, &a.Dismissible,
			&a.StartsAt, &a.EndsAt, &a.Active, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		announcements = append(announcements, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcements: %w", err)
	}

	return announcements, nil
}

// Saved Filter methods

// GetSavedFiltersByUserAndOrg returns all saved filters for a user in an organization,
// including shared filters from other users.
func (db *DB) GetSavedFiltersByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]*models.SavedFilter, error) {
	query := `
		SELECT id, user_id, org_id, name, entity_type, filters, shared, is_default, created_at, updated_at
		FROM saved_filters
		WHERE org_id = $1 AND (user_id = $2 OR shared = TRUE)
	`
	args := []interface{}{orgID, userID}

	if entityType != "" {
		query += ` AND entity_type = $3`
		args = append(args, entityType)
	}

	query += ` ORDER BY name`

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list saved filters: %w", err)
	}
	defer rows.Close()

	var filters []*models.SavedFilter
	for rows.Next() {
		var f models.SavedFilter
		err := rows.Scan(
			&f.ID, &f.UserID, &f.OrgID, &f.Name, &f.EntityType,
			&f.Filters, &f.Shared, &f.IsDefault, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan saved filter: %w", err)
		}
		filters = append(filters, &f)
	}

	return filters, nil
}

// GetSavedFilterByID returns a saved filter by ID.
func (db *DB) GetSavedFilterByID(ctx context.Context, id uuid.UUID) (*models.SavedFilter, error) {
	var f models.SavedFilter
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, org_id, name, entity_type, filters, shared, is_default, created_at, updated_at
		FROM saved_filters
		WHERE id = $1
	`, id).Scan(
		&f.ID, &f.UserID, &f.OrgID, &f.Name, &f.EntityType,
		&f.Filters, &f.Shared, &f.IsDefault, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get saved filter: %w", err)
	}
	return &f, nil
}

// GetDefaultSavedFilter returns the default filter for a user/entity type.
func (db *DB) GetDefaultSavedFilter(ctx context.Context, userID, orgID uuid.UUID, entityType string) (*models.SavedFilter, error) {
	var f models.SavedFilter
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, org_id, name, entity_type, filters, shared, is_default, created_at, updated_at
		FROM saved_filters
		WHERE user_id = $1 AND org_id = $2 AND entity_type = $3 AND is_default = TRUE
	`, userID, orgID, entityType).Scan(
		&f.ID, &f.UserID, &f.OrgID, &f.Name, &f.EntityType,
		&f.Filters, &f.Shared, &f.IsDefault, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get default saved filter: %w", err)
	}
	return &f, nil
}

// CreateSavedFilter creates a new saved filter.
func (db *DB) CreateSavedFilter(ctx context.Context, f *models.SavedFilter) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// If setting as default, clear existing default for this user/entity type
		if f.IsDefault {
			_, err := tx.Exec(ctx, `
				UPDATE saved_filters
				SET is_default = FALSE, updated_at = NOW()
				WHERE user_id = $1 AND org_id = $2 AND entity_type = $3 AND is_default = TRUE
			`, f.UserID, f.OrgID, f.EntityType)
			if err != nil {
				return fmt.Errorf("clear existing default: %w", err)
			}
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO saved_filters (id, user_id, org_id, name, entity_type, filters, shared, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, f.ID, f.UserID, f.OrgID, f.Name, f.EntityType, f.Filters, f.Shared, f.IsDefault, f.CreatedAt, f.UpdatedAt)
		if err != nil {
			return fmt.Errorf("create saved filter: %w", err)
		}
		return nil
	})
}

// UpdateSavedFilter updates an existing saved filter.
func (db *DB) UpdateSavedFilter(ctx context.Context, f *models.SavedFilter) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// If setting as default, clear existing default for this user/entity type
		if f.IsDefault {
			_, err := tx.Exec(ctx, `
				UPDATE saved_filters
				SET is_default = FALSE, updated_at = NOW()
				WHERE user_id = $1 AND org_id = $2 AND entity_type = $3 AND is_default = TRUE AND id != $4
			`, f.UserID, f.OrgID, f.EntityType, f.ID)
			if err != nil {
				return fmt.Errorf("clear existing default: %w", err)
			}
		}

		_, err := tx.Exec(ctx, `
			UPDATE saved_filters
			SET name = $1, filters = $2, shared = $3, is_default = $4, updated_at = NOW()
			WHERE id = $5
		`, f.Name, f.Filters, f.Shared, f.IsDefault, f.ID)
		if err != nil {
			return fmt.Errorf("update saved filter: %w", err)
		}
		return nil
	})
}

// DeleteSavedFilter deletes a saved filter.
func (db *DB) DeleteSavedFilter(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM saved_filters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete saved filter: %w", err)
	}
	return nil
}

// ========== IP Allowlist Methods ==========

// GetIPAllowlistByID returns an IP allowlist entry by ID.
func (db *DB) GetIPAllowlistByID(ctx context.Context, id uuid.UUID) (*models.IPAllowlist, error) {
	var a models.IPAllowlist
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, cidr, description, type, enabled, created_by, updated_by, created_at, updated_at
		FROM ip_allowlists
		WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.CIDR, &a.Description, &a.Type, &a.Enabled,
		&a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ip allowlist: %w", err)
	}
	return &a, nil
}

// ListIPAllowlistsByOrg returns all IP allowlist entries for an organization.
func (db *DB) ListIPAllowlistsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.IPAllowlist, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, cidr, description, type, enabled, created_by, updated_by, created_at, updated_at
		FROM ip_allowlists
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list ip allowlists: %w", err)
	}
	defer rows.Close()

	return scanIPAllowlists(rows)
}

// ListEnabledIPAllowlistsByOrg returns enabled IP allowlist entries for an organization by type.
func (db *DB) ListEnabledIPAllowlistsByOrg(ctx context.Context, orgID uuid.UUID, allowlistType models.IPAllowlistType) ([]*models.IPAllowlist, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, cidr, description, type, enabled, created_by, updated_by, created_at, updated_at
		FROM ip_allowlists
		WHERE org_id = $1 AND enabled = true AND (type = $2 OR type = 'both')
		ORDER BY created_at DESC
	`, orgID, allowlistType)
	if err != nil {
		return nil, fmt.Errorf("list enabled ip allowlists: %w", err)
	}
	defer rows.Close()

	return scanIPAllowlists(rows)
}

// CreateIPAllowlist creates a new IP allowlist entry.
func (db *DB) CreateIPAllowlist(ctx context.Context, a *models.IPAllowlist) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_allowlists (id, org_id, cidr, description, type, enabled, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.OrgID, a.CIDR, a.Description, a.Type, a.Enabled, a.CreatedBy, a.UpdatedBy, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create ip allowlist: %w", err)
	}
	return nil
}

// UpdateIPAllowlist updates an existing IP allowlist entry.
func (db *DB) UpdateIPAllowlist(ctx context.Context, a *models.IPAllowlist) error {
	a.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE ip_allowlists SET
			cidr = $2, description = $3, type = $4, enabled = $5, updated_by = $6, updated_at = $7
		WHERE id = $1
	`, a.ID, a.CIDR, a.Description, a.Type, a.Enabled, a.UpdatedBy, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update ip allowlist: %w", err)
	}
	return nil
}

// DeleteIPAllowlist deletes an IP allowlist entry.
func (db *DB) DeleteIPAllowlist(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ip_allowlists WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ip allowlist: %w", err)
	}
	return nil
}

// GetIPAllowlistSettings returns the IP allowlist settings for an organization.
func (db *DB) GetIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	var s models.IPAllowlistSettings
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, enabled, enforce_for_ui, enforce_for_agent, allow_admin_bypass, created_at, updated_at
		FROM ip_allowlist_settings
		WHERE org_id = $1
	`, orgID).Scan(
		&s.ID, &s.OrgID, &s.Enabled, &s.EnforceForUI, &s.EnforceForAgent, &s.AllowAdminBypass, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ip allowlist settings: %w", err)
	}
	return &s, nil
}

// GetOrCreateIPAllowlistSettings returns existing settings or creates default settings.
func (db *DB) GetOrCreateIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	settings, err := db.GetIPAllowlistSettings(ctx, orgID)
	if err == nil {
		return settings, nil
	}

	// Create default settings
	settings = models.NewIPAllowlistSettings(orgID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO ip_allowlist_settings (id, org_id, enabled, enforce_for_ui, enforce_for_agent, allow_admin_bypass, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id) DO NOTHING
	`, settings.ID, settings.OrgID, settings.Enabled, settings.EnforceForUI, settings.EnforceForAgent, settings.AllowAdminBypass, settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create ip allowlist settings: %w", err)
	}

	// Re-fetch to get the actual values (in case of conflict)
	return db.GetIPAllowlistSettings(ctx, orgID)
}

// UpdateIPAllowlistSettings updates the IP allowlist settings for an organization.
func (db *DB) UpdateIPAllowlistSettings(ctx context.Context, s *models.IPAllowlistSettings) error {
	s.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE ip_allowlist_settings
		SET enabled = $2, enforce_for_ui = $3, enforce_for_agent = $4, allow_admin_bypass = $5, updated_at = $6
		WHERE org_id = $1
	`, s.OrgID, s.Enabled, s.EnforceForUI, s.EnforceForAgent, s.AllowAdminBypass, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update ip allowlist settings: %w", err)
	}
	return nil
}

// CreateIPBlockedAttempt records a blocked access attempt.
func (db *DB) CreateIPBlockedAttempt(ctx context.Context, b *models.IPBlockedAttempt) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_blocked_attempts (id, org_id, ip_address, request_type, path, user_id, agent_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, b.ID, b.OrgID, b.IPAddress, b.RequestType, b.Path, b.UserID, b.AgentID, b.Reason, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create ip blocked attempt: %w", err)
	}
	return nil
}

// ListIPBlockedAttemptsByOrg returns blocked attempts for an organization.
func (db *DB) ListIPBlockedAttemptsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.IPBlockedAttempt, int, error) {
	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ip_blocked_attempts WHERE org_id = $1
	`, orgID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ip blocked attempts: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, ip_address, request_type, path, user_id, agent_id, reason, created_at
		FROM ip_blocked_attempts
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list ip blocked attempts: %w", err)
	}
	defer rows.Close()

	attempts, err := scanIPBlockedAttempts(rows)
	if err != nil {
		return nil, 0, err
	}

	return attempts, total, nil
}

// scanIPAllowlists scans rows into IP allowlist entries.
func scanIPAllowlists(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.IPAllowlist, error) {
	var allowlists []*models.IPAllowlist
	for rows.Next() {
		var a models.IPAllowlist
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.CIDR, &a.Description, &a.Type, &a.Enabled,
			&a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ip allowlist: %w", err)
		}
		allowlists = append(allowlists, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate IP allowlists: %w", err)
	}

	return allowlists, nil
}

// scanIPBlockedAttempts scans rows into blocked attempt records.
func scanIPBlockedAttempts(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.IPBlockedAttempt, error) {
	var attempts []*models.IPBlockedAttempt
	for rows.Next() {
		var b models.IPBlockedAttempt
		err := rows.Scan(
			&b.ID, &b.OrgID, &b.IPAddress, &b.RequestType, &b.Path, &b.UserID, &b.AgentID, &b.Reason, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ip blocked attempt: %w", err)
		}
		attempts = append(attempts, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ip blocked attempts: %w", err)
	}

	return attempts, nil
}

// Rate Limit Config methods

// GetRateLimitConfigByID returns a rate limit config by ID.
func (db *DB) GetRateLimitConfigByID(ctx context.Context, id uuid.UUID) (*models.RateLimitConfig, error) {
	var c models.RateLimitConfig
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, endpoint, requests_per_period, period_seconds, enabled, created_by, created_at, updated_at
		FROM rate_limit_configs
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.Endpoint, &c.RequestsPerPeriod, &c.PeriodSeconds,
		&c.Enabled, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get rate limit config: %w", err)
	}
	return &c, nil
}

// ListRateLimitConfigs returns all rate limit configs for an organization.
func (db *DB) ListRateLimitConfigs(ctx context.Context, orgID uuid.UUID) ([]*models.RateLimitConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, endpoint, requests_per_period, period_seconds, enabled, created_by, created_at, updated_at
		FROM rate_limit_configs
		WHERE org_id = $1
		ORDER BY endpoint
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list rate limit configs: %w", err)
	}
	defer rows.Close()

	return scanRateLimitConfigs(rows)
}

// GetRateLimitConfigByEndpoint returns a rate limit config for a specific endpoint.
func (db *DB) GetRateLimitConfigByEndpoint(ctx context.Context, orgID uuid.UUID, endpoint string) (*models.RateLimitConfig, error) {
	var c models.RateLimitConfig
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, endpoint, requests_per_period, period_seconds, enabled, created_by, created_at, updated_at
		FROM rate_limit_configs
		WHERE org_id = $1 AND endpoint = $2 AND enabled = true
	`, orgID, endpoint).Scan(
		&c.ID, &c.OrgID, &c.Endpoint, &c.RequestsPerPeriod, &c.PeriodSeconds,
		&c.Enabled, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get rate limit config by endpoint: %w", err)
	}
	return &c, nil
}

// CreateRateLimitConfig creates a new rate limit config.
func (db *DB) CreateRateLimitConfig(ctx context.Context, c *models.RateLimitConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO rate_limit_configs (id, org_id, endpoint, requests_per_period, period_seconds, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.OrgID, c.Endpoint, c.RequestsPerPeriod, c.PeriodSeconds, c.Enabled, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create rate limit config: %w", err)
	}
	return nil
}

// UpdateRateLimitConfig updates an existing rate limit config.
func (db *DB) UpdateRateLimitConfig(ctx context.Context, c *models.RateLimitConfig) error {
	c.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE rate_limit_configs SET
			endpoint = $2, requests_per_period = $3, period_seconds = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`, c.ID, c.Endpoint, c.RequestsPerPeriod, c.PeriodSeconds, c.Enabled, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update rate limit config: %w", err)
	}
	return nil
}

// DeleteRateLimitConfig deletes a rate limit config.
func (db *DB) DeleteRateLimitConfig(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM rate_limit_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete rate limit config: %w", err)
	}
	return nil
}

// scanRateLimitConfigs scans rows into rate limit configs.
func scanRateLimitConfigs(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.RateLimitConfig, error) {
	var configs []*models.RateLimitConfig
	for rows.Next() {
		var c models.RateLimitConfig
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.Endpoint, &c.RequestsPerPeriod, &c.PeriodSeconds,
			&c.Enabled, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan rate limit config: %w", err)
		}
		configs = append(configs, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rate limit configs: %w", err)
	}

	return configs, nil
}

// Blocked Request methods

// RecordBlockedRequest records a blocked request for statistics.
func (db *DB) RecordBlockedRequest(ctx context.Context, orgID *uuid.UUID, ipAddress, endpoint, userAgent, reason string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO blocked_requests (id, org_id, ip_address, endpoint, user_agent, blocked_at, reason)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
	`, uuid.New(), orgID, ipAddress, endpoint, userAgent, reason)
	if err != nil {
		return fmt.Errorf("record blocked request: %w", err)
	}
	return nil
}

// GetRateLimitStats returns rate limiting statistics for an organization.
func (db *DB) GetRateLimitStats(ctx context.Context, orgID uuid.UUID) (*models.RateLimitStats, error) {
	stats := &models.RateLimitStats{}

	// Get blocked today count
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM blocked_requests
		WHERE (org_id = $1 OR org_id IS NULL)
		  AND blocked_at >= CURRENT_DATE
	`, orgID).Scan(&stats.BlockedToday)
	if err != nil {
		return nil, fmt.Errorf("get blocked today count: %w", err)
	}

	// Get top blocked IPs
	rows, err := db.Pool.Query(ctx, `
		SELECT ip_address, COUNT(*) as count
		FROM blocked_requests
		WHERE (org_id = $1 OR org_id IS NULL)
		  AND blocked_at >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY ip_address
		ORDER BY count DESC
		LIMIT 10
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get top blocked IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip models.IPBlockCount
		if err := rows.Scan(&ip.IPAddress, &ip.Count); err != nil {
			return nil, fmt.Errorf("scan blocked IP: %w", err)
		}
		stats.TopBlockedIPs = append(stats.TopBlockedIPs, ip)
	}

	// Get top blocked endpoints
	rows2, err := db.Pool.Query(ctx, `
		SELECT endpoint, COUNT(*) as count
		FROM blocked_requests
		WHERE (org_id = $1 OR org_id IS NULL)
		  AND blocked_at >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY endpoint
		ORDER BY count DESC
		LIMIT 10
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get top blocked endpoints: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var route models.RouteBlockCount
		if err := rows2.Scan(&route.Endpoint, &route.Count); err != nil {
			return nil, fmt.Errorf("scan blocked endpoint: %w", err)
		}
		stats.TopBlockedRoutes = append(stats.TopBlockedRoutes, route)
	}

	return stats, nil
}

// ListRecentBlockedRequests returns recent blocked requests.
func (db *DB) ListRecentBlockedRequests(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.BlockedRequest, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, ip_address, endpoint, user_agent, blocked_at, reason
		FROM blocked_requests
		WHERE org_id = $1 OR org_id IS NULL
		ORDER BY blocked_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list blocked requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.BlockedRequest
	for rows.Next() {
		var r models.BlockedRequest
		if err := rows.Scan(&r.ID, &r.OrgID, &r.IPAddress, &r.Endpoint, &r.UserAgent, &r.BlockedAt, &r.Reason); err != nil {
			return nil, fmt.Errorf("scan blocked request: %w", err)
		}
		requests = append(requests, &r)
	}

	return requests, nil
}

// ========== IP Ban Methods ==========

// GetIPBanByID returns an IP ban by ID.
func (db *DB) GetIPBanByID(ctx context.Context, id uuid.UUID) (*models.IPBan, error) {
	var b models.IPBan
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, ip_address, reason, banned_by, expires_at, created_at
		FROM ip_bans
		WHERE id = $1
	`, id).Scan(
		&b.ID, &b.OrgID, &b.IPAddress, &b.Reason, &b.BannedBy, &b.ExpiresAt, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ip ban: %w", err)
	}
	return &b, nil
}

// ListIPBans returns all IP bans for an organization.
func (db *DB) ListIPBans(ctx context.Context, orgID uuid.UUID) ([]*models.IPBan, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, ip_address, reason, banned_by, expires_at, created_at
		FROM ip_bans
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list ip bans: %w", err)
	}
	defer rows.Close()

	return scanIPBans(rows)
}

// ListActiveIPBans returns active (non-expired) IP bans for an organization.
func (db *DB) ListActiveIPBans(ctx context.Context, orgID uuid.UUID) ([]*models.IPBan, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, ip_address, reason, banned_by, expires_at, created_at
		FROM ip_bans
		WHERE org_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active ip bans: %w", err)
	}
	defer rows.Close()

	return scanIPBans(rows)
}

// IsIPBanned checks if an IP address is banned for an organization.
func (db *DB) IsIPBanned(ctx context.Context, orgID uuid.UUID, ipAddress string) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ip_bans
		WHERE org_id = $1 AND ip_address = $2 AND (expires_at IS NULL OR expires_at > NOW())
	`, orgID, ipAddress).Scan(&count)
// GetEnabledSchedulesByOrgID returns all enabled backup schedules for an org.
func (db *DB) GetEnabledSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.backup_type, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps, s.backup_window_start, s.backup_window_end,
		       s.excluded_hours, s.compression_level, s.max_file_size_mb, s.on_mount_unavailable,
		       s.priority, s.preemptible, s.classification_level, s.classification_data_types,
		       s.docker_options, s.pihole_config, s.proxmox_options,
		       s.enabled, s.created_at, s.updated_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.enabled = true
		ORDER BY s.name
	`, orgID)
	if err != nil {
		return false, fmt.Errorf("check ip ban: %w", err)
	}
	return count > 0, nil
}

// CreateIPBan creates a new IP ban.
func (db *DB) CreateIPBan(ctx context.Context, b *models.IPBan) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_bans (id, org_id, ip_address, reason, banned_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, b.ID, b.OrgID, b.IPAddress, b.Reason, b.BannedBy, b.ExpiresAt, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create ip ban: %w", err)
	}
	return nil
}

// DeleteIPBan deletes an IP ban.
func (db *DB) DeleteIPBan(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ip_bans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ip ban: %w", err)
	}
	return nil
}

// DeleteExpiredIPBans removes expired IP bans.
func (db *DB) DeleteExpiredIPBans(ctx context.Context) (int64, error) {
	result, err := db.Pool.Exec(ctx, `DELETE FROM ip_bans WHERE expires_at IS NOT NULL AND expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired ip bans: %w", err)
	}
	return result.RowsAffected(), nil
}

// scanIPBans scans rows into IP bans.
func scanIPBans(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.IPBan, error) {
	var bans []*models.IPBan
	for rows.Next() {
		var b models.IPBan
		err := rows.Scan(
			&b.ID, &b.OrgID, &b.IPAddress, &b.Reason, &b.BannedBy, &b.ExpiresAt, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ip ban: %w", err)
		}
		bans = append(bans, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate IP bans: %w", err)
	}

	return bans, nil
}

// Storage Tiering methods

// GetStorageTierConfigs returns all tier configurations for an organization.
func (db *DB) GetStorageTierConfigs(ctx context.Context, orgID uuid.UUID) ([]*models.StorageTierConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, tier_type, name, description, cost_per_gb_month, retrieval_cost,
		       retrieval_time, enabled, created_at, updated_at
		FROM storage_tier_configs
		WHERE org_id = $1
		ORDER BY
			CASE tier_type
				WHEN 'hot' THEN 1
				WHEN 'warm' THEN 2
				WHEN 'cold' THEN 3
				WHEN 'archive' THEN 4
			END
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get tier configs: %w", err)
	}
	defer rows.Close()

	var configs []*models.StorageTierConfig
	for rows.Next() {
		var c models.StorageTierConfig
		var tierType string
		var desc *string
		err := rows.Scan(
			&c.ID, &c.OrgID, &tierType, &c.Name, &desc, &c.CostPerGBMonth,
			&c.RetrievalCost, &c.RetrievalTime, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tier config: %w", err)
		}
		c.TierType = models.StorageTierType(tierType)
		if desc != nil {
			c.Description = *desc
		}
		configs = append(configs, &c)
	}

	return configs, nil
}

// GetStorageTierConfig returns a single tier configuration by ID.
func (db *DB) GetStorageTierConfig(ctx context.Context, id uuid.UUID) (*models.StorageTierConfig, error) {
	var c models.StorageTierConfig
	var tierType string
	var desc *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, tier_type, name, description, cost_per_gb_month, retrieval_cost,
		       retrieval_time, enabled, created_at, updated_at
		FROM storage_tier_configs
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &tierType, &c.Name, &desc, &c.CostPerGBMonth,
		&c.RetrievalCost, &c.RetrievalTime, &c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get tier config: %w", err)
	}
	c.TierType = models.StorageTierType(tierType)
	if desc != nil {
		c.Description = *desc
	}
	return &c, nil
}

// CreateStorageTierConfig creates a new tier configuration.
func (db *DB) CreateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO storage_tier_configs (id, org_id, tier_type, name, description, cost_per_gb_month,
		            retrieval_cost, retrieval_time, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, config.ID, config.OrgID, string(config.TierType), config.Name, config.Description,
		config.CostPerGBMonth, config.RetrievalCost, config.RetrievalTime, config.Enabled,
		config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tier config: %w", err)
	}
	return nil
}

// UpdateStorageTierConfig updates an existing tier configuration.
func (db *DB) UpdateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error {
	config.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE storage_tier_configs SET
			name = $2, description = $3, cost_per_gb_month = $4, retrieval_cost = $5,
			retrieval_time = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, config.ID, config.Name, config.Description, config.CostPerGBMonth,
		config.RetrievalCost, config.RetrievalTime, config.Enabled, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update tier config: %w", err)
	}
	return nil
}

// CreateDefaultTierConfigs creates default tier configurations for an organization.
func (db *DB) CreateDefaultTierConfigs(ctx context.Context, orgID uuid.UUID) error {
	configs := models.DefaultTierConfigs(orgID)
	for _, config := range configs {
		if err := db.CreateStorageTierConfig(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

// GetTierRules returns all tier rules for an organization.
func (db *DB) GetTierRules(ctx context.Context, orgID uuid.UUID) ([]*models.TierRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, schedule_id, name, description, from_tier, to_tier,
		       age_threshold_days, min_copies, priority, enabled, created_at, updated_at
		FROM tier_rules
		WHERE org_id = $1
		ORDER BY priority, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get tier rules: %w", err)
	}
	defer rows.Close()

	return scanTierRules(rows)
}

// GetTierRule returns a single tier rule by ID.
func (db *DB) GetTierRule(ctx context.Context, id uuid.UUID) (*models.TierRule, error) {
	var r models.TierRule
	var fromTier, toTier string
	var desc *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, repository_id, schedule_id, name, description, from_tier, to_tier,
		       age_threshold_days, min_copies, priority, enabled, created_at, updated_at
		FROM tier_rules
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.OrgID, &r.RepositoryID, &r.ScheduleID, &r.Name, &desc, &fromTier, &toTier,
		&r.AgeThresholdDay, &r.MinCopies, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get tier rule: %w", err)
	}
	r.FromTier = models.StorageTierType(fromTier)
	r.ToTier = models.StorageTierType(toTier)
	if desc != nil {
		r.Description = *desc
	}
	return &r, nil
}

// CreateTierRule creates a new tier rule.
func (db *DB) CreateTierRule(ctx context.Context, rule *models.TierRule) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO tier_rules (id, org_id, repository_id, schedule_id, name, description,
		            from_tier, to_tier, age_threshold_days, min_copies, priority, enabled,
		            created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, rule.ID, rule.OrgID, rule.RepositoryID, rule.ScheduleID, rule.Name, rule.Description,
		string(rule.FromTier), string(rule.ToTier), rule.AgeThresholdDay, rule.MinCopies,
		rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tier rule: %w", err)
	}
	return nil
}

// UpdateTierRule updates an existing tier rule.
func (db *DB) UpdateTierRule(ctx context.Context, rule *models.TierRule) error {
	rule.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE tier_rules SET
			repository_id = $2, schedule_id = $3, name = $4, description = $5,
			from_tier = $6, to_tier = $7, age_threshold_days = $8, min_copies = $9,
			priority = $10, enabled = $11, updated_at = $12
		WHERE id = $1
	`, rule.ID, rule.RepositoryID, rule.ScheduleID, rule.Name, rule.Description,
		string(rule.FromTier), string(rule.ToTier), rule.AgeThresholdDay, rule.MinCopies,
		rule.Priority, rule.Enabled, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update tier rule: %w", err)
	}
	return nil
}

// DeleteTierRule deletes a tier rule.
func (db *DB) DeleteTierRule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM tier_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tier rule: %w", err)
	}
	return nil
}

// GetEnabledTierRules returns all enabled tier rules for an organization, sorted by priority.
func (db *DB) GetEnabledTierRules(ctx context.Context, orgID uuid.UUID) ([]*models.TierRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, repository_id, schedule_id, name, description, from_tier, to_tier,
		       age_threshold_days, min_copies, priority, enabled, created_at, updated_at
		FROM tier_rules
		WHERE org_id = $1 AND enabled = true
		ORDER BY priority, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get enabled tier rules: %w", err)
	}
	defer rows.Close()

	return scanTierRules(rows)
}

// scanTierRules scans rows into tier rules.
func scanTierRules(rows pgx.Rows) ([]*models.TierRule, error) {
	var rules []*models.TierRule
	for rows.Next() {
		var r models.TierRule
		var fromTier, toTier string
		var desc *string
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.RepositoryID, &r.ScheduleID, &r.Name, &desc, &fromTier, &toTier,
			&r.AgeThresholdDay, &r.MinCopies, &r.Priority, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tier rule: %w", err)
		}
		r.FromTier = models.StorageTierType(fromTier)
		r.ToTier = models.StorageTierType(toTier)
		if desc != nil {
			r.Description = *desc
		}
		rules = append(rules, &r)
	}
	return rules, nil
}

// GetSnapshotTier returns the tier info for a specific snapshot.
func (db *DB) GetSnapshotTier(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.SnapshotTier, error) {
	var st models.SnapshotTier
	var tierType string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, snapshot_id, repository_id, org_id, current_tier, size_bytes,
		       snapshot_time, tiered_at, created_at, updated_at
		FROM snapshot_tiers
		WHERE snapshot_id = $1 AND repository_id = $2
	`, snapshotID, repositoryID).Scan(
		&st.ID, &st.SnapshotID, &st.RepositoryID, &st.OrgID, &tierType, &st.SizeBytes,
		&st.SnapshotTime, &st.TieredAt, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get snapshot tier: %w", err)
	}
	st.CurrentTier = models.StorageTierType(tierType)
	return &st, nil
}

// GetSnapshotTierByID returns a snapshot tier by its ID.
func (db *DB) GetSnapshotTierByID(ctx context.Context, id uuid.UUID) (*models.SnapshotTier, error) {
	var st models.SnapshotTier
	var tierType string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, snapshot_id, repository_id, org_id, current_tier, size_bytes,
		       snapshot_time, tiered_at, created_at, updated_at
		FROM snapshot_tiers
		WHERE id = $1
	`, id).Scan(
		&st.ID, &st.SnapshotID, &st.RepositoryID, &st.OrgID, &tierType, &st.SizeBytes,
		&st.SnapshotTime, &st.TieredAt, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get snapshot tier by ID: %w", err)
	}
	st.CurrentTier = models.StorageTierType(tierType)
	return &st, nil
}

// CreateSnapshotTier creates a new snapshot tier record.
func (db *DB) CreateSnapshotTier(ctx context.Context, tier *models.SnapshotTier) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_tiers (id, snapshot_id, repository_id, org_id, current_tier,
		            size_bytes, snapshot_time, tiered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (snapshot_id, repository_id) DO UPDATE SET
			current_tier = EXCLUDED.current_tier,
			size_bytes = EXCLUDED.size_bytes,
			updated_at = EXCLUDED.updated_at
	`, tier.ID, tier.SnapshotID, tier.RepositoryID, tier.OrgID, string(tier.CurrentTier),
		tier.SizeBytes, tier.SnapshotTime, tier.TieredAt, tier.CreatedAt, tier.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot tier: %w", err)
	}
	return nil
}

// UpdateSnapshotTier updates an existing snapshot tier record.
func (db *DB) UpdateSnapshotTier(ctx context.Context, tier *models.SnapshotTier) error {
	tier.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_tiers SET
			current_tier = $2, size_bytes = $3, tiered_at = $4, updated_at = $5
		WHERE id = $1
	`, tier.ID, string(tier.CurrentTier), tier.SizeBytes, tier.TieredAt, tier.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update snapshot tier: %w", err)
	}
	return nil
}

// GetSnapshotsForTiering returns snapshots that are candidates for tier transition.
func (db *DB) GetSnapshotsForTiering(ctx context.Context, orgID uuid.UUID, currentTier models.StorageTierType, olderThanDays int) ([]*models.SnapshotTier, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, snapshot_id, repository_id, org_id, current_tier, size_bytes,
		       snapshot_time, tiered_at, created_at, updated_at
		FROM snapshot_tiers
		WHERE org_id = $1
		  AND current_tier = $2
		  AND tiered_at < NOW() - INTERVAL '1 day' * $3
		ORDER BY tiered_at
		LIMIT 1000
	`, orgID, string(currentTier), olderThanDays)
	if err != nil {
		return nil, fmt.Errorf("get snapshots for tiering: %w", err)
	}
	defer rows.Close()

	return scanSnapshotTiers(rows)
}

// GetSnapshotTiersByRepository returns all snapshot tiers for a repository.
func (db *DB) GetSnapshotTiersByRepository(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotTier, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, snapshot_id, repository_id, org_id, current_tier, size_bytes,
		       snapshot_time, tiered_at, created_at, updated_at
		FROM snapshot_tiers
		WHERE repository_id = $1
		ORDER BY snapshot_time DESC
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("get snapshot tiers by repository: %w", err)
	}
	defer rows.Close()

	return scanSnapshotTiers(rows)
}

// GetSnapshotTiersByOrg returns all snapshot tiers for an organization.
func (db *DB) GetSnapshotTiersByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotTier, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, snapshot_id, repository_id, org_id, current_tier, size_bytes,
		       snapshot_time, tiered_at, created_at, updated_at
		FROM snapshot_tiers
		WHERE org_id = $1
		ORDER BY snapshot_time DESC
		LIMIT 1000
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get snapshot tiers by org: %w", err)
	}
	defer rows.Close()

	return scanSnapshotTiers(rows)
}

// scanSnapshotTiers scans rows into snapshot tiers.
func scanSnapshotTiers(rows pgx.Rows) ([]*models.SnapshotTier, error) {
	var tiers []*models.SnapshotTier
	for rows.Next() {
		var st models.SnapshotTier
		var tierType string
		err := rows.Scan(
			&st.ID, &st.SnapshotID, &st.RepositoryID, &st.OrgID, &tierType, &st.SizeBytes,
			&st.SnapshotTime, &st.TieredAt, &st.CreatedAt, &st.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan snapshot tier: %w", err)
		}
		st.CurrentTier = models.StorageTierType(tierType)
		tiers = append(tiers, &st)
	}
	return tiers, nil
}

// CreateTierTransition creates a new tier transition record.
func (db *DB) CreateTierTransition(ctx context.Context, transition *models.TierTransition) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO tier_transitions (id, snapshot_tier_id, snapshot_id, repository_id, org_id,
		            from_tier, to_tier, trigger_rule_id, trigger_reason, size_bytes,
		            estimated_saving, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, transition.ID, transition.SnapshotTierID, transition.SnapshotID, transition.RepositoryID,
		transition.OrgID, string(transition.FromTier), string(transition.ToTier),
		transition.TriggerRuleID, transition.TriggerReason, transition.SizeBytes,
		transition.EstimatedSaving, transition.Status, transition.CreatedAt)
	if err != nil {
		return fmt.Errorf("create tier transition: %w", err)
	}
	return nil
}

// UpdateTierTransition updates a tier transition record.
func (db *DB) UpdateTierTransition(ctx context.Context, transition *models.TierTransition) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE tier_transitions SET
			status = $2, error_message = $3, started_at = $4, completed_at = $5
		WHERE id = $1
	`, transition.ID, transition.Status, transition.ErrorMessage,
		transition.StartedAt, transition.CompletedAt)
	if err != nil {
		return fmt.Errorf("update tier transition: %w", err)
	}
	return nil
}

// GetPendingTierTransitions returns all pending tier transitions for an organization.
func (db *DB) GetPendingTierTransitions(ctx context.Context, orgID uuid.UUID) ([]*models.TierTransition, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, snapshot_tier_id, snapshot_id, repository_id, org_id, from_tier, to_tier,
		       trigger_rule_id, trigger_reason, size_bytes, estimated_saving, status,
		       error_message, started_at, completed_at, created_at
		FROM tier_transitions
		WHERE org_id = $1 AND status IN ('pending', 'in_progress')
		ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get pending tier transitions: %w", err)
	}
	defer rows.Close()

	return scanTierTransitions(rows)
}

// GetTierTransitionHistory returns the tier transition history for a snapshot.
func (db *DB) GetTierTransitionHistory(ctx context.Context, snapshotID string, repositoryID uuid.UUID, limit int) ([]*models.TierTransition, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, snapshot_tier_id, snapshot_id, repository_id, org_id, from_tier, to_tier,
		       trigger_rule_id, trigger_reason, size_bytes, estimated_saving, status,
		       error_message, started_at, completed_at, created_at
		FROM tier_transitions
		WHERE snapshot_id = $1 AND repository_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, snapshotID, repositoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("get tier transition history: %w", err)
	}
	defer rows.Close()

	return scanTierTransitions(rows)
}

// scanTierTransitions scans rows into tier transitions.
func scanTierTransitions(rows pgx.Rows) ([]*models.TierTransition, error) {
	var transitions []*models.TierTransition
	for rows.Next() {
		var t models.TierTransition
		var fromTier, toTier string
		var errMsg *string
		err := rows.Scan(
			&t.ID, &t.SnapshotTierID, &t.SnapshotID, &t.RepositoryID, &t.OrgID,
			&fromTier, &toTier, &t.TriggerRuleID, &t.TriggerReason, &t.SizeBytes,
			&t.EstimatedSaving, &t.Status, &errMsg, &t.StartedAt, &t.CompletedAt, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tier transition: %w", err)
		}
		t.FromTier = models.StorageTierType(fromTier)
		t.ToTier = models.StorageTierType(toTier)
		if errMsg != nil {
			t.ErrorMessage = *errMsg
		}
		transitions = append(transitions, &t)
	}
	return transitions, nil
}

// CreateColdRestoreRequest creates a new cold restore request.
func (db *DB) CreateColdRestoreRequest(ctx context.Context, req *models.ColdRestoreRequest) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO cold_restore_requests (id, org_id, snapshot_id, repository_id, requested_by,
		            from_tier, target_path, priority, status, estimated_ready_at, retrieval_cost,
		            created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, req.ID, req.OrgID, req.SnapshotID, req.RepositoryID, req.RequestedBy,
		string(req.FromTier), req.TargetPath, req.Priority, req.Status,
		req.EstimatedReady, req.RetrievalCost, req.CreatedAt, req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create cold restore request: %w", err)
	}
	return nil
}

// UpdateColdRestoreRequest updates a cold restore request.
func (db *DB) UpdateColdRestoreRequest(ctx context.Context, req *models.ColdRestoreRequest) error {
	req.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE cold_restore_requests SET
			status = $2, estimated_ready_at = $3, ready_at = $4, expires_at = $5,
			error_message = $6, updated_at = $7
		WHERE id = $1
	`, req.ID, req.Status, req.EstimatedReady, req.ReadyAt, req.ExpiresAt,
		req.ErrorMessage, req.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update cold restore request: %w", err)
	}
	return nil
}

// GetColdRestoreRequest returns a cold restore request by ID.
func (db *DB) GetColdRestoreRequest(ctx context.Context, id uuid.UUID) (*models.ColdRestoreRequest, error) {
	var req models.ColdRestoreRequest
	var fromTier string
	var targetPath, errMsg *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_id, repository_id, requested_by, from_tier, target_path,
		       priority, status, estimated_ready_at, ready_at, expires_at, error_message,
		       retrieval_cost, created_at, updated_at
		FROM cold_restore_requests
		WHERE id = $1
	`, id).Scan(
		&req.ID, &req.OrgID, &req.SnapshotID, &req.RepositoryID, &req.RequestedBy,
		&fromTier, &targetPath, &req.Priority, &req.Status, &req.EstimatedReady,
		&req.ReadyAt, &req.ExpiresAt, &errMsg, &req.RetrievalCost, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get cold restore request: %w", err)
	}
	req.FromTier = models.StorageTierType(fromTier)
	if targetPath != nil {
		req.TargetPath = *targetPath
	}
	if errMsg != nil {
		req.ErrorMessage = *errMsg
	}
	return &req, nil
}

// GetColdRestoreRequestBySnapshot returns an active cold restore request for a snapshot.
func (db *DB) GetColdRestoreRequestBySnapshot(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.ColdRestoreRequest, error) {
	var req models.ColdRestoreRequest
	var fromTier string
	var targetPath, errMsg *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, snapshot_id, repository_id, requested_by, from_tier, target_path,
		       priority, status, estimated_ready_at, ready_at, expires_at, error_message,
		       retrieval_cost, created_at, updated_at
		FROM cold_restore_requests
		WHERE snapshot_id = $1 AND repository_id = $2 AND status NOT IN ('completed', 'failed', 'expired')
		ORDER BY created_at DESC
		LIMIT 1
	`, snapshotID, repositoryID).Scan(
		&req.ID, &req.OrgID, &req.SnapshotID, &req.RepositoryID, &req.RequestedBy,
		&fromTier, &targetPath, &req.Priority, &req.Status, &req.EstimatedReady,
		&req.ReadyAt, &req.ExpiresAt, &errMsg, &req.RetrievalCost, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get cold restore request by snapshot: %w", err)
	}
	req.FromTier = models.StorageTierType(fromTier)
	if targetPath != nil {
		req.TargetPath = *targetPath
	}
	if errMsg != nil {
		req.ErrorMessage = *errMsg
	}
	return &req, nil
}

// GetPendingColdRestoreRequests returns all pending cold restore requests for an organization.
func (db *DB) GetPendingColdRestoreRequests(ctx context.Context, orgID uuid.UUID) ([]*models.ColdRestoreRequest, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, snapshot_id, repository_id, requested_by, from_tier, target_path,
		       priority, status, estimated_ready_at, ready_at, expires_at, error_message,
		       retrieval_cost, created_at, updated_at
		FROM cold_restore_requests
		WHERE org_id = $1 AND status IN ('pending', 'warming')
		ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get pending cold restore requests: %w", err)
	}
	defer rows.Close()

	return scanColdRestoreRequests(rows)
}

// GetActiveColdRestoreRequests returns all active cold restore requests for an organization.
func (db *DB) GetActiveColdRestoreRequests(ctx context.Context, orgID uuid.UUID) ([]*models.ColdRestoreRequest, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, snapshot_id, repository_id, requested_by, from_tier, target_path,
		       priority, status, estimated_ready_at, ready_at, expires_at, error_message,
		       retrieval_cost, created_at, updated_at
		FROM cold_restore_requests
		WHERE org_id = $1 AND status NOT IN ('completed', 'failed', 'expired')
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active cold restore requests: %w", err)
	}
	defer rows.Close()

	return scanColdRestoreRequests(rows)
}

// ExpireColdRestoreRequests marks ready requests as expired if past their expiration time.
func (db *DB) ExpireColdRestoreRequests(ctx context.Context) (int, error) {
	result, err := db.Pool.Exec(ctx, `
		UPDATE cold_restore_requests SET
			status = 'expired', updated_at = NOW()
		WHERE status = 'ready' AND expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("expire cold restore requests: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// scanColdRestoreRequests scans rows into cold restore requests.
func scanColdRestoreRequests(rows pgx.Rows) ([]*models.ColdRestoreRequest, error) {
	var requests []*models.ColdRestoreRequest
	for rows.Next() {
		var req models.ColdRestoreRequest
		var fromTier string
		var targetPath, errMsg *string
		err := rows.Scan(
			&req.ID, &req.OrgID, &req.SnapshotID, &req.RepositoryID, &req.RequestedBy,
			&fromTier, &targetPath, &req.Priority, &req.Status, &req.EstimatedReady,
			&req.ReadyAt, &req.ExpiresAt, &errMsg, &req.RetrievalCost, &req.CreatedAt, &req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cold restore request: %w", err)
		}
		req.FromTier = models.StorageTierType(fromTier)
		if targetPath != nil {
			req.TargetPath = *targetPath
		}
		if errMsg != nil {
			req.ErrorMessage = *errMsg
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

// CreateTierCostReport creates a new tier cost report.
func (db *DB) CreateTierCostReport(ctx context.Context, report *models.TierCostReport) error {
	breakdownJSON, err := json.Marshal(report.TierBreakdown)
	if err != nil {
		return fmt.Errorf("marshal tier breakdown: %w", err)
	}
	suggestionsJSON, err := json.Marshal(report.Suggestions)
	if err != nil {
		return fmt.Errorf("marshal suggestions: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO tier_cost_reports (id, org_id, report_date, total_size_bytes,
		            current_monthly_cost, optimized_monthly_cost, potential_monthly_savings,
		            tier_breakdown, suggestions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, report_date) DO UPDATE SET
			total_size_bytes = EXCLUDED.total_size_bytes,
			current_monthly_cost = EXCLUDED.current_monthly_cost,
			optimized_monthly_cost = EXCLUDED.optimized_monthly_cost,
			potential_monthly_savings = EXCLUDED.potential_monthly_savings,
			tier_breakdown = EXCLUDED.tier_breakdown,
			suggestions = EXCLUDED.suggestions
	`, report.ID, report.OrgID, report.ReportDate.Format("2006-01-02"), report.TotalSize,
		report.CurrentCost, report.OptimizedCost, report.PotentialSave,
		breakdownJSON, suggestionsJSON, report.CreatedAt)
	if err != nil {
		return fmt.Errorf("create tier cost report: %w", err)
	}
	return nil
}

// GetLatestTierCostReport returns the most recent cost report for an organization.
func (db *DB) GetLatestTierCostReport(ctx context.Context, orgID uuid.UUID) (*models.TierCostReport, error) {
	var report models.TierCostReport
	var breakdownJSON, suggestionsJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, report_date, total_size_bytes, current_monthly_cost,
		       optimized_monthly_cost, potential_monthly_savings, tier_breakdown,
		       suggestions, created_at
		FROM tier_cost_reports
		WHERE org_id = $1
		ORDER BY report_date DESC
		LIMIT 1
	`, orgID).Scan(
		&report.ID, &report.OrgID, &report.ReportDate, &report.TotalSize,
		&report.CurrentCost, &report.OptimizedCost, &report.PotentialSave,
		&breakdownJSON, &suggestionsJSON, &report.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest tier cost report: %w", err)
	}

	if err := json.Unmarshal(breakdownJSON, &report.TierBreakdown); err != nil {
		return nil, fmt.Errorf("unmarshal tier breakdown: %w", err)
	}
	if err := json.Unmarshal(suggestionsJSON, &report.Suggestions); err != nil {
		return nil, fmt.Errorf("unmarshal suggestions: %w", err)
	}

	return &report, nil
}

// GetTierCostReports returns recent cost reports for an organization.
func (db *DB) GetTierCostReports(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.TierCostReport, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, report_date, total_size_bytes, current_monthly_cost,
		       optimized_monthly_cost, potential_monthly_savings, tier_breakdown,
		       suggestions, created_at
		FROM tier_cost_reports
		WHERE org_id = $1
		ORDER BY report_date DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("get tier cost reports: %w", err)
	}
	defer rows.Close()

	var reports []*models.TierCostReport
	for rows.Next() {
		var report models.TierCostReport
		var breakdownJSON, suggestionsJSON []byte
		err := rows.Scan(
			&report.ID, &report.OrgID, &report.ReportDate, &report.TotalSize,
			&report.CurrentCost, &report.OptimizedCost, &report.PotentialSave,
			&breakdownJSON, &suggestionsJSON, &report.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tier cost report: %w", err)
		}

		if err := json.Unmarshal(breakdownJSON, &report.TierBreakdown); err != nil {
			return nil, fmt.Errorf("unmarshal tier breakdown: %w", err)
		}
		if err := json.Unmarshal(suggestionsJSON, &report.Suggestions); err != nil {
			return nil, fmt.Errorf("unmarshal suggestions: %w", err)
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

// GetTierStatsSummary returns aggregate tier statistics for an organization.
func (db *DB) GetTierStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.TierStatsSummary, error) {
	// Get tier configurations for cost calculation
	configs, err := db.GetStorageTierConfigs(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get tier configs: %w", err)
	}

	costMap := make(map[models.StorageTierType]float64)
	for _, c := range configs {
		costMap[c.TierType] = c.CostPerGBMonth
	}

	// Default costs if no configs
	if len(costMap) == 0 {
		costMap[models.StorageTierHot] = 0.023
		costMap[models.StorageTierWarm] = 0.0125
		costMap[models.StorageTierCold] = 0.004
		costMap[models.StorageTierArchive] = 0.00099
	}

	// Get statistics by tier
	rows, err := db.Pool.Query(ctx, `
		SELECT current_tier,
		       COUNT(*) as snapshot_count,
		       COALESCE(SUM(size_bytes), 0) as total_size,
		       COALESCE(EXTRACT(DAY FROM NOW() - MIN(snapshot_time)), 0) as oldest_days,
		       COALESCE(EXTRACT(DAY FROM NOW() - MAX(snapshot_time)), 0) as newest_days
		FROM snapshot_tiers
		WHERE org_id = $1
		GROUP BY current_tier
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get tier stats: %w", err)
	}
	defer rows.Close()

	summary := &models.TierStatsSummary{
		ByTier: make(map[models.StorageTierType]models.TierStats),
	}

	for rows.Next() {
		var tierType string
		var stats models.TierStats
		err := rows.Scan(&tierType, &stats.SnapshotCount, &stats.TotalSizeBytes, &stats.OldestDays, &stats.NewestDays)
		if err != nil {
			return nil, fmt.Errorf("scan tier stats: %w", err)
		}

		tier := models.StorageTierType(tierType)
		sizeGB := float64(stats.TotalSizeBytes) / (1024 * 1024 * 1024)
		stats.MonthlyCost = sizeGB * costMap[tier]

		summary.ByTier[tier] = stats
		summary.TotalSnapshots += stats.SnapshotCount
		summary.TotalSizeBytes += stats.TotalSizeBytes
		summary.EstimatedMonthlyCost += stats.MonthlyCost
	}

	// Calculate potential savings (if all hot data older than 30 days moved to warm)
	if hotStats, ok := summary.ByTier[models.StorageTierHot]; ok && hotStats.OldestDays > 30 {
		// Rough estimate: assume 50% of hot data could be moved to warm
		hotSizeGB := float64(hotStats.TotalSizeBytes) / (1024 * 1024 * 1024) * 0.5
		summary.PotentialSavings = hotSizeGB * (costMap[models.StorageTierHot] - costMap[models.StorageTierWarm])
	}

	return summary, nil
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


// DeleteBackup soft-deletes a backup record by setting deleted_at.
func (db *DB) DeleteBackup(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE backups SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete backup: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("backup not found")
	}
	return nil
}


// DeleteRestore soft-deletes a restore record by setting deleted_at.
func (db *DB) DeleteRestore(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE restores SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete restore: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("restore not found")
	}
	return nil
}


// DeleteVerification soft-deletes a verification record by setting deleted_at.
func (db *DB) DeleteVerification(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE verifications SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete verification: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("verification not found")
	}
	return nil
}


// UpdateSnapshotComment updates a snapshot comment's content.
func (db *DB) UpdateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error {
	comment.UpdatedAt = time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_comments
		SET content = $2, updated_at = $3
		WHERE id = $1
	`, comment.ID, comment.Content, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update snapshot comment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot comment not found")
	}
	return nil
}


// DeleteDRTest soft-deletes a DR test record by setting deleted_at.
func (db *DB) DeleteDRTest(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE dr_tests SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("soft delete DR test: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("DR test not found")
	}
	return nil
}


// GetBackupsByOrgID returns all non-deleted backups for an organization.
func (db *DB) GetBackupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.started_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get backups by org: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}


// CreateOrUpdateDailySummary upserts a daily summary record for an organization and date.
func (db *DB) CreateOrUpdateDailySummary(ctx context.Context, summary *models.MetricsDailySummary) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO metrics_daily_summary (
			id, org_id, date, backups_total, backups_successful, backups_failed,
			total_backup_size, total_duration_secs, agents_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (org_id, date) DO UPDATE SET
			backups_total = EXCLUDED.backups_total,
			backups_successful = EXCLUDED.backups_successful,
			backups_failed = EXCLUDED.backups_failed,
			total_backup_size = EXCLUDED.total_backup_size,
			total_duration_secs = EXCLUDED.total_duration_secs,
			agents_active = EXCLUDED.agents_active,
			updated_at = EXCLUDED.updated_at
	`, summary.ID, summary.OrgID, summary.Date,
		summary.TotalBackups, summary.SuccessfulBackups, summary.FailedBackups,
		summary.TotalSizeBytes, summary.TotalDurationSecs, summary.AgentsActive,
		summary.CreatedAt, summary.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert daily summary: %w", err)
	}
	return nil
}


// GetDailySummary returns a daily summary for a specific organization and date.
func (db *DB) GetDailySummary(ctx context.Context, orgID uuid.UUID, date time.Time) (*models.MetricsDailySummary, error) {
	var s models.MetricsDailySummary
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, date, backups_total, backups_successful, backups_failed,
		       total_backup_size, total_duration_secs, agents_active,
		       created_at, updated_at
		FROM metrics_daily_summary
		WHERE org_id = $1 AND date = $2
	`, orgID, date).Scan(
		&s.ID, &s.OrgID, &s.Date,
		&s.TotalBackups, &s.SuccessfulBackups, &s.FailedBackups,
		&s.TotalSizeBytes, &s.TotalDurationSecs, &s.AgentsActive,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get daily summary: %w", err)
	}
	return &s, nil
}


// GetDailySummaries returns daily summaries for an organization within a date range.
func (db *DB) GetDailySummaries(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]models.MetricsDailySummary, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, date, backups_total, backups_successful, backups_failed,
		       total_backup_size, total_duration_secs, agents_active,
		       created_at, updated_at
		FROM metrics_daily_summary
		WHERE org_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get daily summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.MetricsDailySummary
	for rows.Next() {
		var s models.MetricsDailySummary
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Date,
			&s.TotalBackups, &s.SuccessfulBackups, &s.FailedBackups,
			&s.TotalSizeBytes, &s.TotalDurationSecs, &s.AgentsActive,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan daily summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}


// CreateSLAPolicy inserts a new SLA policy.
func (db *DB) CreateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_policies (id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, policy.ID, policy.OrgID, policy.Name, policy.Description, policy.TargetRPOHours, policy.TargetRTOHours, policy.TargetSuccessRate, policy.Enabled, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create SLA policy: %w", err)
	}
	return nil
}


// GetSLAPolicyByID returns a single SLA policy by ID.
func (db *DB) GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*models.SLAPolicy, error) {
	var p models.SLAPolicy
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at
		FROM sla_policies
		WHERE id = $1
	`, id).Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.TargetRPOHours, &p.TargetRTOHours, &p.TargetSuccessRate, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get SLA policy: %w", err)
	}
	return &p, nil
}


// ListSLAPoliciesByOrgID returns all SLA policies for an organization.
func (db *DB) ListSLAPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SLAPolicy, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at
		FROM sla_policies
		WHERE org_id = $1
		ORDER BY name ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list SLA policies: %w", err)
	}
	defer rows.Close()

	var policies []*models.SLAPolicy
	for rows.Next() {
		var p models.SLAPolicy
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.TargetRPOHours, &p.TargetRTOHours, &p.TargetSuccessRate, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan SLA policy: %w", err)
		}
		policies = append(policies, &p)
	}
	return policies, nil
}


// UpdateSLAPolicy updates an existing SLA policy.
func (db *DB) UpdateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	policy.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE sla_policies
		SET name = $2, description = $3, target_rpo_hours = $4, target_rto_hours = $5, target_success_rate = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, policy.ID, policy.Name, policy.Description, policy.TargetRPOHours, policy.TargetRTOHours, policy.TargetSuccessRate, policy.Enabled, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update SLA policy: %w", err)
	}
	return nil
}


// Lifecycle Policy methods

// CreateLifecyclePolicy creates a new lifecycle policy.
func (db *DB) CreateLifecyclePolicy(ctx context.Context, policy *models.LifecyclePolicy) error {
	rulesJSON, err := policy.RulesJSON()
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}

	repoIDsJSON, err := policy.RepositoryIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal repository_ids: %w", err)
	}

	scheduleIDsJSON, err := policy.ScheduleIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal schedule_ids: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO lifecycle_policies (
			id, org_id, name, description, status, rules, repository_ids, schedule_ids,
			deletion_count, bytes_reclaimed, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, policy.ID, policy.OrgID, policy.Name, policy.Description, policy.Status,
		rulesJSON, repoIDsJSON, scheduleIDsJSON, policy.DeletionCount, policy.BytesReclaimed,
		policy.CreatedBy, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create lifecycle policy: %w", err)
	}
	return nil
}

// GetLifecyclePolicyByID returns a lifecycle policy by ID.
func (db *DB) GetLifecyclePolicyByID(ctx context.Context, id uuid.UUID) (*models.LifecyclePolicy, error) {
	var p models.LifecyclePolicy
	var rulesJSON, repoIDsJSON, scheduleIDsJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, status, rules, repository_ids, schedule_ids,
			last_evaluated_at, last_deletion_at, deletion_count, bytes_reclaimed,
			created_by, created_at, updated_at
		FROM lifecycle_policies
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.Description, &p.Status, &rulesJSON, &repoIDsJSON, &scheduleIDsJSON,
		&p.LastEvaluatedAt, &p.LastDeletionAt, &p.DeletionCount, &p.BytesReclaimed,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get lifecycle policy: %w", err)
	}

	if err := p.SetRules(rulesJSON); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	if err := p.SetRepositoryIDs(repoIDsJSON); err != nil {
		return nil, fmt.Errorf("parse repository_ids: %w", err)
	}
	if err := p.SetScheduleIDs(scheduleIDsJSON); err != nil {
		return nil, fmt.Errorf("parse schedule_ids: %w", err)
	}

	return &p, nil
}

// GetLifecyclePoliciesByOrgID returns all lifecycle policies for an organization.
func (db *DB) GetLifecyclePoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LifecyclePolicy, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, status, rules, repository_ids, schedule_ids,
			last_evaluated_at, last_deletion_at, deletion_count, bytes_reclaimed,
			created_by, created_at, updated_at
		FROM lifecycle_policies
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle policies: %w", err)
	}
	defer rows.Close()

	return scanLifecyclePolicies(rows)
}

// GetActiveLifecyclePoliciesByOrgID returns all active lifecycle policies for an organization.
func (db *DB) GetActiveLifecyclePoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LifecyclePolicy, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, status, rules, repository_ids, schedule_ids,
			last_evaluated_at, last_deletion_at, deletion_count, bytes_reclaimed,
			created_by, created_at, updated_at
		FROM lifecycle_policies
		WHERE org_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active lifecycle policies: %w", err)
	}
	defer rows.Close()

	return scanLifecyclePolicies(rows)
}

// UpdateLifecyclePolicy updates a lifecycle policy.
func (db *DB) UpdateLifecyclePolicy(ctx context.Context, policy *models.LifecyclePolicy) error {
	rulesJSON, err := policy.RulesJSON()
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}

	repoIDsJSON, err := policy.RepositoryIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal repository_ids: %w", err)
	}

	scheduleIDsJSON, err := policy.ScheduleIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal schedule_ids: %w", err)
	}

	policy.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE lifecycle_policies
		SET name = $2, description = $3, status = $4, rules = $5, repository_ids = $6, schedule_ids = $7,
			last_evaluated_at = $8, last_deletion_at = $9, deletion_count = $10, bytes_reclaimed = $11,
			updated_at = $12
		WHERE id = $1
	`, policy.ID, policy.Name, policy.Description, policy.Status, rulesJSON, repoIDsJSON, scheduleIDsJSON,
		policy.LastEvaluatedAt, policy.LastDeletionAt, policy.DeletionCount, policy.BytesReclaimed,
		policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update lifecycle policy: %w", err)
	}
	return nil
}


// DeleteSLAPolicy deletes an SLA policy by ID.
func (db *DB) DeleteSLAPolicy(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM sla_policies WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete SLA policy: %w", err)
	}
	return nil
}


// DeleteLifecyclePolicy deletes a lifecycle policy by ID.
func (db *DB) DeleteLifecyclePolicy(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM lifecycle_policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lifecycle policy: %w", err)
	}
	return nil
}


// CreateSLAStatusSnapshot inserts a new SLA status history record.
func (db *DB) CreateSLAStatusSnapshot(ctx context.Context, snapshot *models.SLAStatusSnapshot) error {
	snapshot.ID = uuid.New()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_status_history (id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, snapshot.ID, snapshot.PolicyID, snapshot.RPOHours, snapshot.RTOHours, snapshot.SuccessRate, snapshot.Compliant, snapshot.CalculatedAt)
	if err != nil {
		return fmt.Errorf("create SLA status snapshot: %w", err)
	}
	return nil
}


// CreateLifecycleDeletionEvent creates a deletion event for audit logging.
func (db *DB) CreateLifecycleDeletionEvent(ctx context.Context, event *models.LifecycleDeletionEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO lifecycle_deletion_events (
			id, org_id, policy_id, snapshot_id, repository_id, reason, size_bytes, deleted_by, deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.OrgID, event.PolicyID, event.SnapshotID, event.RepositoryID,
		event.Reason, event.SizeBytes, event.DeletedBy, event.DeletedAt)
	if err != nil {
		return fmt.Errorf("create lifecycle deletion event: %w", err)
	}
	return nil
}


// GetSLAStatusHistory returns SLA status history for a policy, ordered by most recent first.
func (db *DB) GetSLAStatusHistory(ctx context.Context, policyID uuid.UUID, limit int) ([]*models.SLAStatusSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at
		FROM sla_status_history
		WHERE policy_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2
	`, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("get SLA status history: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.SLAStatusSnapshot
	for rows.Next() {
		var s models.SLAStatusSnapshot
		if err := rows.Scan(&s.ID, &s.PolicyID, &s.RPOHours, &s.RTOHours, &s.SuccessRate, &s.Compliant, &s.CalculatedAt); err != nil {
			return nil, fmt.Errorf("scan SLA status snapshot: %w", err)
		}
		snapshots = append(snapshots, &s)
	}
	return snapshots, nil
}

// GetLifecycleDeletionEventsByPolicyID returns deletion events for a policy.
func (db *DB) GetLifecycleDeletionEventsByPolicyID(ctx context.Context, policyID uuid.UUID, limit int) ([]*models.LifecycleDeletionEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, policy_id, snapshot_id, repository_id, reason, size_bytes, deleted_by, deleted_at
		FROM lifecycle_deletion_events
		WHERE policy_id = $1
		ORDER BY deleted_at DESC
		LIMIT $2
	`, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("list deletion events: %w", err)
	}
	defer rows.Close()

	return scanLifecycleDeletionEvents(rows)
}


// GetLatestSLAStatus returns the most recent SLA status snapshot for a policy.
func (db *DB) GetLatestSLAStatus(ctx context.Context, policyID uuid.UUID) (*models.SLAStatusSnapshot, error) {
	var s models.SLAStatusSnapshot
	err := db.Pool.QueryRow(ctx, `
		SELECT id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at
		FROM sla_status_history
		WHERE policy_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`, policyID).Scan(&s.ID, &s.PolicyID, &s.RPOHours, &s.RTOHours, &s.SuccessRate, &s.Compliant, &s.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("get latest SLA status: %w", err)
	}
	return &s, nil
}


// GetBackupSuccessRateForOrg calculates the backup success rate for an org over a given number of hours.
func (db *DB) GetBackupSuccessRateForOrg(ctx context.Context, orgID uuid.UUID, hours int) (float64, error) {
	var total, successful int
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed')
		FROM backups
		WHERE org_id = $1 AND created_at >= NOW() - ($2 || ' hours')::INTERVAL
	`, orgID, fmt.Sprintf("%d", hours)).Scan(&total, &successful)
	if err != nil {
		return 0, fmt.Errorf("get backup success rate: %w", err)
	}
	if total == 0 {
		return 100, nil
	}
	return float64(successful) / float64(total) * 100, nil
}


// GetMaxRPOHoursForOrg returns the maximum hours since the last successful backup across all agents in an org.
func (db *DB) GetMaxRPOHoursForOrg(ctx context.Context, orgID uuid.UUID) (float64, error) {
	var maxHours float64
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(EXTRACT(EPOCH FROM (NOW() - last_backup))/3600), 0)
		FROM (
			SELECT a.id, MAX(b.completed_at) as last_backup
			FROM agents a
			LEFT JOIN backups b ON b.agent_id = a.id AND b.status = 'completed'
			WHERE a.org_id = $1 AND a.status = 'active'
			GROUP BY a.id
		) sub
		WHERE last_backup IS NOT NULL
	`, orgID).Scan(&maxHours)
	if err != nil {
		return 0, fmt.Errorf("get max RPO hours: %w", err)
	}
	return maxHours, nil
}


// GetBrandingSettings returns the branding settings for the given organization.
// Returns nil if no custom branding has been configured.
func (db *DB) GetBrandingSettings(ctx context.Context, orgID uuid.UUID) (*models.BrandingSettings, error) {
	var b models.BrandingSettings
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, logo_url, favicon_url, product_name,
		       primary_color, secondary_color, support_url, custom_css,
		       created_at, updated_at
		FROM branding_settings
		WHERE org_id = $1
	`, orgID).Scan(
		&b.ID, &b.OrgID, &b.LogoURL, &b.FaviconURL, &b.ProductName,
		&b.PrimaryColor, &b.SecondaryColor, &b.SupportURL, &b.CustomCSS,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get branding settings: %w", err)
	}
	return &b, nil
}


// UpsertBrandingSettings creates or updates branding settings for an organization.
func (db *DB) UpsertBrandingSettings(ctx context.Context, b *models.BrandingSettings) error {
	b.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO branding_settings (id, org_id, logo_url, favicon_url, product_name,
		    primary_color, secondary_color, support_url, custom_css, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (org_id) DO UPDATE SET
		    logo_url = EXCLUDED.logo_url,
		    favicon_url = EXCLUDED.favicon_url,
		    product_name = EXCLUDED.product_name,
		    primary_color = EXCLUDED.primary_color,
		    secondary_color = EXCLUDED.secondary_color,
		    support_url = EXCLUDED.support_url,
		    custom_css = EXCLUDED.custom_css,
		    updated_at = EXCLUDED.updated_at
	`, b.ID, b.OrgID, b.LogoURL, b.FaviconURL, b.ProductName,
		b.PrimaryColor, b.SecondaryColor, b.SupportURL, b.CustomCSS,
		b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert branding settings: %w", err)
	}
	return nil
}


// DeleteBrandingSettings removes branding settings for an organization.
func (db *DB) DeleteBrandingSettings(ctx context.Context, orgID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM branding_settings WHERE org_id = $1
	`, orgID)
	if err != nil {
		return fmt.Errorf("delete branding settings: %w", err)
	}
	return nil
}


// GetDockerContainers returns Docker containers for the given agent.
// This queries the agent's reported container state from the database.
func (db *DB) GetDockerContainers(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) ([]models.DockerContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT container_id, name, image, status, state, created, ports
		FROM docker_containers
		WHERE org_id = $1 AND agent_id = $2
		ORDER BY name ASC
	`, orgID, agentID)
	if err != nil {
		return nil, fmt.Errorf("get docker containers: %w", err)
	}
	defer rows.Close()

	var containers []models.DockerContainer
	for rows.Next() {
		var c models.DockerContainer
		var portsJSON []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Image, &c.Status, &c.State, &c.Created, &portsJSON); err != nil {
			return nil, fmt.Errorf("scan docker container: %w", err)
		}
		if portsJSON != nil {
			if err := json.Unmarshal(portsJSON, &c.Ports); err != nil {
				return nil, fmt.Errorf("unmarshal ports: %w", err)
			}
		}
		containers = append(containers, c)
	}
	return containers, nil
}


// GetDockerVolumes returns Docker volumes for the given agent.
func (db *DB) GetDockerVolumes(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) ([]models.DockerVolume, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT name, driver, mountpoint, size_bytes, created
		FROM docker_volumes
		WHERE org_id = $1 AND agent_id = $2
		ORDER BY name ASC
	`, orgID, agentID)
	if err != nil {
		return nil, fmt.Errorf("get docker volumes: %w", err)
	}
	defer rows.Close()

	var volumes []models.DockerVolume
	for rows.Next() {
		var v models.DockerVolume
		if err := rows.Scan(&v.Name, &v.Driver, &v.Mountpoint, &v.SizeBytes, &v.Created); err != nil {
			return nil, fmt.Errorf("scan docker volume: %w", err)
		}
		volumes = append(volumes, v)
	}
	return volumes, nil
}


// GetDockerDaemonStatus returns the Docker daemon status for the given agent.
func (db *DB) GetDockerDaemonStatus(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) (*models.DockerDaemonStatus, error) {
	var s models.DockerDaemonStatus
	err := db.Pool.QueryRow(ctx, `
		SELECT available, version, container_count, volume_count, server_os, docker_root_dir, storage_driver
		FROM docker_daemon_status
		WHERE org_id = $1 AND agent_id = $2
	`, orgID, agentID).Scan(&s.Available, &s.Version, &s.ContainerCount, &s.VolumeCount, &s.ServerOS, &s.DockerRootDir, &s.StorageDriver)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &models.DockerDaemonStatus{Available: false}, nil
		}
		return nil, fmt.Errorf("get docker daemon status: %w", err)
	}
	return &s, nil
}


// UpsertDockerContainers replaces all Docker containers for the given agent.
func (db *DB) UpsertDockerContainers(ctx context.Context, orgID, agentID uuid.UUID, containers []models.DockerContainer) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// Remove stale containers
		if _, err := tx.Exec(ctx, `DELETE FROM docker_containers WHERE org_id = $1 AND agent_id = $2`, orgID, agentID); err != nil {
			return fmt.Errorf("delete old containers: %w", err)
		}
		for _, c := range containers {
			portsJSON, err := json.Marshal(c.Ports)
			if err != nil {
				return fmt.Errorf("marshal ports: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO docker_containers (container_id, org_id, agent_id, name, image, status, state, created, ports, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			`, c.ID, orgID, agentID, c.Name, c.Image, c.Status, c.State, c.Created, portsJSON); err != nil {
				return fmt.Errorf("insert container: %w", err)
			}
		}
		return nil
	})
}


// UpsertDockerVolumes replaces all Docker volumes for the given agent.
func (db *DB) UpsertDockerVolumes(ctx context.Context, orgID, agentID uuid.UUID, volumes []models.DockerVolume) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM docker_volumes WHERE org_id = $1 AND agent_id = $2`, orgID, agentID); err != nil {
			return fmt.Errorf("delete old volumes: %w", err)
		}
		for _, v := range volumes {
			if _, err := tx.Exec(ctx, `
				INSERT INTO docker_volumes (name, org_id, agent_id, driver, mountpoint, size_bytes, created, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			`, v.Name, orgID, agentID, v.Driver, v.Mountpoint, v.SizeBytes, v.Created); err != nil {
				return fmt.Errorf("insert volume: %w", err)
			}
		}
		return nil
	})
}


// UpsertDockerDaemonStatus creates or updates the Docker daemon status for an agent.
func (db *DB) UpsertDockerDaemonStatus(ctx context.Context, orgID, agentID uuid.UUID, status *models.DockerDaemonStatus) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO docker_daemon_status (org_id, agent_id, available, version, container_count, volume_count, server_os, docker_root_dir, storage_driver, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (org_id, agent_id) DO UPDATE SET
			available = EXCLUDED.available,
			version = EXCLUDED.version,
			container_count = EXCLUDED.container_count,
			volume_count = EXCLUDED.volume_count,
			server_os = EXCLUDED.server_os,
			docker_root_dir = EXCLUDED.docker_root_dir,
			storage_driver = EXCLUDED.storage_driver,
			updated_at = NOW()
	`, orgID, agentID, status.Available, status.Version, status.ContainerCount, status.VolumeCount, status.ServerOS, status.DockerRootDir, status.StorageDriver)
	if err != nil {
		return fmt.Errorf("upsert docker daemon status: %w", err)
	}
	return nil
}


// CreateDockerBackup creates a new Docker backup job.
func (db *DB) CreateDockerBackup(ctx context.Context, orgID uuid.UUID, req *models.DockerBackupParams) (*models.DockerBackupResult, error) {
	containerIDs, err := json.Marshal(req.ContainerIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal container ids: %w", err)
	}
	volumeNames, err := json.Marshal(req.VolumeNames)
	if err != nil {
		return nil, fmt.Errorf("marshal volume names: %w", err)
	}

	var result models.DockerBackupResult
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO docker_backups (org_id, agent_id, repository_id, container_ids, volume_names, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'queued', NOW())
		RETURNING id, status, created_at
	`, orgID, req.AgentID, req.RepositoryID, containerIDs, volumeNames).Scan(&result.ID, &result.Status, &result.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create docker backup: %w", err)
	}
	return &result, nil
}


// CleanupAgentHealthHistory deletes health history records older than the
// specified retention period. Returns the number of rows deleted.
func (db *DB) CleanupAgentHealthHistory(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_health_history
		WHERE recorded_at < NOW() - ($1 * INTERVAL '1 day')
	`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup agent health history: %w", err)
	}
	return tag.RowsAffected(), nil
}


// CountAgentsByOrgID returns the number of agents for an organization.
func (db *DB) CountAgentsByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE org_id = $1`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents by org: %w", err)
	}
	return count, nil
}


// CountUsersByOrgID returns the number of users in an organization.
func (db *DB) CountUsersByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM org_members WHERE org_id = $1`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users by org: %w", err)
	}
	return count, nil
}


// CountOrganizations returns the total number of organizations.
func (db *DB) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, nil
}


// OrgCount implements the license.OrgCounter interface.
func (db *DB) OrgCount(ctx context.Context) (int, error) {
	return db.CountOrganizations(ctx)
}


// GetServerSetting retrieves a server setting by key.
func (db *DB) GetServerSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := db.Pool.QueryRow(ctx, `SELECT value FROM server_settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("get server setting %s: %w", key, err)
	}
	return value, nil
}


// SetServerSetting upserts a server setting.
func (db *DB) SetServerSetting(ctx context.Context, key, value string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO server_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)
	if err != nil {
		return fmt.Errorf("set server setting %s: %w", key, err)
	}
	return nil
}


// AgentCount returns the total number of agents across all organizations.
func (db *DB) AgentCount(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count all agents: %w", err)
	}
	return count, nil
}


// UserCount returns the total number of users across all organizations.
func (db *DB) UserCount(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count all users: %w", err)
	}
	return count, nil
}

// GetLifecycleDeletionEventsByOrgID returns deletion events for an organization.
func (db *DB) GetLifecycleDeletionEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.LifecycleDeletionEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, policy_id, snapshot_id, repository_id, reason, size_bytes, deleted_by, deleted_at
		FROM lifecycle_deletion_events
		WHERE org_id = $1
		ORDER BY deleted_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list deletion events: %w", err)
	}
	defer rows.Close()

	return scanLifecycleDeletionEvents(rows)
}

// scanLifecyclePolicies scans rows into lifecycle policies.
func scanLifecyclePolicies(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.LifecyclePolicy, error) {
	var policies []*models.LifecyclePolicy
	for rows.Next() {
		var p models.LifecyclePolicy
		var rulesJSON, repoIDsJSON, scheduleIDsJSON []byte
		err := rows.Scan(
			&p.ID, &p.OrgID, &p.Name, &p.Description, &p.Status, &rulesJSON, &repoIDsJSON, &scheduleIDsJSON,
			&p.LastEvaluatedAt, &p.LastDeletionAt, &p.DeletionCount, &p.BytesReclaimed,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan lifecycle policy: %w", err)
		}

		if err := p.SetRules(rulesJSON); err != nil {
			return nil, fmt.Errorf("parse rules: %w", err)
		}
		if err := p.SetRepositoryIDs(repoIDsJSON); err != nil {
			return nil, fmt.Errorf("parse repository_ids: %w", err)
		}
		if err := p.SetScheduleIDs(scheduleIDsJSON); err != nil {
			return nil, fmt.Errorf("parse schedule_ids: %w", err)
		}

		policies = append(policies, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lifecycle policies: %w", err)
	}

	return policies, nil
}

// scanLifecycleDeletionEvents scans rows into deletion events.
func scanLifecycleDeletionEvents(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.LifecycleDeletionEvent, error) {
	var events []*models.LifecycleDeletionEvent
	for rows.Next() {
		var e models.LifecycleDeletionEvent
		err := rows.Scan(
			&e.ID, &e.OrgID, &e.PolicyID, &e.SnapshotID, &e.RepositoryID,
			&e.Reason, &e.SizeBytes, &e.DeletedBy, &e.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan deletion event: %w", err)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deletion events: %w", err)
	}

	return events, nil
}
// ============================================================================
// SLA Definitions
// ============================================================================

// GetSLADefinitionByID returns an SLA definition by ID.
func (db *DB) GetSLADefinitionByID(ctx context.Context, id uuid.UUID) (*models.SLADefinition, error) {
	var s models.SLADefinition
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, rpo_minutes, rto_minutes, uptime_percentage,
		       scope, active, created_by, created_at, updated_at
		FROM sla_definitions WHERE id = $1
	`, id).Scan(
		&s.ID, &s.OrgID, &s.Name, &s.Description, &s.RPOMinutes, &s.RTOMinutes, &s.UptimePercentage,
		&s.Scope, &s.Active, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get sla definition: %w", err)
	}
	return &s, nil
}

// ListSLADefinitionsByOrg returns all SLA definitions for an organization.
func (db *DB) ListSLADefinitionsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLADefinition, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, rpo_minutes, rto_minutes, uptime_percentage,
		       scope, active, created_by, created_at, updated_at
		FROM sla_definitions WHERE org_id = $1 ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list sla definitions: %w", err)
	}
	defer rows.Close()
	return scanSLADefinitions(rows)
}

// ListActiveSLADefinitionsByOrg returns active SLA definitions for an organization.
func (db *DB) ListActiveSLADefinitionsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLADefinition, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, rpo_minutes, rto_minutes, uptime_percentage,
		       scope, active, created_by, created_at, updated_at
		FROM sla_definitions WHERE org_id = $1 AND active = true ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active sla definitions: %w", err)
	}
	defer rows.Close()
	return scanSLADefinitions(rows)
}

// ListSLADefinitionsWithAssignments returns SLA definitions with assignment counts.
func (db *DB) ListSLADefinitionsWithAssignments(ctx context.Context, orgID uuid.UUID) ([]*models.SLAWithAssignments, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.org_id, s.name, s.description, s.rpo_minutes, s.rto_minutes, s.uptime_percentage,
		       s.scope, s.active, s.created_by, s.created_at, s.updated_at,
		       COALESCE((SELECT COUNT(*) FROM sla_assignments WHERE sla_id = s.id AND agent_id IS NOT NULL), 0) as agent_count,
		       COALESCE((SELECT COUNT(*) FROM sla_assignments WHERE sla_id = s.id AND repository_id IS NOT NULL), 0) as repo_count
		FROM sla_definitions s WHERE s.org_id = $1 ORDER BY s.name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list sla definitions with assignments: %w", err)
	}
	defer rows.Close()

	var slas []*models.SLAWithAssignments
	for rows.Next() {
		var s models.SLAWithAssignments
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Name, &s.Description, &s.RPOMinutes, &s.RTOMinutes, &s.UptimePercentage,
			&s.Scope, &s.Active, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
			&s.AgentCount, &s.RepositoryCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sla with assignments: %w", err)
		}
		slas = append(slas, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla definitions: %w", err)
	}
	return slas, nil
}

// CreateSLADefinition creates a new SLA definition.
func (db *DB) CreateSLADefinition(ctx context.Context, s *models.SLADefinition) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_definitions (id, org_id, name, description, rpo_minutes, rto_minutes, uptime_percentage,
		            scope, active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, s.ID, s.OrgID, s.Name, s.Description, s.RPOMinutes, s.RTOMinutes, s.UptimePercentage,
		s.Scope, s.Active, s.CreatedBy, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create sla definition: %w", err)
	}
	return nil
}

// UpdateSLADefinition updates an existing SLA definition.
func (db *DB) UpdateSLADefinition(ctx context.Context, s *models.SLADefinition) error {
	s.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE sla_definitions
		SET name = $2, description = $3, rpo_minutes = $4, rto_minutes = $5, uptime_percentage = $6,
		    scope = $7, active = $8, updated_at = $9
		WHERE id = $1
	`, s.ID, s.Name, s.Description, s.RPOMinutes, s.RTOMinutes, s.UptimePercentage, s.Scope, s.Active, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update sla definition: %w", err)
	}
	return nil
}

// DeleteSLADefinition deletes an SLA definition and its assignments.
func (db *DB) DeleteSLADefinition(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sla_definitions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sla definition: %w", err)
	}
	return nil
}

// scanSLADefinitions scans rows into SLA definitions.
func scanSLADefinitions(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.SLADefinition, error) {
	var slas []*models.SLADefinition
	for rows.Next() {
		var s models.SLADefinition
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Name, &s.Description, &s.RPOMinutes, &s.RTOMinutes, &s.UptimePercentage,
			&s.Scope, &s.Active, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sla definition: %w", err)
		}
		slas = append(slas, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla definitions: %w", err)
	}
	return slas, nil
}

// ============================================================================
// SLA Assignments
// ============================================================================

// GetSLAAssignmentByID returns an SLA assignment by ID.
func (db *DB) GetSLAAssignmentByID(ctx context.Context, id uuid.UUID) (*models.SLAAssignment, error) {
	var a models.SLAAssignment
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, assigned_by, assigned_at
		FROM sla_assignments WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.SLAID, &a.AgentID, &a.RepositoryID, &a.AssignedBy, &a.AssignedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get sla assignment: %w", err)
	}
	return &a, nil
}

// ListSLAAssignmentsBySLA returns all assignments for an SLA.
func (db *DB) ListSLAAssignmentsBySLA(ctx context.Context, slaID uuid.UUID) ([]*models.SLAAssignment, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, assigned_by, assigned_at
		FROM sla_assignments WHERE sla_id = $1 ORDER BY assigned_at
	`, slaID)
		SELECT DISTINCT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error, b.created_at
		FROM backups b
		JOIN backup_tags bt ON b.id = bt.backup_id
		WHERE bt.tag_id = ANY($1) AND b.deleted_at IS NULL
		ORDER BY b.started_at DESC
	`, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("list sla assignments: %w", err)
	}
	defer rows.Close()
	return scanSLAAssignments(rows)
}

// ListSLAAssignmentsByAgent returns all SLA assignments for an agent.
func (db *DB) ListSLAAssignmentsByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.SLAAssignment, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, assigned_by, assigned_at
		FROM sla_assignments WHERE agent_id = $1 ORDER BY assigned_at
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list sla assignments by agent: %w", err)
	}
	defer rows.Close()
	return scanSLAAssignments(rows)
}

// ListSLAAssignmentsByRepository returns all SLA assignments for a repository.
func (db *DB) ListSLAAssignmentsByRepository(ctx context.Context, repoID uuid.UUID) ([]*models.SLAAssignment, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, assigned_by, assigned_at
		FROM sla_assignments WHERE repository_id = $1 ORDER BY assigned_at
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("list sla assignments by repository: %w", err)
	}
	defer rows.Close()
	return scanSLAAssignments(rows)
}

// CreateSLAAssignment creates a new SLA assignment.
func (db *DB) CreateSLAAssignment(ctx context.Context, a *models.SLAAssignment) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_assignments (id, org_id, sla_id, agent_id, repository_id, assigned_by, assigned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, a.ID, a.OrgID, a.SLAID, a.AgentID, a.RepositoryID, a.AssignedBy, a.AssignedAt)
	if err != nil {
		return fmt.Errorf("create sla assignment: %w", err)
	}
	return nil
}

// DeleteSLAAssignment deletes an SLA assignment.
func (db *DB) DeleteSLAAssignment(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sla_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sla assignment: %w", err)
	}
	return nil
}

// DeleteSLAAssignmentByAgentAndSLA removes an SLA assignment for an agent.
func (db *DB) DeleteSLAAssignmentByAgentAndSLA(ctx context.Context, agentID, slaID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sla_assignments WHERE agent_id = $1 AND sla_id = $2`, agentID, slaID)
	if err != nil {
		return fmt.Errorf("delete sla assignment by agent and sla: %w", err)
	}
	return nil
}

// DeleteSLAAssignmentByRepoAndSLA removes an SLA assignment for a repository.
func (db *DB) DeleteSLAAssignmentByRepoAndSLA(ctx context.Context, repoID, slaID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sla_assignments WHERE repository_id = $1 AND sla_id = $2`, repoID, slaID)
	if err != nil {
		return fmt.Errorf("delete sla assignment by repo and sla: %w", err)
	}
	return nil
}

// scanSLAAssignments scans rows into SLA assignments.
func scanSLAAssignments(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.SLAAssignment, error) {
	var assignments []*models.SLAAssignment
	for rows.Next() {
		var a models.SLAAssignment
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.SLAID, &a.AgentID, &a.RepositoryID, &a.AssignedBy, &a.AssignedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sla assignment: %w", err)
		}
		assignments = append(assignments, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla assignments: %w", err)
	}
	return assignments, nil
}

// ============================================================================
// SLA Compliance
// ============================================================================

// GetSLAComplianceByID returns a compliance record by ID.
func (db *DB) GetSLAComplianceByID(ctx context.Context, id uuid.UUID) (*models.SLACompliance, error) {
	var c models.SLACompliance
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, period_start, period_end,
		       rpo_compliant, rpo_actual_minutes, rpo_breaches, rto_compliant, rto_actual_minutes, rto_breaches,
		       uptime_compliant, uptime_actual_percentage, uptime_downtime_minutes, is_compliant, notes, calculated_at
		FROM sla_compliance WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.SLAID, &c.AgentID, &c.RepositoryID, &c.PeriodStart, &c.PeriodEnd,
		&c.RPOCompliant, &c.RPOActualMinutes, &c.RPOBreaches, &c.RTOCompliant, &c.RTOActualMinutes, &c.RTOBreaches,
		&c.UptimeCompliant, &c.UptimeActualPercentage, &c.UptimeDowntimeMinutes, &c.IsCompliant, &c.Notes, &c.CalculatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get sla compliance: %w", err)
	}
	return &c, nil
}

// ListSLAComplianceBySLA returns compliance records for an SLA.
func (db *DB) ListSLAComplianceBySLA(ctx context.Context, slaID uuid.UUID, limit int) ([]*models.SLACompliance, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, period_start, period_end,
		       rpo_compliant, rpo_actual_minutes, rpo_breaches, rto_compliant, rto_actual_minutes, rto_breaches,
		       uptime_compliant, uptime_actual_percentage, uptime_downtime_minutes, is_compliant, notes, calculated_at
		FROM sla_compliance WHERE sla_id = $1 ORDER BY period_end DESC LIMIT $2
	`, slaID, limit)
	if err != nil {
		return nil, fmt.Errorf("list sla compliance: %w", err)
	}
	defer rows.Close()
	return scanSLACompliance(rows)
}

// ListSLAComplianceByOrg returns compliance records for an organization in a period.
func (db *DB) ListSLAComplianceByOrg(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.SLACompliance, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, period_start, period_end,
		       rpo_compliant, rpo_actual_minutes, rpo_breaches, rto_compliant, rto_actual_minutes, rto_breaches,
		       uptime_compliant, uptime_actual_percentage, uptime_downtime_minutes, is_compliant, notes, calculated_at
		FROM sla_compliance WHERE org_id = $1 AND period_start >= $2 AND period_end <= $3 ORDER BY period_end DESC
	`, orgID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("list sla compliance by org: %w", err)
	}
	defer rows.Close()
	return scanSLACompliance(rows)
}

// CreateSLACompliance creates a new compliance record.
func (db *DB) CreateSLACompliance(ctx context.Context, c *models.SLACompliance) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_compliance (id, org_id, sla_id, agent_id, repository_id, period_start, period_end,
		            rpo_compliant, rpo_actual_minutes, rpo_breaches, rto_compliant, rto_actual_minutes, rto_breaches,
		            uptime_compliant, uptime_actual_percentage, uptime_downtime_minutes, is_compliant, notes, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, c.ID, c.OrgID, c.SLAID, c.AgentID, c.RepositoryID, c.PeriodStart, c.PeriodEnd,
		c.RPOCompliant, c.RPOActualMinutes, c.RPOBreaches, c.RTOCompliant, c.RTOActualMinutes, c.RTOBreaches,
		c.UptimeCompliant, c.UptimeActualPercentage, c.UptimeDowntimeMinutes, c.IsCompliant, c.Notes, c.CalculatedAt)
	if err != nil {
		return fmt.Errorf("create sla compliance: %w", err)
	}
	return nil
}

// scanSLACompliance scans rows into SLA compliance records.
func scanSLACompliance(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.SLACompliance, error) {
	var records []*models.SLACompliance
	for rows.Next() {
		var c models.SLACompliance
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.SLAID, &c.AgentID, &c.RepositoryID, &c.PeriodStart, &c.PeriodEnd,
			&c.RPOCompliant, &c.RPOActualMinutes, &c.RPOBreaches, &c.RTOCompliant, &c.RTOActualMinutes, &c.RTOBreaches,
			&c.UptimeCompliant, &c.UptimeActualPercentage, &c.UptimeDowntimeMinutes, &c.IsCompliant, &c.Notes, &c.CalculatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sla compliance: %w", err)
		}
		records = append(records, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla compliance: %w", err)
	}
	return records, nil
}

// ============================================================================
// SLA Breaches
// ============================================================================

// GetSLABreachByID returns a breach record by ID.
func (db *DB) GetSLABreachByID(ctx context.Context, id uuid.UUID) (*models.SLABreach, error) {
	var b models.SLABreach
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		       breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		       resolved, resolved_at, description, created_at
		FROM sla_breaches WHERE id = $1
	`, id).Scan(
		&b.ID, &b.OrgID, &b.SLAID, &b.AgentID, &b.RepositoryID, &b.BreachType, &b.ExpectedValue, &b.ActualValue,
		&b.BreachStart, &b.BreachEnd, &b.DurationMinutes, &b.Acknowledged, &b.AcknowledgedBy, &b.AcknowledgedAt,
		&b.Resolved, &b.ResolvedAt, &b.Description, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get sla breach: %w", err)
	}
	return &b, nil
}

// ListSLABreachesByOrg returns breaches for an organization.
func (db *DB) ListSLABreachesByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.SLABreach, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		       breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		       resolved, resolved_at, description, created_at
		FROM sla_breaches WHERE org_id = $1 ORDER BY breach_start DESC LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list sla breaches: %w", err)
	}
	defer rows.Close()
	return scanSLABreaches(rows)
}

// ListActiveSLABreachesByOrg returns unresolved breaches for an organization.
func (db *DB) ListActiveSLABreachesByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLABreach, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		       breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		       resolved, resolved_at, description, created_at
		FROM sla_breaches WHERE org_id = $1 AND resolved = false ORDER BY breach_start DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list active sla breaches: %w", err)
	}
	defer rows.Close()
	return scanSLABreaches(rows)
}

// ListSLABreachesBySLA returns breaches for a specific SLA.
func (db *DB) ListSLABreachesBySLA(ctx context.Context, slaID uuid.UUID, limit int) ([]*models.SLABreach, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		       breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		       resolved, resolved_at, description, created_at
		FROM sla_breaches WHERE sla_id = $1 ORDER BY breach_start DESC LIMIT $2
	`, slaID, limit)
	if err != nil {
		return nil, fmt.Errorf("list sla breaches by sla: %w", err)
	}
	defer rows.Close()
	return scanSLABreaches(rows)
}

// CountUnacknowledgedBreachesByOrg returns the count of unacknowledged breaches.
func (db *DB) CountUnacknowledgedBreachesByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sla_breaches WHERE org_id = $1 AND acknowledged = false
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unacknowledged breaches: %w", err)
	}
	return count, nil
}

// CreateSLABreach creates a new breach record.
func (db *DB) CreateSLABreach(ctx context.Context, b *models.SLABreach) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_breaches (id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		            breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		            resolved, resolved_at, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, b.ID, b.OrgID, b.SLAID, b.AgentID, b.RepositoryID, b.BreachType, b.ExpectedValue, b.ActualValue,
		b.BreachStart, b.BreachEnd, b.DurationMinutes, b.Acknowledged, b.AcknowledgedBy, b.AcknowledgedAt,
		b.Resolved, b.ResolvedAt, b.Description, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create sla breach: %w", err)
	}
	return nil
}

// UpdateSLABreach updates a breach record (for acknowledge/resolve).
func (db *DB) UpdateSLABreach(ctx context.Context, b *models.SLABreach) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE sla_breaches
		SET breach_end = $2, duration_minutes = $3, acknowledged = $4, acknowledged_by = $5, acknowledged_at = $6,
		    resolved = $7, resolved_at = $8, description = $9
		WHERE id = $1
	`, b.ID, b.BreachEnd, b.DurationMinutes, b.Acknowledged, b.AcknowledgedBy, b.AcknowledgedAt,
		b.Resolved, b.ResolvedAt, b.Description)
	if err != nil {
		return fmt.Errorf("update sla breach: %w", err)
	}
	return nil
}

// scanSLABreaches scans rows into SLA breaches.
func scanSLABreaches(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.SLABreach, error) {
	var breaches []*models.SLABreach
	for rows.Next() {
		var b models.SLABreach
		err := rows.Scan(
			&b.ID, &b.OrgID, &b.SLAID, &b.AgentID, &b.RepositoryID, &b.BreachType, &b.ExpectedValue, &b.ActualValue,
			&b.BreachStart, &b.BreachEnd, &b.DurationMinutes, &b.Acknowledged, &b.AcknowledgedBy, &b.AcknowledgedAt,
			&b.Resolved, &b.ResolvedAt, &b.Description, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sla breach: %w", err)
		}
		breaches = append(breaches, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla breaches: %w", err)
	}
	return breaches, nil
}

// GetSLADashboardStats returns dashboard statistics for SLAs.
func (db *DB) GetSLADashboardStats(ctx context.Context, orgID uuid.UUID) (*models.SLADashboardStats, error) {
	stats := &models.SLADashboardStats{}

	// Count total and active SLAs
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE active = true)
		FROM sla_definitions WHERE org_id = $1
	`, orgID).Scan(&stats.TotalSLAs, &stats.ActiveSLAs)
	if err != nil {
		return nil, fmt.Errorf("get sla counts: %w", err)
	}

	// Count active breaches and unacknowledged
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE acknowledged = false)
		FROM sla_breaches WHERE org_id = $1 AND resolved = false
	`, orgID).Scan(&stats.ActiveBreaches, &stats.UnacknowledgedCount)
	if err != nil {
		return nil, fmt.Errorf("get breach counts: %w", err)
	}

	// Calculate overall compliance from last 30 days
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	var compliant, total int
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE is_compliant = true), COUNT(*)
		FROM sla_compliance WHERE org_id = $1 AND period_end >= $2
	`, orgID, thirtyDaysAgo).Scan(&compliant, &total)
	if err != nil {
		return nil, fmt.Errorf("get compliance stats: %w", err)
	}

	if total > 0 {
		stats.OverallCompliance = float64(compliant) / float64(total) * 100
	}

	return stats, nil
}

// User Favorites methods

// GetFavoritesByUserAndOrg returns all favorites for a user in an organization,
// optionally filtered by entity type.
func (db *DB) GetFavoritesByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]*models.Favorite, error) {
	query := `
		SELECT id, user_id, org_id, entity_type, entity_id, created_at
		FROM user_favorites
		WHERE user_id = $1 AND org_id = $2
	`
	args := []interface{}{userID, orgID}

	if entityType != "" {
		query += " AND entity_type = $3"
		args = append(args, entityType)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query favorites: %w", err)
	}
	defer rows.Close()

	var favorites []*models.Favorite
	for rows.Next() {
		var f models.Favorite
		err := rows.Scan(
			&f.ID, &f.UserID, &f.OrgID, &f.EntityType, &f.EntityID, &f.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan favorite: %w", err)
		}
		favorites = append(favorites, &f)
	}
	return favorites, nil
}

// GetFavoriteByUserAndEntity returns a favorite by user and entity.
func (db *DB) GetFavoriteByUserAndEntity(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) (*models.Favorite, error) {
	var f models.Favorite
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, org_id, entity_type, entity_id, created_at
		FROM user_favorites
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID).Scan(
		&f.ID, &f.UserID, &f.OrgID, &f.EntityType, &f.EntityID, &f.CreatedAt,
	)
// Metrics methods

// GetBackupsByOrgIDSince returns backups for an organization since a given time.
func (db *DB) GetBackupsByOrgIDSince(ctx context.Context, orgID uuid.UUID, since time.Time) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at, b.completed_at,
		       b.status, b.size_bytes, b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2
		ORDER BY b.started_at DESC
	`, orgID, since)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// CreateFavorite creates a new favorite.
func (db *DB) CreateFavorite(ctx context.Context, f *models.Favorite) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_favorites (id, user_id, org_id, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, f.ID, f.UserID, f.OrgID, f.EntityType, f.EntityID, f.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert favorite: %w", err)
	}
	return nil
}

// DeleteFavorite deletes a favorite by user, entity type and entity ID.
func (db *DB) DeleteFavorite(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM user_favorites WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	return nil
}

// IsFavorite checks if an entity is favorited by a user.
func (db *DB) IsFavorite(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_favorites
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check favorite: %w", err)
	}
	return count > 0, nil
}

// GetFavoriteEntityIDs returns all entity IDs of a given type that are favorited by a user.
func (db *DB) GetFavoriteEntityIDs(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT entity_id FROM user_favorites
		WHERE user_id = $1 AND org_id = $2 AND entity_type = $3
	`, userID, orgID, entityType)
	if err != nil {
		return nil, fmt.Errorf("query favorite entity IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan entity ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Docker Registry methods

// GetDockerRegistriesByOrgID returns all Docker registries for an organization.
func (db *DB) GetDockerRegistriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata, created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list docker registries: %w", err)
	}
	defer rows.Close()

	return scanDockerRegistries(rows)
}

// GetDockerRegistryByID returns a Docker registry by ID.
func (db *DB) GetDockerRegistryByID(ctx context.Context, id uuid.UUID) (*models.DockerRegistry, error) {
	var r models.DockerRegistry
	var typeStr, healthStatusStr string
	var metadataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata, created_by, created_at, updated_at
		FROM docker_registries
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.URL, &r.CredentialsEncrypted, &r.IsDefault, &r.Enabled,
		&healthStatusStr, &r.LastHealthCheck, &r.LastHealthError,
		&r.CredentialsRotatedAt, &r.CredentialsExpiresAt, &metadataBytes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get docker registry: %w", err)
	}
	r.Type = models.DockerRegistryType(typeStr)
	r.HealthStatus = models.DockerRegistryHealthStatus(healthStatusStr)
	if err := r.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("registry_id", r.ID.String()).Msg("failed to parse registry metadata")
	}
	return &r, nil
}

// GetDefaultDockerRegistry returns the default Docker registry for an organization.
func (db *DB) GetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID) (*models.DockerRegistry, error) {
	var r models.DockerRegistry
	var typeStr, healthStatusStr string
	var metadataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata, created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1 AND is_default = true
	`, orgID).Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.URL, &r.CredentialsEncrypted, &r.IsDefault, &r.Enabled,
		&healthStatusStr, &r.LastHealthCheck, &r.LastHealthError,
		&r.CredentialsRotatedAt, &r.CredentialsExpiresAt, &metadataBytes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get default docker registry: %w", err)
	}
	r.Type = models.DockerRegistryType(typeStr)
	r.HealthStatus = models.DockerRegistryHealthStatus(healthStatusStr)
	if err := r.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("registry_id", r.ID.String()).Msg("failed to parse registry metadata")
	}
	return &r, nil
}

// CreateDockerRegistry creates a new Docker registry.
func (db *DB) CreateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error {
	metadataBytes, err := registry.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_registries (id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		                               health_status, credentials_rotated_at, credentials_expires_at, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, registry.ID, registry.OrgID, registry.Name, string(registry.Type), registry.URL, registry.CredentialsEncrypted,
		registry.IsDefault, registry.Enabled, string(registry.HealthStatus), registry.CredentialsRotatedAt,
		registry.CredentialsExpiresAt, metadataBytes, registry.CreatedBy, registry.CreatedAt, registry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create docker registry: %w", err)
	}
	return nil
}

// UpdateDockerRegistry updates an existing Docker registry.
func (db *DB) UpdateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error {
	metadataBytes, err := registry.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	registry.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_registries
		SET name = $2, type = $3, url = $4, is_default = $5, enabled = $6, metadata = $7, updated_at = $8
		WHERE id = $1
	`, registry.ID, registry.Name, string(registry.Type), registry.URL, registry.IsDefault,
		registry.Enabled, metadataBytes, registry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update docker registry: %w", err)
	}
	return nil
}

// DeleteDockerRegistry deletes a Docker registry.
func (db *DB) DeleteDockerRegistry(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_registries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete docker registry: %w", err)
	}
	return nil
}

// UpdateDockerRegistryHealth updates the health status of a Docker registry.
func (db *DB) UpdateDockerRegistryHealth(ctx context.Context, id uuid.UUID, status models.DockerRegistryHealthStatus, errorMsg *string) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_registries
		SET health_status = $2, last_health_check = $3, last_health_error = $4, updated_at = $5
		WHERE id = $1
	`, id, string(status), now, errorMsg, now)
	if err != nil {
		return fmt.Errorf("update docker registry health: %w", err)
	}
	return nil
}

// UpdateDockerRegistryCredentials updates the credentials of a Docker registry (for rotation).
func (db *DB) UpdateDockerRegistryCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte, expiresAt *time.Time) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_registries
		SET credentials_encrypted = $2, credentials_rotated_at = $3, credentials_expires_at = $4, updated_at = $5
		WHERE id = $1
	`, id, credentialsEncrypted, now, expiresAt, now)
	if err != nil {
		return fmt.Errorf("update docker registry credentials: %w", err)
	}
	return nil
}

// SetDefaultDockerRegistry sets a registry as the default for an organization.
func (db *DB) SetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID, registryID uuid.UUID) error {
	now := time.Now()
	// First, unset any existing default
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_registries SET is_default = false, updated_at = $2 WHERE org_id = $1 AND is_default = true
	`, orgID, now)
	if err != nil {
		return fmt.Errorf("unset default docker registry: %w", err)
	}

	// Set the new default
	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_registries SET is_default = true, updated_at = $2 WHERE id = $1
	`, registryID, now)
	if err != nil {
		return fmt.Errorf("set default docker registry: %w", err)
	}
	return nil
}

// GetDockerRegistriesWithExpiringCredentials returns registries with credentials expiring before the given date.
func (db *DB) GetDockerRegistriesWithExpiringCredentials(ctx context.Context, orgID uuid.UUID, before time.Time) ([]*models.DockerRegistry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata, created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1 AND credentials_expires_at IS NOT NULL AND credentials_expires_at < $2
		ORDER BY credentials_expires_at
	`, orgID, before)
	if err != nil {
		return nil, fmt.Errorf("get expiring docker registries: %w", err)
	}
	defer rows.Close()

	return scanDockerRegistries(rows)
}

// CreateDockerRegistryAuditLog creates an audit log entry for a Docker registry operation.
func (db *DB) CreateDockerRegistryAuditLog(ctx context.Context, orgID, registryID uuid.UUID, userID *uuid.UUID, action string, details map[string]interface{}, ipAddress, userAgent string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_registry_audit_log (org_id, registry_id, user_id, action, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, orgID, registryID, userID, action, detailsJSON, ipAddress, userAgent)
	if err != nil {
		return fmt.Errorf("create docker registry audit log: %w", err)
	}
	return nil
}

// scanDockerRegistries scans rows into Docker registries.
func scanDockerRegistries(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.DockerRegistry, error) {
	var registries []*models.DockerRegistry
	for rows.Next() {
		var r models.DockerRegistry
		var typeStr, healthStatusStr string
		var metadataBytes []byte
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.Name, &typeStr, &r.URL, &r.CredentialsEncrypted, &r.IsDefault, &r.Enabled,
			&healthStatusStr, &r.LastHealthCheck, &r.LastHealthError,
			&r.CredentialsRotatedAt, &r.CredentialsExpiresAt, &metadataBytes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan docker registry: %w", err)
		}
		r.Type = models.DockerRegistryType(typeStr)
		r.HealthStatus = models.DockerRegistryHealthStatus(healthStatusStr)
		_ = r.SetMetadata(metadataBytes)
		registries = append(registries, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate docker registries: %w", err)
	}
	return registries, nil
}

// =============================================================================
// Komodo Integration Methods
// =============================================================================

// GetKomodoIntegrationsByOrgID returns all Komodo integrations for an organization.
func (db *DB) GetKomodoIntegrationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoIntegration, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, url, config_encrypted, status, last_sync_at,
		       last_error, enabled, created_at, updated_at
		FROM komodo_integrations
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get komodo integrations: %w", err)
	}
	defer rows.Close()

	var integrations []*models.KomodoIntegration
	for rows.Next() {
		var i models.KomodoIntegration
		err := rows.Scan(
			&i.ID, &i.OrgID, &i.Name, &i.URL, &i.ConfigEncrypted, &i.Status,
			&i.LastSyncAt, &i.LastError, &i.Enabled, &i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan komodo integration: %w", err)
		}
		integrations = append(integrations, &i)
	}

	return integrations, nil
}

// GetKomodoIntegrationByID returns a Komodo integration by ID.
func (db *DB) GetKomodoIntegrationByID(ctx context.Context, id uuid.UUID) (*models.KomodoIntegration, error) {
	var i models.KomodoIntegration
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, url, config_encrypted, status, last_sync_at,
		       last_error, enabled, created_at, updated_at
		FROM komodo_integrations
		WHERE id = $1
	`, id).Scan(
		&i.ID, &i.OrgID, &i.Name, &i.URL, &i.ConfigEncrypted, &i.Status,
		&i.LastSyncAt, &i.LastError, &i.Enabled, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get komodo integration: %w", err)
	}
	return &i, nil
}

// CreateKomodoIntegration creates a new Komodo integration.
func (db *DB) CreateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_integrations (id, org_id, name, url, config_encrypted, status,
		                                 last_sync_at, last_error, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, integration.ID, integration.OrgID, integration.Name, integration.URL, integration.ConfigEncrypted,
		integration.Status, integration.LastSyncAt, integration.LastError, integration.Enabled,
		integration.CreatedAt, integration.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create komodo integration: %w", err)
	}
	return nil
}

// UpdateKomodoIntegration updates a Komodo integration.
func (db *DB) UpdateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error {
	integration.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE komodo_integrations
		SET name = $2, url = $3, config_encrypted = $4, status = $5, last_sync_at = $6,
		    last_error = $7, enabled = $8, updated_at = $9
		WHERE id = $1
	`, integration.ID, integration.Name, integration.URL, integration.ConfigEncrypted,
		integration.Status, integration.LastSyncAt, integration.LastError, integration.Enabled,
		integration.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update komodo integration: %w", err)
	}
	return nil
}

// DeleteKomodoIntegration deletes a Komodo integration by ID.
func (db *DB) DeleteKomodoIntegration(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM komodo_integrations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete komodo integration: %w", err)
	}
	return nil
}

// GetKomodoContainersByOrgID returns all Komodo containers for an organization.
func (db *DB) GetKomodoContainersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled, last_discovered_at,
		       created_at, updated_at
		FROM komodo_containers
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get komodo containers: %w", err)
	}
	defer rows.Close()

	return db.scanKomodoContainers(rows)
}

// GetKomodoContainersByIntegrationID returns all containers for a specific integration.
func (db *DB) GetKomodoContainersByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled, last_discovered_at,
		       created_at, updated_at
		FROM komodo_containers
		WHERE integration_id = $1
		ORDER BY name
	`, integrationID)
	if err != nil {
		return nil, fmt.Errorf("get komodo containers by integration: %w", err)
	}
	defer rows.Close()

	return db.scanKomodoContainers(rows)
}

// GetKomodoContainerByID returns a Komodo container by ID.
func (db *DB) GetKomodoContainerByID(ctx context.Context, id uuid.UUID) (*models.KomodoContainer, error) {
	var c models.KomodoContainer
	var labelsJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled, last_discovered_at,
		       created_at, updated_at
		FROM komodo_containers
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.IntegrationID, &c.KomodoID, &c.Name, &c.Image, &c.StackName,
		&c.StackID, &c.Status, &c.AgentID, &c.Volumes, &labelsJSON, &c.BackupEnabled,
		&c.LastDiscoveredAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get komodo container: %w", err)
	}
	if labelsJSON != nil {
		json.Unmarshal(labelsJSON, &c.Labels)
	}
	return &c, nil
}

// GetKomodoContainerByKomodoID returns a container by its Komodo ID within an integration.
func (db *DB) GetKomodoContainerByKomodoID(ctx context.Context, integrationID uuid.UUID, komodoID string) (*models.KomodoContainer, error) {
	var c models.KomodoContainer
	var labelsJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled, last_discovered_at,
		       created_at, updated_at
		FROM komodo_containers
		WHERE integration_id = $1 AND komodo_id = $2
	`, integrationID, komodoID).Scan(
		&c.ID, &c.OrgID, &c.IntegrationID, &c.KomodoID, &c.Name, &c.Image, &c.StackName,
		&c.StackID, &c.Status, &c.AgentID, &c.Volumes, &labelsJSON, &c.BackupEnabled,
		&c.LastDiscoveredAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get komodo container by komodo_id: %w", err)
	}
	if labelsJSON != nil {
		json.Unmarshal(labelsJSON, &c.Labels)
	}
	return &c, nil
}

// CreateKomodoContainer creates a new Komodo container.
func (db *DB) CreateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	labelsJSON, _ := json.Marshal(container.Labels)
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_containers (id, org_id, integration_id, komodo_id, name, image,
		                               stack_name, stack_id, status, agent_id, volumes, labels,
		                               backup_enabled, last_discovered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, container.ID, container.OrgID, container.IntegrationID, container.KomodoID, container.Name,
		container.Image, container.StackName, container.StackID, container.Status, container.AgentID,
		container.Volumes, labelsJSON, container.BackupEnabled, container.LastDiscoveredAt,
		container.CreatedAt, container.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create komodo container: %w", err)
	}
	return nil
}

// UpdateKomodoContainer updates a Komodo container.
func (db *DB) UpdateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	container.UpdatedAt = time.Now()
	labelsJSON, _ := json.Marshal(container.Labels)
	_, err := db.Pool.Exec(ctx, `
		UPDATE komodo_containers
		SET name = $2, image = $3, stack_name = $4, stack_id = $5, status = $6, agent_id = $7,
		    volumes = $8, labels = $9, backup_enabled = $10, last_discovered_at = $11, updated_at = $12
		WHERE id = $1
	`, container.ID, container.Name, container.Image, container.StackName, container.StackID,
		container.Status, container.AgentID, container.Volumes, labelsJSON, container.BackupEnabled,
		container.LastDiscoveredAt, container.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update komodo container: %w", err)
	}
	return nil
}

// UpsertKomodoContainer inserts or updates a Komodo container based on komodo_id.
func (db *DB) UpsertKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	container.UpdatedAt = time.Now()
	labelsJSON, _ := json.Marshal(container.Labels)
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_containers (id, org_id, integration_id, komodo_id, name, image,
		                               stack_name, stack_id, status, agent_id, volumes, labels,
		                               backup_enabled, last_discovered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (integration_id, komodo_id)
		DO UPDATE SET name = EXCLUDED.name, image = EXCLUDED.image, stack_name = EXCLUDED.stack_name,
		              stack_id = EXCLUDED.stack_id, status = EXCLUDED.status, volumes = EXCLUDED.volumes,
		              labels = EXCLUDED.labels, last_discovered_at = EXCLUDED.last_discovered_at,
		              updated_at = EXCLUDED.updated_at
	`, container.ID, container.OrgID, container.IntegrationID, container.KomodoID, container.Name,
		container.Image, container.StackName, container.StackID, container.Status, container.AgentID,
		container.Volumes, labelsJSON, container.BackupEnabled, container.LastDiscoveredAt,
		container.CreatedAt, container.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert komodo container: %w", err)
	}
	return nil
}

// DeleteKomodoContainer deletes a Komodo container by ID.
func (db *DB) DeleteKomodoContainer(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM komodo_containers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete komodo container: %w", err)
	}
	return nil
}

// scanKomodoContainers scans rows into KomodoContainer slice.
func (db *DB) scanKomodoContainers(rows pgx.Rows) ([]*models.KomodoContainer, error) {
	var containers []*models.KomodoContainer
	for rows.Next() {
		var c models.KomodoContainer
		var labelsJSON []byte
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.IntegrationID, &c.KomodoID, &c.Name, &c.Image, &c.StackName,
			&c.StackID, &c.Status, &c.AgentID, &c.Volumes, &labelsJSON, &c.BackupEnabled,
			&c.LastDiscoveredAt, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan komodo container: %w", err)
		}
		if labelsJSON != nil {
			json.Unmarshal(labelsJSON, &c.Labels)
		}
		containers = append(containers, &c)
	}
	return containers, nil
}

// GetKomodoStacksByOrgID returns all Komodo stacks for an organization.
func (db *DB) GetKomodoStacksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoStack, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get komodo stacks: %w", err)
	}
	defer rows.Close()

	return db.scanKomodoStacks(rows)
}

// GetKomodoStacksByIntegrationID returns all stacks for a specific integration.
func (db *DB) GetKomodoStacksByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoStack, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE integration_id = $1
		ORDER BY name
	`, integrationID)
	if err != nil {
		return nil, fmt.Errorf("get komodo stacks by integration: %w", err)
	}
	defer rows.Close()

	return db.scanKomodoStacks(rows)
}

// GetKomodoStackByID returns a Komodo stack by ID.
func (db *DB) GetKomodoStackByID(ctx context.Context, id uuid.UUID) (*models.KomodoStack, error) {
	var s models.KomodoStack
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.OrgID, &s.IntegrationID, &s.KomodoID, &s.Name, &s.ServerID, &s.ServerName,
		&s.ContainerCount, &s.RunningCount, &s.LastDiscoveredAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get komodo stack: %w", err)
	}
	return &s, nil
}

// UpsertKomodoStack inserts or updates a Komodo stack based on komodo_id.
func (db *DB) UpsertKomodoStack(ctx context.Context, stack *models.KomodoStack) error {
	stack.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_stacks (id, org_id, integration_id, komodo_id, name, server_id,
		                           server_name, container_count, running_count,
		                           last_discovered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (integration_id, komodo_id)
		DO UPDATE SET name = EXCLUDED.name, server_id = EXCLUDED.server_id,
		              server_name = EXCLUDED.server_name, container_count = EXCLUDED.container_count,
		              running_count = EXCLUDED.running_count, last_discovered_at = EXCLUDED.last_discovered_at,
		              updated_at = EXCLUDED.updated_at
	`, stack.ID, stack.OrgID, stack.IntegrationID, stack.KomodoID, stack.Name, stack.ServerID,
		stack.ServerName, stack.ContainerCount, stack.RunningCount, stack.LastDiscoveredAt,
		stack.CreatedAt, stack.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert komodo stack: %w", err)
	}
	return nil
}

// scanKomodoStacks scans rows into KomodoStack slice.
func (db *DB) scanKomodoStacks(rows pgx.Rows) ([]*models.KomodoStack, error) {
	var stacks []*models.KomodoStack
	for rows.Next() {
		var s models.KomodoStack
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.IntegrationID, &s.KomodoID, &s.Name, &s.ServerID, &s.ServerName,
			&s.ContainerCount, &s.RunningCount, &s.LastDiscoveredAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan komodo stack: %w", err)
		}
		stacks = append(stacks, &s)
	}
	return stacks, nil
}

// CreateKomodoWebhookEvent creates a new Komodo webhook event.
func (db *DB) CreateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_webhook_events (id, org_id, integration_id, event_type, payload,
		                                   status, error_message, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.OrgID, event.IntegrationID, event.EventType, event.Payload,
		event.Status, event.ErrorMessage, event.ProcessedAt, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create komodo webhook event: %w", err)
	}
	return nil
}

// GetKomodoWebhookEventsByOrgID returns recent webhook events for an organization.
func (db *DB) GetKomodoWebhookEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.KomodoWebhookEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, event_type, payload, status,
		       error_message, processed_at, created_at
		FROM komodo_webhook_events
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("get komodo webhook events: %w", err)
	}
	defer rows.Close()

	var events []*models.KomodoWebhookEvent
	for rows.Next() {
		var e models.KomodoWebhookEvent
		err := rows.Scan(
			&e.ID, &e.OrgID, &e.IntegrationID, &e.EventType, &e.Payload, &e.Status,
			&e.ErrorMessage, &e.ProcessedAt, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan komodo webhook event: %w", err)
		}
		events = append(events, &e)
	}

	return events, nil
}

// UpdateKomodoWebhookEvent updates a Komodo webhook event.
func (db *DB) UpdateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE komodo_webhook_events
		SET status = $2, error_message = $3, processed_at = $4
		WHERE id = $1
	`, event.ID, event.Status, event.ErrorMessage, event.ProcessedAt)
	if err != nil {
		return fmt.Errorf("update komodo webhook event: %w", err)
	}
	return nil
}

// Offline License methods

// CreateOfflineLicense stores an offline license in the database.
func (db *DB) CreateOfflineLicense(ctx context.Context, license *models.OfflineLicense) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO offline_licenses (id, org_id, customer_id, tier, license_data, expires_at, issued_at, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, license.ID, license.OrgID, license.CustomerID, license.Tier, license.LicenseData, license.ExpiresAt, license.IssuedAt, license.UploadedBy, license.CreatedAt)
// GetBackupsByOrgIDAndDateRange returns backups for an org within a date range.
func (db *DB) GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at,
		       b.completed_at, b.status, b.size_bytes, b.files_new,
		       b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.deleted_at IS NULL AND b.started_at >= $2 AND b.started_at <= $3
		ORDER BY b.started_at DESC
	`, orgID, start, end)
	if err != nil {
		return fmt.Errorf("create offline license: %w", err)
	}
	return nil
}

// GetLatestOfflineLicense returns the most recently uploaded license for an organization.
func (db *DB) GetLatestOfflineLicense(ctx context.Context, orgID uuid.UUID) (*models.OfflineLicense, error) {
	var lic models.OfflineLicense
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, customer_id, tier, license_data, expires_at, issued_at, uploaded_by, created_at
		FROM offline_licenses
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID).Scan(&lic.ID, &lic.OrgID, &lic.CustomerID, &lic.Tier, &lic.LicenseData, &lic.ExpiresAt, &lic.IssuedAt, &lic.UploadedBy, &lic.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest offline license: %w", err)
	}
	return &lic, nil
}


// GetTagsByOrgID returns all tags for an organization.
func (db *DB) GetTagsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Tag, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, color, created_at, updated_at
		FROM tags
		WHERE org_id = $1
		ORDER BY name
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps,
		       s.backup_window_start, s.backup_window_end, s.excluded_hours,
		       s.compression_level, s.on_mount_unavailable, s.enabled,
		       s.created_at, s.updated_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.enabled = true
		ORDER BY s.name
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
// GetSchedulesByAgentGroupID returns all schedules for an agent group.
func (db *DB) GetSchedulesByAgentGroupID(ctx context.Context, agentGroupID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, agent_group_id, policy_id, name, cron_expression, paths, excludes,
		       retention_policy, bandwidth_limit_kbps, backup_window_start, backup_window_end,
		       excluded_hours, compression_level, on_mount_unavailable, enabled, created_at, updated_at
		FROM schedules
		WHERE agent_group_id = $1
		ORDER BY name
	`, agentGroupID)
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
// GetEnabledSchedules returns all enabled schedules.
func (db *DB) GetEnabledSchedules(ctx context.Context) ([]models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, repository_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
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

// GetRepository returns a repository by ID (alias for GetRepositoryByID).
func (db *DB) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	return db.GetRepositoryByID(ctx, id)
}

// Organization methods

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
		SELECT s.id, s.agent_id, s.repository_id, s.name, s.cron_expression,
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

// Config Template methods

// CreateConfigTemplate creates a new config template.
func (db *DB) CreateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO config_templates (
			id, org_id, created_by_id, name, description, type, visibility,
			tags, config, usage_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		template.ID, template.OrgID, template.CreatedByID, template.Name,
		template.Description, template.Type, template.Visibility,
		tagsJSON, template.Config, template.UsageCount, template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create config template: %w", err)
	}
	return nil
}

// GetConfigTemplateByID returns a config template by ID.
func (db *DB) GetConfigTemplateByID(ctx context.Context, id uuid.UUID) (*models.ConfigTemplate, error) {
	var template models.ConfigTemplate
	var tagsBytes []byte
	var typeStr, visibilityStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE id = $1
	`, id).Scan(
		&template.ID, &template.OrgID, &template.CreatedByID, &template.Name,
		&template.Description, &typeStr, &visibilityStr,
		&tagsBytes, &template.Config, &template.UsageCount, &template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get config template: %w", err)
	}

	template.Type = models.TemplateType(typeStr)
	template.Visibility = models.TemplateVisibility(visibilityStr)

	if err := template.SetTags(tagsBytes); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	return &template, nil
}

// GetConfigTemplatesByOrgID returns all config templates for an organization.
func (db *DB) GetConfigTemplatesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list config templates: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// GetPublicConfigTemplates returns all public config templates.
func (db *DB) GetPublicConfigTemplates(ctx context.Context) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE visibility = 'public'
		ORDER BY usage_count DESC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list public config templates: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// GetConfigTemplatesByType returns config templates filtered by type.
func (db *DB) GetConfigTemplatesByType(ctx context.Context, orgID uuid.UUID, templateType models.TemplateType) ([]*models.ConfigTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, type, visibility,
		       tags, config, usage_count, created_at, updated_at
		FROM config_templates
		WHERE (org_id = $1 OR visibility = 'public') AND type = $2
		ORDER BY usage_count DESC, created_at DESC
	`, orgID, templateType)
	if err != nil {
		return nil, fmt.Errorf("list config templates by type: %w", err)
	}
	defer rows.Close()

	return scanConfigTemplates(rows)
}

// UpdateConfigTemplate updates a config template.
func (db *DB) UpdateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE config_templates
		SET name = $2, description = $3, visibility = $4, tags = $5, updated_at = $6
		WHERE id = $1
	`, template.ID, template.Name, template.Description, template.Visibility, tagsJSON, template.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update config template: %w", err)
	}
	return nil
}

// DeleteConfigTemplate deletes a config template.
func (db *DB) DeleteConfigTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM config_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete config template: %w", err)
	}
	return nil
}

// IncrementTemplateUsageCount increments the usage count for a template.
func (db *DB) IncrementTemplateUsageCount(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE config_templates
		SET usage_count = usage_count + 1, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("increment template usage count: %w", err)
	}
	return nil
}

// scanConfigTemplates scans rows into config templates.
func scanConfigTemplates(rows pgx.Rows) ([]*models.ConfigTemplate, error) {
	var templates []*models.ConfigTemplate
	for rows.Next() {
		var template models.ConfigTemplate
		var tagsBytes []byte
		var typeStr, visibilityStr string

		err := rows.Scan(
			&template.ID, &template.OrgID, &template.CreatedByID, &template.Name,
			&template.Description, &typeStr, &visibilityStr,
			&tagsBytes, &template.Config, &template.UsageCount, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan config template: %w", err)
		}

		template.Type = models.TemplateType(typeStr)
		template.Visibility = models.TemplateVisibility(visibilityStr)

		if err := template.SetTags(tagsBytes); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate config templates: %w", err)
	}

	return templates, nil
}

// SLA Policy methods

// CreateSLAPolicy inserts a new SLA policy.
func (db *DB) CreateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_policies (id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, policy.ID, policy.OrgID, policy.Name, policy.Description, policy.TargetRPOHours, policy.TargetRTOHours, policy.TargetSuccessRate, policy.Enabled, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create SLA policy: %w", err)
	}
	return nil
}

// GetSLAPolicyByID returns a single SLA policy by ID.
func (db *DB) GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*models.SLAPolicy, error) {
	var p models.SLAPolicy
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at
		FROM sla_policies
		WHERE id = $1
	`, id).Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.TargetRPOHours, &p.TargetRTOHours, &p.TargetSuccessRate, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get SLA policy: %w", err)
	}
	return &p, nil
}

// ListSLAPoliciesByOrgID returns all SLA policies for an organization.
func (db *DB) ListSLAPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SLAPolicy, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, target_rpo_hours, target_rto_hours, target_success_rate, enabled, created_at, updated_at
		FROM sla_policies
		WHERE org_id = $1
		ORDER BY name ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list SLA policies: %w", err)
	}
	defer rows.Close()

	var policies []*models.SLAPolicy
	for rows.Next() {
		var p models.SLAPolicy
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.TargetRPOHours, &p.TargetRTOHours, &p.TargetSuccessRate, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan SLA policy: %w", err)
		}
		policies = append(policies, &p)
	}
	return policies, nil
}

// UpdateSLAPolicy updates an existing SLA policy.
func (db *DB) UpdateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error {
	policy.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE sla_policies
		SET name = $2, description = $3, target_rpo_hours = $4, target_rto_hours = $5, target_success_rate = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, policy.ID, policy.Name, policy.Description, policy.TargetRPOHours, policy.TargetRTOHours, policy.TargetSuccessRate, policy.Enabled, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update SLA policy: %w", err)
	}
	return nil
}

// DeleteSLAPolicy deletes an SLA policy by ID.
func (db *DB) DeleteSLAPolicy(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM sla_policies WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete SLA policy: %w", err)
	}
	return nil
}

// CreateSLAStatusSnapshot inserts a new SLA status history record.
func (db *DB) CreateSLAStatusSnapshot(ctx context.Context, snapshot *models.SLAStatusSnapshot) error {
	snapshot.ID = uuid.New()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_status_history (id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, snapshot.ID, snapshot.PolicyID, snapshot.RPOHours, snapshot.RTOHours, snapshot.SuccessRate, snapshot.Compliant, snapshot.CalculatedAt)
	if err != nil {
		return fmt.Errorf("create SLA status snapshot: %w", err)
	}
	return nil
}

// GetSLAStatusHistory returns SLA status history for a policy, ordered by most recent first.
func (db *DB) GetSLAStatusHistory(ctx context.Context, policyID uuid.UUID, limit int) ([]*models.SLAStatusSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at
		FROM sla_status_history
		WHERE policy_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2
	`, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("get SLA status history: %w", err)
	}
	defer rows.Close()

	var snapshots []*models.SLAStatusSnapshot
	for rows.Next() {
		var s models.SLAStatusSnapshot
		if err := rows.Scan(&s.ID, &s.PolicyID, &s.RPOHours, &s.RTOHours, &s.SuccessRate, &s.Compliant, &s.CalculatedAt); err != nil {
			return nil, fmt.Errorf("scan SLA status snapshot: %w", err)
		}
		snapshots = append(snapshots, &s)
	}
	return snapshots, nil
}

// GetLatestSLAStatus returns the most recent SLA status snapshot for a policy.
func (db *DB) GetLatestSLAStatus(ctx context.Context, policyID uuid.UUID) (*models.SLAStatusSnapshot, error) {
	var s models.SLAStatusSnapshot
	err := db.Pool.QueryRow(ctx, `
		SELECT id, policy_id, rpo_hours, rto_hours, success_rate, compliant, calculated_at
		FROM sla_status_history
		WHERE policy_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`, policyID).Scan(&s.ID, &s.PolicyID, &s.RPOHours, &s.RTOHours, &s.SuccessRate, &s.Compliant, &s.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("get latest SLA status: %w", err)
	}
	return &s, nil
}

// GetBackupSuccessRateForOrg calculates the backup success rate for an org over a given number of hours.
func (db *DB) GetBackupSuccessRateForOrg(ctx context.Context, orgID uuid.UUID, hours int) (float64, error) {
	var total, successful int
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed')
		FROM backups
		WHERE org_id = $1 AND created_at >= NOW() - ($2 || ' hours')::INTERVAL
	`, orgID, fmt.Sprintf("%d", hours)).Scan(&total, &successful)
	if err != nil {
		return 0, fmt.Errorf("get backup success rate: %w", err)
	}
	if total == 0 {
		return 100, nil
	}
	return float64(successful) / float64(total) * 100, nil
}

// GetMaxRPOHoursForOrg returns the maximum hours since the last successful backup across all agents in an org.
func (db *DB) GetMaxRPOHoursForOrg(ctx context.Context, orgID uuid.UUID) (float64, error) {
	var maxHours float64
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(EXTRACT(EPOCH FROM (NOW() - last_backup))/3600), 0)
		FROM (
			SELECT a.id, MAX(b.completed_at) as last_backup
			FROM agents a
			LEFT JOIN backups b ON b.agent_id = a.id AND b.status = 'completed'
			WHERE a.org_id = $1 AND a.status = 'active'
			GROUP BY a.id
		) sub
		WHERE last_backup IS NOT NULL
	`, orgID).Scan(&maxHours)
	if err != nil {
		return 0, fmt.Errorf("get max RPO hours: %w", err)
	}
	return maxHours, nil
}

// =============================================================================
// Prometheus Metrics Methods
// =============================================================================

// GetAllBackups returns all backups across all organizations (for Prometheus metrics).
func (db *DB) GetAllBackups(ctx context.Context) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all backups: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupsByStatus returns all backups with the specified status (for Prometheus metrics).
func (db *DB) GetBackupsByStatus(ctx context.Context, status models.BackupStatus) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		WHERE status = $1
		ORDER BY started_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("get backups by status: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetStorageStatsSummaryGlobal returns aggregated storage statistics across all organizations.
func (db *DB) GetStorageStatsSummaryGlobal(ctx context.Context) (*models.StorageStatsSummary, error) {
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
			ORDER BY s.repository_id, s.collected_at DESC
		) as latest
	`).Scan(
		&summary.TotalRawSize, &summary.TotalRestoreSize, &summary.TotalSpaceSaved,
		&summary.AvgDedupRatio, &summary.RepositoryCount, &summary.TotalSnapshots,
	)
	if err != nil {
		return nil, fmt.Errorf("get storage stats summary global: %w", err)
	}
	return &summary, nil
}

// Daily Summary methods

// CreateOrUpdateDailySummary upserts a daily summary record for an organization and date.
func (db *DB) CreateOrUpdateDailySummary(ctx context.Context, summary *models.MetricsDailySummary) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO metrics_daily_summary (
			id, org_id, date, backups_total, backups_successful, backups_failed,
			total_backup_size, total_duration_secs, agents_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (org_id, date) DO UPDATE SET
			backups_total = EXCLUDED.backups_total,
			backups_successful = EXCLUDED.backups_successful,
			backups_failed = EXCLUDED.backups_failed,
			total_backup_size = EXCLUDED.total_backup_size,
			total_duration_secs = EXCLUDED.total_duration_secs,
			agents_active = EXCLUDED.agents_active,
			updated_at = EXCLUDED.updated_at
	`, summary.ID, summary.OrgID, summary.Date,
		summary.TotalBackups, summary.SuccessfulBackups, summary.FailedBackups,
		summary.TotalSizeBytes, summary.TotalDurationSecs, summary.AgentsActive,
		summary.CreatedAt, summary.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert daily summary: %w", err)
	}
	return nil
}

// GetDailySummary returns a daily summary for a specific organization and date.
func (db *DB) GetDailySummary(ctx context.Context, orgID uuid.UUID, date time.Time) (*models.MetricsDailySummary, error) {
	var s models.MetricsDailySummary
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, date, backups_total, backups_successful, backups_failed,
		       total_backup_size, total_duration_secs, agents_active,
		       created_at, updated_at
		FROM metrics_daily_summary
		WHERE org_id = $1 AND date = $2
	`, orgID, date).Scan(
		&s.ID, &s.OrgID, &s.Date,
		&s.TotalBackups, &s.SuccessfulBackups, &s.FailedBackups,
		&s.TotalSizeBytes, &s.TotalDurationSecs, &s.AgentsActive,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get daily summary: %w", err)
	}
	return &s, nil
}

// GetDailySummaries returns daily summaries for an organization within a date range.
func (db *DB) GetDailySummaries(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]models.MetricsDailySummary, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, date, backups_total, backups_successful, backups_failed,
		       total_backup_size, total_duration_secs, agents_active,
		       created_at, updated_at
		FROM metrics_daily_summary
		WHERE org_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get daily summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.MetricsDailySummary
	for rows.Next() {
		var s models.MetricsDailySummary
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Date,
			&s.TotalBackups, &s.SuccessfulBackups, &s.FailedBackups,
			&s.TotalSizeBytes, &s.TotalDurationSecs, &s.AgentsActive,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan daily summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// DeleteDailySummariesBefore deletes daily summaries older than the given date for an organization.
func (db *DB) DeleteDailySummariesBefore(ctx context.Context, orgID uuid.UUID, before time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM metrics_daily_summary
		WHERE org_id = $1 AND date < $2
	`, orgID, before)
	if err != nil {
		return fmt.Errorf("delete daily summaries: %w", err)
	}
	return nil
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

// SSO Group Mapping methods

// GetSSOGroupMappingsByOrgID returns all SSO group mappings for an organization.
func (db *DB) GetSSOGroupMappingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE org_id = $1
		ORDER BY oidc_group_name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list SSO group mappings: %w", err)
	}
	defer rows.Close()

	var mappings []*models.SSOGroupMapping
	for rows.Next() {
		var m models.SSOGroupMapping
		var roleStr string
		err := rows.Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan SSO group mapping: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		mappings = append(mappings, &m)
	}

	return mappings, nil
}

// GetSSOGroupMappingsByGroupNames returns all SSO group mappings matching the given group names.
func (db *DB) GetSSOGroupMappingsByGroupNames(ctx context.Context, groupNames []string) ([]*models.SSOGroupMapping, error) {
	if len(groupNames) == 0 {
		return []*models.SSOGroupMapping{}, nil
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE oidc_group_name = ANY($1)
		ORDER BY oidc_group_name
	`, groupNames)
	if err != nil {
		return nil, fmt.Errorf("get SSO group mappings by names: %w", err)
	}
	defer rows.Close()

	var mappings []*models.SSOGroupMapping
	for rows.Next() {
		var m models.SSOGroupMapping
		var roleStr string
		err := rows.Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan SSO group mapping: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		mappings = append(mappings, &m)
	}

	return mappings, nil
}

// GetSSOGroupMappingByID returns an SSO group mapping by ID.
func (db *DB) GetSSOGroupMappingByID(ctx context.Context, id uuid.UUID) (*models.SSOGroupMapping, error) {
	var m models.SSOGroupMapping
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at
		FROM sso_group_mappings
		WHERE id = $1
	`, id).Scan(&m.ID, &m.OrgID, &m.OIDCGroupName, &roleStr, &m.AutoCreateOrg, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get SSO group mapping: %w", err)
	}
	m.Role = models.OrgRole(roleStr)
	return &m, nil
}

// CreateSSOGroupMapping creates a new SSO group mapping.
func (db *DB) CreateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sso_group_mappings (id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.OrgID, m.OIDCGroupName, string(m.Role), m.AutoCreateOrg, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create SSO group mapping: %w", err)
	}
	return nil
}

// UpdateSSOGroupMapping updates an existing SSO group mapping.
func (db *DB) UpdateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error {
	m.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE sso_group_mappings
		SET role = $2, auto_create_org = $3, updated_at = $4
		WHERE id = $1
	`, m.ID, string(m.Role), m.AutoCreateOrg, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update SSO group mapping: %w", err)
	}
	return nil
}

// DeleteSSOGroupMapping deletes an SSO group mapping.
func (db *DB) DeleteSSOGroupMapping(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sso_group_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete SSO group mapping: %w", err)
	}
	return nil
}

// User SSO Groups methods

// GetUserSSOGroups returns a user's SSO groups.
func (db *DB) GetUserSSOGroups(ctx context.Context, userID uuid.UUID) (*models.UserSSOGroups, error) {
	var u models.UserSSOGroups
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, oidc_groups, synced_at
		FROM user_sso_groups
		WHERE user_id = $1
	`, userID).Scan(&u.ID, &u.UserID, &u.OIDCGroups, &u.SyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get user SSO groups: %w", err)
	}
	return &u, nil
}

// UpsertUserSSOGroups creates or updates a user's SSO groups.
func (db *DB) UpsertUserSSOGroups(ctx context.Context, userID uuid.UUID, groups []string) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_sso_groups (id, user_id, oidc_groups, synced_at)
		VALUES (gen_random_uuid(), $1, $2, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET oidc_groups = $2, synced_at = NOW()
	`, userID, groups)
	if err != nil {
		return fmt.Errorf("upsert user SSO groups: %w", err)
	}
	return nil
}

// Organization SSO Settings methods

// GetOrganizationSSOSettings returns an organization's SSO settings.
func (db *DB) GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT sso_default_role, sso_auto_create_orgs
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&defaultRole, &autoCreateOrgs)
	if err != nil {
		return nil, false, fmt.Errorf("get org SSO settings: %w", err)
	}
	return defaultRole, autoCreateOrgs, nil
}

// UpdateOrganizationSSOSettings updates an organization's SSO settings.
func (db *DB) UpdateOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID, defaultRole *string, autoCreateOrgs bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations
		SET sso_default_role = $2, sso_auto_create_orgs = $3, updated_at = NOW()
		WHERE id = $1
	`, orgID, defaultRole, autoCreateOrgs)
	if err != nil {
		return fmt.Errorf("update org SSO settings: %w", err)
	}
	return nil
}

// UpdateMembershipRole updates a membership's role.
func (db *DB) UpdateMembershipRole(ctx context.Context, membershipID uuid.UUID, role models.OrgRole) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_memberships
		SET role = $2, updated_at = NOW()
		WHERE id = $1
	`, membershipID, string(role))
	if err != nil {
		return fmt.Errorf("update membership role: %w", err)
	}
	return nil
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

// =============================================================================
// Prometheus Metrics Methods
// =============================================================================

// GetAllBackups returns all backups across all organizations (for Prometheus metrics).
func (db *DB) GetAllBackups(ctx context.Context) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all backups: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetBackupsByStatus returns all backups with the specified status (for Prometheus metrics).
func (db *DB) GetBackupsByStatus(ctx context.Context, status models.BackupStatus) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error,
		       pre_script_output, pre_script_error, post_script_output, post_script_error,
		       excluded_large_files, resumed, checkpoint_id, original_backup_id, created_at
		FROM backups
		WHERE status = $1
		ORDER BY started_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("get backups by status: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetStorageStatsSummaryGlobal returns aggregated storage statistics across all organizations.
func (db *DB) GetStorageStatsSummaryGlobal(ctx context.Context) (*models.StorageStatsSummary, error) {
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
			ORDER BY s.repository_id, s.collected_at DESC
		) as latest
	`).Scan(
		&summary.TotalRawSize, &summary.TotalRestoreSize, &summary.TotalSpaceSaved,
		&summary.AvgDedupRatio, &summary.RepositoryCount, &summary.TotalSnapshots,
	)
	if err != nil {
		return nil, fmt.Errorf("get storage stats summary global: %w", err)
	}
	return &summary, nil
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
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("iterate backups: %w", err)
	}

	return backups, nil
}

// CleanupAgentHealthHistory deletes health history records older than the
// specified retention period. Returns the number of rows deleted.
func (db *DB) CleanupAgentHealthHistory(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_health_history
		WHERE recorded_at < NOW() - ($1 * INTERVAL '1 day')
	`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup agent health history: %w", err)
	}
	return tag.RowsAffected(), nil
}
