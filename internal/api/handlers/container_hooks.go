package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ContainerHookStore defines the interface for container hook persistence operations.
type ContainerHookStore interface {
	GetContainerBackupHooksByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.ContainerBackupHook, error)
	GetContainerBackupHookByID(ctx context.Context, id uuid.UUID) (*models.ContainerBackupHook, error)
	CreateContainerBackupHook(ctx context.Context, hook *models.ContainerBackupHook) error
	UpdateContainerBackupHook(ctx context.Context, hook *models.ContainerBackupHook) error
	DeleteContainerBackupHook(ctx context.Context, id uuid.UUID) error
	GetContainerHookExecutionsByBackupID(ctx context.Context, backupID uuid.UUID) ([]*models.ContainerHookExecution, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// ContainerHooksHandler handles container hook-related HTTP endpoints.
type ContainerHooksHandler struct {
	store  ContainerHookStore
	logger zerolog.Logger
}

// NewContainerHooksHandler creates a new ContainerHooksHandler.
func NewContainerHooksHandler(store ContainerHookStore, logger zerolog.Logger) *ContainerHooksHandler {
	return &ContainerHooksHandler{
		store:  store,
		logger: logger.With().Str("component", "container_hooks_handler").Logger(),
	}
}

// RegisterRoutes registers container hook routes on the given router group.
func (h *ContainerHooksHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Hooks are managed under schedules
	r.GET("/schedules/:id/container-hooks", h.List)
	r.POST("/schedules/:id/container-hooks", h.Create)
	r.GET("/schedules/:id/container-hooks/:hook_id", h.Get)
	r.PUT("/schedules/:id/container-hooks/:hook_id", h.Update)
	r.DELETE("/schedules/:id/container-hooks/:hook_id", h.Delete)

	// Templates endpoint
	r.GET("/container-hook-templates", h.ListTemplates)

	// Executions by backup
	r.GET("/backups/:id/container-hook-executions", h.ListExecutions)
}

// List returns all container backup hooks for a schedule.
// GET /api/v1/schedules/:schedule_id/container-hooks
func (h *ContainerHooksHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	hooks, err := h.store.GetContainerBackupHooksByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to list container hooks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list container hooks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"hooks": hooks})
}

// Get returns a specific container backup hook by ID.
// GET /api/v1/schedules/:schedule_id/container-hooks/:id
func (h *ContainerHooksHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	idParam := c.Param("hook_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hook ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	hook, err := h.store.GetContainerBackupHookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	// Verify hook belongs to the schedule
	if hook.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	c.JSON(http.StatusOK, hook)
}

// Create creates a new container backup hook.
// POST /api/v1/schedules/:schedule_id/container-hooks
func (h *ContainerHooksHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	var req models.CreateContainerBackupHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate hook type
	if !req.Type.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hook type"})
		return
	}

	// Validate template if provided
	if req.Template != "" && !req.Template.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template"})
		return
	}

	// Validate that either command or template is provided
	if req.Command == "" && (req.Template == "" || req.Template == models.ContainerHookTemplateNone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either command or template must be provided"})
		return
	}

	// Validate template variables if using a template
	if req.Template != "" && req.Template != models.ContainerHookTemplateNone {
		if err := docker.ValidateTemplateVars(req.Template, req.TemplateVars); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Create hook
	hook := models.NewContainerBackupHook(scheduleID, req.ContainerName, req.Type, req.Command)
	hook.Template = req.Template
	hook.WorkingDir = req.WorkingDir
	hook.User = req.User
	hook.Description = req.Description
	hook.TemplateVars = req.TemplateVars

	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 1 || *req.TimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timeout must be between 1 and 3600 seconds"})
			return
		}
		hook.TimeoutSeconds = *req.TimeoutSeconds
	}

	if req.FailOnError != nil {
		hook.FailOnError = *req.FailOnError
	}

	if req.Enabled != nil {
		hook.Enabled = *req.Enabled
	}

	// If using template, get the command from template
	if hook.Template != "" && hook.Template != models.ContainerHookTemplateNone && hook.Command == "" {
		hook.Command = docker.GetTemplateCommand(hook.Template, hook.Type, hook.TemplateVars)
	}

	if err := h.store.CreateContainerBackupHook(c.Request.Context(), hook); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to create container hook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create container hook"})
		return
	}

	h.logger.Info().
		Str("hook_id", hook.ID.String()).
		Str("schedule_id", scheduleID.String()).
		Str("container", hook.ContainerName).
		Str("type", string(hook.Type)).
		Msg("container hook created")

	c.JSON(http.StatusCreated, hook)
}

