package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RuleStore defines the interface for notification rule data access.
type RuleStore interface {
	// Rules
	GetNotificationRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationRule, error)
	GetNotificationRuleByID(ctx context.Context, id uuid.UUID) (*models.NotificationRule, error)
	GetEnabledRulesByTriggerType(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType) ([]*models.NotificationRule, error)
	CreateNotificationRule(ctx context.Context, rule *models.NotificationRule) error
	UpdateNotificationRule(ctx context.Context, rule *models.NotificationRule) error
	DeleteNotificationRule(ctx context.Context, id uuid.UUID) error

	// Events
	CreateNotificationRuleEvent(ctx context.Context, event *models.NotificationRuleEvent) error
	CountEventsInTimeWindow(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType, resourceID *uuid.UUID, windowStart time.Time) (int, error)
	GetRecentEventsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleEvent, error)

	// Executions
	CreateNotificationRuleExecution(ctx context.Context, execution *models.NotificationRuleExecution) error
	GetRecentExecutionsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleExecution, error)

	// Notification channels for actions
	GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
}

// RuleEngine evaluates notification rules and executes actions.
type RuleEngine struct {
	store      RuleStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
}

// NewRuleEngine creates a new rule engine.
func NewRuleEngine(store RuleStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *RuleEngine {
	return &RuleEngine{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "rule_engine").Logger(),
	}
}

// decryptConfig decrypts an encrypted channel config and unmarshals it into dest.
func (e *RuleEngine) decryptConfig(encrypted []byte, dest interface{}) error {
	decrypted, err := e.keyManager.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt config: %w", err)
	}
	if err := json.Unmarshal(decrypted, dest); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return nil
}

// EventContext contains context about the event being processed.
type EventContext struct {
	OrgID        uuid.UUID
	TriggerType  models.RuleTriggerType
	ResourceType string
	ResourceID   *uuid.UUID
	Severity     string
	EventData    map[string]any
}

// EvaluateEvent processes an event against all matching rules.
func (e *RuleEngine) EvaluateEvent(ctx context.Context, eventCtx EventContext) error {
	// Get all enabled rules for this trigger type
	rules, err := e.store.GetEnabledRulesByTriggerType(ctx, eventCtx.OrgID, eventCtx.TriggerType)
	if err != nil {
		e.logger.Error().Err(err).
			Str("org_id", eventCtx.OrgID.String()).
			Str("trigger_type", string(eventCtx.TriggerType)).
			Msg("failed to get rules for trigger type")
		return err
	}

	if len(rules) == 0 {
		e.logger.Debug().
			Str("org_id", eventCtx.OrgID.String()).
			Str("trigger_type", string(eventCtx.TriggerType)).
			Msg("no rules configured for trigger type")
		return nil
	}

	// Sort rules by priority (lower number = higher priority)
	sortedRules := sortRulesByPriority(rules)

	// Evaluate each rule
	for _, rule := range sortedRules {
		if err := e.evaluateRule(ctx, rule, eventCtx); err != nil {
			e.logger.Error().Err(err).
				Str("rule_id", rule.ID.String()).
				Str("rule_name", rule.Name).
				Msg("failed to evaluate rule")
			// Continue with other rules even if one fails
		}
	}

	return nil
}

