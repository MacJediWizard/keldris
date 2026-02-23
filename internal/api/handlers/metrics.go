package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/metrics"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// MetricsStore defines the interface for retrieving metrics data.
type MetricsStore interface {
	Ping(ctx context.Context) error
	Health() map[string]any
	// Prometheus metrics methods
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)
	GetAllBackups(ctx context.Context) ([]*models.Backup, error)
	GetBackupsByStatus(ctx context.Context, status models.BackupStatus) ([]*models.Backup, error)
	GetStorageStatsSummaryGlobal(ctx context.Context) (*models.StorageStatsSummary, error)
}

// MetricsHandler handles Prometheus-compatible metrics endpoints.
type MetricsHandler struct {
	db                  MetricsStore
	prometheusCollector *metrics.PrometheusCollector
	logger              zerolog.Logger
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(db MetricsStore, logger zerolog.Logger) *MetricsHandler {
	h := &MetricsHandler{
		db:     db,
		logger: logger.With().Str("component", "metrics_handler").Logger(),
	}
	if db != nil {
		h.prometheusCollector = metrics.NewPrometheusCollector(db, logger)
	}
	return h
}

// RegisterPublicRoutes registers metrics routes that don't require authentication.
func (h *MetricsHandler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/metrics", h.Metrics)
}

// Metrics returns metrics in Prometheus exposition format.
// @Summary Prometheus metrics endpoint
// @Description Returns metrics in Prometheus exposition format for scraping
// @Tags Monitoring
// @Produce text/plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func (h *MetricsHandler) Metrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var sb strings.Builder

	// Server info metric
	sb.WriteString("# HELP keldris_info Server information\n")
	sb.WriteString("# TYPE keldris_info gauge\n")
	sb.WriteString("keldris_info{component=\"server\"} 1\n")
	sb.WriteString("\n")

	// Server up metric
	sb.WriteString("# HELP keldris_up Server health status (1 = healthy, 0 = unhealthy)\n")
	sb.WriteString("# TYPE keldris_up gauge\n")

	// Check database health
	dbHealthy := 1
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			dbHealthy = 0
			h.logger.Warn().Err(err).Msg("database ping failed for metrics")
		}
	} else {
		dbHealthy = 0
	}
	sb.WriteString(fmt.Sprintf("keldris_up{component=\"database\"} %d\n", dbHealthy))
	sb.WriteString("\n")

	// Database pool metrics
	if h.db != nil {
		poolStats := h.db.Health()

		sb.WriteString("# HELP keldris_db_connections_total Total number of connections in the pool\n")
		sb.WriteString("# TYPE keldris_db_connections_total gauge\n")
		if v, ok := poolStats["total_conns"].(int32); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_connections_total %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_connections_acquired Number of currently acquired connections\n")
		sb.WriteString("# TYPE keldris_db_connections_acquired gauge\n")
		if v, ok := poolStats["acquired_conns"].(int32); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_connections_acquired %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_connections_idle Number of idle connections\n")
		sb.WriteString("# TYPE keldris_db_connections_idle gauge\n")
		if v, ok := poolStats["idle_conns"].(int32); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_connections_idle %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_connections_max Maximum number of connections in the pool\n")
		sb.WriteString("# TYPE keldris_db_connections_max gauge\n")
		if v, ok := poolStats["max_conns"].(int32); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_connections_max %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_connections_constructing Number of connections being constructed\n")
		sb.WriteString("# TYPE keldris_db_connections_constructing gauge\n")
		if v, ok := poolStats["constructing"].(int32); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_connections_constructing %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_acquire_empty_total Total number of acquire attempts that had to wait for a connection\n")
		sb.WriteString("# TYPE keldris_db_acquire_empty_total counter\n")
		if v, ok := poolStats["empty_acquire"].(int64); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_acquire_empty_total %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_acquire_canceled_total Total number of acquire attempts that were canceled\n")
		sb.WriteString("# TYPE keldris_db_acquire_canceled_total counter\n")
		if v, ok := poolStats["canceled_acquire"].(int64); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_acquire_canceled_total %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_lifetime_destroy_total Total number of connections destroyed due to max lifetime\n")
		sb.WriteString("# TYPE keldris_db_lifetime_destroy_total counter\n")
		if v, ok := poolStats["max_lifetime_dest"].(int64); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_lifetime_destroy_total %d\n", v))
		}
		sb.WriteString("\n")

		sb.WriteString("# HELP keldris_db_idle_destroy_total Total number of connections destroyed due to max idle time\n")
		sb.WriteString("# TYPE keldris_db_idle_destroy_total counter\n")
		if v, ok := poolStats["max_idle_dest"].(int64); ok {
			sb.WriteString(fmt.Sprintf("keldris_db_idle_destroy_total %d\n", v))
		}
		sb.WriteString("\n")
	}

	// Collect and append Prometheus metrics (backup, agent, storage)
	if h.prometheusCollector != nil {
		promMetrics, err := h.prometheusCollector.Collect(ctx)
		if err != nil {
			h.logger.Warn().Err(err).Msg("failed to collect prometheus metrics")
		} else {
			sb.WriteString(h.prometheusCollector.Format(promMetrics))
		}
	}

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, sb.String())
}
