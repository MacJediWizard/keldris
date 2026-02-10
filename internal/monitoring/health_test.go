package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// mockMonitorStore implements Store for testing.
type mockMonitorStore struct {
	agents          []*models.Agent
	schedules       []*models.Schedule
	backupBySchedule map[uuid.UUID]*models.Backup
	orgByAgent      map[uuid.UUID]uuid.UUID
	orgBySchedule   map[uuid.UUID]uuid.UUID
	updatedAgents   []*models.Agent

	agentErr        error
	scheduleErr     error
	backupErr       error
	orgByAgentErr   error
	orgByScheduleErr error
	updateAgentErr  error
}

func newMockMonitorStore() *mockMonitorStore {
	return &mockMonitorStore{
		backupBySchedule: make(map[uuid.UUID]*models.Backup),
		orgByAgent:       make(map[uuid.UUID]uuid.UUID),
		orgBySchedule:    make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockMonitorStore) GetAllAgents(_ context.Context) ([]*models.Agent, error) {
	return m.agents, m.agentErr
}

func (m *mockMonitorStore) GetAllSchedules(_ context.Context) ([]*models.Schedule, error) {
	return m.schedules, m.scheduleErr
}

func (m *mockMonitorStore) GetLatestBackupByScheduleID(_ context.Context, scheduleID uuid.UUID) (*models.Backup, error) {
	if m.backupErr != nil {
		return nil, m.backupErr
	}
	if b, ok := m.backupBySchedule[scheduleID]; ok {
		return b, nil
	}
	return nil, pgx.ErrNoRows
}

func (m *mockMonitorStore) GetOrgIDByAgentID(_ context.Context, agentID uuid.UUID) (uuid.UUID, error) {
	if m.orgByAgentErr != nil {
		return uuid.Nil, m.orgByAgentErr
	}
	if id, ok := m.orgByAgent[agentID]; ok {
		return id, nil
	}
	return uuid.Nil, pgx.ErrNoRows
}

func (m *mockMonitorStore) GetOrgIDByScheduleID(_ context.Context, scheduleID uuid.UUID) (uuid.UUID, error) {
	if m.orgByScheduleErr != nil {
		return uuid.Nil, m.orgByScheduleErr
	}
	if id, ok := m.orgBySchedule[scheduleID]; ok {
		return id, nil
	}
	return uuid.Nil, pgx.ErrNoRows
}

func (m *mockMonitorStore) UpdateAgent(_ context.Context, agent *models.Agent) error {
	if m.updateAgentErr != nil {
		return m.updateAgentErr
	}
	m.updatedAgents = append(m.updatedAgents, agent)
	return nil
}

// mockAlertSvc implements AlertService for testing.
type mockAlertSvc struct {
	created   []*models.Alert
	resolved  []resolveCall
	hasActive bool

	createErr  error
	resolveErr error
	hasErr     error
}

type resolveCall struct {
	resourceType models.ResourceType
	resourceID   uuid.UUID
}

func (m *mockAlertSvc) CreateAlert(_ context.Context, alert *models.Alert) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, alert)
	return nil
}

func (m *mockAlertSvc) ResolveAlertsByResource(_ context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error {
	if m.resolveErr != nil {
		return m.resolveErr
	}
	m.resolved = append(m.resolved, resolveCall{resourceType, resourceID})
	return nil
}

func (m *mockAlertSvc) HasActiveAlert(_ context.Context, _ uuid.UUID, _ models.ResourceType, _ uuid.UUID, _ models.AlertType) (bool, error) {
	return m.hasActive, m.hasErr
}

func TestHealthChecker_ServerHealth(t *testing.T) {
	t.Run("creates monitor with default config", func(t *testing.T) {
		cfg := DefaultConfig()
		if cfg.AgentOfflineThreshold != 5*time.Minute {
			t.Errorf("expected 5m threshold, got %v", cfg.AgentOfflineThreshold)
		}
		if cfg.BackupSLAMaxHours != 24 {
			t.Errorf("expected 24h SLA, got %d", cfg.BackupSLAMaxHours)
		}
		if cfg.CheckInterval != 1*time.Minute {
			t.Errorf("expected 1m interval, got %v", cfg.CheckInterval)
		}
	})

	t.Run("monitor starts and stops", func(t *testing.T) {
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         100 * time.Millisecond,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx, cancel := context.WithCancel(context.Background())
		mon.Start(ctx)

		// Let it run a few checks
		time.Sleep(250 * time.Millisecond)

		cancel()
		mon.Stop()
	})

	t.Run("monitor stops via stopCh", func(t *testing.T) {
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)

		time.Sleep(50 * time.Millisecond)
		mon.Stop()
	})
}

