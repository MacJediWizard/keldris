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
	// Snapshot comment methods
	CreateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error
	GetSnapshotCommentsBySnapshotID(ctx context.Context, snapshotID string, orgID uuid.UUID) ([]*models.SnapshotComment, error)
	GetSnapshotCommentByID(ctx context.Context, id uuid.UUID) (*models.SnapshotComment, error)
	DeleteSnapshotComment(ctx context.Context, id uuid.UUID) error
	GetSnapshotCommentCounts(ctx context.Context, snapshotIDs []string, orgID uuid.UUID) (map[string]int, error)
	// Snapshot mount methods
	CreateSnapshotMount(ctx context.Context, mount *models.SnapshotMount) error
	UpdateSnapshotMount(ctx context.Context, mount *models.SnapshotMount) error
	GetSnapshotMountByID(ctx context.Context, id uuid.UUID) (*models.SnapshotMount, error)
	GetActiveSnapshotMountBySnapshotID(ctx context.Context, agentID uuid.UUID, snapshotID string) (*models.SnapshotMount, error)
	GetSnapshotMountsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotMount, error)
	GetActiveSnapshotMountsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.SnapshotMount, error)
	DeleteSnapshotMount(ctx context.Context, id uuid.UUID) error
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
		snapshots.GET("/:id/comments", h.ListSnapshotComments)
		snapshots.POST("/:id/comments", h.CreateSnapshotComment)
		snapshots.GET("/:id1/compare/:id2", h.CompareSnapshots)
		snapshots.GET("/:id1/files/diff/:id2", h.DiffFile)
		// Mount endpoints
		snapshots.POST("/:id/mount", h.MountSnapshot)
		snapshots.DELETE("/:id/mount", h.UnmountSnapshot)
		snapshots.GET("/:id/mount", h.GetSnapshotMount)
	}

	// Comments resource for direct access
	comments := r.Group("/comments")
	{
		comments.DELETE("/:id", h.DeleteSnapshotComment)
	}

	// Mounts resource for listing all mounts
	mounts := r.Group("/mounts")
	{
		mounts.GET("", h.ListMounts)
	}

	restores := r.Group("/restores")
	{
		restores.GET("", h.ListRestores)
		restores.POST("", h.CreateRestore)
		restores.POST("/preview", h.PreviewRestore)
		restores.POST("/cloud", h.CreateCloudRestore)
		restores.GET("/:id", h.GetRestore)
		restores.GET("/:id/progress", h.GetCloudRestoreProgress)
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

// PathMappingRequest represents a path mapping in API requests.
type PathMappingRequest struct {
	SourcePath string `json:"source_path" binding:"required"`
	TargetPath string `json:"target_path" binding:"required"`
}

// CreateRestoreRequest is the request body for creating a restore job.
type CreateRestoreRequest struct {
	SnapshotID    string               `json:"snapshot_id" binding:"required"`
	AgentID       string               `json:"agent_id" binding:"required"`               // Target agent (where restore executes)
	SourceAgentID string               `json:"source_agent_id,omitempty"`                 // Source agent for cross-agent restores
	RepositoryID  string               `json:"repository_id" binding:"required"`
	TargetPath    string               `json:"target_path" binding:"required"`
	IncludePaths  []string             `json:"include_paths,omitempty"`
	ExcludePaths  []string             `json:"exclude_paths,omitempty"`
	PathMappings  []PathMappingRequest `json:"path_mappings,omitempty"`                   // Path remapping for cross-agent restores
}

// CloudRestoreTargetRequest represents the cloud storage target for a restore operation.
type CloudRestoreTargetRequest struct {
	Type string `json:"type" binding:"required"` // "s3", "b2", or "restic"
	// S3/B2 configuration
	Bucket          string `json:"bucket,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	UseSSL          bool   `json:"use_ssl,omitempty"`
	// B2 specific
	AccountID      string `json:"account_id,omitempty"`
	ApplicationKey string `json:"application_key,omitempty"`
	// Restic repository configuration
	Repository         string `json:"repository,omitempty"`
	RepositoryPassword string `json:"repository_password,omitempty"`
}

// CreateCloudRestoreRequest is the request body for creating a cloud restore job.
type CreateCloudRestoreRequest struct {
	SnapshotID   string                    `json:"snapshot_id" binding:"required"`
	AgentID      string                    `json:"agent_id" binding:"required"`
	RepositoryID string                    `json:"repository_id" binding:"required"`
	IncludePaths []string                  `json:"include_paths,omitempty"`
	ExcludePaths []string                  `json:"exclude_paths,omitempty"`
	CloudTarget  CloudRestoreTargetRequest `json:"cloud_target" binding:"required"`
	VerifyUpload bool                      `json:"verify_upload,omitempty"`
}

// CloudRestoreProgressResponse represents the progress of a cloud restore upload operation.
type CloudRestoreProgressResponse struct {
	TotalFiles       int64   `json:"total_files"`
	TotalBytes       int64   `json:"total_bytes"`
	UploadedFiles    int64   `json:"uploaded_files"`
	UploadedBytes    int64   `json:"uploaded_bytes"`
	CurrentFile      string  `json:"current_file,omitempty"`
	PercentComplete  float64 `json:"percent_complete"`
	VerifiedChecksum bool    `json:"verified_checksum"`
}

// CloudRestoreTargetRequest represents the cloud storage target for a restore operation.
type CloudRestoreTargetRequest struct {
	Type string `json:"type" binding:"required"` // "s3", "b2", or "restic"
	// S3/B2 configuration
	Bucket          string `json:"bucket,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	UseSSL          bool   `json:"use_ssl,omitempty"`
	// B2 specific
	AccountID      string `json:"account_id,omitempty"`
	ApplicationKey string `json:"application_key,omitempty"`
	// Restic repository configuration
	Repository         string `json:"repository,omitempty"`
	RepositoryPassword string `json:"repository_password,omitempty"`
}

// CreateCloudRestoreRequest is the request body for creating a cloud restore job.
type CreateCloudRestoreRequest struct {
	SnapshotID   string                    `json:"snapshot_id" binding:"required"`
	AgentID      string                    `json:"agent_id" binding:"required"`
	RepositoryID string                    `json:"repository_id" binding:"required"`
	IncludePaths []string                  `json:"include_paths,omitempty"`
	ExcludePaths []string                  `json:"exclude_paths,omitempty"`
	CloudTarget  CloudRestoreTargetRequest `json:"cloud_target" binding:"required"`
	VerifyUpload bool                      `json:"verify_upload,omitempty"`
}

// CloudRestoreProgressResponse represents the progress of a cloud restore upload operation.
type CloudRestoreProgressResponse struct {
	TotalFiles       int64   `json:"total_files"`
	TotalBytes       int64   `json:"total_bytes"`
	UploadedFiles    int64   `json:"uploaded_files"`
	UploadedBytes    int64   `json:"uploaded_bytes"`
	CurrentFile      string  `json:"current_file,omitempty"`
	PercentComplete  float64 `json:"percent_complete"`
	VerifiedChecksum bool    `json:"verified_checksum"`
}

// RestorePreviewRequest is the request body for previewing a restore operation.
type RestorePreviewRequest struct {
	SnapshotID    string               `json:"snapshot_id" binding:"required"`
	AgentID       string               `json:"agent_id" binding:"required"`               // Target agent
	SourceAgentID string               `json:"source_agent_id,omitempty"`                 // Source agent for cross-agent restores
	RepositoryID  string               `json:"repository_id" binding:"required"`
	TargetPath    string               `json:"target_path" binding:"required"`
	IncludePaths  []string             `json:"include_paths,omitempty"`
	ExcludePaths  []string             `json:"exclude_paths,omitempty"`
	PathMappings  []PathMappingRequest `json:"path_mappings,omitempty"`
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
	SelectedPaths   []string                     `json:"selected_paths,omitempty"`
	SelectedSize    int64                        `json:"selected_size,omitempty"`
}

// PathMappingResponse represents a path mapping in API responses.
type PathMappingResponse struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
}

