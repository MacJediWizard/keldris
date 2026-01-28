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

// DockerStore defines the interface for Docker health persistence operations.
type DockerStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentDockerHealth(ctx context.Context, agentID uuid.UUID) (*models.AgentDockerHealth, error)
	GetAgentDockerHealthByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentDockerHealth, error)
	GetDockerHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.DockerHealthSummary, error)
	GetRecentContainerRestartEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ContainerRestartEvent, error)
}

// DockerHandler handles Docker health-related HTTP endpoints.
type DockerHandler struct {
	store  DockerStore
	logger zerolog.Logger
}

// NewDockerHandler creates a new DockerHandler.
func NewDockerHandler(store DockerStore, logger zerolog.Logger) *DockerHandler {
	return &DockerHandler{
		store:  store,
		logger: logger.With().Str("component", "docker_handler").Logger(),
	}
}

// RegisterRoutes registers Docker health routes on the given router group.
func (h *DockerHandler) RegisterRoutes(r *gin.RouterGroup) {
	docker := r.Group("/docker")
	{
		docker.GET("/summary", h.GetSummary)
		docker.GET("/widget", h.GetDashboardWidget)
		docker.GET("/agents", h.ListAgentDockerHealth)
		docker.GET("/agents/:id", h.GetAgentDockerHealth)
		docker.GET("/restarts", h.GetRecentRestarts)
	}
}

// GetSummary returns a fleet-wide Docker health summary.
//
//	@Summary		Get Docker health summary
//	@Description	Returns aggregated Docker health statistics for all agents in the organization
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.DockerHealthSummary
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/summary [get]
func (h *DockerHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	summary, err := h.store.GetDockerHealthSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get docker health summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get docker health summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetDashboardWidget returns data for the Docker health dashboard widget.
//
//	@Summary		Get Docker dashboard widget data
//	@Description	Returns Docker health data formatted for dashboard display
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.DockerDashboardWidget
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/widget [get]
func (h *DockerHandler) GetDashboardWidget(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Get summary
	summary, err := h.store.GetDockerHealthSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get docker health summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get docker health summary"})
		return
	}

	// Get recent restart events
	restarts, err := h.store.GetRecentContainerRestartEvents(c.Request.Context(), user.CurrentOrgID, 10)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get recent restart events")
		restarts = []*models.ContainerRestartEvent{}
	}

	widget := &models.DockerDashboardWidget{
		Summary:        summary,
		RecentRestarts: make([]models.ContainerRestartEvent, 0, len(restarts)),
	}

	for _, r := range restarts {
		widget.RecentRestarts = append(widget.RecentRestarts, *r)
	}

	c.JSON(http.StatusOK, widget)
}

// ListAgentDockerHealth returns Docker health for all agents in the organization.
//
//	@Summary		List agent Docker health
//	@Description	Returns Docker health information for all agents in the organization
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.AgentDockerHealth
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/agents [get]
func (h *DockerHandler) ListAgentDockerHealth(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	healthRecords, err := h.store.GetAgentDockerHealthByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agent docker health")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agent docker health"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"docker_health": healthRecords})
}

// GetAgentDockerHealth returns Docker health for a specific agent.
//
//	@Summary		Get agent Docker health
//	@Description	Returns Docker health information for a specific agent
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	models.DockerHealth
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/agents/{id} [get]
func (h *DockerHandler) GetAgentDockerHealth(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	agentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent exists and belongs to current org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	health, err := h.store.GetAgentDockerHealth(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "docker health not found for agent"})
		return
	}

	c.JSON(http.StatusOK, health.DockerHealth)
}

// GetRecentRestarts returns recent container restart events.
//
//	@Summary		Get recent container restart events
//	@Description	Returns recent container restart events across all agents
//	@Tags			Docker
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Number of events to return (default 20, max 100)"
//	@Success		200		{object}	map[string][]models.ContainerRestartEvent
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker/restarts [get]
func (h *DockerHandler) GetRecentRestarts(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	events, err := h.store.GetRecentContainerRestartEvents(c.Request.Context(), user.CurrentOrgID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get restart events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get restart events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"restarts": events})
}

// AgentDockerHealthResponse extends agent details with Docker health.
type AgentDockerHealthResponse struct {
	*models.Agent
	DockerHealth *models.DockerHealth `json:"docker_health,omitempty"`
}
