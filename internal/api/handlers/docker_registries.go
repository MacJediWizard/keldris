package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DockerRegistryStore defines the interface for Docker registry persistence operations.
type DockerRegistryStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetDockerRegistriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistry, error)
	GetDockerRegistryByID(ctx context.Context, id uuid.UUID) (*models.DockerRegistry, error)
	GetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID) (*models.DockerRegistry, error)
	CreateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error
	UpdateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error
	DeleteDockerRegistry(ctx context.Context, id uuid.UUID) error
	UpdateDockerRegistryHealth(ctx context.Context, id uuid.UUID, status models.DockerRegistryHealthStatus, errorMsg *string) error
	UpdateDockerRegistryCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte, expiresAt *time.Time) error
	SetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID, registryID uuid.UUID) error
	GetDockerRegistriesWithExpiringCredentials(ctx context.Context, orgID uuid.UUID, before time.Time) ([]*models.DockerRegistry, error)
	CreateDockerRegistryAuditLog(ctx context.Context, orgID, registryID uuid.UUID, userID *uuid.UUID, action string, details map[string]interface{}, ipAddress, userAgent string) error
}

// DockerRegistriesHandler handles Docker registry-related HTTP endpoints.
type DockerRegistriesHandler struct {
	store       DockerRegistryStore
	keyManager  *crypto.KeyManager
	registryMgr *docker.RegistryManager
	logger      zerolog.Logger
}

// NewDockerRegistriesHandler creates a new DockerRegistriesHandler.
func NewDockerRegistriesHandler(store DockerRegistryStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *DockerRegistriesHandler {
	return &DockerRegistriesHandler{
		store:       store,
		keyManager:  keyManager,
		registryMgr: docker.NewRegistryManager(store, keyManager, logger),
		logger:      logger.With().Str("component", "docker_registries_handler").Logger(),
	}
}

// RegisterRoutes registers Docker registry routes on the given router group.
func (h *DockerRegistriesHandler) RegisterRoutes(r *gin.RouterGroup) {
	registries := r.Group("/docker-registries")
	{
		registries.GET("", h.ListRegistries)
		registries.POST("", h.CreateRegistry)
		registries.GET("/types", h.ListRegistryTypes)
		registries.GET("/expiring", h.ListExpiringCredentials)
		registries.GET("/:id", h.GetRegistry)
		registries.PUT("/:id", h.UpdateRegistry)
		registries.DELETE("/:id", h.DeleteRegistry)
		registries.POST("/:id/login", h.Login)
		registries.POST("/:id/health-check", h.HealthCheck)
		registries.POST("/:id/rotate-credentials", h.RotateCredentials)
		registries.POST("/:id/set-default", h.SetDefault)
		registries.POST("/login-all", h.LoginAll)
		registries.POST("/health-check-all", h.HealthCheckAll)
	}
}

// ListRegistries returns all Docker registries for the authenticated user's organization.
// GET /api/v1/docker-registries
func (h *DockerRegistriesHandler) ListRegistries(c *gin.Context) {
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

	registries, err := h.store.GetDockerRegistriesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list docker registries")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list docker registries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registries": registries})
}

// GetRegistry returns a specific Docker registry by ID.
// GET /api/v1/docker-registries/:id
func (h *DockerRegistriesHandler) GetRegistry(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("registry_id", id.String()).Msg("failed to get docker registry")
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registry": registry})
}

// CreateRegistryRequest is the request body for creating a Docker registry.
type CreateRegistryRequest struct {
	Name        string                     `json:"name" binding:"required,min=1,max=255"`
	Type        models.DockerRegistryType  `json:"type" binding:"required"`
	URL         string                     `json:"url"`
	Credentials RegistryCredentialsRequest `json:"credentials" binding:"required"`
	IsDefault   bool                       `json:"is_default"`
}

