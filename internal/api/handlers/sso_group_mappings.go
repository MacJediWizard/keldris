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

// SSOGroupMappingStore defines the interface for SSO group mapping persistence operations.
type SSOGroupMappingStore interface {
	GetSSOGroupMappingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error)
	GetSSOGroupMappingByID(ctx context.Context, id uuid.UUID) (*models.SSOGroupMapping, error)
	CreateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error
	UpdateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error
	DeleteSSOGroupMapping(ctx context.Context, id uuid.UUID) error
	GetUserSSOGroups(ctx context.Context, userID uuid.UUID) (*models.UserSSOGroups, error)
	GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error)
	UpdateOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID, defaultRole *string, autoCreateOrgs bool) error
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// SSOGroupMappingsHandler handles SSO group mapping HTTP endpoints.
type SSOGroupMappingsHandler struct {
	store  SSOGroupMappingStore
	rbac   *auth.RBAC
	logger zerolog.Logger
}

// NewSSOGroupMappingsHandler creates a new SSOGroupMappingsHandler.
func NewSSOGroupMappingsHandler(store SSOGroupMappingStore, rbac *auth.RBAC, logger zerolog.Logger) *SSOGroupMappingsHandler {
	return &SSOGroupMappingsHandler{
		store:  store,
		rbac:   rbac,
		logger: logger.With().Str("component", "sso_group_mappings_handler").Logger(),
	}
}

// RegisterRoutes registers SSO group mapping routes on the given router group.
func (h *SSOGroupMappingsHandler) RegisterRoutes(r *gin.RouterGroup) {
	mappings := r.Group("/organizations/:id/sso-group-mappings")
	{
		mappings.GET("", h.List)
		mappings.POST("", h.Create)
		mappings.GET("/:mapping_id", h.Get)
		mappings.PUT("/:mapping_id", h.Update)
		mappings.DELETE("/:mapping_id", h.Delete)
	}

	// SSO settings for org
	r.GET("/organizations/:id/sso-settings", h.GetSSOSettings)
	r.PUT("/organizations/:id/sso-settings", h.UpdateSSOSettings)

	// User SSO groups endpoint
	r.GET("/users/:id/sso-groups", h.GetUserSSOGroups)
	mappings := r.Group("/organizations/:org_id/sso-group-mappings")
	{
		mappings.GET("", h.List)
		mappings.POST("", h.Create)
		mappings.GET("/:id", h.Get)
		mappings.PUT("/:id", h.Update)
		mappings.DELETE("/:id", h.Delete)
	}

	// SSO settings for org
	r.GET("/organizations/:org_id/sso-settings", h.GetSSOSettings)
	r.PUT("/organizations/:org_id/sso-settings", h.UpdateSSOSettings)

	// User SSO groups endpoint
	r.GET("/users/:user_id/sso-groups", h.GetUserSSOGroups)
}

// SSOGroupMappingResponse wraps the group mapping response.
type SSOGroupMappingResponse struct {
	Mapping *models.SSOGroupMapping `json:"mapping"`
}

// SSOGroupMappingsResponse wraps the list of group mappings.
type SSOGroupMappingsResponse struct {
	Mappings []*models.SSOGroupMapping `json:"mappings"`
}

// SSOSettingsResponse wraps the SSO settings.
type SSOSettingsResponse struct {
	DefaultRole    *string `json:"default_role"`
	AutoCreateOrgs bool    `json:"auto_create_orgs"`
}

// UpdateSSOSettingsRequest is the request body for updating SSO settings.
type UpdateSSOSettingsRequest struct {
	DefaultRole    *string `json:"default_role"`
	AutoCreateOrgs *bool   `json:"auto_create_orgs"`
}

// List returns all SSO group mappings for an organization.
// GET /api/v1/organizations/:id/sso-group-mappings
// GET /api/v1/organizations/:org_id/sso-group-mappings
func (h *SSOGroupMappingsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission - only admins/owners can view SSO settings
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	mappings, err := h.store.GetSSOGroupMappingsByOrgID(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to list SSO group mappings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list SSO group mappings"})
		return
	}

	c.JSON(http.StatusOK, SSOGroupMappingsResponse{Mappings: mappings})
}

