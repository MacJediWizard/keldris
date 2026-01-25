package handlers

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// IPAllowlistStore defines the interface for IP allowlist persistence operations.
type IPAllowlistStore interface {
	GetIPAllowlistByID(ctx context.Context, id uuid.UUID) (*models.IPAllowlist, error)
	ListIPAllowlistsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.IPAllowlist, error)
	CreateIPAllowlist(ctx context.Context, a *models.IPAllowlist) error
	UpdateIPAllowlist(ctx context.Context, a *models.IPAllowlist) error
	DeleteIPAllowlist(ctx context.Context, id uuid.UUID) error
	GetOrCreateIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error)
	UpdateIPAllowlistSettings(ctx context.Context, s *models.IPAllowlistSettings) error
	ListIPBlockedAttemptsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.IPBlockedAttempt, int, error)
}

// IPFilterInvalidator allows invalidating the IP filter cache.
type IPFilterInvalidator interface {
	InvalidateCache(orgID uuid.UUID)
}

// IPAllowlistsHandler handles IP allowlist HTTP endpoints.
type IPAllowlistsHandler struct {
	store       IPAllowlistStore
	ipFilter    IPFilterInvalidator
	logger      zerolog.Logger
}

// NewIPAllowlistsHandler creates a new IPAllowlistsHandler.
func NewIPAllowlistsHandler(store IPAllowlistStore, ipFilter IPFilterInvalidator, logger zerolog.Logger) *IPAllowlistsHandler {
	return &IPAllowlistsHandler{
		store:    store,
		ipFilter: ipFilter,
		logger:   logger.With().Str("component", "ip_allowlists_handler").Logger(),
	}
}

// RegisterRoutes registers IP allowlist routes on the given router group.
func (h *IPAllowlistsHandler) RegisterRoutes(r *gin.RouterGroup) {
	allowlists := r.Group("/ip-allowlists")
	{
		allowlists.GET("", h.List)
		allowlists.POST("", h.Create)
		allowlists.GET("/:id", h.Get)
		allowlists.PUT("/:id", h.Update)
		allowlists.DELETE("/:id", h.Delete)
	}

	// Settings and blocked attempts
	r.GET("/ip-allowlist-settings", h.GetSettings)
	r.PUT("/ip-allowlist-settings", h.UpdateSettings)
	r.GET("/ip-blocked-attempts", h.ListBlockedAttempts)
}

// List returns all IP allowlist entries for the organization.
// GET /api/v1/ip-allowlists
func (h *IPAllowlistsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only admins can manage IP allowlists
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	allowlists, err := h.store.ListIPAllowlistsByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list IP allowlists")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list IP allowlists"})
		return
	}

	c.JSON(http.StatusOK, models.IPAllowlistsResponse{Allowlists: toIPAllowlistSlice(allowlists)})
}

// Get returns a specific IP allowlist entry by ID.
// GET /api/v1/ip-allowlists/:id
func (h *IPAllowlistsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allowlist ID"})
		return
	}

	allowlist, err := h.store.GetIPAllowlistByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	// Verify the allowlist belongs to the user's org
	if allowlist.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	c.JSON(http.StatusOK, allowlist)
}

// Create creates a new IP allowlist entry.
// POST /api/v1/ip-allowlists
func (h *IPAllowlistsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateIPAllowlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate CIDR
	if err := validateCIDR(req.CIDR); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowlist := models.NewIPAllowlist(user.CurrentOrgID, req.CIDR, req.Type)
	allowlist.Description = req.Description
	allowlist.CreatedBy = &user.ID
	allowlist.UpdatedBy = &user.ID

	if req.Enabled != nil {
		allowlist.Enabled = *req.Enabled
	}

	if err := h.store.CreateIPAllowlist(c.Request.Context(), allowlist); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create IP allowlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create IP allowlist"})
		return
	}

	// Invalidate cache
	if h.ipFilter != nil {
		h.ipFilter.InvalidateCache(user.CurrentOrgID)
	}

	h.logger.Info().
		Str("allowlist_id", allowlist.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("cidr", allowlist.CIDR).
		Str("type", string(allowlist.Type)).
		Msg("IP allowlist entry created")

	c.JSON(http.StatusCreated, allowlist)
}

// Update updates an existing IP allowlist entry.
// PUT /api/v1/ip-allowlists/:id
func (h *IPAllowlistsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allowlist ID"})
		return
	}

	allowlist, err := h.store.GetIPAllowlistByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	// Verify the allowlist belongs to the user's org
	if allowlist.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	var req models.UpdateIPAllowlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.CIDR != nil {
		if err := validateCIDR(*req.CIDR); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		allowlist.CIDR = *req.CIDR
	}
	if req.Description != nil {
		allowlist.Description = *req.Description
	}
	if req.Type != nil {
		allowlist.Type = *req.Type
	}
	if req.Enabled != nil {
		allowlist.Enabled = *req.Enabled
	}
	allowlist.UpdatedBy = &user.ID

	if err := h.store.UpdateIPAllowlist(c.Request.Context(), allowlist); err != nil {
		h.logger.Error().Err(err).Str("allowlist_id", id.String()).Msg("failed to update IP allowlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update IP allowlist"})
		return
	}

	// Invalidate cache
	if h.ipFilter != nil {
		h.ipFilter.InvalidateCache(user.CurrentOrgID)
	}

	h.logger.Info().
		Str("allowlist_id", allowlist.ID.String()).
		Str("cidr", allowlist.CIDR).
		Msg("IP allowlist entry updated")

	c.JSON(http.StatusOK, allowlist)
}

