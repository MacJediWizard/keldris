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

// DockerRestoreStore defines the interface for Docker restore persistence operations.
type DockerRestoreStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
	CreateDockerRestore(ctx context.Context, restore *models.DockerRestore) error
	UpdateDockerRestore(ctx context.Context, restore *models.DockerRestore) error
	GetDockerRestoreByID(ctx context.Context, id uuid.UUID) (*models.DockerRestore, error)
	GetDockerRestoresByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRestore, error)
	GetDockerRestoresByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerRestore, error)
}

// DockerRestoreHandler handles Docker restore HTTP endpoints.
type DockerRestoreHandler struct {
	store  DockerRestoreStore
	logger zerolog.Logger
}

// NewDockerRestoreHandler creates a new DockerRestoreHandler.
func NewDockerRestoreHandler(store DockerRestoreStore, logger zerolog.Logger) *DockerRestoreHandler {
	return &DockerRestoreHandler{
		store:  store,
		logger: logger.With().Str("component", "docker_restore_handler").Logger(),
	}
}

// RegisterRoutes registers Docker restore routes on the given router group.
func (h *DockerRestoreHandler) RegisterRoutes(r *gin.RouterGroup) {
	dockerRestores := r.Group("/docker-restores")
	{
		dockerRestores.GET("", h.ListDockerRestores)
		dockerRestores.POST("", h.CreateDockerRestore)
		dockerRestores.POST("/preview", h.PreviewDockerRestore)
		dockerRestores.GET("/snapshot/:snapshot_id/containers", h.ListContainersInSnapshot)
		dockerRestores.GET("/snapshot/:snapshot_id/volumes", h.ListVolumesInSnapshot)
		dockerRestores.GET("/:id", h.GetDockerRestore)
		dockerRestores.GET("/:id/progress", h.GetDockerRestoreProgress)
		dockerRestores.POST("/:id/cancel", h.CancelDockerRestore)
	}
}

// DockerRestoreTargetRequest represents the Docker host target in API requests.
type DockerRestoreTargetRequest struct {
	Type      string `json:"type" binding:"required"` // "local" or "remote"
	Host      string `json:"host,omitempty"`          // For remote: docker host URL
	CertPath  string `json:"cert_path,omitempty"`     // For remote: TLS cert path
	TLSVerify bool   `json:"tls_verify,omitempty"`
}

// CreateDockerRestoreRequest is the request body for creating a Docker restore job.
type CreateDockerRestoreRequest struct {
	SnapshotID        string                      `json:"snapshot_id" binding:"required"`
	AgentID           string                      `json:"agent_id" binding:"required"`
	RepositoryID      string                      `json:"repository_id" binding:"required"`
	ContainerName     string                      `json:"container_name,omitempty"`
	VolumeName        string                      `json:"volume_name,omitempty"`
	NewContainerName  string                      `json:"new_container_name,omitempty"`
	NewVolumeName     string                      `json:"new_volume_name,omitempty"`
	Target            *DockerRestoreTargetRequest `json:"target,omitempty"`
	OverwriteExisting bool                        `json:"overwrite_existing,omitempty"`
	StartAfterRestore bool                        `json:"start_after_restore,omitempty"`
	VerifyStart       bool                        `json:"verify_start,omitempty"`
}

// DockerRestorePreviewRequest is the request body for previewing a Docker restore.
type DockerRestorePreviewRequest struct {
	SnapshotID    string                      `json:"snapshot_id" binding:"required"`
	AgentID       string                      `json:"agent_id" binding:"required"`
	RepositoryID  string                      `json:"repository_id" binding:"required"`
	ContainerName string                      `json:"container_name,omitempty"`
	VolumeName    string                      `json:"volume_name,omitempty"`
	Target        *DockerRestoreTargetRequest `json:"target,omitempty"`
}

