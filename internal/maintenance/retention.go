package maintenance

import (
	"context"
	"errors"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// RetentionStore defines the interface for retention cleanup data access.
type RetentionStore interface {
	CleanupAgentHealthHistory(ctx context.Context, retentionDays int) (int64, error)
}

// RetentionScheduler runs periodic cleanup of old health history records.
type RetentionScheduler struct {
	store         RetentionStore
	retentionDays int
	cron          *cron.Cron
	logger        zerolog.Logger
	mu            sync.Mutex
	running       bool
}

// NewRetentionScheduler creates a new retention cleanup scheduler.
func NewRetentionScheduler(store RetentionStore, retentionDays int, logger zerolog.Logger) *RetentionScheduler {
	return &RetentionScheduler{
		store:         store,
		retentionDays: retentionDays,
		cron:          cron.New(),
		logger:        logger.With().Str("component", "retention").Logger(),
	}
}

// Start begins the daily retention cleanup schedule at 3:00 AM UTC.
func (s *RetentionScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("retention scheduler already running")
	}

	// Schedule cleanup daily at 3:00 AM UTC
	_, err := s.cron.AddFunc("0 3 * * *", s.runCleanup)
	if err != nil {
		return err
	}

	s.cron.Start()
	s.running = true

	s.logger.Info().
		Int("retention_days", s.retentionDays).
		Msg("retention scheduler started (daily at 03:00 UTC)")

	return nil
}

// Stop stops the retention scheduler gracefully.
func (s *RetentionScheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping retention scheduler")
	return s.cron.Stop()
}

// runCleanup executes the health history cleanup.
func (s *RetentionScheduler) runCleanup() {
	ctx := context.Background()

	s.logger.Info().
		Int("retention_days", s.retentionDays).
		Msg("starting health history cleanup")

	deleted, err := s.store.CleanupAgentHealthHistory(ctx, s.retentionDays)
	if err != nil {
		s.logger.Error().Err(err).Msg("health history cleanup failed")
		return
	}

	s.logger.Info().
		Int64("deleted_rows", deleted).
		Int("retention_days", s.retentionDays).
		Msg("health history cleanup completed")
}

// RunNow triggers an immediate cleanup (useful for testing).
func (s *RetentionScheduler) RunNow() {
	s.runCleanup()
}
