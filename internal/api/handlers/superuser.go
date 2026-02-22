package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SuperuserStore defines the interface for superuser-related persistence operations.
type SuperuserStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	SetUserSuperuser(ctx context.Context, userID uuid.UUID, isSuperuser bool) error
	GetSuperusers(ctx context.Context) ([]*models.User, error)
	CreateSuperuserAuditLog(ctx context.Context, log *models.SuperuserAuditLog) error
	GetSuperuserAuditLogs(ctx context.Context, limit, offset int) ([]*models.SuperuserAuditLogWithUser, int, error)
	GetSystemSetting(ctx context.Context, key string) (*models.SystemSetting, error)
	GetSystemSettings(ctx context.Context) ([]*models.SystemSetting, error)
	UpdateSystemSetting(ctx context.Context, key string, value interface{}, updatedBy uuid.UUID) error
	GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
}

// SuperuserHandler handles superuser-related HTTP endpoints.
type SuperuserHandler struct {
	store     SuperuserStore
	sessions  *auth.SessionStore
	superuser *auth.Superuser
	logger    zerolog.Logger
	store      SuperuserStore
	sessions   *auth.SessionStore
	superuser  *auth.Superuser
	logger     zerolog.Logger
}

// NewSuperuserHandler creates a new SuperuserHandler.
func NewSuperuserHandler(store SuperuserStore, sessions *auth.SessionStore, logger zerolog.Logger) *SuperuserHandler {
	return &SuperuserHandler{
		store:     store,
		sessions:  sessions,
		superuser: auth.NewSuperuser(store),
		logger:    logger.With().Str("component", "superuser_handler").Logger(),
	}
}

// RegisterRoutes registers superuser routes on the given router group.
// These routes require superuser privileges.
func (h *SuperuserHandler) RegisterRoutes(r *gin.RouterGroup) {
	su := r.Group("/superuser")
	su.Use(middleware.SuperuserMiddleware(h.sessions, h.logger))
	{
		// Organization management
		su.GET("/organizations", h.ListAllOrganizations)
		su.GET("/organizations/:id", h.GetOrganization)

		// User management
		su.GET("/users", h.ListAllUsers)
		su.GET("/superusers", h.ListSuperusers)
		su.POST("/users/:id/grant-superuser", h.GrantSuperuser)
		su.POST("/users/:id/revoke-superuser", h.RevokeSuperuser)

		// Impersonation
		su.POST("/impersonate/:id", h.StartImpersonation)
		su.POST("/end-impersonation", h.EndImpersonation)

		// System settings
		su.GET("/settings", h.GetSystemSettings)
		su.GET("/settings/:key", h.GetSystemSetting)
		su.PUT("/settings/:key", h.UpdateSystemSetting)

		// Audit logs
		su.GET("/audit-logs", h.GetAuditLogs)
	}
}