// RestoreProgressResponse represents restore progress in API responses.
type RestoreProgressResponse struct {
	FilesRestored int64  `json:"files_restored"`
	BytesRestored int64  `json:"bytes_restored"`
	TotalFiles    *int64 `json:"total_files,omitempty"`
	TotalBytes    *int64 `json:"total_bytes,omitempty"`
	CurrentFile   string `json:"current_file,omitempty"`
}

// RestoreResponse represents a restore job in API responses.
type RestoreResponse struct {
	ID             string                   `json:"id"`
	AgentID        string                   `json:"agent_id"`                      // Target agent
	SourceAgentID  string                   `json:"source_agent_id,omitempty"`     // Source agent for cross-agent restores
	RepositoryID   string                   `json:"repository_id"`
	SnapshotID     string                   `json:"snapshot_id"`
	TargetPath     string                   `json:"target_path"`
	IncludePaths   []string                 `json:"include_paths,omitempty"`
	ExcludePaths   []string                 `json:"exclude_paths,omitempty"`
	PathMappings   []PathMappingResponse    `json:"path_mappings,omitempty"`
	Status         string                   `json:"status"`
	Progress       *RestoreProgressResponse `json:"progress,omitempty"`
	IsCrossAgent   bool                     `json:"is_cross_agent"`
	StartedAt      string                   `json:"started_at,omitempty"`
	CompletedAt    string                   `json:"completed_at,omitempty"`
	ErrorMessage   string                   `json:"error_message,omitempty"`
	CreatedAt      string                   `json:"created_at"`
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
	// Cloud restore fields
	IsCloudRestore      bool                          `json:"is_cloud_restore,omitempty"`
	CloudTarget         *CloudRestoreTargetRequest    `json:"cloud_target,omitempty"`
	CloudProgress       *CloudRestoreProgressResponse `json:"cloud_progress,omitempty"`
	CloudTargetLocation string                        `json:"cloud_target_location,omitempty"`
	VerifyUpload        bool                          `json:"verify_upload,omitempty"`
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
		IsCrossAgent: r.IsCrossAgentRestore(),
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
	}
	if r.SourceAgentID != nil {
		resp.SourceAgentID = r.SourceAgentID.String()
	}
	if r.StartedAt != nil {
		resp.StartedAt = r.StartedAt.Format(time.RFC3339)
	}
	if r.CompletedAt != nil {
		resp.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}
	if r.Progress != nil {
		resp.Progress = &RestoreProgressResponse{
			FilesRestored: r.Progress.FilesRestored,
			BytesRestored: r.Progress.BytesRestored,
			TotalFiles:    r.Progress.TotalFiles,
			TotalBytes:    r.Progress.TotalBytes,
			CurrentFile:   r.Progress.CurrentFile,
		}
	}
	for _, pm := range r.PathMappings {
		resp.PathMappings = append(resp.PathMappings, PathMappingResponse{
			SourcePath: pm.SourcePath,
			TargetPath: pm.TargetPath,
		})
	}

	// Add cloud restore fields
	if r.IsCloudRestore() {
		resp.IsCloudRestore = true
		resp.CloudTargetLocation = r.CloudTargetLocation
		resp.VerifyUpload = r.VerifyUpload

		if r.CloudTarget != nil {
			resp.CloudTarget = &CloudRestoreTargetRequest{
				Type:       string(r.CloudTarget.Type),
				Bucket:     r.CloudTarget.Bucket,
				Prefix:     r.CloudTarget.Prefix,
				Region:     r.CloudTarget.Region,
				Endpoint:   r.CloudTarget.Endpoint,
				UseSSL:     r.CloudTarget.UseSSL,
				Repository: r.CloudTarget.Repository,
				// Note: Credentials are not included in responses for security
			}
		}

		if r.CloudProgress != nil {
			resp.CloudProgress = &CloudRestoreProgressResponse{
				TotalFiles:       r.CloudProgress.TotalFiles,
				TotalBytes:       r.CloudProgress.TotalBytes,
				UploadedFiles:    r.CloudProgress.UploadedFiles,
				UploadedBytes:    r.CloudProgress.UploadedBytes,
				CurrentFile:      r.CloudProgress.CurrentFile,
				PercentComplete:  r.CloudProgress.PercentComplete(),
				VerifiedChecksum: r.CloudProgress.VerifiedChecksum,
			}
		}
	}

	return resp
}

