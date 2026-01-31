package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// System Settings methods

// GetOrgSetting returns a system setting by organization and key.
func (db *DB) GetOrgSetting(ctx context.Context, orgID uuid.UUID, key settings.SettingKey) (*settings.SystemSetting, error) {
	var s settings.SystemSetting
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, setting_key, setting_value, description, created_at, updated_at
		FROM system_settings
		WHERE org_id = $1 AND setting_key = $2
	`, orgID, string(key)).Scan(
		&s.ID, &s.OrgID, &s.Key, &s.Value, &s.Description, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("setting not found: %s", key)
		}
		return nil, fmt.Errorf("get system setting: %w", err)
	}
	return &s, nil
}

// GetAllOrgSettings returns all system settings for an organization.
func (db *DB) GetAllOrgSettings(ctx context.Context, orgID uuid.UUID) ([]*settings.SystemSetting, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, setting_key, setting_value, description, created_at, updated_at
		FROM system_settings
		WHERE org_id = $1
		ORDER BY setting_key
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list system settings: %w", err)
	}
	defer rows.Close()

	var result []*settings.SystemSetting
	for rows.Next() {
		var s settings.SystemSetting
		err := rows.Scan(
			&s.ID, &s.OrgID, &s.Key, &s.Value, &s.Description, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan system setting: %w", err)
		}
		result = append(result, &s)
	}

	return result, nil
}

// UpsertSystemSetting creates or updates a system setting.
func (db *DB) UpsertSystemSetting(ctx context.Context, s *settings.SystemSetting) error {
	s.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_settings (id, org_id, setting_key, setting_value, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, setting_key) DO UPDATE SET
			setting_value = EXCLUDED.setting_value,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
	`, s.ID, s.OrgID, string(s.Key), s.Value, s.Description, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert system setting: %w", err)
	}
	return nil
}

// GetSMTPSettings returns SMTP settings for an organization.
func (db *DB) GetSMTPSettings(ctx context.Context, orgID uuid.UUID) (*settings.SMTPSettings, error) {
	setting, err := db.GetOrgSetting(ctx, orgID, settings.SettingKeySMTP)
	if err != nil {
		// Return defaults if not found
		defaults := settings.DefaultSMTPSettings()
		return &defaults, nil
	}

	var smtp settings.SMTPSettings
	if err := json.Unmarshal(setting.Value, &smtp); err != nil {
		return nil, fmt.Errorf("unmarshal SMTP settings: %w", err)
	}

	return &smtp, nil
}

// UpdateSMTPSettings updates SMTP settings for an organization.
func (db *DB) UpdateSMTPSettings(ctx context.Context, orgID uuid.UUID, smtp *settings.SMTPSettings) error {
	value, err := json.Marshal(smtp)
	if err != nil {
		return fmt.Errorf("marshal SMTP settings: %w", err)
	}

	s := settings.NewSystemSetting(orgID, settings.SettingKeySMTP, "SMTP email server configuration")
	s.Value = value

	return db.UpsertSystemSetting(ctx, s)
}

// GetOIDCSettings returns OIDC settings for an organization.
func (db *DB) GetOIDCSettings(ctx context.Context, orgID uuid.UUID) (*settings.OIDCSettings, error) {
	setting, err := db.GetOrgSetting(ctx, orgID, settings.SettingKeyOIDC)
	if err != nil {
		// Return defaults if not found
		defaults := settings.DefaultOIDCSettings()
		return &defaults, nil
	}

	var oidc settings.OIDCSettings
	if err := json.Unmarshal(setting.Value, &oidc); err != nil {
		return nil, fmt.Errorf("unmarshal OIDC settings: %w", err)
	}

	return &oidc, nil
}

// UpdateOIDCSettings updates OIDC settings for an organization.
func (db *DB) UpdateOIDCSettings(ctx context.Context, orgID uuid.UUID, oidc *settings.OIDCSettings) error {
	value, err := json.Marshal(oidc)
	if err != nil {
		return fmt.Errorf("marshal OIDC settings: %w", err)
	}

	s := settings.NewSystemSetting(orgID, settings.SettingKeyOIDC, "OIDC single sign-on configuration")
	s.Value = value

	return db.UpsertSystemSetting(ctx, s)
}

// GetStorageDefaultSettings returns storage default settings for an organization.
func (db *DB) GetStorageDefaultSettings(ctx context.Context, orgID uuid.UUID) (*settings.StorageDefaultSettings, error) {
	setting, err := db.GetOrgSetting(ctx, orgID, settings.SettingKeyStorageDefaults)
	if err != nil {
		// Return defaults if not found
		defaults := settings.DefaultStorageSettings()
		return &defaults, nil
	}

	var storage settings.StorageDefaultSettings
	if err := json.Unmarshal(setting.Value, &storage); err != nil {
		return nil, fmt.Errorf("unmarshal storage settings: %w", err)
	}

	return &storage, nil
}

// UpdateStorageDefaultSettings updates storage default settings for an organization.
func (db *DB) UpdateStorageDefaultSettings(ctx context.Context, orgID uuid.UUID, storage *settings.StorageDefaultSettings) error {
	value, err := json.Marshal(storage)
	if err != nil {
		return fmt.Errorf("marshal storage settings: %w", err)
	}

	s := settings.NewSystemSetting(orgID, settings.SettingKeyStorageDefaults, "Default storage and retention policies")
	s.Value = value

	return db.UpsertSystemSetting(ctx, s)
}

