package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RateLimitConfigStore defines the interface for rate limit persistence operations.
type RateLimitConfigStore interface {
	ListRateLimitConfigs(ctx context.Context, orgID uuid.UUID) ([]*models.RateLimitConfig, error)
	GetRateLimitConfigByID(ctx context.Context, id uuid.UUID) (*models.RateLimitConfig, error)
	GetRateLimitConfigByEndpoint(ctx context.Context, orgID uuid.UUID, endpoint string) (*models.RateLimitConfig, error)
	CreateRateLimitConfig(ctx context.Context, c *models.RateLimitConfig) error
	UpdateRateLimitConfig(ctx context.Context, c *models.RateLimitConfig) error
	DeleteRateLimitConfig(ctx context.Context, id uuid.UUID) error
	GetRateLimitStats(ctx context.Context, orgID uuid.UUID) (*models.RateLimitStats, error)
	ListRecentBlockedRequests(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.BlockedRequest, error)
	ListIPBans(ctx context.Context, orgID uuid.UUID) ([]*models.IPBan, error)
	ListActiveIPBans(ctx context.Context, orgID uuid.UUID) ([]*models.IPBan, error)
	CreateIPBan(ctx context.Context, b *models.IPBan) error
	DeleteIPBan(ctx context.Context, id uuid.UUID) error
}

// RateLimitsHandler handles rate limit HTTP endpoints.
type RateLimitsHandler struct {
	store  RateLimitConfigStore
	logger zerolog.Logger
}

// NewRateLimitsHandler creates a new RateLimitsHandler.
func NewRateLimitsHandler(store RateLimitConfigStore, logger zerolog.Logger) *RateLimitsHandler {
	return &RateLimitsHandler{
		store:  store,
		logger: logger.With().Str("component", "rate_limits_handler").Logger(),
	}
}

// RegisterRoutes registers rate limit routes on the given router group.
func (h *RateLimitsHandler) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/admin/rate-limit-configs")
	{
		admin.GET("", h.List)
		admin.POST("", h.Create)
		admin.GET("/stats", h.GetStats)
		admin.GET("/blocked", h.ListBlocked)
		admin.GET("/:id", h.Get)
		admin.PUT("/:id", h.Update)
		admin.DELETE("/:id", h.Delete)
	}

	// IP bans routes
	bans := r.Group("/admin/ip-bans")
	{
		bans.GET("", h.ListBans)
		bans.POST("", h.CreateBan)
		bans.DELETE("/:id", h.DeleteBan)
	}
}

// List returns all rate limit configurations for the organization.
// GET /api/v1/admin/rate-limits
func (h *RateLimitsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can view rate limits
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	configs, err := h.store.ListRateLimitConfigs(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list rate limit configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rate limit configs"})
		return
	}

	c.JSON(http.StatusOK, models.RateLimitConfigsResponse{Configs: toConfigSlice(configs)})
}

// Get returns a specific rate limit configuration by ID.
// GET /api/v1/admin/rate-limits/:id
func (h *RateLimitsHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rate limit config ID"})
		return
	}

	config, err := h.store.GetRateLimitConfigByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	// Verify the config belongs to the user's org
	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// Create creates a new rate limit configuration.
// POST /api/v1/admin/rate-limits
func (h *RateLimitsHandler) Create(c *gin.Context) {
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

	var req models.CreateRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.NewRateLimitConfig(user.CurrentOrgID, req.Endpoint)
	config.RequestsPerPeriod = req.RequestsPerPeriod
	config.PeriodSeconds = req.PeriodSeconds
	config.CreatedBy = &user.ID
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.store.CreateRateLimitConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create rate limit config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rate limit config"})
		return
	}

	h.logger.Info().
		Str("config_id", config.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("endpoint", config.Endpoint).
		Int("requests_per_period", config.RequestsPerPeriod).
		Int("period_seconds", config.PeriodSeconds).
		Msg("rate limit config created")

	c.JSON(http.StatusCreated, config)
}

// Update updates an existing rate limit configuration.
// PUT /api/v1/admin/rate-limits/:id
func (h *RateLimitsHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rate limit config ID"})
		return
	}

	config, err := h.store.GetRateLimitConfigByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	// Verify the config belongs to the user's org
	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	var req models.UpdateRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.RequestsPerPeriod != nil {
		if *req.RequestsPerPeriod < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "requests_per_period must be at least 1"})
			return
		}
		config.RequestsPerPeriod = *req.RequestsPerPeriod
	}
	if req.PeriodSeconds != nil {
		if *req.PeriodSeconds < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period_seconds must be at least 1"})
			return
		}
		config.PeriodSeconds = *req.PeriodSeconds
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.store.UpdateRateLimitConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to update rate limit config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rate limit config"})
		return
	}

	h.logger.Info().
		Str("config_id", config.ID.String()).
		Str("endpoint", config.Endpoint).
		Int("requests_per_period", config.RequestsPerPeriod).
		Int("period_seconds", config.PeriodSeconds).
		Bool("enabled", config.Enabled).
		Msg("rate limit config updated")

	c.JSON(http.StatusOK, config)
}

