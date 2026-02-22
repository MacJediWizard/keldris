package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
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
	store                NotificationStore
	keyManager           *crypto.KeyManager
	logger               zerolog.Logger
	webhookSenderFunc    func(zerolog.Logger) *WebhookSender
	slackSenderFunc      func(zerolog.Logger) *SlackSender
	teamsSenderFunc      func(zerolog.Logger) *TeamsSender
	discordSenderFunc    func(zerolog.Logger) *DiscordSender
	pagerDutySenderFunc  func(zerolog.Logger) *PagerDutySender
}

// NewService creates a new notification service.
func NewService(store NotificationStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *Service {
	return &Service{
		store:               store,
		keyManager:          keyManager,
		logger:              logger.With().Str("component", "notification_service").Logger(),
		webhookSenderFunc:   NewWebhookSender,
		slackSenderFunc:     NewSlackSender,
		teamsSenderFunc:     NewTeamsSender,
		discordSenderFunc:   NewDiscordSender,
		pagerDutySenderFunc: NewPagerDutySender,
	}
}

// decryptConfig decrypts an encrypted channel config and unmarshals it into dest.
func (s *Service) decryptConfig(encrypted []byte, dest interface{}) error {
	decrypted, err := s.keyManager.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt config: %w", err)
	}
	if err := json.Unmarshal(decrypted, dest); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return nil
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

	var subject string
	var severity string
	if result.Success {
		subject = fmt.Sprintf("Backup Successful: %s - %s", result.Hostname, result.ScheduleName)
		severity = "info"
	} else {
		subject = fmt.Sprintf("Backup Failed: %s - %s", result.Hostname, result.ScheduleName)
		severity = "error"
	}

	switch channel.Type {
	case models.ChannelTypeEmail:
		s.sendBackupEmail(ctx, channel, pref, result, subject)

	case models.ChannelTypeSlack:
		var cfg models.SlackChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse slack config")
			return
		}
		log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		var body string
		if result.Success {
			duration := result.CompletedAt.Sub(result.StartedAt)
			body = fmt.Sprintf("*Host:* %s\n*Schedule:* %s\n*Snapshot:* %s\n*Duration:* %s\n*Size:* %s",
				result.Hostname, result.ScheduleName, result.SnapshotID, formatDuration(duration), FormatBytes(result.SizeBytes))
		} else {
			body = fmt.Sprintf("*Host:* %s\n*Schedule:* %s\n*Error:* %s",
				result.Hostname, result.ScheduleName, result.ErrorMessage)
		}
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: severity}
		sendErr := s.slackSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeWebhook:
		var cfg models.WebhookChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse webhook config")
			return
		}
		log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), cfg.URL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		payload := WebhookPayload{EventType: string(pref.EventType), Timestamp: time.Now(), Data: result}
		sendErr := s.webhookSenderFunc(s.logger).Send(ctx, cfg.URL, payload, cfg.Secret)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.URL)

	case models.ChannelTypePagerDuty:
		var cfg models.PagerDutyChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse pagerduty config")
			return
		}
		log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), "pagerduty", subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		event := PagerDutyEvent{Summary: subject, Source: result.Hostname, Severity: severity, Group: "backup"}
		sendErr := s.pagerDutySenderFunc(s.logger).Send(ctx, cfg.RoutingKey, event)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), "pagerduty")

	case models.ChannelTypeTeams:
		var cfg models.TeamsChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse teams config")
			return
		}
		log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		var body string
		if result.Success {
			duration := result.CompletedAt.Sub(result.StartedAt)
			body = fmt.Sprintf("**Host:** %s\n\n**Schedule:** %s\n\n**Snapshot:** %s\n\n**Duration:** %s\n\n**Size:** %s",
				result.Hostname, result.ScheduleName, result.SnapshotID, formatDuration(duration), FormatBytes(result.SizeBytes))
		} else {
			body = fmt.Sprintf("**Host:** %s\n\n**Schedule:** %s\n\n**Error:** %s",
				result.Hostname, result.ScheduleName, result.ErrorMessage)
		}
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: severity}
		sendErr := s.teamsSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeDiscord:
		var cfg models.DiscordChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse discord config")
			return
		}
		log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		var body string
		if result.Success {
			duration := result.CompletedAt.Sub(result.StartedAt)
			body = fmt.Sprintf("**Host:** %s\n**Schedule:** %s\n**Snapshot:** %s\n**Duration:** %s\n**Size:** %s",
				result.Hostname, result.ScheduleName, result.SnapshotID, formatDuration(duration), FormatBytes(result.SizeBytes))
		} else {
			body = fmt.Sprintf("**Host:** %s\n**Schedule:** %s\n**Error:** %s",
				result.Hostname, result.ScheduleName, result.ErrorMessage)
		}
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: severity}
		sendErr := s.discordSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
	}
}

