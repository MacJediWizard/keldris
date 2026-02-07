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
	CreateAgentLogs(ctx context.Context, logs []*models.AgentLog) error
	GetPendingCommandsForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.AgentCommand, error)
	GetAgentCommandByID(ctx context.Context, id uuid.UUID) (*models.AgentCommand, error)
	UpdateAgentCommand(ctx context.Context, cmd *models.AgentCommand) error
	CreateBackup(ctx context.Context, backup *models.Backup) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
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
	r.POST("/logs", h.PushLogs)
	r.GET("/commands", h.GetCommands)
	r.POST("/commands/:id/ack", h.AcknowledgeCommand)
	r.POST("/commands/:id/result", h.ReportCommandResult)
	r.POST("/queued-backups", h.ReportQueuedBackups)
	r.POST("/reconnect", h.NotifyReconnection)
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

// PushLogsResponse is the response for log push operations.
type PushLogsResponse struct {
	Acknowledged bool   `json:"acknowledged"`
	Count        int    `json:"count"`
	AgentID      string `json:"agent_id"`
}

// PushLogs handles batched log submissions from agents.
// POST /api/v1/agent/logs
func (h *AgentAPIHandler) PushLogs(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	var req models.AgentLogBatch
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Convert entries to log records
	logs := make([]*models.AgentLog, 0, len(req.Logs))
	for _, entry := range req.Logs {
		log := models.NewAgentLog(agent.ID, agent.OrgID, entry.Level, entry.Message)
		log.Component = entry.Component
		log.Metadata = entry.Metadata
		if !entry.Timestamp.IsZero() {
			log.Timestamp = entry.Timestamp
		}
		logs = append(logs, log)
	}

	if err := h.store.CreateAgentLogs(c.Request.Context(), logs); err != nil {
		h.logger.Error().Err(err).
			Str("agent_id", agent.ID.String()).
			Int("log_count", len(logs)).
			Msg("failed to store agent logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store logs"})
		return
	}

	h.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Int("log_count", len(logs)).
		Msg("agent logs received")

	c.JSON(http.StatusOK, PushLogsResponse{
		Acknowledged: true,
		Count:        len(logs),
		AgentID:      agent.ID.String(),
	})
}

// GetCommandsResponse is the response for the commands polling endpoint.
type GetCommandsResponse struct {
	Commands []*models.AgentCommandResponse `json:"commands"`
}

// GetCommands returns pending commands for the agent.
// GET /api/v1/agent/commands
func (h *AgentAPIHandler) GetCommands(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	commands, err := h.store.GetPendingCommandsForAgent(c.Request.Context(), agent.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to get pending commands")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get commands"})
		return
	}

	// Convert to response format
	resp := GetCommandsResponse{
		Commands: make([]*models.AgentCommandResponse, len(commands)),
	}
	for i, cmd := range commands {
		resp.Commands[i] = cmd.ToResponse()
	}

	h.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Int("command_count", len(commands)).
		Msg("agent polled for commands")

	c.JSON(http.StatusOK, resp)
}

// AcknowledgeCommand marks a command as acknowledged by the agent.
// POST /api/v1/agent/commands/:id/ack
func (h *AgentAPIHandler) AcknowledgeCommand(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	cmdID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command ID"})
		return
	}

	cmd, err := h.store.GetAgentCommandByID(c.Request.Context(), cmdID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Verify command belongs to this agent
	if cmd.AgentID != agent.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Only pending commands can be acknowledged
	if !cmd.IsPending() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command is not pending"})
		return
	}

	cmd.Acknowledge()
	if err := h.store.UpdateAgentCommand(c.Request.Context(), cmd); err != nil {
		h.logger.Error().Err(err).Str("command_id", cmdID.String()).Msg("failed to acknowledge command")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to acknowledge command"})
		return
	}

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("command_id", cmdID.String()).
		Str("command_type", string(cmd.Type)).
		Msg("command acknowledged")

	c.JSON(http.StatusOK, gin.H{"acknowledged": true})
}

