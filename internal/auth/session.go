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
	// CurrentOrgIDKey is the session key for the currently selected organization.
	CurrentOrgIDKey = "current_org_id"
	// CurrentOrgRoleKey is the session key for the user's role in the current org.
	CurrentOrgRoleKey = "current_org_role"
	// LastActivityKey is the session key for the last activity timestamp.
	LastActivityKey = "last_activity"
	// SessionRecordIDKey is the session key for the database session record ID.
	SessionRecordIDKey = "session_record_id"
// IsSuperuserKey is the session key for superuser status.
	IsSuperuserKey = "is_superuser"
	// ImpersonatingKey is the session key for impersonation state.
	ImpersonatingKey = "impersonating"
	// IsSuperuserKey is the session key for superuser status.
	IsSuperuserKey = "is_superuser"
	// ImpersonatingKey is the session key for impersonation state.
	ImpersonatingKey = "impersonating"
	// ImpersonatingUserIDKey is the session key for the user being impersonated.
	ImpersonatingUserIDKey = "impersonating_user_id"
	// OriginalUserIDKey is the session key for the original superuser ID during impersonation.
	OriginalUserIDKey = "original_user_id"
	// OriginalUserEmailKey is the session key for the original user email during impersonation.
	OriginalUserEmailKey = "original_user_email"
	// ImpersonationLogIDKey is the session key for the impersonation log ID.
	ImpersonationLogIDKey = "impersonation_log_id"
)

// SessionConfig holds session store configuration.
type SessionConfig struct {
	Secret      []byte
	MaxAge      int  // seconds
	IdleTimeout int  // seconds, 0 to disable
	Secure      bool // require HTTPS
	HTTPOnly    bool // prevent JavaScript access
	SameSite    http.SameSite
	CookiePath  string
}

// DefaultSessionConfig returns a SessionConfig with secure defaults.
// maxAge in seconds (0 or negative uses default 86400). idleTimeout in seconds (0 disables idle timeout, negative uses default 1800).
func DefaultSessionConfig(secret []byte, secure bool, maxAge, idleTimeout int) SessionConfig {
	if maxAge <= 0 {
		maxAge = 86400 // 24 hours
	}
	if idleTimeout < 0 {
		idleTimeout = 1800 // 30 minutes
	}
	return SessionConfig{
		Secret:      secret,
		MaxAge:      maxAge,
		IdleTimeout: idleTimeout,
		Secure:      secure,
		HTTPOnly:    true,
		SameSite:    http.SameSiteLaxMode,
		CookiePath:  "/",
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
		Secret:      secret,
		MaxAge:      86400, // 24 hours
		IdleTimeout: 1800,  // 30 minutes
		Secure:      secure,
		HTTPOnly:    true,
		SameSite:    http.SameSiteLaxMode,
		CookiePath:  "/",
	}
}

// SessionStore wraps a gorilla/sessions store with helper methods.
type SessionStore struct {
	store       *sessions.CookieStore
	idleTimeout time.Duration
	logger      zerolog.Logger
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
		store:       store,
		idleTimeout: time.Duration(cfg.IdleTimeout) * time.Second,
		logger:      logger.With().Str("component", "session").Logger(),
		store:  store,
		logger: logger.With().Str("component", "session").Logger(),
	}

	s.logger.Info().
		Bool("secure", cfg.Secure).
		Int("max_age", cfg.MaxAge).
		Int("idle_timeout", cfg.IdleTimeout).
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
ID                 uuid.UUID
	ID                 uuid.UUID
	OIDCSubject        string
	Email              string
	Name               string
	AuthenticatedAt    time.Time
	CurrentOrgID       uuid.UUID
	CurrentOrgRole     string
	SessionRecordID    uuid.UUID
	IsSuperuser        bool
	// Impersonation fields
	Impersonating      bool
	ImpersonatingID    uuid.UUID // The user being impersonated (if any)
	OriginalUserID     uuid.UUID // The original superuser ID (during impersonation)
	OriginalUserEmail  string
	ImpersonationLogID uuid.UUID
}

