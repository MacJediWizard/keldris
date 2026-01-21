package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// AgentAPIStore defines the interface for agent API persistence operations.
type AgentAPIStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	UpdateAgent(ctx context.Context, agent *models.Agent) error
	CreateAgentHealthHistory(ctx context.Context, history *models.AgentHealthHistory) error
	CreateAlert(ctx context.Context, alert *models.Alert) error
	GetAlertByResourceAndType(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (*models.Alert, error)
	ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error
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
	CPUUsage        float64 `json:"cpu_usage,omitempty"`
	MemoryUsage     float64 `json:"memory_usage,omitempty"`
	DiskUsage       float64 `json:"disk_usage,omitempty"`
	DiskFreeBytes   int64   `json:"disk_free_bytes,omitempty"`
	DiskTotalBytes  int64   `json:"disk_total_bytes,omitempty"`
	NetworkUp       bool    `json:"network_up"`
	Uptime          int64   `json:"uptime_seconds,omitempty"`
	ResticVersion   string  `json:"restic_version,omitempty"`
	ResticAvailable bool    `json:"restic_available,omitempty"`
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

	// Convert metrics to model
	var healthMetrics *models.HealthMetrics
	if req.Metrics != nil {
		healthMetrics = &models.HealthMetrics{
			CPUUsage:        req.Metrics.CPUUsage,
			MemoryUsage:     req.Metrics.MemoryUsage,
			DiskUsage:       req.Metrics.DiskUsage,
			DiskFreeBytes:   req.Metrics.DiskFreeBytes,
			DiskTotalBytes:  req.Metrics.DiskTotalBytes,
			NetworkUp:       req.Metrics.NetworkUp,
			UptimeSeconds:   req.Metrics.Uptime,
			ResticVersion:   req.Metrics.ResticVersion,
			ResticAvailable: req.Metrics.ResticAvailable,
		}
		agent.HealthMetrics = healthMetrics
	}

	// Evaluate health status based on metrics
	issues := h.evaluateHealth(healthMetrics, req.Status)
	healthStatus := h.determineHealthStatus(req.Status, issues)

	// Store issues in health metrics for API response
	if healthMetrics != nil && len(issues) > 0 {
		healthMetrics.Issues = issues
	}

	// Track if status changed for alerting
	previousStatus := agent.HealthStatus

	// Update health fields
	agent.HealthStatus = healthStatus
	now := time.Now()
	agent.HealthCheckedAt = &now

	// Agent is still active since it's responding
	agent.Status = models.AgentStatusActive

	if err := h.store.UpdateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to update agent health")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent health"})
		return
	}

	// Record health history
	history := models.NewAgentHealthHistory(agent.ID, agent.OrgID, healthStatus, healthMetrics, issues)
	if err := h.store.CreateAgentHealthHistory(c.Request.Context(), history); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to record health history")
		// Don't fail the request, just log the error
	}

	// Handle health status changes for alerting
	if previousStatus != healthStatus {
		// Log status change
		if previousStatus != "" {
			h.logger.Info().
				Str("agent_id", agent.ID.String()).
				Str("hostname", agent.Hostname).
				Str("previous_status", string(previousStatus)).
				Str("new_status", string(healthStatus)).
				Msg("agent health status changed")
		}

		// Trigger alerts on health status change
		h.handleHealthStatusChange(c.Request.Context(), agent, previousStatus, healthStatus, issues)
	}

	h.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Str("health_status", string(healthStatus)).
		Msg("agent health report received")

	c.JSON(http.StatusOK, AgentHealthResponse{
		Acknowledged: true,
		ServerTime:   time.Now().UTC(),
		AgentID:      agent.ID.String(),
	})
}

