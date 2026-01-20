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

// SnapshotStore defines the interface for snapshot-related persistence operations.
type SnapshotStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	CreateRestore(ctx context.Context, restore *models.Restore) error
	GetRestoresByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Restore, error)
	GetRestoreByID(ctx context.Context, id uuid.UUID) (*models.Restore, error)
}

// SnapshotsHandler handles snapshot and restore HTTP endpoints.
type SnapshotsHandler struct {
	store  SnapshotStore
	logger zerolog.Logger
}

// NewSnapshotsHandler creates a new SnapshotsHandler.
func NewSnapshotsHandler(store SnapshotStore, logger zerolog.Logger) *SnapshotsHandler {
	return &SnapshotsHandler{
		store:  store,
		logger: logger.With().Str("component", "snapshots_handler").Logger(),
	}
}

// RegisterRoutes registers snapshot and restore routes on the given router group.
func (h *SnapshotsHandler) RegisterRoutes(r *gin.RouterGroup) {
	snapshots := r.Group("/snapshots")
	{
		snapshots.GET("", h.ListSnapshots)
		snapshots.GET("/:id", h.GetSnapshot)
		snapshots.GET("/:id/files", h.ListFiles)
	}

	restores := r.Group("/restores")
	{
		restores.GET("", h.ListRestores)
		restores.POST("", h.CreateRestore)
		restores.GET("/:id", h.GetRestore)
	}
}

// SnapshotResponse represents a snapshot in API responses.
type SnapshotResponse struct {
	ID           string   `json:"id"`
	ShortID      string   `json:"short_id"`
	Time         string   `json:"time"`
	Hostname     string   `json:"hostname"`
	Paths        []string `json:"paths"`
	AgentID      string   `json:"agent_id"`
	RepositoryID string   `json:"repository_id"`
	BackupID     string   `json:"backup_id,omitempty"`
	SizeBytes    *int64   `json:"size_bytes,omitempty"`
}

// ListSnapshots returns all snapshots for the authenticated user's organization.
//
//	@Summary		List snapshots
//	@Description	Returns all backup snapshots for the current organization
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			agent_id		query		string	false	"Filter by agent ID"
//	@Param			repository_id	query		string	false	"Filter by repository ID"
//	@Success		200				{object}	map[string][]SnapshotResponse
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots [get]
func (h *SnapshotsHandler) ListSnapshots(c *gin.Context) {
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

	// Get all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list snapshots"})
		return
	}

	// Apply agent_id filter if provided
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}
		// Filter to just this agent
		var filtered []*models.Agent
		for _, a := range agents {
			if a.ID == agentID {
				filtered = append(filtered, a)
				break
			}
		}
		agents = filtered
	}

	// Get all backups for the agents (which contain snapshot info)
	var snapshots []SnapshotResponse
	for _, agent := range agents {
		backups, err := h.store.GetBackupsByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to get backups for agent")
			continue
		}

		for _, backup := range backups {
			// Only include completed backups with snapshot IDs
			if backup.Status != models.BackupStatusCompleted || backup.SnapshotID == "" {
				continue
			}

			// Get schedule to find repository ID
			schedule, err := h.store.GetScheduleByID(c.Request.Context(), backup.ScheduleID)
			if err != nil {
				continue
			}

			// Apply repository_id filter if provided
			repositoryIDParam := c.Query("repository_id")
			if repositoryIDParam != "" {
				repoID, err := uuid.Parse(repositoryIDParam)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
					return
				}
				if schedule.RepositoryID != repoID {
					continue
				}
			}

			shortID := backup.SnapshotID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			snapshots = append(snapshots, SnapshotResponse{
				ID:           backup.SnapshotID,
				ShortID:      shortID,
				Time:         backup.StartedAt.Format(time.RFC3339),
				Hostname:     agent.Hostname,
				Paths:        schedule.Paths,
				AgentID:      agent.ID.String(),
				RepositoryID: schedule.RepositoryID.String(),
				BackupID:     backup.ID.String(),
				SizeBytes:    backup.SizeBytes,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// GetSnapshot returns a specific snapshot by ID.
//
//	@Summary		Get snapshot
//	@Description	Returns a specific snapshot by ID
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Snapshot ID"
//	@Success		200	{object}	SnapshotResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id} [get]
func (h *SnapshotsHandler) GetSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	backup, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Verify access through agent
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), backup.ScheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get snapshot details"})
		return
	}

	shortID := backup.SnapshotID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	c.JSON(http.StatusOK, SnapshotResponse{
		ID:           backup.SnapshotID,
		ShortID:      shortID,
		Time:         backup.StartedAt.Format(time.RFC3339),
		Hostname:     agent.Hostname,
		Paths:        schedule.Paths,
		AgentID:      agent.ID.String(),
		RepositoryID: schedule.RepositoryID.String(),
		BackupID:     backup.ID.String(),
		SizeBytes:    backup.SizeBytes,
	})
}