// CreateRestore creates a new restore job.
//
//	@Summary		Create restore
//	@Description	Creates a new restore job to restore files from a snapshot. Supports cross-agent restores by specifying source_agent_id.
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

	// Parse target agent ID (where restore will execute)
	targetAgentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	repositoryID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	// Parse source agent ID if provided (for cross-agent restores)
	var sourceAgentID uuid.UUID
	isCrossAgent := req.SourceAgentID != ""
	if isCrossAgent {
		sourceAgentID, err = uuid.Parse(req.SourceAgentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_agent_id"})
			return
		}
	}

	// Verify user access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify access to target agent
	targetAgent, err := h.store.GetAgentByID(c.Request.Context(), targetAgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target agent not found"})
		return
	}
	if targetAgent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "target agent not found"})
		return
	}

	// For cross-agent restores, verify access to source agent
	if isCrossAgent {
		sourceAgent, err := h.store.GetAgentByID(c.Request.Context(), sourceAgentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "source agent not found"})
			return
		}
		if sourceAgent.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "source agent not found"})
			return
		}
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

	// Convert path mappings
	var pathMappings []models.PathMapping
	for _, pm := range req.PathMappings {
		pathMappings = append(pathMappings, models.PathMapping{
			SourcePath: pm.SourcePath,
			TargetPath: pm.TargetPath,
		})
	}

	// Create restore job
	var restore *models.Restore
	if isCrossAgent {
		restore = models.NewCrossRestore(sourceAgentID, targetAgentID, repositoryID, req.SnapshotID, req.TargetPath, req.IncludePaths, req.ExcludePaths, pathMappings)
	} else {
		restore = models.NewRestore(targetAgentID, repositoryID, req.SnapshotID, req.TargetPath, req.IncludePaths, req.ExcludePaths)
	}

	if err := h.store.CreateRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Msg("failed to create restore job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create restore job"})
		return
	}

	logEvent := h.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("target_agent_id", req.AgentID).
		Str("target_path", req.TargetPath).
		Bool("is_cross_agent", isCrossAgent)
	if isCrossAgent {
		logEvent = logEvent.Str("source_agent_id", req.SourceAgentID)
	}
	logEvent.Msg("restore job created")

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
		Strs("include_paths", req.IncludePaths).
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
		SelectedPaths:   req.IncludePaths,
		SelectedSize:    0,
	})
}

