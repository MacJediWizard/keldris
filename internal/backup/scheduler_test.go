package backup

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockNotificationStore implements notifications.NotificationStore for testing.
type mockNotificationStore struct{}

func (m *mockNotificationStore) GetEnabledPreferencesForEvent(_ context.Context, _ uuid.UUID, _ models.NotificationEventType) ([]*models.NotificationPreference, error) {
	return nil, nil
}

func (m *mockNotificationStore) GetNotificationChannelByID(_ context.Context, _ uuid.UUID) (*models.NotificationChannel, error) {
	return nil, errors.New("not found")
}

func (m *mockNotificationStore) CreateNotificationLog(_ context.Context, _ *models.NotificationLog) error {
	return nil
}

func (m *mockNotificationStore) UpdateNotificationLog(_ context.Context, _ *models.NotificationLog) error {
	return nil
}

func (m *mockNotificationStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	return &models.Agent{ID: id, Hostname: "test"}, nil
}

func (m *mockNotificationStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	return &models.Schedule{ID: id, Name: "test"}, nil
}

var _ notifications.NotificationStore = (*mockNotificationStore)(nil)

// mockStore implements ScheduleStore for testing.
type mockStore struct {
	mu                sync.Mutex
	schedules         []models.Schedule
	repos             map[uuid.UUID]*models.Repository
	backups           []*models.Backup
	agents            map[uuid.UUID]*models.Agent
	scripts           map[uuid.UUID][]*models.BackupScript
	getErr            error
	createErr         error
	updateErr         error
	scriptErr         error
	replicationStatus *models.ReplicationStatus
}

func newMockStore() *mockStore {
	return &mockStore{
		repos:   make(map[uuid.UUID]*models.Repository),
		backups: make([]*models.Backup, 0),
		agents:  make(map[uuid.UUID]*models.Agent),
		scripts: make(map[uuid.UUID][]*models.BackupScript),
	}
}

func (m *mockStore) GetEnabledSchedules(ctx context.Context) ([]models.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	result := make([]models.Schedule, len(m.schedules))
	copy(result, m.schedules)
	return result, nil
}

func (m *mockStore) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	repo, ok := m.repos[id]
	if !ok {
		return nil, context.DeadlineExceeded // Simulate not found
	}
	return repo, nil
}

func (m *mockStore) CreateBackup(ctx context.Context, backup *models.Backup) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.backups = append(m.backups, backup)
	return nil
}

func (m *mockStore) UpdateBackup(ctx context.Context, backup *models.Backup) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateErr
}

func (m *mockStore) GetOrCreateReplicationStatus(ctx context.Context, scheduleID, sourceRepoID, targetRepoID uuid.UUID) (*models.ReplicationStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.replicationStatus != nil {
		return m.replicationStatus, nil
	}
	return models.NewReplicationStatus(scheduleID, sourceRepoID, targetRepoID), nil
}

func (m *mockStore) UpdateReplicationStatus(ctx context.Context, rs *models.ReplicationStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *mockStore) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if agent, ok := m.agents[id]; ok {
		return agent, nil
	}
	return &models.Agent{
		ID:       id,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
	}, nil
}

func (m *mockStore) GetEnabledBackupScriptsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupScript, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scriptErr != nil {
		return nil, m.scriptErr
	}
	if scripts, ok := m.scripts[scheduleID]; ok {
		return scripts, nil
	}
	return nil, nil
}

func (m *mockStore) addSchedule(s models.Schedule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules = append(m.schedules, s)
}

func (m *mockStore) getBackups() []*models.Backup {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.Backup, len(m.backups))
	copy(result, m.backups)
	return result
}