// IsImpersonating returns true if the user is being impersonated.
func (u *SessionUser) IsImpersonating() bool {
	return u.Impersonating && u.OriginalUserID != uuid.Nil
	ID              uuid.UUID
	OIDCSubject     string
	Email           string
	Name            string
	AuthenticatedAt time.Time
	CurrentOrgID    uuid.UUID
	CurrentOrgRole  string
	ImpersonatingID    uuid.UUID // The user being impersonated (if any)
	OriginalUserID     uuid.UUID // The original superuser ID (during impersonation)
	OriginalUserEmail  string
	ImpersonationLogID uuid.UUID
}

// IsImpersonating returns true if the user is being impersonated.
func (u *SessionUser) IsImpersonating() bool {
	return u.Impersonating && u.OriginalUserID != uuid.Nil
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
	session.Values[CurrentOrgIDKey] = user.CurrentOrgID
	session.Values[CurrentOrgRoleKey] = user.CurrentOrgRole
	session.Values[LastActivityKey] = time.Now()
	session.Values[SessionRecordIDKey] = user.SessionRecordID
	session.Values[IsSuperuserKey] = user.IsSuperuser
	session.Values[ImpersonatingUserIDKey] = user.ImpersonatingID
	session.Values[OriginalUserIDKey] = user.OriginalUserID
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

	// Check idle timeout if enabled
	if s.idleTimeout > 0 {
		lastActivity, ok := session.Values[LastActivityKey].(time.Time)
		if ok && time.Since(lastActivity) > s.idleTimeout {
			return nil, fmt.Errorf("session idle timeout exceeded")
		}
	}

	oidcSubject, _ := session.Values[OIDCSubjectKey].(string)
	email, _ := session.Values[EmailKey].(string)
	name, _ := session.Values[NameKey].(string)
	authenticatedAt, _ := session.Values[AuthenticatedAtKey].(time.Time)
	currentOrgID, _ := session.Values[CurrentOrgIDKey].(uuid.UUID)
	currentOrgRole, _ := session.Values[CurrentOrgRoleKey].(string)
	sessionRecordID, _ := session.Values[SessionRecordIDKey].(uuid.UUID)
	isSuperuser, _ := session.Values[IsSuperuserKey].(bool)
	impersonatingID, _ := session.Values[ImpersonatingUserIDKey].(uuid.UUID)
	originalUserID, _ := session.Values[OriginalUserIDKey].(uuid.UUID)

	// Impersonation fields
	impersonating, _ := session.Values[ImpersonatingKey].(bool)
	originalUserEmail, _ := session.Values[OriginalUserEmailKey].(string)
	impersonationLogID, _ := session.Values[ImpersonationLogIDKey].(uuid.UUID)

	return &SessionUser{
		ID:                 userID,
		OIDCSubject:        oidcSubject,
		Email:              email,
		Name:               name,
		AuthenticatedAt:    authenticatedAt,
		CurrentOrgID:       currentOrgID,
		CurrentOrgRole:     currentOrgRole,
		SessionRecordID:    sessionRecordID,
		IsSuperuser:        isSuperuser,
		Impersonating:      impersonating,
		ImpersonatingID:    impersonatingID,
		OriginalUserID:     originalUserID,
		OriginalUserEmail:  originalUserEmail,
		ImpersonationLogID: impersonationLogID,
	}, nil
}

// TouchSession updates the last activity timestamp to keep the session alive.
// Call this on each authenticated request to track idle timeout.
func (s *SessionStore) TouchSession(r *http.Request, w http.ResponseWriter) error {
	if s.idleTimeout <= 0 {
		return nil
	}

	session, err := s.Get(r)
	if err != nil {
		return err
	}

	session.Values[LastActivityKey] = time.Now()
	return s.Save(r, w, session)
}


	return &SessionUser{
		ID:                 userID,
		OIDCSubject:        oidcSubject,
		Email:              email,
		Name:               name,
		AuthenticatedAt:    authenticatedAt,
		CurrentOrgID:       currentOrgID,
		CurrentOrgRole:     currentOrgRole,
		SessionRecordID:    sessionRecordID,
		IsSuperuser:        isSuperuser,
		Impersonating:      impersonating,
		ImpersonatingID:    impersonatingID,
		OriginalUserID:     originalUserID,
		OriginalUserEmail:  originalUserEmail,
		ImpersonationLogID: impersonationLogID,
	}, nil
}