// Delete deletes an IP allowlist entry.
// DELETE /api/v1/ip-allowlists/:id
func (h *IPAllowlistsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allowlist ID"})
		return
	}

	allowlist, err := h.store.GetIPAllowlistByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	// Verify the allowlist belongs to the user's org
	if allowlist.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "allowlist entry not found"})
		return
	}

	if err := h.store.DeleteIPAllowlist(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("allowlist_id", id.String()).Msg("failed to delete IP allowlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete IP allowlist"})
		return
	}

	// Invalidate cache
	if h.ipFilter != nil {
		h.ipFilter.InvalidateCache(user.CurrentOrgID)
	}

	h.logger.Info().
		Str("allowlist_id", id.String()).
		Str("cidr", allowlist.CIDR).
		Msg("IP allowlist entry deleted")

	c.JSON(http.StatusOK, gin.H{"message": "IP allowlist entry deleted"})
}

// GetSettings returns the IP allowlist settings for the organization.
// GET /api/v1/ip-allowlist-settings
func (h *IPAllowlistsHandler) GetSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	settings, err := h.store.GetOrCreateIPAllowlistSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get IP allowlist settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get IP allowlist settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates the IP allowlist settings for the organization.
// PUT /api/v1/ip-allowlist-settings
func (h *IPAllowlistsHandler) UpdateSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	settings, err := h.store.GetOrCreateIPAllowlistSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get IP allowlist settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get IP allowlist settings"})
		return
	}

	var req models.UpdateIPAllowlistSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.EnforceForUI != nil {
		settings.EnforceForUI = *req.EnforceForUI
	}
	if req.EnforceForAgent != nil {
		settings.EnforceForAgent = *req.EnforceForAgent
	}
	if req.AllowAdminBypass != nil {
		settings.AllowAdminBypass = *req.AllowAdminBypass
	}

	if err := h.store.UpdateIPAllowlistSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update IP allowlist settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update IP allowlist settings"})
		return
	}

	// Invalidate cache
	if h.ipFilter != nil {
		h.ipFilter.InvalidateCache(user.CurrentOrgID)
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Bool("enabled", settings.Enabled).
		Bool("enforce_ui", settings.EnforceForUI).
		Bool("enforce_agent", settings.EnforceForAgent).
		Bool("admin_bypass", settings.AllowAdminBypass).
		Msg("IP allowlist settings updated")

	c.JSON(http.StatusOK, settings)
}

// ListBlockedAttempts returns the blocked access attempts for the organization.
// GET /api/v1/ip-blocked-attempts
func (h *IPAllowlistsHandler) ListBlockedAttempts(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Parse pagination params
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	attempts, total, err := h.store.ListIPBlockedAttemptsByOrg(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list blocked attempts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list blocked attempts"})
		return
	}

	c.JSON(http.StatusOK, models.IPBlockedAttemptsResponse{
		Attempts: toIPBlockedAttemptSlice(attempts),
		Total:    total,
	})
}

// validateCIDR validates that the input is a valid IP address or CIDR notation.
func validateCIDR(cidr string) error {
	// Try parsing as CIDR
	_, _, err := net.ParseCIDR(cidr)
	if err == nil {
		return nil
	}

	// Try parsing as single IP
	ip := net.ParseIP(cidr)
	if ip == nil {
		return &net.ParseError{Type: "IP address or CIDR", Text: cidr}
	}

	return nil
}

// toIPAllowlistSlice converts []*models.IPAllowlist to []models.IPAllowlist.
func toIPAllowlistSlice(allowlists []*models.IPAllowlist) []models.IPAllowlist {
	if allowlists == nil {
		return []models.IPAllowlist{}
	}
	result := make([]models.IPAllowlist, len(allowlists))
	for i, a := range allowlists {
		result[i] = *a
	}
	return result
}

// toIPBlockedAttemptSlice converts []*models.IPBlockedAttempt to []models.IPBlockedAttempt.
func toIPBlockedAttemptSlice(attempts []*models.IPBlockedAttempt) []models.IPBlockedAttempt {
	if attempts == nil {
		return []models.IPBlockedAttempt{}
	}
	result := make([]models.IPBlockedAttempt, len(attempts))
	for i, a := range attempts {
		result[i] = *a
	}
	return result
}
