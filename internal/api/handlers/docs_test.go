package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func setupDocsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	docsFS := fstest.MapFS{
		"getting-started.md": &fstest.MapFile{Data: []byte("# Getting Started\n\nWelcome to Keldris backup engine.\n")},
		"installation.md":    &fstest.MapFile{Data: []byte("# Installation\n\nInstall the server first.\n")},
	}
	handler := NewDocsHandler(docsFS, zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

func TestDocsList(t *testing.T) {
	r := setupDocsTestRouter()

	t.Run("returns ordered pages", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDocsGet(t *testing.T) {
	r := setupDocsTestRouter()

	t.Run("returns known page", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/getting-started", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown slug returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/bogus", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("missing file returns 404", func(t *testing.T) {
		// configuration is in metadata but not in our fake FS
		req, _ := http.NewRequest("GET", "/docs/configuration", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDocsGetHTML(t *testing.T) {
	r := setupDocsTestRouter()

	t.Run("returns rendered html", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/getting-started/html", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown slug returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/bogus/html", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDocsSearch(t *testing.T) {
	r := setupDocsTestRouter()

	t.Run("returns results", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/search?q=install", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing query returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs/search", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}
