package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
