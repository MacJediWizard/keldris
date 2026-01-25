package models

import (
	"time"

	"github.com/google/uuid"
)

// UserSession represents an authenticated user session.
type UserSession struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	SessionTokenHash string     `json:"-"`
	IPAddress        string     `json:"ip_address,omitempty"`
	UserAgent        string     `json:"user_agent,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastActiveAt     time.Time  `json:"last_active_at"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Revoked          bool       `json:"revoked"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	IsCurrent        bool       `json:"is_current,omitempty"`
}

// NewUserSession creates a new UserSession with the given details.
func NewUserSession(userID uuid.UUID, sessionTokenHash, ipAddress, userAgent string, expiresAt *time.Time) *UserSession {
	now := time.Now()
	return &UserSession{
		ID:               uuid.New(),
		UserID:           userID,
		SessionTokenHash: sessionTokenHash,
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		CreatedAt:        now,
		LastActiveAt:     now,
		ExpiresAt:        expiresAt,
		Revoked:          false,
	}
}

// IsExpired returns true if the session has expired.
func (s *UserSession) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// IsActive returns true if the session is not revoked and not expired.
func (s *UserSession) IsActive() bool {
	return !s.Revoked && !s.IsExpired()
}

// UserSessionsResponse is the response for listing user sessions.
type UserSessionsResponse struct {
	Sessions []UserSession `json:"sessions"`
}

// RevokeSessionsResponse is the response for revoking sessions.
type RevokeSessionsResponse struct {
	Message       string `json:"message"`
	RevokedCount  int    `json:"revoked_count,omitempty"`
}
