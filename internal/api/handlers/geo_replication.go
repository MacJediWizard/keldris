package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// GeoReplicationStore defines the interface for geo-replication persistence.
type GeoReplicationStore interface {
	CreateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error
	GetGeoReplicationConfig(ctx context.Context, id uuid.UUID) (*models.GeoReplicationConfig, error)
	GetGeoReplicationConfigByRepository(ctx context.Context, repositoryID uuid.UUID) (*models.GeoReplicationConfig, error)
	UpdateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error
	DeleteGeoReplicationConfig(ctx context.Context, id uuid.UUID) error
	ListGeoReplicationConfigsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.GeoReplicationConfig, error)
	GetReplicationEvents(ctx context.Context, configID uuid.UUID, limit int) ([]*models.ReplicationEvent, error)
	GetReplicationLagForConfig(ctx context.Context, configID uuid.UUID) (snapshotsBehind int, lastSyncAt *time.Time, err error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	UpdateRepositoryRegion(ctx context.Context, repositoryID uuid.UUID, region string) error
}

// GeoReplicationHandler handles geo-replication API endpoints.
type GeoReplicationHandler struct {
	store  GeoReplicationStore
	logger zerolog.Logger
}

// NewGeoReplicationHandler creates a new GeoReplicationHandler.
func NewGeoReplicationHandler(store GeoReplicationStore, logger zerolog.Logger) *GeoReplicationHandler {
	return &GeoReplicationHandler{
		store:  store,
		logger: logger.With().Str("component", "geo_replication_handler").Logger(),
	}
}

// RegisterRoutes registers geo-replication routes on the given router group.
func (h *GeoReplicationHandler) RegisterRoutes(r *gin.RouterGroup) {
	geo := r.Group("/geo-replication")
	{
		geo.GET("/regions", h.ListRegions)
		geo.GET("/configs", h.List)
		geo.POST("/configs", h.Create)
		geo.GET("/configs/:id", h.Get)
		geo.PUT("/configs/:id", h.Update)
		geo.DELETE("/configs/:id", h.Delete)
		geo.POST("/configs/:id/trigger", h.TriggerReplication)
		geo.GET("/configs/:id/events", h.GetEvents)
		geo.GET("/repositories/:id/status", h.GetRepositoryStatus)
		geo.PUT("/repositories/:id/region", h.SetRepositoryRegion)
		geo.GET("/summary", h.GetSummary)
	}
}

// ListRegions returns all available regions for geo-replication.
//
//	@Summary		List regions
//	@Description	Returns all available geographic regions for backup replication
//	@Tags			Geo-Replication
//	@Produce		json
//	@Success		200	{object}	map[string][]backup.Region
//	@Security		SessionAuth
//	@Router			/geo-replication/regions [get]
func (h *GeoReplicationHandler) ListRegions(c *gin.Context) {
	regions := backup.AllRegions()
	pairs := backup.DefaultRegionPairs()

	c.JSON(http.StatusOK, gin.H{
		"regions": regions,
		"pairs":   pairs,
	})
}

// List returns all geo-replication configs for the organization.
//
//	@Summary		List geo-replication configs
//	@Description	Returns all geo-replication configurations for the current organization
//	@Tags			Geo-Replication
//	@Produce		json
//	@Success		200	{object}	map[string][]models.GeoReplicationConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs [get]
func (h *GeoReplicationHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	configs, err := h.store.ListGeoReplicationConfigsByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list geo-replication configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list geo-replication configs"})
		return
	}

	responses := make([]models.GeoReplicationResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = h.toResponse(c.Request.Context(), cfg)
	}

	c.JSON(http.StatusOK, gin.H{"configs": responses})
}

// Create creates a new geo-replication configuration.
//
//	@Summary		Create geo-replication config
//	@Description	Creates a new geo-replication configuration for automatic backup replication
//	@Tags			Geo-Replication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.GeoReplicationCreateRequest	true	"Config details"
//	@Success		201		{object}	models.GeoReplicationResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs [post]
func (h *GeoReplicationHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.GeoReplicationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate source and target regions
	if _, err := backup.GetRegionByCode(req.SourceRegion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source region"})
		return
	}
	if _, err := backup.GetRegionByCode(req.TargetRegion); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target region"})
		return
	}

	// Validate repositories exist and belong to this org
	sourceRepo, err := h.store.GetRepositoryByID(c.Request.Context(), req.SourceRepositoryID)
	if err != nil || sourceRepo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source repository not found"})
		return
	}

	targetRepo, err := h.store.GetRepositoryByID(c.Request.Context(), req.TargetRepositoryID)
	if err != nil || targetRepo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target repository not found"})
		return
	}

	config := models.NewGeoReplicationConfig(
		user.CurrentOrgID,
		req.SourceRepositoryID,
		req.TargetRepositoryID,
		req.SourceRegion,
		req.TargetRegion,
	)

	// Apply optional settings
	if req.MaxLagSnapshots != nil {
		config.MaxLagSnapshots = *req.MaxLagSnapshots
	}
	if req.MaxLagDurationHours != nil {
		config.MaxLagDurationHours = *req.MaxLagDurationHours
	}
	if req.AlertOnLag != nil {
		config.AlertOnLag = *req.AlertOnLag
	}

	if err := h.store.CreateGeoReplicationConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Msg("failed to create geo-replication config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create geo-replication config"})
		return
	}

	// Update repository regions
	_ = h.store.UpdateRepositoryRegion(c.Request.Context(), req.SourceRepositoryID, req.SourceRegion)
	_ = h.store.UpdateRepositoryRegion(c.Request.Context(), req.TargetRepositoryID, req.TargetRegion)

	h.logger.Info().
		Str("config_id", config.ID.String()).
		Str("source_repo", req.SourceRepositoryID.String()).
		Str("target_repo", req.TargetRepositoryID.String()).
		Msg("geo-replication config created")

	c.JSON(http.StatusCreated, h.toResponse(c.Request.Context(), config))
}

