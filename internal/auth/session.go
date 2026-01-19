package auth

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
)

func init() {
	// Register types for session serialization
	gob.Register(uuid.UUID{})
	gob.Register(time.Time{})
}

const (
	// SessionName is the name of the session cookie.
	SessionName = "keldris_session"
	// StateKey is the session key for OIDC state.
	StateKey = "oidc_state"
	// UserIDKey is the session key for the authenticated user ID.
	UserIDKey = "user_id"
	// OIDCSubjectKey is the session key for the OIDC subject.
	OIDCSubjectKey = "oidc_subject"
	// EmailKey is the session key for the user's email.
	EmailKey = "email"
	// NameKey is the session key for the user's name.
	NameKey = "name"
	// AuthenticatedAtKey is the session key for when the user authenticated.
	AuthenticatedAtKey = "authenticated_at"
)

// SessionConfig holds session store configuration.
type SessionConfig struct {
	Secret     []byte
	MaxAge     int  // seconds
	Secure     bool // require HTTPS
	HTTPOnly   bool // prevent JavaScript access
	SameSite   http.SameSite
	CookiePath string
}

// DefaultSessionConfig returns a SessionConfig with secure defaults.
func DefaultSessionConfig(secret []byte, secure bool) SessionConfig {
	return SessionConfig{
		Secret:     secret,
		MaxAge:     86400, // 24 hours
		Secure:     secure,
		HTTPOnly:   true,
		SameSite:   http.SameSiteLaxMode,
		CookiePath: "/",
	}
}

// SessionStore wraps a gorilla/sessions store with helper methods.
type SessionStore struct {
	store  *sessions.CookieStore
	logger zerolog.Logger
}

// NewSessionStore creates a new session store.
func NewSessionStore(cfg SessionConfig, logger zerolog.Logger) (*SessionStore, error) {
	if len(cfg.Secret) < 32 {
		return nil, fmt.Errorf("session secret must be at least 32 bytes")
	}

	store := sessions.NewCookieStore(cfg.Secret)
	store.Options = &sessions.Options{
		Path:     cfg.CookiePath,
		MaxAge:   cfg.MaxAge,
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	}

	s := &SessionStore{
		store:  store,
		logger: logger.With().Str("component", "session").Logger(),
	}

	s.logger.Info().
		Bool("secure", cfg.Secure).
		Int("max_age", cfg.MaxAge).
		Msg("session store initialized")

	return s, nil
}

// Get retrieves a session from the request.
func (s *SessionStore) Get(r *http.Request) (*sessions.Session, error) {
	session, err := s.store.Get(r, SessionName)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}

// Save saves the session to the response.
func (s *SessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// SetOIDCState stores the OIDC state in the session.
func (s *SessionStore) SetOIDCState(r *http.Request, w http.ResponseWriter, state string) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[StateKey] = state
	return s.Save(r, w, session)
}

// GetOIDCState retrieves and clears the OIDC state from the session.
func (s *SessionStore) GetOIDCState(r *http.Request, w http.ResponseWriter) (string, error) {
	session, err := s.Get(r)
	if err != nil {
		return "", err
	}
	state, ok := session.Values[StateKey].(string)
	if !ok {
		return "", fmt.Errorf("no state in session")
	}
	// Clear the state after retrieval
	delete(session.Values, StateKey)
	if err := s.Save(r, w, session); err != nil {
		return "", err
	}
	return state, nil
}

// SessionUser represents the authenticated user data stored in session.
type SessionUser struct {
	ID              uuid.UUID
	OIDCSubject     string
	Email           string
	Name            string
	AuthenticatedAt time.Time
}

// SetUser stores user data in the session after successful authentication.
func (s *SessionStore) SetUser(r *http.Request, w http.ResponseWriter, user *SessionUser) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[UserIDKey] = user.ID
	session.Values[OIDCSubjectKey] = user.OIDCSubject
	session.Values[EmailKey] = user.Email
	session.Values[NameKey] = user.Name
	session.Values[AuthenticatedAtKey] = user.AuthenticatedAt
	return s.Save(r, w, session)
}

// GetUser retrieves the authenticated user from the session.
func (s *SessionStore) GetUser(r *http.Request) (*SessionUser, error) {
	session, err := s.Get(r)
	if err != nil {
		return nil, err
	}

	userID, ok := session.Values[UserIDKey].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("no user in session")
	}

	oidcSubject, _ := session.Values[OIDCSubjectKey].(string)
	email, _ := session.Values[EmailKey].(string)
	name, _ := session.Values[NameKey].(string)
	authenticatedAt, _ := session.Values[AuthenticatedAtKey].(time.Time)

	return &SessionUser{
		ID:              userID,
		OIDCSubject:     oidcSubject,
		Email:           email,
		Name:            name,
		AuthenticatedAt: authenticatedAt,
	}, nil
}

// ClearUser removes user data from the session (logout).
func (s *SessionStore) ClearUser(r *http.Request, w http.ResponseWriter) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	delete(session.Values, UserIDKey)
	delete(session.Values, OIDCSubjectKey)
	delete(session.Values, EmailKey)
	delete(session.Values, NameKey)
	delete(session.Values, AuthenticatedAtKey)
	// Set MaxAge to -1 to delete the cookie
	session.Options.MaxAge = -1
	return s.Save(r, w, session)
}

// IsAuthenticated checks if the session has a valid authenticated user.
func (s *SessionStore) IsAuthenticated(r *http.Request) bool {
	session, err := s.Get(r)
	if err != nil {
		return false
	}
	_, ok := session.Values[UserIDKey].(uuid.UUID)
	return ok
}
