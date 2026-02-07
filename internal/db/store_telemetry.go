package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/telemetry"
)

// GetTelemetrySettings retrieves the current telemetry settings.
func (db *DB) GetTelemetrySettings(ctx context.Context) (*telemetry.Settings, error) {
	var settings telemetry.Settings
	var lastData []byte
	var lastSentAt, consentGivenAt *time.Time
	var endpoint, consentVersion *string

	err := db.Pool.QueryRow(ctx, `
		SELECT enabled, install_id, endpoint, last_sent_at, last_data, consent_given_at, consent_version
		FROM telemetry_settings
		LIMIT 1
	`).Scan(
		&settings.Enabled,
		&settings.InstallID,
		&endpoint,
		&lastSentAt,
		&lastData,
		&consentGivenAt,
		&consentVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("get telemetry settings: %w", err)
	}

	settings.LastSentAt = lastSentAt
	settings.ConsentGivenAt = consentGivenAt

	if endpoint != nil {
		settings.Endpoint = *endpoint
	}
	if consentVersion != nil {
		settings.ConsentVersion = *consentVersion
	}

	if len(lastData) > 0 {
		var data telemetry.TelemetryData
		if err := json.Unmarshal(lastData, &data); err == nil {
			settings.LastData = &data
		}
	}

	return &settings, nil
}

// UpdateTelemetrySettings updates the telemetry settings.
func (db *DB) UpdateTelemetrySettings(ctx context.Context, settings *telemetry.Settings) error {
	var lastDataJSON []byte
	if settings.LastData != nil {
		var err error
		lastDataJSON, err = json.Marshal(settings.LastData)
		if err != nil {
			return fmt.Errorf("marshal last data: %w", err)
		}
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE telemetry_settings
		SET enabled = $1,
		    endpoint = $2,
		    last_sent_at = $3,
		    last_data = $4,
		    consent_given_at = $5,
		    consent_version = $6
	`, settings.Enabled, settings.Endpoint, settings.LastSentAt, lastDataJSON, settings.ConsentGivenAt, settings.ConsentVersion)
	if err != nil {
		return fmt.Errorf("update telemetry settings: %w", err)
	}

	return nil
}

// EnableTelemetry enables telemetry and records consent.
func (db *DB) EnableTelemetry(ctx context.Context, consentVersion string) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE telemetry_settings
		SET enabled = TRUE,
		    consent_given_at = $1,
		    consent_version = $2
	`, now, consentVersion)
	if err != nil {
		return fmt.Errorf("enable telemetry: %w", err)
	}
	return nil
}

// DisableTelemetry disables telemetry.
func (db *DB) DisableTelemetry(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE telemetry_settings
		SET enabled = FALSE
	`)
	if err != nil {
		return fmt.Errorf("disable telemetry: %w", err)
	}
	return nil
}

// UpdateTelemetryLastSent updates the last sent timestamp and data.
func (db *DB) UpdateTelemetryLastSent(ctx context.Context, sentAt time.Time, data *telemetry.TelemetryData) error {
	var lastDataJSON []byte
	if data != nil {
		var err error
		lastDataJSON, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal telemetry data: %w", err)
		}
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE telemetry_settings
		SET last_sent_at = $1,
		    last_data = $2
	`, sentAt, lastDataJSON)
	if err != nil {
		return fmt.Errorf("update telemetry last sent: %w", err)
	}
	return nil
}

// CollectTelemetryData gathers aggregate telemetry data from the database.
// This collects only counts and feature flags - no identifying information.
func (db *DB) CollectTelemetryData(ctx context.Context) (*telemetry.TelemetryCounts, *telemetry.TelemetryFeatures, error) {
	counts := &telemetry.TelemetryCounts{}
	features := &telemetry.TelemetryFeatures{}

	// Collect aggregate counts
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents`).Scan(&counts.TotalAgents)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count agents for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE status = 'active'`).Scan(&counts.ActiveAgents)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count active agents for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM backups`).Scan(&counts.TotalBackups)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count backups for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM backups WHERE status = 'success'`).Scan(&counts.SuccessfulBackups)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count successful backups for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM repositories`).Scan(&counts.TotalRepositories)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count repositories for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM schedules WHERE enabled = TRUE`).Scan(&counts.TotalSchedules)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count schedules for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&counts.TotalOrganizations)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count organizations for telemetry")
	}

	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&counts.TotalUsers)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to count users for telemetry")
	}

	// Collect feature usage flags
	// Check if OIDC is enabled for any org
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM system_settings
			WHERE setting_key = 'oidc'
			AND (setting_value->>'enabled')::boolean = TRUE
		)
	`).Scan(&features.OIDCEnabled)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check OIDC feature for telemetry")
	}

	// Check if SMTP is enabled for any org
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM system_settings
			WHERE setting_key = 'smtp'
			AND (setting_value->>'enabled')::boolean = TRUE
		)
	`).Scan(&features.SMTPEnabled)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check SMTP feature for telemetry")
	}

	// Check if Docker backups are being used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM docker_backup_configs LIMIT 1)
	`).Scan(&features.DockerBackupsEnabled)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check Docker backups for telemetry")
	}

	// Check if geo-replication is used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM geo_replication_rules WHERE enabled = TRUE LIMIT 1)
	`).Scan(&features.GeoReplicationEnabled)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check geo-replication for telemetry")
	}

	// Check if SLA monitoring is used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM sla_policies WHERE enabled = TRUE LIMIT 1)
	`).Scan(&features.SLAMonitoringEnabled)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check SLA monitoring for telemetry")
	}

	// Check if ransomware protection features are used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM ransomware_scans LIMIT 1)
	`).Scan(&features.RansomwareProtection)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check ransomware protection for telemetry")
	}

	// Check if legal holds are used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM legal_holds LIMIT 1)
	`).Scan(&features.LegalHoldsUsed)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check legal holds for telemetry")
	}

	// Check if classifications are used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM classifications LIMIT 1)
	`).Scan(&features.ClassificationUsed)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check classifications for telemetry")
	}

	// Check if storage tiering is used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM storage_tiers LIMIT 1)
	`).Scan(&features.StorageTieringUsed)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check storage tiering for telemetry")
	}

	// Check if DR runbooks are used
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM dr_runbooks LIMIT 1)
	`).Scan(&features.DRRunbooksUsed)
	if err != nil {
		db.logger.Warn().Err(err).Msg("failed to check DR runbooks for telemetry")
	}

	return counts, features, nil
}
