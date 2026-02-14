package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockAirGapStore struct {
	license   *models.OfflineLicense
	createErr error
	getErr    error
}

func (m *mockAirGapStore) CreateOfflineLicense(_ context.Context, license *models.OfflineLicense) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.license = license
	return nil
}

func (m *mockAirGapStore) GetLatestOfflineLicense(_ context.Context, _ uuid.UUID) (*models.OfflineLicense, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.license, nil
}

func setupAirGapTestRouter(store AirGapStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAirGapHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAirGapHandler_GetStatus(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("returns status with no license", func(t *testing.T) {
		store := &mockAirGapStore{}
		r := setupAirGapTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/airgap"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var result AirGapStatusResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.License != nil {
			t.Error("expected no license in response")
		}
	})

	t.Run("returns status with valid license", func(t *testing.T) {
		store := &mockAirGapStore{
			license: &models.OfflineLicense{
				ID:         uuid.New(),
				OrgID:      orgID,
				CustomerID: "cust-123",
				Tier:       "enterprise",
				ExpiresAt:  time.Now().Add(24 * time.Hour),
				IssuedAt:   time.Now().Add(-24 * time.Hour),
			},
		}
		r := setupAirGapTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/airgap"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var result AirGapStatusResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.License == nil {
			t.Fatal("expected license in response")
		}
		if result.License.CustomerID != "cust-123" {
			t.Errorf("expected customer_id 'cust-123', got %q", result.License.CustomerID)
		}
		if result.License.Tier != "enterprise" {
			t.Errorf("expected tier 'enterprise', got %q", result.License.Tier)
		}
		if !result.License.Valid {
			t.Error("expected license to be valid")
		}
	})

	t.Run("returns status with expired license", func(t *testing.T) {
		store := &mockAirGapStore{
			license: &models.OfflineLicense{
				ID:         uuid.New(),
				OrgID:      orgID,
				CustomerID: "cust-expired",
				Tier:       "enterprise",
				ExpiresAt:  time.Now().Add(-24 * time.Hour),
				IssuedAt:   time.Now().Add(-48 * time.Hour),
			},
		}
		r := setupAirGapTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/airgap"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var result AirGapStatusResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.License == nil {
			t.Fatal("expected license in response")
		}
		if result.License.Valid {
			t.Error("expected license to be invalid (expired)")
		}
	})

	t.Run("handles store error gracefully", func(t *testing.T) {
		store := &mockAirGapStore{
			getErr: errors.New("db error"),
		}
		r := setupAirGapTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/airgap"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 even with store error, got %d", resp.Code)
		}

		var result AirGapStatusResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if result.License != nil {
			t.Error("expected no license when store errors")
		}
	})
}

func TestAirGapHandler_UploadLicense(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("rejects empty body", func(t *testing.T) {
		store := &mockAirGapStore{}
		r := setupAirGapTestRouter(store, user)

		req := JSONRequest("POST", "/api/v1/system/license", "")
		resp := DoRequest(r, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects invalid license data", func(t *testing.T) {
		store := &mockAirGapStore{}
		r := setupAirGapTestRouter(store, user)

		req, _ := http.NewRequest("POST", "/api/v1/system/license", strings.NewReader("invalid-license-data"))
		req.Header.Set("Content-Type", "application/octet-stream")
		resp := DoRequest(r, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		store := &mockAirGapStore{}
		noOrgUser := TestUserNoOrg()
		r := setupAirGapTestRouter(store, noOrgUser)

		req, _ := http.NewRequest("POST", "/api/v1/system/license", strings.NewReader("some-data"))
		req.Header.Set("Content-Type", "application/octet-stream")
		resp := DoRequest(r, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestAirGapHandler_Unauthorized(t *testing.T) {
	store := &mockAirGapStore{}

	t.Run("get status without auth returns 401", func(t *testing.T) {
		r := setupAirGapTestRouter(store, nil)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system/airgap"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})

	t.Run("upload license without auth returns 401", func(t *testing.T) {
		r := setupAirGapTestRouter(store, nil)

		req, _ := http.NewRequest("POST", "/api/v1/system/license", strings.NewReader("data"))
		req.Header.Set("Content-Type", "application/octet-stream")
		resp := DoRequest(r, req)

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}
