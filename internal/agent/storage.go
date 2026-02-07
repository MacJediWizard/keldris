package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements QueueStore using SQLite for local persistence.
type SQLiteStore struct {
	db     *sql.DB
	logger zerolog.Logger
}

// NewSQLiteStore creates a new SQLite-based queue store.
// The database file is created in the Keldris config directory.
func NewSQLiteStore(configDir string, logger zerolog.Logger) (*SQLiteStore, error) {
	dbPath := filepath.Join(configDir, "queue.db")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &SQLiteStore{
		db:     db,
		logger: logger.With().Str("component", "sqlite_store").Logger(),
	}

	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	store.logger.Info().Str("path", dbPath).Msg("queue database initialized")

	return store, nil
}

// migrate creates the necessary tables.
func (s *SQLiteStore) migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS queued_backups (
			id TEXT PRIMARY KEY,
			schedule_id TEXT NOT NULL,
			schedule_name TEXT NOT NULL,
			scheduled_at TEXT NOT NULL,
			queued_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			retry_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			synced_at TEXT,
			backup_result TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_queued_backups_status ON queued_backups(status);
		CREATE INDEX IF NOT EXISTS idx_queued_backups_scheduled_at ON queued_backups(scheduled_at);
		CREATE INDEX IF NOT EXISTS idx_queued_backups_synced_at ON queued_backups(synced_at);

		CREATE TABLE IF NOT EXISTS queue_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateQueuedBackup stores a new queued backup.
func (s *SQLiteStore) CreateQueuedBackup(ctx context.Context, backup *QueuedBackup) error {
	var resultJSON sql.NullString
	if backup.BackupResult != nil {
		data, err := json.Marshal(backup.BackupResult)
		if err != nil {
			return fmt.Errorf("marshal backup result: %w", err)
		}
		resultJSON = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO queued_backups (id, schedule_id, schedule_name, scheduled_at, queued_at, status, retry_count, last_error, synced_at, backup_result)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var syncedAt sql.NullString
	if backup.SyncedAt != nil {
		syncedAt = sql.NullString{String: backup.SyncedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, query,
		backup.ID.String(),
		backup.ScheduleID.String(),
		backup.ScheduleName,
		backup.ScheduledAt.Format(time.RFC3339),
		backup.QueuedAt.Format(time.RFC3339),
		string(backup.Status),
		backup.RetryCount,
		nullString(backup.LastError),
		syncedAt,
		resultJSON,
	)

	if err != nil {
		return fmt.Errorf("insert queued backup: %w", err)
	}

	return nil
}

// GetQueuedBackup retrieves a queued backup by ID.
func (s *SQLiteStore) GetQueuedBackup(ctx context.Context, id uuid.UUID) (*QueuedBackup, error) {
	query := `
		SELECT id, schedule_id, schedule_name, scheduled_at, queued_at, status, retry_count, last_error, synced_at, backup_result
		FROM queued_backups
		WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id.String())
	return s.scanQueuedBackup(row)
}

// UpdateQueuedBackup updates an existing queued backup.
func (s *SQLiteStore) UpdateQueuedBackup(ctx context.Context, backup *QueuedBackup) error {
	var resultJSON sql.NullString
	if backup.BackupResult != nil {
		data, err := json.Marshal(backup.BackupResult)
		if err != nil {
			return fmt.Errorf("marshal backup result: %w", err)
		}
		resultJSON = sql.NullString{String: string(data), Valid: true}
	}

	var syncedAt sql.NullString
	if backup.SyncedAt != nil {
		syncedAt = sql.NullString{String: backup.SyncedAt.Format(time.RFC3339), Valid: true}
	}

	query := `
		UPDATE queued_backups
		SET status = ?, retry_count = ?, last_error = ?, synced_at = ?, backup_result = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		string(backup.Status),
		backup.RetryCount,
		nullString(backup.LastError),
		syncedAt,
		resultJSON,
		backup.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("update queued backup: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if affected == 0 {
		return ErrBackupNotFound
	}

	return nil
}

// DeleteQueuedBackup removes a queued backup.
func (s *SQLiteStore) DeleteQueuedBackup(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM queued_backups WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("delete queued backup: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if affected == 0 {
		return ErrBackupNotFound
	}

	return nil
}

// ListPendingBackups returns all pending backups ordered by scheduled time.
func (s *SQLiteStore) ListPendingBackups(ctx context.Context) ([]*QueuedBackup, error) {
	query := `
		SELECT id, schedule_id, schedule_name, scheduled_at, queued_at, status, retry_count, last_error, synced_at, backup_result
		FROM queued_backups
		WHERE status = 'pending'
		ORDER BY scheduled_at ASC
	`

	return s.queryBackups(ctx, query)
}

// ListAllBackups returns all backups in the queue.
func (s *SQLiteStore) ListAllBackups(ctx context.Context) ([]*QueuedBackup, error) {
	query := `
		SELECT id, schedule_id, schedule_name, scheduled_at, queued_at, status, retry_count, last_error, synced_at, backup_result
		FROM queued_backups
		ORDER BY queued_at DESC
	`

	return s.queryBackups(ctx, query)
}

// GetQueueCount returns the total number of entries in the queue.
func (s *SQLiteStore) GetQueueCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM queued_backups WHERE status = 'pending'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count queued backups: %w", err)
	}
	return count, nil
}

// GetQueueStatus returns aggregate queue statistics.
func (s *SQLiteStore) GetQueueStatus(ctx context.Context) (*QueueStatus, error) {
	status := &QueueStatus{}

	// Get counts by status
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM queued_backups
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("query status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var statusStr string
		var count int
		if err := rows.Scan(&statusStr, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}

		switch QueuedBackupStatus(statusStr) {
		case QueuedBackupStatusPending, QueuedBackupStatusSyncing:
			status.PendingCount += count
		case QueuedBackupStatusSynced:
			status.SyncedCount += count
		case QueuedBackupStatusFailed:
			status.FailedCount += count
		}
		status.TotalQueued += count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status counts: %w", err)
	}

	// Get oldest queued time
	var oldestStr sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT MIN(queued_at) FROM queued_backups WHERE status = 'pending'
	`).Scan(&oldestStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query oldest queued: %w", err)
	}
	if oldestStr.Valid {
		t, err := time.Parse(time.RFC3339, oldestStr.String)
		if err == nil {
			status.OldestQueuedAt = &t
		}
	}

	return status, nil
}

// PruneOldEntries removes synced/failed entries older than the given duration.
func (s *SQLiteStore) PruneOldEntries(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)

	query := `
		DELETE FROM queued_backups
		WHERE status IN ('synced', 'failed')
		AND queued_at < ?
	`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune old entries: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return int(affected), nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// SetMetadata stores a key-value pair in the metadata table.
func (s *SQLiteStore) SetMetadata(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO queue_metadata (key, value, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`
	_, err := s.db.ExecContext(ctx, query, key, value)
	return err
}

// GetMetadata retrieves a value from the metadata table.
func (s *SQLiteStore) GetMetadata(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM queue_metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// queryBackups executes a query and returns the results as QueuedBackup slice.
func (s *SQLiteStore) queryBackups(ctx context.Context, query string, args ...any) ([]*QueuedBackup, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query backups: %w", err)
	}
	defer rows.Close()

	var backups []*QueuedBackup
	for rows.Next() {
		backup, err := s.scanQueuedBackupRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup row: %w", err)
		}
		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backups: %w", err)
	}

	return backups, nil
}