// Update updates an existing container backup hook.
// PUT /api/v1/schedules/:schedule_id/container-hooks/:id
func (h *ContainerHooksHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	idParam := c.Param("hook_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hook ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	hook, err := h.store.GetContainerBackupHookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	// Verify hook belongs to the schedule
	if hook.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	var req models.UpdateContainerBackupHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Update fields
	if req.ContainerName != nil {
		hook.ContainerName = *req.ContainerName
	}
	if req.Command != nil {
		hook.Command = *req.Command
	}
	if req.WorkingDir != nil {
		hook.WorkingDir = *req.WorkingDir
	}
	if req.User != nil {
		hook.User = *req.User
	}
	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 1 || *req.TimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timeout must be between 1 and 3600 seconds"})
			return
		}
		hook.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.FailOnError != nil {
		hook.FailOnError = *req.FailOnError
	}
	if req.Enabled != nil {
		hook.Enabled = *req.Enabled
	}
	if req.Description != nil {
		hook.Description = *req.Description
	}
	if req.TemplateVars != nil {
		hook.TemplateVars = req.TemplateVars
		// Regenerate command from template if using one
		if hook.Template != "" && hook.Template != models.ContainerHookTemplateNone {
			hook.Command = docker.GetTemplateCommand(hook.Template, hook.Type, hook.TemplateVars)
		}
	}

	if err := h.store.UpdateContainerBackupHook(c.Request.Context(), hook); err != nil {
		h.logger.Error().Err(err).Str("hook_id", id.String()).Msg("failed to update container hook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update container hook"})
		return
	}

	h.logger.Info().Str("hook_id", id.String()).Msg("container hook updated")
	c.JSON(http.StatusOK, hook)
}

// Delete removes a container backup hook.
// DELETE /api/v1/schedules/:schedule_id/container-hooks/:id
func (h *ContainerHooksHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	idParam := c.Param("hook_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hook ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	hook, err := h.store.GetContainerBackupHookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	// Verify hook belongs to the schedule
	if hook.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "hook not found"})
		return
	}

	if err := h.store.DeleteContainerBackupHook(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("hook_id", id.String()).Msg("failed to delete container hook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete container hook"})
		return
	}

	h.logger.Info().Str("hook_id", id.String()).Msg("container hook deleted")
	c.JSON(http.StatusOK, gin.H{"message": "hook deleted"})
}

// ListTemplates returns all available hook templates.
// GET /api/v1/container-hook-templates
func (h *ContainerHooksHandler) ListTemplates(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	templates := docker.ListTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// ListExecutions returns all hook executions for a backup.
// GET /api/v1/backups/:backup_id/container-hook-executions
func (h *ContainerHooksHandler) ListExecutions(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backupIDParam := c.Param("id")
	backupID, err := uuid.Parse(backupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	executions, err := h.store.GetContainerHookExecutionsByBackupID(c.Request.Context(), backupID)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to list container hook executions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list container hook executions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"executions": executions})
}

// verifyScheduleAccess checks if the user has access to the schedule.
func (h *ContainerHooksHandler) verifyScheduleAccess(c *gin.Context, orgID, scheduleID uuid.UUID) error {
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), scheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

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