func TestHealthChecker_AgentHealth(t *testing.T) {
	orgID := uuid.New()

	t.Run("detects agent going offline", func(t *testing.T) {
		agentID := uuid.New()
		lastSeen := time.Now().Add(-10 * time.Minute) // Well past threshold
		store := newMockMonitorStore()
		store.agents = []*models.Agent{
			{
				ID:       agentID,
				OrgID:    orgID,
				Hostname: "server-01",
				Status:   models.AgentStatusActive,
				LastSeen: &lastSeen,
			},
		}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		// Check agent was updated to offline
		if len(store.updatedAgents) == 0 {
			t.Fatal("expected agent to be updated")
		}
		if store.updatedAgents[0].Status != models.AgentStatusOffline {
			t.Errorf("expected offline status, got %s", store.updatedAgents[0].Status)
		}

		// Check alert was created
		if len(alertSvc.created) == 0 {
			t.Fatal("expected alert to be created")
		}
		if alertSvc.created[0].Type != models.AlertTypeAgentOffline {
			t.Errorf("expected agent_offline alert type, got %s", alertSvc.created[0].Type)
		}
	})

	t.Run("detects agent coming back online", func(t *testing.T) {
		agentID := uuid.New()
		lastSeen := time.Now() // Just seen
		store := newMockMonitorStore()
		store.agents = []*models.Agent{
			{
				ID:       agentID,
				OrgID:    orgID,
				Hostname: "server-02",
				Status:   models.AgentStatusOffline, // Was offline
				LastSeen: &lastSeen,
			},
		}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		// Check agent was updated to active
		if len(store.updatedAgents) == 0 {
			t.Fatal("expected agent to be updated")
		}
		if store.updatedAgents[0].Status != models.AgentStatusActive {
			t.Errorf("expected active status, got %s", store.updatedAgents[0].Status)
		}

		// Check alerts were resolved
		if len(alertSvc.resolved) == 0 {
			t.Fatal("expected alerts to be resolved")
		}
		if alertSvc.resolved[0].resourceType != models.ResourceTypeAgent {
			t.Errorf("expected agent resource type, got %s", alertSvc.resolved[0].resourceType)
		}
	})

	t.Run("no action for online active agent", func(t *testing.T) {
		agentID := uuid.New()
		lastSeen := time.Now()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{
			{
				ID:       agentID,
				OrgID:    orgID,
				Hostname: "server-03",
				Status:   models.AgentStatusActive,
				LastSeen: &lastSeen,
			},
		}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(store.updatedAgents) != 0 {
			t.Error("expected no agent updates for healthy agent")
		}
		if len(alertSvc.created) != 0 {
			t.Error("expected no alerts for healthy agent")
		}
	})

	t.Run("agent with nil LastSeen treated as offline", func(t *testing.T) {
		agentID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{
			{
				ID:       agentID,
				OrgID:    orgID,
				Hostname: "server-nil-lastseen",
				Status:   models.AgentStatusActive,
				LastSeen: nil,
			},
		}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(store.updatedAgents) == 0 {
			t.Fatal("expected agent to be marked offline")
		}
		if store.updatedAgents[0].Status != models.AgentStatusOffline {
			t.Errorf("expected offline, got %s", store.updatedAgents[0].Status)
		}
	})

	t.Run("skips alert if already active", func(t *testing.T) {
		agentID := uuid.New()
		lastSeen := time.Now().Add(-10 * time.Minute)
		store := newMockMonitorStore()
		store.agents = []*models.Agent{
			{
				ID:       agentID,
				OrgID:    orgID,
				Hostname: "server-existing-alert",
				Status:   models.AgentStatusActive,
				LastSeen: &lastSeen,
			},
		}
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{hasActive: true}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		// Agent should be updated but no new alert
		if len(store.updatedAgents) == 0 {
			t.Fatal("expected agent status update")
		}
		if len(alertSvc.created) != 0 {
			t.Error("expected no new alert when one is already active")
		}
	})

	t.Run("handles GetAllAgents error", func(t *testing.T) {
		store := newMockMonitorStore()
		store.agentErr = pgx.ErrNoRows
		store.schedules = []*models.Schedule{}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()
		// Should not panic, just log error
	})
}

