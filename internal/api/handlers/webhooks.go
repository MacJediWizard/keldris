package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// WebhooksStore defines the interface for webhook persistence operations.
type WebhooksStore interface {
	GetWebhookEndpointsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.WebhookEndpoint, error)
	GetWebhookEndpointByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error)
	GetEnabledWebhookEndpointsForEvent(ctx context.Context, orgID uuid.UUID, eventType models.WebhookEventType) ([]*models.WebhookEndpoint, error)
	CreateWebhookEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error
	UpdateWebhookEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error
	DeleteWebhookEndpoint(ctx context.Context, id uuid.UUID) error
	GetWebhookDeliveriesByOrgID(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, int, error)
	GetWebhookDeliveriesByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, int, error)
	GetWebhookDeliveryByID(ctx context.Context, id uuid.UUID) (*models.WebhookDelivery, error)
	CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	UpdateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	GetPendingWebhookDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error)
}

// WebhooksHandler handles webhook-related HTTP endpoints.
type WebhooksHandler struct {
	store      WebhooksStore
	keyManager *crypto.KeyManager
	dispatcher *webhooks.Dispatcher
	logger     zerolog.Logger
}

// NewWebhooksHandler creates a new WebhooksHandler.
func NewWebhooksHandler(store WebhooksStore, keyManager *crypto.KeyManager, dispatcher *webhooks.Dispatcher, logger zerolog.Logger) *WebhooksHandler {
	return &WebhooksHandler{
		store:      store,
		keyManager: keyManager,
		dispatcher: dispatcher,
		logger:     logger.With().Str("component", "webhooks_handler").Logger(),
	}
}

// RegisterRoutes registers webhook routes on the given router group.
func (h *WebhooksHandler) RegisterRoutes(r *gin.RouterGroup) {
	wh := r.Group("/webhooks")
	{
		// Event types
		wh.GET("/event-types", h.ListEventTypes)

		// Endpoints CRUD
		wh.GET("/endpoints", h.ListEndpoints)
		wh.POST("/endpoints", h.CreateEndpoint)
		wh.GET("/endpoints/:id", h.GetEndpoint)
		wh.PUT("/endpoints/:id", h.UpdateEndpoint)
		wh.DELETE("/endpoints/:id", h.DeleteEndpoint)
		wh.POST("/endpoints/:id/test", h.TestEndpoint)

		// Delivery logs
		wh.GET("/deliveries", h.ListDeliveries)
		wh.GET("/deliveries/:id", h.GetDelivery)
		wh.GET("/endpoints/:id/deliveries", h.ListEndpointDeliveries)
		wh.POST("/deliveries/:id/retry", h.RetryDelivery)
	}
}

// ListEventTypes returns all available webhook event types.
// GET /api/v1/webhooks/event-types
func (h *WebhooksHandler) ListEventTypes(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	c.JSON(http.StatusOK, models.WebhookEventTypesResponse{
		EventTypes: models.AllWebhookEventTypes(),
	})
}

// ListEndpoints returns all webhook endpoints for the organization.
// GET /api/v1/webhooks/endpoints
func (h *WebhooksHandler) ListEndpoints(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	endpoints, err := h.store.GetWebhookEndpointsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list webhook endpoints")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list webhook endpoints"})
		return
	}

	c.JSON(http.StatusOK, models.WebhookEndpointsResponse{Endpoints: endpoints})
}

// GetEndpoint returns a specific webhook endpoint by ID.
// GET /api/v1/webhooks/endpoints/:id
func (h *WebhooksHandler) GetEndpoint(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	endpoint, err := h.store.GetWebhookEndpointByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if endpoint.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	c.JSON(http.StatusOK, endpoint)
}

// CreateEndpoint creates a new webhook endpoint.
// POST /api/v1/webhooks/endpoints
func (h *WebhooksHandler) CreateEndpoint(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateWebhookEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate secret length
	if len(req.Secret) < 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "secret must be at least 16 characters"})
		return
	}

	// Encrypt the secret
	secretEncrypted, err := h.keyManager.Encrypt([]byte(req.Secret))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt webhook secret")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt secret"})
		return
	}

	endpoint := models.NewWebhookEndpoint(user.CurrentOrgID, req.Name, req.URL, secretEncrypted, req.EventTypes)
	if req.Headers != nil {
		endpoint.Headers = req.Headers
	}
	if req.RetryCount != nil {
		endpoint.RetryCount = *req.RetryCount
	}
	if req.TimeoutSeconds != nil {
		endpoint.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := h.store.CreateWebhookEndpoint(c.Request.Context(), endpoint); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create webhook endpoint")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create webhook endpoint"})
		return
	}

	h.logger.Info().
		Str("endpoint_id", endpoint.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("name", endpoint.Name).
		Msg("webhook endpoint created")

	c.JSON(http.StatusCreated, endpoint)
}

