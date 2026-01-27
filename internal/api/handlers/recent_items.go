package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RecentItemsHandler handles recent items HTTP endpoints.
type RecentItemsHandler struct {
	store  RecentItemsStore
	logger zerolog.Logger
}

// RecentItemsStore defines the interface for recent items persistence operations.
type RecentItemsStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateOrUpdateRecentItem(ctx context.Context, item *models.RecentItem) error
	GetRecentItemsByUser(ctx context.Context, orgID, userID uuid.UUID, limit int) ([]*models.RecentItem, error)
	GetRecentItemsByUserAndType(ctx context.Context, orgID, userID uuid.UUID, itemType models.RecentItemType, limit int) ([]*models.RecentItem, error)
	DeleteRecentItem(ctx context.Context, id uuid.UUID) error
	DeleteRecentItemsForUser(ctx context.Context, orgID, userID uuid.UUID) error
	GetRecentItemByID(ctx context.Context, id uuid.UUID) (*models.RecentItem, error)
}

// NewRecentItemsHandler creates a new RecentItemsHandler.
func NewRecentItemsHandler(store RecentItemsStore, logger zerolog.Logger) *RecentItemsHandler {
	return &RecentItemsHandler{
		store:  store,
		logger: logger.With().Str("component", "recent_items_handler").Logger(),
	}
}

// RegisterRoutes registers recent items routes on the given router group.
func (h *RecentItemsHandler) RegisterRoutes(r *gin.RouterGroup) {
	recent := r.Group("/recent-items")
	{
		recent.GET("", h.List)
		recent.POST("", h.Track)
		recent.DELETE("", h.ClearAll)
		recent.DELETE("/:id", h.Delete)
	}
}

// List returns recent items for the authenticated user.
func (h *RecentItemsHandler) List(c *gin.Context) {
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

	// Parse optional query parameters
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	itemType := c.Query("type")

	var items []*models.RecentItem
	if itemType != "" {
		if !models.IsValidItemType(itemType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item type"})
			return
		}
		items, err = h.store.GetRecentItemsByUserAndType(c.Request.Context(), dbUser.OrgID, dbUser.ID, models.RecentItemType(itemType), limit)
	} else {
		items, err = h.store.GetRecentItemsByUser(c.Request.Context(), dbUser.OrgID, dbUser.ID, limit)
	}

	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to list recent items")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recent items"})
		return
	}

	if items == nil {
		items = []*models.RecentItem{}
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// TrackRequest is the request body for tracking a recently viewed item.
type TrackRequest struct {
	ItemType string `json:"item_type" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	ItemName string `json:"item_name" binding:"required"`
	PagePath string `json:"page_path" binding:"required"`
}

// Track creates or updates a recent item entry.
func (h *RecentItemsHandler) Track(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req TrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate item type
	if !models.IsValidItemType(req.ItemType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item type"})
		return
	}

	// Parse item ID
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	item := models.NewRecentItem(dbUser.OrgID, dbUser.ID, models.RecentItemType(req.ItemType), itemID, req.ItemName, req.PagePath)

	if err := h.store.CreateOrUpdateRecentItem(c.Request.Context(), item); err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to track recent item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to track recent item"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// Delete removes a specific recent item.
func (h *RecentItemsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	// Verify the item exists and belongs to this user
	item, err := h.store.GetRecentItemByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("item_id", id.String()).Msg("failed to get recent item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recent item"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "recent item not found"})
		return
	}

	// Verify user owns this item
	if item.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "recent item not found"})
		return
	}

	if err := h.store.DeleteRecentItem(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("item_id", id.String()).Msg("failed to delete recent item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recent item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recent item deleted"})
}

// ClearAll removes all recent items for the authenticated user.
func (h *RecentItemsHandler) ClearAll(c *gin.Context) {
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

	if err := h.store.DeleteRecentItemsForUser(c.Request.Context(), dbUser.OrgID, dbUser.ID); err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to clear recent items")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear recent items"})
		return
	}

	h.logger.Info().Str("user_id", user.ID.String()).Msg("cleared all recent items")

	c.JSON(http.StatusOK, gin.H{"message": "recent items cleared"})
}
