package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// GetAllUsers returns all users across all organizations.
func (db *DB) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		FROM users
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var roleStr string
		err := rows.Scan(
			&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
			&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.Role = models.UserRole(roleStr)
		users = append(users, &user)
	}

	return users, nil
}

// SetUserSuperuser sets or clears the superuser flag for a user.
func (db *DB) SetUserSuperuser(ctx context.Context, userID uuid.UUID, isSuperuser bool) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users SET is_superuser = $1, updated_at = NOW()
		WHERE id = $2
	`, isSuperuser, userID)
	if err != nil {
		return fmt.Errorf("set user superuser: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// GetSuperusers returns all users with superuser privileges.
func (db *DB) GetSuperusers(ctx context.Context) ([]*models.User, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		FROM users
		WHERE is_superuser = true
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("list superusers: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var roleStr string
		err := rows.Scan(
			&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
			&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan superuser: %w", err)
		}
		user.Role = models.UserRole(roleStr)
		users = append(users, &user)
	}

	return users, nil
}

// CreateSuperuserAuditLog creates a new superuser audit log entry.
func (db *DB) CreateSuperuserAuditLog(ctx context.Context, log *models.SuperuserAuditLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO superuser_audit_logs (
			id, superuser_id, action, target_type, target_id, target_org_id,
			impersonated_user_id, ip_address, user_agent, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		log.ID, log.SuperuserID, string(log.Action), log.TargetType, log.TargetID,
		log.TargetOrgID, log.ImpersonatedUserID, log.IPAddress, log.UserAgent,
		log.Details, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create superuser audit log: %w", err)
	}
	return nil
}

// GetSuperuserAuditLogs returns superuser audit logs with pagination.
func (db *DB) GetSuperuserAuditLogs(ctx context.Context, limit, offset int) ([]*models.SuperuserAuditLogWithUser, int, error) {
	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM superuser_audit_logs`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count superuser audit logs: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT
			sal.id, sal.superuser_id, sal.action, sal.target_type, sal.target_id,
			sal.target_org_id, sal.impersonated_user_id, sal.ip_address, sal.user_agent,
			sal.details, sal.created_at,
			u.email as superuser_email, u.name as superuser_name
		FROM superuser_audit_logs sal
		JOIN users u ON sal.superuser_id = u.id
		ORDER BY sal.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list superuser audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.SuperuserAuditLogWithUser
	for rows.Next() {
		var log models.SuperuserAuditLogWithUser
		var actionStr string
		var detailsBytes []byte
		err := rows.Scan(
			&log.ID, &log.SuperuserID, &actionStr, &log.TargetType, &log.TargetID,
			&log.TargetOrgID, &log.ImpersonatedUserID, &log.IPAddress, &log.UserAgent,
			&detailsBytes, &log.CreatedAt,
			&log.SuperuserEmail, &log.SuperuserName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan superuser audit log: %w", err)
		}
		log.Action = models.SuperuserAction(actionStr)
		log.Details = detailsBytes
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// GetSystemSetting returns a system setting by key.
func (db *DB) GetSystemSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	var setting models.SystemSetting
	var valueBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT key, value, description, updated_by, updated_at
		FROM system_settings
		WHERE key = $1
	`, key).Scan(&setting.Key, &valueBytes, &setting.Description, &setting.UpdatedBy, &setting.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get system setting: %w", err)
	}
	setting.Value = valueBytes
	return &setting, nil
}

// GetSystemSettings returns all system settings.
func (db *DB) GetSystemSettings(ctx context.Context) ([]*models.SystemSetting, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT key, value, description, updated_by, updated_at
		FROM system_settings
		ORDER BY key
	`)
	if err != nil {
		return nil, fmt.Errorf("list system settings: %w", err)
	}
	defer rows.Close()

	var settings []*models.SystemSetting
	for rows.Next() {
		var setting models.SystemSetting
		var valueBytes []byte
		err := rows.Scan(&setting.Key, &valueBytes, &setting.Description, &setting.UpdatedBy, &setting.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan system setting: %w", err)
		}
		setting.Value = valueBytes
		settings = append(settings, &setting)
	}

	return settings, nil
}

// UpdateSystemSetting updates a system setting value.
func (db *DB) UpdateSystemSetting(ctx context.Context, key string, value interface{}, updatedBy uuid.UUID) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal setting value: %w", err)
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE system_settings
		SET value = $1, updated_by = $2, updated_at = NOW()
		WHERE key = $3
	`, valueBytes, updatedBy, key)
	if err != nil {
		return fmt.Errorf("update system setting: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("setting not found")
	}
	return nil
}

// CreateInitialSuperuser creates the first superuser during setup.
// This should be called during initial system setup with an admin email.
func (db *DB) CreateInitialSuperuser(ctx context.Context, email string) error {
	// Check if any superusers exist
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE is_superuser = true`).Scan(&count)
	if err != nil {
		return fmt.Errorf("check superusers: %w", err)
	}

	if count > 0 {
		db.logger.Info().Msg("superuser already exists, skipping initial setup")
		return nil
	}

	// Try to find user by email and make them superuser
	result, err := db.Pool.Exec(ctx, `
		UPDATE users SET is_superuser = true, updated_at = NOW()
		WHERE LOWER(email) = LOWER($1)
	`, email)
	if err != nil {
		return fmt.Errorf("create initial superuser: %w", err)
	}

	if result.RowsAffected() > 0 {
		db.logger.Info().Str("email", email).Msg("initial superuser created")
	} else {
		db.logger.Warn().Str("email", email).Msg("user not found for initial superuser setup, will be granted on first login")
	}

	return nil
}

// GetUserByEmailForSuperuser returns a user by email (case-insensitive) for superuser operations.
func (db *DB) GetUserByEmailForSuperuser(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&user.ID, &user.OrgID, &user.OIDCSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	user.Role = models.UserRole(roleStr)
	return &user, nil
}
