package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DockerBackupRequest is the request body for triggering a Docker backup.
type DockerBackupRequest struct {
	AgentID      string   `json:"agent_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	RepositoryID string   `json:"repository_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	ContainerIDs []string `json:"container_ids,omitempty"`
	VolumeNames  []string `json:"volume_names,omitempty"`
}

// DockerBackupStore defines the interface for Docker backup persistence operations.
type DockerBackupStore interface {
	GetDockerContainers(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) ([]models.DockerContainer, error)
	GetDockerVolumes(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) ([]models.DockerVolume, error)
	GetDockerDaemonStatus(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) (*models.DockerDaemonStatus, error)
	CreateDockerBackup(ctx context.Context, orgID uuid.UUID, req *models.DockerBackupParams) (*models.DockerBackupResult, error)
	GetDockerContainerByID(ctx context.Context, id uuid.UUID) (*models.DockerContainerConfig, error)
	UpdateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error
	DeleteDockerContainer(ctx context.Context, id uuid.UUID) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// DockerBackupHandler handles Docker backup-related HTTP endpoints.
type DockerBackupHandler struct {
	store     DockerBackupStore
	discovery *docker.DiscoveryService
	checker   *license.FeatureChecker
	parser    *docker.LabelParser
	logger    zerolog.Logger
}

// NewDockerBackupHandler creates a new DockerBackupHandler.
func NewDockerBackupHandler(store DockerBackupStore, discovery *docker.DiscoveryService, checker *license.FeatureChecker, logger zerolog.Logger) *DockerBackupHandler {
	return &DockerBackupHandler{
		store:     store,
		discovery: discovery,
		checker:   checker,
		parser:    docker.NewLabelParser(),
		logger:    logger.With().Str("component", "docker_backup_handler").Logger(),
	}
}

// RegisterRoutes registers Docker backup routes on the given router group.
func (h *DockerBackupHandler) RegisterRoutes(r *gin.RouterGroup) {
	dockerGroup := r.Group("/docker")
	{
		dockerGroup.GET("/containers", h.ListContainers)
		dockerGroup.GET("/containers/:id", h.GetContainer)
		dockerGroup.PUT("/containers/:id", h.UpdateContainer)
		dockerGroup.DELETE("/containers/:id", h.DeleteContainer)
		dockerGroup.POST("/containers/:id/override", h.SetOverride)
		dockerGroup.DELETE("/containers/:id/override", h.ClearOverride)
		dockerGroup.POST("/containers/:id/refresh", h.RefreshContainer)
		dockerGroup.GET("/volumes", h.ListVolumes)
		dockerGroup.POST("/backup", h.TriggerBackup)
		dockerGroup.GET("/status", h.DaemonStatus)
		dockerGroup.POST("/discover", h.DiscoverContainers)
		dockerGroup.GET("/labels/docs", h.GetLabelDocs)
		dockerGroup.GET("/labels/examples/compose", h.GetComposeExample)
		dockerGroup.GET("/labels/examples/run", h.GetRunExample)
		dockerGroup.POST("/labels/validate", h.ValidateLabels)
	}
}

// ListContainers returns Docker containers for a given agent.
//
//	@Summary		List Docker containers
//	@Description	Returns all Docker containers on the specified agent
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Success		200			{object}	map[string][]DockerContainer
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

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	containers, err := h.store.GetDockerContainers(c.Request.Context(), user.CurrentOrgID, agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list docker containers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker containers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"containers": containers})
}

// ListVolumes returns Docker volumes for a given agent.
//
//	@Summary		List Docker volumes
//	@Description	Returns all Docker volumes on the specified agent
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Success		200			{object}	map[string][]DockerVolume
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/volumes [get]
func (h *DockerBackupHandler) ListVolumes(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	volumes, err := h.store.GetDockerVolumes(c.Request.Context(), user.CurrentOrgID, agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list docker volumes")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker volumes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"volumes": volumes})
}

// TriggerBackup triggers a Docker backup for the specified containers and volumes.
//
//	@Summary		Trigger Docker backup
//	@Description	Queues a Docker backup for selected containers and volumes
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			request	body		DockerBackupRequest	true	"Backup details"
//	@Success		202		{object}	DockerBackupResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/backup [post]
func (h *DockerBackupHandler) TriggerBackup(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureDockerBackup) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req DockerBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.ContainerIDs) == 0 && len(req.VolumeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one container or volume must be selected"})
		return
	}

	params := &models.DockerBackupParams{
		AgentID:      req.AgentID,
		RepositoryID: req.RepositoryID,
		ContainerIDs: req.ContainerIDs,
		VolumeNames:  req.VolumeNames,
	}

	resp, err := h.store.CreateDockerBackup(c.Request.Context(), user.CurrentOrgID, params)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", req.AgentID).Msg("failed to trigger docker backup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger docker backup"})
		return
	}

	h.logger.Info().
		Str("backup_id", resp.ID).
		Str("agent_id", req.AgentID).
		Int("containers", len(req.ContainerIDs)).
		Int("volumes", len(req.VolumeNames)).
		Msg("docker backup triggered")

	c.JSON(http.StatusAccepted, resp)
}

// DaemonStatus returns the Docker daemon status for a given agent.
//
//	@Summary		Get Docker daemon status
//	@Description	Returns Docker daemon status on the specified agent
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Success		200			{object}	DockerDaemonStatus
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/status [get]
func (h *DockerBackupHandler) DaemonStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	status, err := h.store.GetDockerDaemonStatus(c.Request.Context(), user.CurrentOrgID, agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get docker daemon status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get docker daemon status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ContainerDiscoveryRequest is the request body for container discovery.
type ContainerDiscoveryRequest struct {
	AgentID    uuid.UUID              `json:"agent_id" binding:"required"`
	Containers []ContainerInfoRequest `json:"containers" binding:"required"`
}

// ContainerInfoRequest represents container info in a discovery request.
type ContainerInfoRequest struct {
	ID     string             `json:"id" binding:"required"`
	Name   string             `json:"name" binding:"required"`
	Image  string             `json:"image" binding:"required"`
	Labels map[string]string  `json:"labels"`
	Mounts []MountInfoRequest `json:"mounts,omitempty"`
	Status string             `json:"status,omitempty"`
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
		return fmt.Errorf("container agent org mismatch")
	}

	return nil
}
