package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DockerBackupStore defines the interface for Docker backup persistence operations.
type DockerBackupStore interface {
	GetDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error)
	GetDockerContainerByID(ctx context.Context, id uuid.UUID) (*models.DockerContainerConfig, error)
	GetDockerContainerByContainerID(ctx context.Context, agentID uuid.UUID, containerID string) (*models.DockerContainerConfig, error)
	CreateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error
	UpdateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error
	DeleteDockerContainer(ctx context.Context, id uuid.UUID) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
}

// DockerBackupHandler handles Docker backup-related HTTP endpoints.
type DockerBackupHandler struct {
	store     DockerBackupStore
	discovery *docker.DiscoveryService
	parser    *docker.LabelParser
	logger    zerolog.Logger
}

// NewDockerBackupHandler creates a new DockerBackupHandler.
func NewDockerBackupHandler(store DockerBackupStore, discovery *docker.DiscoveryService, logger zerolog.Logger) *DockerBackupHandler {
	return &DockerBackupHandler{
		store:     store,
		discovery: discovery,
		parser:    docker.NewLabelParser(),
		logger:    logger.With().Str("component", "docker_backup_handler").Logger(),
	}
}

// RegisterRoutes registers Docker backup routes on the given router group.
func (h *DockerBackupHandler) RegisterRoutes(r *gin.RouterGroup) {
	docker := r.Group("/docker")
	{
		// Container backup configurations
		docker.GET("/containers", h.ListContainers)
		docker.GET("/containers/:id", h.GetContainer)
		docker.PUT("/containers/:id", h.UpdateContainer)
		docker.DELETE("/containers/:id", h.DeleteContainer)
		docker.POST("/containers/:id/override", h.SetOverride)
		docker.DELETE("/containers/:id/override", h.ClearOverride)

		// Discovery
		docker.POST("/discover", h.DiscoverContainers)
		docker.POST("/containers/:id/refresh", h.RefreshContainer)

		// Label documentation
		docker.GET("/labels/docs", h.GetLabelDocs)
		docker.GET("/labels/examples/compose", h.GetComposeExample)
		docker.GET("/labels/examples/run", h.GetRunExample)
		docker.POST("/labels/validate", h.ValidateLabels)
	}
}

// ContainerDiscoveryRequest is the request body for container discovery.
type ContainerDiscoveryRequest struct {
	AgentID    uuid.UUID              `json:"agent_id" binding:"required"`
	Containers []ContainerInfoRequest `json:"containers" binding:"required"`
}

// ContainerInfoRequest represents container info in a discovery request.
type ContainerInfoRequest struct {
	ID     string            `json:"id" binding:"required"`
	Name   string            `json:"name" binding:"required"`
	Image  string            `json:"image" binding:"required"`
	Labels map[string]string `json:"labels"`
	Mounts []MountInfoRequest `json:"mounts,omitempty"`
	Status string            `json:"status,omitempty"`
}

// MountInfoRequest represents mount info in a discovery request.
type MountInfoRequest struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only"`
}

// UpdateContainerRequest is the request body for updating a container config.
type UpdateContainerRequest struct {
	Enabled        *bool                        `json:"enabled,omitempty"`
	Schedule       *models.DockerBackupSchedule `json:"schedule,omitempty"`
	CronExpression *string                      `json:"cron_expression,omitempty"`
	Excludes       []string                     `json:"excludes,omitempty"`
	PreHook        *string                      `json:"pre_hook,omitempty"`
	PostHook       *string                      `json:"post_hook,omitempty"`
	StopOnBackup   *bool                        `json:"stop_on_backup,omitempty"`
	BackupVolumes  *bool                        `json:"backup_volumes,omitempty"`
}

// SetOverrideRequest is the request body for setting container overrides.
type SetOverrideRequest struct {
	Enabled        *bool                        `json:"enabled,omitempty"`
	Schedule       *models.DockerBackupSchedule `json:"schedule,omitempty"`
	CronExpression *string                      `json:"cron_expression,omitempty"`
	Excludes       []string                     `json:"excludes,omitempty"`
	PreHook        *string                      `json:"pre_hook,omitempty"`
	PostHook       *string                      `json:"post_hook,omitempty"`
	StopOnBackup   *bool                        `json:"stop_on_backup,omitempty"`
	BackupVolumes  *bool                        `json:"backup_volumes,omitempty"`
}

// ValidateLabelsRequest is the request body for validating labels.
type ValidateLabelsRequest struct {
	Labels map[string]string `json:"labels" binding:"required"`
}