func TestScheduler_StartStop(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = 100 * time.Millisecond

	scheduler := NewScheduler(store, restic, config, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Starting again should error
	if err := scheduler.Start(ctx); err == nil {
		t.Error("Start() expected error when already running")
	}

	stopCtx := scheduler.Stop()
	<-stopCtx.Done()

	// Stopping again should not panic
	scheduler.Stop()
}

func TestScheduler_Reload(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour // Prevent auto-refresh during test

	scheduler := NewScheduler(store, restic, config, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the scheduler so cron can compute next run times
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	// Initially no schedules
	if scheduler.GetActiveSchedules() != 0 {
		t.Errorf("GetActiveSchedules() = %d, want 0", scheduler.GetActiveSchedules())
	}

	// Add a schedule
	repoID := uuid.New()
	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		Name:           "Test Schedule",
		CronExpression: "0 */5 * * * *", // Every 5 minutes with seconds
		Paths:          []string{"/home/user"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	store.addSchedule(schedule)

	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if scheduler.GetActiveSchedules() != 1 {
		t.Errorf("GetActiveSchedules() = %d, want 1", scheduler.GetActiveSchedules())
	}

	// Check next run is set (cron scheduler computes this after starting)
	nextRun, ok := scheduler.GetNextRun(schedule.ID)
	if !ok {
		t.Error("GetNextRun() should return true for active schedule")
	}
	if nextRun.IsZero() {
		t.Error("GetNextRun() should return non-zero time")
	}

	// Remove the schedule by clearing the store
	store.mu.Lock()
	store.schedules = nil
	store.mu.Unlock()
	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if scheduler.GetActiveSchedules() != 0 {
		t.Errorf("GetActiveSchedules() = %d, want 0", scheduler.GetActiveSchedules())
	}
}

func TestScheduler_ReloadError(t *testing.T) {
	store := newMockStore()
	store.getErr = errors.New("database unavailable")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	err := scheduler.Reload(context.Background())
	if err == nil {
		t.Fatal("Reload() expected error when store returns error")
	}
	if err.Error() != "get enabled schedules: database unavailable" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScheduler_GetNextRun_NotFound(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	_, ok := scheduler.GetNextRun(uuid.New())
	if ok {
		t.Error("GetNextRun() should return false for non-existent schedule")
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	if config.RefreshInterval != 5*time.Minute {
		t.Errorf("RefreshInterval = %v, want 5m", config.RefreshInterval)
	}
}

func TestScheduler_InvalidCronExpression(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	// Add a schedule with invalid cron expression
	repoID := uuid.New()
	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		Name:           "Invalid Schedule",
		CronExpression: "invalid cron",
		Paths:          []string{"/home/user"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	store.addSchedule(schedule)

	ctx := context.Background()

	// Reload should not error, but schedule should not be added
	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	// Invalid schedule should not be active
	if scheduler.GetActiveSchedules() != 0 {
		t.Errorf("GetActiveSchedules() = %d, want 0 (invalid cron should not be added)", scheduler.GetActiveSchedules())
	}
}

func TestScheduler_TriggerBackup(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour

	scheduler := NewScheduler(store, restic, config, nil, logger)

	ctx := context.Background()

	// TriggerBackup for non-existent schedule should error
	err := scheduler.TriggerBackup(ctx, uuid.New())
	if err == nil {
		t.Fatal("TriggerBackup() expected error for non-existent schedule")
	}
	if err.Error() == "" || !containsStr(err.Error(), "schedule not found") {
		t.Errorf("error should mention 'schedule not found', got: %v", err)
	}

	// Add a schedule and trigger it
	scheduleID := uuid.New()
	schedule := models.Schedule{
		ID:             scheduleID,
		AgentID:        uuid.New(),
		Name:           "Trigger Test",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/home"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: uuid.New(), Priority: 0, Enabled: true},
		},
	}
	store.addSchedule(schedule)

	// Trigger should not error (executes async)
	err = scheduler.TriggerBackup(ctx, scheduleID)
	if err != nil {
		t.Fatalf("TriggerBackup() error = %v", err)
	}

	// Give the goroutine time to start
	time.Sleep(50 * time.Millisecond)
}

func TestScheduler_TriggerBackup_StoreError(t *testing.T) {
	store := newMockStore()
	store.getErr = errors.New("db error")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	err := scheduler.TriggerBackup(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("TriggerBackup() expected error when store fails")
	}
}

func TestScheduler_GetActiveSchedules(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour

	scheduler := NewScheduler(store, restic, config, nil, logger)

	if scheduler.GetActiveSchedules() != 0 {
		t.Errorf("GetActiveSchedules() = %d, want 0", scheduler.GetActiveSchedules())
	}

	// Add multiple schedules
	for i := 0; i < 3; i++ {
		store.addSchedule(models.Schedule{
			ID:             uuid.New(),
			AgentID:        uuid.New(),
			Name:           "Schedule",
			CronExpression: "0 0 * * * *",
			Paths:          []string{"/data"},
			Enabled:        true,
			Repositories: []models.ScheduleRepository{
				{ID: uuid.New(), RepositoryID: uuid.New(), Priority: 0, Enabled: true},
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	if scheduler.GetActiveSchedules() != 3 {
		t.Errorf("GetActiveSchedules() = %d, want 3", scheduler.GetActiveSchedules())
	}
}

func TestScheduler_ReloadUpdatesSchedules(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour

	scheduler := NewScheduler(store, restic, config, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	// Add first schedule
	id1 := uuid.New()
	store.addSchedule(models.Schedule{
		ID:             id1,
		AgentID:        uuid.New(),
		Name:           "Schedule 1",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: uuid.New(), Priority: 0, Enabled: true},
		},
	})

	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if scheduler.GetActiveSchedules() != 1 {
		t.Errorf("GetActiveSchedules() = %d, want 1", scheduler.GetActiveSchedules())
	}

	// Add second schedule, keep first
	id2 := uuid.New()
	store.addSchedule(models.Schedule{
		ID:             id2,
		AgentID:        uuid.New(),
		Name:           "Schedule 2",
		CronExpression: "0 30 * * * *",
		Paths:          []string{"/backup"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: uuid.New(), Priority: 0, Enabled: true},
		},
	})

	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if scheduler.GetActiveSchedules() != 2 {
		t.Errorf("GetActiveSchedules() = %d, want 2", scheduler.GetActiveSchedules())
	}

	// Verify both schedules have next run times
	if _, ok := scheduler.GetNextRun(id1); !ok {
		t.Error("GetNextRun(id1) should return true")
	}
	if _, ok := scheduler.GetNextRun(id2); !ok {
		t.Error("GetNextRun(id2) should return true")
	}
}

func TestScheduler_GetScriptsByType(t *testing.T) {
	scripts := []*models.BackupScript{
		{Type: models.BackupScriptTypePreBackup, Script: "echo pre"},
		{Type: models.BackupScriptTypePostSuccess, Script: "echo success"},
		{Type: models.BackupScriptTypePostFailure, Script: "echo failure"},
		{Type: models.BackupScriptTypePostAlways, Script: "echo always"},
	}

	t.Run("pre backup", func(t *testing.T) {
		s := getScriptsByType(scripts, models.BackupScriptTypePreBackup)
		if s == nil || s.Script != "echo pre" {
			t.Error("expected pre-backup script")
		}
	})

	t.Run("post success", func(t *testing.T) {
		s := getScriptsByType(scripts, models.BackupScriptTypePostSuccess)
		if s == nil || s.Script != "echo success" {
			t.Error("expected post-success script")
		}
	})

	t.Run("post failure", func(t *testing.T) {
		s := getScriptsByType(scripts, models.BackupScriptTypePostFailure)
		if s == nil || s.Script != "echo failure" {
			t.Error("expected post-failure script")
		}
	})

	t.Run("post always", func(t *testing.T) {
		s := getScriptsByType(scripts, models.BackupScriptTypePostAlways)
		if s == nil || s.Script != "echo always" {
			t.Error("expected post-always script")
		}
	})

	t.Run("not found", func(t *testing.T) {
		s := getScriptsByType(nil, models.BackupScriptTypePreBackup)
		if s != nil {
			t.Error("expected nil for empty scripts list")
		}
	})
}

// DRTestScheduler tests

// mockDRTestStore implements DRTestStore for testing.
type mockDRTestStore struct {
	mu              sync.Mutex
	schedules       []*models.DRTestSchedule
	runbooks        map[uuid.UUID]*models.DRRunbook
	tests           []*models.DRTest
	backupSchedules map[uuid.UUID]*models.Schedule
	repos           map[uuid.UUID]*models.Repository
	getErr          error
	updateErr       error
}

func newMockDRTestStore() *mockDRTestStore {
	return &mockDRTestStore{
		runbooks:        make(map[uuid.UUID]*models.DRRunbook),
		backupSchedules: make(map[uuid.UUID]*models.Schedule),
		repos:           make(map[uuid.UUID]*models.Repository),
	}
}

func (m *mockDRTestStore) GetEnabledDRTestSchedules(ctx context.Context) ([]*models.DRTestSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.schedules, nil
}

func (m *mockDRTestStore) GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rb, ok := m.runbooks[id]; ok {
		return rb, nil
	}
	return nil, errors.New("runbook not found")
}

func (m *mockDRTestStore) CreateDRTest(ctx context.Context, test *models.DRTest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tests = append(m.tests, test)
	return nil
}

func (m *mockDRTestStore) UpdateDRTest(ctx context.Context, test *models.DRTest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateErr
}

func (m *mockDRTestStore) UpdateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *mockDRTestStore) GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.backupSchedules[id]; ok {
		return s, nil
	}
	return nil, errors.New("schedule not found")
}

func (m *mockDRTestStore) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.repos[id]; ok {
		return r, nil
	}
	return nil, errors.New("repository not found")
}

func TestDRTestScheduler_StartStop(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = 100 * time.Millisecond

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Starting again should error
	if err := scheduler.Start(ctx); err == nil {
		t.Error("Start() expected error when already running")
	}

	stopCtx := scheduler.Stop()
	<-stopCtx.Done()

	// Stopping again should not panic
	scheduler.Stop()
}

func TestDRTestScheduler_ReloadSchedules(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	// Initially no schedules
	if scheduler.GetActiveDRSchedules() != 0 {
		t.Errorf("GetActiveDRSchedules() = %d, want 0", scheduler.GetActiveDRSchedules())
	}

	// Add a schedule
	drSchedule := &models.DRTestSchedule{
		ID:             uuid.New(),
		RunbookID:      uuid.New(),
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}
	store.mu.Lock()
	store.schedules = []*models.DRTestSchedule{drSchedule}
	store.mu.Unlock()

	if err := scheduler.ReloadDRSchedules(ctx); err != nil {
		t.Fatalf("ReloadDRSchedules() error = %v", err)
	}
	if scheduler.GetActiveDRSchedules() != 1 {
		t.Errorf("GetActiveDRSchedules() = %d, want 1", scheduler.GetActiveDRSchedules())
	}

	// Remove schedule
	store.mu.Lock()
	store.schedules = nil
	store.mu.Unlock()

	if err := scheduler.ReloadDRSchedules(ctx); err != nil {
		t.Fatalf("ReloadDRSchedules() error = %v", err)
	}
	if scheduler.GetActiveDRSchedules() != 0 {
		t.Errorf("GetActiveDRSchedules() = %d, want 0", scheduler.GetActiveDRSchedules())
	}
}

func TestDRTestScheduler_ReloadError(t *testing.T) {
	store := newMockDRTestStore()
	store.getErr = errors.New("db error")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	err := scheduler.ReloadDRSchedules(context.Background())
	if err == nil {
		t.Fatal("ReloadDRSchedules() expected error")
	}
}

func TestDRTestScheduler_InvalidCron(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	store.mu.Lock()
	store.schedules = []*models.DRTestSchedule{
		{
			ID:             uuid.New(),
			RunbookID:      uuid.New(),
			CronExpression: "invalid",
			Enabled:        true,
		},
	}
	store.mu.Unlock()

	if err := scheduler.ReloadDRSchedules(ctx); err != nil {
		t.Fatalf("ReloadDRSchedules() error = %v", err)
	}
	if scheduler.GetActiveDRSchedules() != 0 {
		t.Errorf("GetActiveDRSchedules() = %d, want 0", scheduler.GetActiveDRSchedules())
	}
}

func TestDRTestScheduler_TriggerDRTest(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	// Add a runbook
	runbookID := uuid.New()
	store.runbooks[runbookID] = &models.DRRunbook{
		ID:   runbookID,
		Name: "Test Runbook",
	}

	err := scheduler.TriggerDRTest(context.Background(), runbookID)
	if err != nil {
		t.Fatalf("TriggerDRTest() error = %v", err)
	}

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)
}

func TestDRTestScheduler_TriggerDRTest_RunbookNotFound(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	err := scheduler.TriggerDRTest(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("TriggerDRTest() expected error for non-existent runbook")
	}
}

func TestScheduler_SetMaintenanceService(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	// SetMaintenanceService should not panic with nil
	scheduler.SetMaintenanceService(nil)

	// Verify maintenance is nil
	if scheduler.maintenance != nil {
		t.Error("maintenance should be nil")
	}
}

func TestScheduler_GetNextAllowedRun(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour

	scheduler := NewScheduler(store, restic, config, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer scheduler.Stop()

	t.Run("non-existent schedule", func(t *testing.T) {
		_, ok := scheduler.GetNextAllowedRun(ctx, uuid.New())
		if ok {
			t.Error("GetNextAllowedRun() should return false for non-existent schedule")
		}
	})

	t.Run("schedule without window", func(t *testing.T) {
		scheduleID := uuid.New()
		store.addSchedule(models.Schedule{
			ID:             scheduleID,
			AgentID:        uuid.New(),
			Name:           "No Window",
			CronExpression: "0 0 * * * *",
			Paths:          []string{"/data"},
			Enabled:        true,
			Repositories: []models.ScheduleRepository{
				{ID: uuid.New(), RepositoryID: uuid.New(), Priority: 0, Enabled: true},
			},
		})

		if err := scheduler.Reload(ctx); err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		nextAllowed, ok := scheduler.GetNextAllowedRun(ctx, scheduleID)
		if !ok {
			t.Fatal("GetNextAllowedRun() should return true")
		}
		if nextAllowed.IsZero() {
			t.Error("GetNextAllowedRun() should return non-zero time")
		}
	})
}

func TestScheduler_RunScript(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	t.Run("success", func(t *testing.T) {
		script := &models.BackupScript{
			Type:           models.BackupScriptTypePreBackup,
			Script:         "echo hello",
			TimeoutSeconds: 10,
		}

		output, err := scheduler.runScript(context.Background(), script, logger)
		if err != nil {
			t.Fatalf("runScript() error = %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("script error", func(t *testing.T) {
		script := &models.BackupScript{
			Type:           models.BackupScriptTypePreBackup,
			Script:         "exit 1",
			TimeoutSeconds: 10,
		}

		_, err := scheduler.runScript(context.Background(), script, logger)
		if err == nil {
			t.Fatal("expected error for failing script")
		}
	})

	t.Run("script with stderr", func(t *testing.T) {
		script := &models.BackupScript{
			Type:           models.BackupScriptTypePreBackup,
			Script:         "echo stdout; echo stderr >&2",
			TimeoutSeconds: 10,
		}

		output, err := scheduler.runScript(context.Background(), script, logger)
		if err != nil {
			t.Fatalf("runScript() error = %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("script timeout", func(t *testing.T) {
		script := &models.BackupScript{
			Type:           models.BackupScriptTypePreBackup,
			Script:         "sleep 60",
			TimeoutSeconds: 1,
		}

		_, err := scheduler.runScript(context.Background(), script, logger)
		if err == nil {
			t.Fatal("expected timeout error")
		}
	})
}

func TestScheduler_RunPreBackupScript(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	t.Run("no pre-backup script", func(t *testing.T) {
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)
		err := scheduler.runPreBackupScript(context.Background(), nil, backup, logger)
		if err != nil {
			t.Fatalf("runPreBackupScript() error = %v", err)
		}
	})

	t.Run("successful pre-backup script", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePreBackup, Script: "echo pre", TimeoutSeconds: 10, FailOnError: false},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		err := scheduler.runPreBackupScript(context.Background(), scripts, backup, logger)
		if err != nil {
			t.Fatalf("runPreBackupScript() error = %v", err)
		}
	})

	t.Run("failing pre-backup script with FailOnError", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePreBackup, Script: "exit 1", TimeoutSeconds: 10, FailOnError: true},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		err := scheduler.runPreBackupScript(context.Background(), scripts, backup, logger)
		if err == nil {
			t.Fatal("expected error for failing script with FailOnError=true")
		}
	})

	t.Run("failing pre-backup script without FailOnError", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePreBackup, Script: "exit 1", TimeoutSeconds: 10, FailOnError: false},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		err := scheduler.runPreBackupScript(context.Background(), scripts, backup, logger)
		if err != nil {
			t.Fatalf("runPreBackupScript() should not error when FailOnError=false, got %v", err)
		}
	})
}

