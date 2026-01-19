package backup

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockStore implements ScheduleStore for testing.
type mockStore struct {
	mu         sync.Mutex
	schedules  []models.Schedule
	repos      map[uuid.UUID]*models.Repository
	backups    []*models.Backup
	getErr     error
	createErr  error
	updateErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		repos:   make(map[uuid.UUID]*models.Repository),
		backups: make([]*models.Backup, 0),
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

func (m *mockStore) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &models.Agent{
		ID:       id,
		OrgID:    uuid.New(),
		Hostname: "test-agent",
	}, nil
}

func (m *mockStore) addSchedule(s models.Schedule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules = append(m.schedules, s)
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
	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "Test Schedule",
		CronExpression: "0 */5 * * * *", // Every 5 minutes with seconds
		Paths:          []string{"/home/user"},
		Enabled:        true,
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
	store.schedules = nil
	if err := scheduler.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if scheduler.GetActiveSchedules() != 0 {
		t.Errorf("GetActiveSchedules() = %d, want 0", scheduler.GetActiveSchedules())
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
	schedule := models.Schedule{
		ID:             uuid.New(),
		AgentID:        uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "Invalid Schedule",
		CronExpression: "invalid cron",
		Paths:          []string{"/home/user"},
		Enabled:        true,
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
