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

// ScheduleStore defines the interface for schedule persistence operations.
type ScheduleStore interface {
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	UpdateSchedule(ctx context.Context, schedule *models.Schedule) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetReplicationStatusBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]*models.ReplicationStatus, error)
}

// SchedulesHandler handles schedule-related HTTP endpoints.
type SchedulesHandler struct {
	store  ScheduleStore
	logger zerolog.Logger
}

// NewSchedulesHandler creates a new SchedulesHandler.
func NewSchedulesHandler(store ScheduleStore, logger zerolog.Logger) *SchedulesHandler {
	return &SchedulesHandler{
		store:  store,
		logger: logger.With().Str("component", "schedules_handler").Logger(),
	}
}

// RegisterRoutes registers schedule routes on the given router group.
func (h *SchedulesHandler) RegisterRoutes(r *gin.RouterGroup) {
	schedules := r.Group("/schedules")
	{
		schedules.GET("", h.List)
		schedules.POST("", h.Create)
		schedules.GET("/:id", h.Get)
		schedules.PUT("/:id", h.Update)
		schedules.DELETE("/:id", h.Delete)
		schedules.POST("/:id/run", h.Run)
		schedules.POST("/:id/dry-run", h.DryRun)
		schedules.GET("/:id/replication", h.GetReplicationStatus)
	}
}

// ScheduleRepositoryRequest represents a repository association in requests.
type ScheduleRepositoryRequest struct {
	RepositoryID uuid.UUID `json:"repository_id" binding:"required"`
	Priority     int       `json:"priority"`
	Enabled      bool      `json:"enabled"`
}

// CreateScheduleRequest is the request body for creating a schedule.
type CreateScheduleRequest struct {
	AgentID            uuid.UUID                   `json:"agent_id" binding:"required"`
	Repositories       []ScheduleRepositoryRequest `json:"repositories" binding:"required,min=1"`
	Name               string                      `json:"name" binding:"required,min=1,max=255"`
	CronExpression     string                      `json:"cron_expression" binding:"required"`
	Paths              []string                    `json:"paths" binding:"required,min=1"`
	Excludes           []string                    `json:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB   *int                        `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours      []int                       `json:"excluded_hours,omitempty"`
	CompressionLevel   *string                     `json:"compression_level,omitempty"`
	MaxFileSizeMB      *int                        `json:"max_file_size_mb,omitempty"` // Max file size in MB (0 = disabled)
	OnMountUnavailable string                      `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Enabled            *bool                       `json:"enabled,omitempty"`
}

// UpdateScheduleRequest is the request body for updating a schedule.
type UpdateScheduleRequest struct {
	Name               string                      `json:"name,omitempty"`
	CronExpression     string                      `json:"cron_expression,omitempty"`
	Paths              []string                    `json:"paths,omitempty"`
	Excludes           []string                    `json:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy     `json:"retention_policy,omitempty"`
	Repositories       []ScheduleRepositoryRequest `json:"repositories,omitempty"`
	BandwidthLimitKB   *int                        `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours      []int                       `json:"excluded_hours,omitempty"`
	CompressionLevel   *string                     `json:"compression_level,omitempty"`
	MaxFileSizeMB      *int                        `json:"max_file_size_mb,omitempty"` // Max file size in MB (0 = disabled)
	OnMountUnavailable *string                     `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Enabled            *bool                       `json:"enabled,omitempty"`
}

// List returns all schedules for agents in the authenticated user's organization.
//
//	@Summary		List schedules
//	@Description	Returns all backup schedules for the current organization
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Success		200			{object}	map[string][]models.Schedule
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules [get]
func (h *SchedulesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Optional agent_id filter
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list schedules")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"schedules": schedules})
		return
	}

	// Get all schedules for all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}

	var allSchedules []*models.Schedule
	for _, agent := range agents {
		schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to list schedules for agent")
			continue
		}
		allSchedules = append(allSchedules, schedules...)
	}

	c.JSON(http.StatusOK, gin.H{"schedules": allSchedules})
}

