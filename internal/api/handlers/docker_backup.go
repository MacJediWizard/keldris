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
}

// DockerBackupHandler handles Docker backup-related HTTP endpoints.
type DockerBackupHandler struct {
	store  DockerBackupStore
	logger zerolog.Logger
}

// NewDockerBackupHandler creates a new DockerBackupHandler.
func NewDockerBackupHandler(store DockerBackupStore, logger zerolog.Logger) *DockerBackupHandler {
	return &DockerBackupHandler{
		store:  store,
		logger: logger.With().Str("component", "docker_backup_handler").Logger(),
	}
}

// RegisterRoutes registers Docker backup routes on the given router group.
func (h *DockerBackupHandler) RegisterRoutes(r *gin.RouterGroup) {
	docker := r.Group("/docker")
	{
		docker.GET("/containers", h.ListContainers)
		docker.GET("/volumes", h.ListVolumes)
		docker.POST("/backup", h.TriggerBackup)
		docker.GET("/status", h.DaemonStatus)
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

