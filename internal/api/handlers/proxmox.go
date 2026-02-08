package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/vms"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ProxmoxStore defines the interface for Proxmox connection persistence operations.
type ProxmoxStore interface {
	CreateProxmoxConnection(ctx context.Context, conn *models.ProxmoxConnection) error
	GetProxmoxConnectionByID(ctx context.Context, id uuid.UUID) (*models.ProxmoxConnection, error)
	GetProxmoxConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ProxmoxConnection, error)
	UpdateProxmoxConnection(ctx context.Context, conn *models.ProxmoxConnection) error
	DeleteProxmoxConnection(ctx context.Context, id uuid.UUID) error
}

// EncryptionService provides encryption/decryption for sensitive data.
type EncryptionService interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// ProxmoxHandler handles Proxmox-related HTTP endpoints.
type ProxmoxHandler struct {
	store      ProxmoxStore
	encryption EncryptionService
	logger     zerolog.Logger
}

// NewProxmoxHandler creates a new ProxmoxHandler.
func NewProxmoxHandler(store ProxmoxStore, encryption EncryptionService, logger zerolog.Logger) *ProxmoxHandler {
	return &ProxmoxHandler{
		store:      store,
		encryption: encryption,
		logger:     logger.With().Str("component", "proxmox_handler").Logger(),
	}
}

// RegisterRoutes registers Proxmox routes on the given router group.
func (h *ProxmoxHandler) RegisterRoutes(r *gin.RouterGroup) {
	proxmox := r.Group("/proxmox")
	{
		connections := proxmox.Group("/connections")
		{
			connections.GET("", h.ListConnections)
			connections.POST("", h.CreateConnection)
			connections.GET("/:id", h.GetConnection)
			connections.PUT("/:id", h.UpdateConnection)
			connections.DELETE("/:id", h.DeleteConnection)
			connections.POST("/:id/test", h.TestConnection)
			connections.GET("/:id/vms", h.ListVMs)
		}
	}
}

// ProxmoxCreateConnectionRequest is the request body for creating a Proxmox connection.
type ProxmoxCreateConnectionRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port,omitempty"`
	Node        string `json:"node" binding:"required"`
	Username    string `json:"username" binding:"required"`
	TokenID     string `json:"token_id,omitempty"`
	TokenSecret string `json:"token_secret,omitempty"`
	VerifySSL   *bool  `json:"verify_ssl,omitempty"`
}

