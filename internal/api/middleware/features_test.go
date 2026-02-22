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

func TestFeatureMiddleware_Allowed(t *testing.T) {
	// Pro tier has OIDC feature
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

func TestFeatureMiddleware_Blocked(t *testing.T) {
	// Free tier does NOT have OIDC
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
	if resp["tier"] != "free" {
		t.Fatalf("expected tier 'free', got %q", resp["tier"])
	}
}

func TestFeatureMiddleware_FreeTier(t *testing.T) {
	lic := &license.License{Tier: license.TierFree}

	features := []struct {
		name    string
		feature license.Feature
		allowed bool
	}{
		{"OIDC blocked", license.FeatureOIDC, false},
		{"audit logs blocked", license.FeatureAuditLogs, false},
		{"multi-org blocked", license.FeatureMultiOrg, false},
		{"Slack notifications blocked", license.FeatureNotificationSlack, false},
		{"Teams notifications blocked", license.FeatureNotificationTeams, false},
		{"PagerDuty notifications blocked", license.FeatureNotificationPagerDuty, false},
		{"Discord notifications blocked", license.FeatureNotificationDiscord, false},
		{"S3 storage blocked", license.FeatureStorageS3, false},
		{"B2 storage blocked", license.FeatureStorageB2, false},
		{"SFTP storage blocked", license.FeatureStorageSFTP, false},
		{"Docker backup blocked", license.FeatureDockerBackup, false},
		{"multi repo blocked", license.FeatureMultiRepo, false},
		{"API access blocked", license.FeatureAPIAccess, false},
		{"DR runbooks blocked", license.FeatureDRRunbooks, false},
		{"DR tests blocked", license.FeatureDRTests, false},
	}

	for _, tt := range features {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(LicenseMiddleware(lic, zerolog.Nop()))
			r.Use(FeatureMiddleware(tt.feature, zerolog.Nop()))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			if tt.allowed && w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}
			if !tt.allowed && w.Code != http.StatusPaymentRequired {
				t.Fatalf("expected status 402, got %d", w.Code)
			}
		})
	}
}

func TestFeatureMiddleware_ProTier(t *testing.T) {
	lic := &license.License{Tier: license.TierPro}

	features := []struct {
		name    string
		feature license.Feature
		allowed bool
	}{
		{"OIDC allowed", license.FeatureOIDC, true},
		{"audit logs allowed", license.FeatureAuditLogs, true},
		{"Slack notifications allowed", license.FeatureNotificationSlack, true},
		{"Teams notifications allowed", license.FeatureNotificationTeams, true},
		{"PagerDuty notifications allowed", license.FeatureNotificationPagerDuty, true},
		{"Discord notifications allowed", license.FeatureNotificationDiscord, true},
		{"S3 storage allowed", license.FeatureStorageS3, true},
		{"B2 storage allowed", license.FeatureStorageB2, true},
		{"SFTP storage allowed", license.FeatureStorageSFTP, true},
		{"Docker backup allowed", license.FeatureDockerBackup, true},
		{"multi repo allowed", license.FeatureMultiRepo, true},
		{"API access allowed", license.FeatureAPIAccess, true},
		{"multi-org blocked", license.FeatureMultiOrg, false},
		{"SLA tracking blocked", license.FeatureSLATracking, false},
		{"white label blocked", license.FeatureWhiteLabel, false},
		{"air gap blocked", license.FeatureAirGap, false},
		{"DR runbooks blocked", license.FeatureDRRunbooks, false},
		{"DR tests blocked", license.FeatureDRTests, false},
	}

	for _, tt := range features {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(LicenseMiddleware(lic, zerolog.Nop()))
			r.Use(FeatureMiddleware(tt.feature, zerolog.Nop()))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			if tt.allowed && w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}
			if !tt.allowed && w.Code != http.StatusPaymentRequired {
				t.Fatalf("expected status 402, got %d", w.Code)
			}
		})
	}
}

func TestFeatureMiddleware_EnterpriseTier(t *testing.T) {
	lic := &license.License{Tier: license.TierEnterprise}

	features := []struct {
		name    string
		feature license.Feature
	}{
		{"OIDC", license.FeatureOIDC},
		{"audit logs", license.FeatureAuditLogs},
		{"Slack notifications", license.FeatureNotificationSlack},
		{"Teams notifications", license.FeatureNotificationTeams},
		{"PagerDuty notifications", license.FeatureNotificationPagerDuty},
		{"Discord notifications", license.FeatureNotificationDiscord},
		{"S3 storage", license.FeatureStorageS3},
		{"B2 storage", license.FeatureStorageB2},
		{"SFTP storage", license.FeatureStorageSFTP},
		{"Docker backup", license.FeatureDockerBackup},
		{"multi repo", license.FeatureMultiRepo},
		{"API access", license.FeatureAPIAccess},
		{"multi-org", license.FeatureMultiOrg},
		{"SLA tracking", license.FeatureSLATracking},
		{"white label", license.FeatureWhiteLabel},
		{"air gap", license.FeatureAirGap},
		{"DR runbooks", license.FeatureDRRunbooks},
		{"DR tests", license.FeatureDRTests},
	}

	for _, tt := range features {
		t.Run(tt.name+" allowed", func(t *testing.T) {
			r := gin.New()
			r.Use(LicenseMiddleware(lic, zerolog.Nop()))
			r.Use(FeatureMiddleware(tt.feature, zerolog.Nop()))
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

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "license required" {
		t.Fatalf("expected error 'license required', got %q", resp["error"])
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

func TestFeatureMiddleware_ResponseIncludesFeatureAndTier(t *testing.T) {
	lic := &license.License{Tier: license.TierFree}

	r := gin.New()
	r.Use(LicenseMiddleware(lic, zerolog.Nop()))
	r.Use(FeatureMiddleware(license.FeatureAuditLogs, zerolog.Nop()))
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
	if resp["feature"] != "audit_logs" {
		t.Fatalf("expected feature 'audit_logs', got %q", resp["feature"])
	}
	if resp["tier"] != "free" {
		t.Fatalf("expected tier 'free', got %q", resp["tier"])
	}
	if resp["error"] != "feature not available on your current plan" {
		t.Fatalf("expected upgrade message, got %q", resp["error"])
	}
}
