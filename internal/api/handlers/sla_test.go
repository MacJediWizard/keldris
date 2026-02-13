package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockSLAStore struct {
	policies      []*models.SLAPolicy
	policyByID    map[uuid.UUID]*models.SLAPolicy
	history       []*models.SLAStatusSnapshot
	latestStatus  *models.SLAStatusSnapshot
	successRate   float64
	maxRPOHours   float64
	user          *models.User
	createErr     error
	updateErr     error
	deleteErr     error
	listErr       error
	historyErr    error
	statusErr     error
	snapshotErr   error
}

func (m *mockSLAStore) CreateSLAPolicy(_ context.Context, _ *models.SLAPolicy) error {
	return m.createErr
}

func (m *mockSLAStore) GetSLAPolicyByID(_ context.Context, id uuid.UUID) (*models.SLAPolicy, error) {
	if p, ok := m.policyByID[id]; ok {
		return p, nil
	}
	return nil, errors.New("policy not found")
}

func (m *mockSLAStore) ListSLAPoliciesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.SLAPolicy, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*models.SLAPolicy
	for _, p := range m.policies {
		if p.OrgID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockSLAStore) UpdateSLAPolicy(_ context.Context, _ *models.SLAPolicy) error {
	return m.updateErr
}

func (m *mockSLAStore) DeleteSLAPolicy(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockSLAStore) CreateSLAStatusSnapshot(_ context.Context, _ *models.SLAStatusSnapshot) error {
	return m.snapshotErr
}

func (m *mockSLAStore) GetSLAStatusHistory(_ context.Context, _ uuid.UUID, _ int) ([]*models.SLAStatusSnapshot, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history, nil
}

func (m *mockSLAStore) GetLatestSLAStatus(_ context.Context, _ uuid.UUID) (*models.SLAStatusSnapshot, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.latestStatus, nil
}

func (m *mockSLAStore) GetBackupSuccessRateForOrg(_ context.Context, _ uuid.UUID, _ int) (float64, error) {
	return m.successRate, nil
}

func (m *mockSLAStore) GetMaxRPOHoursForOrg(_ context.Context, _ uuid.UUID) (float64, error) {
	return m.maxRPOHours, nil
}

func (m *mockSLAStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func setupSLATestRouter(store SLAStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewSLAHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListSLAPolicies(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy", TargetRPOHours: 24, TargetRTOHours: 4, TargetSuccessRate: 99.5}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policies:   []*models.SLAPolicy{policy},
		policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:       dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["policies"]; !ok {
			t.Fatal("expected 'policies' key")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockSLAStore{user: nil}
		wrongUserSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupSLATestRouter(noUserStore, wrongUserSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockSLAStore{user: dbUser, listErr: errors.New("db error")}
		r := setupSLATestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestCreateSLAPolicy(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID: map[uuid.UUID]*models.SLAPolicy{},
		user:       dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"Gold SLA","target_rpo_hours":24,"target_rto_hours":4,"target_success_rate":99.5}`
		req, _ := http.NewRequest("POST", "/api/v1/sla/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"target_rpo_hours":24,"target_rto_hours":4,"target_success_rate":99.5}`
		req, _ := http.NewRequest("POST", "/api/v1/sla/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{},
			user:       dbUser,
			createErr:  errors.New("db error"),
		}
		r := setupSLATestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"Gold SLA","target_rpo_hours":24,"target_rto_hours":4,"target_success_rate":99.5}`
		req, _ := http.NewRequest("POST", "/api/v1/sla/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"Gold SLA","target_rpo_hours":24,"target_rto_hours":4,"target_success_rate":99.5}`
		req, _ := http.NewRequest("POST", "/api/v1/sla/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockSLAStore{user: nil}
		wrongUserSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupSLATestRouter(noUserStore, wrongUserSession)
		w := httptest.NewRecorder()
		body := `{"name":"Gold SLA","target_rpo_hours":24,"target_rto_hours":4,"target_success_rate":99.5}`
		req, _ := http.NewRequest("POST", "/api/v1/sla/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetSLAPolicy(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:       dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupSLATestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestUpdateSLAPolicy(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy", TargetRPOHours: 24, TargetRTOHours: 4, TargetSuccessRate: 99.5}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:       dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"Updated SLA"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"Updated SLA"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"Updated SLA"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherOrgID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: otherOrgID, Role: models.UserRoleAdmin}
		wrongStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       otherUser,
		}
		wrongSession := &auth.SessionUser{ID: otherUserID, CurrentOrgID: otherOrgID}
		r := setupSLATestRouter(wrongStore, wrongSession)
		w := httptest.NewRecorder()
		body := `{"name":"Updated SLA"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		errStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       dbUser,
			updateErr:  errors.New("db error"),
		}
		r := setupSLATestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"Updated SLA"}`
		req, _ := http.NewRequest("PUT", "/api/v1/sla/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDeleteSLAPolicy(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:       dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       dbUser,
			deleteErr:  errors.New("db error"),
		}
		r := setupSLATestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherOrgID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: otherOrgID, Role: models.UserRoleAdmin}
		wrongStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       otherUser,
		}
		wrongSession := &auth.SessionUser{ID: otherUserID, CurrentOrgID: otherOrgID}
		r := setupSLATestRouter(wrongStore, wrongSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/sla/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestGetSLAStatus(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy", TargetRPOHours: 24, TargetRTOHours: 4, TargetSuccessRate: 99.5}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID:  map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:        dbUser,
		successRate: 99.8,
		maxRPOHours: 2.5,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var status models.SLAStatus
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if !status.Compliant {
			t.Fatal("expected compliant status")
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+uuid.New().String()+"/status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/bad-uuid/status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockSLAStore{
			policyByID:  map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:        otherUser,
			successRate: 99.8,
			maxRPOHours: 2.5,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupSLATestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestGetSLAHistory(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	policy := &models.SLAPolicy{ID: policyID, OrgID: orgID, Name: "test-policy"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockSLAStore{
		policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
		user:       dbUser,
		history:    []*models.SLAStatusSnapshot{},
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["history"]; !ok {
			t.Fatal("expected 'history' key")
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+uuid.New().String()+"/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSLATestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       dbUser,
			historyErr: errors.New("db error"),
		}
		r := setupSLATestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupSLATestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/bad-uuid/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockSLAStore{
			policyByID: map[uuid.UUID]*models.SLAPolicy{policyID: policy},
			user:       otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupSLATestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/sla/policies/"+policyID.String()+"/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}
