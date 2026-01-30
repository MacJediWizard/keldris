package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UserManagementStore defines the interface for user management persistence operations.
type UserManagementStore interface {
	GetUsersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UserWithMembership, error)
	GetUserByIDWithMembership(ctx context.Context, userID, orgID uuid.UUID) (*models.UserWithMembership, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, status models.UserStatus) error
	UpdateUserName(ctx context.Context, userID uuid.UUID, name string) error
	SetUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string, mustChange bool) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	IsSuperuser(ctx context.Context, userID uuid.UUID) (bool, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
	UpdateMembership(ctx context.Context, m *models.OrgMembership) error
	CreateInvitation(ctx context.Context, inv *models.OrgInvitation) error
	// Activity logs
	GetUserActivityLogs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserActivityLogWithUser, error)
	GetOrgActivityLogs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.UserActivityLogWithUser, error)
	CreateUserActivityLog(ctx context.Context, log *models.UserActivityLog) error
	// Impersonation logs
	CreateImpersonationLog(ctx context.Context, log *models.UserImpersonationLog) error
	EndImpersonationLog(ctx context.Context, logID uuid.UUID) error
	GetImpersonationLogsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.UserImpersonationLogWithUsers, error)
	GetActiveImpersonation(ctx context.Context, adminUserID uuid.UUID) (*models.UserImpersonationLog, error)
	// Password validation
	GetUserPasswordInfo(ctx context.Context, userID uuid.UUID) (*models.UserPasswordInfo, error)
	// Session management
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) (int64, error)
	// Audit logging
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// UsersHandler handles user management HTTP endpoints.
type UsersHandler struct {
	store    UserManagementStore
	sessions *auth.SessionStore
	rbac     *auth.RBAC
	logger   zerolog.Logger
}

// NewUsersHandler creates a new UsersHandler.
func NewUsersHandler(store UserManagementStore, sessions *auth.SessionStore, rbac *auth.RBAC, logger zerolog.Logger) *UsersHandler {
	return &UsersHandler{
		store:    store,
		sessions: sessions,
		rbac:     rbac,
		logger:   logger.With().Str("component", "users_handler").Logger(),
	}
}

// RegisterRoutes registers user management routes on the given router group.
func (h *UsersHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.List)
		users.POST("/invite", h.Invite)
		users.GET("/:id", h.Get)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", h.Delete)
		users.POST("/:id/reset-password", h.ResetPassword)
		users.POST("/:id/disable", h.Disable)
		users.POST("/:id/enable", h.Enable)
		users.GET("/:id/activity", h.GetUserActivity)
	}

	// Activity logs for the org
	r.GET("/activity-logs", h.GetOrgActivityLogs)

	// Impersonation endpoints
	impersonate := r.Group("/impersonate")
	{
		impersonate.POST("/:id", h.StartImpersonation)
		impersonate.POST("/end", h.EndImpersonation)
		impersonate.GET("/logs", h.GetImpersonationLogs)
	}
}

// List returns all users in the current organization.
//
//	@Summary		List users
//	@Description	Returns all users in the current organization
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users [get]
func (h *UsersHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	users, err := h.store.GetUsersByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Invite sends an invitation to a new user.
//
//	@Summary		Invite user
//	@Description	Sends an invitation to a new user to join the organization
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.InviteUserRequest	true	"Invitation details"
//	@Success		201		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/invite [post]
func (h *UsersHandler) Invite(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserInvite); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user can assign this role
	canAssign, err := h.rbac.CanAssignRole(c.Request.Context(), user.ID, user.CurrentOrgID, req.Role)
	if err != nil || !canAssign {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot invite with this role"})
		return
	}

	// Generate invite token
	token, err := generateInviteToken()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate invite token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}

	// Create invitation
	inv := models.NewOrgInvitation(
		user.CurrentOrgID,
		req.Email,
		req.Role,
		token,
		user.ID,
		models.DefaultInvitationExpiry(),
	)

	if err := h.store.CreateInvitation(c.Request.Context(), inv); err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("failed to create invitation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}

	// Log activity
	h.logActivity(c, user, "invite_user", "user", nil, map[string]string{"email": req.Email, "role": string(req.Role)})

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("email", req.Email).
		Str("role", string(req.Role)).
		Str("inviter_id", user.ID.String()).
		Msg("user invitation created")

	c.JSON(http.StatusCreated, gin.H{
		"message": "invitation sent",
		"token":   token,
	})
}

