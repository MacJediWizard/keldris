package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AlertType represents the type of alert.
type AlertType string

const (
	// AlertTypeAgentOffline indicates an agent has stopped responding.
	AlertTypeAgentOffline AlertType = "agent_offline"
	// AlertTypeBackupSLA indicates a backup SLA has been violated.
	AlertTypeBackupSLA AlertType = "backup_sla"
	// AlertTypeStorageUsage indicates storage usage has exceeded threshold.
	AlertTypeStorageUsage AlertType = "storage_usage"
	// AlertTypeAgentHealthWarning indicates an agent's health status changed to warning.
	AlertTypeAgentHealthWarning AlertType = "agent_health_warning"
	// AlertTypeAgentHealthCritical indicates an agent's health status changed to critical.
	AlertTypeAgentHealthCritical AlertType = "agent_health_critical"
	// AlertTypeReplicationLag indicates geo-replication has fallen behind.
	AlertTypeReplicationLag AlertType = "replication_lag"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	// AlertSeverityInfo indicates informational alert.
	AlertSeverityInfo AlertSeverity = "info"
	// AlertSeverityWarning indicates warning alert.
	AlertSeverityWarning AlertSeverity = "warning"
	// AlertSeverityCritical indicates critical alert.
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert.
type AlertStatus string

const (
	// AlertStatusActive indicates the alert is active.
	AlertStatusActive AlertStatus = "active"
	// AlertStatusAcknowledged indicates the alert has been acknowledged.
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	// AlertStatusResolved indicates the alert has been resolved.
	AlertStatusResolved AlertStatus = "resolved"
)

// ResourceType represents the type of resource an alert relates to.
type ResourceType string

const (
	// ResourceTypeAgent represents an agent resource.
	ResourceTypeAgent ResourceType = "agent"
	// ResourceTypeSchedule represents a schedule resource.
	ResourceTypeSchedule ResourceType = "schedule"
	// ResourceTypeRepository represents a repository resource.
	ResourceTypeRepository ResourceType = "repository"
)

// Alert represents a triggered alert instance.
type Alert struct {
	ID             uuid.UUID      `json:"id"`
	OrgID          uuid.UUID      `json:"org_id"`
	RuleID         *uuid.UUID     `json:"rule_id,omitempty"`
	Type           AlertType      `json:"type"`
	Severity       AlertSeverity  `json:"severity"`
	Status         AlertStatus    `json:"status"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	ResourceType   *ResourceType  `json:"resource_type,omitempty"`
	ResourceID     *uuid.UUID     `json:"resource_id,omitempty"`
	AcknowledgedBy *uuid.UUID     `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// NewAlert creates a new Alert with the given details.
func NewAlert(orgID uuid.UUID, alertType AlertType, severity AlertSeverity, title, message string) *Alert {
	now := time.Now()
	return &Alert{
		ID:        uuid.New(),
		OrgID:     orgID,
		Type:      alertType,
		Severity:  severity,
		Status:    AlertStatusActive,
		Title:     title,
		Message:   message,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetResource sets the resource that this alert relates to.
func (a *Alert) SetResource(resourceType ResourceType, resourceID uuid.UUID) {
	a.ResourceType = &resourceType
	a.ResourceID = &resourceID
}

// SetRuleID sets the rule that triggered this alert.
func (a *Alert) SetRuleID(ruleID uuid.UUID) {
	a.RuleID = &ruleID
}

// Acknowledge marks the alert as acknowledged by the given user.
func (a *Alert) Acknowledge(userID uuid.UUID) {
	now := time.Now()
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedBy = &userID
	a.AcknowledgedAt = &now
	a.UpdatedAt = now
}

// Resolve marks the alert as resolved.
func (a *Alert) Resolve() {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.UpdatedAt = now
}

// SetMetadata sets the metadata from JSON bytes.
func (a *Alert) SetMetadata(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	a.Metadata = metadata
	return nil
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (a *Alert) MetadataJSON() ([]byte, error) {
	if a.Metadata == nil {
		return nil, nil
	}
	return json.Marshal(a.Metadata)
}

// AlertRuleConfig holds type-specific configuration for alert rules.
type AlertRuleConfig struct {
	// For agent_offline: minutes until considered offline
	OfflineThresholdMinutes int `json:"offline_threshold_minutes,omitempty"`
	// For backup_sla: maximum hours since last successful backup
	MaxHoursSinceBackup int `json:"max_hours_since_backup,omitempty"`
	// For storage_usage: percentage threshold
	StorageUsagePercent int `json:"storage_usage_percent,omitempty"`
	// For replication_lag: maximum snapshots behind before alert
	MaxReplicationLagSnapshots int `json:"max_replication_lag_snapshots,omitempty"`
	// For replication_lag: maximum hours behind before alert
	MaxReplicationLagHours int `json:"max_replication_lag_hours,omitempty"`
	// Target-specific filters
	AgentIDs              []uuid.UUID `json:"agent_ids,omitempty"`
	ScheduleIDs           []uuid.UUID `json:"schedule_ids,omitempty"`
	RepositoryID          *uuid.UUID  `json:"repository_id,omitempty"`
	GeoReplicationConfigs []uuid.UUID `json:"geo_replication_configs,omitempty"`
}

// AlertRule defines conditions that trigger alerts.
type AlertRule struct {
	ID        uuid.UUID       `json:"id"`
	OrgID     uuid.UUID       `json:"org_id"`
	Name      string          `json:"name"`
	Type      AlertType       `json:"type"`
	Enabled   bool            `json:"enabled"`
	Config    AlertRuleConfig `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// NewAlertRule creates a new AlertRule with the given details.
func NewAlertRule(orgID uuid.UUID, name string, alertType AlertType, config AlertRuleConfig) *AlertRule {
	now := time.Now()
	return &AlertRule{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Type:      alertType,
		Enabled:   true,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetConfig sets the configuration from JSON bytes.
func (r *AlertRule) SetConfig(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var config AlertRuleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	r.Config = config
	return nil
}

// ConfigJSON returns the config as JSON bytes for database storage.
func (r *AlertRule) ConfigJSON() ([]byte, error) {
	return json.Marshal(r.Config)
}
