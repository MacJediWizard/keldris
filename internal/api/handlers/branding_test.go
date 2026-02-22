package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockBrandingStore struct {
	settings  *models.BrandingSettings
	getErr    error
	upsertErr error
	deleteErr error
}

func (m *mockBrandingStore) GetBrandingSettings(_ context.Context, _ uuid.UUID) (*models.BrandingSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.settings, nil
}

func (m *mockBrandingStore) UpsertBrandingSettings(_ context.Context, b *models.BrandingSettings) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.settings = b
	return nil
}

func (m *mockBrandingStore) DeleteBrandingSettings(_ context.Context, _ uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.settings = nil
	return nil
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
		store := &mockBrandingStore{}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result models.BrandingSettings
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.OrgID != orgID {
			t.Errorf("expected org_id %s, got %s", orgID, result.OrgID)
		}
	})

	t.Run("returns existing settings", func(t *testing.T) {
		settings := &models.BrandingSettings{
			ID:           uuid.New(),
			OrgID:        orgID,
			ProductName:  "MyBackup",
			PrimaryColor: "#FF0000",
		}
		store := &mockBrandingStore{settings: settings}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var result models.BrandingSettings
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
		store := &mockBrandingStore{}
		r := setupBrandingTestRouter(store, noOrgUser)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/branding"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBrandingUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates new settings", func(t *testing.T) {
		store := &mockBrandingStore{}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"product_name": "CustomApp",
			"primary_color": "#123ABC"
		}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result models.BrandingSettings
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

	t.Run("updates existing settings", func(t *testing.T) {
		existing := &models.BrandingSettings{
			ID:          uuid.New(),
			OrgID:       orgID,
			ProductName: "OldName",
		}
		store := &mockBrandingStore{settings: existing}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"product_name": "NewName"
		}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result models.BrandingSettings
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.ProductName != "NewName" {
			t.Errorf("expected NewName, got %s", result.ProductName)
		}
	})

	t.Run("rejects invalid color", func(t *testing.T) {
		store := &mockBrandingStore{}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"primary_color": "not-a-color"
		}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects invalid URL", func(t *testing.T) {
		store := &mockBrandingStore{}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/branding", `{
			"logo_url": "not-a-url"
		}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestBrandingReset(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockBrandingStore{settings: &models.BrandingSettings{OrgID: orgID}}
		r := setupBrandingTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/branding"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		if store.settings != nil {
			t.Error("expected settings to be nil after reset")
		}
	})
}