func TestScheduler_RunPostBackupScripts(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	t.Run("no scripts", func(t *testing.T) {
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)
		// Should not panic
		scheduler.runPostBackupScripts(context.Background(), nil, backup, true, logger)
	})

	t.Run("success scripts", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePostSuccess, Script: "echo success", TimeoutSeconds: 10},
			{Type: models.BackupScriptTypePostAlways, Script: "echo always", TimeoutSeconds: 10},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		scheduler.runPostBackupScripts(context.Background(), scripts, backup, true, logger)
	})

	t.Run("failure scripts", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePostFailure, Script: "echo failure", TimeoutSeconds: 10},
			{Type: models.BackupScriptTypePostAlways, Script: "echo always", TimeoutSeconds: 10},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		scheduler.runPostBackupScripts(context.Background(), scripts, backup, false, logger)
	})

	t.Run("script error in post", func(t *testing.T) {
		scripts := []*models.BackupScript{
			{Type: models.BackupScriptTypePostAlways, Script: "exit 1", TimeoutSeconds: 10},
		}
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		// Should not panic even if script fails
		scheduler.runPostBackupScripts(context.Background(), scripts, backup, true, logger)
	})
}

func TestScheduler_FailBackup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newMockStore()
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewScheduler(store, restic, config, nil, logger)
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		scheduler.failBackup(context.Background(), backup, "test error", logger)

		if backup.Status != models.BackupStatusFailed {
			t.Errorf("Status = %v, want failed", backup.Status)
		}
	})

	t.Run("update error", func(t *testing.T) {
		store := newMockStore()
		store.updateErr = errors.New("db error")
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewScheduler(store, restic, config, nil, logger)
		backup := models.NewBackup(uuid.New(), uuid.New(), nil)

		// Should not panic even with update error
		scheduler.failBackup(context.Background(), backup, "test error", logger)
	})
}

