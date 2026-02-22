package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// WebhookPayload represents the payload sent to generic webhook endpoints.
type WebhookPayload struct {
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ErrAirGapBlocked is returned when webhooks are blocked by air-gap mode.
var ErrAirGapBlocked = errors.New("external webhooks are disabled in air-gap mode")

// WebhookSender sends notifications via generic webhooks with HMAC signing and retry.
type WebhookSender struct {
	client      *http.Client
	logger      zerolog.Logger
	maxRetries  int
	airGapMode  bool
	validateURL func(string) error
}

// NewWebhookSender creates a new webhook sender.
func NewWebhookSender(logger zerolog.Logger) *WebhookSender {
	return &WebhookSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: ValidatingDialer(),
			},
		},
		logger:     logger.With().Str("component", "webhook_sender").Logger(),
		maxRetries: 3,
		validateURL: func(u string) error {
			return ValidateWebhookURL(u, false)
		},
// NewWebhookService creates a new generic webhook notification service.
func NewWebhookService(cfg models.WebhookChannelConfig, logger zerolog.Logger) (*WebhookService, error) {
	return NewWebhookServiceWithProxy(cfg, nil, logger)
}

// NewWebhookServiceWithProxy creates a webhook service with proxy support.
func NewWebhookServiceWithProxy(cfg models.WebhookChannelConfig, proxyConfig *config.ProxyConfig, logger zerolog.Logger) (*WebhookService, error) {
	if err := ValidateWebhookConfig(&cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Method == "" {
		cfg.Method = http.MethodPost
	}
	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}

	client, err := httpclient.New(httpclient.Options{
		Timeout:     30 * time.Second,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &WebhookService{
		config: cfg,
		client: client,
		logger: logger.With().Str("component", "webhook_service").Logger(),
	}, nil
}

// ValidateWebhookConfig validates the webhook configuration.
func ValidateWebhookConfig(config *models.WebhookChannelConfig) error {
	if config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}
}

// SetAirGapMode enables or disables air-gap mode on the webhook sender.
func (w *WebhookSender) SetAirGapMode(enabled bool) {
	w.airGapMode = enabled
}

// Send sends a webhook payload to the given URL with HMAC signature and retry.
// URLs should be validated with ValidateWebhookURL before being stored.
func (w *WebhookSender) Send(ctx context.Context, url string, payload WebhookPayload, secret string) error {
	if w.airGapMode {
		w.logger.Warn().Msg("webhook blocked: air-gap mode enabled")
		return ErrAirGapBlocked
	}

	if err := w.validateURL(url); err != nil {
		return fmt.Errorf("webhook URL blocked: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < w.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			w.logger.Debug().
				Int("attempt", attempt+1).
				Msg("retrying webhook")
		}

		lastErr = w.doSend(ctx, url, body, secret)
		if lastErr == nil {
			return nil
		}
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", w.maxRetries, lastErr)
}

// doSend performs a single webhook HTTP request.
func (w *WebhookSender) doSend(ctx context.Context, url string, body []byte, secret string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		sig := computeHMAC(body, secret)
		req.Header.Set("X-Keldris-Signature", sig)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.logger.Info().
			Int("status", resp.StatusCode).
			Msg("webhook notification sent")
		return nil
	}

	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

// computeHMAC computes an HMAC-SHA256 signature for the given payload.
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
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
