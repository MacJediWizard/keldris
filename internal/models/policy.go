package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Policy represents a backup policy template that can be applied to schedules.
type Policy struct {
	ID               uuid.UUID        `json:"id"`
	OrgID            uuid.UUID        `json:"org_id"`
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	Paths            []string         `json:"paths,omitempty"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"` // Upload limit in KB/s
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`      // Allowed backup time window
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`     // Hours (0-23) when backups should not run
	CronExpression   string           `json:"cron_expression,omitempty"`    // Default schedule cron expression
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// NewPolicy creates a new Policy with the given details.
func NewPolicy(orgID uuid.UUID, name string) *Policy {
	now := time.Now()
	return &Policy{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetPaths sets the paths from JSON bytes.
func (p *Policy) SetPaths(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &p.Paths)
}

// PathsJSON returns the paths as JSON bytes for database storage.
func (p *Policy) PathsJSON() ([]byte, error) {
	if p.Paths == nil {
		return nil, nil
	}
	return json.Marshal(p.Paths)
}

// SetExcludes sets the excludes from JSON bytes.
func (p *Policy) SetExcludes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &p.Excludes)
}

// ExcludesJSON returns the excludes as JSON bytes for database storage.
func (p *Policy) ExcludesJSON() ([]byte, error) {
	if p.Excludes == nil {
		return nil, nil
	}
	return json.Marshal(p.Excludes)
}

// SetRetentionPolicy sets the retention policy from JSON bytes.
func (p *Policy) SetRetentionPolicy(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var policy RetentionPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return err
	}
	p.RetentionPolicy = &policy
	return nil
}

// RetentionPolicyJSON returns the retention policy as JSON bytes for database storage.
func (p *Policy) RetentionPolicyJSON() ([]byte, error) {
	if p.RetentionPolicy == nil {
		return nil, nil
	}
	return json.Marshal(p.RetentionPolicy)
}

// SetBackupWindow sets the backup window from database time strings.
func (p *Policy) SetBackupWindow(startTime, endTime *string) {
	if startTime == nil && endTime == nil {
		p.BackupWindow = nil
		return
	}
	p.BackupWindow = &BackupWindow{}
	if startTime != nil {
		p.BackupWindow.Start = *startTime
	}
	if endTime != nil {
		p.BackupWindow.End = *endTime
	}
}

// SetExcludedHours sets the excluded hours from JSON bytes.
func (p *Policy) SetExcludedHours(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &p.ExcludedHours)
}

// ExcludedHoursJSON returns the excluded hours as JSON bytes for database storage.
func (p *Policy) ExcludedHoursJSON() ([]byte, error) {
	if p.ExcludedHours == nil {
		return nil, nil
	}
	return json.Marshal(p.ExcludedHours)
}

// ApplyToSchedule applies this policy's configuration to a schedule.
// It copies all policy values to the schedule, overwriting existing values.
func (p *Policy) ApplyToSchedule(s *Schedule) {
	s.PolicyID = &p.ID

	if p.Paths != nil {
		s.Paths = make([]string, len(p.Paths))
		copy(s.Paths, p.Paths)
	}

	if p.Excludes != nil {
		s.Excludes = make([]string, len(p.Excludes))
		copy(s.Excludes, p.Excludes)
	}

	if p.RetentionPolicy != nil {
		s.RetentionPolicy = &RetentionPolicy{
			KeepLast:    p.RetentionPolicy.KeepLast,
			KeepHourly:  p.RetentionPolicy.KeepHourly,
			KeepDaily:   p.RetentionPolicy.KeepDaily,
			KeepWeekly:  p.RetentionPolicy.KeepWeekly,
			KeepMonthly: p.RetentionPolicy.KeepMonthly,
			KeepYearly:  p.RetentionPolicy.KeepYearly,
		}
	}

	if p.BandwidthLimitKB != nil {
		val := *p.BandwidthLimitKB
		s.BandwidthLimitKB = &val
	}

	if p.BackupWindow != nil {
		s.BackupWindow = &BackupWindow{
			Start: p.BackupWindow.Start,
			End:   p.BackupWindow.End,
		}
	}

	if p.ExcludedHours != nil {
		s.ExcludedHours = make([]int, len(p.ExcludedHours))
		copy(s.ExcludedHours, p.ExcludedHours)
	}

	if p.CronExpression != "" {
		s.CronExpression = p.CronExpression
	}
}
