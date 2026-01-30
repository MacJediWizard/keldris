package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/hooks"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BackupHookTemplateStore defines the interface for backup hook template persistence operations.
type BackupHookTemplateStore interface {
	GetBackupHookTemplatesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.BackupHookTemplate, error)
	GetBackupHookTemplateByID(ctx context.Context, id uuid.UUID) (*models.BackupHookTemplate, error)
	GetBackupHookTemplatesByServiceType(ctx context.Context, orgID uuid.UUID, serviceType string) ([]*models.BackupHookTemplate, error)
	GetBackupHookTemplatesByVisibility(ctx context.Context, orgID uuid.UUID, visibility models.BackupHookTemplateVisibility) ([]*models.BackupHookTemplate, error)
	CreateBackupHookTemplate(ctx context.Context, template *models.BackupHookTemplate) error
	UpdateBackupHookTemplate(ctx context.Context, template *models.BackupHookTemplate) error
	DeleteBackupHookTemplate(ctx context.Context, id uuid.UUID) error
	IncrementBackupHookTemplateUsage(ctx context.Context, id uuid.UUID) error
	// For applying templates to schedules
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	CreateBackupScript(ctx context.Context, script *models.BackupScript) error
	GetBackupScriptByScheduleAndType(ctx context.Context, scheduleID uuid.UUID, scriptType models.BackupScriptType) (*models.BackupScript, error)
	UpdateBackupScript(ctx context.Context, script *models.BackupScript) error
}

// BackupHookTemplatesHandler handles backup hook template-related HTTP endpoints.
type BackupHookTemplatesHandler struct {
	store  BackupHookTemplateStore
	logger zerolog.Logger
}

// NewBackupHookTemplatesHandler creates a new BackupHookTemplatesHandler.
func NewBackupHookTemplatesHandler(store BackupHookTemplateStore, logger zerolog.Logger) *BackupHookTemplatesHandler {
	return &BackupHookTemplatesHandler{
		store:  store,
		logger: logger.With().Str("component", "backup_hook_templates_handler").Logger(),
	}
}

// RegisterRoutes registers backup hook template routes on the given router group.
func (h *BackupHookTemplatesHandler) RegisterRoutes(r *gin.RouterGroup) {
	templates := r.Group("/backup-hook-templates")
	{
		templates.GET("", h.List)
		templates.GET("/built-in", h.ListBuiltIn)
		templates.POST("", h.Create)
		templates.GET("/:id", h.Get)
		templates.PUT("/:id", h.Update)
		templates.DELETE("/:id", h.Delete)
		templates.POST("/:id/apply", h.Apply)
	}
}

// List returns all backup hook templates accessible to the user's organization.
// GET /api/v1/backup-hook-templates
func (h *BackupHookTemplatesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Load built-in templates first
	builtInTemplates, err := hooks.LoadBuiltInTemplates()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load built-in templates")
		// Continue without built-in templates
		builtInTemplates = []*models.BackupHookTemplate{}
	}

	// Get custom templates from database
	customTemplates, err := h.store.GetBackupHookTemplatesByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list backup hook templates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backup hook templates"})
		return
	}

	// Combine templates - built-in first, then custom
	allTemplates := append(builtInTemplates, customTemplates...)

	// Apply filters if provided
	serviceType := c.Query("service_type")
	visibility := c.Query("visibility")
	tag := c.Query("tag")

	var filtered []*models.BackupHookTemplate
	for _, t := range allTemplates {
		if serviceType != "" && t.ServiceType != serviceType {
			continue
		}
		if visibility != "" && string(t.Visibility) != visibility {
			continue
		}
		if tag != "" {
			hasTag := false
			for _, tt := range t.Tags {
				if strings.EqualFold(tt, tag) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	c.JSON(http.StatusOK, models.BackupHookTemplatesResponse{Templates: filtered})
}

// ListBuiltIn returns all built-in templates.
// GET /api/v1/backup-hook-templates/built-in
func (h *BackupHookTemplatesHandler) ListBuiltIn(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	templates, err := hooks.LoadBuiltInTemplates()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to load built-in templates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load built-in templates"})
		return
	}

	c.JSON(http.StatusOK, models.BackupHookTemplatesResponse{Templates: templates})
}

// Get returns a specific backup hook template by ID.
// GET /api/v1/backup-hook-templates/:id
func (h *BackupHookTemplatesHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	// Try to find in built-in templates first
	builtIn, err := hooks.GetBuiltInTemplateByID(id)
	if err == nil {
		c.JSON(http.StatusOK, builtIn)
		return
	}

	// Try database
	template, err := h.store.GetBackupHookTemplateByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Verify access
	if template.Visibility != models.BackupHookTemplateVisibilityBuiltIn {
		if template.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		if template.Visibility == models.BackupHookTemplateVisibilityPrivate && template.CreatedByID != user.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
	}

	c.JSON(http.StatusOK, template)
}

// Create creates a new custom backup hook template.
// POST /api/v1/backup-hook-templates
func (h *BackupHookTemplatesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateBackupHookTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	template := models.NewBackupHookTemplate(user.CurrentOrgID, user.ID, req.Name, req.ServiceType)
	template.Description = req.Description
	template.Icon = req.Icon
	template.Tags = req.Tags
	template.Variables = req.Variables
	template.Scripts = req.Scripts

	// Set visibility (default to private)
	if req.Visibility != "" {
		template.Visibility = req.Visibility
		if !template.IsValidVisibility() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility"})
			return
		}
		// Cannot create built-in templates via API
		if template.Visibility == models.BackupHookTemplateVisibilityBuiltIn {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot create built-in templates"})
			return
		}
	}

	if err := h.store.CreateBackupHookTemplate(c.Request.Context(), template); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create backup hook template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup hook template"})
		return
	}

	h.logger.Info().
		Str("template_id", template.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("name", template.Name).
		Msg("backup hook template created")

	c.JSON(http.StatusCreated, template)
}

