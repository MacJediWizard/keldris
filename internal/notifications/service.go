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

	// Build notification data
	duration := result.CompletedAt.Sub(result.StartedAt)
	successData := BackupSuccessData{
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
	failedData := BackupFailedData{
		Hostname:     result.Hostname,
		ScheduleName: result.ScheduleName,
		StartedAt:    result.StartedAt,
		FailedAt:     result.CompletedAt,
		ErrorMessage: result.ErrorMessage,
	}

	// Create log entry
	var subject string
	var recipient string
	if result.Success {
		subject = fmt.Sprintf("Backup Successful: %s - %s", result.Hostname, result.ScheduleName)
	} else {
		subject = fmt.Sprintf("Backup Failed: %s - %s", result.Hostname, result.ScheduleName)
	}

	// Determine recipient based on channel type
	switch channel.Type {
	case models.ChannelTypeEmail:
		var config models.EmailChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = config.From
		}
	case models.ChannelTypeSlack:
		var config models.SlackChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = config.WebhookURL
		}
	case models.ChannelTypeTeams:
		var config models.TeamsChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = config.WebhookURL
		}
	case models.ChannelTypeDiscord:
		var config models.DiscordChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = config.WebhookURL
		}
	case models.ChannelTypePagerDuty:
		var config models.PagerDutyChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = "pagerduty:" + config.RoutingKey[:8] + "..."
		}
	case models.ChannelTypeWebhook:
		var config models.WebhookChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			recipient = config.URL
		}
	default:
		recipient = string(channel.Type)
	}

	log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), recipient, subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	// Send notification based on channel type
	var sendErr error
	switch channel.Type {
	case models.ChannelTypeEmail:
		sendErr = s.sendEmailBackup(channel, result.Success, successData, failedData)
	case models.ChannelTypeSlack:
		sendErr = s.sendSlackBackup(channel, result.Success, successData, failedData)
	case models.ChannelTypeTeams:
		sendErr = s.sendTeamsBackup(channel, result.Success, successData, failedData)
	case models.ChannelTypeDiscord:
		sendErr = s.sendDiscordBackup(channel, result.Success, successData, failedData)
	case models.ChannelTypePagerDuty:
		sendErr = s.sendPagerDutyBackup(channel, result.Success, successData, failedData)
	case models.ChannelTypeWebhook:
		sendErr = s.sendWebhookBackup(channel, result.Success, successData, failedData)
	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
		return
	}

	// Update log with result
	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Msg("failed to send notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Str("event_type", string(pref.EventType)).
			Msg("notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// sendEmailBackup sends a backup notification via email.
func (s *Service) sendEmailBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var emailConfig models.EmailChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err != nil {
		return fmt.Errorf("parse email config: %w", err)
	}

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
		return fmt.Errorf("create email service: %w", err)
	}

	recipients := []string{emailConfig.From}
	if success {
		return emailService.SendBackupSuccess(recipients, successData)
	}
	return emailService.SendBackupFailed(recipients, failedData)
}

// sendSlackBackup sends a backup notification via Slack.
func (s *Service) sendSlackBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var config models.SlackChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	slackService, err := NewSlackService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create slack service: %w", err)
	}

	if success {
		return slackService.SendBackupSuccess(successData)
	}
	return slackService.SendBackupFailed(failedData)
}

// sendTeamsBackup sends a backup notification via Microsoft Teams.
func (s *Service) sendTeamsBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var config models.TeamsChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse teams config: %w", err)
	}

	teamsService, err := NewTeamsService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create teams service: %w", err)
	}

	if success {
		return teamsService.SendBackupSuccess(successData)
	}
	return teamsService.SendBackupFailed(failedData)
}

// sendDiscordBackup sends a backup notification via Discord.
func (s *Service) sendDiscordBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var config models.DiscordChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse discord config: %w", err)
	}

	discordService, err := NewDiscordService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create discord service: %w", err)
	}

	if success {
		return discordService.SendBackupSuccess(successData)
	}
	return discordService.SendBackupFailed(failedData)
}

