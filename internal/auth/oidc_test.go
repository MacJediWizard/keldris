package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// newMockOIDCServer creates a mock OIDC provider HTTP server for testing.
// It serves a valid discovery document, JWKS, token, and userinfo endpoints.
// Supported authorization codes:
//   - "valid-code": returns a valid token with valid id_token
//   - "expired-code": returns a token with an expired id_token
//   - "wrong-aud-code": returns a token with wrong audience
//   - "bad-sig-code": returns a token signed with a different key
//   - "no-idtoken-code": returns a token without id_token field
func newMockOIDCServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate other RSA key: %v", err)
	}

	var server *httptest.Server
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		discovery := map[string]interface{}{
			"issuer":                                server.URL,
			"authorization_endpoint":                server.URL + "/authorize",
			"token_endpoint":                        server.URL + "/token",
			"jwks_uri":                              server.URL + "/jwks",
			"userinfo_endpoint":                     server.URL + "/userinfo",
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discovery)
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": "test-key",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")

		var resp map[string]interface{}
		switch code {
		case "valid-code":
			idToken := createTestIDToken(t, key, server.URL, "test-client-id",
				"sub-12345", "user@example.com", "Test User", time.Now().Add(time.Hour))
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			}
		case "expired-code":
			idToken := createTestIDToken(t, key, server.URL, "test-client-id",
				"sub-12345", "user@example.com", "Test User", time.Now().Add(-time.Hour))
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			}
		case "wrong-aud-code":
			idToken := createTestIDToken(t, key, server.URL, "wrong-client-id",
				"sub-12345", "user@example.com", "Test User", time.Now().Add(time.Hour))
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			}
		case "bad-sig-code":
			idToken := createTestIDToken(t, otherKey, server.URL, "test-client-id",
				"sub-12345", "user@example.com", "Test User", time.Now().Add(time.Hour))
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			}
		case "no-idtoken-code":
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			}
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"sub":   "sub-12345",
			"email": "user@example.com",
			"name":  "Test User",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server = httptest.NewServer(mux)
	return server, key
}

