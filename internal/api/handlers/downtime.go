package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/monitoring"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DowntimeHandler handles downtime-related HTTP endpoints.
type DowntimeHandler struct {
	service *monitoring.DowntimeService
	store   DowntimeStore
	logger  zerolog.Logger
}

// DowntimeStore defines the interface for downtime persistence operations.
type DowntimeStore interface {
	GetUserByID(c context.Context, id uuid.UUID) (*models.User, error)
}

// NewDowntimeHandler creates a new DowntimeHandler.
func NewDowntimeHandler(service *monitoring.DowntimeService, store DowntimeStore, logger zerolog.Logger) *DowntimeHandler {
	return &DowntimeHandler{
		service: service,
		store:   store,
		logger:  logger.With().Str("component", "downtime_handler").Logger(),
	}
}

// RegisterRoutes registers downtime routes on the given router group.
func (h *DowntimeHandler) RegisterRoutes(r *gin.RouterGroup) {
	downtime := r.Group("/downtime")
	{
		downtime.GET("", h.ListEvents)
		downtime.GET("/active", h.ListActiveEvents)
		downtime.GET("/summary", h.GetSummary)
		downtime.POST("", h.CreateEvent)
		downtime.GET("/:id", h.GetEvent)
		downtime.PUT("/:id", h.UpdateEvent)
		downtime.POST("/:id/resolve", h.ResolveEvent)
		downtime.DELETE("/:id", h.DeleteEvent)
	}

	uptime := r.Group("/uptime")
	{
		uptime.GET("/badges", h.GetBadges)
		uptime.POST("/badges/refresh", h.RefreshBadges)
		uptime.GET("/report/:year/:month", h.GetMonthlyReport)
	}

	downtimeAlerts := r.Group("/downtime-alerts")
	{
		downtimeAlerts.GET("", h.ListAlerts)
		downtimeAlerts.POST("", h.CreateAlert)
		downtimeAlerts.GET("/:id", h.GetAlert)
		downtimeAlerts.PUT("/:id", h.UpdateAlert)
		downtimeAlerts.DELETE("/:id", h.DeleteAlert)
	}
}

