// Package handlers provides HTTP handlers for the Keldris API.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UserStore defines the interface for user persistence operations.
type UserStore interface {
	GetUserByOIDCSubject(ctx context.Context, subject string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	GetOrCreateDefaultOrg(ctx context.Context) (*models.Organization, error)
	GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error)
	CreateMembership(ctx context.Context, m *models.OrgMembership) error
	// SSO group sync methods (implements auth.GroupSyncStore)
	GetSSOGroupMappingsByGroupNames(ctx context.Context, groupNames []string) ([]*models.SSOGroupMapping, error)
	GetSSOGroupMappingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
	UpdateMembershipRole(ctx context.Context, membershipID uuid.UUID, role models.OrgRole) error
	UpsertUserSSOGroups(ctx context.Context, userID uuid.UUID, groups []string) error
	GetUserSSOGroups(ctx context.Context, userID uuid.UUID) (*models.UserSSOGroups, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error)
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	// User session tracking methods
	CreateUserSession(ctx context.Context, session *models.UserSession) error
	RevokeUserSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	// Password authentication methods
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserPasswordInfo(ctx context.Context, userID uuid.UUID) (*models.UserPasswordInfo, error)
}

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	oidc      *auth.OIDC
	sessions  *auth.SessionStore
	userStore UserStore
	groupSync *auth.GroupSync
	logger    zerolog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(oidc *auth.OIDC, sessions *auth.SessionStore, userStore UserStore, logger zerolog.Logger) *AuthHandler {
	// Create GroupSync using the userStore which implements GroupSyncStore
	groupSync := auth.NewGroupSync(userStore, logger)

	return &AuthHandler{
		oidc:      oidc,
		sessions:  sessions,
		userStore: userStore,
		groupSync: groupSync,
		logger:    logger.With().Str("component", "auth_handler").Logger(),
	}
}

// RegisterRoutes registers auth routes on the given router group.
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/login", h.Login)
	r.GET("/callback", h.Callback)
	r.POST("/logout", h.Logout)
	r.GET("/me", h.Me)
	// Password authentication routes
	r.POST("/login/password", h.PasswordLogin)
}

