package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// SchedulerStore defines the interface for report schedule persistence.
type SchedulerStore interface {
	ReportStore

	GetEnabledReportSchedules(ctx context.Context) ([]*models.ReportSchedule, error)
	GetReportScheduleByID(ctx context.Context, id uuid.UUID) (*models.ReportSchedule, error)
	UpdateReportScheduleLastSent(ctx context.Context, id uuid.UUID, lastSentAt time.Time) error
	CreateReportHistory(ctx context.Context, history *models.ReportHistory) error
	GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
}

// SchedulerConfig holds configuration for the report scheduler.
type SchedulerConfig struct {
	RefreshInterval time.Duration
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		RefreshInterval: 5 * time.Minute,
	}
}

// Scheduler manages report schedules using cron.
type Scheduler struct {
	store     SchedulerStore
	generator *Generator
	config    SchedulerConfig
	cron      *cron.Cron
	logger    zerolog.Logger
	mu        sync.RWMutex
	entries   map[uuid.UUID]cron.EntryID
	running   bool
}

// NewScheduler creates a new report scheduler.
func NewScheduler(store SchedulerStore, config SchedulerConfig, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		store:     store,
		generator: NewGenerator(store, logger),
		config:    config,
		cron:      cron.New(cron.WithSeconds()),
		logger:    logger.With().Str("component", "report_scheduler").Logger(),
		entries:   make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the report scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting report scheduler")

	// Load initial schedules
	if err := s.Reload(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to load initial report schedules")
	}

	s.cron.Start()

	// Start background refresh
	go s.refreshLoop(ctx)

	s.logger.Info().Msg("report scheduler started")
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping report scheduler")
	return s.cron.Stop()
}

// Reload reloads schedules from the database.
func (s *Scheduler) Reload(ctx context.Context) error {
	schedules, err := s.store.GetEnabledReportSchedules(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		if _, exists := s.entries[schedule.ID]; exists {
			continue
		}

		if err := s.addSchedule(schedule); err != nil {
			s.logger.Error().Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("failed to add report schedule")
		}
	}

	// Remove schedules that are no longer enabled
	for id, entryID := range s.entries {
		if !seen[id] {
			s.cron.Remove(entryID)
			delete(s.entries, id)
		}
	}

	s.logger.Info().Int("active_schedules", len(s.entries)).Msg("report schedules reloaded")
	return nil
}

func (s *Scheduler) addSchedule(schedule *models.ReportSchedule) error {
	cronExpr := s.frequencyToCron(schedule.Frequency)

	sched := schedule // Copy for closure
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeReport(sched)
	})
	if err != nil {
		return err
	}

	s.entries[schedule.ID] = entryID
	s.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("cron", cronExpr).
		Msg("added report schedule")

	return nil
}

func (s *Scheduler) frequencyToCron(frequency models.ReportFrequency) string {
	// All reports run at 8:00 AM
	switch frequency {
	case models.ReportFrequencyDaily:
		return "0 0 8 * * *" // 8:00 AM daily
	case models.ReportFrequencyWeekly:
		return "0 0 8 * * 1" // 8:00 AM Monday
	case models.ReportFrequencyMonthly:
		return "0 0 8 1 * *" // 8:00 AM 1st of month
	default:
		return "0 0 8 * * 1" // Default to weekly
	}
}

func (s *Scheduler) executeReport(schedule *models.ReportSchedule) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("org_id", schedule.OrgID.String()).
		Logger()

	logger.Info().Msg("executing scheduled report")

	// Calculate period
	periodStart, periodEnd := CalculatePeriod(schedule.Frequency, schedule.Timezone)

	// Generate report
	reportData, err := s.generator.GenerateReport(ctx, schedule.OrgID, periodStart, periodEnd)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate report")
		return
	}

	// Send report
	err = s.SendReport(ctx, schedule, reportData, periodStart, periodEnd, false)
	if err != nil {
		logger.Error().Err(err).Msg("failed to send report")
		return
	}

	// Update last sent timestamp
	if err := s.store.UpdateReportScheduleLastSent(ctx, schedule.ID, time.Now()); err != nil {
		logger.Error().Err(err).Msg("failed to update last sent timestamp")
	}

	logger.Info().Msg("scheduled report sent successfully")
}