// ListAllOrganizations returns all organizations in the system.
// GET /api/v1/superuser/organizations
func (h *SuperuserHandler) ListAllOrganizations(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	orgs, err := h.store.GetAllOrganizations(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list all organizations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list organizations"})
		return
	}

	// Log the action
	h.logAction(c, user.ID, models.SuperuserActionViewOrgs, "organizations", nil, nil)

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// GetOrganization returns details of a specific organization.
// GET /api/v1/superuser/organizations/:id
func (h *SuperuserHandler) GetOrganization(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	org, err := h.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	// Log the action
	h.logAction(c, user.ID, models.SuperuserActionViewOrg, "organization", &orgID, &orgID)

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// ListAllUsers returns all users across all organizations.
// GET /api/v1/superuser/users
func (h *SuperuserHandler) ListAllUsers(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	users, err := h.store.GetAllUsers(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list all users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	// Log the action
	h.logAction(c, user.ID, models.SuperuserActionViewUsers, "users", nil, nil)

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// ListSuperusers returns all users with superuser privileges.
// GET /api/v1/superuser/superusers
func (h *SuperuserHandler) ListSuperusers(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	superusers, err := h.store.GetSuperusers(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list superusers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list superusers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"superusers": superusers})
}

// GrantSuperuserRequest is the request body for granting superuser.
type GrantSuperuserRequest struct {
	Reason string `json:"reason"`
}

// GrantSuperuser grants superuser privileges to a user.
// POST /api/v1/superuser/users/:id/grant-superuser
func (h *SuperuserHandler) GrantSuperuser(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req GrantSuperuserRequest
	_ = c.ShouldBindJSON(&req) // Reason is optional

	// Verify target user exists
	targetUser, err := h.store.GetUserByID(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if targetUser.IsSuperuser {
		c.JSON(http.StatusConflict, gin.H{"error": "user is already a superuser"})
		return
	}

	if err := h.store.SetUserSuperuser(c.Request.Context(), targetID, true); err != nil {
		h.logger.Error().Err(err).Str("target_id", targetID.String()).Msg("failed to grant superuser")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to grant superuser privileges"})
		return
	}

	// Log the action with reason
	details := map[string]string{"reason": req.Reason, "target_email": targetUser.Email}
	h.logActionWithDetails(c, user.ID, models.SuperuserActionGrantSuperuser, "user", &targetID, nil, details)

	h.logger.Info().
		Str("granter_id", user.ID.String()).
		Str("target_id", targetID.String()).
		Str("reason", req.Reason).
		Msg("superuser privileges granted")

	c.JSON(http.StatusOK, gin.H{"message": "superuser privileges granted"})
}

// RevokeSuperuser revokes superuser privileges from a user.
// POST /api/v1/superuser/users/:id/revoke-superuser
func (h *SuperuserHandler) RevokeSuperuser(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if user.ID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot revoke own superuser privileges"})
		return
	}

	// Check that at least one superuser will remain
	superusers, err := h.store.GetSuperusers(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to check superuser count")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke superuser privileges"})
		return
	}
	if len(superusers) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the last superuser"})
		return
	}

	// Verify target is a superuser
	targetUser, err := h.store.GetUserByID(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if !targetUser.IsSuperuser {
		c.JSON(http.StatusConflict, gin.H{"error": "user is not a superuser"})
		return
	}

	if err := h.store.SetUserSuperuser(c.Request.Context(), targetID, false); err != nil {
		h.logger.Error().Err(err).Str("target_id", targetID.String()).Msg("failed to revoke superuser")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke superuser privileges"})
		return
	}

	// Log the action
	details := map[string]string{"target_email": targetUser.Email}
	h.logActionWithDetails(c, user.ID, models.SuperuserActionRevokeSuperuser, "user", &targetID, nil, details)

	h.logger.Info().
		Str("revoker_id", user.ID.String()).
		Str("target_id", targetID.String()).
		Msg("superuser privileges revoked")

	c.JSON(http.StatusOK, gin.H{"message": "superuser privileges revoked"})
}

// StartImpersonation starts impersonating another user.
// POST /api/v1/superuser/impersonate/:id
func (h *SuperuserHandler) StartImpersonation(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	// Prevent nested impersonation
	if h.sessions.IsImpersonating(c.Request) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already impersonating a user, end current impersonation first"})
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if user.ID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot impersonate yourself"})
		return
	}

	// Get target user
	targetUser, err := h.store.GetUserByID(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get target user's first org membership for initial session
	memberships, err := h.store.GetMembershipsByUserID(c.Request.Context(), targetID)
	if err != nil || len(memberships) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user has no organization memberships"})
		return
	}

	// Create target session user
	targetSession := &auth.SessionUser{
		ID:             targetUser.ID,
		OIDCSubject:    targetUser.OIDCSubject,
		Email:          targetUser.Email,
		Name:           targetUser.Name,
		CurrentOrgID:   memberships[0].OrgID,
		CurrentOrgRole: string(memberships[0].Role),
	}

	// Start impersonation (impersonation logging is handled separately via logActionWithImpersonation)
	if err := h.sessions.StartImpersonation(c.Request, c.Writer, user, targetSession, uuid.Nil); err != nil {
	// Start impersonation
	if err := h.sessions.StartImpersonation(c.Request, c.Writer, user, targetSession); err != nil {
		h.logger.Error().Err(err).Msg("failed to start impersonation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start impersonation"})
		return
	}

	// Log the action
	h.logActionWithImpersonation(c, user.ID, models.SuperuserActionImpersonate, "user", &targetID, nil, targetID)

	h.logger.Info().
		Str("superuser_id", user.ID.String()).
		Str("target_id", targetID.String()).
		Str("target_email", targetUser.Email).
		Msg("impersonation started")

	c.JSON(http.StatusOK, gin.H{
		"message": "impersonation started",
		"impersonating": gin.H{
			"id":    targetUser.ID,
			"email": targetUser.Email,
			"name":  targetUser.Name,
		},
	})
}

// EndImpersonation ends the current impersonation session.
// POST /api/v1/superuser/end-impersonation
func (h *SuperuserHandler) EndImpersonation(c *gin.Context) {
	user := middleware.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	if !h.sessions.IsImpersonating(c.Request) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not currently impersonating"})
		return
	}

	// Get original superuser
	originalUserID := h.sessions.GetOriginalUserID(c.Request)
	if originalUserID == uuid.Nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore original session"})
		return
	}

	originalUser, err := h.store.GetUserByID(c.Request.Context(), originalUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore original session"})
		return
	}

	// Get original user's membership for session
	memberships, err := h.store.GetMembershipsByUserID(c.Request.Context(), originalUserID)
	if err != nil || len(memberships) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore original session"})
		return
	}

	originalSession := &auth.SessionUser{
		ID:             originalUser.ID,
		OIDCSubject:    originalUser.OIDCSubject,
		Email:          originalUser.Email,
		Name:           originalUser.Name,
		CurrentOrgID:   memberships[0].OrgID,
		CurrentOrgRole: string(memberships[0].Role),
		IsSuperuser:    originalUser.IsSuperuser,
	}

	// End impersonation
	if err := h.sessions.EndImpersonation(c.Request, c.Writer, originalSession); err != nil {
		h.logger.Error().Err(err).Msg("failed to end impersonation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end impersonation"})
		return
	}

	// Log the action
	impersonatedID := user.ImpersonatingID
	h.logActionWithImpersonation(c, originalUserID, models.SuperuserActionEndImpersonate, "user", &impersonatedID, nil, impersonatedID)

	h.logger.Info().
		Str("superuser_id", originalUserID.String()).
		Str("impersonated_id", impersonatedID.String()).
		Msg("impersonation ended")

	c.JSON(http.StatusOK, gin.H{"message": "impersonation ended"})
}

