package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// StorageTiersStore defines the interface for storage tiering persistence.
type StorageTiersStore interface {
	// Tier configuration
	GetStorageTierConfigs(ctx context.Context, orgID uuid.UUID) ([]*models.StorageTierConfig, error)
	GetStorageTierConfig(ctx context.Context, id uuid.UUID) (*models.StorageTierConfig, error)
	CreateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error
	UpdateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error
	CreateDefaultTierConfigs(ctx context.Context, orgID uuid.UUID) error

	// Tier rules
	GetTierRules(ctx context.Context, orgID uuid.UUID) ([]*models.TierRule, error)
	GetTierRule(ctx context.Context, id uuid.UUID) (*models.TierRule, error)
	CreateTierRule(ctx context.Context, rule *models.TierRule) error
	UpdateTierRule(ctx context.Context, rule *models.TierRule) error
	DeleteTierRule(ctx context.Context, id uuid.UUID) error

	// Snapshot tiers
	GetSnapshotTier(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.SnapshotTier, error)
	GetSnapshotTiersByRepository(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotTier, error)
	GetTierTransitionHistory(ctx context.Context, snapshotID string, repositoryID uuid.UUID, limit int) ([]*models.TierTransition, error)

	// Cold restore
	GetColdRestoreRequest(ctx context.Context, id uuid.UUID) (*models.ColdRestoreRequest, error)
	GetActiveColdRestoreRequests(ctx context.Context, orgID uuid.UUID) ([]*models.ColdRestoreRequest, error)

	// Cost reports
	GetLatestTierCostReport(ctx context.Context, orgID uuid.UUID) (*models.TierCostReport, error)
	GetTierCostReports(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.TierCostReport, error)

	// Stats
	GetTierStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.TierStatsSummary, error)
}

// TieringSchedulerInterface defines the interface for tiering operations.
type TieringSchedulerInterface interface {
	RequestColdRestore(ctx context.Context, orgID uuid.UUID, snapshotID string, repositoryID, requestedBy uuid.UUID, priority string) (*models.ColdRestoreRequest, error)
	GetRestoreStatus(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.ColdRestoreRequest, error)
	ManualTierChange(ctx context.Context, snapshotID string, repositoryID uuid.UUID, toTier models.StorageTierType, reason string) error
	TriggerProcessing(ctx context.Context)
	TriggerCostReport(ctx context.Context)
}

// StorageTiersHandler handles storage tiering API endpoints.
type StorageTiersHandler struct {
	store     StorageTiersStore
	scheduler TieringSchedulerInterface
	logger    zerolog.Logger
}

// NewStorageTiersHandler creates a new StorageTiersHandler.
func NewStorageTiersHandler(store StorageTiersStore, scheduler TieringSchedulerInterface, logger zerolog.Logger) *StorageTiersHandler {
	return &StorageTiersHandler{
		store:     store,
		scheduler: scheduler,
		logger:    logger.With().Str("component", "storage_tiers_handler").Logger(),
	}
}

// RegisterRoutes registers storage tiering routes on the given router group.
func (h *StorageTiersHandler) RegisterRoutes(r *gin.RouterGroup) {
	tiers := r.Group("/storage-tiers")
	{
		// Tier configuration
		tiers.GET("/configs", h.ListConfigs)
		tiers.PUT("/configs/:id", h.UpdateConfig)
		tiers.POST("/configs/defaults", h.CreateDefaultConfigs)

		// Tier rules
		tiers.GET("/rules", h.ListRules)
		tiers.POST("/rules", h.CreateRule)
		tiers.GET("/rules/:id", h.GetRule)
		tiers.PUT("/rules/:id", h.UpdateRule)
		tiers.DELETE("/rules/:id", h.DeleteRule)

		// Snapshot tiers
		tiers.GET("/snapshots/repository/:repositoryId", h.GetSnapshotTiers)
		tiers.GET("/snapshots/:snapshotId/repository/:repositoryId", h.GetSnapshotTier)
		tiers.POST("/snapshots/:snapshotId/repository/:repositoryId/transition", h.TransitionSnapshot)
		tiers.GET("/snapshots/:snapshotId/repository/:repositoryId/history", h.GetTransitionHistory)

		// Cold restore
		tiers.POST("/cold-restore", h.RequestColdRestore)
		tiers.GET("/cold-restore", h.ListColdRestoreRequests)
		tiers.GET("/cold-restore/:id", h.GetColdRestoreStatus)

		// Cost reports
		tiers.GET("/cost-report", h.GetCostReport)
		tiers.GET("/cost-reports", h.ListCostReports)
		tiers.POST("/cost-report/generate", h.GenerateCostReport)

		// Stats and summary
		tiers.GET("/stats", h.GetTierStats)

		// Admin operations
		tiers.POST("/process", h.TriggerProcessing)
	}
}

