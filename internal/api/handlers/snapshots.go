package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/crypto"
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
	GetRepositoryKeyByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryKey, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	CreateRestore(ctx context.Context, restore *models.Restore) error
	GetRestoresByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Restore, error)
	GetRestoreByID(ctx context.Context, id uuid.UUID) (*models.Restore, error)
	// Snapshot comment methods
	CreateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error
	GetSnapshotCommentsBySnapshotID(ctx context.Context, snapshotID string, orgID uuid.UUID) ([]*models.SnapshotComment, error)
	GetSnapshotCommentByID(ctx context.Context, id uuid.UUID) (*models.SnapshotComment, error)
	DeleteSnapshotComment(ctx context.Context, id uuid.UUID) error
	GetSnapshotCommentCounts(ctx context.Context, snapshotIDs []string, orgID uuid.UUID) (map[string]int, error)
}

// SnapshotsHandler handles snapshot and restore HTTP endpoints.
type SnapshotsHandler struct {
	store      SnapshotStore
	keyManager *crypto.KeyManager
	restic     *backup.Restic
	logger     zerolog.Logger
}

// NewSnapshotsHandler creates a new SnapshotsHandler.
func NewSnapshotsHandler(store SnapshotStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *SnapshotsHandler {
	return &SnapshotsHandler{
		store:      store,
		keyManager: keyManager,
		restic:     backup.NewRestic(logger),
		logger:     logger.With().Str("component", "snapshots_handler").Logger(),
	}
}

// buildResticConfig builds a ResticConfig from a backup's repository credentials.
func (h *SnapshotsHandler) buildResticConfig(ctx context.Context, repositoryID uuid.UUID) (*backup.ResticConfig, error) {
	repo, err := h.store.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	configJSON, err := h.keyManager.Decrypt(repo.ConfigEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}

	backend, err := backends.ParseBackend(repo.Type, configJSON)
	if err != nil {
		return nil, fmt.Errorf("parse backend: %w", err)
	}

	repoKey, err := h.store.GetRepositoryKeyByRepositoryID(ctx, repo.ID)
	if err != nil {
		return nil, fmt.Errorf("get repository key: %w", err)
	}

	password, err := h.keyManager.Decrypt(repoKey.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt password: %w", err)
	}

	cfg := backend.ToResticConfig(string(password))
	return &cfg, nil
}

// RegisterRoutes registers snapshot and restore routes on the given router group.
func (h *SnapshotsHandler) RegisterRoutes(r *gin.RouterGroup) {
	snapshots := r.Group("/snapshots")
	{
		snapshots.GET("", h.ListSnapshots)
		snapshots.GET("/:id", h.GetSnapshot)
		snapshots.GET("/:id/files", h.ListFiles)
		snapshots.GET("/:id/comments", h.ListSnapshotComments)
		snapshots.POST("/:id/comments", h.CreateSnapshotComment)
		snapshots.GET("/compare", h.CompareSnapshots)
	}

	// Comments resource for direct access
	comments := r.Group("/comments")
	{
		comments.DELETE("/:id", h.DeleteSnapshotComment)
	}

	restores := r.Group("/restores")
	{
		restores.GET("", h.ListRestores)
		restores.POST("", h.CreateRestore)
		restores.POST("/preview", h.PreviewRestore)
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
				// Check if the backup's repository matches the filter
				if backup.RepositoryID == nil || *backup.RepositoryID != repoID {
					continue
				}
			}

			shortID := backup.SnapshotID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			repoIDStr := ""
			if backup.RepositoryID != nil {
				repoIDStr = backup.RepositoryID.String()
			}

			snapshots = append(snapshots, SnapshotResponse{
				ID:           backup.SnapshotID,
				ShortID:      shortID,
				Time:         backup.StartedAt.Format(time.RFC3339),
				Hostname:     agent.Hostname,
				Paths:        schedule.Paths,
				AgentID:      agent.ID.String(),
				RepositoryID: repoIDStr,
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

	repoIDStr := ""
	if backup.RepositoryID != nil {
		repoIDStr = backup.RepositoryID.String()
	}

	c.JSON(http.StatusOK, SnapshotResponse{
		ID:           backup.SnapshotID,
		ShortID:      shortID,
		Time:         backup.StartedAt.Format(time.RFC3339),
		Hostname:     agent.Hostname,
		Paths:        schedule.Paths,
		AgentID:      agent.ID.String(),
		RepositoryID: repoIDStr,
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

// SnapshotDiffChangeType represents the type of change in a diff.
type SnapshotDiffChangeType string

const (
	DiffChangeAdded    SnapshotDiffChangeType = "added"
	DiffChangeRemoved  SnapshotDiffChangeType = "removed"
	DiffChangeModified SnapshotDiffChangeType = "modified"
)

// SnapshotDiffEntry represents a single changed file/directory in the diff.
type SnapshotDiffEntry struct {
	Path       string                 `json:"path"`
	ChangeType SnapshotDiffChangeType `json:"change_type"`
	Type       string                 `json:"type"` // "file" or "dir"
	OldSize    int64                  `json:"old_size,omitempty"`
	NewSize    int64                  `json:"new_size,omitempty"`
	SizeChange int64                  `json:"size_change,omitempty"`
}

// SnapshotDiffStats contains summary statistics for a diff operation.
type SnapshotDiffStats struct {
	FilesAdded       int   `json:"files_added"`
	FilesRemoved     int   `json:"files_removed"`
	FilesModified    int   `json:"files_modified"`
	DirsAdded        int   `json:"dirs_added"`
	DirsRemoved      int   `json:"dirs_removed"`
	TotalSizeAdded   int64 `json:"total_size_added"`
	TotalSizeRemoved int64 `json:"total_size_removed"`
}

// SnapshotCompareResponse represents the response from comparing two snapshots.
type SnapshotCompareResponse struct {
	SnapshotID1 string              `json:"snapshot_id_1"`
	SnapshotID2 string              `json:"snapshot_id_2"`
	Snapshot1   *SnapshotResponse   `json:"snapshot_1,omitempty"`
	Snapshot2   *SnapshotResponse   `json:"snapshot_2,omitempty"`
	Stats       SnapshotDiffStats   `json:"stats"`
	Changes     []SnapshotDiffEntry `json:"changes"`
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

	if backup.RepositoryID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backup has no repository"})
		return
	}

	// Build Restic config from backup's repository
	resticCfg, err := h.buildResticConfig(c.Request.Context(), *backup.RepositoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to build restic config for file listing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository"})
		return
	}

	// List files from the actual Restic repository
	files, err := h.restic.ListFiles(c.Request.Context(), *resticCfg, snapshotID, pathPrefix)
	if err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to list files in snapshot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	// Convert to response format
	var fileResponses []SnapshotFileResponse
	for _, f := range files {
		fileResponses = append(fileResponses, SnapshotFileResponse{
			Name:    f.Name,
			Path:    f.Path,
			Type:    f.Type,
			Size:    f.Size,
			ModTime: f.ModTime.Format(time.RFC3339),
		})
	}
	if fileResponses == nil {
		fileResponses = []SnapshotFileResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"files":       fileResponses,
		"snapshot_id": snapshotID,
		"path":        pathPrefix,
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

// RestorePreviewRequest is the request body for previewing a restore operation.
type RestorePreviewRequest struct {
	SnapshotID   string   `json:"snapshot_id" binding:"required"`
	AgentID      string   `json:"agent_id" binding:"required"`
	RepositoryID string   `json:"repository_id" binding:"required"`
	TargetPath   string   `json:"target_path" binding:"required"`
	IncludePaths []string `json:"include_paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty"`
}

// RestorePreviewFileResponse represents a file in the restore preview.
type RestorePreviewFileResponse struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	Size        int64  `json:"size"`
	ModTime     string `json:"mod_time"`
	HasConflict bool   `json:"has_conflict"`
}

// RestorePreviewResponse contains the preview results.
type RestorePreviewResponse struct {
	SnapshotID      string                       `json:"snapshot_id"`
	TargetPath      string                       `json:"target_path"`
	TotalFiles      int                          `json:"total_files"`
	TotalDirs       int                          `json:"total_dirs"`
	TotalSize       int64                        `json:"total_size"`
	ConflictCount   int                          `json:"conflict_count"`
	Files           []RestorePreviewFileResponse `json:"files"`
	DiskSpaceNeeded int64                        `json:"disk_space_needed"`
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

// PreviewRestore previews what would be restored without actually restoring.
//
//	@Summary		Preview restore
//	@Description	Returns a preview of files that would be restored, including potential conflicts
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RestorePreviewRequest	true	"Preview request"
//	@Success		200		{object}	RestorePreviewResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/preview [post]
func (h *SnapshotsHandler) PreviewRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req RestorePreviewRequest
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

	h.logger.Info().
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("target_path", req.TargetPath).
		Msg("restore preview requested")

	// Note: In a full implementation, this would communicate with the agent
	// to run the actual restore preview. For now, we return a placeholder
	// response indicating the preview is available but requires agent communication.
	// The actual preview would be generated by the agent using restic's --dry-run flag.
	c.JSON(http.StatusOK, RestorePreviewResponse{
		SnapshotID:      req.SnapshotID,
		TargetPath:      req.TargetPath,
		TotalFiles:      0,
		TotalDirs:       0,
		TotalSize:       0,
		ConflictCount:   0,
		Files:           []RestorePreviewFileResponse{},
		DiskSpaceNeeded: 0,
	})
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

// SnapshotCommentResponse represents a comment in API responses.
type SnapshotCommentResponse struct {
	ID         string `json:"id"`
	SnapshotID string `json:"snapshot_id"`
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func toSnapshotCommentResponse(c *models.SnapshotComment, user *models.User) SnapshotCommentResponse {
	userName := ""
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}
	return SnapshotCommentResponse{
		ID:         c.ID.String(),
		SnapshotID: c.SnapshotID,
		UserID:     c.UserID.String(),
		UserName:   userName,
		UserEmail:  userEmail,
		Content:    c.Content,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}

// ListSnapshotComments returns all comments for a snapshot.
// GET /api/v1/snapshots/:id/comments
func (h *SnapshotsHandler) ListSnapshotComments(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	// Verify the snapshot exists and user has access
	backup, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

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

	comments, err := h.store.GetSnapshotCommentsBySnapshotID(c.Request.Context(), snapshotID, dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to list comments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}

	// Build user cache for response enrichment
	userCache := make(map[uuid.UUID]*models.User)
	var responses []SnapshotCommentResponse
	for _, comment := range comments {
		var commentUser *models.User
		if cached, ok := userCache[comment.UserID]; ok {
			commentUser = cached
		} else {
			commentUser, _ = h.store.GetUserByID(c.Request.Context(), comment.UserID)
			userCache[comment.UserID] = commentUser
		}
		responses = append(responses, toSnapshotCommentResponse(comment, commentUser))
	}

	c.JSON(http.StatusOK, gin.H{"comments": responses})
}

// CreateSnapshotComment creates a new comment on a snapshot.
// POST /api/v1/snapshots/:id/comments
func (h *SnapshotsHandler) CreateSnapshotComment(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	var req models.CreateSnapshotCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	// Verify the snapshot exists and user has access
	backup, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

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

	comment := models.NewSnapshotComment(dbUser.OrgID, snapshotID, dbUser.ID, req.Content)

	if err := h.store.CreateSnapshotComment(c.Request.Context(), comment); err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to create comment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	h.logger.Info().
		Str("comment_id", comment.ID.String()).
		Str("snapshot_id", snapshotID).
		Str("user_id", dbUser.ID.String()).
		Msg("snapshot comment created")

	c.JSON(http.StatusCreated, toSnapshotCommentResponse(comment, dbUser))
}

// DeleteSnapshotComment deletes a comment.
// DELETE /api/v1/comments/:id
func (h *SnapshotsHandler) DeleteSnapshotComment(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment ID"})
		return
	}

	comment, err := h.store.GetSnapshotCommentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	// Verify user has access (must be in same org and either own the comment or be admin)
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if comment.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
		return
	}

	// Only allow deletion by the comment author or admins
	if comment.UserID != dbUser.ID && !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own comments"})
		return
	}

	if err := h.store.DeleteSnapshotComment(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("comment_id", id.String()).Msg("failed to delete comment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}

	h.logger.Info().
		Str("comment_id", id.String()).
		Str("deleted_by", dbUser.ID.String()).
		Msg("snapshot comment deleted")

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}

// CompareSnapshots compares two snapshots and returns their differences.
// GET /api/v1/snapshots/compare?id1=xxx&id2=xxx
func (h *SnapshotsHandler) CompareSnapshots(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID1 := c.Query("id1")
	snapshotID2 := c.Query("id2")

	if snapshotID1 == "" || snapshotID2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both snapshot IDs are required (id1 and id2 query params)"})
		return
	}

	// Get user for org verification
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify access to first snapshot
	backup1, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID1)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "first snapshot not found"})
		return
	}

	agent1, err := h.store.GetAgentByID(c.Request.Context(), backup1.AgentID)
	if err != nil || agent1.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "first snapshot not found"})
		return
	}

	// Verify access to second snapshot
	backup2, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "second snapshot not found"})
		return
	}

	agent2, err := h.store.GetAgentByID(c.Request.Context(), backup2.AgentID)
	if err != nil || agent2.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "second snapshot not found"})
		return
	}

	// Get schedules for paths info
	schedule1, err := h.store.GetScheduleByID(c.Request.Context(), backup1.ScheduleID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get schedule for first snapshot")
	}

	schedule2, err := h.store.GetScheduleByID(c.Request.Context(), backup2.ScheduleID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get schedule for second snapshot")
	}

	// Build snapshot info for response
	shortID1 := snapshotID1
	if len(shortID1) > 8 {
		shortID1 = shortID1[:8]
	}
	shortID2 := snapshotID2
	if len(shortID2) > 8 {
		shortID2 = shortID2[:8]
	}

	var paths1, paths2 []string
	var repoID1, repoID2 string
	if schedule1 != nil {
		paths1 = schedule1.Paths
		if len(schedule1.Repositories) > 0 {
			repoID1 = schedule1.Repositories[0].RepositoryID.String()
		}
	}
	if schedule2 != nil {
		paths2 = schedule2.Paths
		if len(schedule2.Repositories) > 0 {
			repoID2 = schedule2.Repositories[0].RepositoryID.String()
		}
	}

	snapshot1Info := &SnapshotResponse{
		ID:           snapshotID1,
		ShortID:      shortID1,
		Time:         backup1.StartedAt.Format(time.RFC3339),
		Hostname:     agent1.Hostname,
		Paths:        paths1,
		AgentID:      agent1.ID.String(),
		RepositoryID: repoID1,
		BackupID:     backup1.ID.String(),
		SizeBytes:    backup1.SizeBytes,
	}

	snapshot2Info := &SnapshotResponse{
		ID:           snapshotID2,
		ShortID:      shortID2,
		Time:         backup2.StartedAt.Format(time.RFC3339),
		Hostname:     agent2.Hostname,
		Paths:        paths2,
		AgentID:      agent2.ID.String(),
		RepositoryID: repoID2,
		BackupID:     backup2.ID.String(),
		SizeBytes:    backup2.SizeBytes,
	}

	if backup1.RepositoryID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backup has no repository"})
		return
	}

	// Build Restic config from backup's repository
	resticCfg, err := h.buildResticConfig(c.Request.Context(), *backup1.RepositoryID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to build restic config for comparison")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository"})
		return
	}

	// Run restic diff between the two snapshots
	diffResult, err := h.restic.Diff(c.Request.Context(), *resticCfg, snapshotID1, snapshotID2)
	if err != nil {
		h.logger.Error().Err(err).
			Str("snapshot_id_1", snapshotID1).
			Str("snapshot_id_2", snapshotID2).
			Msg("failed to compare snapshots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compare snapshots"})
		return
	}

	// Convert diff entries to response format
	var changes []SnapshotDiffEntry
	for _, entry := range diffResult.Changes {
		changes = append(changes, SnapshotDiffEntry{
			Path:       entry.Path,
			ChangeType: SnapshotDiffChangeType(entry.ChangeType),
			Type:       entry.Type,
		})
	}
	if changes == nil {
		changes = []SnapshotDiffEntry{}
	}

	c.JSON(http.StatusOK, SnapshotCompareResponse{
		SnapshotID1: snapshotID1,
		SnapshotID2: snapshotID2,
		Snapshot1:   snapshot1Info,
		Snapshot2:   snapshot2Info,
		Stats: SnapshotDiffStats{
			FilesAdded:       diffResult.Stats.FilesAdded,
			FilesRemoved:     diffResult.Stats.FilesRemoved,
			FilesModified:    diffResult.Stats.FilesModified,
			DirsAdded:        diffResult.Stats.DirsAdded,
			DirsRemoved:      diffResult.Stats.DirsRemoved,
			TotalSizeAdded:   diffResult.Stats.TotalSizeAdded,
			TotalSizeRemoved: diffResult.Stats.TotalSizeRemoved,
		},
		Changes: changes,
	})
}
