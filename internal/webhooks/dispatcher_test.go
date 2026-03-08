package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// --- Mock Store ---

type mockWebhookStore struct {
	mu        sync.Mutex
	endpoints []*models.WebhookEndpoint
	pending   []*models.WebhookDelivery

	getEndpointsErr  error
	getEndpointErr   error
	createErr        error
	updateErr        error
	getPendingErr    error

	// Track calls for assertions
	createdDeliveries []*models.WebhookDelivery
	updatedDeliveries []*models.WebhookDelivery
}

func (m *mockWebhookStore) GetEnabledWebhookEndpointsForEvent(_ context.Context, _ uuid.UUID, _ models.WebhookEventType) ([]*models.WebhookEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getEndpointsErr != nil {
		return nil, m.getEndpointsErr
	}
	return m.endpoints, nil
}

func (m *mockWebhookStore) GetWebhookEndpointByID(_ context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getEndpointErr != nil {
		return nil, m.getEndpointErr
	}
	for _, ep := range m.endpoints {
		if ep.ID == id {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("endpoint not found: %s", id)
}

func (m *mockWebhookStore) CreateWebhookDelivery(_ context.Context, delivery *models.WebhookDelivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.createdDeliveries = append(m.createdDeliveries, delivery)
	return nil
}

func (m *mockWebhookStore) UpdateWebhookDelivery(_ context.Context, delivery *models.WebhookDelivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedDeliveries = append(m.updatedDeliveries, delivery)
	return nil
}

func (m *mockWebhookStore) GetPendingWebhookDeliveries(_ context.Context, _ int) ([]*models.WebhookDelivery, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getPendingErr != nil {
		return nil, m.getPendingErr
	}
	return m.pending, nil
}

// --- Test Helpers ---

func testKeyManager(t *testing.T) *crypto.KeyManager {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	km, err := crypto.NewKeyManager(key)
	if err != nil {
		t.Fatalf("failed to create key manager: %v", err)
	}
	return km
}

func encryptSecret(t *testing.T, km *crypto.KeyManager, secret string) []byte {
	t.Helper()
	encrypted, err := km.Encrypt([]byte(secret))
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}
	return encrypted
}

func newTestDispatcher(store Store, km *crypto.KeyManager) *Dispatcher {
	cfg := Config{
		WorkerCount:    2,
		BatchSize:      10,
		RequestTimeout: 5 * time.Second,
	}
	return NewDispatcher(store, km, cfg, zerolog.Nop())
}

func makeEndpoint(t *testing.T, km *crypto.KeyManager, url, secret string) *models.WebhookEndpoint {
	t.Helper()
	return &models.WebhookEndpoint{
		ID:              uuid.New(),
		OrgID:           uuid.New(),
		Name:            "test-endpoint",
		URL:             url,
		SecretEncrypted: encryptSecret(t, km, secret),
		Enabled:         true,
		EventTypes:      []models.WebhookEventType{models.WebhookEventBackupCompleted},
		Headers:         map[string]string{},
		RetryCount:      3,
		TimeoutSeconds:  30,
	}
}

// --- Webhook Delivery Tests ---

func TestSendWebhook_Success(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	km := testKeyManager(t)
	secret := "test-webhook-secret-key"
	endpoint := makeEndpoint(t, km, srv.URL, secret)

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{
		"id":         uuid.New().String(),
		"event_type": "backup.completed",
		"data":       map[string]any{"backup_id": "abc"},
	}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	// Verify delivery was marked successful
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.updatedDeliveries) == 0 {
		t.Fatal("expected delivery to be updated")
	}
	updated := store.updatedDeliveries[len(store.updatedDeliveries)-1]
	if updated.Status != models.WebhookDeliveryStatusDelivered {
		t.Errorf("delivery status = %s, want delivered", updated.Status)
	}

	// Verify body was sent
	if len(receivedBody) == 0 {
		t.Error("expected non-empty request body")
	}

	// Verify standard headers
	if got := receivedHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", got)
	}
	if got := receivedHeaders.Get("User-Agent"); got != "Keldris-Webhook/1.0" {
		t.Errorf("User-Agent = %s, want Keldris-Webhook/1.0", got)
	}
	if got := receivedHeaders.Get("X-Keldris-Event"); got != "backup.completed" {
		t.Errorf("X-Keldris-Event = %s, want backup.completed", got)
	}
	if got := receivedHeaders.Get("X-Keldris-Delivery"); got != delivery.ID.String() {
		t.Errorf("X-Keldris-Delivery = %s, want %s", got, delivery.ID.String())
	}
	if got := receivedHeaders.Get("X-Keldris-Signature-256"); got == "" {
		t.Error("expected X-Keldris-Signature-256 header to be set")
	}
	if got := receivedHeaders.Get("X-Keldris-Timestamp"); got == "" {
		t.Error("expected X-Keldris-Timestamp header to be set")
	}
}

