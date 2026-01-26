package models

import (
	"time"

	"github.com/google/uuid"
)

// ComponentType represents the type of component that can have downtime.
type ComponentType string

const (
	// ComponentTypeAgent represents an agent component.
	ComponentTypeAgent ComponentType = "agent"
	// ComponentTypeServer represents a server component.
	ComponentTypeServer ComponentType = "server"
	// ComponentTypeRepository represents a repository component.
	ComponentTypeRepository ComponentType = "repository"
	// ComponentTypeService represents a service component.
	ComponentTypeService ComponentType = "service"
)

// DowntimeSeverity represents the severity of a downtime event.
type DowntimeSeverity string

const (
	// DowntimeSeverityInfo indicates informational downtime.
	DowntimeSeverityInfo DowntimeSeverity = "info"
	// DowntimeSeverityWarning indicates warning-level downtime.
	DowntimeSeverityWarning DowntimeSeverity = "warning"
	// DowntimeSeverityCritical indicates critical downtime.
	DowntimeSeverityCritical DowntimeSeverity = "critical"
)

// BadgeType represents the time period for uptime badges.
type BadgeType string

const (
	// BadgeType7D represents a 7-day uptime badge.
	BadgeType7D BadgeType = "7d"
	// BadgeType30D represents a 30-day uptime badge.
	BadgeType30D BadgeType = "30d"
	// BadgeType90D represents a 90-day uptime badge.
	BadgeType90D BadgeType = "90d"
	// BadgeType365D represents a 365-day uptime badge.
	BadgeType365D BadgeType = "365d"
)