// scanQueuedBackup scans a single row into a QueuedBackup.
func (s *SQLiteStore) scanQueuedBackup(row *sql.Row) (*QueuedBackup, error) {
	var (
		idStr, scheduleIDStr, scheduleName, scheduledAtStr, queuedAtStr, statusStr string
		retryCount                                                                  int
		lastError, syncedAtStr, resultJSON                                          sql.NullString
	)

	err := row.Scan(&idStr, &scheduleIDStr, &scheduleName, &scheduledAtStr, &queuedAtStr, &statusStr, &retryCount, &lastError, &syncedAtStr, &resultJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return s.parseBackup(idStr, scheduleIDStr, scheduleName, scheduledAtStr, queuedAtStr, statusStr, retryCount, lastError, syncedAtStr, resultJSON)
}

// scanQueuedBackupRows scans a row from Rows into a QueuedBackup.
func (s *SQLiteStore) scanQueuedBackupRows(rows *sql.Rows) (*QueuedBackup, error) {
	var (
		idStr, scheduleIDStr, scheduleName, scheduledAtStr, queuedAtStr, statusStr string
		retryCount                                                                  int
		lastError, syncedAtStr, resultJSON                                          sql.NullString
	)

	err := rows.Scan(&idStr, &scheduleIDStr, &scheduleName, &scheduledAtStr, &queuedAtStr, &statusStr, &retryCount, &lastError, &syncedAtStr, &resultJSON)
	if err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return s.parseBackup(idStr, scheduleIDStr, scheduleName, scheduledAtStr, queuedAtStr, statusStr, retryCount, lastError, syncedAtStr, resultJSON)
}

// parseBackup converts scanned values into a QueuedBackup.
func (s *SQLiteStore) parseBackup(
	idStr, scheduleIDStr, scheduleName, scheduledAtStr, queuedAtStr, statusStr string,
	retryCount int,
	lastError, syncedAtStr, resultJSON sql.NullString,
) (*QueuedBackup, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}

	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse schedule id: %w", err)
	}

	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse scheduled_at: %w", err)
	}

	queuedAt, err := time.Parse(time.RFC3339, queuedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse queued_at: %w", err)
	}

	backup := &QueuedBackup{
		ID:           id,
		ScheduleID:   scheduleID,
		ScheduleName: scheduleName,
		ScheduledAt:  scheduledAt,
		QueuedAt:     queuedAt,
		Status:       QueuedBackupStatus(statusStr),
		RetryCount:   retryCount,
	}

	if lastError.Valid {
		backup.LastError = lastError.String
	}

	if syncedAtStr.Valid {
		t, err := time.Parse(time.RFC3339, syncedAtStr.String)
		if err == nil {
			backup.SyncedAt = &t
		}
	}

	if resultJSON.Valid {
		var result BackupResult
		if err := json.Unmarshal([]byte(resultJSON.String), &result); err == nil {
			backup.BackupResult = &result
		}
	}

	return backup, nil
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
