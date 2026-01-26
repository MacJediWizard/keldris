package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Notification Rules methods

// GetNotificationRulesByOrgID returns all notification rules for an organization.
func (db *DB) GetNotificationRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, trigger_type, enabled, priority,
		       conditions, actions, created_at, updated_at
		FROM notification_rules
		WHERE org_id = $1
		ORDER BY priority, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get notification rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.NotificationRule
	for rows.Next() {
		r, err := scanNotificationRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification rules: %w", err)
	}
	return rules, nil
}

// GetNotificationRuleByID returns a notification rule by ID.
func (db *DB) GetNotificationRuleByID(ctx context.Context, id uuid.UUID) (*models.NotificationRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, trigger_type, enabled, priority,
		       conditions, actions, created_at, updated_at
		FROM notification_rules
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get notification rule: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("notification rule not found")
	}
	return scanNotificationRule(rows)
}

// GetEnabledRulesByTriggerType returns all enabled rules for a trigger type.
func (db *DB) GetEnabledRulesByTriggerType(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType) ([]*models.NotificationRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, description, trigger_type, enabled, priority,
		       conditions, actions, created_at, updated_at
		FROM notification_rules
		WHERE org_id = $1 AND trigger_type = $2 AND enabled = true
		ORDER BY priority
	`, orgID, string(triggerType))
	if err != nil {
		return nil, fmt.Errorf("get enabled rules by trigger: %w", err)
	}
	defer rows.Close()

	var rules []*models.NotificationRule
	for rows.Next() {
		r, err := scanNotificationRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled rules: %w", err)
	}
	return rules, nil
}

// CreateNotificationRule creates a new notification rule.
func (db *DB) CreateNotificationRule(ctx context.Context, rule *models.NotificationRule) error {
	conditionsJSON, err := rule.ConditionsJSON()
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}
	actionsJSON, err := rule.ActionsJSON()
	if err != nil {
		return fmt.Errorf("marshal actions: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO notification_rules (id, org_id, name, description, trigger_type, enabled,
		                                priority, conditions, actions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, rule.ID, rule.OrgID, rule.Name, rule.Description, string(rule.TriggerType),
		rule.Enabled, rule.Priority, conditionsJSON, actionsJSON, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification rule: %w", err)
	}
	return nil
}

// UpdateNotificationRule updates an existing notification rule.
func (db *DB) UpdateNotificationRule(ctx context.Context, rule *models.NotificationRule) error {
	rule.UpdatedAt = time.Now()

	conditionsJSON, err := rule.ConditionsJSON()
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}
	actionsJSON, err := rule.ActionsJSON()
	if err != nil {
		return fmt.Errorf("marshal actions: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE notification_rules
		SET name = $2, description = $3, enabled = $4, priority = $5,
		    conditions = $6, actions = $7, updated_at = $8
		WHERE id = $1
	`, rule.ID, rule.Name, rule.Description, rule.Enabled, rule.Priority,
		conditionsJSON, actionsJSON, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update notification rule: %w", err)
	}
	return nil
}

// DeleteNotificationRule deletes a notification rule.
func (db *DB) DeleteNotificationRule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM notification_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete notification rule: %w", err)
	}
	return nil
}

// scanNotificationRule scans a row into a NotificationRule.
func scanNotificationRule(rows interface{ Scan(dest ...any) error }) (*models.NotificationRule, error) {
	var r models.NotificationRule
	var triggerTypeStr string
	var description *string
	var conditionsBytes, actionsBytes []byte

	err := rows.Scan(
		&r.ID, &r.OrgID, &r.Name, &description, &triggerTypeStr, &r.Enabled,
		&r.Priority, &conditionsBytes, &actionsBytes, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan notification rule: %w", err)
	}

	r.TriggerType = models.RuleTriggerType(triggerTypeStr)
	if description != nil {
		r.Description = *description
	}
	if err := r.SetConditions(conditionsBytes); err != nil {
		return nil, fmt.Errorf("parse conditions: %w", err)
	}
	if err := r.SetActions(actionsBytes); err != nil {
		return nil, fmt.Errorf("parse actions: %w", err)
	}

	return &r, nil
}

// Notification Rule Events methods

// CreateNotificationRuleEvent creates a new rule event record.
func (db *DB) CreateNotificationRuleEvent(ctx context.Context, event *models.NotificationRuleEvent) error {
	eventDataJSON, err := event.EventDataJSON()
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO notification_rule_events (id, org_id, rule_id, trigger_type, resource_type,
		                                      resource_id, event_data, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.OrgID, event.RuleID, string(event.TriggerType), event.ResourceType,
		event.ResourceID, eventDataJSON, event.OccurredAt, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create notification rule event: %w", err)
	}
	return nil
}

