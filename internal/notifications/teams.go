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

// TeamsSender sends notifications via Microsoft Teams incoming webhooks.
type TeamsSender struct {
	client *http.Client
	logger zerolog.Logger
}

// NewTeamsSender creates a new Teams sender.
func NewTeamsSender(logger zerolog.Logger) *TeamsSender {
	return &TeamsSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: ValidatingDialer(),
			},
		},
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
// SendMaintenanceScheduled sends a maintenance scheduled notification to Teams.
func (s *TeamsService) SendMaintenanceScheduled(data MaintenanceScheduledData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Scheduled Maintenance: %s", data.Title), "Accent")
	msg.AddText(data.Message, true)
	msg.AddFactSet([]TeamsCardFact{
		{Title: "Starts At", Value: data.StartsAt.Format(time.RFC822)},
		{Title: "Ends At", Value: data.EndsAt.Format(time.RFC822)},
		{Title: "Duration", Value: data.Duration},
	})

	s.logger.Debug().
		Str("title", data.Title).
		Time("starts_at", data.StartsAt).
		Msg("sending maintenance scheduled notification to Teams")

	return s.Send(msg)
}

// SendTestRestoreFailed sends a test restore failed notification to Teams.
func (s *TeamsService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName), "Attention")

	facts := []TeamsCardFact{
		{Title: "Repository", Value: data.RepositoryName},
		{Title: "Snapshot ID", Value: data.SnapshotID},
		{Title: "Sample Size", Value: fmt.Sprintf("%d%%", data.SamplePercentage)},
		{Title: "Files Restored", Value: fmt.Sprintf("%d", data.FilesRestored)},
		{Title: "Files Verified", Value: fmt.Sprintf("%d", data.FilesVerified)},
		{Title: "Failed At", Value: data.FailedAt.Format(time.RFC822)},
	}

	if data.ConsecutiveFails > 1 {
		facts = append([]TeamsCardFact{
			{Title: "Consecutive Failures", Value: fmt.Sprintf("%d", data.ConsecutiveFails)},
		}, facts...)
	}

	msg.AddFactSet(facts)
	msg.AddText(fmt.Sprintf("**Error:** %s", data.ErrorMessage), true)

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification to Teams")

	return s.Send(msg)
}

// SendValidationFailed sends a backup validation failed notification to Teams.
func (s *TeamsService) SendValidationFailed(data ValidationFailedData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Backup Validation Failed: %s", data.Hostname), "Attention")

	facts := []TeamsCardFact{
		{Title: "Host", Value: data.Hostname},
		{Title: "Schedule", Value: data.ScheduleName},
		{Title: "Snapshot ID", Value: data.SnapshotID},
		{Title: "Backup Completed", Value: data.BackupCompletedAt.Format(time.RFC822)},
		{Title: "Validation Failed", Value: data.ValidationFailedAt.Format(time.RFC822)},
	}

	msg.AddFactSet(facts)
	msg.AddText(fmt.Sprintf("**Error:** %s", data.ErrorMessage), true)

	if data.ValidationSummary != "" {
		msg.AddText(fmt.Sprintf("**Summary:** %s", data.ValidationSummary), true)
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending validation failed notification to Teams")

	return s.Send(msg)
}

// TestConnection sends a test message to verify the Teams webhook is working.
func (s *TeamsService) TestConnection() error {
	msg := NewTeamsMessage()
	msg.AddHeader("Keldris Backup - Test Notification", "Good")
	msg.AddText("Your Microsoft Teams integration is working correctly!", true)
	return s.Send(msg)
}