func TestHealthChecker_DatabaseHealth(t *testing.T) {
	orgID := uuid.New()

	t.Run("checks backup SLA violation", func(t *testing.T) {
		scheduleID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{
			{
				ID:        scheduleID,
				Name:      "daily-backup",
				Enabled:   true,
				CreatedAt: time.Now().Add(-48 * time.Hour),
			},
		}
		store.orgBySchedule[scheduleID] = orgID
		// Last backup was 30 hours ago, SLA is 24 hours
		startedAt := time.Now().Add(-30 * time.Hour)
		store.backupBySchedule[scheduleID] = &models.Backup{
			ID:        uuid.New(),
			Status:    models.BackupStatusCompleted,
			StartedAt: startedAt,
		}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(alertSvc.created) == 0 {
			t.Fatal("expected SLA alert to be created")
		}
		if alertSvc.created[0].Type != models.AlertTypeBackupSLA {
			t.Errorf("expected backup_sla type, got %s", alertSvc.created[0].Type)
		}
	})

	t.Run("resolves SLA alert when backup is within SLA", func(t *testing.T) {
		scheduleID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{
			{
				ID:        scheduleID,
				Name:      "hourly-backup",
				Enabled:   true,
				CreatedAt: time.Now().Add(-72 * time.Hour),
			},
		}
		store.orgBySchedule[scheduleID] = orgID
		// Last backup was 2 hours ago, within 24h SLA
		startedAt := time.Now().Add(-2 * time.Hour)
		store.backupBySchedule[scheduleID] = &models.Backup{
			ID:        uuid.New(),
			Status:    models.BackupStatusCompleted,
			StartedAt: startedAt,
		}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(alertSvc.created) != 0 {
			t.Error("expected no new alerts for within-SLA backup")
		}
		if len(alertSvc.resolved) == 0 {
			t.Fatal("expected SLA alerts to be resolved")
		}
	})

	t.Run("alerts for schedule with no backups after SLA period", func(t *testing.T) {
		scheduleID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{
			{
				ID:        scheduleID,
				Name:      "never-ran",
				Enabled:   true,
				CreatedAt: time.Now().Add(-48 * time.Hour), // Created 48h ago
			},
		}
		store.orgBySchedule[scheduleID] = orgID
		// No backup exists (will return pgx.ErrNoRows)
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(alertSvc.created) == 0 {
			t.Fatal("expected SLA alert for never-ran schedule")
		}
	})

	t.Run("no alert for new schedule without backups", func(t *testing.T) {
		scheduleID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{
			{
				ID:        scheduleID,
				Name:      "just-created",
				Enabled:   true,
				CreatedAt: time.Now().Add(-1 * time.Hour), // Created 1h ago
			},
		}
		store.orgBySchedule[scheduleID] = orgID
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(alertSvc.created) != 0 {
			t.Error("expected no alerts for recently created schedule")
		}
	})

	t.Run("alerts for failed backup exceeding SLA", func(t *testing.T) {
		scheduleID := uuid.New()
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.schedules = []*models.Schedule{
			{
				ID:        scheduleID,
				Name:      "failed-backup-sched",
				Enabled:   true,
				CreatedAt: time.Now().Add(-72 * time.Hour),
			},
		}
		store.orgBySchedule[scheduleID] = orgID
		startedAt := time.Now().Add(-30 * time.Hour)
		store.backupBySchedule[scheduleID] = &models.Backup{
			ID:        uuid.New(),
			Status:    models.BackupStatusFailed,
			StartedAt: startedAt,
		}
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()

		if len(alertSvc.created) == 0 {
			t.Fatal("expected SLA alert for failed backup exceeding SLA")
		}
	})

	t.Run("handles GetAllSchedules error", func(t *testing.T) {
		store := newMockMonitorStore()
		store.agents = []*models.Agent{}
		store.scheduleErr = pgx.ErrNoRows
		alertSvc := &mockAlertSvc{}

		cfg := Config{
			AgentOfflineThreshold: 5 * time.Minute,
			BackupSLAMaxHours:     24,
			CheckInterval:         10 * time.Second,
		}
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		ctx := context.Background()
		mon.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		mon.Stop()
		// Should not panic
	})
}