// Get returns a specific user.
//
//	@Summary		Get user
//	@Description	Returns a specific user by ID
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	models.UserWithMembership
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id} [get]
func (h *UsersHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	targetUser, err := h.store.GetUserByIDWithMembership(c.Request.Context(), targetUserID, user.CurrentOrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, targetUser)
}

// Update updates a user's details.
//
//	@Summary		Update user
//	@Description	Updates a user's name, role, or status
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"User ID"
//	@Param			request	body		models.UpdateUserRequest	true	"Update details"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id} [put]
func (h *UsersHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if actor can manage this user
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, user.CurrentOrgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot manage this user"})
		return
	}

	// Update name if provided
	if req.Name != "" {
		if err := h.store.UpdateUserName(c.Request.Context(), targetUserID, req.Name); err != nil {
			h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to update user name")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}

	// Update role if provided
	if req.Role != nil {
		// Check if actor can assign this role
		canAssign, err := h.rbac.CanAssignRole(c.Request.Context(), user.ID, user.CurrentOrgID, *req.Role)
		if err != nil || !canAssign {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot assign this role"})
			return
		}

		membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), targetUserID, user.CurrentOrgID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found in organization"})
			return
		}
		membership.Role = *req.Role
		if err := h.store.UpdateMembership(c.Request.Context(), membership); err != nil {
			h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to update user role")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}

	// Update status if provided
	if req.Status != nil {
		if err := h.store.UpdateUserStatus(c.Request.Context(), targetUserID, *req.Status); err != nil {
			h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to update user status")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}

	h.logActivity(c, user, "update_user", "user", &targetUserID, nil)

	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

// Delete removes a user from the organization.
//
//	@Summary		Delete user
//	@Description	Removes a user from the organization
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id} [delete]
func (h *UsersHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserDelete); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Prevent self-deletion
	if targetUserID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	// Check if actor can manage this user
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, user.CurrentOrgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete this user"})
		return
	}

	if err := h.store.DeleteUser(c.Request.Context(), targetUserID); err != nil {
		h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	h.logActivity(c, user, "delete_user", "user", &targetUserID, nil)

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("target_user_id", targetUserID.String()).
		Str("actor_id", user.ID.String()).
		Msg("user deleted")

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// ResetPassword resets a user's password (for non-OIDC users).
//
//	@Summary		Reset password
//	@Description	Resets a user's password
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"User ID"
//	@Param			request	body		models.ResetPasswordRequest	true	"Password reset details"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id}/reset-password [post]
func (h *UsersHandler) ResetPassword(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserResetPassword); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if actor can manage this user
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, user.CurrentOrgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot manage this user"})
		return
	}

	// Check if target user is an OIDC-only user
	targetUser, err := h.store.GetUserByID(c.Request.Context(), targetUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if targetUser.OIDCSubject != "" {
		// User has OIDC - check if they also have password auth
		passwordInfo, err := h.store.GetUserPasswordInfo(c.Request.Context(), targetUserID)
		if err != nil || passwordInfo.PasswordHash == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user uses OIDC authentication, cannot reset password"})
			return
		}
	}

	// Hash the new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	if err := h.store.SetUserPassword(c.Request.Context(), targetUserID, passwordHash, req.RequireChangeOnUse); err != nil {
		h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	// Revoke all sessions for the user
	_, _ = h.store.RevokeAllUserSessions(c.Request.Context(), targetUserID, nil)

	h.logActivity(c, user, "reset_password", "user", &targetUserID, nil)

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("target_user_id", targetUserID.String()).
		Str("actor_id", user.ID.String()).
		Msg("user password reset")

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// Disable disables a user account.
//
//	@Summary		Disable user
//	@Description	Disables a user account
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id}/disable [post]
func (h *UsersHandler) Disable(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserDisable); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Prevent self-disable
	if targetUserID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable yourself"})
		return
	}

	// Check if actor can manage this user
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, user.CurrentOrgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot disable this user"})
		return
	}

	if err := h.store.UpdateUserStatus(c.Request.Context(), targetUserID, models.UserStatusDisabled); err != nil {
		h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to disable user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
		return
	}

	// Revoke all sessions for the user
	_, _ = h.store.RevokeAllUserSessions(c.Request.Context(), targetUserID, nil)

	h.logActivity(c, user, "disable_user", "user", &targetUserID, nil)

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("target_user_id", targetUserID.String()).
		Str("actor_id", user.ID.String()).
		Msg("user disabled")

	c.JSON(http.StatusOK, gin.H{"message": "user disabled"})
}

