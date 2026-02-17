// Package db provides PostgreSQL database connectivity using pgx.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds database connection configuration.
type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// DB wraps a pgxpool.Pool with helper methods.
type DB struct {
	Pool   *pgxpool.Pool
	logger zerolog.Logger
}

// New creates a new database connection pool.
func New(ctx context.Context, cfg Config, logger zerolog.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	db := &DB{
		Pool:   pool,
		logger: logger.With().Str("component", "db").Logger(),
	}

	if err := db.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db.logger.Info().Msg("database connection pool established")
	return db, nil
}

// Ping verifies the database connection is alive.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
	db.logger.Info().Msg("database connection pool closed")
}

// Health returns basic health information about the connection pool.
func (db *DB) Health() map[string]any {
	stats := db.Pool.Stat()
	return map[string]any{
		"total_conns":       stats.TotalConns(),
		"acquired_conns":    stats.AcquiredConns(),
		"idle_conns":        stats.IdleConns(),
		"max_conns":         stats.MaxConns(),
		"constructing":      stats.ConstructingConns(),
		"empty_acquire":     stats.EmptyAcquireCount(),
		"canceled_acquire":  stats.CanceledAcquireCount(),
		"acquire_duration":  stats.AcquireDuration().String(),
		"max_lifetime_dest": stats.MaxLifetimeDestroyCount(),
		"max_idle_dest":     stats.MaxIdleDestroyCount(),
	}
}

// ExecTx executes a function within a database transaction.
func (db *DB) ExecTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Migration represents a database migration.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// GetMigrations returns all embedded migrations sorted by version.
func GetMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		var version int
		var name string
		_, err = fmt.Sscanf(entry.Name(), "%d_%s", &version, &name)
		if err != nil {
			return nil, fmt.Errorf("parse migration filename %s: %w", entry.Name(), err)
		}

		name = strings.TrimSuffix(entry.Name(), ".sql")

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Migrate runs all pending database migrations.
func (db *DB) Migrate(ctx context.Context) error {
	// Acquire advisory lock to prevent concurrent migrations
	const migrationLockID int64 = 7364827163 // arbitrary unique lock ID for Keldris migrations
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for migration lock: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	// Create migrations tracking table if it doesn't exist
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	migrations, err := GetMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		var exists bool
		err := db.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			m.Version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", m.Version, err)
		}

		if exists {
			db.logger.Debug().Int("version", m.Version).Str("name", m.Name).Msg("migration already applied")
			continue
		}

		db.logger.Info().Int("version", m.Version).Str("name", m.Name).Msg("applying migration")

		err = db.ExecTx(ctx, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.SQL); err != nil {
				return fmt.Errorf("execute migration SQL: %w", err)
			}

			if _, err := tx.Exec(ctx,
				"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
				m.Version, m.Name,
			); err != nil {
				return fmt.Errorf("record migration: %w", err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
		}

		db.logger.Info().Int("version", m.Version).Str("name", m.Name).Msg("migration applied successfully")
	}

	return nil
}

// CurrentVersion returns the current schema version.
func (db *DB) CurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := db.Pool.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM schema_migrations",
	).Scan(&version)
	if err != nil {
		// Table might not exist yet
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("get current version: %w", err)
	}
	return version, nil
}