// Create creates a new SSO group mapping.
// POST /api/v1/organizations/:id/sso-group-mappings
// POST /api/v1/organizations/:org_id/sso-group-mappings
func (h *SSOGroupMappingsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission - only admins/owners can manage SSO settings
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.CreateSSOGroupMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate role
	if !models.IsValidOrgRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	mapping := models.NewSSOGroupMapping(orgID, req.OIDCGroupName, models.OrgRole(req.Role))
	mapping.AutoCreateOrg = req.AutoCreateOrg

	if err := h.store.CreateSSOGroupMapping(c.Request.Context(), mapping); err != nil {
		h.logger.Error().Err(err).
			Str("org_id", orgID.String()).
			Str("oidc_group", req.OIDCGroupName).
			Msg("failed to create SSO group mapping")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create SSO group mapping"})
		return
	}

	// Audit log
	h.logAuditEvent(c.Request.Context(), orgID, user.ID, models.AuditActionCreate, "sso_group_mapping", mapping.ID)

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("oidc_group", req.OIDCGroupName).
		Str("role", req.Role).
		Str("user_id", user.ID.String()).
		Msg("SSO group mapping created")

	c.JSON(http.StatusCreated, SSOGroupMappingResponse{Mapping: mapping})
}

// Get returns a specific SSO group mapping.
// GET /api/v1/organizations/:id/sso-group-mappings/:mapping_id
// GET /api/v1/organizations/:org_id/sso-group-mappings/:id
func (h *SSOGroupMappingsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	mappingID, err := uuid.Parse(c.Param("mapping_id"))
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mapping ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	mapping, err := h.store.GetSSOGroupMappingByID(c.Request.Context(), mappingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}

	// Verify mapping belongs to org
	if mapping.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}

	c.JSON(http.StatusOK, SSOGroupMappingResponse{Mapping: mapping})
}

// Update updates an SSO group mapping.
// PUT /api/v1/organizations/:id/sso-group-mappings/:mapping_id
// PUT /api/v1/organizations/:org_id/sso-group-mappings/:id
func (h *SSOGroupMappingsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	mappingID, err := uuid.Parse(c.Param("mapping_id"))
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mapping ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req models.UpdateSSOGroupMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	mapping, err := h.store.GetSSOGroupMappingByID(c.Request.Context(), mappingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}

	// Verify mapping belongs to org
	if mapping.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}

	// Update fields
	if req.Role != nil {
		if !models.IsValidOrgRole(*req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}
		mapping.Role = models.OrgRole(*req.Role)
	}
	if req.AutoCreateOrg != nil {
		mapping.AutoCreateOrg = *req.AutoCreateOrg
	}

	if err := h.store.UpdateSSOGroupMapping(c.Request.Context(), mapping); err != nil {
		h.logger.Error().Err(err).Str("mapping_id", mappingID.String()).Msg("failed to update SSO group mapping")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update SSO group mapping"})
		return
	}

	// Audit log
	h.logAuditEvent(c.Request.Context(), orgID, user.ID, models.AuditActionUpdate, "sso_group_mapping", mapping.ID)

	h.logger.Info().
		Str("mapping_id", mappingID.String()).
		Str("user_id", user.ID.String()).
		Msg("SSO group mapping updated")

	c.JSON(http.StatusOK, SSOGroupMappingResponse{Mapping: mapping})
}

// Delete deletes an SSO group mapping.
// DELETE /api/v1/organizations/:id/sso-group-mappings/:mapping_id
// DELETE /api/v1/organizations/:org_id/sso-group-mappings/:id
func (h *SSOGroupMappingsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	mappingID, err := uuid.Parse(c.Param("mapping_id"))
	mappingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mapping ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Verify mapping exists and belongs to org
	mapping, err := h.store.GetSSOGroupMappingByID(c.Request.Context(), mappingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}
	if mapping.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSO group mapping not found"})
		return
	}

	if err := h.store.DeleteSSOGroupMapping(c.Request.Context(), mappingID); err != nil {
		h.logger.Error().Err(err).Str("mapping_id", mappingID.String()).Msg("failed to delete SSO group mapping")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete SSO group mapping"})
		return
	}

	// Audit log
	h.logAuditEvent(c.Request.Context(), orgID, user.ID, models.AuditActionDelete, "sso_group_mapping", mappingID)

	h.logger.Info().
		Str("mapping_id", mappingID.String()).
		Str("user_id", user.ID.String()).
		Msg("SSO group mapping deleted")

	c.JSON(http.StatusOK, gin.H{"message": "SSO group mapping deleted"})
}

