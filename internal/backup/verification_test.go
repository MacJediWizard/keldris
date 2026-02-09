package backup

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockVerificationStore implements VerificationStore for testing.
type mockVerificationStore struct {
	mu                   sync.Mutex
	schedules            []*models.VerificationSchedule
	schedulesByRepo      map[uuid.UUID][]*models.VerificationSchedule
	repos                map[uuid.UUID]*models.Repository
	verifications        []*models.Verification
	latestByRepo         map[uuid.UUID]*models.Verification
	consecutiveFails     map[uuid.UUID]int
	getErr               error
	createErr            error
	updateErr            error
}

func newMockVerificationStore() *mockVerificationStore {
	return &mockVerificationStore{
		schedulesByRepo:  make(map[uuid.UUID][]*models.VerificationSchedule),
		repos:            make(map[uuid.UUID]*models.Repository),
		latestByRepo:     make(map[uuid.UUID]*models.Verification),
		consecutiveFails: make(map[uuid.UUID]int),
	}
}

func (m *mockVerificationStore) GetEnabledVerificationSchedules(ctx context.Context) ([]*models.VerificationSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.schedules, nil
}

func (m *mockVerificationStore) GetVerificationSchedulesByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.VerificationSchedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.schedulesByRepo[repoID], nil
}

func (m *mockVerificationStore) GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	if repo, ok := m.repos[id]; ok {
		return repo, nil
	}
	return nil, errors.New("repository not found")
}

func (m *mockVerificationStore) CreateVerification(ctx context.Context, v *models.Verification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.verifications = append(m.verifications, v)
	return nil
}

func (m *mockVerificationStore) UpdateVerification(ctx context.Context, v *models.Verification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateErr
}

func (m *mockVerificationStore) GetLatestVerificationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.Verification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.latestByRepo[repoID]; ok {
		return v, nil
	}
	return nil, errors.New("no verification found")
}

func (m *mockVerificationStore) GetConsecutiveFailedVerifications(ctx context.Context, repoID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.consecutiveFails[repoID], nil
}

// mockVerificationNotifier implements VerificationNotifier for testing.
type mockVerificationNotifier struct {
	mu            sync.Mutex
	notifications []mockNotification
	notifyErr     error
}

type mockNotification struct {
	verification     *models.Verification
	repo             *models.Repository
	consecutiveFails int
}

func newMockVerificationNotifier() *mockVerificationNotifier {
	return &mockVerificationNotifier{}
}

func (m *mockVerificationNotifier) NotifyVerificationFailed(ctx context.Context, v *models.Verification, repo *models.Repository, consecutiveFails int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.notifyErr != nil {
		return m.notifyErr
	}
	m.notifications = append(m.notifications, mockNotification{
		verification:     v,
		repo:             repo,
		consecutiveFails: consecutiveFails,
	})
	return nil
}

func TestVerificationScheduler_StartStop(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = 100 * time.Millisecond

	vs := NewVerificationScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Starting again should error
	if err := vs.Start(ctx); err == nil {
		t.Error("Start() expected error when already running")
	}

	stopCtx := vs.Stop()
	<-stopCtx.Done()

	// Stopping again should not panic
	vs.Stop()
}

func TestVerificationScheduler_Reload(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = time.Hour

	vs := NewVerificationScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer vs.Stop()

	// Initially no schedules
	vs.mu.RLock()
	count := len(vs.entries)
	vs.mu.RUnlock()
	if count != 0 {
		t.Errorf("entries count = %d, want 0", count)
	}

	// Add a verification schedule
	repoID := uuid.New()
	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}
	store.mu.Lock()
	store.schedules = []*models.VerificationSchedule{schedule}
	store.mu.Unlock()

	if err := vs.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	vs.mu.RLock()
	count = len(vs.entries)
	vs.mu.RUnlock()
	if count != 1 {
		t.Errorf("entries count = %d, want 1", count)
	}

	// Remove schedule
	store.mu.Lock()
	store.schedules = nil
	store.mu.Unlock()

	if err := vs.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	vs.mu.RLock()
	count = len(vs.entries)
	vs.mu.RUnlock()
	if count != 0 {
		t.Errorf("entries count = %d, want 0", count)
	}
}

