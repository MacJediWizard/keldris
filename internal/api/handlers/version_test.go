package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func setupVersionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewVersionHandler("1.0.0", "abc1234", "2024-01-15T10:30:00Z", zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

func TestVersionGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := setupVersionTestRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/version", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp VersionInfo
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Version != "1.0.0" {
			t.Fatalf("expected version '1.0.0', got %q", resp.Version)
		}
		if resp.Commit != "abc1234" {
			t.Fatalf("expected commit 'abc1234', got %q", resp.Commit)
		}
		if resp.BuildDate != "2024-01-15T10:30:00Z" {
			t.Fatalf("expected build_date '2024-01-15T10:30:00Z', got %q", resp.BuildDate)
		}
	})

	t.Run("via router group", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		handler := NewVersionHandler("2.0.0", "", "", zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/version", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp VersionInfo
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Version != "2.0.0" {
			t.Fatalf("expected version '2.0.0', got %q", resp.Version)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		handler := NewVersionHandler("", "", "", zerolog.Nop())
		handler.RegisterPublicRoutes(r)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/version", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}
