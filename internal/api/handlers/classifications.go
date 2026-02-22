package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/classification"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ClassificationStore defines the interface for classification persistence operations.
type ClassificationStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)

	// Path classification rules
	GetPathClassificationRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.PathClassificationRule, error)
	GetPathClassificationRuleByID(ctx context.Context, id uuid.UUID) (*models.PathClassificationRule, error)
	CreatePathClassificationRule(ctx context.Context, rule *models.PathClassificationRule) error
	UpdatePathClassificationRule(ctx context.Context, rule *models.PathClassificationRule) error
	DeletePathClassificationRule(ctx context.Context, id uuid.UUID) error

	// Schedule classifications
	GetScheduleClassification(ctx context.Context, scheduleID uuid.UUID) (*models.ScheduleClassification, error)
	SetScheduleClassification(ctx context.Context, c *models.ScheduleClassification) error
	UpdateScheduleClassificationLevel(ctx context.Context, scheduleID uuid.UUID, level string, dataTypes []string) error
	GetSchedulesByClassificationLevel(ctx context.Context, orgID uuid.UUID, level string) ([]*models.Schedule, error)

	// Backup classifications
	GetBackupClassification(ctx context.Context, backupID uuid.UUID) (*models.BackupClassification, error)
	GetBackupsByClassificationLevel(ctx context.Context, orgID uuid.UUID, level string, limit int) ([]*models.Backup, error)

	// Compliance reports
	GetClassificationSummary(ctx context.Context, orgID uuid.UUID) (*models.ClassificationSummary, error)
}

// ClassificationsHandler handles classification-related HTTP endpoints.
type ClassificationsHandler struct {
	store  ClassificationStore
	logger zerolog.Logger
}

// NewClassificationsHandler creates a new ClassificationsHandler.
func NewClassificationsHandler(store ClassificationStore, logger zerolog.Logger) *ClassificationsHandler {
	return &ClassificationsHandler{
		store:  store,
		logger: logger.With().Str("component", "classifications_handler").Logger(),
	}
}

// RegisterRoutes registers classification routes on the given router group.
func (h *ClassificationsHandler) RegisterRoutes(r *gin.RouterGroup) {
	cls := r.Group("/classifications")
	{
		// Classification levels and types (reference data)
		cls.GET("/levels", h.ListLevels)
		cls.GET("/data-types", h.ListDataTypes)
		cls.GET("/default-rules", h.ListDefaultRules)

		// Path classification rules
		cls.GET("/rules", h.ListRules)
		cls.POST("/rules", h.CreateRule)
		cls.GET("/rules/:id", h.GetRule)
		cls.PUT("/rules/:id", h.UpdateRule)
		cls.DELETE("/rules/:id", h.DeleteRule)

		// Schedule classifications
		cls.GET("/schedules", h.ListScheduleClassifications)
		cls.GET("/schedules/:id", h.GetScheduleClassification)
		cls.PUT("/schedules/:id", h.SetScheduleClassification)
		cls.POST("/schedules/:id/auto-classify", h.AutoClassifySchedule)

		// Backup classifications
		cls.GET("/backups", h.ListBackupsByClassification)

		// Compliance reports
		cls.GET("/summary", h.GetSummary)
		cls.GET("/compliance-report", h.GetComplianceReport)
	}
}

// ListLevels returns all available classification levels.
// GET /api/v1/classifications/levels
func (h *ClassificationsHandler) ListLevels(c *gin.Context) {
	levels := []map[string]interface{}{
		{"value": "public", "label": "Public", "description": "Non-sensitive, publicly shareable data", "priority": 1},
		{"value": "internal", "label": "Internal", "description": "Internal business data with limited access", "priority": 2},
		{"value": "confidential", "label": "Confidential", "description": "Sensitive data requiring protection", "priority": 3},
		{"value": "restricted", "label": "Restricted", "description": "Highly sensitive data with strict access controls", "priority": 4},
	}
	c.JSON(http.StatusOK, gin.H{"levels": levels})
}

// ListDataTypes returns all available data types.
// GET /api/v1/classifications/data-types
func (h *ClassificationsHandler) ListDataTypes(c *gin.Context) {
	dataTypes := []map[string]interface{}{
		{"value": "pii", "label": "PII", "description": "Personally Identifiable Information"},
		{"value": "phi", "label": "PHI", "description": "Protected Health Information (HIPAA)"},
		{"value": "pci", "label": "PCI", "description": "Payment Card Industry data (PCI-DSS)"},
		{"value": "proprietary", "label": "Proprietary", "description": "Proprietary business data"},
		{"value": "general", "label": "General", "description": "General unclassified data"},
	}
	c.JSON(http.StatusOK, gin.H{"data_types": dataTypes})
}

