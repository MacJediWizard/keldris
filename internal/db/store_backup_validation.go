package db

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Backup Validation methods

// CreateBackupValidation creates a new backup validation record.
func (db *DB) CreateBackupValidation(ctx context.Context, v *models.BackupValidation) error {
	detailsBytes, err := v.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal validation details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO backup_validations (
			id, backup_id, repository_id, snapshot_id, started_at,
			completed_at, status, duration_ms, error_message, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, v.ID, v.BackupID, v.RepositoryID, v.SnapshotID, v.StartedAt,
		v.CompletedAt, string(v.Status), v.DurationMs, v.ErrorMessage, detailsBytes, v.CreatedAt)
	if err != nil {
		return fmt.Errorf("create backup validation: %w", err)
	}
	return nil
}

// UpdateBackupValidation updates an existing backup validation record.
func (db *DB) UpdateBackupValidation(ctx context.Context, v *models.BackupValidation) error {
	detailsBytes, err := v.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal validation details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE backup_validations SET
			completed_at = $2,
			status = $3,
			duration_ms = $4,
			error_message = $5,
			details = $6
		WHERE id = $1
	`, v.ID, v.CompletedAt, string(v.Status), v.DurationMs, v.ErrorMessage, detailsBytes)
	if err != nil {
		return fmt.Errorf("update backup validation: %w", err)
	}
	return nil
}

// GetBackupValidationByID returns a backup validation by its ID.
func (db *DB) GetBackupValidationByID(ctx context.Context, id uuid.UUID) (*models.BackupValidation, error) {
	var v models.BackupValidation
	var statusStr string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, backup_id, repository_id, snapshot_id, started_at,
		       completed_at, status, duration_ms, error_message, details, created_at
		FROM backup_validations
		WHERE id = $1
	`, id).Scan(
		&v.ID, &v.BackupID, &v.RepositoryID, &v.SnapshotID, &v.StartedAt,
		&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage, &detailsBytes, &v.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get backup validation: %w", err)
	}
	v.Status = models.BackupValidationStatus(statusStr)
	if err := v.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("validation_id", v.ID.String()).Msg("failed to parse validation details")
	}
	return &v, nil
}

// GetBackupValidationByBackupID returns the validation for a specific backup.
func (db *DB) GetBackupValidationByBackupID(ctx context.Context, backupID uuid.UUID) (*models.BackupValidation, error) {
	var v models.BackupValidation
	var statusStr string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, backup_id, repository_id, snapshot_id, started_at,
		       completed_at, status, duration_ms, error_message, details, created_at
		FROM backup_validations
		WHERE backup_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, backupID).Scan(
		&v.ID, &v.BackupID, &v.RepositoryID, &v.SnapshotID, &v.StartedAt,
		&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage, &detailsBytes, &v.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get backup validation by backup ID: %w", err)
	}
	v.Status = models.BackupValidationStatus(statusStr)
	if err := v.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("validation_id", v.ID.String()).Msg("failed to parse validation details")
	}
	return &v, nil
}

// GetLatestBackupValidationByRepoID returns the most recent validation for a repository.
func (db *DB) GetLatestBackupValidationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.BackupValidation, error) {
	var v models.BackupValidation
	var statusStr string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, backup_id, repository_id, snapshot_id, started_at,
		       completed_at, status, duration_ms, error_message, details, created_at
		FROM backup_validations
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, repoID).Scan(
		&v.ID, &v.BackupID, &v.RepositoryID, &v.SnapshotID, &v.StartedAt,
		&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage, &detailsBytes, &v.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest backup validation by repo ID: %w", err)
	}
	v.Status = models.BackupValidationStatus(statusStr)
	if err := v.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("validation_id", v.ID.String()).Msg("failed to parse validation details")
	}
	return &v, nil
}

// GetBackupValidationsByRepoID returns validation records for a repository.
func (db *DB) GetBackupValidationsByRepoID(ctx context.Context, repoID uuid.UUID, limit int) ([]*models.BackupValidation, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, backup_id, repository_id, snapshot_id, started_at,
		       completed_at, status, duration_ms, error_message, details, created_at
		FROM backup_validations
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list backup validations: %w", err)
	}
	defer rows.Close()

	return db.scanBackupValidations(rows)
}

// scanBackupValidations is a helper to scan multiple backup validation records.
func (db *DB) scanBackupValidations(rows pgx.Rows) ([]*models.BackupValidation, error) {
	var validations []*models.BackupValidation
	for rows.Next() {
		var v models.BackupValidation
		var statusStr string
		var detailsBytes []byte
		err := rows.Scan(
			&v.ID, &v.BackupID, &v.RepositoryID, &v.SnapshotID, &v.StartedAt,
			&v.CompletedAt, &statusStr, &v.DurationMs, &v.ErrorMessage, &detailsBytes, &v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup validation: %w", err)
		}
		v.Status = models.BackupValidationStatus(statusStr)
		if err := v.SetDetails(detailsBytes); err != nil {
			db.logger.Warn().Err(err).Str("validation_id", v.ID.String()).Msg("failed to parse validation details")
		}
		validations = append(validations, &v)
	}
	return validations, nil
}
