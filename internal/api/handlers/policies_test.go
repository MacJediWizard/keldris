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

type mockPolicyStore struct {
	policies     []*models.Policy
	policyByID   map[uuid.UUID]*models.Policy
	schedules    []*models.Schedule
	agents       []*models.Agent
	agentByID    map[uuid.UUID]*models.Agent
	repo         *models.Repository
	createErr    error
	updateErr    error
	deleteErr    error
	createSchedErr error
	listErr      error
}

func (m *mockPolicyStore) GetPoliciesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Policy, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
}

func (m *mockPolicyStore) GetPoliciesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Policy, error) {
	var result []*models.Policy
	for _, p := range m.policies {
		if p.OrgID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPolicyStore) GetPolicyByID(_ context.Context, id uuid.UUID) (*models.Policy, error) {
	if p, ok := m.policyByID[id]; ok {
		return p, nil
	}
	return nil, errors.New("policy not found")
}

func (m *mockPolicyStore) CreatePolicy(_ context.Context, _ *models.Policy) error {
	return m.createErr
}

func (m *mockPolicyStore) UpdatePolicy(_ context.Context, _ *models.Policy) error {
	return m.updateErr
}

func (m *mockPolicyStore) DeletePolicy(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockPolicyStore) GetSchedulesByPolicyID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, nil
}

func (m *mockPolicyStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if a, ok := m.agentByID[id]; ok {
		return a, nil
	}
	return nil, errors.New("agent not found")
}

func (m *mockPolicyStore) GetAgentsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	var result []*models.Agent
	for _, a := range m.agents {
		if a.OrgID == orgID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockPolicyStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	return nil, errors.New("repository not found")
}

func (m *mockPolicyStore) CreateSchedule(_ context.Context, _ *models.Schedule) error {
	return m.createSchedErr
}

func setupPolicyTestRouter(store PolicyStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewPoliciesHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListPolicies(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "default"}
	store := &mockPolicyStore{
		policies:   []*models.Policy{policy},
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["policies"]; !ok {
			t.Fatal("expected 'policies' key")
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockPolicyStore{
			policies:   []*models.Policy{policy},
			policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
			listErr:    errors.New("db error"),
		}
		r := setupPolicyTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetPolicy(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "default"}
	store := &mockPolicyStore{
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupPolicyTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestCreatePolicy(t *testing.T) {
	orgID := uuid.New()
	store := &mockPolicyStore{policyByID: map[uuid.UUID]*models.Policy{}}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"daily-backup","cron_expression":"0 2 * * *","paths":["/home"]}`
		req, _ := http.NewRequest("POST", "/api/v1/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockPolicyStore{policyByID: map[uuid.UUID]*models.Policy{}, createErr: errors.New("db error")}
		r := setupPolicyTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"fail"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"name":"fail"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestUpdatePolicy(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "default"}
	store := &mockPolicyStore{
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupPolicyTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"x"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		errStore := &mockPolicyStore{
			policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
			updateErr:  errors.New("db error"),
		}
		r := setupPolicyTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"name":"x"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"x"}`
		req, _ := http.NewRequest("PUT", "/api/v1/policies/"+policyID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestDeletePolicy(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "default"}
	store := &mockPolicyStore{
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockPolicyStore{
			policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
			deleteErr:  errors.New("db error"),
		}
		r := setupPolicyTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupPolicyTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+policyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestPolicyListSchedules(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "default"}
	store := &mockPolicyStore{
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
		schedules:  []*models.Schedule{},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String()+"/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("policy not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+uuid.New().String()+"/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/bad-uuid/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupPolicyTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String()+"/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String()+"/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String()+"/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestApplyPolicy(t *testing.T) {
	orgID := uuid.New()
	policyID := uuid.New()
	agentID := uuid.New()
	repoID := uuid.New()

	policy := &models.Policy{ID: policyID, OrgID: orgID, Name: "daily", CronExpression: "0 2 * * *", Paths: []string{"/data"}}
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "server-1"}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup", Type: models.RepositoryTypeS3}

	store := &mockPolicyStore{
		policyByID: map[uuid.UUID]*models.Policy{policyID: policy},
		agentByID:  map[uuid.UUID]*models.Agent{agentID: agent},
		agents:     []*models.Agent{agent},
		repo:       repo,
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp ApplyPolicyResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.SchedulesCreated != 1 {
			t.Fatalf("expected 1 schedule created, got %d", resp.SchedulesCreated)
		}
	})

	t.Run("missing body", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("policy not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+uuid.New().String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + uuid.New().String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org policy", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupPolicyTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/bad-uuid/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupPolicyTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupPolicyTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("with schedule name", func(t *testing.T) {
		r := setupPolicyTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_ids":["` + agentID.String() + `"],"repository_id":"` + repoID.String() + `","schedule_name":"custom-schedule"}`
		req, _ := http.NewRequest("POST", "/api/v1/policies/"+policyID.String()+"/apply", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp ApplyPolicyResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.SchedulesCreated != 1 {
			t.Fatalf("expected 1 schedule created, got %d", resp.SchedulesCreated)
		}
	})
}
