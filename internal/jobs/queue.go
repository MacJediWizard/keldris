// Package jobs provides a job queue system for managing background operations.
package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// JobStore defines the interface for job persistence operations.
type JobStore interface {
	CreateJob(ctx context.Context, job *models.Job) error
	GetJobByID(ctx context.Context, id uuid.UUID) (*models.Job, error)
	UpdateJob(ctx context.Context, job *models.Job) error
	GetNextPendingJob(ctx context.Context, orgID uuid.UUID) (*models.Job, error)
	ListJobsReadyForRetry(ctx context.Context, limit int) ([]*models.Job, error)
	GetJobQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.JobQueueSummary, error)
	CleanupOldJobs(ctx context.Context, retentionDays int) (int64, error)
}

// JobHandler processes jobs of a specific type.
type JobHandler interface {
	// Handle processes the job and returns a result map or error.
	Handle(ctx context.Context, job *models.Job) (map[string]interface{}, error)
}

// QueueConfig holds configuration for the job queue.
type QueueConfig struct {
	// WorkerCount is the number of concurrent workers per organization.
	WorkerCount int
	// PollInterval is how often to check for new jobs.
	PollInterval time.Duration
	// RetryPollInterval is how often to check for jobs ready to retry.
	RetryPollInterval time.Duration
	// CleanupInterval is how often to clean up old jobs.
	CleanupInterval time.Duration
	// JobRetentionDays is how long to keep completed/dead letter jobs.
	JobRetentionDays int
	// MaxJobDuration is the maximum time a job can run before timing out.
	MaxJobDuration time.Duration
}

// DefaultQueueConfig returns a QueueConfig with sensible defaults.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		WorkerCount:       3,
		PollInterval:      5 * time.Second,
		RetryPollInterval: 30 * time.Second,
		CleanupInterval:   1 * time.Hour,
		JobRetentionDays:  30,
		MaxJobDuration:    1 * time.Hour,
	}
}

// Queue manages job processing for an organization.
type Queue struct {
	store    JobStore
	config   QueueConfig
	handlers map[models.JobType]JobHandler
	logger   zerolog.Logger

	mu       sync.RWMutex
	orgID    uuid.UUID
	running  bool
	stopCh   chan struct{}
	workerWg sync.WaitGroup
}

// NewQueue creates a new job queue for the given organization.
func NewQueue(store JobStore, orgID uuid.UUID, config QueueConfig, logger zerolog.Logger) *Queue {
	return &Queue{
		store:    store,
		config:   config,
		handlers: make(map[models.JobType]JobHandler),
		logger:   logger.With().Str("component", "job_queue").Str("org_id", orgID.String()).Logger(),
		orgID:    orgID,
		stopCh:   make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a specific job type.
func (q *Queue) RegisterHandler(jobType models.JobType, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
	q.logger.Info().Str("job_type", string(jobType)).Msg("registered job handler")
}

// Enqueue adds a new job to the queue.
func (q *Queue) Enqueue(ctx context.Context, job *models.Job) error {
	if job.OrgID != q.orgID {
		return fmt.Errorf("job org_id does not match queue org_id")
	}

	if err := q.store.CreateJob(ctx, job); err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}

	q.logger.Info().
		Str("job_id", job.ID.String()).
		Str("job_type", string(job.JobType)).
		Int("priority", job.Priority).
		Msg("job enqueued")

	return nil
}

// EnqueueBackup creates and enqueues a backup job.
func (q *Queue) EnqueueBackup(ctx context.Context, agentID, repositoryID, scheduleID uuid.UUID, priority int) (*models.Job, error) {
	job := models.NewBackupJob(q.orgID, agentID, repositoryID, scheduleID, priority)
	if err := q.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// EnqueueRestore creates and enqueues a restore job.
func (q *Queue) EnqueueRestore(ctx context.Context, agentID, repositoryID uuid.UUID, snapshotID, targetPath string) (*models.Job, error) {
	job := models.NewRestoreJob(q.orgID, agentID, repositoryID, snapshotID, targetPath)
	if err := q.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// EnqueueVerification creates and enqueues a verification job.
func (q *Queue) EnqueueVerification(ctx context.Context, repositoryID uuid.UUID, verificationType string, priority int) (*models.Job, error) {
	job := models.NewVerificationJob(q.orgID, repositoryID, verificationType, priority)
	if err := q.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// Start begins processing jobs.
func (q *Queue) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return fmt.Errorf("queue already running")
	}
	q.running = true
	q.stopCh = make(chan struct{})
	q.mu.Unlock()

	q.logger.Info().Int("workers", q.config.WorkerCount).Msg("starting job queue")

	// Start workers
	for i := 0; i < q.config.WorkerCount; i++ {
		q.workerWg.Add(1)
		go q.worker(ctx, i)
	}

	// Start retry processor
	q.workerWg.Add(1)
	go q.retryProcessor(ctx)

	// Start cleanup processor
	q.workerWg.Add(1)
	go q.cleanupProcessor(ctx)

	return nil
}

// Stop gracefully stops the queue.
func (q *Queue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	close(q.stopCh)
	q.mu.Unlock()

	q.logger.Info().Msg("stopping job queue")
	q.workerWg.Wait()
	q.logger.Info().Msg("job queue stopped")
}

// Summary returns current queue statistics.
func (q *Queue) Summary(ctx context.Context) (*models.JobQueueSummary, error) {
	return q.store.GetJobQueueSummary(ctx, q.orgID)
}

// worker processes jobs from the queue.
func (q *Queue) worker(ctx context.Context, workerID int) {
	defer q.workerWg.Done()

	logger := q.logger.With().Int("worker_id", workerID).Logger()
	logger.Debug().Msg("worker started")

	ticker := time.NewTicker(q.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msg("worker stopping due to context cancellation")
			return
		case <-q.stopCh:
			logger.Debug().Msg("worker stopping due to stop signal")
			return
		case <-ticker.C:
			q.processNextJob(ctx, logger)
		}
	}
}

// processNextJob attempts to claim and process the next pending job.
func (q *Queue) processNextJob(ctx context.Context, logger zerolog.Logger) {
	job, err := q.store.GetNextPendingJob(ctx, q.orgID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get next pending job")
		return
	}

	if job == nil {
		return // No jobs available
	}

	logger = logger.With().
		Str("job_id", job.ID.String()).
		Str("job_type", string(job.JobType)).
		Logger()

	logger.Info().Msg("processing job")

	q.mu.RLock()
	handler, exists := q.handlers[job.JobType]
	q.mu.RUnlock()

	if !exists {
		logger.Error().Msg("no handler registered for job type")
		job.Fail("no handler registered for job type")
		if err := q.store.UpdateJob(ctx, job); err != nil {
			logger.Error().Err(err).Msg("failed to update job after handler error")
		}
		return
	}

	// Create a timeout context for the job
	jobCtx, cancel := context.WithTimeout(ctx, q.config.MaxJobDuration)
	defer cancel()

	// Process the job
	result, err := handler.Handle(jobCtx, job)
	if err != nil {
		shouldRetry := job.Fail(err.Error())
		if shouldRetry {
			logger.Warn().
				Err(err).
				Int("retry_count", job.RetryCount).
				Time("next_retry_at", *job.NextRetryAt).
				Msg("job failed, will retry")
		} else {
			logger.Error().
				Err(err).
				Int("retry_count", job.RetryCount).
				Msg("job failed, moved to dead letter queue")
		}
	} else {
		job.Complete(result)
		logger.Info().
			Dur("duration", job.Duration()).
			Msg("job completed successfully")
	}

	if err := q.store.UpdateJob(ctx, job); err != nil {
		logger.Error().Err(err).Msg("failed to update job after processing")
	}
}

// retryProcessor checks for jobs ready to retry and requeues them.
func (q *Queue) retryProcessor(ctx context.Context) {
	defer q.workerWg.Done()

	logger := q.logger.With().Str("processor", "retry").Logger()
	logger.Debug().Msg("retry processor started")

	ticker := time.NewTicker(q.config.RetryPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopCh:
			return
		case <-ticker.C:
			q.processRetries(ctx, logger)
		}
	}
}

