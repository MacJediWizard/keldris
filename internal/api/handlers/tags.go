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

// TagStore defines the interface for tag persistence operations.
type TagStore interface {
	GetTagsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Tag, error)
	GetTagByID(ctx context.Context, id uuid.UUID) (*models.Tag, error)
	CreateTag(ctx context.Context, tag *models.Tag) error
	UpdateTag(ctx context.Context, tag *models.Tag) error
	DeleteTag(ctx context.Context, id uuid.UUID) error
	GetTagsByBackupID(ctx context.Context, backupID uuid.UUID) ([]*models.Tag, error)
	SetBackupTags(ctx context.Context, backupID uuid.UUID, tagIDs []uuid.UUID) error
	GetBackupByID(ctx context.Context, id uuid.UUID) (*models.Backup, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// TagsHandler handles tag-related HTTP endpoints.
type TagsHandler struct {
	store  TagStore
	logger zerolog.Logger
}

// NewTagsHandler creates a new TagsHandler.
func NewTagsHandler(store TagStore, logger zerolog.Logger) *TagsHandler {
	return &TagsHandler{
		store:  store,
		logger: logger.With().Str("component", "tags_handler").Logger(),
	}
}

// RegisterRoutes registers tag routes on the given router group.
func (h *TagsHandler) RegisterRoutes(r *gin.RouterGroup) {
	tags := r.Group("/tags")
	{
		tags.GET("", h.List)
		tags.POST("", h.Create)
		tags.GET("/:id", h.Get)
		tags.PUT("/:id", h.Update)
		tags.DELETE("/:id", h.Delete)
	}

	// Backup tag associations
	r.GET("/backups/:id/tags", h.GetBackupTags)
	r.POST("/backups/:id/tags", h.SetBackupTags)
}

// List returns all tags for the authenticated user's organization.
// GET /api/v1/tags
func (h *TagsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	tags, err := h.store.GetTagsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list tags")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// Get returns a specific tag by ID.
// GET /api/v1/tags/:id
func (h *TagsHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
		return
	}

	tag, err := h.store.GetTagByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("tag_id", id.String()).Msg("failed to get tag")
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	// Verify tag belongs to user's org
	if tag.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	c.JSON(http.StatusOK, tag)
}

// Create creates a new tag.
// POST /api/v1/tags
func (h *TagsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag := models.NewTag(user.CurrentOrgID, req.Name, req.Color)

	if err := h.store.CreateTag(c.Request.Context(), tag); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create tag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tag"})
		return
	}

	h.logger.Info().Str("tag_id", tag.ID.String()).Str("name", tag.Name).Msg("tag created")
	c.JSON(http.StatusCreated, tag)
}

// Update updates an existing tag.
// PUT /api/v1/tags/:id
func (h *TagsHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
		return
	}

	tag, err := h.store.GetTagByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	// Verify tag belongs to user's org
	if tag.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	var req models.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		tag.Name = *req.Name
	}
	if req.Color != nil {
		tag.Color = *req.Color
	}

	if err := h.store.UpdateTag(c.Request.Context(), tag); err != nil {
		h.logger.Error().Err(err).Str("tag_id", id.String()).Msg("failed to update tag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tag"})
		return
	}

	h.logger.Info().Str("tag_id", tag.ID.String()).Msg("tag updated")
	c.JSON(http.StatusOK, tag)
}

// Delete deletes a tag.
// DELETE /api/v1/tags/:id
func (h *TagsHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
		return
	}

	tag, err := h.store.GetTagByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	// Verify tag belongs to user's org
	if tag.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	if err := h.store.DeleteTag(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("tag_id", id.String()).Msg("failed to delete tag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tag"})
		return
	}

	h.logger.Info().Str("tag_id", id.String()).Msg("tag deleted")
	c.JSON(http.StatusOK, gin.H{"message": "tag deleted"})
}

// GetBackupTags returns all tags for a specific backup.
// GET /api/v1/backups/:id/tags
func (h *TagsHandler) GetBackupTags(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	backupID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	// Verify backup exists and belongs to user's org
	backup, err := h.store.GetBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	tags, err := h.store.GetTagsByBackupID(c.Request.Context(), backupID)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to get backup tags")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get backup tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// SetBackupTags sets the tags for a specific backup.
// POST /api/v1/backups/:id/tags
func (h *TagsHandler) SetBackupTags(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	backupID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	// Verify backup exists and belongs to user's org
	backup, err := h.store.GetBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	var req models.AssignTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify all tags belong to user's org
	for _, tagID := range req.TagIDs {
		tag, err := h.store.GetTagByID(c.Request.Context(), tagID)
		if err != nil || tag.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
			return
		}
	}

	if err := h.store.SetBackupTags(c.Request.Context(), backupID, req.TagIDs); err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to set backup tags")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set backup tags"})
		return
	}

	// Return the updated tags
	tags, err := h.store.GetTagsByBackupID(c.Request.Context(), backupID)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to get updated backup tags")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated backup tags"})
		return
	}

	h.logger.Info().Str("backup_id", backupID.String()).Int("tag_count", len(req.TagIDs)).Msg("backup tags updated")
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
