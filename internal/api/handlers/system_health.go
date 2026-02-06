package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SystemHealthStore defines the interface for system health-related persistence operations.
type SystemHealthStore interface {
	// Database health
	Ping(ctx context.Context) error
	Health() map[string]any
	GetDatabaseSize(ctx context.Context) (int64, error)
	GetActiveConnections(ctx context.Context) (int, error)

	// Queue status
	GetPendingBackupsCount(ctx context.Context, orgID uuid.UUID) (int, error)
	GetRunningBackupsCount(ctx context.Context, orgID uuid.UUID) (int, error)

	// Recent errors
	GetRecentServerErrors(ctx context.Context, limit int) ([]*db.ServerError, error)
	GetHealthHistoryRecords(ctx context.Context, since time.Time) ([]*db.HealthHistoryRecord, error)
	SaveHealthHistoryRecord(ctx context.Context, record *db.HealthHistoryRecord) error
}

// SystemHealthStatus represents the overall system health status.
type SystemHealthStatus string

const (
	SystemHealthStatusHealthy  SystemHealthStatus = "healthy"
	SystemHealthStatusWarning  SystemHealthStatus = "warning"
	SystemHealthStatusCritical SystemHealthStatus = "critical"
)

// ServerStatus contains server resource information.
type ServerStatus struct {
	Status           SystemHealthStatus `json:"status"`
	CPUUsage         float64            `json:"cpu_usage"`
	MemoryUsage      float64            `json:"memory_usage"`
	MemoryAllocMB    float64            `json:"memory_alloc_mb"`
	MemoryTotalAllocMB float64          `json:"memory_total_alloc_mb"`
	MemorySysMB      float64            `json:"memory_sys_mb"`
	GoroutineCount   int                `json:"goroutine_count"`
	NumCPU           int                `json:"num_cpu"`
	GoVersion        string             `json:"go_version"`
	UptimeSeconds    int64              `json:"uptime_seconds"`
}

// DatabaseStatus contains database health information.
type DatabaseStatus struct {
	Status          SystemHealthStatus `json:"status"`
	Connected       bool               `json:"connected"`
	Latency         string             `json:"latency"`
	ActiveConnections int              `json:"active_connections"`
	MaxConnections  int                `json:"max_connections"`
	SizeBytes       int64              `json:"size_bytes"`
	SizeFormatted   string             `json:"size_formatted"`
}

// QueueStatus contains backup queue information.
type QueueStatus struct {
	Status         SystemHealthStatus `json:"status"`
	PendingBackups int                `json:"pending_backups"`
	RunningBackups int                `json:"running_backups"`
	TotalQueued    int                `json:"total_queued"`
}

// BackgroundJobStatus contains background job information.
type BackgroundJobStatus struct {
	Status         SystemHealthStatus `json:"status"`
	GoroutineCount int                `json:"goroutine_count"`
	ActiveJobs     int                `json:"active_jobs"`
}

// SystemHealthResponse is the response for the admin health endpoint.
type SystemHealthResponse struct {
	Status         SystemHealthStatus `json:"status"`
	Timestamp      time.Time          `json:"timestamp"`
	Server         ServerStatus       `json:"server"`
	Database       DatabaseStatus     `json:"database"`
	Queue          QueueStatus        `json:"queue"`
	BackgroundJobs BackgroundJobStatus `json:"background_jobs"`
	RecentErrors   []*db.ServerError  `json:"recent_errors"`
	Issues         []string           `json:"issues,omitempty"`
}

// SystemHealthHistoryResponse is the response for health history.
type SystemHealthHistoryResponse struct {
	Records []*db.HealthHistoryRecord `json:"records"`
	Since   time.Time                 `json:"since"`
	Until   time.Time                 `json:"until"`
}

// SystemHealthHandler handles system health-related HTTP endpoints.
type SystemHealthHandler struct {
	store     SystemHealthStore
	sessions  *auth.SessionStore
	logger    zerolog.Logger
	startTime time.Time
}

// NewSystemHealthHandler creates a new SystemHealthHandler.
func NewSystemHealthHandler(store SystemHealthStore, sessions *auth.SessionStore, logger zerolog.Logger) *SystemHealthHandler {
	return &SystemHealthHandler{
		store:     store,
		sessions:  sessions,
		logger:    logger.With().Str("component", "system_health_handler").Logger(),
		startTime: time.Now(),
	}
}

