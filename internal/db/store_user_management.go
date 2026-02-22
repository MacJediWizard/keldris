package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// GetUsersByOrgID returns all users in an organization with their membership info.
func (db *DB) GetUsersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UserWithMembership, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT u.id, u.org_id, u.oidc_subject, u.email, u.name, u.role,
		       COALESCE(u.status, 'active') as status,
		       u.last_login_at, u.last_login_ip,
		       COALESCE(u.failed_login_attempts, 0) as failed_login_attempts,
		       u.locked_until, u.invited_by, u.invited_at,
		       COALESCE(u.is_superuser, false) as is_superuser,
		       u.created_at, u.updated_at,
		       m.role as org_role
		FROM users u
		JOIN org_memberships m ON u.id = m.user_id
		WHERE m.org_id = $1
		ORDER BY u.email
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get users by org id: %w", err)
	}
	defer rows.Close()

	var users []*models.UserWithMembership
	for rows.Next() {
		var u models.UserWithMembership
		var oidcSubject *string
		var roleStr string
		var orgRoleStr string
		err := rows.Scan(
			&u.ID, &u.OrgID, &oidcSubject, &u.Email, &u.Name, &roleStr,
			&u.Status, &u.LastLoginAt, &u.LastLoginIP,
			&u.FailedLoginAttempts, &u.LockedUntil, &u.InvitedBy, &u.InvitedAt,
			&u.IsSuperuser, &u.CreatedAt, &u.UpdatedAt,
			&orgRoleStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user with membership: %w", err)
		}
		if oidcSubject != nil {
			u.OIDCSubject = *oidcSubject
		}
		u.Role = models.UserRole(roleStr)
		u.OrgRole = models.OrgRole(orgRoleStr)
		users = append(users, &u)
	}

	return users, nil
}

// GetUserByIDWithMembership returns a user by ID with their org membership info.
func (db *DB) GetUserByIDWithMembership(ctx context.Context, userID, orgID uuid.UUID) (*models.UserWithMembership, error) {
	var u models.UserWithMembership
	var oidcSubject *string
	var roleStr string
	var orgRoleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT u.id, u.org_id, u.oidc_subject, u.email, u.name, u.role,
		       COALESCE(u.status, 'active') as status,
		       u.last_login_at, u.last_login_ip,
		       COALESCE(u.failed_login_attempts, 0) as failed_login_attempts,
		       u.locked_until, u.invited_by, u.invited_at,
		       COALESCE(u.is_superuser, false) as is_superuser,
		       u.created_at, u.updated_at,
		       m.role as org_role
		FROM users u
		JOIN org_memberships m ON u.id = m.user_id
		WHERE u.id = $1 AND m.org_id = $2
	`, userID, orgID).Scan(
		&u.ID, &u.OrgID, &oidcSubject, &u.Email, &u.Name, &roleStr,
		&u.Status, &u.LastLoginAt, &u.LastLoginIP,
		&u.FailedLoginAttempts, &u.LockedUntil, &u.InvitedBy, &u.InvitedAt,
		&u.IsSuperuser, &u.CreatedAt, &u.UpdatedAt,
		&orgRoleStr,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by id with membership: %w", err)
	}
	if oidcSubject != nil {
		u.OIDCSubject = *oidcSubject
	}
	u.Role = models.UserRole(roleStr)
	u.OrgRole = models.OrgRole(orgRoleStr)
	return &u, nil
}

// UpdateUserStatus updates a user's status.
func (db *DB) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status models.UserStatus) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, string(status))
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateUserName updates a user's name.
func (db *DB) UpdateUserName(ctx context.Context, userID uuid.UUID, name string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET name = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, name)
	if err != nil {
		return fmt.Errorf("update user name: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateUserLastLogin updates the user's last login information.
func (db *DB) UpdateUserLastLogin(ctx context.Context, userID uuid.UUID, ipAddress string) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET last_login_at = NOW(), last_login_ip = $2, failed_login_attempts = 0, updated_at = NOW()
		WHERE id = $1
	`, userID, ipAddress)
	if err != nil {
		return fmt.Errorf("update user last login: %w", err)
	}
	return nil
}

