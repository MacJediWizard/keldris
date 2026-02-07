package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/metering"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UsageStore defines the interface for usage persistence operations.
type UsageStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)

	// Usage metrics
	GetUsageMetricsByOrgID(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]*models.UsageMetrics, error)
	GetLatestUsageMetrics(ctx context.Context, orgID uuid.UUID) (*models.UsageMetrics, error)

	// Usage limits
	GetOrgUsageLimits(ctx context.Context, orgID uuid.UUID) (*models.OrgUsageLimits, error)
	CreateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error
	UpdateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error
	UpsertOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error

	// Usage alerts
	GetActiveUsageAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UsageAlert, error)
	AcknowledgeUsageAlert(ctx context.Context, id, userID uuid.UUID) error
	ResolveUsageAlert(ctx context.Context, id uuid.UUID) error

	// Monthly summaries
	GetMonthlyUsageSummary(ctx context.Context, orgID uuid.UUID, yearMonth string) (*models.MonthlyUsageSummary, error)
	GetMonthlyUsageSummariesByOrgID(ctx context.Context, orgID uuid.UUID, months int) ([]*models.MonthlyUsageSummary, error)
}

// UsageHandler handles usage-related HTTP endpoints.
type UsageHandler struct {
	store          UsageStore
	meteringService *metering.Service
	logger         zerolog.Logger
}

// NewUsageHandler creates a new UsageHandler.
func NewUsageHandler(store UsageStore, meteringService *metering.Service, logger zerolog.Logger) *UsageHandler {
	return &UsageHandler{
		store:          store,
		meteringService: meteringService,
		logger:         logger.With().Str("component", "usage_handler").Logger(),
	}
}

// RegisterRoutes registers usage routes on the given router group.
func (h *UsageHandler) RegisterRoutes(r *gin.RouterGroup) {
	usage := r.Group("/usage")
	{
		// Current usage dashboard
		usage.GET("/current", h.GetCurrentUsage)

		// Usage history for charts
		usage.GET("/history", h.GetUsageHistory)

		// Usage limits management
		usage.GET("/limits", h.GetUsageLimits)
		usage.PUT("/limits", h.UpdateUsageLimits)

		// Usage alerts
		usage.GET("/alerts", h.GetUsageAlerts)
		usage.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)

		// Monthly summaries for billing
		usage.GET("/monthly", h.GetMonthlySummaries)
		usage.GET("/monthly/:year_month", h.GetMonthlySummary)

		// Billing report endpoint
		usage.GET("/billing-report", h.GetBillingReport)
	}
}

// GetCurrentUsage returns the current usage state for the organization.
//
//	@Summary		Get current usage
//	@Description	Returns current usage metrics, limits, and active alerts for the organization
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.CurrentUsage
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/current [get]
func (h *UsageHandler) GetCurrentUsage(c *gin.Context) {
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

	usage, err := h.meteringService.GetCurrentUsage(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get current usage")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve current usage"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetUsageHistory returns usage history for charts.
//
//	@Summary		Get usage history
//	@Description	Returns daily usage metrics for the specified number of days
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			days	query		int	false	"Number of days of history (default 30, max 365)"
//	@Success		200	{object}	map[string][]models.UsageHistoryPoint
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/history [get]
func (h *UsageHandler) GetUsageHistory(c *gin.Context) {
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

	history, err := h.meteringService.GetUsageHistory(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get usage history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve usage history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GetUsageLimits returns the usage limits for the organization.
//
//	@Summary		Get usage limits
//	@Description	Returns the configured usage limits for the organization
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.OrgUsageLimits
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/limits [get]
func (h *UsageHandler) GetUsageLimits(c *gin.Context) {
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

	limits, err := h.store.GetOrgUsageLimits(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get usage limits")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve usage limits"})
		return
	}

	if limits == nil {
		// Return default limits
		limits = models.NewOrgUsageLimits(dbUser.OrgID)
	}

	c.JSON(http.StatusOK, limits)
}

// UpdateUsageLimitsRequest is the request body for updating usage limits.
type UpdateUsageLimitsRequest struct {
	MaxAgents          *int       `json:"max_agents,omitempty"`
	MaxUsers           *int       `json:"max_users,omitempty"`
	MaxStorageBytes    *int64     `json:"max_storage_bytes,omitempty"`
	MaxBackupsPerMonth *int       `json:"max_backups_per_month,omitempty"`
	MaxRepositories    *int       `json:"max_repositories,omitempty"`
	WarningThreshold   *int       `json:"warning_threshold,omitempty"`
	CriticalThreshold  *int       `json:"critical_threshold,omitempty"`
	BillingTier        *string    `json:"billing_tier,omitempty"`
	BillingPeriodStart *time.Time `json:"billing_period_start,omitempty"`
	BillingPeriodEnd   *time.Time `json:"billing_period_end,omitempty"`
}

// UpdateUsageLimits updates the usage limits for the organization.
//
//	@Summary		Update usage limits
//	@Description	Updates the configured usage limits for the organization (admin only)
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			request	body		UpdateUsageLimitsRequest	true	"Limits to update"
//	@Success		200		{object}	models.OrgUsageLimits
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/limits [put]
func (h *UsageHandler) UpdateUsageLimits(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Check admin role
	if user.CurrentOrgRole != "admin" && user.CurrentOrgRole != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	var req UpdateUsageLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate thresholds
	if req.WarningThreshold != nil && (*req.WarningThreshold < 0 || *req.WarningThreshold > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "warning_threshold must be between 0 and 100"})
		return
	}
	if req.CriticalThreshold != nil && (*req.CriticalThreshold < 0 || *req.CriticalThreshold > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "critical_threshold must be between 0 and 100"})
		return
	}

	// Get existing limits or create new
	limits, err := h.store.GetOrgUsageLimits(c.Request.Context(), dbUser.OrgID)
	if err != nil || limits == nil {
		limits = models.NewOrgUsageLimits(dbUser.OrgID)
	}

	// Update fields
	if req.MaxAgents != nil {
		limits.MaxAgents = req.MaxAgents
	}
	if req.MaxUsers != nil {
		limits.MaxUsers = req.MaxUsers
	}
	if req.MaxStorageBytes != nil {
		limits.MaxStorageBytes = req.MaxStorageBytes
	}
	if req.MaxBackupsPerMonth != nil {
		limits.MaxBackupsPerMonth = req.MaxBackupsPerMonth
	}
	if req.MaxRepositories != nil {
		limits.MaxRepositories = req.MaxRepositories
	}
	if req.WarningThreshold != nil {
		limits.WarningThreshold = *req.WarningThreshold
	}
	if req.CriticalThreshold != nil {
		limits.CriticalThreshold = *req.CriticalThreshold
	}
	if req.BillingTier != nil {
		limits.BillingTier = *req.BillingTier
	}
	if req.BillingPeriodStart != nil {
		limits.BillingPeriodStart = req.BillingPeriodStart
	}
	if req.BillingPeriodEnd != nil {
		limits.BillingPeriodEnd = req.BillingPeriodEnd
	}

	limits.UpdatedAt = time.Now()

	if err := h.store.UpsertOrgUsageLimits(c.Request.Context(), limits); err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to update usage limits")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update usage limits"})
		return
	}

	h.logger.Info().
		Str("org_id", dbUser.OrgID.String()).
		Str("updated_by", user.ID.String()).
		Msg("usage limits updated")

	c.JSON(http.StatusOK, limits)
}

