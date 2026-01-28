package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateDockerLogBackup creates a new docker log backup record.
func (db *DB) CreateDockerLogBackup(ctx context.Context, backup *models.DockerLogBackup) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO docker_log_backups (
			id, agent_id, container_id, container_name, image_name, log_path,
			original_size, compressed_size, compressed, start_time, end_time,
			line_count, status, error_message, backup_schedule_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, backup.ID, backup.AgentID, backup.ContainerID, backup.ContainerName, backup.ImageName,
		backup.LogPath, backup.OriginalSize, backup.CompressedSize, backup.Compressed,
		nullTime(backup.StartTime), nullTime(backup.EndTime), backup.LineCount,
		string(backup.Status), nullString(backup.ErrorMessage), backup.BackupScheduleID,
		backup.CreatedAt, backup.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create docker log backup: %w", err)
	}
	return nil
}

// UpdateDockerLogBackup updates an existing docker log backup record.
func (db *DB) UpdateDockerLogBackup(ctx context.Context, backup *models.DockerLogBackup) error {
	backup.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_log_backups SET
			log_path = $2,
			original_size = $3,
			compressed_size = $4,
			compressed = $5,
			start_time = $6,
			end_time = $7,
			line_count = $8,
			status = $9,
			error_message = $10,
			updated_at = $11
		WHERE id = $1
	`, backup.ID, backup.LogPath, backup.OriginalSize, backup.CompressedSize,
		backup.Compressed, nullTime(backup.StartTime), nullTime(backup.EndTime),
		backup.LineCount, string(backup.Status), nullString(backup.ErrorMessage), backup.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update docker log backup: %w", err)
	}
	return nil
}

// GetDockerLogBackupByID retrieves a docker log backup by ID.
func (db *DB) GetDockerLogBackupByID(ctx context.Context, id uuid.UUID) (*models.DockerLogBackup, error) {
	var backup models.DockerLogBackup
	var statusStr string
	var startTime, endTime *time.Time
	var errorMsg *string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, log_path,
			original_size, compressed_size, compressed, start_time, end_time,
			line_count, status, error_message, backup_schedule_id, created_at, updated_at
		FROM docker_log_backups
		WHERE id = $1
	`, id).Scan(
		&backup.ID, &backup.AgentID, &backup.ContainerID, &backup.ContainerName,
		&backup.ImageName, &backup.LogPath, &backup.OriginalSize, &backup.CompressedSize,
		&backup.Compressed, &startTime, &endTime, &backup.LineCount, &statusStr,
		&errorMsg, &backup.BackupScheduleID, &backup.CreatedAt, &backup.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("docker log backup not found: %s", id)
		}
		return nil, fmt.Errorf("get docker log backup: %w", err)
	}

	backup.Status = models.DockerLogBackupStatus(statusStr)
	if startTime != nil {
		backup.StartTime = *startTime
	}
	if endTime != nil {
		backup.EndTime = *endTime
	}
	if errorMsg != nil {
		backup.ErrorMessage = *errorMsg
	}

	return &backup, nil
}

// GetDockerLogBackupsByAgentID retrieves all docker log backups for an agent.
func (db *DB) GetDockerLogBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerLogBackup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, log_path,
			original_size, compressed_size, compressed, start_time, end_time,
			line_count, status, error_message, backup_schedule_id, created_at, updated_at
		FROM docker_log_backups
		WHERE agent_id = $1
		ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("get docker log backups by agent: %w", err)
	}
	defer rows.Close()

	return scanDockerLogBackups(rows)
}

// GetDockerLogBackupsByContainer retrieves backups for a specific container.
func (db *DB) GetDockerLogBackupsByContainer(ctx context.Context, agentID uuid.UUID, containerID string) ([]*models.DockerLogBackup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, log_path,
			original_size, compressed_size, compressed, start_time, end_time,
			line_count, status, error_message, backup_schedule_id, created_at, updated_at
		FROM docker_log_backups
		WHERE agent_id = $1 AND container_id = $2
		ORDER BY created_at DESC
	`, agentID, containerID)
	if err != nil {
		return nil, fmt.Errorf("get docker log backups by container: %w", err)
	}
	defer rows.Close()

	return scanDockerLogBackups(rows)
}

// GetDockerLogBackupsByOrgID retrieves all docker log backups for an organization.
func (db *DB) GetDockerLogBackupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerLogBackup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT dlb.id, dlb.agent_id, dlb.container_id, dlb.container_name, dlb.image_name,
			dlb.log_path, dlb.original_size, dlb.compressed_size, dlb.compressed,
			dlb.start_time, dlb.end_time, dlb.line_count, dlb.status, dlb.error_message,
			dlb.backup_schedule_id, dlb.created_at, dlb.updated_at
		FROM docker_log_backups dlb
		JOIN agents a ON dlb.agent_id = a.id
		WHERE a.org_id = $1
		ORDER BY dlb.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get docker log backups by org: %w", err)
	}
	defer rows.Close()

	return scanDockerLogBackups(rows)
}

// DeleteDockerLogBackup deletes a docker log backup record.
func (db *DB) DeleteDockerLogBackup(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_log_backups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete docker log backup: %w", err)
	}
	return nil
}

// DeleteOldDockerLogBackups deletes backups older than the specified age.
func (db *DB) DeleteOldDockerLogBackups(ctx context.Context, agentID uuid.UUID, olderThan time.Time) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM docker_log_backups
		WHERE agent_id = $1 AND created_at < $2
	`, agentID, olderThan)
	if err != nil {
		return 0, fmt.Errorf("delete old docker log backups: %w", err)
	}
	return result.RowsAffected(), nil
}

