package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/export"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ConfigExportStore defines the interface for config export/import persistence operations.
type ConfigExportStore interface {
	// Agent operations
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	CreateAgent(ctx context.Context, agent *models.Agent) error

	// Schedule operations
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	UpdateSchedule(ctx context.Context, schedule *models.Schedule) error
	SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error

	// Repository operations
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)

	// Template operations
	CreateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error
	GetConfigTemplateByID(ctx context.Context, id uuid.UUID) (*models.ConfigTemplate, error)
	GetConfigTemplatesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ConfigTemplate, error)
	GetPublicConfigTemplates(ctx context.Context) ([]*models.ConfigTemplate, error)
	UpdateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error
	DeleteConfigTemplate(ctx context.Context, id uuid.UUID) error
	IncrementTemplateUsageCount(ctx context.Context, id uuid.UUID) error
}

// ConfigExportHandler handles configuration export/import HTTP endpoints.
type ConfigExportHandler struct {
	store    ConfigExportStore
	exporter *export.Exporter
	importer *export.Importer
	logger   zerolog.Logger
}

// NewConfigExportHandler creates a new ConfigExportHandler.
func NewConfigExportHandler(store ConfigExportStore, logger zerolog.Logger) *ConfigExportHandler {
	return &ConfigExportHandler{
		store:    store,
		exporter: export.NewExporter(store, logger),
		importer: export.NewImporter(store, logger),
		logger:   logger.With().Str("component", "config_export_handler").Logger(),
	}
}

// RegisterRoutes registers config export/import routes on the given router group.
func (h *ConfigExportHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Export routes
	exportRoutes := r.Group("/export")
	{
		exportRoutes.GET("/agents/:id", h.ExportAgent)
		exportRoutes.GET("/schedules/:id", h.ExportSchedule)
		exportRoutes.GET("/repositories/:id", h.ExportRepository)
		exportRoutes.POST("/bundle", h.ExportBundle)
	}

	// Import routes
	r.POST("/import", h.Import)
	r.POST("/import/validate", h.ValidateImport)

	// Template routes
	templates := r.Group("/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.POST("", h.CreateTemplate)
		templates.GET("/:id", h.GetTemplate)
		templates.PUT("/:id", h.UpdateTemplate)
		templates.DELETE("/:id", h.DeleteTemplate)
		templates.POST("/:id/use", h.UseTemplate)
	}
}

// ExportAgentResponse is the response for exporting an agent.
type ExportAgentResponse struct {
	Format string `json:"format"`
	Config string `json:"config"`
}

// ExportAgent exports an agent configuration.
// GET /api/v1/export/agents/:id
func (h *ConfigExportHandler) ExportAgent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	agentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
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

	// Get format from query param
	format := export.Format(c.DefaultQuery("format", "json"))
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	opts := export.ExportOptions{
		Format:     format,
		ExportedBy: user.Email,
	}

	data, err := h.exporter.ExportAgent(c.Request.Context(), agentID, opts)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to export agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export agent"})
		return
	}

	// Set appropriate content type and return
	if format == export.FormatYAML {
		c.Header("Content-Type", "application/x-yaml")
		c.Header("Content-Disposition", "attachment; filename=agent-"+agent.Hostname+".yaml")
	} else {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=agent-"+agent.Hostname+".json")
	}

	c.String(http.StatusOK, string(data))
}

// ExportSchedule exports a schedule configuration.
// GET /api/v1/export/schedules/:id
func (h *ConfigExportHandler) ExportSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	scheduleID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule belongs to user's org
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

	// Get format from query param
	format := export.Format(c.DefaultQuery("format", "json"))
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	opts := export.ExportOptions{
		Format:     format,
		ExportedBy: user.Email,
	}

	data, err := h.exporter.ExportSchedule(c.Request.Context(), scheduleID, opts)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to export schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export schedule"})
		return
	}

	// Set appropriate content type and return
	if format == export.FormatYAML {
		c.Header("Content-Type", "application/x-yaml")
		c.Header("Content-Disposition", "attachment; filename=schedule-"+schedule.Name+".yaml")
	} else {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=schedule-"+schedule.Name+".json")
	}

	c.String(http.StatusOK, string(data))
}