// DowntimeEvent represents a historical downtime/outage event.
type DowntimeEvent struct {
	ID              uuid.UUID         `json:"id"`
	OrgID           uuid.UUID         `json:"org_id"`
	ComponentType   ComponentType     `json:"component_type"`
	ComponentID     *uuid.UUID        `json:"component_id,omitempty"`
	ComponentName   string            `json:"component_name"`
	StartedAt       time.Time         `json:"started_at"`
	EndedAt         *time.Time        `json:"ended_at,omitempty"`
	DurationSeconds *int              `json:"duration_seconds,omitempty"`
	Severity        DowntimeSeverity  `json:"severity"`
	Cause           *string           `json:"cause,omitempty"`
	Notes           *string           `json:"notes,omitempty"`
	ResolvedBy      *uuid.UUID        `json:"resolved_by,omitempty"`
	AutoDetected    bool              `json:"auto_detected"`
	AlertID         *uuid.UUID        `json:"alert_id,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// NewDowntimeEvent creates a new DowntimeEvent with the given details.
func NewDowntimeEvent(orgID uuid.UUID, componentType ComponentType, componentName string, severity DowntimeSeverity) *DowntimeEvent {
	now := time.Now()
	return &DowntimeEvent{
		ID:            uuid.New(),
		OrgID:         orgID,
		ComponentType: componentType,
		ComponentName: componentName,
		StartedAt:     now,
		Severity:      severity,
		AutoDetected:  true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SetComponent sets the component reference for this downtime event.
func (d *DowntimeEvent) SetComponent(componentID uuid.UUID) {
	d.ComponentID = &componentID
}

// SetAlert links this downtime event to an alert.
func (d *DowntimeEvent) SetAlert(alertID uuid.UUID) {
	d.AlertID = &alertID
}

// End marks the downtime event as resolved.
func (d *DowntimeEvent) End(resolvedBy *uuid.UUID) {
	now := time.Now()
	d.EndedAt = &now
	d.ResolvedBy = resolvedBy
	d.UpdatedAt = now

	// Calculate duration
	duration := int(now.Sub(d.StartedAt).Seconds())
	d.DurationSeconds = &duration
}

// IsActive returns true if the downtime event is still ongoing.
func (d *DowntimeEvent) IsActive() bool {
	return d.EndedAt == nil
}

// Duration returns the duration of the downtime event.
func (d *DowntimeEvent) Duration() time.Duration {
	if d.DurationSeconds != nil {
		return time.Duration(*d.DurationSeconds) * time.Second
	}
	if d.EndedAt != nil {
		return d.EndedAt.Sub(d.StartedAt)
	}
	return time.Since(d.StartedAt)
}

// UptimeStats represents aggregated uptime statistics for a period.
type UptimeStats struct {
	ID              uuid.UUID     `json:"id"`
	OrgID           uuid.UUID     `json:"org_id"`
	ComponentType   ComponentType `json:"component_type"`
	ComponentID     *uuid.UUID    `json:"component_id,omitempty"`
	ComponentName   string        `json:"component_name"`
	PeriodStart     time.Time     `json:"period_start"`
	PeriodEnd       time.Time     `json:"period_end"`
	TotalSeconds    int           `json:"total_seconds"`
	DowntimeSeconds int           `json:"downtime_seconds"`
	UptimePercent   float64       `json:"uptime_percent"`
	IncidentCount   int           `json:"incident_count"`
	CreatedAt       time.Time     `json:"created_at"`
}

// NewUptimeStats creates a new UptimeStats with the given details.
func NewUptimeStats(orgID uuid.UUID, componentType ComponentType, componentName string, periodStart, periodEnd time.Time) *UptimeStats {
	totalSeconds := int(periodEnd.Sub(periodStart).Seconds())
	return &UptimeStats{
		ID:              uuid.New(),
		OrgID:           orgID,
		ComponentType:   componentType,
		ComponentName:   componentName,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		TotalSeconds:    totalSeconds,
		DowntimeSeconds: 0,
		UptimePercent:   100.0,
		IncidentCount:   0,
		CreatedAt:       time.Now(),
	}
}

// CalculateUptime recalculates the uptime percentage.
func (u *UptimeStats) CalculateUptime() {
	if u.TotalSeconds > 0 {
		uptimeSeconds := u.TotalSeconds - u.DowntimeSeconds
		u.UptimePercent = float64(uptimeSeconds) / float64(u.TotalSeconds) * 100.0
	}
}

// UptimeBadge represents an uptime badge for display.
type UptimeBadge struct {
	ID            uuid.UUID      `json:"id"`
	OrgID         uuid.UUID      `json:"org_id"`
	ComponentType *ComponentType `json:"component_type,omitempty"`
	ComponentID   *uuid.UUID     `json:"component_id,omitempty"`
	ComponentName *string        `json:"component_name,omitempty"`
	BadgeType     BadgeType      `json:"badge_type"`
	UptimePercent float64        `json:"uptime_percent"`
	LastUpdated   time.Time      `json:"last_updated"`
	CreatedAt     time.Time      `json:"created_at"`
}

// NewUptimeBadge creates a new UptimeBadge with the given details.
func NewUptimeBadge(orgID uuid.UUID, badgeType BadgeType, uptimePercent float64) *UptimeBadge {
	now := time.Now()
	return &UptimeBadge{
		ID:            uuid.New(),
		OrgID:         orgID,
		BadgeType:     badgeType,
		UptimePercent: uptimePercent,
		LastUpdated:   now,
		CreatedAt:     now,
	}
}

// SetComponent sets the component reference for this badge.
func (b *UptimeBadge) SetComponent(componentType ComponentType, componentID uuid.UUID, componentName string) {
	b.ComponentType = &componentType
	b.ComponentID = &componentID
	b.ComponentName = &componentName
}

// BadgeColor returns a color indicator based on uptime percentage.
func (b *UptimeBadge) BadgeColor() string {
	switch {
	case b.UptimePercent >= 99.9:
		return "green"
	case b.UptimePercent >= 99.0:
		return "yellow"
	case b.UptimePercent >= 95.0:
		return "orange"
	default:
		return "red"
	}
}

// DowntimeAlert represents a configured alert for uptime thresholds.
type DowntimeAlert struct {
	ID               uuid.UUID      `json:"id"`
	OrgID            uuid.UUID      `json:"org_id"`
	Name             string         `json:"name"`
	Enabled          bool           `json:"enabled"`
	UptimeThreshold  float64        `json:"uptime_threshold"`
	EvaluationPeriod string         `json:"evaluation_period"`
	ComponentType    *ComponentType `json:"component_type,omitempty"`
	NotifyOnBreach   bool           `json:"notify_on_breach"`
	NotifyOnRecovery bool           `json:"notify_on_recovery"`
	LastTriggeredAt  *time.Time     `json:"last_triggered_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// NewDowntimeAlert creates a new DowntimeAlert with the given details.
func NewDowntimeAlert(orgID uuid.UUID, name string, threshold float64, period string) *DowntimeAlert {
	now := time.Now()
	return &DowntimeAlert{
		ID:               uuid.New(),
		OrgID:            orgID,
		Name:             name,
		Enabled:          true,
		UptimeThreshold:  threshold,
		EvaluationPeriod: period,
		NotifyOnBreach:   true,
		NotifyOnRecovery: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// UptimeSummary provides a summary of uptime statistics for the dashboard.
type UptimeSummary struct {
	TotalComponents    int                `json:"total_components"`
	ComponentsUp       int                `json:"components_up"`
	ComponentsDown     int                `json:"components_down"`
	ActiveIncidents    int                `json:"active_incidents"`
	OverallUptime7D    float64            `json:"overall_uptime_7d"`
	OverallUptime30D   float64            `json:"overall_uptime_30d"`
	OverallUptime90D   float64            `json:"overall_uptime_90d"`
	Badges             []*UptimeBadge     `json:"badges,omitempty"`
	RecentIncidents    []*DowntimeEvent   `json:"recent_incidents,omitempty"`
	ComponentBreakdown []ComponentUptime  `json:"component_breakdown,omitempty"`
}

// ComponentUptime represents uptime statistics for a specific component.
type ComponentUptime struct {
	ComponentType   ComponentType `json:"component_type"`
	ComponentID     *uuid.UUID    `json:"component_id,omitempty"`
	ComponentName   string        `json:"component_name"`
	Status          string        `json:"status"` // "up", "down", "degraded"
	UptimePercent7D float64       `json:"uptime_percent_7d"`
	UptimePercent30D float64      `json:"uptime_percent_30d"`
	IncidentCount30D int          `json:"incident_count_30d"`
	LastIncidentAt   *time.Time   `json:"last_incident_at,omitempty"`
}

// MonthlyUptimeReport represents a monthly uptime report.
type MonthlyUptimeReport struct {
	OrgID           uuid.UUID           `json:"org_id"`
	Month           string              `json:"month"` // "2024-01"
	Year            int                 `json:"year"`
	MonthNum        int                 `json:"month_num"`
	OverallUptime   float64             `json:"overall_uptime"`
	TotalDowntime   int                 `json:"total_downtime_seconds"`
	IncidentCount   int                 `json:"incident_count"`
	MostAffected    []ComponentUptime   `json:"most_affected,omitempty"`
	DailyBreakdown  []DailyUptime       `json:"daily_breakdown,omitempty"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// DailyUptime represents uptime statistics for a single day.
type DailyUptime struct {
	Date            string  `json:"date"` // "2024-01-15"
	UptimePercent   float64 `json:"uptime_percent"`
	DowntimeSeconds int     `json:"downtime_seconds"`
	IncidentCount   int     `json:"incident_count"`
}
