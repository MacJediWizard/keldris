package models

import (
	"time"

	"github.com/google/uuid"
)

// MaintenanceWindow represents a scheduled maintenance period during which backups are paused.
type MaintenanceWindow struct {
	ID                    uuid.UUID  `json:"id"`
	OrgID                 uuid.UUID  `json:"org_id"`
	Title                 string     `json:"title"`
	Message               string     `json:"message,omitempty"`
	StartsAt              time.Time  `json:"starts_at"`
	EndsAt                time.Time  `json:"ends_at"`
	NotifyBeforeMinutes   int        `json:"notify_before_minutes"`
	NotificationSent      bool       `json:"notification_sent"`
	ReadOnly              bool       `json:"read_only"`
	CountdownStartMinutes int        `json:"countdown_start_minutes"`
	EmergencyOverride     bool       `json:"emergency_override"`
	OverriddenBy          *uuid.UUID `json:"overridden_by,omitempty"`
	OverriddenAt          *time.Time `json:"overridden_at,omitempty"`
	CreatedBy             *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	Title               string     `json:"title"`
	Message             string     `json:"message,omitempty"`
	StartsAt            time.Time  `json:"starts_at"`
	EndsAt              time.Time  `json:"ends_at"`
	NotifyBeforeMinutes int        `json:"notify_before_minutes"`
	NotificationSent    bool       `json:"notification_sent"`
	CreatedBy           *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// NewMaintenanceWindow creates a new MaintenanceWindow with the given details.
func NewMaintenanceWindow(orgID uuid.UUID, title string, startsAt, endsAt time.Time) *MaintenanceWindow {
	now := time.Now()
	return &MaintenanceWindow{
		ID:                    uuid.New(),
		OrgID:                 orgID,
		Title:                 title,
		StartsAt:              startsAt,
		EndsAt:                endsAt,
		NotifyBeforeMinutes:   60,
		CountdownStartMinutes: 30,
		CreatedAt:             now,
		UpdatedAt:             now,
		ID:                  uuid.New(),
		OrgID:               orgID,
		Title:               title,
		StartsAt:            startsAt,
		EndsAt:              endsAt,
		NotifyBeforeMinutes: 60,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

// IsActive returns true if the maintenance window is currently active.
func (m *MaintenanceWindow) IsActive(now time.Time) bool {
	return now.After(m.StartsAt) && now.Before(m.EndsAt)
}

// IsUpcoming returns true if the maintenance window starts within the given duration.
func (m *MaintenanceWindow) IsUpcoming(now time.Time, within time.Duration) bool {
	notifyTime := m.StartsAt.Add(-within)
	return now.After(notifyTime) && now.Before(m.StartsAt)
}

// IsPast returns true if the maintenance window has ended.
func (m *MaintenanceWindow) IsPast(now time.Time) bool {
	return now.After(m.EndsAt)
}

// TimeUntilStart returns the duration until the maintenance window starts.
// Returns a negative duration if the window has already started.
func (m *MaintenanceWindow) TimeUntilStart(now time.Time) time.Duration {
	return m.StartsAt.Sub(now)
}

// TimeUntilEnd returns the duration until the maintenance window ends.
// Returns a negative duration if the window has already ended.
func (m *MaintenanceWindow) TimeUntilEnd(now time.Time) time.Duration {
	return m.EndsAt.Sub(now)
}

// Duration returns the length of the maintenance window.
func (m *MaintenanceWindow) Duration() time.Duration {
	return m.EndsAt.Sub(m.StartsAt)
}

// ShouldNotify returns true if notifications should be sent for this window.
// Notifications are sent when the window is upcoming and not yet notified.
func (m *MaintenanceWindow) ShouldNotify(now time.Time) bool {
	if m.NotificationSent {
		return false
	}
	within := time.Duration(m.NotifyBeforeMinutes) * time.Minute
	return m.IsUpcoming(now, within)
}

// IsReadOnlyActive returns true if read-only mode is active and not overridden.
func (m *MaintenanceWindow) IsReadOnlyActive(now time.Time) bool {
	if !m.ReadOnly {
		return false
	}
	if m.EmergencyOverride {
		return false
	}
	return m.IsActive(now)
}

// ShouldShowCountdown returns true if countdown should be displayed.
func (m *MaintenanceWindow) ShouldShowCountdown(now time.Time) bool {
	if m.CountdownStartMinutes <= 0 {
		return false
	}
	countdownStart := m.StartsAt.Add(-time.Duration(m.CountdownStartMinutes) * time.Minute)
	return now.After(countdownStart) && now.Before(m.EndsAt)
}

// TimeUntilCountdownStart returns the duration until countdown should start.
func (m *MaintenanceWindow) TimeUntilCountdownStart(now time.Time) time.Duration {
	countdownStart := m.StartsAt.Add(-time.Duration(m.CountdownStartMinutes) * time.Minute)
	return countdownStart.Sub(now)
}

// CreateMaintenanceWindowRequest is the request body for creating a maintenance window.
type CreateMaintenanceWindowRequest struct {
	Title                 string    `json:"title" binding:"required,min=1,max=255"`
	Message               string    `json:"message,omitempty"`
	StartsAt              time.Time `json:"starts_at" binding:"required"`
	EndsAt                time.Time `json:"ends_at" binding:"required"`
	NotifyBeforeMinutes   *int      `json:"notify_before_minutes,omitempty"`
	ReadOnly              *bool     `json:"read_only,omitempty"`
	CountdownStartMinutes *int      `json:"countdown_start_minutes,omitempty"`
// CreateMaintenanceWindowRequest is the request body for creating a maintenance window.
type CreateMaintenanceWindowRequest struct {
	Title                 string    `json:"title" binding:"required,min=1,max=255"`
	Message               string    `json:"message,omitempty"`
	StartsAt              time.Time `json:"starts_at" binding:"required"`
	EndsAt                time.Time `json:"ends_at" binding:"required"`
	NotifyBeforeMinutes   *int      `json:"notify_before_minutes,omitempty"`
	ReadOnly              *bool     `json:"read_only,omitempty"`
	CountdownStartMinutes *int      `json:"countdown_start_minutes,omitempty"`
}

// UpdateMaintenanceWindowRequest is the request body for updating a maintenance window.
type UpdateMaintenanceWindowRequest struct {
	Title                 *string    `json:"title,omitempty"`
	Message               *string    `json:"message,omitempty"`
	StartsAt              *time.Time `json:"starts_at,omitempty"`
	EndsAt                *time.Time `json:"ends_at,omitempty"`
	NotifyBeforeMinutes   *int       `json:"notify_before_minutes,omitempty"`
	ReadOnly              *bool      `json:"read_only,omitempty"`
	CountdownStartMinutes *int       `json:"countdown_start_minutes,omitempty"`
}

// EmergencyOverrideRequest is the request body for emergency override.
type EmergencyOverrideRequest struct {
	Override bool `json:"override" binding:"required"`
	Title               *string    `json:"title,omitempty"`
	Message             *string    `json:"message,omitempty"`
	StartsAt            *time.Time `json:"starts_at,omitempty"`
	EndsAt              *time.Time `json:"ends_at,omitempty"`
	NotifyBeforeMinutes *int       `json:"notify_before_minutes,omitempty"`
}

// MaintenanceWindowsResponse is the response for listing maintenance windows.
type MaintenanceWindowsResponse struct {
	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows"`
}

// ActiveMaintenanceResponse is the response for the active maintenance endpoint.
type ActiveMaintenanceResponse struct {
	Active        *MaintenanceWindow `json:"active"`
	Upcoming      *MaintenanceWindow `json:"upcoming"`
	ReadOnlyMode  bool               `json:"read_only_mode"`
	ShowCountdown bool               `json:"show_countdown"`
	CountdownTo   *time.Time         `json:"countdown_to,omitempty"`
	Active   *MaintenanceWindow `json:"active"`
	Upcoming *MaintenanceWindow `json:"upcoming"`
}
