package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RepositoryStore defines the interface for repository persistence operations.
type RepositoryStore interface {
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	CreateRepository(ctx context.Context, repo *models.Repository) error
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	DeleteRepository(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	// Repository key operations
	CreateRepositoryKey(ctx context.Context, rk *models.RepositoryKey) error
	GetRepositoryKeyByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryKey, error)
	GetRepositoryKeysWithEscrowByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RepositoryKey, error)
}

// RepositoriesHandler handles repository-related HTTP endpoints.
type RepositoriesHandler struct {
	store      RepositoryStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
}

// NewRepositoriesHandler creates a new RepositoriesHandler.
func NewRepositoriesHandler(store RepositoryStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *RepositoriesHandler {
	return &RepositoriesHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "repositories_handler").Logger(),
	}
}

// RegisterRoutes registers repository routes on the given router group.
func (h *RepositoriesHandler) RegisterRoutes(r *gin.RouterGroup) {
	repos := r.Group("/repositories")
	{
		repos.GET("", h.List)
		repos.POST("", h.Create)
		repos.GET("/:id", h.Get)
		repos.PUT("/:id", h.Update)
		repos.DELETE("/:id", h.Delete)
		repos.POST("/:id/test", h.Test)
		repos.POST("/test-connection", h.TestConnection)
		repos.GET("/:id/key/recover", h.RecoverKey)
	}
}

// CreateRepositoryRequest is the request body for creating a repository.
type CreateRepositoryRequest struct {
	Name          string                `json:"name" binding:"required,min=1,max=255" example:"My S3 Backup"`
	Type          models.RepositoryType `json:"type" binding:"required" example:"s3"`
	Config        map[string]any        `json:"config" binding:"required"`
	EscrowEnabled bool                  `json:"escrow_enabled" example:"true"`
}

// UpdateRepositoryRequest is the request body for updating a repository.
type UpdateRepositoryRequest struct {
	Name   string         `json:"name,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

// RepositoryResponse is the API response for a repository (without encrypted config).
type RepositoryResponse struct {
	ID            uuid.UUID             `json:"id"`
	Name          string                `json:"name"`
	Type          models.RepositoryType `json:"type"`
	EscrowEnabled bool                  `json:"escrow_enabled"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
}

// CreateRepositoryResponse is returned when creating a new repository.
// It includes the repository password which is shown only once.
type CreateRepositoryResponse struct {
	Repository RepositoryResponse `json:"repository"`
	Password   string             `json:"password"`
}

// KeyRecoveryResponse is the response for key recovery.
type KeyRecoveryResponse struct {
	RepositoryID   uuid.UUID `json:"repository_id"`
	RepositoryName string    `json:"repository_name"`
	Password       string    `json:"password"`
}

// toResponse converts a Repository model to a RepositoryResponse.
func toRepositoryResponse(r *models.Repository, escrowEnabled bool) RepositoryResponse {
	return RepositoryResponse{
		ID:            r.ID,
		Name:          r.Name,
		Type:          r.Type,
		EscrowEnabled: escrowEnabled,
		CreatedAt:     r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List returns all repositories for the authenticated user's organization.
//
//	@Summary		List repositories
//	@Description	Returns all backup repositories for the current organization
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]RepositoryResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories [get]
func (h *RepositoriesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	repos, err := h.store.GetRepositoriesByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list repositories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list repositories"})
		return
	}

	responses := make([]RepositoryResponse, len(repos))
	for i, r := range repos {
		// Check if escrow is enabled for this repository
		escrowEnabled := false
		repoKey, err := h.store.GetRepositoryKeyByRepositoryID(c.Request.Context(), r.ID)
		if err == nil && repoKey != nil {
			escrowEnabled = repoKey.EscrowEnabled
		}
		responses[i] = toRepositoryResponse(r, escrowEnabled)
	}

	c.JSON(http.StatusOK, gin.H{"repositories": responses})
}

