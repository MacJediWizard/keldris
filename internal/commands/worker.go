// Package commands provides command queue management for agents.
package commands

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// Store defines the interface for command persistence operations.
type Store interface {
	MarkTimedOutCommands(ctx context.Context) (int64, error)
}

// TimeoutWorker handles marking commands as timed out.
type TimeoutWorker struct {
	store    Store
	interval time.Duration
	logger   zerolog.Logger
}

// NewTimeoutWorker creates a new TimeoutWorker.
func NewTimeoutWorker(store Store, interval time.Duration, logger zerolog.Logger) *TimeoutWorker {
	return &TimeoutWorker{
		store:    store,
		interval: interval,
		logger:   logger.With().Str("component", "command_timeout_worker").Logger(),
	}
}

// Start begins the timeout worker. It runs until the context is canceled.
func (w *TimeoutWorker) Start(ctx context.Context) {
	w.logger.Info().Dur("interval", w.interval).Msg("starting command timeout worker")

	// Run immediately on startup
	w.processTimeouts(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("stopping command timeout worker")
			return
		case <-ticker.C:
			w.processTimeouts(ctx)
		}
	}
}

// processTimeouts marks any commands that have exceeded their timeout.
func (w *TimeoutWorker) processTimeouts(ctx context.Context) {
	count, err := w.store.MarkTimedOutCommands(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to process command timeouts")
		return
	}

	if count > 0 {
		w.logger.Info().Int64("count", count).Msg("marked commands as timed out")
	}
}

// DefaultTimeoutCheckInterval is the default interval for checking command timeouts.
const DefaultTimeoutCheckInterval = 30 * time.Second
