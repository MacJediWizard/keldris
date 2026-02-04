package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OrganizationLicense represents a license record for an organization.
type OrganizationLicense struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	Tier        license.Tier
	ActivatedAt *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LicenseAuditLog represents an audit log entry for license changes.
type LicenseAuditLog struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	UserID    *uuid.UUID
	Action    string
	OldTier   *license.Tier
	NewTier   *license.Tier
	Details   map[string]any
	CreatedAt time.Time
}

// GetOrgTier returns the license tier for an organization.
// Returns TierFree if no license record exists.
func (db *DB) GetOrgTier(ctx context.Context, orgID uuid.UUID) (license.Tier, error) {
	var tierStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT tier FROM organization_licenses
		WHERE org_id = $1
	`, orgID).Scan(&tierStr)

	if err != nil {
		if err == pgx.ErrNoRows {
			return license.TierFree, nil
		}
		return license.TierFree, fmt.Errorf("get org tier: %w", err)
	}

	return license.Tier(tierStr), nil
}

// SetOrgTier sets the license tier for an organization.
// Creates a new license record if one doesn't exist.
func (db *DB) SetOrgTier(ctx context.Context, orgID uuid.UUID, tier license.Tier) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO organization_licenses (id, org_id, tier, activated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (org_id)
		DO UPDATE SET tier = $3, updated_at = $6
	`, uuid.New(), orgID, string(tier), now, now, now)

	if err != nil {
		return fmt.Errorf("set org tier: %w", err)
	}
	return nil
}

// GetOrgLicense returns the full license record for an organization.
// Returns nil if no license record exists.
func (db *DB) GetOrgLicense(ctx context.Context, orgID uuid.UUID) (*OrganizationLicense, error) {
	var lic OrganizationLicense
	var tierStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, tier, activated_at, expires_at, created_at, updated_at
		FROM organization_licenses
		WHERE org_id = $1
	`, orgID).Scan(
		&lic.ID, &lic.OrgID, &tierStr, &lic.ActivatedAt, &lic.ExpiresAt, &lic.CreatedAt, &lic.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get org license: %w", err)
	}

	lic.Tier = license.Tier(tierStr)
	return &lic, nil
}

// CreateOrgLicense creates a new license record for an organization.
func (db *DB) CreateOrgLicense(ctx context.Context, lic *OrganizationLicense) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO organization_licenses (id, org_id, tier, activated_at, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, lic.ID, lic.OrgID, string(lic.Tier), lic.ActivatedAt, lic.ExpiresAt, lic.CreatedAt, lic.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create org license: %w", err)
	}
	return nil
}

// UpdateOrgLicense updates an existing license record.
func (db *DB) UpdateOrgLicense(ctx context.Context, lic *OrganizationLicense) error {
	lic.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE organization_licenses
		SET tier = $2, activated_at = $3, expires_at = $4, updated_at = $5
		WHERE id = $1
	`, lic.ID, string(lic.Tier), lic.ActivatedAt, lic.ExpiresAt, lic.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update org license: %w", err)
	}
	return nil
}

// CreateLicenseAuditLog creates a new license audit log entry.
func (db *DB) CreateLicenseAuditLog(ctx context.Context, log *LicenseAuditLog) error {
	var oldTierStr, newTierStr *string
	if log.OldTier != nil {
		s := string(*log.OldTier)
		oldTierStr = &s
	}
	if log.NewTier != nil {
		s := string(*log.NewTier)
		newTierStr = &s
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO license_audit_logs (id, org_id, user_id, action, old_tier, new_tier, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, log.ID, log.OrgID, log.UserID, log.Action, oldTierStr, newTierStr, log.Details, log.CreatedAt)

	if err != nil {
		return fmt.Errorf("create license audit log: %w", err)
	}
	return nil
}

// GetLicenseAuditLogs returns license audit logs for an organization.
func (db *DB) GetLicenseAuditLogs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*LicenseAuditLog, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, user_id, action, old_tier, new_tier, details, created_at
		FROM license_audit_logs
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("get license audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*LicenseAuditLog
	for rows.Next() {
		var log LicenseAuditLog
		var oldTierStr, newTierStr *string

		err := rows.Scan(
			&log.ID, &log.OrgID, &log.UserID, &log.Action, &oldTierStr, &newTierStr, &log.Details, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan license audit log: %w", err)
		}

		if oldTierStr != nil {
			t := license.Tier(*oldTierStr)
			log.OldTier = &t
		}
		if newTierStr != nil {
			t := license.Tier(*newTierStr)
			log.NewTier = &t
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// GetExpiredLicenses returns licenses that have expired.
func (db *DB) GetExpiredLicenses(ctx context.Context) ([]*OrganizationLicense, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, tier, activated_at, expires_at, created_at, updated_at
		FROM organization_licenses
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`)

	if err != nil {
		return nil, fmt.Errorf("get expired licenses: %w", err)
	}
	defer rows.Close()

	var licenses []*OrganizationLicense
	for rows.Next() {
		var lic OrganizationLicense
		var tierStr string

		err := rows.Scan(
			&lic.ID, &lic.OrgID, &tierStr, &lic.ActivatedAt, &lic.ExpiresAt, &lic.CreatedAt, &lic.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan org license: %w", err)
		}

		lic.Tier = license.Tier(tierStr)
		licenses = append(licenses, &lic)
	}

	return licenses, nil
}

// DeleteOrgLicense removes a license record for an organization.
func (db *DB) DeleteOrgLicense(ctx context.Context, orgID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM organization_licenses WHERE org_id = $1
	`, orgID)

	if err != nil {
		return fmt.Errorf("delete org license: %w", err)
	}
	return nil
}
