package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/integrations/komodo"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// KomodoStore defines the interface for Komodo persistence operations.
type KomodoStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	// Integration CRUD
	GetKomodoIntegrationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoIntegration, error)
	GetKomodoIntegrationByID(ctx context.Context, id uuid.UUID) (*models.KomodoIntegration, error)
	CreateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error
	UpdateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error
	DeleteKomodoIntegration(ctx context.Context, id uuid.UUID) error
	// Container CRUD
	GetKomodoContainersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoContainer, error)
	GetKomodoContainersByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoContainer, error)
	GetKomodoContainerByID(ctx context.Context, id uuid.UUID) (*models.KomodoContainer, error)
	GetKomodoContainerByKomodoID(ctx context.Context, integrationID uuid.UUID, komodoID string) (*models.KomodoContainer, error)
	CreateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error
	UpdateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error
	DeleteKomodoContainer(ctx context.Context, id uuid.UUID) error
	UpsertKomodoContainer(ctx context.Context, container *models.KomodoContainer) error
	// Stack CRUD
	GetKomodoStacksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoStack, error)
	GetKomodoStacksByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoStack, error)
	GetKomodoStackByID(ctx context.Context, id uuid.UUID) (*models.KomodoStack, error)
	UpsertKomodoStack(ctx context.Context, stack *models.KomodoStack) error
	// Webhook events
	CreateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error
	GetKomodoWebhookEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.KomodoWebhookEvent, error)
	UpdateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error
}

// KomodoHandler handles Komodo integration HTTP endpoints.
type KomodoHandler struct {
	store          KomodoStore
	webhookHandler *komodo.WebhookHandler
	logger         zerolog.Logger
}

// NewKomodoHandler creates a new KomodoHandler.
func NewKomodoHandler(store KomodoStore, logger zerolog.Logger) *KomodoHandler {
	return &KomodoHandler{
		store:          store,
		webhookHandler: komodo.NewWebhookHandler(logger),
		logger:         logger.With().Str("component", "komodo_handler").Logger(),
	}
}

// RegisterRoutes registers Komodo routes on the given router group.
func (h *KomodoHandler) RegisterRoutes(r *gin.RouterGroup) {
	komodoRoutes := r.Group("/integrations/komodo")
	{
		// Integration management
		komodoRoutes.GET("", h.ListIntegrations)
		komodoRoutes.POST("", h.CreateIntegration)
		komodoRoutes.GET("/:id", h.GetIntegration)
		komodoRoutes.PUT("/:id", h.UpdateIntegration)
		komodoRoutes.DELETE("/:id", h.DeleteIntegration)
		komodoRoutes.POST("/:id/test", h.TestConnection)
		komodoRoutes.POST("/:id/sync", h.SyncIntegration)

		// Discovery
		komodoRoutes.GET("/:id/discover", h.DiscoverContainers)

		// Containers
		komodoRoutes.GET("/containers", h.ListContainers)
		komodoRoutes.GET("/containers/:containerID", h.GetContainer)
		komodoRoutes.PUT("/containers/:containerID", h.UpdateContainer)

		// Stacks
		komodoRoutes.GET("/stacks", h.ListStacks)
		komodoRoutes.GET("/stacks/:stackID", h.GetStack)

		// Webhook events
		komodoRoutes.GET("/events", h.ListWebhookEvents)
	}
}

// RegisterWebhookRoutes registers public webhook routes (no auth required).
func (h *KomodoHandler) RegisterWebhookRoutes(r *gin.Engine) {
	r.POST("/webhooks/komodo/:integrationID", h.HandleWebhook)
}

// ListIntegrations returns all Komodo integrations for the authenticated user's organization.
// GET /api/v1/integrations/komodo
func (h *KomodoHandler) ListIntegrations(c *gin.Context) {
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

	integrations, err := h.store.GetKomodoIntegrationsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list Komodo integrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list integrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"integrations": integrations})
}

