package maintenance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
)

// mockRetentionStore implements RetentionStore for testing.
type mockRetentionStore struct {
	mu            sync.Mutex
	calls         int
	lastDays      int
	deletedCount  int64
	err           error
}

func (m *mockRetentionStore) CleanupAgentHealthHistory(_ context.Context, retentionDays int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastDays = retentionDays
	if m.err != nil {
		return 0, m.err
	}
	return m.deletedCount, nil
}

func (m *mockRetentionStore) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *mockRetentionStore) getLastDays() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastDays
}

func TestNewRetentionScheduler(t *testing.T) {
	store := &mockRetentionStore{}
	s := NewRetentionScheduler(store, 90, zerolog.Nop())

	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if s.retentionDays != 90 {
		t.Errorf("expected retentionDays=90, got %d", s.retentionDays)
	}
	if s.running {
		t.Error("expected scheduler to not be running initially")
	}
}

func TestRetentionScheduler_StartStop(t *testing.T) {
	store := &mockRetentionStore{}
	s := NewRetentionScheduler(store, 30, zerolog.Nop())

	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error starting scheduler: %v", err)
	}

	if !s.running {
		t.Error("expected scheduler to be running after Start()")
	}

	// Starting again should return an error
	if err := s.Start(); err == nil {
		t.Error("expected error when starting already-running scheduler")
	}

	s.Stop()

	if s.running {
		t.Error("expected scheduler to not be running after Stop()")
	}
}

func TestRetentionScheduler_StopWhenNotRunning(t *testing.T) {
	store := &mockRetentionStore{}
	s := NewRetentionScheduler(store, 30, zerolog.Nop())

	// Stopping without starting should not panic
	ctx := s.Stop()
	if ctx == nil {
		t.Error("expected non-nil context from Stop()")
	}
}

func TestRetentionScheduler_RunNow(t *testing.T) {
	store := &mockRetentionStore{deletedCount: 42}
	s := NewRetentionScheduler(store, 60, zerolog.Nop())

	s.RunNow()

	if store.getCalls() != 1 {
		t.Errorf("expected 1 call, got %d", store.getCalls())
	}
	if store.getLastDays() != 60 {
		t.Errorf("expected retentionDays=60, got %d", store.getLastDays())
	}
}

func TestRetentionScheduler_RunNow_Error(t *testing.T) {
	store := &mockRetentionStore{err: errors.New("db connection lost")}
	s := NewRetentionScheduler(store, 90, zerolog.Nop())

	// Should not panic on error
	s.RunNow()

	if store.getCalls() != 1 {
		t.Errorf("expected 1 call, got %d", store.getCalls())
	}
}

func TestRetentionScheduler_RunNow_ZeroDeleted(t *testing.T) {
	store := &mockRetentionStore{deletedCount: 0}
	s := NewRetentionScheduler(store, 90, zerolog.Nop())

	s.RunNow()

	if store.getCalls() != 1 {
		t.Errorf("expected 1 call, got %d", store.getCalls())
	}
}

func TestRetentionScheduler_CustomRetentionDays(t *testing.T) {
	tests := []struct {
		name string
		days int
	}{
		{"7 days", 7},
		{"30 days", 30},
		{"90 days", 90},
		{"365 days", 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockRetentionStore{deletedCount: 10}
			s := NewRetentionScheduler(store, tt.days, zerolog.Nop())

			s.RunNow()

			if store.getLastDays() != tt.days {
				t.Errorf("expected retentionDays=%d, got %d", tt.days, store.getLastDays())
			}
		})
	}
}

func TestRetentionScheduler_ConcurrentRunNow(t *testing.T) {
	store := &mockRetentionStore{deletedCount: 5}
	s := NewRetentionScheduler(store, 90, zerolog.Nop())

	var wg sync.WaitGroup
	var completed atomic.Int32

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.RunNow()
			completed.Add(1)
		}()
	}

	wg.Wait()

	if completed.Load() != 10 {
		t.Errorf("expected 10 completions, got %d", completed.Load())
	}
	if store.getCalls() != 10 {
		t.Errorf("expected 10 calls, got %d", store.getCalls())
	}
}