// Get returns a specific geo-replication configuration.
//
//	@Summary		Get geo-replication config
//	@Description	Returns a specific geo-replication configuration by ID
//	@Tags			Geo-Replication
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	models.GeoReplicationResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs/{id} [get]
func (h *GeoReplicationHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	config, err := h.store.GetGeoReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, h.toResponse(c.Request.Context(), config))
}

// Update updates a geo-replication configuration.
//
//	@Summary		Update geo-replication config
//	@Description	Updates an existing geo-replication configuration
//	@Tags			Geo-Replication
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Config ID"
//	@Param			request	body		models.GeoReplicationUpdateRequest	true	"Config updates"
//	@Success		200		{object}	models.GeoReplicationResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs/{id} [put]
func (h *GeoReplicationHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	var req models.GeoReplicationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	config, err := h.store.GetGeoReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// Apply updates
	if req.Enabled != nil {
		config.SetEnabled(*req.Enabled)
	}
	if req.MaxLagSnapshots != nil {
		config.MaxLagSnapshots = *req.MaxLagSnapshots
	}
	if req.MaxLagDurationHours != nil {
		config.MaxLagDurationHours = *req.MaxLagDurationHours
	}
	if req.AlertOnLag != nil {
		config.AlertOnLag = *req.AlertOnLag
	}

	if err := h.store.UpdateGeoReplicationConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to update geo-replication config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config"})
		return
	}

	h.logger.Info().Str("config_id", id.String()).Msg("geo-replication config updated")
	c.JSON(http.StatusOK, h.toResponse(c.Request.Context(), config))
}

// Delete removes a geo-replication configuration.
//
//	@Summary		Delete geo-replication config
//	@Description	Removes a geo-replication configuration
//	@Tags			Geo-Replication
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs/{id} [delete]
func (h *GeoReplicationHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	config, err := h.store.GetGeoReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if err := h.store.DeleteGeoReplicationConfig(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to delete geo-replication config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config"})
		return
	}

	h.logger.Info().Str("config_id", id.String()).Msg("geo-replication config deleted")
	c.JSON(http.StatusOK, gin.H{"message": "config deleted"})
}

// TriggerReplication manually triggers replication for a config.
//
//	@Summary		Trigger replication
//	@Description	Manually triggers replication for a geo-replication configuration
//	@Tags			Geo-Replication
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs/{id}/trigger [post]
func (h *GeoReplicationHandler) TriggerReplication(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	config, err := h.store.GetGeoReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "replication is disabled for this config"})
		return
	}

	// Mark as pending to be picked up by the replication processor
	config.Status = string(models.GeoReplicationStatusPending)
	if err := h.store.UpdateGeoReplicationConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to trigger replication")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger replication"})
		return
	}

	h.logger.Info().Str("config_id", id.String()).Msg("replication triggered")
	c.JSON(http.StatusOK, gin.H{"message": "replication triggered"})
}

