package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
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

// FileSearchStore defines the interface for file search persistence operations.
type FileSearchStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoryKeyByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryKey, error)
}

// FileSearchHandler handles file search HTTP endpoints.
type FileSearchHandler struct {
	store      FileSearchStore
	keyManager *crypto.KeyManager
	restic     *backup.Restic
	logger     zerolog.Logger
}

// NewFileSearchHandler creates a new FileSearchHandler.
func NewFileSearchHandler(store FileSearchStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *FileSearchHandler {
	return &FileSearchHandler{
		store:      store,
		keyManager: keyManager,
		restic:     backup.NewRestic(logger),
		logger:     logger.With().Str("component", "file_search_handler").Logger(),
	}
}

// RegisterRoutes registers file search routes on the given router group.
func (h *FileSearchHandler) RegisterRoutes(r *gin.RouterGroup) {
	search := r.Group("/search")
	{
		search.GET("/files", h.SearchFiles)
	}
}

// FileSearchResultResponse represents a single file search result.
type FileSearchResultResponse struct {
	SnapshotID   string `json:"snapshot_id"`
	SnapshotTime string `json:"snapshot_time"`
	Hostname     string `json:"hostname"`
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	FileSize     int64  `json:"file_size"`
	FileType     string `json:"file_type"`
	ModTime      string `json:"mod_time"`
}

// SnapshotFileGroupResponse groups files by snapshot.
type SnapshotFileGroupResponse struct {
	SnapshotID   string                     `json:"snapshot_id"`
	SnapshotTime string                     `json:"snapshot_time"`
	Hostname     string                     `json:"hostname"`
	FileCount    int                        `json:"file_count"`
	Files        []FileSearchResultResponse `json:"files"`
}

// FileSearchResponse is the API response for file search.
type FileSearchResponse struct {
	Query         string                      `json:"query"`
	AgentID       string                      `json:"agent_id"`
	RepositoryID  string                      `json:"repository_id"`
	TotalCount    int                         `json:"total_count"`
	SnapshotCount int                         `json:"snapshot_count"`
	Snapshots     []SnapshotFileGroupResponse `json:"snapshots"`
	Message       string                      `json:"message,omitempty"`
}

// SearchFiles searches for files matching a pattern across all snapshots.
// GET /api/v1/search/files
// Query params:
//   - q: search query (required) - filename pattern to search for
//   - agent_id: agent ID (required) - agent that performed the backups
//   - repository_id: repository ID (required) - repository to search in
//   - path: path prefix filter (optional) - filter to files under this path
//   - date_from: date range start (optional) - RFC3339 format
//   - date_to: date range end (optional) - RFC3339 format
//   - size_min: minimum file size in bytes (optional)
//   - size_max: maximum file size in bytes (optional)
//   - limit: max results (optional, default 100)
func (h *FileSearchHandler) SearchFiles(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Parse and validate required parameters
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query (q) is required"})
		return
	}

	agentIDStr := c.Query("agent_id")
	if agentIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	repositoryIDStr := c.Query("repository_id")
	if repositoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository_id is required"})
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

	// Get repository key
	repoKey, err := h.store.GetRepositoryKeyByRepositoryID(c.Request.Context(), repositoryID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repositoryID.String()).Msg("failed to get repository key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository credentials"})
		return
	}

	// Decrypt the repository password
	password, err := h.keyManager.Decrypt(repoKey.EncryptedKey)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repositoryID.String()).Msg("failed to decrypt repository password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository"})
		return
	}

	// Decrypt and parse the repository config
	configJSON, err := h.keyManager.Decrypt(repo.ConfigEncrypted)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repositoryID.String()).Msg("failed to decrypt repository config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository"})
		return
	}

	backend, err := backends.ParseBackend(repo.Type, configJSON)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repositoryID.String()).Msg("failed to parse repository backend")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access repository"})
		return
	}

	// Create restic config
	resticCfg := backend.ToResticConfig(string(password))

	// Build search filter
	filter := backup.FileSearchFilter{
		Query: query,
		Limit: 100, // Default limit
	}

	// Parse optional filters
	if pathPrefix := c.Query("path"); pathPrefix != "" {
		filter.PathPrefix = pathPrefix
	}

	if snapshotIDsStr := c.Query("snapshot_ids"); snapshotIDsStr != "" {
		filter.SnapshotIDs = strings.Split(snapshotIDsStr, ",")
	}

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format (use RFC3339)"})
			return
		}
		filter.DateFrom = &dateFrom
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format (use RFC3339)"})
			return
		}
		filter.DateTo = &dateTo
	}

	if sizeMinStr := c.Query("size_min"); sizeMinStr != "" {
		sizeMin, err := strconv.ParseInt(sizeMinStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_min"})
			return
		}
		filter.SizeMin = &sizeMin
	}

	if sizeMaxStr := c.Query("size_max"); sizeMaxStr != "" {
		sizeMax, err := strconv.ParseInt(sizeMaxStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_max"})
			return
		}
		filter.SizeMax = &sizeMax
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit (1-1000)"})
			return
		}
		filter.Limit = limit
	}

	h.logger.Info().
		Str("query", query).
		Str("agent_id", agentIDStr).
		Str("repository_id", repositoryIDStr).
		Str("path_prefix", filter.PathPrefix).
		Int("limit", filter.Limit).
		Msg("searching files across snapshots")

	// Execute the search
	searchResult, err := h.restic.SearchFiles(c.Request.Context(), resticCfg, filter)
	if err != nil {
		h.logger.Error().Err(err).
			Str("query", query).
			Str("repo_id", repositoryID.String()).
			Msg("file search failed")

		// Return a meaningful error without exposing internal details
		if err == backup.ErrRepositoryNotInitialized {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository not initialized"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "file search failed"})
		return
	}

	// Convert to response format
	response := FileSearchResponse{
		Query:         query,
		AgentID:       agentIDStr,
		RepositoryID:  repositoryIDStr,
		TotalCount:    searchResult.TotalCount,
		SnapshotCount: len(searchResult.Snapshots),
		Snapshots:     make([]SnapshotFileGroupResponse, len(searchResult.Snapshots)),
	}

	for i, group := range searchResult.Snapshots {
		files := make([]FileSearchResultResponse, len(group.Files))
		for j, file := range group.Files {
			files[j] = FileSearchResultResponse{
				SnapshotID:   file.SnapshotID,
				SnapshotTime: file.SnapshotTime.Format(time.RFC3339),
				Hostname:     file.Hostname,
				FileName:     file.FileName,
				FilePath:     file.FilePath,
				FileSize:     file.FileSize,
				FileType:     file.FileType,
				ModTime:      file.ModTime.Format(time.RFC3339),
			}
		}

		response.Snapshots[i] = SnapshotFileGroupResponse{
			SnapshotID:   group.SnapshotID,
			SnapshotTime: group.SnapshotTime.Format(time.RFC3339),
			Hostname:     group.Hostname,
			FileCount:    group.FileCount,
			Files:        files,
		}
	}

	if response.TotalCount == 0 {
		response.Message = "No files found matching the search criteria"
	}

	h.logger.Info().
		Str("query", query).
		Int("total_count", response.TotalCount).
		Int("snapshot_count", response.SnapshotCount).
		Msg("file search completed")

	c.JSON(http.StatusOK, response)
}