func TestSendWebhook_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "secret-for-custom-headers")
	endpoint.Headers = map[string]string{
		"X-Custom-Header": "custom-value",
		"Authorization":   "Bearer ext-token",
	}

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	if got := receivedHeaders.Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("X-Custom-Header = %s, want custom-value", got)
	}
	if got := receivedHeaders.Get("Authorization"); got != "Bearer ext-token" {
		t.Errorf("Authorization = %s, want Bearer ext-token", got)
	}
}

// --- HMAC-SHA256 Signature Tests ---

func TestSignPayload(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)

	payload := []byte(`{"event":"test"}`)
	secret := []byte("my-webhook-secret")

	sig := d.signPayload(payload, secret)

	// Verify format: sha256=<hex>
	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("signature should start with 'sha256=', got: %s", sig)
	}

	// Verify the HMAC is correct
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sig != expected {
		t.Errorf("signature mismatch:\ngot:  %s\nwant: %s", sig, expected)
	}
}

func TestSignPayload_DifferentSecretsDifferentSignatures(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)

	payload := []byte(`{"event":"test"}`)

	sig1 := d.signPayload(payload, []byte("secret-one"))
	sig2 := d.signPayload(payload, []byte("secret-two"))

	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestSignPayload_DifferentPayloadsDifferentSignatures(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)

	secret := []byte("same-secret")
	sig1 := d.signPayload([]byte(`{"a":1}`), secret)
	sig2 := d.signPayload([]byte(`{"a":2}`), secret)

	if sig1 == sig2 {
		t.Error("different payloads should produce different signatures")
	}
}

