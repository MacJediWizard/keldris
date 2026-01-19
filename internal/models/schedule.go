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

// Schedule represents a backup schedule configuration.
type Schedule struct {
	ID              uuid.UUID        `json:"id"`
	AgentID         uuid.UUID        `json:"agent_id"`
	RepositoryID    uuid.UUID        `json:"repository_id"`
	Name            string           `json:"name"`
	CronExpression  string           `json:"cron_expression"`
	Paths           []string         `json:"paths"`
	Excludes        []string         `json:"excludes,omitempty"`
	RetentionPolicy *RetentionPolicy `json:"retention_policy,omitempty"`
	Enabled         bool             `json:"enabled"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
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
