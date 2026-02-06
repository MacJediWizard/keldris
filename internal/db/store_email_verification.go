package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Email Verification Token methods

// CreateEmailVerificationToken stores a new verification token.
func (db *DB) CreateEmailVerificationToken(ctx context.Context, token *auth.EmailVerificationToken) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	return nil
}

// GetEmailVerificationTokenByHash retrieves a verification token by its hash.
func (db *DB) GetEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (*auth.EmailVerificationToken, error) {
	var token auth.EmailVerificationToken
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash,
		&token.ExpiresAt, &token.UsedAt, &token.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, auth.ErrTokenNotFound
		}
		return nil, fmt.Errorf("get email verification token: %w", err)
	}
	return &token, nil
}

// MarkEmailVerificationTokenUsed marks a verification token as used.
func (db *DB) MarkEmailVerificationTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL
	`, tokenID)
	if err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("token not found or already used")
	}
	return nil
}

// SetUserEmailVerified marks a user's email as verified.
func (db *DB) SetUserEmailVerified(ctx context.Context, userID uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET email_verified = true, email_verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("set user email verified: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// InvalidateUserVerificationTokens marks all unused tokens for a user as expired.
func (db *DB) InvalidateUserVerificationTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("invalidate verification tokens: %w", err)
	}
	return nil
}

// IsUserEmailVerified checks if a user's email is verified.
func (db *DB) IsUserEmailVerified(ctx context.Context, userID uuid.UUID) (bool, error) {
	var verified bool
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(email_verified, false) FROM users WHERE id = $1
	`, userID).Scan(&verified)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, auth.ErrUserNotFound
		}
		return false, fmt.Errorf("check email verification: %w", err)
	}
	return verified, nil
}

// GetUserEmailVerificationStatus returns detailed verification status.
type UserEmailVerificationStatus struct {
	UserID          uuid.UUID  `json:"user_id"`
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	IsOIDCUser      bool       `json:"is_oidc_user"`
}

// GetUserEmailVerificationStatus returns the email verification status for a user.
func (db *DB) GetUserEmailVerificationStatus(ctx context.Context, userID uuid.UUID) (*UserEmailVerificationStatus, error) {
	var status UserEmailVerificationStatus
	var oidcSubject *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(email_verified, false), email_verified_at, oidc_subject
		FROM users WHERE id = $1
	`, userID).Scan(
		&status.UserID, &status.Email, &status.EmailVerified,
		&status.EmailVerifiedAt, &oidcSubject,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user email verification status: %w", err)
	}
	status.IsOIDCUser = oidcSubject != nil && *oidcSubject != ""
	return &status, nil
}

// CleanupExpiredVerificationTokens removes expired and used verification tokens.
func (db *DB) CleanupExpiredVerificationTokens(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM email_verification_tokens
		WHERE (expires_at < NOW() OR used_at IS NOT NULL) AND created_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}
	return result.RowsAffected(), nil
}

// GetPendingVerificationUsers returns users who have not verified their email.
func (db *DB) GetPendingVerificationUsers(ctx context.Context, orgID uuid.UUID) ([]*UserEmailVerificationStatus, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT u.id, u.email, COALESCE(u.email_verified, false), u.email_verified_at, u.oidc_subject
		FROM users u
		JOIN org_memberships m ON u.id = m.user_id
		WHERE m.org_id = $1 AND COALESCE(u.email_verified, false) = false AND u.oidc_subject IS NULL
		ORDER BY u.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get pending verification users: %w", err)
	}
	defer rows.Close()

	var users []*UserEmailVerificationStatus
	for rows.Next() {
		var status UserEmailVerificationStatus
		var oidcSubject *string
		if err := rows.Scan(&status.UserID, &status.Email, &status.EmailVerified, &status.EmailVerifiedAt, &oidcSubject); err != nil {
			return nil, fmt.Errorf("scan pending verification user: %w", err)
		}
		status.IsOIDCUser = oidcSubject != nil && *oidcSubject != ""
		users = append(users, &status)
	}

	return users, nil
}

// verifiableUserWrapper wraps a user for the VerifiableUser interface.
type verifiableUserWrapper struct {
	id            uuid.UUID
	email         string
	emailVerified bool
	oidcSubject   *string
}

func (u *verifiableUserWrapper) GetID() uuid.UUID       { return u.id }
func (u *verifiableUserWrapper) GetEmail() string       { return u.email }
func (u *verifiableUserWrapper) IsEmailVerified() bool  { return u.emailVerified }
func (u *verifiableUserWrapper) IsOIDCUser() bool       { return u.oidcSubject != nil && *u.oidcSubject != "" }

// GetUserByIDForVerification returns a user for verification purposes.
func (db *DB) GetUserByIDForVerification(ctx context.Context, userID uuid.UUID) (auth.VerifiableUser, error) {
	var u verifiableUserWrapper
	u.id = userID
	err := db.Pool.QueryRow(ctx, `
		SELECT email, COALESCE(email_verified, false), oidc_subject
		FROM users WHERE id = $1
	`, userID).Scan(&u.email, &u.emailVerified, &u.oidcSubject)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user for verification: %w", err)
	}
	return &u, nil
}

// AdminSetUserEmailVerified allows an admin to bypass email verification for a user.
func (db *DB) AdminSetUserEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error {
	var verifiedAt interface{} = nil
	if verified {
		verifiedAt = time.Now()
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE users
		SET email_verified = $2, email_verified_at = $3, updated_at = NOW()
		WHERE id = $1
	`, userID, verified, verifiedAt)
	if err != nil {
		return fmt.Errorf("admin set email verified: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
