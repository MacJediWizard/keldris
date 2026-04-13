// Package main is the entrypoint for the Keldris Customer Portal.
//
// @title           Keldris Customer Portal API
// @version         1.0
// @description     Keldris Customer Portal - License management portal for customers to view licenses, download keys, and access invoices.
// @termsOfService  https://keldris.io/terms
//
// @contact.name   Keldris Support
// @contact.url    https://github.com/MacJediWizard/keldris
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host      localhost:8081
// @BasePath  /api/v1
//
// @securityDefinitions.apikey PortalSession
// @in cookie
// @name keldris_portal_session
// @description Portal session cookie authentication
//
// @tag.name Portal Auth
// @tag.description Customer authentication endpoints
// @tag.name Portal Licenses
// @tag.description Customer license management
// @tag.name Portal Invoices
// @tag.description Customer invoice access
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MacJediWizard/keldris/internal/db"
	portal "github.com/MacJediWizard/keldris/internal/portal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Version is set at build time.
	Version = "dev"
	// Commit is set at build time.
	Commit = "unknown"
	// BuildDate is set at build time.
	BuildDate = "unknown"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().
		Str("version", Version).
		Str("commit", Commit).
		Str("build_date", BuildDate).
		Msg("Starting Keldris Customer Portal")

	// Get configuration from environment
	port := os.Getenv("PORTAL_PORT")
	if port == "" {
		port = "8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal().Msg("DATABASE_URL environment variable is required")
	}

	// Initialize database connection
	ctx := context.Background()
	database, err := db.New(ctx, db.DefaultConfig(databaseURL), log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize portal router
	portalConfig := portal.Config{
		AllowedOrigins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
		Version:        Version,
		Commit:         Commit,
		BuildDate:      BuildDate,
	}

	portalStore := db.NewPortalStore(database)
	router, err := portal.NewRouter(portalConfig, portalStore, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create router")
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router.Engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("port", port).Msg("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	log.Info().Msg("Server stopped")
}