// CreateIntegrationRequest is the request body for creating a Komodo integration.
type CreateIntegrationRequest struct {
	Name   string                          `json:"name" binding:"required,min=1,max=255"`
	URL    string                          `json:"url" binding:"required"`
	Config models.KomodoIntegrationConfig  `json:"config" binding:"required"`
}

// CreateIntegration creates a new Komodo integration.
// POST /api/v1/integrations/komodo
func (h *KomodoHandler) CreateIntegration(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Serialize config (should be encrypted in production)
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process configuration"})
		return
	}

	integration := models.NewKomodoIntegration(dbUser.OrgID, req.Name, req.URL, configJSON)

	if err := h.store.CreateKomodoIntegration(c.Request.Context(), integration); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create Komodo integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create integration"})
		return
	}

	h.logger.Info().
		Str("integration_id", integration.ID.String()).
		Str("name", req.Name).
		Str("org_id", dbUser.OrgID.String()).
		Msg("Komodo integration created")

	c.JSON(http.StatusCreated, integration)
}

// GetIntegration returns a specific Komodo integration by ID.
// GET /api/v1/integrations/komodo/:id
func (h *KomodoHandler) GetIntegration(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("integration_id", id.String()).Msg("failed to get Komodo integration")
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Get containers for this integration
	containers, err := h.store.GetKomodoContainersByIntegrationID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("integration_id", id.String()).Msg("failed to get containers")
	}

	// Get stacks for this integration
	stacks, err := h.store.GetKomodoStacksByIntegrationID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("integration_id", id.String()).Msg("failed to get stacks")
	}

	c.JSON(http.StatusOK, gin.H{
		"integration": integration,
		"containers":  containers,
		"stacks":      stacks,
	})
}

// UpdateIntegrationRequest is the request body for updating a Komodo integration.
type UpdateIntegrationRequest struct {
	Name    *string                          `json:"name,omitempty"`
	URL     *string                          `json:"url,omitempty"`
	Config  *models.KomodoIntegrationConfig  `json:"config,omitempty"`
	Enabled *bool                            `json:"enabled,omitempty"`
}

// UpdateIntegration updates an existing Komodo integration.
// PUT /api/v1/integrations/komodo/:id
func (h *KomodoHandler) UpdateIntegration(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	var req UpdateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		integration.Name = *req.Name
	}
	if req.URL != nil {
		integration.URL = *req.URL
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process configuration"})
			return
		}
		integration.ConfigEncrypted = configJSON
	}
	if req.Enabled != nil {
		integration.Enabled = *req.Enabled
	}

	if err := h.store.UpdateKomodoIntegration(c.Request.Context(), integration); err != nil {
		h.logger.Error().Err(err).Str("integration_id", id.String()).Msg("failed to update Komodo integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update integration"})
		return
	}

	h.logger.Info().Str("integration_id", id.String()).Msg("Komodo integration updated")
	c.JSON(http.StatusOK, integration)
}

// DeleteIntegration removes a Komodo integration.
// DELETE /api/v1/integrations/komodo/:id
func (h *KomodoHandler) DeleteIntegration(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	if err := h.store.DeleteKomodoIntegration(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("integration_id", id.String()).Msg("failed to delete Komodo integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete integration"})
		return
	}

	h.logger.Info().Str("integration_id", id.String()).Msg("Komodo integration deleted")
	c.JSON(http.StatusOK, gin.H{"message": "integration deleted"})
}

// TestConnection tests the connection to a Komodo instance.
// POST /api/v1/integrations/komodo/:id/test
func (h *KomodoHandler) TestConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Parse config
	var config models.KomodoIntegrationConfig
	if err := json.Unmarshal(integration.ConfigEncrypted, &config); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse integration config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse configuration"})
		return
	}

	// Create client and test connection
	client, err := komodo.NewClient(komodo.ClientConfig{
		BaseURL:  integration.URL,
		APIKey:   config.APIKey,
		Username: config.Username,
		Password: config.Password,
	}, h.logger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.TestConnection(c.Request.Context()); err != nil {
		integration.MarkError(err.Error())
		h.store.UpdateKomodoIntegration(c.Request.Context(), integration)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "connection test failed",
			"details": err.Error(),
		})
		return
	}

	integration.MarkConnected()
	if err := h.store.UpdateKomodoIntegration(c.Request.Context(), integration); err != nil {
		h.logger.Error().Err(err).Msg("failed to update integration status")
	}

	h.logger.Info().
		Str("integration_id", id.String()).
		Msg("Komodo connection test successful")

	c.JSON(http.StatusOK, gin.H{
		"message": "connection successful",
		"status":  integration.Status,
	})
}

