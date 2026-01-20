// Package backup provides retention policy enforcement for Restic backups.
package backup

import (
	"context"
	"errors"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// RetentionEnforcer applies retention policies after successful backups.
type RetentionEnforcer struct {
	restic *Restic
	logger zerolog.Logger
}

// NewRetentionEnforcer creates a new RetentionEnforcer.
func NewRetentionEnforcer(restic *Restic, logger zerolog.Logger) *RetentionEnforcer {
	return &RetentionEnforcer{
		restic: restic,
		logger: logger.With().Str("component", "retention").Logger(),
	}
}

// RetentionResult contains the results of a retention enforcement operation.
type RetentionResult struct {
	Applied          bool     `json:"applied"`
	SnapshotsRemoved int      `json:"snapshots_removed"`
	SnapshotsKept    int      `json:"snapshots_kept"`
	RemovedIDs       []string `json:"removed_ids,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// ApplyPolicy enforces the retention policy on a repository after a successful backup.
// It removes old snapshots according to the policy and optionally prunes unreferenced data.
func (r *RetentionEnforcer) ApplyPolicy(ctx context.Context, cfg ResticConfig, policy *models.RetentionPolicy, prune bool) (*RetentionResult, error) {
	if policy == nil {
		return &RetentionResult{Applied: false}, nil
	}

	if err := ValidateRetentionPolicy(policy); err != nil {
		return &RetentionResult{
			Applied: false,
			Error:   err.Error(),
		}, err
	}

	r.logger.Info().
		Interface("policy", policy).
		Bool("prune", prune).
		Msg("applying retention policy")

	var result *ForgetResult
	var err error

	if prune {
		result, err = r.restic.Prune(ctx, cfg, policy)
	} else {
		result, err = r.restic.Forget(ctx, cfg, policy)
	}

	if err != nil {
		r.logger.Error().Err(err).Msg("failed to apply retention policy")
		return &RetentionResult{
			Applied: false,
			Error:   err.Error(),
		}, err
	}

	retentionResult := &RetentionResult{
		Applied:          true,
		SnapshotsRemoved: result.SnapshotsRemoved,
		SnapshotsKept:    result.SnapshotsKept,
		RemovedIDs:       result.RemovedIDs,
	}

	r.logger.Info().
		Int("snapshots_removed", result.SnapshotsRemoved).
		Int("snapshots_kept", result.SnapshotsKept).
		Msg("retention policy applied successfully")

	return retentionResult, nil
}

// ValidateRetentionPolicy validates a retention policy configuration.
func ValidateRetentionPolicy(policy *models.RetentionPolicy) error {
	if policy == nil {
		return errors.New("retention policy is nil")
	}

	// At least one retention rule must be set
	if policy.KeepLast <= 0 &&
		policy.KeepHourly <= 0 &&
		policy.KeepDaily <= 0 &&
		policy.KeepWeekly <= 0 &&
		policy.KeepMonthly <= 0 &&
		policy.KeepYearly <= 0 {
		return errors.New("at least one retention rule must be specified")
	}

	// Validate individual values are non-negative
	if policy.KeepLast < 0 {
		return errors.New("keep_last cannot be negative")
	}
	if policy.KeepHourly < 0 {
		return errors.New("keep_hourly cannot be negative")
	}
	if policy.KeepDaily < 0 {
		return errors.New("keep_daily cannot be negative")
	}
	if policy.KeepWeekly < 0 {
		return errors.New("keep_weekly cannot be negative")
	}
	if policy.KeepMonthly < 0 {
		return errors.New("keep_monthly cannot be negative")
	}
	if policy.KeepYearly < 0 {
		return errors.New("keep_yearly cannot be negative")
	}

	// Warn about potentially aggressive policies (but don't error)
	// Minimum recommended: keep at least 1 daily
	if policy.KeepLast == 0 && policy.KeepDaily == 0 {
		// This is valid but risky - all backups could be removed
	}

	return nil
}

// ParseRetentionConfig parses retention configuration from a map.
// This is useful for parsing configuration from JSON or environment variables.
func ParseRetentionConfig(cfg map[string]int) (*models.RetentionPolicy, error) {
	policy := &models.RetentionPolicy{}

	if v, ok := cfg["keep_last"]; ok {
		policy.KeepLast = v
	}
	if v, ok := cfg["keep_hourly"]; ok {
		policy.KeepHourly = v
	}
	if v, ok := cfg["keep_daily"]; ok {
		policy.KeepDaily = v
	}
	if v, ok := cfg["keep_weekly"]; ok {
		policy.KeepWeekly = v
	}
	if v, ok := cfg["keep_monthly"]; ok {
		policy.KeepMonthly = v
	}
	if v, ok := cfg["keep_yearly"]; ok {
		policy.KeepYearly = v
	}

	if err := ValidateRetentionPolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid retention config: %w", err)
	}

	return policy, nil
}

// MergeRetentionPolicy merges a partial policy into a base policy.
// Non-zero values in the override policy replace values in the base.
func MergeRetentionPolicy(base, override *models.RetentionPolicy) *models.RetentionPolicy {
	if base == nil {
		if override == nil {
			return nil
		}
		return override
	}
	if override == nil {
		return base
	}

	merged := &models.RetentionPolicy{
		KeepLast:    base.KeepLast,
		KeepHourly:  base.KeepHourly,
		KeepDaily:   base.KeepDaily,
		KeepWeekly:  base.KeepWeekly,
		KeepMonthly: base.KeepMonthly,
		KeepYearly:  base.KeepYearly,
	}

	if override.KeepLast > 0 {
		merged.KeepLast = override.KeepLast
	}
	if override.KeepHourly > 0 {
		merged.KeepHourly = override.KeepHourly
	}
	if override.KeepDaily > 0 {
		merged.KeepDaily = override.KeepDaily
	}
	if override.KeepWeekly > 0 {
		merged.KeepWeekly = override.KeepWeekly
	}
	if override.KeepMonthly > 0 {
		merged.KeepMonthly = override.KeepMonthly
	}
	if override.KeepYearly > 0 {
		merged.KeepYearly = override.KeepYearly
	}

	return merged
}

// RetentionPolicyDescription returns a human-readable description of the policy.
func RetentionPolicyDescription(policy *models.RetentionPolicy) string {
	if policy == nil {
		return "No retention policy"
	}

	parts := []string{}
	if policy.KeepLast > 0 {
		parts = append(parts, fmt.Sprintf("last %d", policy.KeepLast))
	}
	if policy.KeepHourly > 0 {
		parts = append(parts, fmt.Sprintf("%d hourly", policy.KeepHourly))
	}
	if policy.KeepDaily > 0 {
		parts = append(parts, fmt.Sprintf("%d daily", policy.KeepDaily))
	}
	if policy.KeepWeekly > 0 {
		parts = append(parts, fmt.Sprintf("%d weekly", policy.KeepWeekly))
	}
	if policy.KeepMonthly > 0 {
		parts = append(parts, fmt.Sprintf("%d monthly", policy.KeepMonthly))
	}
	if policy.KeepYearly > 0 {
		parts = append(parts, fmt.Sprintf("%d yearly", policy.KeepYearly))
	}

	if len(parts) == 0 {
		return "Empty retention policy"
	}

	result := "Keep: "
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		result += part
	}
	return result
}