// CreateCloudRestore creates a new restore job that uploads to cloud storage.
//
//	@Summary		Create cloud restore
//	@Description	Creates a new restore job that restores files from a snapshot and uploads to cloud storage (S3, B2, or another Restic repository)
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateCloudRestoreRequest	true	"Cloud restore details"
//	@Success		201		{object}	RestoreResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/cloud [post]
func (h *SnapshotsHandler) CreateCloudRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateCloudRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate cloud target type
	targetType := models.CloudRestoreTargetType(req.CloudTarget.Type)
	switch targetType {
	case models.CloudTargetS3:
		if req.CloudTarget.Bucket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required for S3 target"})
			return
		}
		if req.CloudTarget.AccessKeyID == "" || req.CloudTarget.SecretAccessKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "access_key_id and secret_access_key are required for S3 target"})
			return
		}
	case models.CloudTargetB2:
		if req.CloudTarget.Bucket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required for B2 target"})
			return
		}
		if req.CloudTarget.AccountID == "" || req.CloudTarget.ApplicationKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account_id and application_key are required for B2 target"})
			return
		}
	case models.CloudTargetRestic:
		if req.CloudTarget.Repository == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository is required for Restic target"})
			return
		}
		if req.CloudTarget.RepositoryPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository_password is required for Restic target"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cloud target type: must be 's3', 'b2', or 'restic'"})
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

	// Create cloud restore target
	cloudTarget := &models.CloudRestoreTarget{
		Type:               targetType,
		Bucket:             req.CloudTarget.Bucket,
		Prefix:             req.CloudTarget.Prefix,
		Region:             req.CloudTarget.Region,
		Endpoint:           req.CloudTarget.Endpoint,
		AccessKeyID:        req.CloudTarget.AccessKeyID,
		SecretAccessKey:    req.CloudTarget.SecretAccessKey,
		UseSSL:             req.CloudTarget.UseSSL,
		AccountID:          req.CloudTarget.AccountID,
		ApplicationKey:     req.CloudTarget.ApplicationKey,
		Repository:         req.CloudTarget.Repository,
		RepositoryPassword: req.CloudTarget.RepositoryPassword,
	}

	// Create cloud restore job
	restore := models.NewCloudRestore(agentID, repositoryID, req.SnapshotID, req.IncludePaths, req.ExcludePaths, cloudTarget, req.VerifyUpload)

	if err := h.store.CreateRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Msg("failed to create cloud restore job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cloud restore job"})
		return
	}

	h.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("cloud_target_type", string(targetType)).
		Bool("verify_upload", req.VerifyUpload).
		Msg("cloud restore job created")

	c.JSON(http.StatusCreated, toRestoreResponse(restore))
}

