package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RepositoryImportStore defines the interface for repository import persistence operations.
type RepositoryImportStore interface {
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	CreateRepository(ctx context.Context, repo *models.Repository) error
	CreateRepositoryKey(ctx context.Context, rk *models.RepositoryKey) error
	CreateImportedSnapshot(ctx context.Context, snap *models.ImportedSnapshot) error
	CreateImportedSnapshots(ctx context.Context, snapshots []*models.ImportedSnapshot) error
	MarkRepositoryAsImported(ctx context.Context, repositoryID uuid.UUID, snapshotCount int) error
	GetImportedSnapshotsByRepositoryID(ctx context.Context, repositoryID uuid.UUID) ([]*models.ImportedSnapshot, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
}

// RepositoryImportHandler handles repository import endpoints.
type RepositoryImportHandler struct {
	store      RepositoryImportStore
	keyManager *crypto.KeyManager
	importer   *backup.Importer
	logger     zerolog.Logger
}

// NewRepositoryImportHandler creates a new RepositoryImportHandler.
func NewRepositoryImportHandler(store RepositoryImportStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *RepositoryImportHandler {
	return &RepositoryImportHandler{
		store:      store,
		keyManager: keyManager,
		importer:   backup.NewImporter(logger),
		logger:     logger.With().Str("component", "repository_import_handler").Logger(),
	}
}

// RegisterRoutes registers repository import routes on the given router group.
func (h *RepositoryImportHandler) RegisterRoutes(r *gin.RouterGroup) {
	importRoutes := r.Group("/repositories/import")
	{
		importRoutes.POST("/verify", h.VerifyAccess)
		importRoutes.POST("/preview", h.Preview)
		importRoutes.POST("", h.Import)
	}
}

// VerifyAccessRequest is the request body for verifying repository access.
type VerifyAccessRequest struct {
	Type     models.RepositoryType `json:"type" binding:"required"`
	Config   map[string]any        `json:"config" binding:"required"`
	Password string                `json:"password" binding:"required"`
}

// VerifyAccessResponse is the response for verifying repository access.
type VerifyAccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VerifyAccess verifies that the repository can be accessed with the given credentials.
// POST /api/v1/repositories/import/verify
func (h *RepositoryImportHandler) VerifyAccess(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req VerifyAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse and validate backend config
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config format"})
		return
	}

	backend, err := backends.ParseBackend(req.Type, configJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backend config: " + err.Error()})
		return
	}

	if err := backend.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify repository access
	h.logger.Info().Str("type", string(req.Type)).Msg("verifying repository access for import")

	if err := h.importer.VerifyAccess(c.Request.Context(), backend, req.Password); err != nil {
		h.logger.Warn().Err(err).Str("type", string(req.Type)).Msg("repository access verification failed")
		c.JSON(http.StatusOK, VerifyAccessResponse{
			Success: false,
			Message: "Access verification failed: " + err.Error(),
		})
		return
	}

	h.logger.Info().Str("type", string(req.Type)).Msg("repository access verified successfully")
	c.JSON(http.StatusOK, VerifyAccessResponse{
		Success: true,
		Message: "Repository access verified successfully",
	})
}

// PreviewRequest is the request body for previewing a repository.
type PreviewRequest struct {
	Type     models.RepositoryType `json:"type" binding:"required"`
	Config   map[string]any        `json:"config" binding:"required"`
	Password string                `json:"password" binding:"required"`
}

// SnapshotPreview represents a snapshot for preview.
type SnapshotPreview struct {
	ID       string   `json:"id"`
	ShortID  string   `json:"short_id"`
	Time     string   `json:"time"`
	Hostname string   `json:"hostname"`
	Username string   `json:"username"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags,omitempty"`
}

// PreviewResponse is the response for previewing a repository.
type PreviewResponse struct {
	SnapshotCount  int               `json:"snapshot_count"`
	Snapshots      []SnapshotPreview `json:"snapshots"`
	Hostnames      []string          `json:"hostnames"`
	TotalSize      int64             `json:"total_size"`
	TotalFileCount int               `json:"total_file_count"`
}

// Preview retrieves information about an existing repository without modifying it.
// POST /api/v1/repositories/import/preview
func (h *RepositoryImportHandler) Preview(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse and validate backend config
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config format"})
		return
	}

	backend, err := backends.ParseBackend(req.Type, configJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backend config: " + err.Error()})
		return
	}

	if err := backend.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get repository preview
	h.logger.Info().Str("type", string(req.Type)).Msg("previewing repository for import")

	preview, err := h.importer.Preview(c.Request.Context(), backend, req.Password)
	if err != nil {
		h.logger.Error().Err(err).Str("type", string(req.Type)).Msg("failed to preview repository")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to preview repository: " + err.Error()})
		return
	}

	// Convert snapshots to preview format
	snapshots := make([]SnapshotPreview, len(preview.Snapshots))
	for i, snap := range preview.Snapshots {
		snapshots[i] = SnapshotPreview{
			ID:       snap.ID,
			ShortID:  snap.ShortID,
			Time:     snap.Time.Format("2006-01-02T15:04:05Z07:00"),
			Hostname: snap.Hostname,
			Username: snap.Username,
			Paths:    snap.Paths,
			Tags:     snap.Tags,
		}
	}

	h.logger.Info().
		Int("snapshot_count", preview.SnapshotCount).
		Int("hostname_count", len(preview.Hostnames)).
		Msg("repository preview completed")

	c.JSON(http.StatusOK, PreviewResponse{
		SnapshotCount:  preview.SnapshotCount,
		Snapshots:      snapshots,
		Hostnames:      preview.Hostnames,
		TotalSize:      preview.TotalSize,
		TotalFileCount: preview.TotalFileCount,
	})
}

