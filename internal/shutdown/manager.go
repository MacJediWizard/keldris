// Package shutdown provides graceful shutdown coordination for the Keldris server.
package shutdown

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// State represents the current shutdown state.
type State string

const (
	// StateRunning indicates the server is running normally.
	StateRunning State = "running"
	// StateDraining indicates the server is draining connections and not accepting new jobs.
	StateDraining State = "draining"
	// StateCheckpointing indicates the server is checkpointing in-progress backups.
	StateCheckpointing State = "checkpointing"
	// StateComplete indicates shutdown is complete.
	StateComplete State = "complete"
)

// BackupTracker provides methods to track and checkpoint running backups.
type BackupTracker interface {
	// GetRunningBackupIDs returns the IDs of all currently running backups.
	GetRunningBackupIDs() []uuid.UUID

	// CheckpointBackup saves a checkpoint for the given backup.
	// Returns true if the backup was successfully checkpointed.
	CheckpointBackup(ctx context.Context, backupID uuid.UUID) (bool, error)

	// IsBackupRunning checks if a backup is still running.
	IsBackupRunning(backupID uuid.UUID) bool

	// CancelBackup attempts to gracefully cancel a running backup.
	CancelBackup(ctx context.Context, backupID uuid.UUID) error
}

// Status represents the current shutdown status.
type Status struct {
	State              State         `json:"state"`
	StartedAt          *time.Time    `json:"started_at,omitempty"`
	TimeRemaining      time.Duration `json:"time_remaining,omitempty"`
	RunningBackups     int           `json:"running_backups"`
	CheckpointedCount  int           `json:"checkpointed_count"`
	AcceptingNewJobs   bool          `json:"accepting_new_jobs"`
	Message            string        `json:"message,omitempty"`
}

// Config holds configuration for the shutdown manager.
type Config struct {
	// Timeout is the maximum time to wait for graceful shutdown.
	Timeout time.Duration

	// DrainTimeout is the time to wait for existing connections to drain.
	DrainTimeout time.Duration

	// CheckpointRunningBackups determines whether to checkpoint running backups.
	CheckpointRunningBackups bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:                  30 * time.Second,
		DrainTimeout:             5 * time.Second,
		CheckpointRunningBackups: true,
	}
}

// Manager coordinates graceful shutdown of the Keldris server.
type Manager struct {
	config        Config
	tracker       BackupTracker
	logger        zerolog.Logger
	mu            sync.RWMutex
	state         State
	startedAt     *time.Time
	checkpointed  int32
	acceptingJobs atomic.Bool
	doneCh        chan struct{}
	shutdownOnce  sync.Once
}

// NewManager creates a new shutdown manager.
func NewManager(config Config, tracker BackupTracker, logger zerolog.Logger) *Manager {
	m := &Manager{
		config:  config,
		tracker: tracker,
		logger:  logger.With().Str("component", "shutdown_manager").Logger(),
		state:   StateRunning,
		doneCh:  make(chan struct{}),
	}
	m.acceptingJobs.Store(true)
	return m
}

// IsAcceptingJobs returns true if the server is accepting new backup jobs.
func (m *Manager) IsAcceptingJobs() bool {
	return m.acceptingJobs.Load()
}

// GetState returns the current shutdown state.
func (m *Manager) GetState() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// GetStatus returns the current shutdown status.
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := Status{
		State:             m.state,
		StartedAt:         m.startedAt,
		CheckpointedCount: int(atomic.LoadInt32(&m.checkpointed)),
		AcceptingNewJobs:  m.acceptingJobs.Load(),
	}

	if m.tracker != nil {
		status.RunningBackups = len(m.tracker.GetRunningBackupIDs())
	}

	if m.startedAt != nil {
		elapsed := time.Since(*m.startedAt)
		remaining := m.config.Timeout - elapsed
		if remaining > 0 {
			status.TimeRemaining = remaining
		}
	}

	switch m.state {
	case StateRunning:
		status.Message = "Server is running normally"
	case StateDraining:
		status.Message = "Server is draining connections, not accepting new jobs"
	case StateCheckpointing:
		status.Message = "Checkpointing in-progress backups"
	case StateComplete:
		status.Message = "Shutdown complete"
	}

	return status
}

// Shutdown initiates graceful shutdown and blocks until complete or timeout.
// Returns a context that will be cancelled when shutdown is complete.
func (m *Manager) Shutdown(ctx context.Context) error {
	var shutdownErr error
	m.shutdownOnce.Do(func() {
		shutdownErr = m.doShutdown(ctx)
	})
	return shutdownErr
}