// evaluateRule evaluates a single rule against an event.
func (e *RuleEngine) evaluateRule(ctx context.Context, rule *models.NotificationRule, eventCtx EventContext) error {
	// Check if event matches rule filters
	if !e.matchesFilters(rule, eventCtx) {
		e.logger.Debug().
			Str("rule_id", rule.ID.String()).
			Str("rule_name", rule.Name).
			Msg("event does not match rule filters")
		return nil
	}

	// Record the event
	event := models.NewNotificationRuleEvent(eventCtx.OrgID, rule.ID, eventCtx.TriggerType)
	event.ResourceType = eventCtx.ResourceType
	event.ResourceID = eventCtx.ResourceID
	event.EventData = eventCtx.EventData

	if err := e.store.CreateNotificationRuleEvent(ctx, event); err != nil {
		e.logger.Error().Err(err).
			Str("rule_id", rule.ID.String()).
			Msg("failed to record rule event")
		return err
	}

	// Check if conditions are met
	conditionsMet, err := e.checkConditions(ctx, rule, eventCtx)
	if err != nil {
		e.logger.Error().Err(err).
			Str("rule_id", rule.ID.String()).
			Msg("failed to check rule conditions")
		return err
	}

	if !conditionsMet {
		e.logger.Debug().
			Str("rule_id", rule.ID.String()).
			Str("rule_name", rule.Name).
			Msg("rule conditions not met")
		return nil
	}

	// Execute actions
	execution := models.NewNotificationRuleExecution(eventCtx.OrgID, rule.ID, &event.ID)
	if err := e.executeActions(ctx, rule, eventCtx, execution); err != nil {
		execution.MarkFailed(err.Error())
		e.logger.Error().Err(err).
			Str("rule_id", rule.ID.String()).
			Str("rule_name", rule.Name).
			Msg("failed to execute rule actions")
	}

	// Record execution
	if err := e.store.CreateNotificationRuleExecution(ctx, execution); err != nil {
		e.logger.Error().Err(err).
			Str("rule_id", rule.ID.String()).
			Msg("failed to record rule execution")
	}

	e.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Bool("success", execution.Success).
		Int("actions_taken", len(execution.ActionsTaken)).
		Msg("rule executed")

	return nil
}

