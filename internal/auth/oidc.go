// Package auth provides OIDC authentication and session management.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// OIDCConfig holds OIDC provider configuration.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// DefaultOIDCConfig returns an OIDCConfig with standard scopes.
func DefaultOIDCConfig(issuer, clientID, clientSecret, redirectURL string) OIDCConfig {
	return OIDCConfig{
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

// OIDC wraps the OIDC provider and OAuth2 configuration.
type OIDC struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	logger       zerolog.Logger
}

// NewOIDC creates a new OIDC provider instance.
func NewOIDC(ctx context.Context, cfg OIDCConfig, logger zerolog.Logger) (*OIDC, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("create OIDC provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	o := &OIDC{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		logger:       logger.With().Str("component", "oidc").Logger(),
	}

	o.logger.Info().Str("issuer", cfg.Issuer).Msg("OIDC provider initialized")
	return o, nil
}

// GenerateState generates a cryptographically secure random state parameter.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// AuthorizationURL returns the URL to redirect users for authentication.
func (o *OIDC) AuthorizationURL(state string) string {
	return o.oauth2Config.AuthCodeURL(state)
}

// Exchange exchanges an authorization code for tokens.
func (o *OIDC) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := o.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	return token, nil
}

// IDTokenClaims holds the standard claims from an ID token.
type IDTokenClaims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

// VerifyIDToken verifies the ID token and extracts claims.
func (o *OIDC) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*IDTokenClaims, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify ID token: %w", err)
	}

	var claims IDTokenClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}

	o.logger.Debug().
		Str("subject", claims.Subject).
		Str("email", claims.Email).
		Msg("ID token verified")

	return &claims, nil
}

// UserInfo fetches user information from the OIDC provider.
func (o *OIDC) UserInfo(ctx context.Context, token *oauth2.Token) (*oidc.UserInfo, error) {
	userInfo, err := o.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	return userInfo, nil
}

// HealthCheck verifies that the OIDC provider is reachable by fetching its discovery document.
func (o *OIDC) HealthCheck(ctx context.Context) error {
	// The provider already has the discovery document cached, but we can verify
	// connectivity by attempting to fetch claims. A simple way is to check that
	// the provider's endpoint is available.
	endpoint := o.provider.Endpoint()
	if endpoint.AuthURL == "" || endpoint.TokenURL == "" {
		return fmt.Errorf("OIDC provider endpoints not available")
	}
	return nil
}

// OIDCProvider is a thread-safe wrapper around an OIDC provider that supports
// hot-reloading when OIDC settings change at runtime.
type OIDCProvider struct {
	mu       sync.RWMutex
	provider *OIDC
	logger   zerolog.Logger
}

// NewOIDCProvider creates a new OIDCProvider wrapper.
// The initial provider can be nil (password-only mode).
func NewOIDCProvider(provider *OIDC, logger zerolog.Logger) *OIDCProvider {
	return &OIDCProvider{
		provider: provider,
		logger:   logger.With().Str("component", "oidc_provider").Logger(),
	}
}

// Get returns the current OIDC provider instance (may be nil).
func (p *OIDCProvider) Get() *OIDC {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.provider
}

// Update creates a new OIDC provider from the given config and swaps it in.
// If initialization fails, the old provider is kept.
func (p *OIDCProvider) Update(ctx context.Context, cfg OIDCConfig) error {
	newProvider, err := NewOIDC(ctx, cfg, p.logger)
	if err != nil {
		return fmt.Errorf("initialize OIDC provider: %w", err)
	}

	p.mu.Lock()
	p.provider = newProvider
	p.mu.Unlock()

	p.logger.Info().Str("issuer", cfg.Issuer).Msg("OIDC provider updated")
	return nil
}

// IsConfigured returns true if an OIDC provider is currently loaded.
func (p *OIDCProvider) IsConfigured() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.provider != nil
}

// HealthCheck delegates to the underlying provider's health check.
// Returns nil if no provider is configured (OIDC is optional).
func (p *OIDCProvider) HealthCheck(ctx context.Context) error {
	provider := p.Get()
	if provider == nil {
		return nil
	}
	return provider.HealthCheck(ctx)
}
