package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Password Policy methods

// GetPasswordPolicyByOrgID returns the password policy for an organization.
func (db *DB) GetPasswordPolicyByOrgID(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	var p models.PasswordPolicy
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, min_length, require_uppercase, require_lowercase,
		       require_number, require_special, max_age_days, history_count,
		       created_at, updated_at
		FROM password_policies
		WHERE org_id = $1
	`, orgID).Scan(
		&p.ID, &p.OrgID, &p.MinLength, &p.RequireUppercase, &p.RequireLowercase,
		&p.RequireNumber, &p.RequireSpecial, &p.MaxAgeDays, &p.HistoryCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("password policy not found for org")
		}
		return nil, fmt.Errorf("get password policy: %w", err)
	}
	return &p, nil
}

// GetOrCreatePasswordPolicy returns the password policy for an organization, creating a default if none exists.
func (db *DB) GetOrCreatePasswordPolicy(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	policy, err := db.GetPasswordPolicyByOrgID(ctx, orgID)
	if err == nil {
		return policy, nil
	}

	// Create default policy
	policy = models.NewPasswordPolicy(orgID)
	if err := db.CreatePasswordPolicy(ctx, policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// CreatePasswordPolicy creates a new password policy.
func (db *DB) CreatePasswordPolicy(ctx context.Context, p *models.PasswordPolicy) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO password_policies (id, org_id, min_length, require_uppercase, require_lowercase,
		                               require_number, require_special, max_age_days, history_count,
		                               created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, p.ID, p.OrgID, p.MinLength, p.RequireUppercase, p.RequireLowercase,
		p.RequireNumber, p.RequireSpecial, p.MaxAgeDays, p.HistoryCount,
		p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create password policy: %w", err)
	}
	return nil
}

// UpdatePasswordPolicy updates an existing password policy.
func (db *DB) UpdatePasswordPolicy(ctx context.Context, p *models.PasswordPolicy) error {
	p.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE password_policies
		SET min_length = $2, require_uppercase = $3, require_lowercase = $4,
		    require_number = $5, require_special = $6, max_age_days = $7,
		    history_count = $8, updated_at = $9
		WHERE id = $1
	`, p.ID, p.MinLength, p.RequireUppercase, p.RequireLowercase,
		p.RequireNumber, p.RequireSpecial, p.MaxAgeDays, p.HistoryCount, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update password policy: %w", err)
	}
	return nil
}

// Password History methods

// GetPasswordHistory returns the password history for a user.
func (db *DB) GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*models.PasswordHistory, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, password_hash, created_at
		FROM password_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list password history: %w", err)
	}
	defer rows.Close()

	var history []*models.PasswordHistory
	for rows.Next() {
		var h models.PasswordHistory
		err := rows.Scan(&h.ID, &h.UserID, &h.PasswordHash, &h.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan password history: %w", err)
		}
		history = append(history, &h)
	}

	return history, nil
}

// CreatePasswordHistory creates a new password history entry.
func (db *DB) CreatePasswordHistory(ctx context.Context, h *models.PasswordHistory) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO password_history (id, user_id, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`, h.ID, h.UserID, h.PasswordHash, h.CreatedAt)
	if err != nil {
		return fmt.Errorf("create password history: %w", err)
	}
	return nil
}

// CleanupPasswordHistory removes old password history entries beyond the keep count.
func (db *DB) CleanupPasswordHistory(ctx context.Context, userID uuid.UUID, keepCount int) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM password_history
		WHERE user_id = $1 AND id NOT IN (
			SELECT id FROM password_history
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		)
	`, userID, keepCount)
	if err != nil {
		return fmt.Errorf("cleanup password history: %w", err)
	}
	return nil
}

// User Password methods

// GetUserByEmail returns a user by their email address within an organization.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	var roleStr string
	var oidcSubject *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, is_superuser,
		       COALESCE(email_verified, false), email_verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.OrgID, &oidcSubject, &user.Email,
		&user.Name, &roleStr, &user.IsSuperuser,
		&user.EmailVerified, &user.EmailVerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	user.Role = models.UserRole(roleStr)
	if oidcSubject != nil {
		user.OIDCSubject = *oidcSubject
	}
	return &user, nil
}

// GetUserPasswordInfo returns password-related information for a user.
func (db *DB) GetUserPasswordInfo(ctx context.Context, userID uuid.UUID) (*models.UserPasswordInfo, error) {
	var info models.UserPasswordInfo
	err := db.Pool.QueryRow(ctx, `
		SELECT id, password_hash, password_changed_at, password_expires_at, must_change_password
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&info.UserID, &info.PasswordHash, &info.PasswordChangedAt,
		&info.PasswordExpiresAt, &info.MustChangePassword,
	)
	if err != nil {
		return nil, fmt.Errorf("get user password info: %w", err)
	}
	return &info, nil
}

// UpdateUserPassword updates a user's password and related fields.
func (db *DB) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string, expiresAt *time.Time) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, password_changed_at = $3, password_expires_at = $4,
		    must_change_password = false, updated_at = $3
		WHERE id = $1
	`, userID, passwordHash, now, expiresAt)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

// SetUserMustChangePassword sets the must_change_password flag for a user.
func (db *DB) SetUserMustChangePassword(ctx context.Context, userID uuid.UUID, mustChange bool) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET must_change_password = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, mustChange)
	if err != nil {
		return fmt.Errorf("set user must change password: %w", err)
	}
	return nil
}