// RegistryCredentialsRequest is the request body for registry credentials.
type RegistryCredentialsRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password,omitempty"`
	AccessToken        string `json:"access_token,omitempty"`
	AWSAccessKeyID     string `json:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey string `json:"aws_secret_access_key,omitempty"`
	AWSRegion          string `json:"aws_region,omitempty"`
	AzureTenantID      string `json:"azure_tenant_id,omitempty"`
	AzureClientID      string `json:"azure_client_id,omitempty"`
	AzureClientSecret  string `json:"azure_client_secret,omitempty"`
	GCRKeyJSON         string `json:"gcr_key_json,omitempty"`
}

// CreateRegistry creates a new Docker registry.
// POST /api/v1/docker-registries
func (h *DockerRegistriesHandler) CreateRegistry(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate registry type
	validType := false
	for _, t := range models.ValidDockerRegistryTypes() {
		if req.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry type"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	credentials := &models.DockerRegistryCredentials{
		Username:           req.Credentials.Username,
		Password:           req.Credentials.Password,
		AccessToken:        req.Credentials.AccessToken,
		AWSAccessKeyID:     req.Credentials.AWSAccessKeyID,
		AWSSecretAccessKey: req.Credentials.AWSSecretAccessKey,
		AWSRegion:          req.Credentials.AWSRegion,
		AzureTenantID:      req.Credentials.AzureTenantID,
		AzureClientID:      req.Credentials.AzureClientID,
		AzureClientSecret:  req.Credentials.AzureClientSecret,
		GCRKeyJSON:         req.Credentials.GCRKeyJSON,
	}

	registry, err := h.registryMgr.CreateRegistry(c.Request.Context(), dbUser.OrgID, req.Name, req.Type, req.URL, credentials, &user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create docker registry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create docker registry: " + err.Error()})
		return
	}

	// Set as default if requested
	if req.IsDefault {
		if err := h.store.SetDefaultDockerRegistry(c.Request.Context(), dbUser.OrgID, registry.ID); err != nil {
			h.logger.Warn().Err(err).Str("registry_id", registry.ID.String()).Msg("failed to set registry as default")
		} else {
			registry.IsDefault = true
		}
	}

	// Create audit log
	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, registry.ID, &user.ID, "create",
		map[string]interface{}{"name": req.Name, "type": req.Type}, c.ClientIP(), c.Request.UserAgent())

	h.logger.Info().
		Str("registry_id", registry.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Str("org_id", dbUser.OrgID.String()).
		Msg("docker registry created")

	c.JSON(http.StatusCreated, registry)
}

// UpdateRegistryRequest is the request body for updating a Docker registry.
type UpdateRegistryRequest struct {
	Name      *string `json:"name,omitempty"`
	URL       *string `json:"url,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
}

// UpdateRegistry updates an existing Docker registry.
// PUT /api/v1/docker-registries/:id
func (h *DockerRegistriesHandler) UpdateRegistry(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	var req UpdateRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		registry.Name = *req.Name
	}
	if req.URL != nil {
		registry.URL = *req.URL
	}
	if req.Enabled != nil {
		registry.Enabled = *req.Enabled
	}

	if err := h.store.UpdateDockerRegistry(c.Request.Context(), registry); err != nil {
		h.logger.Error().Err(err).Str("registry_id", id.String()).Msg("failed to update docker registry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update docker registry"})
		return
	}

	// Handle default setting separately
	if req.IsDefault != nil && *req.IsDefault {
		if err := h.store.SetDefaultDockerRegistry(c.Request.Context(), dbUser.OrgID, id); err != nil {
			h.logger.Warn().Err(err).Str("registry_id", id.String()).Msg("failed to set registry as default")
		} else {
			registry.IsDefault = true
		}
	}

	// Create audit log
	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, registry.ID, &user.ID, "update",
		map[string]interface{}{"name": registry.Name}, c.ClientIP(), c.Request.UserAgent())

	h.logger.Info().Str("registry_id", id.String()).Msg("docker registry updated")
	c.JSON(http.StatusOK, registry)
}

// DeleteRegistry removes a Docker registry.
// DELETE /api/v1/docker-registries/:id
func (h *DockerRegistriesHandler) DeleteRegistry(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	// Create audit log before deletion
	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, id, &user.ID, "delete",
		map[string]interface{}{"name": registry.Name}, c.ClientIP(), c.Request.UserAgent())

	if err := h.registryMgr.DeleteRegistry(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("registry_id", id.String()).Msg("failed to delete docker registry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete docker registry"})
		return
	}

	h.logger.Info().Str("registry_id", id.String()).Msg("docker registry deleted")
	c.JSON(http.StatusOK, gin.H{"message": "docker registry deleted"})
}

// Login performs Docker login to a registry.
// POST /api/v1/docker-registries/:id/login
func (h *DockerRegistriesHandler) Login(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	result, err := h.registryMgr.Login(c.Request.Context(), id)
	if err != nil {
		_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, id, &user.ID, "login",
			map[string]interface{}{"success": false, "error": err.Error()}, c.ClientIP(), c.Request.UserAgent())
		c.JSON(http.StatusBadRequest, gin.H{"error": "login failed", "result": result})
		return
	}

	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, id, &user.ID, "login",
		map[string]interface{}{"success": true}, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// LoginAll logs into all enabled registries for the organization.
