package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
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

// GetAgentByID returns an agent by ID.
func (db *DB) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
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
	if err := a.SetOSInfo(osInfoBytes); err != nil {
		db.logger.Warn().Err(err).Str("agent_id", a.ID.String()).Msg("failed to parse OS info")
	}
	return &a, nil
}

// GetAgentByAPIKeyHash returns an agent by API key hash.
func (db *DB) GetAgentByAPIKeyHash(ctx context.Context, hash string) (*models.Agent, error) {
	var a models.Agent
	var osInfoBytes []byte
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

	agent.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE agents
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

	return schedules, nil
}

// GetScheduleByID returns a schedule by ID.
func (db *DB) GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, repository_id, name, cron_expression, paths, excludes,
		       retention_policy, enabled, created_at, updated_at
		FROM schedules
		WHERE id = $1
	`, id)

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

	return &s, nil
}

// scanScheduleRow scans a schedule from a single row.
func scanScheduleRow(row interface {
	Scan(dest ...any) error
}) (*models.Schedule, error) {
	return scanSchedule(row)
}

// Backup methods

// GetBackupsByScheduleID returns all backups for a schedule.
func (db *DB) GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message, created_at
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
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message, created_at
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
		SELECT id, schedule_id, agent_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message, created_at
		FROM backups
		WHERE id = $1
	`, id).Scan(
		&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
		&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
		&b.FilesChanged, &b.ErrorMessage, &b.CreatedAt,
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
		                     status, size_bytes, files_new, files_changed, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, backup.ID, backup.ScheduleID, backup.AgentID, backup.SnapshotID,
		backup.StartedAt, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged,
		backup.ErrorMessage, backup.CreatedAt)
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
		    files_new = $6, files_changed = $7, error_message = $8
		WHERE id = $1
	`, backup.ID, backup.SnapshotID, backup.CompletedAt, string(backup.Status),
		backup.SizeBytes, backup.FilesNew, backup.FilesChanged, backup.ErrorMessage)
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
			&b.ID, &b.ScheduleID, &b.AgentID, &b.SnapshotID, &b.StartedAt,
			&b.CompletedAt, &statusStr, &b.SizeBytes, &b.FilesNew,
			&b.FilesChanged, &b.ErrorMessage, &b.CreatedAt,
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
