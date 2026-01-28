package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetDockerContainersByAgentID returns all Docker containers for an agent.
func (db *DB) GetDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, enabled, schedule,
		       cron_expression, excludes, pre_hook, post_hook, stop_on_backup,
		       backup_volumes, backup_bind_mounts, labels, overrides,
		       discovered_at, last_backup_at, created_at, updated_at
		FROM docker_containers
		WHERE agent_id = $1
		ORDER BY container_name
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list docker containers: %w", err)
	}
	defer rows.Close()

	var containers []*models.DockerContainerConfig
	for rows.Next() {
		c, err := scanDockerContainer(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, nil
}

// GetDockerContainerByID returns a Docker container by ID.
func (db *DB) GetDockerContainerByID(ctx context.Context, id uuid.UUID) (*models.DockerContainerConfig, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, enabled, schedule,
		       cron_expression, excludes, pre_hook, post_hook, stop_on_backup,
		       backup_volumes, backup_bind_mounts, labels, overrides,
		       discovered_at, last_backup_at, created_at, updated_at
		FROM docker_containers
		WHERE id = $1
	`, id)

	return scanDockerContainerRow(row)
}

// GetDockerContainerByContainerID returns a Docker container by its Docker container ID.
func (db *DB) GetDockerContainerByContainerID(ctx context.Context, agentID uuid.UUID, containerID string) (*models.DockerContainerConfig, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, enabled, schedule,
		       cron_expression, excludes, pre_hook, post_hook, stop_on_backup,
		       backup_volumes, backup_bind_mounts, labels, overrides,
		       discovered_at, last_backup_at, created_at, updated_at
		FROM docker_containers
		WHERE agent_id = $1 AND container_id = $2
	`, agentID, containerID)

	return scanDockerContainerRow(row)
}

// CreateDockerContainer creates a new Docker container configuration.
func (db *DB) CreateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error {
	excludesJSON, err := config.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	labelsJSON, err := config.LabelsJSON()
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	overridesJSON, err := config.OverridesJSON()
	if err != nil {
		return fmt.Errorf("marshal overrides: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_containers (
			id, agent_id, container_id, container_name, image_name, enabled, schedule,
			cron_expression, excludes, pre_hook, post_hook, stop_on_backup,
			backup_volumes, backup_bind_mounts, labels, overrides,
			discovered_at, last_backup_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`,
		config.ID, config.AgentID, config.ContainerID, config.ContainerName, config.ImageName,
		config.Enabled, string(config.Schedule), config.CronExpression, excludesJSON,
		config.PreHook, config.PostHook, config.StopOnBackup, config.BackupVolumes,
		config.BackupBindMounts, labelsJSON, overridesJSON,
		config.DiscoveredAt, config.LastBackupAt, config.CreatedAt, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create docker container: %w", err)
	}

	return nil
}

// UpdateDockerContainer updates an existing Docker container configuration.
func (db *DB) UpdateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error {
	config.UpdatedAt = time.Now()

	excludesJSON, err := config.ExcludesJSON()
	if err != nil {
		return fmt.Errorf("marshal excludes: %w", err)
	}

	labelsJSON, err := config.LabelsJSON()
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	overridesJSON, err := config.OverridesJSON()
	if err != nil {
		return fmt.Errorf("marshal overrides: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_containers SET
			container_name = $2, image_name = $3, enabled = $4, schedule = $5,
			cron_expression = $6, excludes = $7, pre_hook = $8, post_hook = $9,
			stop_on_backup = $10, backup_volumes = $11, backup_bind_mounts = $12,
			labels = $13, overrides = $14, last_backup_at = $15, updated_at = $16
		WHERE id = $1
	`,
		config.ID, config.ContainerName, config.ImageName, config.Enabled, string(config.Schedule),
		config.CronExpression, excludesJSON, config.PreHook, config.PostHook,
		config.StopOnBackup, config.BackupVolumes, config.BackupBindMounts,
		labelsJSON, overridesJSON, config.LastBackupAt, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update docker container: %w", err)
	}

	return nil
}

// DeleteDockerContainer deletes a Docker container configuration.
func (db *DB) DeleteDockerContainer(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_containers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete docker container: %w", err)
	}
	return nil
}

