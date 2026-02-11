package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockMaintenanceStore struct {
	windows   []*models.MaintenanceWindow
	active    []*models.MaintenanceWindow
	upcoming  []*models.MaintenanceWindow
	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
	activeErr error
	upcomErr  error
}

func (m *mockMaintenanceStore) GetMaintenanceWindowByID(_ context.Context, id uuid.UUID) (*models.MaintenanceWindow, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, w := range m.windows {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockMaintenanceStore) ListMaintenanceWindowsByOrg(_ context.Context, _ uuid.UUID) ([]*models.MaintenanceWindow, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.windows, nil
}

func (m *mockMaintenanceStore) ListActiveMaintenanceWindows(_ context.Context, _ uuid.UUID, _ time.Time) ([]*models.MaintenanceWindow, error) {
	if m.activeErr != nil {
		return nil, m.activeErr
	}
	return m.active, nil
}

func (m *mockMaintenanceStore) ListUpcomingMaintenanceWindows(_ context.Context, _ uuid.UUID, _ time.Time, _ int) ([]*models.MaintenanceWindow, error) {
	if m.upcomErr != nil {
		return nil, m.upcomErr
	}
	return m.upcoming, nil
}

func (m *mockMaintenanceStore) CreateMaintenanceWindow(_ context.Context, _ *models.MaintenanceWindow) error {
	return m.createErr
}

func (m *mockMaintenanceStore) UpdateMaintenanceWindow(_ context.Context, _ *models.MaintenanceWindow) error {
	return m.updateErr
}

func (m *mockMaintenanceStore) DeleteMaintenanceWindow(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func setupMaintenanceTestRouter(store MaintenanceStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewMaintenanceHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func testMaintenanceWindow(orgID uuid.UUID) *models.MaintenanceWindow {
	return &models.MaintenanceWindow{
		ID:       uuid.New(),
		OrgID:    orgID,
		Title:    "Test Maintenance",
		Message:  "Scheduled maintenance",
		StartsAt: time.Now().Add(1 * time.Hour),
		EndsAt:   time.Now().Add(3 * time.Hour),
	}
}

func TestMaintenanceList(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		w1 := testMaintenanceWindow(orgID)
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}

		var body models.MaintenanceWindowsResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body.MaintenanceWindows) != 1 {
			t.Fatalf("expected 1 window, got %d", len(body.MaintenanceWindows))
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, TestUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockMaintenanceStore{listErr: errors.New("db error")}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, nil)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestMaintenanceGet(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	w1 := testMaintenanceWindow(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows/"+w1.ID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows/not-a-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockMaintenanceStore{getErr: errors.New("not found")}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows/"+uuid.New().String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherWindow := testMaintenanceWindow(uuid.New()) // different org
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{otherWindow}}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance-windows/"+otherWindow.ID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestMaintenanceCreate(t *testing.T) {
	orgID := uuid.New()
	adminUser := TestUser(orgID)
	adminUser.CurrentOrgRole = "admin"

	startsAt := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
	endsAt := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)

	t.Run("success", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, adminUser)

		body := fmt.Sprintf(`{"title":"Test","starts_at":"%s","ends_at":"%s"}`,
			startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339))
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/maintenance-windows", body))

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		memberUser := TestUser(orgID)
		memberUser.CurrentOrgRole = "member"
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, memberUser)

		body := fmt.Sprintf(`{"title":"Test","starts_at":"%s","ends_at":"%s"}`,
			startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339))
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/maintenance-windows", body))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/maintenance-windows", `{}`))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("ends before starts", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, adminUser)

		body := fmt.Sprintf(`{"title":"Test","starts_at":"%s","ends_at":"%s"}`,
			endsAt.Format(time.RFC3339), startsAt.Format(time.RFC3339))
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/maintenance-windows", body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockMaintenanceStore{createErr: errors.New("db error")}
		r := setupMaintenanceTestRouter(store, adminUser)

		body := fmt.Sprintf(`{"title":"Test","starts_at":"%s","ends_at":"%s"}`,
			startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339))
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/maintenance-windows", body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestMaintenanceUpdate(t *testing.T) {
	orgID := uuid.New()
	adminUser := TestUser(orgID)
	adminUser.CurrentOrgRole = "admin"
	w1 := testMaintenanceWindow(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, adminUser)

		body := `{"title":"Updated Title"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/"+w1.ID.String(), body))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		memberUser := TestUser(orgID)
		memberUser.CurrentOrgRole = "member"
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, memberUser)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/"+w1.ID.String(), `{"title":"x"}`))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/bad-id", `{"title":"x"}`))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockMaintenanceStore{getErr: errors.New("not found")}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/"+uuid.New().String(), `{"title":"x"}`))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherWindow := testMaintenanceWindow(uuid.New())
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{otherWindow}}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/"+otherWindow.ID.String(), `{"title":"x"}`))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockMaintenanceStore{
			windows:   []*models.MaintenanceWindow{w1},
			updateErr: errors.New("db error"),
		}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/maintenance-windows/"+w1.ID.String(), `{"title":"x"}`))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestMaintenanceDelete(t *testing.T) {
	orgID := uuid.New()
	adminUser := TestUser(orgID)
	adminUser.CurrentOrgRole = "admin"
	w1 := testMaintenanceWindow(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/maintenance-windows/"+w1.ID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		memberUser := TestUser(orgID)
		memberUser.CurrentOrgRole = "member"
		store := &mockMaintenanceStore{windows: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, memberUser)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/maintenance-windows/"+w1.ID.String()))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockMaintenanceStore{getErr: errors.New("not found")}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/maintenance-windows/"+uuid.New().String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockMaintenanceStore{
			windows:   []*models.MaintenanceWindow{w1},
			deleteErr: errors.New("db error"),
		}
		r := setupMaintenanceTestRouter(store, adminUser)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/maintenance-windows/"+w1.ID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestMaintenanceGetActive(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("no active or upcoming", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance/active"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}

		var body models.ActiveMaintenanceResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Active != nil {
			t.Fatal("expected no active window")
		}
	})

	t.Run("with active window", func(t *testing.T) {
		w1 := testMaintenanceWindow(orgID)
		store := &mockMaintenanceStore{active: []*models.MaintenanceWindow{w1}}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance/active"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}

		var body models.ActiveMaintenanceResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Active == nil {
			t.Fatal("expected active window")
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockMaintenanceStore{}
		r := setupMaintenanceTestRouter(store, TestUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance/active"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("active store error", func(t *testing.T) {
		store := &mockMaintenanceStore{activeErr: errors.New("db error")}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance/active"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("upcoming store error", func(t *testing.T) {
		store := &mockMaintenanceStore{upcomErr: errors.New("db error")}
		r := setupMaintenanceTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/maintenance/active"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
