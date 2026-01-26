package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BackupQueueStore defines the interface for backup queue operations.
type BackupQueueStore interface {
	GetQueuedBackupsWithDetails(ctx context.Context, orgID uuid.UUID) ([]*models.BackupQueueEntryWithDetails, error)
	GetBackupQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.BackupQueueSummary, error)
	DeleteBackupQueueEntry(ctx context.Context, id uuid.UUID) error
	GetRunningBackupsCountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
	GetRunningBackupsCountByAgent(ctx context.Context, agentID uuid.UUID) (int, error)
	UpdateOrganizationConcurrencyLimit(ctx context.Context, orgID uuid.UUID, limit *int) error
	UpdateAgentConcurrencyLimit(ctx context.Context, agentID uuid.UUID, limit *int) error
	GetOrganizationByIDWithConcurrency(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetAgentByIDWithConcurrency(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// BackupQueueHandler handles backup queue-related HTTP endpoints.
type BackupQueueHandler struct {
	store  BackupQueueStore
	rbac   *auth.RBAC
	logger zerolog.Logger
}

// NewBackupQueueHandler creates a new BackupQueueHandler.
func NewBackupQueueHandler(store BackupQueueStore, rbac *auth.RBAC, logger zerolog.Logger) *BackupQueueHandler {
	return &BackupQueueHandler{
		store:  store,
		rbac:   rbac,
		logger: logger.With().Str("component", "backup_queue_handler").Logger(),
	}
}

// RegisterRoutes registers backup queue routes on the given router group.
func (h *BackupQueueHandler) RegisterRoutes(r *gin.RouterGroup) {
	queue := r.Group("/backup-queue")
	{
		queue.GET("", h.ListQueue)
		queue.GET("/summary", h.GetSummary)
		queue.DELETE("/:id", h.CancelQueuedBackup)
	}

	// Concurrency settings endpoints
	r.GET("/organizations/:id/concurrency", h.GetOrgConcurrency)
	r.PUT("/organizations/:id/concurrency", h.UpdateOrgConcurrency)
	r.GET("/agents/:id/concurrency", h.GetAgentConcurrency)
	r.PUT("/agents/:id/concurrency", h.UpdateAgentConcurrency)
}

// UpdateConcurrencyRequest is the request body for updating concurrency limits.
type UpdateConcurrencyRequest struct {
	MaxConcurrentBackups *int `json:"max_concurrent_backups"` // nil means unlimited
}

// ConcurrencyResponse is the response for concurrency endpoints.
type ConcurrencyResponse struct {
	MaxConcurrentBackups *int `json:"max_concurrent_backups"`
	RunningCount         int  `json:"running_count"`
	QueuedCount          int  `json:"queued_count"`
}

// QueueEntryResponse represents a queued backup for API responses.
type QueueEntryResponse struct {
	ID            string `json:"id"`
	ScheduleID    string `json:"schedule_id"`
	ScheduleName  string `json:"schedule_name"`
	AgentID       string `json:"agent_id"`
	AgentHostname string `json:"agent_hostname"`
	Priority      int    `json:"priority"`
	QueuePosition int    `json:"queue_position"`
	QueuedAt      string `json:"queued_at"`
}

// QueueSummaryResponse is the response for queue summary.
type QueueSummaryResponse struct {
	TotalQueued     int                   `json:"total_queued"`
	TotalRunning    int                   `json:"total_running"`
	AvgWaitMinutes  float64               `json:"avg_wait_minutes"`
	OldestQueuedAt  *string               `json:"oldest_queued_at,omitempty"`
	QueuedByAgent   map[string]int        `json:"queued_by_agent,omitempty"`
	Entries         []QueueEntryResponse  `json:"entries,omitempty"`
}

// ListQueue returns all queued backups for the current organization.
// GET /api/v1/backup-queue
func (h *BackupQueueHandler) ListQueue(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	entries, err := h.store.GetQueuedBackupsWithDetails(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get backup queue")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get backup queue"})
		return
	}

	response := make([]QueueEntryResponse, len(entries))
	for i, e := range entries {
		response[i] = QueueEntryResponse{
			ID:            e.ID.String(),
			ScheduleID:    e.ScheduleID.String(),
			ScheduleName:  e.ScheduleName,
			AgentID:       e.AgentID.String(),
			AgentHostname: e.AgentName,
			Priority:      e.Priority,
			QueuePosition: e.QueuePosition,
			QueuedAt:      e.QueuedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"queue": response})
}

