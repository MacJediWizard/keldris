package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
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
	CreateBackup(ctx context.Context, backup *models.Backup) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
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
		schedules.POST("/:id/clone", h.Clone)
		schedules.POST("/bulk-clone", h.BulkClone)
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
	AgentID            uuid.UUID                    `json:"agent_id" binding:"required"`
	Repositories       []ScheduleRepositoryRequest  `json:"repositories" binding:"required,min=1"`
	Name               string                       `json:"name" binding:"required,min=1,max=255"`
	BackupType         string                       `json:"backup_type,omitempty"`     // "file" (default) or "docker"
	CronExpression     string                       `json:"cron_expression" binding:"required"`
	Paths              []string                     `json:"paths,omitempty"`           // Required for file backups, optional for docker
	Excludes           []string                     `json:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy      `json:"retention_policy,omitempty"`
	BandwidthLimitKB   *int                         `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow         `json:"backup_window,omitempty"`
	ExcludedHours      []int                        `json:"excluded_hours,omitempty"`
	CompressionLevel   *string                      `json:"compression_level,omitempty"`
	MaxFileSizeMB      *int                         `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable string                       `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Priority           *int                         `json:"priority,omitempty"`             // 1=high, 2=medium, 3=low
	Preemptible        *bool                        `json:"preemptible,omitempty"`          // Can be preempted by higher priority
	DockerOptions      *models.DockerBackupOptions  `json:"docker_options,omitempty"`       // Docker-specific backup options
	Enabled            *bool                        `json:"enabled,omitempty"`
	AgentID          uuid.UUID               `json:"agent_id" binding:"required"`
	RepositoryID     uuid.UUID               `json:"repository_id" binding:"required"`
	Name             string                  `json:"name" binding:"required,min=1,max=255"`
	CronExpression   string                  `json:"cron_expression" binding:"required"`
	Paths            []string                `json:"paths" binding:"required,min=1"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	Enabled          *bool                   `json:"enabled,omitempty"`
	AgentID          uuid.UUID                   `json:"agent_id" binding:"required"`
	Repositories     []ScheduleRepositoryRequest `json:"repositories" binding:"required,min=1"`
	Name             string                      `json:"name" binding:"required,min=1,max=255"`
	CronExpression   string                      `json:"cron_expression" binding:"required"`
	Paths            []string                    `json:"paths" binding:"required,min=1"`
	Excludes         []string                    `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                        `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours    []int                       `json:"excluded_hours,omitempty"`
	CompressionLevel *string                     `json:"compression_level,omitempty"`
	Enabled          *bool                       `json:"enabled,omitempty"`
	AgentID            uuid.UUID                     `json:"agent_id" binding:"required"`
	Repositories       []ScheduleRepositoryRequest   `json:"repositories" binding:"required,min=1"`
	Name               string                        `json:"name" binding:"required,min=1,max=255"`
	BackupType         string                        `json:"backup_type,omitempty"`     // "file" (default), "docker", "pihole", "postgres", or "proxmox"
	CronExpression     string                        `json:"cron_expression" binding:"required"`
	Paths              []string                      `json:"paths,omitempty"`           // Required for file backups, optional for docker/postgres/proxmox
	AgentID            uuid.UUID                     `json:"agent_id" binding:"required"`
	Repositories       []ScheduleRepositoryRequest   `json:"repositories" binding:"required,min=1"`
	Name               string                        `json:"name" binding:"required,min=1,max=255"`
	BackupType         string                        `json:"backup_type,omitempty"`     // "file" (default), "docker", "pihole", "postgres", or "proxmox"
	CronExpression     string                        `json:"cron_expression" binding:"required"`
	Paths              []string                      `json:"paths,omitempty"`           // Required for file backups, optional for docker/postgres/proxmox
	Excludes           []string                      `json:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy       `json:"retention_policy,omitempty"`
	BandwidthLimitKB   *int                          `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow          `json:"backup_window,omitempty"`
	ExcludedHours      []int                         `json:"excluded_hours,omitempty"`
	CompressionLevel   *string                       `json:"compression_level,omitempty"`
	MaxFileSizeMB      *int                          `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable string                        `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Priority           *int                          `json:"priority,omitempty"`             // 1=high, 2=medium, 3=low
	Preemptible        *bool                         `json:"preemptible,omitempty"`          // Can be preempted by higher priority
	DockerOptions      *models.DockerBackupOptions   `json:"docker_options,omitempty"`       // Docker-specific backup options
	PostgresOptions    *models.PostgresBackupConfig  `json:"postgres_options,omitempty"`     // PostgreSQL-specific backup options
	ProxmoxOptions     *models.ProxmoxBackupOptions  `json:"proxmox_options,omitempty"`      // Proxmox-specific backup options
	Enabled            *bool                         `json:"enabled,omitempty"`
	Enabled          *bool                       `json:"enabled,omitempty"`
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
	OnMountUnavailable string                      `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Enabled            *bool                       `json:"enabled,omitempty"`
	Enabled            *bool                         `json:"enabled,omitempty"`
	}
}

