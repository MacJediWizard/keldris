package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func setupLicenseInfoTestRouter(injectLicense *license.License) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if injectLicense != nil {
		r.Use(func(c *gin.Context) {
			c.Set(string(middleware.LicenseContextKey), injectLicense)
			c.Next()
		})
	}
	// Pass nil validator – Get handler tolerates this and reports "none" as source.
	handler := NewLicenseInfoHandler(nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestLicenseInfoGet(t *testing.T) {
	t.Run("returns free tier when no license in context", func(t *testing.T) {
		r := setupLicenseInfoTestRouter(nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result LicenseInfoResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.Tier != string(license.TierFree) {
			t.Errorf("expected tier %s, got %s", license.TierFree, result.Tier)
		}
		if result.LicenseKeySource != "none" {
			t.Errorf("expected source 'none', got %s", result.LicenseKeySource)
		}
	})

	t.Run("returns provided license tier", func(t *testing.T) {
		lic := &license.License{
			Tier:       license.TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
			IssuedAt:   time.Now(),
		}
		r := setupLicenseInfoTestRouter(lic)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/license"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result LicenseInfoResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.Tier != string(license.TierPro) {
			t.Errorf("expected tier %s, got %s", license.TierPro, result.Tier)
		}
		if result.CustomerID != "cust-123" {
			t.Errorf("expected customer cust-123, got %s", result.CustomerID)
		}
	})
}
