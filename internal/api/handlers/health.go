package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/maintenance"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDraining  HealthStatus = "draining"
)

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	Status   HealthStatus   `json:"status"`
	Duration string         `json:"duration,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// HealthResponse is the response for health check endpoints.
type HealthResponse struct {
	Status HealthStatus                  `json:"status"`
	Checks map[string]*HealthCheckResult `json:"checks,omitempty"`
	Error  string                        `json:"error,omitempty"`
	Status HealthStatus              `json:"status"`
	Checks map[string]*HealthCheckResult `json:"checks,omitempty"`
	Error  string                        `json:"error,omitempty"`
}

// DatabaseHealthChecker defines the interface for database health checking.
type DatabaseHealthChecker interface {
	Ping(ctx context.Context) error
	Health() map[string]any
}

// OIDCHealthChecker defines the interface for OIDC provider health checking.
type OIDCHealthChecker interface {
	// HealthCheck attempts to verify the OIDC provider is reachable.
	HealthCheck(ctx context.Context) error
}

// ShutdownStatus represents the current shutdown status.
type ShutdownStatus struct {
	State             string        `json:"state"`
	StartedAt         *time.Time    `json:"started_at,omitempty"`
	TimeRemaining     time.Duration `json:"time_remaining,omitempty"`
	RunningBackups    int           `json:"running_backups"`
	CheckpointedCount int           `json:"checkpointed_count"`
	AcceptingNewJobs  bool          `json:"accepting_new_jobs"`
	Message           string        `json:"message,omitempty"`
}

// ShutdownStatusProvider defines the interface for providing shutdown status.
type ShutdownStatusProvider interface {
	// GetStatus returns the current shutdown status.
	GetStatus() ShutdownStatus
	// IsAcceptingJobs returns true if the server is accepting new backup jobs.
	IsAcceptingJobs() bool
}

// HealthHandler handles health-related HTTP endpoints.
type HealthHandler struct {
	db            DatabaseHealthChecker
	oidc          OIDCHealthChecker
	sessions      *auth.SessionStore
	backupService *maintenance.DatabaseBackupService
	shutdown      ShutdownStatusProvider
	logger        zerolog.Logger
// HealthHandler handles health-related HTTP endpoints.
type HealthHandler struct {
	db            DatabaseHealthChecker
	oidc          OIDCHealthChecker
	sessions      *auth.SessionStore
	backupService *maintenance.DatabaseBackupService
	logger        zerolog.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db DatabaseHealthChecker, oidc OIDCHealthChecker, logger zerolog.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		oidc:   oidc,
		logger: logger.With().Str("component", "health_handler").Logger(),
	}
}

// SetSessionStore sets the session store for authenticated routes.
func (h *HealthHandler) SetSessionStore(sessions *auth.SessionStore) {
	h.sessions = sessions
}

// SetDatabaseBackupService sets the database backup service for health checks.
func (h *HealthHandler) SetDatabaseBackupService(service *maintenance.DatabaseBackupService) {
	h.backupService = service
}

// SetShutdownStatusProvider sets the shutdown status provider.
// This should be called after server initialization to enable shutdown status endpoint.
func (h *HealthHandler) SetShutdownStatusProvider(provider ShutdownStatusProvider) {
	h.shutdown = provider
}

// RegisterPublicRoutes registers health check routes that don't require authentication.
func (h *HealthHandler) RegisterPublicRoutes(r *gin.Engine) {
	health := r.Group("/health")
	{
		health.GET("", h.Overall)
		health.GET("/db", h.Database)
		health.GET("/oidc", h.OIDC)
		health.GET("/shutdown", h.Shutdown)
	}
}

// RegisterRoutes registers authenticated health check routes.
func (h *HealthHandler) RegisterRoutes(r *gin.RouterGroup) {
	health := r.Group("/health")
	{
		health.GET("/system", h.SystemHealth)
	}
}

// RegisterRoutes registers authenticated health check routes.
func (h *HealthHandler) RegisterRoutes(r *gin.RouterGroup) {
	health := r.Group("/health")
	{
		health.GET("/system", h.SystemHealth)
	}
}

// Overall returns the overall server health status.
// GET /health
func (h *HealthHandler) Overall(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := &HealthResponse{
		Status: HealthStatusHealthy,
		Checks: make(map[string]*HealthCheckResult),
	}

	// Check database health
	dbResult := h.checkDatabase(ctx)
	response.Checks["database"] = dbResult

	// Check OIDC health
	oidcResult := h.checkOIDC(ctx)
	response.Checks["oidc"] = oidcResult

	// Determine overall status
	if dbResult.Status == HealthStatusUnhealthy || oidcResult.Status == HealthStatusUnhealthy {
		response.Status = HealthStatusUnhealthy
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Database returns the database health status.
// GET /health/db
func (h *HealthHandler) Database(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result := h.checkDatabase(ctx)

	response := &HealthResponse{
		Status: result.Status,
		Checks: map[string]*HealthCheckResult{
			"database": result,
		},
	}

	if result.Status == HealthStatusUnhealthy {
		response.Error = result.Error
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// OIDC returns the OIDC provider health status.
// GET /health/oidc
func (h *HealthHandler) OIDC(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result := h.checkOIDC(ctx)

	response := &HealthResponse{
		Status: result.Status,
		Checks: map[string]*HealthCheckResult{
			"oidc": result,
		},
	}

	if result.Status == HealthStatusUnhealthy {
		response.Error = result.Error
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// checkDatabase performs a database health check.
func (h *HealthHandler) checkDatabase(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		Status: HealthStatusHealthy,
	}

	if h.db == nil {
		result.Status = HealthStatusUnhealthy
		result.Error = "database not configured"
		result.Duration = time.Since(start).String()
		return result
	}

	err := h.db.Ping(ctx)
	result.Duration = time.Since(start).String()

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = "database ping failed"
		h.logger.Warn().Err(err).Msg("database health check failed")
		return result
	}

	// Include pool stats
	result.Details = h.db.Health()

	return result
}

// checkOIDC performs an OIDC provider health check.
func (h *HealthHandler) checkOIDC(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		Status: HealthStatusHealthy,
	}

	if h.oidc == nil {
		// OIDC is optional - if not configured, it's not unhealthy
		result.Details = map[string]any{"configured": false}
		result.Duration = time.Since(start).String()
		return result
	}

	err := h.oidc.HealthCheck(ctx)
	result.Duration = time.Since(start).String()

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = "OIDC provider unreachable"
		h.logger.Warn().Err(err).Msg("OIDC health check failed")
		return result
	}

	result.Details = map[string]any{"configured": true}

	return result
}

// ShutdownResponse is the response for the shutdown status endpoint.
type ShutdownResponse struct {
	State             string `json:"state"`
	StartedAt         string `json:"started_at,omitempty"`
	TimeRemaining     string `json:"time_remaining,omitempty"`
	RunningBackups    int    `json:"running_backups"`
	CheckpointedCount int    `json:"checkpointed_count"`
	AcceptingNewJobs  bool   `json:"accepting_new_jobs"`
	Message           string `json:"message,omitempty"`
}

// Shutdown returns the current shutdown status.
// GET /health/shutdown
// @Summary Get shutdown status
// @Description Returns the current shutdown state, including whether the server is accepting new jobs and the status of running backups
// @Tags Health
// @Produce json
// @Success 200 {object} ShutdownResponse
// @Router /health/shutdown [get]
func (h *HealthHandler) Shutdown(c *gin.Context) {
	if h.shutdown == nil {
		// Shutdown provider not configured, return default running state
		response := ShutdownResponse{
			State:            "running",
			AcceptingNewJobs: true,
			Message:          "Server is running normally",
		}
		c.JSON(http.StatusOK, response)
		return
	}

	status := h.shutdown.GetStatus()

	response := ShutdownResponse{
		State:             status.State,
		RunningBackups:    status.RunningBackups,
		CheckpointedCount: status.CheckpointedCount,
		AcceptingNewJobs:  status.AcceptingNewJobs,
		Message:           status.Message,
	}

	if status.StartedAt != nil {
		response.StartedAt = status.StartedAt.Format(time.RFC3339)
	}

	if status.TimeRemaining > 0 {
		response.TimeRemaining = status.TimeRemaining.String()
	}

	// Return 503 Service Unavailable if the server is shutting down
	if status.State != "running" {
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// SystemHealth returns comprehensive system health including database backups.
// GET /api/v1/health/system
// This endpoint requires superuser privileges.
func (h *HealthHandler) SystemHealth(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := &HealthResponse{
		Status: HealthStatusHealthy,
		Checks: make(map[string]*HealthCheckResult),
	}

	// Check database health
	dbResult := h.checkDatabase(ctx)
	response.Checks["database"] = dbResult

	// Check OIDC health
	oidcResult := h.checkOIDC(ctx)
	response.Checks["oidc"] = oidcResult

	// Check database backup health
	backupResult := h.checkDatabaseBackup(ctx)
	response.Checks["database_backup"] = backupResult

	// Determine overall status
	hasUnhealthy := false
	for _, check := range response.Checks {
		if check.Status == HealthStatusUnhealthy {
			hasUnhealthy = true
			break
		}
	}

	if hasUnhealthy {
		response.Status = HealthStatusUnhealthy
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// checkDatabaseBackup performs a database backup service health check.
func (h *HealthHandler) checkDatabaseBackup(ctx context.Context) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		Status: HealthStatusHealthy,
	}

	if h.backupService == nil {
		result.Details = map[string]any{
			"configured": false,
			"message":    "database backup service not configured",
		}
		result.Duration = time.Since(start).String()
		return result
	}

	healthy, message := h.backupService.IsHealthy(ctx)
	result.Duration = time.Since(start).String()

	if !healthy {
		result.Status = HealthStatusUnhealthy
		result.Error = message
		result.Details = map[string]any{
			"configured": true,
			"healthy":    false,
		}
		h.logger.Warn().Str("message", message).Msg("database backup health check failed")
		return result
	}

	// Get additional status info
	status, err := h.backupService.GetStatus(ctx)
	if err == nil && status != nil {
		result.Details = map[string]any{
			"configured":       true,
			"healthy":          true,
			"enabled":          status.Enabled,
			"total_backups":    status.TotalBackups,
			"total_size_bytes": status.TotalSizeBytes,
			"schedule":         status.Schedule,
			"retention_days":   status.Retention,
		}
		if status.LastBackupTime != nil {
			result.Details["last_backup_time"] = status.LastBackupTime
		}
		if status.NextBackupTime != nil {
			result.Details["next_backup_time"] = status.NextBackupTime
		}
		if status.LastBackupStatus != "" {
			result.Details["last_backup_status"] = status.LastBackupStatus
		}
	} else {
		result.Details = map[string]any{
			"configured": true,
			"healthy":    true,
		}
	}

	return result
}
