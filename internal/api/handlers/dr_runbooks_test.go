package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockDRRunbookStore implements DRRunbookStore for testing.
type mockDRRunbookStore struct {
	runbooks       []*models.DRRunbook
	runbookByID    map[uuid.UUID]*models.DRRunbook
	user           *models.User
	userErr        error
	schedule       *models.Schedule
	scheduleErr    error
	agent          *models.Agent
	agentErr       error
	repository     *models.Repository
	repoErr        error
	lastTest       *models.DRTest
	lastTestErr    error
	testSchedules  []*models.DRTestSchedule
	testSchedErr   error
	drStatus       *models.DRStatus
	drStatusErr    error
	createErr      error
	updateErr      error
	deleteErr      error
	listErr        error
	createSchedErr error
}

func (m *mockDRRunbookStore) GetDRRunbooksByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.DRRunbook, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.runbooks, nil
}

func (m *mockDRRunbookStore) GetDRRunbookByID(_ context.Context, id uuid.UUID) (*models.DRRunbook, error) {
	if rb, ok := m.runbookByID[id]; ok {
		return rb, nil
	}
	return nil, errors.New("runbook not found")
}

func (m *mockDRRunbookStore) CreateDRRunbook(_ context.Context, runbook *models.DRRunbook) error {
	return m.createErr
}

func (m *mockDRRunbookStore) UpdateDRRunbook(_ context.Context, runbook *models.DRRunbook) error {
	return m.updateErr
}

func (m *mockDRRunbookStore) DeleteDRRunbook(_ context.Context, id uuid.UUID) error {
	return m.deleteErr
}