// RegisterRoutes registers system health routes on the given router group.
// These routes require superuser privileges.
func (h *SystemHealthHandler) RegisterRoutes(r *gin.RouterGroup) {
	health := r.Group("/admin/health")
	health.Use(middleware.SuperuserMiddleware(h.sessions, h.logger))
	{
		health.GET("", h.GetSystemHealth)
		health.GET("/history", h.GetHealthHistory)
	}
}

// GetSystemHealth returns the current system health status.
// GET /api/v1/admin/health
func (h *SystemHealthHandler) GetSystemHealth(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx := c.Request.Context()
	response := &SystemHealthResponse{
		Status:    SystemHealthStatusHealthy,
		Timestamp: time.Now(),
		Issues:    []string{},
	}

	// Collect server status
	response.Server = h.getServerStatus()

	// Collect database status
	response.Database = h.getDatabaseStatus(ctx)

	// Collect queue status
	response.Queue = h.getQueueStatus(ctx)

	// Collect background job status
	response.BackgroundJobs = h.getBackgroundJobStatus()

	// Get recent errors
	errors, err := h.store.GetRecentServerErrors(ctx, 10)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get recent errors")
		response.RecentErrors = []*db.ServerError{}
	} else {
		response.RecentErrors = errors
	}

	// Determine overall status
	response.Status = h.determineOverallStatus(response)

	// Add issues if any component is unhealthy
	if response.Server.Status != SystemHealthStatusHealthy {
		response.Issues = append(response.Issues, "Server resource usage is elevated")
	}
	if response.Database.Status != SystemHealthStatusHealthy {
		response.Issues = append(response.Issues, "Database health degraded")
	}
	if response.Queue.Status != SystemHealthStatusHealthy {
		response.Issues = append(response.Issues, "Backup queue is growing")
	}
	if len(response.RecentErrors) > 5 {
		response.Issues = append(response.Issues, "High number of recent errors")
	}

	// Save health record for history
	h.saveHealthRecord(ctx, response)

	c.JSON(http.StatusOK, response)
}

// GetHealthHistory returns historical health data.
// GET /api/v1/admin/health/history
func (h *SystemHealthHandler) GetHealthHistory(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx := c.Request.Context()

	// Default to last 24 hours
	since := time.Now().Add(-24 * time.Hour)

	records, err := h.store.GetHealthHistoryRecords(ctx, since)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get health history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get health history"})
		return
	}

	response := &SystemHealthHistoryResponse{
		Records: records,
		Since:   since,
		Until:   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// getServerStatus collects server resource information.
func (h *SystemHealthHandler) getServerStatus() ServerStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := ServerStatus{
		Status:           SystemHealthStatusHealthy,
		MemoryAllocMB:    float64(memStats.Alloc) / 1024 / 1024,
		MemoryTotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		MemorySysMB:      float64(memStats.Sys) / 1024 / 1024,
		GoroutineCount:   runtime.NumGoroutine(),
		NumCPU:           runtime.NumCPU(),
		GoVersion:        runtime.Version(),
		UptimeSeconds:    int64(time.Since(h.startTime).Seconds()),
	}

	// Calculate memory usage percentage (of Sys memory)
	if memStats.Sys > 0 {
		status.MemoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	}

	// Check thresholds
	if status.MemoryUsage > 90 || status.GoroutineCount > 10000 {
		status.Status = SystemHealthStatusCritical
	} else if status.MemoryUsage > 75 || status.GoroutineCount > 5000 {
		status.Status = SystemHealthStatusWarning
	}

	return status
}

// getDatabaseStatus checks database health.
func (h *SystemHealthHandler) getDatabaseStatus(ctx context.Context) DatabaseStatus {
	status := DatabaseStatus{
		Status:    SystemHealthStatusHealthy,
		Connected: true,
	}

	// Check database connectivity with timing
	start := time.Now()
	err := h.store.Ping(ctx)
	status.Latency = time.Since(start).String()

	if err != nil {
		status.Status = SystemHealthStatusCritical
		status.Connected = false
		return status
	}

	// Get pool stats
	poolStats := h.store.Health()
	if maxConns, ok := poolStats["max_connections"].(int32); ok {
		status.MaxConnections = int(maxConns)
	}
	if totalConns, ok := poolStats["total_connections"].(int32); ok {
		status.ActiveConnections = int(totalConns)
	}

	// Get database size
	size, err := h.store.GetDatabaseSize(ctx)
	if err == nil {
		status.SizeBytes = size
		status.SizeFormatted = formatBytes(size)
	}

	// Get active connections count
	activeConns, err := h.store.GetActiveConnections(ctx)
	if err == nil {
		status.ActiveConnections = activeConns
	}

	// Check thresholds
	latency := time.Since(start)
	if latency > 5*time.Second || (status.MaxConnections > 0 && status.ActiveConnections >= status.MaxConnections-5) {
		status.Status = SystemHealthStatusCritical
	} else if latency > 1*time.Second || (status.MaxConnections > 0 && float64(status.ActiveConnections)/float64(status.MaxConnections) > 0.8) {
		status.Status = SystemHealthStatusWarning
	}

	return status
}

