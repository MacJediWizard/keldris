package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NotificationRuleStore defines the interface for notification rule persistence operations.
type NotificationRuleStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetNotificationRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationRule, error)
	GetNotificationRuleByID(ctx context.Context, id uuid.UUID) (*models.NotificationRule, error)
	GetEnabledRulesByTriggerType(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType) ([]*models.NotificationRule, error)
	CreateNotificationRule(ctx context.Context, rule *models.NotificationRule) error
	UpdateNotificationRule(ctx context.Context, rule *models.NotificationRule) error
	DeleteNotificationRule(ctx context.Context, id uuid.UUID) error
	GetRecentEventsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleEvent, error)
	GetRecentExecutionsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleExecution, error)
	GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	CreateNotificationRuleEvent(ctx context.Context, event *models.NotificationRuleEvent) error
	CountEventsInTimeWindow(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType, resourceID *uuid.UUID, windowStart time.Time) (int, error)
	CreateNotificationRuleExecution(ctx context.Context, execution *models.NotificationRuleExecution) error
}

// NotificationRulesHandler handles notification rule HTTP endpoints.
type NotificationRulesHandler struct {
	store      NotificationRuleStore
	ruleEngine *notifications.RuleEngine
	logger     zerolog.Logger
}

// NewNotificationRulesHandler creates a new NotificationRulesHandler.
func NewNotificationRulesHandler(store NotificationRuleStore, logger zerolog.Logger) *NotificationRulesHandler {
	return &NotificationRulesHandler{
		store:      store,
		ruleEngine: notifications.NewRuleEngine(store, logger),
		logger:     logger.With().Str("component", "notification_rules_handler").Logger(),
	}
}

// RegisterRoutes registers notification rule routes on the given router group.
func (h *NotificationRulesHandler) RegisterRoutes(r *gin.RouterGroup) {
	rules := r.Group("/notification-rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.GET("/:id", h.GetRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
		rules.POST("/:id/test", h.TestRule)
		rules.GET("/:id/events", h.ListRuleEvents)
		rules.GET("/:id/executions", h.ListRuleExecutions)
	}
}

// ListRules returns all notification rules for the authenticated user's organization.
// GET /api/v1/notification-rules
func (h *NotificationRulesHandler) ListRules(c *gin.Context) {
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

	rules, err := h.store.GetNotificationRulesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list notification rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notification rules"})
		return
	}

	c.JSON(http.StatusOK, models.NotificationRulesResponse{Rules: rules})
}

// GetRule returns a specific notification rule by ID.
// GET /api/v1/notification-rules/:id
func (h *NotificationRulesHandler) GetRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to get notification rule")
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// CreateRule creates a new notification rule.
// POST /api/v1/notification-rules
func (h *NotificationRulesHandler) CreateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateNotificationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if !isValidTriggerType(req.TriggerType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trigger type"})
		return
	}

	// Validate actions
	for _, action := range req.Actions {
		if !isValidActionType(action.Type) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action type: " + string(action.Type)})
			return
		}
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Verify any referenced channels exist and belong to org
	for _, action := range req.Actions {
		if action.ChannelID != nil {
			channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), *action.ChannelID)
			if err != nil || channel.OrgID != dbUser.OrgID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID in action"})
				return
			}
		}
		if action.EscalateToChannelID != nil {
			channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), *action.EscalateToChannelID)
			if err != nil || channel.OrgID != dbUser.OrgID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid escalation channel ID in action"})
				return
			}
		}
	}

	rule := models.NewNotificationRule(dbUser.OrgID, req.Name, req.TriggerType)
	rule.Description = req.Description
	rule.Enabled = req.Enabled
	rule.Priority = req.Priority
	rule.Conditions = req.Conditions
	rule.Actions = req.Actions

	if err := h.store.CreateNotificationRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification rule"})
		return
	}

	h.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("name", req.Name).
		Str("trigger_type", string(req.TriggerType)).
		Str("org_id", dbUser.OrgID.String()).
		Msg("notification rule created")

	c.JSON(http.StatusCreated, rule)
}