func TestScheduler_SendBackupNotification(t *testing.T) {
	t.Run("nil notifier", func(t *testing.T) {
		store := newMockStore()
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewScheduler(store, restic, config, nil, logger)
		schedule := models.Schedule{ID: uuid.New(), AgentID: uuid.New(), Name: "test"}
		backup := models.NewBackup(schedule.ID, schedule.AgentID, nil)

		// Should not panic with nil notifier
		scheduler.sendBackupNotification(context.Background(), schedule, backup, true, "")
	})
}

func TestScheduler_CheckNetworkMounts(t *testing.T) {
	t.Run("agent with no mounts", func(t *testing.T) {
		store := newMockStore()
		agentID := uuid.New()
		store.agents[agentID] = &models.Agent{
			ID:       agentID,
			OrgID:    uuid.New(),
			Hostname: "test-agent",
		}

		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewScheduler(store, restic, config, nil, logger)
		schedule := models.Schedule{
			ID:      uuid.New(),
			AgentID: agentID,
			Paths:   []string{"/home/user/data"},
		}

		err := scheduler.checkNetworkMounts(context.Background(), schedule, logger)
		if err != nil {
			t.Fatalf("checkNetworkMounts() error = %v", err)
		}
	})

	t.Run("agent not found uses default", func(t *testing.T) {
		store := newMockStore()

		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewScheduler(store, restic, config, nil, logger)
		schedule := models.Schedule{
			ID:      uuid.New(),
			AgentID: uuid.New(),
			Paths:   []string{"/data"},
		}

		err := scheduler.checkNetworkMounts(context.Background(), schedule, logger)
		if err != nil {
			t.Fatalf("checkNetworkMounts() error = %v", err)
		}
	})
}

