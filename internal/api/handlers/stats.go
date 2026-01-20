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

// StatsStore defines the interface for storage stats persistence operations.
type StatsStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetLatestStorageStats(ctx context.Context, repositoryID uuid.UUID) (*models.StorageStats, error)
	GetStorageStatsByRepositoryID(ctx context.Context, repositoryID uuid.UUID, limit int) ([]*models.StorageStats, error)
	GetStorageStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageStatsSummary, error)
	GetStorageGrowth(ctx context.Context, repositoryID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error)
	GetAllStorageGrowth(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error)
	GetLatestStatsForAllRepos(ctx context.Context, orgID uuid.UUID) ([]*models.StorageStats, error)
}

// StatsHandler handles storage stats related HTTP endpoints.
type StatsHandler struct {
	store  StatsStore
	logger zerolog.Logger
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(store StatsStore, logger zerolog.Logger) *StatsHandler {
	return &StatsHandler{
		store:  store,
		logger: logger.With().Str("component", "stats_handler").Logger(),
	}
}

// RegisterRoutes registers storage stats routes on the given router group.
func (h *StatsHandler) RegisterRoutes(r *gin.RouterGroup) {
	stats := r.Group("/stats")
	{
		stats.GET("/summary", h.GetSummary)
		stats.GET("/growth", h.GetGrowth)
		stats.GET("/repositories", h.ListRepositoryStats)
		stats.GET("/repositories/:id", h.GetRepositoryStats)
		stats.GET("/repositories/:id/growth", h.GetRepositoryGrowth)
		stats.GET("/repositories/:id/history", h.GetRepositoryHistory)
	}
}

// GetSummary returns aggregated storage statistics for the organization.
// GET /api/v1/stats/summary
func (h *StatsHandler) GetSummary(c *gin.Context) {
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

	summary, err := h.store.GetStorageStatsSummary(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get stats summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve stats summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetGrowth returns storage growth data for all repositories in the organization.
// GET /api/v1/stats/growth?days=30
func (h *StatsHandler) GetGrowth(c *gin.Context) {
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

	growth, err := h.store.GetAllStorageGrowth(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage growth")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage growth"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"growth": growth})
}

// ListRepositoryStats returns the latest storage stats for all repositories.
// GET /api/v1/stats/repositories
func (h *StatsHandler) ListRepositoryStats(c *gin.Context) {
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

	stats, err := h.store.GetLatestStatsForAllRepos(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get repository stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve repository stats"})
		return
	}

	// Get repository names for enriched response
	repos, err := h.store.GetRepositoriesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get repositories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve repositories"})
		return
	}

	// Create a map of repository IDs to names
	repoNames := make(map[uuid.UUID]string)
	for _, repo := range repos {
		repoNames[repo.ID] = repo.Name
	}

	// Enrich stats with repository names
	type enrichedStats struct {
		*models.StorageStats
		RepositoryName string `json:"repository_name"`
	}

	enriched := make([]enrichedStats, 0, len(stats))
	for _, s := range stats {
		enriched = append(enriched, enrichedStats{
			StorageStats:   s,
			RepositoryName: repoNames[s.RepositoryID],
		})
	}

	c.JSON(http.StatusOK, gin.H{"stats": enriched})
}

// GetRepositoryStats returns the latest storage stats for a specific repository.
// GET /api/v1/stats/repositories/:id
func (h *StatsHandler) GetRepositoryStats(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify repository belongs to user's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	stats, err := h.store.GetLatestStorageStats(c.Request.Context(), repoID)
	if err != nil {
		// No stats yet is not an error
		c.JSON(http.StatusOK, gin.H{
			"repository_id":   repoID,
			"repository_name": repo.Name,
			"message":         "no stats collected yet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":           stats,
		"repository_name": repo.Name,
	})
}

// GetRepositoryGrowth returns storage growth data for a specific repository.
// GET /api/v1/stats/repositories/:id/growth?days=30
func (h *StatsHandler) GetRepositoryGrowth(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify repository belongs to user's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	growth, err := h.store.GetStorageGrowth(c.Request.Context(), repoID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to get storage growth")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage growth"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repository_id":   repoID,
		"repository_name": repo.Name,
		"growth":          growth,
	})
}

// GetRepositoryHistory returns historical storage stats for a specific repository.
// GET /api/v1/stats/repositories/:id/history?limit=30
func (h *StatsHandler) GetRepositoryHistory(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify repository belongs to user's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	limit := 30
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 365 {
			limit = l
		}
	}

	history, err := h.store.GetStorageStatsByRepositoryID(c.Request.Context(), repoID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to get storage history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repository_id":   repoID,
		"repository_name": repo.Name,
		"history":         history,
	})
}
