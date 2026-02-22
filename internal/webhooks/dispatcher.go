// Package webhooks provides outbound webhook dispatch functionality.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Store defines the interface for webhook persistence operations.
type Store interface {
	GetEnabledWebhookEndpointsForEvent(ctx context.Context, orgID uuid.UUID, eventType models.WebhookEventType) ([]*models.WebhookEndpoint, error)
	GetWebhookEndpointByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error)
	CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	UpdateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	GetPendingWebhookDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error)
}

// Dispatcher handles sending webhooks to registered endpoints.
type Dispatcher struct {
	store      Store
	keyManager *crypto.KeyManager
	client     *http.Client
	logger     zerolog.Logger

	// Worker pool configuration
	workerCount int
	batchSize   int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// Config holds configuration for the webhook dispatcher.
type Config struct {
	WorkerCount    int
	BatchSize      int
	RequestTimeout time.Duration
}

// DefaultConfig returns default dispatcher configuration.
func DefaultConfig() Config {
	return Config{
		WorkerCount:    5,
		BatchSize:      100,
		RequestTimeout: 30 * time.Second,
	}
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(store Store, keyManager *crypto.KeyManager, cfg Config, logger zerolog.Logger) *Dispatcher {
	return &Dispatcher{
		store:      store,
		keyManager: keyManager,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:      logger.With().Str("component", "webhook_dispatcher").Logger(),
		workerCount: cfg.WorkerCount,
		batchSize:   cfg.BatchSize,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the background retry worker.
func (d *Dispatcher) Start(ctx context.Context) {
	d.logger.Info().Int("workers", d.workerCount).Msg("starting webhook dispatcher")

	d.wg.Add(1)
	go d.retryWorker(ctx)
}

// Stop gracefully stops the dispatcher.
func (d *Dispatcher) Stop() {
	d.logger.Info().Msg("stopping webhook dispatcher")
	close(d.stopCh)
	d.wg.Wait()
}

// retryWorker processes pending webhook deliveries.
func (d *Dispatcher) retryWorker(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.processRetries(ctx)
		}
	}
}

// processRetries fetches and processes pending webhook deliveries.
func (d *Dispatcher) processRetries(ctx context.Context) {
	deliveries, err := d.store.GetPendingWebhookDeliveries(ctx, d.batchSize)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to get pending webhook deliveries")
		return
	}

	if len(deliveries) == 0 {
		return
	}

	d.logger.Debug().Int("count", len(deliveries)).Msg("processing pending webhook deliveries")

	// Process deliveries concurrently using a worker pool
	sem := make(chan struct{}, d.workerCount)
	var wg sync.WaitGroup

	for _, delivery := range deliveries {
		wg.Add(1)
		sem <- struct{}{} // Acquire worker slot

		go func(del *models.WebhookDelivery) {
			defer wg.Done()
			defer func() { <-sem }() // Release worker slot

			endpoint, err := d.store.GetWebhookEndpointByID(ctx, del.EndpointID)
			if err != nil {
				d.logger.Error().Err(err).Str("endpoint_id", del.EndpointID.String()).Msg("failed to get endpoint for retry")
				del.MarkFailed("endpoint not found")
				_ = d.store.UpdateWebhookDelivery(ctx, del)
				return
			}

			d.sendWebhook(ctx, endpoint, del)
		}(delivery)
	}

	wg.Wait()
}

// Dispatch sends a webhook event to all subscribed endpoints.
func (d *Dispatcher) Dispatch(ctx context.Context, orgID uuid.UUID, eventType models.WebhookEventType, eventID *uuid.UUID, data map[string]any) error {
	endpoints, err := d.store.GetEnabledWebhookEndpointsForEvent(ctx, orgID, eventType)
	if err != nil {
		return fmt.Errorf("get endpoints for event: %w", err)
	}

	if len(endpoints) == 0 {
		return nil
	}

	d.logger.Debug().
		Str("org_id", orgID.String()).
		Str("event_type", string(eventType)).
		Int("endpoint_count", len(endpoints)).
		Msg("dispatching webhook event")

	// Build the standard payload
	payload := d.buildPayload(orgID, eventType, eventID, data)

	// Dispatch to each endpoint asynchronously
	for _, endpoint := range endpoints {
		delivery := models.NewWebhookDelivery(orgID, endpoint.ID, eventType, eventID, payload, endpoint.RetryCount)

		if err := d.store.CreateWebhookDelivery(ctx, delivery); err != nil {
			d.logger.Error().Err(err).
				Str("endpoint_id", endpoint.ID.String()).
				Msg("failed to create webhook delivery")
			continue
		}

		// Send immediately in background
		go func(ep *models.WebhookEndpoint, del *models.WebhookDelivery) {
			d.sendWebhook(context.Background(), ep, del)
		}(endpoint, delivery)
	}

	return nil
}

// buildPayload creates the standard webhook payload.
func (d *Dispatcher) buildPayload(orgID uuid.UUID, eventType models.WebhookEventType, eventID *uuid.UUID, data map[string]any) map[string]any {
	payloadID := uuid.New().String()
	if eventID != nil {
		payloadID = eventID.String()
	}

	return map[string]any{
		"id":         payloadID,
		"event_type": string(eventType),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"org_id":     orgID.String(),
		"data":       data,
	}
}

// sendWebhook sends the webhook and updates the delivery status.
func (d *Dispatcher) sendWebhook(ctx context.Context, endpoint *models.WebhookEndpoint, delivery *models.WebhookDelivery) {
	// Serialize the payload
	payloadBytes, err := json.Marshal(delivery.Payload)
	if err != nil {
		d.logger.Error().Err(err).Str("delivery_id", delivery.ID.String()).Msg("failed to marshal payload")
		delivery.MarkFailed(fmt.Sprintf("failed to marshal payload: %v", err))
		_ = d.store.UpdateWebhookDelivery(ctx, delivery)
		return
	}

	// Decrypt the secret for HMAC signing
	secret, err := d.keyManager.Decrypt(endpoint.SecretEncrypted)
	if err != nil {
		d.logger.Error().Err(err).Str("delivery_id", delivery.ID.String()).Msg("failed to decrypt webhook secret")
		delivery.MarkFailed("failed to decrypt secret")
		_ = d.store.UpdateWebhookDelivery(ctx, delivery)
		return
	}

	// Create the signature
	signature := d.signPayload(payloadBytes, secret)

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		d.logger.Error().Err(err).Str("delivery_id", delivery.ID.String()).Msg("failed to create request")
		delivery.MarkFailed(fmt.Sprintf("failed to create request: %v", err))
		_ = d.store.UpdateWebhookDelivery(ctx, delivery)
		return
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Keldris-Webhook/1.0")
	req.Header.Set("X-Keldris-Delivery", delivery.ID.String())
	req.Header.Set("X-Keldris-Event", string(delivery.EventType))
	req.Header.Set("X-Keldris-Signature-256", signature)
	req.Header.Set("X-Keldris-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	// Add custom headers from endpoint configuration
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	// Record request headers on delivery
	delivery.RequestHeaders = make(map[string]string)
	for key := range req.Header {
		delivery.RequestHeaders[key] = req.Header.Get(key)
	}

	// Send the request
	startTime := time.Now()
	resp, err := d.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		d.handleDeliveryError(ctx, delivery, err, duration)
		return
	}
	defer resp.Body.Close()

	// Read response body (limit to 64KB)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	bodyStr := string(bodyBytes)

	// Record response headers
	responseHeaders := make(map[string]string)
	for key := range resp.Header {
		responseHeaders[key] = resp.Header.Get(key)
	}

	// Check if successful (2xx status)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.MarkDelivered(resp.StatusCode, bodyStr, responseHeaders)
		d.logger.Info().
			Str("delivery_id", delivery.ID.String()).
			Str("endpoint_id", endpoint.ID.String()).
			Int("status", resp.StatusCode).
			Dur("duration", duration).
			Msg("webhook delivered successfully")
	} else {
		d.handleDeliveryError(ctx, delivery, fmt.Errorf("unexpected status code: %d", resp.StatusCode), duration)
		delivery.ResponseStatus = &resp.StatusCode
		delivery.ResponseBody = bodyStr
		delivery.ResponseHeaders = responseHeaders
	}

	if err := d.store.UpdateWebhookDelivery(ctx, delivery); err != nil {
		d.logger.Error().Err(err).Str("delivery_id", delivery.ID.String()).Msg("failed to update delivery status")
	}
}

