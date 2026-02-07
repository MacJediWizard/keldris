package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Database Backup methods

// CreateDatabaseBackup creates a new database backup record.
func (db *DB) CreateDatabaseBackup(ctx context.Context, backup *models.DatabaseBackup) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO database_backups (
			id, status, file_path, size_bytes, checksum,
			started_at, completed_at, duration_ms, error_message,
			triggered_by, is_scheduled, verified, verified_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15
		)
	`,
		backup.ID, backup.Status, backup.FilePath, backup.SizeBytes, backup.Checksum,
		backup.StartedAt, backup.CompletedAt, backup.Duration, backup.ErrorMessage,
		backup.TriggeredBy, backup.IsScheduled, backup.Verified, backup.VerifiedAt,
		backup.CreatedAt, backup.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create database backup: %w", err)
	}
	return nil
}

// UpdateDatabaseBackup updates an existing database backup record.
func (db *DB) UpdateDatabaseBackup(ctx context.Context, backup *models.DatabaseBackup) error {
	backup.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE database_backups SET
			status = $2,
			file_path = $3,
			size_bytes = $4,
			checksum = $5,
			started_at = $6,
			completed_at = $7,
			duration_ms = $8,
			error_message = $9,
			verified = $10,
			verified_at = $11,
			updated_at = $12
		WHERE id = $1
	`,
		backup.ID, backup.Status, backup.FilePath, backup.SizeBytes, backup.Checksum,
		backup.StartedAt, backup.CompletedAt, backup.Duration, backup.ErrorMessage,
		backup.Verified, backup.VerifiedAt, backup.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update database backup: %w", err)
	}
	return nil
}

// GetDatabaseBackupByID returns a database backup by ID.
func (db *DB) GetDatabaseBackupByID(ctx context.Context, id uuid.UUID) (*models.DatabaseBackup, error) {
	var backup models.DatabaseBackup
	err := db.Pool.QueryRow(ctx, `
		SELECT id, status, file_path, size_bytes, checksum,
		       started_at, completed_at, duration_ms, error_message,
		       triggered_by, is_scheduled, verified, verified_at,
		       created_at, updated_at
		FROM database_backups
		WHERE id = $1
	`, id).Scan(
		&backup.ID, &backup.Status, &backup.FilePath, &backup.SizeBytes, &backup.Checksum,
		&backup.StartedAt, &backup.CompletedAt, &backup.Duration, &backup.ErrorMessage,
		&backup.TriggeredBy, &backup.IsScheduled, &backup.Verified, &backup.VerifiedAt,
		&backup.CreatedAt, &backup.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("database backup not found: %s", id)
		}
		return nil, fmt.Errorf("get database backup: %w", err)
	}
	return &backup, nil
}

// ListDatabaseBackups returns a paginated list of database backups.
// If limit is 0, returns all backups.
func (db *DB) ListDatabaseBackups(ctx context.Context, limit, offset int) ([]*models.DatabaseBackup, int, error) {
	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM database_backups`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count database backups: %w", err)
	}

	// Build query
	query := `
		SELECT id, status, file_path, size_bytes, checksum,
		       started_at, completed_at, duration_ms, error_message,
		       triggered_by, is_scheduled, verified, verified_at,
		       created_at, updated_at
		FROM database_backups
		ORDER BY created_at DESC
	`

	var rows pgx.Rows
	if limit > 0 {
		query += ` LIMIT $1 OFFSET $2`
		rows, err = db.Pool.Query(ctx, query, limit, offset)
	} else {
		rows, err = db.Pool.Query(ctx, query)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list database backups: %w", err)
	}
	defer rows.Close()

	var backups []*models.DatabaseBackup
	for rows.Next() {
		var backup models.DatabaseBackup
		err := rows.Scan(
			&backup.ID, &backup.Status, &backup.FilePath, &backup.SizeBytes, &backup.Checksum,
			&backup.StartedAt, &backup.CompletedAt, &backup.Duration, &backup.ErrorMessage,
			&backup.TriggeredBy, &backup.IsScheduled, &backup.Verified, &backup.VerifiedAt,
			&backup.CreatedAt, &backup.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan database backup: %w", err)
		}
		backups = append(backups, &backup)
	}

	return backups, total, nil
}

// GetLatestDatabaseBackup returns the most recent database backup.
func (db *DB) GetLatestDatabaseBackup(ctx context.Context) (*models.DatabaseBackup, error) {
	var backup models.DatabaseBackup
	err := db.Pool.QueryRow(ctx, `
		SELECT id, status, file_path, size_bytes, checksum,
		       started_at, completed_at, duration_ms, error_message,
		       triggered_by, is_scheduled, verified, verified_at,
		       created_at, updated_at
		FROM database_backups
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(
		&backup.ID, &backup.Status, &backup.FilePath, &backup.SizeBytes, &backup.Checksum,
		&backup.StartedAt, &backup.CompletedAt, &backup.Duration, &backup.ErrorMessage,
		&backup.TriggeredBy, &backup.IsScheduled, &backup.Verified, &backup.VerifiedAt,
		&backup.CreatedAt, &backup.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest database backup: %w", err)
	}
	return &backup, nil
}