// ImportRequest is the request body for importing a repository.
type ImportRequest struct {
	Name          string                `json:"name" binding:"required,min=1,max=255"`
	Type          models.RepositoryType `json:"type" binding:"required"`
	Config        map[string]any        `json:"config" binding:"required"`
	Password      string                `json:"password" binding:"required"`
	EscrowEnabled bool                  `json:"escrow_enabled"`
	// Optional filters for which snapshots to import
	SnapshotIDs []string `json:"snapshot_ids,omitempty"`
	Hostnames   []string `json:"hostnames,omitempty"`
	// Optional agent ID to associate imported snapshots with
	AgentID string `json:"agent_id,omitempty"`
}

// ImportResponse is the response for importing a repository.
type ImportResponse struct {
	Repository        RepositoryResponse `json:"repository"`
	SnapshotsImported int                `json:"snapshots_imported"`
}

// Import imports an existing Restic repository.
// POST /api/v1/repositories/import
func (h *RepositoryImportHandler) Import(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate repository type
	validTypes := models.ValidRepositoryTypes()
	valid := false
	for _, t := range validTypes {
		if req.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository type", "valid_types": validTypes})
		return
	}

	// Parse agent ID if provided
	var agentID *uuid.UUID
	if req.AgentID != "" {
		parsed, err := uuid.Parse(req.AgentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
			return
		}
		// Verify agent belongs to the organization
		agent, err := h.store.GetAgentByID(c.Request.Context(), parsed)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
			return
		}
		if agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
			return
		}
		agentID = &parsed
	}

	// Parse and validate backend config
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config format"})
		return
	}

	backend, err := backends.ParseBackend(req.Type, configJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backend config: " + err.Error()})
		return
	}

	if err := backend.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify repository access first
	if err := h.importer.VerifyAccess(c.Request.Context(), backend, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to access repository: " + err.Error()})
		return
	}

	// Get snapshots to import
	importOpts := backup.ImportOptions{
		SnapshotIDs: req.SnapshotIDs,
		Hostnames:   req.Hostnames,
	}
	if agentID != nil {
		importOpts.AgentID = agentID.String()
	}

	snapshots, err := h.importer.GetSnapshots(c.Request.Context(), backend, req.Password, importOpts)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get snapshots: " + err.Error()})
		return
	}

	// Encrypt the config
	configEncrypted, err := h.keyManager.Encrypt(configJSON)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt config"})
		return
	}

	// Create the repository
	repo := models.NewRepository(user.CurrentOrgID, req.Name, req.Type, configEncrypted)

	if err := h.store.CreateRepository(c.Request.Context(), repo); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create repository"})
		return
	}

	// Encrypt and store the password (using the provided password, not generating a new one)
	encryptedKey, err := h.keyManager.Encrypt([]byte(req.Password))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt repository password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt repository password"})
		return
	}

	// If escrow is enabled, also store an escrow copy
	var escrowEncryptedKey []byte
	if req.EscrowEnabled {
		escrowEncryptedKey, err = h.keyManager.Encrypt([]byte(req.Password))
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt escrow key")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt escrow key"})
			return
		}
	}

	repoKey := models.NewRepositoryKey(repo.ID, encryptedKey, req.EscrowEnabled, escrowEncryptedKey)
	if err := h.store.CreateRepositoryKey(c.Request.Context(), repoKey); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repo.ID.String()).Msg("failed to create repository key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store repository key"})
		return
	}

	// Import snapshot metadata
	importedSnapshots := make([]*models.ImportedSnapshot, len(snapshots))
	for i, snap := range snapshots {
		importedSnapshots[i] = models.NewImportedSnapshot(
			repo.ID,
			agentID,
			snap.ID,
			snap.ShortID,
			snap.Hostname,
			snap.Username,
			snap.Time,
			snap.Paths,
			snap.Tags,
		)
	}

	if err := h.store.CreateImportedSnapshots(c.Request.Context(), importedSnapshots); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repo.ID.String()).Msg("failed to import snapshots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import snapshots"})
		return
	}

	// Mark repository as imported
	if err := h.store.MarkRepositoryAsImported(c.Request.Context(), repo.ID, len(snapshots)); err != nil {
		h.logger.Warn().Err(err).Str("repo_id", repo.ID.String()).Msg("failed to mark repository as imported")
		// Don't fail the request, just log the warning
	}

	h.logger.Info().
		Str("repo_id", repo.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Int("snapshots_imported", len(snapshots)).
		Bool("escrow_enabled", req.EscrowEnabled).
		Msg("repository imported successfully")

	c.JSON(http.StatusCreated, ImportResponse{
		Repository:        toRepositoryResponse(repo, req.EscrowEnabled),
		SnapshotsImported: len(snapshots),
	})
}
