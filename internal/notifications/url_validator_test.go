package notifications

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	// Start a local HTTPS-less test server to get a resolvable non-private URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("empty URL", func(t *testing.T) {
		err := ValidateWebhookURL("", false)
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("expected 'required' in error, got: %s", err)
		}
	})

	t.Run("whitespace only URL", func(t *testing.T) {
		err := ValidateWebhookURL("   ", false)
		if err == nil {
			t.Fatal("expected error for whitespace URL")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		err := ValidateWebhookURL("://not-a-url", false)
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
	})

	t.Run("non-http scheme", func(t *testing.T) {
		err := ValidateWebhookURL("ftp://example.com/hook", false)
		if err == nil {
			t.Fatal("expected error for ftp scheme")
		}
		if !strings.Contains(err.Error(), "HTTP or HTTPS") {
			t.Errorf("expected scheme error, got: %s", err)
		}
	})

	t.Run("http allowed when HTTPS not required", func(t *testing.T) {
		// Use the test server URL which is HTTP and resolves to 127.0.0.1
		// This will be blocked by private IP check, so we just test scheme separately
		err := ValidateWebhookURL("http://example.com/hook", false)
		// This should pass scheme validation (may fail on DNS in CI, but scheme is fine)
		if err != nil && strings.Contains(err.Error(), "HTTPS") {
			t.Errorf("http should be allowed when requireHTTPS is false, got: %s", err)
		}
	})

	t.Run("http blocked when HTTPS required", func(t *testing.T) {
		err := ValidateWebhookURL("http://example.com/hook", true)
		if err == nil {
			t.Fatal("expected error for http when HTTPS required")
		}
		if !strings.Contains(err.Error(), "must use HTTPS") {
			t.Errorf("expected HTTPS error, got: %s", err)
		}
	})

	t.Run("https allowed when HTTPS required", func(t *testing.T) {
		err := ValidateWebhookURL("https://example.com/hook", true)
		if err != nil {
			t.Errorf("expected no error for https URL, got: %s", err)
		}
	})

	t.Run("block localhost 127.0.0.1", func(t *testing.T) {
		err := ValidateWebhookURL("http://127.0.0.1/hook", false)
		if err == nil {
			t.Fatal("expected error for localhost IP")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block localhost 127.0.0.2", func(t *testing.T) {
		err := ValidateWebhookURL("http://127.0.0.2/hook", false)
		if err == nil {
			t.Fatal("expected error for 127.0.0.2")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block 10.x.x.x", func(t *testing.T) {
		err := ValidateWebhookURL("http://10.0.0.1/hook", false)
		if err == nil {
			t.Fatal("expected error for 10.x.x.x range")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block 172.16.x.x", func(t *testing.T) {
		err := ValidateWebhookURL("http://172.16.0.1/hook", false)
		if err == nil {
			t.Fatal("expected error for 172.16.x.x range")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block 192.168.x.x", func(t *testing.T) {
		err := ValidateWebhookURL("http://192.168.1.1/hook", false)
		if err == nil {
			t.Fatal("expected error for 192.168.x.x range")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block link-local 169.254.x.x", func(t *testing.T) {
		err := ValidateWebhookURL("http://169.254.1.1/hook", false)
		if err == nil {
			t.Fatal("expected error for link-local range")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block metadata endpoint 169.254.169.254", func(t *testing.T) {
		err := ValidateWebhookURL("http://169.254.169.254/latest/meta-data/", false)
		if err == nil {
			t.Fatal("expected error for metadata endpoint")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("block localhost hostname", func(t *testing.T) {
		err := ValidateWebhookURL("http://localhost/hook", false)
		if err == nil {
			t.Fatal("expected error for localhost hostname")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})

	t.Run("no host", func(t *testing.T) {
		err := ValidateWebhookURL("http:///hook", false)
		if err == nil {
			t.Fatal("expected error for URL without host")
		}
		if !strings.Contains(err.Error(), "must have a host") {
			t.Errorf("expected host error, got: %s", err)
		}
	})

	t.Run("unresolvable host", func(t *testing.T) {
		err := ValidateWebhookURL("https://this-host-does-not-exist-keldris-test.invalid/hook", false)
		if err == nil {
			t.Fatal("expected error for unresolvable host")
		}
		if !strings.Contains(err.Error(), "resolve") {
			t.Errorf("expected resolve error, got: %s", err)
		}
	})

	t.Run("valid public HTTPS URL", func(t *testing.T) {
		err := ValidateWebhookURL("https://example.com/webhook", false)
		if err != nil {
			t.Errorf("expected no error for valid public URL, got: %s", err)
		}
	})

	t.Run("valid public HTTPS URL with port", func(t *testing.T) {
		err := ValidateWebhookURL("https://example.com:8443/webhook", true)
		if err != nil {
			t.Errorf("expected no error for valid public URL with port, got: %s", err)
		}
	})

	t.Run("block IPv6 loopback", func(t *testing.T) {
		err := ValidateWebhookURL("http://[::1]/hook", false)
		if err == nil {
			t.Fatal("expected error for IPv6 loopback")
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Errorf("expected 'blocked' in error, got: %s", err)
		}
	})
}
