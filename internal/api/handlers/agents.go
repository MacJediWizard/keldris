package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"

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
func (h *AgentsHandler) RegisterRoutes(r *gin.RouterGroup) {
	agents := r.Group("/agents")
	{
		agents.GET("", h.List)
		agents.POST("", h.Create)
		agents.GET("/:id", h.Get)
		agents.DELETE("/:id", h.Delete)
		agents.POST("/:id/heartbeat", h.Heartbeat)
		agents.POST("/:id/apikey/rotate", h.RotateAPIKey)
		agents.DELETE("/:id/apikey", h.RevokeAPIKey)
		agents.GET("/:id/stats", h.Stats)
		agents.GET("/:id/backups", h.Backups)
		agents.GET("/:id/schedules", h.Schedules)
	}
}

// CreateAgentRequest is the request body for creating an agent.
type CreateAgentRequest struct {
	Hostname string `json:"hostname" binding:"required,min=1,max=255"`
}

// CreateAgentResponse is the response for agent creation.
type CreateAgentResponse struct {
	ID       uuid.UUID `json:"id"`
	Hostname string    `json:"hostname"`
	APIKey   string    `json:"api_key"` // Only returned once at creation
}

// List returns all agents for the authenticated user's organization.
// GET /api/v1/agents
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
	OSInfo *models.OSInfo `json:"os_info,omitempty"`
}

// Heartbeat updates an agent's last seen timestamp.
// POST /api/v1/agents/:id/heartbeat
func (h *AgentsHandler) Heartbeat(c *gin.Context) {
	// This endpoint can be called with either session auth or API key auth
	// For now, we support session auth. API key auth will be added later.
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

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, errors.New("EOF")) {
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
	ID       uuid.UUID `json:"id"`
	Hostname string    `json:"hostname"`
	APIKey   string    `json:"api_key"` // Only returned once at rotation
}

// RotateAPIKey generates a new API key for an agent, invalidating the old one.
// POST /api/v1/agents/:id/apikey/rotate
func (h *AgentsHandler) RotateAPIKey(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
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

	// Verify user has access to this agent's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
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
// DELETE /api/v1/agents/:id/apikey
func (h *AgentsHandler) RevokeAPIKey(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
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

	// Verify user has access to this agent's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
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