// GetSSOSettings returns the SSO settings for an organization.
// GET /api/v1/organizations/:id/sso-settings
// GET /api/v1/organizations/:org_id/sso-settings
func (h *SSOGroupMappingsHandler) GetSSOSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	defaultRole, autoCreateOrgs, err := h.store.GetOrganizationSSOSettings(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get SSO settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get SSO settings"})
		return
	}

	c.JSON(http.StatusOK, SSOSettingsResponse{
		DefaultRole:    defaultRole,
		AutoCreateOrgs: autoCreateOrgs,
	})
}

// UpdateSSOSettings updates the SSO settings for an organization.
// PUT /api/v1/organizations/:id/sso-settings
// PUT /api/v1/organizations/:org_id/sso-settings
func (h *SSOGroupMappingsHandler) UpdateSSOSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	orgID, err := uuid.Parse(c.Param("org_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req UpdateSSOSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate default role if provided
	if req.DefaultRole != nil && *req.DefaultRole != "" && !models.IsValidOrgRole(*req.DefaultRole) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid default role"})
		return
	}

	// Get current settings to merge
	currentDefaultRole, currentAutoCreate, err := h.store.GetOrganizationSSOSettings(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get current SSO settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update SSO settings"})
		return
	}

	newDefaultRole := currentDefaultRole
	if req.DefaultRole != nil {
		if *req.DefaultRole == "" {
			newDefaultRole = nil
		} else {
			newDefaultRole = req.DefaultRole
		}
	}

	newAutoCreate := currentAutoCreate
	if req.AutoCreateOrgs != nil {
		newAutoCreate = *req.AutoCreateOrgs
	}

	if err := h.store.UpdateOrganizationSSOSettings(c.Request.Context(), orgID, newDefaultRole, newAutoCreate); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to update SSO settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update SSO settings"})
		return
	}

	// Audit log
	h.logAuditEvent(c.Request.Context(), orgID, user.ID, models.AuditActionUpdate, "sso_settings", orgID)

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("user_id", user.ID.String()).
		Msg("SSO settings updated")

	c.JSON(http.StatusOK, SSOSettingsResponse{
		DefaultRole:    newDefaultRole,
		AutoCreateOrgs: newAutoCreate,
	})
}

// GetUserSSOGroups returns the SSO groups for a user.
// GET /api/v1/users/:id/sso-groups
// GET /api/v1/users/:user_id/sso-groups
func (h *SSOGroupMappingsHandler) GetUserSSOGroups(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	targetUserID, err := uuid.Parse(c.Param("id"))
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Users can only view their own SSO groups (or admins viewing members of their org)
	if user.ID != targetUserID {
		// Check if current user is admin of an org that target user belongs to
		// For simplicity, only allow users to view their own SSO groups
		c.JSON(http.StatusForbidden, gin.H{"error": "can only view your own SSO groups"})
		return
	}

	ssoGroups, err := h.store.GetUserSSOGroups(c.Request.Context(), targetUserID)
	if err != nil {
		// User may not have any SSO groups recorded yet
		c.JSON(http.StatusOK, gin.H{
			"user_id":     targetUserID,
			"oidc_groups": []string{},
			"synced_at":   nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     ssoGroups.UserID,
		"oidc_groups": ssoGroups.OIDCGroups,
		"synced_at":   ssoGroups.SyncedAt,
	})
}

// logAuditEvent logs an audit event for SSO group mapping operations.
func (h *SSOGroupMappingsHandler) logAuditEvent(ctx context.Context, orgID, userID uuid.UUID, action models.AuditAction, resourceType string, resourceID uuid.UUID) {
	auditLog := models.NewAuditLog(orgID, action, resourceType, models.AuditResultSuccess).
		WithUser(userID).
		WithResource(resourceID)

	if err := h.store.CreateAuditLog(ctx, auditLog); err != nil {
		h.logger.Error().Err(err).Msg("failed to create audit log")
	}
}