// SyncIntegration triggers a sync with Komodo to update local data.
// POST /api/v1/integrations/komodo/:id/sync
func (h *KomodoHandler) SyncIntegration(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Parse config
	var config models.KomodoIntegrationConfig
	if err := json.Unmarshal(integration.ConfigEncrypted, &config); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse integration config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse configuration"})
		return
	}

	// Create client
	client, err := komodo.NewClient(komodo.ClientConfig{
		BaseURL:  integration.URL,
		APIKey:   config.APIKey,
		Username: config.Username,
		Password: config.Password,
	}, h.logger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run discovery
	discoveryService := komodo.NewDiscoveryService(client, h.logger)
	result, err := discoveryService.DiscoverAll(c.Request.Context(), dbUser.OrgID, id)
	if err != nil {
		h.logger.Error().Err(err).Msg("discovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "discovery failed: " + err.Error()})
		return
	}

	// Upsert discovered stacks
	for _, stack := range result.Stacks {
		if err := h.store.UpsertKomodoStack(c.Request.Context(), stack); err != nil {
			h.logger.Error().Err(err).Str("stack_id", stack.KomodoID).Msg("failed to upsert stack")
		}
	}

	// Upsert discovered containers
	for _, container := range result.Containers {
		if err := h.store.UpsertKomodoContainer(c.Request.Context(), container); err != nil {
			h.logger.Error().Err(err).Str("container_id", container.KomodoID).Msg("failed to upsert container")
		}
	}

	integration.MarkConnected()
	h.store.UpdateKomodoIntegration(c.Request.Context(), integration)

	h.logger.Info().
		Str("integration_id", id.String()).
		Int("stacks", len(result.Stacks)).
		Int("containers", len(result.Containers)).
		Msg("Komodo sync completed")

	c.JSON(http.StatusOK, gin.H{
		"message":    "sync completed",
		"stacks":     len(result.Stacks),
		"containers": len(result.Containers),
		"result":     result,
	})
}

// DiscoverContainers discovers containers from Komodo without persisting.
// GET /api/v1/integrations/komodo/:id/discover
func (h *KomodoHandler) DiscoverContainers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if integration.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Parse config
	var config models.KomodoIntegrationConfig
	if err := json.Unmarshal(integration.ConfigEncrypted, &config); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse integration config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse configuration"})
		return
	}

	// Create client
	client, err := komodo.NewClient(komodo.ClientConfig{
		BaseURL:  integration.URL,
		APIKey:   config.APIKey,
		Username: config.Username,
		Password: config.Password,
	}, h.logger)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Run discovery
	discoveryService := komodo.NewDiscoveryService(client, h.logger)
	result, err := discoveryService.DiscoverAll(c.Request.Context(), dbUser.OrgID, id)
	if err != nil {
		h.logger.Error().Err(err).Msg("discovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "discovery failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListContainers returns all discovered Komodo containers.
// GET /api/v1/integrations/komodo/containers
func (h *KomodoHandler) ListContainers(c *gin.Context) {
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

	containers, err := h.store.GetKomodoContainersByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list Komodo containers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list containers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"containers": containers})
}

// GetContainer returns a specific Komodo container by ID.
// GET /api/v1/integrations/komodo/containers/:containerID
func (h *KomodoHandler) GetContainer(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("containerID")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	container, err := h.store.GetKomodoContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if container.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	c.JSON(http.StatusOK, container)
}