// SnapshotFileResponse represents a file in a snapshot.
type SnapshotFileResponse struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"` // "file" or "dir"
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ListFiles returns files in a snapshot.
//
//	@Summary		List snapshot files
//	@Description	Returns files in a snapshot, optionally filtered by path
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Snapshot ID"
//	@Param			path	query		string	false	"Filter to specific directory (default: root)"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/files [get]
func (h *SnapshotsHandler) ListFiles(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	backup, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Verify access through agent
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Get the path prefix for filtering
	pathPrefix := c.Query("path")

	// Note: In a full implementation, this would call the agent to list files
	// from the actual Restic repository. For now, we return a placeholder response
	// indicating the functionality is available but requires agent communication.
	h.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("path_prefix", pathPrefix).
		Msg("file listing requested")

	// Placeholder response - in production this would query the agent
	c.JSON(http.StatusOK, gin.H{
		"files":       []SnapshotFileResponse{},
		"snapshot_id": snapshotID,
		"path":        pathPrefix,
		"message":     "File listing requires agent communication. Files will be populated when agent connectivity is implemented.",
	})
}

// CreateRestoreRequest is the request body for creating a restore job.
type CreateRestoreRequest struct {
	SnapshotID   string   `json:"snapshot_id" binding:"required"`
	AgentID      string   `json:"agent_id" binding:"required"`
	RepositoryID string   `json:"repository_id" binding:"required"`
	TargetPath   string   `json:"target_path" binding:"required"`
	IncludePaths []string `json:"include_paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty"`
}

// RestoreResponse represents a restore job in API responses.
type RestoreResponse struct {
	ID           string   `json:"id"`
	AgentID      string   `json:"agent_id"`
	RepositoryID string   `json:"repository_id"`
	SnapshotID   string   `json:"snapshot_id"`
	TargetPath   string   `json:"target_path"`
	IncludePaths []string `json:"include_paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty"`
	Status       string   `json:"status"`
	StartedAt    string   `json:"started_at,omitempty"`
	CompletedAt  string   `json:"completed_at,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	CreatedAt    string   `json:"created_at"`
}

func toRestoreResponse(r *models.Restore) RestoreResponse {
	resp := RestoreResponse{
		ID:           r.ID.String(),
		AgentID:      r.AgentID.String(),
		RepositoryID: r.RepositoryID.String(),
		SnapshotID:   r.SnapshotID,
		TargetPath:   r.TargetPath,
		IncludePaths: r.IncludePaths,
		ExcludePaths: r.ExcludePaths,
		Status:       string(r.Status),
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
	}
	if r.StartedAt != nil {
		resp.StartedAt = r.StartedAt.Format(time.RFC3339)
	}
	if r.CompletedAt != nil {
		resp.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}
	return resp
}

// CreateRestore creates a new restore job.
//
//	@Summary		Create restore
//	@Description	Creates a new restore job to restore files from a snapshot
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateRestoreRequest	true	"Restore details"
//	@Success		201		{object}	RestoreResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores [post]
func (h *SnapshotsHandler) CreateRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateRestoreRequest
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

	// Create restore job
	restore := models.NewRestore(agentID, repositoryID, req.SnapshotID, req.TargetPath, req.IncludePaths, req.ExcludePaths)

	if err := h.store.CreateRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Msg("failed to create restore job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create restore job"})
		return
	}

	h.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("target_path", req.TargetPath).
		Msg("restore job created")

	c.JSON(http.StatusCreated, toRestoreResponse(restore))
}

// ListRestores returns all restore jobs for the authenticated user's organization.
//
//	@Summary		List restores
//	@Description	Returns all restore jobs for the current organization
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	map[string][]RestoreResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores [get]
func (h *SnapshotsHandler) ListRestores(c *gin.Context) {
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

	// Get all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list restores"})
		return
	}

	// Apply agent_id filter if provided
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}
		// Filter to just this agent
		var filtered []*models.Agent
		for _, a := range agents {
			if a.ID == agentID {
				filtered = append(filtered, a)
				break
			}
		}
		agents = filtered
	}

	statusFilter := c.Query("status")

	var restores []RestoreResponse
	for _, agent := range agents {
		agentRestores, err := h.store.GetRestoresByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to get restores for agent")
			continue
		}

		for _, restore := range agentRestores {
			if statusFilter != "" && string(restore.Status) != statusFilter {
				continue
			}
			restores = append(restores, toRestoreResponse(restore))
		}
	}

	c.JSON(http.StatusOK, gin.H{"restores": restores})
}

// GetRestore returns a specific restore job by ID.
//
//	@Summary		Get restore
//	@Description	Returns a specific restore job by ID
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	RestoreResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/{id} [get]
func (h *SnapshotsHandler) GetRestore(c *gin.Context) {
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

	restore, err := h.store.GetRestoreByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	// Verify access through agent
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), restore.AgentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "restore not found"})
		return
	}

	c.JSON(http.StatusOK, toRestoreResponse(restore))
}
