package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Trial methods

// GetTrialInfo returns the trial status for an organization.
func (db *DB) GetTrialInfo(ctx context.Context, orgID uuid.UUID) (*license.TrialInfo, error) {
	var info license.TrialInfo
	var planTier, trialStatus string
	var trialEmail *string

	err := db.Pool.QueryRow(ctx, `
		SELECT id,
			COALESCE(plan_tier::text, 'free') as plan_tier,
			COALESCE(trial_status::text, 'none') as trial_status,
			trial_started_at,
			trial_ends_at,
			trial_email,
			trial_converted_at
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(
		&info.OrgID,
		&planTier,
		&trialStatus,
		&info.TrialStartedAt,
		&info.TrialEndsAt,
		&trialEmail,
		&info.TrialConvertedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found: %s", orgID)
		}
		return nil, fmt.Errorf("get trial info: %w", err)
	}

	info.PlanTier = license.PlanTier(planTier)
	info.TrialStatus = license.TrialStatus(trialStatus)
	if trialEmail != nil {
		info.TrialEmail = *trialEmail
	}

	return &info, nil
}

// StartTrial begins a new trial for an organization.
func (db *DB) StartTrial(ctx context.Context, orgID uuid.UUID, email string) error {
	now := time.Now()
	endsAt := now.AddDate(0, 0, license.DefaultTrialDays)

	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations SET
			trial_status = 'active',
			trial_started_at = $2,
			trial_ends_at = $3,
			trial_email = $4,
			updated_at = $2
		WHERE id = $1
	`, orgID, now, endsAt, email)
	if err != nil {
		return fmt.Errorf("start trial: %w", err)
	}

	db.logger.Info().
		Str("org_id", orgID.String()).
		Time("ends_at", endsAt).
		Msg("started trial")

	return nil
}

// ExtendTrial extends an organization's trial by the specified number of days.
func (db *DB) ExtendTrial(ctx context.Context, orgID, extendedBy uuid.UUID, days int, reason string) (*license.TrialExtension, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current trial end date
	var previousEndsAt time.Time
	var trialStatus string
	err = tx.QueryRow(ctx, `
		SELECT trial_ends_at, COALESCE(trial_status::text, 'none')
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&previousEndsAt, &trialStatus)
	if err != nil {
		return nil, fmt.Errorf("get current trial end: %w", err)
	}

	// Calculate new end date - extend from current end date if in future, else from now
	var newEndsAt time.Time
	if previousEndsAt.After(time.Now()) {
		newEndsAt = previousEndsAt.AddDate(0, 0, days)
	} else {
		newEndsAt = time.Now().AddDate(0, 0, days)
	}

	// Update organization
	_, err = tx.Exec(ctx, `
		UPDATE organizations SET
			trial_ends_at = $2,
			trial_status = 'active',
			updated_at = NOW()
		WHERE id = $1
	`, orgID, newEndsAt)
	if err != nil {
		return nil, fmt.Errorf("update trial end: %w", err)
	}

	// Record extension
	extension := &license.TrialExtension{
		ID:             uuid.New(),
		OrgID:          orgID,
		ExtendedBy:     extendedBy,
		ExtensionDays:  days,
		Reason:         reason,
		PreviousEndsAt: previousEndsAt,
		NewEndsAt:      newEndsAt,
		CreatedAt:      time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO trial_extensions (id, org_id, extended_by, extension_days, reason, previous_ends_at, new_ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, extension.ID, extension.OrgID, extension.ExtendedBy, extension.ExtensionDays,
		extension.Reason, extension.PreviousEndsAt, extension.NewEndsAt, extension.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert trial extension: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	db.logger.Info().
		Str("org_id", orgID.String()).
		Int("days", days).
		Time("new_ends_at", newEndsAt).
		Str("extended_by", extendedBy.String()).
		Msg("extended trial")

	return extension, nil
}

// ConvertTrial converts a trial to a paid subscription.
func (db *DB) ConvertTrial(ctx context.Context, orgID uuid.UUID, tier license.PlanTier) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE organizations SET
			plan_tier = $2,
			trial_status = 'converted',
			trial_converted_at = $3,
			updated_at = $3
		WHERE id = $1
	`, orgID, string(tier), now)
	if err != nil {
		return fmt.Errorf("convert trial: %w", err)
	}

	db.logger.Info().
		Str("org_id", orgID.String()).
		Str("tier", string(tier)).
		Msg("converted trial to paid")

	return nil
}