// DeleteDatabaseBackup deletes a database backup record by ID.
func (db *DB) DeleteDatabaseBackup(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM database_backups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete database backup: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("database backup not found: %s", id)
	}
	return nil
}

// GetDatabaseBackupsOlderThan returns backups created before the specified time.
func (db *DB) GetDatabaseBackupsOlderThan(ctx context.Context, before time.Time) ([]*models.DatabaseBackup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, status, file_path, size_bytes, checksum,
		       started_at, completed_at, duration_ms, error_message,
		       triggered_by, is_scheduled, verified, verified_at,
		       created_at, updated_at
		FROM database_backups
		WHERE created_at < $1
		ORDER BY created_at ASC
	`, before)
	if err != nil {
		return nil, fmt.Errorf("get old database backups: %w", err)
	}
	defer rows.Close()

	var backups []*models.DatabaseBackup
	for rows.Next() {
		var backup models.DatabaseBackup
		err := rows.Scan(
			&backup.ID, &backup.Status, &backup.FilePath, &backup.SizeBytes, &backup.Checksum,
			&backup.StartedAt, &backup.CompletedAt, &backup.Duration, &backup.ErrorMessage,
			&backup.TriggeredBy, &backup.IsScheduled, &backup.Verified, &backup.VerifiedAt,
			&backup.CreatedAt, &backup.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan database backup: %w", err)
		}
		backups = append(backups, &backup)
	}

	return backups, nil
}

// GetDatabaseBackupSummary returns a summary of database backups.
func (db *DB) GetDatabaseBackupSummary(ctx context.Context) (*models.DatabaseBackupSummary, error) {
	summary := &models.DatabaseBackupSummary{}

	// Get counts and totals
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as successful,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(SUM(size_bytes) FILTER (WHERE status = 'completed'), 0) as total_size
		FROM database_backups
	`).Scan(&summary.TotalBackups, &summary.SuccessfulBackups, &summary.FailedBackups, &summary.TotalSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("get backup summary counts: %w", err)
	}

	// Get latest backup info
	var lastBackupAt *time.Time
	var lastStatus *string
	err = db.Pool.QueryRow(ctx, `
		SELECT created_at, status
		FROM database_backups
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&lastBackupAt, &lastStatus)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("get last backup info: %w", err)
	}
	summary.LastBackupAt = lastBackupAt
	if lastStatus != nil {
		summary.LastBackupStatus = *lastStatus
	}

	// Get oldest backup
	var oldestBackupAt *time.Time
	err = db.Pool.QueryRow(ctx, `
		SELECT MIN(created_at)
		FROM database_backups
		WHERE status = 'completed'
	`).Scan(&oldestBackupAt)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("get oldest backup: %w", err)
	}
	summary.OldestBackupAt = oldestBackupAt

	return summary, nil
}

// MarkDatabaseBackupVerified marks a database backup as verified.
func (db *DB) MarkDatabaseBackupVerified(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE database_backups
		SET verified = true, verified_at = $2, updated_at = $3
		WHERE id = $1
	`, id, now, now)
	if err != nil {
		return fmt.Errorf("mark backup verified: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("database backup not found: %s", id)
	}
	return nil
}

// GetLatestSuccessfulDatabaseBackup returns the most recent successful backup.
func (db *DB) GetLatestSuccessfulDatabaseBackup(ctx context.Context) (*models.DatabaseBackup, error) {
	var backup models.DatabaseBackup
	err := db.Pool.QueryRow(ctx, `
		SELECT id, status, file_path, size_bytes, checksum,
		       started_at, completed_at, duration_ms, error_message,
		       triggered_by, is_scheduled, verified, verified_at,
		       created_at, updated_at
		FROM database_backups
		WHERE status = 'completed'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(
		&backup.ID, &backup.Status, &backup.FilePath, &backup.SizeBytes, &backup.Checksum,
		&backup.StartedAt, &backup.CompletedAt, &backup.Duration, &backup.ErrorMessage,
		&backup.TriggeredBy, &backup.IsScheduled, &backup.Verified, &backup.VerifiedAt,
		&backup.CreatedAt, &backup.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest successful backup: %w", err)
	}
	return &backup, nil
}
