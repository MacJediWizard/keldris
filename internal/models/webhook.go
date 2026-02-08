package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventBackupStarted   WebhookEventType = "backup.started"
	WebhookEventBackupCompleted WebhookEventType = "backup.completed"
	WebhookEventBackupFailed    WebhookEventType = "backup.failed"
	WebhookEventAgentOnline     WebhookEventType = "agent.online"
	WebhookEventAgentOffline    WebhookEventType = "agent.offline"
	WebhookEventRestoreStarted  WebhookEventType = "restore.started"
	WebhookEventRestoreComplete WebhookEventType = "restore.completed"
	WebhookEventRestoreFailed   WebhookEventType = "restore.failed"
	WebhookEventAlertTriggered  WebhookEventType = "alert.triggered"
	WebhookEventAlertResolved   WebhookEventType = "alert.resolved"
)

// AllWebhookEventTypes returns all available webhook event types
func AllWebhookEventTypes() []WebhookEventType {
	return []WebhookEventType{
		WebhookEventBackupStarted,
		WebhookEventBackupCompleted,
		WebhookEventBackupFailed,
		WebhookEventAgentOnline,
		WebhookEventAgentOffline,
		WebhookEventRestoreStarted,
		WebhookEventRestoreComplete,
		WebhookEventRestoreFailed,
		WebhookEventAlertTriggered,
		WebhookEventAlertResolved,
	}
}

// WebhookDeliveryStatus represents the status of a webhook delivery
type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryStatusFailed    WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusRetrying  WebhookDeliveryStatus = "retrying"
)

// WebhookEndpoint represents an outbound webhook endpoint
type WebhookEndpoint struct {
	ID              uuid.UUID          `json:"id"`
	OrgID           uuid.UUID          `json:"org_id"`
	Name            string             `json:"name"`
	URL             string             `json:"url"`
	SecretEncrypted []byte             `json:"-"`
	Enabled         bool               `json:"enabled"`
	EventTypes      []WebhookEventType `json:"event_types"`
	Headers         map[string]string  `json:"headers,omitempty"`
	RetryCount      int                `json:"retry_count"`
	TimeoutSeconds  int                `json:"timeout_seconds"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// NewWebhookEndpoint creates a new webhook endpoint
func NewWebhookEndpoint(orgID uuid.UUID, name, url string, secretEncrypted []byte, eventTypes []WebhookEventType) *WebhookEndpoint {
	now := time.Now()
	return &WebhookEndpoint{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            name,
		URL:             url,
		SecretEncrypted: secretEncrypted,
		Enabled:         true,
		EventTypes:      eventTypes,
		Headers:         make(map[string]string),
		RetryCount:      3,
		TimeoutSeconds:  30,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// EventTypesJSON returns the event types as JSON bytes
func (w *WebhookEndpoint) EventTypesJSON() ([]byte, error) {
	return json.Marshal(w.EventTypes)
}

// HeadersJSON returns the headers as JSON bytes
func (w *WebhookEndpoint) HeadersJSON() ([]byte, error) {
	return json.Marshal(w.Headers)
}

// SetEventTypes sets the event types from JSON bytes
func (w *WebhookEndpoint) SetEventTypes(data []byte) error {
	if len(data) == 0 {
		w.EventTypes = []WebhookEventType{}
		return nil
	}
	return json.Unmarshal(data, &w.EventTypes)
}

// SetHeaders sets the headers from JSON bytes
func (w *WebhookEndpoint) SetHeaders(data []byte) error {
	if len(data) == 0 {
		w.Headers = make(map[string]string)
		return nil
	}
	return json.Unmarshal(data, &w.Headers)
}

// IsSubscribedTo checks if the endpoint is subscribed to an event type
func (w *WebhookEndpoint) IsSubscribedTo(eventType WebhookEventType) bool {
	for _, et := range w.EventTypes {
		if et == eventType {
			return true
		}
	}
	return false
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID              uuid.UUID             `json:"id"`
	OrgID           uuid.UUID             `json:"org_id"`
	EndpointID      uuid.UUID             `json:"endpoint_id"`
	EventType       WebhookEventType      `json:"event_type"`
	EventID         *uuid.UUID            `json:"event_id,omitempty"`
	Payload         map[string]any        `json:"payload"`
	RequestHeaders  map[string]string     `json:"request_headers,omitempty"`
	ResponseStatus  *int                  `json:"response_status,omitempty"`
	ResponseBody    string                `json:"response_body,omitempty"`
	ResponseHeaders map[string]string     `json:"response_headers,omitempty"`
	AttemptNumber   int                   `json:"attempt_number"`
	MaxAttempts     int                   `json:"max_attempts"`
	Status          WebhookDeliveryStatus `json:"status"`
	ErrorMessage    string                `json:"error_message,omitempty"`
	DeliveredAt     *time.Time            `json:"delivered_at,omitempty"`
	NextRetryAt     *time.Time            `json:"next_retry_at,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
}

// NewWebhookDelivery creates a new webhook delivery record
func NewWebhookDelivery(orgID, endpointID uuid.UUID, eventType WebhookEventType, eventID *uuid.UUID, payload map[string]any, maxAttempts int) *WebhookDelivery {
	return &WebhookDelivery{
		ID:            uuid.New(),
		OrgID:         orgID,
		EndpointID:    endpointID,
		EventType:     eventType,
		EventID:       eventID,
		Payload:       payload,
		AttemptNumber: 1,
		MaxAttempts:   maxAttempts,
		Status:        WebhookDeliveryStatusPending,
		CreatedAt:     time.Now(),
	}
}