// DockerRestoreProgressResponse represents Docker restore progress in API responses.
type DockerRestoreProgressResponse struct {
	Status          string  `json:"status"`
	CurrentStep     string  `json:"current_step"`
	TotalSteps      int     `json:"total_steps"`
	CompletedSteps  int     `json:"completed_steps"`
	PercentComplete float64 `json:"percent_complete"`
	TotalBytes      int64   `json:"total_bytes"`
	RestoredBytes   int64   `json:"restored_bytes"`
	CurrentVolume   string  `json:"current_volume,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
}

// DockerContainerResponse represents a Docker container in API responses.
type DockerContainerResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Volumes   []string `json:"volumes,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Networks  []string `json:"networks,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// DockerVolumeResponse represents a Docker volume in API responses.
type DockerVolumeResponse struct {
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

// DockerRestoreConflictResponse represents a restore conflict in API responses.
type DockerRestoreConflictResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	ExistingID  string `json:"existing_id,omitempty"`
	Description string `json:"description"`
}

// DockerRestorePlanResponse represents a restore plan preview in API responses.
type DockerRestorePlanResponse struct {
	Container      *DockerContainerResponse        `json:"container,omitempty"`
	Volumes        []DockerVolumeResponse          `json:"volumes,omitempty"`
	TotalSizeBytes int64                           `json:"total_size_bytes"`
	Conflicts      []DockerRestoreConflictResponse `json:"conflicts,omitempty"`
	Dependencies   []string                        `json:"dependencies,omitempty"`
}

// DockerRestoreResponse represents a Docker restore job in API responses.
type DockerRestoreResponse struct {
	ID                  string                         `json:"id"`
	AgentID             string                         `json:"agent_id"`
	RepositoryID        string                         `json:"repository_id"`
	SnapshotID          string                         `json:"snapshot_id"`
	ContainerName       string                         `json:"container_name,omitempty"`
	VolumeName          string                         `json:"volume_name,omitempty"`
	NewContainerName    string                         `json:"new_container_name,omitempty"`
	NewVolumeName       string                         `json:"new_volume_name,omitempty"`
	Target              *DockerRestoreTargetRequest    `json:"target,omitempty"`
	OverwriteExisting   bool                           `json:"overwrite_existing"`
	StartAfterRestore   bool                           `json:"start_after_restore"`
	VerifyStart         bool                           `json:"verify_start"`
	Status              string                         `json:"status"`
	Progress            *DockerRestoreProgressResponse `json:"progress,omitempty"`
	RestoredContainerID string                         `json:"restored_container_id,omitempty"`
	RestoredVolumes     []string                       `json:"restored_volumes,omitempty"`
	StartVerified       bool                           `json:"start_verified"`
	Warnings            []string                       `json:"warnings,omitempty"`
	StartedAt           string                         `json:"started_at,omitempty"`
	CompletedAt         string                         `json:"completed_at,omitempty"`
	ErrorMessage        string                         `json:"error_message,omitempty"`
	CreatedAt           string                         `json:"created_at"`
}

func toDockerRestoreResponse(r *models.DockerRestore) DockerRestoreResponse {
	resp := DockerRestoreResponse{
		ID:                  r.ID.String(),
		AgentID:             r.AgentID.String(),
		RepositoryID:        r.RepositoryID.String(),
		SnapshotID:          r.SnapshotID,
		ContainerName:       r.ContainerName,
		VolumeName:          r.VolumeName,
		NewContainerName:    r.NewContainerName,
		NewVolumeName:       r.NewVolumeName,
		OverwriteExisting:   r.OverwriteExisting,
		StartAfterRestore:   r.StartAfterRestore,
		VerifyStart:         r.VerifyStart,
		Status:              string(r.Status),
		RestoredContainerID: r.RestoredContainerID,
		RestoredVolumes:     r.RestoredVolumes,
		StartVerified:       r.StartVerified,
		Warnings:            r.Warnings,
		ErrorMessage:        r.ErrorMessage,
		CreatedAt:           r.CreatedAt.Format(time.RFC3339),
	}

	if r.Target != nil {
		resp.Target = &DockerRestoreTargetRequest{
			Type:      string(r.Target.Type),
			Host:      r.Target.Host,
			CertPath:  r.Target.CertPath,
			TLSVerify: r.Target.TLSVerify,
		}
	}

	if r.StartedAt != nil {
		resp.StartedAt = r.StartedAt.Format(time.RFC3339)
	}
	if r.CompletedAt != nil {
		resp.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}

	if r.Progress != nil {
		resp.Progress = &DockerRestoreProgressResponse{
			Status:          r.Progress.Status,
			CurrentStep:     r.Progress.CurrentStep,
			TotalSteps:      r.Progress.TotalSteps,
			CompletedSteps:  r.Progress.CompletedSteps,
			PercentComplete: r.Progress.PercentComplete(),
			TotalBytes:      r.Progress.TotalBytes,
			RestoredBytes:   r.Progress.RestoredBytes,
			CurrentVolume:   r.Progress.CurrentVolume,
			ErrorMessage:    r.Progress.ErrorMessage,
		}
	}

	return resp
}

// CreateDockerRestore creates a new Docker restore job.
//
//	@Summary		Create Docker restore
//	@Description	Creates a new Docker container or volume restore job
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateDockerRestoreRequest	true	"Docker restore details"
//	@Success		201		{object}	DockerRestoreResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores [post]
func (h *DockerRestoreHandler) CreateDockerRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateDockerRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate at least one of container or volume is specified
	if req.ContainerName == "" && req.VolumeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either container_name or volume_name must be specified"})
		return
	}

	// Parse IDs
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	repositoryID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	// Verify user access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify access to agent
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify repository access
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repositoryID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Verify snapshot exists
	_, err = h.store.GetBackupBySnapshotID(c.Request.Context(), req.SnapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Validate target if provided
	if req.Target != nil {
		targetType := models.DockerRestoreTargetType(req.Target.Type)
		if targetType != models.DockerTargetLocal && targetType != models.DockerTargetRemote {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target type: must be 'local' or 'remote'"})
			return
		}
		if targetType == models.DockerTargetRemote && req.Target.Host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "host is required for remote target"})
			return
		}
	}

	// Create Docker restore job
	restore := models.NewDockerRestore(dbUser.OrgID, agentID, repositoryID, req.SnapshotID)
	restore.ContainerName = req.ContainerName
	restore.VolumeName = req.VolumeName
	restore.NewContainerName = req.NewContainerName
	restore.NewVolumeName = req.NewVolumeName
	restore.OverwriteExisting = req.OverwriteExisting
	restore.StartAfterRestore = req.StartAfterRestore
	restore.VerifyStart = req.VerifyStart

	if req.Target != nil {
		restore.Target = &models.DockerRestoreTarget{
			Type:      models.DockerRestoreTargetType(req.Target.Type),
			Host:      req.Target.Host,
			CertPath:  req.Target.CertPath,
			TLSVerify: req.Target.TLSVerify,
		}
	} else {
		// Default to local target
		restore.Target = &models.DockerRestoreTarget{
			Type: models.DockerTargetLocal,
		}
	}

	if err := h.store.CreateDockerRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Msg("failed to create Docker restore job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create Docker restore job"})
		return
	}

	h.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("container_name", req.ContainerName).
		Str("volume_name", req.VolumeName).
		Msg("Docker restore job created")

	c.JSON(http.StatusCreated, toDockerRestoreResponse(restore))
}

