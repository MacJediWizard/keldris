package models

import (
	"encoding/json"
	"time"

	"github.com/MacJediWizard/keldris/internal/classification"
	"github.com/google/uuid"
)

// LifecyclePolicyStatus represents the status of a lifecycle policy.
type LifecyclePolicyStatus string

const (
	// LifecyclePolicyStatusActive indicates the policy is active and being enforced.
	LifecyclePolicyStatusActive LifecyclePolicyStatus = "active"
	// LifecyclePolicyStatusDraft indicates the policy is a draft and not enforced.
	LifecyclePolicyStatusDraft LifecyclePolicyStatus = "draft"
	// LifecyclePolicyStatusDisabled indicates the policy is disabled.
	LifecyclePolicyStatusDisabled LifecyclePolicyStatus = "disabled"
)

// RetentionDuration represents min/max retention periods.
type RetentionDuration struct {
	MinDays int `json:"min_days"`
	MaxDays int `json:"max_days"`
}

// DataTypeOverride allows different retention for specific data types.
type DataTypeOverride struct {
	DataType  classification.DataType `json:"data_type"`
	Retention RetentionDuration       `json:"retention"`
}

// ClassificationRetention defines retention rules for a classification level.
type ClassificationRetention struct {
	Level             classification.Level `json:"level"`
	Retention         RetentionDuration    `json:"retention"`
	DataTypeOverrides []DataTypeOverride   `json:"data_type_overrides,omitempty"`
}