// CountEventsInTimeWindow counts events matching criteria within a time window.
func (db *DB) CountEventsInTimeWindow(ctx context.Context, orgID uuid.UUID, triggerType models.RuleTriggerType, resourceID *uuid.UUID, windowStart time.Time) (int, error) {
	var count int
	var err error

	if resourceID != nil {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM notification_rule_events
			WHERE org_id = $1 AND trigger_type = $2 AND resource_id = $3 AND occurred_at >= $4
		`, orgID, string(triggerType), resourceID, windowStart).Scan(&count)
	} else {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM notification_rule_events
			WHERE org_id = $1 AND trigger_type = $2 AND occurred_at >= $3
		`, orgID, string(triggerType), windowStart).Scan(&count)
	}

	if err != nil {
		return 0, fmt.Errorf("count events in window: %w", err)
	}
	return count, nil
}

// GetRecentEventsForRule returns recent events for a rule.
func (db *DB) GetRecentEventsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, rule_id, trigger_type, resource_type, resource_id,
		       event_data, occurred_at, created_at
		FROM notification_rule_events
		WHERE rule_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`, ruleID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent events: %w", err)
	}
	defer rows.Close()

	var events []*models.NotificationRuleEvent
	for rows.Next() {
		e, err := scanNotificationRuleEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

// scanNotificationRuleEvent scans a row into a NotificationRuleEvent.
func scanNotificationRuleEvent(rows interface{ Scan(dest ...any) error }) (*models.NotificationRuleEvent, error) {
	var e models.NotificationRuleEvent
	var triggerTypeStr string
	var resourceType *string
	var eventDataBytes []byte

	err := rows.Scan(
		&e.ID, &e.OrgID, &e.RuleID, &triggerTypeStr, &resourceType,
		&e.ResourceID, &eventDataBytes, &e.OccurredAt, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan notification rule event: %w", err)
	}

	e.TriggerType = models.RuleTriggerType(triggerTypeStr)
	if resourceType != nil {
		e.ResourceType = *resourceType
	}
	if err := e.SetEventData(eventDataBytes); err != nil {
		return nil, fmt.Errorf("parse event data: %w", err)
	}

	return &e, nil
}

// Notification Rule Executions methods

// CreateNotificationRuleExecution creates a new rule execution record.
func (db *DB) CreateNotificationRuleExecution(ctx context.Context, execution *models.NotificationRuleExecution) error {
	actionsTakenJSON, err := execution.ActionsTakenJSON()
	if err != nil {
		return fmt.Errorf("marshal actions taken: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO notification_rule_executions (id, org_id, rule_id, triggered_by_event_id,
		                                          actions_taken, success, error_message,
		                                          executed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, execution.ID, execution.OrgID, execution.RuleID, execution.TriggeredByEventID,
		actionsTakenJSON, execution.Success, execution.ErrorMessage,
		execution.ExecutedAt, execution.CreatedAt)
	if err != nil {
		return fmt.Errorf("create notification rule execution: %w", err)
	}
	return nil
}

// GetRecentExecutionsForRule returns recent executions for a rule.
func (db *DB) GetRecentExecutionsForRule(ctx context.Context, ruleID uuid.UUID, limit int) ([]*models.NotificationRuleExecution, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, rule_id, triggered_by_event_id, actions_taken,
		       success, error_message, executed_at, created_at
		FROM notification_rule_executions
		WHERE rule_id = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`, ruleID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent executions: %w", err)
	}
	defer rows.Close()

	var executions []*models.NotificationRuleExecution
	for rows.Next() {
		e, err := scanNotificationRuleExecution(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate executions: %w", err)
	}
	return executions, nil
}

// scanNotificationRuleExecution scans a row into a NotificationRuleExecution.
func scanNotificationRuleExecution(rows interface{ Scan(dest ...any) error }) (*models.NotificationRuleExecution, error) {
	var e models.NotificationRuleExecution
	var actionsTakenBytes []byte
	var errorMessage *string

	err := rows.Scan(
		&e.ID, &e.OrgID, &e.RuleID, &e.TriggeredByEventID, &actionsTakenBytes,
		&e.Success, &errorMessage, &e.ExecutedAt, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan notification rule execution: %w", err)
	}

	if errorMessage != nil {
		e.ErrorMessage = *errorMessage
	}
	if err := e.SetActionsTaken(actionsTakenBytes); err != nil {
		return nil, fmt.Errorf("parse actions taken: %w", err)
	}

	return &e, nil
}
