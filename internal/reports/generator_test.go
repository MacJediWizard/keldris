package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockReportStore implements ReportStore for testing.
type mockReportStore struct {
	backups      []*models.Backup
	schedules    []*models.Schedule
	storageStats *models.StorageStatsSummary
	agents       []*models.Agent
	alerts       []*models.Alert

	backupsErr      error
	schedulesErr    error
	storageStatsErr error
	agentsErr       error
	alertsErr       error
}

func (m *mockReportStore) GetBackupsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Backup, error) {
	return m.backups, m.backupsErr
}

func (m *mockReportStore) GetEnabledSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.schedulesErr
}

func (m *mockReportStore) GetStorageStatsSummary(_ context.Context, _ uuid.UUID) (*models.StorageStatsSummary, error) {
	return m.storageStats, m.storageStatsErr
}

func (m *mockReportStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.agentsErr
}

func (m *mockReportStore) GetAlertsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Alert, error) {
	return m.alerts, m.alertsErr
}

func newTestGenerator(store ReportStore) *Generator {
	logger := zerolog.Nop()
	return NewGenerator(store, logger)
}

func int64Ptr(v int64) *int64 { return &v }

func TestGenerator_GenerateReport(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	size1 := int64(1000)
	size2 := int64(2000)

	store := &mockReportStore{
		backups: []*models.Backup{
			{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: &size1},
			{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: &size2},
			{ID: uuid.New(), Status: models.BackupStatusFailed},
		},
		schedules: []*models.Schedule{
			{ID: uuid.New(), Enabled: true},
			{ID: uuid.New(), Enabled: true},
		},
		storageStats: &models.StorageStatsSummary{
			TotalRawSize:     5000,
			TotalRestoreSize: 10000,
			TotalSpaceSaved:  5000,
			RepositoryCount:  2,
			TotalSnapshots:   10,
		},
		agents: []*models.Agent{
			{ID: uuid.New(), Status: models.AgentStatusActive},
			{ID: uuid.New(), Status: models.AgentStatusOffline},
		},
		alerts: []*models.Alert{
			{ID: uuid.New(), Severity: models.AlertSeverityCritical, Status: models.AlertStatusActive, Type: models.AlertTypeAgentOffline, Title: "Agent down", Message: "Agent is offline", CreatedAt: now},
			{ID: uuid.New(), Severity: models.AlertSeverityWarning, Status: models.AlertStatusAcknowledged, Type: models.AlertTypeStorageUsage, Title: "Storage high", Message: "Above 80%", CreatedAt: now},
		},
	}

	gen := newTestGenerator(store)
	report, err := gen.GenerateReport(context.Background(), orgID, start, now)
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}

	// Backup summary
	if report.BackupSummary.TotalBackups != 3 {
		t.Errorf("TotalBackups = %d, want 3", report.BackupSummary.TotalBackups)
	}
	if report.BackupSummary.SuccessfulBackups != 2 {
		t.Errorf("SuccessfulBackups = %d, want 2", report.BackupSummary.SuccessfulBackups)
	}
	if report.BackupSummary.FailedBackups != 1 {
		t.Errorf("FailedBackups = %d, want 1", report.BackupSummary.FailedBackups)
	}
	if report.BackupSummary.TotalDataBacked != 3000 {
		t.Errorf("TotalDataBacked = %d, want 3000", report.BackupSummary.TotalDataBacked)
	}
	if report.BackupSummary.SchedulesActive != 2 {
		t.Errorf("SchedulesActive = %d, want 2", report.BackupSummary.SchedulesActive)
	}

	// Storage summary
	if report.StorageSummary.TotalRawSize != 5000 {
		t.Errorf("TotalRawSize = %d, want 5000", report.StorageSummary.TotalRawSize)
	}
	if report.StorageSummary.SpaceSavedPct != 50.0 {
		t.Errorf("SpaceSavedPct = %f, want 50.0", report.StorageSummary.SpaceSavedPct)
	}

	// Agent summary
	if report.AgentSummary.TotalAgents != 2 {
		t.Errorf("TotalAgents = %d, want 2", report.AgentSummary.TotalAgents)
	}
	if report.AgentSummary.ActiveAgents != 1 {
		t.Errorf("ActiveAgents = %d, want 1", report.AgentSummary.ActiveAgents)
	}

	// Alert summary
	if report.AlertSummary.TotalAlerts != 2 {
		t.Errorf("TotalAlerts = %d, want 2", report.AlertSummary.TotalAlerts)
	}
	if report.AlertSummary.CriticalAlerts != 1 {
		t.Errorf("CriticalAlerts = %d, want 1", report.AlertSummary.CriticalAlerts)
	}

	// Top issues
	if len(report.TopIssues) != 1 {
		t.Errorf("TopIssues len = %d, want 1", len(report.TopIssues))
	}
}

