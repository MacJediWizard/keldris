package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

func TestDefaultConfig(t *testing.T) {
	url := "postgres://user:pass@localhost:5432/testdb"
	cfg := DefaultConfig(url)

	if cfg.URL != url {
		t.Errorf("expected URL %q, got %q", url, cfg.URL)
	}
	if cfg.MaxConns != 25 {
		t.Errorf("expected MaxConns 25, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("expected MinConns 5, got %d", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour {
		t.Errorf("expected MaxConnLifetime 1h, got %v", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 30*time.Minute {
		t.Errorf("expected MaxConnIdleTime 30m, got %v", cfg.MaxConnIdleTime)
	}
}

func TestGetMigrations(t *testing.T) {
	migrations, err := GetMigrations()
	if err != nil {
		t.Fatalf("GetMigrations() error: %v", err)
	}

	if len(migrations) == 0 {
		t.Fatal("expected at least one migration")
	}

	// Check first migration
	m := migrations[0]
	if m.Version != 1 {
		t.Errorf("expected first migration version 1, got %d", m.Version)
	}
	if m.Name == "" {
		t.Error("expected migration name to be non-empty")
	}
	if m.SQL == "" {
		t.Error("expected migration SQL to be non-empty")
	}
}

func TestMigrationsSorted(t *testing.T) {
	migrations, err := GetMigrations()
	if err != nil {
		t.Fatalf("GetMigrations() error: %v", err)
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			t.Errorf("migrations not sorted: version %d comes after %d",
				migrations[i].Version, migrations[i-1].Version)
		}
	}
}

// TestNewWithInvalidURL tests that New returns an error for invalid URLs.
func TestNewWithInvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	cfg := Config{
		URL:             "not-a-valid-url",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	_, err := New(ctx, cfg, logger)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

// TestNewWithUnreachableDB tests that New returns an error for unreachable databases.
func TestNewWithUnreachableDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	cfg := Config{
		URL:             "postgres://user:pass@localhost:59999/nonexistent",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}

	_, err := New(ctx, cfg, logger)
	if err == nil {
		t.Error("expected error for unreachable database")
	}
}

// Integration tests that require a real database.
// Set TEST_DATABASE_URL to run these tests.

func TestIntegration(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	cfg := DefaultConfig(dbURL)
	cfg.MaxConns = 5
	cfg.MinConns = 1

	t.Run("Connect", func(t *testing.T) {
		database, err := New(ctx, cfg, logger)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer database.Close()

		if err := database.Ping(ctx); err != nil {
			t.Errorf("Ping() error: %v", err)
		}
	})

	t.Run("Health", func(t *testing.T) {
		database, err := New(ctx, cfg, logger)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer database.Close()

		health := database.Health()
		if health == nil {
			t.Error("Health() returned nil")
		}
		if _, ok := health["total_conns"]; !ok {
			t.Error("Health() missing total_conns")
		}
	})

	t.Run("Migrate", func(t *testing.T) {
		database, err := New(ctx, cfg, logger)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer database.Close()

		if err := database.Migrate(ctx); err != nil {
			t.Errorf("Migrate() error: %v", err)
		}

		version, err := database.CurrentVersion(ctx)
		if err != nil {
			t.Errorf("CurrentVersion() error: %v", err)
		}
		if version < 1 {
			t.Errorf("expected version >= 1, got %d", version)
		}
	})

	t.Run("ExecTx", func(t *testing.T) {
		database, err := New(ctx, cfg, logger)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer database.Close()

		// Test successful transaction
		err = database.ExecTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "SELECT 1")
			return err
		})
		if err != nil {
			t.Errorf("ExecTx() error: %v", err)
		}
	})
}