// Enable enables a user account.
//
//	@Summary		Enable user
//	@Description	Enables a disabled user account
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id}/enable [post]
func (h *UsersHandler) Enable(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserDisable); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Check if actor can manage this user
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, user.CurrentOrgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot enable this user"})
		return
	}

	if err := h.store.UpdateUserStatus(c.Request.Context(), targetUserID, models.UserStatusActive); err != nil {
		h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to enable user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable user"})
		return
	}

	h.logActivity(c, user, "enable_user", "user", &targetUserID, nil)

	c.JSON(http.StatusOK, gin.H{"message": "user enabled"})
}

// GetUserActivity returns activity logs for a specific user.
//
//	@Summary		Get user activity
//	@Description	Returns activity logs for a specific user
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"User ID"
//	@Param			limit	query		int		false	"Limit"
//	@Param			offset	query		int		false	"Offset"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/{id}/activity [get]
func (h *UsersHandler) GetUserActivity(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserActivityView); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.store.GetUserActivityLogs(c.Request.Context(), targetUserID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", targetUserID.String()).Msg("failed to get user activity logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activity_logs": logs})
}

// GetOrgActivityLogs returns activity logs for the organization.
//
//	@Summary		Get organization activity logs
//	@Description	Returns activity logs for the organization
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"
//	@Param			offset	query		int	false	"Offset"
//	@Success		200		{object}	map[string]any
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity-logs [get]
func (h *UsersHandler) GetOrgActivityLogs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserActivityView); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.store.GetOrgActivityLogs(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get org activity logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activity_logs": logs})
}

// StartImpersonation starts impersonating a user (superuser only).
//
//	@Summary		Start impersonation
//	@Description	Starts impersonating a user (superuser only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"User ID to impersonate"
//	@Param			request	body		models.ImpersonateUserRequest	true	"Impersonation reason"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/impersonate/{id} [post]
func (h *UsersHandler) StartImpersonation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Check if already impersonating
	if user.Impersonating {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already impersonating a user"})
		return
	}

	// Check if user is a superuser
	isSuperuser, err := h.store.IsSuperuser(c.Request.Context(), user.ID)
	if err != nil || !isSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Prevent impersonating yourself
	if targetUserID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot impersonate yourself"})
		return
	}

	var req models.ImpersonateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.store.GetUserByIDWithMembership(c.Request.Context(), targetUserID, user.CurrentOrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Create impersonation log
	impersonationLog := models.NewUserImpersonationLog(user.ID, targetUserID, user.CurrentOrgID, req.Reason)
	impersonationLog.IPAddress = c.ClientIP()
	impersonationLog.UserAgent = c.Request.UserAgent()

	if err := h.store.CreateImpersonationLog(c.Request.Context(), impersonationLog); err != nil {
		h.logger.Error().Err(err).Msg("failed to create impersonation log")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start impersonation"})
		return
	}

	// Start impersonation in session
	originalUser := &auth.SessionUser{
		ID:             user.ID,
		OIDCSubject:    user.OIDCSubject,
		Email:          user.Email,
		Name:           user.Name,
		CurrentOrgID:   user.CurrentOrgID,
		CurrentOrgRole: user.CurrentOrgRole,
		IsSuperuser:    true,
	}
	targetSessionUser := &auth.SessionUser{
		ID:             targetUser.ID,
		OIDCSubject:    targetUser.OIDCSubject,
		Email:          targetUser.Email,
		Name:           targetUser.Name,
		CurrentOrgID:   user.CurrentOrgID,
		CurrentOrgRole: string(targetUser.OrgRole),
	}

	if err := h.sessions.StartImpersonation(c.Request, c.Writer, originalUser, targetSessionUser, impersonationLog.ID); err != nil {
		h.logger.Error().Err(err).Msg("failed to start impersonation session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start impersonation"})
		return
	}

	h.logger.Info().
		Str("admin_id", user.ID.String()).
		Str("target_id", targetUserID.String()).
		Str("reason", req.Reason).
		Msg("impersonation started")

	c.JSON(http.StatusOK, gin.H{
		"message":         "impersonation started",
		"impersonating":   targetUser.Email,
		"impersonated_id": targetUserID.String(),
	})
}

