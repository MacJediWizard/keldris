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

// FavoriteStore defines the interface for favorite persistence operations.
type FavoriteStore interface {
	GetFavoritesByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]*models.Favorite, error)
	GetFavoriteByUserAndEntity(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) (*models.Favorite, error)
	CreateFavorite(ctx context.Context, f *models.Favorite) error
	DeleteFavorite(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) error
	IsFavorite(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) (bool, error)
	GetFavoriteEntityIDs(ctx context.Context, userID, orgID uuid.UUID, entityType string) ([]uuid.UUID, error)
	// Entity existence checks
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
}

// FavoritesHandler handles favorite HTTP endpoints.
type FavoritesHandler struct {
	store  FavoriteStore
	logger zerolog.Logger
}

// NewFavoritesHandler creates a new FavoritesHandler.
func NewFavoritesHandler(store FavoriteStore, logger zerolog.Logger) *FavoritesHandler {
	return &FavoritesHandler{
		store:  store,
		logger: logger.With().Str("component", "favorites_handler").Logger(),
	}
}

// RegisterRoutes registers favorite routes on the given router group.
func (h *FavoritesHandler) RegisterRoutes(r *gin.RouterGroup) {
	favorites := r.Group("/favorites")
	{
		favorites.GET("", h.List)
		favorites.POST("", h.Create)
		favorites.DELETE("/:type/:id", h.Delete)
	}
}

// List returns all favorites for the authenticated user's organization.
// GET /api/v1/favorites?entity_type=
func (h *FavoritesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	entityType := c.Query("entity_type")

	favorites, err := h.store.GetFavoritesByUserAndOrg(c.Request.Context(), user.ID, user.CurrentOrgID, entityType)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list favorites")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list favorites"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": favorites})
}

// Create creates a new favorite.
// POST /api/v1/favorites
func (h *FavoritesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity ID"})
		return
	}

	// Validate entity type
	if !isValidEntityType(req.EntityType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity type, must be one of: agent, schedule, repository"})
		return
	}

	// Verify the entity exists and belongs to the user's org
	if err := h.verifyEntityAccess(c.Request.Context(), req.EntityType, entityID, user.CurrentOrgID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
		return
	}

	// Check if already favorited
	exists, err := h.store.IsFavorite(c.Request.Context(), user.ID, string(req.EntityType), entityID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to check if favorite exists")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create favorite"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "already favorited"})
		return
	}

	favorite := models.NewFavorite(user.ID, user.CurrentOrgID, entityID, req.EntityType)

	if err := h.store.CreateFavorite(c.Request.Context(), favorite); err != nil {
		h.logger.Error().Err(err).Str("entity_type", string(req.EntityType)).Str("entity_id", entityID.String()).Msg("failed to create favorite")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create favorite"})
		return
	}

	h.logger.Info().Str("favorite_id", favorite.ID.String()).Str("entity_type", string(req.EntityType)).Msg("favorite created")
	c.JSON(http.StatusCreated, favorite)
}

// Delete deletes a favorite.
// DELETE /api/v1/favorites/:type/:id
func (h *FavoritesHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	entityType := c.Param("type")
	entityIDParam := c.Param("id")

	entityID, err := uuid.Parse(entityIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity ID"})
		return
	}

	// Validate entity type
	if !isValidEntityType(models.FavoriteEntityType(entityType)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity type"})
		return
	}

	if err := h.store.DeleteFavorite(c.Request.Context(), user.ID, entityType, entityID); err != nil {
		h.logger.Error().Err(err).Str("entity_type", entityType).Str("entity_id", entityID.String()).Msg("failed to delete favorite")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete favorite"})
		return
	}

	h.logger.Info().Str("entity_type", entityType).Str("entity_id", entityID.String()).Msg("favorite deleted")
	c.JSON(http.StatusOK, gin.H{"message": "favorite deleted"})
}

// isValidEntityType checks if the entity type is valid.
func isValidEntityType(entityType models.FavoriteEntityType) bool {
	switch entityType {
	case models.FavoriteEntityTypeAgent,
		models.FavoriteEntityTypeSchedule,
		models.FavoriteEntityTypeRepository:
		return true
	}
	return false
}

// verifyEntityAccess verifies that an entity exists and belongs to the specified org.
func (h *FavoritesHandler) verifyEntityAccess(ctx context.Context, entityType models.FavoriteEntityType, entityID, orgID uuid.UUID) error {
	switch entityType {
	case models.FavoriteEntityTypeAgent:
		agent, err := h.store.GetAgentByID(ctx, entityID)
		if err != nil {
			return err
		}
		if agent.OrgID != orgID {
			return err
		}
	case models.FavoriteEntityTypeSchedule:
		schedule, err := h.store.GetScheduleByID(ctx, entityID)
		if err != nil {
			return err
		}
		// Schedules need to check via agent's org
		agent, err := h.store.GetAgentByID(ctx, schedule.AgentID)
		if err != nil {
			return err
		}
		if agent.OrgID != orgID {
			return err
		}
	case models.FavoriteEntityTypeRepository:
		repo, err := h.store.GetRepositoryByID(ctx, entityID)
		if err != nil {
			return err
		}
		if repo.OrgID != orgID {
			return err
		}
	}
	return nil
}
