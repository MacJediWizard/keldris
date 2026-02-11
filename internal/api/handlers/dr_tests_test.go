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

// mockDRTestStore implements DRTestStore for testing.
type mockDRTestStore struct {
	testsByOrg     []*models.DRTest
	testsByRunbook []*models.DRTest
	testByID       map[uuid.UUID]*models.DRTest
	runbookByID    map[uuid.UUID]*models.DRRunbook
	user           *models.User
	userErr        error
	listErr        error
	listByRunErr   error
	createErr      error
	updateErr      error
}

func (m *mockDRTestStore) GetDRTestsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.DRTest, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.testsByOrg, nil
}

func (m *mockDRTestStore) GetDRTestsByRunbookID(_ context.Context, runbookID uuid.UUID) ([]*models.DRTest, error) {
	if m.listByRunErr != nil {
		return nil, m.listByRunErr
	}
	return m.testsByRunbook, nil
}

func (m *mockDRTestStore) GetDRTestByID(_ context.Context, id uuid.UUID) (*models.DRTest, error) {
	if t, ok := m.testByID[id]; ok {
		return t, nil
	}
	return nil, errors.New("test not found")
}

func (m *mockDRTestStore) CreateDRTest(_ context.Context, test *models.DRTest) error {
	return m.createErr
}

func (m *mockDRTestStore) UpdateDRTest(_ context.Context, test *models.DRTest) error {
	return m.updateErr
}

func (m *mockDRTestStore) GetDRRunbookByID(_ context.Context, id uuid.UUID) (*models.DRRunbook, error) {
	if rb, ok := m.runbookByID[id]; ok {
		return rb, nil
	}
	return nil, errors.New("runbook not found")
}

