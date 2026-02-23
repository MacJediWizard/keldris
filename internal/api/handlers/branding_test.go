package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/branding"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockBrandingStore struct {
	settings       *branding.BrandingSettings
	publicSettings *branding.PublicBrandingSettings
	hasFeature     bool
	getErr         error
	updateErr      error
	publicErr      error
	featureErr     error
}

func (m *mockBrandingStore) GetBrandingSettings(_ context.Context, _ uuid.UUID) (*branding.BrandingSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.settings != nil {
		return m.settings, nil
	}
	defaults := branding.DefaultBrandingSettings()
	return &defaults, nil
}

func (m *mockBrandingStore) UpdateBrandingSettings(_ context.Context, _ uuid.UUID, b *branding.BrandingSettings) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.settings = b
	return nil
}

func (m *mockBrandingStore) GetPublicBrandingSettings(_ context.Context, _ string) (*branding.PublicBrandingSettings, error) {
	if m.publicErr != nil {
		return nil, m.publicErr
	}
	return m.publicSettings, nil
}

func (m *mockBrandingStore) CreateSettingsAuditLog(_ context.Context, _ *settings.SettingsAuditLog) error {
	return nil
}

func (m *mockBrandingStore) HasFeatureFlag(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	if m.featureErr != nil {
		return false, m.featureErr
	}
	return m.hasFeature, nil
}

func setupBrandingTestRouter(store BrandingStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewBrandingHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestBrandingGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns defaults when no settings", func(t *testing.T) {
		store := &mockBrandingStore{hasFeature: true}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result branding.BrandingSettings
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.ProductName != "Keldris" {
			t.Errorf("expected default product_name Keldris, got %s", result.ProductName)
		}
	})

	t.Run("returns existing settings", func(t *testing.T) {
		s := branding.DefaultBrandingSettings()
		s.ProductName = "MyBackup"
		s.PrimaryColor = "#FF0000"
		store := &mockBrandingStore{settings: &s, hasFeature: true}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var result branding.BrandingSettings
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.ProductName != "MyBackup" {
			t.Errorf("expected product_name MyBackup, got %s", result.ProductName)
		}
		if result.PrimaryColor != "#FF0000" {
			t.Errorf("expected primary_color #FF0000, got %s", result.PrimaryColor)
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := testUserNoOrg()
		store := &mockBrandingStore{hasFeature: true}
		r := setupBrandingTestRouter(store, noOrgUser)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("forbidden without feature flag", func(t *testing.T) {
		store := &mockBrandingStore{hasFeature: false}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestBrandingUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("updates settings", func(t *testing.T) {
		store := &mockBrandingStore{hasFeature: true}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"product_name": "CustomApp",
			"primary_color": "#123ABC"
		}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result branding.BrandingSettings
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.ProductName != "CustomApp" {
			t.Errorf("expected product_name CustomApp, got %s", result.ProductName)
		}
		if result.PrimaryColor != "#123ABC" {
			t.Errorf("expected primary_color #123ABC, got %s", result.PrimaryColor)
		}
	})

	t.Run("forbidden without feature flag", func(t *testing.T) {
		store := &mockBrandingStore{hasFeature: false}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"product_name": "CustomApp"
		}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}
