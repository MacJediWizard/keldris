package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestAPIKeyMiddleware_ValidKey(t *testing.T) {
	agent := newTestAgent()
	keyHash := auth.HashAPIKey(testAPIKey)
	validator := newTestValidator(map[string]*models.Agent{keyHash: agent})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		a := GetAgent(c)
		if a == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent not in context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"agent_id": a.ID.String(),
			"hostname": a.Hostname,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["agent_id"] != agent.ID.String() {
		t.Fatalf("expected agent_id %s, got %s", agent.ID, resp["agent_id"])
	}
	if resp["hostname"] != agent.Hostname {
		t.Fatalf("expected hostname %s, got %s", agent.Hostname, resp["hostname"])
	}
}

func TestAPIKeyMiddleware_InvalidKey(t *testing.T) {
	validator := newTestValidator(map[string]*models.Agent{})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Use a valid format key that doesn't match any agent
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer kld_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "invalid API key" {
		t.Fatalf("expected error 'invalid API key', got %q", resp["error"])
	}
}

func TestAPIKeyMiddleware_MissingKey(t *testing.T) {
	validator := newTestValidator(map[string]*models.Agent{})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// No Authorization header at all
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "authorization required" {
		t.Fatalf("expected error 'authorization required', got %q", resp["error"])
	}
}

func TestAPIKeyMiddleware_ExpiredKey(t *testing.T) {
	// An "expired" key is modeled as a disabled agent
	agent := &models.Agent{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Hostname: "disabled-agent",
		Status:   models.AgentStatusDisabled,
	}
	keyHash := auth.HashAPIKey(testAPIKey)
	validator := newTestValidator(map[string]*models.Agent{keyHash: agent})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for disabled agent, got %d", w.Code)
	}
}

func TestAPIKeyMiddleware_RevokedKey(t *testing.T) {
	// A revoked key returns no agent from the store (key hash not found)
	validator := newTestValidator(map[string]*models.Agent{})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for revoked key, got %d", w.Code)
	}
}

func TestAPIKeyMiddleware_InvalidFormat(t *testing.T) {
	validator := newTestValidator(map[string]*models.Agent{})

	mw := APIKeyMiddleware(validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	tests := []struct {
		name   string
		header string
		errMsg string
	}{
		{
			name:   "no bearer prefix",
			header: "Basic some-token",
			errMsg: "invalid authorization format",
		},
		{
			name:   "empty bearer token",
			header: "Bearer ",
			errMsg: "invalid authorization format",
		},
		{
			name:   "wrong key prefix",
			header: "Bearer xyz_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			errMsg: "invalid API key",
		},
		{
			name:   "too short key",
			header: "Bearer kld_aabb",
			errMsg: "invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d", w.Code)
			}

			var resp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if resp["error"] != tt.errMsg {
				t.Fatalf("expected error %q, got %q", tt.errMsg, resp["error"])
			}
		})
	}
}
