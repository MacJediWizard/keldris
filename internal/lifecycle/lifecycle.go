// Package lifecycle provides snapshot lifecycle management with compliance retention.
// This enables automated deletion of snapshots while respecting minimum retention
// requirements based on data classification levels.
package lifecycle

import (
	"time"

	"github.com/MacJediWizard/keldris/internal/classification"
)

// RetentionDuration represents the minimum and maximum retention periods.
type RetentionDuration struct {
	// MinDays is the minimum retention period in days (compliance requirement).
	// Snapshots cannot be deleted before this period.
	MinDays int `json:"min_days"`
	// MaxDays is the maximum retention period in days (auto-delete).
	// Snapshots will be automatically deleted after this period.
	// Set to 0 to disable auto-delete (keep forever after min retention).
	MaxDays int `json:"max_days"`
}

// Validate checks if the retention duration is valid.
func (r RetentionDuration) Validate() bool {
	if r.MinDays < 0 || r.MaxDays < 0 {
		return false
	}
	// If max is set, it must be >= min
	if r.MaxDays > 0 && r.MaxDays < r.MinDays {
		return false
	}
	return true
}

// ClassificationRule defines retention rules based on data classification.
type ClassificationRule struct {
	// Level is the classification level this rule applies to.
	Level classification.Level `json:"level"`
	// Retention defines the retention periods for this classification level.
	Retention RetentionDuration `json:"retention"`
	// DataTypeOverrides allows different retention for specific data types.
	DataTypeOverrides map[classification.DataType]RetentionDuration `json:"data_type_overrides,omitempty"`
}

// GetRetentionForDataType returns the retention duration for a specific data type,
// falling back to the default retention if no override exists.
func (r ClassificationRule) GetRetentionForDataType(dataType classification.DataType) RetentionDuration {
	if override, ok := r.DataTypeOverrides[dataType]; ok {
		return override
	}
	return r.Retention
}

// DefaultClassificationRules returns the default retention rules per classification level.
// These align with common compliance frameworks:
// - Public: 30 days min, 90 days max (general data retention)
// - Internal: 90 days min, 365 days max (business records)
// - Confidential: 365 days min, 2555 days (7 years) max (regulatory compliance)
// - Restricted: 2555 days (7 years) min, no max (strict compliance, manual deletion only)
func DefaultClassificationRules() []ClassificationRule {
	return []ClassificationRule{
		{
			Level: classification.LevelPublic,
			Retention: RetentionDuration{
				MinDays: 30,
				MaxDays: 90,
			},
		},
		{
			Level: classification.LevelInternal,
			Retention: RetentionDuration{
				MinDays: 90,
				MaxDays: 365,
			},
		},
		{
			Level: classification.LevelConfidential,
			Retention: RetentionDuration{
				MinDays: 365,
				MaxDays: 2555, // 7 years
			},
			DataTypeOverrides: map[classification.DataType]RetentionDuration{
				// PII may require longer retention for audit trails
				classification.DataTypePII: {
					MinDays: 365,
					MaxDays: 2555,
				},
			},
		},
		{
			Level: classification.LevelRestricted,
			Retention: RetentionDuration{
				MinDays: 2555, // 7 years
				MaxDays: 0,    // No auto-delete, manual only
			},
			DataTypeOverrides: map[classification.DataType]RetentionDuration{
				// PHI requires strict HIPAA compliance
				classification.DataTypePHI: {
					MinDays: 2190, // 6 years (HIPAA minimum)
					MaxDays: 0,    // No auto-delete
				},
				// PCI may have different requirements
				classification.DataTypePCI: {
					MinDays: 365, // 1 year minimum for PCI-DSS
					MaxDays: 2555,
				},
			},
		},
	}
}

// SnapshotAction represents what action should be taken on a snapshot.
type SnapshotAction string

const (
	// ActionKeep indicates the snapshot should be kept (within min retention).
	ActionKeep SnapshotAction = "keep"
	// ActionCanDelete indicates the snapshot can be deleted (past min, before max).
	ActionCanDelete SnapshotAction = "can_delete"
	// ActionMustDelete indicates the snapshot must be deleted (past max retention).
	ActionMustDelete SnapshotAction = "must_delete"
	// ActionHold indicates the snapshot is under legal hold and cannot be deleted.
	ActionHold SnapshotAction = "hold"
)

// SnapshotEvaluation represents the lifecycle evaluation result for a snapshot.
type SnapshotEvaluation struct {
	// SnapshotID is the ID of the evaluated snapshot.
	SnapshotID string `json:"snapshot_id"`
	// Action is the recommended action for this snapshot.
	Action SnapshotAction `json:"action"`
	// Reason explains why this action was determined.
	Reason string `json:"reason"`
	// SnapshotAge is the age of the snapshot in days.
	SnapshotAge int `json:"snapshot_age_days"`
	// MinRetention is the minimum retention period that applies.
	MinRetention int `json:"min_retention_days"`
	// MaxRetention is the maximum retention period that applies (0 = no max).
	MaxRetention int `json:"max_retention_days"`
	// DaysUntilDeletable is the number of days until the snapshot can be deleted.
	// Negative if already past minimum retention.
	DaysUntilDeletable int `json:"days_until_deletable"`
	// DaysUntilAutoDelete is the number of days until auto-delete (0 if no auto-delete).
	// Negative if past auto-delete date.
	DaysUntilAutoDelete int `json:"days_until_auto_delete"`
	// ClassificationLevel is the classification level used for evaluation.
	ClassificationLevel classification.Level `json:"classification_level"`
	// IsOnLegalHold indicates if the snapshot is under legal hold.
	IsOnLegalHold bool `json:"is_on_legal_hold"`
}