func TestScheduler_ExecuteBackup_NoEnabledRepos(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		Name:           "No Repos",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories:   []models.ScheduleRepository{},
	}

	// Should handle no repos gracefully
	scheduler.executeBackup(schedule)
}

func TestScheduler_ExecuteBackup_OutsideTimeWindow(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	currentHour := time.Now().Hour()
	repoID := uuid.New()
	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		Name:           "Window Test",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
		ExcludedHours: []int{currentHour},
	}

	scheduler.executeBackup(schedule)

	backups := store.getBackups()
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestScheduler_RunBackupToRepo_NoDecryptFunc(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:   repoID,
		Name: "test-repo",
		Type: models.RepositoryTypeLocal,
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: uuid.New(),
		Name:    "Test",
		Paths:   []string{"/data"},
	}
	schedRepo := &models.ScheduleRepository{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Priority:     0,
		Enabled:      true,
	}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when DecryptFunc is nil")
	}
}

func TestScheduler_RunBackupToRepo_NoPasswordFunc(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: uuid.New(),
		Name:    "Test",
		Paths:   []string{"/data"},
	}
	schedRepo := &models.ScheduleRepository{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Priority:     0,
		Enabled:      true,
	}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when PasswordFunc is nil")
	}
}

func TestScheduler_RunBackupToRepo_RepoNotFound(t *testing.T) {
	store := newMockStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: uuid.New(),
		Name:    "Test",
		Paths:   []string{"/data"},
	}
	schedRepo := &models.ScheduleRepository{
		ID:           uuid.New(),
		RepositoryID: uuid.New(),
		Priority:     0,
		Enabled:      true,
	}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when repo not found")
	}
}

func TestScheduler_RunBackupToRepo_CreateBackupError(t *testing.T) {
	store := newMockStore()
	store.createErr = errors.New("db error")

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: uuid.New(),
		Name:    "Test",
		Paths:   []string{"/data"},
	}
	schedRepo := &models.ScheduleRepository{
		ID:           uuid.New(),
		RepositoryID: uuid.New(),
		Priority:     0,
		Enabled:      true,
	}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when CreateBackup fails")
	}
}

func TestScheduler_RunBackupToRepo_Success(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()

	backupResponse := `{"message_type":"summary","snapshot_id":"abc123","files_new":5,"files_changed":2,"data_added":1024}`
	r, cleanup := newTestRestic(backupResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: uuid.New(),
		Name:    "Test",
		Paths:   []string{"/data"},
	}
	schedRepo := &models.ScheduleRepository{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Priority:     0,
		Enabled:      true,
	}

	backup, stats, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err != nil {
		t.Fatalf("runBackupToRepo() error = %v", err)
	}
	if backup == nil {
		t.Fatal("expected non-nil backup")
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.SnapshotID != "abc123" {
		t.Errorf("SnapshotID = %v, want abc123", stats.SnapshotID)
	}
}

func TestScheduler_ReplicateToOtherRepos(t *testing.T) {
	t.Run("skip source repo", func(t *testing.T) {
		store := newMockStore()
		logger := zerolog.Nop()
		r, cleanup := newTestRestic("")
		defer cleanup()
		config := DefaultSchedulerConfig()
		config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
			return []byte(`{"path":"/tmp/repo"}`), nil
		}
		config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
			return "test-password", nil
		}

		scheduler := NewScheduler(store, r, config, nil, logger)

		sourceRepoID := uuid.New()
		schedule := models.Schedule{ID: uuid.New()}
		sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
		allRepos := []models.ScheduleRepository{
			{RepositoryID: sourceRepoID},
		}

		scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
	})

	t.Run("replicate to target", func(t *testing.T) {
		store := newMockStore()
		targetRepoID := uuid.New()
		store.repos[targetRepoID] = &models.Repository{
			ID:              targetRepoID,
			Name:            "target-repo",
			Type:            models.RepositoryTypeLocal,
			ConfigEncrypted: []byte("encrypted"),
		}

		logger := zerolog.Nop()
		r, cleanup := newTestRestic("")
		defer cleanup()
		config := DefaultSchedulerConfig()
		config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
			return []byte(`{"path":"/tmp/target"}`), nil
		}
		config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
			return "target-password", nil
		}

		scheduler := NewScheduler(store, r, config, nil, logger)

		sourceRepoID := uuid.New()
		schedule := models.Schedule{ID: uuid.New()}
		sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
		allRepos := []models.ScheduleRepository{
			{RepositoryID: sourceRepoID},
			{RepositoryID: targetRepoID},
		}

		scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
	})
}