// GetUsersWithExpiredPasswords returns users whose passwords have expired.
func (db *DB) GetUsersWithExpiredPasswords(ctx context.Context, orgID uuid.UUID) ([]*models.User, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, oidc_subject, email, name, role, created_at, updated_at
		FROM users
		WHERE org_id = $1
		  AND password_expires_at IS NOT NULL
		  AND password_expires_at < NOW()
		  AND password_hash IS NOT NULL
		ORDER BY email
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list users with expired passwords: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var roleStr string
		var oidcSubject *string
		err := rows.Scan(
			&user.ID, &user.OrgID, &oidcSubject, &user.Email,
			&user.Name, &roleStr, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.Role = models.UserRole(roleStr)
		if oidcSubject != nil {
			user.OIDCSubject = *oidcSubject
		}
		users = append(users, &user)
	}

	return users, nil
}

// CreatePasswordUser creates a new user with password authentication (no OIDC).
func (db *DB) CreatePasswordUser(ctx context.Context, user *models.User, passwordHash string, expiresAt *time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, org_id, oidc_subject, email, name, role,
		                   password_hash, password_changed_at, password_expires_at,
		                   must_change_password, created_at, updated_at)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, user.ID, user.OrgID, user.Email, user.Name, string(user.Role),
		passwordHash, time.Now(), expiresAt, false, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create password user: %w", err)
	}
	return nil
}

// HasPasswordAuth returns true if the user has password authentication configured.
func (db *DB) HasPasswordAuth(ctx context.Context, userID uuid.UUID) (bool, error) {
	var hasPassword bool
	err := db.Pool.QueryRow(ctx, `
		SELECT password_hash IS NOT NULL
		FROM users
		WHERE id = $1
	`, userID).Scan(&hasPassword)
	if err != nil {
		return false, fmt.Errorf("check password auth: %w", err)
	}
	return hasPassword, nil
}

// Password Reset Token methods

// CreatePasswordResetToken creates a new password reset token.
func (db *DB) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.IPAddress, token.UserAgent, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

// GetPasswordResetTokenByHash returns a password reset token by its hash.
func (db *DB) GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, ip_address, user_agent, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
		&token.UsedAt, &token.IPAddress, &token.UserAgent, &token.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("password reset token not found")
		}
		return nil, fmt.Errorf("get password reset token: %w", err)
	}
	return &token, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used.
func (db *DB) MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = $2
		WHERE id = $1
	`, tokenID, now)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

// InvalidateUserResetTokens marks all unused reset tokens for a user as used.
func (db *DB) InvalidateUserResetTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = $2
		WHERE user_id = $1 AND used_at IS NULL
	`, userID, now)
	if err != nil {
		return fmt.Errorf("invalidate user reset tokens: %w", err)
	}
	return nil
}

// CleanupExpiredResetTokens removes expired password reset tokens.
func (db *DB) CleanupExpiredResetTokens(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM password_reset_tokens
		WHERE expires_at < NOW() - INTERVAL '24 hours'
		   OR (used_at IS NOT NULL AND used_at < NOW() - INTERVAL '24 hours')
	`)
	if err != nil {
		return fmt.Errorf("cleanup expired reset tokens: %w", err)
	}
	return nil
}

// Password Reset Rate Limiting methods

// GetResetRateLimit returns the rate limit entry for an identifier.
func (db *DB) GetResetRateLimit(ctx context.Context, identifier, identifierType string) (*models.PasswordResetRateLimit, error) {
	var limit models.PasswordResetRateLimit
	err := db.Pool.QueryRow(ctx, `
		SELECT id, identifier, identifier_type, request_count, window_start, created_at
		FROM password_reset_rate_limits
		WHERE identifier = $1 AND identifier_type = $2
		  AND window_start > NOW() - INTERVAL '15 minutes'
	`, identifier, identifierType).Scan(
		&limit.ID, &limit.Identifier, &limit.IdentifierType,
		&limit.RequestCount, &limit.WindowStart, &limit.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No rate limit entry exists
		}
		return nil, fmt.Errorf("get reset rate limit: %w", err)
	}
	return &limit, nil
}

// IncrementResetRateLimit increments the rate limit counter for an identifier.
func (db *DB) IncrementResetRateLimit(ctx context.Context, identifier, identifierType string, windowDuration time.Duration) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO password_reset_rate_limits (identifier, identifier_type, request_count, window_start)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (identifier, identifier_type)
		DO UPDATE SET
			request_count = CASE
				WHEN password_reset_rate_limits.window_start < NOW() - $3::interval
				THEN 1
				ELSE password_reset_rate_limits.request_count + 1
			END,
			window_start = CASE
				WHEN password_reset_rate_limits.window_start < NOW() - $3::interval
				THEN NOW()
				ELSE password_reset_rate_limits.window_start
			END
	`, identifier, identifierType, windowDuration.String())
	if err != nil {
		return fmt.Errorf("increment reset rate limit: %w", err)
	}
	return nil
}

// CleanupExpiredRateLimits removes expired rate limit entries.
func (db *DB) CleanupExpiredRateLimits(ctx context.Context, windowDuration time.Duration) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM password_reset_rate_limits
		WHERE window_start < NOW() - $1::interval
	`, windowDuration.String())
	if err != nil {
		return fmt.Errorf("cleanup expired rate limits: %w", err)
	}
	return nil
}