// Get returns a specific schedule by ID.
//
//	@Summary		Get schedule
//	@Description	Returns a specific backup schedule by ID
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	models.Schedule
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id} [get]
func (h *SchedulesHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to get schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify schedule's agent belongs to user's org
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// Create creates a new schedule.
//
//	@Summary		Create schedule
//	@Description	Creates a new backup schedule for an agent
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateScheduleRequest	true	"Schedule details"
//	@Success		201		{object}	models.Schedule
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules [post]
func (h *SchedulesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}

	// Verify all repositories belong to user's org
	var scheduleRepos []models.ScheduleRepository
	for _, repoReq := range req.Repositories {
		repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoReq.RepositoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoReq.RepositoryID.String()})
			return
		}
		if repo.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoReq.RepositoryID.String()})
			return
		}

		scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
			RepositoryID: repoReq.RepositoryID,
			Priority:     repoReq.Priority,
			Enabled:      repoReq.Enabled,
		})
	}

	// TODO: Validate cron expression using robfig/cron parser
	// For now we accept any string

	schedule := models.NewSchedule(req.AgentID, req.Name, req.CronExpression, req.Paths)
	schedule.Repositories = scheduleRepos

	if req.Excludes != nil {
		schedule.Excludes = req.Excludes
	}

	if req.RetentionPolicy != nil {
		schedule.RetentionPolicy = req.RetentionPolicy
	} else {
		schedule.RetentionPolicy = models.DefaultRetentionPolicy()
	}

	if req.BandwidthLimitKB != nil {
		schedule.BandwidthLimitKB = req.BandwidthLimitKB
	}

	if req.BackupWindow != nil {
		schedule.BackupWindow = req.BackupWindow
	}

	if req.ExcludedHours != nil {
		schedule.ExcludedHours = req.ExcludedHours
	}

	if req.CompressionLevel != nil {
		schedule.CompressionLevel = req.CompressionLevel
	}

	if req.MaxFileSizeMB != nil {
		schedule.MaxFileSizeMB = req.MaxFileSizeMB
	}

	if req.OnMountUnavailable != "" {
		schedule.OnMountUnavailable = models.MountBehavior(req.OnMountUnavailable)
	}

	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.store.CreateSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule"})
		return
	}

	h.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("name", req.Name).
		Str("agent_id", req.AgentID.String()).
		Int("num_repos", len(scheduleRepos)).
		Msg("schedule created")

	c.JSON(http.StatusCreated, schedule)
}

// Update updates an existing schedule.
//
//	@Summary		Update schedule
//	@Description	Updates an existing backup schedule
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Schedule ID"
//	@Param			request	body		UpdateScheduleRequest	true	"Schedule updates"
//	@Success		200		{object}	models.Schedule
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id} [put]
func (h *SchedulesHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.CronExpression != "" {
		schedule.CronExpression = req.CronExpression
	}
	if req.Paths != nil {
		schedule.Paths = req.Paths
	}
	if req.Excludes != nil {
		schedule.Excludes = req.Excludes
	}
	if req.RetentionPolicy != nil {
		schedule.RetentionPolicy = req.RetentionPolicy
	}
	if req.BandwidthLimitKB != nil {
		schedule.BandwidthLimitKB = req.BandwidthLimitKB
	}
	if req.BackupWindow != nil {
		schedule.BackupWindow = req.BackupWindow
	}
	if req.ExcludedHours != nil {
		schedule.ExcludedHours = req.ExcludedHours
	}
	if req.CompressionLevel != nil {
		schedule.CompressionLevel = req.CompressionLevel
	}
	if req.MaxFileSizeMB != nil {
		schedule.MaxFileSizeMB = req.MaxFileSizeMB
	}
	if req.OnMountUnavailable != nil {
		schedule.OnMountUnavailable = models.MountBehavior(*req.OnMountUnavailable)
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	// Update repositories if provided
	if req.Repositories != nil {
		// Verify all repositories belong to user's org
		var scheduleRepos []models.ScheduleRepository
		for _, repoReq := range req.Repositories {
			repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoReq.RepositoryID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoReq.RepositoryID.String()})
				return
			}
			if repo.OrgID != user.CurrentOrgID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoReq.RepositoryID.String()})
				return
			}

			scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
				RepositoryID: repoReq.RepositoryID,
				Priority:     repoReq.Priority,
				Enabled:      repoReq.Enabled,
			})
		}

		// Update repository associations
		if err := h.store.SetScheduleRepositories(c.Request.Context(), schedule.ID, scheduleRepos); err != nil {
			h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule repositories")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule repositories"})
			return
		}
		schedule.Repositories = scheduleRepos
	}

	if err := h.store.UpdateSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("schedule updated")
	c.JSON(http.StatusOK, schedule)
}