// ListEvents returns all downtime events for the authenticated user's organization.
func (h *DowntimeHandler) ListEvents(c *gin.Context) {
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

	// Parse pagination params
	limit := 100
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	events, err := h.service.ListDowntimeEvents(c.Request.Context(), dbUser.OrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list downtime events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list downtime events"})
		return
	}

	if events == nil {
		events = []*models.DowntimeEvent{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// ListActiveEvents returns active (ongoing) downtime events.
func (h *DowntimeHandler) ListActiveEvents(c *gin.Context) {
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

	events, err := h.service.ListActiveDowntime(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list active downtime events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list active downtime events"})
		return
	}

	if events == nil {
		events = []*models.DowntimeEvent{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetSummary returns the uptime summary for the organization.
func (h *DowntimeHandler) GetSummary(c *gin.Context) {
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

	summary, err := h.service.GetUptimeSummary(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get uptime summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get uptime summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// CreateEventRequest is the request body for creating a downtime event.
type CreateEventRequest struct {
	ComponentType string  `json:"component_type" binding:"required"`
	ComponentID   *string `json:"component_id,omitempty"`
	ComponentName string  `json:"component_name" binding:"required"`
	Severity      string  `json:"severity" binding:"required"`
	Cause         string  `json:"cause,omitempty"`
}

// CreateEvent creates a new downtime event.
func (h *DowntimeHandler) CreateEvent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate component type
	componentType := models.ComponentType(req.ComponentType)
	switch componentType {
	case models.ComponentTypeAgent, models.ComponentTypeServer, models.ComponentTypeRepository, models.ComponentTypeService:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid component type"})
		return
	}

	// Validate severity
	severity := models.DowntimeSeverity(req.Severity)
	switch severity {
	case models.DowntimeSeverityInfo, models.DowntimeSeverityWarning, models.DowntimeSeverityCritical:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	var componentID *uuid.UUID
	if req.ComponentID != nil {
		id, err := uuid.Parse(*req.ComponentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid component ID"})
			return
		}
		componentID = &id
	}

	event, err := h.service.RecordDowntimeStart(c.Request.Context(), dbUser.OrgID, componentType, componentID, req.ComponentName, severity, req.Cause)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create downtime event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create downtime event"})
		return
	}

	h.logger.Info().
		Str("event_id", event.ID.String()).
		Str("component_type", req.ComponentType).
		Str("component_name", req.ComponentName).
		Msg("downtime event created")

	c.JSON(http.StatusCreated, event)
}

// GetEvent returns a specific downtime event by ID.
func (h *DowntimeHandler) GetEvent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	event, err := h.service.GetDowntimeEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	// Verify user has access to this event's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if event.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// UpdateEventRequest is the request body for updating a downtime event.
type UpdateEventRequest struct {
	Severity *string `json:"severity,omitempty"`
	Cause    *string `json:"cause,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// UpdateEvent updates an existing downtime event.
func (h *DowntimeHandler) UpdateEvent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	event, err := h.service.GetDowntimeEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	// Verify user has access to this event's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if event.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Severity != nil {
		severity := models.DowntimeSeverity(*req.Severity)
		switch severity {
		case models.DowntimeSeverityInfo, models.DowntimeSeverityWarning, models.DowntimeSeverityCritical:
			event.Severity = severity
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity"})
			return
		}
	}

	if req.Cause != nil {
		event.Cause = req.Cause
	}

	if req.Notes != nil {
		event.Notes = req.Notes
	}

	if err := h.service.UpdateDowntimeEvent(c.Request.Context(), event); err != nil {
		h.logger.Error().Err(err).Str("event_id", id.String()).Msg("failed to update downtime event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update downtime event"})
		return
	}

	h.logger.Info().Str("event_id", id.String()).Msg("downtime event updated")

	c.JSON(http.StatusOK, event)
}

// ResolveEventRequest is the request body for resolving a downtime event.
type ResolveEventRequest struct {
	Notes string `json:"notes,omitempty"`
}

// ResolveEvent marks a downtime event as resolved.
func (h *DowntimeHandler) ResolveEvent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	event, err := h.service.GetDowntimeEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	// Verify user has access to this event's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if event.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	if !event.IsActive() {
		c.JSON(http.StatusOK, event) // Already resolved
		return
	}

	var req ResolveEventRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	resolvedEvent, err := h.service.RecordDowntimeEnd(c.Request.Context(), id, &user.ID, req.Notes)
	if err != nil {
		h.logger.Error().Err(err).Str("event_id", id.String()).Msg("failed to resolve downtime event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve downtime event"})
		return
	}

	h.logger.Info().
		Str("event_id", id.String()).
		Str("resolved_by", user.ID.String()).
		Msg("downtime event resolved")

	c.JSON(http.StatusOK, resolvedEvent)
}

// DeleteEvent deletes a downtime event.
func (h *DowntimeHandler) DeleteEvent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	event, err := h.service.GetDowntimeEvent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	// Verify user has access to this event's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if event.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime event not found"})
		return
	}

	if err := h.service.DeleteDowntimeEvent(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("event_id", id.String()).Msg("failed to delete downtime event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete downtime event"})
		return
	}

	h.logger.Info().Str("event_id", id.String()).Msg("downtime event deleted")

	c.JSON(http.StatusOK, gin.H{"message": "downtime event deleted"})
}

// Uptime Badge Handlers

// GetBadges returns uptime badges for the organization.
func (h *DowntimeHandler) GetBadges(c *gin.Context) {
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

	summary, err := h.service.GetUptimeSummary(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get badges")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get badges"})
		return
	}

	badges := summary.Badges
	if badges == nil {
		badges = []*models.UptimeBadge{}
	}

	c.JSON(http.StatusOK, gin.H{"badges": badges})
}

// RefreshBadges refreshes uptime badges for the organization.
func (h *DowntimeHandler) RefreshBadges(c *gin.Context) {
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

	if err := h.service.UpdateUptimeBadges(c.Request.Context(), dbUser.OrgID); err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to refresh badges")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh badges"})
		return
	}

	h.logger.Info().Str("org_id", dbUser.OrgID.String()).Msg("uptime badges refreshed")

	c.JSON(http.StatusOK, gin.H{"message": "badges refreshed"})
}

// GetMonthlyReport returns the monthly uptime report.
func (h *DowntimeHandler) GetMonthlyReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	yearStr := c.Param("year")
	monthStr := c.Param("month")

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	report, err := h.service.GetMonthlyReport(c.Request.Context(), dbUser.OrgID, year, month)
	if err != nil {
		h.logger.Error().Err(err).
			Str("org_id", dbUser.OrgID.String()).
			Int("year", year).
			Int("month", month).
			Msg("failed to get monthly report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get monthly report"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// Downtime Alert Handlers

// ListAlerts returns all downtime alerts for the organization.
func (h *DowntimeHandler) ListAlerts(c *gin.Context) {
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

	alerts, err := h.service.ListDowntimeAlerts(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list downtime alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list downtime alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.DowntimeAlert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// CreateAlertRequest is the request body for creating a downtime alert.
type CreateAlertRequest struct {
	Name             string  `json:"name" binding:"required,min=1,max=255"`
	UptimeThreshold  float64 `json:"uptime_threshold" binding:"required,min=0,max=100"`
	EvaluationPeriod string  `json:"evaluation_period" binding:"required"`
	ComponentType    *string `json:"component_type,omitempty"`
	NotifyOnBreach   *bool   `json:"notify_on_breach,omitempty"`
	NotifyOnRecovery *bool   `json:"notify_on_recovery,omitempty"`
}

// CreateAlert creates a new downtime alert.
func (h *DowntimeHandler) CreateAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate evaluation period
	switch req.EvaluationPeriod {
	case "7d", "30d", "90d":
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evaluation period (must be 7d, 30d, or 90d)"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	alert := models.NewDowntimeAlert(dbUser.OrgID, req.Name, req.UptimeThreshold, req.EvaluationPeriod)

	if req.ComponentType != nil {
		ct := models.ComponentType(*req.ComponentType)
		switch ct {
		case models.ComponentTypeAgent, models.ComponentTypeServer, models.ComponentTypeRepository, models.ComponentTypeService:
			alert.ComponentType = &ct
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid component type"})
			return
		}
	}

	if req.NotifyOnBreach != nil {
		alert.NotifyOnBreach = *req.NotifyOnBreach
	}

	if req.NotifyOnRecovery != nil {
		alert.NotifyOnRecovery = *req.NotifyOnRecovery
	}

	if err := h.service.CreateDowntimeAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create downtime alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create downtime alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("name", alert.Name).
		Float64("threshold", alert.UptimeThreshold).
		Msg("downtime alert created")

	c.JSON(http.StatusCreated, alert)
}

// GetAlert returns a specific downtime alert by ID.
func (h *DowntimeHandler) GetAlert(c *gin.Context) {
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

	alert, err := h.service.GetDowntimeAlert(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// UpdateAlertRequest is the request body for updating a downtime alert.
type UpdateAlertRequest struct {
	Name             *string  `json:"name,omitempty"`
	Enabled          *bool    `json:"enabled,omitempty"`
	UptimeThreshold  *float64 `json:"uptime_threshold,omitempty"`
	EvaluationPeriod *string  `json:"evaluation_period,omitempty"`
	NotifyOnBreach   *bool    `json:"notify_on_breach,omitempty"`
	NotifyOnRecovery *bool    `json:"notify_on_recovery,omitempty"`
}

// UpdateAlert updates an existing downtime alert.
func (h *DowntimeHandler) UpdateAlert(c *gin.Context) {
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

	alert, err := h.service.GetDowntimeAlert(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
		return
	}

	var req UpdateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != nil {
		alert.Name = *req.Name
	}
	if req.Enabled != nil {
		alert.Enabled = *req.Enabled
	}
	if req.UptimeThreshold != nil {
		alert.UptimeThreshold = *req.UptimeThreshold
	}
	if req.EvaluationPeriod != nil {
		switch *req.EvaluationPeriod {
		case "7d", "30d", "90d":
			alert.EvaluationPeriod = *req.EvaluationPeriod
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evaluation period"})
			return
		}
	}
	if req.NotifyOnBreach != nil {
		alert.NotifyOnBreach = *req.NotifyOnBreach
	}
	if req.NotifyOnRecovery != nil {
		alert.NotifyOnRecovery = *req.NotifyOnRecovery
	}

	if err := h.service.UpdateDowntimeAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update downtime alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update downtime alert"})
		return
	}

	h.logger.Info().Str("alert_id", id.String()).Msg("downtime alert updated")

	c.JSON(http.StatusOK, alert)
}

// DeleteAlert deletes a downtime alert.
func (h *DowntimeHandler) DeleteAlert(c *gin.Context) {
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

	alert, err := h.service.GetDowntimeAlert(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "downtime alert not found"})
		return
	}

	if err := h.service.DeleteDowntimeAlert(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to delete downtime alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete downtime alert"})
		return
	}

	h.logger.Info().Str("alert_id", id.String()).Msg("downtime alert deleted")

	c.JSON(http.StatusOK, gin.H{"message": "downtime alert deleted"})
}