// ValidateLabelsResponse is the response for label validation.
type ValidateLabelsResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ListContainers returns all Docker containers configured for backup.
//
//	@Summary		List Docker containers
//	@Description	Returns all Docker containers configured for backup in the organization
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Success		200			{object}	map[string][]models.DockerContainerConfig
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers [get]
func (h *DockerBackupHandler) ListContainers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Optional agent_id filter
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		containers, err := h.store.GetDockerContainersByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list containers")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list containers"})
			return
		}

		// Apply overrides
		for _, container := range containers {
			container.ApplyOverrides()
		}

		c.JSON(http.StatusOK, gin.H{"containers": containers})
		return
	}

	// Get all containers for all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list containers"})
		return
	}

	var allContainers []*models.DockerContainerConfig
	for _, agent := range agents {
		containers, err := h.store.GetDockerContainersByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to list containers for agent")
			continue
		}
		// Apply overrides
		for _, container := range containers {
			container.ApplyOverrides()
		}
		allContainers = append(allContainers, containers...)
	}

	c.JSON(http.StatusOK, gin.H{"containers": allContainers})
}

// GetContainer returns a specific Docker container configuration.
//
//	@Summary		Get Docker container
//	@Description	Returns a specific Docker container backup configuration by ID
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Container config ID"
//	@Success		200	{object}	models.DockerContainerConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id} [get]
func (h *DockerBackupHandler) GetContainer(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Verify container's agent belongs to user's org
	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	container.ApplyOverrides()
	c.JSON(http.StatusOK, container)
}

// UpdateContainer updates a Docker container backup configuration.
//
//	@Summary		Update Docker container
//	@Description	Updates a Docker container backup configuration
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Container config ID"
//	@Param			request	body		UpdateContainerRequest	true	"Update details"
//	@Success		200		{object}	models.DockerContainerConfig
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id} [put]
func (h *DockerBackupHandler) UpdateContainer(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	var req UpdateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	// Update fields via overrides to preserve label-based configuration
	if container.Overrides == nil {
		container.Overrides = &models.ContainerOverrides{}
	}

	if req.Enabled != nil {
		container.Overrides.Enabled = req.Enabled
	}
	if req.Schedule != nil {
		container.Overrides.Schedule = req.Schedule
	}
	if req.CronExpression != nil {
		container.Overrides.CronExpression = req.CronExpression
	}
	if req.Excludes != nil {
		container.Overrides.Excludes = req.Excludes
	}
	if req.PreHook != nil {
		container.Overrides.PreHook = req.PreHook
	}
	if req.PostHook != nil {
		container.Overrides.PostHook = req.PostHook
	}
	if req.StopOnBackup != nil {
		container.Overrides.StopOnBackup = req.StopOnBackup
	}
	if req.BackupVolumes != nil {
		container.Overrides.BackupVolumes = req.BackupVolumes
	}

	if err := h.store.UpdateDockerContainer(c.Request.Context(), container); err != nil {
		h.logger.Error().Err(err).Str("container_id", id.String()).Msg("failed to update container")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update container"})
		return
	}

	h.logger.Info().Str("container_id", id.String()).Msg("container updated")
	container.ApplyOverrides()
	c.JSON(http.StatusOK, container)
}

// DeleteContainer removes a Docker container backup configuration.
//
//	@Summary		Delete Docker container
//	@Description	Removes a Docker container backup configuration
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Container config ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id} [delete]
func (h *DockerBackupHandler) DeleteContainer(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	if err := h.store.DeleteDockerContainer(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("container_id", id.String()).Msg("failed to delete container")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete container"})
		return
	}

	h.logger.Info().Str("container_id", id.String()).Msg("container deleted")
	c.JSON(http.StatusOK, gin.H{"message": "container deleted"})
}

// SetOverride sets UI overrides for a container configuration.
//
//	@Summary		Set container override
//	@Description	Sets UI-configured overrides for a container's label-based settings
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Container config ID"
//	@Param			request	body		SetOverrideRequest	true	"Override settings"
//	@Success		200		{object}	models.DockerContainerConfig
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id}/override [post]
func (h *DockerBackupHandler) SetOverride(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	var req SetOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	container.Overrides = &models.ContainerOverrides{
		Enabled:        req.Enabled,
		Schedule:       req.Schedule,
		CronExpression: req.CronExpression,
		Excludes:       req.Excludes,
		PreHook:        req.PreHook,
		PostHook:       req.PostHook,
		StopOnBackup:   req.StopOnBackup,
		BackupVolumes:  req.BackupVolumes,
	}

	if err := h.store.UpdateDockerContainer(c.Request.Context(), container); err != nil {
		h.logger.Error().Err(err).Str("container_id", id.String()).Msg("failed to set override")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set override"})
		return
	}

	h.logger.Info().Str("container_id", id.String()).Msg("container override set")
	container.ApplyOverrides()
	c.JSON(http.StatusOK, container)
}

// ClearOverride clears UI overrides for a container configuration.
//
//	@Summary		Clear container override
//	@Description	Clears UI-configured overrides, reverting to label-based settings
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Container config ID"
//	@Success		200	{object}	models.DockerContainerConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id}/override [delete]
func (h *DockerBackupHandler) ClearOverride(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	container.Overrides = nil

	if err := h.store.UpdateDockerContainer(c.Request.Context(), container); err != nil {
		h.logger.Error().Err(err).Str("container_id", id.String()).Msg("failed to clear override")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear override"})
		return
	}

	h.logger.Info().Str("container_id", id.String()).Msg("container override cleared")
	c.JSON(http.StatusOK, container)
}

