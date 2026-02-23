package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// newTestWebhookSender creates a webhook sender with URL validation and IP
// blocking disabled so it can reach local httptest servers on 127.0.0.1.
func newTestWebhookSender() *WebhookSender {
	s := NewWebhookSender(zerolog.Nop())
	s.validateURL = func(_ string) error { return nil }
	s.client.Transport = http.DefaultTransport
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
	sender.client.Transport = http.DefaultTransport
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
	sender.client.Transport = http.DefaultTransport
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

func TestWebhookSender_DNSRebindingProtection(t *testing.T) {
	// Simulate a DNS rebinding attack: URL validation is bypassed (as if DNS
	// returned a public IP during the first check), but the validating dialer
	// blocks the connection because the IP resolved at dial time is private.
	sender := NewWebhookSender(zerolog.Nop())
	// Bypass the pre-flight URL validation to simulate the attacker's DNS
	// returning a safe IP during the first lookup.
	sender.validateURL = func(_ string) error { return nil }

	payload := WebhookPayload{EventType: "test", Timestamp: time.Now(), Data: nil}

	// Attempt to send to a private IP. The validating dialer in the transport
	// must block the connection even though validateURL was bypassed.
	err := sender.Send(context.Background(), "http://127.0.0.1:1/hook", payload, "")
	if err == nil {
		t.Fatal("expected error: validating dialer should block private IPs at dial time")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}

	// Also test 10.x range
	err = sender.Send(context.Background(), "http://10.0.0.1:1/hook", payload, "")
	if err == nil {
		t.Fatal("expected error: validating dialer should block 10.x.x.x at dial time")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}

	// Also test metadata endpoint
	err = sender.Send(context.Background(), "http://169.254.169.254:80/latest/meta-data/", payload, "")
	if err == nil {
		t.Fatal("expected error: validating dialer should block metadata endpoint at dial time")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got: %s", err)
	}
}

func TestValidatingDialer_BlocksPrivateIPs(t *testing.T) {
	dial := ValidatingDialer()
	ctx := context.Background()

	tests := []struct {
		name string
		addr string
	}{
		{"loopback", "127.0.0.1:80"},
		{"private 10.x", "10.0.0.1:80"},
		{"private 172.16.x", "172.16.0.1:80"},
		{"private 192.168.x", "192.168.1.1:80"},
		{"link-local", "169.254.1.1:80"},
		{"metadata", "169.254.169.254:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := dial(ctx, "tcp", tt.addr)
			if conn != nil {
				conn.Close()
			}
			if err == nil {
				t.Fatalf("expected error dialing %s, got nil", tt.addr)
			}
			if !strings.Contains(err.Error(), "blocked") {
				t.Errorf("expected 'blocked' in error for %s, got: %s", tt.addr, err)
			}
		})
	}
}

func TestValidatingDialer_AllowsPublicIPs(t *testing.T) {
	// Start a local test server to have something to connect to.
	// The test server binds to 127.0.0.1, so we can't test public connectivity
	// directly. Instead, verify the dialer accepts a known public IP by mocking
	// the resolver. We test the isBlockedIP function directly.
	publicIPs := []string{"93.184.216.34", "8.8.8.8", "1.1.1.1"}
	for _, ipStr := range publicIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Fatalf("failed to parse IP %s", ipStr)
		}
		if isBlockedIP(ip) {
			t.Errorf("expected %s to NOT be blocked", ipStr)
		}
	}
}