// sendBackupEmail sends a backup notification via email (extracted from original logic).
func (s *Service) sendBackupEmail(ctx context.Context, channel *models.NotificationChannel, pref *models.NotificationPreference, result BackupResult, subject string) {
	var emailConfig models.EmailChannelConfig
	if err := s.decryptConfig(channel.ConfigEncrypted, &emailConfig); err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("failed to parse email config")
		return
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
		s.logger.Error().Err(err).Msg("failed to create email service")
		return
	}

	recipients := []string{emailConfig.From}

	log := models.NewNotificationLog(result.OrgID, &channel.ID, string(pref.EventType), recipients[0], subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
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

	var sendErr error
	if result.Success {
		sendErr = emailService.SendBackupSuccess(recipients, successData)
	} else {
		sendErr = emailService.SendBackupFailed(recipients, failedData)
	}
	s.finalizeLog(ctx, log, sendErr, channel.ID.String(), recipients[0])
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

	switch channel.Type {
	case models.ChannelTypeEmail:
		s.sendAgentOfflineEmail(ctx, channel, pref, data, orgID, subject)

	case models.ChannelTypeSlack:
		var cfg models.SlackChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse slack config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("*Host:* %s\n*Agent ID:* %s\n*Offline for:* %s", data.Hostname, data.AgentID, data.OfflineSince)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(models.EventAgentOffline), Severity: "warning"}
		sendErr := s.slackSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeWebhook:
		var cfg models.WebhookChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse webhook config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), cfg.URL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		payload := WebhookPayload{EventType: string(models.EventAgentOffline), Timestamp: time.Now(), Data: data}
		sendErr := s.webhookSenderFunc(s.logger).Send(ctx, cfg.URL, payload, cfg.Secret)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.URL)

	case models.ChannelTypePagerDuty:
		var cfg models.PagerDutyChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse pagerduty config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), "pagerduty", subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		event := PagerDutyEvent{Summary: subject, Source: data.Hostname, Severity: "warning", Group: "agent"}
		sendErr := s.pagerDutySenderFunc(s.logger).Send(ctx, cfg.RoutingKey, event)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), "pagerduty")

	case models.ChannelTypeTeams:
		var cfg models.TeamsChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse teams config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("**Host:** %s\n\n**Agent ID:** %s\n\n**Offline for:** %s", data.Hostname, data.AgentID, data.OfflineSince)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(models.EventAgentOffline), Severity: "warning"}
		sendErr := s.teamsSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeDiscord:
		var cfg models.DiscordChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse discord config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventAgentOffline), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("**Host:** %s\n**Agent ID:** %s\n**Offline for:** %s", data.Hostname, data.AgentID, data.OfflineSince)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(models.EventAgentOffline), Severity: "warning"}
		sendErr := s.discordSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
	}
}