// ProxmoxUpdateConnectionRequest is the request body for updating a Proxmox connection.
type ProxmoxUpdateConnectionRequest struct {
	Name        string `json:"name,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        *int   `json:"port,omitempty"`
	Node        string `json:"node,omitempty"`
	Username    string `json:"username,omitempty"`
	TokenID     string `json:"token_id,omitempty"`
	TokenSecret string `json:"token_secret,omitempty"`
	VerifySSL   *bool  `json:"verify_ssl,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// ProxmoxConnectionResponse is the response for a Proxmox connection (excludes sensitive data).
type ProxmoxConnectionResponse struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	Name            string    `json:"name"`
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	Node            string    `json:"node"`
	Username        string    `json:"username"`
	TokenID         string    `json:"token_id,omitempty"`
	HasToken        bool      `json:"has_token"`
	VerifySSL       bool      `json:"verify_ssl"`
	Enabled         bool      `json:"enabled"`
	LastConnectedAt *string   `json:"last_connected_at,omitempty"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

// toResponse converts a ProxmoxConnection to a response (without sensitive data).
func toProxmoxResponse(conn *models.ProxmoxConnection) ProxmoxConnectionResponse {
	resp := ProxmoxConnectionResponse{
		ID:        conn.ID,
		OrgID:     conn.OrgID,
		Name:      conn.Name,
		Host:      conn.Host,
		Port:      conn.Port,
		Node:      conn.Node,
		Username:  conn.Username,
		TokenID:   conn.TokenID,
		HasToken:  conn.HasTokenAuth(),
		VerifySSL: conn.VerifySSL,
		Enabled:   conn.Enabled,
		CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if conn.LastConnectedAt != nil {
		t := conn.LastConnectedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastConnectedAt = &t
	}
	return resp
}

// ListConnections returns all Proxmox connections for the user's organization.
//
//	@Summary		List Proxmox connections
//	@Description	Returns all Proxmox connections for the current organization
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]ProxmoxConnectionResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections [get]
func (h *ProxmoxHandler) ListConnections(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	connections, err := h.store.GetProxmoxConnectionsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list Proxmox connections")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list connections"})
		return
	}

	responses := make([]ProxmoxConnectionResponse, len(connections))
	for i, conn := range connections {
		responses[i] = toProxmoxResponse(conn)
	}

	c.JSON(http.StatusOK, gin.H{"connections": responses})
}

// GetConnection returns a specific Proxmox connection by ID.
//
//	@Summary		Get Proxmox connection
//	@Description	Returns a specific Proxmox connection by ID
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Connection ID"
//	@Success		200	{object}	ProxmoxConnectionResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections/{id} [get]
func (h *ProxmoxHandler) GetConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetProxmoxConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if conn.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	c.JSON(http.StatusOK, toProxmoxResponse(conn))
}

// CreateConnection creates a new Proxmox connection.
//
//	@Summary		Create Proxmox connection
//	@Description	Creates a new Proxmox connection for the organization
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ProxmoxCreateConnectionRequest	true	"Connection details"
//	@Success		201		{object}	ProxmoxConnectionResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections [post]
func (h *ProxmoxHandler) CreateConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ProxmoxCreateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Default port
	port := req.Port
	if port == 0 {
		port = 8006
	}

	conn := models.NewProxmoxConnection(user.CurrentOrgID, req.Name, req.Host, port, req.Node, req.Username)

	// Set SSL verification
	if req.VerifySSL != nil {
		conn.VerifySSL = *req.VerifySSL
	}

	// Handle token authentication
	if req.TokenID != "" && req.TokenSecret != "" {
		encrypted, err := h.encryption.Encrypt([]byte(req.TokenSecret))
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt token secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credentials"})
			return
		}
		conn.SetTokenAuth(req.TokenID, encrypted)
	}

	if err := h.store.CreateProxmoxConnection(c.Request.Context(), conn); err != nil {
		h.logger.Error().Err(err).Msg("failed to create Proxmox connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create connection"})
		return
	}

	h.logger.Info().
		Str("connection_id", conn.ID.String()).
		Str("name", conn.Name).
		Str("host", conn.Host).
		Msg("Proxmox connection created")

	c.JSON(http.StatusCreated, toProxmoxResponse(conn))
}

// UpdateConnection updates an existing Proxmox connection.
//
//	@Summary		Update Proxmox connection
//	@Description	Updates an existing Proxmox connection
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Connection ID"
//	@Param			request	body		ProxmoxUpdateConnectionRequest	true	"Connection updates"
//	@Success		200		{object}	ProxmoxConnectionResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections/{id} [put]
func (h *ProxmoxHandler) UpdateConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetProxmoxConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if conn.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	var req ProxmoxUpdateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		conn.Name = req.Name
	}
	if req.Host != "" {
		conn.Host = req.Host
	}
	if req.Port != nil {
		conn.Port = *req.Port
	}
	if req.Node != "" {
		conn.Node = req.Node
	}
	if req.Username != "" {
		conn.Username = req.Username
	}
	if req.VerifySSL != nil {
		conn.VerifySSL = *req.VerifySSL
	}
	if req.Enabled != nil {
		conn.Enabled = *req.Enabled
	}

	// Update token if provided
	if req.TokenID != "" {
		conn.TokenID = req.TokenID
	}
	if req.TokenSecret != "" {
		encrypted, err := h.encryption.Encrypt([]byte(req.TokenSecret))
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt token secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credentials"})
			return
		}
		conn.TokenSecretEncrypted = encrypted
	}

	if err := h.store.UpdateProxmoxConnection(c.Request.Context(), conn); err != nil {
		h.logger.Error().Err(err).Msg("failed to update Proxmox connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update connection"})
		return
	}

	h.logger.Info().Str("connection_id", id.String()).Msg("Proxmox connection updated")
	c.JSON(http.StatusOK, toProxmoxResponse(conn))
}

// DeleteConnection removes a Proxmox connection.
//
//	@Summary		Delete Proxmox connection
//	@Description	Removes a Proxmox connection
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Connection ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections/{id} [delete]
func (h *ProxmoxHandler) DeleteConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetProxmoxConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if conn.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if err := h.store.DeleteProxmoxConnection(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete Proxmox connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete connection"})
		return
	}

	h.logger.Info().Str("connection_id", id.String()).Msg("Proxmox connection deleted")
	c.JSON(http.StatusOK, gin.H{"message": "connection deleted"})
}

// TestProxmoxConnectionResponse is the response for testing a Proxmox connection.
type TestProxmoxConnectionResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version,omitempty"`
	Message string `json:"message"`
}