// Delete deletes a rate limit configuration.
// DELETE /api/v1/admin/rate-limits/:id
func (h *RateLimitsHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rate limit config ID"})
		return
	}

	config, err := h.store.GetRateLimitConfigByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	// Verify the config belongs to the user's org
	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rate limit config not found"})
		return
	}

	if err := h.store.DeleteRateLimitConfig(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to delete rate limit config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rate limit config"})
		return
	}

	h.logger.Info().
		Str("config_id", id.String()).
		Str("endpoint", config.Endpoint).
		Msg("rate limit config deleted")

	c.JSON(http.StatusOK, gin.H{"message": "rate limit config deleted"})
}

// GetStats returns rate limiting statistics.
// GET /api/v1/admin/rate-limits/stats
func (h *RateLimitsHandler) GetStats(c *gin.Context) {
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

	stats, err := h.store.GetRateLimitStats(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get rate limit stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rate limit stats"})
		return
	}

	c.JSON(http.StatusOK, models.RateLimitStatsResponse{Stats: *stats})
}

// ListBlocked returns recent blocked requests.
// GET /api/v1/admin/rate-limits/blocked
func (h *RateLimitsHandler) ListBlocked(c *gin.Context) {
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

	blocked, err := h.store.ListRecentBlockedRequests(c.Request.Context(), user.CurrentOrgID, 50)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list blocked requests")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list blocked requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blocked_requests": blocked})
}

// ListBans returns all IP bans.
// GET /api/v1/admin/ip-bans
func (h *RateLimitsHandler) ListBans(c *gin.Context) {
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

	bans, err := h.store.ListIPBans(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list IP bans")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list IP bans"})
		return
	}

	c.JSON(http.StatusOK, models.IPBansResponse{Bans: toBanSlice(bans)})
}

// CreateBan creates a new IP ban.
// POST /api/v1/admin/ip-bans
func (h *RateLimitsHandler) CreateBan(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req models.CreateIPBanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	ban := &models.IPBan{
		ID:        uuid.New(),
		OrgID:     &user.CurrentOrgID,
		IPAddress: req.IPAddress,
		Reason:    req.Reason,
		BanCount:  1,
		BannedBy:  &user.ID,
		BannedAt:  now,
		CreatedAt: now,
	}

	// Set expiration if duration is provided
	if req.DurationMinutes != nil && *req.DurationMinutes > 0 {
		expiresAt := now.Add(time.Duration(*req.DurationMinutes) * time.Minute)
		ban.ExpiresAt = &expiresAt
	}

	if err := h.store.CreateIPBan(c.Request.Context(), ban); err != nil {
		h.logger.Error().Err(err).Str("ip_address", req.IPAddress).Msg("failed to create IP ban")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create IP ban"})
		return
	}

	h.logger.Info().
		Str("ban_id", ban.ID.String()).
		Str("ip_address", ban.IPAddress).
		Str("reason", ban.Reason).
		Str("banned_by", user.ID.String()).
		Msg("IP ban created")

	c.JSON(http.StatusCreated, ban)
}

// DeleteBan removes an IP ban.
// DELETE /api/v1/admin/ip-bans/:id
func (h *RateLimitsHandler) DeleteBan(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ban ID"})
		return
	}

	if err := h.store.DeleteIPBan(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("ban_id", id.String()).Msg("failed to delete IP ban")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete IP ban"})
		return
	}

	h.logger.Info().
		Str("ban_id", id.String()).
		Str("deleted_by", user.ID.String()).
		Msg("IP ban deleted")

	c.JSON(http.StatusOK, gin.H{"message": "IP ban removed"})
}

// toConfigSlice converts []*models.RateLimitConfig to []models.RateLimitConfig.
func toConfigSlice(configs []*models.RateLimitConfig) []models.RateLimitConfig {
	if configs == nil {
		return []models.RateLimitConfig{}
	}
	result := make([]models.RateLimitConfig, len(configs))
	for i, c := range configs {
		result[i] = *c
	}
	return result
}

// toBanSlice converts []*models.IPBan to []models.IPBan.
func toBanSlice(bans []*models.IPBan) []models.IPBan {
	if bans == nil {
		return []models.IPBan{}
	}
	result := make([]models.IPBan, len(bans))
	for i, b := range bans {
		result[i] = *b
	}
	return result
}
