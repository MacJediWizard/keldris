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

// FileHistoryStore defines the interface for file history-related persistence operations.
type FileHistoryStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
}

// FileHistoryHandler handles file history HTTP endpoints.
type FileHistoryHandler struct {
	store  FileHistoryStore
	logger zerolog.Logger
}

// NewFileHistoryHandler creates a new FileHistoryHandler.
func NewFileHistoryHandler(store FileHistoryStore, logger zerolog.Logger) *FileHistoryHandler {
	return &FileHistoryHandler{
		store:  store,
		logger: logger.With().Str("component", "file_history_handler").Logger(),
	}
}

// RegisterRoutes registers file history routes on the given router group.
func (h *FileHistoryHandler) RegisterRoutes(r *gin.RouterGroup) {
	files := r.Group("/files")
	{
		files.GET("/history", h.GetFileHistory)
	}
}

// FileVersionResponse represents a file version in API responses.
type FileVersionResponse struct {
	SnapshotID   string `json:"snapshot_id"`
	ShortID      string `json:"short_id"`
	SnapshotTime string `json:"snapshot_time"`
	Size         int64  `json:"size"`
	ModTime      string `json:"mod_time"`
}

// FileHistoryResponse represents the file history API response.
type FileHistoryResponse struct {
	FilePath     string                `json:"file_path"`
	AgentID      string                `json:"agent_id"`
	RepositoryID string                `json:"repository_id"`
	AgentName    string                `json:"agent_name"`
	RepoName     string                `json:"repo_name"`
	Versions     []FileVersionResponse `json:"versions"`
	Message      string                `json:"message,omitempty"`
}

// GetFileHistory returns the history of a specific file across all snapshots.
// GET /api/v1/files/history?path=/path/to/file&agent_id=xxx&repository_id=xxx
func (h *FileHistoryHandler) GetFileHistory(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	agentIDStr := c.Query("agent_id")
	if agentIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}

	repositoryIDStr := c.Query("repository_id")
	if repositoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository_id parameter is required"})
		return
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	repositoryID, err := uuid.Parse(repositoryIDStr)
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

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify repository belongs to user's org
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repositoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}
	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get all backups for this agent
	backups, err := h.store.GetBackupsByAgentID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve file history"})
		return
	}

	// Filter to only completed backups for the specified repository
	var versions []FileVersionResponse
	for _, backup := range backups {
		if backup.Status != models.BackupStatusCompleted || backup.SnapshotID == "" {
			continue
		}

		schedule, err := h.store.GetScheduleByID(c.Request.Context(), backup.ScheduleID)
		if err != nil {
			continue
		}

		// Check if schedule uses the specified repository
		hasRepo := false
		for _, sr := range schedule.Repositories {
			if sr.RepositoryID == repositoryID {
				hasRepo = true
				break
			}
		}
		if !hasRepo {
			continue
		}

		// Check if the file path might be in this backup's paths
		pathMatches := false
		for _, p := range schedule.Paths {
			if len(filePath) >= len(p) && filePath[:len(p)] == p {
				pathMatches = true
				break
			}
		}
		if !pathMatches {
			continue
		}

		shortID := backup.SnapshotID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		versions = append(versions, FileVersionResponse{
			SnapshotID:   backup.SnapshotID,
			ShortID:      shortID,
			SnapshotTime: backup.StartedAt.Format(time.RFC3339),
			Size:         0, // Size would come from actual restic query
			ModTime:      backup.StartedAt.Format(time.RFC3339),
		})
	}

	h.logger.Info().
		Str("file_path", filePath).
		Str("agent_id", agentIDStr).
		Str("repository_id", repositoryIDStr).
		Int("version_count", len(versions)).
		Msg("file history requested")

	response := FileHistoryResponse{
		FilePath:     filePath,
		AgentID:      agentIDStr,
		RepositoryID: repositoryIDStr,
		AgentName:    agent.Hostname,
		RepoName:     repo.Name,
		Versions:     versions,
	}

	if len(versions) == 0 {
		response.Message = "No versions found. File history requires agent communication to query restic repositories directly. Available backups have been listed based on schedule paths."
	}

	c.JSON(http.StatusOK, response)
}