// TestConnection tests the connection to a Proxmox server.
//
//	@Summary		Test Proxmox connection
//	@Description	Tests the connection to a Proxmox server
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Connection ID"
//	@Success		200	{object}	TestProxmoxConnectionResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections/{id}/test [post]
func (h *ProxmoxHandler) TestConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetProxmoxConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if conn.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	// Decrypt token secret
	var tokenSecret string
	if len(conn.TokenSecretEncrypted) > 0 {
		decrypted, err := h.encryption.Decrypt(conn.TokenSecretEncrypted)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to decrypt token secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt credentials"})
			return
		}
		tokenSecret = string(decrypted)
	}

	// Create client and test connection
	client := vms.NewProxmoxClientFromConnection(conn, tokenSecret, h.logger)
	version, err := client.GetVersion(c.Request.Context())
	if err != nil {
		h.logger.Warn().Err(err).Str("connection_id", id.String()).Msg("Proxmox connection test failed")
		c.JSON(http.StatusOK, TestProxmoxConnectionResponse{
			Success: false,
			Message: "Connection failed: " + err.Error(),
		})
		return
	}

	// Update last connected timestamp
	conn.MarkConnected()
	if err := h.store.UpdateProxmoxConnection(c.Request.Context(), conn); err != nil {
		h.logger.Warn().Err(err).Msg("failed to update last connected timestamp")
	}

	c.JSON(http.StatusOK, TestProxmoxConnectionResponse{
		Success: true,
		Version: version.Version,
		Message: "Connection successful",
	})
}

// ListVMsResponse is the response for listing VMs from a Proxmox connection.
type ListVMsResponse struct {
	VMs []models.ProxmoxVMInfo `json:"vms"`
}

// ListVMs lists all VMs and containers from a Proxmox connection.
//
//	@Summary		List Proxmox VMs
//	@Description	Lists all VMs and containers from a Proxmox connection
//	@Tags			Proxmox
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Connection ID"
//	@Success		200	{object}	ListVMsResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/proxmox/connections/{id}/vms [get]
func (h *ProxmoxHandler) ListVMs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetProxmoxConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	if conn.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return
	}

	// Decrypt token secret
	var tokenSecret string
	if len(conn.TokenSecretEncrypted) > 0 {
		decrypted, err := h.encryption.Decrypt(conn.TokenSecretEncrypted)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to decrypt token secret")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt credentials"})
			return
		}
		tokenSecret = string(decrypted)
	}

	// Create client and list VMs
	client := vms.NewProxmoxClientFromConnection(conn, tokenSecret, h.logger)
	proxmoxVMs, err := client.ListAll(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to list Proxmox VMs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list VMs: " + err.Error()})
		return
	}

	// Convert to model format
	vmInfos := make([]models.ProxmoxVMInfo, len(proxmoxVMs))
	for i, vm := range proxmoxVMs {
		vmInfos[i] = vm.ToModelVMInfo()
	}

	c.JSON(http.StatusOK, ListVMsResponse{VMs: vmInfos})
}