// TouchSession updates the last activity timestamp to keep the session alive.
// Call this on each authenticated request to track idle timeout.
func (s *SessionStore) TouchSession(r *http.Request, w http.ResponseWriter) error {
	if s.idleTimeout <= 0 {
		return nil
	}

	session, err := s.Get(r)
	if err != nil {
		return err
	}

	session.Values[LastActivityKey] = time.Now()
	return s.Save(r, w, session)
}

// SetCurrentOrg updates the current organization in the session.
func (s *SessionStore) SetCurrentOrg(r *http.Request, w http.ResponseWriter, orgID uuid.UUID, role string) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[CurrentOrgIDKey] = orgID
	session.Values[CurrentOrgRoleKey] = role
	return s.Save(r, w, session)
}

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
	delete(session.Values, CurrentOrgIDKey)
	delete(session.Values, CurrentOrgRoleKey)
	delete(session.Values, LastActivityKey)
	delete(session.Values, SessionRecordIDKey)
	delete(session.Values, IsSuperuserKey)
	delete(session.Values, ImpersonatingUserIDKey)
	delete(session.Values, OriginalUserIDKey)
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
	if !ok {
		return false
	}

	// Check idle timeout if enabled
	if s.idleTimeout > 0 {
		lastActivity, ok := session.Values[LastActivityKey].(time.Time)
		if ok && time.Since(lastActivity) > s.idleTimeout {
			return false
		}
	}

	return true
}

// SetSuperuserStatus updates the superuser status in the session.
func (s *SessionStore) SetSuperuserStatus(r *http.Request, w http.ResponseWriter, isSuperuser bool) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[IsSuperuserKey] = isSuperuser
	return s.Save(r, w, session)
}

// StartImpersonation sets up impersonation mode where a superuser acts as another user.
func (s *SessionStore) StartImpersonation(r *http.Request, w http.ResponseWriter, originalUser *SessionUser, targetUser *SessionUser, logID uuid.UUID) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}

	// Store original superuser info
	session.Values[ImpersonatingKey] = true
	session.Values[OriginalUserIDKey] = originalUser.ID
	session.Values[OriginalUserEmailKey] = originalUser.Email
	session.Values[ImpersonatingUserIDKey] = targetUser.ID
	session.Values[ImpersonationLogIDKey] = logID

	// Switch to target user's identity
	session.Values[UserIDKey] = targetUser.ID
	session.Values[OIDCSubjectKey] = targetUser.OIDCSubject
	session.Values[EmailKey] = targetUser.Email
	session.Values[NameKey] = targetUser.Name
	session.Values[CurrentOrgIDKey] = targetUser.CurrentOrgID
	session.Values[CurrentOrgRoleKey] = targetUser.CurrentOrgRole

	// Maintain superuser status but mark as impersonating
	session.Values[IsSuperuserKey] = true

	return s.Save(r, w, session)
}

// EndImpersonation restores the original superuser session.
func (s *SessionStore) EndImpersonation(r *http.Request, w http.ResponseWriter, originalUser *SessionUser) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}

	// Restore original user
	session.Values[UserIDKey] = originalUser.ID
	session.Values[OIDCSubjectKey] = originalUser.OIDCSubject
	session.Values[EmailKey] = originalUser.Email
	session.Values[NameKey] = originalUser.Name
	session.Values[CurrentOrgIDKey] = originalUser.CurrentOrgID
	session.Values[CurrentOrgRoleKey] = originalUser.CurrentOrgRole
	session.Values[IsSuperuserKey] = originalUser.IsSuperuser

	// Clear impersonation state
	delete(session.Values, ImpersonatingKey)
	delete(session.Values, ImpersonatingUserIDKey)
	delete(session.Values, OriginalUserIDKey)
	delete(session.Values, OriginalUserEmailKey)
	delete(session.Values, ImpersonationLogIDKey)

	return s.Save(r, w, session)
}

// GetImpersonationLogID returns the current impersonation log ID if impersonating.
func (s *SessionStore) GetImpersonationLogID(r *http.Request) (uuid.UUID, error) {
	session, err := s.Get(r)
	if err != nil {
		return uuid.Nil, err
	}

	logID, ok := session.Values[ImpersonationLogIDKey].(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("not impersonating")
	}
	return logID, nil
}