// handleDeliveryError handles a failed delivery attempt and schedules a retry if appropriate.
func (d *Dispatcher) handleDeliveryError(ctx context.Context, delivery *models.WebhookDelivery, err error, duration time.Duration) {
	errMsg := err.Error()

	if delivery.ShouldRetry() {
		// Calculate next retry with exponential backoff
		// Base delay: 30 seconds, doubles with each attempt
		// Attempt 1: 30s, Attempt 2: 60s, Attempt 3: 120s
		backoffSeconds := 30 * (1 << (delivery.AttemptNumber - 1))
		nextRetry := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

		delivery.MarkRetrying(errMsg, nextRetry)

		d.logger.Warn().
			Str("delivery_id", delivery.ID.String()).
			Int("attempt", delivery.AttemptNumber).
			Int("max_attempts", delivery.MaxAttempts).
			Time("next_retry", nextRetry).
			Dur("duration", duration).
			Str("error", errMsg).
			Msg("webhook delivery failed, scheduling retry")
	} else {
		delivery.MarkFailed(errMsg)

		d.logger.Error().
			Str("delivery_id", delivery.ID.String()).
			Int("attempts", delivery.AttemptNumber).
			Dur("duration", duration).
			Str("error", errMsg).
			Msg("webhook delivery failed permanently")
	}
}

// signPayload creates an HMAC-SHA256 signature for the payload.
func (d *Dispatcher) signPayload(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// TestEndpoint sends a test webhook to an endpoint and returns the result.
func (d *Dispatcher) TestEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint, eventType models.WebhookEventType) (*models.TestWebhookResponse, error) {
	// Build test payload
	payload := d.buildPayload(endpoint.OrgID, eventType, nil, map[string]any{
		"test":    true,
		"message": "This is a test webhook from Keldris",
	})

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Decrypt secret
	secret, err := d.keyManager.Decrypt(endpoint.SecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}

	signature := d.signPayload(payloadBytes, secret)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Keldris-Webhook/1.0")
	req.Header.Set("X-Keldris-Delivery", uuid.New().String())
	req.Header.Set("X-Keldris-Event", string(eventType))
	req.Header.Set("X-Keldris-Signature-256", signature)
	req.Header.Set("X-Keldris-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	startTime := time.Now()
	resp, err := d.client.Do(req)
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return &models.TestWebhookResponse{
			Success:      false,
			ErrorMessage: err.Error(),
			DurationMs:   durationMs,
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	return &models.TestWebhookResponse{
		Success:        resp.StatusCode >= 200 && resp.StatusCode < 300,
		ResponseStatus: resp.StatusCode,
		ResponseBody:   string(bodyBytes),
		DurationMs:     durationMs,
	}, nil
}

// VerifySignature verifies an incoming webhook signature.
// This can be used by external systems to verify webhooks from Keldris.
func VerifySignature(payload []byte, signature string, secret []byte) bool {
	// Create expected signature
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