// ExportRepository exports a repository configuration (without secrets).
// GET /api/v1/export/repositories/:id
func (h *ConfigExportHandler) ExportRepository(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify repository belongs to user's org
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}
	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Get format from query param
	format := export.Format(c.DefaultQuery("format", "json"))
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	opts := export.ExportOptions{
		Format:     format,
		ExportedBy: user.Email,
	}

	data, err := h.exporter.ExportRepository(c.Request.Context(), repoID, opts)
	if err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to export repository")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export repository"})
		return
	}

	// Set appropriate content type and return
	if format == export.FormatYAML {
		c.Header("Content-Type", "application/x-yaml")
		c.Header("Content-Disposition", "attachment; filename=repository-"+repo.Name+".yaml")
	} else {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=repository-"+repo.Name+".json")
	}

	c.String(http.StatusOK, string(data))
}

// ExportBundleRequest is the request body for exporting a bundle.
type ExportBundleRequest struct {
	AgentIDs      []string `json:"agent_ids,omitempty"`
	ScheduleIDs   []string `json:"schedule_ids,omitempty"`
	RepositoryIDs []string `json:"repository_ids,omitempty"`
	Format        string   `json:"format,omitempty"`
	Description   string   `json:"description,omitempty"`
}

// ExportBundle exports multiple configurations as a bundle.
// POST /api/v1/export/bundle
func (h *ConfigExportHandler) ExportBundle(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req ExportBundleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse and validate IDs
	var agentIDs, scheduleIDs, repositoryIDs []uuid.UUID

	for _, idStr := range req.AgentIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID: " + idStr})
			return
		}
		// Verify ownership
		agent, err := h.store.GetAgentByID(c.Request.Context(), id)
		if err != nil || agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found: " + idStr})
			return
		}
		agentIDs = append(agentIDs, id)
	}

	for _, idStr := range req.ScheduleIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID: " + idStr})
			return
		}
		schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found: " + idStr})
			return
		}
		agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
		if err != nil || agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found: " + idStr})
			return
		}
		scheduleIDs = append(scheduleIDs, id)
	}

	for _, idStr := range req.RepositoryIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID: " + idStr})
			return
		}
		repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
		if err != nil || repo.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "repository not found: " + idStr})
			return
		}
		repositoryIDs = append(repositoryIDs, id)
	}

	if len(agentIDs) == 0 && len(scheduleIDs) == 0 && len(repositoryIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one ID must be provided"})
		return
	}

	format := export.Format(req.Format)
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	opts := export.ExportOptions{
		Format:      format,
		ExportedBy:  user.Email,
		Description: req.Description,
	}

	data, err := h.exporter.ExportBundle(c.Request.Context(), agentIDs, scheduleIDs, repositoryIDs, opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to export bundle")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export bundle"})
		return
	}

	// Set appropriate content type and return
	if format == export.FormatYAML {
		c.Header("Content-Type", "application/x-yaml")
		c.Header("Content-Disposition", "attachment; filename=keldris-bundle.yaml")
	} else {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=keldris-bundle.json")
	}

	c.String(http.StatusOK, string(data))
}

// ImportConfigRequest is the request body for importing a configuration.
type ImportConfigRequest struct {
	Config             string            `json:"config" binding:"required"`
	Format             string            `json:"format,omitempty"`
	TargetAgentID      string            `json:"target_agent_id,omitempty"`
	RepositoryMappings map[string]string `json:"repository_mappings,omitempty"`
	ConflictResolution string            `json:"conflict_resolution,omitempty"`
}

// Import imports a configuration.
// POST /api/v1/import
func (h *ConfigExportHandler) Import(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ImportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	format := export.Format(req.Format)
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	// Parse the config
	config, configType, err := h.importer.ParseConfig([]byte(req.Config), format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse config: " + err.Error()})
		return
	}

	// Build import request
	importReq := export.ImportRequest{
		Config:             []byte(req.Config),
		Format:             format,
		TargetAgentID:      req.TargetAgentID,
		RepositoryMappings: req.RepositoryMappings,
		ConflictResolution: export.ConflictResolution(req.ConflictResolution),
	}

	if importReq.ConflictResolution == "" {
		importReq.ConflictResolution = export.ConflictResolutionSkip
	}

	// Perform import
	result, err := h.importer.Import(c.Request.Context(), user.CurrentOrgID, config, configType, importReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to import config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import config: " + err.Error()})
		return
	}

	h.logger.Info().
		Bool("success", result.Success).
		Int("agents_imported", result.Imported.AgentCount).
		Int("schedules_imported", result.Imported.ScheduleCount).
		Int("repositories_imported", result.Imported.RepositoryCount).
		Msg("config import completed")

	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusUnprocessableEntity, result)
	}
}

