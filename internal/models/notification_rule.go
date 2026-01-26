package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleTriggerType represents the type of event that triggers a rule.
type RuleTriggerType string

const (
	TriggerBackupFailed         RuleTriggerType = "backup_failed"
	TriggerBackupSuccess        RuleTriggerType = "backup_success"
	TriggerAgentOffline         RuleTriggerType = "agent_offline"
	TriggerAgentHealthWarning   RuleTriggerType = "agent_health_warning"
	TriggerAgentHealthCritical  RuleTriggerType = "agent_health_critical"
	TriggerStorageUsageHigh     RuleTriggerType = "storage_usage_high"
	TriggerReplicationLag       RuleTriggerType = "replication_lag"
	TriggerRansomwareSuspected  RuleTriggerType = "ransomware_suspected"
	TriggerMaintenanceScheduled RuleTriggerType = "maintenance_scheduled"
)

// RuleActionType represents the type of action to take when a rule is triggered.
type RuleActionType string

const (
	ActionNotifyChannel RuleActionType = "notify_channel"
	ActionEscalate      RuleActionType = "escalate"
	ActionSuppress      RuleActionType = "suppress"
	ActionWebhook       RuleActionType = "webhook"
)

// RuleConditions defines the conditions that must be met for a rule to trigger.
type RuleConditions struct {
	// Count is the number of events required to trigger the rule
	Count int `json:"count,omitempty"`
	// TimeWindowMinutes is the time window in minutes for counting events
	TimeWindowMinutes int `json:"time_window_minutes,omitempty"`
	// Severity filters events by severity level
	Severity string `json:"severity,omitempty"`
	// AgentIDs filters events to specific agents
	AgentIDs []uuid.UUID `json:"agent_ids,omitempty"`
	// ScheduleIDs filters events to specific schedules
	ScheduleIDs []uuid.UUID `json:"schedule_ids,omitempty"`
	// RepositoryIDs filters events to specific repositories
	RepositoryIDs []uuid.UUID `json:"repository_ids,omitempty"`
}

// RuleAction defines an action to take when rule conditions are met.
type RuleAction struct {
	// Type is the action type
	Type RuleActionType `json:"type"`
	// ChannelID is the notification channel to use (for notify_channel)
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	// EscalateToChannelID is the channel to escalate to (for escalate)
	EscalateToChannelID *uuid.UUID `json:"escalate_to_channel_id,omitempty"`
	// WebhookURL is the URL to call (for webhook)
	WebhookURL string `json:"webhook_url,omitempty"`
	// SuppressDurationMinutes is how long to suppress notifications
	SuppressDurationMinutes int `json:"suppress_duration_minutes,omitempty"`
	// Message is a custom message template
	Message string `json:"message,omitempty"`
}