// Evaluator evaluates snapshots against lifecycle policies.
type Evaluator struct {
	rules []ClassificationRule
}

// NewEvaluator creates a new lifecycle evaluator with the given rules.
func NewEvaluator(rules []ClassificationRule) *Evaluator {
	return &Evaluator{rules: rules}
}

// NewDefaultEvaluator creates an evaluator with default classification rules.
func NewDefaultEvaluator() *Evaluator {
	return NewEvaluator(DefaultClassificationRules())
}

// findRule finds the classification rule for a given level.
func (e *Evaluator) findRule(level classification.Level) *ClassificationRule {
	for i := range e.rules {
		if e.rules[i].Level == level {
			return &e.rules[i]
		}
	}
	return nil
}

// EvaluateSnapshot evaluates a single snapshot and returns the recommended action.
func (e *Evaluator) EvaluateSnapshot(
	snapshotID string,
	snapshotTime time.Time,
	classLevel classification.Level,
	dataTypes []classification.DataType,
	isOnLegalHold bool,
) SnapshotEvaluation {
	eval := SnapshotEvaluation{
		SnapshotID:          snapshotID,
		ClassificationLevel: classLevel,
		IsOnLegalHold:       isOnLegalHold,
	}

	// Calculate snapshot age
	age := time.Since(snapshotTime)
	eval.SnapshotAge = int(age.Hours() / 24)

	// If on legal hold, cannot delete regardless of policy
	if isOnLegalHold {
		eval.Action = ActionHold
		eval.Reason = "Snapshot is under legal hold and cannot be deleted"
		return eval
	}

	// Find the applicable rule
	rule := e.findRule(classLevel)
	if rule == nil {
		// No rule found, use most restrictive default (restricted level)
		rule = e.findRule(classification.LevelRestricted)
		if rule == nil {
			// Absolute fallback
			eval.Action = ActionKeep
			eval.Reason = "No lifecycle policy found, keeping by default"
			eval.MinRetention = 365
			eval.DaysUntilDeletable = 365 - eval.SnapshotAge
			return eval
		}
	}

	// Determine the applicable retention based on data types
	retention := rule.Retention
	for _, dt := range dataTypes {
		dtRetention := rule.GetRetentionForDataType(dt)
		// Use the most restrictive retention (longest min, longest max)
		if dtRetention.MinDays > retention.MinDays {
			retention.MinDays = dtRetention.MinDays
		}
		if dtRetention.MaxDays == 0 || (retention.MaxDays > 0 && dtRetention.MaxDays > retention.MaxDays) {
			retention.MaxDays = dtRetention.MaxDays
		}
	}

	eval.MinRetention = retention.MinDays
	eval.MaxRetention = retention.MaxDays
	eval.DaysUntilDeletable = retention.MinDays - eval.SnapshotAge

	if retention.MaxDays > 0 {
		eval.DaysUntilAutoDelete = retention.MaxDays - eval.SnapshotAge
	}

	// Determine action based on age and retention
	if eval.SnapshotAge < retention.MinDays {
		eval.Action = ActionKeep
		eval.Reason = "Snapshot is within minimum retention period (compliance)"
	} else if retention.MaxDays > 0 && eval.SnapshotAge >= retention.MaxDays {
		eval.Action = ActionMustDelete
		eval.Reason = "Snapshot has exceeded maximum retention period"
	} else {
		eval.Action = ActionCanDelete
		if retention.MaxDays > 0 {
			eval.Reason = "Snapshot is past minimum retention and can be deleted before max retention"
		} else {
			eval.Reason = "Snapshot is past minimum retention, no auto-delete configured"
		}
	}

	return eval
}

// DryRunResult represents the result of a dry-run lifecycle evaluation.
type DryRunResult struct {
	// EvaluatedAt is when the dry run was performed.
	EvaluatedAt time.Time `json:"evaluated_at"`
	// PolicyID is the ID of the policy used for evaluation.
	PolicyID string `json:"policy_id,omitempty"`
	// TotalSnapshots is the total number of snapshots evaluated.
	TotalSnapshots int `json:"total_snapshots"`
	// KeepCount is the number of snapshots to keep.
	KeepCount int `json:"keep_count"`
	// CanDeleteCount is the number of snapshots that can be deleted.
	CanDeleteCount int `json:"can_delete_count"`
	// MustDeleteCount is the number of snapshots that must be deleted.
	MustDeleteCount int `json:"must_delete_count"`
	// HoldCount is the number of snapshots under legal hold.
	HoldCount int `json:"hold_count"`
	// Evaluations contains the detailed evaluation for each snapshot.
	Evaluations []SnapshotEvaluation `json:"evaluations"`
	// TotalSizeToDelete is the total size of snapshots to delete (if available).
	TotalSizeToDelete int64 `json:"total_size_to_delete,omitempty"`
}

// AddEvaluation adds an evaluation to the dry run result and updates counts.
func (r *DryRunResult) AddEvaluation(eval SnapshotEvaluation, sizeBytes int64) {
	r.Evaluations = append(r.Evaluations, eval)
	r.TotalSnapshots++

	switch eval.Action {
	case ActionKeep:
		r.KeepCount++
	case ActionCanDelete:
		r.CanDeleteCount++
		r.TotalSizeToDelete += sizeBytes
	case ActionMustDelete:
		r.MustDeleteCount++
		r.TotalSizeToDelete += sizeBytes
	case ActionHold:
		r.HoldCount++
	}
}
