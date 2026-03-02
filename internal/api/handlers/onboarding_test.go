package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockOnboardingStore struct {
	progress  *models.OnboardingProgress
	getErr    error
	updateErr error
	skipErr   error
}

func (m *mockOnboardingStore) GetOnboardingProgress(_ context.Context, _ uuid.UUID) (*models.OnboardingProgress, error) {
	return m.progress, m.getErr
}

func (m *mockOnboardingStore) GetOrCreateOnboardingProgress(_ context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.progress != nil {
		return m.progress, nil
	}
	return models.NewOnboardingProgress(orgID), nil
}

func (m *mockOnboardingStore) UpdateOnboardingProgress(_ context.Context, _ *models.OnboardingProgress) error {
	return m.updateErr
}

func (m *mockOnboardingStore) SkipOnboarding(_ context.Context, _ uuid.UUID) error {
	return m.skipErr
}

func (m *mockOnboardingStore) GetOIDCSettings(_ context.Context, _ uuid.UUID) (*settings.OIDCSettings, error) {
	return nil, nil
}
func (m *mockOnboardingStore) UpdateOIDCSettings(_ context.Context, _ uuid.UUID, _ *settings.OIDCSettings) error {
	return nil
}
func (m *mockOnboardingStore) EnsureSystemSettingsExist(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockOnboardingStore) GetSMTPSettings(_ context.Context, _ uuid.UUID) (*settings.SMTPSettings, error) {
	return nil, nil
}
func (m *mockOnboardingStore) UpdateSMTPSettings(_ context.Context, _ uuid.UUID, _ *settings.SMTPSettings) error {
	return nil
}

