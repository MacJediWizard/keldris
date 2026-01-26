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

// SavedFilterStore defines the interface for saved filter persistence operations.
type SavedFilterStore interface {
	GetSavedFiltersByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]*models.SavedFilter, error)
	GetSavedFilterByID(ctx context.Context, id uuid.UUID) (*models.SavedFilter, error)
	GetDefaultSavedFilter(ctx context.Context, userID, orgID uuid.UUID, entityType string) (*models.SavedFilter, error)
	CreateSavedFilter(ctx context.Context, filter *models.SavedFilter) error
	UpdateSavedFilter(ctx context.Context, filter *models.SavedFilter) error
	DeleteSavedFilter(ctx context.Context, id uuid.UUID) error
}

// FiltersHandler handles saved filter HTTP endpoints.
type FiltersHandler struct {
	store  SavedFilterStore
	logger zerolog.Logger
}

// NewFiltersHandler creates a new FiltersHandler.
func NewFiltersHandler(store SavedFilterStore, logger zerolog.Logger) *FiltersHandler {
	return &FiltersHandler{
		store:  store,
		logger: logger.With().Str("component", "filters_handler").Logger(),
	}
}

// RegisterRoutes registers saved filter routes on the given router group.
func (h *FiltersHandler) RegisterRoutes(r *gin.RouterGroup) {
	filters := r.Group("/filters")
	{
		filters.GET("", h.List)
		filters.POST("", h.Create)
		filters.GET("/:id", h.Get)
		filters.PUT("/:id", h.Update)
		filters.DELETE("/:id", h.Delete)
	}
}

// List returns all saved filters for the authenticated user's organization.
// GET /api/v1/filters?entity_type=
func (h *FiltersHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	entityType := c.Query("entity_type")

	filters, err := h.store.GetSavedFiltersByUserAndOrg(c.Request.Context(), user.ID, user.CurrentOrgID, entityType)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list saved filters")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list saved filters"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"filters": filters})
}

// Get returns a specific saved filter by ID.
// GET /api/v1/filters/:id
func (h *FiltersHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filter ID"})
		return
	}

	filter, err := h.store.GetSavedFilterByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("filter_id", id.String()).Msg("failed to get saved filter")
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// Verify filter belongs to user's org and is accessible
	if filter.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// User can only access their own filters or shared filters
	if filter.UserID != user.ID && !filter.Shared {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	c.JSON(http.StatusOK, filter)
}

// Create creates a new saved filter.
// POST /api/v1/filters
func (h *FiltersHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateSavedFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := models.NewSavedFilter(user.ID, user.CurrentOrgID, req.Name, req.EntityType, req.Filters)
	filter.Shared = req.Shared
	filter.IsDefault = req.IsDefault

	if err := h.store.CreateSavedFilter(c.Request.Context(), filter); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create saved filter")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create saved filter"})
		return
	}

	h.logger.Info().Str("filter_id", filter.ID.String()).Str("name", filter.Name).Msg("saved filter created")
	c.JSON(http.StatusCreated, filter)
}

// Update updates an existing saved filter.
// PUT /api/v1/filters/:id
func (h *FiltersHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filter ID"})
		return
	}

	filter, err := h.store.GetSavedFilterByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// Verify filter belongs to user's org
	if filter.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// Only the owner can update their filter
	if filter.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify another user's filter"})
		return
	}

	var req models.UpdateSavedFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		filter.Name = *req.Name
	}
	if req.Filters != nil {
		filter.Filters = *req.Filters
	}
	if req.Shared != nil {
		filter.Shared = *req.Shared
	}
	if req.IsDefault != nil {
		filter.IsDefault = *req.IsDefault
	}

	if err := h.store.UpdateSavedFilter(c.Request.Context(), filter); err != nil {
		h.logger.Error().Err(err).Str("filter_id", id.String()).Msg("failed to update saved filter")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update saved filter"})
		return
	}

	h.logger.Info().Str("filter_id", filter.ID.String()).Msg("saved filter updated")
	c.JSON(http.StatusOK, filter)
}

// Delete deletes a saved filter.
// DELETE /api/v1/filters/:id
func (h *FiltersHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filter ID"})
		return
	}

	filter, err := h.store.GetSavedFilterByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// Verify filter belongs to user's org
	if filter.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	// Only the owner can delete their filter
	if filter.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete another user's filter"})
		return
	}

	if err := h.store.DeleteSavedFilter(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("filter_id", id.String()).Msg("failed to delete saved filter")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete saved filter"})
		return
	}

	h.logger.Info().Str("filter_id", id.String()).Msg("saved filter deleted")
	c.JSON(http.StatusOK, gin.H{"message": "filter deleted"})
}
