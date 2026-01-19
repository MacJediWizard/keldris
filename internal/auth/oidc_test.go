package auth

import (
	"strings"
	"testing"
)

func TestDefaultOIDCConfig(t *testing.T) {
	cfg := DefaultOIDCConfig(
		"https://auth.example.com",
		"client-id",
		"client-secret",
		"https://app.example.com/auth/callback",
	)

	if cfg.Issuer != "https://auth.example.com" {
		t.Errorf("expected issuer https://auth.example.com, got %s", cfg.Issuer)
	}
	if cfg.ClientID != "client-id" {
		t.Errorf("expected client ID client-id, got %s", cfg.ClientID)
	}
	if cfg.ClientSecret != "client-secret" {
		t.Errorf("expected client secret client-secret, got %s", cfg.ClientSecret)
	}
	if cfg.RedirectURL != "https://app.example.com/auth/callback" {
		t.Errorf("expected redirect URL https://app.example.com/auth/callback, got %s", cfg.RedirectURL)
	}

	// Check default scopes
	if len(cfg.Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(cfg.Scopes))
	}

	hasOpenID := false
	hasProfile := false
	hasEmail := false
	for _, scope := range cfg.Scopes {
		switch scope {
		case "openid":
			hasOpenID = true
		case "profile":
			hasProfile = true
		case "email":
			hasEmail = true
		}
	}

	if !hasOpenID {
		t.Error("expected openid scope")
	}
	if !hasProfile {
		t.Error("expected profile scope")
	}
	if !hasEmail {
		t.Error("expected email scope")
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state1 == "" {
		t.Error("expected non-empty state")
	}

	// State should be base64 encoded (no special characters that break URLs)
	if strings.Contains(state1, "+") || strings.Contains(state1, "/") {
		t.Error("state should use URL-safe base64 encoding")
	}

	// Generate another state - should be different
	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state1 == state2 {
		t.Error("expected different states from multiple calls")
	}

	// State should be reasonably long (32 bytes = ~43 chars base64)
	if len(state1) < 40 {
		t.Errorf("state seems too short: %d chars", len(state1))
	}
}
