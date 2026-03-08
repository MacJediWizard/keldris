package jobs

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockJobStore struct {
	mu   sync.Mutex
	jobs []*models.Job
	// getNextPendingFn lets individual tests override dequeue behaviour.
	getNextPendingFn func(ctx context.Context, orgID uuid.UUID) (*models.Job, error)
	// cleanupOldJobsFn lets individual tests override cleanup behaviour.
	cleanupOldJobsFn func(ctx context.Context, retentionDays int) (int64, error)
	// listJobsReadyForRetryFn lets individual tests override retry listing.
	listJobsReadyForRetryFn func(ctx context.Context, limit int) ([]*models.Job, error)
}

func newMockJobStore() *mockJobStore {
	return &mockJobStore{}
}

func (s *mockJobStore) CreateJob(_ context.Context, job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	return nil
}

// copyJob returns a shallow copy of the job (mirrors real DB: each query yields
// its own object). This prevents data races between the worker goroutine that
// mutates the returned job and the test goroutine that reads via GetJobByID.
func copyJob(j *models.Job) *models.Job {
	cp := *j
	return &cp
}

func (s *mockJobStore) GetJobByID(_ context.Context, id uuid.UUID) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.ID == id {
			return copyJob(j), nil
		}
	}
	return nil, fmt.Errorf("job not found")
}

func (s *mockJobStore) UpdateJob(_ context.Context, job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.ID == job.ID {
			cp := copyJob(job)
			s.jobs[i] = cp
			return nil
		}
	}
	return fmt.Errorf("job not found")
}

// GetNextPendingJob returns the first pending job (FIFO). Each call atomically
// transitions the job to running so that concurrent workers don't double-claim.
// Returns a copy to prevent races between the caller and later GetJobByID reads.
func (s *mockJobStore) GetNextPendingJob(ctx context.Context, orgID uuid.UUID) (*models.Job, error) {
	if s.getNextPendingFn != nil {
		return s.getNextPendingFn(ctx, orgID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.OrgID == orgID && j.Status == models.JobStatusPending {
			j.Status = models.JobStatusRunning
			return copyJob(j), nil
		}
	}
	return nil, nil
}

func (s *mockJobStore) ListJobsReadyForRetry(ctx context.Context, limit int) ([]*models.Job, error) {
	if s.listJobsReadyForRetryFn != nil {
		return s.listJobsReadyForRetryFn(ctx, limit)
	}
	return nil, nil
}

func (s *mockJobStore) GetJobQueueSummary(_ context.Context, _ uuid.UUID) (*models.JobQueueSummary, error) {
	return &models.JobQueueSummary{}, nil
}

func (s *mockJobStore) CleanupOldJobs(ctx context.Context, retentionDays int) (int64, error) {
	if s.cleanupOldJobsFn != nil {
		return s.cleanupOldJobsFn(ctx, retentionDays)
	}
	return 0, nil
}

// allJobs returns a snapshot (test helper).
func (s *mockJobStore) allJobs() []*models.Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*models.Job, len(s.jobs))
	copy(out, s.jobs)
	return out
}

// ---------------------------------------------------------------------------
// Mock handler
// ---------------------------------------------------------------------------

type mockHandler struct {
	handleFn func(ctx context.Context, job *models.Job) (map[string]interface{}, error)
}

func (h *mockHandler) Handle(ctx context.Context, job *models.Job) (map[string]interface{}, error) {
	return h.handleFn(ctx, job)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.Disabled)
}

func fastConfig(workerCount int) QueueConfig {
	return QueueConfig{
		WorkerCount:       workerCount,
		PollInterval:      10 * time.Millisecond,
		RetryPollInterval: 50 * time.Millisecond,
		CleanupInterval:   50 * time.Millisecond,
		JobRetentionDays:  1,
		MaxJobDuration:    5 * time.Second,
	}
}

func makeJob(orgID uuid.UUID, jobType models.JobType, priority int) *models.Job {
	return models.NewJob(orgID, jobType, priority, models.JobPayload{
		Description: fmt.Sprintf("test %s priority %d", jobType, priority),
	})
}

