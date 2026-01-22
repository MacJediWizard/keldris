package backup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// CheckpointStore defines the interface for checkpoint persistence.
type CheckpointStore interface {
	// CreateCheckpoint creates a new backup checkpoint.
	CreateCheckpoint(ctx context.Context, checkpoint *models.BackupCheckpoint) error

	// UpdateCheckpoint updates an existing checkpoint.
	UpdateCheckpoint(ctx context.Context, checkpoint *models.BackupCheckpoint) error

	// GetCheckpointByID returns a checkpoint by ID.
	GetCheckpointByID(ctx context.Context, id uuid.UUID) (*models.BackupCheckpoint, error)

	// GetActiveCheckpointForSchedule returns the active checkpoint for a schedule if one exists.
	GetActiveCheckpointForSchedule(ctx context.Context, scheduleID uuid.UUID) (*models.BackupCheckpoint, error)

	// GetActiveCheckpointsForAgent returns all active checkpoints for an agent.
	GetActiveCheckpointsForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupCheckpoint, error)

	// GetExpiredCheckpoints returns all checkpoints that have expired.
	GetExpiredCheckpoints(ctx context.Context) ([]*models.BackupCheckpoint, error)

	// DeleteCheckpoint deletes a checkpoint by ID.
	DeleteCheckpoint(ctx context.Context, id uuid.UUID) error
}

// CheckpointConfig holds configuration for checkpoint management.
type CheckpointConfig struct {
	// SaveInterval is how often to save checkpoint progress during a backup.
	SaveInterval time.Duration

	// ExpirationDuration is how long checkpoints remain valid for resume.
	ExpirationDuration time.Duration

	// CleanupInterval is how often to clean up expired checkpoints.
	CleanupInterval time.Duration

	// MaxResumeAttempts is the maximum number of times a backup can be resumed.
	MaxResumeAttempts int
}

// DefaultCheckpointConfig returns sensible defaults for checkpoint configuration.
func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		SaveInterval:       30 * time.Second,
		ExpirationDuration: 7 * 24 * time.Hour, // 7 days
		CleanupInterval:    1 * time.Hour,
		MaxResumeAttempts:  5,
	}
}

// CheckpointManager manages backup checkpoints for resumable backups.
type CheckpointManager struct {
	store   CheckpointStore
	config  CheckpointConfig
	logger  zerolog.Logger
	mu      sync.RWMutex
	active  map[uuid.UUID]*activeCheckpoint // keyed by backup ID
	running bool
}

