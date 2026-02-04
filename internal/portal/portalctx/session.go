package portalctx

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionCookieName is the name of the portal session cookie.
	SessionCookieName = "keldris_portal_session"
	// SessionDuration is the default session duration (24 hours).
	SessionDuration = 24 * time.Hour
	// TokenLength is the length of the session token in bytes.
	TokenLength = 32
)

// Session represents a customer session in the portal.
type Session struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	TokenHash  string    `json:"-"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// SessionUser represents the authenticated customer in a request context.
type SessionUser struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	Name    string    `json:"name"`
	Company string    `json:"company,omitempty"`
}

// GenerateSessionToken generates a cryptographically secure session token.
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashSessionToken creates a SHA-256 hash of a session token for storage.
func HashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// NewSession creates a new session for a customer.
func NewSession(customerID uuid.UUID, ipAddress, userAgent string) (*Session, string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return nil, "", err
	}

	session := &Session{
		ID:         uuid.New(),
		CustomerID: customerID,
		TokenHash:  HashSessionToken(token),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		ExpiresAt:  time.Now().Add(SessionDuration),
		CreatedAt:  time.Now(),
	}

	return session, token, nil
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