// CreateScheduleRequest is the request body for creating a schedule.
type CreateScheduleRequest struct {
	AgentID         uuid.UUID               `json:"agent_id" binding:"required"`
	RepositoryID    uuid.UUID               `json:"repository_id" binding:"required"`
	Name            string                  `json:"name" binding:"required,min=1,max=255"`
	CronExpression  string                  `json:"cron_expression" binding:"required"`
	Paths           []string                `json:"paths" binding:"required,min=1"`
	Excludes        []string                `json:"excludes,omitempty"`
	RetentionPolicy *models.RetentionPolicy `json:"retention_policy,omitempty"`
	Enabled         *bool                   `json:"enabled,omitempty"`
}

// UpdateScheduleRequest is the request body for updating a schedule.
type UpdateScheduleRequest struct {
	Name               string                        `json:"name,omitempty"`
	BackupType         string                        `json:"backup_type,omitempty"` // "file", "docker", "pihole", "postgres", or "proxmox"
	BackupType         string                        `json:"backup_type,omitempty"` // "file", "docker", "pihole", or "postgres"
	CronExpression     string                        `json:"cron_expression,omitempty"`
	Paths              []string                      `json:"paths,omitempty"`
	Excludes           []string                      `json:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy       `json:"retention_policy,omitempty"`
	Repositories       []ScheduleRepositoryRequest   `json:"repositories,omitempty"`
	BandwidthLimitKB   *int                          `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow          `json:"backup_window,omitempty"`
	ExcludedHours      []int                         `json:"excluded_hours,omitempty"`
	CompressionLevel   *string                       `json:"compression_level,omitempty"`
	MaxFileSizeMB      *int                          `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable *string                       `json:"on_mount_unavailable,omitempty"` // "skip" or "fail"
	Priority           *int                          `json:"priority,omitempty"`             // 1=high, 2=medium, 3=low
	Preemptible        *bool                         `json:"preemptible,omitempty"`          // Can be preempted by higher priority
	DockerOptions      *models.DockerBackupOptions   `json:"docker_options,omitempty"`       // Docker-specific backup options
	PostgresOptions    *models.PostgresBackupConfig  `json:"postgres_options,omitempty"`     // PostgreSQL-specific backup options
	ProxmoxOptions     *models.ProxmoxBackupOptions  `json:"proxmox_options,omitempty"`      // Proxmox-specific backup options
	Enabled            *bool                         `json:"enabled,omitempty"`
}

// CloneScheduleRequest is the request body for cloning a schedule.
type CloneScheduleRequest struct {
	Name          string      `json:"name,omitempty"`            // Optional new name (default: "Copy of X")
	TargetAgentID *uuid.UUID  `json:"target_agent_id,omitempty"` // Optional target agent (default: same agent)
	TargetRepoIDs []uuid.UUID `json:"target_repo_ids,omitempty"` // Optional target repositories (default: same repos)
}

// BulkCloneScheduleRequest is the request body for cloning a schedule to multiple agents.
type BulkCloneScheduleRequest struct {
	ScheduleID     uuid.UUID   `json:"schedule_id" binding:"required"`
	TargetAgentIDs []uuid.UUID `json:"target_agent_ids" binding:"required,min=1"`
	NamePrefix     string      `json:"name_prefix,omitempty"` // Optional prefix for cloned schedule names
}

// BulkCloneResponse is the response for bulk clone operations.
type BulkCloneResponse struct {
	Schedules []*models.Schedule `json:"schedules"`
	Errors    []string           `json:"errors,omitempty"`
	Name             string                  `json:"name,omitempty"`
	CronExpression   string                  `json:"cron_expression,omitempty"`
	Paths            []string                `json:"paths,omitempty"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	Enabled          *bool                   `json:"enabled,omitempty"`
	Name             string                      `json:"name,omitempty"`
	CronExpression   string                      `json:"cron_expression,omitempty"`
	Paths            []string                    `json:"paths,omitempty"`
	Excludes         []string                    `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy     `json:"retention_policy,omitempty"`
	Repositories     []ScheduleRepositoryRequest `json:"repositories,omitempty"`
	BandwidthLimitKB *int                        `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow        `json:"backup_window,omitempty"`
	ExcludedHours    []int                       `json:"excluded_hours,omitempty"`
	CompressionLevel *string                     `json:"compression_level,omitempty"`
	Enabled          *bool                       `json:"enabled,omitempty"`
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
	Name            string                  `json:"name,omitempty"`
	CronExpression  string                  `json:"cron_expression,omitempty"`
	Paths           []string                `json:"paths,omitempty"`
	Excludes        []string                `json:"excludes,omitempty"`
	RetentionPolicy *models.RetentionPolicy `json:"retention_policy,omitempty"`
	Enabled         *bool                   `json:"enabled,omitempty"`
}