// GetSecuritySettings returns security settings for an organization.
func (db *DB) GetSecuritySettings(ctx context.Context, orgID uuid.UUID) (*settings.SecuritySettings, error) {
	setting, err := db.GetOrgSetting(ctx, orgID, settings.SettingKeySecurity)
	if err != nil {
		// Return defaults if not found
		defaults := settings.DefaultSecuritySettings()
		return &defaults, nil
	}

	var security settings.SecuritySettings
	if err := json.Unmarshal(setting.Value, &security); err != nil {
		return nil, fmt.Errorf("unmarshal security settings: %w", err)
	}

	return &security, nil
}

// UpdateSecuritySettings updates security settings for an organization.
func (db *DB) UpdateSecuritySettings(ctx context.Context, orgID uuid.UUID, security *settings.SecuritySettings) error {
	value, err := json.Marshal(security)
	if err != nil {
		return fmt.Errorf("marshal security settings: %w", err)
	}

	s := settings.NewSystemSetting(orgID, settings.SettingKeySecurity, "Security and access control settings")
	s.Value = value

	return db.UpsertSystemSetting(ctx, s)
}

// GetAllSettings returns all settings as a combined response for an organization.
func (db *DB) GetAllSettings(ctx context.Context, orgID uuid.UUID) (*settings.SystemSettingsResponse, error) {
	smtp, err := db.GetSMTPSettings(ctx, orgID)
	if err != nil {
		return nil, err
	}

	oidc, err := db.GetOIDCSettings(ctx, orgID)
	if err != nil {
		return nil, err
	}

	storage, err := db.GetStorageDefaultSettings(ctx, orgID)
	if err != nil {
		return nil, err
	}

	security, err := db.GetSecuritySettings(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Get the most recent updated_at
	var updatedAt time.Time
	err = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(updated_at), NOW())
		FROM system_settings
		WHERE org_id = $1
	`, orgID).Scan(&updatedAt)
	if err != nil {
		updatedAt = time.Now()
	}

	return &settings.SystemSettingsResponse{
		SMTP:            *smtp,
		OIDC:            *oidc,
		StorageDefaults: *storage,
		Security:        *security,
		UpdatedAt:       updatedAt,
	}, nil
}

// Settings Audit Log methods

// CreateSettingsAuditLog creates a new audit log entry for a settings change.
func (db *DB) CreateSettingsAuditLog(ctx context.Context, log *settings.SettingsAuditLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO settings_audit_log (id, org_id, setting_key, old_value, new_value, changed_by, changed_at, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet)
	`, log.ID, log.OrgID, string(log.SettingKey), log.OldValue, log.NewValue, log.ChangedBy, log.ChangedAt, log.IPAddress)
	if err != nil {
		return fmt.Errorf("create settings audit log: %w", err)
	}
	return nil
}

// GetSettingsAuditLogs returns audit logs for settings changes.
func (db *DB) GetSettingsAuditLogs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*settings.SettingsAuditLog, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT sal.id, sal.org_id, sal.setting_key, sal.old_value, sal.new_value,
		       sal.changed_by, sal.changed_at, sal.ip_address, u.email
		FROM settings_audit_log sal
		LEFT JOIN users u ON sal.changed_by = u.id
		WHERE sal.org_id = $1
		ORDER BY sal.changed_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list settings audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*settings.SettingsAuditLog
	for rows.Next() {
		var log settings.SettingsAuditLog
		var ipAddress *string
		var email *string
		err := rows.Scan(
			&log.ID, &log.OrgID, &log.SettingKey, &log.OldValue, &log.NewValue,
			&log.ChangedBy, &log.ChangedAt, &ipAddress, &email,
		)
		if err != nil {
			return nil, fmt.Errorf("scan settings audit log: %w", err)
		}
		if ipAddress != nil {
			log.IPAddress = *ipAddress
		}
		if email != nil {
			log.ChangedByEmail = *email
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// EnsureSystemSettingsExist creates default system settings for an organization if they don't exist.
func (db *DB) EnsureSystemSettingsExist(ctx context.Context, orgID uuid.UUID) error {
	// Check if settings exist
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM system_settings WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("check system settings: %w", err)
	}

	// If settings already exist, return
	if count >= 4 {
		return nil
	}

	// Create default settings
	now := time.Now()

	// SMTP defaults
	smtpValue, _ := json.Marshal(settings.DefaultSMTPSettings())
	smtpSetting := &settings.SystemSetting{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         settings.SettingKeySMTP,
		Value:       smtpValue,
		Description: "SMTP email server configuration",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.UpsertSystemSetting(ctx, smtpSetting); err != nil {
		return err
	}

	// OIDC defaults
	oidcValue, _ := json.Marshal(settings.DefaultOIDCSettings())
	oidcSetting := &settings.SystemSetting{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         settings.SettingKeyOIDC,
		Value:       oidcValue,
		Description: "OIDC single sign-on configuration",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.UpsertSystemSetting(ctx, oidcSetting); err != nil {
		return err
	}

	// Storage defaults
	storageValue, _ := json.Marshal(settings.DefaultStorageSettings())
	storageSetting := &settings.SystemSetting{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         settings.SettingKeyStorageDefaults,
		Value:       storageValue,
		Description: "Default storage and retention policies",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.UpsertSystemSetting(ctx, storageSetting); err != nil {
		return err
	}

	// Security defaults
	securityValue, _ := json.Marshal(settings.DefaultSecuritySettings())
	securitySetting := &settings.SystemSetting{
		ID:          uuid.New(),
		OrgID:       orgID,
		Key:         settings.SettingKeySecurity,
		Value:       securityValue,
		Description: "Security and access control settings",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.UpsertSystemSetting(ctx, securitySetting); err != nil {
		return err
	}

	return nil
}