func TestScheduler_ExecuteBackup_FullSuccess(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	agentID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
	}

	logger := zerolog.Nop()

	backupResponse := `{"message_type":"summary","snapshot_id":"abc123","files_new":5,"files_changed":2,"data_added":1024}`
	r, cleanup := newTestRestic(backupResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		Name:           "Full Success Test",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)

	backups := store.getBackups()
	if len(backups) == 0 {
		t.Fatal("expected at least one backup record")
	}
}

func TestScheduler_ExecuteBackup_WithRetention(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	agentID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
	}

	logger := zerolog.Nop()

	backupResponse := `{"message_type":"summary","snapshot_id":"abc123","files_new":5,"files_changed":2,"data_added":1024}`
	r, cleanup := newTestRestic(backupResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.RefreshInterval = time.Hour
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		Name:           "Retention Test",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
		RetentionPolicy: &models.RetentionPolicy{
			KeepDaily: 7,
		},
	}

	scheduler.executeBackup(schedule)
}

func TestScheduler_ExecuteBackup_WithMountUnavailableSkip(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
		NetworkMounts: []models.NetworkMount{
			{Path: "/mnt/nfs", Type: models.MountTypeNFS, Remote: "server:/share", Status: models.MountStatusDisconnected},
		},
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	repoID := uuid.New()
	schedule := models.Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               "Mount Skip Test",
		CronExpression:     "0 0 * * * *",
		Paths:              []string{"/mnt/nfs/data"},
		Enabled:            true,
		OnMountUnavailable: models.MountBehaviorSkip,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)

	backups := store.getBackups()
	if len(backups) != 0 {
		t.Errorf("expected 0 backups (skipped due to mount), got %d", len(backups))
	}
}

func TestScheduler_ExecuteBackup_WithMountUnavailableFail(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	repoID := uuid.New()
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
		NetworkMounts: []models.NetworkMount{
			{Path: "/mnt/nfs", Type: models.MountTypeNFS, Remote: "server:/share", Status: models.MountStatusDisconnected},
		},
	}
	store.repos[repoID] = &models.Repository{
		ID:   repoID,
		Name: "test-repo",
		Type: models.RepositoryTypeLocal,
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               "Mount Fail Test",
		CronExpression:     "0 0 * * * *",
		Paths:              []string{"/mnt/nfs/data"},
		Enabled:            true,
		OnMountUnavailable: models.MountBehaviorFail,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)

	backups := store.getBackups()
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup (failed due to mount), got %d", len(backups))
	}
	if backups[0].Status != models.BackupStatusFailed {
		t.Errorf("backup status = %v, want failed", backups[0].Status)
	}
}

func TestDRTestScheduler_ExecuteDRTest_NoSchedule(t *testing.T) {
	store := newMockDRTestStore()
	runbookID := uuid.New()
	store.runbooks[runbookID] = &models.DRRunbook{
		ID:   runbookID,
		Name: "Test Runbook",
		// No ScheduleID - manual verification path
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	schedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbookID,
	}

	// Should handle runbook without schedule gracefully
	scheduler.executeDRTest(schedule)

	store.mu.Lock()
	count := len(store.tests)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("tests count = %d, want 1", count)
	}
}

func TestDRTestScheduler_ExecuteDRTest_RunbookNotFound(t *testing.T) {
	store := newMockDRTestStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	schedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: uuid.New(), // non-existent
	}

	// Should not panic
	scheduler.executeDRTest(schedule)
}

func TestDRTestScheduler_ExecuteDRTest_WithSchedule(t *testing.T) {
	store := newMockDRTestStore()

	scheduleID := uuid.New()
	repoID := uuid.New()
	runbookID := uuid.New()

	store.runbooks[runbookID] = &models.DRRunbook{
		ID:         runbookID,
		Name:       "Test Runbook",
		ScheduleID: &scheduleID,
	}
	store.backupSchedules[scheduleID] = &models.Schedule{
		ID:   scheduleID,
		Name: "Backup Schedule",
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	// No DecryptFunc - will fail at decrypt step

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	schedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbookID,
	}

	// Should handle missing decrypt func gracefully
	scheduler.executeDRTest(schedule)
}

func TestDRTestScheduler_FailDRTest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newMockDRTestStore()
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewDRTestScheduler(store, restic, config, logger)

		test := models.NewDRTest(uuid.New())
		scheduler.failDRTest(context.Background(), test, "test error", logger)
	})

	t.Run("update error", func(t *testing.T) {
		store := newMockDRTestStore()
		store.updateErr = errors.New("db error")
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultSchedulerConfig()

		scheduler := NewDRTestScheduler(store, restic, config, logger)

		test := models.NewDRTest(uuid.New())
		// Should not panic
		scheduler.failDRTest(context.Background(), test, "test error", logger)
	})
}

func TestScheduler_CheckNetworkMountsWithMounts(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()

	// Create a temp directory to simulate a connected mount
	tmpDir := t.TempDir()

	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
		NetworkMounts: []models.NetworkMount{
			{Path: tmpDir, Type: models.MountTypeNFS, Remote: "server:/share", Status: models.MountStatusConnected},
		},
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)
	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: agentID,
		Paths:   []string{tmpDir + "/data"},
	}

	err := scheduler.checkNetworkMounts(context.Background(), schedule, logger)
	if err != nil {
		t.Fatalf("checkNetworkMounts() error = %v", err)
	}
}