// GetCloudRestoreProgress returns the progress of a cloud restore upload operation.
//
//	@Summary		Get cloud restore progress
//	@Description	Returns the current progress of a cloud restore upload operation
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	CloudRestoreProgressResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/{id}/progress [get]
func (h *SnapshotsHandler) GetCloudRestoreProgress(c *gin.Context) {
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

	// Check if this is a cloud restore
	if !restore.IsCloudRestore() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this is not a cloud restore operation"})
		return
	}

	// Return progress
	if restore.CloudProgress == nil {
		c.JSON(http.StatusOK, CloudRestoreProgressResponse{
			TotalFiles:       0,
			TotalBytes:       0,
			UploadedFiles:    0,
			UploadedBytes:    0,
			PercentComplete:  0,
			VerifiedChecksum: false,
		})
		return
	}

	c.JSON(http.StatusOK, CloudRestoreProgressResponse{
		TotalFiles:       restore.CloudProgress.TotalFiles,
		TotalBytes:       restore.CloudProgress.TotalBytes,
		UploadedFiles:    restore.CloudProgress.UploadedFiles,
		UploadedBytes:    restore.CloudProgress.UploadedBytes,
		CurrentFile:      restore.CloudProgress.CurrentFile,
		PercentComplete:  restore.CloudProgress.PercentComplete(),
		VerifiedChecksum: restore.CloudProgress.VerifiedChecksum,
	})
}

// CreateCloudRestore creates a new restore job that uploads to cloud storage.
//
//	@Summary		Create cloud restore
//	@Description	Creates a new restore job that restores files from a snapshot and uploads to cloud storage (S3, B2, or another Restic repository)
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateCloudRestoreRequest	true	"Cloud restore details"
//	@Success		201		{object}	RestoreResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/cloud [post]
func (h *SnapshotsHandler) CreateCloudRestore(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateCloudRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate cloud target type
	targetType := models.CloudRestoreTargetType(req.CloudTarget.Type)
	switch targetType {
	case models.CloudTargetS3:
		if req.CloudTarget.Bucket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required for S3 target"})
			return
		}
		if req.CloudTarget.AccessKeyID == "" || req.CloudTarget.SecretAccessKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "access_key_id and secret_access_key are required for S3 target"})
			return
		}
	case models.CloudTargetB2:
		if req.CloudTarget.Bucket == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required for B2 target"})
			return
		}
		if req.CloudTarget.AccountID == "" || req.CloudTarget.ApplicationKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account_id and application_key are required for B2 target"})
			return
		}
	case models.CloudTargetRestic:
		if req.CloudTarget.Repository == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository is required for Restic target"})
			return
		}
		if req.CloudTarget.RepositoryPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository_password is required for Restic target"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cloud target type: must be 's3', 'b2', or 'restic'"})
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

	// Create cloud restore target
	cloudTarget := &models.CloudRestoreTarget{
		Type:               targetType,
		Bucket:             req.CloudTarget.Bucket,
		Prefix:             req.CloudTarget.Prefix,
		Region:             req.CloudTarget.Region,
		Endpoint:           req.CloudTarget.Endpoint,
		AccessKeyID:        req.CloudTarget.AccessKeyID,
		SecretAccessKey:    req.CloudTarget.SecretAccessKey,
		UseSSL:             req.CloudTarget.UseSSL,
		AccountID:          req.CloudTarget.AccountID,
		ApplicationKey:     req.CloudTarget.ApplicationKey,
		Repository:         req.CloudTarget.Repository,
		RepositoryPassword: req.CloudTarget.RepositoryPassword,
	}

	// Create cloud restore job
	restore := models.NewCloudRestore(agentID, repositoryID, req.SnapshotID, req.IncludePaths, req.ExcludePaths, cloudTarget, req.VerifyUpload)

	if err := h.store.CreateRestore(c.Request.Context(), restore); err != nil {
		h.logger.Error().Err(err).Msg("failed to create cloud restore job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cloud restore job"})
		return
	}

	h.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("agent_id", req.AgentID).
		Str("cloud_target_type", string(targetType)).
		Bool("verify_upload", req.VerifyUpload).
		Msg("cloud restore job created")

	c.JSON(http.StatusCreated, toRestoreResponse(restore))
}

