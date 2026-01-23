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

// AnnouncementStore defines the interface for announcement persistence operations.
type AnnouncementStore interface {
	GetAnnouncementByID(ctx context.Context, id uuid.UUID) (*models.Announcement, error)
	ListAnnouncementsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Announcement, error)
	ListActiveAnnouncements(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, now time.Time) ([]*models.Announcement, error)
	CreateAnnouncement(ctx context.Context, a *models.Announcement) error
	UpdateAnnouncement(ctx context.Context, a *models.Announcement) error
	DeleteAnnouncement(ctx context.Context, id uuid.UUID) error
	CreateAnnouncementDismissal(ctx context.Context, d *models.AnnouncementDismissal) error
}

// AnnouncementsHandler handles announcement HTTP endpoints.
type AnnouncementsHandler struct {
	store  AnnouncementStore
	logger zerolog.Logger
}

// NewAnnouncementsHandler creates a new AnnouncementsHandler.
func NewAnnouncementsHandler(store AnnouncementStore, logger zerolog.Logger) *AnnouncementsHandler {
	return &AnnouncementsHandler{
		store:  store,
		logger: logger.With().Str("component", "announcements_handler").Logger(),
	}
}

// RegisterRoutes registers announcement routes on the given router group.
func (h *AnnouncementsHandler) RegisterRoutes(r *gin.RouterGroup) {
	announcements := r.Group("/announcements")
	{
		announcements.GET("", h.List)
		announcements.GET("/active", h.GetActive)
		announcements.POST("", h.Create)
		announcements.GET("/:id", h.Get)
		announcements.PUT("/:id", h.Update)
		announcements.DELETE("/:id", h.Delete)
		announcements.POST("/:id/dismiss", h.Dismiss)
	}
}

// List returns all announcements for the organization.
// GET /api/v1/announcements
func (h *AnnouncementsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only admins can list all announcements
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	announcements, err := h.store.ListAnnouncementsByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list announcements")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list announcements"})
		return
	}

	c.JSON(http.StatusOK, models.AnnouncementsResponse{Announcements: toAnnouncementSlice(announcements)})
}

// GetActive returns active announcements that the user hasn't dismissed.
// GET /api/v1/announcements/active
func (h *AnnouncementsHandler) GetActive(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	now := time.Now()
	announcements, err := h.store.ListActiveAnnouncements(c.Request.Context(), user.CurrentOrgID, user.ID, now)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list active announcements")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list active announcements"})
		return
	}

	c.JSON(http.StatusOK, models.AnnouncementsResponse{Announcements: toAnnouncementSlice(announcements)})
}

// Get returns a specific announcement by ID.
// GET /api/v1/announcements/:id
func (h *AnnouncementsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement ID"})
		return
	}

	announcement, err := h.store.GetAnnouncementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	// Verify the announcement belongs to the user's org
	if announcement.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	c.JSON(http.StatusOK, announcement)
}

// Create creates a new announcement.
// POST /api/v1/announcements
func (h *AnnouncementsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can create announcements
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate schedule times if both are provided
	if req.StartsAt != nil && req.EndsAt != nil {
		if !req.EndsAt.After(*req.StartsAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be after start time"})
			return
		}
	}

	announcement := models.NewAnnouncement(user.CurrentOrgID, req.Title, req.Type)
	announcement.Message = req.Message
	announcement.CreatedBy = &user.ID

	if req.Dismissible != nil {
		announcement.Dismissible = *req.Dismissible
	}
	if req.StartsAt != nil {
		announcement.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		announcement.EndsAt = req.EndsAt
	}
	if req.Active != nil {
		announcement.Active = *req.Active
	}

	if err := h.store.CreateAnnouncement(c.Request.Context(), announcement); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create announcement")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create announcement"})
		return
	}

	h.logger.Info().
		Str("announcement_id", announcement.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("title", announcement.Title).
		Str("type", string(announcement.Type)).
		Msg("announcement created")

	c.JSON(http.StatusCreated, announcement)
}

// Update updates an existing announcement.
// PUT /api/v1/announcements/:id
func (h *AnnouncementsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can update announcements
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement ID"})
		return
	}

	announcement, err := h.store.GetAnnouncementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	// Verify the announcement belongs to the user's org
	if announcement.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	var req models.UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.Title != nil {
		announcement.Title = *req.Title
	}
	if req.Message != nil {
		announcement.Message = *req.Message
	}
	if req.Type != nil {
		announcement.Type = *req.Type
	}
	if req.Dismissible != nil {
		announcement.Dismissible = *req.Dismissible
	}
	if req.StartsAt != nil {
		announcement.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		announcement.EndsAt = req.EndsAt
	}
	if req.Active != nil {
		announcement.Active = *req.Active
	}

	// Validate schedule times if both are set
	if announcement.StartsAt != nil && announcement.EndsAt != nil {
		if !announcement.EndsAt.After(*announcement.StartsAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end time must be after start time"})
			return
		}
	}

	if err := h.store.UpdateAnnouncement(c.Request.Context(), announcement); err != nil {
		h.logger.Error().Err(err).Str("announcement_id", id.String()).Msg("failed to update announcement")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update announcement"})
		return
	}

	h.logger.Info().
		Str("announcement_id", announcement.ID.String()).
		Str("title", announcement.Title).
		Msg("announcement updated")

	c.JSON(http.StatusOK, announcement)
}

// Delete deletes an announcement.
// DELETE /api/v1/announcements/:id
func (h *AnnouncementsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can delete announcements
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement ID"})
		return
	}

	announcement, err := h.store.GetAnnouncementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	// Verify the announcement belongs to the user's org
	if announcement.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	if err := h.store.DeleteAnnouncement(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("announcement_id", id.String()).Msg("failed to delete announcement")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete announcement"})
		return
	}

	h.logger.Info().
		Str("announcement_id", id.String()).
		Str("title", announcement.Title).
		Msg("announcement deleted")

	c.JSON(http.StatusOK, gin.H{"message": "announcement deleted"})
}

// Dismiss marks an announcement as dismissed for the current user.
// POST /api/v1/announcements/:id/dismiss
func (h *AnnouncementsHandler) Dismiss(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement ID"})
		return
	}

	announcement, err := h.store.GetAnnouncementByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	// Verify the announcement belongs to the user's org
	if announcement.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "announcement not found"})
		return
	}

	// Check if the announcement is dismissible
	if !announcement.Dismissible {
		c.JSON(http.StatusBadRequest, gin.H{"error": "announcement cannot be dismissed"})
		return
	}

	dismissal := models.NewAnnouncementDismissal(user.CurrentOrgID, id, user.ID)
	if err := h.store.CreateAnnouncementDismissal(c.Request.Context(), dismissal); err != nil {
		h.logger.Error().Err(err).
			Str("announcement_id", id.String()).
			Str("user_id", user.ID.String()).
			Msg("failed to dismiss announcement")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dismiss announcement"})
		return
	}

	h.logger.Info().
		Str("announcement_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("announcement dismissed")

	c.JSON(http.StatusOK, gin.H{"message": "announcement dismissed"})
}

// toAnnouncementSlice converts []*models.Announcement to []models.Announcement.
func toAnnouncementSlice(announcements []*models.Announcement) []models.Announcement {
	if announcements == nil {
		return []models.Announcement{}
	}
	result := make([]models.Announcement, len(announcements))
	for i, a := range announcements {
		result[i] = *a
	}
	return result
}
