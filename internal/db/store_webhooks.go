package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Webhook Endpoints methods

// GetWebhookEndpointsByOrgID returns all webhook endpoints for an organization.
func (db *DB) GetWebhookEndpointsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.WebhookEndpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, url, secret_encrypted, enabled, event_types,
		       headers, retry_count, timeout_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get webhook endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*models.WebhookEndpoint
	for rows.Next() {
		e, err := scanWebhookEndpoint(rows)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhook endpoints: %w", err)
	}
	return endpoints, nil
}

// GetWebhookEndpointByID returns a webhook endpoint by ID.
func (db *DB) GetWebhookEndpointByID(ctx context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, url, secret_encrypted, enabled, event_types,
		       headers, retry_count, timeout_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get webhook endpoint: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("webhook endpoint not found")
	}
	return scanWebhookEndpoint(rows)
}

// GetEnabledWebhookEndpointsForEvent returns all enabled endpoints subscribed to an event type.
func (db *DB) GetEnabledWebhookEndpointsForEvent(ctx context.Context, orgID uuid.UUID, eventType models.WebhookEventType) ([]*models.WebhookEndpoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, url, secret_encrypted, enabled, event_types,
		       headers, retry_count, timeout_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE org_id = $1 AND enabled = true AND event_types @> $2::jsonb
		ORDER BY name
	`, orgID, fmt.Sprintf(`["%s"]`, eventType))
	if err != nil {
		return nil, fmt.Errorf("get enabled webhook endpoints for event: %w", err)
	}
	defer rows.Close()

	var endpoints []*models.WebhookEndpoint
	for rows.Next() {
		e, err := scanWebhookEndpoint(rows)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhook endpoints: %w", err)
	}
	return endpoints, nil
}

// CreateWebhookEndpoint creates a new webhook endpoint.
func (db *DB) CreateWebhookEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	eventTypesJSON, err := endpoint.EventTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal event types: %w", err)
	}
	headersJSON, err := endpoint.HeadersJSON()
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO webhook_endpoints (id, org_id, name, url, secret_encrypted, enabled,
		                               event_types, headers, retry_count, timeout_seconds,
		                               created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, endpoint.ID, endpoint.OrgID, endpoint.Name, endpoint.URL, endpoint.SecretEncrypted,
		endpoint.Enabled, eventTypesJSON, headersJSON, endpoint.RetryCount,
		endpoint.TimeoutSeconds, endpoint.CreatedAt, endpoint.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create webhook endpoint: %w", err)
	}
	return nil
}

// UpdateWebhookEndpoint updates an existing webhook endpoint.
func (db *DB) UpdateWebhookEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	endpoint.UpdatedAt = time.Now()

	eventTypesJSON, err := endpoint.EventTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal event types: %w", err)
	}
	headersJSON, err := endpoint.HeadersJSON()
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE webhook_endpoints
		SET name = $2, url = $3, secret_encrypted = $4, enabled = $5, event_types = $6,
		    headers = $7, retry_count = $8, timeout_seconds = $9, updated_at = $10
		WHERE id = $1
	`, endpoint.ID, endpoint.Name, endpoint.URL, endpoint.SecretEncrypted, endpoint.Enabled,
		eventTypesJSON, headersJSON, endpoint.RetryCount, endpoint.TimeoutSeconds, endpoint.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update webhook endpoint: %w", err)
	}
	return nil
}

// DeleteWebhookEndpoint deletes a webhook endpoint.
func (db *DB) DeleteWebhookEndpoint(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM webhook_endpoints WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete webhook endpoint: %w", err)
	}
	return nil
}

// scanWebhookEndpoint scans a row into a WebhookEndpoint.
func scanWebhookEndpoint(rows interface{ Scan(dest ...any) error }) (*models.WebhookEndpoint, error) {
	var e models.WebhookEndpoint
	var eventTypesBytes, headersBytes []byte

	err := rows.Scan(
		&e.ID, &e.OrgID, &e.Name, &e.URL, &e.SecretEncrypted, &e.Enabled,
		&eventTypesBytes, &headersBytes, &e.RetryCount, &e.TimeoutSeconds,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan webhook endpoint: %w", err)
	}

	if err := e.SetEventTypes(eventTypesBytes); err != nil {
		return nil, fmt.Errorf("parse event types: %w", err)
	}
	if err := e.SetHeaders(headersBytes); err != nil {
		return nil, fmt.Errorf("parse headers: %w", err)
	}

	return &e, nil
}

// Webhook Deliveries methods

// GetWebhookDeliveriesByEndpointID returns deliveries for an endpoint with pagination.
func (db *DB) GetWebhookDeliveriesByEndpointID(ctx context.Context, endpointID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, int, error) {
	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM webhook_deliveries WHERE endpoint_id = $1
	`, endpointID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, endpoint_id, event_type, event_id, payload, request_headers,
		       response_status, response_body, response_headers, attempt_number, max_attempts,
		       status, error_message, delivered_at, next_retry_at, created_at
		FROM webhook_deliveries
		WHERE endpoint_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, endpointID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		d, err := scanWebhookDelivery(rows)
		if err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook deliveries: %w", err)
	}
	return deliveries, total, nil
}

// GetWebhookDeliveriesByOrgID returns all deliveries for an organization with pagination.
func (db *DB) GetWebhookDeliveriesByOrgID(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.WebhookDelivery, int, error) {
	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM webhook_deliveries WHERE org_id = $1
	`, orgID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, endpoint_id, event_type, event_id, payload, request_headers,
		       response_status, response_body, response_headers, attempt_number, max_attempts,
		       status, error_message, delivered_at, next_retry_at, created_at
		FROM webhook_deliveries
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		d, err := scanWebhookDelivery(rows)
		if err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate webhook deliveries: %w", err)
	}
	return deliveries, total, nil
}