// List returns all schedules for agents in the authenticated user's organization.
// GET /api/v1/schedules
// Optional query param: agent_id to filter by agent
func (h *SchedulesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
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
		if agent.OrgID != dbUser.OrgID {
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
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list agents")
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
// GET /api/v1/schedules/:id
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
	if err := h.verifyScheduleAccess(c, user.ID, schedule); err != nil {
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
// POST /api/v1/schedules
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

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != user.CurrentOrgID {
	if agent.OrgID != dbUser.OrgID {
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

	// Validate cron expression
	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := cronParser.Parse(req.CronExpression); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron expression"})
	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found"})
		return
	}

	// Verify repository belongs to user's org
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found"})
		return
	}
	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found"})
		return
	}

	// TODO: Validate cron expression using robfig/cron parser
	// For now we accept any string

	schedule := models.NewSchedule(req.AgentID, req.Name, req.CronExpression, req.Paths)
	schedule.Repositories = scheduleRepos
	schedule := models.NewSchedule(req.AgentID, req.RepositoryID, req.Name, req.CronExpression, req.Paths)

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

	if req.Priority != nil {
		if *req.Priority >= 1 && *req.Priority <= 3 {
			schedule.Priority = models.SchedulePriority(*req.Priority)
		}
	}

	if req.Preemptible != nil {
		schedule.Preemptible = *req.Preemptible
	}

	// Handle backup type
	if req.BackupType != "" {
		schedule.BackupType = models.BackupType(req.BackupType)
	} else {
		schedule.BackupType = models.BackupTypeFile
	}

	// Handle Docker-specific options
	if req.DockerOptions != nil {
		schedule.DockerOptions = req.DockerOptions
	}

	// Handle PostgreSQL-specific options
	if req.PostgresOptions != nil {
		schedule.PostgresConfig = req.PostgresOptions
	}

	// Handle Proxmox-specific options
	if req.ProxmoxOptions != nil {
		schedule.ProxmoxOptions = req.ProxmoxOptions
	}

	// Validate paths for file backups
	if schedule.BackupType == models.BackupTypeFile && len(schedule.Paths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paths are required for file backups"})
		return
	}

	// Validate PostgreSQL backups have config
	if schedule.BackupType == models.BackupTypePostgres && schedule.PostgresConfig == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "postgres_options are required for PostgreSQL backups"})
		return
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
// PUT /api/v1/schedules/:id
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
	if err := h.verifyScheduleAccess(c, user.ID, schedule); err != nil {
		return
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.CronExpression != "" {
		cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := cronParser.Parse(req.CronExpression); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron expression"})
			return
		}
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
	if req.Priority != nil {
		if *req.Priority >= 1 && *req.Priority <= 3 {
			schedule.Priority = models.SchedulePriority(*req.Priority)
		}
	}
	if req.Preemptible != nil {
		schedule.Preemptible = *req.Preemptible
	}

	// Handle backup type
	if req.BackupType != "" {
		schedule.BackupType = models.BackupType(req.BackupType)
	}

	// Handle Docker-specific options
	if req.DockerOptions != nil {
		schedule.DockerOptions = req.DockerOptions
	}

	// Handle PostgreSQL-specific options
	if req.PostgresOptions != nil {
		schedule.PostgresConfig = req.PostgresOptions
	}

	// Handle Proxmox-specific options
	if req.ProxmoxOptions != nil {
		schedule.ProxmoxOptions = req.ProxmoxOptions
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
// DELETE /api/v1/schedules/:id
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
	if err := h.verifyScheduleAccess(c, user.ID, schedule); err != nil {
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
	ScheduleID     uuid.UUID            `json:"schedule_id"`
	TotalFiles     int                  `json:"total_files"`
	TotalSize      int64                `json:"total_size"`
	NewFiles       int                  `json:"new_files"`
	ChangedFiles   int                  `json:"changed_files"`
	UnchangedFiles int                  `json:"unchanged_files"`
	FilesToBackup  []DryRunFileResponse `json:"files_to_backup"`
	ExcludedFiles  []DryRunExcluded     `json:"excluded_files"`
	Message        string               `json:"message"`
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
// Run triggers an immediate backup for this schedule.
// POST /api/v1/schedules/:id/run
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

	backup := models.NewBackup(schedule.ID, schedule.AgentID, nil)
	if err := h.store.CreateBackup(c.Request.Context(), backup); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to create backup record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup"})
		return
	}

	h.logger.Info().
		Str("schedule_id", id.String()).
		Str("agent_id", schedule.AgentID.String()).
		Str("backup_id", backup.ID.String()).
		Msg("manual backup run triggered")

	c.JSON(http.StatusAccepted, RunScheduleResponse{
		BackupID: backup.ID,
		Message:  "Backup triggered successfully",
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
	if err := h.verifyScheduleAccess(c, user.ID, schedule); err != nil {
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

// verifyScheduleAccess checks if the user has access to the schedule.
// Returns nil if access is granted, or sends an error response and returns error.
func (h *SchedulesHandler) verifyScheduleAccess(c *gin.Context, userID uuid.UUID, schedule *models.Schedule) error {
	dbUser, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return err
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

	if agent.OrgID != orgID {
	if agent.OrgID != dbUser.OrgID {
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

// Clone creates a copy of an existing schedule.
//
//	@Summary		Clone schedule
//	@Description	Creates a copy of an existing schedule, optionally targeting a different agent or repository
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Schedule ID to clone"
//	@Param			request	body		CloneScheduleRequest	true	"Clone options"
//	@Success		201		{object}	models.Schedule
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/{id}/clone [post]
func (h *SchedulesHandler) Clone(c *gin.Context) {
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

	var req CloneScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get the source schedule
	source, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, source); err != nil {
		return
	}

	// Determine target agent
	targetAgentID := source.AgentID
	if req.TargetAgentID != nil {
		// Verify target agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), *req.TargetAgentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target agent not found"})
			return
		}
		if agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target agent not found"})
			return
		}
		targetAgentID = *req.TargetAgentID
	}

	// Determine name
	name := req.Name
	if name == "" {
		name = "Copy of " + source.Name
	}

	// Create new schedule with copied settings
	cloned := models.NewSchedule(targetAgentID, name, source.CronExpression, source.Paths)
	cloned.BackupType = source.BackupType
	cloned.Excludes = source.Excludes
	cloned.RetentionPolicy = source.RetentionPolicy
	cloned.BandwidthLimitKB = source.BandwidthLimitKB
	cloned.BackupWindow = source.BackupWindow
	cloned.ExcludedHours = source.ExcludedHours
	cloned.CompressionLevel = source.CompressionLevel
	cloned.MaxFileSizeMB = source.MaxFileSizeMB
	cloned.OnMountUnavailable = source.OnMountUnavailable
	cloned.ClassificationLevel = source.ClassificationLevel
	cloned.ClassificationDataTypes = source.ClassificationDataTypes
	cloned.DockerOptions = source.DockerOptions
	cloned.PostgresConfig = source.PostgresConfig
	cloned.ProxmoxOptions = source.ProxmoxOptions
	cloned.Enabled = source.Enabled

	// Handle repositories
	if len(req.TargetRepoIDs) > 0 {
		// Verify all target repositories belong to user's org
		var scheduleRepos []models.ScheduleRepository
		for i, repoID := range req.TargetRepoIDs {
			repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoID.String()})
				return
			}
			if repo.OrgID != user.CurrentOrgID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found: " + repoID.String()})
				return
			}
			scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
				RepositoryID: repoID,
				Priority:     i,
				Enabled:      true,
			})
		}
		cloned.Repositories = scheduleRepos
	} else {
		// Copy repositories from source
		var scheduleRepos []models.ScheduleRepository
		for _, repo := range source.Repositories {
			scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
				RepositoryID: repo.RepositoryID,
				Priority:     repo.Priority,
				Enabled:      repo.Enabled,
			})
		}
		cloned.Repositories = scheduleRepos
	}

	if err := h.store.CreateSchedule(c.Request.Context(), cloned); err != nil {
		h.logger.Error().Err(err).Str("source_id", id.String()).Msg("failed to clone schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clone schedule"})
		return
	}

	h.logger.Info().
		Str("source_id", id.String()).
		Str("cloned_id", cloned.ID.String()).
		Str("name", name).
		Str("target_agent_id", targetAgentID.String()).
		Msg("schedule cloned")

	c.JSON(http.StatusCreated, cloned)
}