// SendReport sends a report (used by both scheduler and manual trigger).
func (s *Scheduler) SendReport(ctx context.Context, schedule *models.ReportSchedule, data *models.ReportData, periodStart, periodEnd time.Time, preview bool) error {
	// Create history entry
	history := models.NewReportHistory(
		schedule.OrgID,
		&schedule.ID,
		string(schedule.Frequency),
		periodStart,
		periodEnd,
		schedule.Recipients,
	)
	history.ReportData = data

	if preview {
		history.Status = models.ReportStatusPreview
		return s.store.CreateReportHistory(ctx, history)
	}

	// Get notification channel for SMTP config
	var smtpConfig *notifications.SMTPConfig
	if schedule.ChannelID != nil {
		channel, err := s.store.GetNotificationChannelByID(ctx, *schedule.ChannelID)
		if err == nil && channel.Type == models.ChannelTypeEmail {
			var emailConfig models.EmailChannelConfig
			if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err == nil {
				smtpConfig = &notifications.SMTPConfig{
					Host:     emailConfig.Host,
					Port:     emailConfig.Port,
					Username: emailConfig.Username,
					Password: emailConfig.Password,
					From:     emailConfig.From,
					TLS:      emailConfig.TLS,
				}
			}
		}
	}

	if smtpConfig == nil {
		history.MarkFailed("no email channel configured")
		if err := s.store.CreateReportHistory(ctx, history); err != nil {
			s.logger.Error().Err(err).Msg("failed to create report history")
		}
		return fmt.Errorf("no email channel configured")
	}

	// Get organization name
	orgName := schedule.Name
	org, err := s.store.GetOrganizationByID(ctx, schedule.OrgID)
	if err == nil && org != nil {
		orgName = org.Name
	}

	// Create email service and send
	emailService, err := notifications.NewEmailService(*smtpConfig, s.logger)
	if err != nil {
		history.MarkFailed(err.Error())
		if err := s.store.CreateReportHistory(ctx, history); err != nil {
			s.logger.Error().Err(err).Msg("failed to create report history")
		}
		return err
	}

	// Convert to email data format
	emailData := notifications.ReportEmailData{
		OrgName:              orgName,
		Frequency:            strings.Title(string(schedule.Frequency)),
		FrequencyLower:       string(schedule.Frequency),
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		TotalDataFormatted:   notifications.FormatBytes(data.BackupSummary.TotalDataBacked),
		RawSizeFormatted:     notifications.FormatBytes(data.StorageSummary.TotalRawSize),
		RestoreSizeFormatted: notifications.FormatBytes(data.StorageSummary.TotalRestoreSize),
		SpaceSavedFormatted:  notifications.FormatBytes(data.StorageSummary.SpaceSaved),
		Data: &notifications.ReportData{
			BackupSummary: notifications.ReportBackupSummary{
				TotalBackups:      data.BackupSummary.TotalBackups,
				SuccessfulBackups: data.BackupSummary.SuccessfulBackups,
				FailedBackups:     data.BackupSummary.FailedBackups,
				SuccessRate:       data.BackupSummary.SuccessRate,
				TotalDataBacked:   data.BackupSummary.TotalDataBacked,
				SchedulesActive:   data.BackupSummary.SchedulesActive,
			},
			StorageSummary: notifications.ReportStorageSummary{
				TotalRawSize:     data.StorageSummary.TotalRawSize,
				TotalRestoreSize: data.StorageSummary.TotalRestoreSize,
				SpaceSaved:       data.StorageSummary.SpaceSaved,
				SpaceSavedPct:    data.StorageSummary.SpaceSavedPct,
				RepositoryCount:  data.StorageSummary.RepositoryCount,
				TotalSnapshots:   data.StorageSummary.TotalSnapshots,
			},
			AgentSummary: notifications.ReportAgentSummary{
				TotalAgents:   data.AgentSummary.TotalAgents,
				ActiveAgents:  data.AgentSummary.ActiveAgents,
				OfflineAgents: data.AgentSummary.OfflineAgents,
				PendingAgents: data.AgentSummary.PendingAgents,
			},
			AlertSummary: notifications.ReportAlertSummary{
				TotalAlerts:        data.AlertSummary.TotalAlerts,
				CriticalAlerts:     data.AlertSummary.CriticalAlerts,
				WarningAlerts:      data.AlertSummary.WarningAlerts,
				AcknowledgedAlerts: data.AlertSummary.AcknowledgedAlerts,
				ResolvedAlerts:     data.AlertSummary.ResolvedAlerts,
			},
		},
	}

	// Convert top issues
	if len(data.TopIssues) > 0 {
		emailData.Data.TopIssues = make([]notifications.ReportIssue, len(data.TopIssues))
		for i, issue := range data.TopIssues {
			emailData.Data.TopIssues[i] = notifications.ReportIssue{
				Type:        issue.Type,
				Severity:    issue.Severity,
				Title:       issue.Title,
				Description: issue.Description,
				OccurredAt:  issue.OccurredAt,
			}
		}
	}

	// Send report email
	err = emailService.SendReport(schedule.Recipients, emailData)

	if err != nil {
		history.MarkFailed(err.Error())
	} else {
		history.MarkSent()
	}

	if err := s.store.CreateReportHistory(ctx, history); err != nil {
		s.logger.Error().Err(err).Msg("failed to create report history")
	}
	return err
}

// GeneratePreview generates a report preview without sending.
func (s *Scheduler) GeneratePreview(ctx context.Context, orgID uuid.UUID, frequency models.ReportFrequency, timezone string) (*models.ReportData, time.Time, time.Time, error) {
	periodStart, periodEnd := CalculatePeriod(frequency, timezone)
	data, err := s.generator.GenerateReport(ctx, orgID, periodStart, periodEnd)
	return data, periodStart, periodEnd, err
}

// GetGenerator returns the generator for direct access.
func (s *Scheduler) GetGenerator() *Generator {
	return s.generator
}

func (s *Scheduler) refreshLoop(ctx context.Context) {
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
				s.logger.Error().Err(err).Msg("failed to reload report schedules")
			}
		}
	}
}
