package models

import (
	"encoding/json"
	"time"

	"github.com/MacJediWizard/keldris/internal/classification"
	"github.com/google/uuid"
)

// PathClassificationRule represents a path-based classification rule.
type PathClassificationRule struct {
	ID          uuid.UUID                   `json:"id"`
	OrgID       uuid.UUID                   `json:"org_id"`
	Pattern     string                      `json:"pattern"`
	Level       classification.Level        `json:"level"`
	DataTypes   []classification.DataType   `json:"data_types"`
	Description string                      `json:"description,omitempty"`
	IsBuiltin   bool                        `json:"is_builtin"`
	Priority    int                         `json:"priority"`
	Enabled     bool                        `json:"enabled"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// NewPathClassificationRule creates a new path classification rule.
func NewPathClassificationRule(orgID uuid.UUID, pattern string, level classification.Level, dataTypes []classification.DataType) *PathClassificationRule {
	now := time.Now()
	if dataTypes == nil {
		dataTypes = []classification.DataType{classification.DataTypeGeneral}
	}
	return &PathClassificationRule{
		ID:        uuid.New(),
		OrgID:     orgID,
		Pattern:   pattern,
		Level:     level,
		DataTypes: dataTypes,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetDataTypes sets the data types from JSON bytes.
func (r *PathClassificationRule) SetDataTypes(data []byte) error {
	if len(data) == 0 {
		r.DataTypes = []classification.DataType{classification.DataTypeGeneral}
		return nil
	}
	return json.Unmarshal(data, &r.DataTypes)
}

// DataTypesJSON returns the data types as JSON bytes.
func (r *PathClassificationRule) DataTypesJSON() ([]byte, error) {
	if r.DataTypes == nil {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(r.DataTypes)
}

// ToPathRule converts to a classification.PathRule for the classifier.
func (r *PathClassificationRule) ToPathRule() classification.PathRule {
	return classification.PathRule{
		Pattern:     r.Pattern,
		Level:       r.Level,
		DataTypes:   r.DataTypes,
		Description: r.Description,
	}
}

// ScheduleClassification represents the classification assigned to a schedule.
type ScheduleClassification struct {
	ID             uuid.UUID                   `json:"id"`
	ScheduleID     uuid.UUID                   `json:"schedule_id"`
	Level          classification.Level        `json:"level"`
	DataTypes      []classification.DataType   `json:"data_types"`
	AutoClassified bool                        `json:"auto_classified"`
	ClassifiedAt   time.Time                   `json:"classified_at"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

// NewScheduleClassification creates a new schedule classification.
func NewScheduleClassification(scheduleID uuid.UUID, level classification.Level, dataTypes []classification.DataType, auto bool) *ScheduleClassification {
	now := time.Now()
	if dataTypes == nil {
		dataTypes = []classification.DataType{classification.DataTypeGeneral}
	}
	return &ScheduleClassification{
		ID:             uuid.New(),
		ScheduleID:     scheduleID,
		Level:          level,
		DataTypes:      dataTypes,
		AutoClassified: auto,
		ClassifiedAt:   now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetDataTypes sets the data types from JSON bytes.
func (c *ScheduleClassification) SetDataTypes(data []byte) error {
	if len(data) == 0 {
		c.DataTypes = []classification.DataType{classification.DataTypeGeneral}
		return nil
	}
	return json.Unmarshal(data, &c.DataTypes)
}

// DataTypesJSON returns the data types as JSON bytes.
func (c *ScheduleClassification) DataTypesJSON() ([]byte, error) {
	if c.DataTypes == nil {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(c.DataTypes)
}

// BackupClassification represents the classification of a backup.
type BackupClassification struct {
	ID              uuid.UUID                   `json:"id"`
	BackupID        uuid.UUID                   `json:"backup_id"`
	ScheduleID      *uuid.UUID                  `json:"schedule_id,omitempty"`
	Level           classification.Level        `json:"level"`
	DataTypes       []classification.DataType   `json:"data_types"`
	PathsClassified []string                    `json:"paths_classified,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
}

// NewBackupClassification creates a new backup classification.
func NewBackupClassification(backupID uuid.UUID, scheduleID *uuid.UUID, level classification.Level, dataTypes []classification.DataType) *BackupClassification {
	if dataTypes == nil {
		dataTypes = []classification.DataType{classification.DataTypeGeneral}
	}
	return &BackupClassification{
		ID:         uuid.New(),
		BackupID:   backupID,
		ScheduleID: scheduleID,
		Level:      level,
		DataTypes:  dataTypes,
		CreatedAt:  time.Now(),
	}
}

// SetDataTypes sets the data types from JSON bytes.
func (c *BackupClassification) SetDataTypes(data []byte) error {
	if len(data) == 0 {
		c.DataTypes = []classification.DataType{classification.DataTypeGeneral}
		return nil
	}
	return json.Unmarshal(data, &c.DataTypes)
}

// DataTypesJSON returns the data types as JSON bytes.
func (c *BackupClassification) DataTypesJSON() ([]byte, error) {
	if c.DataTypes == nil {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(c.DataTypes)
}

// SetPathsClassified sets the paths from JSON bytes.
func (c *BackupClassification) SetPathsClassified(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &c.PathsClassified)
}

// PathsClassifiedJSON returns the paths as JSON bytes.
func (c *BackupClassification) PathsClassifiedJSON() ([]byte, error) {
	if c.PathsClassified == nil {
		return []byte(`[]`), nil
	}
	return json.Marshal(c.PathsClassified)
}

// ClassificationSummary provides aggregated classification statistics.
type ClassificationSummary struct {
	TotalSchedules    int            `json:"total_schedules"`
	TotalBackups      int            `json:"total_backups"`
	ByLevel           map[string]int `json:"by_level"`
	ByDataType        map[string]int `json:"by_data_type"`
	RestrictedCount   int            `json:"restricted_count"`
	ConfidentialCount int            `json:"confidential_count"`
	InternalCount     int            `json:"internal_count"`
	PublicCount       int            `json:"public_count"`
}

// ComplianceReport provides a compliance-focused view of classifications.
type ComplianceReport struct {
	GeneratedAt       time.Time                       `json:"generated_at"`
	OrgID             uuid.UUID                       `json:"org_id"`
	Summary           ClassificationSummary           `json:"summary"`
	SchedulesByLevel  map[string][]ScheduleSummary    `json:"schedules_by_level"`
	DataTypeBreakdown map[string]DataTypeStats        `json:"data_type_breakdown"`
	UnclassifiedCount int                             `json:"unclassified_count"`
}

// ScheduleSummary provides a brief schedule overview for reports.
type ScheduleSummary struct {
	ID        uuid.UUID                   `json:"id"`
	Name      string                      `json:"name"`
	Level     classification.Level        `json:"level"`
	DataTypes []classification.DataType   `json:"data_types"`
	Paths     []string                    `json:"paths"`
	AgentID   uuid.UUID                   `json:"agent_id"`
}

// DataTypeStats provides statistics for a data type.
type DataTypeStats struct {
	ScheduleCount int   `json:"schedule_count"`
	BackupCount   int   `json:"backup_count"`
	TotalSizeBytes int64 `json:"total_size_bytes,omitempty"`
}

// CreatePathClassificationRuleRequest is the request for creating a classification rule.
type CreatePathClassificationRuleRequest struct {
	Pattern     string   `json:"pattern" binding:"required"`
	Level       string   `json:"level" binding:"required"`
	DataTypes   []string `json:"data_types,omitempty"`
	Description string   `json:"description,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
}

// UpdatePathClassificationRuleRequest is the request for updating a classification rule.
type UpdatePathClassificationRuleRequest struct {
	Pattern     *string  `json:"pattern,omitempty"`
	Level       *string  `json:"level,omitempty"`
	DataTypes   []string `json:"data_types,omitempty"`
	Description *string  `json:"description,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

// SetScheduleClassificationRequest is the request for setting a schedule's classification.
type SetScheduleClassificationRequest struct {
	Level     string   `json:"level" binding:"required"`
	DataTypes []string `json:"data_types,omitempty"`
}
