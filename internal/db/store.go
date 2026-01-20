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
