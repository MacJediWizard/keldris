package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ActivityEventType represents the type of activity event.
type ActivityEventType string

const (
	// Backup events
	ActivityEventBackupStarted   ActivityEventType = "backup_started"
	ActivityEventBackupCompleted ActivityEventType = "backup_completed"
	ActivityEventBackupFailed    ActivityEventType = "backup_failed"

	// Restore events
	ActivityEventRestoreStarted   ActivityEventType = "restore_started"
	ActivityEventRestoreCompleted ActivityEventType = "restore_completed"
	ActivityEventRestoreFailed    ActivityEventType = "restore_failed"

	// Agent events
	ActivityEventAgentConnected    ActivityEventType = "agent_connected"
	ActivityEventAgentDisconnected ActivityEventType = "agent_disconnected"
	ActivityEventAgentCreated      ActivityEventType = "agent_created"
	ActivityEventAgentDeleted      ActivityEventType = "agent_deleted"

	// User events
	ActivityEventUserLogin  ActivityEventType = "user_login"
	ActivityEventUserLogout ActivityEventType = "user_logout"

	// Schedule events
	ActivityEventScheduleCreated  ActivityEventType = "schedule_created"
	ActivityEventScheduleUpdated  ActivityEventType = "schedule_updated"
	ActivityEventScheduleDeleted  ActivityEventType = "schedule_deleted"
	ActivityEventScheduleEnabled  ActivityEventType = "schedule_enabled"
	ActivityEventScheduleDisabled ActivityEventType = "schedule_disabled"

	// Repository events
	ActivityEventRepositoryCreated ActivityEventType = "repository_created"
	ActivityEventRepositoryDeleted ActivityEventType = "repository_deleted"

	// Alert events
	ActivityEventAlertTriggered    ActivityEventType = "alert_triggered"
	ActivityEventAlertAcknowledged ActivityEventType = "alert_acknowledged"
	ActivityEventAlertResolved     ActivityEventType = "alert_resolved"

	// Policy events
	ActivityEventPolicyApplied ActivityEventType = "policy_applied"

	// Maintenance events
	ActivityEventMaintenanceStarted ActivityEventType = "maintenance_started"
	ActivityEventMaintenanceEnded   ActivityEventType = "maintenance_ended"

	// System events
	ActivityEventSystemStartup  ActivityEventType = "system_startup"
	ActivityEventSystemShutdown ActivityEventType = "system_shutdown"
)

// ActivityEventCategory categorizes activity events for filtering.
type ActivityEventCategory string

const (
	ActivityCategoryBackup      ActivityEventCategory = "backup"
	ActivityCategoryRestore     ActivityEventCategory = "restore"
	ActivityCategoryAgent       ActivityEventCategory = "agent"
	ActivityCategoryUser        ActivityEventCategory = "user"
	ActivityCategorySchedule    ActivityEventCategory = "schedule"
	ActivityCategoryRepository  ActivityEventCategory = "repository"
	ActivityCategoryAlert       ActivityEventCategory = "alert"
	ActivityCategoryPolicy      ActivityEventCategory = "policy"
	ActivityCategoryMaintenance ActivityEventCategory = "maintenance"
	ActivityCategorySystem      ActivityEventCategory = "system"
)

// GetCategory returns the category for an event type.
func (t ActivityEventType) GetCategory() ActivityEventCategory {
	switch t {
	case ActivityEventBackupStarted, ActivityEventBackupCompleted, ActivityEventBackupFailed:
		return ActivityCategoryBackup
	case ActivityEventRestoreStarted, ActivityEventRestoreCompleted, ActivityEventRestoreFailed:
		return ActivityCategoryRestore
	case ActivityEventAgentConnected, ActivityEventAgentDisconnected, ActivityEventAgentCreated, ActivityEventAgentDeleted:
		return ActivityCategoryAgent
	case ActivityEventUserLogin, ActivityEventUserLogout:
		return ActivityCategoryUser
	case ActivityEventScheduleCreated, ActivityEventScheduleUpdated, ActivityEventScheduleDeleted, ActivityEventScheduleEnabled, ActivityEventScheduleDisabled:
		return ActivityCategorySchedule
	case ActivityEventRepositoryCreated, ActivityEventRepositoryDeleted:
		return ActivityCategoryRepository
	case ActivityEventAlertTriggered, ActivityEventAlertAcknowledged, ActivityEventAlertResolved:
		return ActivityCategoryAlert
	case ActivityEventPolicyApplied:
		return ActivityCategoryPolicy
	case ActivityEventMaintenanceStarted, ActivityEventMaintenanceEnded:
		return ActivityCategoryMaintenance
	case ActivityEventSystemStartup, ActivityEventSystemShutdown:
		return ActivityCategorySystem
	default:
		return ActivityCategorySystem
	}
}

// ActivityEvent represents a system activity event.
type ActivityEvent struct {
	ID           uuid.UUID             `json:"id"`
	OrgID        uuid.UUID             `json:"org_id"`
	Type         ActivityEventType     `json:"type"`
	Category     ActivityEventCategory `json:"category"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	UserID       *uuid.UUID            `json:"user_id,omitempty"`
	UserName     *string               `json:"user_name,omitempty"`
	AgentID      *uuid.UUID            `json:"agent_id,omitempty"`
	AgentName    *string               `json:"agent_name,omitempty"`
	ResourceType *string               `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID            `json:"resource_id,omitempty"`
	ResourceName *string               `json:"resource_name,omitempty"`
	Metadata     map[string]any        `json:"metadata,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
}

// NewActivityEvent creates a new ActivityEvent with the given details.
func NewActivityEvent(orgID uuid.UUID, eventType ActivityEventType, title, description string) *ActivityEvent {
	return &ActivityEvent{
		ID:          uuid.New(),
		OrgID:       orgID,
		Type:        eventType,
		Category:    eventType.GetCategory(),
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
	}
}

// SetUser sets the user associated with this event.
func (e *ActivityEvent) SetUser(userID uuid.UUID, userName string) {
	e.UserID = &userID
	e.UserName = &userName
}

// SetAgent sets the agent associated with this event.
func (e *ActivityEvent) SetAgent(agentID uuid.UUID, agentName string) {
	e.AgentID = &agentID
	e.AgentName = &agentName
}

// SetResource sets the resource associated with this event.
func (e *ActivityEvent) SetResource(resourceType string, resourceID uuid.UUID, resourceName string) {
	e.ResourceType = &resourceType
	e.ResourceID = &resourceID
	e.ResourceName = &resourceName
}

// SetMetadata sets the metadata from a map.
func (e *ActivityEvent) SetMetadata(metadata map[string]any) {
	e.Metadata = metadata
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (e *ActivityEvent) MetadataJSON() ([]byte, error) {
	if e.Metadata == nil {
		return nil, nil
	}
	return json.Marshal(e.Metadata)
}

// ParseMetadata sets the metadata from JSON bytes.
func (e *ActivityEvent) ParseMetadata(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	e.Metadata = metadata
	return nil
}

// ActivityEventFilter holds filter options for listing activity events.
type ActivityEventFilter struct {
	Category   *ActivityEventCategory `json:"category,omitempty"`
	Type       *ActivityEventType     `json:"type,omitempty"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	AgentID    *uuid.UUID             `json:"agent_id,omitempty"`
	StartTime  *time.Time             `json:"start_time,omitempty"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
}