func TestVerificationScheduler_ReloadError(t *testing.T) {
	store := newMockVerificationStore()
	store.getErr = errors.New("db error")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	err := vs.Reload(context.Background())
	if err == nil {
		t.Fatal("Reload() expected error")
	}
}

func TestVerificationScheduler_InvalidCron(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer vs.Stop()

	store.mu.Lock()
	store.schedules = []*models.VerificationSchedule{
		{
			ID:             uuid.New(),
			RepositoryID:   uuid.New(),
			Type:           models.VerificationTypeCheck,
			CronExpression: "invalid",
			Enabled:        true,
		},
	}
	store.mu.Unlock()

	if err := vs.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	vs.mu.RLock()
	count := len(vs.entries)
	vs.mu.RUnlock()
	if count != 0 {
		t.Errorf("entries count = %d, want 0", count)
	}
}

func TestVerificationScheduler_GetNextRun(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = time.Hour

	vs := NewVerificationScheduler(store, restic, config, logger)

	t.Run("non-existent", func(t *testing.T) {
		_, ok := vs.GetNextRun(uuid.New())
		if ok {
			t.Error("GetNextRun() should return false for non-existent schedule")
		}
	})

	t.Run("existing schedule", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := vs.Start(ctx); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		defer vs.Stop()

		schedID := uuid.New()
		store.mu.Lock()
		store.schedules = []*models.VerificationSchedule{
			{
				ID:             schedID,
				RepositoryID:   uuid.New(),
				Type:           models.VerificationTypeCheck,
				CronExpression: "0 0 * * * *",
				Enabled:        true,
			},
		}
		store.mu.Unlock()

		if err := vs.Reload(ctx); err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		nextRun, ok := vs.GetNextRun(schedID)
		if !ok {
			t.Fatal("GetNextRun() should return true")
		}
		if nextRun.IsZero() {
			t.Error("GetNextRun() should return non-zero time")
		}
	})
}

func TestVerificationScheduler_TriggerVerification(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	repoID := uuid.New()

	v, err := vs.TriggerVerification(context.Background(), repoID, models.VerificationTypeCheck)
	if err != nil {
		t.Fatalf("TriggerVerification() error = %v", err)
	}
	if v == nil {
		t.Fatal("TriggerVerification() returned nil verification")
	}
	if v.RepositoryID != repoID {
		t.Errorf("RepositoryID = %v, want %v", v.RepositoryID, repoID)
	}
	if v.Type != models.VerificationTypeCheck {
		t.Errorf("Type = %v, want check", v.Type)
	}

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)
}

func TestVerificationScheduler_TriggerVerification_CreateError(t *testing.T) {
	store := newMockVerificationStore()
	store.createErr = errors.New("create failed")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	_, err := vs.TriggerVerification(context.Background(), uuid.New(), models.VerificationTypeCheck)
	if err == nil {
		t.Fatal("expected error when create fails")
	}
}

func TestVerificationScheduler_GetRepositoryVerificationStatus(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = time.Hour

	vs := NewVerificationScheduler(store, restic, config, logger)

	repoID := uuid.New()

	t.Run("no data", func(t *testing.T) {
		status, err := vs.GetRepositoryVerificationStatus(context.Background(), repoID)
		if err != nil {
			t.Fatalf("GetRepositoryVerificationStatus() error = %v", err)
		}
		if status.RepositoryID != repoID {
			t.Errorf("RepositoryID = %v, want %v", status.RepositoryID, repoID)
		}
	})

	t.Run("with latest verification", func(t *testing.T) {
		latest := models.NewVerification(repoID, models.VerificationTypeCheck)
		latest.Pass(nil)
		store.mu.Lock()
		store.latestByRepo[repoID] = latest
		store.mu.Unlock()

		status, err := vs.GetRepositoryVerificationStatus(context.Background(), repoID)
		if err != nil {
			t.Fatalf("GetRepositoryVerificationStatus() error = %v", err)
		}
		if status.LastVerification == nil {
			t.Error("LastVerification should not be nil")
		}
	})

	t.Run("with consecutive failures", func(t *testing.T) {
		store.mu.Lock()
		store.consecutiveFails[repoID] = 3
		store.mu.Unlock()

		status, err := vs.GetRepositoryVerificationStatus(context.Background(), repoID)
		if err != nil {
			t.Fatalf("GetRepositoryVerificationStatus() error = %v", err)
		}
		if status.ConsecutiveFails != 3 {
			t.Errorf("ConsecutiveFails = %v, want 3", status.ConsecutiveFails)
		}
	})
}