// NotificationRule represents a rule for conditional notifications and escalations.
type NotificationRule struct {
	ID          uuid.UUID       `json:"id"`
	OrgID       uuid.UUID       `json:"org_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	TriggerType RuleTriggerType `json:"trigger_type"`
	Enabled     bool            `json:"enabled"`
	Priority    int             `json:"priority"`
	Conditions  RuleConditions  `json:"conditions"`
	Actions     []RuleAction    `json:"actions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NewNotificationRule creates a new notification rule.
func NewNotificationRule(orgID uuid.UUID, name string, triggerType RuleTriggerType) *NotificationRule {
	now := time.Now()
	return &NotificationRule{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		TriggerType: triggerType,
		Enabled:     true,
		Priority:    0,
		Conditions:  RuleConditions{},
		Actions:     []RuleAction{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetConditions sets the conditions from JSON bytes.
func (r *NotificationRule) SetConditions(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var conditions RuleConditions
	if err := json.Unmarshal(data, &conditions); err != nil {
		return err
	}
	r.Conditions = conditions
	return nil
}

// ConditionsJSON returns the conditions as JSON bytes.
func (r *NotificationRule) ConditionsJSON() ([]byte, error) {
	return json.Marshal(r.Conditions)
}

// SetActions sets the actions from JSON bytes.
func (r *NotificationRule) SetActions(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var actions []RuleAction
	if err := json.Unmarshal(data, &actions); err != nil {
		return err
	}
	r.Actions = actions
	return nil
}

// ActionsJSON returns the actions as JSON bytes.
func (r *NotificationRule) ActionsJSON() ([]byte, error) {
	return json.Marshal(r.Actions)
}

// NotificationRuleEvent represents an event tracked for rule evaluation.
type NotificationRuleEvent struct {
	ID           uuid.UUID       `json:"id"`
	OrgID        uuid.UUID       `json:"org_id"`
	RuleID       uuid.UUID       `json:"rule_id"`
	TriggerType  RuleTriggerType `json:"trigger_type"`
	ResourceType string          `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty"`
	EventData    map[string]any  `json:"event_data,omitempty"`
	OccurredAt   time.Time       `json:"occurred_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

// NewNotificationRuleEvent creates a new rule event.
func NewNotificationRuleEvent(orgID, ruleID uuid.UUID, triggerType RuleTriggerType) *NotificationRuleEvent {
	now := time.Now()
	return &NotificationRuleEvent{
		ID:          uuid.New(),
		OrgID:       orgID,
		RuleID:      ruleID,
		TriggerType: triggerType,
		OccurredAt:  now,
		CreatedAt:   now,
	}
}

// SetEventData sets the event data from JSON bytes.
func (e *NotificationRuleEvent) SetEventData(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var eventData map[string]any
	if err := json.Unmarshal(data, &eventData); err != nil {
		return err
	}
	e.EventData = eventData
	return nil
}

// EventDataJSON returns the event data as JSON bytes.
func (e *NotificationRuleEvent) EventDataJSON() ([]byte, error) {
	if e.EventData == nil {
		return nil, nil
	}
	return json.Marshal(e.EventData)
}

// NotificationRuleExecution represents a record of a rule being executed.
type NotificationRuleExecution struct {
	ID                 uuid.UUID    `json:"id"`
	OrgID              uuid.UUID    `json:"org_id"`
	RuleID             uuid.UUID    `json:"rule_id"`
	TriggeredByEventID *uuid.UUID   `json:"triggered_by_event_id,omitempty"`
	ActionsTaken       []RuleAction `json:"actions_taken"`
	Success            bool         `json:"success"`
	ErrorMessage       string       `json:"error_message,omitempty"`
	ExecutedAt         time.Time    `json:"executed_at"`
	CreatedAt          time.Time    `json:"created_at"`
}

// NewNotificationRuleExecution creates a new rule execution record.
func NewNotificationRuleExecution(orgID, ruleID uuid.UUID, eventID *uuid.UUID) *NotificationRuleExecution {
	now := time.Now()
	return &NotificationRuleExecution{
		ID:                 uuid.New(),
		OrgID:              orgID,
		RuleID:             ruleID,
		TriggeredByEventID: eventID,
		ActionsTaken:       []RuleAction{},
		Success:            true,
		ExecutedAt:         now,
		CreatedAt:          now,
	}
}

// SetActionsTaken sets the actions from JSON bytes.
func (e *NotificationRuleExecution) SetActionsTaken(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var actions []RuleAction
	if err := json.Unmarshal(data, &actions); err != nil {
		return err
	}
	e.ActionsTaken = actions
	return nil
}

// ActionsTakenJSON returns the actions as JSON bytes.
func (e *NotificationRuleExecution) ActionsTakenJSON() ([]byte, error) {
	return json.Marshal(e.ActionsTaken)
}

// MarkFailed marks the execution as failed with an error message.
func (e *NotificationRuleExecution) MarkFailed(errMsg string) {
	e.Success = false
	e.ErrorMessage = errMsg
}

// CreateNotificationRuleRequest represents a request to create a rule.
type CreateNotificationRuleRequest struct {
	Name        string          `json:"name" binding:"required,min=1,max=255"`
	Description string          `json:"description,omitempty"`
	TriggerType RuleTriggerType `json:"trigger_type" binding:"required"`
	Enabled     bool            `json:"enabled"`
	Priority    int             `json:"priority"`
	Conditions  RuleConditions  `json:"conditions" binding:"required"`
	Actions     []RuleAction    `json:"actions" binding:"required,min=1"`
}

// UpdateNotificationRuleRequest represents a request to update a rule.
type UpdateNotificationRuleRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Priority    *int            `json:"priority,omitempty"`
	Conditions  *RuleConditions `json:"conditions,omitempty"`
	Actions     []RuleAction    `json:"actions,omitempty"`
}

// TestNotificationRuleRequest represents a request to test a rule.
type TestNotificationRuleRequest struct {
	EventData map[string]any `json:"event_data,omitempty"`
}

// NotificationRulesResponse wraps a list of rules.
type NotificationRulesResponse struct {
	Rules []*NotificationRule `json:"rules"`
}

// NotificationRuleEventsResponse wraps a list of rule events.
type NotificationRuleEventsResponse struct {
	Events []*NotificationRuleEvent `json:"events"`
}

// NotificationRuleExecutionsResponse wraps a list of rule executions.
type NotificationRuleExecutionsResponse struct {
	Executions []*NotificationRuleExecution `json:"executions"`
}