// GetSummary returns queue statistics for the current organization.
// GET /api/v1/backup-queue/summary
func (h *BackupQueueHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	summary, err := h.store.GetBackupQueueSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get queue summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get queue summary"})
		return
	}

	runningCount, _ := h.store.GetRunningBackupsCountByOrg(c.Request.Context(), user.CurrentOrgID)

	response := QueueSummaryResponse{
		TotalQueued:    summary.TotalQueued,
		TotalRunning:   runningCount,
		AvgWaitMinutes: summary.AvgWaitMinutes,
	}

	if summary.OldestQueued != nil {
		oldest := summary.OldestQueued.Format("2006-01-02T15:04:05Z07:00")
		response.OldestQueuedAt = &oldest
	}

	// Convert agent UUIDs to strings for response
	if len(summary.ByAgent) > 0 {
		response.QueuedByAgent = make(map[string]int)
		for agentID, count := range summary.ByAgent {
			response.QueuedByAgent[agentID.String()] = count
		}
	}

	c.JSON(http.StatusOK, response)
}

// CancelQueuedBackup removes a backup from the queue.
// DELETE /api/v1/backup-queue/:id
func (h *BackupQueueHandler) CancelQueuedBackup(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid queue entry ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	if err := h.store.DeleteBackupQueueEntry(c.Request.Context(), entryID); err != nil {
		h.logger.Error().Err(err).Str("entry_id", entryID.String()).Msg("failed to cancel queued backup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel queued backup"})
		return
	}

	h.logger.Info().
		Str("entry_id", entryID.String()).
		Str("user_id", user.ID.String()).
		Msg("queued backup canceled")

	c.JSON(http.StatusOK, gin.H{"message": "queued backup canceled"})
}

// GetOrgConcurrency returns concurrency settings for an organization.
// GET /api/v1/organizations/:id/concurrency
func (h *BackupQueueHandler) GetOrgConcurrency(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	org, err := h.store.GetOrganizationByIDWithConcurrency(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get organization")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get organization"})
		return
	}

	runningCount, _ := h.store.GetRunningBackupsCountByOrg(c.Request.Context(), orgID)
	summary, _ := h.store.GetBackupQueueSummary(c.Request.Context(), orgID)

	queuedCount := 0
	if summary != nil {
		queuedCount = summary.TotalQueued
	}

	c.JSON(http.StatusOK, ConcurrencyResponse{
		MaxConcurrentBackups: org.MaxConcurrentBackups,
		RunningCount:         runningCount,
		QueuedCount:          queuedCount,
	})
}

// UpdateOrgConcurrency updates concurrency settings for an organization.
// PUT /api/v1/organizations/:id/concurrency
func (h *BackupQueueHandler) UpdateOrgConcurrency(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission (require org update permission)
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req UpdateConcurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate limit (must be positive if set)
	if req.MaxConcurrentBackups != nil && *req.MaxConcurrentBackups < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_concurrent_backups must be non-negative"})
		return
	}

	if err := h.store.UpdateOrganizationConcurrencyLimit(c.Request.Context(), orgID, req.MaxConcurrentBackups); err != nil {
		h.logger.Error().Err(err).Msg("failed to update organization concurrency limit")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update concurrency limit"})
		return
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Interface("max_concurrent_backups", req.MaxConcurrentBackups).
		Str("user_id", user.ID.String()).
		Msg("organization concurrency limit updated")

	c.JSON(http.StatusOK, gin.H{
		"message":              "concurrency limit updated",
		"max_concurrent_backups": req.MaxConcurrentBackups,
	})
}

// GetAgentConcurrency returns concurrency settings for an agent.
// GET /api/v1/agents/:id/concurrency
func (h *BackupQueueHandler) GetAgentConcurrency(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByIDWithConcurrency(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, agent.OrgID, auth.PermAgentRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	runningCount, _ := h.store.GetRunningBackupsCountByAgent(c.Request.Context(), agentID)

	c.JSON(http.StatusOK, ConcurrencyResponse{
		MaxConcurrentBackups: agent.MaxConcurrentBackups,
		RunningCount:         runningCount,
		QueuedCount:          0, // TODO: Get agent-specific queued count
	})
}

// UpdateAgentConcurrency updates concurrency settings for an agent.
// PUT /api/v1/agents/:id/concurrency
func (h *BackupQueueHandler) UpdateAgentConcurrency(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByIDWithConcurrency(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, agent.OrgID, auth.PermAgentUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req UpdateConcurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate limit (must be positive if set)
	if req.MaxConcurrentBackups != nil && *req.MaxConcurrentBackups < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_concurrent_backups must be non-negative"})
		return
	}

	if err := h.store.UpdateAgentConcurrencyLimit(c.Request.Context(), agentID, req.MaxConcurrentBackups); err != nil {
		h.logger.Error().Err(err).Msg("failed to update agent concurrency limit")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update concurrency limit"})
		return
	}

	h.logger.Info().
		Str("agent_id", agentID.String()).
		Interface("max_concurrent_backups", req.MaxConcurrentBackups).
		Str("user_id", user.ID.String()).
		Msg("agent concurrency limit updated")

	c.JSON(http.StatusOK, gin.H{
		"message":              "concurrency limit updated",
		"max_concurrent_backups": req.MaxConcurrentBackups,
	})
}