// ValidateImportRequest is the request body for validating an import.
type ValidateImportRequest struct {
	Config string `json:"config" binding:"required"`
	Format string `json:"format,omitempty"`
}

// ValidateImport validates an import configuration without performing it.
// POST /api/v1/import/validate
func (h *ConfigExportHandler) ValidateImport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ValidateImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	format := export.Format(req.Format)
	if format != export.FormatJSON && format != export.FormatYAML {
		format = export.FormatJSON
	}

	// Parse the config
	config, configType, err := h.importer.ParseConfig([]byte(req.Config), format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse config: " + err.Error()})
		return
	}

	// Validate import
	result, err := h.importer.ValidateImport(c.Request.Context(), user.CurrentOrgID, config, configType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateTemplateRequest is the request body for creating a template.
type CreateTemplateRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=255"`
	Description string   `json:"description,omitempty"`
	Config      string   `json:"config" binding:"required"`
	Visibility  string   `json:"visibility,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ListTemplates returns all templates accessible to the user.
// GET /api/v1/templates
func (h *ConfigExportHandler) ListTemplates(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get org templates
	orgTemplates, err := h.store.GetConfigTemplatesByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list org templates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list templates"})
		return
	}

	// Get public templates
	publicTemplates, err := h.store.GetPublicConfigTemplates(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list public templates")
		// Continue with just org templates
		publicTemplates = []*models.ConfigTemplate{}
	}

	// Convert to response format
	var templates []*models.ConfigTemplateWithConfig
	seen := make(map[uuid.UUID]bool)

	for _, t := range orgTemplates {
		if seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		tc, err := t.ToConfigTemplateWithConfig()
		if err != nil {
			h.logger.Warn().Err(err).Str("template_id", t.ID.String()).Msg("failed to parse template config")
			continue
		}
		templates = append(templates, tc)
	}

	for _, t := range publicTemplates {
		if seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		tc, err := t.ToConfigTemplateWithConfig()
		if err != nil {
			h.logger.Warn().Err(err).Str("template_id", t.ID.String()).Msg("failed to parse template config")
			continue
		}
		templates = append(templates, tc)
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// CreateTemplate creates a new template from an exported configuration.
// POST /api/v1/templates
func (h *ConfigExportHandler) CreateTemplate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse config to determine type
	_, configType, err := h.importer.ParseConfig([]byte(req.Config), export.FormatJSON)
	if err != nil {
		// Try YAML
		_, configType, err = h.importer.ParseConfig([]byte(req.Config), export.FormatYAML)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse config: " + err.Error()})
			return
		}
	}

	// Map config type to template type
	var templateType models.TemplateType
	switch configType {
	case export.ConfigTypeAgent:
		templateType = models.TemplateTypeAgent
	case export.ConfigTypeSchedule:
		templateType = models.TemplateTypeSchedule
	case export.ConfigTypeRepository:
		templateType = models.TemplateTypeRepository
	case export.ConfigTypeBundle:
		templateType = models.TemplateTypeBundle
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown config type"})
		return
	}

	// Create template
	template := models.NewConfigTemplate(user.CurrentOrgID, user.ID, req.Name, templateType, []byte(req.Config))
	template.Description = req.Description
	template.Tags = req.Tags

	if req.Visibility != "" {
		template.Visibility = models.TemplateVisibility(req.Visibility)
		if !template.IsValidVisibility() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility", "valid_values": models.ValidTemplateVisibilities()})
			return
		}
	}

	if err := h.store.CreateConfigTemplate(c.Request.Context(), template); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create template"})
		return
	}

	h.logger.Info().
		Str("template_id", template.ID.String()).
		Str("name", req.Name).
		Str("type", string(templateType)).
		Msg("template created")

	tc, _ := template.ToConfigTemplateWithConfig()
	c.JSON(http.StatusCreated, tc)
}

// GetTemplate returns a specific template by ID.
// GET /api/v1/templates/:id
func (h *ConfigExportHandler) GetTemplate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	template, err := h.store.GetConfigTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Check access
	if template.Visibility != models.TemplateVisibilityPublic &&
		template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	tc, err := template.ToConfigTemplateWithConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse template config"})
		return
	}

	c.JSON(http.StatusOK, tc)
}

// UpdateTemplateRequest is the request body for updating a template.
type UpdateTemplateRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateTemplate updates a template.
// PUT /api/v1/templates/:id
func (h *ConfigExportHandler) UpdateTemplate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	template, err := h.store.GetConfigTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Only org members can update org templates
	if template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify template from another organization"})
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Visibility != "" {
		template.Visibility = models.TemplateVisibility(req.Visibility)
		if !template.IsValidVisibility() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visibility"})
			return
		}
	}
	if req.Tags != nil {
		template.Tags = req.Tags
	}
	template.UpdatedAt = time.Now()

	if err := h.store.UpdateConfigTemplate(c.Request.Context(), template); err != nil {
		h.logger.Error().Err(err).Str("template_id", templateID.String()).Msg("failed to update template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update template"})
		return
	}

	tc, _ := template.ToConfigTemplateWithConfig()
	c.JSON(http.StatusOK, tc)
}

// DeleteTemplate deletes a template.
// DELETE /api/v1/templates/:id
func (h *ConfigExportHandler) DeleteTemplate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	template, err := h.store.GetConfigTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Only org members can delete org templates
	if template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete template from another organization"})
		return
	}

	if err := h.store.DeleteConfigTemplate(c.Request.Context(), templateID); err != nil {
		h.logger.Error().Err(err).Str("template_id", templateID.String()).Msg("failed to delete template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete template"})
		return
	}

	h.logger.Info().Str("template_id", templateID.String()).Msg("template deleted")
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// UseTemplateRequest is the request body for using a template.
type UseTemplateRequest struct {
	TargetAgentID      string            `json:"target_agent_id,omitempty"`
	RepositoryMappings map[string]string `json:"repository_mappings,omitempty"`
	ConflictResolution string            `json:"conflict_resolution,omitempty"`
}

// UseTemplate imports a template configuration.
// POST /api/v1/templates/:id/use
func (h *ConfigExportHandler) UseTemplate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template ID"})
		return
	}

	template, err := h.store.GetConfigTemplateByID(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Check access
	if template.Visibility != models.TemplateVisibilityPublic &&
		template.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	var req UseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = UseTemplateRequest{}
	}

	// Parse the config
	config, configType, err := h.importer.ParseConfig(template.Config, export.FormatJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse template config: " + err.Error()})
		return
	}

	// Build import request
	importReq := export.ImportRequest{
		Config:             template.Config,
		Format:             export.FormatJSON,
		TargetAgentID:      req.TargetAgentID,
		RepositoryMappings: req.RepositoryMappings,
		ConflictResolution: export.ConflictResolution(req.ConflictResolution),
	}

	if importReq.ConflictResolution == "" {
		importReq.ConflictResolution = export.ConflictResolutionSkip
	}

	// Perform import
	result, err := h.importer.Import(c.Request.Context(), user.CurrentOrgID, config, configType, importReq)
	if err != nil {
		h.logger.Error().Err(err).Str("template_id", templateID.String()).Msg("failed to use template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to use template: " + err.Error()})
		return
	}

	// Increment usage count
	if err := h.store.IncrementTemplateUsageCount(c.Request.Context(), templateID); err != nil {
		h.logger.Warn().Err(err).Str("template_id", templateID.String()).Msg("failed to increment usage count")
	}

	h.logger.Info().
		Str("template_id", templateID.String()).
		Bool("success", result.Success).
		Msg("template used")

	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusUnprocessableEntity, result)
	}
}

// toTemplateResponse converts template for JSON response (internal helper).
func toTemplateResponse(template *models.ConfigTemplate) map[string]any {
	var config map[string]any
	if len(template.Config) > 0 {
		json.Unmarshal(template.Config, &config)
	}

	return map[string]any{
		"id":          template.ID,
		"org_id":      template.OrgID,
		"created_by":  template.CreatedByID,
		"name":        template.Name,
		"description": template.Description,
		"type":        template.Type,
		"visibility":  template.Visibility,
		"tags":        template.Tags,
		"config":      config,
		"usage_count": template.UsageCount,
		"created_at":  template.CreatedAt,
		"updated_at":  template.UpdatedAt,
	}
}