// ListConfigs returns all tier configurations for the organization.
//
//	@Summary		List tier configurations
//	@Description	Returns all storage tier configurations for the current organization
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	map[string][]models.StorageTierConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/configs [get]
func (h *StorageTiersHandler) ListConfigs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	configs, err := h.store.GetStorageTierConfigs(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list tier configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tier configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// UpdateConfig updates a tier configuration.
//
//	@Summary		Update tier configuration
//	@Description	Updates an existing storage tier configuration
//	@Tags			Storage Tiers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Config ID"
//	@Param			request	body		models.StorageTierConfigUpdateRequest	true	"Config updates"
//	@Success		200		{object}	models.StorageTierConfig
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/configs/{id} [put]
func (h *StorageTiersHandler) UpdateConfig(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config ID"})
		return
	}

	config, err := h.store.GetStorageTierConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	if config.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	var req struct {
		Name           *string  `json:"name"`
		Description    *string  `json:"description"`
		CostPerGBMonth *float64 `json:"cost_per_gb_month"`
		RetrievalCost  *float64 `json:"retrieval_cost"`
		RetrievalTime  *string  `json:"retrieval_time"`
		Enabled        *bool    `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Description != nil {
		config.Description = *req.Description
	}
	if req.CostPerGBMonth != nil {
		config.CostPerGBMonth = *req.CostPerGBMonth
	}
	if req.RetrievalCost != nil {
		config.RetrievalCost = *req.RetrievalCost
	}
	if req.RetrievalTime != nil {
		config.RetrievalTime = *req.RetrievalTime
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.store.UpdateStorageTierConfig(c.Request.Context(), config); err != nil {
		h.logger.Error().Err(err).Msg("failed to update tier config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tier config"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// CreateDefaultConfigs creates default tier configurations for the organization.
//
//	@Summary		Create default tier configurations
//	@Description	Creates the default hot/warm/cold/archive tier configurations
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		201	{object}	map[string][]models.StorageTierConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/configs/defaults [post]
func (h *StorageTiersHandler) CreateDefaultConfigs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if err := h.store.CreateDefaultTierConfigs(c.Request.Context(), user.CurrentOrgID); err != nil {
		h.logger.Error().Err(err).Msg("failed to create default tier configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create default tier configs"})
		return
	}

	configs, err := h.store.GetStorageTierConfigs(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list tier configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tier configs"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"configs": configs})
}

// ListRules returns all tier rules for the organization.
//
//	@Summary		List tier rules
//	@Description	Returns all tier transition rules for the current organization
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	map[string][]models.TierRule
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/rules [get]
func (h *StorageTiersHandler) ListRules(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	rules, err := h.store.GetTierRules(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list tier rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tier rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// CreateRule creates a new tier rule.
//
//	@Summary		Create tier rule
//	@Description	Creates a new tier transition rule
//	@Tags			Storage Tiers
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.TierRuleCreateRequest	true	"Rule details"
//	@Success		201		{object}	models.TierRule
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/rules [post]
func (h *StorageTiersHandler) CreateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req struct {
		Name            string  `json:"name" binding:"required"`
		Description     string  `json:"description"`
		FromTier        string  `json:"from_tier" binding:"required"`
		ToTier          string  `json:"to_tier" binding:"required"`
		AgeThresholdDay int     `json:"age_threshold_days" binding:"required,min=1"`
		MinCopies       int     `json:"min_copies"`
		Priority        int     `json:"priority"`
		RepositoryID    *string `json:"repository_id"`
		ScheduleID      *string `json:"schedule_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate tier types
	fromTier := models.StorageTierType(req.FromTier)
	toTier := models.StorageTierType(req.ToTier)
	validTiers := map[models.StorageTierType]bool{
		models.StorageTierHot:     true,
		models.StorageTierWarm:    true,
		models.StorageTierCold:    true,
		models.StorageTierArchive: true,
	}
	if !validTiers[fromTier] || !validTiers[toTier] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tier type"})
		return
	}
	if fromTier == toTier {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_tier and to_tier must be different"})
		return
	}

	rule := models.NewTierRule(user.CurrentOrgID, req.Name, fromTier, toTier, req.AgeThresholdDay)
	rule.Description = req.Description
	if req.MinCopies > 0 {
		rule.MinCopies = req.MinCopies
	}
	if req.Priority > 0 {
		rule.Priority = req.Priority
	}

	if req.RepositoryID != nil {
		repoID, err := uuid.Parse(*req.RepositoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
			return
		}
		rule.RepositoryID = &repoID
	}

	if req.ScheduleID != nil {
		schedID, err := uuid.Parse(*req.ScheduleID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule_id"})
			return
		}
		rule.ScheduleID = &schedID
	}

	if err := h.store.CreateTierRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Msg("failed to create tier rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tier rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRule returns a single tier rule.
//
//	@Summary		Get tier rule
//	@Description	Returns a specific tier transition rule
//	@Tags			Storage Tiers
//	@Produce		json
//	@Param			id	path		string	true	"Rule ID"
//	@Success		200	{object}	models.TierRule
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/rules/{id} [get]
func (h *StorageTiersHandler) GetRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetTierRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	if rule.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRule updates an existing tier rule.
//
//	@Summary		Update tier rule
//	@Description	Updates an existing tier transition rule
//	@Tags			Storage Tiers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Rule ID"
//	@Param			request	body		models.TierRuleUpdateRequest	true	"Rule updates"
//	@Success		200		{object}	models.TierRule
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/rules/{id} [put]
func (h *StorageTiersHandler) UpdateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetTierRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	if rule.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req struct {
		Name            *string `json:"name"`
		Description     *string `json:"description"`
		AgeThresholdDay *int    `json:"age_threshold_days"`
		MinCopies       *int    `json:"min_copies"`
		Priority        *int    `json:"priority"`
		Enabled         *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.AgeThresholdDay != nil {
		rule.AgeThresholdDay = *req.AgeThresholdDay
	}
	if req.MinCopies != nil {
		rule.MinCopies = *req.MinCopies
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.store.UpdateTierRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Msg("failed to update tier rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tier rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes a tier rule.
//
//	@Summary		Delete tier rule
//	@Description	Deletes an existing tier transition rule
//	@Tags			Storage Tiers
//	@Param			id	path	string	true	"Rule ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/rules/{id} [delete]
func (h *StorageTiersHandler) DeleteRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetTierRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	if rule.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	if err := h.store.DeleteTierRule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete tier rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tier rule"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetSnapshotTiers returns all snapshot tiers for a repository.
//
//	@Summary		List snapshot tiers
//	@Description	Returns tier information for all snapshots in a repository
//	@Tags			Storage Tiers
//	@Produce		json
//	@Param			repositoryId	path		string	true	"Repository ID"
//	@Success		200				{object}	map[string][]models.SnapshotTier
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/snapshots/repository/{repositoryId} [get]
func (h *StorageTiersHandler) GetSnapshotTiers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	repoIDStr := c.Param("repositoryId")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	tiers, err := h.store.GetSnapshotTiersByRepository(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get snapshot tiers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get snapshot tiers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tiers": tiers})
}

// GetSnapshotTier returns the tier info for a specific snapshot.
//
//	@Summary		Get snapshot tier
//	@Description	Returns tier information for a specific snapshot
//	@Tags			Storage Tiers
//	@Produce		json
//	@Param			snapshotId		path		string	true	"Snapshot ID"
//	@Param			repositoryId	path		string	true	"Repository ID"
//	@Success		200				{object}	models.SnapshotTier
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		404				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/snapshots/{snapshotId}/repository/{repositoryId} [get]
func (h *StorageTiersHandler) GetSnapshotTier(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("snapshotId")
	repoIDStr := c.Param("repositoryId")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	tier, err := h.store.GetSnapshotTier(c.Request.Context(), snapshotID, repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot tier not found"})
		return
	}

	if tier.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot tier not found"})
		return
	}

	c.JSON(http.StatusOK, tier)
}

// TransitionSnapshot manually transitions a snapshot to a different tier.
//
//	@Summary		Transition snapshot tier
//	@Description	Manually moves a snapshot to a different storage tier
//	@Tags			Storage Tiers
//	@Accept			json
//	@Produce		json
//	@Param			snapshotId		path		string	true	"Snapshot ID"
//	@Param			repositoryId	path		string	true	"Repository ID"
//	@Param			request			body		object	true	"Transition request"
//	@Success		200				{object}	map[string]string
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/snapshots/{snapshotId}/repository/{repositoryId}/transition [post]
func (h *StorageTiersHandler) TransitionSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if h.scheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tiering scheduler not available"})
		return
	}

	snapshotID := c.Param("snapshotId")
	repoIDStr := c.Param("repositoryId")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req struct {
		ToTier string `json:"to_tier" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	toTier := models.StorageTierType(req.ToTier)
	validTiers := map[models.StorageTierType]bool{
		models.StorageTierHot:     true,
		models.StorageTierWarm:    true,
		models.StorageTierCold:    true,
		models.StorageTierArchive: true,
	}
	if !validTiers[toTier] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tier type"})
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "Manual transition by user"
	}

	if err := h.scheduler.ManualTierChange(c.Request.Context(), snapshotID, repoID, toTier, reason); err != nil {
		h.logger.Error().Err(err).Msg("failed to transition snapshot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "snapshot transition initiated"})
}

// GetTransitionHistory returns the tier transition history for a snapshot.
//
//	@Summary		Get transition history
//	@Description	Returns the tier transition history for a specific snapshot
//	@Tags			Storage Tiers
//	@Produce		json
//	@Param			snapshotId		path		string	true	"Snapshot ID"
//	@Param			repositoryId	path		string	true	"Repository ID"
//	@Success		200				{object}	map[string][]models.TierTransition
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/snapshots/{snapshotId}/repository/{repositoryId}/history [get]
func (h *StorageTiersHandler) GetTransitionHistory(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("snapshotId")
	repoIDStr := c.Param("repositoryId")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	history, err := h.store.GetTierTransitionHistory(c.Request.Context(), snapshotID, repoID, 50)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get transition history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get transition history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// RequestColdRestore initiates a restore request for cold/archive data.
//
//	@Summary		Request cold restore
//	@Description	Initiates a restore request for data in cold or archive storage
//	@Tags			Storage Tiers
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object	true	"Restore request"
//	@Success		201		{object}	models.ColdRestoreRequest
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cold-restore [post]
func (h *StorageTiersHandler) RequestColdRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if h.scheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tiering scheduler not available"})
		return
	}

	var req struct {
		SnapshotID   string `json:"snapshot_id" binding:"required"`
		RepositoryID string `json:"repository_id" binding:"required"`
		Priority     string `json:"priority"` // standard, expedited, bulk
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	repoID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "standard"
	}
	if priority != "standard" && priority != "expedited" && priority != "bulk" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority, must be: standard, expedited, or bulk"})
		return
	}

	restoreReq, err := h.scheduler.RequestColdRestore(c.Request.Context(), user.CurrentOrgID, req.SnapshotID, repoID, user.ID, priority)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create cold restore request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, restoreReq)
}

// ListColdRestoreRequests returns all active cold restore requests.
//
//	@Summary		List cold restore requests
//	@Description	Returns all active cold/archive restore requests for the organization
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	map[string][]models.ColdRestoreRequest
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cold-restore [get]
func (h *StorageTiersHandler) ListColdRestoreRequests(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	requests, err := h.store.GetActiveColdRestoreRequests(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list cold restore requests")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list cold restore requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// GetColdRestoreStatus returns the status of a cold restore request.
//
//	@Summary		Get cold restore status
//	@Description	Returns the status of a specific cold restore request
//	@Tags			Storage Tiers
//	@Produce		json
//	@Param			id	path		string	true	"Request ID"
//	@Success		200	{object}	models.ColdRestoreRequest
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cold-restore/{id} [get]
func (h *StorageTiersHandler) GetColdRestoreStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	req, err := h.store.GetColdRestoreRequest(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	if req.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// GetCostReport returns the latest cost optimization report.
//
//	@Summary		Get cost report
//	@Description	Returns the latest storage tier cost optimization report
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	models.TierCostReport
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cost-report [get]
func (h *StorageTiersHandler) GetCostReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	report, err := h.store.GetLatestTierCostReport(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no cost report available"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ListCostReports returns recent cost reports.
//
//	@Summary		List cost reports
//	@Description	Returns recent storage tier cost optimization reports
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	map[string][]models.TierCostReport
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cost-reports [get]
func (h *StorageTiersHandler) ListCostReports(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	reports, err := h.store.GetTierCostReports(c.Request.Context(), user.CurrentOrgID, 30)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list cost reports")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list cost reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// GenerateCostReport manually triggers cost report generation.
//
//	@Summary		Generate cost report
//	@Description	Manually triggers generation of a cost optimization report
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		202	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		503	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/cost-report/generate [post]
func (h *StorageTiersHandler) GenerateCostReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if h.scheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tiering scheduler not available"})
		return
	}

	h.scheduler.TriggerCostReport(c.Request.Context())
	c.JSON(http.StatusAccepted, gin.H{"message": "cost report generation initiated"})
}

// GetTierStats returns aggregate tier statistics.
//
//	@Summary		Get tier statistics
//	@Description	Returns aggregate storage tier statistics for the organization
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		200	{object}	models.TierStatsSummary
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/stats [get]
func (h *StorageTiersHandler) GetTierStats(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	stats, err := h.store.GetTierStatsSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get tier stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tier stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// TriggerProcessing manually triggers tiering rule processing.
//
//	@Summary		Trigger tiering processing
//	@Description	Manually triggers processing of tier transition rules
//	@Tags			Storage Tiers
//	@Produce		json
//	@Success		202	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		503	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/storage-tiers/process [post]
func (h *StorageTiersHandler) TriggerProcessing(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can trigger processing
	if user.CurrentOrgRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if h.scheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tiering scheduler not available"})
		return
	}

	h.scheduler.TriggerProcessing(c.Request.Context())
	c.JSON(http.StatusAccepted, gin.H{"message": "tiering processing initiated"})
}

// Ensure backup package is imported for any tiering-related operations
var _ = backup.DefaultTieringConfig