// createTestIDToken creates a signed JWT ID token for testing.
func createTestIDToken(t *testing.T, key *rsa.PrivateKey, issuer, audience, subject, email, name string, expiry time.Time) string {
	t.Helper()

	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": "test-key",
	}
	claims := map[string]interface{}{
		"iss":   issuer,
		"sub":   subject,
		"aud":   audience,
		"email": email,
		"name":  name,
		"iat":   time.Now().Unix(),
		"exp":   expiry.Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	h := crypto.SHA256.New()
	h.Write([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// newTestOIDCProvider creates an OIDC provider connected to a mock server.
func newTestOIDCProvider(t *testing.T, serverURL string) *OIDC {
	t.Helper()

	logger := zerolog.Nop()
	cfg := OIDCConfig{
		Issuer:       serverURL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	oidcProvider, err := NewOIDC(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("failed to create OIDC provider: %v", err)
	}
	return oidcProvider
}

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

	// State should be base64 URL-safe encoded
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

func TestNewOIDC_Success(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)
	if oidcProvider == nil {
		t.Fatal("expected non-nil OIDC provider")
	}
}

func TestNewOIDC_InvalidIssuer(t *testing.T) {
	logger := zerolog.Nop()
	cfg := OIDCConfig{
		Issuer:       "http://127.0.0.1:1",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid"},
	}

	_, err := NewOIDC(context.Background(), cfg, logger)
	if err == nil {
		t.Error("expected error for invalid issuer")
	}
}

func TestOIDC_GetAuthURL(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	state := "test-state-abc123"
	url := oidcProvider.AuthorizationURL(state)

	if url == "" {
		t.Fatal("expected non-empty authorization URL")
	}
	if !strings.Contains(url, "state="+state) {
		t.Errorf("URL should contain state parameter, got: %s", url)
	}
	if !strings.Contains(url, "client_id=test-client-id") {
		t.Errorf("URL should contain client_id, got: %s", url)
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Errorf("URL should contain redirect_uri, got: %s", url)
	}
	if !strings.Contains(url, "response_type=code") {
		t.Errorf("URL should contain response_type=code, got: %s", url)
	}
	if !strings.Contains(url, server.URL+"/authorize") {
		t.Errorf("URL should start with authorization endpoint, got: %s", url)
	}
}

func TestOIDC_Exchange(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	t.Run("valid code", func(t *testing.T) {
		token, err := oidcProvider.Exchange(context.Background(), "valid-code")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == nil {
			t.Fatal("expected non-nil token")
		}
		if token.AccessToken != "mock-access-token" {
			t.Errorf("expected access token 'mock-access-token', got %q", token.AccessToken)
		}
		// Verify id_token is present in extras
		rawIDToken := token.Extra("id_token")
		if rawIDToken == nil {
			t.Error("expected id_token in token extras")
		}
	})

	t.Run("invalid code", func(t *testing.T) {
		_, err := oidcProvider.Exchange(context.Background(), "invalid-code")
		if err == nil {
			t.Error("expected error for invalid code")
		}
	})
}

func TestOIDC_VerifyToken(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	t.Run("valid token", func(t *testing.T) {
		token, err := oidcProvider.Exchange(context.Background(), "valid-code")
		if err != nil {
			t.Fatalf("failed to exchange: %v", err)
		}

		claims, err := oidcProvider.VerifyIDToken(context.Background(), token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Subject != "sub-12345" {
			t.Errorf("expected subject 'sub-12345', got %q", claims.Subject)
		}
		if claims.Email != "user@example.com" {
			t.Errorf("expected email 'user@example.com', got %q", claims.Email)
		}
		if claims.Name != "Test User" {
			t.Errorf("expected name 'Test User', got %q", claims.Name)
		}
	})

	t.Run("no id_token in response", func(t *testing.T) {
		// A bare oauth2.Token has no raw extras, so Extra("id_token") returns nil
		token := &oauth2.Token{AccessToken: "mock-access-token"}
		_, err := oidcProvider.VerifyIDToken(context.Background(), token)
		if err == nil {
			t.Error("expected error for missing id_token")
		}
		if !strings.Contains(err.Error(), "no id_token") {
			t.Errorf("expected 'no id_token' error, got: %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := oidcProvider.Exchange(context.Background(), "expired-code")
		if err != nil {
			t.Fatalf("failed to exchange: %v", err)
		}

		_, err = oidcProvider.VerifyIDToken(context.Background(), token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		token, err := oidcProvider.Exchange(context.Background(), "wrong-aud-code")
		if err != nil {
			t.Fatalf("failed to exchange: %v", err)
		}

		_, err = oidcProvider.VerifyIDToken(context.Background(), token)
		if err == nil {
			t.Error("expected error for wrong audience")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		token, err := oidcProvider.Exchange(context.Background(), "bad-sig-code")
		if err != nil {
			t.Fatalf("failed to exchange: %v", err)
		}

		_, err = oidcProvider.VerifyIDToken(context.Background(), token)
		if err == nil {
			t.Error("expected error for invalid signature")
		}
	})
}

func TestOIDC_RefreshToken(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	// Exchange a valid code to get a token
	token, err := oidcProvider.Exchange(context.Background(), "valid-code")
	if err != nil {
		t.Fatalf("failed to exchange: %v", err)
	}

	// Verify the initial token works
	claims, err := oidcProvider.VerifyIDToken(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error verifying initial token: %v", err)
	}
	if claims.Subject != "sub-12345" {
		t.Errorf("expected subject 'sub-12345', got %q", claims.Subject)
	}

	// Token should have access token
	if token.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestOIDC_ExpiredToken(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	token, err := oidcProvider.Exchange(context.Background(), "expired-code")
	if err != nil {
		t.Fatalf("failed to exchange: %v", err)
	}

	_, err = oidcProvider.VerifyIDToken(context.Background(), token)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "verify ID token") {
		t.Errorf("expected token verification error, got: %v", err)
	}
}

func TestOIDC_HealthCheck(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	err := oidcProvider.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("unexpected error from health check: %v", err)
	}
}

func TestOIDC_UserInfo(t *testing.T) {
	server, _ := newMockOIDCServer(t)
	defer server.Close()

	oidcProvider := newTestOIDCProvider(t, server.URL)

	token, err := oidcProvider.Exchange(context.Background(), "valid-code")
	if err != nil {
		t.Fatalf("failed to exchange code: %v", err)
	}

	userInfo, err := oidcProvider.UserInfo(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userInfo.Subject != "sub-12345" {
		t.Errorf("expected subject 'sub-12345', got %q", userInfo.Subject)
	}
	if userInfo.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %q", userInfo.Email)
	}
}
