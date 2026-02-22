package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestAirGapHandler_GetStatus(t *testing.T) {
	t.Run("air-gap disabled", func(t *testing.T) {
		os.Unsetenv("AIR_GAP_MODE")

		orgID := uuid.New()
		user := testUser(orgID)
		r := SetupTestRouter(user)
		handler := NewAirGapHandler(zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		req := AuthenticatedRequest("GET", "/api/v1/system/airgap")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp AirGapStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Enabled {
			t.Error("expected air-gap to be disabled")
		}
		if len(resp.DisabledFeatures) != 0 {
			t.Errorf("expected no disabled features when air-gap is off, got %d", len(resp.DisabledFeatures))
		}
	})

	t.Run("air-gap enabled", func(t *testing.T) {
		os.Setenv("AIR_GAP_MODE", "true")
		defer os.Unsetenv("AIR_GAP_MODE")

		orgID := uuid.New()
		user := testUser(orgID)
		r := SetupTestRouter(user)
		handler := NewAirGapHandler(zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		req := AuthenticatedRequest("GET", "/api/v1/system/airgap")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp AirGapStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.Enabled {
			t.Error("expected air-gap to be enabled")
		}
		if len(resp.DisabledFeatures) == 0 {
			t.Error("expected disabled features when air-gap is on")
		}

		// Verify expected disabled features are present
		featureNames := make(map[string]bool)
		for _, f := range resp.DisabledFeatures {
			featureNames[f.Name] = true
		}
		expectedFeatures := []string{"auto_update", "external_webhooks", "telemetry", "cloud_storage_validation"}
		for _, name := range expectedFeatures {
			if !featureNames[name] {
				t.Errorf("expected disabled feature %q not found", name)
			}
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := SetupTestRouter(nil) // no user injected
		handler := NewAirGapHandler(zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		req := AuthenticatedRequest("GET", "/api/v1/system/airgap")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}
