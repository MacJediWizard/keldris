package shutdown

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// CheckpointFunc is a function that checkpoints a backup by ID.
type CheckpointFunc func(ctx context.Context, backupID uuid.UUID) error

// CancelFunc is a function that cancels a running backup by ID.
type CancelFunc func(ctx context.Context, backupID uuid.UUID) error

// SchedulerBackupTracker implements BackupTracker for the backup scheduler.
type SchedulerBackupTracker struct {
	checkpointFn CheckpointFunc
	cancelFn     CancelFunc
	logger       zerolog.Logger
	mu           sync.RWMutex
	running      map[uuid.UUID]struct{}
}

// NewSchedulerBackupTracker creates a new backup tracker for the scheduler.
func NewSchedulerBackupTracker(checkpointFn CheckpointFunc, cancelFn CancelFunc, logger zerolog.Logger) *SchedulerBackupTracker {
	return &SchedulerBackupTracker{
		checkpointFn: checkpointFn,
		cancelFn:     cancelFn,
		logger:       logger.With().Str("component", "scheduler_backup_tracker").Logger(),
		running:      make(map[uuid.UUID]struct{}),
	}
}

// RegisterBackup registers a backup as running.
func (t *SchedulerBackupTracker) RegisterBackup(backupID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running[backupID] = struct{}{}
	t.logger.Debug().Str("backup_id", backupID.String()).Msg("backup registered")
}

// UnregisterBackup removes a backup from the running set.
func (t *SchedulerBackupTracker) UnregisterBackup(backupID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.running, backupID)
	t.logger.Debug().Str("backup_id", backupID.String()).Msg("backup unregistered")
}

// GetRunningBackupIDs returns the IDs of all currently running backups.
func (t *SchedulerBackupTracker) GetRunningBackupIDs() []uuid.UUID {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(t.running))
	for id := range t.running {
		ids = append(ids, id)
	}
	return ids
}

// CheckpointBackup saves a checkpoint for the given backup.
func (t *SchedulerBackupTracker) CheckpointBackup(ctx context.Context, backupID uuid.UUID) (bool, error) {
	if t.checkpointFn == nil {
		return false, nil
	}

	err := t.checkpointFn(ctx, backupID)
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsBackupRunning checks if a backup is still running.
func (t *SchedulerBackupTracker) IsBackupRunning(backupID uuid.UUID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, running := t.running[backupID]
	return running
}

// CancelBackup attempts to gracefully cancel a running backup.
func (t *SchedulerBackupTracker) CancelBackup(ctx context.Context, backupID uuid.UUID) error {
	if t.cancelFn == nil {
		return nil
	}
	return t.cancelFn(ctx, backupID)
}

// RunningCount returns the number of currently running backups.
func (t *SchedulerBackupTracker) RunningCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.running)
}
