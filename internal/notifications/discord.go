package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// DiscordService handles sending notifications via Discord webhooks.
type DiscordService struct {
	config models.DiscordChannelConfig
	client *http.Client
	logger zerolog.Logger
}

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
	return nil
}

// DiscordMessage represents a Discord webhook message.
type DiscordMessage struct {
	Content   string         `json:"content,omitempty"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed object.
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents a footer in a Discord embed.
type DiscordEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// Discord embed colors (decimal values)
const (
	DiscordColorGreen  = 3066993  // #2ECC71
	DiscordColorRed    = 15158332 // #E74C3C
	DiscordColorYellow = 16776960 // #FFFF00
	DiscordColorBlue   = 3447003  // #3498DB
)

// Send sends a message to Discord.
func (s *DiscordService) Send(msg *DiscordMessage) error {
	if msg.Username == "" && s.config.Username != "" {
		msg.Username = s.config.Username
	}
	if msg.AvatarURL == "" && s.config.AvatarURL != "" {
		msg.AvatarURL = s.config.AvatarURL
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discord message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send discord request: %w", err)
	}
	defer resp.Body.Close()

	// Discord webhooks return 204 No Content on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendBackupSuccess sends a backup success notification to Discord.
func (s *DiscordService) SendBackupSuccess(data BackupSuccessData) error {
	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Backup Successful: %s", data.Hostname),
				Description: fmt.Sprintf("Backup completed successfully for schedule **%s**", data.ScheduleName),
				Color:       DiscordColorGreen,
				Fields: []DiscordEmbedField{
					{Name: "Host", Value: data.Hostname, Inline: true},
					{Name: "Schedule", Value: data.ScheduleName, Inline: true},
					{Name: "Snapshot ID", Value: fmt.Sprintf("`%s`", data.SnapshotID), Inline: false},
					{Name: "Duration", Value: data.Duration, Inline: true},
					{Name: "Files New", Value: fmt.Sprintf("%d", data.FilesNew), Inline: true},
					{Name: "Files Changed", Value: fmt.Sprintf("%d", data.FilesChanged), Inline: true},
					{Name: "Size", Value: FormatBytes(data.SizeBytes), Inline: true},
				},
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
		Msg("sending backup success notification to Discord")

	return s.Send(msg)
}

// SendBackupFailed sends a backup failed notification to Discord.
func (s *DiscordService) SendBackupFailed(data BackupFailedData) error {
	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Backup Failed: %s", data.Hostname),
				Description: fmt.Sprintf("Backup failed for schedule **%s**", data.ScheduleName),
				Color:       DiscordColorRed,
				Fields: []DiscordEmbedField{
					{Name: "Host", Value: data.Hostname, Inline: true},
					{Name: "Schedule", Value: data.ScheduleName, Inline: true},
					{Name: "Started At", Value: data.StartedAt.Format(time.RFC822), Inline: true},
					{Name: "Failed At", Value: data.FailedAt.Format(time.RFC822), Inline: true},
					{Name: "Error", Value: fmt.Sprintf("```\n%s\n```", data.ErrorMessage), Inline: false},
				},
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
		Msg("sending backup failed notification to Discord")

	return s.Send(msg)
}

// SendAgentOffline sends an agent offline notification to Discord.
func (s *DiscordService) SendAgentOffline(data AgentOfflineData) error {
	msg := &DiscordMessage{
		Embeds: []DiscordEmbed{
			{
				Title:       fmt.Sprintf("Agent Offline: %s", data.Hostname),
				Description: fmt.Sprintf("Agent has been offline for **%s**", data.OfflineSince),
				Color:       DiscordColorYellow,
				Fields: []DiscordEmbedField{
					{Name: "Host", Value: data.Hostname, Inline: true},
					{Name: "Agent ID", Value: fmt.Sprintf("`%s`", data.AgentID), Inline: true},
					{Name: "Last Seen", Value: data.LastSeen.Format(time.RFC822), Inline: true},
					{Name: "Offline Duration", Value: data.OfflineSince, Inline: true},
				},
				Footer: &DiscordEmbedFooter{
					Text: "Keldris Backup",
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("agent_id", data.AgentID).
		Msg("sending agent offline notification to Discord")

	return s.Send(msg)
}

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
