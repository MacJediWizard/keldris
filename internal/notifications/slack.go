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

// NotificationMessage represents a generic notification message for non-email senders.
type NotificationMessage struct {
	Title     string
	Body      string
	EventType string
	Severity  string // info, warning, error, critical
}

// SlackSender sends notifications via Slack incoming webhooks.
type SlackSender struct {
	client *http.Client
	logger zerolog.Logger
}

// NewSlackSender creates a new Slack sender.
func NewSlackSender(logger zerolog.Logger) *SlackSender {
	return &SlackSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: ValidatingDialer(),
			},
		},
		logger: logger.With().Str("component", "slack_sender").Logger(),
	}
}

// SlackService handles sending Slack notifications via webhooks for maintenance, validation, etc.
type SlackService struct {
	config models.SlackChannelConfig
	client *http.Client
	logger zerolog.Logger
}

// SlackMessage represents a Slack webhook message with attachments.
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Fallback  string       `json:"fallback,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// NewSlackService creates a new Slack notification service.
func NewSlackService(cfg models.SlackChannelConfig, logger zerolog.Logger) (*SlackService, error) {
	return NewSlackServiceWithProxy(cfg, nil, logger)
}

// NewSlackServiceWithProxy creates a Slack notification service with proxy support.
func NewSlackServiceWithProxy(cfg models.SlackChannelConfig, proxyConfig *config.ProxyConfig, logger zerolog.Logger) (*SlackService, error) {
	if err := ValidateSlackConfig(&cfg); err != nil {
		return nil, err
	}

	client, err := httpclient.New(httpclient.Options{
		Timeout:     30 * time.Second,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &SlackService{
		config: cfg,
		client: client,
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

// slackMessage represents a Slack webhook payload with Block Kit.
type slackMessage struct {
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string         `json:"type"`
	Text *slackTextObj  `json:"text,omitempty"`
}

type slackTextObj struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// severityColor maps notification severity to Slack attachment colors.
func severityColor(severity string) string {
	switch severity {
	case "critical", "error":
		return "#dc2626" // red
	case "warning":
		return "#f59e0b" // amber
	default:
		return "#22c55e" // green
	}
}

// Send sends a notification message to a Slack webhook URL.
func (s *SlackSender) Send(ctx context.Context, webhookURL string, msg NotificationMessage) error {
	payload := slackMessage{
		Attachments: []slackAttachment{
			{
				Color: severityColor(msg.Severity),
				Blocks: []slackBlock{
					{
						Type: "header",
						Text: &slackTextObj{Type: "plain_text", Text: msg.Title},
					},
					{
						Type: "section",
						Text: &slackTextObj{Type: "mrkdwn", Text: msg.Body},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	s.logger.Info().
		Str("event_type", msg.EventType).
		Msg("slack notification sent")

	return nil
}

// Send sends a Slack message via the configured webhook URL.
func (s *SlackService) Send(msg *SlackMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal slack message: %w", err)
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SendBackupSuccess sends a backup success notification to Slack.
func (s *SlackService) SendBackupSuccess(data BackupSuccessData) error {
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#28a745", // Green
				Title:    fmt.Sprintf("Backup Successful: %s", data.Hostname),
				Fallback: fmt.Sprintf("Backup succeeded for %s - %s", data.Hostname, data.ScheduleName),
				Fields: []SlackField{
					{Title: "Host", Value: data.Hostname, Short: true},
					{Title: "Schedule", Value: data.ScheduleName, Short: true},
					{Title: "Snapshot ID", Value: data.SnapshotID, Short: true},
					{Title: "Duration", Value: data.Duration, Short: true},
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

// SendTestRestoreFailed sends a test restore failed notification to Slack.
func (s *SlackService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	fields := []SlackField{
		{Title: "Repository", Value: data.RepositoryName, Short: true},
		{Title: "Snapshot ID", Value: data.SnapshotID, Short: true},
		{Title: "Sample Size", Value: fmt.Sprintf("%d%%", data.SamplePercentage), Short: true},
		{Title: "Files Restored", Value: fmt.Sprintf("%d", data.FilesRestored), Short: true},
		{Title: "Files Verified", Value: fmt.Sprintf("%d", data.FilesVerified), Short: true},
		{Title: "Failed At", Value: data.FailedAt.Format(time.RFC822), Short: true},
		{Title: "Error", Value: data.ErrorMessage, Short: false},
	}

	if data.ConsecutiveFails > 1 {
		fields = append([]SlackField{
			{Title: "Consecutive Failures", Value: fmt.Sprintf("%d", data.ConsecutiveFails), Short: true},
		}, fields...)
	}

	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:    "#dc2626", // Red
				Title:    fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName),
				Fallback: fmt.Sprintf("Test restore failed for repository %s: %s", data.RepositoryName, data.ErrorMessage),
				Fields:   fields,
				Footer:   "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification to Slack")

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
					{Title: "Validation Failed", Value: data.ValidationFailedAt.Format(time.RFC822), Short: true},
					{Title: "Error", Value: data.ErrorMessage, Short: false},
				},
				Footer:    "Keldris Backup",
				Timestamp: time.Now().Unix(),
			},
		},
	}

	if data.ValidationSummary != "" {
		msg.Attachments[0].Fields = append(msg.Attachments[0].Fields, SlackField{
			Title: "Validation Summary",
			Value: data.ValidationSummary,
			Short: false,
		})
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
