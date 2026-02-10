package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
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
