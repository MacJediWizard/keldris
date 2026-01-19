// Package auth provides OIDC authentication and session management.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

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
