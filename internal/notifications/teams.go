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

// TeamsService handles sending notifications via Microsoft Teams webhooks.
type TeamsService struct {
	config models.TeamsChannelConfig
	client *http.Client
	logger zerolog.Logger
}

// NewTeamsService creates a new Microsoft Teams notification service.
func NewTeamsService(config models.TeamsChannelConfig, logger zerolog.Logger) (*TeamsService, error) {
	if err := ValidateTeamsConfig(&config); err != nil {
		return nil, err
	}

	return &TeamsService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "teams_service").Logger(),
	}, nil
}

// ValidateTeamsConfig validates the Teams configuration.
func ValidateTeamsConfig(config *models.TeamsChannelConfig) error {
	if config.WebhookURL == "" {
		return fmt.Errorf("teams webhook URL is required")
	}
	return nil
}

// TeamsMessage represents a Microsoft Teams Adaptive Card message.
type TeamsMessage struct {
	Type        string            `json:"type"`
	Attachments []TeamsAttachment `json:"attachments"`
}

// TeamsAttachment represents an Adaptive Card attachment.
type TeamsAttachment struct {
	ContentType string      `json:"contentType"`
	ContentURL  interface{} `json:"contentUrl"`
	Content     TeamsCard   `json:"content"`
}

// TeamsCard represents an Adaptive Card structure.
type TeamsCard struct {
	Schema  string             `json:"$schema"`
	Type    string             `json:"type"`
	Version string             `json:"version"`
	Body    []TeamsCardElement `json:"body"`
}

// TeamsCardElement represents an element in an Adaptive Card body.
type TeamsCardElement struct {
	Type      string             `json:"type"`
	Text      string             `json:"text,omitempty"`
	Weight    string             `json:"weight,omitempty"`
	Size      string             `json:"size,omitempty"`
	Color     string             `json:"color,omitempty"`
	Wrap      bool               `json:"wrap,omitempty"`
	Spacing   string             `json:"spacing,omitempty"`
	Separator bool               `json:"separator,omitempty"`
	Facts     []TeamsCardFact    `json:"facts,omitempty"`
	Columns   []TeamsCardColumn  `json:"columns,omitempty"`
	Items     []TeamsCardElement `json:"items,omitempty"`
}

// TeamsCardFact represents a fact in a FactSet.
type TeamsCardFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// TeamsCardColumn represents a column in a ColumnSet.
type TeamsCardColumn struct {
	Type  string             `json:"type"`
	Width string             `json:"width,omitempty"`
	Items []TeamsCardElement `json:"items,omitempty"`
}

// NewTeamsMessage creates a new Teams message with the adaptive card format.
func NewTeamsMessage() *TeamsMessage {
	return &TeamsMessage{
		Type: "message",
		Attachments: []TeamsAttachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				ContentURL:  nil,
				Content: TeamsCard{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body:    []TeamsCardElement{},
				},
			},
		},
	}
}

// AddHeader adds a header element to the card.
func (m *TeamsMessage) AddHeader(text string, color string) {
	if len(m.Attachments) == 0 {
		return
	}
	element := TeamsCardElement{
		Type:   "TextBlock",
		Text:   text,
		Weight: "Bolder",
		Size:   "Medium",
		Color:  color,
	}
	m.Attachments[0].Content.Body = append(m.Attachments[0].Content.Body, element)
}

// AddText adds a text block to the card.
func (m *TeamsMessage) AddText(text string, wrap bool) {
	if len(m.Attachments) == 0 {
		return
	}
	element := TeamsCardElement{
		Type: "TextBlock",
		Text: text,
		Wrap: wrap,
	}
	m.Attachments[0].Content.Body = append(m.Attachments[0].Content.Body, element)
}

// AddFactSet adds a FactSet element to the card.
func (m *TeamsMessage) AddFactSet(facts []TeamsCardFact) {
	if len(m.Attachments) == 0 {
		return
	}
	element := TeamsCardElement{
		Type:  "FactSet",
		Facts: facts,
	}
	m.Attachments[0].Content.Body = append(m.Attachments[0].Content.Body, element)
}

// Send sends a message to Microsoft Teams.
func (s *TeamsService) Send(msg *TeamsMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal teams message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create teams request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send teams request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("teams API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendBackupSuccess sends a backup success notification to Teams.
func (s *TeamsService) SendBackupSuccess(data BackupSuccessData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Backup Successful: %s", data.Hostname), "Good")
	msg.AddFactSet([]TeamsCardFact{
		{Title: "Host", Value: data.Hostname},
		{Title: "Schedule", Value: data.ScheduleName},
		{Title: "Snapshot ID", Value: data.SnapshotID},
		{Title: "Duration", Value: data.Duration},
		{Title: "Files New", Value: fmt.Sprintf("%d", data.FilesNew)},
		{Title: "Files Changed", Value: fmt.Sprintf("%d", data.FilesChanged)},
		{Title: "Size", Value: FormatBytes(data.SizeBytes)},
		{Title: "Completed At", Value: data.CompletedAt.Format(time.RFC822)},
	})

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Msg("sending backup success notification to Teams")

	return s.Send(msg)
}

// SendBackupFailed sends a backup failed notification to Teams.
func (s *TeamsService) SendBackupFailed(data BackupFailedData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Backup Failed: %s", data.Hostname), "Attention")
	msg.AddFactSet([]TeamsCardFact{
		{Title: "Host", Value: data.Hostname},
		{Title: "Schedule", Value: data.ScheduleName},
		{Title: "Started At", Value: data.StartedAt.Format(time.RFC822)},
		{Title: "Failed At", Value: data.FailedAt.Format(time.RFC822)},
	})
	msg.AddText(fmt.Sprintf("**Error:** %s", data.ErrorMessage), true)

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending backup failed notification to Teams")

	return s.Send(msg)
}

// SendAgentOffline sends an agent offline notification to Teams.
func (s *TeamsService) SendAgentOffline(data AgentOfflineData) error {
	msg := NewTeamsMessage()
	msg.AddHeader(fmt.Sprintf("Agent Offline: %s", data.Hostname), "Warning")
	msg.AddFactSet([]TeamsCardFact{
		{Title: "Host", Value: data.Hostname},
		{Title: "Agent ID", Value: data.AgentID},
		{Title: "Last Seen", Value: data.LastSeen.Format(time.RFC822)},
		{Title: "Offline Duration", Value: data.OfflineSince},
	})

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("agent_id", data.AgentID).
		Msg("sending agent offline notification to Teams")

	return s.Send(msg)
}

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

// TestConnection sends a test message to verify the Teams webhook is working.
func (s *TeamsService) TestConnection() error {
	msg := NewTeamsMessage()
	msg.AddHeader("Keldris Backup - Test Notification", "Good")
	msg.AddText("Your Microsoft Teams integration is working correctly!", true)
	return s.Send(msg)
}