// ListDefaultRules returns the built-in default classification rules.
// GET /api/v1/classifications/default-rules
func (h *ClassificationsHandler) ListDefaultRules(c *gin.Context) {
	rules := classification.DefaultRules()
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// ListRules returns all classification rules for the organization.
// GET /api/v1/classifications/rules
func (h *ClassificationsHandler) ListRules(c *gin.Context) {
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

	rules, err := h.store.GetPathClassificationRulesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list classification rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list classification rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// GetRule returns a specific classification rule.
// GET /api/v1/classifications/rules/:id
func (h *ClassificationsHandler) GetRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetPathClassificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// CreateRule creates a new classification rule.
// POST /api/v1/classifications/rules
func (h *ClassificationsHandler) CreateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreatePathClassificationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate level
	if !classification.ValidateLevel(req.Level) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid classification level"})
		return
	}

	// Validate data types
	dataTypes := make([]classification.DataType, 0)
	if len(req.DataTypes) > 0 {
		for _, dt := range req.DataTypes {
			if !classification.ValidateDataType(dt) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data type: " + dt})
				return
			}
			dataTypes = append(dataTypes, classification.DataType(dt))
		}
	} else {
		dataTypes = []classification.DataType{classification.DataTypeGeneral}
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	rule := models.NewPathClassificationRule(dbUser.OrgID, req.Pattern, classification.Level(req.Level), dataTypes)
	rule.Description = req.Description
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}

	if err := h.store.CreatePathClassificationRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Msg("failed to create classification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create classification rule"})
		return
	}

	h.logger.Info().Str("rule_id", rule.ID.String()).Str("pattern", rule.Pattern).Msg("classification rule created")
	c.JSON(http.StatusCreated, rule)
}

// UpdateRule updates an existing classification rule.
// PUT /api/v1/classifications/rules/:id
func (h *ClassificationsHandler) UpdateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	var req models.UpdatePathClassificationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.store.GetPathClassificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	// Don't allow modifying built-in rules
	if rule.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify built-in rules"})
		return
	}

	// Apply updates
	if req.Pattern != nil {
		rule.Pattern = *req.Pattern
	}
	if req.Level != nil {
		if !classification.ValidateLevel(*req.Level) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid classification level"})
			return
		}
		rule.Level = classification.Level(*req.Level)
	}
	if req.DataTypes != nil {
		dataTypes := make([]classification.DataType, 0)
		for _, dt := range req.DataTypes {
			if !classification.ValidateDataType(dt) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data type: " + dt})
				return
			}
			dataTypes = append(dataTypes, classification.DataType(dt))
		}
		rule.DataTypes = dataTypes
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := h.store.UpdatePathClassificationRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to update classification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update classification rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule deletes a classification rule.
// DELETE /api/v1/classifications/rules/:id
func (h *ClassificationsHandler) DeleteRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetPathClassificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "classification rule not found"})
		return
	}

	// Don't allow deleting built-in rules
	if rule.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete built-in rules"})
		return
	}

	if err := h.store.DeletePathClassificationRule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to delete classification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete classification rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "classification rule deleted"})
}

// ListScheduleClassifications returns schedules with their classifications.
// GET /api/v1/classifications/schedules
func (h *ClassificationsHandler) ListScheduleClassifications(c *gin.Context) {
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

	// Optional filter by level
	level := c.Query("level")
	var schedules []*models.Schedule

	if level != "" {
		if !classification.ValidateLevel(level) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid classification level"})
			return
		}
		schedules, err = h.store.GetSchedulesByClassificationLevel(c.Request.Context(), dbUser.OrgID, level)
	} else {
		schedules, err = h.store.GetSchedulesByOrgID(c.Request.Context(), dbUser.OrgID)
	}

	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// GetScheduleClassification returns the classification for a schedule.
// GET /api/v1/classifications/schedules/:id
func (h *ClassificationsHandler) GetScheduleClassification(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedule_id": schedule.ID,
		"level":       schedule.ClassificationLevel,
		"data_types":  schedule.ClassificationDataTypes,
	})
}