// scanDockerContainer scans a Docker container from a row.
func scanDockerContainer(rows pgx.Rows) (*models.DockerContainerConfig, error) {
	var c models.DockerContainerConfig
	var scheduleStr string
	var excludesBytes []byte
	var labelsBytes []byte
	var overridesBytes []byte

	err := rows.Scan(
		&c.ID, &c.AgentID, &c.ContainerID, &c.ContainerName, &c.ImageName,
		&c.Enabled, &scheduleStr, &c.CronExpression, &excludesBytes,
		&c.PreHook, &c.PostHook, &c.StopOnBackup, &c.BackupVolumes,
		&c.BackupBindMounts, &labelsBytes, &overridesBytes,
		&c.DiscoveredAt, &c.LastBackupAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan docker container: %w", err)
	}

	c.Schedule = models.DockerBackupSchedule(scheduleStr)

	if err := c.SetExcludes(excludesBytes); err != nil {
		return nil, fmt.Errorf("parse excludes: %w", err)
	}
	if err := c.SetLabelsFromJSON(labelsBytes); err != nil {
		return nil, fmt.Errorf("parse labels: %w", err)
	}
	if err := c.SetOverrides(overridesBytes); err != nil {
		return nil, fmt.Errorf("parse overrides: %w", err)
	}

	return &c, nil
}

// scanDockerContainerRow scans a Docker container from a single row.
func scanDockerContainerRow(row pgx.Row) (*models.DockerContainerConfig, error) {
	var c models.DockerContainerConfig
	var scheduleStr string
	var excludesBytes []byte
	var labelsBytes []byte
	var overridesBytes []byte

	err := row.Scan(
		&c.ID, &c.AgentID, &c.ContainerID, &c.ContainerName, &c.ImageName,
		&c.Enabled, &scheduleStr, &c.CronExpression, &excludesBytes,
		&c.PreHook, &c.PostHook, &c.StopOnBackup, &c.BackupVolumes,
		&c.BackupBindMounts, &labelsBytes, &overridesBytes,
		&c.DiscoveredAt, &c.LastBackupAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan docker container: %w", err)
	}

	c.Schedule = models.DockerBackupSchedule(scheduleStr)

	if err := c.SetExcludes(excludesBytes); err != nil {
		return nil, fmt.Errorf("parse excludes: %w", err)
	}
	if err := c.SetLabelsFromJSON(labelsBytes); err != nil {
		return nil, fmt.Errorf("parse labels: %w", err)
	}
	if err := c.SetOverrides(overridesBytes); err != nil {
		return nil, fmt.Errorf("parse overrides: %w", err)
	}

	return &c, nil
}

// GetEnabledDockerContainersForBackup returns all enabled Docker containers ready for backup on an agent.
func (db *DB) GetEnabledDockerContainersForBackup(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, agent_id, container_id, container_name, image_name, enabled, schedule,
		       cron_expression, excludes, pre_hook, post_hook, stop_on_backup,
		       backup_volumes, backup_bind_mounts, labels, overrides,
		       discovered_at, last_backup_at, created_at, updated_at
		FROM docker_containers
		WHERE agent_id = $1 AND enabled = true
		ORDER BY container_name
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list enabled docker containers: %w", err)
	}
	defer rows.Close()

	var containers []*models.DockerContainerConfig
	for rows.Next() {
		c, err := scanDockerContainer(rows)
		if err != nil {
			return nil, err
		}
		// Apply overrides to get effective configuration
		c.ApplyOverrides()
		// Only include if still enabled after overrides
		if c.Enabled {
			containers = append(containers, c)
		}
	}

	return containers, nil
}

// UpdateDockerContainerLastBackup updates the last backup time for a Docker container.
func (db *DB) UpdateDockerContainerLastBackup(ctx context.Context, id uuid.UUID, backupTime time.Time) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_containers SET last_backup_at = $2, updated_at = $3 WHERE id = $1
	`, id, backupTime, time.Now())
	if err != nil {
		return fmt.Errorf("update docker container last backup: %w", err)
	}
	return nil
}

// CountDockerContainersByAgentID returns the count of Docker containers for an agent.
func (db *DB) CountDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM docker_containers WHERE agent_id = $1
	`, agentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count docker containers: %w", err)
	}
	return count, nil
}

// CountEnabledDockerContainersByAgentID returns the count of enabled Docker containers for an agent.
func (db *DB) CountEnabledDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM docker_containers WHERE agent_id = $1 AND enabled = true
	`, agentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count enabled docker containers: %w", err)
	}
	return count, nil
}

// DeleteDockerContainersByAgentID deletes all Docker containers for an agent.
func (db *DB) DeleteDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_containers WHERE agent_id = $1`, agentID)
	if err != nil {
		return fmt.Errorf("delete docker containers by agent: %w", err)
	}
	return nil
}