// ExpireTrials marks all expired trials as expired.
func (db *DB) ExpireTrials(ctx context.Context) (int, error) {
	result, err := db.Pool.Exec(ctx, `
		UPDATE organizations SET
			trial_status = 'expired',
			updated_at = NOW()
		WHERE trial_status = 'active' AND trial_ends_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("expire trials: %w", err)
	}

	count := int(result.RowsAffected())
	if count > 0 {
		db.logger.Info().Int("count", count).Msg("expired trials")
	}

	return count, nil
}

// GetTrialExtensions returns all extensions for an organization.
func (db *DB) GetTrialExtensions(ctx context.Context, orgID uuid.UUID) ([]*license.TrialExtension, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT te.id, te.org_id, te.extended_by, u.name, te.extension_days, te.reason,
			te.previous_ends_at, te.new_ends_at, te.created_at
		FROM trial_extensions te
		LEFT JOIN users u ON te.extended_by = u.id
		WHERE te.org_id = $1
		ORDER BY te.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list trial extensions: %w", err)
	}
	defer rows.Close()

	var extensions []*license.TrialExtension
	for rows.Next() {
		var ext license.TrialExtension
		var extendedByName *string
		err := rows.Scan(
			&ext.ID, &ext.OrgID, &ext.ExtendedBy, &extendedByName, &ext.ExtensionDays,
			&ext.Reason, &ext.PreviousEndsAt, &ext.NewEndsAt, &ext.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trial extension: %w", err)
		}
		if extendedByName != nil {
			ext.ExtendedByName = *extendedByName
		}
		extensions = append(extensions, &ext)
	}

	return extensions, nil
}

// LogTrialActivity logs feature access during trial.
func (db *DB) LogTrialActivity(ctx context.Context, activity *license.TrialActivity) error {
	detailsJSON, err := json.Marshal(activity.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO trial_activity_log (id, org_id, user_id, feature_name, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, activity.ID, activity.OrgID, activity.UserID, activity.FeatureName, activity.Action, detailsJSON, activity.CreatedAt)
	if err != nil {
		return fmt.Errorf("log trial activity: %w", err)
	}

	return nil
}

// GetTrialActivity returns feature access logs for an organization.
func (db *DB) GetTrialActivity(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*license.TrialActivity, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, user_id, feature_name, action, details, created_at
		FROM trial_activity_log
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list trial activity: %w", err)
	}
	defer rows.Close()

	var activities []*license.TrialActivity
	for rows.Next() {
		var a license.TrialActivity
		var detailsJSON []byte
		err := rows.Scan(&a.ID, &a.OrgID, &a.UserID, &a.FeatureName, &a.Action, &detailsJSON, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan trial activity: %w", err)
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &a.Details)
		}
		activities = append(activities, &a)
	}

	return activities, nil
}

// GetExpiringTrials returns trials expiring within the specified number of days.
func (db *DB) GetExpiringTrials(ctx context.Context, withinDays int) ([]*license.TrialInfo, error) {
	deadline := time.Now().AddDate(0, 0, withinDays)

	rows, err := db.Pool.Query(ctx, `
		SELECT id,
			COALESCE(plan_tier::text, 'free') as plan_tier,
			COALESCE(trial_status::text, 'none') as trial_status,
			trial_started_at,
			trial_ends_at,
			trial_email,
			trial_converted_at
		FROM organizations
		WHERE trial_status = 'active' AND trial_ends_at <= $1
		ORDER BY trial_ends_at ASC
	`, deadline)
	if err != nil {
		return nil, fmt.Errorf("list expiring trials: %w", err)
	}
	defer rows.Close()

	var trials []*license.TrialInfo
	for rows.Next() {
		var info license.TrialInfo
		var planTier, trialStatus string
		var trialEmail *string

		err := rows.Scan(
			&info.OrgID, &planTier, &trialStatus,
			&info.TrialStartedAt, &info.TrialEndsAt, &trialEmail, &info.TrialConvertedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan expiring trial: %w", err)
		}

		info.PlanTier = license.PlanTier(planTier)
		info.TrialStatus = license.TrialStatus(trialStatus)
		if trialEmail != nil {
			info.TrialEmail = *trialEmail
		}
		trials = append(trials, &info)
	}

	return trials, nil
}
