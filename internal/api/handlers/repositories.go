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

// RepositoryStore defines the interface for repository persistence operations.
type RepositoryStore interface {
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	CreateRepository(ctx context.Context, repo *models.Repository) error
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	DeleteRepository(ctx context.Context, id uuid.UUID) error
}

// RepositoriesHandler handles repository-related HTTP endpoints.
type RepositoriesHandler struct {
	store  RepositoryStore
	logger zerolog.Logger
}

// NewRepositoriesHandler creates a new RepositoriesHandler.
func NewRepositoriesHandler(store RepositoryStore, logger zerolog.Logger) *RepositoriesHandler {
	return &RepositoriesHandler{
		store:  store,
		logger: logger.With().Str("component", "repositories_handler").Logger(),
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
	}
}

// CreateRepositoryRequest is the request body for creating a repository.
type CreateRepositoryRequest struct {
	Name   string                `json:"name" binding:"required,min=1,max=255"`
	Type   models.RepositoryType `json:"type" binding:"required"`
	Config map[string]any        `json:"config" binding:"required"`
}

// UpdateRepositoryRequest is the request body for updating a repository.
type UpdateRepositoryRequest struct {
	Name   string         `json:"name,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

// RepositoryResponse is the API response for a repository (without encrypted config).
type RepositoryResponse struct {
	ID        uuid.UUID             `json:"id"`
	Name      string                `json:"name"`
	Type      models.RepositoryType `json:"type"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
}

// toResponse converts a Repository model to a RepositoryResponse.
func toRepositoryResponse(r *models.Repository) RepositoryResponse {
	return RepositoryResponse{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List returns all repositories for the authenticated user's organization.
// GET /api/v1/repositories
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
		responses[i] = toRepositoryResponse(r)
	}

	c.JSON(http.StatusOK, gin.H{"repositories": responses})
}

// Get returns a specific repository by ID.
// GET /api/v1/repositories/:id
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

	c.JSON(http.StatusOK, toRepositoryResponse(repo))
}

// Create creates a new repository.
// POST /api/v1/repositories
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

	// TODO: Encrypt config using internal/crypto package (AES-256-GCM)
	// For now, we store an empty config. Encryption will be added when
	// the crypto package is implemented.
	var configEncrypted []byte

	repo := models.NewRepository(user.CurrentOrgID, req.Name, req.Type, configEncrypted)

	if err := h.store.CreateRepository(c.Request.Context(), repo); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create repository"})
		return
	}

	h.logger.Info().
		Str("repo_id", repo.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Msg("repository created")

	c.JSON(http.StatusCreated, toRepositoryResponse(repo))
}

// Update updates an existing repository.
// PUT /api/v1/repositories/:id
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
	// TODO: Update encrypted config when crypto package is available
	// if req.Config != nil { ... }

	if err := h.store.UpdateRepository(c.Request.Context(), repo); err != nil {
		h.logger.Error().Err(err).Str("repo_id", id.String()).Msg("failed to update repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update repository"})
		return
	}

	h.logger.Info().Str("repo_id", id.String()).Msg("repository updated")
	c.JSON(http.StatusOK, toRepositoryResponse(repo))
}

// Delete removes a repository.
// DELETE /api/v1/repositories/:id
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

// Test checks the repository connection.
// POST /api/v1/repositories/:id/test
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

	// TODO: Implement actual repository connection test using Restic
	// This will be implemented when the backup package is created.
	// For now, return a placeholder response.
	h.logger.Info().Str("repo_id", id.String()).Msg("repository test requested")

	c.JSON(http.StatusOK, TestRepositoryResponse{
		Success: true,
		Message: "Repository connection test not yet implemented. Repository exists and is accessible.",
	})
}