func TestGenerator_BackupSummary(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	t.Run("all successful", func(t *testing.T) {
		size := int64(500)
		store := &mockReportStore{
			backups: []*models.Backup{
				{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: &size},
				{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: &size},
			},
			schedules:    []*models.Schedule{{ID: uuid.New()}},
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.BackupSummary.SuccessRate != 100.0 {
			t.Errorf("SuccessRate = %f, want 100.0", report.BackupSummary.SuccessRate)
		}
		if report.BackupSummary.TotalDataBacked != 1000 {
			t.Errorf("TotalDataBacked = %d, want 1000", report.BackupSummary.TotalDataBacked)
		}
	})

	t.Run("all failed", func(t *testing.T) {
		store := &mockReportStore{
			backups: []*models.Backup{
				{ID: uuid.New(), Status: models.BackupStatusFailed},
				{ID: uuid.New(), Status: models.BackupStatusFailed},
			},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.BackupSummary.SuccessRate != 0.0 {
			t.Errorf("SuccessRate = %f, want 0.0", report.BackupSummary.SuccessRate)
		}
		if report.BackupSummary.FailedBackups != 2 {
			t.Errorf("FailedBackups = %d, want 2", report.BackupSummary.FailedBackups)
		}
	})

	t.Run("nil size bytes", func(t *testing.T) {
		store := &mockReportStore{
			backups: []*models.Backup{
				{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: nil},
			},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.BackupSummary.TotalDataBacked != 0 {
			t.Errorf("TotalDataBacked = %d, want 0", report.BackupSummary.TotalDataBacked)
		}
	})

	t.Run("mixed statuses", func(t *testing.T) {
		size := int64(100)
		store := &mockReportStore{
			backups: []*models.Backup{
				{ID: uuid.New(), Status: models.BackupStatusCompleted, SizeBytes: &size},
				{ID: uuid.New(), Status: models.BackupStatusFailed},
				{ID: uuid.New(), Status: models.BackupStatusRunning},
			},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.BackupSummary.TotalBackups != 3 {
			t.Errorf("TotalBackups = %d, want 3", report.BackupSummary.TotalBackups)
		}
		if report.BackupSummary.SuccessfulBackups != 1 {
			t.Errorf("SuccessfulBackups = %d, want 1", report.BackupSummary.SuccessfulBackups)
		}
		if report.BackupSummary.FailedBackups != 1 {
			t.Errorf("FailedBackups = %d, want 1", report.BackupSummary.FailedBackups)
		}
		// Running status is not counted as success or failure
		wantRate := float64(1) / float64(3) * 100
		if report.BackupSummary.SuccessRate != wantRate {
			t.Errorf("SuccessRate = %f, want %f", report.BackupSummary.SuccessRate, wantRate)
		}
	})

	t.Run("backups store error", func(t *testing.T) {
		store := &mockReportStore{
			backupsErr:   errors.New("db connection failed"),
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, err := gen.GenerateReport(context.Background(), orgID, start, now)
		if err != nil {
			t.Fatalf("GenerateReport should not return error for partial failure: %v", err)
		}
		// Backup summary should be zero-valued when the store errors
		if report.BackupSummary.TotalBackups != 0 {
			t.Errorf("TotalBackups = %d, want 0 on error", report.BackupSummary.TotalBackups)
		}
	})

	t.Run("schedules store error", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedulesErr: errors.New("schedules unavailable"),
			storageStats: &models.StorageStatsSummary{},
		}
		gen := newTestGenerator(store)
		report, err := gen.GenerateReport(context.Background(), orgID, start, now)
		if err != nil {
			t.Fatalf("GenerateReport should not return error: %v", err)
		}
		// When schedules fail, the whole backup summary is not set
		if report.BackupSummary.TotalBackups != 0 {
			t.Errorf("TotalBackups = %d, want 0", report.BackupSummary.TotalBackups)
		}
	})
}

func TestGenerator_StorageSummary(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	t.Run("with deduplication savings", func(t *testing.T) {
		store := &mockReportStore{
			backups:   []*models.Backup{},
			schedules: []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{
				TotalRawSize:     8000,
				TotalRestoreSize: 20000,
				TotalSpaceSaved:  12000,
				RepositoryCount:  3,
				TotalSnapshots:   25,
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.StorageSummary.TotalRawSize != 8000 {
			t.Errorf("TotalRawSize = %d, want 8000", report.StorageSummary.TotalRawSize)
		}
		if report.StorageSummary.TotalRestoreSize != 20000 {
			t.Errorf("TotalRestoreSize = %d, want 20000", report.StorageSummary.TotalRestoreSize)
		}
		if report.StorageSummary.SpaceSaved != 12000 {
			t.Errorf("SpaceSaved = %d, want 12000", report.StorageSummary.SpaceSaved)
		}
		wantPct := float64(12000) / float64(20000) * 100
		if report.StorageSummary.SpaceSavedPct != wantPct {
			t.Errorf("SpaceSavedPct = %f, want %f", report.StorageSummary.SpaceSavedPct, wantPct)
		}
		if report.StorageSummary.RepositoryCount != 3 {
			t.Errorf("RepositoryCount = %d, want 3", report.StorageSummary.RepositoryCount)
		}
		if report.StorageSummary.TotalSnapshots != 25 {
			t.Errorf("TotalSnapshots = %d, want 25", report.StorageSummary.TotalSnapshots)
		}
	})

	t.Run("zero restore size", func(t *testing.T) {
		store := &mockReportStore{
			backups:   []*models.Backup{},
			schedules: []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{
				TotalRawSize:     0,
				TotalRestoreSize: 0,
				TotalSpaceSaved:  0,
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.StorageSummary.SpaceSavedPct != 0.0 {
			t.Errorf("SpaceSavedPct = %f, want 0.0 for zero restore size", report.StorageSummary.SpaceSavedPct)
		}
	})

	t.Run("storage stats error", func(t *testing.T) {
		store := &mockReportStore{
			backups:         []*models.Backup{},
			schedules:       []*models.Schedule{},
			storageStatsErr: errors.New("storage unavailable"),
		}
		gen := newTestGenerator(store)
		report, err := gen.GenerateReport(context.Background(), orgID, start, now)
		if err != nil {
			t.Fatalf("GenerateReport should not return error: %v", err)
		}
		if report.StorageSummary.TotalRawSize != 0 {
			t.Errorf("TotalRawSize = %d, want 0 on error", report.StorageSummary.TotalRawSize)
		}
	})
}

func TestGenerator_AgentSummary(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	t.Run("mixed agent statuses", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			agents: []*models.Agent{
				{ID: uuid.New(), Status: models.AgentStatusActive},
				{ID: uuid.New(), Status: models.AgentStatusActive},
				{ID: uuid.New(), Status: models.AgentStatusOffline},
				{ID: uuid.New(), Status: models.AgentStatusPending},
				{ID: uuid.New(), Status: models.AgentStatusPending},
				{ID: uuid.New(), Status: models.AgentStatusPending},
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.AgentSummary.TotalAgents != 6 {
			t.Errorf("TotalAgents = %d, want 6", report.AgentSummary.TotalAgents)
		}
		if report.AgentSummary.ActiveAgents != 2 {
			t.Errorf("ActiveAgents = %d, want 2", report.AgentSummary.ActiveAgents)
		}
		if report.AgentSummary.OfflineAgents != 1 {
			t.Errorf("OfflineAgents = %d, want 1", report.AgentSummary.OfflineAgents)
		}
		if report.AgentSummary.PendingAgents != 3 {
			t.Errorf("PendingAgents = %d, want 3", report.AgentSummary.PendingAgents)
		}
	})

	t.Run("all active", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			agents: []*models.Agent{
				{ID: uuid.New(), Status: models.AgentStatusActive},
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.AgentSummary.ActiveAgents != 1 {
			t.Errorf("ActiveAgents = %d, want 1", report.AgentSummary.ActiveAgents)
		}
		if report.AgentSummary.OfflineAgents != 0 {
			t.Errorf("OfflineAgents = %d, want 0", report.AgentSummary.OfflineAgents)
		}
	})

	t.Run("agents store error", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			agentsErr:    errors.New("agents unavailable"),
		}
		gen := newTestGenerator(store)
		report, err := gen.GenerateReport(context.Background(), orgID, start, now)
		if err != nil {
			t.Fatalf("GenerateReport should not return error: %v", err)
		}
		if report.AgentSummary.TotalAgents != 0 {
			t.Errorf("TotalAgents = %d, want 0 on error", report.AgentSummary.TotalAgents)
		}
	})
}

func TestGenerator_AlertSummary(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	t.Run("mixed alerts", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			alerts: []*models.Alert{
				{ID: uuid.New(), Severity: models.AlertSeverityCritical, Status: models.AlertStatusActive, Type: models.AlertTypeAgentOffline, Title: "Agent offline", Message: "Host-1 is down", CreatedAt: now},
				{ID: uuid.New(), Severity: models.AlertSeverityCritical, Status: models.AlertStatusAcknowledged, Type: models.AlertTypeBackupSLA, Title: "SLA breach", Message: "Backup overdue", CreatedAt: now},
				{ID: uuid.New(), Severity: models.AlertSeverityWarning, Status: models.AlertStatusResolved, Type: models.AlertTypeStorageUsage, Title: "Storage warning", Message: "Usage at 85%", CreatedAt: now},
				{ID: uuid.New(), Severity: models.AlertSeverityWarning, Status: models.AlertStatusActive, Type: models.AlertTypeStorageUsage, Title: "Storage high", Message: "Usage at 90%", CreatedAt: now},
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if report.AlertSummary.TotalAlerts != 4 {
			t.Errorf("TotalAlerts = %d, want 4", report.AlertSummary.TotalAlerts)
		}
		if report.AlertSummary.CriticalAlerts != 2 {
			t.Errorf("CriticalAlerts = %d, want 2", report.AlertSummary.CriticalAlerts)
		}
		if report.AlertSummary.WarningAlerts != 2 {
			t.Errorf("WarningAlerts = %d, want 2", report.AlertSummary.WarningAlerts)
		}
		if report.AlertSummary.AcknowledgedAlerts != 1 {
			t.Errorf("AcknowledgedAlerts = %d, want 1", report.AlertSummary.AcknowledgedAlerts)
		}
		if report.AlertSummary.ResolvedAlerts != 1 {
			t.Errorf("ResolvedAlerts = %d, want 1", report.AlertSummary.ResolvedAlerts)
		}
	})

	t.Run("top issues limited to 5", func(t *testing.T) {
		var alerts []*models.Alert
		for i := 0; i < 8; i++ {
			alerts = append(alerts, &models.Alert{
				ID:        uuid.New(),
				Severity:  models.AlertSeverityCritical,
				Status:    models.AlertStatusActive,
				Type:      models.AlertTypeAgentOffline,
				Title:     "Critical alert",
				Message:   "Details",
				CreatedAt: now,
			})
		}
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			alerts:       alerts,
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if len(report.TopIssues) != 5 {
			t.Errorf("TopIssues len = %d, want 5 (max)", len(report.TopIssues))
		}
	})

	t.Run("warnings not in top issues", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			alerts: []*models.Alert{
				{ID: uuid.New(), Severity: models.AlertSeverityWarning, Status: models.AlertStatusActive, Type: models.AlertTypeStorageUsage, Title: "Warning only", Message: "Not critical", CreatedAt: now},
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if len(report.TopIssues) != 0 {
			t.Errorf("TopIssues len = %d, want 0 (warnings excluded)", len(report.TopIssues))
		}
	})

	t.Run("top issues field mapping", func(t *testing.T) {
		ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			alerts: []*models.Alert{
				{
					ID:        uuid.New(),
					Severity:  models.AlertSeverityCritical,
					Status:    models.AlertStatusActive,
					Type:      models.AlertTypeBackupSLA,
					Title:     "SLA Breach",
					Message:   "Backup overdue by 2h",
					CreatedAt: ts,
				},
			},
		}
		gen := newTestGenerator(store)
		report, _ := gen.GenerateReport(context.Background(), orgID, start, now)

		if len(report.TopIssues) != 1 {
			t.Fatalf("TopIssues len = %d, want 1", len(report.TopIssues))
		}
		issue := report.TopIssues[0]
		if issue.Type != string(models.AlertTypeBackupSLA) {
			t.Errorf("Type = %q, want %q", issue.Type, models.AlertTypeBackupSLA)
		}
		if issue.Severity != string(models.AlertSeverityCritical) {
			t.Errorf("Severity = %q, want %q", issue.Severity, models.AlertSeverityCritical)
		}
		if issue.Title != "SLA Breach" {
			t.Errorf("Title = %q, want %q", issue.Title, "SLA Breach")
		}
		if issue.Description != "Backup overdue by 2h" {
			t.Errorf("Description = %q, want %q", issue.Description, "Backup overdue by 2h")
		}
		if !issue.OccurredAt.Equal(ts) {
			t.Errorf("OccurredAt = %v, want %v", issue.OccurredAt, ts)
		}
	})

	t.Run("alerts store error", func(t *testing.T) {
		store := &mockReportStore{
			backups:      []*models.Backup{},
			schedules:    []*models.Schedule{},
			storageStats: &models.StorageStatsSummary{},
			alertsErr:    errors.New("alerts unavailable"),
		}
		gen := newTestGenerator(store)
		report, err := gen.GenerateReport(context.Background(), orgID, start, now)
		if err != nil {
			t.Fatalf("GenerateReport should not return error: %v", err)
		}
		if report.AlertSummary.TotalAlerts != 0 {
			t.Errorf("TotalAlerts = %d, want 0 on error", report.AlertSummary.TotalAlerts)
		}
	})
}

func TestGenerator_EmptyData(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	store := &mockReportStore{
		backups:      []*models.Backup{},
		schedules:    []*models.Schedule{},
		storageStats: &models.StorageStatsSummary{},
		agents:       []*models.Agent{},
		alerts:       []*models.Alert{},
	}

	gen := newTestGenerator(store)
	report, err := gen.GenerateReport(context.Background(), orgID, start, now)
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}

	if report.BackupSummary.TotalBackups != 0 {
		t.Errorf("TotalBackups = %d, want 0", report.BackupSummary.TotalBackups)
	}
	if report.BackupSummary.SuccessRate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", report.BackupSummary.SuccessRate)
	}
	if report.BackupSummary.TotalDataBacked != 0 {
		t.Errorf("TotalDataBacked = %d, want 0", report.BackupSummary.TotalDataBacked)
	}
	if report.StorageSummary.TotalRawSize != 0 {
		t.Errorf("TotalRawSize = %d, want 0", report.StorageSummary.TotalRawSize)
	}
	if report.AgentSummary.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", report.AgentSummary.TotalAgents)
	}
	if report.AlertSummary.TotalAlerts != 0 {
		t.Errorf("TotalAlerts = %d, want 0", report.AlertSummary.TotalAlerts)
	}
	if len(report.TopIssues) != 0 {
		t.Errorf("TopIssues len = %d, want 0", len(report.TopIssues))
	}
}

func TestGenerator_AllStoresError(t *testing.T) {
	orgID := uuid.New()
	now := time.Now()
	start := now.Add(-24 * time.Hour)

	store := &mockReportStore{
		backupsErr:      errors.New("backup error"),
		storageStatsErr: errors.New("storage error"),
		agentsErr:       errors.New("agents error"),
		alertsErr:       errors.New("alerts error"),
	}

	gen := newTestGenerator(store)
	report, err := gen.GenerateReport(context.Background(), orgID, start, now)
	if err != nil {
		t.Fatalf("GenerateReport should not return error even when all stores fail: %v", err)
	}

	// Report should still be returned with zero values
	if report == nil {
		t.Fatal("report should not be nil")
	}
	if report.BackupSummary.TotalBackups != 0 {
		t.Errorf("TotalBackups = %d, want 0", report.BackupSummary.TotalBackups)
	}
	if report.AgentSummary.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", report.AgentSummary.TotalAgents)
	}
}

