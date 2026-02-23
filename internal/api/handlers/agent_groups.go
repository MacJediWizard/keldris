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

// AgentGroupStore defines the interface for agent group persistence operations.
type AgentGroupStore interface {
	GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error)
	GetAgentGroupByID(ctx context.Context, id uuid.UUID) (*models.AgentGroup, error)
	CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error
	UpdateAgentGroup(ctx context.Context, group *models.AgentGroup) error
	DeleteAgentGroup(ctx context.Context, id uuid.UUID) error
	GetAgentGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*models.Agent, error)
	AddAgentToGroup(ctx context.Context, agentID, groupID uuid.UUID) error
	RemoveAgentFromGroup(ctx context.Context, agentID, groupID uuid.UUID) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsWithGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentWithGroups, error)
}

// AgentGroupsHandler handles agent group-related HTTP endpoints.
type AgentGroupsHandler struct {
	store  AgentGroupStore
	logger zerolog.Logger
}

// NewAgentGroupsHandler creates a new AgentGroupsHandler.
func NewAgentGroupsHandler(store AgentGroupStore, logger zerolog.Logger) *AgentGroupsHandler {
	return &AgentGroupsHandler{
		store:  store,
		logger: logger.With().Str("component", "agent_groups_handler").Logger(),
	}
}

// RegisterRoutes registers agent group routes on the given router group.
func (h *AgentGroupsHandler) RegisterRoutes(r *gin.RouterGroup) {
	groups := r.Group("/agent-groups")
	{
		groups.GET("", h.List)
		groups.POST("", h.Create)
		groups.GET("/:id", h.Get)
		groups.PUT("/:id", h.Update)
		groups.DELETE("/:id", h.Delete)
		groups.GET("/:id/agents", h.ListMembers)
		groups.POST("/:id/agents", h.AddAgent)
		groups.DELETE("/:id/agents/:agent_id", h.RemoveAgent)
		groups.DELETE("/:id/agents/:agentId", h.RemoveAgent)
	}

	// Also add endpoint for listing agents with their groups
	r.GET("/agents/with-groups", h.ListAgentsWithGroups)
}

// CreateAgentGroupRequest is the request body for creating an agent group.
type CreateAgentGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty" binding:"omitempty,hexcolor"`
}

// UpdateAgentGroupRequest is the request body for updating an agent group.
type UpdateAgentGroupRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty" binding:"omitempty,hexcolor"`
}

// AddAgentRequest is the request body for adding an agent to a group.
type AddAgentRequest struct {
	AgentID string `json:"agent_id" binding:"required,uuid"`
}

// List returns all agent groups for the authenticated user's organization.
// GET /api/v1/agent-groups
func (h *AgentGroupsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	groups, err := h.store.GetAgentGroupsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agent groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agent groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// Get returns a specific agent group by ID.
// GET /api/v1/agent-groups/:id
func (h *AgentGroupsHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	group, err := h.store.GetAgentGroupByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("group_id", id.String()).Msg("failed to get agent group")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	// Verify group belongs to current org
	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// Create creates a new agent group.
// POST /api/v1/agent-groups
func (h *AgentGroupsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateAgentGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	group := models.NewAgentGroup(user.CurrentOrgID, req.Name, req.Description, req.Color)

	if err := h.store.CreateAgentGroup(c.Request.Context(), group); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create agent group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent group"})
		return
	}

	h.logger.Info().
		Str("group_id", group.ID.String()).
		Str("name", req.Name).
		Str("org_id", user.CurrentOrgID.String()).
		Msg("agent group created")

	c.JSON(http.StatusCreated, group)
}

// Update updates an existing agent group.
// PUT /api/v1/agent-groups/:id
func (h *AgentGroupsHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req UpdateAgentGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get existing group
	group, err := h.store.GetAgentGroupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	// Verify group belongs to current org
	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.Color != nil {
		group.Color = *req.Color
	}

	if err := h.store.UpdateAgentGroup(c.Request.Context(), group); err != nil {
		h.logger.Error().Err(err).Str("group_id", id.String()).Msg("failed to update agent group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent group"})
		return
	}

	h.logger.Info().Str("group_id", id.String()).Msg("agent group updated")
	c.JSON(http.StatusOK, group)
}

// Delete removes an agent group.
// DELETE /api/v1/agent-groups/:id
func (h *AgentGroupsHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	// Get existing group to verify ownership
	group, err := h.store.GetAgentGroupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	// Verify group belongs to current org
	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	if err := h.store.DeleteAgentGroup(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("group_id", id.String()).Msg("failed to delete agent group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent group"})
		return
	}

	h.logger.Info().Str("group_id", id.String()).Msg("agent group deleted")
	c.JSON(http.StatusOK, gin.H{"message": "agent group deleted"})
}

// ListMembers returns all agents in a specific group.
// GET /api/v1/agent-groups/:id/agents
func (h *AgentGroupsHandler) ListMembers(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	// Verify group exists and belongs to org
	group, err := h.store.GetAgentGroupByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	agents, err := h.store.GetAgentGroupMembers(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("group_id", id.String()).Msg("failed to list group members")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list group members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// AddAgent adds an agent to a group.
// POST /api/v1/agent-groups/:id/agents
func (h *AgentGroupsHandler) AddAgent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	groupID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	var req AddAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	agentID, _ := uuid.Parse(req.AgentID) // Already validated by binding

	// Verify group exists and belongs to org
	group, err := h.store.GetAgentGroupByID(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	// Verify agent exists and belongs to org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if err := h.store.AddAgentToGroup(c.Request.Context(), agentID, groupID); err != nil {
		h.logger.Error().Err(err).
			Str("group_id", groupID.String()).
			Str("agent_id", agentID.String()).
			Msg("failed to add agent to group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add agent to group"})
		return
	}

	h.logger.Info().
		Str("group_id", groupID.String()).
		Str("agent_id", agentID.String()).
		Msg("agent added to group")

	c.JSON(http.StatusOK, gin.H{"message": "agent added to group"})
}

// RemoveAgent removes an agent from a group.
// DELETE /api/v1/agent-groups/:id/agents/:agentId
func (h *AgentGroupsHandler) RemoveAgent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
		return
	}

	agentIDParam := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify group exists and belongs to org
	group, err := h.store.GetAgentGroupByID(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	if group.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})
		return
	}

	if err := h.store.RemoveAgentFromGroup(c.Request.Context(), agentID, groupID); err != nil {
		h.logger.Error().Err(err).
			Str("group_id", groupID.String()).
			Str("agent_id", agentID.String()).
			Msg("failed to remove agent from group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove agent from group"})
		return
	}

	h.logger.Info().
		Str("group_id", groupID.String()).
		Str("agent_id", agentID.String()).
		Msg("agent removed from group")

	c.JSON(http.StatusOK, gin.H{"message": "agent removed from group"})
}

// ListAgentsWithGroups returns all agents for the org with their group memberships.
// GET /api/v1/agents/with-groups
func (h *AgentGroupsHandler) ListAgentsWithGroups(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agents, err := h.store.GetAgentsWithGroupsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents with groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}