// IsImpersonating checks if the current session is in impersonation mode.
func (s *SessionStore) IsImpersonating(r *http.Request) bool {
	session, err := s.Get(r)
	if err != nil {
		return false
	}
	_, ok := session.Values[ImpersonatingUserIDKey].(uuid.UUID)
	return ok
}

// GetOriginalUserID returns the original superuser ID during impersonation.
func (s *SessionStore) GetOriginalUserID(r *http.Request) uuid.UUID {
	session, err := s.Get(r)
	if err != nil {
		return uuid.Nil
	}
	id, _ := session.Values[OriginalUserIDKey].(uuid.UUID)
	return id
}
	return ok
}

// SetSuperuserStatus updates the superuser status in the session.
func (s *SessionStore) SetSuperuserStatus(r *http.Request, w http.ResponseWriter, isSuperuser bool) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}
	session.Values[IsSuperuserKey] = isSuperuser
	return s.Save(r, w, session)
}

// StartImpersonation sets up impersonation mode where a superuser acts as another user.
func (s *SessionStore) StartImpersonation(r *http.Request, w http.ResponseWriter, originalUser *SessionUser, targetUser *SessionUser, logID uuid.UUID) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}

	// Store original superuser info
	session.Values[ImpersonatingKey] = true
	session.Values[OriginalUserIDKey] = originalUser.ID
	session.Values[OriginalUserEmailKey] = originalUser.Email
	session.Values[ImpersonatingUserIDKey] = targetUser.ID
	session.Values[ImpersonationLogIDKey] = logID

	// Switch to target user's identity
	session.Values[UserIDKey] = targetUser.ID
	session.Values[OIDCSubjectKey] = targetUser.OIDCSubject
	session.Values[EmailKey] = targetUser.Email
	session.Values[NameKey] = targetUser.Name
	session.Values[CurrentOrgIDKey] = targetUser.CurrentOrgID
	session.Values[CurrentOrgRoleKey] = targetUser.CurrentOrgRole

	// Maintain superuser status but mark as impersonating
	session.Values[IsSuperuserKey] = true

	return s.Save(r, w, session)
}

// EndImpersonation restores the original superuser session.
func (s *SessionStore) EndImpersonation(r *http.Request, w http.ResponseWriter, originalUser *SessionUser) error {
	session, err := s.Get(r)
	if err != nil {
		return err
	}

	// Restore original user
	session.Values[UserIDKey] = originalUser.ID
	session.Values[OIDCSubjectKey] = originalUser.OIDCSubject
	session.Values[EmailKey] = originalUser.Email
	session.Values[NameKey] = originalUser.Name
	session.Values[CurrentOrgIDKey] = originalUser.CurrentOrgID
	session.Values[CurrentOrgRoleKey] = originalUser.CurrentOrgRole
	session.Values[IsSuperuserKey] = originalUser.IsSuperuser

	// Clear impersonation state
	delete(session.Values, ImpersonatingKey)
	delete(session.Values, ImpersonatingUserIDKey)
	delete(session.Values, OriginalUserIDKey)
	delete(session.Values, OriginalUserEmailKey)
	delete(session.Values, ImpersonationLogIDKey)

	return s.Save(r, w, session)
}

// GetImpersonationLogID returns the current impersonation log ID if impersonating.
func (s *SessionStore) GetImpersonationLogID(r *http.Request) (uuid.UUID, error) {
	session, err := s.Get(r)
	if err != nil {
		return uuid.Nil, err
	}

	logID, ok := session.Values[ImpersonationLogIDKey].(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("not impersonating")
	}
	return logID, nil
}

// IsImpersonating checks if the current session is in impersonation mode.
func (s *SessionStore) IsImpersonating(r *http.Request) bool {
	session, err := s.Get(r)
	if err != nil {
		return false
	}
	_, ok := session.Values[ImpersonatingUserIDKey].(uuid.UUID)
	return ok
}

// GetOriginalUserID returns the original superuser ID during impersonation.
func (s *SessionStore) GetOriginalUserID(r *http.Request) uuid.UUID {
	session, err := s.Get(r)
	if err != nil {
		return uuid.Nil
	}
	id, _ := session.Values[OriginalUserIDKey].(uuid.UUID)
	return id
}