// DiscoverContainers processes container discovery from an agent.
//
//	@Summary		Discover containers
//	@Description	Processes container discovery report from an agent and updates backup configurations
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ContainerDiscoveryRequest	true	"Discovery data"
//	@Success		200		{object}	models.DockerDiscoveryResult
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/discover [post]
func (h *DockerBackupHandler) DiscoverContainers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ContainerDiscoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}

	// Convert request containers to docker.ContainerInfo
	containers := make([]docker.ContainerInfo, len(req.Containers))
	for i, rc := range req.Containers {
		mounts := make([]docker.MountInfo, len(rc.Mounts))
		for j, rm := range rc.Mounts {
			mounts[j] = docker.MountInfo{
				Type:        rm.Type,
				Source:      rm.Source,
				Destination: rm.Destination,
				ReadOnly:    rm.ReadOnly,
			}
		}
		containers[i] = docker.ContainerInfo{
			ID:     rc.ID,
			Name:   rc.Name,
			Image:  rc.Image,
			Labels: rc.Labels,
			Mounts: mounts,
			Status: rc.Status,
		}
	}

	// Run discovery
	result, err := h.discovery.DiscoverContainers(c.Request.Context(), req.AgentID, containers)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", req.AgentID.String()).Msg("failed to discover containers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to discover containers"})
		return
	}

	h.logger.Info().
		Str("agent_id", req.AgentID.String()).
		Int("total_discovered", result.TotalDiscovered).
		Int("new_containers", result.NewContainers).
		Msg("container discovery completed")

	c.JSON(http.StatusOK, result)
}

// RefreshContainer refreshes a single container's configuration.
//
//	@Summary		Refresh container
//	@Description	Refreshes a container's backup configuration from Docker labels
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Container config ID"
//	@Success		200	{object}	models.DockerContainerConfig
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/containers/{id}/refresh [post]
func (h *DockerBackupHandler) RefreshContainer(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	container, err := h.store.GetDockerContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if err := h.verifyContainerAccess(c, user.CurrentOrgID, container); err != nil {
		return
	}

	// Refresh requires re-discovery from the agent
	// For now, just return the current config
	container.ApplyOverrides()
	c.JSON(http.StatusOK, gin.H{
		"container": container,
		"message":   "Container refresh requires agent re-discovery. Trigger discovery to update.",
	})
}

// GetLabelDocs returns documentation for Docker backup labels.
//
//	@Summary		Get label documentation
//	@Description	Returns documentation for all supported Docker backup labels
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.DockerLabelDocs
//	@Security		SessionAuth
//	@Router			/docker/labels/docs [get]
func (h *DockerBackupHandler) GetLabelDocs(c *gin.Context) {
	docs := h.parser.GenerateLabelDocs()
	c.JSON(http.StatusOK, docs)
}

// GetComposeExample returns an example Docker Compose configuration.
//
//	@Summary		Get Docker Compose example
//	@Description	Returns an example Docker Compose configuration with backup labels
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/labels/examples/compose [get]
func (h *DockerBackupHandler) GetComposeExample(c *gin.Context) {
	example := h.parser.GenerateDockerComposeExample()
	c.JSON(http.StatusOK, gin.H{"example": example})
}

// GetRunExample returns an example docker run command.
//
//	@Summary		Get docker run example
//	@Description	Returns an example docker run command with backup labels
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/labels/examples/run [get]
func (h *DockerBackupHandler) GetRunExample(c *gin.Context) {
	example := h.parser.GenerateDockerRunExample()
	c.JSON(http.StatusOK, gin.H{"example": example})
}

// ValidateLabels validates Docker backup labels.
//
//	@Summary		Validate labels
//	@Description	Validates Docker backup labels and returns any errors
//	@Tags			Docker Backup
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ValidateLabelsRequest	true	"Labels to validate"
//	@Success		200		{object}	ValidateLabelsResponse
//	@Failure		400		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/labels/validate [post]
func (h *DockerBackupHandler) ValidateLabels(c *gin.Context) {
	var req ValidateLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	errors := h.parser.ValidateLabels(req.Labels)

	c.JSON(http.StatusOK, ValidateLabelsResponse{
		Valid:  len(errors) == 0,
		Errors: errors,
	})
}

// verifyContainerAccess checks if the user has access to the container.
func (h *DockerBackupHandler) verifyContainerAccess(c *gin.Context, orgID uuid.UUID, container *models.DockerContainerConfig) error {
	agent, err := h.store.GetAgentByID(c.Request.Context(), container.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return err
	}

	if agent.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return err
	}

	return nil
}
