package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// CreateDockerRestore creates a new Docker restore record.
func (db *DB) CreateDockerRestore(ctx context.Context, restore *models.DockerRestore) error {
	targetJSON, err := restore.TargetJSON()
	if err != nil {
		return fmt.Errorf("marshal target: %w", err)
	}

	progressJSON, err := restore.ProgressJSON()
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	restoredVolumesJSON, err := restore.RestoredVolumesJSON()
	if err != nil {
		return fmt.Errorf("marshal restored volumes: %w", err)
	}

	warningsJSON, err := restore.WarningsJSON()
	if err != nil {
		return fmt.Errorf("marshal warnings: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_restores (
			id, org_id, agent_id, repository_id, snapshot_id,
			container_name, volume_name, new_container_name, new_volume_name,
			target, overwrite_existing, start_after_restore, verify_start,
			status, progress, restored_container_id, restored_volumes,
			start_verified, warnings, started_at, completed_at, error_message,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
	`, restore.ID, restore.OrgID, restore.AgentID, restore.RepositoryID, restore.SnapshotID,
		restore.ContainerName, restore.VolumeName, restore.NewContainerName, restore.NewVolumeName,
		targetJSON, restore.OverwriteExisting, restore.StartAfterRestore, restore.VerifyStart,
		string(restore.Status), progressJSON, restore.RestoredContainerID, restoredVolumesJSON,
		restore.StartVerified, warningsJSON, restore.StartedAt, restore.CompletedAt, restore.ErrorMessage,
		restore.CreatedAt, restore.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create docker restore: %w", err)
	}
	return nil
}

// UpdateDockerRestore updates an existing Docker restore record.
func (db *DB) UpdateDockerRestore(ctx context.Context, restore *models.DockerRestore) error {
	restore.UpdatedAt = time.Now()

	progressJSON, err := restore.ProgressJSON()
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	restoredVolumesJSON, err := restore.RestoredVolumesJSON()
	if err != nil {
		return fmt.Errorf("marshal restored volumes: %w", err)
	}

	warningsJSON, err := restore.WarningsJSON()
	if err != nil {
		return fmt.Errorf("marshal warnings: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_restores
		SET status = $2, progress = $3, restored_container_id = $4, restored_volumes = $5,
		    start_verified = $6, warnings = $7, started_at = $8, completed_at = $9,
		    error_message = $10, updated_at = $11
		WHERE id = $1
	`, restore.ID, string(restore.Status), progressJSON, restore.RestoredContainerID,
		restoredVolumesJSON, restore.StartVerified, warningsJSON, restore.StartedAt,
		restore.CompletedAt, restore.ErrorMessage, restore.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update docker restore: %w", err)
	}
	return nil
}

// GetDockerRestoreByID returns a Docker restore by its ID.
func (db *DB) GetDockerRestoreByID(ctx context.Context, id uuid.UUID) (*models.DockerRestore, error) {
	var restore models.DockerRestore
	var targetJSON, progressJSON, restoredVolumesJSON, warningsJSON []byte
	var statusStr string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id,
		       container_name, volume_name, new_container_name, new_volume_name,
		       target, overwrite_existing, start_after_restore, verify_start,
		       status, progress, restored_container_id, restored_volumes,
		       start_verified, warnings, started_at, completed_at, error_message,
		       created_at, updated_at
		FROM docker_restores
		WHERE id = $1
	`, id).Scan(
		&restore.ID, &restore.OrgID, &restore.AgentID, &restore.RepositoryID, &restore.SnapshotID,
		&restore.ContainerName, &restore.VolumeName, &restore.NewContainerName, &restore.NewVolumeName,
		&targetJSON, &restore.OverwriteExisting, &restore.StartAfterRestore, &restore.VerifyStart,
		&statusStr, &progressJSON, &restore.RestoredContainerID, &restoredVolumesJSON,
		&restore.StartVerified, &warningsJSON, &restore.StartedAt, &restore.CompletedAt, &restore.ErrorMessage,
		&restore.CreatedAt, &restore.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get docker restore by ID: %w", err)
	}

	restore.Status = models.DockerRestoreStatus(statusStr)

	if err := restore.SetTargetFromJSON(targetJSON); err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}
	if err := restore.SetProgressFromJSON(progressJSON); err != nil {
		return nil, fmt.Errorf("parse progress: %w", err)
	}
	if err := restore.SetRestoredVolumesFromJSON(restoredVolumesJSON); err != nil {
		return nil, fmt.Errorf("parse restored volumes: %w", err)
	}
	if err := restore.SetWarningsFromJSON(warningsJSON); err != nil {
		return nil, fmt.Errorf("parse warnings: %w", err)
	}

	return &restore, nil
}

// GetDockerRestoresByOrgID returns all Docker restores for an organization.
func (db *DB) GetDockerRestoresByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRestore, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id,
		       container_name, volume_name, new_container_name, new_volume_name,
		       target, overwrite_existing, start_after_restore, verify_start,
		       status, progress, restored_container_id, restored_volumes,
		       start_verified, warnings, started_at, completed_at, error_message,
		       created_at, updated_at
		FROM docker_restores
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list docker restores: %w", err)
	}
	defer rows.Close()

	return scanDockerRestores(rows)
}

// GetDockerRestoresByAgentID returns all Docker restores for an agent.
func (db *DB) GetDockerRestoresByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerRestore, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, repository_id, snapshot_id,
		       container_name, volume_name, new_container_name, new_volume_name,
		       target, overwrite_existing, start_after_restore, verify_start,
		       status, progress, restored_container_id, restored_volumes,
		       start_verified, warnings, started_at, completed_at, error_message,
		       created_at, updated_at
		FROM docker_restores
		WHERE agent_id = $1
		ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("list docker restores by agent: %w", err)
	}
	defer rows.Close()

	return scanDockerRestores(rows)
}

func scanDockerRestores(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]*models.DockerRestore, error) {
	var restores []*models.DockerRestore

	for rows.Next() {
		var restore models.DockerRestore
		var targetJSON, progressJSON, restoredVolumesJSON, warningsJSON []byte
		var statusStr string

		err := rows.Scan(
			&restore.ID, &restore.OrgID, &restore.AgentID, &restore.RepositoryID, &restore.SnapshotID,
			&restore.ContainerName, &restore.VolumeName, &restore.NewContainerName, &restore.NewVolumeName,
			&targetJSON, &restore.OverwriteExisting, &restore.StartAfterRestore, &restore.VerifyStart,
			&statusStr, &progressJSON, &restore.RestoredContainerID, &restoredVolumesJSON,
			&restore.StartVerified, &warningsJSON, &restore.StartedAt, &restore.CompletedAt, &restore.ErrorMessage,
			&restore.CreatedAt, &restore.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan docker restore: %w", err)
		}

		restore.Status = models.DockerRestoreStatus(statusStr)

		if err := restore.SetTargetFromJSON(targetJSON); err != nil {
			return nil, fmt.Errorf("parse target: %w", err)
		}
		if err := restore.SetProgressFromJSON(progressJSON); err != nil {
			return nil, fmt.Errorf("parse progress: %w", err)
		}
		if err := restore.SetRestoredVolumesFromJSON(restoredVolumesJSON); err != nil {
			return nil, fmt.Errorf("parse restored volumes: %w", err)
		}
		if err := restore.SetWarningsFromJSON(warningsJSON); err != nil {
			return nil, fmt.Errorf("parse warnings: %w", err)
		}

		restores = append(restores, &restore)
	}

	return restores, nil
}
