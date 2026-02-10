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

type mockStatsStore struct {
	mu            sync.Mutex
	orgs          []*models.Organization
	repos         map[uuid.UUID][]*models.Repository
	repoByID      map[uuid.UUID]*models.Repository
	stats         []*models.StorageStats
	getOrgsErr    error
	getReposErr   error
	getRepoErr    error
	createStatsErr error
}

func newMockStatsStore() *mockStatsStore {
	return &mockStatsStore{
		repos:    make(map[uuid.UUID][]*models.Repository),
		repoByID: make(map[uuid.UUID]*models.Repository),
	}
}

func (m *mockStatsStore) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.orgs, m.getOrgsErr
}

func (m *mockStatsStore) GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getReposErr != nil {
		return nil, m.getReposErr
	}
	return m.repos[orgID], nil
}

func (m *mockStatsStore) GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	repo, ok := m.repoByID[id]
	if !ok {
		return nil, errors.New("repository not found")
	}
	return repo, nil
}

func (m *mockStatsStore) CreateStorageStats(ctx context.Context, stats *models.StorageStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createStatsErr != nil {
		return m.createStatsErr
	}
	m.stats = append(m.stats, stats)
	return nil
}

func TestDefaultStatsCollectorConfig(t *testing.T) {
	cfg := DefaultStatsCollectorConfig()
	if cfg.CronSchedule != "0 0 2 * * *" {
		t.Errorf("CronSchedule = %v, want 0 0 2 * * *", cfg.CronSchedule)
	}
	if cfg.PasswordFunc != nil {
		t.Error("PasswordFunc should be nil by default")
	}
	if cfg.DecryptFunc != nil {
		t.Error("DecryptFunc should be nil by default")
	}
}

func TestNewStatsCollector(t *testing.T) {
	store := newMockStatsStore()
	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())
	if collector == nil {
		t.Fatal("expected non-nil collector")
	}
	if collector.IsRunning() {
		t.Error("collector should not be running before Start")
	}
}

func TestStatsCollector_StartStop(t *testing.T) {
	store := newMockStatsStore()
	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	// Start
	err := collector.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !collector.IsRunning() {
		t.Error("collector should be running after Start")
	}

	// Starting again should be a no-op
	err = collector.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() second call error = %v", err)
	}

	// Stop
	stopCtx := collector.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not complete in time")
	}

	if collector.IsRunning() {
		t.Error("collector should not be running after Stop")
	}

	// Stopping again should be safe
	stopCtx = collector.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("second Stop() did not complete in time")
	}
}

func TestStatsCollector_GetNextRun(t *testing.T) {
	store := newMockStatsStore()
	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	// Not running
	_, ok := collector.GetNextRun()
	if ok {
		t.Error("GetNextRun() should return false when not running")
	}

	// Start
	if err := collector.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer collector.Stop()

	nextRun, ok := collector.GetNextRun()
	if !ok {
		t.Error("GetNextRun() should return true when running")
	}
	if nextRun.IsZero() {
		t.Error("next run time should not be zero")
	}
}

func TestStatsCollector_CollectNow(t *testing.T) {
	t.Run("no organizations", func(t *testing.T) {
		store := newMockStatsStore()
		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		err := collector.CollectNow(context.Background())
		if err != nil {
			t.Fatalf("CollectNow() error = %v", err)
		}
	})

	t.Run("get orgs error", func(t *testing.T) {
		store := newMockStatsStore()
		store.getOrgsErr = errors.New("db error")
		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		err := collector.CollectNow(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with org but no decrypt func", func(t *testing.T) {
		store := newMockStatsStore()
		orgID := uuid.New()
		repoID := uuid.New()
		store.orgs = []*models.Organization{{ID: orgID, Name: "test-org"}}
		repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo", Type: models.RepositoryTypeLocal}
		store.repos[orgID] = []*models.Repository{repo}
		store.repoByID[repoID] = repo

		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()
		// DecryptFunc is nil, so collectRepoStats should skip with warning

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		err := collector.CollectNow(context.Background())
		if err != nil {
			t.Fatalf("CollectNow() error = %v", err)
		}
	})

	t.Run("get repos error continues", func(t *testing.T) {
		store := newMockStatsStore()
		orgID := uuid.New()
		store.orgs = []*models.Organization{{ID: orgID, Name: "test-org"}}
		store.getReposErr = errors.New("repos error")

		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		// Should not return error, just log and continue
		err := collector.CollectNow(context.Background())
		if err != nil {
			t.Fatalf("CollectNow() error = %v", err)
		}
	})
}

func TestStatsCollector_CollectForRepository(t *testing.T) {
	t.Run("repo not found", func(t *testing.T) {
		store := newMockStatsStore()
		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		err := collector.CollectForRepository(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error for non-existent repo")
		}
	})

	t.Run("no password func", func(t *testing.T) {
		store := newMockStatsStore()
		repoID := uuid.New()
		repo := &models.Repository{ID: repoID, Name: "test-repo", Type: models.RepositoryTypeLocal}
		store.repoByID[repoID] = repo

		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()
		cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
			return []byte(`{"path":"/tmp/repo"}`), nil
		}
		// PasswordFunc is nil

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

		err := collector.CollectForRepository(context.Background(), repoID)
		if err != nil {
			t.Fatalf("CollectForRepository() should succeed with nil password func (skip), got error = %v", err)
		}
	})
}

