package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationChannelType represents the type of notification channel
type NotificationChannelType string

const (
	ChannelTypeEmail     NotificationChannelType = "email"
	ChannelTypeSlack     NotificationChannelType = "slack"
	ChannelTypeTeams     NotificationChannelType = "teams"
	ChannelTypeDiscord   NotificationChannelType = "discord"
	ChannelTypeWebhook   NotificationChannelType = "webhook"
	ChannelTypePagerDuty NotificationChannelType = "pagerduty"
)

// NotificationEventType represents the type of notification event
type NotificationEventType string

const (
	EventBackupSuccess        NotificationEventType = "backup_success"
	EventBackupFailed         NotificationEventType = "backup_failed"
	EventAgentOffline         NotificationEventType = "agent_offline"
	EventMaintenanceScheduled NotificationEventType = "maintenance_scheduled"
	EventTestRestoreFailed    NotificationEventType = "test_restore_failed"
	EventValidationFailed     NotificationEventType = "validation_failed"
	EventBackupSuccess NotificationEventType = "backup_success"
	EventBackupFailed  NotificationEventType = "backup_failed"
	EventAgentOffline  NotificationEventType = "agent_offline"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusQueued NotificationStatus = "queued"
	NotificationStatusSent   NotificationStatus = "sent"
	NotificationStatusFailed NotificationStatus = "failed"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel struct {
	ID              uuid.UUID               `json:"id"`
	OrgID           uuid.UUID               `json:"org_id"`
	Name            string                  `json:"name"`
	Type            NotificationChannelType `json:"type"`
	ConfigEncrypted []byte                  `json:"-"`
	Enabled         bool                    `json:"enabled"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// NewNotificationChannel creates a new notification channel
func NewNotificationChannel(orgID uuid.UUID, name string, channelType NotificationChannelType, configEncrypted []byte) *NotificationChannel {
	now := time.Now()
	return &NotificationChannel{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            name,
		Type:            channelType,
		ConfigEncrypted: configEncrypted,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NotificationPreference represents notification settings for a channel and event type
type NotificationPreference struct {
	ID        uuid.UUID             `json:"id"`
	OrgID     uuid.UUID             `json:"org_id"`
	ChannelID uuid.UUID             `json:"channel_id"`
	EventType NotificationEventType `json:"event_type"`
	Enabled   bool                  `json:"enabled"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// NewNotificationPreference creates a new notification preference
func NewNotificationPreference(orgID, channelID uuid.UUID, eventType NotificationEventType) *NotificationPreference {
	now := time.Now()
	return &NotificationPreference{
		ID:        uuid.New(),
		OrgID:     orgID,
		ChannelID: channelID,
		EventType: eventType,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NotificationLog represents a record of a sent notification
type NotificationLog struct {
	ID           uuid.UUID          `json:"id"`
	OrgID        uuid.UUID          `json:"org_id"`
	ChannelID    *uuid.UUID         `json:"channel_id,omitempty"`
	EventType    string             `json:"event_type"`
	Recipient    string             `json:"recipient"`
	Subject      string             `json:"subject,omitempty"`
	Status       NotificationStatus `json:"status"`
	ErrorMessage string             `json:"error_message,omitempty"`
	SentAt       *time.Time         `json:"sent_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
}

// NewNotificationLog creates a new notification log entry
func NewNotificationLog(orgID uuid.UUID, channelID *uuid.UUID, eventType, recipient, subject string) *NotificationLog {
	return &NotificationLog{
		ID:        uuid.New(),
		OrgID:     orgID,
		ChannelID: channelID,
		EventType: eventType,
		Recipient: recipient,
		Subject:   subject,
		Status:    NotificationStatusQueued,
		CreatedAt: time.Now(),
	}
}

// MarkSent marks the notification as sent
func (n *NotificationLog) MarkSent() {
	now := time.Now()
	n.Status = NotificationStatusSent
	n.SentAt = &now
}

// MarkFailed marks the notification as failed with an error message
func (n *NotificationLog) MarkFailed(errMsg string) {
	n.Status = NotificationStatusFailed
	n.ErrorMessage = errMsg
}

// EmailChannelConfig represents SMTP configuration for email channels
type EmailChannelConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      bool   `json:"tls"`
}

// SlackChannelConfig represents Slack webhook configuration
type SlackChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// WebhookChannelConfig represents generic webhook configuration
type WebhookChannelConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// PagerDutyChannelConfig represents PagerDuty integration configuration
type PagerDutyChannelConfig struct {
	RoutingKey string `json:"routing_key"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

// TeamsChannelConfig represents Microsoft Teams webhook configuration
type TeamsChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// DiscordChannelConfig represents Discord webhook configuration
type DiscordChannelConfig struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
}
// CreateNotificationChannelRequest represents a request to create a notification channel
type CreateNotificationChannelRequest struct {
	Name   string                  `json:"name" binding:"required"`
	Type   NotificationChannelType `json:"type" binding:"required"`
	Config interface{}             `json:"config" binding:"required"`
}

// UpdateNotificationChannelRequest represents a request to update a notification channel
type UpdateNotificationChannelRequest struct {
	Name    *string     `json:"name,omitempty"`
	Config  interface{} `json:"config,omitempty"`
	Enabled *bool       `json:"enabled,omitempty"`
}

// UpdateNotificationPreferenceRequest represents a request to update notification preferences
type UpdateNotificationPreferenceRequest struct {
	EventType NotificationEventType `json:"event_type" binding:"required"`
	Enabled   bool                  `json:"enabled"`
}

// NotificationChannelWithPreferences combines a channel with its preferences
type NotificationChannelWithPreferences struct {
	Channel     NotificationChannel      `json:"channel"`
	Preferences []NotificationPreference `json:"preferences"`
}
