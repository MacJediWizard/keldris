package handlers

import (
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func setupLicenseManageTestRouter(validator *license.Validator) *gin.Engine {
	r := SetupTestRouter(testUser(uuid.New()))
	checker := license.NewFeatureChecker(&stubFeatureStore{tier: license.TierFree})
	handler := NewLicenseManageHandler(validator, checker, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestLicenseManageActivateWithoutValidator(t *testing.T) {
	r := setupLicenseManageTestRouter(nil)

	resp := DoRequest(r, JSONRequest("POST", "/api/v1/system/license/activate", `{"license_key":"x"}`))
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
}

func TestLicenseManageDeactivateWithoutValidator(t *testing.T) {
	r := setupLicenseManageTestRouter(nil)

	resp := DoRequest(r, JSONRequest("POST", "/api/v1/system/license/deactivate", `{}`))
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
}

func TestLicenseManageGetPlansWithoutValidator(t *testing.T) {
	r := setupLicenseManageTestRouter(nil)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/license/plans"))
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
}

func TestLicenseManageStartTrialWithoutValidator(t *testing.T) {
	r := setupLicenseManageTestRouter(nil)

	resp := DoRequest(r, JSONRequest("POST", "/api/v1/system/license/trial/start", `{}`))
	if resp.Code != http.StatusServiceUnavailable && resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 503/400, got %d", resp.Code)
	}
}
