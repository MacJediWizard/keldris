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

// BackupScriptStore defines the interface for backup script persistence operations.
type BackupScriptStore interface {
	GetBackupScriptsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupScript, error)
	GetBackupScriptByID(ctx context.Context, id uuid.UUID) (*models.BackupScript, error)
	CreateBackupScript(ctx context.Context, script *models.BackupScript) error
	UpdateBackupScript(ctx context.Context, script *models.BackupScript) error
	DeleteBackupScript(ctx context.Context, id uuid.UUID) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// BackupScriptsHandler handles backup script-related HTTP endpoints.
type BackupScriptsHandler struct {
	store  BackupScriptStore
	logger zerolog.Logger
}

// NewBackupScriptsHandler creates a new BackupScriptsHandler.
func NewBackupScriptsHandler(store BackupScriptStore, logger zerolog.Logger) *BackupScriptsHandler {
	return &BackupScriptsHandler{
		store:  store,
		logger: logger.With().Str("component", "backup_scripts_handler").Logger(),
	}
}

// RegisterRoutes registers backup script routes on the given router group.
func (h *BackupScriptsHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Scripts are managed under schedules
	r.GET("/schedules/:id/scripts", h.List)
	r.POST("/schedules/:id/scripts", h.Create)
	r.GET("/schedules/:id/scripts/:script_id", h.Get)
	r.PUT("/schedules/:id/scripts/:script_id", h.Update)
	r.DELETE("/schedules/:id/scripts/:script_id", h.Delete)
}

// List returns all backup scripts for a schedule.
// GET /api/v1/schedules/:id/scripts
	r.GET("/schedules/:schedule_id/scripts", h.List)
	r.POST("/schedules/:schedule_id/scripts", h.Create)
	r.GET("/schedules/:schedule_id/scripts/:id", h.Get)
	r.PUT("/schedules/:schedule_id/scripts/:id", h.Update)
	r.DELETE("/schedules/:schedule_id/scripts/:id", h.Delete)
}

// List returns all backup scripts for a schedule.
// GET /api/v1/schedules/:schedule_id/scripts
func (h *BackupScriptsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	scripts, err := h.store.GetBackupScriptsByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to list backup scripts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backup scripts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"scripts": scripts})
}

// Get returns a specific backup script by ID.
// GET /api/v1/schedules/:id/scripts/:script_id
// GET /api/v1/schedules/:schedule_id/scripts/:id
func (h *BackupScriptsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	scriptIDParam := c.Param("script_id")
	id, err := uuid.Parse(scriptIDParam)
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid script ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	script, err := h.store.GetBackupScriptByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	// Verify script belongs to the schedule
	if script.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	c.JSON(http.StatusOK, script)
}

// Create creates a new backup script.
// POST /api/v1/schedules/:id/scripts
// POST /api/v1/schedules/:schedule_id/scripts
func (h *BackupScriptsHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	var req models.CreateBackupScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate script type
	if !req.Type.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid script type"})
		return
	}

	script := models.NewBackupScript(scheduleID, req.Type, req.Script)

	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 1 || *req.TimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timeout must be between 1 and 3600 seconds"})
			return
		}
		script.TimeoutSeconds = *req.TimeoutSeconds
	}

	if req.FailOnError != nil {
		script.FailOnError = *req.FailOnError
	}

	if req.Enabled != nil {
		script.Enabled = *req.Enabled
	}

	if err := h.store.CreateBackupScript(c.Request.Context(), script); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to create backup script")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup script"})
		return
	}

	h.logger.Info().
		Str("script_id", script.ID.String()).
		Str("schedule_id", scheduleID.String()).
		Str("type", string(script.Type)).
		Msg("backup script created")

	c.JSON(http.StatusCreated, script)
}

// Update updates an existing backup script.
// PUT /api/v1/schedules/:id/scripts/:script_id
// PUT /api/v1/schedules/:schedule_id/scripts/:id
func (h *BackupScriptsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	scriptIDParam := c.Param("script_id")
	id, err := uuid.Parse(scriptIDParam)
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid script ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	script, err := h.store.GetBackupScriptByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	// Verify script belongs to the schedule
	if script.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	var req models.UpdateBackupScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Update fields
	if req.Script != nil {
		script.Script = *req.Script
	}
	if req.TimeoutSeconds != nil {
		if *req.TimeoutSeconds < 1 || *req.TimeoutSeconds > 3600 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timeout must be between 1 and 3600 seconds"})
			return
		}
		script.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.FailOnError != nil {
		script.FailOnError = *req.FailOnError
	}
	if req.Enabled != nil {
		script.Enabled = *req.Enabled
	}

	if err := h.store.UpdateBackupScript(c.Request.Context(), script); err != nil {
		h.logger.Error().Err(err).Str("script_id", id.String()).Msg("failed to update backup script")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update backup script"})
		return
	}

	h.logger.Info().Str("script_id", id.String()).Msg("backup script updated")
	c.JSON(http.StatusOK, script)
}

// Delete removes a backup script.
// DELETE /api/v1/schedules/:id/scripts/:script_id
// DELETE /api/v1/schedules/:schedule_id/scripts/:id
func (h *BackupScriptsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	scheduleIDParam := c.Param("id")
	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	scriptIDParam := c.Param("script_id")
	id, err := uuid.Parse(scriptIDParam)
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid script ID"})
		return
	}

	// Verify schedule access
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, scheduleID); err != nil {
		return
	}

	script, err := h.store.GetBackupScriptByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	// Verify script belongs to the schedule
	if script.ScheduleID != scheduleID {
		c.JSON(http.StatusNotFound, gin.H{"error": "script not found"})
		return
	}

	if err := h.store.DeleteBackupScript(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("script_id", id.String()).Msg("failed to delete backup script")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup script"})
		return
	}

	h.logger.Info().Str("script_id", id.String()).Msg("backup script deleted")
	c.JSON(http.StatusOK, gin.H{"message": "script deleted"})
}

// verifyScheduleAccess checks if the user has access to the schedule.
func (h *BackupScriptsHandler) verifyScheduleAccess(c *gin.Context, orgID, scheduleID uuid.UUID) error {
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