func (m *Manager) doShutdown(ctx context.Context) error {
	m.logger.Info().
		Dur("timeout", m.config.Timeout).
		Dur("drain_timeout", m.config.DrainTimeout).
		Bool("checkpoint_backups", m.config.CheckpointRunningBackups).
		Msg("initiating graceful shutdown")

	now := time.Now()
	m.mu.Lock()
	m.startedAt = &now
	m.state = StateDraining
	m.mu.Unlock()

	// Stop accepting new jobs immediately
	m.acceptingJobs.Store(false)
	m.logger.Info().Msg("stopped accepting new backup jobs")

	// Reserve 20% of the remaining time after drain for checkpointing
	remainingAfterDrain := m.config.Timeout - m.config.DrainTimeout
	checkpointReserve := remainingAfterDrain / 5
	// Ensure reasonable bounds - minimum 1 second, maximum 30 seconds or half the remaining time
	if checkpointReserve < 1*time.Second {
		checkpointReserve = 1 * time.Second
	}
	if checkpointReserve > 30*time.Second {
		checkpointReserve = 30 * time.Second
	}
	if checkpointReserve > remainingAfterDrain/2 {
		checkpointReserve = remainingAfterDrain / 2
	}
	waitTimeout := remainingAfterDrain - checkpointReserve
	if waitTimeout < 0 {
		waitTimeout = 0
	}

	// Phase 1: Drain existing connections
	m.logger.Info().Dur("drain_timeout", m.config.DrainTimeout).Msg("draining connections")
	drainCtx, drainCancel := context.WithTimeout(ctx, m.config.DrainTimeout)
	<-drainCtx.Done()
	drainCancel()
	m.logger.Debug().Msg("drain timeout reached")

	// Check if parent context is cancelled
	if ctx.Err() != nil {
		m.logger.Warn().Msg("shutdown cancelled during drain phase")
		return m.forceShutdown()
	}

	// Phase 2: Wait for running backups with periodic status logging
	if m.tracker != nil {
		waitCtx, waitCancel := context.WithTimeout(ctx, waitTimeout)
		_ = m.waitForBackups(waitCtx)
		waitCancel()
	}

	// Check if parent context is cancelled
	if ctx.Err() != nil {
		m.logger.Warn().Msg("shutdown cancelled during wait phase")
		return m.forceShutdown()
	}

	// Phase 3: Checkpoint any remaining in-progress backups
	if m.config.CheckpointRunningBackups && m.tracker != nil {
		m.mu.Lock()
		m.state = StateCheckpointing
		m.mu.Unlock()

		checkpointCtx, checkpointCancel := context.WithTimeout(ctx, checkpointReserve)
		if err := m.checkpointBackups(checkpointCtx); err != nil {
			m.logger.Warn().Err(err).Msg("error during checkpoint phase, continuing shutdown")
		}
		checkpointCancel()
	}

	// Complete shutdown
	m.mu.Lock()
	m.state = StateComplete
	m.mu.Unlock()
	close(m.doneCh)

	duration := time.Since(now)
	m.logger.Info().
		Dur("duration", duration).
		Int("checkpointed", int(atomic.LoadInt32(&m.checkpointed))).
		Msg("graceful shutdown complete")

	return nil
}

// waitForBackups waits for running backups to complete.
func (m *Manager) waitForBackups(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		runningIDs := m.tracker.GetRunningBackupIDs()
		if len(runningIDs) == 0 {
			m.logger.Info().Msg("all backups completed")
			return nil
		}

		m.logger.Info().
			Int("running_backups", len(runningIDs)).
			Msg("waiting for running backups to complete")

		select {
		case <-ctx.Done():
			m.logger.Warn().
				Int("running_backups", len(runningIDs)).
				Msg("shutdown timeout reached with backups still running")
			return nil // Continue to checkpoint phase
		case <-ticker.C:
			continue
		}
	}
}

// checkpointBackups checkpoints all running backups.
func (m *Manager) checkpointBackups(ctx context.Context) error {
	if m.tracker == nil {
		return nil
	}

	runningIDs := m.tracker.GetRunningBackupIDs()
	if len(runningIDs) == 0 {
		m.logger.Debug().Msg("no running backups to checkpoint")
		return nil
	}

	m.logger.Info().
		Int("count", len(runningIDs)).
		Msg("checkpointing running backups")

	var wg sync.WaitGroup
	for _, backupID := range runningIDs {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()

			logger := m.logger.With().Str("backup_id", id.String()).Logger()
			logger.Debug().Msg("checkpointing backup")

			success, err := m.tracker.CheckpointBackup(ctx, id)
			if err != nil {
				logger.Error().Err(err).Msg("failed to checkpoint backup")
				return
			}

			if success {
				atomic.AddInt32(&m.checkpointed, 1)
				logger.Info().Msg("backup checkpointed successfully")
			} else {
				logger.Warn().Msg("backup could not be checkpointed")
			}
		}(backupID)
	}

	// Wait for all checkpoints to complete or context to cancel
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info().
			Int("checkpointed", int(atomic.LoadInt32(&m.checkpointed))).
			Int("total", len(runningIDs)).
			Msg("checkpoint phase complete")
	case <-ctx.Done():
		m.logger.Warn().Msg("checkpoint phase timed out")
	}

	return nil
}

// forceShutdown performs immediate shutdown without waiting.
func (m *Manager) forceShutdown() error {
	m.logger.Warn().Msg("forcing immediate shutdown")

	m.mu.Lock()
	m.state = StateComplete
	m.mu.Unlock()
	close(m.doneCh)

	return nil
}

// Done returns a channel that is closed when shutdown is complete.
func (m *Manager) Done() <-chan struct{} {
	return m.doneCh
}

// WaitForShutdown blocks until shutdown is complete.
func (m *Manager) WaitForShutdown() {
	<-m.doneCh
}