// GetCloudRestoreProgress returns the progress of a cloud restore upload operation.
//
//	@Summary		Get cloud restore progress
//	@Description	Returns the current progress of a cloud restore upload operation
//	@Tags			Restores
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Restore ID"
//	@Success		200	{object}	CloudRestoreProgressResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/restores/{id}/progress [get]
func (h *SnapshotsHandler) GetCloudRestoreProgress(c *gin.Context) {
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

	// Check if this is a cloud restore
	if !restore.IsCloudRestore() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this is not a cloud restore operation"})
		return
	}

	// Return progress
	if restore.CloudProgress == nil {
		c.JSON(http.StatusOK, CloudRestoreProgressResponse{
			TotalFiles:       0,
			TotalBytes:       0,
			UploadedFiles:    0,
			UploadedBytes:    0,
			PercentComplete:  0,
			VerifiedChecksum: false,
		})
		return
	}

	c.JSON(http.StatusOK, CloudRestoreProgressResponse{
		TotalFiles:       restore.CloudProgress.TotalFiles,
		TotalBytes:       restore.CloudProgress.TotalBytes,
		UploadedFiles:    restore.CloudProgress.UploadedFiles,
		UploadedBytes:    restore.CloudProgress.UploadedBytes,
		CurrentFile:      restore.CloudProgress.CurrentFile,
		PercentComplete:  restore.CloudProgress.PercentComplete(),
		VerifiedChecksum: restore.CloudProgress.VerifiedChecksum,
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

// FileDiffResponse represents the response from diffing a file between two snapshots.
type FileDiffResponse struct {
	Path        string `json:"path"`
	IsBinary    bool   `json:"is_binary"`
	ChangeType  string `json:"change_type"` // "modified", "added", "removed"
	OldSize     int64  `json:"old_size,omitempty"`
	NewSize     int64  `json:"new_size,omitempty"`
	OldHash     string `json:"old_hash,omitempty"`
	NewHash     string `json:"new_hash,omitempty"`
	UnifiedDiff string `json:"unified_diff,omitempty"` // For text files
	OldContent  string `json:"old_content,omitempty"`  // For side-by-side view
	NewContent  string `json:"new_content,omitempty"`  // For side-by-side view
	SnapshotID1 string `json:"snapshot_id_1"`
	SnapshotID2 string `json:"snapshot_id_2"`
}

// DiffFile returns the diff of a specific file between two snapshots.
// GET /api/v1/snapshots/:id1/files/diff/:id2?path=<file_path>
//
//	@Summary		Get file diff between snapshots
//	@Description	Returns the diff of a specific file between two snapshots
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id1		path		string	true	"First snapshot ID"
//	@Param			id2		path		string	true	"Second snapshot ID"
//	@Param			path	query		string	true	"File path to diff"
//	@Success		200		{object}	FileDiffResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id1}/files/diff/{id2} [get]
func (h *SnapshotsHandler) DiffFile(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID1 := c.Param("id1")
	snapshotID2 := c.Param("id2")
	filePath := c.Query("path")

	if snapshotID1 == "" || snapshotID2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both snapshot IDs are required"})
		return
	}

	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file path is required"})
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

	h.logger.Info().
		Str("snapshot_id_1", snapshotID1).
		Str("snapshot_id_2", snapshotID2).
		Str("file_path", filePath).
		Msg("file diff requested")

	// Note: In a full implementation, this would call the agent to extract
	// and diff the file from the actual Restic repository. For now, we return
	// a placeholder response indicating the functionality is available but
	// requires agent communication.
	c.JSON(http.StatusOK, FileDiffResponse{
		Path:        filePath,
		IsBinary:    false,
		ChangeType:  "modified",
		SnapshotID1: snapshotID1,
		SnapshotID2: snapshotID2,
		OldSize:     0,
		NewSize:     0,
		UnifiedDiff: "",
		OldContent:  "",
		NewContent:  "",
	})
}

// CompareSnapshots compares two snapshots and returns their differences.
// GET /api/v1/snapshots/:id1/compare/:id2
func (h *SnapshotsHandler) CompareSnapshots(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID1 := c.Param("id1")
	snapshotID2 := c.Param("id2")

	if snapshotID1 == "" || snapshotID2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both snapshot IDs are required"})
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

	h.logger.Info().
		Str("snapshot_id_1", snapshotID1).
		Str("snapshot_id_2", snapshotID2).
		Msg("snapshot comparison requested")

	// Note: In a full implementation, this would call the agent to run restic diff
	// on the actual repository. For now, we return a placeholder response
	// indicating the functionality is available but requires agent communication.
	c.JSON(http.StatusOK, SnapshotCompareResponse{
		SnapshotID1: snapshotID1,
		SnapshotID2: snapshotID2,
		Snapshot1:   snapshot1Info,
		Snapshot2:   snapshot2Info,
		Stats: SnapshotDiffStats{
			FilesAdded:       0,
			FilesRemoved:     0,
			FilesModified:    0,
			DirsAdded:        0,
			DirsRemoved:      0,
			TotalSizeAdded:   0,
			TotalSizeRemoved: 0,
		},
		Changes: []SnapshotDiffEntry{},
	})
}

// SnapshotMountResponse represents a snapshot mount in API responses.
type SnapshotMountResponse struct {
	ID           string  `json:"id"`
	AgentID      string  `json:"agent_id"`
	RepositoryID string  `json:"repository_id"`
	SnapshotID   string  `json:"snapshot_id"`
	MountPath    string  `json:"mount_path"`
	Status       string  `json:"status"`
	MountedAt    *string `json:"mounted_at,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
	UnmountedAt  *string `json:"unmounted_at,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

func toSnapshotMountResponse(m *models.SnapshotMount) SnapshotMountResponse {
	resp := SnapshotMountResponse{
		ID:           m.ID.String(),
		AgentID:      m.AgentID.String(),
		RepositoryID: m.RepositoryID.String(),
		SnapshotID:   m.SnapshotID,
		MountPath:    m.MountPath,
		Status:       string(m.Status),
		ErrorMessage: m.ErrorMessage,
		CreatedAt:    m.CreatedAt.Format(time.RFC3339),
	}
	if m.MountedAt != nil {
		t := m.MountedAt.Format(time.RFC3339)
		resp.MountedAt = &t
	}
	if m.ExpiresAt != nil {
		t := m.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &t
	}
	if m.UnmountedAt != nil {
		t := m.UnmountedAt.Format(time.RFC3339)
		resp.UnmountedAt = &t
	}
	return resp
}

// MountSnapshotRequest is the request body for mounting a snapshot.
type MountSnapshotRequest struct {
	AgentID        string `json:"agent_id" binding:"required"`
	RepositoryID   string `json:"repository_id" binding:"required"`
	TimeoutMinutes int    `json:"timeout_minutes,omitempty"`
}

// MountSnapshot mounts a snapshot as a FUSE filesystem.
//
//	@Summary		Mount snapshot
//	@Description	Mounts a snapshot as a read-only FUSE filesystem for browsing
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Snapshot ID"
//	@Param			request	body		MountSnapshotRequest	true	"Mount request"
//	@Success		201		{object}	SnapshotMountResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/mount [post]
func (h *SnapshotsHandler) MountSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	var req MountSnapshotRequest
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

	// Get user and verify org access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify agent access
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
	_, err = h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Check if already mounted
	existingMount, err := h.store.GetActiveSnapshotMountBySnapshotID(c.Request.Context(), agentID, snapshotID)
	if err == nil && existingMount != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "snapshot already mounted",
			"mount": toSnapshotMountResponse(existingMount),
		})
		return
	}

	// Create mount record
	mountPath := "" // Will be set by agent when mount actually happens
	mount := models.NewSnapshotMount(dbUser.OrgID, agentID, repositoryID, snapshotID, mountPath)

	if err := h.store.CreateSnapshotMount(c.Request.Context(), mount); err != nil {
		h.logger.Error().Err(err).Msg("failed to create snapshot mount")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mount"})
		return
	}

	h.logger.Info().
		Str("mount_id", mount.ID.String()).
		Str("snapshot_id", snapshotID).
		Str("agent_id", req.AgentID).
		Msg("snapshot mount created")

	c.JSON(http.StatusCreated, toSnapshotMountResponse(mount))
}

