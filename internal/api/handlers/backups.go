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

// BackupStore defines the interface for backup persistence operations.
type BackupStore interface {
	GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetBackupByID(ctx context.Context, id uuid.UUID) (*models.Backup, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
}

// BackupsHandler handles backup-related HTTP endpoints.
type BackupsHandler struct {
	store  BackupStore
	logger zerolog.Logger
}

// NewBackupsHandler creates a new BackupsHandler.
func NewBackupsHandler(store BackupStore, logger zerolog.Logger) *BackupsHandler {
	return &BackupsHandler{
		store:  store,
		logger: logger.With().Str("component", "backups_handler").Logger(),
	}
}

// RegisterRoutes registers backup routes on the given router group.
func (h *BackupsHandler) RegisterRoutes(r *gin.RouterGroup) {
	backups := r.Group("/backups")
	{
		backups.GET("", h.List)
		backups.GET("/:id", h.Get)
	}
}

// List returns backups for the authenticated user's organization.
// GET /api/v1/backups
// Optional query params:
//   - agent_id: filter by agent
//   - schedule_id: filter by schedule
//   - status: filter by status (running, completed, failed, canceled)
func (h *BackupsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Handle schedule_id filter
	scheduleIDParam := c.Query("schedule_id")
	if scheduleIDParam != "" {
		scheduleID, err := uuid.Parse(scheduleIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule_id"})
			return
		}

		// Verify schedule access
		schedule, err := h.store.GetScheduleByID(c.Request.Context(), scheduleID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}

		agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
		if err != nil || agent.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}

		backups, err := h.store.GetBackupsByScheduleID(c.Request.Context(), scheduleID)
		if err != nil {
			h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to list backups")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"backups": filterByStatus(backups, c.Query("status"))})
		return
	}

	// Handle agent_id filter
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil || agent.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		backups, err := h.store.GetBackupsByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list backups")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"backups": filterByStatus(backups, c.Query("status"))})
		return
	}

	// Get all backups for all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	var allBackups []*models.Backup
	for _, agent := range agents {
		backups, err := h.store.GetBackupsByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to list backups for agent")
			continue
		}
		allBackups = append(allBackups, backups...)
	}

	c.JSON(http.StatusOK, gin.H{"backups": filterByStatus(allBackups, c.Query("status"))})
}

// Get returns a specific backup by ID.
// GET /api/v1/backups/:id
func (h *BackupsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetBackupByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", id.String()).Msg("failed to get backup")
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify backup's agent belongs to user's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// filterByStatus filters backups by status if status param is provided.
func filterByStatus(backups []*models.Backup, status string) []*models.Backup {
	if status == "" {
		return backups
	}

	statusFilter := models.BackupStatus(status)
	var filtered []*models.Backup
	for _, b := range backups {
		if b.Status == statusFilter {
			filtered = append(filtered, b)
		}
	}
	return filtered
}