// BulkClone clones a schedule to multiple target agents.
//
//	@Summary		Bulk clone schedule
//	@Description	Creates copies of a schedule for multiple target agents
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			request	body		BulkCloneScheduleRequest	true	"Bulk clone options"
//	@Success		201		{object}	BulkCloneResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/schedules/bulk-clone [post]
func (h *SchedulesHandler) BulkClone(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req BulkCloneScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get the source schedule
	source, err := h.store.GetScheduleByID(c.Request.Context(), req.ScheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, source); err != nil {
		return
	}

	var clonedSchedules []*models.Schedule
	var errors []string

	for _, targetAgentID := range req.TargetAgentIDs {
		// Verify target agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), targetAgentID)
		if err != nil {
			errors = append(errors, "agent not found: "+targetAgentID.String())
			continue
		}
		if agent.OrgID != user.CurrentOrgID {
			errors = append(errors, "agent not found: "+targetAgentID.String())
			continue
		}

		// Generate name
		var name string
		if req.NamePrefix != "" {
			name = req.NamePrefix + " - " + agent.Hostname
		} else {
			name = source.Name + " (" + agent.Hostname + ")"
		}

		// Create new schedule with copied settings
		cloned := models.NewSchedule(targetAgentID, name, source.CronExpression, source.Paths)
		cloned.BackupType = source.BackupType
		cloned.Excludes = source.Excludes
		cloned.RetentionPolicy = source.RetentionPolicy
		cloned.BandwidthLimitKB = source.BandwidthLimitKB
		cloned.BackupWindow = source.BackupWindow
		cloned.ExcludedHours = source.ExcludedHours
		cloned.CompressionLevel = source.CompressionLevel
		cloned.MaxFileSizeMB = source.MaxFileSizeMB
		cloned.OnMountUnavailable = source.OnMountUnavailable
		cloned.ClassificationLevel = source.ClassificationLevel
		cloned.ClassificationDataTypes = source.ClassificationDataTypes
		cloned.DockerOptions = source.DockerOptions
		cloned.PostgresConfig = source.PostgresConfig
		cloned.ProxmoxOptions = source.ProxmoxOptions
		cloned.Enabled = source.Enabled

		// Copy repositories from source
		var scheduleRepos []models.ScheduleRepository
		for _, repo := range source.Repositories {
			scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
				RepositoryID: repo.RepositoryID,
				Priority:     repo.Priority,
				Enabled:      repo.Enabled,
			})
		}
		cloned.Repositories = scheduleRepos

		if err := h.store.CreateSchedule(c.Request.Context(), cloned); err != nil {
			h.logger.Error().Err(err).
				Str("source_id", req.ScheduleID.String()).
				Str("target_agent_id", targetAgentID.String()).
				Msg("failed to clone schedule to agent")
			errors = append(errors, "failed to clone to agent "+agent.Hostname+": "+err.Error())
			continue
		}

		clonedSchedules = append(clonedSchedules, cloned)
		h.logger.Info().
			Str("source_id", req.ScheduleID.String()).
			Str("cloned_id", cloned.ID.String()).
			Str("target_agent_id", targetAgentID.String()).
			Msg("schedule cloned to agent")
	}

	c.JSON(http.StatusCreated, BulkCloneResponse{
		Schedules: clonedSchedules,
		Errors:    errors,
	})
}