// GetEvents returns recent replication events for a config.
//
//	@Summary		Get replication events
//	@Description	Returns recent replication events for a geo-replication configuration
//	@Tags			Geo-Replication
//	@Produce		json
//	@Param			id	path		string	true	"Config ID"
//	@Success		200	{object}	map[string][]models.ReplicationEvent
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/configs/{id}/events [get]
func (h *GeoReplicationHandler) GetEvents(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	config, err := h.store.GetGeoReplicationConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	events, err := h.store.GetReplicationEvents(c.Request.Context(), id, 50)
	if err != nil {
		h.logger.Error().Err(err).Str("config_id", id.String()).Msg("failed to get replication events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetRepositoryStatus returns the geo-replication status for a repository.
//
//	@Summary		Get repository replication status
//	@Description	Returns the geo-replication status for a specific repository
//	@Tags			Geo-Replication
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	models.GeoReplicationResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/repositories/{id}/status [get]
func (h *GeoReplicationHandler) GetRepositoryStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	config, err := h.store.GetGeoReplicationConfigByRepository(c.Request.Context(), repoID)
	if err != nil {
		// No replication configured
		c.JSON(http.StatusOK, gin.H{
			"configured": false,
			"message":    "no geo-replication configured for this repository",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"config":     h.toResponse(c.Request.Context(), config),
	})
}

// SetRepositoryRegion sets the region for a repository.
//
//	@Summary		Set repository region
//	@Description	Sets the geographic region for a repository
//	@Tags			Geo-Replication
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Repository ID"
//	@Param			request	body		object{region=string}	true	"Region code"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/repositories/{id}/region [put]
func (h *GeoReplicationHandler) SetRepositoryRegion(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req struct {
		Region string `json:"region" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate region
	if _, err := backup.GetRegionByCode(req.Region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid region code"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if err := h.store.UpdateRepositoryRegion(c.Request.Context(), repoID, req.Region); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to update repository region")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update region"})
		return
	}

	h.logger.Info().
		Str("repo_id", repoID.String()).
		Str("region", req.Region).
		Msg("repository region updated")

	c.JSON(http.StatusOK, gin.H{"message": "region updated", "region": req.Region})
}

// GetSummary returns a summary of all geo-replication for the organization.
//
//	@Summary		Get replication summary
//	@Description	Returns a summary of all geo-replication status for the organization
//	@Tags			Geo-Replication
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/geo-replication/summary [get]
func (h *GeoReplicationHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	configs, err := h.store.ListGeoReplicationConfigsByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get replication summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get summary"})
		return
	}

	summary := struct {
		TotalConfigs   int `json:"total_configs"`
		EnabledConfigs int `json:"enabled_configs"`
		SyncedCount    int `json:"synced_count"`
		SyncingCount   int `json:"syncing_count"`
		PendingCount   int `json:"pending_count"`
		FailedCount    int `json:"failed_count"`
	}{
		TotalConfigs: len(configs),
	}

	for _, cfg := range configs {
		if cfg.Enabled {
			summary.EnabledConfigs++
		}
		switch models.GeoReplicationStatus(cfg.Status) {
		case models.GeoReplicationStatusSynced:
			summary.SyncedCount++
		case models.GeoReplicationStatusSyncing:
			summary.SyncingCount++
		case models.GeoReplicationStatusPending:
			summary.PendingCount++
		case models.GeoReplicationStatusFailed:
			summary.FailedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
		"regions": backup.AllRegions(),
	})
}

// toResponse converts a GeoReplicationConfig to a GeoReplicationResponse.
func (h *GeoReplicationHandler) toResponse(ctx context.Context, cfg *models.GeoReplicationConfig) models.GeoReplicationResponse {
	sourceRegion, _ := backup.GetRegionByCode(cfg.SourceRegion)
	targetRegion, _ := backup.GetRegionByCode(cfg.TargetRegion)

	resp := models.GeoReplicationResponse{
		ID:                  cfg.ID,
		SourceRepositoryID:  cfg.SourceRepositoryID,
		TargetRepositoryID:  cfg.TargetRepositoryID,
		SourceRegion: models.GeoRegion{
			Code:        sourceRegion.Code,
			Name:        sourceRegion.Name,
			DisplayName: sourceRegion.DisplayName,
			Latitude:    sourceRegion.Latitude,
			Longitude:   sourceRegion.Longitude,
		},
		TargetRegion: models.GeoRegion{
			Code:        targetRegion.Code,
			Name:        targetRegion.Name,
			DisplayName: targetRegion.DisplayName,
			Latitude:    targetRegion.Latitude,
			Longitude:   targetRegion.Longitude,
		},
		Enabled:             cfg.Enabled,
		Status:              cfg.Status,
		LastSnapshotID:      cfg.LastSnapshotID,
		LastSyncAt:          cfg.LastSyncAt,
		LastError:           cfg.LastError,
		MaxLagSnapshots:     cfg.MaxLagSnapshots,
		MaxLagDurationHours: cfg.MaxLagDurationHours,
		AlertOnLag:          cfg.AlertOnLag,
		CreatedAt:           cfg.CreatedAt,
		UpdatedAt:           cfg.UpdatedAt,
	}

	// Get replication lag
	snapshotsBehind, lastSyncAt, err := h.store.GetReplicationLagForConfig(ctx, cfg.ID)
	if err == nil {
		timeBehindHours := 0
		if lastSyncAt != nil {
			timeBehindHours = int(time.Since(*lastSyncAt).Hours())
		}
		isHealthy := snapshotsBehind <= cfg.MaxLagSnapshots && timeBehindHours <= cfg.MaxLagDurationHours

		lastSyncStr := ""
		if lastSyncAt != nil {
			lastSyncStr = lastSyncAt.Format(time.RFC3339)
		}

		resp.ReplicationLag = &models.ReplicationLagResponse{
			SnapshotsBehind: snapshotsBehind,
			TimeBehindHours: timeBehindHours,
			IsHealthy:       isHealthy,
			LastSyncAt:      lastSyncStr,
		}
	}

	return resp
}
