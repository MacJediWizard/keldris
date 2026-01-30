package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// BackupStore defines the interface for backup persistence operations.
type BackupStore interface {
	GetBackupsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.Backup, error)
	GetBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Backup, error)
	GetBackupByID(ctx context.Context, id uuid.UUID) (*models.Backup, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error)
	GetBackupValidationByBackupID(ctx context.Context, backupID uuid.UUID) (*models.BackupValidation, error)
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
		backups.GET("/calendar", h.Calendar)
		backups.GET("/:id", h.Get)
		backups.GET("/:id/validation", h.GetValidation)
	}
}

// List returns backups for the authenticated user's organization.
//
//	@Summary		List backups
//	@Description	Returns backup jobs for the current organization. Supports filtering by agent, schedule, or status.
//	@Tags			Backups
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Param			schedule_id	query		string	false	"Filter by schedule ID"
//	@Param			status		query		string	false	"Filter by status (running, completed, failed, canceled)"
//	@Success		200			{object}	map[string][]models.Backup
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/backups [get]
func (h *BackupsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
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
		if err != nil || agent.OrgID != user.CurrentOrgID {
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
		if err != nil || agent.OrgID != user.CurrentOrgID {
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
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
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
//
//	@Summary		Get backup
//	@Description	Returns details of a specific backup job
//	@Tags			Backups
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		200	{object}	models.Backup
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/backups/{id} [get]
func (h *BackupsHandler) Get(c *gin.Context) {
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
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// GetValidation returns the validation details for a specific backup.
//
//	@Summary		Get backup validation
//	@Description	Returns validation details for a specific backup job
//	@Tags			Backups
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		200	{object}	models.BackupValidation
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/backups/{id}/validation [get]
func (h *BackupsHandler) GetValidation(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	// Get the backup first
	backup, err := h.store.GetBackupByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", id.String()).Msg("failed to get backup")
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify backup's agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Get validation for this backup
	validation, err := h.store.GetBackupValidationByBackupID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", id.String()).Msg("failed to get backup validation")
		c.JSON(http.StatusNotFound, gin.H{"error": "validation not found for this backup"})
		return
	}

	c.JSON(http.StatusOK, validation)
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

// BackupCalendarDay represents backup statistics for a single day.
type BackupCalendarDay struct {
	Date      string           `json:"date"`
	Completed int              `json:"completed"`
	Failed    int              `json:"failed"`
	Running   int              `json:"running"`
	Scheduled int              `json:"scheduled"`
	Backups   []*models.Backup `json:"backups,omitempty"`
}

// ScheduledBackup represents a future scheduled backup.
type ScheduledBackup struct {
	ScheduleID   uuid.UUID `json:"schedule_id"`
	ScheduleName string    `json:"schedule_name"`
	AgentID      uuid.UUID `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

// BackupCalendarResponse is the response for the calendar endpoint.
type BackupCalendarResponse struct {
	Days      []BackupCalendarDay `json:"days"`
	Scheduled []ScheduledBackup   `json:"scheduled"`
}

// Calendar returns backup calendar data for a given month.
//
//	@Summary		Get backup calendar
//	@Description	Returns backup statistics and scheduled backups for a given month
//	@Tags			Backups
//	@Accept			json
//	@Produce		json
//	@Param			month	query		string	true	"Month in YYYY-MM format"
//	@Success		200		{object}	BackupCalendarResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/backups/calendar [get]
func (h *BackupsHandler) Calendar(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Parse month parameter (YYYY-MM format)
	monthParam := c.Query("month")
	if monthParam == "" {
		// Default to current month
		now := time.Now()
		monthParam = now.Format("2006-01")
	}

	// Parse the month
	monthTime, err := time.Parse("2006-01", monthParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month format, use YYYY-MM"})
		return
	}

	// Calculate start and end of month
	startOfMonth := monthTime
	endOfMonth := monthTime.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Get all backups for the month
	backups, err := h.store.GetBackupsByOrgIDAndDateRange(c.Request.Context(), user.CurrentOrgID, startOfMonth, endOfMonth)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get backups for calendar")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get backup calendar"})
		return
	}

	// Get all agents and their schedules
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get backup calendar"})
		return
	}

	// Create agent map for quick lookup
	agentMap := make(map[uuid.UUID]*models.Agent)
	for _, agent := range agents {
		agentMap[agent.ID] = agent
	}

	// Group backups by date
	backupsByDate := make(map[string][]*models.Backup)
	for _, backup := range backups {
		dateKey := backup.StartedAt.Format("2006-01-02")
		backupsByDate[dateKey] = append(backupsByDate[dateKey], backup)
	}

	// Build days response
	var days []BackupCalendarDay
	for dateKey, dayBackups := range backupsByDate {
		day := BackupCalendarDay{
			Date:    dateKey,
			Backups: dayBackups,
		}
		for _, b := range dayBackups {
			switch b.Status {
			case models.BackupStatusCompleted:
				day.Completed++
			case models.BackupStatusFailed:
				day.Failed++
			case models.BackupStatusRunning:
				day.Running++
			}
		}
		days = append(days, day)
	}

	// Sort days by date
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date < days[j].Date
	})

	// Calculate future scheduled backups
	var scheduled []ScheduledBackup
	now := time.Now()

	// Only calculate future schedules if we're looking at a month that includes the future
	if endOfMonth.After(now) {
		scheduleStart := now
		if startOfMonth.After(now) {
			scheduleStart = startOfMonth
		}

		for _, agent := range agents {
			if agent.Status != models.AgentStatusActive {
				continue
			}

			schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), agent.ID)
			if err != nil {
				h.logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to get schedules for agent")
				continue
			}

			for _, schedule := range schedules {
				if !schedule.Enabled {
					continue
				}

				// Parse cron expression
				parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
				cronSchedule, err := parser.Parse(schedule.CronExpression)
				if err != nil {
					h.logger.Warn().Err(err).Str("schedule_id", schedule.ID.String()).Msg("failed to parse cron expression")
					continue
				}

				// Get next run times within the month
				nextRun := scheduleStart
				for i := 0; i < 100; i++ { // Limit iterations
					nextRun = cronSchedule.Next(nextRun)
					if nextRun.After(endOfMonth) {
						break
					}

					scheduled = append(scheduled, ScheduledBackup{
						ScheduleID:   schedule.ID,
						ScheduleName: schedule.Name,
						AgentID:      agent.ID,
						AgentName:    agent.Hostname,
						ScheduledAt:  nextRun,
					})
				}
			}
		}
	}

	// Sort scheduled backups by time
	sort.Slice(scheduled, func(i, j int) bool {
		return scheduled[i].ScheduledAt.Before(scheduled[j].ScheduledAt)
	})

	c.JSON(http.StatusOK, BackupCalendarResponse{
		Days:      days,
		Scheduled: scheduled,
	})
}