// sendAgentOfflineEmail sends an agent offline notification via email.
func (s *Service) sendAgentOfflineEmail(ctx context.Context, channel *models.NotificationChannel, pref *models.NotificationPreference, data AgentOfflineData, orgID uuid.UUID, subject string) {
	var emailConfig models.EmailChannelConfig
	if err := s.decryptConfig(channel.ConfigEncrypted, &emailConfig); err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("failed to parse email config")
		return
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
		s.logger.Error().Err(err).Msg("failed to create email service")
		return
	}

	recipients := []string{emailConfig.From}

	log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), recipients[0], subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	sendErr := emailService.SendAgentOffline(recipients, data)
	s.finalizeLog(ctx, log, sendErr, channel.ID.String(), recipients[0])
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
			if config.WebhookURL != "" {
				return "slack-webhook"
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

	switch channel.Type {
	case models.ChannelTypeEmail:
		s.sendMaintenanceEmail(ctx, channel, pref, data, orgID, subject)

	case models.ChannelTypeSlack:
		var cfg models.SlackChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse slack config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("*%s*\n%s\n*Starts:* %s\n*Ends:* %s\n*Duration:* %s",
			data.Title, data.Message, data.StartsAt.Format(time.RFC822), data.EndsAt.Format(time.RFC822), data.Duration)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: "warning"}
		sendErr := s.slackSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeWebhook:
		var cfg models.WebhookChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse webhook config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), cfg.URL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		payload := WebhookPayload{EventType: string(pref.EventType), Timestamp: time.Now(), Data: data}
		sendErr := s.webhookSenderFunc(s.logger).Send(ctx, cfg.URL, payload, cfg.Secret)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.URL)

	case models.ChannelTypePagerDuty:
		var cfg models.PagerDutyChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse pagerduty config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), "pagerduty", subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		event := PagerDutyEvent{Summary: subject, Source: "keldris", Severity: "info", Group: "maintenance"}
		sendErr := s.pagerDutySenderFunc(s.logger).Send(ctx, cfg.RoutingKey, event)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), "pagerduty")

	case models.ChannelTypeTeams:
		var cfg models.TeamsChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse teams config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("**%s**\n\n%s\n\n**Starts:** %s\n\n**Ends:** %s\n\n**Duration:** %s",
			data.Title, data.Message, data.StartsAt.Format(time.RFC822), data.EndsAt.Format(time.RFC822), data.Duration)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: "warning"}
		sendErr := s.teamsSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	case models.ChannelTypeDiscord:
		var cfg models.DiscordChannelConfig
		if err := s.decryptConfig(channel.ConfigEncrypted, &cfg); err != nil {
			s.logger.Error().Err(err).Str("channel_id", channel.ID.String()).Msg("failed to parse discord config")
			return
		}
		log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), cfg.WebhookURL, subject)
		if err := s.store.CreateNotificationLog(ctx, log); err != nil {
			s.logger.Error().Err(err).Msg("failed to create notification log")
		}
		body := fmt.Sprintf("**%s**\n%s\n**Starts:** %s\n**Ends:** %s\n**Duration:** %s",
			data.Title, data.Message, data.StartsAt.Format(time.RFC822), data.EndsAt.Format(time.RFC822), data.Duration)
		msg := NotificationMessage{Title: subject, Body: body, EventType: string(pref.EventType), Severity: "warning"}
		sendErr := s.discordSenderFunc(s.logger).Send(ctx, cfg.WebhookURL, msg)
		s.finalizeLog(ctx, log, sendErr, channel.ID.String(), cfg.WebhookURL)

	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type")
		return
	}
}

