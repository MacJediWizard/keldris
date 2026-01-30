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

// DockerStackStore defines the interface for docker stack persistence operations.
type DockerStackStore interface {
	// Stack operations
	CreateDockerStack(ctx context.Context, stack *models.DockerStack) error
	GetDockerStackByID(ctx context.Context, id uuid.UUID) (*models.DockerStack, error)
	GetDockerStacksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerStack, error)
	GetDockerStacksByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerStack, error)
	UpdateDockerStack(ctx context.Context, stack *models.DockerStack) error
	DeleteDockerStack(ctx context.Context, id uuid.UUID) error

	// Backup operations
	CreateDockerStackBackup(ctx context.Context, backup *models.DockerStackBackup) error
	GetDockerStackBackupByID(ctx context.Context, id uuid.UUID) (*models.DockerStackBackup, error)
	GetDockerStackBackupsByStackID(ctx context.Context, stackID uuid.UUID) ([]*models.DockerStackBackup, error)
	UpdateDockerStackBackup(ctx context.Context, backup *models.DockerStackBackup) error
	DeleteDockerStackBackup(ctx context.Context, id uuid.UUID) error

	// Restore operations
	CreateDockerStackRestore(ctx context.Context, restore *models.DockerStackRestore) error
	GetDockerStackRestoreByID(ctx context.Context, id uuid.UUID) (*models.DockerStackRestore, error)
	GetDockerStackRestoresByBackupID(ctx context.Context, backupID uuid.UUID) ([]*models.DockerStackRestore, error)
	UpdateDockerStackRestore(ctx context.Context, restore *models.DockerStackRestore) error

	// Agent access
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// DockerStackBackupService defines the interface for docker stack backup operations.
type DockerStackBackupService interface {
	TriggerBackup(ctx context.Context, stack *models.DockerStack) (*models.DockerStackBackup, error)
	RestoreStack(ctx context.Context, backup *models.DockerStackBackup, req models.RestoreDockerStackRequest) (*models.DockerStackRestore, error)
	DiscoverStacks(ctx context.Context, agentID uuid.UUID, searchPaths []string) ([]models.DiscoveredDockerStack, error)
}

// DockerStacksHandler handles docker stack-related HTTP endpoints.
type DockerStacksHandler struct {
	store         DockerStackStore
	backupService DockerStackBackupService
	logger        zerolog.Logger
}

// NewDockerStacksHandler creates a new DockerStacksHandler.
func NewDockerStacksHandler(store DockerStackStore, backupService DockerStackBackupService, logger zerolog.Logger) *DockerStacksHandler {
	return &DockerStacksHandler{
		store:         store,
		backupService: backupService,
		logger:        logger.With().Str("component", "docker_stacks_handler").Logger(),
	}
}

// RegisterRoutes registers docker stack routes on the given router group.
func (h *DockerStacksHandler) RegisterRoutes(r *gin.RouterGroup) {
	stacks := r.Group("/docker-stacks")
	{
		stacks.GET("", h.List)
		stacks.POST("", h.Create)
		stacks.GET("/:id", h.Get)
		stacks.PUT("/:id", h.Update)
		stacks.DELETE("/:id", h.Delete)
		stacks.POST("/:id/backup", h.TriggerBackup)
		stacks.GET("/:id/backups", h.ListBackups)
		stacks.POST("/discover", h.DiscoverStacks)
	}

	backups := r.Group("/docker-stack-backups")
	{
		backups.GET("/:id", h.GetBackup)
		backups.DELETE("/:id", h.DeleteBackup)
		backups.POST("/:id/restore", h.RestoreBackup)
	}

	restores := r.Group("/docker-stack-restores")
	{
		restores.GET("/:id", h.GetRestore)
	}
}

// List returns all docker stacks for the authenticated user's organization.
//
//	@Summary		List Docker stacks
//	@Description	Returns all registered Docker Compose stacks for the current organization
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Success		200			{object}	models.DockerStackListResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks [get]
func (h *DockerStacksHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Handle agent_id filter
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil || agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		stacks, err := h.store.GetDockerStacksByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list docker stacks")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker stacks"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stacks": stacks})
		return
	}

	stacks, err := h.store.GetDockerStacksByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list docker stacks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker stacks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stacks": stacks})
}