func (m *mockDRTestStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	if m.user != nil {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

// mockDRTestRunner implements DRTestRunner for testing.
type mockDRTestRunner struct {
	err error
}

func (m *mockDRTestRunner) TriggerDRTest(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func setupDRTestsTestRouter(store DRTestStore, runner DRTestRunner, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDRTestsHandler(store, runner, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDRTestsList(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	runbookID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	runbook := &models.DRRunbook{ID: runbookID, OrgID: orgID, Name: "Test Runbook"}

	test1 := models.NewDRTest(runbookID)
	test2 := models.NewDRTest(runbookID)
	test2.Start()
	completedTest := models.NewDRTest(runbookID)
	completedTest.Complete("snap-1", 1024, 60, true)

	store := &mockDRTestStore{
		testsByOrg:     []*models.DRTest{test1, test2, completedTest},
		testsByRunbook: []*models.DRTest{test1, test2},
		testByID:       map[uuid.UUID]*models.DRTest{test1.ID: test1, test2.ID: test2, completedTest.ID: completedTest},
		runbookByID:    map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
		user:           dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success all", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["tests"]; !ok {
			t.Fatal("expected 'tests' key in response")
		}
	})

	t.Run("with runbook_id filter", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?runbook_id="+runbookID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid runbook_id", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?runbook_id=not-a-uuid")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("runbook not found for filter", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?runbook_id="+uuid.New().String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("runbook wrong org for filter", func(t *testing.T) {
		otherOrgRunbook := &models.DRRunbook{ID: uuid.New(), OrgID: uuid.New(), Name: "Other"}
		wrongStore := &mockDRTestStore{
			testsByOrg:  []*models.DRTest{},
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{otherOrgRunbook.ID: otherOrgRunbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(wrongStore, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?runbook_id="+otherOrgRunbook.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("with status filter", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?status=completed")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string][]*models.DRTest
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		tests := resp["tests"]
		for _, test := range tests {
			if test.Status != models.DRTestStatusCompleted {
				t.Fatalf("expected all tests to have completed status, got %q", test.Status)
			}
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		errStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			user:        dbUser,
			listErr:     errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("list by runbook error", func(t *testing.T) {
		errStore := &mockDRTestStore{
			testByID:     map[uuid.UUID]*models.DRTest{},
			runbookByID:  map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
			user:         dbUser,
			listByRunErr: errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests?runbook_id="+runbookID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRTestsGet(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	runbookID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	runbook := &models.DRRunbook{ID: runbookID, OrgID: orgID, Name: "Test Runbook"}
	test := models.NewDRTest(runbookID)

	store := &mockDRTestStore{
		testByID:    map[uuid.UUID]*models.DRTest{test.ID: test},
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests/"+test.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests/not-a-uuid")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests/"+uuid.New().String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org via runbook", func(t *testing.T) {
		otherOrgRunbook := &models.DRRunbook{ID: runbookID, OrgID: uuid.New(), Name: "Other"}
		wrongStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{test.ID: test},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: otherOrgRunbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(wrongStore, nil, user)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests/"+test.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, nil)
		req := AuthenticatedRequest("GET", "/api/v1/dr-tests/"+test.ID.String())
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRTestsRun(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	runbookID := uuid.New()
	scheduleID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	runbook := &models.DRRunbook{ID: runbookID, OrgID: orgID, Name: "Run Target", ScheduleID: &scheduleID}
	runbookNoSchedule := &models.DRRunbook{ID: uuid.New(), OrgID: orgID, Name: "No Schedule"}

	store := &mockDRTestStore{
		testByID:    map[uuid.UUID]*models.DRTest{},
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook, runbookNoSchedule.ID: runbookNoSchedule},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookID.String()+`","notes":"Manual test run"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRTest
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.RunbookID != runbookID {
			t.Fatalf("expected runbook_id %s, got %s", runbookID, resp.RunbookID)
		}
		if resp.ScheduleID == nil || *resp.ScheduleID != scheduleID {
			t.Fatal("expected schedule_id to be set from runbook")
		}
	})

	t.Run("success without schedule", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookNoSchedule.ID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRTest
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.ScheduleID != nil {
			t.Fatal("expected schedule_id to be nil")
		}
	})

	t.Run("success with runner", func(t *testing.T) {
		runner := &mockDRTestRunner{}
		r := setupDRTestsTestRouter(store, runner, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"notes":"no runbook_id"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("user error", func(t *testing.T) {
		errStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{},
			userErr:     errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("runbook not found", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+uuid.New().String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("runbook wrong org", func(t *testing.T) {
		otherOrgRunbook := &models.DRRunbook{ID: uuid.New(), OrgID: uuid.New(), Name: "Other Org"}
		wrongStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{otherOrgRunbook.ID: otherOrgRunbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(wrongStore, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+otherOrgRunbook.ID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		errStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
			user:        dbUser,
			createErr:   errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, nil)
		req := JSONRequest("POST", "/api/v1/dr-tests", `{"runbook_id":"`+runbookID.String()+`"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestDRTestsCancel(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	runbookID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	runbook := &models.DRRunbook{ID: runbookID, OrgID: orgID, Name: "Cancel Target"}

	scheduledTest := models.NewDRTest(runbookID) // status = scheduled
	runningTest := models.NewDRTest(runbookID)
	runningTest.Start() // status = running
	completedTest := models.NewDRTest(runbookID)
	completedTest.Complete("snap-1", 1024, 60, true) // status = completed

	store := &mockDRTestStore{
		testByID: map[uuid.UUID]*models.DRTest{
			scheduledTest.ID: scheduledTest,
			runningTest.ID:   runningTest,
			completedTest.ID: completedTest,
		},
		runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
		user:        dbUser,
	}

	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success cancel scheduled", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests/"+scheduledTest.ID.String()+"/cancel", `{"notes":"No longer needed"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DRTest
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != models.DRTestStatusCanceled {
			t.Fatalf("expected status 'canceled', got %q", resp.Status)
		}
	})

	t.Run("success cancel running", func(t *testing.T) {
		// Reset running test status since previous test may have mutated it
		freshRunning := models.NewDRTest(runbookID)
		freshRunning.Start()
		freshStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{freshRunning.ID: freshRunning},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(freshStore, nil, user)
		req := JSONRequest("POST", "/api/v1/dr-tests/"+freshRunning.ID.String()+"/cancel", `{}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("success cancel with empty body", func(t *testing.T) {
		freshScheduled := models.NewDRTest(runbookID)
		freshStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{freshScheduled.ID: freshScheduled},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(freshStore, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+freshScheduled.ID.String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/not-a-uuid/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+uuid.New().String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrgRunbook := &models.DRRunbook{ID: runbookID, OrgID: uuid.New(), Name: "Other Org"}
		wrongStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{scheduledTest.ID: scheduledTest},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: otherOrgRunbook},
			user:        dbUser,
		}
		r := setupDRTestsTestRouter(wrongStore, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+scheduledTest.ID.String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("already completed", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+completedTest.ID.String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		freshTest := models.NewDRTest(runbookID)
		errStore := &mockDRTestStore{
			testByID:    map[uuid.UUID]*models.DRTest{freshTest.ID: freshTest},
			runbookByID: map[uuid.UUID]*models.DRRunbook{runbookID: runbook},
			user:        dbUser,
			updateErr:   errors.New("db error"),
		}
		r := setupDRTestsTestRouter(errStore, nil, user)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+freshTest.ID.String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupDRTestsTestRouter(store, nil, nil)
		req := AuthenticatedRequest("POST", "/api/v1/dr-tests/"+scheduledTest.ID.String()+"/cancel")
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}