// sendMaintenanceEmail sends a maintenance notification via email.
func (s *Service) sendMaintenanceEmail(ctx context.Context, channel *models.NotificationChannel, pref *models.NotificationPreference, data MaintenanceScheduledData, orgID uuid.UUID, subject string) {
	var emailConfig models.EmailChannelConfig
	if err := s.decryptConfig(channel.ConfigEncrypted, &emailConfig); err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channel.ID.String()).
			Msg("failed to parse email config")
		return
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
		s.logger.Error().Err(err).Msg("failed to create email service")
		return
	}

	recipients := []string{emailConfig.From}

	log := models.NewNotificationLog(orgID, &channel.ID, string(pref.EventType), recipients[0], subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	sendErr := emailService.SendMaintenanceScheduled(recipients, data)
	s.finalizeLog(ctx, log, sendErr, channel.ID.String(), recipients[0])
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
// finalizeLog updates a notification log with the send result.
func (s *Service) finalizeLog(ctx context.Context, log *models.NotificationLog, sendErr error, channelID, recipient string) {
	// Redact recipient if it looks like a URL (webhook URLs contain auth tokens)
	logRecipient := recipient
	if strings.HasPrefix(recipient, "http://") || strings.HasPrefix(recipient, "https://") {
		logRecipient = "[redacted-url]"
	}

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channelID).
			Str("recipient", logRecipient).
			Msg("failed to send notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channelID).
			Str("recipient", logRecipient).
			Str("event_type", log.EventType).
			Msg("notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
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
		msg := NotificationMessage{Title: "Test Notification", Body: "This is a test notification from Keldris.", EventType: "test", Severity: "info"}
		return s.slackSenderFunc(s.logger).Send(context.Background(), config.WebhookURL, msg)

	case models.ChannelTypeTeams:
		var config models.TeamsChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse teams config: %w", err)
		}
		msg := NotificationMessage{Title: "Test Notification", Body: "This is a test notification from Keldris.", EventType: "test", Severity: "info"}
		return s.teamsSenderFunc(s.logger).Send(context.Background(), config.WebhookURL, msg)

	case models.ChannelTypeDiscord:
		var config models.DiscordChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse discord config: %w", err)
		}
		msg := NotificationMessage{Title: "Test Notification", Body: "This is a test notification from Keldris.", EventType: "test", Severity: "info"}
		return s.discordSenderFunc(s.logger).Send(context.Background(), config.WebhookURL, msg)

	case models.ChannelTypePagerDuty:
		var config models.PagerDutyChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse pagerduty config: %w", err)
		}
		event := PagerDutyEvent{Summary: "Test Notification from Keldris", Source: "keldris", Severity: "info", Group: "test"}
		return s.pagerDutySenderFunc(s.logger).Send(context.Background(), config.RoutingKey, event)

	case models.ChannelTypeWebhook:
		var config models.WebhookChannelConfig
		if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
			return fmt.Errorf("parse webhook config: %w", err)
		}
		payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: map[string]string{"message": "Test notification from Keldris"}}
		return s.webhookSenderFunc(s.logger).Send(context.Background(), config.URL, payload, config.Secret)

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// NotifyTestRestoreFailed sends notifications when a test restore fails.
func (s *Service) NotifyTestRestoreFailed(ctx context.Context, result *models.TestRestoreResult, repo *models.Repository, consecutiveFails int) {
	prefs, err := s.store.GetEnabledPreferencesForEvent(ctx, repo.OrgID, models.EventTestRestoreFailed)
	if err != nil {
		s.logger.Error().Err(err).
			Str("org_id", repo.OrgID.String()).
			Str("repo_id", repo.ID.String()).
			Msg("failed to get notification preferences for test restore failed")
		return
	}

	if len(prefs) == 0 {
		s.logger.Debug().
			Str("org_id", repo.OrgID.String()).
			Str("event_type", string(models.EventTestRestoreFailed)).
			Msg("no notification preferences enabled for test restore failed event")
		return
	}

	data := TestRestoreFailedData{
		RepositoryName:   repo.Name,
		RepositoryID:     repo.ID.String(),
		SnapshotID:       result.SnapshotID,
		SamplePercentage: result.SamplePercentage,
		StartedAt:        result.StartedAt,
		FilesRestored:    result.FilesRestored,
		FilesVerified:    result.FilesVerified,
		ErrorMessage:     result.ErrorMessage,
		ConsecutiveFails: consecutiveFails,
	}
	if result.CompletedAt != nil {
		data.FailedAt = *result.CompletedAt
	}

	for _, pref := range prefs {
		go s.sendTestRestoreFailedNotification(ctx, pref, data, repo.OrgID)
	}
}