// POST /api/v1/docker-registries/login-all
func (h *DockerRegistriesHandler) LoginAll(c *gin.Context) {
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

	results, err := h.registryMgr.LoginAll(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// HealthCheck performs a health check on a registry.
// POST /api/v1/docker-registries/:id/health-check
func (h *DockerRegistriesHandler) HealthCheck(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	result, err := h.registryMgr.HealthCheck(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "health check failed: " + err.Error()})
		return
	}

	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, id, &user.ID, "health_check",
		map[string]interface{}{"status": result.Status}, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// HealthCheckAll performs health checks on all registries for the organization.
// POST /api/v1/docker-registries/health-check-all
func (h *DockerRegistriesHandler) HealthCheckAll(c *gin.Context) {
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

	results, err := h.registryMgr.HealthCheckAll(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "health check failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// RotateCredentialsRequest is the request body for rotating registry credentials.
type RotateCredentialsRequest struct {
	Credentials RegistryCredentialsRequest `json:"credentials" binding:"required"`
	ExpiresAt   *time.Time                 `json:"expires_at,omitempty"`
}

// RotateCredentials rotates the credentials for a registry.
// POST /api/v1/docker-registries/:id/rotate-credentials
func (h *DockerRegistriesHandler) RotateCredentials(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	var req RotateCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	credentials := &models.DockerRegistryCredentials{
		Username:           req.Credentials.Username,
		Password:           req.Credentials.Password,
		AccessToken:        req.Credentials.AccessToken,
		AWSAccessKeyID:     req.Credentials.AWSAccessKeyID,
		AWSSecretAccessKey: req.Credentials.AWSSecretAccessKey,
		AWSRegion:          req.Credentials.AWSRegion,
		AzureTenantID:      req.Credentials.AzureTenantID,
		AzureClientID:      req.Credentials.AzureClientID,
		AzureClientSecret:  req.Credentials.AzureClientSecret,
		GCRKeyJSON:         req.Credentials.GCRKeyJSON,
	}

	if err := h.registryMgr.UpdateCredentials(c.Request.Context(), id, credentials, req.ExpiresAt); err != nil {
		h.logger.Error().Err(err).Str("registry_id", id.String()).Msg("failed to rotate credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate credentials: " + err.Error()})
		return
	}

	_ = h.store.CreateDockerRegistryAuditLog(c.Request.Context(), dbUser.OrgID, id, &user.ID, "rotate_credentials",
		map[string]interface{}{"expires_at": req.ExpiresAt}, c.ClientIP(), c.Request.UserAgent())

	h.logger.Info().Str("registry_id", id.String()).Msg("docker registry credentials rotated")
	c.JSON(http.StatusOK, gin.H{"message": "credentials rotated successfully"})
}

// SetDefault sets a registry as the default for the organization.
// POST /api/v1/docker-registries/:id/set-default
func (h *DockerRegistriesHandler) SetDefault(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry ID"})
		return
	}

	registry, err := h.store.GetDockerRegistryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if registry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker registry not found"})
		return
	}

	if err := h.store.SetDefaultDockerRegistry(c.Request.Context(), dbUser.OrgID, id); err != nil {
		h.logger.Error().Err(err).Str("registry_id", id.String()).Msg("failed to set default registry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set default registry"})
		return
	}

	h.logger.Info().Str("registry_id", id.String()).Msg("docker registry set as default")
	c.JSON(http.StatusOK, gin.H{"message": "registry set as default"})
}

// ListRegistryTypes returns information about available Docker registry types.
// GET /api/v1/docker-registries/types
func (h *DockerRegistriesHandler) ListRegistryTypes(c *gin.Context) {
	types := []map[string]interface{}{
		{
			"type":        models.DockerRegistryTypeDockerHub,
			"name":        "Docker Hub",
			"description": "Docker Hub registry (hub.docker.com)",
			"default_url": models.GetDefaultURL(models.DockerRegistryTypeDockerHub),
			"fields":      []string{"username", "password"},
		},
		{
			"type":        models.DockerRegistryTypeGCR,
			"name":        "Google Container Registry",
			"description": "Google Cloud Container Registry (gcr.io)",
			"default_url": models.GetDefaultURL(models.DockerRegistryTypeGCR),
			"fields":      []string{"gcr_key_json"},
		},
		{
			"type":        models.DockerRegistryTypeECR,
			"name":        "Amazon ECR",
			"description": "Amazon Elastic Container Registry",
			"default_url": "",
			"fields":      []string{"aws_access_key_id", "aws_secret_access_key", "aws_region"},
		},
		{
			"type":        models.DockerRegistryTypeACR,
			"name":        "Azure Container Registry",
			"description": "Azure Container Registry (azurecr.io)",
			"default_url": "",
			"fields":      []string{"azure_tenant_id", "azure_client_id", "azure_client_secret"},
		},
		{
			"type":        models.DockerRegistryTypeGHCR,
			"name":        "GitHub Container Registry",
			"description": "GitHub Container Registry (ghcr.io)",
			"default_url": models.GetDefaultURL(models.DockerRegistryTypeGHCR),
			"fields":      []string{"username", "access_token"},
		},
		{
			"type":        models.DockerRegistryTypePrivate,
			"name":        "Private Registry",
			"description": "Self-hosted private Docker registry",
			"default_url": "",
			"fields":      []string{"username", "password"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// ListExpiringCredentials returns registries with credentials expiring soon.
// GET /api/v1/docker-registries/expiring
func (h *DockerRegistriesHandler) ListExpiringCredentials(c *gin.Context) {
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

	// Default to 30 days warning
	days := 30
	warningThreshold := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	registries, err := h.store.GetDockerRegistriesWithExpiringCredentials(c.Request.Context(), dbUser.OrgID, warningThreshold)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get expiring credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get expiring credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registries": registries, "warning_days": days})
}
