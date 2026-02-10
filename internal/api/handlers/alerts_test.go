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

type mockAlertStore struct {
	alerts      []*models.Alert
	alertByID   map[uuid.UUID]*models.Alert
	activeCount int
	rules       []*models.AlertRule
	ruleByID    map[uuid.UUID]*models.AlertRule
	user        *models.User
	updateErr   error
	createErr   error
	deleteErr   error
}

func (m *mockAlertStore) GetAlertsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	var result []*models.Alert
	for _, a := range m.alerts {
		if a.OrgID == orgID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAlertStore) GetActiveAlertsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	var result []*models.Alert
	for _, a := range m.alerts {
		if a.OrgID == orgID && a.Status == models.AlertStatusActive {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAlertStore) GetActiveAlertCountByOrgID(_ context.Context, _ uuid.UUID) (int, error) {
	return m.activeCount, nil
}

func (m *mockAlertStore) GetAlertByID(_ context.Context, id uuid.UUID) (*models.Alert, error) {
	if a, ok := m.alertByID[id]; ok {
		return a, nil
	}
	return nil, errors.New("alert not found")
}

func (m *mockAlertStore) UpdateAlert(_ context.Context, _ *models.Alert) error {
	return m.updateErr
}

func (m *mockAlertStore) GetAlertRulesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	var result []*models.AlertRule
	for _, r := range m.rules {
		if r.OrgID == orgID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockAlertStore) GetAlertRuleByID(_ context.Context, id uuid.UUID) (*models.AlertRule, error) {
	if r, ok := m.ruleByID[id]; ok {
		return r, nil
	}
	return nil, errors.New("rule not found")
}

func (m *mockAlertStore) CreateAlertRule(_ context.Context, _ *models.AlertRule) error {
	return m.createErr
}

func (m *mockAlertStore) UpdateAlertRule(_ context.Context, _ *models.AlertRule) error {
	return m.updateErr
}

func (m *mockAlertStore) DeleteAlertRule(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockAlertStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func setupAlertTestRouter(store AlertStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewAlertsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListAlerts(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusActive, Title: "test"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		alerts:    []*models.Alert{alert},
		alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
		user:      dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["alerts"]; !ok {
			t.Fatal("expected 'alerts' key")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAlertTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestListActiveAlerts(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{user: dbUser}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/active", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestAlertCount(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{user: dbUser, activeCount: 3}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/count", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]int
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["count"] != 3 {
			t.Fatalf("expected count 3, got %d", resp["count"])
		}
	})
}

func TestGetAlert(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusActive, Title: "test"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
		user:      dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/"+alertID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
			user:      otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupAlertTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alerts/"+alertID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestAcknowledgeAlert(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	t.Run("success", func(t *testing.T) {
		alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusActive}
		store := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
			user:      dbUser,
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/actions/acknowledge", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("already resolved", func(t *testing.T) {
		alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusResolved}
		store := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
			user:      dbUser,
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/actions/acknowledge", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{},
			user:      dbUser,
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/alerts/"+uuid.New().String()+"/actions/acknowledge", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestResolveAlert(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	t.Run("success", func(t *testing.T) {
		alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusActive}
		store := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
			user:      dbUser,
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/actions/resolve", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("already resolved idempotent", func(t *testing.T) {
		alert := &models.Alert{ID: alertID, OrgID: orgID, Status: models.AlertStatusResolved}
		store := &mockAlertStore{
			alertByID: map[uuid.UUID]*models.Alert{alertID: alert},
			user:      dbUser,
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/actions/resolve", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestListAlertRules(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	rule := &models.AlertRule{ID: ruleID, OrgID: orgID, Name: "test-rule", Type: models.AlertTypeAgentOffline}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		rules:    []*models.AlertRule{rule},
		ruleByID: map[uuid.UUID]*models.AlertRule{ruleID: rule},
		user:     dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alert-rules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestCreateAlertRule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		ruleByID: map[uuid.UUID]*models.AlertRule{},
		user:     dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"offline-rule","type":"agent_offline","enabled":true,"config":{"offline_threshold_minutes":15}}`
		req, _ := http.NewRequest("POST", "/api/v1/alert-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"bad-rule","type":"invalid_type","enabled":true,"config":{}}`
		req, _ := http.NewRequest("POST", "/api/v1/alert-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"type":"agent_offline","config":{}}`
		req, _ := http.NewRequest("POST", "/api/v1/alert-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAlertStore{
			ruleByID:  map[uuid.UUID]*models.AlertRule{},
			user:      dbUser,
			createErr: errors.New("db error"),
		}
		r := setupAlertTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail","type":"agent_offline","config":{"offline_threshold_minutes":15}}`
		req, _ := http.NewRequest("POST", "/api/v1/alert-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetAlertRule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	rule := &models.AlertRule{ID: ruleID, OrgID: orgID, Name: "test-rule", Type: models.AlertTypeAgentOffline}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		ruleByID: map[uuid.UUID]*models.AlertRule{ruleID: rule},
		user:     dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alert-rules/"+ruleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alert-rules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockAlertStore{
			ruleByID: map[uuid.UUID]*models.AlertRule{ruleID: rule},
			user:     otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupAlertTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/alert-rules/"+ruleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestUpdateAlertRule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	rule := &models.AlertRule{ID: ruleID, OrgID: orgID, Name: "test-rule", Type: models.AlertTypeAgentOffline}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		ruleByID: map[uuid.UUID]*models.AlertRule{ruleID: rule},
		user:     dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated-rule"}`
		req, _ := http.NewRequest("PUT", "/api/v1/alert-rules/"+ruleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/alert-rules/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDeleteAlertRule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ruleID := uuid.New()

	rule := &models.AlertRule{ID: ruleID, OrgID: orgID, Name: "test-rule"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAlertStore{
		ruleByID: map[uuid.UUID]*models.AlertRule{ruleID: rule},
		user:     dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/alert-rules/"+ruleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAlertTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/alert-rules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAlertStore{
			ruleByID:  map[uuid.UUID]*models.AlertRule{ruleID: rule},
			user:      dbUser,
			deleteErr: errors.New("db error"),
		}
		r := setupAlertTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/alert-rules/"+ruleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}