func TestScheduler_SendBackupNotification_WithNotifier(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	orgID := uuid.New()
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-agent",
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	notifStore := &mockNotificationStore{}
	notifier := notifications.NewService(notifStore, zerolog.Nop())

	scheduler := NewScheduler(store, restic, config, notifier, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: agentID,
		Name:    "test-schedule",
	}

	now := time.Now()
	completedAt := now.Add(10 * time.Second)
	sizeBytes := int64(1024)
	filesNew := 5
	filesChanged := 2

	backup := models.NewBackup(schedule.ID, schedule.AgentID, nil)
	backup.SnapshotID = "snap123"
	backup.CompletedAt = &completedAt
	backup.SizeBytes = &sizeBytes
	backup.FilesNew = &filesNew
	backup.FilesChanged = &filesChanged

	// Test success notification
	scheduler.sendBackupNotification(context.Background(), schedule, backup, true, "")
	// Test failure notification
	scheduler.sendBackupNotification(context.Background(), schedule, backup, false, "test error")
}

func TestScheduler_SendBackupNotification_WithMinimalBackup(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	store.agents[agentID] = &models.Agent{
		ID:       agentID,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	notifStore := &mockNotificationStore{}
	notifier := notifications.NewService(notifStore, zerolog.Nop())

	scheduler := NewScheduler(store, restic, config, notifier, logger)

	schedule := models.Schedule{
		ID:      uuid.New(),
		AgentID: agentID,
		Name:    "test-schedule",
	}

	// Backup with nil optional fields
	backup := models.NewBackup(schedule.ID, schedule.AgentID, nil)

	scheduler.sendBackupNotification(context.Background(), schedule, backup, true, "")
}

func TestScheduler_ReplicateToOtherRepos_DecryptError(t *testing.T) {
	store := newMockStore()
	targetRepoID := uuid.New()
	store.repos[targetRepoID] = &models.Repository{
		ID:              targetRepoID,
		Name:            "target-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("")
	defer cleanup()
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return nil, errors.New("decrypt failed")
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	sourceRepoID := uuid.New()
	schedule := models.Schedule{ID: uuid.New()}
	sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
	allRepos := []models.ScheduleRepository{
		{RepositoryID: sourceRepoID},
		{RepositoryID: targetRepoID},
	}

	scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
}

func TestScheduler_ReplicateToOtherRepos_PasswordError(t *testing.T) {
	store := newMockStore()
	targetRepoID := uuid.New()
	store.repos[targetRepoID] = &models.Repository{
		ID:              targetRepoID,
		Name:            "target-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("")
	defer cleanup()
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "", errors.New("password error")
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	sourceRepoID := uuid.New()
	schedule := models.Schedule{ID: uuid.New()}
	sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
	allRepos := []models.ScheduleRepository{
		{RepositoryID: sourceRepoID},
		{RepositoryID: targetRepoID},
	}

	scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
}

func TestScheduler_ReplicateToOtherRepos_ParseBackendError(t *testing.T) {
	store := newMockStore()
	targetRepoID := uuid.New()
	store.repos[targetRepoID] = &models.Repository{
		ID:              targetRepoID,
		Name:            "target-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("")
	defer cleanup()
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`invalid json`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	sourceRepoID := uuid.New()
	schedule := models.Schedule{ID: uuid.New()}
	sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
	allRepos := []models.ScheduleRepository{
		{RepositoryID: sourceRepoID},
		{RepositoryID: targetRepoID},
	}

	scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
}

func TestScheduler_ReplicateToOtherRepos_TargetRepoNotFound(t *testing.T) {
	store := newMockStore()

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("")
	defer cleanup()
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	sourceRepoID := uuid.New()
	schedule := models.Schedule{ID: uuid.New()}
	sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
	allRepos := []models.ScheduleRepository{
		{RepositoryID: sourceRepoID},
		{RepositoryID: uuid.New()}, // Non-existent target
	}

	scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
}

func TestScheduler_ReplicateToOtherRepos_CopyError(t *testing.T) {
	store := newMockStore()
	targetRepoID := uuid.New()
	store.repos[targetRepoID] = &models.Repository{
		ID:              targetRepoID,
		Name:            "target-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestResticError("copy failed")
	defer cleanup()
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "target-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	sourceRepoID := uuid.New()
	schedule := models.Schedule{ID: uuid.New()}
	sourceRepo := &models.ScheduleRepository{RepositoryID: sourceRepoID}
	allRepos := []models.ScheduleRepository{
		{RepositoryID: sourceRepoID},
		{RepositoryID: targetRepoID},
	}

	scheduler.replicateToOtherRepos(context.Background(), schedule, sourceRepo, "snap1", testResticConfig(), allRepos, logger)
}

func TestScheduler_RunBackupToRepo_DecryptError(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return nil, errors.New("decrypt failed")
	}

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{ID: uuid.New(), AgentID: uuid.New(), Name: "Test", Paths: []string{"/data"}}
	schedRepo := &models.ScheduleRepository{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when decrypt fails")
	}
}

func TestScheduler_RunBackupToRepo_ParseBackendError(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`invalid json`), nil
	}

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{ID: uuid.New(), AgentID: uuid.New(), Name: "Test", Paths: []string{"/data"}}
	schedRepo := &models.ScheduleRepository{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when parse backend fails")
	}
}

func TestScheduler_RunBackupToRepo_PasswordError(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "", errors.New("password failed")
	}

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{ID: uuid.New(), AgentID: uuid.New(), Name: "Test", Paths: []string{"/data"}}
	schedRepo := &models.ScheduleRepository{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when password func fails")
	}
}

func TestScheduler_RunBackupToRepo_BackupError(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestResticError("backup failed")
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	schedule := models.Schedule{ID: uuid.New(), AgentID: uuid.New(), Name: "Test", Paths: []string{"/data"}}
	schedRepo := &models.ScheduleRepository{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true}

	_, _, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err == nil {
		t.Fatal("expected error when backup fails")
	}
}

func TestScheduler_RunBackupToRepo_WithBandwidthLimit(t *testing.T) {
	store := newMockStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()

	backupResponse := `{"message_type":"summary","snapshot_id":"abc123","files_new":5,"files_changed":2,"data_added":1024}`
	r, cleanup := newTestRestic(backupResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	bwLimit := 1024
	compressionLevel := "auto"
	schedule := models.Schedule{
		ID:               uuid.New(),
		AgentID:          uuid.New(),
		Name:             "Test",
		Paths:            []string{"/data"},
		BandwidthLimitKB: &bwLimit,
		CompressionLevel: &compressionLevel,
	}
	schedRepo := &models.ScheduleRepository{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true}

	backup, stats, _, err := scheduler.runBackupToRepo(context.Background(), schedule, schedRepo, nil, 1, logger)
	if err != nil {
		t.Fatalf("runBackupToRepo() error = %v", err)
	}
	if backup == nil {
		t.Fatal("expected non-nil backup")
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestScheduler_ExecuteBackup_PreScriptFailure(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{ID: repoID, Name: "test-repo", Type: models.RepositoryTypeLocal}
	store.agents[agentID] = &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "test-agent"}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	scheduleID := uuid.New()
	store.scripts[scheduleID] = []*models.BackupScript{
		{Type: models.BackupScriptTypePreBackup, Script: "exit 1", TimeoutSeconds: 10, FailOnError: true},
	}

	schedule := models.Schedule{
		ID:             scheduleID,
		AgentID:        agentID,
		Name:           "Pre Script Fail",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)
}

func TestScheduler_ExecuteBackup_AllReposFail(t *testing.T) {
	store := newMockStore()
	agentID := uuid.New()
	repoID := uuid.New()
	store.agents[agentID] = &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "test-agent"}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewScheduler(store, restic, config, nil, logger)

	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		Name:           "All Fail",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)
}

func TestScheduler_ExecuteBackup_ScriptsLoadError(t *testing.T) {
	store := newMockStore()
	store.scriptErr = errors.New("scripts db error")
	agentID := uuid.New()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.agents[agentID] = &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "test-agent"}

	logger := zerolog.Nop()
	backupResponse := `{"message_type":"summary","snapshot_id":"abc123","files_new":5,"files_changed":2,"data_added":1024}`
	r, cleanup := newTestRestic(backupResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewScheduler(store, r, config, nil, logger)

	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		Name:           "Scripts Error",
		CronExpression: "0 0 * * * *",
		Paths:          []string{"/data"},
		Enabled:        true,
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}

	scheduler.executeBackup(schedule)
}