// SetScheduleClassification sets the classification for a schedule.
// PUT /api/v1/classifications/schedules/:id
func (h *ClassificationsHandler) SetScheduleClassification(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req models.SetScheduleClassificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !classification.ValidateLevel(req.Level) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid classification level"})
		return
	}

	// Validate data types
	dataTypes := []string{"general"}
	if len(req.DataTypes) > 0 {
		dataTypes = make([]string, 0)
		for _, dt := range req.DataTypes {
			if !classification.ValidateDataType(dt) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data type: " + dt})
				return
			}
			dataTypes = append(dataTypes, dt)
		}
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.store.UpdateScheduleClassificationLevel(c.Request.Context(), id, req.Level, dataTypes); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule classification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update classification"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Str("level", req.Level).Msg("schedule classification updated")
	c.JSON(http.StatusOK, gin.H{
		"schedule_id": id,
		"level":       req.Level,
		"data_types":  dataTypes,
	})
}

// AutoClassifySchedule automatically classifies a schedule based on its paths and rules.
// POST /api/v1/classifications/schedules/:id/auto-classify
func (h *ClassificationsHandler) AutoClassifySchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Get organization's classification rules
	rules, err := h.store.GetPathClassificationRulesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get classification rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get classification rules"})
		return
	}

	// Convert to classifier rules, combining with defaults
	classifierRules := classification.DefaultRules()
	for _, rule := range rules {
		if rule.Enabled {
			classifierRules = append(classifierRules, rule.ToPathRule())
		}
	}

	// Classify the schedule's paths
	classifier := classification.NewClassifier(classifierRules)
	result := classifier.ClassifyPaths(schedule.Paths)

	// Convert data types to strings
	dataTypes := make([]string, len(result.DataTypes))
	for i, dt := range result.DataTypes {
		dataTypes[i] = string(dt)
	}

	// Update the schedule
	if err := h.store.UpdateScheduleClassificationLevel(c.Request.Context(), id, string(result.Level), dataTypes); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule classification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update classification"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Str("level", string(result.Level)).Msg("schedule auto-classified")
	c.JSON(http.StatusOK, gin.H{
		"schedule_id":     id,
		"level":           result.Level,
		"data_types":      dataTypes,
		"auto_classified": true,
	})
}

// ListBackupsByClassification returns backups filtered by classification level.
// GET /api/v1/classifications/backups
func (h *ClassificationsHandler) ListBackupsByClassification(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	level := c.Query("level")
	if level == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "level parameter is required"})
		return
	}

	if !classification.ValidateLevel(level) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid classification level"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	backups, err := h.store.GetBackupsByClassificationLevel(c.Request.Context(), dbUser.OrgID, level, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list backups by classification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups, "level": level})
}

// GetSummary returns a summary of classifications for the organization.
// GET /api/v1/classifications/summary
func (h *ClassificationsHandler) GetSummary(c *gin.Context) {
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

	summary, err := h.store.GetClassificationSummary(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get classification summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get classification summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetComplianceReport returns a detailed compliance report by classification.
// GET /api/v1/classifications/compliance-report
func (h *ClassificationsHandler) GetComplianceReport(c *gin.Context) {
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

	// Get summary
	summary, err := h.store.GetClassificationSummary(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get classification summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get classification summary"})
		return
	}

	// Get schedules grouped by level
	schedulesByLevel := make(map[string][]models.ScheduleSummary)
	for _, level := range []string{"public", "internal", "confidential", "restricted"} {
		schedules, err := h.store.GetSchedulesByClassificationLevel(c.Request.Context(), dbUser.OrgID, level)
		if err != nil {
			h.logger.Error().Err(err).Str("level", level).Msg("failed to get schedules by level")
			continue
		}
		summaries := make([]models.ScheduleSummary, len(schedules))
		for i, s := range schedules {
			dataTypes := make([]classification.DataType, len(s.ClassificationDataTypes))
			for j, dt := range s.ClassificationDataTypes {
				dataTypes[j] = classification.DataType(dt)
			}
			summaries[i] = models.ScheduleSummary{
				ID:        s.ID,
				Name:      s.Name,
				Level:     classification.Level(s.ClassificationLevel),
				DataTypes: dataTypes,
				Paths:     s.Paths,
				AgentID:   s.AgentID,
			}
		}
		schedulesByLevel[level] = summaries
	}

	report := gin.H{
		"generated_at":       time.Now(),
		"org_id":             dbUser.OrgID,
		"summary":            summary,
		"schedules_by_level": schedulesByLevel,
	}

	c.JSON(http.StatusOK, report)
}