// UpdateContainerRequest is the request body for updating a Komodo container.
type UpdateContainerRequest struct {
	AgentID       *uuid.UUID `json:"agent_id,omitempty"`
	BackupEnabled *bool      `json:"backup_enabled,omitempty"`
}

// UpdateContainer updates a Komodo container's Keldris settings.
// PUT /api/v1/integrations/komodo/containers/:containerID
func (h *KomodoHandler) UpdateContainer(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("containerID")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid container ID"})
		return
	}

	var req UpdateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	container, err := h.store.GetKomodoContainerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if container.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	// Apply updates
	if req.AgentID != nil {
		container.AgentID = req.AgentID
	}
	if req.BackupEnabled != nil {
		container.BackupEnabled = *req.BackupEnabled
	}

	if err := h.store.UpdateKomodoContainer(c.Request.Context(), container); err != nil {
		h.logger.Error().Err(err).Str("container_id", id.String()).Msg("failed to update Komodo container")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update container"})
		return
	}

	h.logger.Info().Str("container_id", id.String()).Msg("Komodo container updated")
	c.JSON(http.StatusOK, container)
}

// ListStacks returns all discovered Komodo stacks.
// GET /api/v1/integrations/komodo/stacks
func (h *KomodoHandler) ListStacks(c *gin.Context) {
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

	stacks, err := h.store.GetKomodoStacksByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list Komodo stacks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list stacks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stacks": stacks})
}

// GetStack returns a specific Komodo stack by ID.
// GET /api/v1/integrations/komodo/stacks/:stackID
func (h *KomodoHandler) GetStack(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("stackID")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stack ID"})
		return
	}

	stack, err := h.store.GetKomodoStackByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stack not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if stack.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "stack not found"})
		return
	}

	c.JSON(http.StatusOK, stack)
}

// ListWebhookEvents returns recent webhook events.
// GET /api/v1/integrations/komodo/events
func (h *KomodoHandler) ListWebhookEvents(c *gin.Context) {
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

	events, err := h.store.GetKomodoWebhookEventsByOrgID(c.Request.Context(), dbUser.OrgID, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list webhook events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// HandleWebhook processes incoming webhooks from Komodo.
// POST /webhooks/komodo/:integrationID
func (h *KomodoHandler) HandleWebhook(c *gin.Context) {
	idParam := c.Param("integrationID")
	integrationID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid integration ID"})
		return
	}

	integration, err := h.store.GetKomodoIntegrationByID(c.Request.Context(), integrationID)
	if err != nil {
		h.logger.Error().Err(err).Str("integration_id", integrationID.String()).Msg("webhook: integration not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}

	// Read payload
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to read webhook payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read payload"})
		return
	}

	// Process the webhook
	result, err := h.webhookHandler.Process(c.Request.Context(), integration.OrgID, integrationID, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to process webhook")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to process webhook: " + err.Error()})
		return
	}

	// Store the webhook event
	if err := h.store.CreateKomodoWebhookEvent(c.Request.Context(), result.Event); err != nil {
		h.logger.Error().Err(err).Msg("failed to store webhook event")
	}

	h.logger.Info().
		Str("integration_id", integrationID.String()).
		Str("event_type", string(result.EventType)).
		Bool("should_backup", result.ShouldBackup).
		Msg("Komodo webhook received")

	// If backup should be triggered, we would initiate it here
	// This would integrate with the backup scheduling system
	if result.ShouldBackup && result.BackupTrigger != nil {
		h.logger.Info().
			Str("container_id", result.BackupTrigger.ContainerID).
			Str("stack_id", result.BackupTrigger.StackID).
			Msg("backup trigger received from Komodo webhook")
		// TODO: Integrate with backup scheduler to trigger backup
	}

	// Mark event as processed
	result.Event.MarkProcessed()
	h.store.UpdateKomodoWebhookEvent(c.Request.Context(), result.Event)

	c.JSON(http.StatusOK, gin.H{
		"message":       "webhook processed",
		"event_type":    result.EventType,
		"should_backup": result.ShouldBackup,
	})
}
