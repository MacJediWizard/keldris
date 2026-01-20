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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Version = "dev"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Str("version", Version).Msg("Starting Keldris server")

	// TODO: Load configuration
	// TODO: Initialize database
	// TODO: Setup OIDC provider
	// TODO: Initialize Gin router
	// TODO: Start HTTP server

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Keldris server - implement me!")
}
