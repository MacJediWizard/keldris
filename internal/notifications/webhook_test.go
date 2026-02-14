package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// newTestWebhookSender creates a webhook sender with URL validation disabled for local test servers.
func newTestWebhookSender() *WebhookSender {
	s := NewWebhookSender(zerolog.Nop())
	s.validateURL = func(_ string) error { return nil }
	return s
}

func TestWebhookSender_Send(t *testing.T) {
	var receivedPayload WebhookPayload
	var receivedSig string

	secret := "test-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		receivedSig = r.Header.Get("X-Keldris-Signature")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		// Verify HMAC
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if receivedSig != expectedSig {
			t.Errorf("signature mismatch: got %q, want %q", receivedSig, expectedSig)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newTestWebhookSender()
	payload := WebhookPayload{
		EventType: "backup_success",
		Timestamp: time.Now(),
		Data:      map[string]string{"hostname": "server1"},
	}

	err := sender.Send(context.Background(), server.URL, payload, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload.EventType != "backup_success" {
		t.Errorf("expected event_type backup_success, got %s", receivedPayload.EventType)
	}
	if receivedSig == "" {
		t.Error("expected X-Keldris-Signature header to be set")
	}
}

func TestWebhookSender_SendNoSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Keldris-Signature")
		if sig != "" {
			t.Errorf("expected no signature header, got %s", sig)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newTestWebhookSender()
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), server.URL, payload, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhookSender_Retry(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := newTestWebhookSender()
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), server.URL, payload, "")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestWebhookSender_RetryExhausted(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := newTestWebhookSender()
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), server.URL, payload, "")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestWebhookSender_SSRFBlocked(t *testing.T) {
	sender := NewWebhookSender(zerolog.Nop())
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), "http://127.0.0.1:8080/hook", payload, "")
	if err == nil {
		t.Fatal("expected error for localhost URL")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}
}

func TestWebhookSender_SSRFPrivateIP(t *testing.T) {
	sender := NewWebhookSender(zerolog.Nop())
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), "http://10.0.0.1/hook", payload, "")
	if err == nil {
		t.Fatal("expected error for private IP URL")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}
}

func TestWebhookSender_SSRFMetadata(t *testing.T) {
	sender := NewWebhookSender(zerolog.Nop())
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), "http://169.254.169.254/latest/meta-data/", payload, "")
	if err == nil {
		t.Fatal("expected error for metadata endpoint")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}
}

func TestWebhookSender_URLNotInLogs(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(logger)
	sender.validateURL = func(_ string) error { return nil }
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), server.URL, payload, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logOutput := logBuf.String()
	if strings.Contains(logOutput, server.URL) {
		t.Errorf("log output should not contain webhook URL, got: %s", logOutput)
	}
}

func TestWebhookSender_URLNotInLogsAirGap(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	sender := NewWebhookSender(logger)
	sender.SetAirGapMode(true)

	webhookURL := "https://hooks.example.com/secret-token/callback"
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	_ = sender.Send(context.Background(), webhookURL, payload, "")

	logOutput := logBuf.String()
	if strings.Contains(logOutput, webhookURL) {
		t.Errorf("log output should not contain webhook URL, got: %s", logOutput)
	}
}

func TestWebhookSender_URLNotInLogsRetry(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(logger)
	sender.validateURL = func(_ string) error { return nil }
	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	err := sender.Send(context.Background(), server.URL, payload, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logOutput := logBuf.String()
	if strings.Contains(logOutput, server.URL) {
		t.Errorf("log output should not contain webhook URL during retries, got: %s", logOutput)
	}
}

func TestComputeHMAC(t *testing.T) {
	payload := []byte(`{"test":"data"}`)
	secret := "my-secret"

	result := computeHMAC(payload, secret)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if result != expected {
		t.Errorf("computeHMAC() = %q, want %q", result, expected)
	}
}
