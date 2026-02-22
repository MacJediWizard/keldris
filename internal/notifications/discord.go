package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// DiscordSender sends notifications via Discord webhooks.
type DiscordSender struct {
	client *http.Client
	logger zerolog.Logger
}

// NewDiscordSender creates a new Discord sender.
func NewDiscordSender(logger zerolog.Logger) *DiscordSender {
	return &DiscordSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: ValidatingDialer(),
			},
		},
		logger: logger.With().Str("component", "discord_sender").Logger(),
// NewDiscordService creates a new Discord notification service.
func NewDiscordService(cfg models.DiscordChannelConfig, logger zerolog.Logger) (*DiscordService, error) {
	return NewDiscordServiceWithProxy(cfg, nil, logger)
}

// NewDiscordServiceWithProxy creates a Discord notification service with proxy support.
func NewDiscordServiceWithProxy(cfg models.DiscordChannelConfig, proxyConfig *config.ProxyConfig, logger zerolog.Logger) (*DiscordService, error) {
	if err := ValidateDiscordConfig(&cfg); err != nil {
		return nil, err
	}

	client, err := httpclient.New(httpclient.Options{
		Timeout:     30 * time.Second,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &DiscordService{
		config: cfg,
		client: client,
		logger: logger.With().Str("component", "discord_service").Logger(),
	}, nil
}

// ValidateDiscordConfig validates the Discord configuration.
func ValidateDiscordConfig(config *models.DiscordChannelConfig) error {
	if config.WebhookURL == "" {
		return fmt.Errorf("discord webhook URL is required")
	}
}

// discordWebhookPayload represents a Discord webhook payload with embeds.
type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// discordSeverityColor maps notification severity to Discord embed colors (decimal).
func discordSeverityColor(severity string) int {
	switch severity {
	case "critical", "error":
		return 0xdc2626 // red
	case "warning":
		return 0xf59e0b // amber
	default:
		return 0x22c55e // green
	}
}

// Send sends a notification message to a Discord webhook URL.
func (d *DiscordSender) Send(ctx context.Context, webhookURL string, msg NotificationMessage) error {
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{
			{
				Title:       msg.Title,
				Description: msg.Body,
				Color:       discordSeverityColor(msg.Severity),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send discord webhook: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	d.logger.Info().
		Str("event_type", msg.EventType).
		Msg("discord notification sent")

	return nil
// SendMaintenanceScheduled sends a maintenance scheduled notification to Discord.
func (s *DiscordService) SendMaintenanceScheduled(data MaintenanceScheduledData) error {
	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Scheduled Maintenance: %s", data.Title),
				Description: data.Message,
				Color:       DiscordColorBlue,
				Fields: []DiscordEmbedField{
					{Name: "Starts At", Value: data.StartsAt.Format(time.RFC822), Inline: true},
					{Name: "Ends At", Value: data.EndsAt.Format(time.RFC822), Inline: true},
					{Name: "Duration", Value: data.Duration, Inline: true},
				},
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	s.logger.Debug().
		Str("title", data.Title).
		Time("starts_at", data.StartsAt).
		Msg("sending maintenance scheduled notification to Discord")

	return s.Send(msg)
}

// SendTestRestoreFailed sends a test restore failed notification to Discord.
func (s *DiscordService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	fields := []DiscordEmbedField{
		{Name: "Repository", Value: data.RepositoryName, Inline: true},
		{Name: "Snapshot ID", Value: fmt.Sprintf("`%s`", data.SnapshotID), Inline: true},
		{Name: "Sample Size", Value: fmt.Sprintf("%d%%", data.SamplePercentage), Inline: true},
		{Name: "Files Restored", Value: fmt.Sprintf("%d", data.FilesRestored), Inline: true},
		{Name: "Files Verified", Value: fmt.Sprintf("%d", data.FilesVerified), Inline: true},
		{Name: "Failed At", Value: data.FailedAt.Format(time.RFC822), Inline: true},
		{Name: "Error", Value: fmt.Sprintf("```\n%s\n```", data.ErrorMessage), Inline: false},
	}

	description := "A scheduled test restore has failed"
	if data.ConsecutiveFails > 1 {
		description = fmt.Sprintf("**Warning:** %d consecutive test restore failures", data.ConsecutiveFails)
	}

	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName),
				Description: description,
				Color:       DiscordColorRed,
				Fields:      fields,
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification to Discord")

	return s.Send(msg)
}

// SendValidationFailed sends a backup validation failed notification to Discord.
func (s *DiscordService) SendValidationFailed(data ValidationFailedData) error {
	fields := []DiscordEmbedField{
		{Name: "Host", Value: data.Hostname, Inline: true},
		{Name: "Schedule", Value: data.ScheduleName, Inline: true},
		{Name: "Snapshot ID", Value: fmt.Sprintf("`%s`", data.SnapshotID), Inline: false},
		{Name: "Backup Completed", Value: data.BackupCompletedAt.Format(time.RFC822), Inline: true},
		{Name: "Validation Failed", Value: data.ValidationFailedAt.Format(time.RFC822), Inline: true},
		{Name: "Error", Value: fmt.Sprintf("```\n%s\n```", data.ErrorMessage), Inline: false},
	}

	if data.ValidationSummary != "" {
		fields = append(fields, DiscordEmbedField{
			Name:   "Validation Summary",
			Value:  data.ValidationSummary,
			Inline: false,
		})
	}

	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Backup Validation Failed: %s", data.Hostname),
				Description: fmt.Sprintf("Backup validation failed for schedule **%s**", data.ScheduleName),
				Color:       DiscordColorRed,
				Fields:      fields,
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending validation failed notification to Discord")

	return s.Send(msg)
}

// SendTestRestoreFailed sends a test restore failed notification to Discord.
func (s *DiscordService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	fields := []DiscordEmbedField{
		{Name: "Repository", Value: data.RepositoryName, Inline: true},
		{Name: "Snapshot ID", Value: fmt.Sprintf("`%s`", data.SnapshotID), Inline: true},
		{Name: "Sample Size", Value: fmt.Sprintf("%d%%", data.SamplePercentage), Inline: true},
		{Name: "Files Restored", Value: fmt.Sprintf("%d", data.FilesRestored), Inline: true},
		{Name: "Files Verified", Value: fmt.Sprintf("%d", data.FilesVerified), Inline: true},
		{Name: "Failed At", Value: data.FailedAt.Format(time.RFC822), Inline: true},
		{Name: "Error", Value: fmt.Sprintf("```\n%s\n```", data.ErrorMessage), Inline: false},
	}

	description := "A scheduled test restore has failed"
	if data.ConsecutiveFails > 1 {
		description = fmt.Sprintf("**Warning:** %d consecutive test restore failures", data.ConsecutiveFails)
	}

	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName),
				Description: description,
				Color:       DiscordColorRed,
				Fields:      fields,
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification to Discord")

	return s.Send(msg)
}

// TestConnection sends a test message to verify the Discord webhook is working.
func (s *DiscordService) TestConnection() error {
	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       "Keldris Backup - Test Notification",
				Description: "Your Discord integration is working correctly!",
				Color:       DiscordColorGreen,
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	return s.Send(msg)
}