// Update updates an existing backup hook template.
// PUT /api/v1/backup-hook-templates/:id
func (h *BackupHookTemplatesHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	// Check if it's a built-in template
	if _, err := hooks.GetBuiltInTemplateByID(id); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify built-in templates"})
		return
	}

	template, err := h.store.GetBackupHookTemplateByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Verify ownership
	if template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	if template.Visibility == models.BackupHookTemplateVisibilityPrivate && template.CreatedByID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only modify your own private templates"})
		return
	}

	var req models.UpdateBackupHookTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Apply updates
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.ServiceType != nil {
		template.ServiceType = *req.ServiceType
	}
	if req.Icon != nil {
		template.Icon = *req.Icon
	}
	if req.Tags != nil {
		template.Tags = req.Tags
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.Scripts != nil {
		template.Scripts = *req.Scripts
	}
	if req.Visibility != nil {
		// Cannot change to built-in
		if *req.Visibility == models.BackupHookTemplateVisibilityBuiltIn {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot set visibility to built-in"})
			return
		}
		template.Visibility = *req.Visibility
	}

	if err := h.store.UpdateBackupHookTemplate(c.Request.Context(), template); err != nil {
		h.logger.Error().Err(err).Str("template_id", id.String()).Msg("failed to update backup hook template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update backup hook template"})
		return
	}

	h.logger.Info().Str("template_id", id.String()).Msg("backup hook template updated")
	c.JSON(http.StatusOK, template)
}

// Delete removes a backup hook template.
// DELETE /api/v1/backup-hook-templates/:id
func (h *BackupHookTemplatesHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	// Check if it's a built-in template
	if _, err := hooks.GetBuiltInTemplateByID(id); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete built-in templates"})
		return
	}

	template, err := h.store.GetBackupHookTemplateByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Verify ownership
	if template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	if template.Visibility == models.BackupHookTemplateVisibilityPrivate && template.CreatedByID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own private templates"})
		return
	}

	if err := h.store.DeleteBackupHookTemplate(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("template_id", id.String()).Msg("failed to delete backup hook template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup hook template"})
		return
	}

	h.logger.Info().Str("template_id", id.String()).Msg("backup hook template deleted")
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// Apply applies a backup hook template to a schedule, creating backup scripts.
// POST /api/v1/backup-hook-templates/:id/apply
func (h *BackupHookTemplatesHandler) Apply(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	var req models.ApplyBackupHookTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get the template (try built-in first)
	var template *models.BackupHookTemplate
	builtIn, err := hooks.GetBuiltInTemplateByID(id)
	if err == nil {
		template = builtIn
	} else {
		template, err = h.store.GetBackupHookTemplateByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}

		// Verify access for custom templates
		if template.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		if template.Visibility == models.BackupHookTemplateVisibilityPrivate && template.CreatedByID != user.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
	}

	// Verify schedule access
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), req.ScheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Create or update backup scripts from template
	var createdScripts []*models.BackupScript

	scriptTypes := []struct {
		scriptType models.BackupScriptType
		script     *models.BackupHookTemplateScript
	}{
		{models.BackupScriptTypePreBackup, template.Scripts.PreBackup},
		{models.BackupScriptTypePostSuccess, template.Scripts.PostSuccess},
		{models.BackupScriptTypePostFailure, template.Scripts.PostFailure},
		{models.BackupScriptTypePostAlways, template.Scripts.PostAlways},
	}

	for _, st := range scriptTypes {
		if st.script == nil {
			continue
		}

		// Render script with variables
		renderedScript := hooks.RenderScript(st.script.Script, template.Variables, req.VariableValues)

		// Check if script already exists for this schedule and type
		existing, err := h.store.GetBackupScriptByScheduleAndType(c.Request.Context(), req.ScheduleID, st.scriptType)
		if err == nil && existing != nil {
			// Update existing script
			existing.Script = renderedScript
			existing.TimeoutSeconds = st.script.TimeoutSeconds
			existing.FailOnError = st.script.FailOnError
			existing.UpdatedAt = time.Now()

			if err := h.store.UpdateBackupScript(c.Request.Context(), existing); err != nil {
				h.logger.Error().Err(err).
					Str("schedule_id", req.ScheduleID.String()).
					Str("script_type", string(st.scriptType)).
					Msg("failed to update backup script")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply template"})
				return
			}
			createdScripts = append(createdScripts, existing)
		} else {
			// Create new script
			script := models.NewBackupScript(req.ScheduleID, st.scriptType, renderedScript)
			script.TimeoutSeconds = st.script.TimeoutSeconds
			script.FailOnError = st.script.FailOnError

			if err := h.store.CreateBackupScript(c.Request.Context(), script); err != nil {
				h.logger.Error().Err(err).
					Str("schedule_id", req.ScheduleID.String()).
					Str("script_type", string(st.scriptType)).
					Msg("failed to create backup script")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply template"})
				return
			}
			createdScripts = append(createdScripts, script)
		}
	}

	// Increment usage count for the template (only for custom templates)
	if template.Visibility != models.BackupHookTemplateVisibilityBuiltIn {
		_ = h.store.IncrementBackupHookTemplateUsage(c.Request.Context(), id)
	}

	h.logger.Info().
		Str("template_id", id.String()).
		Str("schedule_id", req.ScheduleID.String()).
		Int("scripts_created", len(createdScripts)).
		Msg("backup hook template applied")

	c.JSON(http.StatusOK, models.ApplyBackupHookTemplateResponse{
		Scripts: createdScripts,
		Message: "Template applied successfully",
	})
}
