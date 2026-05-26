package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockLicenseStore struct {
	tier    license.Tier
	getErr  error
	setErr  error
	setTier license.Tier
}

func (m *mockLicenseStore) GetOrgTier(_ context.Context, _ uuid.UUID) (license.Tier, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if m.tier == "" {
		return license.TierFree, nil
	}
	return m.tier, nil
}

func (m *mockLicenseStore) SetOrgTier(_ context.Context, _ uuid.UUID, tier license.Tier) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.setTier = tier
	return nil
}

func TestLicenseGetLicense(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns license tier", func(t *testing.T) {
		store := &mockLicenseStore{tier: license.TierPro}
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(InjectUser(user))
		checker := license.NewFeatureChecker(store)
		handler := NewLicenseHandler(store, checker, zerolog.Nop())
		r.GET("/api/v1/license", handler.GetLicense)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result LicenseResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.License.Tier != license.TierPro {
			t.Errorf("expected tier %s, got %s", license.TierPro, result.License.Tier)
		}
	})

	t.Run("returns 500 on store error", func(t *testing.T) {
		store := &mockLicenseStore{getErr: errors.New("db down")}
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(InjectUser(user))
		checker := license.NewFeatureChecker(store)
		handler := NewLicenseHandler(store, checker, zerolog.Nop())
		r.GET("/api/v1/license", handler.GetLicense)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestLicenseListFeatures(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns all features", func(t *testing.T) {
		store := &mockLicenseStore{tier: license.TierFree}
		checker := license.NewFeatureChecker(store)
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(InjectUser(user))
		handler := NewLicenseHandler(store, checker, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license/features"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result FeaturesResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(result.Features) == 0 {
			t.Error("expected at least one feature")
		}
	})
}

func TestLicenseCheckFeature(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	setup := func(store *mockLicenseStore) *gin.Engine {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(InjectUser(user))
		checker := license.NewFeatureChecker(store)
		handler := NewLicenseHandler(store, checker, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)
		return r
	}

	t.Run("invalid feature name returns 400", func(t *testing.T) {
		store := &mockLicenseStore{tier: license.TierFree}
		r := setup(store)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license/features/not-a-feature/check"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("valid feature returns result", func(t *testing.T) {
		store := &mockLicenseStore{tier: license.TierPro}
		r := setup(store)
		// Pick the first valid feature from the catalog.
		features := license.AllFeatures()
		if len(features) == 0 {
			t.Skip("no features defined")
		}
		feat := string(features[0])
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license/features/"+feat+"/check"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestLicenseListTiers(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns tier info", func(t *testing.T) {
		store := &mockLicenseStore{tier: license.TierFree}
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(InjectUser(user))
		checker := license.NewFeatureChecker(store)
		handler := NewLicenseHandler(store, checker, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license/tiers"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result TiersResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(result.Tiers) == 0 {
			t.Error("expected at least one tier")
		}
	})
}
