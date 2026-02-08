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

// AgentCommandsStore defines the interface for agent command persistence operations.
type AgentCommandsStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	CreateAgentCommand(ctx context.Context, cmd *models.AgentCommand) error
	GetAgentCommandByID(ctx context.Context, id uuid.UUID) (*models.AgentCommand, error)
	GetCommandsByAgentID(ctx context.Context, agentID uuid.UUID, limit int) ([]*models.AgentCommand, error)
	CancelAgentCommand(ctx context.Context, id uuid.UUID) error
}

// AgentCommandsHandler handles agent command management endpoints.
type AgentCommandsHandler struct {
	store  AgentCommandsStore
	logger zerolog.Logger
}

// NewAgentCommandsHandler creates a new AgentCommandsHandler.
func NewAgentCommandsHandler(store AgentCommandsStore, logger zerolog.Logger) *AgentCommandsHandler {
	return &AgentCommandsHandler{
		store:  store,
		logger: logger.With().Str("component", "agent_commands_handler").Logger(),
	}
}

// RegisterRoutes registers agent command management routes.
func (h *AgentCommandsHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Commands are nested under agents
	r.GET("/agents/:id/commands", h.ListCommands)
	r.POST("/agents/:id/commands", h.CreateCommand)
	r.GET("/agents/:id/commands/:commandId", h.GetCommand)
	r.DELETE("/agents/:id/commands/:commandId", h.CancelCommand)
}

// CreateCommandRequest is the request body for creating a command.
type CreateCommandRequest struct {
	Type    string                 `json:"type" binding:"required,oneof=backup_now update restart diagnostics"`
	Payload *models.CommandPayload `json:"payload,omitempty"`
}

// ListCommandsResponse is the response for listing commands.
type ListCommandsResponse struct {
	Commands []*models.AgentCommand `json:"commands"`
}

// ListCommands returns commands for an agent.
// GET /api/v1/agents/:id/commands
func (h *AgentCommandsHandler) ListCommands(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent exists and belongs to the org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Get limit from query params, default to 50
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	commands, err := h.store.GetCommandsByAgentID(c.Request.Context(), agentID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list commands")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list commands"})
		return
	}

	c.JSON(http.StatusOK, ListCommandsResponse{Commands: commands})
}

// CreateCommand creates a new command for an agent.
// POST /api/v1/agents/:id/commands
func (h *AgentCommandsHandler) CreateCommand(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	var req CreateCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify agent exists and belongs to the org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Create the command
	cmd := models.NewAgentCommand(
		agentID,
		user.CurrentOrgID,
		models.CommandType(req.Type),
		req.Payload,
		&user.ID,
	)

	if err := h.store.CreateAgentCommand(c.Request.Context(), cmd); err != nil {
		h.logger.Error().Err(err).
			Str("agent_id", agentID.String()).
			Str("type", req.Type).
			Msg("failed to create command")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create command"})
		return
	}

	h.logger.Info().
		Str("agent_id", agentID.String()).
		Str("command_id", cmd.ID.String()).
		Str("type", req.Type).
		Str("created_by", user.ID.String()).
		Msg("command created")

	c.JSON(http.StatusCreated, cmd)
}

// GetCommand returns a specific command.
// GET /api/v1/agents/:id/commands/:commandId
func (h *AgentCommandsHandler) GetCommand(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	commandID, err := uuid.Parse(c.Param("commandId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command ID"})
		return
	}

	// Get the command
	cmd, err := h.store.GetAgentCommandByID(c.Request.Context(), commandID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Verify command belongs to this agent and org
	if cmd.AgentID != agentID || cmd.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	c.JSON(http.StatusOK, cmd)
}

// CancelCommand cancels a pending or running command.
// DELETE /api/v1/agents/:id/commands/:commandId
func (h *AgentCommandsHandler) CancelCommand(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	commandID, err := uuid.Parse(c.Param("commandId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command ID"})
		return
	}

	// Get the command to verify access
	cmd, err := h.store.GetAgentCommandByID(c.Request.Context(), commandID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Verify command belongs to this agent and org
	if cmd.AgentID != agentID || cmd.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Cancel the command
	if err := h.store.CancelAgentCommand(c.Request.Context(), commandID); err != nil {
		h.logger.Error().Err(err).
			Str("command_id", commandID.String()).
			Msg("failed to cancel command")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("agent_id", agentID.String()).
		Str("command_id", commandID.String()).
		Str("canceled_by", user.ID.String()).
		Msg("command canceled")

	c.JSON(http.StatusOK, gin.H{"message": "command canceled"})
}