// matchesFilters checks if an event matches the rule's filter conditions.
func (e *RuleEngine) matchesFilters(rule *models.NotificationRule, eventCtx EventContext) bool {
	cond := rule.Conditions

	// Check severity filter
	if cond.Severity != "" && cond.Severity != eventCtx.Severity {
		return false
	}

	// Check agent filter
	if len(cond.AgentIDs) > 0 && eventCtx.ResourceType == "agent" && eventCtx.ResourceID != nil {
		found := false
		for _, id := range cond.AgentIDs {
			if id == *eventCtx.ResourceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check schedule filter
	if len(cond.ScheduleIDs) > 0 && eventCtx.ResourceType == "schedule" && eventCtx.ResourceID != nil {
		found := false
		for _, id := range cond.ScheduleIDs {
			if id == *eventCtx.ResourceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check repository filter
	if len(cond.RepositoryIDs) > 0 && eventCtx.ResourceType == "repository" && eventCtx.ResourceID != nil {
		found := false
		for _, id := range cond.RepositoryIDs {
			if id == *eventCtx.ResourceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// checkConditions checks if the rule's count/time window conditions are met.
func (e *RuleEngine) checkConditions(ctx context.Context, rule *models.NotificationRule, eventCtx EventContext) (bool, error) {
	cond := rule.Conditions

	// If no count condition, rule triggers immediately
	if cond.Count <= 1 {
		return true, nil
	}

	// Calculate time window
	windowMinutes := cond.TimeWindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = 60 // Default to 1 hour
	}
	windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	// Count events in the time window
	count, err := e.store.CountEventsInTimeWindow(ctx, eventCtx.OrgID, eventCtx.TriggerType, eventCtx.ResourceID, windowStart)
	if err != nil {
		return false, err
	}

	e.logger.Debug().
		Str("rule_id", rule.ID.String()).
		Int("required_count", cond.Count).
		Int("actual_count", count).
		Int("window_minutes", windowMinutes).
		Msg("checked event count condition")

	return count >= cond.Count, nil
}

// executeActions executes all actions defined in the rule.
func (e *RuleEngine) executeActions(ctx context.Context, rule *models.NotificationRule, eventCtx EventContext, execution *models.NotificationRuleExecution) error {
	for _, action := range rule.Actions {
		if err := e.executeAction(ctx, action, rule, eventCtx); err != nil {
			e.logger.Error().Err(err).
				Str("rule_id", rule.ID.String()).
				Str("action_type", string(action.Type)).
				Msg("failed to execute action")
			// Record the action even if it failed
			execution.ActionsTaken = append(execution.ActionsTaken, action)
			return err
		}
		execution.ActionsTaken = append(execution.ActionsTaken, action)
	}
	return nil
}

// executeAction executes a single action.
func (e *RuleEngine) executeAction(ctx context.Context, action models.RuleAction, rule *models.NotificationRule, eventCtx EventContext) error {
	switch action.Type {
	case models.ActionNotifyChannel:
		return e.executeNotifyChannel(ctx, action, rule, eventCtx)
	case models.ActionEscalate:
		return e.executeEscalate(ctx, action, rule, eventCtx)
	case models.ActionSuppress:
		return e.executeSuppress(ctx, action, rule, eventCtx)
	case models.ActionWebhook:
		return e.executeWebhook(ctx, action, rule, eventCtx)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeNotifyChannel sends a notification to a channel.
func (e *RuleEngine) executeNotifyChannel(ctx context.Context, action models.RuleAction, rule *models.NotificationRule, eventCtx EventContext) error {
	if action.ChannelID == nil {
		return fmt.Errorf("channel ID required for notify_channel action")
	}

	channel, err := e.store.GetNotificationChannelByID(ctx, *action.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get notification channel: %w", err)
	}

	if !channel.Enabled {
		e.logger.Debug().
			Str("channel_id", channel.ID.String()).
			Str("channel_name", channel.Name).
			Msg("notification channel is disabled, skipping")
		return nil
	}

	e.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("channel_type", string(channel.Type)).
		Msg("sending notification via channel")

	// Handle different channel types
	switch channel.Type {
	case models.ChannelTypePagerDuty:
		return e.sendPagerDutyNotification(ctx, channel, rule, eventCtx, action.Message)
	case models.ChannelTypeSlack:
		return e.sendSlackNotification(ctx, channel, rule, eventCtx, action.Message)
	case models.ChannelTypeEmail:
		return e.sendEmailNotification(ctx, channel, rule, eventCtx, action.Message)
	case models.ChannelTypeWebhook:
		return e.sendWebhookNotification(ctx, channel, rule, eventCtx, action.Message)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// executeEscalate escalates to a different channel (typically PagerDuty).
func (e *RuleEngine) executeEscalate(ctx context.Context, action models.RuleAction, rule *models.NotificationRule, eventCtx EventContext) error {
	if action.EscalateToChannelID == nil {
		return fmt.Errorf("escalate_to_channel_id required for escalate action")
	}

	channel, err := e.store.GetNotificationChannelByID(ctx, *action.EscalateToChannelID)
	if err != nil {
		return fmt.Errorf("failed to get escalation channel: %w", err)
	}

	e.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Msg("escalating to channel")

	// Use the same notification logic as notify_channel
	notifyAction := models.RuleAction{
		Type:      models.ActionNotifyChannel,
		ChannelID: action.EscalateToChannelID,
		Message:   fmt.Sprintf("[ESCALATION] %s", action.Message),
	}
	return e.executeNotifyChannel(ctx, notifyAction, rule, eventCtx)
}

// executeSuppress suppresses further notifications for a duration.
func (e *RuleEngine) executeSuppress(ctx context.Context, action models.RuleAction, rule *models.NotificationRule, eventCtx EventContext) error {
	// Suppression would typically set a flag or record in the database
	// For now, just log that suppression was requested
	duration := action.SuppressDurationMinutes
	if duration <= 0 {
		duration = 60 // Default 1 hour
	}

	e.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Int("duration_minutes", duration).
		Msg("suppressing notifications")

	// TODO: Implement suppression tracking
	return nil
}

// executeWebhook calls a webhook URL.
func (e *RuleEngine) executeWebhook(ctx context.Context, action models.RuleAction, rule *models.NotificationRule, eventCtx EventContext) error {
	if action.WebhookURL == "" {
		return fmt.Errorf("webhook URL required for webhook action")
	}

	e.logger.Warn().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Str("webhook_url", action.WebhookURL).
		Str("trigger_type", string(eventCtx.TriggerType)).
		Msg("rule engine: direct webhook action delivery not yet implemented")

	return fmt.Errorf("webhook action delivery not yet implemented")
}

// sendPagerDutyNotification sends a PagerDuty alert.
func (e *RuleEngine) sendPagerDutyNotification(ctx context.Context, channel *models.NotificationChannel, rule *models.NotificationRule, eventCtx EventContext, message string) error {
	var config models.PagerDutyChannelConfig
	if err := e.decryptConfig(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("failed to decrypt PagerDuty config: %w", err)
	}

	e.logger.Warn().
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("trigger_type", string(eventCtx.TriggerType)).
		Str("rule_name", rule.Name).
		Str("message", message).
		Str("routing_key_prefix", config.RoutingKey[:min(8, len(config.RoutingKey))]+"...").
		Msg("rule engine: PagerDuty notification delivery not yet implemented")

	return fmt.Errorf("pagerduty notification delivery not yet implemented")
}

// sendSlackNotification sends a Slack message.
func (e *RuleEngine) sendSlackNotification(ctx context.Context, channel *models.NotificationChannel, rule *models.NotificationRule, eventCtx EventContext, message string) error {
	var config models.SlackChannelConfig
	if err := e.decryptConfig(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("failed to decrypt Slack config: %w", err)
	}

	e.logger.Warn().
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("trigger_type", string(eventCtx.TriggerType)).
		Str("rule_name", rule.Name).
		Str("message", message).
		Msg("rule engine: Slack notification delivery not yet implemented")

	return fmt.Errorf("slack notification delivery not yet implemented")
}

// sendEmailNotification sends an email notification.
func (e *RuleEngine) sendEmailNotification(ctx context.Context, channel *models.NotificationChannel, rule *models.NotificationRule, eventCtx EventContext, message string) error {
	var config models.EmailChannelConfig
	if err := e.decryptConfig(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("failed to decrypt email config: %w", err)
	}

	e.logger.Warn().
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("from", config.From).
		Str("host", config.Host).
		Str("trigger_type", string(eventCtx.TriggerType)).
		Str("rule_name", rule.Name).
		Str("message", message).
		Msg("rule engine: email notification delivery not yet implemented")

	return fmt.Errorf("email notification delivery not yet implemented")
}

// sendWebhookNotification sends a generic webhook notification.
func (e *RuleEngine) sendWebhookNotification(ctx context.Context, channel *models.NotificationChannel, rule *models.NotificationRule, eventCtx EventContext, message string) error {
	var config models.WebhookChannelConfig
	if err := e.decryptConfig(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("failed to decrypt webhook config: %w", err)
	}

	e.logger.Warn().
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("trigger_type", string(eventCtx.TriggerType)).
		Str("rule_name", rule.Name).
		Str("url", config.URL).
		Str("message", message).
		Msg("rule engine: webhook notification delivery not yet implemented")

	return fmt.Errorf("webhook notification delivery not yet implemented")
}

// TestRule tests a rule with sample event data without persisting.
func (e *RuleEngine) TestRule(ctx context.Context, rule *models.NotificationRule, eventData map[string]any) (*models.NotificationRuleExecution, error) {
	eventCtx := EventContext{
		OrgID:       rule.OrgID,
		TriggerType: rule.TriggerType,
		EventData:   eventData,
	}

	// Check filters
	if !e.matchesFilters(rule, eventCtx) {
		return nil, fmt.Errorf("event does not match rule filters")
	}

	// For testing, we don't check count conditions - just verify actions can be executed
	execution := models.NewNotificationRuleExecution(rule.OrgID, rule.ID, nil)
	execution.ActionsTaken = rule.Actions

	e.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Int("actions_count", len(rule.Actions)).
		Msg("rule test completed")

	return execution, nil
}

// sortRulesByPriority sorts rules by priority (lower number = higher priority).
func sortRulesByPriority(rules []*models.NotificationRule) []*models.NotificationRule {
	sorted := make([]*models.NotificationRule, len(rules))
	copy(sorted, rules)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Priority > sorted[j+1].Priority {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}
