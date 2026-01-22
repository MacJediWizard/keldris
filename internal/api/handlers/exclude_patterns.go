package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/excludes"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ExcludePatternStore defines the interface for exclude pattern persistence operations.
type ExcludePatternStore interface {
	GetExcludePatternsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ExcludePattern, error)
	GetExcludePatternByID(ctx context.Context, id uuid.UUID) (*models.ExcludePattern, error)
	GetExcludePatternsByCategory(ctx context.Context, orgID uuid.UUID, category string) ([]*models.ExcludePattern, error)
	GetBuiltinExcludePatterns(ctx context.Context) ([]*models.ExcludePattern, error)
	CreateExcludePattern(ctx context.Context, pattern *models.ExcludePattern) error
	UpdateExcludePattern(ctx context.Context, pattern *models.ExcludePattern) error
	DeleteExcludePattern(ctx context.Context, id uuid.UUID) error
	SeedBuiltinExcludePatterns(ctx context.Context, patterns []*models.ExcludePattern) error
}

// ExcludePatternsHandler handles exclude pattern-related HTTP endpoints.
type ExcludePatternsHandler struct {
	store  ExcludePatternStore
	logger zerolog.Logger
}

// NewExcludePatternsHandler creates a new ExcludePatternsHandler.
func NewExcludePatternsHandler(store ExcludePatternStore, logger zerolog.Logger) *ExcludePatternsHandler {
	return &ExcludePatternsHandler{
		store:  store,
		logger: logger.With().Str("component", "exclude_patterns_handler").Logger(),
	}
}

// RegisterRoutes registers exclude pattern routes on the given router group.
func (h *ExcludePatternsHandler) RegisterRoutes(r *gin.RouterGroup) {
	patterns := r.Group("/exclude-patterns")
	{
		patterns.GET("", h.List)
		patterns.GET("/library", h.Library)
		patterns.GET("/categories", h.Categories)
		patterns.POST("", h.Create)
		patterns.GET("/:id", h.Get)
		patterns.PUT("/:id", h.Update)
		patterns.DELETE("/:id", h.Delete)
	}
}

// CreateExcludePatternRequest is the request body for creating an exclude pattern.
type CreateExcludePatternRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=255"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns" binding:"required,min=1"`
	Category    string   `json:"category" binding:"required"`
}

// UpdateExcludePatternRequest is the request body for updating an exclude pattern.
type UpdateExcludePatternRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Patterns    []string `json:"patterns,omitempty"`
	Category    *string  `json:"category,omitempty"`
}

// List returns all exclude patterns (built-in and custom) for the user's organization.
// GET /api/v1/exclude-patterns
// Optional query param: category to filter by category
func (h *ExcludePatternsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Optional category filter
	category := c.Query("category")
	if category != "" {
		patterns, err := h.store.GetExcludePatternsByCategory(c.Request.Context(), user.CurrentOrgID, category)
		if err != nil {
			h.logger.Error().Err(err).Str("category", category).Msg("failed to list exclude patterns by category")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list exclude patterns"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"patterns": patterns})
		return
	}

	patterns, err := h.store.GetExcludePatternsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list exclude patterns")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list exclude patterns"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}

// Library returns the built-in exclude patterns library without requiring authentication.
// GET /api/v1/exclude-patterns/library
func (h *ExcludePatternsHandler) Library(c *gin.Context) {
	// Return the in-memory library (no database required)
	c.JSON(http.StatusOK, gin.H{"patterns": excludes.Library})
}

// Categories returns all available pattern categories with metadata.
// GET /api/v1/exclude-patterns/categories
func (h *ExcludePatternsHandler) Categories(c *gin.Context) {
	categories := make([]map[string]interface{}, 0, len(excludes.Categories))
	for cat, info := range excludes.Categories {
		categories = append(categories, map[string]interface{}{
			"id":          string(cat),
			"name":        info.Name,
			"description": info.Description,
			"icon":        info.Icon,
		})
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// Get returns a specific exclude pattern by ID.
// GET /api/v1/exclude-patterns/:id
func (h *ExcludePatternsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	pattern, err := h.store.GetExcludePatternByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	// Verify access: must be built-in or belong to user's org
	if !pattern.IsBuiltin && (pattern.OrgID == nil || *pattern.OrgID != user.CurrentOrgID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	c.JSON(http.StatusOK, pattern)
}

// Create creates a new custom exclude pattern.
// POST /api/v1/exclude-patterns
func (h *ExcludePatternsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateExcludePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate category
	validCategory := false
	for _, cat := range excludes.GetAllCategories() {
		if string(cat) == req.Category {
			validCategory = true
			break
		}
	}
	if !validCategory {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}

	pattern := models.NewExcludePattern(
		user.CurrentOrgID,
		req.Name,
		req.Description,
		req.Category,
		req.Patterns,
	)

	if err := h.store.CreateExcludePattern(c.Request.Context(), pattern); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create exclude pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exclude pattern"})
		return
	}

	h.logger.Info().
		Str("pattern_id", pattern.ID.String()).
		Str("name", pattern.Name).
		Str("org_id", user.CurrentOrgID.String()).
		Msg("created exclude pattern")

	c.JSON(http.StatusCreated, pattern)
}

// Update updates an existing custom exclude pattern.
// PUT /api/v1/exclude-patterns/:id
func (h *ExcludePatternsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	pattern, err := h.store.GetExcludePatternByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	// Cannot update built-in patterns
	if pattern.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify built-in patterns"})
		return
	}

	// Verify ownership
	if pattern.OrgID == nil || *pattern.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	var req UpdateExcludePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.Name != nil {
		pattern.Name = *req.Name
	}
	if req.Description != nil {
		pattern.Description = *req.Description
	}
	if req.Patterns != nil {
		pattern.Patterns = req.Patterns
	}
	if req.Category != nil {
		// Validate category
		validCategory := false
		for _, cat := range excludes.GetAllCategories() {
			if string(cat) == *req.Category {
				validCategory = true
				break
			}
		}
		if !validCategory {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
			return
		}
		pattern.Category = *req.Category
	}

	if err := h.store.UpdateExcludePattern(c.Request.Context(), pattern); err != nil {
		h.logger.Error().Err(err).Str("pattern_id", id.String()).Msg("failed to update exclude pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update exclude pattern"})
		return
	}

	h.logger.Info().
		Str("pattern_id", pattern.ID.String()).
		Str("name", pattern.Name).
		Msg("updated exclude pattern")

	c.JSON(http.StatusOK, pattern)
}

// Delete deletes a custom exclude pattern.
// DELETE /api/v1/exclude-patterns/:id
func (h *ExcludePatternsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	// Verify ownership and that it's not built-in
	pattern, err := h.store.GetExcludePatternByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	if pattern.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete built-in patterns"})
		return
	}

	if pattern.OrgID == nil || *pattern.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "exclude pattern not found"})
		return
	}

	if err := h.store.DeleteExcludePattern(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("pattern_id", id.String()).Msg("failed to delete exclude pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete exclude pattern"})
		return
	}

	h.logger.Info().
		Str("pattern_id", id.String()).
		Msg("deleted exclude pattern")

	c.JSON(http.StatusOK, gin.H{"message": "exclude pattern deleted"})
}