func TestDefaultVerificationConfig(t *testing.T) {
	config := DefaultVerificationConfig()

	if config.RefreshInterval != 5*time.Minute {
		t.Errorf("RefreshInterval = %v, want 5m", config.RefreshInterval)
	}
	if config.AlertAfterConsecutiveFails != 1 {
		t.Errorf("AlertAfterConsecutiveFails = %v, want 1", config.AlertAfterConsecutiveFails)
	}
	if config.TempDir == "" {
		t.Error("TempDir should not be empty")
	}
}

func TestVerificationScheduler_CheckAndNotify(t *testing.T) {
	store := newMockVerificationStore()
	notifier := newMockVerificationNotifier()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.Notifier = notifier
	config.AlertAfterConsecutiveFails = 2

	vs := NewVerificationScheduler(store, restic, config, logger)

	repoID := uuid.New()
	repo := &models.Repository{
		ID:   repoID,
		Name: "test-repo",
	}
	store.repos[repoID] = repo

	verification := models.NewVerification(repoID, models.VerificationTypeCheck)
	verification.Fail("check failed", nil)

	t.Run("below threshold", func(t *testing.T) {
		store.mu.Lock()
		store.consecutiveFails[repoID] = 1
		store.mu.Unlock()

		vs.checkAndNotify(context.Background(), verification, repo, logger)

		notifier.mu.Lock()
		count := len(notifier.notifications)
		notifier.mu.Unlock()
		if count != 0 {
			t.Errorf("notifications count = %d, want 0 (below threshold)", count)
		}
	})

	t.Run("at threshold", func(t *testing.T) {
		store.mu.Lock()
		store.consecutiveFails[repoID] = 2
		store.mu.Unlock()

		vs.checkAndNotify(context.Background(), verification, repo, logger)

		notifier.mu.Lock()
		count := len(notifier.notifications)
		notifier.mu.Unlock()
		if count != 1 {
			t.Errorf("notifications count = %d, want 1", count)
		}
	})

	t.Run("above threshold", func(t *testing.T) {
		store.mu.Lock()
		store.consecutiveFails[repoID] = 5
		store.mu.Unlock()

		vs.checkAndNotify(context.Background(), verification, repo, logger)

		notifier.mu.Lock()
		count := len(notifier.notifications)
		notifier.mu.Unlock()
		if count != 2 {
			t.Errorf("notifications count = %d, want 2", count)
		}
	})

	t.Run("nil notifier", func(t *testing.T) {
		vsNoNotify := NewVerificationScheduler(store, restic, DefaultVerificationConfig(), logger)
		// Should not panic
		vsNoNotify.checkAndNotify(context.Background(), verification, repo, logger)
	})
}

func TestVerificationScheduler_RunCheck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestRestic("no errors found")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		details, err := vs.runCheck(context.Background(), testResticConfig(), false, "")
		if err != nil {
			t.Fatalf("runCheck() error = %v", err)
		}
		if details == nil {
			t.Fatal("expected non-nil details")
		}
	})

	t.Run("with read data", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestRestic("")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		details, err := vs.runCheck(context.Background(), testResticConfig(), true, "")
		if err != nil {
			t.Fatalf("runCheck() error = %v", err)
		}
		if details == nil {
			t.Fatal("expected non-nil details")
		}
	})

	t.Run("with read data subset", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestRestic("")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		details, err := vs.runCheck(context.Background(), testResticConfig(), true, "5%")
		if err != nil {
			t.Fatalf("runCheck() error = %v", err)
		}
		if details == nil {
			t.Fatal("expected non-nil details")
		}
		if details.ReadDataSubset != "5%" {
			t.Errorf("ReadDataSubset = %v, want 5%%", details.ReadDataSubset)
		}
	})

	t.Run("check error", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestResticError("check failed: pack errors")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		details, err := vs.runCheck(context.Background(), testResticConfig(), false, "")
		if err == nil {
			t.Fatal("expected error")
		}
		if details == nil {
			t.Fatal("expected non-nil details even on error")
		}
		if len(details.ErrorsFound) == 0 {
			t.Error("expected errors in details")
		}
	})
}