// getQueueStatus checks backup queue status.
func (h *SystemHealthHandler) getQueueStatus(ctx context.Context) QueueStatus {
	status := QueueStatus{
		Status: SystemHealthStatusHealthy,
	}

	// Get pending backups count across all orgs (uuid.Nil for all)
	pending, err := h.store.GetPendingBackupsCount(ctx, uuid.Nil)
	if err == nil {
		status.PendingBackups = pending
	}

	// Get running backups count
	running, err := h.store.GetRunningBackupsCount(ctx, uuid.Nil)
	if err == nil {
		status.RunningBackups = running
	}

	status.TotalQueued = status.PendingBackups + status.RunningBackups

	// Check thresholds
	if status.PendingBackups > 100 {
		status.Status = SystemHealthStatusCritical
	} else if status.PendingBackups > 50 {
		status.Status = SystemHealthStatusWarning
	}

	return status
}

// getBackgroundJobStatus checks background job status.
func (h *SystemHealthHandler) getBackgroundJobStatus() BackgroundJobStatus {
	goroutines := runtime.NumGoroutine()

	status := BackgroundJobStatus{
		Status:         SystemHealthStatusHealthy,
		GoroutineCount: goroutines,
		ActiveJobs:     goroutines - 10, // Subtract base goroutines
	}

	if status.ActiveJobs < 0 {
		status.ActiveJobs = 0
	}

	// Check thresholds
	if goroutines > 10000 {
		status.Status = SystemHealthStatusCritical
	} else if goroutines > 5000 {
		status.Status = SystemHealthStatusWarning
	}

	return status
}

// determineOverallStatus determines the overall system health status.
func (h *SystemHealthHandler) determineOverallStatus(response *SystemHealthResponse) SystemHealthStatus {
	// Critical if any component is critical
	if response.Server.Status == SystemHealthStatusCritical ||
		response.Database.Status == SystemHealthStatusCritical ||
		response.Queue.Status == SystemHealthStatusCritical ||
		response.BackgroundJobs.Status == SystemHealthStatusCritical {
		return SystemHealthStatusCritical
	}

	// Warning if any component is warning
	if response.Server.Status == SystemHealthStatusWarning ||
		response.Database.Status == SystemHealthStatusWarning ||
		response.Queue.Status == SystemHealthStatusWarning ||
		response.BackgroundJobs.Status == SystemHealthStatusWarning {
		return SystemHealthStatusWarning
	}

	// Check recent error count
	if len(response.RecentErrors) > 5 {
		return SystemHealthStatusWarning
	}

	return SystemHealthStatusHealthy
}

// saveHealthRecord saves the current health status to history.
func (h *SystemHealthHandler) saveHealthRecord(ctx context.Context, response *SystemHealthResponse) {
	record := &db.HealthHistoryRecord{
		ID:                  uuid.New().String(),
		Timestamp:           response.Timestamp,
		Status:              string(response.Status),
		CPUUsage:            response.Server.CPUUsage,
		MemoryUsage:         response.Server.MemoryUsage,
		MemoryAllocMB:       response.Server.MemoryAllocMB,
		MemoryTotalAllocMB:  response.Server.MemoryTotalAllocMB,
		GoroutineCount:      response.Server.GoroutineCount,
		DatabaseConnections: response.Database.ActiveConnections,
		DatabaseSizeBytes:   response.Database.SizeBytes,
		PendingBackups:      response.Queue.PendingBackups,
		RunningBackups:      response.Queue.RunningBackups,
		ErrorCount:          len(response.RecentErrors),
	}

	go func() {
		if err := h.store.SaveHealthHistoryRecord(context.Background(), record); err != nil {
			h.logger.Warn().Err(err).Msg("failed to save health history record")
		}
	}()
}

// formatBytes formats bytes as a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