func (m *mockDRRunbookStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	if m.user != nil {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockDRRunbookStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	if m.scheduleErr != nil {
		return nil, m.scheduleErr
	}
	if m.schedule != nil {
		return m.schedule, nil
	}
	return nil, errors.New("schedule not found")
}

func (m *mockDRRunbookStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	if m.agent != nil {
		return m.agent, nil
	}
	return nil, errors.New("agent not found")
}

func (m *mockDRRunbookStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.repoErr != nil {
		return nil, m.repoErr
	}
	if m.repository != nil {
		return m.repository, nil
	}
	return nil, errors.New("repository not found")
}

func (m *mockDRRunbookStore) GetLatestDRTestByRunbookID(_ context.Context, runbookID uuid.UUID) (*models.DRTest, error) {
	if m.lastTestErr != nil {
		return nil, m.lastTestErr
	}
	return m.lastTest, nil
}

func (m *mockDRRunbookStore) GetDRTestSchedulesByRunbookID(_ context.Context, runbookID uuid.UUID) ([]*models.DRTestSchedule, error) {
	if m.testSchedErr != nil {
		return nil, m.testSchedErr
	}
	return m.testSchedules, nil
}

func (m *mockDRRunbookStore) CreateDRTestSchedule(_ context.Context, schedule *models.DRTestSchedule) error {
	return m.createSchedErr
}

func (m *mockDRRunbookStore) UpdateDRTestSchedule(_ context.Context, schedule *models.DRTestSchedule) error {
	return nil
}

func (m *mockDRRunbookStore) DeleteDRTestSchedule(_ context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockDRRunbookStore) GetDRStatus(_ context.Context, orgID uuid.UUID) (*models.DRStatus, error) {
	if m.drStatusErr != nil {
		return nil, m.drStatusErr
	}
	if m.drStatus != nil {
		return m.drStatus, nil
	}
	return nil, errors.New("no status")
}

func setupDRRunbookTestRouter(store DRRunbookStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDRRunbooksHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDRRunbooksList(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Test Runbook")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbooks:    []*models.DRRunbook{runbook},
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["runbooks"]; !ok {
			t.Fatal("expected 'runbooks' key in response")
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			listErr:     errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksGet(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Test Runbook")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/not-a-uuid")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+uuid.New().String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrgID := uuid.New()
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: otherOrgID, Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: otherOrgID}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksCreate(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	agentID := uuid.New()
	scheduleID := uuid.New()
	repoID := uuid.New()

	schedule := &models.Schedule{
		ID:      scheduleID,
		AgentID: agentID,
		Name:    "Test Schedule",
		Paths:   []string{"/data"},
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), ScheduleID: scheduleID, RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{},
		user:        dbUser,
		schedule:    schedule,
		agent:       agent,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"My Runbook","description":"A test runbook"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("success with schedule_id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Scheduled Runbook","schedule_id":"`+scheduleID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"description":"no name"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("schedule not found", func(t *testing.T) {
		noSchedStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			scheduleErr: errors.New("not found"),
		}
		r := setupDRRunbookTestRouter(noSchedStore, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Test","schedule_id":"`+uuid.New().String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other-host"}
		wrongAgentStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			schedule:    schedule,
			agent:       wrongAgent,
		}
		r := setupDRRunbookTestRouter(wrongAgentStore, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Test","schedule_id":"`+scheduleID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			createErr:   errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := JSONRequest("POST", "/api/v1/dr-runbooks", `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksUpdate(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Original Name")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/"+runbook.ID.String(), `{"name":"Updated Name"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/not-a-uuid", `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/"+uuid.New().String(), `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/"+runbook.ID.String(), `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        dbUser,
			updateErr:   errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/"+runbook.ID.String(), `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := JSONRequest("PUT", "/api/v1/dr-runbooks/"+runbook.ID.String(), `{"name":"Test"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksDelete(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Delete Me")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/not-a-uuid")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/"+uuid.New().String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        dbUser,
			deleteErr:   errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("DELETE", "/api/v1/dr-runbooks/"+runbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksActivate(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Activate Me")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRRunbook
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != models.DRRunbookStatusActive {
			t.Fatalf("expected status 'active', got %q", resp.Status)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/not-a-uuid/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+uuid.New().String()+"/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        dbUser,
			updateErr:   errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/activate")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksArchive(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Archive Me")
	runbook.Activate()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/archive")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRRunbook
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != models.DRRunbookStatusArchived {
			t.Fatalf("expected status 'archived', got %q", resp.Status)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/not-a-uuid/archive")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+uuid.New().String()+"/archive")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/archive")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/archive")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksRender(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Render Me")
	runbook.Description = "A test runbook"
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
		lastTestErr: errors.New("no test"), // no last test, generator handles gracefully
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/render")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["format"] != "markdown" {
			t.Fatalf("expected format 'markdown', got %q", resp["format"])
		}
		if resp["content"] == "" {
			t.Fatal("expected non-empty content")
		}
	})

	t.Run("success with schedule data", func(t *testing.T) {
		scheduleID := uuid.New()
		agentID := uuid.New()
		repoID := uuid.New()

		schedRunbook := models.NewDRRunbook(orgID, "Scheduled Render")
		schedRunbook.ScheduleID = &scheduleID

		schedStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{schedRunbook.ID: schedRunbook},
			user:        dbUser,
			schedule: &models.Schedule{
				ID:      scheduleID,
				AgentID: agentID,
				Name:    "Daily Backup",
				Paths:   []string{"/data"},
				Repositories: []models.ScheduleRepository{
					{RepositoryID: repoID, Priority: 0, Enabled: true},
				},
			},
			agent:      &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"},
			repository: &models.Repository{ID: repoID, OrgID: orgID, Name: "My Repo", Type: models.RepositoryTypeS3},
			lastTestErr: errors.New("no test"),
		}
		r := setupDRRunbookTestRouter(schedStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+schedRunbook.ID.String()+"/render")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+uuid.New().String()+"/render")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/render")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/render")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksGenerateFromSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	agentID := uuid.New()
	scheduleID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	schedule := &models.Schedule{
		ID:      scheduleID,
		AgentID: agentID,
		Name:    "Test Schedule",
		Paths:   []string{"/data", "/config"},
		Repositories: []models.ScheduleRepository{
			{ID: uuid.New(), ScheduleID: scheduleID, RepositoryID: repoID, Priority: 0, Enabled: true},
		},
	}
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "gen-host"}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "Gen Repo", Type: models.RepositoryTypeS3}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{},
		user:        dbUser,
		schedule:    schedule,
		agent:       agent,
		repository:  repo,
		lastTestErr: errors.New("no test"),
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+scheduleID.String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/not-a-uuid/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+scheduleID.String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("schedule not found", func(t *testing.T) {
		noSchedStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			scheduleErr: errors.New("not found"),
		}
		r := setupDRRunbookTestRouter(noSchedStore, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+uuid.New().String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		wrongAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "wrong-host"}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			schedule:    schedule,
			agent:       wrongAgent,
		}
		r := setupDRRunbookTestRouter(wrongStore, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+scheduleID.String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			schedule:    schedule,
			agent:       agent,
			repository:  repo,
			lastTestErr: errors.New("no test"),
			createErr:   errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+scheduleID.String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("POST", "/api/v1/dr-runbooks/"+scheduleID.String()+"/generate")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksGetStatus(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	drStatus := &models.DRStatus{
		TotalRunbooks:  5,
		ActiveRunbooks: 3,
	}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{},
		user:        dbUser,
		drStatus:    drStatus,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/status")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRStatus
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.TotalRunbooks != 5 {
			t.Fatalf("expected 5 total runbooks, got %d", resp.TotalRunbooks)
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/status")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("status error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			drStatusErr: errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/status")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/status")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksListTestSchedules(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "With Schedules")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	testSchedule := models.NewDRTestSchedule(runbook.ID, "0 0 * * *")

	store := &mockDRRunbookStore{
		runbookByID:   map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:          dbUser,
		testSchedules: []*models.DRTestSchedule{testSchedule},
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["schedules"]; !ok {
			t.Fatal("expected 'schedules' key in response")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/not-a-uuid/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+uuid.New().String()+"/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID:  map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:         dbUser,
			testSchedErr: errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRRunbooksCreateTestSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	runbook := models.NewDRRunbook(orgID, "Schedule Target")
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockDRRunbookStore{
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("success with enabled false", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{"cron_expression":"0 0 * * *","enabled":false}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRTestSchedule
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Enabled {
			t.Fatal("expected enabled to be false")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/not-a-uuid/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+uuid.New().String()+"/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherDBUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockDRRunbookStore{
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:        otherDBUser,
		}
		wrongUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupDRRunbookTestRouter(wrongStore, wrongUser)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		errStore := &mockDRRunbookStore{
			runbookByID:    map[uuid.UUID]*models.DRRunbook{runbook.ID: runbook},
			user:           dbUser,
			createSchedErr: errors.New("db error"),
		}
		r := setupDRRunbookTestRouter(errStore, user)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRRunbookTestRouter(store, nil)
		req := JSONRequest("POST", "/api/v1/dr-runbooks/"+runbook.ID.String()+"/test-schedules", `{"cron_expression":"0 0 * * *"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}
