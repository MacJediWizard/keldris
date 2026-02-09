package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

// TeamsSender sends notifications via Microsoft Teams incoming webhooks.
type TeamsSender struct {
	client *http.Client
	logger zerolog.Logger
}

// NewTeamsSender creates a new Teams sender.
func NewTeamsSender(logger zerolog.Logger) *TeamsSender {
	return &TeamsSender{
		client: &http.Client{},
		logger: logger.With().Str("component", "teams_sender").Logger(),
	}
}

// teamsAdaptiveCard represents a Teams webhook payload using Adaptive Cards.
type teamsAdaptiveCard struct {
	Type        string              `json:"type"`
	Attachments []teamsAttachment   `json:"attachments"`
}

type teamsAttachment struct {
	ContentType string            `json:"contentType"`
	ContentURL  *string           `json:"contentUrl"`
	Content     teamsCardContent  `json:"content"`
}

type teamsCardContent struct {
	Schema  string          `json:"$schema"`
	Type    string          `json:"type"`
	Version string          `json:"version"`
	Body    []teamsCardBody `json:"body"`
}

type teamsCardBody struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Size   string `json:"size,omitempty"`
	Weight string `json:"weight,omitempty"`
	Color  string `json:"color,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
}

// teamsSeverityColor maps notification severity to Adaptive Card color names.
func teamsSeverityColor(severity string) string {
	switch severity {
	case "critical", "error":
		return "attention"
	case "warning":
		return "warning"
	default:
		return "good"
	}
}

// Send sends a notification message to a Teams webhook URL.
func (t *TeamsSender) Send(ctx context.Context, webhookURL string, msg NotificationMessage) error {
	payload := teamsAdaptiveCard{
		Type: "message",
		Attachments: []teamsAttachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				ContentURL:  nil,
				Content: teamsCardContent{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body: []teamsCardBody{
						{
							Type:   "TextBlock",
							Text:   msg.Title,
							Size:   "Medium",
							Weight: "Bolder",
							Color:  teamsSeverityColor(msg.Severity),
						},
						{
							Type: "TextBlock",
							Text: msg.Body,
							Wrap: true,
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal teams payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create teams request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send teams webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("teams webhook returned status %d", resp.StatusCode)
	}

	t.logger.Info().
		Str("event_type", msg.EventType).
		Msg("teams notification sent")

	return nil
}
