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

// AlertStore defines the interface for alert persistence operations.
type AlertStore interface {
	GetAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error)
	GetActiveAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error)
	GetActiveAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetAlertByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	UpdateAlert(ctx context.Context, alert *models.Alert) error
	GetAlertRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error)
	GetAlertRuleByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error)
	CreateAlertRule(ctx context.Context, rule *models.AlertRule) error
	UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error
	DeleteAlertRule(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// AlertsHandler handles alert-related HTTP endpoints.
type AlertsHandler struct {
	store  AlertStore
	logger zerolog.Logger
}

// NewAlertsHandler creates a new AlertsHandler.
func NewAlertsHandler(store AlertStore, logger zerolog.Logger) *AlertsHandler {
	return &AlertsHandler{
		store:  store,
		logger: logger.With().Str("component", "alerts_handler").Logger(),
	}
}

// RegisterRoutes registers alert routes on the given router group.
func (h *AlertsHandler) RegisterRoutes(r *gin.RouterGroup) {
	alerts := r.Group("/alerts")
	{
		alerts.GET("", h.List)
		alerts.GET("/active", h.ListActive)
		alerts.GET("/count", h.Count)
		alerts.GET("/:id", h.Get)
		alerts.POST("/:id/actions/acknowledge", h.Acknowledge)
		alerts.POST("/:id/actions/resolve", h.Resolve)
	}

	rules := r.Group("/alert-rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.GET("/:id", h.GetRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}
}

// List returns all alerts for the authenticated user's organization.
//
//	@Summary		List alerts
//	@Description	Returns all alerts for the current organization
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.Alert
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/alerts [get]
// GET /api/v1/alerts
func (h *AlertsHandler) List(c *gin.Context) {
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

	alerts, err := h.store.GetAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.Alert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// ListActive returns active (non-resolved) alerts for the authenticated user's organization.
// GET /api/v1/alerts/active
func (h *AlertsHandler) ListActive(c *gin.Context) {
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

	alerts, err := h.store.GetActiveAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list active alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.Alert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// Count returns the count of active alerts for the authenticated user's organization.
//
//	@Summary		Count active alerts
//	@Description	Returns the count of active (non-resolved) alerts for the current organization
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]int
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/alerts/count [get]
// GET /api/v1/alerts/count
func (h *AlertsHandler) Count(c *gin.Context) {
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

	count, err := h.store.GetActiveAlertCountByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to count alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// Get returns a specific alert by ID.
// GET /api/v1/alerts/:id
func (h *AlertsHandler) Get(c *gin.Context) {
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

	alert, err := h.store.GetAlertByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to get alert")
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	// Verify user has access to this alert's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// Acknowledge marks an alert as acknowledged.
// POST /api/v1/alerts/:id/actions/acknowledge
func (h *AlertsHandler) Acknowledge(c *gin.Context) {
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

	alert, err := h.store.GetAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	// Verify user has access to this alert's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if alert.Status == models.AlertStatusResolved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot acknowledge a resolved alert"})
		return
	}

	alert.Acknowledge(user.ID)

	if err := h.store.UpdateAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to acknowledge alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to acknowledge alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("acknowledged_by", user.ID.String()).
		Msg("alert acknowledged")

	c.JSON(http.StatusOK, alert)
}

// Resolve marks an alert as resolved.
// POST /api/v1/alerts/:id/actions/resolve
func (h *AlertsHandler) Resolve(c *gin.Context) {
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

	alert, err := h.store.GetAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	// Verify user has access to this alert's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if alert.Status == models.AlertStatusResolved {
		c.JSON(http.StatusOK, alert) // Already resolved
		return
	}

	alert.Resolve()

	if err := h.store.UpdateAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to resolve alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve alert"})
		return
	}

	h.logger.Info().Str("alert_id", id.String()).Msg("alert resolved")

	c.JSON(http.StatusOK, alert)
}

// Alert Rule Handlers

// ListRules returns all alert rules for the authenticated user's organization.
// GET /api/v1/alert-rules
func (h *AlertsHandler) ListRules(c *gin.Context) {
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

	rules, err := h.store.GetAlertRulesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list alert rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alert rules"})
		return
	}

	if rules == nil {
		rules = []*models.AlertRule{}
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// CreateRuleRequest is the request body for creating an alert rule.
type CreateRuleRequest struct {
	Name    string                 `json:"name" binding:"required,min=1,max=255"`
	Type    models.AlertType       `json:"type" binding:"required"`
	Enabled bool                   `json:"enabled"`
	Config  models.AlertRuleConfig `json:"config" binding:"required"`
	Name    string                  `json:"name" binding:"required,min=1,max=255"`
	Type    models.AlertType        `json:"type" binding:"required"`
	Enabled bool                    `json:"enabled"`
	Config  models.AlertRuleConfig  `json:"config" binding:"required"`
}

// CreateRule creates a new alert rule.
// POST /api/v1/alert-rules
func (h *AlertsHandler) CreateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate rule type
	switch req.Type {
	case models.AlertTypeAgentOffline, models.AlertTypeBackupSLA, models.AlertTypeStorageUsage,
		models.AlertTypeAgentHealthWarning, models.AlertTypeAgentHealthCritical, models.AlertTypeRansomwareSuspected:
	case models.AlertTypeAgentOffline, models.AlertTypeBackupSLA, models.AlertTypeStorageUsage:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert type"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	rule := models.NewAlertRule(dbUser.OrgID, req.Name, req.Type, req.Config)
	rule.Enabled = req.Enabled

	if err := h.store.CreateAlertRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create alert rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alert rule"})
		return
	}

	h.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("name", rule.Name).
		Str("type", string(rule.Type)).
		Msg("alert rule created")

	c.JSON(http.StatusCreated, rule)
}

// GetRule returns a specific alert rule by ID.
// GET /api/v1/alert-rules/:id
func (h *AlertsHandler) GetRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetAlertRuleByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to get alert rule")
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	// Verify user has access to this rule's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRuleRequest is the request body for updating an alert rule.
type UpdateRuleRequest struct {
	Name    *string                 `json:"name,omitempty"`
	Enabled *bool                   `json:"enabled,omitempty"`
	Config  *models.AlertRuleConfig `json:"config,omitempty"`
}

// UpdateRule updates an existing alert rule.
// PUT /api/v1/alert-rules/:id
func (h *AlertsHandler) UpdateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetAlertRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	// Verify user has access to this rule's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Config != nil {
		rule.Config = *req.Config
	}

	if err := h.store.UpdateAlertRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to update alert rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert rule"})
		return
	}

	h.logger.Info().Str("rule_id", id.String()).Msg("alert rule updated")

	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes an alert rule.
// DELETE /api/v1/alert-rules/:id
func (h *AlertsHandler) DeleteRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetAlertRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	// Verify user has access to this rule's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert rule not found"})
		return
	}

	if err := h.store.DeleteAlertRule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to delete alert rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete alert rule"})
		return
	}

	h.logger.Info().Str("rule_id", id.String()).Msg("alert rule deleted")
	c.JSON(http.StatusOK, gin.H{"message": "alert rule deleted"})
}
