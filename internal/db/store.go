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
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
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
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
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
