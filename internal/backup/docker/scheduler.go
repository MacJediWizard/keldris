// Package docker provides Docker image backup scheduling functionality.
package docker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// ImageBackupStore defines the interface for image backup persistence.
type ImageBackupStore interface {
	// GetEnabledDockerImageSchedules returns all enabled Docker image backup schedules.
	GetEnabledDockerImageSchedules(ctx context.Context) ([]*models.DockerImageSchedule, error)

	// GetDockerImageScheduleByID returns a schedule by ID.
	GetDockerImageScheduleByID(ctx context.Context, id uuid.UUID) (*models.DockerImageSchedule, error)

	// CreateDockerImageBackup creates a new backup record.
	CreateDockerImageBackup(ctx context.Context, backup *models.DockerImageBackupJob) error

	// UpdateDockerImageBackup updates an existing backup record.
	UpdateDockerImageBackup(ctx context.Context, backup *models.DockerImageBackupJob) error

	// CreateDockerImageVersion creates a new image version record.
	CreateDockerImageVersion(ctx context.Context, version *models.DockerImageVersion) error

	// GetDeduplicationEntry returns an existing deduplication entry for an image.
	GetDeduplicationEntry(ctx context.Context, orgID uuid.UUID, imageID string) (*models.DockerImageDeduplicationEntry, error)

	// CreateDeduplicationEntry creates a new deduplication entry.
	CreateDeduplicationEntry(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error

	// UpdateDeduplicationEntry updates a deduplication entry.
	UpdateDeduplicationEntry(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error

	// GetAgentByID returns an agent by ID.
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// ImageBackupSchedulerConfig holds configuration for the image backup scheduler.
type ImageBackupSchedulerConfig struct {
	// RefreshInterval is how often to reload schedules from the database.
	RefreshInterval time.Duration

	// BackupConfig is the configuration for image backups.
	BackupConfig ImageBackupConfig
}

// DefaultImageBackupSchedulerConfig returns default scheduler configuration.
func DefaultImageBackupSchedulerConfig() ImageBackupSchedulerConfig {
	return ImageBackupSchedulerConfig{
		RefreshInterval: 5 * time.Minute,
		BackupConfig:    DefaultImageBackupConfig(),
	}
}

// ImageBackupScheduler manages Docker image backup schedules.
type ImageBackupScheduler struct {
	store    ImageBackupStore
	service  *ImageBackupService
	config   ImageBackupSchedulerConfig
	cron     *cron.Cron
	logger   zerolog.Logger
	mu       sync.RWMutex
	entries  map[uuid.UUID]cron.EntryID
	running  bool
}

// NewImageBackupScheduler creates a new image backup scheduler.
func NewImageBackupScheduler(
	store ImageBackupStore,
	config ImageBackupSchedulerConfig,
	logger zerolog.Logger,
) *ImageBackupScheduler {
	service := NewImageBackupService(config.BackupConfig, logger)

	return &ImageBackupScheduler{
		store:   store,
		service: service,
		config:  config,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With().Str("component", "docker_image_scheduler").Logger(),
		entries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the scheduler and loads initial schedules.
func (s *ImageBackupScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("docker image backup scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting Docker image backup scheduler")

	// Load checksum cache for deduplication
	if err := s.service.LoadChecksumCache(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("failed to load checksum cache")
	}

	// Load initial schedules
	if err := s.Reload(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to load initial schedules")
	}

	// Start cron scheduler
	s.cron.Start()

	// Start background refresh goroutine
	go s.refreshLoop(ctx)

	s.logger.Info().Msg("Docker image backup scheduler started")
	return nil
}

// Stop stops the scheduler gracefully.
func (s *ImageBackupScheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping Docker image backup scheduler")

	return s.cron.Stop()
}

// Reload reloads all schedules from the database.
func (s *ImageBackupScheduler) Reload(ctx context.Context) error {
	s.logger.Debug().Msg("reloading Docker image backup schedules")

	schedules, err := s.store.GetEnabledDockerImageSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get enabled schedules: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		if entryID, exists := s.entries[schedule.ID]; exists {
			entry := s.cron.Entry(entryID)
			if entry.Valid() {
				continue
			}
			s.cron.Remove(entryID)
			delete(s.entries, schedule.ID)
		}

		if err := s.addSchedule(schedule); err != nil {
			s.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("failed to add schedule")
			continue
		}
	}

	for id, entryID := range s.entries {
		if !seen[id] {
			s.cron.Remove(entryID)
			delete(s.entries, id)
			s.logger.Debug().
				Str("schedule_id", id.String()).
				Msg("removed disabled schedule")
		}
	}

	s.logger.Info().
		Int("active_schedules", len(s.entries)).
		Msg("Docker image backup schedules reloaded")

	return nil
}

// addSchedule adds a schedule to the cron scheduler.
func (s *ImageBackupScheduler) addSchedule(schedule *models.DockerImageSchedule) error {
	sched := schedule

	entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
		s.executeBackup(sched)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	s.entries[schedule.ID] = entryID
	s.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("cron_expression", schedule.CronExpression).
		Msg("added Docker image backup schedule")

	return nil
}

// executeBackup runs a Docker image backup for the given schedule.
func (s *ImageBackupScheduler) executeBackup(schedule *models.DockerImageSchedule) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Logger()

	logger.Info().Msg("starting scheduled Docker image backup")

	// Get agent info
	agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get agent")
		return
	}

	// Create backup record
	backup := models.NewDockerImageBackupJob(agent.OrgID, schedule.AgentID, schedule.Options.ImageBackupMode)
	backup.ScheduleID = &schedule.ID

	if err := s.store.CreateDockerImageBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to create backup record")
		return
	}

	// Start backup
	backup.Start()
	if err := s.store.UpdateDockerImageBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update backup record")
	}

	// Configure service based on schedule options
	s.service.config.ExcludePublicImages = schedule.Options.ExcludePublicImages
	if len(schedule.Options.CustomRegistries) > 0 {
		// Add custom registries to public list if excluding public
		s.service.config.PublicRegistries = append(
			DefaultImageBackupConfig().PublicRegistries,
			schedule.Options.CustomRegistries...,
		)
	}

	// Determine containers to backup
	var containerIDs []string
	switch schedule.Options.ImageBackupMode {
	case models.DockerImageBackupModeAll:
		// Empty containerIDs means all images
	case models.DockerImageBackupModeContainers:
		// Get container images
		versions, err := s.service.GetContainerImages(ctx)
		if err != nil {
			s.failBackup(ctx, backup, fmt.Sprintf("get container images: %v", err), logger)
			return
		}
		for _, v := range versions {
			containerIDs = append(containerIDs, v.ContainerID)
		}
	case models.DockerImageBackupModeSelected:
		containerIDs = schedule.Options.SelectedImages
	}

	// Execute backup
	result, err := s.service.BackupImages(ctx, containerIDs)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("backup failed: %v", err), logger)
		return
	}

	// Record image versions
	for _, v := range result.ImageVersions {
		version := models.NewDockerImageVersion(backup.ID, v.ContainerID, v.ContainerName, v.ImageID, v.ImageTag)
		version.SizeBytes = v.SizeBytes
		version.BackupPath = v.BackupPath

		// Check for deduplication
		if checksum, ok := s.service.checksumCache[v.ImageID]; ok {
			version.Checksum = checksum
			existingEntry, err := s.store.GetDeduplicationEntry(ctx, agent.OrgID, v.ImageID)
			if err == nil && existingEntry != nil {
				version.MarkDeduplicated(existingEntry.OriginalPath)
				existingEntry.IncrementReference()
				s.store.UpdateDeduplicationEntry(ctx, existingEntry)
			} else {
				// Create new deduplication entry
				entry := models.NewDockerImageDeduplicationEntry(
					agent.OrgID,
					v.ImageID,
					checksum,
					backup.ID,
					v.BackupPath,
					v.SizeBytes,
				)
				s.store.CreateDeduplicationEntry(ctx, entry)
			}
		}

		if err := s.store.CreateDockerImageVersion(ctx, version); err != nil {
			logger.Warn().Err(err).Str("image_id", v.ImageID).Msg("failed to create version record")
		}
	}

	// Complete backup
	backup.Complete(
		s.config.BackupConfig.BackupDir+"/"+result.BackupID.String(),
		result.ImagesBackedUp,
		result.ImagesSkipped,
		len(result.DeduplicatedIDs),
		result.TotalSizeBytes,
	)

	if err := s.store.UpdateDockerImageBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update backup record")
	}

	// Log errors if any
	if len(result.Errors) > 0 {
		for _, errMsg := range result.Errors {
			logger.Warn().Str("error", errMsg).Msg("backup warning")
		}
	}

	logger.Info().
		Int("images_backed_up", result.ImagesBackedUp).
		Int("images_skipped", result.ImagesSkipped).
		Int("deduplicated", len(result.DeduplicatedIDs)).
		Int64("total_size_bytes", result.TotalSizeBytes).
		Msg("Docker image backup completed")
}

