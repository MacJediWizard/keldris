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

// mockScheduleStore implements ScheduleStore for testing.
type mockScheduleStore struct {
	schedulesByAgent    map[uuid.UUID][]*models.Schedule
	scheduleByID        map[uuid.UUID]*models.Schedule
	agentByID           map[uuid.UUID]*models.Agent
	agentsByOrg         map[uuid.UUID][]*models.Agent
	repositoryByID      map[uuid.UUID]*models.Repository
	replicationStatuses map[uuid.UUID][]*models.ReplicationStatus
	createErr           error
	updateErr           error
	deleteErr           error
	setReposErr         error
	listAgentsErr       error
	replStatusErr       error
}

func newMockScheduleStore() *mockScheduleStore {
	return &mockScheduleStore{
		schedulesByAgent:    make(map[uuid.UUID][]*models.Schedule),
		scheduleByID:        make(map[uuid.UUID]*models.Schedule),
		agentByID:           make(map[uuid.UUID]*models.Agent),
		agentsByOrg:         make(map[uuid.UUID][]*models.Agent),
		repositoryByID:      make(map[uuid.UUID]*models.Repository),
		replicationStatuses: make(map[uuid.UUID][]*models.ReplicationStatus),
	}
}

func (m *mockScheduleStore) GetSchedulesByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Schedule, error) {
	return m.schedulesByAgent[agentID], nil
}

func (m *mockScheduleStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	if s, ok := m.scheduleByID[id]; ok {
		return s, nil
	}
	return nil, errors.New("schedule not found")
}

func (m *mockScheduleStore) CreateSchedule(_ context.Context, schedule *models.Schedule) error {
	return m.createErr
}

func (m *mockScheduleStore) UpdateSchedule(_ context.Context, schedule *models.Schedule) error {
	return m.updateErr
}

func (m *mockScheduleStore) DeleteSchedule(_ context.Context, id uuid.UUID) error {
	return m.deleteErr
}

func (m *mockScheduleStore) SetScheduleRepositories(_ context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error {
	return m.setReposErr
}

func (m *mockScheduleStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if a, ok := m.agentByID[id]; ok {
		return a, nil
	}
	return nil, errors.New("agent not found")
}

func (m *mockScheduleStore) GetAgentsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	if m.listAgentsErr != nil {
		return nil, m.listAgentsErr
	}
	return m.agentsByOrg[orgID], nil
}

func (m *mockScheduleStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if r, ok := m.repositoryByID[id]; ok {
		return r, nil
	}
	return nil, errors.New("repository not found")
}

func (m *mockScheduleStore) GetReplicationStatusBySchedule(_ context.Context, scheduleID uuid.UUID) ([]*models.ReplicationStatus, error) {
	if m.replStatusErr != nil {
		return nil, m.replStatusErr
	}
	return m.replicationStatuses[scheduleID], nil
}