// Login initiates the OIDC authentication flow.
//
//	@Summary		Initiate login
//	@Description	Redirects to the OIDC provider for authentication. After successful authentication, user is redirected to /auth/callback.
//	@Tags			Auth
//	@Produce		html
//	@Success		307	"Redirect to OIDC provider"
//	@Failure		500	{object}	map[string]string
//	@Router			/auth/login [get]
func (h *AuthHandler) Login(c *gin.Context) {
	state, err := auth.GenerateState()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate state")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate login"})
		return
	}

	if err := h.sessions.SetOIDCState(c.Request, c.Writer, state); err != nil {
		h.logger.Error().Err(err).Msg("failed to save state to session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate login"})
		return
	}

	authURL := h.oidc.AuthorizationURL(state)
	h.logger.Debug().Str("redirect_url", authURL).Msg("redirecting to OIDC provider")
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// Callback handles the OIDC callback after authentication.
//
//	@Summary		OIDC callback
//	@Description	Handles the callback from the OIDC provider after authentication. Creates or updates user session.
//	@Tags			Auth
//	@Param			code	query	string	true	"Authorization code"
//	@Param			state	query	string	true	"State parameter for CSRF protection"
//	@Success		307		"Redirect to dashboard"
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/auth/callback [get]
func (h *AuthHandler) Callback(c *gin.Context) {
	// Check for errors from the OIDC provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		h.logger.Warn().
			Str("error", errParam).
			Str("description", errDesc).
			Msg("OIDC provider returned error")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errParam,
			"description": errDesc,
		})
		return
	}

	// Verify state parameter
	state := c.Query("state")
	if state == "" {
		h.logger.Warn().Msg("missing state parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing state parameter"})
		return
	}

	savedState, err := h.sessions.GetOIDCState(c.Request, c.Writer)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to retrieve state from session")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session state"})
		return
	}

	if state != savedState {
		h.logger.Warn().Msg("state parameter mismatch")
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}

	// Exchange authorization code for tokens
	code := c.Query("code")
	if code == "" {
		h.logger.Warn().Msg("missing authorization code")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	token, err := h.oidc.Exchange(c.Request.Context(), code)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to exchange authorization code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	// Verify ID token and extract claims
	claims, err := h.oidc.VerifyIDToken(c.Request.Context(), token)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to verify ID token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	// Find or create user
	user, err := h.findOrCreateUser(c.Request.Context(), claims)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to find or create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	// Extract and sync OIDC groups
	groups, err := h.groupSync.ExtractGroupsFromToken(c.Request.Context(), h.oidc, token)
	if err != nil {
		// Log the error but don't fail authentication
		h.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to extract groups from token")
	} else if len(groups) > 0 {
		// Sync user's groups to memberships
		syncResult, err := h.groupSync.SyncUserGroups(c.Request.Context(), user.ID, groups)
		if err != nil {
			h.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to sync user groups")
		} else {
			h.logger.Info().
				Str("user_id", user.ID.String()).
				Int("groups_received", len(syncResult.GroupsReceived)).
				Int("memberships_added", len(syncResult.MembershipsAdded)).
				Msg("user groups synced on login")

			// Create audit log for group sync if any memberships were added
			if len(syncResult.MembershipsAdded) > 0 {
				for _, mapping := range syncResult.MembershipsAdded {
					auditLog := models.NewAuditLog(mapping.OrgID, models.AuditActionCreate, "membership", models.AuditResultSuccess).
						WithUser(user.ID).
						WithDetails("SSO group sync: " + mapping.OIDCGroupName)
					if err := h.userStore.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
						h.logger.Warn().Err(err).Msg("failed to create audit log for group sync")
					}
				}
			}
		}
	}

	// Get user's memberships to set current org (after group sync)
	memberships, err := h.userStore.GetMembershipsByUserID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user memberships")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	var currentOrgID uuid.UUID
	var currentOrgRole string
	if len(memberships) > 0 {
		// Default to first org
		currentOrgID = memberships[0].OrgID
		currentOrgRole = string(memberships[0].Role)
	}

	// Create a session record in the database
	sessionRecordID := uuid.New()
	sessionTokenHash := generateSessionTokenHash(sessionRecordID)
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Calculate session expiration (24 hours by default, matching cookie)
	expiresAt := time.Now().Add(24 * time.Hour)

	userSession := models.NewUserSession(user.ID, sessionTokenHash, ipAddress, userAgent, &expiresAt)
	userSession.ID = sessionRecordID

	if err := h.userStore.CreateUserSession(c.Request.Context(), userSession); err != nil {
		h.logger.Error().Err(err).Msg("failed to create user session record")
		// Continue anyway - session tracking is not critical for authentication
	}

	// Store user in session
	sessionUser := &auth.SessionUser{
		ID:              user.ID,
		OIDCSubject:     user.OIDCSubject,
		Email:           user.Email,
		Name:            user.Name,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    currentOrgID,
		CurrentOrgRole:  currentOrgRole,
		SessionRecordID: sessionRecordID,
	}

	if err := h.sessions.SetUser(c.Request, c.Writer, sessionUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to save user to session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Str("session_id", sessionRecordID.String()).
		Msg("user authenticated successfully")

	// Redirect to frontend dashboard
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// findOrCreateUser finds an existing user by OIDC subject or creates a new one.
func (h *AuthHandler) findOrCreateUser(ctx context.Context, claims *auth.IDTokenClaims) (*models.User, error) {
	// Try to find existing user
	user, err := h.userStore.GetUserByOIDCSubject(ctx, claims.Subject)
	if err == nil {
		return user, nil
	}

	// Create new user - get or create default organization
	org, err := h.userStore.GetOrCreateDefaultOrg(ctx)
	if err != nil {
		return nil, err
	}

	user = models.NewUser(org.ID, claims.Subject, claims.Email, claims.Name, models.UserRoleUser)
	if err := h.userStore.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Create membership for the user in the default org as owner (first user becomes owner)
	membership := models.NewOrgMembership(user.ID, org.ID, models.OrgRoleOwner)
	if err := h.userStore.CreateMembership(ctx, membership); err != nil {
		h.logger.Error().Err(err).Msg("failed to create membership for new user")
		// Don't fail the user creation, just log the error
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Str("org_id", org.ID.String()).
		Msg("created new user with org membership")

	return user, nil
}

// Logout terminates the user session.
//
//	@Summary		Logout
//	@Description	Terminates the current user session and clears the session cookie
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionUser, err := h.sessions.GetUser(c.Request)
	if err == nil {
		h.logger.Info().
			Str("user_id", sessionUser.ID.String()).
			Msg("user logging out")

		// Revoke the session record in the database
		if sessionUser.SessionRecordID != uuid.Nil {
			if err := h.userStore.RevokeUserSession(c.Request.Context(), sessionUser.SessionRecordID, sessionUser.ID); err != nil {
				h.logger.Warn().Err(err).
					Str("session_id", sessionUser.SessionRecordID.String()).
					Msg("failed to revoke session record")
			}
		}
	}

	if err := h.sessions.ClearUser(c.Request, c.Writer); err != nil {
		h.logger.Error().Err(err).Msg("failed to clear session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// generateSessionTokenHash creates a hash from a session ID for storage.
func generateSessionTokenHash(sessionID uuid.UUID) string {
	hash := sha256.Sum256([]byte(sessionID.String()))
	return hex.EncodeToString(hash[:])
}

// MeResponse is the response for the /auth/me endpoint.
type MeResponse struct {
	ID                uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email             string     `json:"email" example:"user@example.com"`
	Name              string     `json:"name" example:"John Doe"`
	CurrentOrgID      uuid.UUID  `json:"current_org_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	CurrentOrgRole    string     `json:"current_org_role,omitempty" example:"admin"`
	SSOGroups         []string   `json:"sso_groups,omitempty"`
	SSOGroupsSyncedAt *time.Time `json:"sso_groups_synced_at,omitempty"`
}

// Me returns the current authenticated user.
//
//	@Summary		Get current user
//	@Description	Returns information about the currently authenticated user including their current organization
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	MeResponse
//	@Failure		401	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	sessionUser, err := h.sessions.GetUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	response := MeResponse{
		ID:             sessionUser.ID,
		Email:          sessionUser.Email,
		Name:           sessionUser.Name,
		CurrentOrgID:   sessionUser.CurrentOrgID,
		CurrentOrgRole: sessionUser.CurrentOrgRole,
	}

	// Fetch user's SSO groups if available
	ssoGroups, err := h.userStore.GetUserSSOGroups(c.Request.Context(), sessionUser.ID)
	if err == nil && ssoGroups != nil {
		response.SSOGroups = ssoGroups.OIDCGroups
		response.SSOGroupsSyncedAt = &ssoGroups.SyncedAt
	}

	c.JSON(http.StatusOK, response)
}

// PasswordLoginResponse is the response for password-based login.
type PasswordLoginResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	CurrentOrgID       uuid.UUID  `json:"current_org_id,omitempty"`
	CurrentOrgRole     string     `json:"current_org_role,omitempty"`
	PasswordExpired    bool       `json:"password_expired,omitempty"`
	MustChangePassword bool       `json:"must_change_password,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// PasswordLogin handles password-based authentication.
//
//	@Summary		Password login
//	@Description	Authenticates a user with email and password. Creates a session on successful authentication.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.PasswordLoginRequest	true	"Login credentials"
//	@Success		200		{object}	PasswordLoginResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/auth/login/password [post]
func (h *AuthHandler) PasswordLogin(c *gin.Context) {
	var req models.PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	user, err := h.userStore.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		h.logger.Debug().Str("email", req.Email).Msg("user not found for password login")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Get password info
	passwordInfo, err := h.userStore.GetUserPasswordInfo(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get password info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	// Check if user has password authentication
	if passwordInfo.PasswordHash == nil {
		h.logger.Debug().Str("user_id", user.ID.String()).Msg("user does not have password auth configured")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, *passwordInfo.PasswordHash); err != nil {
		h.logger.Debug().Str("user_id", user.ID.String()).Msg("password verification failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Get user's memberships
	memberships, err := h.userStore.GetMembershipsByUserID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user memberships")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	var currentOrgID uuid.UUID
	var currentOrgRole string
	if len(memberships) > 0 {
		currentOrgID = memberships[0].OrgID
		currentOrgRole = string(memberships[0].Role)
	}

	// Store user in session
	sessionUser := &auth.SessionUser{
		ID:              user.ID,
		OIDCSubject:     user.OIDCSubject, // Will be empty for password-only users
		Email:           user.Email,
		Name:            user.Name,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    currentOrgID,
		CurrentOrgRole:  currentOrgRole,
	}

	if err := h.sessions.SetUser(c.Request, c.Writer, sessionUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to save user to session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(currentOrgID, models.AuditActionLogin, "user", models.AuditResultSuccess).
		WithUser(user.ID).
		WithDetails("Password-based login")
	if err := h.userStore.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log for login")
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("user authenticated successfully via password")

	// Check password expiration
	var passwordExpired bool
	if passwordInfo.PasswordExpiresAt != nil && time.Now().After(*passwordInfo.PasswordExpiresAt) {
		passwordExpired = true
	}

	c.JSON(http.StatusOK, PasswordLoginResponse{
		ID:                 user.ID,
		Email:              user.Email,
		Name:               user.Name,
		CurrentOrgID:       currentOrgID,
		CurrentOrgRole:     currentOrgRole,
		PasswordExpired:    passwordExpired,
		MustChangePassword: passwordInfo.MustChangePassword,
		ExpiresAt:          passwordInfo.PasswordExpiresAt,
	})
}