func TestCheckStorageUsage(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	t.Run("resolves alert when usage below threshold", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsedBytes:      50 * 1024 * 1024,
			TotalBytes:     100 * 1024 * 1024,
			UsagePercent:   50.0,
		}, 80)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alertSvc.resolved) != 1 {
			t.Errorf("expected 1 resolve call, got %d", len(alertSvc.resolved))
		}
		if len(alertSvc.created) != 0 {
			t.Error("expected no new alerts when below threshold")
		}
	})

	t.Run("creates warning alert at 85%", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsedBytes:      85 * 1024 * 1024,
			TotalBytes:     100 * 1024 * 1024,
			UsagePercent:   85.0,
		}, 80)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alertSvc.created) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alertSvc.created))
		}
		if alertSvc.created[0].Severity != models.AlertSeverityWarning {
			t.Errorf("expected warning severity, got %s", alertSvc.created[0].Severity)
		}
	})

	t.Run("creates critical alert at 95%", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsedBytes:      95 * 1024 * 1024,
			TotalBytes:     100 * 1024 * 1024,
			UsagePercent:   95.0,
		}, 80)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alertSvc.created) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alertSvc.created))
		}
		if alertSvc.created[0].Severity != models.AlertSeverityCritical {
			t.Errorf("expected critical severity, got %s", alertSvc.created[0].Severity)
		}
	})

	t.Run("creates info alert between threshold and 85%", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsedBytes:      82 * 1024 * 1024,
			TotalBytes:     100 * 1024 * 1024,
			UsagePercent:   82.0,
		}, 80)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alertSvc.created) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alertSvc.created))
		}
		if alertSvc.created[0].Severity != models.AlertSeverityInfo {
			t.Errorf("expected info severity, got %s", alertSvc.created[0].Severity)
		}
	})

	t.Run("skips alert if already active", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{hasActive: true}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsedBytes:      95 * 1024 * 1024,
			TotalBytes:     100 * 1024 * 1024,
			UsagePercent:   95.0,
		}, 80)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alertSvc.created) != 0 {
			t.Error("expected no new alert when one is already active")
		}
	})

	t.Run("returns error on resolve failure", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{resolveErr: errors.New("resolve error")}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsagePercent:   50.0,
		}, 80)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns error on create alert failure", func(t *testing.T) {
		store := newMockMonitorStore()
		alertSvc := &mockAlertSvc{createErr: errors.New("create error")}

		cfg := DefaultConfig()
		mon := NewMonitor(store, alertSvc, cfg, zerolog.Nop())

		err := mon.CheckStorageUsage(context.Background(), orgID, StorageUsageResult{
			RepositoryID:   repoID,
			RepositoryName: "test-repo",
			UsagePercent:   90.0,
		}, 80)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