// Get returns a specific repository by ID.
//
//	@Summary		Get repository
//	@Description	Returns a specific repository by ID
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	RepositoryResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id} [get]
func (h *RepositoriesHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to get repository")
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Check if escrow is enabled
	escrowEnabled := false
	repoKey, err := h.store.GetRepositoryKeyByRepositoryID(c.Request.Context(), repo.ID)
	if err == nil && repoKey != nil {
		escrowEnabled = repoKey.EscrowEnabled
	}

	c.JSON(http.StatusOK, toRepositoryResponse(repo, escrowEnabled))
}

// Create creates a new repository.
//
//	@Summary		Create repository
//	@Description	Creates a new backup repository and returns the repository password. Save this password securely as it cannot be retrieved again (unless escrow is enabled).
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateRepositoryRequest	true	"Repository details"
//	@Success		201		{object}	CreateRepositoryResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories [post]
func (h *RepositoriesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateRepositoryRequest
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

	// Encrypt the config using AES-256-GCM
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process config"})
		return
	}

	configEncrypted, err := h.keyManager.Encrypt(configJSON)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt config"})
		return
	}

	repo := models.NewRepository(user.CurrentOrgID, req.Name, req.Type, configEncrypted)

	if err := h.store.CreateRepository(c.Request.Context(), repo); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create repository"})
		return
	}

	// Generate a password for the Restic repository
	password, err := h.keyManager.GeneratePassword()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate repository password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate repository password"})
		return
	}

	// Encrypt the password for storage
	encryptedKey, err := h.keyManager.Encrypt([]byte(password))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt repository password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt repository password"})
		return
	}

	// If escrow is enabled, also store an escrow copy
	var escrowEncryptedKey []byte
	if req.EscrowEnabled {
		escrowEncryptedKey, err = h.keyManager.Encrypt([]byte(password))
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

	h.logger.Info().
		Str("repo_id", repo.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Bool("escrow_enabled", req.EscrowEnabled).
		Msg("repository created with encryption key")

	c.JSON(http.StatusCreated, CreateRepositoryResponse{
		Repository: toRepositoryResponse(repo, req.EscrowEnabled),
		Password:   password,
	})
}

// Update updates an existing repository.
//
//	@Summary		Update repository
//	@Description	Updates an existing repository's name or configuration
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Repository ID"
//	@Param			request	body		UpdateRepositoryRequest	true	"Repository updates"
//	@Success		200		{object}	RepositoryResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id} [put]
func (h *RepositoriesHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		repo.Name = req.Name
	}

	// Update encrypted config if provided
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to marshal config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process config"})
			return
		}

		configEncrypted, err := h.keyManager.Encrypt(configJSON)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt config"})
			return
		}
		repo.ConfigEncrypted = configEncrypted
	}

	if err := h.store.UpdateRepository(c.Request.Context(), repo); err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to update repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update repository"})
		return
	}

	// Check if escrow is enabled
	escrowEnabled := false
	repoKey, err := h.store.GetRepositoryKeyByRepositoryID(c.Request.Context(), repo.ID)
	if err == nil && repoKey != nil {
		escrowEnabled = repoKey.EscrowEnabled
	}

	h.logger.Info().Str("repo_id", id.String()).Msg("repository updated")
	c.JSON(http.StatusOK, toRepositoryResponse(repo, escrowEnabled))
}

// Delete removes a repository.
//
//	@Summary		Delete repository
//	@Description	Removes a repository from the organization
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id} [delete]
func (h *RepositoriesHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if err := h.store.DeleteRepository(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to delete repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete repository"})
		return
	}

	h.logger.Info().Str("repo_id", id.String()).Msg("repository deleted")
	c.JSON(http.StatusOK, gin.H{"message": "repository deleted"})
}

// TestRepositoryResponse is the response for repository connection test.
type TestRepositoryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestConnectionRequest is the request body for testing a backend connection.
type TestConnectionRequest struct {
	Type   models.RepositoryType `json:"type" binding:"required"`
	Config map[string]any        `json:"config" binding:"required"`
}