// processRetries requeues jobs that are ready for retry.
func (q *Queue) processRetries(ctx context.Context, logger zerolog.Logger) {
	jobs, err := q.store.ListJobsReadyForRetry(ctx, 100)
	if err != nil {
		logger.Error().Err(err).Msg("failed to list jobs ready for retry")
		return
	}

	for _, job := range jobs {
		// Reset job to pending status
		job.Status = models.JobStatusPending
		job.StartedAt = nil

		if err := q.store.UpdateJob(ctx, job); err != nil {
			logger.Error().
				Err(err).
				Str("job_id", job.ID.String()).
				Msg("failed to requeue job for retry")
			continue
		}

		logger.Info().
			Str("job_id", job.ID.String()).
			Int("retry_count", job.RetryCount).
			Msg("job requeued for retry")
	}
}

// cleanupProcessor periodically cleans up old completed jobs.
func (q *Queue) cleanupProcessor(ctx context.Context) {
	defer q.workerWg.Done()

	logger := q.logger.With().Str("processor", "cleanup").Logger()
	logger.Debug().Msg("cleanup processor started")

	ticker := time.NewTicker(q.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.stopCh:
			return
		case <-ticker.C:
			deleted, err := q.store.CleanupOldJobs(ctx, q.config.JobRetentionDays)
			if err != nil {
				logger.Error().Err(err).Msg("failed to cleanup old jobs")
			} else if deleted > 0 {
				logger.Info().Int64("deleted", deleted).Msg("cleaned up old jobs")
			}
		}
	}
}

// QueueManager manages queues for multiple organizations.
type QueueManager struct {
	store   JobStore
	config  QueueConfig
	logger  zerolog.Logger

	mu      sync.RWMutex
	queues  map[uuid.UUID]*Queue
	running bool
}

// NewQueueManager creates a new queue manager.
func NewQueueManager(store JobStore, config QueueConfig, logger zerolog.Logger) *QueueManager {
	return &QueueManager{
		store:  store,
		config: config,
		logger: logger.With().Str("component", "queue_manager").Logger(),
		queues: make(map[uuid.UUID]*Queue),
	}
}

// GetQueue returns the queue for an organization, creating it if necessary.
func (m *QueueManager) GetQueue(orgID uuid.UUID) *Queue {
	m.mu.RLock()
	q, exists := m.queues[orgID]
	m.mu.RUnlock()

	if exists {
		return q
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if q, exists = m.queues[orgID]; exists {
		return q
	}

	q = NewQueue(m.store, orgID, m.config, m.logger)
	m.queues[orgID] = q
	return q
}

// RegisterHandler registers a handler for all queues.
func (m *QueueManager) RegisterHandler(jobType models.JobType, handler JobHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, q := range m.queues {
		q.RegisterHandler(jobType, handler)
	}
}

// Start starts all queues.
func (m *QueueManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("queue manager already running")
	}

	for orgID, q := range m.queues {
		if err := q.Start(ctx); err != nil {
			m.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to start queue")
		}
	}

	m.running = true
	return nil
}

// Stop stops all queues.
func (m *QueueManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, q := range m.queues {
		q.Stop()
	}

	m.running = false
}
