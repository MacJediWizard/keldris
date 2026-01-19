// Package main provides the database migration CLI tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/rs/zerolog"
)

func main() {
	var (
		dbURL   = flag.String("db", "", "Database URL (or set DATABASE_URL env var)")
		showVer = flag.Bool("version", false, "Show current schema version")
		list    = flag.Bool("list", false, "List all migrations")
	)
	flag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Logger()

	if *list {
		listMigrations(logger)
		return
	}

	url := *dbURL
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		logger.Fatal().Msg("database URL required: use -db flag or set DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg := db.DefaultConfig(url)
	cfg.MaxConns = 5
	cfg.MinConns = 1

	database, err := db.New(ctx, cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer database.Close()

	if *showVer {
		showVersion(ctx, database, logger)
		return
	}

	logger.Info().Msg("running database migrations")
	if err := database.Migrate(ctx); err != nil {
		logger.Fatal().Err(err).Msg("migration failed")
	}

	version, err := database.CurrentVersion(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("could not get current version")
	} else {
		logger.Info().Int("version", version).Msg("migrations complete")
	}
}

func showVersion(ctx context.Context, database *db.DB, logger zerolog.Logger) {
	version, err := database.CurrentVersion(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to get schema version")
	}
	fmt.Printf("Current schema version: %d\n", version)
}

func listMigrations(logger zerolog.Logger) {
	migrations, err := db.GetMigrations()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to list migrations")
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return
	}

	fmt.Println("Available migrations:")
	for _, m := range migrations {
		fmt.Printf("  %03d: %s\n", m.Version, m.Name)
	}
}
