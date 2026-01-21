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

// MaintenanceStore defines the interface for maintenance persistence operations.
type MaintenanceStore interface {
	GetMaintenanceWindowByID(ctx context.Context, id uuid.UUID) (*models.MaintenanceWindow, error)
	ListMaintenanceWindowsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.MaintenanceWindow, error)
	ListActiveMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time) ([]*models.MaintenanceWindow, error)
	ListUpcomingMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time, withinMinutes int) ([]*models.MaintenanceWindow, error)
	CreateMaintenanceWindow(ctx context.Context, m *models.MaintenanceWindow) error
	UpdateMaintenanceWindow(ctx context.Context, m *models.MaintenanceWindow) error
	DeleteMaintenanceWindow(ctx context.Context, id uuid.UUID) error
}

// MaintenanceHandler handles maintenance window HTTP endpoints.
type MaintenanceHandler struct {
	store  MaintenanceStore
	logger zerolog.Logger
}

// NewMaintenanceHandler creates a new MaintenanceHandler.
func NewMaintenanceHandler(store MaintenanceStore, logger zerolog.Logger) *MaintenanceHandler {
	return &MaintenanceHandler{
		store:  store,
		logger: logger.With().Str("component", "maintenance_handler").Logger(),
	}
}

// RegisterRoutes registers maintenance routes on the given router group.
func (h *MaintenanceHandler) RegisterRoutes(r *gin.RouterGroup) {
	windows := r.Group("/maintenance-windows")
	{
		windows.GET("", h.List)
		windows.POST("", h.Create)
		windows.GET("/:id", h.Get)
		windows.PUT("/:id", h.Update)
		windows.DELETE("/:id", h.Delete)
	}

	// Active maintenance endpoint (for banner display)
	r.GET("/maintenance/active", h.GetActive)
}

// isAdmin checks if the user has admin or owner role.
func isAdmin(role string) bool {
	return role == string(models.OrgRoleAdmin) || role == string(models.OrgRoleOwner)
}

// List returns all maintenance windows for the organization.
// GET /api/v1/maintenance-windows
func (h *MaintenanceHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	windows, err := h.store.ListMaintenanceWindowsByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list maintenance windows")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list maintenance windows"})
		return
	}

	c.JSON(http.StatusOK, models.MaintenanceWindowsResponse{MaintenanceWindows: toWindowSlice(windows)})
}

// Get returns a specific maintenance window by ID.
// GET /api/v1/maintenance-windows/:id
func (h *MaintenanceHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance window ID"})
		return
	}

	window, err := h.store.GetMaintenanceWindowByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	// Verify the window belongs to the user's org
	if window.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	c.JSON(http.StatusOK, window)
}

// Create creates a new maintenance window.
// POST /api/v1/maintenance-windows
func (h *MaintenanceHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can create maintenance windows
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateMaintenanceWindowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that ends_at > starts_at
	if !req.EndsAt.After(req.StartsAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be after start time"})
		return
	}

	window := models.NewMaintenanceWindow(user.CurrentOrgID, req.Title, req.StartsAt, req.EndsAt)
	window.Message = req.Message
	window.CreatedBy = &user.ID
	if req.NotifyBeforeMinutes != nil {
		window.NotifyBeforeMinutes = *req.NotifyBeforeMinutes
	}

	if err := h.store.CreateMaintenanceWindow(c.Request.Context(), window); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create maintenance window")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create maintenance window"})
		return
	}

	h.logger.Info().
		Str("window_id", window.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("title", window.Title).
		Time("starts_at", window.StartsAt).
		Time("ends_at", window.EndsAt).
		Msg("maintenance window created")

	c.JSON(http.StatusCreated, window)
}

// Update updates an existing maintenance window.
// PUT /api/v1/maintenance-windows/:id
func (h *MaintenanceHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can update maintenance windows
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance window ID"})
		return
	}

	window, err := h.store.GetMaintenanceWindowByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	// Verify the window belongs to the user's org
	if window.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	var req models.UpdateMaintenanceWindowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.Title != nil {
		window.Title = *req.Title
	}
	if req.Message != nil {
		window.Message = *req.Message
	}
	if req.StartsAt != nil {
		window.StartsAt = *req.StartsAt
	}
	if req.EndsAt != nil {
		window.EndsAt = *req.EndsAt
	}
	if req.NotifyBeforeMinutes != nil {
		window.NotifyBeforeMinutes = *req.NotifyBeforeMinutes
		// Reset notification sent flag if the notify time changed
		window.NotificationSent = false
	}

	// Validate that ends_at > starts_at
	if !window.EndsAt.After(window.StartsAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be after start time"})
		return
	}

	if err := h.store.UpdateMaintenanceWindow(c.Request.Context(), window); err != nil {
		h.logger.Error().Err(err).Str("window_id", id.String()).Msg("failed to update maintenance window")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update maintenance window"})
		return
	}

	h.logger.Info().
		Str("window_id", window.ID.String()).
		Str("title", window.Title).
		Msg("maintenance window updated")

	c.JSON(http.StatusOK, window)
}

// Delete deletes a maintenance window.
// DELETE /api/v1/maintenance-windows/:id
func (h *MaintenanceHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can delete maintenance windows
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maintenance window ID"})
		return
	}

	window, err := h.store.GetMaintenanceWindowByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	// Verify the window belongs to the user's org
	if window.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "maintenance window not found"})
		return
	}

	if err := h.store.DeleteMaintenanceWindow(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("window_id", id.String()).Msg("failed to delete maintenance window")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete maintenance window"})
		return
	}

	h.logger.Info().
		Str("window_id", id.String()).
		Str("title", window.Title).
		Msg("maintenance window deleted")

	c.JSON(http.StatusOK, gin.H{"message": "maintenance window deleted"})
}

// GetActive returns the currently active or upcoming maintenance window.
// GET /api/v1/maintenance/active
func (h *MaintenanceHandler) GetActive(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	now := time.Now()
	response := models.ActiveMaintenanceResponse{}

	// Get active windows
	activeWindows, err := h.store.ListActiveMaintenanceWindows(c.Request.Context(), user.CurrentOrgID, now)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get active maintenance windows")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active maintenance"})
		return
	}
	if len(activeWindows) > 0 {
		response.Active = activeWindows[0]
	}

	// Get upcoming windows (within notify threshold of any window, max 120 minutes)
	upcomingWindows, err := h.store.ListUpcomingMaintenanceWindows(c.Request.Context(), user.CurrentOrgID, now, 120)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get upcoming maintenance windows")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get upcoming maintenance"})
		return
	}
	if len(upcomingWindows) > 0 {
		response.Upcoming = upcomingWindows[0]
	}

	c.JSON(http.StatusOK, response)
}

// toWindowSlice converts []*models.MaintenanceWindow to []models.MaintenanceWindow.
func toWindowSlice(windows []*models.MaintenanceWindow) []models.MaintenanceWindow {
	if windows == nil {
		return []models.MaintenanceWindow{}
	}
	result := make([]models.MaintenanceWindow, len(windows))
	for i, w := range windows {
		result[i] = *w
	}
	return result
}