func TestVerificationScheduler_RunTestRestore(t *testing.T) {
	t.Run("no snapshots", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestRestic("[]")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		_, err := vs.runTestRestore(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("expected error for no snapshots")
		}
	})

	t.Run("snapshot error", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		r, cleanup := newTestResticError("connection refused")
		defer cleanup()
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		_, err := vs.runTestRestore(context.Background(), testResticConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with snapshots - restore succeeds", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()

		// Mock returns snapshot JSON for Snapshots call, then empty for Restore call.
		// Our mock returns same response for all calls, so we use a response that
		// can be parsed as both snapshots JSON array and survive a restore call.
		snapshotsJSON := `[{"id":"snap1","short_id":"snap1","time":"2024-01-01T00:00:00Z","hostname":"test","paths":["/data"]}]`
		r, cleanup := newTestRestic(snapshotsJSON)
		defer cleanup()

		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, r, config, logger)

		// runTestRestore calls Snapshots (OK), then Restore (gets snapshot JSON but
		// succeeds because mock exits 0), then walks temp dir
		details, err := vs.runTestRestore(context.Background(), testResticConfig())
		if err != nil {
			t.Fatalf("runTestRestore() error = %v", err)
		}
		if details == nil {
			t.Fatal("expected non-nil details")
		}
	})
}

func TestVerificationScheduler_FailVerification(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, restic, config, logger)

		repoID := uuid.New()
		verification := models.NewVerification(repoID, models.VerificationTypeCheck)

		vs.failVerification(context.Background(), verification, "test error", nil, logger)

		if verification.Status != models.VerificationStatusFailed {
			t.Errorf("Status = %v, want failed", verification.Status)
		}
	})

	t.Run("with details", func(t *testing.T) {
		store := newMockVerificationStore()
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, restic, config, logger)

		repoID := uuid.New()
		verification := models.NewVerification(repoID, models.VerificationTypeCheck)
		details := &models.VerificationDetails{
			ErrorsFound: []string{"pack error"},
		}

		vs.failVerification(context.Background(), verification, "check error", details, logger)
	})

	t.Run("update error", func(t *testing.T) {
		store := newMockVerificationStore()
		store.updateErr = errors.New("db error")
		logger := zerolog.Nop()
		restic := NewRestic(logger)
		config := DefaultVerificationConfig()

		vs := NewVerificationScheduler(store, restic, config, logger)

		repoID := uuid.New()
		verification := models.NewVerification(repoID, models.VerificationTypeCheck)

		// Should not panic
		vs.failVerification(context.Background(), verification, "test error", nil, logger)
	})
}

func TestVerificationScheduler_ExecuteVerification_CreateError(t *testing.T) {
	store := newMockVerificationStore()
	store.createErr = errors.New("create failed")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   uuid.New(),
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	// Should not panic even when create fails
	vs.executeVerification(schedule)
}

func TestVerificationScheduler_ExecuteVerification_NoDecryptFunc(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:   repoID,
		Name: "test-repo",
		Type: models.RepositoryTypeLocal,
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	// No DecryptFunc

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	// Should not panic, will fail internally due to no DecryptFunc
	vs.executeVerification(schedule)

	// Verify a verification record was created and is failed
	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_NoPasswordFunc(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	// No PasswordFunc

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)
}

