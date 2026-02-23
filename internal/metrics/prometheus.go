package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// PrometheusMetrics holds all Prometheus metric collectors for Keldris.
type PrometheusMetrics struct {
	BackupCounter  *prometheus.CounterVec
	BackupDuration *prometheus.HistogramVec
	AgentGauge     *prometheus.GaugeVec
	StorageGauge   *prometheus.GaugeVec
}

// NewPrometheusMetrics creates and registers Prometheus metrics.
func NewPrometheusMetrics(reg prometheus.Registerer) (*PrometheusMetrics, error) {
	m := &PrometheusMetrics{
		BackupCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "keldris",
				Subsystem: "backup",
				Name:      "total",
				Help:      "Total number of backups by status.",
			},
			[]string{"status"},
		),
		BackupDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "keldris",
				Subsystem: "backup",
				Name:      "duration_seconds",
				Help:      "Duration of backup operations in seconds.",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 12),
			},
			[]string{"schedule"},
		),
		AgentGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "keldris",
				Subsystem: "agent",
				Name:      "count",
				Help:      "Current number of agents by status.",
			},
			[]string{"status"},
		),
		StorageGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "keldris",
				Subsystem: "storage",
				Name:      "bytes",
				Help:      "Current storage usage in bytes by type.",
			},
			[]string{"type"},
		),
	}

	collectors := []prometheus.Collector{
		m.BackupCounter,
		m.BackupDuration,
		m.AgentGauge,
		m.StorageGauge,
	}
	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// RecordBackup increments the backup counter for the given status.
func (m *PrometheusMetrics) RecordBackup(status string) {
	m.BackupCounter.WithLabelValues(status).Inc()
}

// RecordBackupDuration observes a backup duration for the given schedule.
func (m *PrometheusMetrics) RecordBackupDuration(schedule string, durationSeconds float64) {
	m.BackupDuration.WithLabelValues(schedule).Observe(durationSeconds)
}

// SetAgentCount sets the current agent count for a given status.
func (m *PrometheusMetrics) SetAgentCount(status string, count float64) {
	m.AgentGauge.WithLabelValues(status).Set(count)
}

// SetStorageBytes sets the current storage bytes for a given type.
func (m *PrometheusMetrics) SetStorageBytes(storageType string, bytes float64) {
	m.StorageGauge.WithLabelValues(storageType).Set(bytes)
}

// PrometheusStore defines the interface for retrieving metrics data.
type PrometheusStore interface {
	// Agent metrics
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)

	// Backup metrics
	GetAllBackups(ctx context.Context) ([]*models.Backup, error)
	GetBackupsByStatus(ctx context.Context, status models.BackupStatus) ([]*models.Backup, error)

	// Storage metrics
	GetStorageStatsSummaryGlobal(ctx context.Context) (*models.StorageStatsSummary, error)
}

// PrometheusCollector collects and exposes Prometheus metrics.
type PrometheusCollector struct {
	store  PrometheusStore
	logger zerolog.Logger

	// Cached metrics with mutex for thread safety
	mu               sync.RWMutex
	lastCollected    time.Time
	cachedMetrics    *CollectedMetrics
	cacheExpiry      time.Duration
}

// CollectedMetrics holds all collected Prometheus metrics data.
type CollectedMetrics struct {
	// Backup metrics
	BackupTotal           int64              // Total number of backups
	BackupByStatus        map[string]int64   // Backups by status (completed, failed, running, canceled)
	BackupDurationBuckets map[float64]int64  // Histogram buckets for backup duration
	BackupDurationSum     float64            // Sum of all backup durations
	BackupDurationCount   int64              // Count of backups with duration
	BackupSizeBytes       int64              // Total size of completed backups

	// Agent metrics
	AgentsTotal  int64 // Total number of agents
	AgentsOnline int64 // Number of online agents

	// Storage metrics
	StorageUsedBytes int64 // Total storage used
}

// NewPrometheusCollector creates a new PrometheusCollector.
func NewPrometheusCollector(store PrometheusStore, logger zerolog.Logger) *PrometheusCollector {
	return &PrometheusCollector{
		store:       store,
		logger:      logger.With().Str("component", "prometheus_collector").Logger(),
		cacheExpiry: 15 * time.Second,
	}
}

// Collect gathers all metrics from the database.
func (c *PrometheusCollector) Collect(ctx context.Context) (*CollectedMetrics, error) {
	// Check cache first
	c.mu.RLock()
	if c.cachedMetrics != nil && time.Since(c.lastCollected) < c.cacheExpiry {
		metrics := c.cachedMetrics
		c.mu.RUnlock()
		return metrics, nil
	}
	c.mu.RUnlock()

	// Collect fresh metrics
	metrics := &CollectedMetrics{
		BackupByStatus:        make(map[string]int64),
		BackupDurationBuckets: make(map[float64]int64),
	}

	// Collect agent metrics
	if err := c.collectAgentMetrics(ctx, metrics); err != nil {
		c.logger.Warn().Err(err).Msg("failed to collect agent metrics")
	}

	// Collect backup metrics
	if err := c.collectBackupMetrics(ctx, metrics); err != nil {
		c.logger.Warn().Err(err).Msg("failed to collect backup metrics")
	}

	// Collect storage metrics
	if err := c.collectStorageMetrics(ctx, metrics); err != nil {
		c.logger.Warn().Err(err).Msg("failed to collect storage metrics")
	}

	// Update cache
	c.mu.Lock()
	c.cachedMetrics = metrics
	c.lastCollected = time.Now()
	c.mu.Unlock()

	return metrics, nil
}