// CommandResultRequest is the request body for reporting command results.
type CommandResultRequest struct {
	Status  string                `json:"status" binding:"required,oneof=running completed failed"`
	Result  *models.CommandResult `json:"result,omitempty"`
}

// ReportCommandResult reports the result of a command execution.
// POST /api/v1/agent/commands/:id/result
func (h *AgentAPIHandler) ReportCommandResult(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	cmdID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command ID"})
		return
	}

	var req CommandResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	cmd, err := h.store.GetAgentCommandByID(c.Request.Context(), cmdID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Verify command belongs to this agent
	if cmd.AgentID != agent.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	// Only acknowledged or running commands can have results reported
	if cmd.IsTerminal() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command is already in terminal state"})
		return
	}

	// Update command based on status
	switch req.Status {
	case "running":
		cmd.MarkRunning()
	case "completed":
		cmd.Complete(req.Result)
	case "failed":
		errorMsg := "command failed"
		if req.Result != nil && req.Result.Error != "" {
			errorMsg = req.Result.Error
		}
		cmd.Fail(errorMsg)
	}

	if err := h.store.UpdateAgentCommand(c.Request.Context(), cmd); err != nil {
		h.logger.Error().Err(err).Str("command_id", cmdID.String()).Msg("failed to update command result")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update command"})
		return
	}

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("command_id", cmdID.String()).
		Str("command_type", string(cmd.Type)).
		Str("status", req.Status).
		Msg("command result reported")

	c.JSON(http.StatusOK, gin.H{"updated": true})
}

// QueuedBackupResult represents a backup that was executed while offline.
type QueuedBackupResult struct {
	ID           string     `json:"id" binding:"required"`
	ScheduleID   string     `json:"schedule_id" binding:"required"`
	ScheduleName string     `json:"schedule_name"`
	ScheduledAt  time.Time  `json:"scheduled_at" binding:"required"`
	QueuedAt     time.Time  `json:"queued_at" binding:"required"`
	Success      bool       `json:"success"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	BytesAdded   int64      `json:"bytes_added,omitempty"`
	FilesNew     int        `json:"files_new,omitempty"`
	FilesChanged int        `json:"files_changed,omitempty"`
	SnapshotID   string     `json:"snapshot_id,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	RepositoryID string     `json:"repository_id,omitempty"`
}

// ReportQueuedBackupsRequest is the request body for reporting queued backups.
type ReportQueuedBackupsRequest struct {
	Backups []QueuedBackupResult `json:"backups" binding:"required"`
}

// ReportQueuedBackupsResponse is the response for queued backup reporting.
type ReportQueuedBackupsResponse struct {
	Acknowledged bool `json:"acknowledged"`
	Processed    int  `json:"processed"`
}

