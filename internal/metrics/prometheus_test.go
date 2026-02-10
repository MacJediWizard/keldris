package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestPrometheus_BackupCounter(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := NewPrometheusMetrics(reg)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	t.Run("increments completed counter", func(t *testing.T) {
		m.RecordBackup("completed")
		m.RecordBackup("completed")
		m.RecordBackup("completed")

		val := getCounterValue(t, m.BackupCounter, "completed")
		if val != 3 {
			t.Errorf("expected 3, got %f", val)
		}
	})

	t.Run("increments failed counter independently", func(t *testing.T) {
		m.RecordBackup("failed")

		val := getCounterValue(t, m.BackupCounter, "failed")
		if val != 1 {
			t.Errorf("expected 1, got %f", val)
		}
	})

	t.Run("tracks multiple statuses separately", func(t *testing.T) {
		completedVal := getCounterValue(t, m.BackupCounter, "completed")
		failedVal := getCounterValue(t, m.BackupCounter, "failed")
		if completedVal == failedVal {
			t.Errorf("counters should differ: completed=%f, failed=%f", completedVal, failedVal)
		}
	})
}

func TestPrometheus_BackupDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := NewPrometheusMetrics(reg)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	t.Run("observes backup duration", func(t *testing.T) {
		m.RecordBackupDuration("daily-backup", 120.5)
		m.RecordBackupDuration("daily-backup", 60.0)

		count, sum := getHistogramValues(t, m.BackupDuration, "daily-backup")
		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
		if sum != 180.5 {
			t.Errorf("expected sum 180.5, got %f", sum)
		}
	})

	t.Run("tracks different schedules separately", func(t *testing.T) {
		m.RecordBackupDuration("weekly-backup", 300.0)

		count, _ := getHistogramValues(t, m.BackupDuration, "weekly-backup")
		if count != 1 {
			t.Errorf("expected count 1 for weekly, got %d", count)
		}
	})
}

func TestPrometheus_AgentGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := NewPrometheusMetrics(reg)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	t.Run("sets active agent count", func(t *testing.T) {
		m.SetAgentCount("active", 5)

		val := getGaugeValue(t, m.AgentGauge, "active")
		if val != 5 {
			t.Errorf("expected 5, got %f", val)
		}
	})

	t.Run("sets offline agent count", func(t *testing.T) {
		m.SetAgentCount("offline", 2)

		val := getGaugeValue(t, m.AgentGauge, "offline")
		if val != 2 {
			t.Errorf("expected 2, got %f", val)
		}
	})

	t.Run("updates gauge value", func(t *testing.T) {
		m.SetAgentCount("active", 10)

		val := getGaugeValue(t, m.AgentGauge, "active")
		if val != 10 {
			t.Errorf("expected 10 after update, got %f", val)
		}
	})

	t.Run("supports zero value", func(t *testing.T) {
		m.SetAgentCount("pending", 0)

		val := getGaugeValue(t, m.AgentGauge, "pending")
		if val != 0 {
			t.Errorf("expected 0, got %f", val)
		}
	})
}

func TestPrometheus_StorageGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	m, err := NewPrometheusMetrics(reg)
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	t.Run("sets raw storage bytes", func(t *testing.T) {
		m.SetStorageBytes("raw", 1024*1024*100)

		val := getGaugeValue(t, m.StorageGauge, "raw")
		if val != 1024*1024*100 {
			t.Errorf("expected %f, got %f", float64(1024*1024*100), val)
		}
	})

	t.Run("sets backup storage bytes", func(t *testing.T) {
		m.SetStorageBytes("backup", 1024*1024*50)

		val := getGaugeValue(t, m.StorageGauge, "backup")
		if val != 1024*1024*50 {
			t.Errorf("expected %f, got %f", float64(1024*1024*50), val)
		}
	})

	t.Run("updates storage value", func(t *testing.T) {
		m.SetStorageBytes("raw", 1024*1024*200)

		val := getGaugeValue(t, m.StorageGauge, "raw")
		if val != 1024*1024*200 {
			t.Errorf("expected %f after update, got %f", float64(1024*1024*200), val)
		}
	})
}

func TestPrometheus_Registration(t *testing.T) {
	t.Run("creates metrics successfully", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		m, err := NewPrometheusMetrics(reg)
		if err != nil {
			t.Fatalf("failed to create metrics: %v", err)
		}
		if m == nil {
			t.Fatal("expected non-nil metrics")
		}
		if m.BackupCounter == nil {
			t.Error("BackupCounter should not be nil")
		}
		if m.BackupDuration == nil {
			t.Error("BackupDuration should not be nil")
		}
		if m.AgentGauge == nil {
			t.Error("AgentGauge should not be nil")
		}
		if m.StorageGauge == nil {
			t.Error("StorageGauge should not be nil")
		}
	})

	t.Run("fails on duplicate registration", func(t *testing.T) {
		reg := prometheus.NewRegistry()
		_, err := NewPrometheusMetrics(reg)
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}
		_, err = NewPrometheusMetrics(reg)
		if err == nil {
			t.Fatal("expected error on duplicate registration")
		}
	})
}

// Helper functions for extracting Prometheus metric values.

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, label string) float64 {
	t.Helper()
	var m dto.Metric
	if err := counter.WithLabelValues(label).(prometheus.Metric).Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

func getGaugeValue(t *testing.T, gauge *prometheus.GaugeVec, label string) float64 {
	t.Helper()
	var m dto.Metric
	if err := gauge.WithLabelValues(label).(prometheus.Metric).Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetGauge().GetValue()
}

func getHistogramValues(t *testing.T, hist *prometheus.HistogramVec, label string) (uint64, float64) {
	t.Helper()
	observer := hist.WithLabelValues(label)
	var m dto.Metric
	if err := observer.(prometheus.Metric).Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum()
}