// evaluateHealth evaluates health issues based on metrics.
func (h *AgentAPIHandler) evaluateHealth(metrics *models.HealthMetrics, reportedStatus string) []models.HealthIssue {
	var issues []models.HealthIssue

	if metrics == nil {
		return issues
	}

	// Check disk usage (critical if >= 90%, warning if >= 80%)
	if metrics.DiskUsage >= 90 {
		issues = append(issues, models.HealthIssue{
			Component: "disk",
			Severity:  models.HealthStatusCritical,
			Message:   "Disk space critically low",
			Value:     metrics.DiskUsage,
			Threshold: 90,
		})
	} else if metrics.DiskUsage >= 80 {
		issues = append(issues, models.HealthIssue{
			Component: "disk",
			Severity:  models.HealthStatusWarning,
			Message:   "Disk space running low",
			Value:     metrics.DiskUsage,
			Threshold: 80,
		})
	}

	// Check memory usage (critical if >= 95%, warning if >= 85%)
	if metrics.MemoryUsage >= 95 {
		issues = append(issues, models.HealthIssue{
			Component: "memory",
			Severity:  models.HealthStatusCritical,
			Message:   "Memory usage critically high",
			Value:     metrics.MemoryUsage,
			Threshold: 95,
		})
	} else if metrics.MemoryUsage >= 85 {
		issues = append(issues, models.HealthIssue{
			Component: "memory",
			Severity:  models.HealthStatusWarning,
			Message:   "Memory usage high",
			Value:     metrics.MemoryUsage,
			Threshold: 85,
		})
	}

	// Check CPU usage (critical if >= 95%, warning if >= 80%)
	if metrics.CPUUsage >= 95 {
		issues = append(issues, models.HealthIssue{
			Component: "cpu",
			Severity:  models.HealthStatusCritical,
			Message:   "CPU usage critically high",
			Value:     metrics.CPUUsage,
			Threshold: 95,
		})
	} else if metrics.CPUUsage >= 80 {
		issues = append(issues, models.HealthIssue{
			Component: "cpu",
			Severity:  models.HealthStatusWarning,
			Message:   "CPU usage high",
			Value:     metrics.CPUUsage,
			Threshold: 80,
		})
	}

	// Check network connectivity
	if !metrics.NetworkUp {
		issues = append(issues, models.HealthIssue{
			Component: "network",
			Severity:  models.HealthStatusWarning,
			Message:   "Network connectivity issues detected",
		})
	}

	// Check restic availability
	if !metrics.ResticAvailable {
		issues = append(issues, models.HealthIssue{
			Component: "restic",
			Severity:  models.HealthStatusWarning,
			Message:   "Restic binary not available",
		})
	}

	return issues
}

// determineHealthStatus determines overall health status.
func (h *AgentAPIHandler) determineHealthStatus(reportedStatus string, issues []models.HealthIssue) models.HealthStatus {
	// If agent reports unhealthy, use critical
	if reportedStatus == "unhealthy" {
		return models.HealthStatusCritical
	}

	// Check for critical issues
	for _, issue := range issues {
		if issue.Severity == models.HealthStatusCritical {
			return models.HealthStatusCritical
		}
	}

	// Check for warning issues
	for _, issue := range issues {
		if issue.Severity == models.HealthStatusWarning {
			return models.HealthStatusWarning
		}
	}

	// If agent reports degraded but no issues found
	if reportedStatus == "degraded" {
		return models.HealthStatusWarning
	}

	return models.HealthStatusHealthy
}

// handleHealthStatusChange creates or resolves alerts based on health status changes.
func (h *AgentAPIHandler) handleHealthStatusChange(ctx context.Context, agent *models.Agent, previousStatus, newStatus models.HealthStatus, issues []models.HealthIssue) {
	// If status improved to healthy, resolve any existing health alerts
	if newStatus == models.HealthStatusHealthy {
		if err := h.store.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, agent.ID); err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to resolve health alerts")
		}
		return
	}

	// Determine alert type and severity based on new status
	var alertType models.AlertType
	var severity models.AlertSeverity
	var title string

	switch newStatus {
	case models.HealthStatusCritical:
		alertType = models.AlertTypeAgentHealthCritical
		severity = models.AlertSeverityCritical
		title = fmt.Sprintf("Agent %s is in critical state", agent.Hostname)
	case models.HealthStatusWarning:
		alertType = models.AlertTypeAgentHealthWarning
		severity = models.AlertSeverityWarning
		title = fmt.Sprintf("Agent %s health warning", agent.Hostname)
	default:
		return // Unknown status, no alert needed
	}

	// Check if there's already an active alert for this agent and type
	existingAlert, err := h.store.GetAlertByResourceAndType(ctx, agent.OrgID, models.ResourceTypeAgent, agent.ID, alertType)
	if err != nil && err != pgx.ErrNoRows {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check existing health alert")
		return
	}

	// If alert already exists, don't create a duplicate
	if existingAlert != nil {
		return
	}

	// Build alert message from issues
	var issueMessages []string
	for _, issue := range issues {
		issueMessages = append(issueMessages, issue.Message)
	}
	message := "Health issues detected"
	if len(issueMessages) > 0 {
		message = strings.Join(issueMessages, "; ")
	}

	// Create new alert
	alert := models.NewAlert(agent.OrgID, alertType, severity, title, message)
	alert.SetResource(models.ResourceTypeAgent, agent.ID)

	// Add metadata with health metrics
	alert.Metadata = map[string]any{
		"hostname":        agent.Hostname,
		"previous_status": string(previousStatus),
		"new_status":      string(newStatus),
	}
	if agent.HealthMetrics != nil {
		alert.Metadata["cpu_usage"] = agent.HealthMetrics.CPUUsage
		alert.Metadata["memory_usage"] = agent.HealthMetrics.MemoryUsage
		alert.Metadata["disk_usage"] = agent.HealthMetrics.DiskUsage
	}

	if err := h.store.CreateAlert(ctx, alert); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to create health alert")
		return
	}

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Str("alert_type", string(alertType)).
		Str("severity", string(severity)).
		Msg("health alert created")
}