func (c *PrometheusCollector) collectAgentMetrics(ctx context.Context, metrics *CollectedMetrics) error {
	agents, err := c.store.GetAllAgents(ctx)
	if err != nil {
		return fmt.Errorf("get agents: %w", err)
	}

	metrics.AgentsTotal = int64(len(agents))
	for _, agent := range agents {
		if agent.Status == models.AgentStatusActive {
			metrics.AgentsOnline++
		}
	}

	return nil
}

func (c *PrometheusCollector) collectBackupMetrics(ctx context.Context, metrics *CollectedMetrics) error {
	backups, err := c.store.GetAllBackups(ctx)
	if err != nil {
		return fmt.Errorf("get backups: %w", err)
	}

	metrics.BackupTotal = int64(len(backups))

	// Histogram buckets for backup duration (in seconds)
	buckets := []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800}
	for _, b := range buckets {
		metrics.BackupDurationBuckets[b] = 0
	}

	for _, backup := range backups {
		// Count by status
		status := string(backup.Status)
		metrics.BackupByStatus[status]++

		// Calculate duration for completed backups
		if backup.CompletedAt != nil && !backup.StartedAt.IsZero() {
			duration := backup.CompletedAt.Sub(backup.StartedAt).Seconds()
			metrics.BackupDurationSum += duration
			metrics.BackupDurationCount++

			// Populate histogram buckets
			for _, b := range buckets {
				if duration <= b {
					metrics.BackupDurationBuckets[b]++
				}
			}
		}

		// Sum size for completed backups
		if backup.Status == models.BackupStatusCompleted && backup.SizeBytes != nil {
			metrics.BackupSizeBytes += *backup.SizeBytes
		}
	}

	return nil
}

func (c *PrometheusCollector) collectStorageMetrics(ctx context.Context, metrics *CollectedMetrics) error {
	stats, err := c.store.GetStorageStatsSummaryGlobal(ctx)
	if err != nil {
		return fmt.Errorf("get storage stats: %w", err)
	}

	if stats != nil {
		metrics.StorageUsedBytes = stats.TotalRawSize
	}

	return nil
}

// Format returns the metrics in Prometheus exposition format.
func (c *PrometheusCollector) Format(metrics *CollectedMetrics) string {
	var sb strings.Builder

	// backup_total - Counter for total number of backups
	sb.WriteString("# HELP keldris_backup_total Total number of backups\n")
	sb.WriteString("# TYPE keldris_backup_total counter\n")
	sb.WriteString(fmt.Sprintf("keldris_backup_total %d\n", metrics.BackupTotal))
	sb.WriteString("\n")

	// backup_total by status
	sb.WriteString("# HELP keldris_backup_status_total Total number of backups by status\n")
	sb.WriteString("# TYPE keldris_backup_status_total counter\n")
	for status, count := range metrics.BackupByStatus {
		sb.WriteString(fmt.Sprintf("keldris_backup_status_total{status=\"%s\"} %d\n", status, count))
	}
	sb.WriteString("\n")

	// backup_duration_seconds - Histogram for backup duration
	sb.WriteString("# HELP keldris_backup_duration_seconds Histogram of backup duration in seconds\n")
	sb.WriteString("# TYPE keldris_backup_duration_seconds histogram\n")

	buckets := []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800}
	for _, b := range buckets {
		count := metrics.BackupDurationBuckets[b]
		sb.WriteString(fmt.Sprintf("keldris_backup_duration_seconds_bucket{le=\"%.0f\"} %d\n", b, count))
	}
	sb.WriteString(fmt.Sprintf("keldris_backup_duration_seconds_bucket{le=\"+Inf\"} %d\n", metrics.BackupDurationCount))
	sb.WriteString(fmt.Sprintf("keldris_backup_duration_seconds_sum %.2f\n", metrics.BackupDurationSum))
	sb.WriteString(fmt.Sprintf("keldris_backup_duration_seconds_count %d\n", metrics.BackupDurationCount))
	sb.WriteString("\n")

	// backup_size_bytes - Gauge for total backup size
	sb.WriteString("# HELP keldris_backup_size_bytes Total size of completed backups in bytes\n")
	sb.WriteString("# TYPE keldris_backup_size_bytes gauge\n")
	sb.WriteString(fmt.Sprintf("keldris_backup_size_bytes %d\n", metrics.BackupSizeBytes))
	sb.WriteString("\n")

	// agents_total - Gauge for total number of agents
	sb.WriteString("# HELP keldris_agents_total Total number of registered agents\n")
	sb.WriteString("# TYPE keldris_agents_total gauge\n")
	sb.WriteString(fmt.Sprintf("keldris_agents_total %d\n", metrics.AgentsTotal))
	sb.WriteString("\n")

	// agents_online - Gauge for number of online agents
	sb.WriteString("# HELP keldris_agents_online Number of online agents\n")
	sb.WriteString("# TYPE keldris_agents_online gauge\n")
	sb.WriteString(fmt.Sprintf("keldris_agents_online %d\n", metrics.AgentsOnline))
	sb.WriteString("\n")

	// storage_used_bytes - Gauge for total storage used
	sb.WriteString("# HELP keldris_storage_used_bytes Total storage used in bytes\n")
	sb.WriteString("# TYPE keldris_storage_used_bytes gauge\n")
	sb.WriteString(fmt.Sprintf("keldris_storage_used_bytes %d\n", metrics.StorageUsedBytes))

	return sb.String()
}