// waitFor polls the given predicate at short intervals and fails the test if
// the predicate is not satisfied within the timeout.
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

func TestEnqueueDequeue_FIFO(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	ctx := context.Background()

	// Enqueue three jobs in order.
	for i := 0; i < 3; i++ {
		job := makeJob(orgID, models.JobTypeBackup, 0)
		job.Payload.Description = fmt.Sprintf("job-%d", i)
		if err := q.Enqueue(ctx, job); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	// Dequeue them and verify FIFO order.
	for i := 0; i < 3; i++ {
		got, err := store.GetNextPendingJob(ctx, orgID)
		if err != nil {
			t.Fatalf("dequeue %d: %v", i, err)
		}
		if got == nil {
			t.Fatalf("expected a job at position %d, got nil", i)
		}
		want := fmt.Sprintf("job-%d", i)
		if got.Payload.Description != want {
			t.Errorf("order mismatch at %d: got %q, want %q", i, got.Payload.Description, want)
		}
	}

	// Queue should be empty now.
	got, err := store.GetNextPendingJob(ctx, orgID)
	if err != nil {
		t.Fatalf("dequeue after empty: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after draining queue, got %v", got)
	}
}

func TestEnqueue_OrgMismatch(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	job := makeJob(uuid.New(), models.JobTypeBackup, 0)
	err := q.Enqueue(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for org mismatch, got nil")
	}
}

func TestWorkerPoolStartsCorrectWorkers(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()

	for _, wc := range []int{1, 3, 5} {
		t.Run(fmt.Sprintf("workers=%d", wc), func(t *testing.T) {
			cfg := fastConfig(wc)
			q := NewQueue(store, orgID, cfg, testLogger())

			// Register a dummy handler so no "unregistered" error is logged.
			q.RegisterHandler(models.JobTypeBackup, &mockHandler{
				handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
					return nil, nil
				},
			})

			ctx, cancel := context.WithCancel(context.Background())
			if err := q.Start(ctx); err != nil {
				t.Fatalf("start: %v", err)
			}

			// Start should not be callable twice.
			if err := q.Start(ctx); err == nil {
				t.Fatal("expected error from double Start")
			}

			cancel()
			// Stop waits for all goroutines (workers + retry + cleanup).
			q.Stop()
		})
	}
}