// LifecyclePolicy represents a lifecycle policy for automatic snapshot deletion.
type LifecyclePolicy struct {
	ID          uuid.UUID                 `json:"id"`
	OrgID       uuid.UUID                 `json:"org_id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Status      LifecyclePolicyStatus     `json:"status"`
	Rules       []ClassificationRetention `json:"rules"`
	// RepositoryIDs limits the policy to specific repositories (empty = all)
	RepositoryIDs []uuid.UUID `json:"repository_ids,omitempty"`
	// ScheduleIDs limits the policy to specific schedules (empty = all)
	ScheduleIDs []uuid.UUID `json:"schedule_ids,omitempty"`
	// LastEvaluatedAt is when the policy was last evaluated.
	LastEvaluatedAt *time.Time `json:"last_evaluated_at,omitempty"`
	// LastDeletionAt is when snapshots were last deleted by this policy.
	LastDeletionAt *time.Time `json:"last_deletion_at,omitempty"`
	// DeletionCount is the total number of snapshots deleted by this policy.
	DeletionCount int64 `json:"deletion_count"`
	// BytesReclaimed is the total bytes reclaimed by this policy.
	BytesReclaimed int64     `json:"bytes_reclaimed"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewLifecyclePolicy creates a new LifecyclePolicy with the given details.
func NewLifecyclePolicy(orgID uuid.UUID, name string, createdBy uuid.UUID) *LifecyclePolicy {
	now := time.Now()
	return &LifecyclePolicy{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Status:    LifecyclePolicyStatusDraft,
		Rules:     []ClassificationRetention{},
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetRules sets the rules from JSON bytes.
func (p *LifecyclePolicy) SetRules(data []byte) error {
	if len(data) == 0 {
		p.Rules = []ClassificationRetention{}
		return nil
	}
	return json.Unmarshal(data, &p.Rules)
}

// RulesJSON returns the rules as JSON bytes for database storage.
func (p *LifecyclePolicy) RulesJSON() ([]byte, error) {
	if p.Rules == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(p.Rules)
}

// SetRepositoryIDs sets the repository IDs from JSON bytes.
func (p *LifecyclePolicy) SetRepositoryIDs(data []byte) error {
	if len(data) == 0 {
		p.RepositoryIDs = nil
		return nil
	}
	return json.Unmarshal(data, &p.RepositoryIDs)
}

// RepositoryIDsJSON returns the repository IDs as JSON bytes for database storage.
func (p *LifecyclePolicy) RepositoryIDsJSON() ([]byte, error) {
	if p.RepositoryIDs == nil {
		return nil, nil
	}
	return json.Marshal(p.RepositoryIDs)
}

// SetScheduleIDs sets the schedule IDs from JSON bytes.
func (p *LifecyclePolicy) SetScheduleIDs(data []byte) error {
	if len(data) == 0 {
		p.ScheduleIDs = nil
		return nil
	}
	return json.Unmarshal(data, &p.ScheduleIDs)
}

// ScheduleIDsJSON returns the schedule IDs as JSON bytes for database storage.
func (p *LifecyclePolicy) ScheduleIDsJSON() ([]byte, error) {
	if p.ScheduleIDs == nil {
		return nil, nil
	}
	return json.Marshal(p.ScheduleIDs)
}

// IsActive returns true if the policy is actively enforced.
func (p *LifecyclePolicy) IsActive() bool {
	return p.Status == LifecyclePolicyStatusActive
}

// CreateLifecyclePolicyRequest represents the request body for creating a lifecycle policy.
type CreateLifecyclePolicyRequest struct {
	Name          string                    `json:"name" binding:"required"`
	Description   string                    `json:"description,omitempty"`
	Status        LifecyclePolicyStatus     `json:"status,omitempty"`
	Rules         []ClassificationRetention `json:"rules" binding:"required"`
	RepositoryIDs []string                  `json:"repository_ids,omitempty"`
	ScheduleIDs   []string                  `json:"schedule_ids,omitempty"`
}

// UpdateLifecyclePolicyRequest represents the request body for updating a lifecycle policy.
type UpdateLifecyclePolicyRequest struct {
	Name          *string                    `json:"name,omitempty"`
	Description   *string                    `json:"description,omitempty"`
	Status        *LifecyclePolicyStatus     `json:"status,omitempty"`
	Rules         *[]ClassificationRetention `json:"rules,omitempty"`
	RepositoryIDs *[]string                  `json:"repository_ids,omitempty"`
	ScheduleIDs   *[]string                  `json:"schedule_ids,omitempty"`
}

// LifecycleDryRunRequest represents a request to preview what would be deleted.
type LifecycleDryRunRequest struct {
	PolicyID string `json:"policy_id,omitempty"`
	// If PolicyID is empty, use these temporary rules for the dry run
	Rules []ClassificationRetention `json:"rules,omitempty"`
	// Limit to specific repositories
	RepositoryIDs []string `json:"repository_ids,omitempty"`
	// Limit to specific schedules
	ScheduleIDs []string `json:"schedule_ids,omitempty"`
}

// LifecycleSnapshotEvaluation represents a single snapshot evaluation.
type LifecycleSnapshotEvaluation struct {
	SnapshotID          string               `json:"snapshot_id"`
	Action              string               `json:"action"` // keep, can_delete, must_delete, hold
	Reason              string               `json:"reason"`
	SnapshotAgeDays     int                  `json:"snapshot_age_days"`
	MinRetentionDays    int                  `json:"min_retention_days"`
	MaxRetentionDays    int                  `json:"max_retention_days"`
	DaysUntilDeletable  int                  `json:"days_until_deletable"`
	DaysUntilAutoDelete int                  `json:"days_until_auto_delete"`
	ClassificationLevel classification.Level `json:"classification_level"`
	IsOnLegalHold       bool                 `json:"is_on_legal_hold"`
	SizeBytes           int64                `json:"size_bytes,omitempty"`
	SnapshotTime        time.Time            `json:"snapshot_time"`
	RepositoryID        string               `json:"repository_id"`
	ScheduleName        string               `json:"schedule_name,omitempty"`
}

// LifecycleDryRunResult represents the result of a dry-run evaluation.
type LifecycleDryRunResult struct {
	EvaluatedAt       time.Time                     `json:"evaluated_at"`
	PolicyID          string                        `json:"policy_id,omitempty"`
	TotalSnapshots    int                           `json:"total_snapshots"`
	KeepCount         int                           `json:"keep_count"`
	CanDeleteCount    int                           `json:"can_delete_count"`
	MustDeleteCount   int                           `json:"must_delete_count"`
	HoldCount         int                           `json:"hold_count"`
	TotalSizeToDelete int64                         `json:"total_size_to_delete"`
	Evaluations       []LifecycleSnapshotEvaluation `json:"evaluations"`
}

// LifecycleExecutionRequest represents a request to execute lifecycle deletion.
type LifecycleExecutionRequest struct {
	PolicyID string `json:"policy_id" binding:"required"`
	// DryRun if true, only evaluates without deleting.
	DryRun bool `json:"dry_run"`
	// SnapshotIDs if provided, only processes these specific snapshots.
	SnapshotIDs []string `json:"snapshot_ids,omitempty"`
}

// LifecycleDeletionEvent represents a single snapshot deletion event.
type LifecycleDeletionEvent struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	PolicyID     uuid.UUID `json:"policy_id"`
	SnapshotID   string    `json:"snapshot_id"`
	RepositoryID uuid.UUID `json:"repository_id"`
	Reason       string    `json:"reason"`
	SizeBytes    int64     `json:"size_bytes"`
	DeletedBy    uuid.UUID `json:"deleted_by"`
	DeletedAt    time.Time `json:"deleted_at"`
}

// NewLifecycleDeletionEvent creates a new deletion event for audit logging.
func NewLifecycleDeletionEvent(orgID, policyID, repositoryID, deletedBy uuid.UUID, snapshotID, reason string, sizeBytes int64) *LifecycleDeletionEvent {
	return &LifecycleDeletionEvent{
		ID:           uuid.New(),
		OrgID:        orgID,
		PolicyID:     policyID,
		SnapshotID:   snapshotID,
		RepositoryID: repositoryID,
		Reason:       reason,
		SizeBytes:    sizeBytes,
		DeletedBy:    deletedBy,
		DeletedAt:    time.Now(),
	}
}