func TestDRTestScheduler_ExecuteDRTest_FullPath(t *testing.T) {
	store := newMockDRTestStore()

	scheduleID := uuid.New()
	repoID := uuid.New()
	runbookID := uuid.New()

	store.runbooks[runbookID] = &models.DRRunbook{
		ID:         runbookID,
		Name:       "Test Runbook",
		ScheduleID: &scheduleID,
	}
	store.backupSchedules[scheduleID] = &models.Schedule{
		ID:   scheduleID,
		Name: "Backup Schedule",
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()

	// Use a snapshot JSON array - Snapshots will parse it, Stats will fail
	snapshotResponse := `[{"id":"snap1","short_id":"snap1","time":"2024-01-01T00:00:00Z","hostname":"test"}]`
	r, cleanup := newTestRestic(snapshotResponse)
	defer cleanup()

	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	scheduler := NewDRTestScheduler(store, r, config, logger)

	drSchedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbookID,
	}

	// Snapshots succeeds, Stats fails since mock returns snapshot JSON for all calls
	scheduler.executeDRTest(drSchedule)

	store.mu.Lock()
	count := len(store.tests)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("tests count = %d, want 1", count)
	}
}

func TestDRTestScheduler_ExecuteDRTest_NoPrimaryRepo(t *testing.T) {
	store := newMockDRTestStore()

	scheduleID := uuid.New()
	runbookID := uuid.New()

	store.runbooks[runbookID] = &models.DRRunbook{
		ID:         runbookID,
		Name:       "Test Runbook",
		ScheduleID: &scheduleID,
	}
	store.backupSchedules[scheduleID] = &models.Schedule{
		ID:           scheduleID,
		Name:         "Backup Schedule",
		Repositories: []models.ScheduleRepository{}, // No repos
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	drSchedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbookID,
	}

	scheduler.executeDRTest(drSchedule)
}

func TestDRTestScheduler_ExecuteDRTest_PasswordFuncNil(t *testing.T) {
	store := newMockDRTestStore()

	scheduleID := uuid.New()
	repoID := uuid.New()
	runbookID := uuid.New()

	store.runbooks[runbookID] = &models.DRRunbook{
		ID:         runbookID,
		Name:       "Test Runbook",
		ScheduleID: &scheduleID,
	}
	store.backupSchedules[scheduleID] = &models.Schedule{
		ID:   scheduleID,
		Name: "Backup Schedule",
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultSchedulerConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	// PasswordFunc is nil

	scheduler := NewDRTestScheduler(store, restic, config, logger)

	drSchedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbookID,
	}

	scheduler.executeDRTest(drSchedule)
}

// containsStr is a helper to check if a string contains a substring.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
