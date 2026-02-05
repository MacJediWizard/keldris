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
	Status HealthStatus              `json:"status"`
	Checks map[string]*HealthCheckResult `json:"checks,omitempty"`
	Error  string                    `json:"error,omitempty"`
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

// RegisterPublicRoutes registers health check routes that don't require authentication.
func (h *HealthHandler) RegisterPublicRoutes(r *gin.Engine) {
	health := r.Group("/health")
	{
		health.GET("", h.Overall)
		health.GET("/db", h.Database)
		health.GET("/oidc", h.OIDC)
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