// activeCheckpoint tracks an in-progress checkpoint being updated.
type activeCheckpoint struct {
	checkpoint *models.BackupCheckpoint
	lastSave   time.Time
	mu         sync.Mutex
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(store CheckpointStore, config CheckpointConfig, logger zerolog.Logger) *CheckpointManager {
	return &CheckpointManager{
		store:  store,
		config: config,
		logger: logger.With().Str("component", "checkpoint_manager").Logger(),
		active: make(map[uuid.UUID]*activeCheckpoint),
	}
}

// Start starts the checkpoint manager background tasks.
func (m *CheckpointManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("checkpoint manager already running")
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info().Msg("starting checkpoint manager")

	// Start cleanup goroutine
	go m.cleanupLoop(ctx)

	return nil
}

// Stop stops the checkpoint manager.
func (m *CheckpointManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.logger.Info().Msg("stopping checkpoint manager")

	// Save all active checkpoints before stopping
	for backupID, ac := range m.active {
		ac.mu.Lock()
		if err := m.store.UpdateCheckpoint(context.Background(), ac.checkpoint); err != nil {
			m.logger.Error().Err(err).
				Str("backup_id", backupID.String()).
				Msg("failed to save checkpoint on shutdown")
		}
		ac.mu.Unlock()
	}
}

// StartCheckpoint creates a new checkpoint for a backup.
func (m *CheckpointManager) StartCheckpoint(ctx context.Context, scheduleID, agentID, repositoryID uuid.UUID) (*models.BackupCheckpoint, error) {
	checkpoint := models.NewBackupCheckpoint(scheduleID, agentID, repositoryID)

	if err := m.store.CreateCheckpoint(ctx, checkpoint); err != nil {
		return nil, fmt.Errorf("create checkpoint: %w", err)
	}

	m.logger.Debug().
		Str("checkpoint_id", checkpoint.ID.String()).
		Str("schedule_id", scheduleID.String()).
		Msg("checkpoint created")

	return checkpoint, nil
}

// TrackBackup starts tracking progress for a backup with a checkpoint.
func (m *CheckpointManager) TrackBackup(backupID uuid.UUID, checkpoint *models.BackupCheckpoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.active[backupID] = &activeCheckpoint{
		checkpoint: checkpoint,
		lastSave:   time.Now(),
	}

	m.logger.Debug().
		Str("backup_id", backupID.String()).
		Str("checkpoint_id", checkpoint.ID.String()).
		Msg("started tracking backup progress")
}

// UpdateProgress updates the progress of a tracked backup.
// Returns true if the checkpoint was saved to the database.
func (m *CheckpointManager) UpdateProgress(ctx context.Context, backupID uuid.UUID, filesProcessed, bytesProcessed int64, lastPath string) (bool, error) {
	m.mu.RLock()
	ac, exists := m.active[backupID]
	m.mu.RUnlock()

	if !exists {
		return false, nil
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.checkpoint.UpdateProgress(filesProcessed, bytesProcessed, lastPath)

	// Only save if enough time has passed since last save
	if time.Since(ac.lastSave) >= m.config.SaveInterval {
		if err := m.store.UpdateCheckpoint(ctx, ac.checkpoint); err != nil {
			return false, fmt.Errorf("save checkpoint: %w", err)
		}
		ac.lastSave = time.Now()
		return true, nil
	}

	return false, nil
}

// SetTotals sets the estimated totals for a tracked backup.
func (m *CheckpointManager) SetTotals(ctx context.Context, backupID uuid.UUID, totalFiles, totalBytes int64) error {
	m.mu.RLock()
	ac, exists := m.active[backupID]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.checkpoint.SetTotals(totalFiles, totalBytes)

	if err := m.store.UpdateCheckpoint(ctx, ac.checkpoint); err != nil {
		return fmt.Errorf("save checkpoint totals: %w", err)
	}

	return nil
}

// AssociateBackup associates a backup record with a checkpoint.
func (m *CheckpointManager) AssociateBackup(ctx context.Context, checkpointID, backupID uuid.UUID) error {
	checkpoint, err := m.store.GetCheckpointByID(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("get checkpoint: %w", err)
	}

	checkpoint.SetBackupID(backupID)

	if err := m.store.UpdateCheckpoint(ctx, checkpoint); err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}

	return nil
}

// CompleteBackup marks a backup as completed and stops tracking.
func (m *CheckpointManager) CompleteBackup(ctx context.Context, backupID uuid.UUID) error {
	m.mu.Lock()
	ac, exists := m.active[backupID]
	if exists {
		delete(m.active, backupID)
	}
	m.mu.Unlock()

	if !exists {
		return nil
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.checkpoint.MarkCompleted()

	if err := m.store.UpdateCheckpoint(ctx, ac.checkpoint); err != nil {
		return fmt.Errorf("complete checkpoint: %w", err)
	}

	m.logger.Debug().
		Str("backup_id", backupID.String()).
		Str("checkpoint_id", ac.checkpoint.ID.String()).
		Msg("backup completed, checkpoint marked as completed")

	return nil
}

// InterruptBackup marks a backup as interrupted with an error.
func (m *CheckpointManager) InterruptBackup(ctx context.Context, backupID uuid.UUID, errMsg string) error {
	m.mu.Lock()
	ac, exists := m.active[backupID]
	if exists {
		delete(m.active, backupID)
	}
	m.mu.Unlock()

	if !exists {
		return nil
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.checkpoint.MarkInterrupted(errMsg)

	if err := m.store.UpdateCheckpoint(ctx, ac.checkpoint); err != nil {
		return fmt.Errorf("save interrupted checkpoint: %w", err)
	}

	m.logger.Info().
		Str("backup_id", backupID.String()).
		Str("checkpoint_id", ac.checkpoint.ID.String()).
		Str("error", errMsg).
		Int64("files_processed", ac.checkpoint.FilesProcessed).
		Int64("bytes_processed", ac.checkpoint.BytesProcessed).
		Msg("backup interrupted, checkpoint saved for potential resume")

	return nil
}

// CancelCheckpoint cancels a checkpoint, making it non-resumable.
func (m *CheckpointManager) CancelCheckpoint(ctx context.Context, checkpointID uuid.UUID) error {
	checkpoint, err := m.store.GetCheckpointByID(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("get checkpoint: %w", err)
	}

	checkpoint.MarkCanceled()

	if err := m.store.UpdateCheckpoint(ctx, checkpoint); err != nil {
		return fmt.Errorf("cancel checkpoint: %w", err)
	}

	// Also remove from active tracking if present
	m.mu.Lock()
	for backupID, ac := range m.active {
		if ac.checkpoint.ID == checkpointID {
			delete(m.active, backupID)
			break
		}
	}
	m.mu.Unlock()

	m.logger.Info().
		Str("checkpoint_id", checkpointID.String()).
		Msg("checkpoint canceled")

	return nil
}

// GetResumableCheckpoint returns a resumable checkpoint for a schedule if one exists.
func (m *CheckpointManager) GetResumableCheckpoint(ctx context.Context, scheduleID uuid.UUID) (*models.BackupCheckpoint, error) {
	checkpoint, err := m.store.GetActiveCheckpointForSchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("get active checkpoint: %w", err)
	}

	if checkpoint == nil {
		return nil, nil
	}

	if !checkpoint.IsResumable() {
		return nil, nil
	}

	if checkpoint.ResumeCount >= m.config.MaxResumeAttempts {
		m.logger.Warn().
			Str("checkpoint_id", checkpoint.ID.String()).
			Int("resume_count", checkpoint.ResumeCount).
			Int("max_attempts", m.config.MaxResumeAttempts).
			Msg("checkpoint exceeded max resume attempts")
		return nil, nil
	}

	return checkpoint, nil
}

// PrepareResume prepares a checkpoint for resumption.
func (m *CheckpointManager) PrepareResume(ctx context.Context, checkpoint *models.BackupCheckpoint) error {
	checkpoint.IncrementResumeCount()

	if err := m.store.UpdateCheckpoint(ctx, checkpoint); err != nil {
		return fmt.Errorf("update checkpoint for resume: %w", err)
	}

	m.logger.Info().
		Str("checkpoint_id", checkpoint.ID.String()).
		Int("resume_count", checkpoint.ResumeCount).
		Int64("files_processed", checkpoint.FilesProcessed).
		Int64("bytes_processed", checkpoint.BytesProcessed).
		Msg("preparing to resume backup from checkpoint")

	return nil
}

// GetIncompleteBackups returns all incomplete backups for an agent that can be resumed.
func (m *CheckpointManager) GetIncompleteBackups(ctx context.Context, agentID uuid.UUID) ([]*models.BackupCheckpoint, error) {
	checkpoints, err := m.store.GetActiveCheckpointsForAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get active checkpoints: %w", err)
	}

	var resumable []*models.BackupCheckpoint
	for _, cp := range checkpoints {
		if cp.IsResumable() && cp.ResumeCount < m.config.MaxResumeAttempts {
			resumable = append(resumable, cp)
		}
	}

	return resumable, nil
}

// cleanupLoop periodically cleans up expired checkpoints.
func (m *CheckpointManager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			running := m.running
			m.mu.RUnlock()

			if !running {
				return
			}

			if err := m.cleanupExpired(ctx); err != nil {
				m.logger.Error().Err(err).Msg("failed to cleanup expired checkpoints")
			}
		}
	}
}

