package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// CreateUserSession creates a new user session.
func (db *DB) CreateUserSession(ctx context.Context, session *models.UserSession) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, session_token_hash, ip_address, user_agent, created_at, last_active_at, expires_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, session.ID, session.UserID, session.SessionTokenHash, session.IPAddress, session.UserAgent,
		session.CreatedAt, session.LastActiveAt, session.ExpiresAt, session.Revoked)
	if err != nil {
		return fmt.Errorf("create user session: %w", err)
	}
	return nil
}

// GetUserSessionByID returns a user session by its ID.
func (db *DB) GetUserSessionByID(ctx context.Context, id uuid.UUID) (*models.UserSession, error) {
	var session models.UserSession
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, session_token_hash, ip_address, user_agent, created_at, last_active_at, expires_at, revoked, revoked_at
		FROM user_sessions
		WHERE id = $1
	`, id).Scan(
		&session.ID, &session.UserID, &session.SessionTokenHash, &session.IPAddress, &session.UserAgent,
		&session.CreatedAt, &session.LastActiveAt, &session.ExpiresAt, &session.Revoked, &session.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user session by ID: %w", err)
	}
	return &session, nil
}

// GetUserSessionByTokenHash returns a user session by its token hash.
func (db *DB) GetUserSessionByTokenHash(ctx context.Context, tokenHash string) (*models.UserSession, error) {
	var session models.UserSession
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, session_token_hash, ip_address, user_agent, created_at, last_active_at, expires_at, revoked, revoked_at
		FROM user_sessions
		WHERE session_token_hash = $1 AND revoked = false
	`, tokenHash).Scan(
		&session.ID, &session.UserID, &session.SessionTokenHash, &session.IPAddress, &session.UserAgent,
		&session.CreatedAt, &session.LastActiveAt, &session.ExpiresAt, &session.Revoked, &session.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user session by token hash: %w", err)
	}
	return &session, nil
}

// ListUserSessionsByUserID returns all sessions for a user.
func (db *DB) ListUserSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, session_token_hash, ip_address, user_agent, created_at, last_active_at, expires_at, revoked, revoked_at
		FROM user_sessions
		WHERE user_id = $1
		ORDER BY last_active_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.UserSession
	for rows.Next() {
		var session models.UserSession
		err := rows.Scan(
			&session.ID, &session.UserID, &session.SessionTokenHash, &session.IPAddress, &session.UserAgent,
			&session.CreatedAt, &session.LastActiveAt, &session.ExpiresAt, &session.Revoked, &session.RevokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// ListActiveUserSessionsByUserID returns only active (non-revoked, non-expired) sessions for a user.
func (db *DB) ListActiveUserSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, session_token_hash, ip_address, user_agent, created_at, last_active_at, expires_at, revoked, revoked_at
		FROM user_sessions
		WHERE user_id = $1
		  AND revoked = false
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY last_active_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list active user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.UserSession
	for rows.Next() {
		var session models.UserSession
		err := rows.Scan(
			&session.ID, &session.UserID, &session.SessionTokenHash, &session.IPAddress, &session.UserAgent,
			&session.CreatedAt, &session.LastActiveAt, &session.ExpiresAt, &session.Revoked, &session.RevokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// UpdateUserSessionLastActive updates the last_active_at timestamp for a session.
func (db *DB) UpdateUserSessionLastActive(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE user_sessions
		SET last_active_at = $2
		WHERE id = $1 AND revoked = false
	`, id, time.Now())
	if err != nil {
		return fmt.Errorf("update user session last active: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found or already revoked")
	}
	return nil
}

// RevokeUserSession revokes a single session.
func (db *DB) RevokeUserSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE user_sessions
		SET revoked = true, revoked_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked = false
	`, id, userID, now)
	if err != nil {
		return fmt.Errorf("revoke user session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found or already revoked")
	}
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user except the current one.
func (db *DB) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) (int64, error) {
	now := time.Now()
	var result any
	var err error

	if exceptSessionID != nil {
		result, err = db.Pool.Exec(ctx, `
			UPDATE user_sessions
			SET revoked = true, revoked_at = $3
			WHERE user_id = $1 AND id != $2 AND revoked = false
		`, userID, *exceptSessionID, now)
	} else {
		result, err = db.Pool.Exec(ctx, `
			UPDATE user_sessions
			SET revoked = true, revoked_at = $2
			WHERE user_id = $1 AND revoked = false
		`, userID, now)
	}
	if err != nil {
		return 0, fmt.Errorf("revoke all user sessions: %w", err)
	}

	// Type assert to get RowsAffected
	if r, ok := result.(interface{ RowsAffected() int64 }); ok {
		return r.RowsAffected(), nil
	}
	return 0, nil
}

// CleanupExpiredSessions removes expired sessions older than the given duration.
func (db *DB) CleanupExpiredSessions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM user_sessions
		WHERE (revoked = true AND revoked_at < $1)
		   OR (expires_at IS NOT NULL AND expires_at < $1)
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return result.RowsAffected(), nil
}