func TestStatsCollector_CollectForRepository_DecryptError(t *testing.T) {
	store := newMockStatsStore()
	repoID := uuid.New()
	repo := &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.repoByID[repoID] = repo

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return nil, errors.New("decrypt failed")
	}

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	err := collector.CollectForRepository(context.Background(), repoID)
	if err == nil {
		t.Fatal("expected error when decrypt fails")
	}
}

func TestStatsCollector_CollectForRepository_ParseBackendError(t *testing.T) {
	store := newMockStatsStore()
	repoID := uuid.New()
	repo := &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.repoByID[repoID] = repo

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`invalid json`), nil
	}

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	err := collector.CollectForRepository(context.Background(), repoID)
	if err == nil {
		t.Fatal("expected error when parse backend fails")
	}
}

func TestStatsCollector_CollectForRepository_PasswordError(t *testing.T) {
	store := newMockStatsStore()
	repoID := uuid.New()
	repo := &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.repoByID[repoID] = repo

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	cfg.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "", errors.New("password error")
	}

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	err := collector.CollectForRepository(context.Background(), repoID)
	if err == nil {
		t.Fatal("expected error when password func fails")
	}
}

func TestStatsCollector_CollectNow_WithMultipleOrgs(t *testing.T) {
	store := newMockStatsStore()
	orgID1 := uuid.New()
	orgID2 := uuid.New()

	store.orgs = []*models.Organization{
		{ID: orgID1, Name: "org1"},
		{ID: orgID2, Name: "org2"},
	}

	repo1 := &models.Repository{
		ID:    uuid.New(),
		OrgID: orgID1,
		Name:  "repo1",
		Type:  models.RepositoryTypeLocal,
	}
	repo2 := &models.Repository{
		ID:    uuid.New(),
		OrgID: orgID2,
		Name:  "repo2",
		Type:  models.RepositoryTypeLocal,
	}
	store.repos[orgID1] = []*models.Repository{repo1}
	store.repos[orgID2] = []*models.Repository{repo2}

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	// DecryptFunc is nil so repos will be skipped

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	err := collector.CollectNow(context.Background())
	if err != nil {
		t.Fatalf("CollectNow() error = %v", err)
	}
}

func TestStatsCollector_CollectNow_RepoStatsFail(t *testing.T) {
	store := newMockStatsStore()
	orgID := uuid.New()

	store.orgs = []*models.Organization{{ID: orgID, Name: "test-org"}}
	repo := &models.Repository{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.repos[orgID] = []*models.Repository{repo}

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return nil, errors.New("decrypt failed")
	}

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	err := collector.CollectNow(context.Background())
	if err != nil {
		t.Fatalf("CollectNow() error = %v", err)
	}
}

func TestStatsCollector_CollectForRepository_GetExtendedStatsError(t *testing.T) {
	store := newMockStatsStore()
	repoID := uuid.New()
	repo := &models.Repository{
		ID:              repoID,
		Name:            "test-repo",
		Type:            models.RepositoryTypeLocal,
		ConfigEncrypted: []byte("encrypted"),
	}
	store.repoByID[repoID] = repo

	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()
	cfg.DecryptFunc = func(encrypted []byte) ([]byte, error) {
		return []byte(`{"path":"/tmp/repo"}`), nil
	}
	cfg.PasswordFunc = func(repoID uuid.UUID) (string, error) {
		return "test-password", nil
	}

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	// GetExtendedStats will fail because real restic binary isn't available
	err := collector.CollectForRepository(context.Background(), repoID)
	if err == nil {
		t.Fatal("expected error from GetExtendedStats")
	}
}

func TestStatsCollector_CollectAllStats(t *testing.T) {
	t.Run("no orgs", func(t *testing.T) {
		store := newMockStatsStore()
		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())
		// collectAllStats is the cron callback wrapper
		collector.collectAllStats()
	})

	t.Run("with error", func(t *testing.T) {
		store := newMockStatsStore()
		store.getOrgsErr = errors.New("db error")
		r := NewRestic(zerolog.Nop())
		cfg := DefaultStatsCollectorConfig()

		collector := NewStatsCollector(store, r, cfg, zerolog.Nop())
		// Should not panic, logs error internally
		collector.collectAllStats()
	})
}

func TestStatsCollector_IsRunning(t *testing.T) {
	store := newMockStatsStore()
	r := NewRestic(zerolog.Nop())
	cfg := DefaultStatsCollectorConfig()

	collector := NewStatsCollector(store, r, cfg, zerolog.Nop())

	if collector.IsRunning() {
		t.Error("should not be running initially")
	}

	collector.Start(context.Background())
	if !collector.IsRunning() {
		t.Error("should be running after Start")
	}

	collector.Stop()
	if collector.IsRunning() {
		t.Error("should not be running after Stop")
	}
}