// cleanupExpired marks expired checkpoints as expired.
func (m *CheckpointManager) cleanupExpired(ctx context.Context) error {
	expired, err := m.store.GetExpiredCheckpoints(ctx)
	if err != nil {
		return fmt.Errorf("get expired checkpoints: %w", err)
	}

	for _, cp := range expired {
		cp.MarkExpired()
		if err := m.store.UpdateCheckpoint(ctx, cp); err != nil {
			m.logger.Error().Err(err).
				Str("checkpoint_id", cp.ID.String()).
				Msg("failed to mark checkpoint as expired")
			continue
		}

		m.logger.Debug().
			Str("checkpoint_id", cp.ID.String()).
			Msg("checkpoint marked as expired")
	}

	if len(expired) > 0 {
		m.logger.Info().
			Int("count", len(expired)).
			Msg("expired checkpoints cleaned up")
	}

	return nil
}

// ResumeDecision represents the user's choice for handling an incomplete backup.
type ResumeDecision string

const (
	// ResumeDecisionResume indicates the user wants to resume the interrupted backup.
	ResumeDecisionResume ResumeDecision = "resume"
	// ResumeDecisionRestart indicates the user wants to restart the backup from scratch.
	ResumeDecisionRestart ResumeDecision = "restart"
	// ResumeDecisionSkip indicates the user wants to skip the backup for now.
	ResumeDecisionSkip ResumeDecision = "skip"
)

// ResumeInfo contains information about a resumable backup.
type ResumeInfo struct {
	Checkpoint       *models.BackupCheckpoint
	ProgressPercent  *float64
	FilesProcessed   int64
	BytesProcessed   int64
	TotalFiles       *int64
	TotalBytes       *int64
	InterruptedAt    time.Time
	InterruptedError string
	ResumeCount      int
	CanResume        bool
}

// GetResumeInfo returns information about a resumable backup.
func (m *CheckpointManager) GetResumeInfo(ctx context.Context, checkpointID uuid.UUID) (*ResumeInfo, error) {
	checkpoint, err := m.store.GetCheckpointByID(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("get checkpoint: %w", err)
	}

	info := &ResumeInfo{
		Checkpoint:       checkpoint,
		ProgressPercent:  checkpoint.ProgressPercent(),
		FilesProcessed:   checkpoint.FilesProcessed,
		BytesProcessed:   checkpoint.BytesProcessed,
		TotalFiles:       checkpoint.TotalFiles,
		TotalBytes:       checkpoint.TotalBytes,
		InterruptedAt:    checkpoint.LastUpdatedAt,
		InterruptedError: checkpoint.ErrorMessage,
		ResumeCount:      checkpoint.ResumeCount,
		CanResume:        checkpoint.IsResumable() && checkpoint.ResumeCount < m.config.MaxResumeAttempts,
	}

	return info, nil
}