// UpdateEndpoint updates an existing webhook endpoint.
// PUT /api/v1/webhooks/endpoints/:id
func (h *WebhooksHandler) UpdateEndpoint(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	endpoint, err := h.store.GetWebhookEndpointByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if endpoint.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	var req models.UpdateWebhookEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Apply updates
	if req.Name != nil {
		endpoint.Name = *req.Name
	}
	if req.URL != nil {
		endpoint.URL = *req.URL
	}
	if req.Secret != nil {
		if len(*req.Secret) < 16 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "secret must be at least 16 characters"})
			return
		}
		secretEncrypted, err := h.keyManager.Encrypt([]byte(*req.Secret))
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt webhook secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt secret"})
			return
		}
		endpoint.SecretEncrypted = secretEncrypted
	}
	if req.Enabled != nil {
		endpoint.Enabled = *req.Enabled
	}
	if req.EventTypes != nil {
		endpoint.EventTypes = req.EventTypes
	}
	if req.Headers != nil {
		endpoint.Headers = req.Headers
	}
	if req.RetryCount != nil {
		endpoint.RetryCount = *req.RetryCount
	}
	if req.TimeoutSeconds != nil {
		endpoint.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := h.store.UpdateWebhookEndpoint(c.Request.Context(), endpoint); err != nil {
		h.logger.Error().Err(err).Str("endpoint_id", id.String()).Msg("failed to update webhook endpoint")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update webhook endpoint"})
		return
	}

	h.logger.Info().Str("endpoint_id", id.String()).Msg("webhook endpoint updated")
	c.JSON(http.StatusOK, endpoint)
}

// DeleteEndpoint deletes a webhook endpoint.
// DELETE /api/v1/webhooks/endpoints/:id
func (h *WebhooksHandler) DeleteEndpoint(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	endpoint, err := h.store.GetWebhookEndpointByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if endpoint.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if err := h.store.DeleteWebhookEndpoint(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("endpoint_id", id.String()).Msg("failed to delete webhook endpoint")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete webhook endpoint"})
		return
	}

	h.logger.Info().Str("endpoint_id", id.String()).Msg("webhook endpoint deleted")
	c.JSON(http.StatusOK, gin.H{"message": "endpoint deleted"})
}

// TestEndpoint sends a test webhook to an endpoint.
// POST /api/v1/webhooks/endpoints/:id/test
func (h *WebhooksHandler) TestEndpoint(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	endpoint, err := h.store.GetWebhookEndpointByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if endpoint.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	var req models.TestWebhookRequest
	_ = c.ShouldBindJSON(&req)

	eventType := models.WebhookEventBackupCompleted
	if req.EventType != "" {
		eventType = req.EventType
	}

	result, err := h.dispatcher.TestEndpoint(c.Request.Context(), endpoint, eventType)
	if err != nil {
		h.logger.Error().Err(err).Str("endpoint_id", id.String()).Msg("failed to test webhook endpoint")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to test webhook: " + err.Error()})
		return
	}

	h.logger.Info().
		Str("endpoint_id", id.String()).
		Bool("success", result.Success).
		Int("status", result.ResponseStatus).
		Int64("duration_ms", result.DurationMs).
		Msg("webhook endpoint tested")

	c.JSON(http.StatusOK, result)
}

// ListDeliveries returns webhook deliveries for the organization.
// GET /api/v1/webhooks/deliveries
func (h *WebhooksHandler) ListDeliveries(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	limit, offset := getPaginationParams(c)

	deliveries, total, err := h.store.GetWebhookDeliveriesByOrgID(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list webhook deliveries")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list webhook deliveries"})
		return
	}

	c.JSON(http.StatusOK, models.WebhookDeliveriesResponse{
		Deliveries: deliveries,
		Total:      total,
	})
}

// ListEndpointDeliveries returns webhook deliveries for a specific endpoint.
// GET /api/v1/webhooks/endpoints/:id/deliveries
func (h *WebhooksHandler) ListEndpointDeliveries(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint ID"})
		return
	}

	// Verify endpoint belongs to org
	endpoint, err := h.store.GetWebhookEndpointByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	if endpoint.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}

	limit, offset := getPaginationParams(c)

	deliveries, total, err := h.store.GetWebhookDeliveriesByEndpointID(c.Request.Context(), id, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("endpoint_id", id.String()).Msg("failed to list endpoint deliveries")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deliveries"})
		return
	}

	c.JSON(http.StatusOK, models.WebhookDeliveriesResponse{
		Deliveries: deliveries,
		Total:      total,
	})
}

// GetDelivery returns a specific webhook delivery by ID.
// GET /api/v1/webhooks/deliveries/:id
func (h *WebhooksHandler) GetDelivery(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delivery ID"})
		return
	}

	delivery, err := h.store.GetWebhookDeliveryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "delivery not found"})
		return
	}

	if delivery.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "delivery not found"})
		return
	}

	c.JSON(http.StatusOK, delivery)
}

// RetryDelivery manually retries a failed webhook delivery.
// POST /api/v1/webhooks/deliveries/:id/retry
func (h *WebhooksHandler) RetryDelivery(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delivery ID"})
		return
	}

	delivery, err := h.store.GetWebhookDeliveryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "delivery not found"})
		return
	}

	if delivery.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "delivery not found"})
		return
	}

	if delivery.Status == models.WebhookDeliveryStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{"error": "delivery was already successful"})
		return
	}

	// Reset for retry
	delivery.Status = models.WebhookDeliveryStatusPending
	delivery.AttemptNumber = 1
	delivery.NextRetryAt = nil
	delivery.ErrorMessage = ""

	if err := h.store.UpdateWebhookDelivery(c.Request.Context(), delivery); err != nil {
		h.logger.Error().Err(err).Str("delivery_id", id.String()).Msg("failed to queue delivery for retry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue retry"})
		return
	}

	h.logger.Info().Str("delivery_id", id.String()).Msg("webhook delivery queued for retry")
	c.JSON(http.StatusOK, gin.H{"message": "delivery queued for retry"})
}

// getPaginationParams extracts pagination parameters from the request.
func getPaginationParams(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return
}