// Delete removes a schedule.
//
//	@Summary		Delete schedule
//	@Description	Removes a backup schedule
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id} [delete]
func (h *SchedulesHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	if err := h.store.DeleteSchedule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to delete schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("schedule deleted")
	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

// RunScheduleRequest is the request body for running a schedule.
type RunScheduleRequest struct {
	DryRun bool `json:"dry_run,omitempty"`
}

// RunScheduleResponse is the response for running a schedule.
type RunScheduleResponse struct {
	BackupID uuid.UUID `json:"backup_id"`
	Message  string    `json:"message"`
}

// DryRunResponse is the response for a dry run operation.
type DryRunResponse struct {
	ScheduleID     uuid.UUID              `json:"schedule_id"`
	TotalFiles     int                    `json:"total_files"`
	TotalSize      int64                  `json:"total_size"`
	NewFiles       int                    `json:"new_files"`
	ChangedFiles   int                    `json:"changed_files"`
	UnchangedFiles int                    `json:"unchanged_files"`
	FilesToBackup  []DryRunFileResponse   `json:"files_to_backup"`
	ExcludedFiles  []DryRunExcluded       `json:"excluded_files"`
	Message        string                 `json:"message"`
}

// DryRunFileResponse represents a file in the dry run response.
type DryRunFileResponse struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	Action string `json:"action"`
}

// DryRunExcluded represents an excluded file in the dry run response.
type DryRunExcluded struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Run triggers an immediate backup for this schedule.
//
//	@Summary		Run schedule
//	@Description	Triggers an immediate backup for the specified schedule
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		202	{object}	RunScheduleResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id}/run [post]
func (h *SchedulesHandler) Run(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	// TODO: Implement actual backup trigger when backup package is ready
	// This will create a backup record and dispatch to the agent
	h.logger.Info().
		Str("schedule_id", id.String()).
		Str("agent_id", schedule.AgentID.String()).
		Msg("manual backup run requested")

	c.JSON(http.StatusAccepted, RunScheduleResponse{
		BackupID: uuid.New(), // Placeholder - would be actual backup ID
		Message:  "Backup run not yet implemented. Schedule exists and is accessible.",
	})
}

// DryRun performs a dry run backup simulation for this schedule.
//
//	@Summary		Dry run schedule
//	@Description	Performs a dry run to preview what would be backed up for the specified schedule
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	DryRunResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id}/dry-run [post]
func (h *SchedulesHandler) DryRun(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	// TODO: Implement actual dry run when backup package is ready
	// This will use the Restic.DryRun method with the schedule's paths and excludes
	h.logger.Info().
		Str("schedule_id", id.String()).
		Str("agent_id", schedule.AgentID.String()).
		Strs("paths", schedule.Paths).
		Strs("excludes", schedule.Excludes).
		Msg("dry run requested")

	// Return placeholder response showing what would be backed up
	excludedFiles := make([]DryRunExcluded, 0, len(schedule.Excludes))
	for _, pattern := range schedule.Excludes {
		excludedFiles = append(excludedFiles, DryRunExcluded{
			Path:   pattern,
			Reason: "matched exclude pattern",
		})
	}

	c.JSON(http.StatusOK, DryRunResponse{
		ScheduleID:     id,
		TotalFiles:     0,
		TotalSize:      0,
		NewFiles:       0,
		ChangedFiles:   0,
		UnchangedFiles: 0,
		FilesToBackup:  []DryRunFileResponse{},
		ExcludedFiles:  excludedFiles,
		Message:        "Dry run not yet fully implemented. Schedule paths: " + formatPaths(schedule.Paths),
	})
}

// formatPaths formats a slice of paths as a comma-separated string.
func formatPaths(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	result := paths[0]
	for i := 1; i < len(paths); i++ {
		result += ", " + paths[i]
	}
	return result
}

// verifyScheduleAccess checks if the user has access to the schedule.
// Returns nil if access is granted, or sends an error response and returns error.
func (h *SchedulesHandler) verifyScheduleAccess(c *gin.Context, orgID uuid.UUID, schedule *models.Schedule) error {
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

	if agent.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

	return nil
}

// GetReplicationStatus returns the replication status for a schedule.
// GET /api/v1/schedules/:id/replication
func (h *SchedulesHandler) GetReplicationStatus(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	statuses, err := h.store.GetReplicationStatusBySchedule(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to get replication status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get replication status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"replication_status": statuses})
}