func setupScheduleTestRouter(store ScheduleStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})

	handler := NewSchedulesHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestCreateSchedule(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	repoID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo", Type: models.RepositoryTypeLocal}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.repositoryByID[repoID] = repo

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + agentID.String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.Schedule
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Name != "daily-backup" {
			t.Fatalf("expected name 'daily-backup', got %q", resp.Name)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test"}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + uuid.New().String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: uuid.New(), OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[otherAgent.ID] = otherAgent
		store2.repositoryByID[repoID] = repo

		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + otherAgent.ID.String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + agentID.String() + `",
			"repositories": [{"repository_id": "` + uuid.New().String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockScheduleStore()
		errStore.agentByID[agentID] = agent
		errStore.repositoryByID[repoID] = repo
		errStore.createErr = errors.New("db error")

		r := setupScheduleTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + agentID.String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + agentID.String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{
			"agent_id": "` + agentID.String() + `",
			"repositories": [{"repository_id": "` + repoID.String() + `", "priority": 0, "enabled": true}],
			"name": "daily-backup",
			"cron_expression": "0 2 * * *",
			"paths": ["/data"]
		}`
		req, _ := http.NewRequest("POST", "/api/v1/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestUpdateSchedule(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "old-name", CronExpression: "0 2 * * *", Paths: []string{"/data"}}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.scheduleByID[scheduleID] = schedule

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[agentID] = otherAgent
		store2.scheduleByID[scheduleID] = schedule

		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockScheduleStore()
		errStore.agentByID[agentID] = agent
		errStore.scheduleByID[scheduleID] = schedule
		errStore.updateErr = errors.New("db error")

		r := setupScheduleTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("update with repositories", func(t *testing.T) {
		repoID := uuid.New()
		repoStore := newMockScheduleStore()
		repoStore.agentByID[agentID] = agent
		repoStore.scheduleByID[scheduleID] = schedule
		repoStore.repositoryByID[repoID] = &models.Repository{ID: repoID, OrgID: orgID, Name: "repo", Type: models.RepositoryTypeLocal}
		r := setupScheduleTestRouter(repoStore, user)
		w := httptest.NewRecorder()
		body := `{"repositories":[{"repository_id":"` + repoID.String() + `","priority":0,"enabled":true}]}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update with repos - repo not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"repositories":[{"repository_id":"` + uuid.New().String() + `","priority":0,"enabled":true}]}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("set repos error", func(t *testing.T) {
		repoID := uuid.New()
		repoErrStore := newMockScheduleStore()
		repoErrStore.agentByID[agentID] = agent
		repoErrStore.scheduleByID[scheduleID] = schedule
		repoErrStore.repositoryByID[repoID] = &models.Repository{ID: repoID, OrgID: orgID, Name: "repo", Type: models.RepositoryTypeLocal}
		repoErrStore.setReposErr = errors.New("db error")
		r := setupScheduleTestRouter(repoErrStore, user)
		w := httptest.NewRecorder()
		body := `{"repositories":[{"repository_id":"` + repoID.String() + `","priority":0,"enabled":true}]}`
		req, _ := http.NewRequest("PUT", "/api/v1/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestListSchedules(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.agentsByOrg[orgID] = []*models.Agent{agent}
	store.schedulesByAgent[agentID] = []*models.Schedule{schedule}
	store.scheduleByID[scheduleID] = schedule

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("list all", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["schedules"]; !ok {
			t.Fatal("expected 'schedules' key")
		}
	})

	t.Run("filter by agent_id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules?agent_id=bad", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("agent not found filter", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules?agent_id="+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("agent wrong org filter", func(t *testing.T) {
		otherAgent := &models.Agent{ID: uuid.New(), OrgID: uuid.New(), Hostname: "other"}
		filterStore := newMockScheduleStore()
		filterStore.agentByID[otherAgent.ID] = otherAgent
		filterStore.agentsByOrg[orgID] = []*models.Agent{agent}
		r := setupScheduleTestRouter(filterStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules?agent_id="+otherAgent.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store list agents error", func(t *testing.T) {
		errStore := newMockScheduleStore()
		errStore.listAgentsErr = errors.New("db error")
		r := setupScheduleTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDeleteSchedule(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.scheduleByID[scheduleID] = schedule

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockScheduleStore()
		errStore.agentByID[agentID] = agent
		errStore.scheduleByID[scheduleID] = schedule
		errStore.deleteErr = errors.New("db error")

		r := setupScheduleTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[agentID] = otherAgent
		store2.scheduleByID[scheduleID] = schedule
		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestRunSchedule(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.scheduleByID[scheduleID] = schedule

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/run", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/run", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/run", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/bad-uuid/run", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/"+uuid.New().String()+"/run", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[agentID] = otherAgent
		store2.scheduleByID[scheduleID] = schedule
		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/run", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestGetReplicationStatus(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.scheduleByID[scheduleID] = schedule
	store.replicationStatuses[scheduleID] = []*models.ReplicationStatus{}

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/replication", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/bad-uuid/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+uuid.New().String()+"/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[agentID] = otherAgent
		store2.scheduleByID[scheduleID] = schedule
		store2.replicationStatuses[scheduleID] = []*models.ReplicationStatus{}
		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockScheduleStore()
		errStore.agentByID[agentID] = agent
		errStore.scheduleByID[scheduleID] = schedule
		errStore.replStatusErr = errors.New("db error")
		r := setupScheduleTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/replication", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetSchedule(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}

	store := newMockScheduleStore()
	store.agentByID[agentID] = agent
	store.scheduleByID[scheduleID] = schedule

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupScheduleTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockScheduleStore()
		store2.agentByID[agentID] = otherAgent
		store2.scheduleByID[scheduleID] = schedule

		r := setupScheduleTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupScheduleTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupScheduleTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}