// failBackup marks a backup as failed.
func (s *ImageBackupScheduler) failBackup(ctx context.Context, backup *models.DockerImageBackupJob, errMsg string, logger zerolog.Logger) {
	backup.Fail(errMsg)
	if err := s.store.UpdateDockerImageBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update backup record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("Docker image backup failed")
}

// refreshLoop periodically reloads schedules.
func (s *ImageBackupScheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()

			if !running {
				return
			}

			if err := s.Reload(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to reload schedules")
			}
		}
	}
}

// TriggerBackup manually triggers a Docker image backup.
func (s *ImageBackupScheduler) TriggerBackup(ctx context.Context, scheduleID uuid.UUID) error {
	schedule, err := s.store.GetDockerImageScheduleByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("get schedule: %w", err)
	}

	go s.executeBackup(schedule)
	return nil
}

// GetActiveSchedules returns the number of active schedules.
func (s *ImageBackupScheduler) GetActiveSchedules() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// GetNextRun returns the next scheduled run time.
func (s *ImageBackupScheduler) GetNextRun(scheduleID uuid.UUID) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.entries[scheduleID]
	if !exists {
		return time.Time{}, false
	}

	entry := s.cron.Entry(entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	return entry.Next, true
}

// CleanupOldBackups removes old image backups based on retention.
func (s *ImageBackupScheduler) CleanupOldBackups(ctx context.Context, retentionDays int) (int, error) {
	return s.service.CleanupOldBackups(ctx, retentionDays)
}
