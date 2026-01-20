package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NotificationStore defines the interface for notification data access.
type NotificationStore interface {
	GetEnabledPreferencesForEvent(ctx context.Context, orgID uuid.UUID, eventType models.NotificationEventType) ([]*models.NotificationPreference, error)
	GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	CreateNotificationLog(ctx context.Context, log *models.NotificationLog) error
	UpdateNotificationLog(ctx context.Context, log *models.NotificationLog) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
}

// Service handles sending notifications for backup events.
type Service struct {
	store  NotificationStore
	logger zerolog.Logger
}

// NewService creates a new notification service.
func NewService(store NotificationStore, logger zerolog.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger.With().Str("component", "notification_service").Logger(),
	}
}

// BackupResult contains information about a completed backup.
type BackupResult struct {
	OrgID        uuid.UUID
	ScheduleID   uuid.UUID
	ScheduleName string
	AgentID      uuid.UUID
	Hostname     string
	SnapshotID   string
	StartedAt    time.Time
	CompletedAt  time.Time
	SizeBytes    int64
	FilesNew     int
	FilesChanged int
	Success      bool
	ErrorMessage string
}

// NotifyBackupComplete sends notifications for a completed backup.
func (s *Service) NotifyBackupComplete(ctx context.Context, result BackupResult) {
	var eventType models.NotificationEventType
	if result.Success {
		eventType = models.EventBackupSuccess
	} else {
		eventType = models.EventBackupFailed
	}

	prefs, err := s.store.GetEnabledPreferencesForEvent(ctx, result.OrgID, eventType)
	if err != nil {
		s.logger.Error().Err(err).
			Str("org_id", result.OrgID.String()).
			Str("event_type", string(eventType)).
			Msg("failed to get notification preferences")
		return
	}

	if len(prefs) == 0 {
		s.logger.Debug().
			Str("org_id", result.OrgID.String()).
			Str("event_type", string(eventType)).
			Msg("no notification preferences enabled for event")
		return
	}

	for _, pref := range prefs {
		go s.sendNotification(ctx, pref, result)
	}
}

// NotifyAgentOffline sends notifications when an agent goes offline.
func (s *Service) NotifyAgentOffline(ctx context.Context, agent *models.Agent, orgID uuid.UUID, offlineSince time.Duration) {
	prefs, err := s.store.GetEnabledPreferencesForEvent(ctx, orgID, models.EventAgentOffline)
	if err != nil {
		s.logger.Error().Err(err).
			Str("org_id", orgID.String()).
			Str("agent_id", agent.ID.String()).
			Msg("failed to get notification preferences")
		return
	}

	if len(prefs) == 0 {
		s.logger.Debug().
			Str("org_id", orgID.String()).
			Str("event_type", string(models.EventAgentOffline)).
			Msg("no notification preferences enabled for event")
		return
	}

	data := AgentOfflineData{
		Hostname:     agent.Hostname,
		AgentID:      agent.ID.String(),
		OfflineSince: formatDuration(offlineSince),
	}
	if agent.LastSeen != nil {
		data.LastSeen = *agent.LastSeen
	}

	for _, pref := range prefs {
		go s.sendAgentOfflineNotification(ctx, pref, data, orgID)
	}
}

// sendNotification sends a notification for a backup result.
func (s *Service) sendNotification(ctx context.Context, pref *models.NotificationPreference, result BackupResult) {
	channel, err := s.store.GetNotificationChannelByID(ctx, pref.ChannelID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", pref.ChannelID.String()).
			Msg("failed to get notification channel")
		return
	}

	// Only handle email channels for now
	if channel.Type != models.ChannelTypeEmail {
		s.logger.Debug().
			Str("channel_type", string(channel.Type)).
			Msg("skipping non-email channel")
		return
	}

	// Parse email config
	var emailConfig models.EmailChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("failed to parse email config")
		return
	}

	// Create email service
	smtpConfig := SMTPConfig{
		Host:     emailConfig.Host,
		Port:     emailConfig.Port,
		Username: emailConfig.Username,
		Password: emailConfig.Password,
		From:     emailConfig.From,
		TLS:      emailConfig.TLS,
	}

	emailService, err := NewEmailService(smtpConfig, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create email service")
		return
	}

	// Get recipient from config (use the from address as default recipient)
	recipients := []string{emailConfig.From}

	// Create log entry
	var subject string
	if result.Success {
		subject = fmt.Sprintf("Backup Successful: %s - %s", result.Hostname, result.ScheduleName)
	} else {
		subject = fmt.Sprintf("Backup Failed: %s - %s", result.Hostname, result.ScheduleName)
	}

	log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), recipients[0], subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	// Send the email
	var sendErr error
	if result.Success {
		duration := result.CompletedAt.Sub(result.StartedAt)
		data := BackupSuccessData{
			Hostname:     result.Hostname,
			ScheduleName: result.ScheduleName,
			SnapshotID:   result.SnapshotID,
			StartedAt:    result.StartedAt,
			CompletedAt:  result.CompletedAt,
			Duration:     formatDuration(duration),
			SizeBytes:    result.SizeBytes,
			FilesNew:     result.FilesNew,
			FilesChanged: result.FilesChanged,
		}
		sendErr = emailService.SendBackupSuccess(recipients, data)
	} else {
		data := BackupFailedData{
			Hostname:     result.Hostname,
			ScheduleName: result.ScheduleName,
			StartedAt:    result.StartedAt,
			FailedAt:     result.CompletedAt,
			ErrorMessage: result.ErrorMessage,
		}
		sendErr = emailService.SendBackupFailed(recipients, data)
	}

	// Update log with result
	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("recipient", recipients[0]).
			Msg("failed to send email notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("recipient", recipients[0]).
			Str("event_type", string(pref.EventType)).
			Msg("email notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// sendAgentOfflineNotification sends an agent offline notification.
func (s *Service) sendAgentOfflineNotification(ctx context.Context, pref *models.NotificationPreference, data AgentOfflineData, orgID uuid.UUID) {
	channel, err := s.store.GetNotificationChannelByID(ctx, pref.ChannelID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", pref.ChannelID.String()).
			Msg("failed to get notification channel")
		return
	}

	// Only handle email channels for now
	if channel.Type != models.ChannelTypeEmail {
		s.logger.Debug().
			Str("channel_type", string(channel.Type)).
			Msg("skipping non-email channel")
		return
	}

	// Parse email config
	var emailConfig models.EmailChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("failed to parse email config")
		return
	}

	// Create email service
	smtpConfig := SMTPConfig{
		Host:     emailConfig.Host,
		Port:     emailConfig.Port,
		Username: emailConfig.Username,
		Password: emailConfig.Password,
		From:     emailConfig.From,
		TLS:      emailConfig.TLS,
	}

	emailService, err := NewEmailService(smtpConfig, s.logger)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create email service")
		return
	}

	recipients := []string{emailConfig.From}
	subject := fmt.Sprintf("Agent Offline: %s", data.Hostname)

	log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), recipients[0], subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	sendErr := emailService.SendAgentOffline(recipients, data)

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("recipient", recipients[0]).
			Msg("failed to send agent offline notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("recipient", recipients[0]).
			Str("agent_id", data.AgentID).
			Msg("agent offline notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		if seconds > 0 {
			return fmt.Sprintf("%d min %d sec", minutes, seconds)
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%d hr %d min", hours, minutes)
	}
	return fmt.Sprintf("%d hours", hours)
}