// IncrementFailedLoginAttempts increments the failed login attempt counter.
func (db *DB) IncrementFailedLoginAttempts(ctx context.Context, userID uuid.UUID) (int, error) {
	var attempts int
	err := db.Pool.QueryRow(ctx, `
		UPDATE users
		SET failed_login_attempts = COALESCE(failed_login_attempts, 0) + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING failed_login_attempts
	`, userID).Scan(&attempts)
	if err != nil {
		return 0, fmt.Errorf("increment failed login attempts: %w", err)
	}
	return attempts, nil
}

// LockUserUntil locks a user account until the specified time.
func (db *DB) LockUserUntil(ctx context.Context, userID uuid.UUID, until time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET status = 'locked', locked_until = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, until)
	if err != nil {
		return fmt.Errorf("lock user: %w", err)
	}
	return nil
}

// UnlockUser unlocks a user account.
func (db *DB) UnlockUser(ctx context.Context, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET status = 'active', locked_until = NULL, failed_login_attempts = 0, updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("unlock user: %w", err)
	}
	return nil
}

// SetUserPassword sets a user's password hash and optionally requires change on use.
func (db *DB) SetUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string, mustChange bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, password_changed_at = NOW(), must_change_password = $3, updated_at = NOW()
		WHERE id = $1
	`, userID, passwordHash, mustChange)
	if err != nil {
		return fmt.Errorf("set user password: %w", err)
	}
	return nil
}

// CreateInvitedUser creates a new user with pending status from an invitation.
func (db *DB) CreateInvitedUser(ctx context.Context, orgID uuid.UUID, email, name string, invitedBy uuid.UUID) (*models.User, error) {
	user := &models.User{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     email,
		Name:      name,
		Role:      models.UserRoleUser,
		Status:    models.UserStatusPending,
		InvitedBy: &invitedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	now := time.Now()
	user.InvitedAt = &now

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, org_id, email, name, role, status, invited_by, invited_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, user.ID, user.OrgID, user.Email, user.Name, string(user.Role), string(user.Status),
		user.InvitedBy, user.InvitedAt, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create invited user: %w", err)
	}
	return user, nil
}

// DeleteUser deletes a user by ID.
func (db *DB) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// IsSuperuser checks if a user is a superuser.
func (db *DB) IsSuperuser(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isSuperuser bool
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(is_superuser, false) FROM users WHERE id = $1
	`, userID).Scan(&isSuperuser)
	if err != nil {
		return false, fmt.Errorf("check superuser: %w", err)
	}
	return isSuperuser, nil
}