// Create registers a new Docker stack.
//
//	@Summary		Register Docker stack
//	@Description	Registers a new Docker Compose stack for backup
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.CreateDockerStackRequest	true	"Stack configuration"
//	@Success		201		{object}	models.DockerStack
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks [post]
func (h *DockerStacksHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateDockerStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate required fields
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}
	if req.ComposePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "compose_path is required"})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	stack := models.NewDockerStack(user.CurrentOrgID, agentID, req.Name, req.ComposePath)
	stack.Description = req.Description
	stack.ExportImages = req.ExportImages
	stack.IncludeEnvFiles = req.IncludeEnvFiles
	stack.StopForBackup = req.StopForBackup
	stack.ExcludePaths = req.ExcludePaths

	if err := h.store.CreateDockerStack(c.Request.Context(), stack); err != nil {
		h.logger.Error().Err(err).Msg("failed to create docker stack")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create docker stack"})
		return
	}

	h.logger.Info().
		Str("stack_id", stack.ID.String()).
		Str("name", stack.Name).
		Str("agent_id", agentID.String()).
		Msg("docker stack registered")

	c.JSON(http.StatusCreated, stack)
}

// Get returns a specific docker stack by ID.
//
//	@Summary		Get Docker stack
//	@Description	Returns details of a specific Docker Compose stack
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Stack ID"
//	@Success		200	{object}	models.DockerStack
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/{id} [get]
func (h *DockerStacksHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetDockerStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if stack.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	c.JSON(http.StatusOK, stack)
}

// Update updates a docker stack configuration.
//
//	@Summary		Update Docker stack
//	@Description	Updates configuration for a Docker Compose stack
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Stack ID"
//	@Param			body	body		models.UpdateDockerStackRequest	true	"Updated configuration"
//	@Success		200		{object}	models.DockerStack
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/{id} [put]
func (h *DockerStacksHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetDockerStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if stack.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	var req models.UpdateDockerStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Apply updates
	if req.Name != nil {
		stack.Name = *req.Name
	}
	if req.Description != nil {
		stack.Description = *req.Description
	}
	if req.ExportImages != nil {
		stack.ExportImages = *req.ExportImages
	}
	if req.IncludeEnvFiles != nil {
		stack.IncludeEnvFiles = *req.IncludeEnvFiles
	}
	if req.StopForBackup != nil {
		stack.StopForBackup = *req.StopForBackup
	}
	if req.ExcludePaths != nil {
		stack.ExcludePaths = *req.ExcludePaths
	}

	if err := h.store.UpdateDockerStack(c.Request.Context(), stack); err != nil {
		h.logger.Error().Err(err).Str("stack_id", id.String()).Msg("failed to update docker stack")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update docker stack"})
		return
	}

	c.JSON(http.StatusOK, stack)
}

// Delete removes a docker stack registration.
//
//	@Summary		Delete Docker stack
//	@Description	Removes a Docker Compose stack from backup management
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Stack ID"
//	@Success		204	{object}	nil
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/{id} [delete]
func (h *DockerStacksHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetDockerStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if stack.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if err := h.store.DeleteDockerStack(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("stack_id", id.String()).Msg("failed to delete docker stack")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete docker stack"})
		return
	}

	c.Status(http.StatusNoContent)
}

// TriggerBackup triggers a manual backup for a docker stack.
//
//	@Summary		Trigger Docker stack backup
//	@Description	Triggers an immediate backup of a Docker Compose stack
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string									true	"Stack ID"
//	@Param			body	body		models.TriggerDockerStackBackupRequest	false	"Backup options"
//	@Success		202		{object}	models.DockerStackBackup
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/{id}/backup [post]
func (h *DockerStacksHandler) TriggerBackup(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetDockerStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if stack.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	// Parse optional override options
	var req models.TriggerDockerStackBackupRequest
	_ = c.ShouldBindJSON(&req) // Ignore errors, request body is optional

	// Apply overrides
	if req.ExportImages != nil {
		stack.ExportImages = *req.ExportImages
	}
	if req.StopForBackup != nil {
		stack.StopForBackup = *req.StopForBackup
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service not available"})
		return
	}

	backup, err := h.backupService.TriggerBackup(c.Request.Context(), stack)
	if err != nil {
		h.logger.Error().Err(err).Str("stack_id", id.String()).Msg("failed to trigger backup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger backup"})
		return
	}

	h.logger.Info().
		Str("stack_id", stack.ID.String()).
		Str("backup_id", backup.ID.String()).
		Msg("docker stack backup triggered")

	c.JSON(http.StatusAccepted, backup)
}

