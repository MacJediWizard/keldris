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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MacJediWizard/keldris/internal/api"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/maintenance"
	"github.com/rs/zerolog"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("version", Version).Logger()
	if os.Getenv("ENV") != string(config.EnvProduction) {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	logger.Info().
		Str("version", Version).
		Str("commit", Commit).
		Str("build_date", BuildDate).
		Msg("Starting Keldris server")

	// Load configuration
	cfg := config.LoadServerConfig()

	// Connect to database
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal().Msg("DATABASE_URL environment variable is required")
		return 1
	}

	database, err := db.New(ctx, db.DefaultConfig(databaseURL), logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
		return 1
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
		return 1
	}

	// Initialize crypto key manager
	encryptionKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyHex == "" {
		logger.Fatal().Msg("ENCRYPTION_KEY environment variable is required")
		return 1
	}

	masterKey, err := crypto.MasterKeyFromHex(encryptionKeyHex)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to decode ENCRYPTION_KEY")
		return 1
	}

	keyManager, err := crypto.NewKeyManager(masterKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize key manager")
		return 1
	}

	// Initialize OIDC provider (optional - nil if not configured)
	var oidcProvider *auth.OIDC
	oidcIssuer := os.Getenv("OIDC_ISSUER")
	if oidcIssuer != "" {
		oidcCfg := auth.DefaultOIDCConfig(
			oidcIssuer,
			os.Getenv("OIDC_CLIENT_ID"),
			os.Getenv("OIDC_CLIENT_SECRET"),
			os.Getenv("OIDC_REDIRECT_URL"),
		)
		oidcProvider, err = auth.NewOIDC(ctx, oidcCfg, logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to initialize OIDC provider")
			return 1
		}
		logger.Info().Str("issuer", oidcIssuer).Msg("OIDC provider initialized")
	} else {
		logger.Warn().Msg("OIDC not configured - authentication will be unavailable")
	}

	// Initialize session store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		logger.Fatal().Msg("SESSION_SECRET environment variable is required")
		return 1
	}

	isSecure := cfg.Environment == config.EnvProduction
	sessionCfg := auth.DefaultSessionConfig([]byte(sessionSecret), isSecure)
	sessions, err := auth.NewSessionStore(sessionCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize session store")
		return 1
	}

	// Build API router
	allowedOrigins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	if os.Getenv("CORS_ORIGINS") == "" {
		allowedOrigins = []string{}
	}

	routerCfg := api.Config{
		Environment:       cfg.Environment,
		AllowedOrigins:    allowedOrigins,
		RateLimitRequests: 100,
		RateLimitPeriod:   "1m",
		RedisURL:          os.Getenv("REDIS_URL"),
		Version:           Version,
		Commit:            Commit,
		BuildDate:         BuildDate,
	}

	router, err := api.NewRouter(routerCfg, database, oidcProvider, sessions, keyManager, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize router")
		return 1
	}

	// Start HTTP server
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           router.Engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	// Start server in background
	go func() {
		logger.Info().Str("addr", listenAddr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Start retention cleanup scheduler
	retentionScheduler := maintenance.NewRetentionScheduler(database, cfg.RetentionDays, logger)
	if err := retentionScheduler.Start(); err != nil {
		logger.Error().Err(err).Msg("Failed to start retention scheduler")
	}
	defer retentionScheduler.Stop()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info().Str("signal", sig.String()).Msg("Shutting down server")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
		return 1
	}

	logger.Info().Msg("Server stopped gracefully")
	return 0
}
