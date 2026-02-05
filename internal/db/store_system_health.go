package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ServerError represents a recent server error for health display.
type ServerError struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthHistoryRecord represents a historical health data point.
type HealthHistoryRecord struct {
	ID                  string    `json:"id"`
	Timestamp           time.Time `json:"timestamp"`
	Status              string    `json:"status"`
	CPUUsage            float64   `json:"cpu_usage"`
	MemoryUsage         float64   `json:"memory_usage"`
	MemoryAllocMB       float64   `json:"memory_alloc_mb"`
	MemoryTotalAllocMB  float64   `json:"memory_total_alloc_mb"`
	GoroutineCount      int       `json:"goroutine_count"`
	DatabaseConnections int       `json:"database_connections"`
	DatabaseSizeBytes   int64     `json:"database_size_bytes"`
	PendingBackups      int       `json:"pending_backups"`
	RunningBackups      int       `json:"running_backups"`
	ErrorCount          int       `json:"error_count"`
}

// GetDatabaseSize returns the size of the database in bytes.
func (db *DB) GetDatabaseSize(ctx context.Context) (int64, error) {
	var size int64
	err := db.Pool.QueryRow(ctx, `
		SELECT pg_database_size(current_database())
	`).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("get database size: %w", err)
	}
	return size, nil
}

// GetActiveConnections returns the count of active database connections.
func (db *DB) GetActiveConnections(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE datname = current_database()
		AND state = 'active'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get active connections: %w", err)
	}
	return count, nil
}

// GetPendingBackupsCount returns the count of pending (queued) backups.
// If orgID is uuid.Nil, returns count across all organizations.
func (db *DB) GetPendingBackupsCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	var err error
	if orgID == uuid.Nil {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM backup_queue
			WHERE status = 'queued'
		`).Scan(&count)
	} else {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM backup_queue
			WHERE org_id = $1 AND status = 'queued'
		`, orgID).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("get pending backups count: %w", err)
	}
	return count, nil
}

// GetRunningBackupsCount returns the count of running backups.
// If orgID is uuid.Nil, returns count across all organizations.
func (db *DB) GetRunningBackupsCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	var err error
	if orgID == uuid.Nil {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM backups
			WHERE status = 'running'
		`).Scan(&count)
	} else {
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM backups b
			JOIN schedules s ON b.schedule_id = s.id
			JOIN agents a ON s.agent_id = a.id
			WHERE a.org_id = $1 AND b.status = 'running'
		`, orgID).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("get running backups count: %w", err)
	}
	return count, nil
}

// GetRecentServerErrors returns recent server errors from the server_logs table.
func (db *DB) GetRecentServerErrors(ctx context.Context, limit int) ([]*ServerError, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, level, message, COALESCE(component, ''), timestamp
		FROM server_logs
		WHERE level IN ('error', 'fatal')
		ORDER BY timestamp DESC
		LIMIT $1
	`, limit)
	if err != nil {
		// Table might not exist or be empty
		db.logger.Debug().Err(err).Msg("failed to query server_logs for errors")
		return []*ServerError{}, nil
	}
	defer rows.Close()

	var errors []*ServerError
	for rows.Next() {
		var e ServerError
		err := rows.Scan(&e.ID, &e.Level, &e.Message, &e.Component, &e.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("scan server error: %w", err)
		}
		errors = append(errors, &e)
	}

	return errors, nil
}

// GetHealthHistoryRecords returns health history records since the given time.
func (db *DB) GetHealthHistoryRecords(ctx context.Context, since time.Time) ([]*HealthHistoryRecord, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, timestamp, status, cpu_usage, memory_usage, memory_alloc_mb,
		       memory_total_alloc_mb, goroutine_count, database_connections,
		       database_size_bytes, pending_backups, running_backups, error_count
		FROM system_health_history
		WHERE timestamp >= $1
		ORDER BY timestamp DESC
	`, since)
	if err != nil {
		return nil, fmt.Errorf("get health history records: %w", err)
	}
	defer rows.Close()

	var records []*HealthHistoryRecord
	for rows.Next() {
		var r HealthHistoryRecord
		err := rows.Scan(
			&r.ID, &r.Timestamp, &r.Status, &r.CPUUsage, &r.MemoryUsage,
			&r.MemoryAllocMB, &r.MemoryTotalAllocMB, &r.GoroutineCount,
			&r.DatabaseConnections, &r.DatabaseSizeBytes, &r.PendingBackups,
			&r.RunningBackups, &r.ErrorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan health history record: %w", err)
		}
		records = append(records, &r)
	}

	return records, nil
}

// SaveHealthHistoryRecord saves a health history record.
func (db *DB) SaveHealthHistoryRecord(ctx context.Context, record *HealthHistoryRecord) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_health_history (
			id, timestamp, status, cpu_usage, memory_usage, memory_alloc_mb,
			memory_total_alloc_mb, goroutine_count, database_connections,
			database_size_bytes, pending_backups, running_backups, error_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		record.ID, record.Timestamp, record.Status, record.CPUUsage,
		record.MemoryUsage, record.MemoryAllocMB, record.MemoryTotalAllocMB,
		record.GoroutineCount, record.DatabaseConnections, record.DatabaseSizeBytes,
		record.PendingBackups, record.RunningBackups, record.ErrorCount,
	)
	if err != nil {
		return fmt.Errorf("save health history record: %w", err)
	}
	return nil
}

// CleanupOldHealthHistoryRecords removes health history records older than the specified duration.
func (db *DB) CleanupOldHealthHistoryRecords(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM system_health_history
		WHERE timestamp < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup old health history records: %w", err)
	}
	return result.RowsAffected(), nil
}