// sendPagerDutyBackup sends a backup notification via PagerDuty.
func (s *Service) sendPagerDutyBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var config models.PagerDutyChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse pagerduty config: %w", err)
	}

	pdService, err := NewPagerDutyService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create pagerduty service: %w", err)
	}

	if success {
		return pdService.SendBackupSuccess(successData)
	}
	return pdService.SendBackupFailed(failedData)
}

// sendWebhookBackup sends a backup notification via generic webhook.
func (s *Service) sendWebhookBackup(channel *models.NotificationChannel, success bool, successData BackupSuccessData, failedData BackupFailedData) error {
	var config models.WebhookChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	webhookService, err := NewWebhookService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create webhook service: %w", err)
	}

	if success {
		return webhookService.SendBackupSuccess(successData)
	}
	return webhookService.SendBackupFailed(failedData)
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

	subject := fmt.Sprintf("Agent Offline: %s", data.Hostname)
	recipient := s.getChannelRecipient(channel)

	log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), recipient, subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	var sendErr error
	switch channel.Type {
	case models.ChannelTypeEmail:
		sendErr = s.sendEmailAgentOffline(channel, data)
	case models.ChannelTypeSlack:
		sendErr = s.sendSlackAgentOffline(channel, data)
	case models.ChannelTypeTeams:
		sendErr = s.sendTeamsAgentOffline(channel, data)
	case models.ChannelTypeDiscord:
		sendErr = s.sendDiscordAgentOffline(channel, data)
	case models.ChannelTypePagerDuty:
		sendErr = s.sendPagerDutyAgentOffline(channel, data)
	case models.ChannelTypeWebhook:
		sendErr = s.sendWebhookAgentOffline(channel, data)
	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
		return
	}

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Msg("failed to send agent offline notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Str("agent_id", data.AgentID).
			Msg("agent offline notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// getChannelRecipient returns a display-friendly recipient string for logging.
func (s *Service) getChannelRecipient(channel *models.NotificationChannel) string {
	switch channel.Type {
	case models.ChannelTypeEmail:
		var config models.EmailChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			return config.From
		}
	case models.ChannelTypeSlack:
		var config models.SlackChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			if config.Channel != "" {
				return config.Channel
			}
			return "slack-webhook"
		}
	case models.ChannelTypeTeams:
		return "teams-webhook"
	case models.ChannelTypeDiscord:
		return "discord-webhook"
	case models.ChannelTypePagerDuty:
		var config models.PagerDutyChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil && len(config.RoutingKey) > 8 {
			return "pagerduty:" + config.RoutingKey[:8] + "..."
		}
		return "pagerduty"
	case models.ChannelTypeWebhook:
		var config models.WebhookChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err == nil {
			return config.URL
		}
	}
	return string(channel.Type)
}

// sendEmailAgentOffline sends an agent offline notification via email.
func (s *Service) sendEmailAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var emailConfig models.EmailChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err != nil {
		return fmt.Errorf("parse email config: %w", err)
	}

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
		return fmt.Errorf("create email service: %w", err)
	}

	return emailService.SendAgentOffline([]string{emailConfig.From}, data)
}

// sendSlackAgentOffline sends an agent offline notification via Slack.
func (s *Service) sendSlackAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var config models.SlackChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	slackService, err := NewSlackService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create slack service: %w", err)
	}

	return slackService.SendAgentOffline(data)
}

// sendTeamsAgentOffline sends an agent offline notification via Microsoft Teams.
func (s *Service) sendTeamsAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var config models.TeamsChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse teams config: %w", err)
	}

	teamsService, err := NewTeamsService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create teams service: %w", err)
	}

	return teamsService.SendAgentOffline(data)
}