// ReportQueuedBackups handles reports of backups executed while offline.
// POST /api/v1/agent/queued-backups
func (h *AgentAPIHandler) ReportQueuedBackups(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	var req ReportQueuedBackupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Backups) == 0 {
		c.JSON(http.StatusOK, ReportQueuedBackupsResponse{
			Acknowledged: true,
			Processed:    0,
		})
		return
	}

	ctx := c.Request.Context()
	processed := 0

	for _, qb := range req.Backups {
		scheduleID, err := uuid.Parse(qb.ScheduleID)
		if err != nil {
			h.logger.Warn().Err(err).Str("schedule_id", qb.ScheduleID).Msg("invalid schedule ID in queued backup")
			continue
		}

		// Verify schedule exists and belongs to this agent
		schedule, err := h.store.GetScheduleByID(ctx, scheduleID)
		if err != nil {
			h.logger.Warn().Err(err).Str("schedule_id", qb.ScheduleID).Msg("schedule not found for queued backup")
			continue
		}

		// Verify schedule belongs to this agent
		if schedule.AgentID != agent.ID {
			h.logger.Warn().
				Str("schedule_id", qb.ScheduleID).
				Str("agent_id", agent.ID.String()).
				Str("schedule_agent_id", schedule.AgentID.String()).
				Msg("schedule agent mismatch for queued backup")
			continue
		}

		// Create backup record using the existing constructor
		backup := models.NewBackup(scheduleID, agent.ID, nil)
		backup.BackupType = schedule.BackupType
		backup.CreatedAt = qb.ScheduledAt

		if qb.StartedAt != nil {
			backup.StartedAt = *qb.StartedAt
		} else {
			backup.StartedAt = qb.ScheduledAt
		}

		if qb.CompletedAt != nil {
			backup.CompletedAt = qb.CompletedAt
		}

		if !qb.Success {
			backup.Status = models.BackupStatusFailed
			backup.ErrorMessage = qb.ErrorMessage
		} else {
			backup.Status = models.BackupStatusCompleted
			backup.SnapshotID = qb.SnapshotID
			if qb.BytesAdded > 0 {
				backup.SizeBytes = &qb.BytesAdded
			}
			if qb.FilesNew > 0 {
				backup.FilesNew = &qb.FilesNew
			}
			if qb.FilesChanged > 0 {
				backup.FilesChanged = &qb.FilesChanged
			}
		}

		if err := h.store.CreateBackup(ctx, backup); err != nil {
			h.logger.Error().Err(err).
				Str("schedule_id", qb.ScheduleID).
				Msg("failed to create backup record from queue")
			continue
		}

		processed++
	}

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("hostname", agent.Hostname).
		Int("received", len(req.Backups)).
		Int("processed", processed).
		Msg("queued backups reported")

	c.JSON(http.StatusOK, ReportQueuedBackupsResponse{
		Acknowledged: true,
		Processed:    processed,
	})
}

// ReconnectionNotification is the request body for reconnection alerts.
type ReconnectionNotification struct {
	QueuedCount int `json:"queued_count" binding:"required"`
}

// ReconnectionResponse is the response for reconnection notification.
type ReconnectionResponse struct {
	Acknowledged bool `json:"acknowledged"`
	AlertCreated bool `json:"alert_created"`
}

// NotifyReconnection handles agent reconnection notifications.
// POST /api/v1/agent/reconnect
func (h *AgentAPIHandler) NotifyReconnection(c *gin.Context) {
	agent := middleware.RequireAgent(c)
	if agent == nil {
		return
	}

	var req ReconnectionNotification
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	alertCreated := false

	if req.QueuedCount > 0 {
		// Create alert for reconnection with queued backups
		alert := models.NewAlert(
			agent.OrgID,
			models.AlertTypeAgentReconnectedWithQueue,
			models.AlertSeverityInfo,
			fmt.Sprintf("Agent %s reconnected with queued backups", agent.Hostname),
			fmt.Sprintf("Agent reconnected after being offline and has %d backups queued for sync.", req.QueuedCount),
		)
		alert.SetResource(models.ResourceTypeAgent, agent.ID)
		alert.Metadata = map[string]any{
			"hostname":      agent.Hostname,
			"queued_count":  req.QueuedCount,
			"reconnected_at": time.Now().Format(time.RFC3339),
		}

		if err := h.store.CreateAlert(ctx, alert); err != nil {
			h.logger.Error().Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("failed to create reconnection alert")
		} else {
			alertCreated = true
			h.logger.Info().
				Str("agent_id", agent.ID.String()).
				Str("hostname", agent.Hostname).
				Int("queued_count", req.QueuedCount).
				Msg("agent reconnection alert created")
		}
	}

	// Update agent's last seen time
	agent.MarkSeen()
	if err := h.store.UpdateAgent(ctx, agent); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to update agent on reconnection")
	}

	c.JSON(http.StatusOK, ReconnectionResponse{
		Acknowledged: true,
		AlertCreated: alertCreated,
	})
}
