package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RateLimitStore defines the interface for fetching user and membership data.
type RateLimitStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
}

// RateLimitHandler handles rate limit related requests.
type RateLimitHandler struct {
	store  RateLimitStore
	logger zerolog.Logger
}

// NewRateLimitHandler creates a new rate limit handler.
func NewRateLimitHandler(store RateLimitStore, logger zerolog.Logger) *RateLimitHandler {
	return &RateLimitHandler{
		store:  store,
		logger: logger.With().Str("component", "ratelimit_handler").Logger(),
	}
}

// RegisterRoutes registers rate limit routes on the given router group.
func (h *RateLimitHandler) RegisterRoutes(r *gin.RouterGroup) {
	adminRateLimits := r.Group("/admin/rate-limits")
	{
		adminRateLimits.GET("", h.GetDashboardStats)
	}
}

// isAdmin checks if the current user is an admin or owner.
func (h *RateLimitHandler) isAdmin(c *gin.Context) bool {
	user := middleware.GetUser(c)
	if user == nil {
		return false
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		return false
	}

	membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get membership")
		return false
	}

	return membership.Role == models.OrgRoleOwner || membership.Role == models.OrgRoleAdmin
}

// requireAdmin ensures the user is an admin and aborts if not.
func (h *RateLimitHandler) requireAdmin(c *gin.Context) bool {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return false
	}
	return true
}

// GetDashboardStats godoc
// @Summary Get rate limit dashboard statistics
// @Description Returns rate limit statistics for the admin dashboard including client stats and endpoint configurations
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} RateLimitDashboardStatsResponse "Rate limit statistics"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /admin/rate-limits [get]
func (h *RateLimitHandler) GetDashboardStats(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	manager := middleware.GetRateLimitManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit manager not initialized"})
		return
	}

	stats := manager.GetDashboardStats()
	c.JSON(http.StatusOK, stats)
}

// RateLimitDashboardStatsResponse is the response for rate limit dashboard stats.
type RateLimitDashboardStatsResponse struct {
	DefaultLimit    int64                        `json:"default_limit"`
	DefaultPeriod   string                       `json:"default_period"`
	EndpointConfigs []EndpointRateLimitInfoResp  `json:"endpoint_configs"`
	ClientStats     []RateLimitClientStatsResp   `json:"client_stats"`
	TotalRequests   int64                        `json:"total_requests"`
	TotalRejected   int64                        `json:"total_rejected"`
}

// EndpointRateLimitInfoResp holds rate limit info for an endpoint.
type EndpointRateLimitInfoResp struct {
	Pattern string `json:"pattern"`
	Limit   int64  `json:"limit"`
	Period  string `json:"period"`
}

// RateLimitClientStatsResp holds per-client statistics.
type RateLimitClientStatsResp struct {
	ClientIP      string `json:"client_ip"`
	TotalRequests int64  `json:"total_requests"`
	RejectedCount int64  `json:"rejected_count"`
	LastRequest   string `json:"last_request"`
}

// RateLimitStatusResponse represents the rate limit status included in API responses.
type RateLimitStatusResponse struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
	Reset     int64 `json:"reset"`
}