// sendTestRestoreFailedNotification sends a test restore failed notification.
func (s *Service) sendTestRestoreFailedNotification(ctx context.Context, pref *models.NotificationPreference, data TestRestoreFailedData, orgID uuid.UUID) {
	channel, err := s.store.GetNotificationChannelByID(ctx, pref.ChannelID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", pref.ChannelID.String()).
			Msg("failed to get notification channel")
		return
	}

	subject := fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName)
	recipient := s.getChannelRecipient(channel)

	log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventTestRestoreFailed), recipient, subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	var sendErr error
	switch channel.Type {
	case models.ChannelTypeEmail:
		sendErr = s.sendEmailTestRestoreFailed(channel, data)
	case models.ChannelTypeSlack:
		sendErr = s.sendSlackTestRestoreFailed(channel, data)
	case models.ChannelTypeTeams:
		sendErr = s.sendTeamsTestRestoreFailed(channel, data)
	case models.ChannelTypeDiscord:
		sendErr = s.sendDiscordTestRestoreFailed(channel, data)
	case models.ChannelTypePagerDuty:
		sendErr = s.sendPagerDutyTestRestoreFailed(channel, data)
	case models.ChannelTypeWebhook:
		sendErr = s.sendWebhookTestRestoreFailed(channel, data)
	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type for test restore failed")
		return
	}

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Msg("failed to send test restore failed notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Str("repository", data.RepositoryName).
			Msg("test restore failed notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// sendEmailTestRestoreFailed sends a test restore failed notification via email.
func (s *Service) sendEmailTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
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

	return emailService.SendTestRestoreFailed([]string{emailConfig.From}, data)
}

// sendSlackTestRestoreFailed sends a test restore failed notification via Slack.
func (s *Service) sendSlackTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
	var config models.SlackChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	slackService, err := NewSlackService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create slack service: %w", err)
	}

	return slackService.SendTestRestoreFailed(data)
}

// sendTeamsTestRestoreFailed sends a test restore failed notification via Microsoft Teams.
func (s *Service) sendTeamsTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
	var config models.TeamsChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse teams config: %w", err)
	}

	teamsService, err := NewTeamsService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create teams service: %w", err)
	}

	return teamsService.SendTestRestoreFailed(data)
}

// sendDiscordTestRestoreFailed sends a test restore failed notification via Discord.
func (s *Service) sendDiscordTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
	var config models.DiscordChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse discord config: %w", err)
	}

	discordService, err := NewDiscordService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create discord service: %w", err)
	}

	return discordService.SendTestRestoreFailed(data)
}

// sendPagerDutyTestRestoreFailed sends a test restore failed notification via PagerDuty.
func (s *Service) sendPagerDutyTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
	var config models.PagerDutyChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse pagerduty config: %w", err)
	}

	pdService, err := NewPagerDutyService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create pagerduty service: %w", err)
	}

	return pdService.SendTestRestoreFailed(data)
}

// sendWebhookTestRestoreFailed sends a test restore failed notification via generic webhook.
func (s *Service) sendWebhookTestRestoreFailed(channel *models.NotificationChannel, data TestRestoreFailedData) error {
	var config models.WebhookChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	webhookService, err := NewWebhookService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create webhook service: %w", err)
	}

	return webhookService.SendTestRestoreFailed(data)
}

// ValidationFailedResult contains information about a failed backup validation.
type ValidationFailedResult struct {
	OrgID              uuid.UUID
	ScheduleID         uuid.UUID
	ScheduleName       string
	AgentID            uuid.UUID
	Hostname           string
	SnapshotID         string
	BackupCompletedAt  time.Time
	ValidationFailedAt time.Time
	ErrorMessage       string
	ValidationSummary  string
	ValidationDetails  string
}

// NotifyValidationFailed sends notifications when a backup validation fails.
func (s *Service) NotifyValidationFailed(ctx context.Context, result ValidationFailedResult) {
	prefs, err := s.store.GetEnabledPreferencesForEvent(ctx, result.OrgID, models.EventValidationFailed)
	if err != nil {
		s.logger.Error().Err(err).
			Str("org_id", result.OrgID.String()).
			Str("schedule_id", result.ScheduleID.String()).
			Msg("failed to get notification preferences for validation failed")
		return
	}

	if len(prefs) == 0 {
		s.logger.Debug().
			Str("org_id", result.OrgID.String()).
			Str("event_type", string(models.EventValidationFailed)).
			Msg("no notification preferences enabled for validation failed event")
		return
	}

	data := ValidationFailedData{
		Hostname:           result.Hostname,
		ScheduleName:       result.ScheduleName,
		SnapshotID:         result.SnapshotID,
		BackupCompletedAt:  result.BackupCompletedAt,
		ValidationFailedAt: result.ValidationFailedAt,
		ErrorMessage:       result.ErrorMessage,
		ValidationSummary:  result.ValidationSummary,
		ValidationDetails:  result.ValidationDetails,
	}

	for _, pref := range prefs {
		go s.sendValidationFailedNotification(ctx, pref, data, result.OrgID)
	}
}