// GetWebhookDeliveryByID returns a webhook delivery by ID.
func (db *DB) GetWebhookDeliveryByID(ctx context.Context, id uuid.UUID) (*models.WebhookDelivery, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, endpoint_id, event_type, event_id, payload, request_headers,
		       response_status, response_body, response_headers, attempt_number, max_attempts,
		       status, error_message, delivered_at, next_retry_at, created_at
		FROM webhook_deliveries
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get webhook delivery: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("webhook delivery not found")
	}
	return scanWebhookDelivery(rows)
}

// GetPendingWebhookDeliveries returns deliveries that need to be retried.
func (db *DB) GetPendingWebhookDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, endpoint_id, event_type, event_id, payload, request_headers,
		       response_status, response_body, response_headers, attempt_number, max_attempts,
		       status, error_message, delivered_at, next_retry_at, created_at
		FROM webhook_deliveries
		WHERE status IN ('pending', 'retrying')
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		d, err := scanWebhookDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending webhook deliveries: %w", err)
	}
	return deliveries, nil
}

// CreateWebhookDelivery creates a new webhook delivery record.
func (db *DB) CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	payloadJSON, err := delivery.PayloadJSON()
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	requestHeadersJSON, err := delivery.RequestHeadersJSON()
	if err != nil {
		return fmt.Errorf("marshal request headers: %w", err)
	}
	responseHeadersJSON, err := delivery.ResponseHeadersJSON()
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO webhook_deliveries (id, org_id, endpoint_id, event_type, event_id, payload,
		                                request_headers, response_status, response_body,
		                                response_headers, attempt_number, max_attempts, status,
		                                error_message, delivered_at, next_retry_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, delivery.ID, delivery.OrgID, delivery.EndpointID, delivery.EventType, delivery.EventID,
		payloadJSON, requestHeadersJSON, delivery.ResponseStatus, delivery.ResponseBody,
		responseHeadersJSON, delivery.AttemptNumber, delivery.MaxAttempts, delivery.Status,
		delivery.ErrorMessage, delivery.DeliveredAt, delivery.NextRetryAt, delivery.CreatedAt)
	if err != nil {
		return fmt.Errorf("create webhook delivery: %w", err)
	}
	return nil
}

// UpdateWebhookDelivery updates an existing webhook delivery.
func (db *DB) UpdateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	responseHeadersJSON, err := delivery.ResponseHeadersJSON()
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET response_status = $2, response_body = $3, response_headers = $4, attempt_number = $5,
		    status = $6, error_message = $7, delivered_at = $8, next_retry_at = $9
		WHERE id = $1
	`, delivery.ID, delivery.ResponseStatus, delivery.ResponseBody, responseHeadersJSON,
		delivery.AttemptNumber, delivery.Status, delivery.ErrorMessage, delivery.DeliveredAt,
		delivery.NextRetryAt)
	if err != nil {
		return fmt.Errorf("update webhook delivery: %w", err)
	}
	return nil
}

// DeleteOldWebhookDeliveries deletes deliveries older than the specified duration.
func (db *DB) DeleteOldWebhookDeliveries(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM webhook_deliveries WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old webhook deliveries: %w", err)
	}
	return result.RowsAffected(), nil
}

// scanWebhookDelivery scans a row into a WebhookDelivery.
func scanWebhookDelivery(rows interface{ Scan(dest ...any) error }) (*models.WebhookDelivery, error) {
	var d models.WebhookDelivery
	var eventID *uuid.UUID
	var payloadBytes, requestHeadersBytes, responseHeadersBytes []byte
	var responseStatus *int
	var responseBody, errorMessage *string
	var deliveredAt, nextRetryAt *time.Time
	var statusStr, eventTypeStr string

	err := rows.Scan(
		&d.ID, &d.OrgID, &d.EndpointID, &eventTypeStr, &eventID, &payloadBytes,
		&requestHeadersBytes, &responseStatus, &responseBody, &responseHeadersBytes,
		&d.AttemptNumber, &d.MaxAttempts, &statusStr, &errorMessage, &deliveredAt,
		&nextRetryAt, &d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan webhook delivery: %w", err)
	}

	d.EventType = models.WebhookEventType(eventTypeStr)
	d.Status = models.WebhookDeliveryStatus(statusStr)
	d.EventID = eventID
	d.ResponseStatus = responseStatus
	d.DeliveredAt = deliveredAt
	d.NextRetryAt = nextRetryAt
	if responseBody != nil {
		d.ResponseBody = *responseBody
	}
	if errorMessage != nil {
		d.ErrorMessage = *errorMessage
	}

	if err := d.SetPayload(payloadBytes); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if err := d.SetRequestHeaders(requestHeadersBytes); err != nil {
		return nil, fmt.Errorf("parse request headers: %w", err)
	}
	if err := d.SetResponseHeaders(responseHeadersBytes); err != nil {
		return nil, fmt.Errorf("parse response headers: %w", err)
	}

	return &d, nil
}
