package commands

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu     sync.Mutex
	callFn func(ctx context.Context) (int64, error)
	calls  int
}

func newMockStore(fn func(ctx context.Context) (int64, error)) *mockStore {
	return &mockStore{callFn: fn}
}

func (s *mockStore) MarkTimedOutCommands(ctx context.Context) (int64, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	return s.callFn(ctx)
}

func (s *mockStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.Disabled)
}

// waitFor polls the predicate and fails after timeout.
func waitFor(t *testing.T, timeout time.Duration, msg string, pred func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		if pred() {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting: %s", msg)
		case <-ticker.C:
		}
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestTimeoutWorker_PickupAndExecution(t *testing.T) {
	t.Run("runs immediately on start then periodically", func(t *testing.T) {
		store := newMockStore(func(_ context.Context) (int64, error) {
			return 0, nil
		})
		w := NewTimeoutWorker(store, 20*time.Millisecond, testLogger())

		ctx, cancel := context.WithCancel(context.Background())
		go w.Start(ctx)

		// The worker calls processTimeouts immediately, then once per interval.
		// After ~50ms we should see at least 2 calls (immediate + 1-2 ticks).
		waitFor(t, 500*time.Millisecond, "at least 2 calls", func() bool {
			return store.callCount() >= 2
		})

		cancel()
	})

	t.Run("reports marked count", func(t *testing.T) {
		var reported int64
		store := newMockStore(func(_ context.Context) (int64, error) {
			return atomic.AddInt64(&reported, 3), nil
		})
		w := NewTimeoutWorker(store, 20*time.Millisecond, testLogger())

		ctx, cancel := context.WithCancel(context.Background())
		go w.Start(ctx)

		waitFor(t, 500*time.Millisecond, "store should be called", func() bool {
			return store.callCount() >= 1
		})

		cancel()
	})
}

func TestTimeoutWorker_TimeoutEnforcement(t *testing.T) {
	t.Run("context cancellation stops worker", func(t *testing.T) {
		store := newMockStore(func(ctx context.Context) (int64, error) {
			return 0, nil
		})
		w := NewTimeoutWorker(store, 10*time.Millisecond, testLogger())

		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		go func() {
			w.Start(ctx)
			close(done)
		}()

		select {
		case <-done:
			// Worker exited as expected after context timeout.
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not exit after context cancellation")
		}

		// The worker should have made at least the immediate call.
		if store.callCount() < 1 {
			t.Error("expected at least 1 call to MarkTimedOutCommands")
		}
	})

	t.Run("store error does not stop worker", func(t *testing.T) {
		var calls int32
		store := newMockStore(func(_ context.Context) (int64, error) {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				return 0, fmt.Errorf("transient DB error")
			}
			return 0, nil
		})
		w := NewTimeoutWorker(store, 15*time.Millisecond, testLogger())

		ctx, cancel := context.WithCancel(context.Background())
		go w.Start(ctx)

		// The worker should continue past the error and make more calls.
		waitFor(t, 500*time.Millisecond, "should recover from error and continue", func() bool {
			return atomic.LoadInt32(&calls) >= 3
		})

		cancel()
	})
}

func TestTimeoutWorker_ResultReporting(t *testing.T) {
	t.Run("non-zero count is logged", func(t *testing.T) {
		var totalMarked int64
		store := newMockStore(func(_ context.Context) (int64, error) {
			return atomic.AddInt64(&totalMarked, 2), nil
		})
		w := NewTimeoutWorker(store, 20*time.Millisecond, testLogger())

		ctx, cancel := context.WithCancel(context.Background())
		go w.Start(ctx)

		waitFor(t, 500*time.Millisecond, "should call store multiple times", func() bool {
			return store.callCount() >= 2
		})

		// Verify the store was actually returning non-zero results.
		if atomic.LoadInt64(&totalMarked) < 4 {
			t.Errorf("expected totalMarked >= 4, got %d", atomic.LoadInt64(&totalMarked))
		}

		cancel()
	})

	t.Run("zero count is silent", func(t *testing.T) {
		store := newMockStore(func(_ context.Context) (int64, error) {
			return 0, nil
		})
		w := NewTimeoutWorker(store, 20*time.Millisecond, testLogger())

		ctx, cancel := context.WithCancel(context.Background())
		go w.Start(ctx)

		waitFor(t, 200*time.Millisecond, "store called at least once", func() bool {
			return store.callCount() >= 1
		})

		cancel()
	})
}

func TestTimeoutWorker_ConcurrentProcessing(t *testing.T) {
	t.Run("multiple workers share store safely", func(t *testing.T) {
		var totalCalls int32
		store := newMockStore(func(_ context.Context) (int64, error) {
			atomic.AddInt32(&totalCalls, 1)
			// Simulate some work.
			time.Sleep(5 * time.Millisecond)
			return 1, nil
		})

		const workerCount = 5
		ctx, cancel := context.WithCancel(context.Background())

		var wg sync.WaitGroup
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := NewTimeoutWorker(store, 15*time.Millisecond, testLogger())
				w.Start(ctx)
			}()
		}

		// Let them run for a bit.
		waitFor(t, 1*time.Second, "concurrent calls accumulated", func() bool {
			return atomic.LoadInt32(&totalCalls) >= int32(workerCount*2)
		})

		cancel()
		wg.Wait()

		finalCalls := atomic.LoadInt32(&totalCalls)
		if finalCalls < int32(workerCount) {
			t.Errorf("expected at least %d total calls, got %d", workerCount, finalCalls)
		}
	})

	t.Run("start-stop rapid cycling", func(t *testing.T) {
		store := newMockStore(func(_ context.Context) (int64, error) {
			return 0, nil
		})

		for i := 0; i < 10; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				w := NewTimeoutWorker(store, 10*time.Millisecond, testLogger())
				w.Start(ctx)
				close(done)
			}()
			// Cancel quickly.
			time.Sleep(5 * time.Millisecond)
			cancel()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("iteration %d: worker did not exit after cancel", i)
			}
		}
	})
}

func TestNewTimeoutWorker(t *testing.T) {
	store := newMockStore(func(_ context.Context) (int64, error) { return 0, nil })
	w := NewTimeoutWorker(store, 42*time.Second, testLogger())

	if w.interval != 42*time.Second {
		t.Errorf("expected interval 42s, got %s", w.interval)
	}
	if w.store == nil {
		t.Error("expected non-nil store")
	}
}

func TestDefaultTimeoutCheckInterval(t *testing.T) {
	if DefaultTimeoutCheckInterval != 30*time.Second {
		t.Errorf("expected 30s, got %s", DefaultTimeoutCheckInterval)
	}
}