// scanDockerLogBackups scans rows into docker log backup models.
func scanDockerLogBackups(rows pgx.Rows) ([]*models.DockerLogBackup, error) {
	var backups []*models.DockerLogBackup

	for rows.Next() {
		var backup models.DockerLogBackup
		var statusStr string
		var startTime, endTime *time.Time
		var errorMsg *string

		err := rows.Scan(
			&backup.ID, &backup.AgentID, &backup.ContainerID, &backup.ContainerName,
			&backup.ImageName, &backup.LogPath, &backup.OriginalSize, &backup.CompressedSize,
			&backup.Compressed, &startTime, &endTime, &backup.LineCount, &statusStr,
			&errorMsg, &backup.BackupScheduleID, &backup.CreatedAt, &backup.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan docker log backup: %w", err)
		}

		backup.Status = models.DockerLogBackupStatus(statusStr)
		if startTime != nil {
			backup.StartTime = *startTime
		}
		if endTime != nil {
			backup.EndTime = *endTime
		}
		if errorMsg != nil {
			backup.ErrorMessage = *errorMsg
		}

		backups = append(backups, &backup)
	}

	return backups, nil
}

// Docker Log Settings methods

// CreateDockerLogSettings creates new docker log settings for an agent.
func (db *DB) CreateDockerLogSettings(ctx context.Context, settings *models.DockerLogSettings) error {
	retentionJSON, err := json.Marshal(settings.RetentionPolicy)
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	includeContainersJSON, err := json.Marshal(settings.IncludeContainers)
	if err != nil {
		return fmt.Errorf("marshal include containers: %w", err)
	}

	excludeContainersJSON, err := json.Marshal(settings.ExcludeContainers)
	if err != nil {
		return fmt.Errorf("marshal exclude containers: %w", err)
	}

	includeLabelsJSON, err := json.Marshal(settings.IncludeLabels)
	if err != nil {
		return fmt.Errorf("marshal include labels: %w", err)
	}

	excludeLabelsJSON, err := json.Marshal(settings.ExcludeLabels)
	if err != nil {
		return fmt.Errorf("marshal exclude labels: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_log_settings (
			id, agent_id, enabled, cron_expression, retention_policy,
			include_containers, exclude_containers, include_labels, exclude_labels,
			timestamps, tail, since, until, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, settings.ID, settings.AgentID, settings.Enabled, settings.CronExpression,
		retentionJSON, includeContainersJSON, excludeContainersJSON,
		includeLabelsJSON, excludeLabelsJSON, settings.Timestamps, settings.Tail,
		nullString(settings.Since), nullString(settings.Until), settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create docker log settings: %w", err)
	}
	return nil
}

// UpdateDockerLogSettings updates existing docker log settings.
func (db *DB) UpdateDockerLogSettings(ctx context.Context, settings *models.DockerLogSettings) error {
	settings.UpdatedAt = time.Now()

	retentionJSON, err := json.Marshal(settings.RetentionPolicy)
	if err != nil {
		return fmt.Errorf("marshal retention policy: %w", err)
	}

	includeContainersJSON, err := json.Marshal(settings.IncludeContainers)
	if err != nil {
		return fmt.Errorf("marshal include containers: %w", err)
	}

	excludeContainersJSON, err := json.Marshal(settings.ExcludeContainers)
	if err != nil {
		return fmt.Errorf("marshal exclude containers: %w", err)
	}

	includeLabelsJSON, err := json.Marshal(settings.IncludeLabels)
	if err != nil {
		return fmt.Errorf("marshal include labels: %w", err)
	}

	excludeLabelsJSON, err := json.Marshal(settings.ExcludeLabels)
	if err != nil {
		return fmt.Errorf("marshal exclude labels: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_log_settings SET
			enabled = $2,
			cron_expression = $3,
			retention_policy = $4,
			include_containers = $5,
			exclude_containers = $6,
			include_labels = $7,
			exclude_labels = $8,
			timestamps = $9,
			tail = $10,
			since = $11,
			until = $12,
			updated_at = $13
		WHERE id = $1
	`, settings.ID, settings.Enabled, settings.CronExpression, retentionJSON,
		includeContainersJSON, excludeContainersJSON, includeLabelsJSON, excludeLabelsJSON,
		settings.Timestamps, settings.Tail, nullString(settings.Since),
		nullString(settings.Until), settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update docker log settings: %w", err)
	}
	return nil
}

// GetDockerLogSettingsByAgentID retrieves docker log settings for an agent.
func (db *DB) GetDockerLogSettingsByAgentID(ctx context.Context, agentID uuid.UUID) (*models.DockerLogSettings, error) {
	var settings models.DockerLogSettings
	var retentionJSON, includeContainersJSON, excludeContainersJSON []byte
	var includeLabelsJSON, excludeLabelsJSON []byte
	var since, until *string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, enabled, cron_expression, retention_policy,
			include_containers, exclude_containers, include_labels, exclude_labels,
			timestamps, tail, since, until, created_at, updated_at
		FROM docker_log_settings
		WHERE agent_id = $1
	`, agentID).Scan(
		&settings.ID, &settings.AgentID, &settings.Enabled, &settings.CronExpression,
		&retentionJSON, &includeContainersJSON, &excludeContainersJSON,
		&includeLabelsJSON, &excludeLabelsJSON, &settings.Timestamps, &settings.Tail,
		&since, &until, &settings.CreatedAt, &settings.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found, return nil without error
		}
		return nil, fmt.Errorf("get docker log settings: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(retentionJSON, &settings.RetentionPolicy); err != nil {
		db.logger.Warn().Err(err).Msg("failed to unmarshal retention policy")
		settings.RetentionPolicy = models.DefaultDockerLogRetentionPolicy()
	}

	if includeContainersJSON != nil {
		if err := json.Unmarshal(includeContainersJSON, &settings.IncludeContainers); err != nil {
			db.logger.Warn().Err(err).Msg("failed to unmarshal include containers")
		}
	}

	if excludeContainersJSON != nil {
		if err := json.Unmarshal(excludeContainersJSON, &settings.ExcludeContainers); err != nil {
			db.logger.Warn().Err(err).Msg("failed to unmarshal exclude containers")
		}
	}

	if includeLabelsJSON != nil {
		if err := json.Unmarshal(includeLabelsJSON, &settings.IncludeLabels); err != nil {
			db.logger.Warn().Err(err).Msg("failed to unmarshal include labels")
		}
	}

	if excludeLabelsJSON != nil {
		if err := json.Unmarshal(excludeLabelsJSON, &settings.ExcludeLabels); err != nil {
			db.logger.Warn().Err(err).Msg("failed to unmarshal exclude labels")
		}
	}

	if since != nil {
		settings.Since = *since
	}
	if until != nil {
		settings.Until = *until
	}

	return &settings, nil
}

// GetOrCreateDockerLogSettings gets or creates docker log settings for an agent.
func (db *DB) GetOrCreateDockerLogSettings(ctx context.Context, agentID uuid.UUID) (*models.DockerLogSettings, error) {
	settings, err := db.GetDockerLogSettingsByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if settings != nil {
		return settings, nil
	}

	// Create default settings
	settings = models.NewDockerLogSettings(agentID)
	if err := db.CreateDockerLogSettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetEnabledDockerLogSettings returns all enabled docker log settings.
func (db *DB) GetEnabledDockerLogSettings(ctx context.Context) ([]*models.DockerLogSettings, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, enabled, cron_expression, retention_policy,
			include_containers, exclude_containers, include_labels, exclude_labels,
			timestamps, tail, since, until, created_at, updated_at
		FROM docker_log_settings
		WHERE enabled = true
	`)
	if err != nil {
		return nil, fmt.Errorf("get enabled docker log settings: %w", err)
	}
	defer rows.Close()

	var settingsList []*models.DockerLogSettings

	for rows.Next() {
		var settings models.DockerLogSettings
		var retentionJSON, includeContainersJSON, excludeContainersJSON []byte
		var includeLabelsJSON, excludeLabelsJSON []byte
		var since, until *string

		err := rows.Scan(
			&settings.ID, &settings.AgentID, &settings.Enabled, &settings.CronExpression,
			&retentionJSON, &includeContainersJSON, &excludeContainersJSON,
			&includeLabelsJSON, &excludeLabelsJSON, &settings.Timestamps, &settings.Tail,
			&since, &until, &settings.CreatedAt, &settings.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan docker log settings: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(retentionJSON, &settings.RetentionPolicy); err != nil {
			settings.RetentionPolicy = models.DefaultDockerLogRetentionPolicy()
		}

		if includeContainersJSON != nil {
			json.Unmarshal(includeContainersJSON, &settings.IncludeContainers)
		}
		if excludeContainersJSON != nil {
			json.Unmarshal(excludeContainersJSON, &settings.ExcludeContainers)
		}
		if includeLabelsJSON != nil {
			json.Unmarshal(includeLabelsJSON, &settings.IncludeLabels)
		}
		if excludeLabelsJSON != nil {
			json.Unmarshal(excludeLabelsJSON, &settings.ExcludeLabels)
		}

		if since != nil {
			settings.Since = *since
		}
		if until != nil {
			settings.Until = *until
		}

		settingsList = append(settingsList, &settings)
	}

	return settingsList, nil
}

// DeleteDockerLogSettings deletes docker log settings for an agent.
func (db *DB) DeleteDockerLogSettings(ctx context.Context, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_log_settings WHERE agent_id = $1`, agentID)
	if err != nil {
		return fmt.Errorf("delete docker log settings: %w", err)
	}
	return nil
}

// Helper functions for nullable values

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
