package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DashboardMetricsStore defines the interface for dashboard metrics persistence operations.
type DashboardMetricsStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error)
	GetBackupSuccessRates(ctx context.Context, orgID uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error)
	GetStorageGrowthTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthTrend, error)
	GetBackupDurationTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.BackupDurationTrend, error)
	GetDailyBackupStats(ctx context.Context, orgID uuid.UUID, days int) ([]*models.DailyBackupStats, error)
	GetDailySummaries(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]models.MetricsDailySummary, error)
	GetActiveRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetCriticalRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
}

// DashboardMetricsHandler handles dashboard metrics related HTTP endpoints.
type DashboardMetricsHandler struct {
	store  DashboardMetricsStore
	logger zerolog.Logger
}

// NewDashboardMetricsHandler creates a new DashboardMetricsHandler.
func NewDashboardMetricsHandler(store DashboardMetricsStore, logger zerolog.Logger) *DashboardMetricsHandler {
	return &DashboardMetricsHandler{
		store:  store,
		logger: logger.With().Str("component", "dashboard_metrics_handler").Logger(),
	}
}

// RegisterRoutes registers dashboard metrics routes on the given router group.
func (h *DashboardMetricsHandler) RegisterRoutes(r *gin.RouterGroup) {
	metrics := r.Group("/dashboard-metrics")
	{
		metrics.GET("/stats", h.GetDashboardStats)
		metrics.GET("/success-rates", h.GetBackupSuccessRates)
		metrics.GET("/storage-growth", h.GetStorageGrowthTrend)
		metrics.GET("/backup-duration", h.GetBackupDurationTrend)
		metrics.GET("/daily-backups", h.GetDailyBackupStats)
	}
}

// GetDashboardStats returns aggregated dashboard statistics.
// GET /api/v1/dashboard-metrics/stats
func (h *DashboardMetricsHandler) GetDashboardStats(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	stats, err := h.store.GetDashboardStats(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get dashboard stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve dashboard stats"})
		return
	}

	// Get success rates
	rate7d, rate30d, err := h.store.GetBackupSuccessRates(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get success rates")
	} else {
		if rate7d != nil {
			stats.SuccessRate7d = rate7d.SuccessPercent
		}
		if rate30d != nil {
			stats.SuccessRate30d = rate30d.SuccessPercent
		}
	}

	// Get ransomware alert counts (displayed prominently)
	activeCount, err := h.store.GetActiveRansomwareAlertCountByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get active ransomware alert count")
	} else {
		stats.RansomwareAlertsActive = activeCount
	}

	criticalCount, err := h.store.GetCriticalRansomwareAlertCountByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get critical ransomware alert count")
	} else {
		stats.RansomwareAlertsCritical = criticalCount
	}

	c.JSON(http.StatusOK, stats)
}

// GetBackupSuccessRates returns backup success rates for different time periods.
// GET /api/v1/dashboard-metrics/success-rates
func (h *DashboardMetricsHandler) GetBackupSuccessRates(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	rate7d, rate30d, err := h.store.GetBackupSuccessRates(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get success rates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve success rates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rate_7d":  rate7d,
		"rate_30d": rate30d,
	})
}

// GetStorageGrowthTrend returns storage growth over time.
// GET /api/v1/dashboard-metrics/storage-growth?days=30
func (h *DashboardMetricsHandler) GetStorageGrowthTrend(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	trend, err := h.store.GetStorageGrowthTrend(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage growth trend")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage growth trend"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trend": trend})
}

// GetBackupDurationTrend returns backup duration trends over time.
// GET /api/v1/dashboard-metrics/backup-duration?days=30
func (h *DashboardMetricsHandler) GetBackupDurationTrend(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	trend, err := h.store.GetBackupDurationTrend(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get backup duration trend")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve backup duration trend"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trend": trend})
}

// GetDailyBackupStats returns daily backup statistics.
// For ranges > 7 days, uses pre-aggregated daily summaries for performance.
// Falls back to real-time query for recent data (7 days or less).
// GET /api/v1/dashboard-metrics/daily-backups?days=30
func (h *DashboardMetricsHandler) GetDailyBackupStats(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Use pre-aggregated summaries for longer ranges
	if days > 7 {
		now := time.Now().UTC()
		startDate := now.AddDate(0, 0, -days)
		endDate := now.AddDate(0, 0, -1) // Exclude today (not yet complete)

		summaries, err := h.store.GetDailySummaries(c.Request.Context(), dbUser.OrgID, startDate, endDate)
		if err == nil && len(summaries) > 0 {
			stats := make([]*models.DailyBackupStats, 0, len(summaries))
			for _, s := range summaries {
				stats = append(stats, &models.DailyBackupStats{
					Date:       s.Date,
					Total:      s.TotalBackups,
					Successful: s.SuccessfulBackups,
					Failed:     s.FailedBackups,
					TotalSize:  s.TotalSizeBytes,
				})
			}
			c.JSON(http.StatusOK, gin.H{"stats": stats})
			return
		}
		if err != nil {
			h.logger.Warn().Err(err).Msg("failed to get daily summaries, falling back to real-time query")
		}
	}

	// Fall back to real-time query
	stats, err := h.store.GetDailyBackupStats(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get daily backup stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve daily backup stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}
