package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/updates"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestNewUpdatesHandler(t *testing.T) {
	logger := zerolog.Nop()
	cfg := updates.DefaultConfig("1.0.0")
	checker := updates.NewChecker(cfg, logger)

	handler := NewUpdatesHandler(checker, logger)

	if handler == nil {
		t.Fatal("NewUpdatesHandler returned nil")
	}
}

func TestUpdatesHandler_Status_NilChecker(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewUpdatesHandler(nil, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/status", nil)

	handler.Status(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status() code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp UpdateStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Enabled {
		t.Error("Expected Enabled to be false when checker is nil")
	}
	if resp.ChangelogURL == "" {
		t.Error("Expected ChangelogURL to be set")
	}
	if resp.UpgradeDocsURL == "" {
		t.Error("Expected UpgradeDocsURL to be set")
	}
}

func TestUpdatesHandler_Status_Enabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := updates.DefaultConfig("1.0.0")
	checker := updates.NewChecker(cfg, logger)
	handler := NewUpdatesHandler(checker, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/status", nil)

	handler.Status(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status() code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp UpdateStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestUpdatesHandler_Status_Disabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := updates.DefaultConfig("1.0.0")
	cfg.Enabled = false
	checker := updates.NewChecker(cfg, logger)
	handler := NewUpdatesHandler(checker, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/status", nil)

	handler.Status(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status() code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp UpdateStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Enabled {
		t.Error("Expected Enabled to be false when disabled")
	}
}

func TestUpdatesHandler_Check_NilChecker(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewUpdatesHandler(nil, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/check", nil)

	handler.Check(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Check() code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestUpdatesHandler_Check_Disabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := updates.DefaultConfig("1.0.0")
	cfg.Enabled = false
	checker := updates.NewChecker(cfg, logger)
	handler := NewUpdatesHandler(checker, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/check", nil)

	handler.Check(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Check() code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["enabled"] != false {
		t.Error("Expected enabled to be false in response")
	}
}

func TestUpdatesHandler_Check_AirGapMode(t *testing.T) {
	logger := zerolog.Nop()
	cfg := updates.DefaultConfig("1.0.0")
	cfg.AirGapMode = true
	checker := updates.NewChecker(cfg, logger)
	handler := NewUpdatesHandler(checker, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/updates/check", nil)

	handler.Check(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Check() code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["air_gap_mode"] != true {
		t.Error("Expected air_gap_mode to be true in response")
	}
}

func TestUpdatesHandler_ForceCheck_NilChecker(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewUpdatesHandler(nil, logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/updates/check", nil)

	// Set up a mock user with admin role
	c.Set(string(middleware.UserContextKey), &auth.SessionUser{
		ID:             uuid.New(),
		CurrentOrgRole: "admin",
	})

	handler.ForceCheck(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ForceCheck() code = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestUpdateStatusResponse_JSON(t *testing.T) {
	resp := UpdateStatusResponse{
		Enabled:        true,
		AirGapMode:     false,
		ChangelogURL:   "https://example.com/changelog",
		UpgradeDocsURL: "https://example.com/upgrade",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled UpdateStatusResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Enabled != resp.Enabled {
		t.Errorf("Enabled = %v, want %v", unmarshaled.Enabled, resp.Enabled)
	}
	if unmarshaled.ChangelogURL != resp.ChangelogURL {
		t.Errorf("ChangelogURL = %q, want %q", unmarshaled.ChangelogURL, resp.ChangelogURL)
	}
}