func TestJobStateTransitions_PendingToCompleted(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(1)
	q := NewQueue(store, orgID, cfg, testLogger())

	processed := make(chan uuid.UUID, 1)
	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, job *models.Job) (map[string]interface{}, error) {
			processed <- job.ID
			return map[string]interface{}{"ok": true}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := makeJob(orgID, models.JobTypeBackup, 0)
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if job.Status != models.JobStatusPending {
		t.Fatalf("expected pending, got %s", job.Status)
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	select {
	case id := <-processed:
		if id != job.ID {
			t.Fatalf("processed wrong job: got %s, want %s", id, job.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for job to be processed")
	}

	// Give a moment for the store update to propagate.
	waitFor(t, 1*time.Second, "job should be completed", func() bool {
		j, _ := store.GetJobByID(ctx, job.ID)
		return j != nil && j.Status == models.JobStatusCompleted
	})

	j, _ := store.GetJobByID(ctx, job.ID)
	if j.Payload.Result == nil {
		t.Error("expected result to be set")
	}
	if j.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	cancel()
	q.Stop()
}

func TestJobStateTransitions_PendingToFailed(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(1)
	q := NewQueue(store, orgID, cfg, testLogger())

	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
			return nil, fmt.Errorf("simulated failure")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := makeJob(orgID, models.JobTypeBackup, 0)
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// The job should eventually be marked as failed.
	waitFor(t, 2*time.Second, "job should be failed", func() bool {
		j, _ := store.GetJobByID(ctx, job.ID)
		return j != nil && (j.Status == models.JobStatusFailed || j.Status == models.JobStatusDeadLetter)
	})

	j, _ := store.GetJobByID(ctx, job.ID)
	if j.ErrorMessage == "" {
		t.Error("expected ErrorMessage to be set")
	}
	if j.RetryCount == 0 {
		t.Error("expected RetryCount to be incremented")
	}

	cancel()
	q.Stop()
}

func TestJobStateTransitions_ExhaustRetries(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(1)
	q := NewQueue(store, orgID, cfg, testLogger())

	var callCount int32
	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
			atomic.AddInt32(&callCount, 1)
			return nil, fmt.Errorf("always fail")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := makeJob(orgID, models.JobTypeBackup, 0)
	job.MaxRetries = 1 // fail on first attempt -> dead letter
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	waitFor(t, 2*time.Second, "job should reach dead_letter", func() bool {
		j, _ := store.GetJobByID(ctx, job.ID)
		return j != nil && j.Status == models.JobStatusDeadLetter
	})

	j, _ := store.GetJobByID(ctx, job.ID)
	if j.Status != models.JobStatusDeadLetter {
		t.Errorf("expected dead_letter, got %s", j.Status)
	}

	cancel()
	q.Stop()
}

func TestNoHandlerRegistered(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(1)
	q := NewQueue(store, orgID, cfg, testLogger())

	// Deliberately do NOT register a handler.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := makeJob(orgID, models.JobTypeBackup, 0)
	// The "no handler" code path calls job.Fail(), which only reaches dead_letter
	// after exhausting retries. Set MaxRetries=1 so the first Fail moves it there.
	job.MaxRetries = 1
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// The job should reach dead_letter because there is no handler registered.
	waitFor(t, 2*time.Second, "job should fail with no handler", func() bool {
		j, _ := store.GetJobByID(ctx, job.ID)
		return j != nil && j.Status == models.JobStatusDeadLetter
	})

	j, _ := store.GetJobByID(ctx, job.ID)
	if j.ErrorMessage != "no handler registered for job type" {
		t.Errorf("unexpected error message: %q", j.ErrorMessage)
	}

	cancel()
	q.Stop()
}

func TestConcurrentJobProcessing(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	workerCount := 4
	cfg := fastConfig(workerCount)
	q := NewQueue(store, orgID, cfg, testLogger())

	const jobCount = 20
	var processed int32
	startBarrier := make(chan struct{})
	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
			// Small delay to increase concurrency overlap.
			<-startBarrier
			atomic.AddInt32(&processed, 1)
			return nil, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < jobCount; i++ {
		job := makeJob(orgID, models.JobTypeBackup, 0)
		if err := q.Enqueue(ctx, job); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Allow all handlers to proceed.
	close(startBarrier)

	waitFor(t, 5*time.Second, "all jobs should be processed", func() bool {
		return atomic.LoadInt32(&processed) == jobCount
	})

	if n := atomic.LoadInt32(&processed); n != jobCount {
		t.Errorf("expected %d processed, got %d", jobCount, n)
	}

	cancel()
	q.Stop()
}

func TestConcurrentEnqueue(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	const goroutines = 10
	const perGoroutine = 10

	var wg sync.WaitGroup
	ctx := context.Background()
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				job := makeJob(orgID, models.JobTypeBackup, 0)
				if err := q.Enqueue(ctx, job); err != nil {
					t.Errorf("enqueue: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	total := len(store.allJobs())
	want := goroutines * perGoroutine
	if total != want {
		t.Errorf("expected %d jobs, got %d", want, total)
	}
}

func TestJobCancellationMidExecution(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(1)
	// Short max job duration to observe timeout.
	cfg.MaxJobDuration = 100 * time.Millisecond
	q := NewQueue(store, orgID, cfg, testLogger())

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})
	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(ctx context.Context, _ *models.Job) (map[string]interface{}, error) {
			close(handlerStarted)
			// Block until the context is canceled (timeout).
			<-ctx.Done()
			close(handlerDone)
			return nil, ctx.Err()
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := makeJob(orgID, models.JobTypeBackup, 0)
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for the handler to start.
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	// The job context should time out after MaxJobDuration (100ms).
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler to observe cancellation")
	}

	// The handler returned an error, so the job should be failed.
	waitFor(t, 1*time.Second, "job should be failed after timeout", func() bool {
		j, _ := store.GetJobByID(ctx, job.ID)
		return j != nil && (j.Status == models.JobStatusFailed || j.Status == models.JobStatusDeadLetter)
	})

	cancel()
	q.Stop()
}

func TestQueueShutdown_GracefulDrain(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	cfg := fastConfig(2)
	q := NewQueue(store, orgID, cfg, testLogger())

	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
			return nil, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Cancel the parent context.
	cancel()

	// Stop should complete within a reasonable time (all workers exit).
	done := make(chan struct{})
	go func() {
		q.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return within timeout; workers may be leaked")
	}
}

func TestQueueStopIdempotent(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	if err := q.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	cancel()

	// Calling Stop multiple times should not panic or deadlock.
	q.Stop()
	q.Stop()
}

func TestQueueManager_GetQueueConcurrent(t *testing.T) {
	store := newMockJobStore()
	cfg := fastConfig(1)
	mgr := NewQueueManager(store, cfg, testLogger())

	orgID := uuid.New()
	const goroutines = 20
	queues := make(chan *Queue, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			queues <- mgr.GetQueue(orgID)
		}()
	}
	wg.Wait()
	close(queues)

	// All goroutines should get the same queue instance.
	var first *Queue
	for q := range queues {
		if first == nil {
			first = q
		} else if q != first {
			t.Fatal("GetQueue returned different instances for the same orgID")
		}
	}
}

func TestQueueManager_StartStop(t *testing.T) {
	store := newMockJobStore()
	cfg := fastConfig(1)
	mgr := NewQueueManager(store, cfg, testLogger())

	orgID := uuid.New()
	q := mgr.GetQueue(orgID)
	q.RegisterHandler(models.JobTypeBackup, &mockHandler{
		handleFn: func(_ context.Context, _ *models.Job) (map[string]interface{}, error) {
			return nil, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Double start should error.
	if err := mgr.Start(ctx); err == nil {
		t.Fatal("expected error from double Start")
	}

	cancel()
	mgr.Stop()
}

func TestSummary(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	summary, err := q.Summary(context.Background())
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
}

func TestEnqueueBackup(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	agentID := uuid.New()
	repoID := uuid.New()
	schedID := uuid.New()

	job, err := q.EnqueueBackup(context.Background(), agentID, repoID, schedID, 5)
	if err != nil {
		t.Fatalf("enqueue backup: %v", err)
	}
	if job.JobType != models.JobTypeBackup {
		t.Errorf("expected backup type, got %s", job.JobType)
	}
	if job.Priority != 5 {
		t.Errorf("expected priority 5, got %d", job.Priority)
	}
	if job.OrgID != orgID {
		t.Errorf("expected org %s, got %s", orgID, job.OrgID)
	}
}

func TestEnqueueRestore(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	job, err := q.EnqueueRestore(context.Background(), uuid.New(), uuid.New(), "snap-123", "/tmp/restore")
	if err != nil {
		t.Fatalf("enqueue restore: %v", err)
	}
	if job.JobType != models.JobTypeRestore {
		t.Errorf("expected restore type, got %s", job.JobType)
	}
	if job.Payload.SnapshotID != "snap-123" {
		t.Errorf("expected snapshot snap-123, got %s", job.Payload.SnapshotID)
	}
}

func TestEnqueueVerification(t *testing.T) {
	orgID := uuid.New()
	store := newMockJobStore()
	q := NewQueue(store, orgID, fastConfig(1), testLogger())

	job, err := q.EnqueueVerification(context.Background(), uuid.New(), "full", 3)
	if err != nil {
		t.Fatalf("enqueue verification: %v", err)
	}
	if job.JobType != models.JobTypeVerification {
		t.Errorf("expected verification type, got %s", job.JobType)
	}
	if job.Payload.VerificationType != "full" {
		t.Errorf("expected full verification, got %s", job.Payload.VerificationType)
	}
}