func TestCalculatePeriod(t *testing.T) {
	t.Run("daily", func(t *testing.T) {
		start, end := CalculatePeriod(models.ReportFrequencyDaily, "UTC")
		duration := end.Sub(start)
		// Daily period should be approximately 24 hours minus a nanosecond
		if duration < 23*time.Hour || duration > 25*time.Hour {
			t.Errorf("daily period duration = %v, want ~24h", duration)
		}
	})

	t.Run("weekly", func(t *testing.T) {
		start, end := CalculatePeriod(models.ReportFrequencyWeekly, "UTC")
		duration := end.Sub(start)
		if duration < 6*24*time.Hour || duration > 8*24*time.Hour {
			t.Errorf("weekly period duration = %v, want ~7 days", duration)
		}
	})

	t.Run("monthly", func(t *testing.T) {
		start, end := CalculatePeriod(models.ReportFrequencyMonthly, "UTC")
		duration := end.Sub(start)
		if duration < 27*24*time.Hour || duration > 32*24*time.Hour {
			t.Errorf("monthly period duration = %v, want ~28-31 days", duration)
		}
	})

	t.Run("invalid timezone defaults to UTC", func(t *testing.T) {
		start, end := CalculatePeriod(models.ReportFrequencyDaily, "Invalid/TZ")
		if start.Location() != time.UTC {
			t.Errorf("start location = %v, want UTC", start.Location())
		}
		if end.Location() != time.UTC {
			t.Errorf("end location = %v, want UTC", end.Location())
		}
	})

	t.Run("valid timezone", func(t *testing.T) {
		start, end := CalculatePeriod(models.ReportFrequencyDaily, "America/New_York")
		loc, _ := time.LoadLocation("America/New_York")
		if start.Location().String() != loc.String() {
			t.Errorf("start location = %v, want %v", start.Location(), loc)
		}
		if end.Location().String() != loc.String() {
			t.Errorf("end location = %v, want %v", end.Location(), loc)
		}
	})

	t.Run("default frequency", func(t *testing.T) {
		start, end := CalculatePeriod("unknown", "UTC")
		duration := end.Sub(start)
		// Default falls back to 7 days
		if duration < 6*24*time.Hour || duration > 8*24*time.Hour {
			t.Errorf("default period duration = %v, want ~7 days", duration)
		}
	})
}