// sendValidationFailedNotification sends a validation failed notification.
func (s *Service) sendValidationFailedNotification(ctx context.Context, pref *models.NotificationPreference, data ValidationFailedData, orgID uuid.UUID) {
	channel, err := s.store.GetNotificationChannelByID(ctx, pref.ChannelID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", pref.ChannelID.String()).
			Msg("failed to get notification channel")
		return
	}

	subject := fmt.Sprintf("Backup Validation Failed: %s - %s", data.Hostname, data.ScheduleName)
	recipient := s.getChannelRecipient(channel)

	log := models.NewNotificationLog(orgID, &channel.ID, string(models.EventValidationFailed), recipient, subject)
	if err := s.store.CreateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to create notification log")
	}

	var sendErr error
	switch channel.Type {
	case models.ChannelTypeEmail:
		sendErr = s.sendEmailValidationFailed(channel, data)
	case models.ChannelTypeSlack:
		sendErr = s.sendSlackValidationFailed(channel, data)
	case models.ChannelTypeTeams:
		sendErr = s.sendTeamsValidationFailed(channel, data)
	case models.ChannelTypeDiscord:
		sendErr = s.sendDiscordValidationFailed(channel, data)
	case models.ChannelTypePagerDuty:
		sendErr = s.sendPagerDutyValidationFailed(channel, data)
	case models.ChannelTypeWebhook:
		sendErr = s.sendWebhookValidationFailed(channel, data)
	default:
		s.logger.Warn().
			Str("channel_type", string(channel.Type)).
			Msg("unsupported notification channel type for validation failed")
		return
	}

	if sendErr != nil {
		log.MarkFailed(sendErr.Error())
		s.logger.Error().Err(sendErr).
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Msg("failed to send validation failed notification")
	} else {
		log.MarkSent()
		s.logger.Info().
			Str("channel_id", channel.ID.String()).
			Str("channel_type", string(channel.Type)).
			Str("hostname", data.Hostname).
			Msg("validation failed notification sent")
	}

	if err := s.store.UpdateNotificationLog(ctx, log); err != nil {
		s.logger.Error().Err(err).Msg("failed to update notification log")
	}
}

// sendEmailValidationFailed sends a validation failed notification via email.
func (s *Service) sendEmailValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
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

	return emailService.SendValidationFailed([]string{emailConfig.From}, data)
}

// sendSlackValidationFailed sends a validation failed notification via Slack.
func (s *Service) sendSlackValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
	var config models.SlackChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	slackService, err := NewSlackService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create slack service: %w", err)
	}

	return slackService.SendValidationFailed(data)
}

// sendTeamsValidationFailed sends a validation failed notification via Microsoft Teams.
func (s *Service) sendTeamsValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
	var config models.TeamsChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse teams config: %w", err)
	}

	teamsService, err := NewTeamsService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create teams service: %w", err)
	}

	return teamsService.SendValidationFailed(data)
}

// sendDiscordValidationFailed sends a validation failed notification via Discord.
func (s *Service) sendDiscordValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
	var config models.DiscordChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse discord config: %w", err)
	}

	discordService, err := NewDiscordService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create discord service: %w", err)
	}

	return discordService.SendValidationFailed(data)
}

// sendPagerDutyValidationFailed sends a validation failed notification via PagerDuty.
func (s *Service) sendPagerDutyValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
	var config models.PagerDutyChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse pagerduty config: %w", err)
	}

	pdService, err := NewPagerDutyService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create pagerduty service: %w", err)
	}

	return pdService.SendValidationFailed(data)
}

// sendWebhookValidationFailed sends a validation failed notification via generic webhook.
func (s *Service) sendWebhookValidationFailed(channel *models.NotificationChannel, data ValidationFailedData) error {
	var config models.WebhookChannelConfig
	if err := json.Unmarshal(channel.ConfigEncrypted, &config); err != nil {
		return fmt.Errorf("parse webhook config: %w", err)
	}

	webhookService, err := NewWebhookService(config, s.logger)
	if err != nil {
		return fmt.Errorf("create webhook service: %w", err)
	}

	return webhookService.SendValidationFailed(data)
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
