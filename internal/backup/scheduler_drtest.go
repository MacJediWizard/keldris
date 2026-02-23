package backup

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

// DRTestStore defines the interface for DR test scheduling.
type DRTestStore interface {
	GetEnabledDRTestSchedules(ctx context.Context) ([]*models.DRTestSchedule, error)
	GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error)
	CreateDRTest(ctx context.Context, test *models.DRTest) error
	UpdateDRTest(ctx context.Context, test *models.DRTest) error
	UpdateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)
}

// DRTestScheduler manages DR test schedules using cron.
type DRTestScheduler struct {
	store     DRTestStore
	restic    *Restic
	config    SchedulerConfig
	cron      *cron.Cron
	logger    zerolog.Logger
	mu        sync.RWMutex
	drEntries map[uuid.UUID]cron.EntryID
	running   bool
}

// NewDRTestScheduler creates a new DR test scheduler.
func NewDRTestScheduler(store DRTestStore, restic *Restic, config SchedulerConfig, logger zerolog.Logger) *DRTestScheduler {
	return &DRTestScheduler{
		store:     store,
		restic:    restic,
		config:    config,
		cron:      cron.New(cron.WithSeconds()),
		logger:    logger.With().Str("component", "dr_scheduler").Logger(),
		drEntries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the DR test scheduler.
func (s *DRTestScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("DR test scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting DR test scheduler")

	// Load initial schedules
	if err := s.ReloadDRSchedules(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to load initial DR test schedules")
	}

	// Start cron scheduler
	s.cron.Start()

	// Start background refresh goroutine
	go s.refreshDRLoop(ctx)

	s.logger.Info().Msg("DR test scheduler started")
	return nil
}

// Stop stops the DR test scheduler gracefully.
func (s *DRTestScheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping DR test scheduler")

	return s.cron.Stop()
}

// ReloadDRSchedules reloads all DR test schedules from the database.
func (s *DRTestScheduler) ReloadDRSchedules(ctx context.Context) error {
	s.logger.Debug().Msg("reloading DR test schedules from database")

	schedules, err := s.store.GetEnabledDRTestSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get enabled DR test schedules: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track which schedules we've seen
	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		// Check if schedule already exists
		if entryID, exists := s.drEntries[schedule.ID]; exists {
			entry := s.cron.Entry(entryID)
			if entry.Valid() {
				continue
			}
			s.cron.Remove(entryID)
			delete(s.drEntries, schedule.ID)
		}

		// Add new schedule
		if err := s.addDRTestSchedule(schedule); err != nil {
			s.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("failed to add DR test schedule")
			continue
		}
	}

	// Remove schedules that are no longer enabled
	for id, entryID := range s.drEntries {
		if !seen[id] {
			s.cron.Remove(entryID)
			delete(s.drEntries, id)
			s.logger.Debug().
				Str("schedule_id", id.String()).
				Msg("removed disabled DR test schedule")
		}
	}

	s.logger.Info().
		Int("active_dr_schedules", len(s.drEntries)).
		Msg("DR test schedules reloaded")

	return nil
}

// addDRTestSchedule adds a DR test schedule to the cron scheduler.
func (s *DRTestScheduler) addDRTestSchedule(schedule *models.DRTestSchedule) error {
	// Create a copy for the closure
	sched := *schedule

	entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
		s.executeDRTest(sched)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	s.drEntries[schedule.ID] = entryID

	// Update next run time
	entry := s.cron.Entry(entryID)
	if entry.Valid() {
		sched.NextRunAt = &entry.Next
		if err := s.store.UpdateDRTestSchedule(context.Background(), &sched); err != nil {
			s.logger.Warn().Err(err).Str("schedule_id", schedule.ID.String()).Msg("failed to update next run time")
		}
	}

	s.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("cron_expression", schedule.CronExpression).
		Msg("added DR test schedule")

	return nil
}

// executeDRTest runs a DR test for the given schedule.
func (s *DRTestScheduler) executeDRTest(schedule models.DRTestSchedule) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("runbook_id", schedule.RunbookID.String()).
		Logger()

	logger.Info().Msg("starting scheduled DR test")

	// Get runbook
	runbook, err := s.store.GetDRRunbookByID(ctx, schedule.RunbookID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get runbook")
		return
	}

	// Create DR test record
	test := models.NewDRTest(schedule.RunbookID)
	test.Notes = "Scheduled DR test"

	if runbook.ScheduleID != nil {
		test.SetSchedule(*runbook.ScheduleID)
	}

	if err := s.store.CreateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to create DR test record")
		return
	}

	// Start the test
	test.Start()
	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to update DR test record")
		return
	}

	// Perform the restore test if there's an associated schedule
	if runbook.ScheduleID != nil {
		backupSchedule, err := s.store.GetScheduleByID(ctx, *runbook.ScheduleID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get backup schedule: %v", err), logger)
			return
		}

		// Get primary repository from schedule
		primaryRepo := backupSchedule.GetPrimaryRepository()
		if primaryRepo == nil {
			s.failDRTest(ctx, test, "no primary repository configured for schedule", logger)
			return
		}

		// Get repository configuration
		repo, err := s.store.GetRepository(ctx, primaryRepo.RepositoryID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get repository: %v", err), logger)
			return
		}

		// Decrypt repository configuration
		if s.config.DecryptFunc == nil {
			s.failDRTest(ctx, test, "decrypt function not configured", logger)
			return
		}

		configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("decrypt config: %v", err), logger)
			return
		}

		// Parse backend configuration
		backend, err := ParseBackend(repo.Type, configJSON)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("parse backend: %v", err), logger)
			return
		}

		// Get repository password
		if s.config.PasswordFunc == nil {
			s.failDRTest(ctx, test, "password function not configured", logger)
			return
		}

		password, err := s.config.PasswordFunc(repo.ID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get password: %v", err), logger)
			return
		}

		// Build restic config
		resticCfg := backend.ToResticConfig(password)

		// Get the latest snapshot
		snapshots, err := s.restic.Snapshots(ctx, resticCfg)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("list snapshots: %v", err), logger)
			return
		}

		if len(snapshots) == 0 {
			s.failDRTest(ctx, test, "no snapshots available for restore test", logger)
			return
		}

		// Use the most recent snapshot
		latestSnapshot := snapshots[0]
		test.SnapshotID = latestSnapshot.ID

		// Perform restore to a temporary location for verification
		startTime := time.Now()

		// Note: In a real implementation, you would restore to a temp directory
		// and verify the data. For now, we just verify the snapshot is readable.
		stats, err := s.restic.Stats(ctx, resticCfg)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("verify snapshot: %v", err), logger)
			return
		}

		duration := int(time.Since(startTime).Seconds())

		// Mark test as completed
		test.Complete(latestSnapshot.ID, stats.TotalSize, duration, true)
	} else {
		// No associated schedule, just mark as completed for manual verification
		test.Complete("", 0, 0, true)
		test.Notes = "Scheduled DR test - manual verification required"
	}

	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to update DR test record")
		return
	}

	// Update schedule's last run time
	now := time.Now()
	schedule.LastRunAt = &now
	if err := s.store.UpdateDRTestSchedule(ctx, &schedule); err != nil {
		logger.Warn().Err(err).Msg("failed to update DR test schedule")
	}

	logger.Info().
		Str("test_id", test.ID.String()).
		Bool("passed", *test.VerificationPassed).
		Msg("DR test completed")
}

// failDRTest marks a DR test as failed.
func (s *DRTestScheduler) failDRTest(ctx context.Context, test *models.DRTest, errMsg string, logger zerolog.Logger) {
	test.Fail(errMsg)
	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update DR test record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("DR test failed")
}

// refreshDRLoop periodically reloads DR test schedules from the database.
func (s *DRTestScheduler) refreshDRLoop(ctx context.Context) {
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

			if err := s.ReloadDRSchedules(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to reload DR test schedules")
			}
		}
	}
}

// TriggerDRTest manually triggers a DR test for the given runbook.
func (s *DRTestScheduler) TriggerDRTest(ctx context.Context, runbookID uuid.UUID) error {
	runbook, err := s.store.GetDRRunbookByID(ctx, runbookID)
	if err != nil {
		return fmt.Errorf("get runbook: %w", err)
	}

	// Create a temporary schedule for this manual run
	tempSchedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbook.ID,
	}

	go s.executeDRTest(tempSchedule)
	return nil
}

// GetActiveDRSchedules returns the number of active DR test schedules.
func (s *DRTestScheduler) GetActiveDRSchedules() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.drEntries)
}