// sendDiscordAgentOffline sends an agent offline notification via Discord.
func (s *Service) sendDiscordAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var config models.DiscordChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse discord config: %w", err)
	}

	discordService, err := NewDiscordService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create discord service: %w", err)
	}

	return discordService.SendAgentOffline(data)
}

// sendPagerDutyAgentOffline sends an agent offline notification via PagerDuty.
func (s *Service) sendPagerDutyAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var config models.PagerDutyChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse pagerduty config: %w", err)
	}

	pdService, err := NewPagerDutyService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create pagerduty service: %w", err)
	}

	return pdService.SendAgentOffline(data)
}

// sendWebhookAgentOffline sends an agent offline notification via generic webhook.
func (s *Service) sendWebhookAgentOffline(channel *models.NotificationChannel, data AgentOfflineData) error {
	var config models.WebhookChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	webhookService, err := NewWebhookService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create webhook service: %w", err)
	}

	return webhookService.SendAgentOffline(data)
}

// NotifyMaintenanceScheduled sends notifications for an upcoming maintenance window.
func (s *Service) NotifyMaintenanceScheduled(ctx context.Context, window *models.MaintenanceWindow) {
	prefs, err := s.store.GetEnabledPreferencesForEvent(ctx, window.OrgID, models.EventMaintenanceScheduled)
	if err != nil {
		s.logger.Error().Err(err).
			Str("org_id", window.OrgID.String()).
			Str("window_id", window.ID.String()).
			Msg("failed to get notification preferences")
		return
	}

	if len(prefs) == 0 {
		s.logger.Debug().
			Str("org_id", window.OrgID.String()).
			Str("event_type", string(models.EventMaintenanceScheduled)).
			Msg("no notification preferences enabled for event")
		return
	}

	data := MaintenanceScheduledData{
		Title:    window.Title,
		Message:  window.Message,
		StartsAt: window.StartsAt,
		EndsAt:   window.EndsAt,
		Duration: formatDuration(window.Duration()),
	}

	for _, pref := range prefs {
		go s.sendMaintenanceNotification(ctx, pref, data, window.OrgID)
	}
}

// sendMaintenanceNotification sends a maintenance scheduled notification.
func (s *Service) sendMaintenanceNotification(ctx context.Context, pref *models.NotificationPreference, data MaintenanceScheduledData, orgID uuid.UUID) {
	channel, err := s.store.GetNotificationChannelByID(ctx, pref.ChannelID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", pref.ChannelID.String()).
			Msg("failed to get notification channel")
		return
	}

	subject := fmt.Sprintf("Scheduled Maintenance: %s", data.Title)
	recipient := s.getChannelRecipient(channel)

	log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), recipient, subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	var sendErr error
	switch channel.Type {
	case models.ChannelTypeEmail:
		sendErr = s.sendEmailMaintenance(channel, data)
	case models.ChannelTypeSlack:
		sendErr = s.sendSlackMaintenance(channel, data)
	case models.ChannelTypeTeams:
		sendErr = s.sendTeamsMaintenance(channel, data)
	case models.ChannelTypeDiscord:
		sendErr = s.sendDiscordMaintenance(channel, data)
	case models.ChannelTypePagerDuty:
		sendErr = s.sendPagerDutyMaintenance(channel, data)
	case models.ChannelTypeWebhook:
		sendErr = s.sendWebhookMaintenance(channel, data)
	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
		return
	}

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Msg("failed to send maintenance notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Str("title", data.Title).
			Msg("maintenance notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// sendEmailMaintenance sends a maintenance notification via email.
func (s *Service) sendEmailMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var emailConfig models.EmailChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &emailConfig); err != nil {
		return fmt.Errorf("parse email config: %w", err)
	}

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
		return fmt.Errorf("create email service: %w", err)
	}

	return emailService.SendMaintenanceScheduled([]string{emailConfig.From}, data)
}

// sendSlackMaintenance sends a maintenance notification via Slack.
func (s *Service) sendSlackMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var config models.SlackChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	slackService, err := NewSlackService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create slack service: %w", err)
	}

	return slackService.SendMaintenanceScheduled(data)
}