// Test checks an existing repository's connection.
//
//	@Summary		Test repository
//	@Description	Tests an existing repository's connection to verify it's accessible
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	TestRepositoryResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id}/test [post]
func (h *RepositoriesHandler) Test(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Decrypt the stored config
	configJSON, err := h.keyManager.Decrypt(repo.ConfigEncrypted)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to decrypt repository config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt repository config"})
		return
	}

	// Parse the backend from decrypted config
	backend, err := backends.ParseBackend(repo.Type, configJSON)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to parse backend config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid backend config"})
		return
	}

	// Test the connection
	h.logger.Info().Str("repo_id", id.String()).Msg("testing repository connection")

	if err := backend.TestConnection(); err != nil {
		h.logger.Warn().Err(err).Str("repo_id", id.String()).Msg("repository connection test failed")
		c.JSON(http.StatusOK, TestRepositoryResponse{
			Success: false,
			Message: "Connection failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TestRepositoryResponse{
		Success: true,
		Message: "Connection successful",
	})
}

// TestConnection tests a backend configuration without saving.
//
//	@Summary		Test connection
//	@Description	Tests a backend configuration without creating a repository. Useful for validating credentials before saving.
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			request	body		TestConnectionRequest	true	"Connection details"
//	@Success		200		{object}	TestRepositoryResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/test-connection [post]
func (h *RepositoriesHandler) TestConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req TestConnectionRequest
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

	// Convert config map to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config format"})
		return
	}

	// Parse the backend from config
	backend, err := backends.ParseBackend(req.Type, configJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backend config: " + err.Error()})
		return
	}

	// Validate the backend configuration
	if err := backend.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Test the connection
	h.logger.Info().Str("type", string(req.Type)).Msg("testing backend connection")

	if err := backend.TestConnection(); err != nil {
		h.logger.Warn().Err(err).Str("type", string(req.Type)).Msg("backend connection test failed")
		c.JSON(http.StatusOK, TestRepositoryResponse{
			Success: false,
			Message: "Connection failed: " + err.Error(),
		})
		return
	}

	h.logger.Info().Str("type", string(req.Type)).Msg("backend connection test successful")
	c.JSON(http.StatusOK, TestRepositoryResponse{
		Success: true,
		Message: "Connection successful",
	})
}

// RecoverKey recovers the repository password for admins (requires escrow to be enabled).
//
//	@Summary		Recover repository key
//	@Description	Recovers the repository password for administrators. Only available when key escrow is enabled for the repository.
//	@Tags			Repositories
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	KeyRecoveryResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id}/key/recover [get]
func (h *RepositoriesHandler) RecoverKey(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to get repository")
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Only admins can recover keys
	if !dbUser.IsAdmin() {
		h.logger.Warn().
			Str("user_id", user.ID.String()).
			Str("repo_id", id.String()).
			Msg("non-admin attempted key recovery")
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can recover repository keys"})
		return
	}

	// Get the repository key
	repoKey, err := h.store.GetRepositoryKeyByRepositoryID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to get repository key")
		c.JSON(http.StatusNotFound, gin.H{"error": "repository key not found"})
		return
	}

	// Check if escrow is enabled
	if !repoKey.EscrowEnabled || len(repoKey.EscrowEncryptedKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key escrow is not enabled for this repository"})
		return
	}

	// Decrypt the escrow key
	password, err := h.keyManager.Decrypt(repoKey.EscrowEncryptedKey)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to decrypt escrow key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to recover key"})
		return
	}

	h.logger.Info().
		Str("repo_id", id.String()).
		Str("admin_id", dbUser.ID.String()).
		Msg("repository key recovered by admin")

	c.JSON(http.StatusOK, KeyRecoveryResponse{
		RepositoryID:   repo.ID,
		RepositoryName: repo.Name,
		Password:       string(password),
	})
}