// EndImpersonation ends the current impersonation session.
//
//	@Summary		End impersonation
//	@Description	Ends the current impersonation session
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/impersonate/end [post]
func (h *UsersHandler) EndImpersonation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !user.Impersonating {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not currently impersonating"})
		return
	}

	// Get the original user
	originalUser, err := h.store.GetUserByIDWithMembership(c.Request.Context(), user.OriginalUserID, user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("original_user_id", user.OriginalUserID.String()).Msg("failed to get original user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end impersonation"})
		return
	}

	// End impersonation log
	if err := h.store.EndImpersonationLog(c.Request.Context(), user.ImpersonationLogID); err != nil {
		h.logger.Warn().Err(err).Msg("failed to end impersonation log")
	}

	// Restore original user in session
	originalSessionUser := &auth.SessionUser{
		ID:             originalUser.ID,
		OIDCSubject:    originalUser.OIDCSubject,
		Email:          originalUser.Email,
		Name:           originalUser.Name,
		CurrentOrgID:   user.CurrentOrgID,
		CurrentOrgRole: string(originalUser.OrgRole),
	}

	if err := h.sessions.EndImpersonation(c.Request, c.Writer, originalSessionUser); err != nil {
		h.logger.Error().Err(err).Msg("failed to end impersonation session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end impersonation"})
		return
	}

	h.logger.Info().
		Str("admin_id", user.OriginalUserID.String()).
		Str("target_id", user.ID.String()).
		Msg("impersonation ended")

	c.JSON(http.StatusOK, gin.H{"message": "impersonation ended"})
}

// GetImpersonationLogs returns impersonation logs for the organization.
//
//	@Summary		Get impersonation logs
//	@Description	Returns impersonation logs for the organization
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"
//	@Param			offset	query		int	false	"Offset"
//	@Success		200		{object}	map[string]any
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/impersonate/logs [get]
func (h *UsersHandler) GetImpersonationLogs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only superusers or owners can view impersonation logs
	isSuperuser, _ := h.store.IsSuperuser(c.Request.Context(), user.ID)
	if !isSuperuser {
		if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermUserActivityView); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.store.GetImpersonationLogsByOrg(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get impersonation logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get impersonation logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"impersonation_logs": logs})
}

// logActivity logs user activity.
func (h *UsersHandler) logActivity(c *gin.Context, user *auth.SessionUser, action, resourceType string, resourceID *uuid.UUID, details map[string]string) {
	log := models.NewUserActivityLog(user.ID, user.CurrentOrgID, action)
	if resourceID != nil {
		log.WithResource(resourceType, *resourceID)
	}
	log.WithIPAddress(c.ClientIP())
	log.WithUserAgent(c.Request.UserAgent())

	if details != nil {
		if detailsJSON, err := encodeDetails(details); err == nil {
			log.WithDetails(detailsJSON)
		}
	}

	if err := h.store.CreateUserActivityLog(c.Request.Context(), log); err != nil {
		h.logger.Warn().Err(err).Str("action", action).Msg("failed to log user activity")
	}
}

// encodeDetails encodes details to JSON.
func encodeDetails(details map[string]string) ([]byte, error) {
	if details == nil {
		return nil, nil
	}
	return json.Marshal(details)
}