// PreviewDockerRestore previews what would be restored without performing the restore.
//
//	@Summary		Preview Docker restore
//	@Description	Returns a preview of containers and volumes that would be restored, including potential conflicts
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		DockerRestorePreviewRequest	true	"Preview request"
//	@Success		200		{object}	DockerRestorePlanResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/preview [post]
func (h *DockerRestoreHandler) PreviewDockerRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req DockerRestorePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse IDs
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	repositoryID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	// Verify user access to agent
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify repository access
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repositoryID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Verify snapshot exists
	_, err = h.store.GetBackupBySnapshotID(c.Request.Context(), req.SnapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	h.logger.Info().
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("container_name", req.ContainerName).
		Str("volume_name", req.VolumeName).
		Msg("Docker restore preview requested")

	// In a full implementation, this would communicate with the agent
	// to run the actual preview. For now, return a placeholder response.
	c.JSON(http.StatusOK, DockerRestorePlanResponse{
		Container:      nil,
		Volumes:        []DockerVolumeResponse{},
		TotalSizeBytes: 0,
		Conflicts:      []DockerRestoreConflictResponse{},
		Dependencies:   []string{},
	})
}

// ListContainersInSnapshot lists all containers available in a snapshot.
//
//	@Summary		List containers in snapshot
//	@Description	Returns all Docker containers backed up in a snapshot
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			snapshot_id	path		string	true	"Snapshot ID"
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Param			repository_id	query	string	true	"Repository ID"
//	@Success		200			{object}	map[string][]DockerContainerResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/snapshot/{snapshot_id}/containers [get]
func (h *DockerRestoreHandler) ListContainersInSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("snapshot_id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id query parameter is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Verify user access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify snapshot exists
	_, err = h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	h.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("agent_id", agentIDParam).
		Msg("listing containers in snapshot")

	// In a full implementation, this would communicate with the agent
	// to list containers in the snapshot. For now, return a placeholder.
	c.JSON(http.StatusOK, gin.H{
		"containers": []DockerContainerResponse{},
		"message":    "Container listing requires agent communication. Containers will be populated when agent connectivity is implemented.",
	})
}

