// Package main is the entrypoint for the Keldris server.
//
// @title           Keldris API
// @version         1.0
// @description     Keldris - Keeper of your data. Self-hosted backup solution with OIDC authentication, Restic backup engine, and cloud storage support.
// @termsOfService  https://keldris.io/terms
//
// @contact.name   Keldris Support
// @contact.url    https://github.com/MacJediWizard/keldris
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /api/v1
//
// @securityDefinitions.apikey SessionAuth
// @in cookie
// @name session
// @description Session cookie authentication (for web UI)
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Agent API key authentication. Use format: Bearer kld_xxx
//
// @tag.name Auth
// @tag.description OIDC authentication endpoints
// @tag.name Agents
// @tag.description Backup agent management
// @tag.name Repositories
// @tag.description Backup storage repository configuration
// @tag.name Schedules
// @tag.description Backup schedule configuration
// @tag.name Backups
// @tag.description Backup job execution records
// @tag.name Snapshots
// @tag.description Restic snapshot browsing and file listing
// @tag.name Restores
// @tag.description Data restoration jobs
// @tag.name Organizations
// @tag.description Multi-tenant organization management
// @tag.name Alerts
// @tag.description Alert management and alert rules
// @tag.name Notifications
// @tag.description Notification channels and preferences
// @tag.name AuditLogs
// @tag.description Audit trail and compliance logging
// @tag.name Stats
// @tag.description Storage statistics and growth metrics
// @tag.name Verifications
// @tag.description Backup integrity verification
// @tag.name Version
// @tag.description Server version information
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/shutdown"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Version = "dev"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	logger := log.With().Str("version", Version).Logger()
	logger.Info().Msg("starting Keldris server")

	// Load configuration
	cfg := config.DefaultServerConfig()
	config.LoadServerConfigFromEnv(&cfg)

	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}

	logger.Info().
		Str("http_addr", cfg.HTTPAddr).
		Dur("shutdown_timeout", cfg.Shutdown.Timeout).
		Bool("checkpoint_backups", cfg.Shutdown.CheckpointRunningBackups).
		Bool("resume_on_start", cfg.Shutdown.ResumeCheckpointsOnStart).
		Msg("configuration loaded")

	// Create shutdown manager
	// In a full implementation, the backup tracker would be connected to the scheduler
	shutdownConfig := shutdown.Config{
		Timeout:                  cfg.Shutdown.Timeout,
		DrainTimeout:             cfg.Shutdown.DrainTimeout,
		CheckpointRunningBackups: cfg.Shutdown.CheckpointRunningBackups,
	}

	// Create a backup tracker (will be connected to scheduler in full implementation)
	backupTracker := shutdown.NewSchedulerBackupTracker(nil, nil, logger)
	shutdownMgr := shutdown.NewManager(shutdownConfig, backupTracker, logger)

	// TODO: Initialize database
	// TODO: Setup OIDC provider
	// TODO: Initialize Gin router with shutdown status endpoint
	// TODO: Start HTTP server

	// TODO: If resume on start is enabled, resume checkpointed backups
	if cfg.Shutdown.ResumeCheckpointsOnStart {
		logger.Info().Msg("checking for checkpointed backups to resume")
		// This would be implemented when the scheduler is fully integrated
	}

	// Setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Wait for shutdown signal
	<-ctx.Done()
	stop() // Stop receiving further signals

	logger.Info().Msg("shutdown signal received, initiating graceful shutdown")

	// Perform graceful shutdown
	if err := shutdownMgr.Shutdown(context.Background()); err != nil {
		logger.Error().Err(err).Msg("error during graceful shutdown")
		os.Exit(1)
	}

	// Log final status
	status := shutdownMgr.GetStatus()
	logger.Info().
		Str("state", string(status.State)).
		Int("checkpointed", status.CheckpointedCount).
		Msg("Keldris server shutdown complete")
}