// GetSystemSettings returns all system settings.
// GET /api/v1/superuser/settings
func (h *SuperuserHandler) GetSystemSettings(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	settings, err := h.store.GetSystemSettings(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get system settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get system settings"})
		return
	}

	// Log the action
	h.logAction(c, user.ID, models.SuperuserActionViewSettings, "system_settings", nil, nil)

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// GetSystemSetting returns a specific system setting.
// GET /api/v1/superuser/settings/:key
func (h *SuperuserHandler) GetSystemSetting(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	key := c.Param("key")
	setting, err := h.store.GetSystemSetting(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"setting": setting})
}

// UpdateSystemSettingRequest is the request body for updating a system setting.
type UpdateSystemSettingRequest struct {
	Value interface{} `json:"value" binding:"required"`
}

// UpdateSystemSetting updates a system setting.
// PUT /api/v1/superuser/settings/:key
func (h *SuperuserHandler) UpdateSystemSetting(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	key := c.Param("key")

	var req UpdateSystemSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.store.UpdateSystemSetting(c.Request.Context(), key, req.Value, user.ID); err != nil {
		h.logger.Error().Err(err).Str("key", key).Msg("failed to update system setting")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	// Log the action
	details := map[string]interface{}{"key": key, "new_value": req.Value}
	h.logActionWithDetails(c, user.ID, models.SuperuserActionUpdateSettings, "system_setting", nil, nil, details)

	h.logger.Info().
		Str("superuser_id", user.ID.String()).
		Str("key", key).
		Msg("system setting updated")

	c.JSON(http.StatusOK, gin.H{"message": "setting updated"})
}

// GetAuditLogs returns superuser audit logs.
// GET /api/v1/superuser/audit-logs
func (h *SuperuserHandler) GetAuditLogs(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if _, err := uuid.Parse(l); err == nil {
			// Invalid limit, use default
		}
	}

	logs, total, err := h.store.GetSuperuserAuditLogs(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get superuser audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	// Log the action
	h.logAction(c, user.ID, models.SuperuserActionViewAuditLogs, "superuser_audit_logs", nil, nil)

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": logs,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// logAction creates a superuser audit log entry.
func (h *SuperuserHandler) logAction(c *gin.Context, superuserID uuid.UUID, action models.SuperuserAction, targetType string, targetID, targetOrgID *uuid.UUID) {
	log := models.NewSuperuserAuditLog(superuserID, action, targetType).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent())

	if targetID != nil {
		log.WithTargetID(*targetID)
	}
	if targetOrgID != nil {
		log.WithTargetOrgID(*targetOrgID)
	}

	go func() {
		if err := h.store.CreateSuperuserAuditLog(context.Background(), log); err != nil {
			h.logger.Error().Err(err).Str("action", string(action)).Msg("failed to create superuser audit log")
		}
	}()
}

// logActionWithDetails creates a superuser audit log entry with additional details.
func (h *SuperuserHandler) logActionWithDetails(c *gin.Context, superuserID uuid.UUID, action models.SuperuserAction, targetType string, targetID, targetOrgID *uuid.UUID, details interface{}) {
	log := models.NewSuperuserAuditLog(superuserID, action, targetType).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent()).
		WithDetails(details)

	if targetID != nil {
		log.WithTargetID(*targetID)
	}
	if targetOrgID != nil {
		log.WithTargetOrgID(*targetOrgID)
	}

	go func() {
		if err := h.store.CreateSuperuserAuditLog(context.Background(), log); err != nil {
			h.logger.Error().Err(err).Str("action", string(action)).Msg("failed to create superuser audit log")
		}
	}()
}

// logActionWithImpersonation creates a superuser audit log entry with impersonation info.
func (h *SuperuserHandler) logActionWithImpersonation(c *gin.Context, superuserID uuid.UUID, action models.SuperuserAction, targetType string, targetID, targetOrgID *uuid.UUID, impersonatedUserID uuid.UUID) {
	log := models.NewSuperuserAuditLog(superuserID, action, targetType).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent()).
		WithImpersonatedUser(impersonatedUserID)

	if targetID != nil {
		log.WithTargetID(*targetID)
	}
	if targetOrgID != nil {
		log.WithTargetOrgID(*targetOrgID)
	}

	go func() {
		if err := h.store.CreateSuperuserAuditLog(context.Background(), log); err != nil {
			h.logger.Error().Err(err).Str("action", string(action)).Msg("failed to create superuser audit log")
		}
	}()
}
