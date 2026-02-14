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
}