// GetUsageAlerts returns active usage alerts for the organization.
//
//	@Summary		Get usage alerts
//	@Description	Returns active usage alerts for the organization
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.UsageAlert
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/alerts [get]
func (h *UsageHandler) GetUsageAlerts(c *gin.Context) {
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

	alerts, err := h.store.GetActiveUsageAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get usage alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve usage alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.UsageAlert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// AcknowledgeAlert acknowledges a usage alert.
//
//	@Summary		Acknowledge usage alert
//	@Description	Marks a usage alert as acknowledged
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Alert ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/alerts/{id}/acknowledge [post]
func (h *UsageHandler) AcknowledgeAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	if err := h.store.AcknowledgeUsageAlert(c.Request.Context(), id, user.ID); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to acknowledge alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to acknowledge alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("acknowledged_by", user.ID.String()).
		Msg("usage alert acknowledged")

	c.JSON(http.StatusOK, gin.H{"message": "alert acknowledged"})
}

// GetMonthlySummaries returns monthly usage summaries for the organization.
//
//	@Summary		Get monthly summaries
//	@Description	Returns monthly usage summaries for billing
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			months	query		int	false	"Number of months of history (default 12)"
//	@Success		200	{object}	map[string][]models.MonthlyUsageSummary
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/monthly [get]
func (h *UsageHandler) GetMonthlySummaries(c *gin.Context) {
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

	months := 12
	if monthsParam := c.Query("months"); monthsParam != "" {
		if m, err := strconv.Atoi(monthsParam); err == nil && m > 0 && m <= 36 {
			months = m
		}
	}

	summaries, err := h.store.GetMonthlyUsageSummariesByOrgID(c.Request.Context(), dbUser.OrgID, months)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get monthly summaries")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve monthly summaries"})
		return
	}

	if summaries == nil {
		summaries = []*models.MonthlyUsageSummary{}
	}

	c.JSON(http.StatusOK, gin.H{"summaries": summaries})
}

// GetMonthlySummary returns a specific monthly usage summary.
//
//	@Summary		Get monthly summary
//	@Description	Returns the usage summary for a specific month
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			year_month	path		string	true	"Month in YYYY-MM format"
//	@Success		200		{object}	models.MonthlyUsageSummary
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/monthly/{year_month} [get]
func (h *UsageHandler) GetMonthlySummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	yearMonth := c.Param("year_month")
	// Validate format
	if _, err := time.Parse("2006-01", yearMonth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year_month format, expected YYYY-MM"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	summary, err := h.store.GetMonthlyUsageSummary(c.Request.Context(), dbUser.OrgID, yearMonth)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get monthly summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve monthly summary"})
		return
	}

	if summary == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "summary not found for the specified month"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetBillingReport returns a billing report for the organization.
//
//	@Summary		Get billing report
//	@Description	Returns a billing-ready usage report for the specified month
//	@Tags			Usage
//	@Accept			json
//	@Produce		json
//	@Param			year_month	query		string	false	"Month in YYYY-MM format (default: current month)"
//	@Success		200		{object}	models.BillingUsageReport
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/usage/billing-report [get]
func (h *UsageHandler) GetBillingReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	yearMonth := c.Query("year_month")
	if yearMonth == "" {
		yearMonth = time.Now().Format("2006-01")
	}

	// Validate format
	if _, err := time.Parse("2006-01", yearMonth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year_month format, expected YYYY-MM"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	report, err := h.meteringService.GetBillingReport(c.Request.Context(), dbUser.OrgID, yearMonth)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get billing report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate billing report"})
		return
	}

	c.JSON(http.StatusOK, report)
}
