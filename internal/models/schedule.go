package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy defines how long backups are retained.
type RetentionPolicy struct {
	KeepLast    int `json:"keep_last,omitempty"`
	KeepHourly  int `json:"keep_hourly,omitempty"`
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

// BackupWindow represents a time window during which backups are allowed.
type BackupWindow struct {
	Start string `json:"start,omitempty"` // HH:MM format (e.g., "02:00")
	End   string `json:"end,omitempty"`   // HH:MM format (e.g., "06:00")
}

// Schedule represents a backup schedule configuration.
type Schedule struct {
	ID               uuid.UUID        `json:"id"`
	AgentID          uuid.UUID        `json:"agent_id"`
	RepositoryID     uuid.UUID        `json:"repository_id"`
	Name             string           `json:"name"`
	CronExpression   string           `json:"cron_expression"`
	Paths            []string         `json:"paths"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"` // Upload limit in KB/s
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`      // Allowed backup time window
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`     // Hours (0-23) when backups should not run
	Enabled          bool             `json:"enabled"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// NewSchedule creates a new Schedule with the given details.
func NewSchedule(agentID, repositoryID uuid.UUID, name, cronExpr string, paths []string) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		RepositoryID:   repositoryID,
		Name:           name,
		CronExpression: cronExpr,
		Paths:          paths,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetPaths sets the paths from JSON bytes.
func (s *Schedule) SetPaths(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Paths)
}

// PathsJSON returns the paths as JSON bytes for database storage.
func (s *Schedule) PathsJSON() ([]byte, error) {
	if s.Paths == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s.Paths)
}

// SetExcludes sets the excludes from JSON bytes.
func (s *Schedule) SetExcludes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Excludes)
}

// ExcludesJSON returns the excludes as JSON bytes for database storage.
func (s *Schedule) ExcludesJSON() ([]byte, error) {
	if s.Excludes == nil {
		return nil, nil
	}
	return json.Marshal(s.Excludes)
}

// SetRetentionPolicy sets the retention policy from JSON bytes.
func (s *Schedule) SetRetentionPolicy(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var policy RetentionPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return err
	}
	s.RetentionPolicy = &policy
	return nil
}

// RetentionPolicyJSON returns the retention policy as JSON bytes for database storage.
func (s *Schedule) RetentionPolicyJSON() ([]byte, error) {
	if s.RetentionPolicy == nil {
		return nil, nil
	}
	return json.Marshal(s.RetentionPolicy)
}

// DefaultRetentionPolicy returns a sensible default retention policy.
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		KeepLast:    5,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
	}
}

// SetBackupWindow sets the backup window from JSON bytes.
func (s *Schedule) SetBackupWindow(startTime, endTime *string) {
	if startTime == nil && endTime == nil {
		s.BackupWindow = nil
		return
	}
	s.BackupWindow = &BackupWindow{}
	if startTime != nil {
		s.BackupWindow.Start = *startTime
	}
	if endTime != nil {
		s.BackupWindow.End = *endTime
	}
}

// SetExcludedHours sets the excluded hours from JSON bytes.
func (s *Schedule) SetExcludedHours(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.ExcludedHours)
}

// ExcludedHoursJSON returns the excluded hours as JSON bytes for database storage.
func (s *Schedule) ExcludedHoursJSON() ([]byte, error) {
	if s.ExcludedHours == nil {
		return nil, nil
	}
	return json.Marshal(s.ExcludedHours)
}

// IsWithinBackupWindow checks if the given time is within the allowed backup window.
// Returns true if no window is set (always allowed) or if the time is within the window.
func (s *Schedule) IsWithinBackupWindow(t time.Time) bool {
	if s.BackupWindow == nil || (s.BackupWindow.Start == "" && s.BackupWindow.End == "") {
		return true
	}

	currentTime := t.Format("15:04")
	start := s.BackupWindow.Start
	end := s.BackupWindow.End

	// Handle window that doesn't cross midnight (e.g., 02:00 to 06:00)
	if start <= end {
		return currentTime >= start && currentTime < end
	}

	// Handle window that crosses midnight (e.g., 22:00 to 06:00)
	return currentTime >= start || currentTime < end
}

// IsHourExcluded checks if the given hour is in the excluded hours list.
func (s *Schedule) IsHourExcluded(hour int) bool {
	for _, h := range s.ExcludedHours {
		if h == hour {
			return true
		}
	}
	return false
}

// CanRunAt checks if a backup can run at the given time based on window and excluded hours.
func (s *Schedule) CanRunAt(t time.Time) bool {
	if !s.IsWithinBackupWindow(t) {
		return false
	}
	if s.IsHourExcluded(t.Hour()) {
		return false
	}
	return true
}

// NextAllowedTime finds the next time when a backup can run, starting from the given time.
// Returns the input time if it's already allowed.
func (s *Schedule) NextAllowedTime(t time.Time) time.Time {
	// Check up to 24 hours ahead
	for i := 0; i < 24*60; i++ {
		checkTime := t.Add(time.Duration(i) * time.Minute)
		if s.CanRunAt(checkTime) {
			return checkTime
		}
	}
	// Fallback: return original time if no valid window found
	return t
}