// UpdateRule updates an existing notification rule.
// PUT /api/v1/notification-rules/:id
func (h *NotificationRulesHandler) UpdateRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	var req models.UpdateNotificationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Conditions != nil {
		rule.Conditions = *req.Conditions
	}
	if req.Actions != nil {
		// Validate actions
		for _, action := range req.Actions {
			if !isValidActionType(action.Type) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action type: " + string(action.Type)})
				return
			}
			if action.ChannelID != nil {
				channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), *action.ChannelID)
				if err != nil || channel.OrgID != dbUser.OrgID {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID in action"})
					return
				}
			}
			if action.EscalateToChannelID != nil {
				channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), *action.EscalateToChannelID)
				if err != nil || channel.OrgID != dbUser.OrgID {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid escalation channel ID in action"})
					return
				}
			}
		}
		rule.Actions = req.Actions
	}

	if err := h.store.UpdateNotificationRule(c.Request.Context(), rule); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to update notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification rule"})
		return
	}

	h.logger.Info().Str("rule_id", id.String()).Msg("notification rule updated")
	c.JSON(http.StatusOK, rule)
}

// DeleteRule removes a notification rule.
// DELETE /api/v1/notification-rules/:id
func (h *NotificationRulesHandler) DeleteRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	if err := h.store.DeleteNotificationRule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to delete notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notification rule"})
		return
	}

	h.logger.Info().Str("rule_id", id.String()).Msg("notification rule deleted")
	c.JSON(http.StatusOK, gin.H{"message": "notification rule deleted"})
}

// TestRule tests a rule with sample event data.
// POST /api/v1/notification-rules/:id/test
func (h *NotificationRulesHandler) TestRule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	var req models.TestNotificationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK for testing
		req = models.TestNotificationRuleRequest{}
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	execution, err := h.ruleEngine.TestRule(c.Request.Context(), rule, req.EventData)
	if err != nil {
		h.logger.Warn().Err(err).Str("rule_id", id.String()).Msg("rule test did not match")
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	h.logger.Info().Str("rule_id", id.String()).Msg("rule test executed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"execution": execution,
	})
}

// ListRuleEvents returns recent events for a rule.
// GET /api/v1/notification-rules/:id/events
func (h *NotificationRulesHandler) ListRuleEvents(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	events, err := h.store.GetRecentEventsForRule(c.Request.Context(), id, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to get rule events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rule events"})
		return
	}

	c.JSON(http.StatusOK, models.NotificationRuleEventsResponse{Events: events})
}

// ListRuleExecutions returns recent executions for a rule.
// GET /api/v1/notification-rules/:id/executions
func (h *NotificationRulesHandler) ListRuleExecutions(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule ID"})
		return
	}

	rule, err := h.store.GetNotificationRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if rule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification rule not found"})
		return
	}

	executions, err := h.store.GetRecentExecutionsForRule(c.Request.Context(), id, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("rule_id", id.String()).Msg("failed to get rule executions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rule executions"})
		return
	}

	c.JSON(http.StatusOK, models.NotificationRuleExecutionsResponse{Executions: executions})
}

// isValidTriggerType checks if a trigger type is valid.
func isValidTriggerType(t models.RuleTriggerType) bool {
	switch t {
	case models.TriggerBackupFailed, models.TriggerBackupSuccess, models.TriggerAgentOffline,
		models.TriggerAgentHealthWarning, models.TriggerAgentHealthCritical, models.TriggerStorageUsageHigh,
		models.TriggerReplicationLag, models.TriggerRansomwareSuspected, models.TriggerMaintenanceScheduled:
		return true
	default:
		return false
	}
}

// isValidActionType checks if an action type is valid.
func isValidActionType(t models.RuleActionType) bool {
	switch t {
	case models.ActionNotifyChannel, models.ActionEscalate, models.ActionSuppress, models.ActionWebhook:
		return true
	default:
		return false
	}
}