// SetSuperuser sets or clears the superuser flag for a user.
func (db *DB) SetSuperuser(ctx context.Context, userID uuid.UUID, isSuperuser bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET is_superuser = $2, updated_at = NOW() WHERE id = $1
	`, userID, isSuperuser)
	if err != nil {
		return fmt.Errorf("set superuser: %w", err)
	}
	return nil
}

// User Activity Log methods

// CreateUserActivityLog creates a new user activity log entry.
func (db *DB) CreateUserActivityLog(ctx context.Context, log *models.UserActivityLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_activity_logs (id, user_id, org_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, log.ID, log.UserID, log.OrgID, log.Action, log.ResourceType, log.ResourceID,
		log.IPAddress, log.UserAgent, log.Details, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("create user activity log: %w", err)
	}
	return nil
}

// GetUserActivityLogs returns activity logs for a user.
func (db *DB) GetUserActivityLogs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserActivityLogWithUser, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT l.id, l.user_id, l.org_id, l.action, l.resource_type, l.resource_id,
		       l.ip_address, l.user_agent, l.details, l.created_at,
		       u.email, u.name
		FROM user_activity_logs l
		JOIN users u ON l.user_id = u.id
		WHERE l.user_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get user activity logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.UserActivityLogWithUser
	for rows.Next() {
		var l models.UserActivityLogWithUser
		var details []byte
		err := rows.Scan(
			&l.ID, &l.UserID, &l.OrgID, &l.Action, &l.ResourceType, &l.ResourceID,
			&l.IPAddress, &l.UserAgent, &details, &l.CreatedAt,
			&l.UserEmail, &l.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user activity log: %w", err)
		}
		if details != nil {
			l.Details = json.RawMessage(details)
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

// GetOrgActivityLogs returns activity logs for an organization.
func (db *DB) GetOrgActivityLogs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.UserActivityLogWithUser, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT l.id, l.user_id, l.org_id, l.action, l.resource_type, l.resource_id,
		       l.ip_address, l.user_agent, l.details, l.created_at,
		       u.email, u.name
		FROM user_activity_logs l
		JOIN users u ON l.user_id = u.id
		WHERE l.org_id = $1
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get org activity logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.UserActivityLogWithUser
	for rows.Next() {
		var l models.UserActivityLogWithUser
		var details []byte
		err := rows.Scan(
			&l.ID, &l.UserID, &l.OrgID, &l.Action, &l.ResourceType, &l.ResourceID,
			&l.IPAddress, &l.UserAgent, &details, &l.CreatedAt,
			&l.UserEmail, &l.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user activity log: %w", err)
		}
		if details != nil {
			l.Details = json.RawMessage(details)
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

// User Impersonation Log methods

// CreateImpersonationLog creates a new impersonation log entry.
func (db *DB) CreateImpersonationLog(ctx context.Context, log *models.UserImpersonationLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_impersonation_logs (id, admin_user_id, target_user_id, org_id, reason, started_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.ID, log.AdminUserID, log.TargetUserID, log.OrgID, log.Reason,
		log.StartedAt, log.IPAddress, log.UserAgent)
	if err != nil {
		return fmt.Errorf("create impersonation log: %w", err)
	}
	return nil
}

// EndImpersonationLog marks an impersonation session as ended.
func (db *DB) EndImpersonationLog(ctx context.Context, logID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE user_impersonation_logs SET ended_at = NOW() WHERE id = $1
	`, logID)
	if err != nil {
		return fmt.Errorf("end impersonation log: %w", err)
	}
	return nil
}

// GetImpersonationLogsByOrg returns impersonation logs for an organization.
func (db *DB) GetImpersonationLogsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.UserImpersonationLogWithUsers, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT l.id, l.admin_user_id, l.target_user_id, l.org_id, l.reason,
		       l.started_at, l.ended_at, l.ip_address, l.user_agent,
		       a.email as admin_email, a.name as admin_name,
		       t.email as target_email, t.name as target_name
		FROM user_impersonation_logs l
		JOIN users a ON l.admin_user_id = a.id
		JOIN users t ON l.target_user_id = t.id
		WHERE l.org_id = $1
		ORDER BY l.started_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get impersonation logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.UserImpersonationLogWithUsers
	for rows.Next() {
		var l models.UserImpersonationLogWithUsers
		err := rows.Scan(
			&l.ID, &l.AdminUserID, &l.TargetUserID, &l.OrgID, &l.Reason,
			&l.StartedAt, &l.EndedAt, &l.IPAddress, &l.UserAgent,
			&l.AdminEmail, &l.AdminName, &l.TargetEmail, &l.TargetName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan impersonation log: %w", err)
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

// GetActiveImpersonation returns the active impersonation log for a user if any.
func (db *DB) GetActiveImpersonation(ctx context.Context, adminUserID uuid.UUID) (*models.UserImpersonationLog, error) {
	var log models.UserImpersonationLog
	err := db.Pool.QueryRow(ctx, `
		SELECT id, admin_user_id, target_user_id, org_id, reason, started_at, ended_at, ip_address, user_agent
		FROM user_impersonation_logs
		WHERE admin_user_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`, adminUserID).Scan(
		&log.ID, &log.AdminUserID, &log.TargetUserID, &log.OrgID, &log.Reason,
		&log.StartedAt, &log.EndedAt, &log.IPAddress, &log.UserAgent,
	)
	if err != nil {
		return nil, err
	}
	return &log, nil
}
