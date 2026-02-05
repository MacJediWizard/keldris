package shutdown

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// CheckpointInfo contains information about a checkpoint available for resume.
type CheckpointInfo struct {
	ID             uuid.UUID
	ScheduleID     uuid.UUID
	AgentID        uuid.UUID
	FilesProcessed int64
	BytesProcessed int64
	LastUpdatedAt  time.Time
	ErrorMessage   string
	ResumeCount    int
}

// CheckpointStore defines the interface for accessing checkpoints.
type CheckpointStore interface {
	// GetActiveCheckpoints returns all active checkpoints that can be resumed.
	GetActiveCheckpoints(ctx context.Context) ([]CheckpointInfo, error)
}

// BackupResumer defines the interface for resuming backups.
type BackupResumer interface {
	// ResumeBackup attempts to resume a backup from the given checkpoint.
	ResumeBackup(ctx context.Context, checkpointID uuid.UUID) error
}

// StartupConfig holds configuration for the startup service.
type StartupConfig struct {
	// ResumeCheckpoints determines whether to resume checkpointed backups on startup.
	ResumeCheckpoints bool

	// MaxResumesPerStartup limits how many backups can be resumed at once.
	MaxResumesPerStartup int

	// ResumeDelay is the delay between resuming individual backups.
	ResumeDelay time.Duration
}

// DefaultStartupConfig returns sensible defaults for startup configuration.
func DefaultStartupConfig() StartupConfig {
	return StartupConfig{
		ResumeCheckpoints:    true,
		MaxResumesPerStartup: 5,
		ResumeDelay:          5 * time.Second,
	}
}

// StartupService handles startup tasks like resuming checkpointed backups.
type StartupService struct {
	config  StartupConfig
	store   CheckpointStore
	resumer BackupResumer
	logger  zerolog.Logger
}

// NewStartupService creates a new startup service.
func NewStartupService(config StartupConfig, store CheckpointStore, resumer BackupResumer, logger zerolog.Logger) *StartupService {
	return &StartupService{
		config:  config,
		store:   store,
		resumer: resumer,
		logger:  logger.With().Str("component", "startup_service").Logger(),
	}
}

// ResumeCheckpointedBackups attempts to resume all eligible checkpointed backups.
// Returns the number of backups that were successfully queued for resume.
func (s *StartupService) ResumeCheckpointedBackups(ctx context.Context) (int, error) {
	if !s.config.ResumeCheckpoints {
		s.logger.Info().Msg("checkpoint resume is disabled, skipping")
		return 0, nil
	}

	if s.store == nil || s.resumer == nil {
		s.logger.Warn().Msg("checkpoint store or resumer not configured, skipping resume")
		return 0, nil
	}

	s.logger.Info().Msg("checking for checkpointed backups to resume")

	checkpoints, err := s.store.GetActiveCheckpoints(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get active checkpoints")
		return 0, err
	}

	if len(checkpoints) == 0 {
		s.logger.Info().Msg("no checkpointed backups found to resume")
		return 0, nil
	}

	s.logger.Info().
		Int("count", len(checkpoints)).
		Msg("found checkpointed backups to resume")

	resumed := 0
	for i, cp := range checkpoints {
		if s.config.MaxResumesPerStartup > 0 && resumed >= s.config.MaxResumesPerStartup {
			s.logger.Warn().
				Int("remaining", len(checkpoints)-i).
				Int("max", s.config.MaxResumesPerStartup).
				Msg("max resume limit reached, remaining backups will resume on next startup or manually")
			break
		}

		logger := s.logger.With().
			Str("checkpoint_id", cp.ID.String()).
			Str("schedule_id", cp.ScheduleID.String()).
			Int("resume_count", cp.ResumeCount).
			Logger()

		logger.Info().Msg("attempting to resume checkpointed backup")

		if err := s.resumer.ResumeBackup(ctx, cp.ID); err != nil {
			logger.Error().Err(err).Msg("failed to resume backup")
			continue
		}

		logger.Info().Msg("backup resume initiated")
		resumed++

		// Delay between resumes to avoid overwhelming the system
		if i < len(checkpoints)-1 && s.config.ResumeDelay > 0 {
			select {
			case <-ctx.Done():
				s.logger.Warn().Msg("context cancelled during resume delay")
				return resumed, ctx.Err()
			case <-time.After(s.config.ResumeDelay):
			}
		}
	}

	s.logger.Info().
		Int("resumed", resumed).
		Int("total", len(checkpoints)).
		Msg("checkpoint resume complete")

	return resumed, nil
}

// GetPendingCheckpoints returns information about checkpoints that can be resumed.
func (s *StartupService) GetPendingCheckpoints(ctx context.Context) ([]CheckpointInfo, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetActiveCheckpoints(ctx)
}
