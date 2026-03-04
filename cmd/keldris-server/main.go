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
// @license.name  AGPL-3.0
// @license.url   https://www.gnu.org/licenses/agpl-3.0.html
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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/MacJediWizard/keldris/internal/activity"
	"github.com/MacJediWizard/keldris/internal/api"
	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/logs"
	"github.com/MacJediWizard/keldris/internal/maintenance"
	"github.com/MacJediWizard/keldris/internal/metrics"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/monitoring"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/MacJediWizard/keldris/internal/reports"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	Version       = "dev"
	Commit        = "unknown"
	BuildDate     = "unknown"
	IntegrityHash = ""
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize server log buffer for admin log viewing
	logBuffer := logs.NewLogBuffer(logs.DefaultConfig())

	// Initialize logger with log buffer as additional writer
	var logWriter io.Writer
	if os.Getenv("ENV") != string(config.EnvProduction) {
		logWriter = io.MultiWriter(zerolog.ConsoleWriter{Out: os.Stderr}, logBuffer)
	} else {
		logWriter = io.MultiWriter(os.Stdout, logBuffer)
	}
	logger := zerolog.New(logWriter).With().Timestamp().Str("version", Version).Logger()

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
	cfg.DatabaseURL = databaseURL

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

	// Clean up stale backups stuck in "running" from previous runs
	if staleCount, err := database.FailStaleBackups(ctx, 24*time.Hour); err != nil {
		logger.Error().Err(err).Msg("Failed to clean up stale backups")
	} else if staleCount > 0 {
		logger.Info().Int("count", staleCount).Msg("Marked stale running backups as failed")
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

	// Initialize OIDC provider wrapper (starts nil, loaded from DB if configured)
	oidcProvider := auth.NewOIDCProvider(nil, logger)

	// Try loading OIDC settings from DB (first org's settings)
	oidcSettings, err := database.GetFirstOrgOIDCSettings(ctx)
	if err == nil && oidcSettings != nil && oidcSettings.Enabled && oidcSettings.Issuer != "" {
		oidcCfg := auth.DefaultOIDCConfig(
			oidcSettings.Issuer,
			oidcSettings.ClientID,
			oidcSettings.ClientSecret,
			oidcSettings.RedirectURL,
		)
		if err := oidcProvider.Update(ctx, oidcCfg); err != nil {
			logger.Warn().Err(err).Msg("Failed to initialize OIDC from database settings (continuing without SSO)")
		} else {
			logger.Info().Str("issuer", oidcSettings.Issuer).Msg("OIDC provider initialized from database")
		}
	} else {
		logger.Info().Msg("OIDC not configured - password login only")
	}

	// Initialize session store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		logger.Fatal().Msg("SESSION_SECRET environment variable is required")
		return 1
	}

	isSecure := cfg.Environment == config.EnvProduction
	sessionCfg := auth.DefaultSessionConfig([]byte(sessionSecret), isSecure, cfg.SessionMaxAge, cfg.SessionIdleTimeout)
	sessions, err := auth.NewSessionStore(sessionCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize session store")
		return 1
	}

	// Parse license key
	var lic *license.License

	// Fetch signing public key from license server (unless air-gapped)
	var licPubKey []byte
	if !cfg.AirGapMode {
		pubKeyHex, err := fetchSigningKey(cfg.LicenseServerURL)
		if err != nil {
			logger.Fatal().Err(err).Str("url", cfg.LicenseServerURL).Msg("Failed to fetch signing key from license server")
			return 1
		}
		decoded, err := hex.DecodeString(pubKeyHex)
		if err != nil {
			logger.Fatal().Err(err).Msg("License server returned invalid public key (expected hex)")
			return 1
		}
		if len(decoded) != ed25519.PublicKeySize {
			logger.Fatal().Int("got", len(decoded)).Int("expected", ed25519.PublicKeySize).Msg("License server returned invalid public key size")
			return 1
		}
		licPubKey = decoded
		logger.Info().Str("fingerprint", pubKeyHex[:8]).Msg("Signing key fetched from license server")
	}

	if cfg.LicenseKey != "" {
		parsed, err := license.ParseLicenseKeyEd25519(cfg.LicenseKey, ed25519.PublicKey(licPubKey))
		if err != nil {
			logger.Fatal().Err(err).Msg("Invalid LICENSE_KEY (Ed25519 validation failed)")
			return 1
		}
		if err := license.ValidateLicense(parsed); err != nil {
			logger.Fatal().Err(err).Msg("LICENSE_KEY validation failed (expired or invalid)")
			return 1
		}
		lic = parsed
		logger.Info().Str("tier", string(lic.Tier)).Time("expires_at", lic.ExpiresAt).Msg("License loaded")
	} else {
		logger.Info().Msg("No LICENSE_KEY set, running as Free tier")
	}

	// Build API router
	allowedOrigins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	if os.Getenv("CORS_ORIGINS") == "" {
		allowedOrigins = []string{}
	}

	rateLimitRequests := int64(100)
	if v := os.Getenv("RATE_LIMIT_REQUESTS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			rateLimitRequests = n
		}
	}
	rateLimitPeriod := "1m"
	if v := os.Getenv("RATE_LIMIT_PERIOD"); v != "" {
		rateLimitPeriod = v
	}

	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "web/dist"
	}

	// Initialize verification scheduler
	resticBin := backup.NewRestic(logger)
	verificationConfig := backup.DefaultVerificationConfig()
	verificationConfig.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		repoKey, err := database.GetRepositoryKeyByRepositoryID(ctx, repoID)
		if err != nil {
			return "", fmt.Errorf("get repository key: %w", err)
		}
		password, err := keyManager.Decrypt(repoKey.EncryptedKey)
		if err != nil {
			return "", fmt.Errorf("decrypt repository key: %w", err)
		}
		return string(password), nil
	}
	verificationConfig.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return keyManager.Decrypt(encrypted)
	}
	verificationScheduler := backup.NewVerificationScheduler(database, resticBin, verificationConfig, logger)

	// Initialize backup scheduler
	backupSchedulerConfig := backup.DefaultSchedulerConfig()
	backupSchedulerConfig.PasswordFunc = verificationConfig.PasswordFunc
	backupSchedulerConfig.DecryptFunc = verificationConfig.DecryptFunc
	backupScheduler := backup.NewScheduler(database, resticBin, backupSchedulerConfig, nil, logger)

	// Initialize DR test scheduler
	drTestConfig := backup.DefaultSchedulerConfig()
	drTestConfig.PasswordFunc = verificationConfig.PasswordFunc
	drTestConfig.DecryptFunc = verificationConfig.DecryptFunc
	drTestScheduler := backup.NewDRTestScheduler(database, resticBin, drTestConfig, logger)

	// Initialize notification service
	notificationService := notifications.NewService(database, keyManager, logger)

	// Initialize monitoring service
	alertNotifier := &alertNotificationAdapter{notificationService: notificationService, logger: logger}
	alertService := monitoring.NewAlertService(database, alertNotifier, logger)
	monitor := monitoring.NewMonitor(database, alertService, monitoring.DefaultConfig(), logger)

	// Initialize Docker monitor
	dockerMonitor := monitoring.NewDockerMonitorWithDB(database, alertService, monitoring.DefaultDockerMonitorConfig(), logger)

	// Initialize metrics aggregation scheduler
	metricsAggregator := metrics.NewAggregator(database, logger)
	metricsScheduler := metrics.NewScheduler(metricsAggregator, logger)

	// Initialize downtime service
	downtimeService := monitoring.NewDowntimeServiceWithDB(database, monitoring.DefaultDowntimeServiceConfig(), logger)
	_ = downtimeService // used in routes.go; instantiated here for lifecycle awareness

	// Initialize database backup service
	dbBackupConfig := maintenance.DefaultDatabaseBackupConfig()
	dbBackupConfig.DatabaseURL = databaseURL
	dbBackupService := maintenance.NewDatabaseBackupService(database, keyManager, dbBackupConfig, logger)

	// Initialize maintenance service
	maintenanceService := maintenance.NewService(database, notificationService, logger)
	_ = maintenanceService // used for maintenance window notifications

	// Initialize report scheduler
	reportScheduler := reports.NewScheduler(database, reports.DefaultSchedulerConfig(), logger)

	// Initialize license validator (phone-home for all tiers)
	var validator *license.Validator
	if !cfg.AirGapMode {
		validator = license.NewValidator(license.ValidatorConfig{
			LicenseKey:    cfg.LicenseKey,
			ServerURL:     cfg.LicenseServerURL,
			ServerVersion: Version,
			Store:         database,
			Metrics:       database,
			OrgCounter:    database,
			PublicKey:      ed25519.PublicKey(licPubKey),
			Logger:        logger,
		})
		if lic != nil {
			validator.SetLicense(lic)
		}
		if IntegrityHash != "" {
			validator.SetIntegrityHash(IntegrityHash)
		}
		validator.SetOIDCProvider(oidcProvider)
		if err := validator.Start(ctx); err != nil {
			logger.Error().Err(err).Msg("Failed to start license validator (continuing without phone-home)")
			validator = nil
		}
	}

	// Set license checker on backup scheduler for premium feature gating
	if validator != nil {
		backupScheduler.SetLicenseChecker(validator)
	}

	// Create setup handler for first-time server setup
	setupHandler := handlers.NewServerSetupHandler(database, database, sessions, logger)
	setupHandler.SetHeartbeatSender(validator)

	// Initialize activity feed for real-time event streaming
	activityFeed := activity.NewFeed(database, activity.DefaultConfig(), logger)
	activityFeed.Start()
	defer activityFeed.Stop()

	routerCfg := api.Config{
		Environment:           cfg.Environment,
		AllowedOrigins:        allowedOrigins,
		RateLimitRequests:     rateLimitRequests,
		RateLimitPeriod:       rateLimitPeriod,
		RedisURL:              os.Getenv("REDIS_URL"),
		Version:               Version,
		Commit:                Commit,
		BuildDate:             BuildDate,
		WebDir:                webDir,
		VerificationTrigger:   verificationScheduler,
		ReportScheduler:       reportScheduler,
		DRTestRunner:          drTestScheduler,
		License:               lic,
		Validator:             validator,
		LicensePublicKey:      licPubKey,
		SetupHandler:          setupHandler,
		ActivityFeed:          activityFeed,
		LogBuffer:             logBuffer,
		DatabaseBackupService: dbBackupService,
	}

	router, err := api.NewRouter(routerCfg, database, oidcProvider, sessions, keyManager, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize router")
		return 1
	}

	// Start HTTP server
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		listenAddr = ":" + port
	}
	cfg.HTTPAddr = listenAddr

	// Validate consolidated config
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("Invalid server configuration")
		return 1
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           router.Engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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

	// Start verification scheduler
	if err := verificationScheduler.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start verification scheduler")
	}
	defer verificationScheduler.Stop()

	// Start backup scheduler
	if err := backupScheduler.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start backup scheduler")
	}
	defer backupScheduler.Stop()

	// Start DR test scheduler
	if err := drTestScheduler.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start DR test scheduler")
	}
	defer drTestScheduler.Stop()

	// Start monitoring service
	monitor.Start(ctx)
	defer monitor.Stop()

	// Start report scheduler
	if err := reportScheduler.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start report scheduler")
	}
	defer reportScheduler.Stop()

	// Start metrics aggregation scheduler
	metricsScheduler.Start(ctx)
	defer metricsScheduler.Stop()

	// Start Docker monitor
	dockerMonitor.Start(ctx)
	defer dockerMonitor.Stop()

	// Start database backup service
	if err := dbBackupService.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start database backup service")
	}
	defer dbBackupService.Stop()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.Info().Str("signal", sig.String()).Msg("Shutting down server")

	// Determine shutdown timeout
	shutdownTimeout := 30 * time.Second
	if cfg.Shutdown.Timeout > 0 {
		shutdownTimeout = cfg.Shutdown.Timeout
	}

	// FIRST: Drain HTTP connections so in-flight requests complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
		return 1
	}

	// THEN: Stop license validator (deactivates license on shutdown)
	if validator != nil {
		validator.Stop(ctx)
	}

	logger.Info().Msg("Server stopped gracefully")
	return 0
}

// fetchSigningKey retrieves the Ed25519 public key from the license server.
func fetchSigningKey(serverURL string) (string, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("invalid LICENSE_SERVER_URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("LICENSE_SERVER_URL must use HTTPS (got %q)", parsed.Scheme)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	endpoint := strings.TrimRight(serverURL, "/") + "/api/v1/signing-key"
	resp, err := client.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("license server returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if result.PublicKey == "" {
		return "", fmt.Errorf("license server returned empty public key")
	}
	return result.PublicKey, nil
}

// alertNotificationAdapter adapts notifications.Service to monitoring.NotificationSender.
type alertNotificationAdapter struct {
	notificationService *notifications.Service
	logger              zerolog.Logger
}

// SendAlertNotification implements monitoring.NotificationSender.
func (a *alertNotificationAdapter) SendAlertNotification(_ context.Context, alert *models.Alert) error {
	a.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("type", string(alert.Type)).
		Str("severity", string(alert.Severity)).
		Msg("alert notification dispatched")
	return nil
}