func setupOnboardingTestRouter(store OnboardingStore, orgID uuid.UUID) *gin.Engine {
	user := testUser(orgID)
	r := SetupTestRouter(user)
	handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

// injectLicense returns a gin middleware that sets a license with the given tier in the context.
func injectLicense(tier license.LicenseTier) gin.HandlerFunc {
	return func(c *gin.Context) {
		lic := &license.License{Tier: tier}
		c.Set(string(middleware.LicenseContextKey), lic)
		c.Next()
	}
}

func TestOnboardingGetStatus(t *testing.T) {
	orgID := uuid.New()

	t.Run("new org needs onboarding", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=true for new org")
		}
		if resp.CurrentStep != models.OnboardingStepWelcome {
			t.Fatalf("expected welcome step, got %q", resp.CurrentStep)
		}
	})

	t.Run("completed onboarding", func(t *testing.T) {
		progress := models.NewOnboardingProgress(orgID)
		progress.Skip()
		store := &mockOnboardingStore{progress: progress}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=false")
		}
		if !resp.IsComplete {
			t.Fatal("expected is_complete=true")
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := SetupTestRouter(nil)
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestOnboardingCompleteStep(t *testing.T) {
	orgID := uuid.New()

	t.Run("valid step", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.CurrentStep != models.OnboardingStepLicense {
			t.Fatalf("expected step to advance to license, got %q", resp.CurrentStep)
		}
	})

	t.Run("invalid step", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/bogus", "{}"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store get error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		store := &mockOnboardingStore{updateErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestOnboardingSkip(t *testing.T) {
	orgID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Skipped {
			t.Fatal("expected skipped=true")
		}
		if resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=false after skip")
		}
	})

	t.Run("store get error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store skip error", func(t *testing.T) {
		store := &mockOnboardingStore{skipErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestOnboarding_StatusIncludesTier(t *testing.T) {
	orgID := uuid.New()

	t.Run("with pro tier license", func(t *testing.T) {
		store := &mockOnboardingStore{}

		user := testUser(orgID)
		r := SetupTestRouter(user)
		r.Use(injectLicense(license.TierPro))
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if resp.LicenseTier != string(license.TierPro) {
			t.Fatalf("expected license_tier=%q, got %q", license.TierPro, resp.LicenseTier)
		}
	})

	t.Run("with enterprise tier license", func(t *testing.T) {
		store := &mockOnboardingStore{}

		user := testUser(orgID)
		r := SetupTestRouter(user)
		r.Use(injectLicense(license.TierEnterprise))
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if resp.LicenseTier != string(license.TierEnterprise) {
			t.Fatalf("expected license_tier=%q, got %q", license.TierEnterprise, resp.LicenseTier)
		}
	})

	t.Run("no license omits tier", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if resp.LicenseTier != "" {
			t.Fatalf("expected empty license_tier with no license, got %q", resp.LicenseTier)
		}
	})
}

func TestOnboarding_OIDCStep_NoLicense(t *testing.T) {
	orgID := uuid.New()
	store := &mockOnboardingStore{}

	// Create a mock OIDC server so the issuer URL is a valid URL
	oidcServer := newMockOIDCServer(t)
	defer oidcServer.Close()

	// Handler with no license in context and nil oidcProvider
	user := testUser(orgID)
	r := SetupTestRouter(user)
	handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	body := fmt.Sprintf(`{
		"issuer": %q,
		"client_id": "test-client",
		"client_secret": "test-secret",
		"redirect_url": "http://localhost/callback"
	}`, oidcServer.URL)

	w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/oidc", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp OnboardingStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify the OIDC step was completed
	foundOIDC := false
	for _, step := range resp.CompletedSteps {
		if step == models.OnboardingStepOIDC {
			foundOIDC = true
			break
		}
	}
	if !foundOIDC {
		t.Fatalf("expected OIDC step in completed steps, got %v", resp.CompletedSteps)
	}
}

func TestOnboarding_OIDCStep_HotReload(t *testing.T) {
	orgID := uuid.New()
	store := &mockOnboardingStore{}

	// Create a mock OIDC discovery server
	oidcServer := newMockOIDCServer(t)
	defer oidcServer.Close()

	// Create a real OIDCProvider starting with nil (not configured)
	oidcProvider := auth.NewOIDCProvider(nil, zerolog.Nop())
	if oidcProvider.IsConfigured() {
		t.Fatal("expected OIDCProvider to not be configured initially")
	}

	// Handler with no license (allows all) and real oidcProvider
	user := testUser(orgID)
	r := SetupTestRouter(user)
	handler := NewOnboardingHandler(store, nil, oidcProvider, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	body := fmt.Sprintf(`{
		"issuer": %q,
		"client_id": "test-client",
		"client_secret": "test-secret",
		"redirect_url": "http://localhost/callback"
	}`, oidcServer.URL)

	w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/oidc", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// After the OIDC step completes, the provider should be hot-reloaded
	if !oidcProvider.IsConfigured() {
		t.Fatal("expected OIDCProvider.IsConfigured() to return true after OIDC step")
	}

	// Verify the underlying OIDC instance is non-nil
	if oidcProvider.Get() == nil {
		t.Fatal("expected OIDCProvider.Get() to return non-nil after hot-reload")
	}
}

func TestOnboarding_TestOIDC(t *testing.T) {
	orgID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockOnboardingStore{}
		oidcServer := newMockOIDCServer(t)
		defer oidcServer.Close()

		user := testUser(orgID)
		r := SetupTestRouter(user)
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		body := fmt.Sprintf(`{
			"issuer": %q,
			"client_id": "test-client",
			"client_secret": "test-secret",
			"redirect_url": "http://localhost/callback"
		}`, oidcServer.URL)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/test-oidc", body))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if resp["success"] != true {
			t.Fatalf("expected success=true, got %v: %s", resp["success"], resp["message"])
		}
	})

	t.Run("bad issuer", func(t *testing.T) {
		store := &mockOnboardingStore{}

		user := testUser(orgID)
		r := SetupTestRouter(user)
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		body := `{
			"issuer": "http://127.0.0.1:1",
			"client_id": "test-client",
			"client_secret": "test-secret",
			"redirect_url": "http://localhost/callback"
		}`

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/test-oidc", body))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if resp["success"] != false {
			t.Fatalf("expected success=false for unreachable issuer, got %v", resp["success"])
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		store := &mockOnboardingStore{}

		user := testUser(orgID)
		r := SetupTestRouter(user)
		handler := NewOnboardingHandler(store, nil, nil, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/test-oidc", `{}`))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}
