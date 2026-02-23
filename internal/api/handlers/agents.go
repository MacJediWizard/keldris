package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AgentStore defines the interface for agent persistence operations.
type AgentStore interface {
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	CreateAgent(ctx context.Context, agent *models.Agent) error
	UpdateAgent(ctx context.Context, agent *models.Agent) error
	DeleteAgent(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByAPIKeyHash(ctx context.Context, hash string) (*models.Agent, error)
	UpdateAgentAPIKeyHash(ctx context.Context, id uuid.UUID, apiKeyHash string) error
	RevokeAgentAPIKey(ctx context.Context, id uuid.UUID) error
	GetAgentStats(ctx context.Context, agentID uuid.UUID) (*models.AgentStats, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	GetAgentHealthHistory(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.AgentHealthHistory, error)
	GetFleetHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.FleetHealthSummary, error)
	SetAgentDebugMode(ctx context.Context, id uuid.UUID, enabled bool, expiresAt *time.Time, enabledBy *uuid.UUID) error
	GetAgentLogs(ctx context.Context, agentID uuid.UUID, filter *models.AgentLogFilter) ([]*models.AgentLog, int, error)
	GetAgentDockerHealth(ctx context.Context, agentID uuid.UUID) (*models.AgentDockerHealth, error)
}

// AgentsHandler handles agent-related HTTP endpoints.
type AgentsHandler struct {
	store  AgentStore
	logger zerolog.Logger
}

// NewAgentsHandler creates a new AgentsHandler.
func NewAgentsHandler(store AgentStore, logger zerolog.Logger) *AgentsHandler {
	return &AgentsHandler{
		store:  store,
		logger: logger.With().Str("component", "agents_handler").Logger(),
	}
}

// RegisterRoutes registers agent routes on the given router group.
// Optional createMiddleware is applied before the Create handler.
func (h *AgentsHandler) RegisterRoutes(r *gin.RouterGroup, createMiddleware ...gin.HandlerFunc) {
	agents := r.Group("/agents")
	{
		agents.GET("", h.List)
		createChain := append(createMiddleware, h.Create)
		agents.POST("", createChain...)
		agents.GET("/fleet-health", h.FleetHealth)
		agents.GET("/:id", h.Get)
		agents.DELETE("/:id", h.Delete)
		agents.POST("/:id/heartbeat", h.Heartbeat)
		agents.POST("/:id/apikey/rotate", h.RotateAPIKey)
		agents.DELETE("/:id/apikey", h.RevokeAPIKey)
		agents.GET("/:id/stats", h.Stats)
		agents.GET("/:id/backups", h.Backups)
		agents.GET("/:id/schedules", h.Schedules)
		agents.GET("/:id/health-history", h.HealthHistory)
		agents.POST("/:id/debug", h.SetDebugMode)
		agents.GET("/:id/logs", h.Logs)
		agents.GET("/:id/docker-health", h.DockerHealth)
	}
}

// CreateAgentRequest is the request body for creating an agent.
type CreateAgentRequest struct {
	Hostname string `json:"hostname" binding:"required,min=1,max=255" example:"backup-server-01"`
}

// CreateAgentResponse is the response for agent creation.
type CreateAgentResponse struct {
	ID       uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Hostname string    `json:"hostname" example:"backup-server-01"`
	APIKey   string    `json:"api_key" example:"kld_abc123..."` // Only returned once at creation
}

// List returns all agents for the authenticated user's organization.
//
//	@Summary		List agents
//	@Description	Returns all agents registered in the current organization
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.Agent
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents [get]
func (h *AgentsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Use current org from session
	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// Get returns a specific agent by ID.
//
//	@Summary		Get agent
//	@Description	Returns a specific agent by ID
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	models.Agent
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id} [get]
// GET /api/v1/agents/:id
func (h *AgentsHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get agent")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// Create creates a new agent and returns an API key.
//
//	@Summary		Create agent
//	@Description	Registers a new backup agent and returns a one-time API key. Save this key securely as it cannot be retrieved again.
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateAgentRequest	true	"Agent details"
//	@Success		201		{object}	CreateAgentResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents [post]
// POST /api/v1/agents
func (h *AgentsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}

	// Hash the API key for storage
	apiKeyHash := hashAPIKey(apiKey)

	agent := models.NewAgent(user.CurrentOrgID, req.Hostname, apiKeyHash)

	if err := h.store.CreateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error().Err(err).Str("hostname", req.Hostname).Msg("failed to create agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("hostname", req.Hostname).
		Str("org_id", user.CurrentOrgID.String()).
		Msg("agent created")

	c.JSON(http.StatusCreated, CreateAgentResponse{
		ID:       agent.ID,
		Hostname: agent.Hostname,
		APIKey:   apiKey,
	})
}

// Delete removes an agent.
//
//	@Summary		Delete agent
//	@Description	Removes an agent from the organization
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id} [delete]
// DELETE /api/v1/agents/:id
func (h *AgentsHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get agent to verify ownership
	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if err := h.store.DeleteAgent(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to delete agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent"})
		return
	}

	h.logger.Info().Str("agent_id", id.String()).Msg("agent deleted")
	c.JSON(http.StatusOK, gin.H{"message": "agent deleted"})
}

// HeartbeatRequest is the request body for agent heartbeat.
type HeartbeatRequest struct {
	OSInfo        *models.OSInfo        `json:"os_info,omitempty"`
	NetworkMounts []models.NetworkMount `json:"network_mounts,omitempty"`
}

// Heartbeat updates an agent's last seen timestamp.
//
//	@Summary		Agent heartbeat
//	@Description	Updates an agent's last seen timestamp and optionally updates OS information. Returns debug configuration if debug mode is enabled.
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Agent ID"
//	@Param			request	body		HeartbeatRequest	false	"Heartbeat data"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/heartbeat [post]
func (h *AgentsHandler) Heartbeat(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Heartbeat can be empty - just update last seen
	}

	// Update agent
	agent.MarkSeen()
	if req.OSInfo != nil {
		agent.OSInfo = req.OSInfo
	}

	if err := h.store.UpdateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to update agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// RotateAPIKeyResponse is the response for API key rotation.
type RotateAPIKeyResponse struct {
	ID       uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Hostname string    `json:"hostname" example:"backup-server-01"`
	APIKey   string    `json:"api_key" example:"kld_abc123..."` // Only returned once at rotation
}

// RotateAPIKey generates a new API key for an agent, invalidating the old one.
//
//	@Summary		Rotate API key
//	@Description	Generates a new API key for an agent, invalidating the previous key. Save the new key securely.
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	RotateAPIKeyResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/apikey/rotate [post]
func (h *AgentsHandler) RotateAPIKey(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get agent to verify ownership
	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Generate new API key
	apiKey, err := generateAPIKey()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate API key"})
		return
	}

	// Hash and store new API key
	apiKeyHash := hashAPIKey(apiKey)
	if err := h.store.UpdateAgentAPIKeyHash(c.Request.Context(), id, apiKeyHash); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to update API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate API key"})
		return
	}

	h.logger.Info().
		Str("agent_id", id.String()).
		Str("hostname", agent.Hostname).
		Msg("API key rotated")

	c.JSON(http.StatusOK, RotateAPIKeyResponse{
		ID:       agent.ID,
		Hostname: agent.Hostname,
		APIKey:   apiKey,
	})
}