func TestVerificationScheduler_ExecuteVerification_CheckSuccess(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("no errors found")
	defer cleanup()

	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(id uuid.UUID) (string, error) {
		return "test-password", nil
	}

	vs := NewVerificationScheduler(store, r, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_TestRestoreType(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	// Provide snapshot JSON for Snapshots call (runTestRestore calls Snapshots then Restore)
	snapshotsJSON := `[{"id":"snap1","short_id":"snap1","time":"2024-01-01T00:00:00Z","hostname":"test","paths":["/data"]}]`
	r, cleanup := newTestRestic(snapshotsJSON)
	defer cleanup()

	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(id uuid.UUID) (string, error) {
		return "test-password", nil
	}

	vs := NewVerificationScheduler(store, r, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeTestRestore,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_CheckReadDataType(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestRestic("no errors found")
	defer cleanup()

	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(id uuid.UUID) (string, error) {
		return "test-password", nil
	}

	vs := NewVerificationScheduler(store, r, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheckReadData,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
		ReadDataSubset: "5%",
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_DecryptError(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return nil, errors.New("decrypt failed")
	}

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_PasswordError(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(id uuid.UUID) (string, error) {
		return "", errors.New("password error")
	}

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_RepoNotFound(t *testing.T) {
	store := newMockVerificationStore()
	// Don't add repo to store
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   uuid.New(),
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_ExecuteVerification_ParseBackendError(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`invalid json`), nil
	}

	vs := NewVerificationScheduler(store, restic, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)
}

func TestVerificationScheduler_ExecuteVerification_CheckFails(t *testing.T) {
	store := newMockVerificationStore()
	repoID := uuid.New()
	store.repos[repoID] = &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}

	logger := zerolog.Nop()
	r, cleanup := newTestResticError("pack errors found")
	defer cleanup()

	config := DefaultVerificationConfig()
	config.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	config.PasswordFunc = func(id uuid.UUID) (string, error) {
		return "test-password", nil
	}

	vs := NewVerificationScheduler(store, r, config, logger)

	schedule := &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 0 * * * *",
		Enabled:        true,
	}

	vs.executeVerification(schedule)

	store.mu.Lock()
	count := len(store.verifications)
	store.mu.Unlock()
	if count != 1 {
		t.Fatalf("verifications count = %d, want 1", count)
	}
}

func TestVerificationScheduler_GetRepositoryVerificationStatus_WithSchedules(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = time.Hour

	vs := NewVerificationScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer vs.Stop()

	repoID := uuid.New()
	schedID := uuid.New()

	// Add schedule and repo data
	store.mu.Lock()
	store.schedules = []*models.VerificationSchedule{
		{
			ID:             schedID,
			RepositoryID:   repoID,
			Type:           models.VerificationTypeCheck,
			CronExpression: "0 0 * * * *",
			Enabled:        true,
		},
	}
	store.schedulesByRepo[repoID] = store.schedules
	store.mu.Unlock()

	if err := vs.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	status, err := vs.GetRepositoryVerificationStatus(ctx, repoID)
	if err != nil {
		t.Fatalf("GetRepositoryVerificationStatus() error = %v", err)
	}
	if status.RepositoryID != repoID {
		t.Errorf("RepositoryID = %v, want %v", status.RepositoryID, repoID)
	}
	if status.NextScheduledAt == nil {
		t.Error("NextScheduledAt should not be nil for a scheduled repo")
	}
}

func TestVerificationScheduler_GetRepositoryVerificationStatus_GetErr(t *testing.T) {
	store := newMockVerificationStore()
	store.getErr = errors.New("db error")
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()

	vs := NewVerificationScheduler(store, restic, config, logger)

	// getErr affects GetVerificationSchedulesByRepoID but the function
	// still returns a status (just with nil fields for the errored lookups)
	status, err := vs.GetRepositoryVerificationStatus(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetRepositoryVerificationStatus() error = %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestVerificationScheduler_ReloadMultipleSchedules(t *testing.T) {
	store := newMockVerificationStore()
	logger := zerolog.Nop()
	restic := NewRestic(logger)
	config := DefaultVerificationConfig()
	config.RefreshInterval = time.Hour

	vs := NewVerificationScheduler(store, restic, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vs.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer vs.Stop()

	// Add multiple schedules for different repos and types
	repo1 := uuid.New()
	repo2 := uuid.New()
	store.mu.Lock()
	store.schedules = []*models.VerificationSchedule{
		{
			ID:             uuid.New(),
			RepositoryID:   repo1,
			Type:           models.VerificationTypeCheck,
			CronExpression: "0 0 * * * *",
			Enabled:        true,
		},
		{
			ID:             uuid.New(),
			RepositoryID:   repo1,
			Type:           models.VerificationTypeTestRestore,
			CronExpression: "0 0 0 * * *",
			Enabled:        true,
		},
		{
			ID:             uuid.New(),
			RepositoryID:   repo2,
			Type:           models.VerificationTypeCheckReadData,
			CronExpression: "0 0 12 * * *",
			Enabled:        true,
			ReadDataSubset: "5%",
		},
	}
	store.mu.Unlock()

	if err := vs.Reload(ctx); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	vs.mu.RLock()
	count := len(vs.entries)
	vs.mu.RUnlock()
	if count != 3 {
		t.Errorf("entries count = %d, want 3", count)
	}
}