// ListVolumesInSnapshot lists all volumes available in a snapshot.
//
//	@Summary		List volumes in snapshot
//	@Description	Returns all Docker volumes backed up in a snapshot
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			snapshot_id	path		string	true	"Snapshot ID"
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Param			repository_id	query	string	true	"Repository ID"
//	@Success		200			{object}	map[string][]DockerVolumeResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/snapshot/{snapshot_id}/volumes [get]
func (h *DockerRestoreHandler) ListVolumesInSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("snapshot_id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id query parameter is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Verify user access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify snapshot exists
	_, err = h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	h.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("agent_id", agentIDParam).
		Msg("listing volumes in snapshot")

	// In a full implementation, this would communicate with the agent
	// to list volumes in the snapshot. For now, return a placeholder.
	c.JSON(http.StatusOK, gin.H{
		"volumes": []DockerVolumeResponse{},
		"message": "Volume listing requires agent communication. Volumes will be populated when agent connectivity is implemented.",
	})
}

// GetDockerRestore returns a specific Docker restore job by ID.
//
//	@Summary		Get Docker restore
//	@Description	Returns a specific Docker restore job by ID
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	DockerRestoreResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/{id} [get]
func (h *DockerRestoreHandler) GetDockerRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore ID"})
		return
	}

	restore, err := h.store.GetDockerRestoreByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Verify access through organization
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if restore.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	c.JSON(http.StatusOK, toDockerRestoreResponse(restore))
}

// GetDockerRestoreProgress returns the progress of a Docker restore operation.
//
//	@Summary		Get Docker restore progress
//	@Description	Returns the current progress of a Docker restore operation
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	DockerRestoreProgressResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/{id}/progress [get]
func (h *DockerRestoreHandler) GetDockerRestoreProgress(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore ID"})
		return
	}

	restore, err := h.store.GetDockerRestoreByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Verify access through organization
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if restore.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Return progress
	if restore.Progress == nil {
		c.JSON(http.StatusOK, DockerRestoreProgressResponse{
			Status:          string(restore.Status),
			CurrentStep:     "",
			TotalSteps:      0,
			CompletedSteps:  0,
			PercentComplete: 0,
			TotalBytes:      0,
			RestoredBytes:   0,
		})
		return
	}

	c.JSON(http.StatusOK, DockerRestoreProgressResponse{
		Status:          restore.Progress.Status,
		CurrentStep:     restore.Progress.CurrentStep,
		TotalSteps:      restore.Progress.TotalSteps,
		CompletedSteps:  restore.Progress.CompletedSteps,
		PercentComplete: restore.Progress.PercentComplete(),
		TotalBytes:      restore.Progress.TotalBytes,
		RestoredBytes:   restore.Progress.RestoredBytes,
		CurrentVolume:   restore.Progress.CurrentVolume,
		ErrorMessage:    restore.Progress.ErrorMessage,
	})
}

// ListDockerRestores returns all Docker restore jobs for the authenticated user's organization.
//
//	@Summary		List Docker restores
//	@Description	Returns all Docker restore jobs for the current organization
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	map[string][]DockerRestoreResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores [get]
func (h *DockerRestoreHandler) ListDockerRestores(c *gin.Context) {
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

	var restores []*models.DockerRestore

	// Filter by agent if specified
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent access
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil || agent.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		restores, err = h.store.GetDockerRestoresByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to list Docker restores by agent")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list Docker restores"})
			return
		}
	} else {
		restores, err = h.store.GetDockerRestoresByOrgID(c.Request.Context(), dbUser.OrgID)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to list Docker restores")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list Docker restores"})
			return
		}
	}

	// Apply status filter if provided
	statusFilter := c.Query("status")
	var responses []DockerRestoreResponse
	for _, restore := range restores {
		if statusFilter != "" && string(restore.Status) != statusFilter {
			continue
		}
		responses = append(responses, toDockerRestoreResponse(restore))
	}

	c.JSON(http.StatusOK, gin.H{"docker_restores": responses})
}

// CancelDockerRestore cancels a pending or running Docker restore job.
//
//	@Summary		Cancel Docker restore
//	@Description	Cancels a pending or running Docker restore job
//	@Tags			Docker Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	DockerRestoreResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-restores/{id}/cancel [post]
func (h *DockerRestoreHandler) CancelDockerRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore ID"})
		return
	}

	restore, err := h.store.GetDockerRestoreByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Verify access through organization
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if restore.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Check if restore can be canceled
	if restore.IsComplete() {
		c.JSON(http.StatusConflict, gin.H{"error": "restore has already completed"})
		return
	}

	// Cancel the restore
	restore.Cancel()
	if err := h.store.UpdateDockerRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Str("restore_id", id.String()).Msg("failed to cancel Docker restore")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel restore"})
		return
	}

	h.logger.Info().
		Str("restore_id", id.String()).
		Str("canceled_by", user.ID.String()).
		Msg("Docker restore canceled")

	c.JSON(http.StatusOK, toDockerRestoreResponse(restore))
}