// PayloadJSON returns the payload as JSON bytes
func (w *WebhookDelivery) PayloadJSON() ([]byte, error) {
	return json.Marshal(w.Payload)
}

// RequestHeadersJSON returns the request headers as JSON bytes
func (w *WebhookDelivery) RequestHeadersJSON() ([]byte, error) {
	if w.RequestHeaders == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(w.RequestHeaders)
}

// ResponseHeadersJSON returns the response headers as JSON bytes
func (w *WebhookDelivery) ResponseHeadersJSON() ([]byte, error) {
	if w.ResponseHeaders == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(w.ResponseHeaders)
}

// SetPayload sets the payload from JSON bytes
func (w *WebhookDelivery) SetPayload(data []byte) error {
	if len(data) == 0 {
		w.Payload = make(map[string]any)
		return nil
	}
	return json.Unmarshal(data, &w.Payload)
}

// SetRequestHeaders sets the request headers from JSON bytes
func (w *WebhookDelivery) SetRequestHeaders(data []byte) error {
	if len(data) == 0 {
		w.RequestHeaders = nil
		return nil
	}
	return json.Unmarshal(data, &w.RequestHeaders)
}

// SetResponseHeaders sets the response headers from JSON bytes
func (w *WebhookDelivery) SetResponseHeaders(data []byte) error {
	if len(data) == 0 {
		w.ResponseHeaders = nil
		return nil
	}
	return json.Unmarshal(data, &w.ResponseHeaders)
}

// MarkDelivered marks the delivery as successful
func (w *WebhookDelivery) MarkDelivered(status int, body string, headers map[string]string) {
	now := time.Now()
	w.Status = WebhookDeliveryStatusDelivered
	w.ResponseStatus = &status
	w.ResponseBody = body
	w.ResponseHeaders = headers
	w.DeliveredAt = &now
}

// MarkFailed marks the delivery as failed
func (w *WebhookDelivery) MarkFailed(errMsg string) {
	w.Status = WebhookDeliveryStatusFailed
	w.ErrorMessage = errMsg
}

// MarkRetrying marks the delivery for retry with exponential backoff
func (w *WebhookDelivery) MarkRetrying(errMsg string, nextRetry time.Time) {
	w.Status = WebhookDeliveryStatusRetrying
	w.ErrorMessage = errMsg
	w.NextRetryAt = &nextRetry
	w.AttemptNumber++
}

// ShouldRetry returns true if the delivery should be retried
func (w *WebhookDelivery) ShouldRetry() bool {
	return w.AttemptNumber < w.MaxAttempts
}

// WebhookPayload represents the standard payload sent to webhook endpoints
type WebhookPayload struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	Timestamp time.Time      `json:"timestamp"`
	OrgID     string         `json:"org_id"`
	Data      map[string]any `json:"data"`
}

// CreateWebhookEndpointRequest represents a request to create a webhook endpoint
type CreateWebhookEndpointRequest struct {
	Name           string             `json:"name" binding:"required"`
	URL            string             `json:"url" binding:"required,url"`
	Secret         string             `json:"secret" binding:"required,min=16"`
	EventTypes     []WebhookEventType `json:"event_types" binding:"required,min=1"`
	Headers        map[string]string  `json:"headers,omitempty"`
	RetryCount     *int               `json:"retry_count,omitempty"`
	TimeoutSeconds *int               `json:"timeout_seconds,omitempty"`
}

// UpdateWebhookEndpointRequest represents a request to update a webhook endpoint
type UpdateWebhookEndpointRequest struct {
	Name           *string            `json:"name,omitempty"`
	URL            *string            `json:"url,omitempty"`
	Secret         *string            `json:"secret,omitempty"`
	Enabled        *bool              `json:"enabled,omitempty"`
	EventTypes     []WebhookEventType `json:"event_types,omitempty"`
	Headers        map[string]string  `json:"headers,omitempty"`
	RetryCount     *int               `json:"retry_count,omitempty"`
	TimeoutSeconds *int               `json:"timeout_seconds,omitempty"`
}

// WebhookEndpointsResponse represents a list of webhook endpoints
type WebhookEndpointsResponse struct {
	Endpoints []*WebhookEndpoint `json:"endpoints"`
}

// WebhookDeliveriesResponse represents a list of webhook deliveries
type WebhookDeliveriesResponse struct {
	Deliveries []*WebhookDelivery `json:"deliveries"`
	Total      int                `json:"total"`
}

// WebhookEventTypesResponse represents available event types
type WebhookEventTypesResponse struct {
	EventTypes []WebhookEventType `json:"event_types"`
}

// TestWebhookRequest represents a request to test a webhook endpoint
type TestWebhookRequest struct {
	EventType WebhookEventType `json:"event_type,omitempty"`
}

// TestWebhookResponse represents the result of a webhook test
type TestWebhookResponse struct {
	Success        bool   `json:"success"`
	ResponseStatus int    `json:"response_status,omitempty"`
	ResponseBody   string `json:"response_body,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	DurationMs     int64  `json:"duration_ms"`
}