func TestVerifySignature(t *testing.T) {
	payload := []byte(`{"event":"backup.completed"}`)
	secret := []byte("my-secret-key-1234567890")

	// Generate a valid signature
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    []byte
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			signature: validSig,
			secret:    secret,
			want:      true,
		},
		{
			name:      "wrong secret",
			payload:   payload,
			signature: validSig,
			secret:    []byte("wrong-secret"),
			want:      false,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"event":"backup.failed"}`),
			signature: validSig,
			secret:    secret,
			want:      false,
		},
		{
			name:      "empty signature",
			payload:   payload,
			signature: "",
			secret:    secret,
			want:      false,
		},
		{
			name:      "garbage signature",
			payload:   payload,
			signature: "sha256=0000000000000000000000000000000000000000000000000000000000000000",
			secret:    secret,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifySignature(tt.payload, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("VerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignPayload_MatchesVerify(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)

	payload := []byte(`{"id":"123","event_type":"backup.completed","data":{"size":1024}}`)
	secret := []byte("roundtrip-secret-key")

	sig := d.signPayload(payload, secret)
	if !VerifySignature(payload, sig, secret) {
		t.Error("signPayload output should be verifiable by VerifySignature")
	}
}

// --- Signature Verification in Delivery ---

func TestSendWebhook_SignatureIsVerifiable(t *testing.T) {
	secret := "verifiable-secret-key-here"
	var receivedBody []byte
	var receivedSig string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Keldris-Signature-256")
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, secret)

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	// Verify the signature using the public function
	if !VerifySignature(receivedBody, receivedSig, []byte(secret)) {
		t.Error("delivered signature should be verifiable with the original secret")
	}
}

// --- Retry Logic Tests ---

func TestSendWebhook_RetryOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "retry-secret")
	endpoint.RetryCount = 3

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	// First attempt should schedule retry (attempt 1 < max 3)
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.updatedDeliveries) == 0 {
		t.Fatal("expected delivery to be updated")
	}
	updated := store.updatedDeliveries[len(store.updatedDeliveries)-1]
	if updated.Status != models.WebhookDeliveryStatusRetrying {
		t.Errorf("delivery status = %s, want retrying", updated.Status)
	}
	if updated.NextRetryAt == nil {
		t.Error("expected next_retry_at to be set")
	}
	if updated.AttemptNumber != 2 {
		t.Errorf("attempt number = %d, want 2 (incremented by MarkRetrying)", updated.AttemptNumber)
	}
}

func TestSendWebhook_MaxRetriesExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "maxretry-secret")

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{"event_type": "backup.completed"}
	// MaxAttempts=3, AttemptNumber=3 means ShouldRetry() returns false
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)
	delivery.AttemptNumber = 3

	d.sendWebhook(context.Background(), endpoint, delivery)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.updatedDeliveries) == 0 {
		t.Fatal("expected delivery to be updated")
	}
	updated := store.updatedDeliveries[len(store.updatedDeliveries)-1]
	if updated.Status != models.WebhookDeliveryStatusFailed {
		t.Errorf("delivery status = %s, want failed", updated.Status)
	}
}

func TestSendWebhook_ConnectionRefused(t *testing.T) {
	km := testKeyManager(t)
	// Use a port that nothing is listening on
	endpoint := makeEndpoint(t, km, "http://127.0.0.1:1", "connrefused-secret")

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
	d := newTestDispatcher(store, km)

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	// When client.Do returns a connection error, handleDeliveryError mutates
	// the delivery in-memory but sendWebhook returns before persisting to store.
	// Verify the in-memory state was set correctly for retry.
	if delivery.Status != models.WebhookDeliveryStatusRetrying {
		t.Errorf("delivery status = %s, want retrying", delivery.Status)
	}
	if delivery.ErrorMessage == "" {
		t.Error("expected error message to be set on connection failure")
	}
}

// --- Concurrent Dispatch Tests ---

func TestDispatch_MultipleEndpoints(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	km := testKeyManager(t)
	orgID := uuid.New()

	endpoints := make([]*models.WebhookEndpoint, 3)
	for i := range endpoints {
		ep := makeEndpoint(t, km, srv.URL, fmt.Sprintf("secret-%d", i))
		ep.OrgID = orgID
		endpoints[i] = ep
	}

	store := &mockWebhookStore{endpoints: endpoints}
	d := newTestDispatcher(store, km)

	err := d.Dispatch(context.Background(), orgID, models.WebhookEventBackupCompleted, nil, map[string]any{
		"backup_id": "test-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for async goroutines to finish
	time.Sleep(500 * time.Millisecond)

	// Verify all 3 deliveries were created
	store.mu.Lock()
	createdCount := len(store.createdDeliveries)
	store.mu.Unlock()
	if createdCount != 3 {
		t.Errorf("created delivery count = %d, want 3", createdCount)
	}

	// Verify the server was called for each endpoint
	if got := int(callCount.Load()); got != 3 {
		t.Errorf("server call count = %d, want 3", got)
	}
}

func TestDispatch_NoEndpoints(t *testing.T) {
	km := testKeyManager(t)
	store := &mockWebhookStore{endpoints: nil}
	d := newTestDispatcher(store, km)

	err := d.Dispatch(context.Background(), uuid.New(), models.WebhookEventBackupCompleted, nil, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.createdDeliveries) != 0 {
		t.Error("expected no deliveries when no endpoints exist")
	}
}

func TestDispatch_EndpointsQueryError(t *testing.T) {
	km := testKeyManager(t)
	store := &mockWebhookStore{getEndpointsErr: fmt.Errorf("db connection lost")}
	d := newTestDispatcher(store, km)

	err := d.Dispatch(context.Background(), uuid.New(), models.WebhookEventBackupCompleted, nil, map[string]any{})
	if err == nil {
		t.Error("expected error when endpoint query fails")
	}
}

// --- Payload Serialization Tests ---

func TestBuildPayload(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)
	orgID := uuid.New()

	tests := []struct {
		name      string
		eventType models.WebhookEventType
		eventID   *uuid.UUID
		data      map[string]any
	}{
		{
			name:      "backup completed",
			eventType: models.WebhookEventBackupCompleted,
			data: map[string]any{
				"backup_id":   "abc-123",
				"duration_ms": 5000,
				"size_bytes":  1024000,
			},
		},
		{
			name:      "agent offline",
			eventType: models.WebhookEventAgentOffline,
			data: map[string]any{
				"agent_id": "agent-001",
				"hostname": "server-01",
			},
		},
		{
			name:      "with event ID",
			eventType: models.WebhookEventAlertTriggered,
			eventID:   uuidPtr(uuid.New()),
			data: map[string]any{
				"alert_id":   "alert-001",
				"risk_score": 85,
			},
		},
		{
			name:      "empty data",
			eventType: models.WebhookEventBackupStarted,
			data:      map[string]any{},
		},
		{
			name:      "nil data",
			eventType: models.WebhookEventBackupStarted,
			data:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := d.buildPayload(orgID, tt.eventType, tt.eventID, tt.data)

			// Verify required fields exist
			if _, ok := payload["id"]; !ok {
				t.Error("payload missing 'id' field")
			}
			if got, ok := payload["event_type"]; !ok || got != string(tt.eventType) {
				t.Errorf("payload event_type = %v, want %s", got, tt.eventType)
			}
			if _, ok := payload["timestamp"]; !ok {
				t.Error("payload missing 'timestamp' field")
			}
			if got, ok := payload["org_id"]; !ok || got != orgID.String() {
				t.Errorf("payload org_id = %v, want %s", got, orgID.String())
			}

			// Verify event ID is used when provided
			if tt.eventID != nil {
				if got := payload["id"]; got != tt.eventID.String() {
					t.Errorf("payload id = %v, want %s", got, tt.eventID.String())
				}
			}

			// Verify payload is JSON-serializable
			_, err := json.Marshal(payload)
			if err != nil {
				t.Errorf("payload not JSON-serializable: %v", err)
			}
		})
	}
}

func TestBuildPayload_AllEventTypes(t *testing.T) {
	km := testKeyManager(t)
	d := newTestDispatcher(&mockWebhookStore{}, km)
	orgID := uuid.New()

	for _, eventType := range models.AllWebhookEventTypes() {
		t.Run(string(eventType), func(t *testing.T) {
			payload := d.buildPayload(orgID, eventType, nil, map[string]any{"test": true})

			if got := payload["event_type"]; got != string(eventType) {
				t.Errorf("event_type = %v, want %s", got, eventType)
			}

			// Verify it serializes cleanly
			data, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			if len(data) == 0 {
				t.Error("expected non-empty JSON")
			}
		})
	}
}

// --- Timeout Handling ---

func TestSendWebhook_Timeout(t *testing.T) {
	km := testKeyManager(t)
	// Use an unroutable IP (RFC 5737 TEST-NET-1) to guarantee a connect timeout
	// without needing a real server. This is fast and reliable cross-platform.
	endpoint := makeEndpoint(t, km, "http://192.0.2.1:12345", "timeout-secret")

	store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}

	// Create dispatcher with very short client timeout
	cfg := Config{
		WorkerCount:    1,
		BatchSize:      10,
		RequestTimeout: 100 * time.Millisecond,
	}
	d := NewDispatcher(store, km, cfg, zerolog.Nop())

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

	d.sendWebhook(context.Background(), endpoint, delivery)

	// When client.Do returns a timeout error, handleDeliveryError mutates
	// the delivery in-memory but sendWebhook returns before persisting.
	// Verify the in-memory state was updated for retry.
	if delivery.Status != models.WebhookDeliveryStatusRetrying {
		t.Errorf("delivery status = %s, want retrying (due to timeout)", delivery.Status)
	}
	if delivery.ErrorMessage == "" {
		t.Error("expected error message for timeout")
	}
}

// --- Non-2xx Status Codes ---

func TestSendWebhook_Non2xxStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus models.WebhookDeliveryStatus
	}{
		{"301 redirect", http.StatusMovedPermanently, models.WebhookDeliveryStatusRetrying},
		{"400 bad request", http.StatusBadRequest, models.WebhookDeliveryStatusRetrying},
		{"403 forbidden", http.StatusForbidden, models.WebhookDeliveryStatusRetrying},
		{"404 not found", http.StatusNotFound, models.WebhookDeliveryStatusRetrying},
		{"429 too many requests", http.StatusTooManyRequests, models.WebhookDeliveryStatusRetrying},
		{"500 server error", http.StatusInternalServerError, models.WebhookDeliveryStatusRetrying},
		{"502 bad gateway", http.StatusBadGateway, models.WebhookDeliveryStatusRetrying},
		{"503 service unavailable", http.StatusServiceUnavailable, models.WebhookDeliveryStatusRetrying},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("error"))
			}))
			defer srv.Close()

			km := testKeyManager(t)
			endpoint := makeEndpoint(t, km, srv.URL, "status-secret")

			store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
			d := newTestDispatcher(store, km)

			payload := map[string]any{"event_type": "backup.completed"}
			delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

			d.sendWebhook(context.Background(), endpoint, delivery)

			store.mu.Lock()
			defer store.mu.Unlock()
			if len(store.updatedDeliveries) == 0 {
				t.Fatal("expected delivery to be updated")
			}
			updated := store.updatedDeliveries[len(store.updatedDeliveries)-1]
			if updated.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", updated.Status, tt.wantStatus)
			}
		})
	}
}

// --- 2xx Status Codes ---

func TestSendWebhook_2xxStatusCodes(t *testing.T) {
	codes := []int{200, 201, 202, 204}
	for _, code := range codes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			}))
			defer srv.Close()

			km := testKeyManager(t)
			endpoint := makeEndpoint(t, km, srv.URL, "ok-secret")

			store := &mockWebhookStore{endpoints: []*models.WebhookEndpoint{endpoint}}
			d := newTestDispatcher(store, km)

			payload := map[string]any{"event_type": "backup.completed"}
			delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)

			d.sendWebhook(context.Background(), endpoint, delivery)

			store.mu.Lock()
			defer store.mu.Unlock()
			if len(store.updatedDeliveries) == 0 {
				t.Fatal("expected delivery to be updated")
			}
			updated := store.updatedDeliveries[len(store.updatedDeliveries)-1]
			if updated.Status != models.WebhookDeliveryStatusDelivered {
				t.Errorf("status = %s, want delivered for HTTP %d", updated.Status, code)
			}
		})
	}
}

// --- TestEndpoint Tests ---

func TestTestEndpoint_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it sends test payload
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("invalid JSON payload: %v", err)
		}
		data, ok := payload["data"].(map[string]any)
		if !ok {
			t.Error("expected data field in payload")
		} else {
			if data["test"] != true {
				t.Error("expected data.test = true")
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "test-endpoint-secret")

	store := &mockWebhookStore{}
	d := newTestDispatcher(store, km)

	resp, err := d.TestEndpoint(context.Background(), endpoint, models.WebhookEventBackupCompleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, got error: %s", resp.ErrorMessage)
	}
	if resp.ResponseStatus != 200 {
		t.Errorf("response status = %d, want 200", resp.ResponseStatus)
	}
	if resp.DurationMs < 0 {
		t.Errorf("duration should be >= 0, got %d", resp.DurationMs)
	}
}

func TestTestEndpoint_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "test-error-secret")

	store := &mockWebhookStore{}
	d := newTestDispatcher(store, km)

	resp, err := d.TestEndpoint(context.Background(), endpoint, models.WebhookEventBackupCompleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure for 500 response")
	}
	if resp.ResponseStatus != 500 {
		t.Errorf("response status = %d, want 500", resp.ResponseStatus)
	}
}

func TestTestEndpoint_Unreachable(t *testing.T) {
	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, "http://127.0.0.1:1", "unreachable-secret")

	store := &mockWebhookStore{}
	d := newTestDispatcher(store, km)

	resp, err := d.TestEndpoint(context.Background(), endpoint, models.WebhookEventBackupCompleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure for unreachable endpoint")
	}
	if resp.ErrorMessage == "" {
		t.Error("expected error message for unreachable endpoint")
	}
}

// --- DefaultConfig Tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.WorkerCount != 5 {
		t.Errorf("default worker count = %d, want 5", cfg.WorkerCount)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("default batch size = %d, want 100", cfg.BatchSize)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("default request timeout = %v, want 30s", cfg.RequestTimeout)
	}
}

// --- ProcessRetries Tests ---

func TestProcessRetries_Success(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	km := testKeyManager(t)
	endpoint := makeEndpoint(t, km, srv.URL, "retry-success-secret")

	payload := map[string]any{"event_type": "backup.completed"}
	delivery := models.NewWebhookDelivery(endpoint.OrgID, endpoint.ID, models.WebhookEventBackupCompleted, nil, payload, 3)
	delivery.AttemptNumber = 2 // Second attempt (first retry)

	store := &mockWebhookStore{
		endpoints: []*models.WebhookEndpoint{endpoint},
		pending:   []*models.WebhookDelivery{delivery},
	}
	d := newTestDispatcher(store, km)

	d.processRetries(context.Background())

	if got := int(callCount.Load()); got != 1 {
		t.Errorf("server call count = %d, want 1", got)
	}
}

func TestProcessRetries_NoPending(t *testing.T) {
	km := testKeyManager(t)
	store := &mockWebhookStore{pending: nil}
	d := newTestDispatcher(store, km)

	// Should not panic or error
	d.processRetries(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.updatedDeliveries) != 0 {
		t.Error("expected no updates when no pending deliveries")
	}
}

// --- Helper ---

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}
