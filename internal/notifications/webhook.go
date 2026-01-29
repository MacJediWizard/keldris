package notifications

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// WebhookService handles sending notifications via generic webhooks.
type WebhookService struct {
	config models.WebhookChannelConfig
	client *http.Client
	logger zerolog.Logger
}

// NewWebhookService creates a new generic webhook notification service.
func NewWebhookService(config models.WebhookChannelConfig, logger zerolog.Logger) (*WebhookService, error) {
	if err := ValidateWebhookConfig(&config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Method == "" {
		config.Method = http.MethodPost
	}
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}

	return &WebhookService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "webhook_service").Logger(),
	}, nil
}

// ValidateWebhookConfig validates the webhook configuration.
func ValidateWebhookConfig(config *models.WebhookChannelConfig) error {
	if config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	return nil
}

// WebhookPayload represents the standard payload sent to webhooks.
type WebhookPayload struct {
	EventType   string                 `json:"event_type"`
	EventTime   time.Time              `json:"event_time"`
	Source      string                 `json:"source"`
	Summary     string                 `json:"summary"`
	Severity    string                 `json:"severity"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Send sends a payload to the configured webhook.
func (s *WebhookService) Send(payload *WebhookPayload) error {
	var body []byte
	var err error

	// If a custom template is configured, use it
	if s.config.Template != "" {
		body, err = s.renderTemplate(payload)
		if err != nil {
			return fmt.Errorf("render template: %w", err)
		}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal webhook payload: %w", err)
		}
	}

	req, err := http.NewRequest(s.config.Method, s.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", s.config.ContentType)

	// Apply custom headers
	for key, value := range s.config.Headers {
		req.Header.Set(key, value)
	}

	// Apply authentication
	if err := s.applyAuth(req); err != nil {
		return fmt.Errorf("apply authentication: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Accept 2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// applyAuth applies authentication to the request based on config.
func (s *WebhookService) applyAuth(req *http.Request) error {
	switch strings.ToLower(s.config.AuthType) {
	case "bearer", "token":
		if s.config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.config.AuthToken)
		}
	case "basic":
		if s.config.BasicUser != "" {
			auth := base64.StdEncoding.EncodeToString(
				[]byte(s.config.BasicUser + ":" + s.config.BasicPass),
			)
			req.Header.Set("Authorization", "Basic "+auth)
		}
	case "header":
		// Auth token is set as a custom header - already handled by custom headers
	case "":
		// No authentication
	default:
		return fmt.Errorf("unsupported auth type: %s", s.config.AuthType)
	}
	return nil
}

// renderTemplate renders the custom template with the payload data.
func (s *WebhookService) renderTemplate(payload *WebhookPayload) ([]byte, error) {
	tmpl, err := template.New("webhook").Parse(s.config.Template)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// SendBackupSuccess sends a backup success notification via webhook.
func (s *WebhookService) SendBackupSuccess(data BackupSuccessData) error {
	payload := &WebhookPayload{
		EventType: "backup_success",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Backup Successful: %s - %s", data.Hostname, data.ScheduleName),
		Severity:  "info",
		Details: map[string]interface{}{
			"hostname":      data.Hostname,
			"schedule":      data.ScheduleName,
			"snapshot_id":   data.SnapshotID,
			"started_at":    data.StartedAt.Format(time.RFC3339),
			"completed_at":  data.CompletedAt.Format(time.RFC3339),
			"duration":      data.Duration,
			"files_new":     data.FilesNew,
			"files_changed": data.FilesChanged,
			"size_bytes":    data.SizeBytes,
			"size_human":    FormatBytes(data.SizeBytes),
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Msg("sending backup success notification via webhook")

	return s.Send(payload)
}

// SendBackupFailed sends a backup failed notification via webhook.
func (s *WebhookService) SendBackupFailed(data BackupFailedData) error {
	payload := &WebhookPayload{
		EventType: "backup_failed",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Backup Failed: %s - %s", data.Hostname, data.ScheduleName),
		Severity:  "error",
		Details: map[string]interface{}{
			"hostname":      data.Hostname,
			"schedule":      data.ScheduleName,
			"started_at":    data.StartedAt.Format(time.RFC3339),
			"failed_at":     data.FailedAt.Format(time.RFC3339),
			"error_message": data.ErrorMessage,
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending backup failed notification via webhook")

	return s.Send(payload)
}

// SendAgentOffline sends an agent offline notification via webhook.
func (s *WebhookService) SendAgentOffline(data AgentOfflineData) error {
	payload := &WebhookPayload{
		EventType: "agent_offline",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Agent Offline: %s", data.Hostname),
		Severity:  "warning",
		Details: map[string]interface{}{
			"hostname":       data.Hostname,
			"agent_id":       data.AgentID,
			"last_seen":      data.LastSeen.Format(time.RFC3339),
			"offline_since":  data.OfflineSince,
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("agent_id", data.AgentID).
		Msg("sending agent offline notification via webhook")

	return s.Send(payload)
}

// SendMaintenanceScheduled sends a maintenance scheduled notification via webhook.
func (s *WebhookService) SendMaintenanceScheduled(data MaintenanceScheduledData) error {
	payload := &WebhookPayload{
		EventType: "maintenance_scheduled",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Scheduled Maintenance: %s", data.Title),
		Severity:  "info",
		Details: map[string]interface{}{
			"title":     data.Title,
			"message":   data.Message,
			"starts_at": data.StartsAt.Format(time.RFC3339),
			"ends_at":   data.EndsAt.Format(time.RFC3339),
			"duration":  data.Duration,
		},
	}

	s.logger.Debug().
		Str("title", data.Title).
		Time("starts_at", data.StartsAt).
		Msg("sending maintenance scheduled notification via webhook")

	return s.Send(payload)
}

// SendTestRestoreFailed sends a test restore failed notification via webhook.
func (s *WebhookService) SendTestRestoreFailed(data TestRestoreFailedData) error {
	severity := "error"
	if data.ConsecutiveFails > 2 {
		severity = "critical"
	}

	payload := &WebhookPayload{
		EventType: "test_restore_failed",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Test Restore Failed: %s", data.RepositoryName),
		Severity:  severity,
		Details: map[string]interface{}{
			"repository_name":   data.RepositoryName,
			"repository_id":     data.RepositoryID,
			"snapshot_id":       data.SnapshotID,
			"sample_percentage": data.SamplePercentage,
			"files_restored":    data.FilesRestored,
			"files_verified":    data.FilesVerified,
			"started_at":        data.StartedAt.Format(time.RFC3339),
			"failed_at":         data.FailedAt.Format(time.RFC3339),
			"error_message":     data.ErrorMessage,
			"consecutive_fails": data.ConsecutiveFails,
		},
	}

	s.logger.Debug().
		Str("repository", data.RepositoryName).
		Str("error", data.ErrorMessage).
		Int("consecutive_fails", data.ConsecutiveFails).
		Msg("sending test restore failed notification via webhook")

	return s.Send(payload)
}

// SendValidationFailed sends a backup validation failed notification via webhook.
func (s *WebhookService) SendValidationFailed(data ValidationFailedData) error {
	payload := &WebhookPayload{
		EventType: "validation_failed",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   fmt.Sprintf("Backup Validation Failed: %s - %s", data.Hostname, data.ScheduleName),
		Severity:  "error",
		Details: map[string]interface{}{
			"hostname":             data.Hostname,
			"schedule":             data.ScheduleName,
			"snapshot_id":          data.SnapshotID,
			"backup_completed_at":  data.BackupCompletedAt.Format(time.RFC3339),
			"validation_failed_at": data.ValidationFailedAt.Format(time.RFC3339),
			"error_message":        data.ErrorMessage,
			"validation_summary":   data.ValidationSummary,
			"validation_details":   data.ValidationDetails,
		},
	}

	s.logger.Debug().
		Str("hostname", data.Hostname).
		Str("schedule", data.ScheduleName).
		Str("error", data.ErrorMessage).
		Msg("sending validation failed notification via webhook")

	return s.Send(payload)
}

// TestConnection sends a test event to verify the webhook is working.
func (s *WebhookService) TestConnection() error {
	payload := &WebhookPayload{
		EventType: "test",
		EventTime: time.Now().UTC(),
		Source:    "keldris-backup",
		Summary:   "Keldris Backup - Test Notification",
		Severity:  "info",
		Details: map[string]interface{}{
			"message": "Your webhook integration is working correctly!",
			"test":    true,
		},
	}
	return s.Send(payload)
}
