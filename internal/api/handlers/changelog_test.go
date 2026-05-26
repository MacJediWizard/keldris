package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func setupChangelogTestRouter(path, version string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewChangelogHandler(path, version, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func writeChangelog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write changelog: %v", err)
	}
	return path
}

func TestChangelogList(t *testing.T) {
	t.Run("returns parsed entries", func(t *testing.T) {
		content := `# Changelog

## [1.0.0] - 2026-01-01
### Added
- New feature

### Fixed
- Bug fix
`
		path := writeChangelog(t, content)
		r := setupChangelogTestRouter(path, "1.0.0")

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/changelog"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result ChangelogResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.CurrentVersion != "1.0.0" {
			t.Errorf("expected current version 1.0.0, got %s", result.CurrentVersion)
		}
		if len(result.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result.Entries))
		}
		if result.Entries[0].Version != "1.0.0" {
			t.Errorf("expected version 1.0.0, got %s", result.Entries[0].Version)
		}
		if len(result.Entries[0].Added) != 1 || result.Entries[0].Added[0] != "New feature" {
			t.Errorf("expected Added[0]=New feature, got %v", result.Entries[0].Added)
		}
	})

	t.Run("missing file returns empty entries", func(t *testing.T) {
		r := setupChangelogTestRouter("/nonexistent/CHANGELOG.md", "0.0.1")
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/changelog"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var result ChangelogResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(result.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(result.Entries))
		}
	})
}

func TestChangelogGet(t *testing.T) {
	content := `# Changelog

## [2.0.0] - 2026-02-01
### Added
- Cool stuff
`
	path := writeChangelog(t, content)
	r := setupChangelogTestRouter(path, "2.0.0")

	t.Run("returns specific version", func(t *testing.T) {
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/changelog/2.0.0"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var entry ChangelogEntry
		if err := json.Unmarshal(resp.Body.Bytes(), &entry); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if entry.Version != "2.0.0" {
			t.Errorf("expected 2.0.0, got %s", entry.Version)
		}
	})

	t.Run("unknown version returns 404", func(t *testing.T) {
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/changelog/9.9.9"))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}