// UnmountSnapshot unmounts a previously mounted snapshot.
//
//	@Summary		Unmount snapshot
//	@Description	Unmounts a snapshot that was previously mounted as a FUSE filesystem
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Snapshot ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/mount [delete]
func (h *SnapshotsHandler) UnmountSnapshot(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id query parameter required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Get user and verify org access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify agent access
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Find the active mount
	mount, err := h.store.GetActiveSnapshotMountBySnapshotID(c.Request.Context(), agentID, snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active mount found for this snapshot"})
		return
	}

	// Mark as unmounting
	mount.StartUnmounting()
	if err := h.store.UpdateSnapshotMount(c.Request.Context(), mount); err != nil {
		h.logger.Error().Err(err).Msg("failed to update mount status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate unmount"})
		return
	}

	h.logger.Info().
		Str("mount_id", mount.ID.String()).
		Str("snapshot_id", snapshotID).
		Msg("snapshot unmount initiated")

	c.JSON(http.StatusOK, gin.H{"message": "unmount initiated", "mount": toSnapshotMountResponse(mount)})
}

// GetSnapshotMount returns the mount status for a snapshot.
//
//	@Summary		Get snapshot mount status
//	@Description	Returns the current mount status for a snapshot
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string	true	"Snapshot ID"
//	@Param			agent_id	query		string	true	"Agent ID"
//	@Success		200			{object}	SnapshotMountResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/mount [get]
func (h *SnapshotsHandler) GetSnapshotMount(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	agentIDParam := c.Query("agent_id")
	if agentIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id query parameter required"})
		return
	}

	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Get user and verify org access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Verify agent access
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Find the active mount
	mount, err := h.store.GetActiveSnapshotMountBySnapshotID(c.Request.Context(), agentID, snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active mount found for this snapshot"})
		return
	}

	c.JSON(http.StatusOK, toSnapshotMountResponse(mount))
}

// ListMounts returns all snapshot mounts for the organization.
//
//	@Summary		List mounts
//	@Description	Returns all snapshot mounts for the current organization
//	@Tags			Snapshots
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Success		200			{object}	map[string][]SnapshotMountResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/mounts [get]
func (h *SnapshotsHandler) ListMounts(c *gin.Context) {
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

	var mounts []*models.SnapshotMount

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

		mounts, err = h.store.GetActiveSnapshotMountsByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to list mounts by agent")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list mounts"})
			return
		}
	} else {
		mounts, err = h.store.GetSnapshotMountsByOrgID(c.Request.Context(), dbUser.OrgID)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to list mounts")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list mounts"})
			return
		}
	}

	var responses []SnapshotMountResponse
	for _, mount := range mounts {
		responses = append(responses, toSnapshotMountResponse(mount))
	}

	c.JSON(http.StatusOK, gin.H{"mounts": responses})
}