// sendTeamsMaintenance sends a maintenance notification via Microsoft Teams.
func (s *Service) sendTeamsMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var config models.TeamsChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse teams config: %w", err)
	}

	teamsService, err := NewTeamsService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create teams service: %w", err)
	}

	return teamsService.SendMaintenanceScheduled(data)
}

// sendDiscordMaintenance sends a maintenance notification via Discord.
func (s *Service) sendDiscordMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var config models.DiscordChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse discord config: %w", err)
	}

	discordService, err := NewDiscordService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create discord service: %w", err)
	}

	return discordService.SendMaintenanceScheduled(data)
}

// sendPagerDutyMaintenance sends a maintenance notification via PagerDuty.
func (s *Service) sendPagerDutyMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var config models.PagerDutyChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse pagerduty config: %w", err)
	}

	pdService, err := NewPagerDutyService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create pagerduty service: %w", err)
	}

	return pdService.SendMaintenanceScheduled(data)
}

// sendWebhookMaintenance sends a maintenance notification via generic webhook.
func (s *Service) sendWebhookMaintenance(channel *models.NotificationChannel, data MaintenanceScheduledData) error {
	var config models.WebhookChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	webhookService, err := NewWebhookService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create webhook service: %w", err)
	}

	return webhookService.SendMaintenanceScheduled(data)
}

// TestChannel sends a test notification to verify the channel configuration is working.
func (s *Service) TestChannel(channel *models.NotificationChannel) error {
	switch channel.Type {
	case models.ChannelTypeEmail:
		var config models.EmailChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse email config: %w", err)
		}
		smtpConfig := SMTPConfig{
			Host:     config.Host,
			Port:     config.Port,
			Username: config.Username,
			Password: config.Password,
			From:     config.From,
			TLS:      config.TLS,
		}
		emailService, err := NewEmailService(smtpConfig, s.logger)
		if err != nil {
			return fmt.Errorf("create email service: %w", err)
		}
		testData := BackupSuccessData{
			Hostname:     "test-host",
			ScheduleName: "Test Schedule",
			SnapshotID:   "test-snapshot-123",
			StartedAt:    time.Now().Add(-5 * time.Minute),
			CompletedAt:  time.Now(),
			Duration:     "5 minutes",
			SizeBytes:    1024 * 1024 * 100,
			FilesNew:     10,
			FilesChanged: 5,
		}
		return emailService.SendBackupSuccess([]string{config.From}, testData)

	case models.ChannelTypeSlack:
		var config models.SlackChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse slack config: %w", err)
		}
		slackService, err := NewSlackService(config, s.logger)
		if err != nil {
			return fmt.Errorf("create slack service: %w", err)
		}
		return slackService.TestConnection()

	case models.ChannelTypeTeams:
		var config models.TeamsChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse teams config: %w", err)
		}
		teamsService, err := NewTeamsService(config, s.logger)
		if err != nil {
			return fmt.Errorf("create teams service: %w", err)
		}
		return teamsService.TestConnection()

	case models.ChannelTypeDiscord:
		var config models.DiscordChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse discord config: %w", err)
		}
		discordService, err := NewDiscordService(config, s.logger)
		if err != nil {
			return fmt.Errorf("create discord service: %w", err)
		}
		return discordService.TestConnection()

	case models.ChannelTypePagerDuty:
		var config models.PagerDutyChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse pagerduty config: %w", err)
		}
		pdService, err := NewPagerDutyService(config, s.logger)
		if err != nil {
			return fmt.Errorf("create pagerduty service: %w", err)
		}
		return pdService.TestConnection()

	case models.ChannelTypeWebhook:
		var config models.WebhookChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse webhook config: %w", err)
		}
		webhookService, err := NewWebhookService(config, s.logger)
		if err != nil {
			return fmt.Errorf("create webhook service: %w", err)
		}
		return webhookService.TestConnection()

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
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