// RevokeAPIKey revokes an agent's API key, disabling its ability to authenticate.
//
//	@Summary		Revoke API key
//	@Description	Revokes an agent's API key, preventing it from authenticating until a new key is issued
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/apikey [delete]
func (h *AgentsHandler) RevokeAPIKey(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get agent to verify ownership
	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if err := h.store.RevokeAgentAPIKey(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to revoke API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke API key"})
		return
	}

	h.logger.Info().
		Str("agent_id", id.String()).
		Str("hostname", agent.Hostname).
		Msg("API key revoked")

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}
// generateAPIKey generates a secure random API key.
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "kld_" + hex.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA-256 hash of an API key for storage.
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// Stats returns statistics for a specific agent.
// GET /api/v1/agents/:id/stats
func (h *AgentsHandler) Stats(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	stats, err := h.store.GetAgentStats(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get agent stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent stats"})
		return
	}

	c.JSON(http.StatusOK, models.AgentStatsResponse{
		Agent: agent,
		Stats: stats,
	})
}

// Backups returns backup history for a specific agent.
// GET /api/v1/agents/:id/backups
func (h *AgentsHandler) Backups(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	backups, err := h.store.GetBackupsByAgentID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get agent backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent backups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// Schedules returns schedules for a specific agent.
// GET /api/v1/agents/:id/schedules
func (h *AgentsHandler) Schedules(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get agent schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// HealthHistory returns health history for a specific agent.
//
//	@Summary		Get agent health history
//	@Description	Returns health metrics history for an agent
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Agent ID"
//	@Param			limit	query		int		false	"Number of records to return (default 100)"
//	@Success		200		{object}	map[string][]models.AgentHealthHistory
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/health-history [get]
func (h *AgentsHandler) HealthHistory(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Parse limit parameter
	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	history, err := h.store.GetAgentHealthHistory(c.Request.Context(), id, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get health history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get health history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// FleetHealth returns aggregated health statistics for all agents.
//
//	@Summary		Get fleet health summary
//	@Description	Returns aggregated health statistics for all agents in the organization
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.FleetHealthSummary
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/fleet-health [get]
func (h *AgentsHandler) FleetHealth(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	summary, err := h.store.GetFleetHealthSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get fleet health")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get fleet health"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// parseIntParam parses a string to int.
func parseIntParam(s string) (int, error) {
	var i int
	_, err := getParamInt(s, &i)
	return i, err
}

// getParamInt is a helper to parse int from string.
func getParamInt(s string, result *int) (bool, error) {
	if s == "" {
		return false, nil
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false, errors.New("invalid integer")
		}
		n = n*10 + int(ch-'0')
	}
	*result = n
	return true, nil
}

// SetDebugMode enables or disables debug mode on an agent.
//
//	@Summary		Set agent debug mode
//	@Description	Enables or disables verbose/debug logging on an agent. When enabled, the agent will produce detailed logs including restic output and file operations. Debug mode auto-disables after the specified duration (default 4 hours).
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Agent ID"
//	@Param			request	body		models.SetDebugModeRequest	true	"Debug mode settings"
//	@Success		200		{object}	models.SetDebugModeResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/debug [post]
func (h *AgentsHandler) SetDebugMode(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	var req models.SetDebugModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	var expiresAt *time.Time
	var message string

	if req.Enabled {
		// Default to 4 hours if not specified
		durationHours := req.DurationHours
		if durationHours <= 0 {
			durationHours = 4
		}
		expiry := time.Now().Add(time.Duration(durationHours) * time.Hour)
		expiresAt = &expiry
		message = fmt.Sprintf("Debug mode enabled for %d hours", durationHours)
	} else {
		message = "Debug mode disabled"
	}

	if err := h.store.SetAgentDebugMode(c.Request.Context(), id, req.Enabled, expiresAt, &user.ID); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Bool("enabled", req.Enabled).Msg("failed to set debug mode")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set debug mode"})
		return
	}

	h.logger.Info().
		Str("agent_id", id.String()).
		Str("hostname", agent.Hostname).
		Bool("enabled", req.Enabled).
		Str("user_id", user.ID.String()).
		Msg("debug mode changed")

	c.JSON(http.StatusOK, models.SetDebugModeResponse{
		DebugMode:          req.Enabled,
		DebugModeExpiresAt: expiresAt,
		Message:            message,
	})
}

// DockerHealth returns Docker health for a specific agent.
//
//	@Summary		Get agent Docker health
//	@Description	Returns Docker container and volume health information for an agent
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	models.DockerHealth
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/docker-health [get]
func (h *AgentsHandler) DockerHealth(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	dockerHealth, err := h.store.GetAgentDockerHealth(c.Request.Context(), id)
	if err != nil {
		// No Docker health data - return empty but indicate Docker is not available
		c.JSON(http.StatusOK, gin.H{
			"available": false,
			"message":   "Docker health data not available for this agent",
		})
		return
	}

	c.JSON(http.StatusOK, dockerHealth.DockerHealth)
}

// Logs returns logs for a specific agent.
//
//	@Summary		Get agent logs
//	@Description	Returns logs for an agent with optional filtering
//	@Tags			Agents
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string	true	"Agent ID"
//	@Param			level		query		string	false	"Log level filter (debug, info, warn, error)"
//	@Param			component	query		string	false	"Component filter"
//	@Param			search		query		string	false	"Search text in log messages"
//	@Param			limit		query		int		false	"Number of records to return (default 100, max 1000)"
//	@Param			offset		query		int		false	"Pagination offset"
//	@Success		200			{object}	models.AgentLogsResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/agents/{id}/logs [get]
func (h *AgentsHandler) Logs(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Build filter from query params
	filter := &models.AgentLogFilter{}

	if level := c.Query("level"); level != "" {
		filter.Level = models.LogLevel(level)
	}
	if component := c.Query("component"); component != "" {
		filter.Component = component
	}
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := parseIntParam(offsetParam); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	logs, totalCount, err := h.store.GetAgentLogs(c.Request.Context(), id, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to get agent logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent logs"})
		return
	}

	hasMore := filter.Offset+len(logs) < totalCount

	c.JSON(http.StatusOK, models.AgentLogsResponse{
		Logs:       logs,
		TotalCount: totalCount,
		HasMore:    hasMore,
	})
}
