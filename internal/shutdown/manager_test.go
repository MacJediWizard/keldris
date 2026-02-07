package shutdown

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockBackupTracker is a mock implementation of BackupTracker for testing.
type mockBackupTracker struct {
	mu             sync.RWMutex
	running        map[uuid.UUID]struct{}
	checkpointed   map[uuid.UUID]bool
	checkpointErr  error
	cancelErr      error
}

func newMockBackupTracker() *mockBackupTracker {
	return &mockBackupTracker{
		running:      make(map[uuid.UUID]struct{}),
		checkpointed: make(map[uuid.UUID]bool),
	}
}

func (m *mockBackupTracker) addRunning(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running[id] = struct{}{}
}

func (m *mockBackupTracker) removeRunning(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.running, id)
}

func (m *mockBackupTracker) GetRunningBackupIDs() []uuid.UUID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(m.running))
	for id := range m.running {
		ids = append(ids, id)
	}
	return ids
}

func (m *mockBackupTracker) CheckpointBackup(ctx context.Context, backupID uuid.UUID) (bool, error) {
	if m.checkpointErr != nil {
		return false, m.checkpointErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpointed[backupID] = true
	return true, nil
}

func (m *mockBackupTracker) IsBackupRunning(backupID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.running[backupID]
	return ok
}

func (m *mockBackupTracker) CancelBackup(ctx context.Context, backupID uuid.UUID) error {
	return m.cancelErr
}

func TestManager_NewManager(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracker := newMockBackupTracker()

	m := NewManager(config, tracker, logger)

	if m == nil {
		t.Fatal("expected manager to be created")
	}

	if !m.IsAcceptingJobs() {
		t.Error("expected manager to accept jobs initially")
	}

	if m.GetState() != StateRunning {
		t.Errorf("expected state to be running, got %s", m.GetState())
	}
}

func TestManager_GetStatus(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultConfig()
	tracker := newMockBackupTracker()
	tracker.addRunning(uuid.New())
	tracker.addRunning(uuid.New())

	m := NewManager(config, tracker, logger)

	status := m.GetStatus()

	if status.State != StateRunning {
		t.Errorf("expected state running, got %s", status.State)
	}

	if !status.AcceptingNewJobs {
		t.Error("expected accepting new jobs to be true")
	}

	if status.RunningBackups != 2 {
		t.Errorf("expected 2 running backups, got %d", status.RunningBackups)
	}
}

func TestManager_ShutdownNoBackups(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:                  5 * time.Second,
		DrainTimeout:             100 * time.Millisecond,
		CheckpointRunningBackups: true,
	}
	tracker := newMockBackupTracker()

	m := NewManager(config, tracker, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.Shutdown(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.GetState() != StateComplete {
		t.Errorf("expected state complete, got %s", m.GetState())
	}

	if m.IsAcceptingJobs() {
		t.Error("expected not accepting jobs after shutdown")
	}
}

func TestManager_ShutdownWithBackups(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:                  2 * time.Second,
		DrainTimeout:             100 * time.Millisecond,
		CheckpointRunningBackups: true,
	}
	tracker := newMockBackupTracker()

	backupID := uuid.New()
	tracker.addRunning(backupID)

	m := NewManager(config, tracker, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.Shutdown(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for done signal to ensure all goroutines have completed
	<-m.Done()

	if m.GetState() != StateComplete {
		t.Errorf("expected state complete, got %s", m.GetState())
	}

	// Check that backup was checkpointed (it was still running when we got to checkpoint phase)
	tracker.mu.RLock()
	checkpointed := tracker.checkpointed[backupID]
	tracker.mu.RUnlock()

	if !checkpointed {
		t.Error("expected backup to be checkpointed")
	}
}

func TestManager_ShutdownBackupsComplete(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:                  5 * time.Second,
		DrainTimeout:             100 * time.Millisecond,
		CheckpointRunningBackups: true,
	}
	tracker := newMockBackupTracker()

	backupID := uuid.New()
	tracker.addRunning(backupID)

	m := NewManager(config, tracker, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start shutdown in goroutine
	done := make(chan error)
	go func() {
		done <- m.Shutdown(ctx)
	}()

	// Wait for drain to complete, then simulate backup completion
	time.Sleep(200 * time.Millisecond)
	tracker.removeRunning(backupID)

	err := <-done
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	<-m.Done()

	if m.GetState() != StateComplete {
		t.Errorf("expected state complete, got %s", m.GetState())
	}

	// Backup completed before checkpoint phase, so it shouldn't be checkpointed
	tracker.mu.RLock()
	checkpointed := tracker.checkpointed[backupID]
	tracker.mu.RUnlock()

	if checkpointed {
		t.Error("expected backup not to be checkpointed since it completed")
	}
}

func TestManager_ShutdownTimeout(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:                  2 * time.Second,
		DrainTimeout:             100 * time.Millisecond,
		CheckpointRunningBackups: true,
	}
	tracker := newMockBackupTracker()

	// Add a backup that never completes
	backupID := uuid.New()
	tracker.addRunning(backupID)

	m := NewManager(config, tracker, logger)

	ctx := context.Background()

	start := time.Now()
	err := m.Shutdown(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete around the timeout duration (within 2 seconds + some margin)
	if elapsed > 3*time.Second {
		t.Errorf("expected shutdown to complete within timeout, took %v", elapsed)
	}

	<-m.Done()

	if m.GetState() != StateComplete {
		t.Errorf("expected state complete, got %s", m.GetState())
	}
}

func TestManager_ShutdownOnce(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:      1 * time.Second,
		DrainTimeout: 100 * time.Millisecond,
	}
	tracker := newMockBackupTracker()

	m := NewManager(config, tracker, logger)

	ctx := context.Background()

	// Call shutdown multiple times
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Shutdown(ctx)
		}()
	}

	wg.Wait()

	// Should only complete once
	if m.GetState() != StateComplete {
		t.Errorf("expected state complete, got %s", m.GetState())
	}
}

func TestManager_Done(t *testing.T) {
	logger := zerolog.Nop()
	config := Config{
		Timeout:      1 * time.Second,
		DrainTimeout: 100 * time.Millisecond,
	}
	tracker := newMockBackupTracker()

	m := NewManager(config, tracker, logger)

	// Done channel should not be closed yet
	select {
	case <-m.Done():
		t.Fatal("expected done channel to not be closed before shutdown")
	default:
	}

	// Start shutdown
	go func() {
		_ = m.Shutdown(context.Background())
	}()

	// Wait for done
	select {
	case <-m.Done():
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for done channel")
	}
}
