package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AgentAPIStore defines the interface for agent API persistence operations.
type AgentAPIStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	UpdateAgent(ctx context.Context, agent *models.Agent) error
}

// AgentAPIHandler handles agent-facing API endpoints (authenticated via API key).
type AgentAPIHandler struct {
	store  AgentAPIStore
	logger zerolog.Logger
}

// NewAgentAPIHandler creates a new AgentAPIHandler.
func NewAgentAPIHandler(store AgentAPIStore, logger zerolog.Logger) *AgentAPIHandler {
	return &AgentAPIHandler{
		store:  store,
		logger: logger.With().Str("component", "agent_api_handler").Logger(),
	}
}

// RegisterRoutes registers agent API routes on the given router group.
// This group should have APIKeyMiddleware applied.
func (h *AgentAPIHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/health", h.ReportHealth)
}

// AgentHealthReport is the request body for agent health reporting.
type AgentHealthReport struct {
	Status  string         `json:"status" binding:"required,oneof=healthy unhealthy degraded"`
	OSInfo  *models.OSInfo `json:"os_info,omitempty"`
	Metrics *AgentMetrics  `json:"metrics,omitempty"`
}

// AgentMetrics contains optional metrics from the agent.
type AgentMetrics struct {
	CPUUsage    float64 `json:"cpu_usage,omitempty"`
	MemoryUsage float64 `json:"memory_usage,omitempty"`
	DiskUsage   float64 `json:"disk_usage,omitempty"`
	Uptime      int64   `json:"uptime_seconds,omitempty"`
}

// AgentHealthResponse is the response for agent health reporting.
type AgentHealthResponse struct {
	Acknowledged bool      `json:"acknowledged"`
	ServerTime   time.Time `json:"server_time"`
	AgentID      string    `json:"agent_id"`
}

// ReportHealth handles agent health reports.
// POST /api/v1/agent/health
func (h *AgentAPIHandler) ReportHealth(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	var req AgentHealthReport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Update agent's last seen and status
	agent.MarkSeen()
	if req.OSInfo != nil {
		agent.OSInfo = req.OSInfo
	}

	// Map agent-reported status to internal status
	switch req.Status {
	case "healthy":
		agent.Status = models.AgentStatusActive
	case "unhealthy", "degraded":
		// Keep as active since agent is still responding, but could add a separate health field
		agent.Status = models.AgentStatusActive
	}

	if err := h.store.UpdateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to update agent health")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent health"})
		return
	}

	h.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Str("status", req.Status).
		Msg("agent health report received")

	c.JSON(http.StatusOK, AgentHealthResponse{
		Acknowledged: true,
		ServerTime:   time.Now().UTC(),
		AgentID:      agent.ID.String(),
	})
}
