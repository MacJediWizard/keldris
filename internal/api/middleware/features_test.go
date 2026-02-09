package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestLicenseMiddleware(t *testing.T) {
	lic := license.FreeLicense()

	r := gin.New()
	r.Use(LicenseMiddleware(lic, zerolog.Nop()))
	r.GET("/test", func(c *gin.Context) {
		got := GetLicense(c)
		if got == nil {
			t.Fatal("expected license in context")
		}
		if got.Tier != license.TierFree {
			t.Fatalf("expected free tier, got %s", got.Tier)
		}
		c.JSON(http.StatusOK, gin.H{"tier": string(got.Tier)})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestFeatureMiddleware_FeatureAvailable(t *testing.T) {
	lic := &license.License{Tier: license.TierPro}

	r := gin.New()
	r.Use(LicenseMiddleware(lic, zerolog.Nop()))
	r.Use(FeatureMiddleware(license.FeatureOIDC, zerolog.Nop()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFeatureMiddleware_FeatureNotAvailable(t *testing.T) {
	lic := &license.License{Tier: license.TierFree}

	r := gin.New()
	r.Use(LicenseMiddleware(lic, zerolog.Nop()))
	r.Use(FeatureMiddleware(license.FeatureOIDC, zerolog.Nop()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status 402, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["feature"] != "oidc" {
		t.Fatalf("expected feature 'oidc', got %q", resp["feature"])
	}
}

func TestFeatureMiddleware_NoLicense(t *testing.T) {
	r := gin.New()
	// No LicenseMiddleware - license won't be in context
	r.Use(FeatureMiddleware(license.FeatureOIDC, zerolog.Nop()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status 402, got %d", w.Code)
	}
}

func TestFeatureMiddleware_EnterpriseFeature(t *testing.T) {
	t.Run("enterprise tier has access", func(t *testing.T) {
		lic := &license.License{Tier: license.TierEnterprise}

		r := gin.New()
		r.Use(LicenseMiddleware(lic, zerolog.Nop()))
		r.Use(FeatureMiddleware(license.FeatureMultiOrg, zerolog.Nop()))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("pro tier lacks enterprise feature", func(t *testing.T) {
		lic := &license.License{Tier: license.TierPro}

		r := gin.New()
		r.Use(LicenseMiddleware(lic, zerolog.Nop()))
		r.Use(FeatureMiddleware(license.FeatureMultiOrg, zerolog.Nop()))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusPaymentRequired {
			t.Fatalf("expected status 402, got %d", w.Code)
		}
	})
}

func TestGetLicense_NoLicense(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		lic := GetLicense(c)
		if lic != nil {
			t.Fatal("expected nil license")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetLicense_WrongType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(LicenseContextKey), "not-a-license")
		c.Next()
	})
	r.GET("/test", func(c *gin.Context) {
		lic := GetLicense(c)
		if lic != nil {
			t.Fatal("expected nil license for wrong type")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}
