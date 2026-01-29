package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// SlackService handles sending notifications via Slack webhooks.
type SlackService struct {
	config models.SlackChannelConfig
	client *http.Client
	logger zerolog.Logger
}

// NewSlackService creates a new Slack notification service.
func NewSlackService(config models.SlackChannelConfig, logger zerolog.Logger) (*SlackService, error) {
	if err := ValidateSlackConfig(&config); err != nil {
		return nil, err
	}

	return &SlackService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "slack_service").Logger(),
	}, nil
}

// ValidateSlackConfig validates the Slack configuration.
func ValidateSlackConfig(config *models.SlackChannelConfig) error {
	if config.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is required")
	}
	return nil
}

// SlackMessage represents a Slack message payload.
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fallback   string       `json:"fallback,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// SlackBlock represents a Slack Block Kit element.
type SlackBlock struct {
	Type     string      `json:"type"`
	Text     *SlackText  `json:"text,omitempty"`
	Elements interface{} `json:"elements,omitempty"`
}

// SlackText represents text in a Slack block.
type SlackText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// Send sends a message to Slack.
func (s *SlackService) Send(msg *SlackMessage) error {
	if msg.Channel == "" && s.config.Channel != "" {
		msg.Channel = s.config.Channel
	}
	if msg.Username == "" && s.config.Username != "" {
		msg.Username = s.config.Username
	}
	if msg.IconEmoji == "" && s.config.IconEmoji != "" {
		msg.IconEmoji = s.config.IconEmoji
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal slack message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendBackupSuccess sends a backup success notification to Slack.
func (s *SlackService) SendBackupSuccess(data BackupSuccessData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#36a64f", // Green
				Title:    fmt.Sprintf("Backup Successful: %s", data.Hostname),
				Fallback: fmt.Sprintf("Backup completed successfully for %s - %s", data.Hostname, data.ScheduleName),
				Fields: []SlackField{
					{Title: "Host", Value: data.Hostname, Short: true},
					{Title: "Schedule", Value: data.ScheduleName, Short: true},
					{Title: "Snapshot ID", Value: data.SnapshotID, Short: true},
					{Title: "Duration", Value: data.Duration, Short: true},
					{Title: "Files New", Value: fmt.Sprintf("%d", data.FilesNew), Short: true},
					{Title: "Files Changed", Value: fmt.Sprintf("%d", data.FilesChanged), Short: true},
					{Title: "Size", Value: FormatBytes(data.SizeBytes), Short: true},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Msg("sending backup success notification to Slack")

	return s.Send(msg)
}

// SendBackupFailed sends a backup failed notification to Slack.
func (s *SlackService) SendBackupFailed(data BackupFailedData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#dc3545", // Red
				Title:    fmt.Sprintf("Backup Failed: %s", data.Hostname),
				Fallback: fmt.Sprintf("Backup failed for %s - %s: %s", data.Hostname, data.ScheduleName, data.ErrorMessage),
				Fields: []SlackField{
					{Title: "Host", Value: data.Hostname, Short: true},
					{Title: "Schedule", Value: data.ScheduleName, Short: true},
					{Title: "Started At", Value: data.StartedAt.Format(time.RFC822), Short: true},
					{Title: "Failed At", Value: data.FailedAt.Format(time.RFC822), Short: true},
					{Title: "Error", Value: data.ErrorMessage, Short: false},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending backup failed notification to Slack")

	return s.Send(msg)
}

// SendAgentOffline sends an agent offline notification to Slack.
func (s *SlackService) SendAgentOffline(data AgentOfflineData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#ffc107", // Yellow/Warning
				Title:    fmt.Sprintf("Agent Offline: %s", data.Hostname),
				Fallback: fmt.Sprintf("Agent %s has been offline for %s", data.Hostname, data.OfflineSince),
				Fields: []SlackField{
					{Title: "Host", Value: data.Hostname, Short: true},
					{Title: "Agent ID", Value: data.AgentID, Short: true},
					{Title: "Last Seen", Value: data.LastSeen.Format(time.RFC822), Short: true},
					{Title: "Offline Duration", Value: data.OfflineSince, Short: true},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("agent_id", data.AgentID).
		Msg("sending agent offline notification to Slack")

	return s.Send(msg)
}

// SendMaintenanceScheduled sends a maintenance scheduled notification to Slack.
func (s *SlackService) SendMaintenanceScheduled(data MaintenanceScheduledData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#17a2b8", // Blue/Info
				Title:    fmt.Sprintf("Scheduled Maintenance: %s", data.Title),
				Fallback: fmt.Sprintf("Scheduled maintenance: %s from %s to %s", data.Title, data.StartsAt.Format(time.RFC822), data.EndsAt.Format(time.RFC822)),
				Text:     data.Message,
				Fields: []SlackField{
					{Title: "Starts At", Value: data.StartsAt.Format(time.RFC822), Short: true},
					{Title: "Ends At", Value: data.EndsAt.Format(time.RFC822), Short: true},
					{Title: "Duration", Value: data.Duration, Short: true},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("title", data.Title).
		Time("starts_at", data.StartsAt).
		Msg("sending maintenance scheduled notification to Slack")

	return s.Send(msg)
}

// SendValidationFailed sends a backup validation failed notification to Slack.
func (s *SlackService) SendValidationFailed(data ValidationFailedData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#dc3545", // Red
				Title:    fmt.Sprintf("Backup Validation Failed: %s", data.Hostname),
				Fallback: fmt.Sprintf("Backup validation failed for %s - %s: %s", data.Hostname, data.ScheduleName, data.ErrorMessage),
				Fields: []SlackField{
					{Title: "Host", Value: data.Hostname, Short: true},
					{Title: "Schedule", Value: data.ScheduleName, Short: true},
					{Title: "Snapshot ID", Value: data.SnapshotID, Short: true},
					{Title: "Backup Completed", Value: data.BackupCompletedAt.Format(time.RFC822), Short: true},
					{Title: "Validation Summary", Value: data.ValidationSummary, Short: false},
					{Title: "Error", Value: data.ErrorMessage, Short: false},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending validation failed notification to Slack")

	return s.Send(msg)
}

// TestConnection sends a test message to verify the Slack webhook is working.
func (s *SlackService) TestConnection() error {
	msg := &SlackMessage{
		Text: "Test notification from Keldris Backup - your Slack integration is working!",
	}
	return s.Send(msg)
}