// ListBackups returns all backups for a docker stack.
//
//	@Summary		List Docker stack backups
//	@Description	Returns all backups for a specific Docker Compose stack
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Stack ID"
//	@Success		200	{object}	models.DockerStackBackupListResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/{id}/backups [get]
func (h *DockerStacksHandler) ListBackups(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetDockerStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	if stack.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker stack not found"})
		return
	}

	backups, err := h.store.GetDockerStackBackupsByStackID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("stack_id", id.String()).Msg("failed to list backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// GetBackup returns a specific docker stack backup by ID.
//
//	@Summary		Get Docker stack backup
//	@Description	Returns details of a specific Docker stack backup
//	@Tags			Docker Stack Backups
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		200	{object}	models.DockerStackBackup
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stack-backups/{id} [get]
func (h *DockerStacksHandler) GetBackup(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerStackBackupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if backup.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// DeleteBackup removes a docker stack backup.
//
//	@Summary		Delete Docker stack backup
//	@Description	Removes a Docker stack backup
//	@Tags			Docker Stack Backups
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		204	{object}	nil
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stack-backups/{id} [delete]
func (h *DockerStacksHandler) DeleteBackup(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerStackBackupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if backup.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if err := h.store.DeleteDockerStackBackup(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("backup_id", id.String()).Msg("failed to delete backup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RestoreBackup restores a docker stack from a backup.
//
//	@Summary		Restore Docker stack
//	@Description	Restores a Docker Compose stack from a backup
//	@Tags			Docker Stack Backups
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Backup ID"
//	@Param			body	body		models.RestoreDockerStackRequest	true	"Restore options"
//	@Success		202		{object}	models.DockerStackRestore
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stack-backups/{id}/restore [post]
func (h *DockerStacksHandler) RestoreBackup(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerStackBackupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if backup.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	var req models.RestoreDockerStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate required fields
	if req.TargetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_path is required"})
		return
	}
	if req.TargetAgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_agent_id is required"})
		return
	}

	targetAgentID, err := uuid.Parse(req.TargetAgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_agent_id"})
		return
	}

	// Verify target agent belongs to user's org
	targetAgent, err := h.store.GetAgentByID(c.Request.Context(), targetAgentID)
	if err != nil || targetAgent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "target agent not found"})
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service not available"})
		return
	}

	restore, err := h.backupService.RestoreStack(c.Request.Context(), backup, req)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", id.String()).Msg("failed to start restore")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start restore"})
		return
	}

	h.logger.Info().
		Str("backup_id", backup.ID.String()).
		Str("restore_id", restore.ID.String()).
		Str("target_path", req.TargetPath).
		Msg("docker stack restore started")

	c.JSON(http.StatusAccepted, restore)
}

// GetRestore returns a specific docker stack restore by ID.
//
//	@Summary		Get Docker stack restore
//	@Description	Returns details of a specific Docker stack restore operation
//	@Tags			Docker Stack Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	models.DockerStackRestore
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stack-restores/{id} [get]
func (h *DockerStacksHandler) GetRestore(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore ID"})
		return
	}

	restore, err := h.store.GetDockerStackRestoreByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	if restore.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	c.JSON(http.StatusOK, restore)
}

// DiscoverStacks discovers Docker Compose stacks on an agent.
//
//	@Summary		Discover Docker stacks
//	@Description	Discovers Docker Compose stacks on an agent by searching specified paths
//	@Tags			Docker Stacks
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.DiscoverDockerStacksRequest	true	"Discovery request"
//	@Success		200		{object}	models.DiscoverDockerStacksResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-stacks/discover [post]
func (h *DockerStacksHandler) DiscoverStacks(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.DiscoverDockerStacksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}
	if len(req.SearchPaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search_paths is required"})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service not available"})
		return
	}

	stacks, err := h.backupService.DiscoverStacks(c.Request.Context(), agentID, req.SearchPaths)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to discover stacks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to discover stacks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stacks": stacks})
}
