package license

import (
	"testing"

	"github.com/rs/zerolog"
)

// mockOIDCChecker implements OIDCConfiguredChecker for testing.
type mockOIDCChecker struct {
	configured bool
}

func (m *mockOIDCChecker) IsConfigured() bool {
	return m.configured
}

func TestHeartbeat_IncludesOIDCConfigured(t *testing.T) {
	v := NewValidator(ValidatorConfig{
		Logger: zerolog.Nop(),
	})

	// Without any OIDC provider set, isOIDCConfigured should return false.
	if v.isOIDCConfigured() {
		t.Fatal("expected isOIDCConfigured() == false when no provider is set")
	}

	// Set a provider that reports OIDC is NOT configured.
	v.SetOIDCProvider(&mockOIDCChecker{configured: false})
	if v.isOIDCConfigured() {
		t.Fatal("expected isOIDCConfigured() == false when provider reports not configured")
	}

	// Set a provider that reports OIDC IS configured.
	v.SetOIDCProvider(&mockOIDCChecker{configured: true})
	if !v.isOIDCConfigured() {
		t.Fatal("expected isOIDCConfigured() == true when provider reports configured")
	}

	// Replace with a provider that flips back to false.
	v.SetOIDCProvider(&mockOIDCChecker{configured: false})
	if v.isOIDCConfigured() {
		t.Fatal("expected isOIDCConfigured() == false after replacing provider with unconfigured one")
	}
}
