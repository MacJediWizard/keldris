package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockVerificationStore implements VerificationStore for tests.
type mockVerificationStore struct {
	user              *models.User
	repo              *models.Repository
	verifications     []*models.Verification
	verification      *models.Verification
	latestVer         *models.Verification
	consecutiveFails  int
	schedules         []*models.VerificationSchedule
	schedule          *models.VerificationSchedule
	getUserErr        error
	getRepoErr        error
	getVerErr         error
	listVerErr        error
	getLatestErr      error
	getConsecErr      error
	listScheduleErr   error
	getScheduleErr    error
	createScheduleErr error
	updateScheduleErr error
	deleteScheduleErr error
}

func (m *mockVerificationStore) GetVerificationsByRepoID(_ context.Context, _ uuid.UUID) ([]*models.Verification, error) {
	if m.listVerErr != nil {
		return nil, m.listVerErr
	}
	return m.verifications, nil
}

func (m *mockVerificationStore) GetVerificationByID(_ context.Context, id uuid.UUID) (*models.Verification, error) {
	if m.getVerErr != nil {
		return nil, m.getVerErr
	}
	if m.verification != nil && m.verification.ID == id {
		return m.verification, nil
	}
	return nil, errors.New("not found")
}

func (m *mockVerificationStore) GetLatestVerificationByRepoID(_ context.Context, _ uuid.UUID) (*models.Verification, error) {
	if m.getLatestErr != nil {
		return nil, m.getLatestErr
	}
	return m.latestVer, nil
}

func (m *mockVerificationStore) GetConsecutiveFailedVerifications(_ context.Context, _ uuid.UUID) (int, error) {
	if m.getConsecErr != nil {
		return 0, m.getConsecErr
	}
	return m.consecutiveFails, nil
}

func (m *mockVerificationStore) GetVerificationSchedulesByRepoID(_ context.Context, _ uuid.UUID) ([]*models.VerificationSchedule, error) {
	if m.listScheduleErr != nil {
		return nil, m.listScheduleErr
	}
	return m.schedules, nil
}

func (m *mockVerificationStore) GetVerificationScheduleByID(_ context.Context, id uuid.UUID) (*models.VerificationSchedule, error) {
	if m.getScheduleErr != nil {
		return nil, m.getScheduleErr
	}
	if m.schedule != nil && m.schedule.ID == id {
		return m.schedule, nil
	}
	return nil, errors.New("not found")
}

func (m *mockVerificationStore) CreateVerificationSchedule(_ context.Context, _ *models.VerificationSchedule) error {
	return m.createScheduleErr
}

func (m *mockVerificationStore) UpdateVerificationSchedule(_ context.Context, _ *models.VerificationSchedule) error {
	return m.updateScheduleErr
}

func (m *mockVerificationStore) DeleteVerificationSchedule(_ context.Context, _ uuid.UUID) error {
	return m.deleteScheduleErr
}

func (m *mockVerificationStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockVerificationStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

// mockVerificationTrigger implements VerificationTrigger for tests.
type mockVerificationTrigger struct {
	verification *models.Verification
	status       *models.RepositoryVerificationStatus
	triggerErr   error
	statusErr    error
}

func (m *mockVerificationTrigger) TriggerVerification(_ context.Context, _ uuid.UUID, _ models.VerificationType) (*models.Verification, error) {
	if m.triggerErr != nil {
		return nil, m.triggerErr
	}
	return m.verification, nil
}

func (m *mockVerificationTrigger) GetRepositoryVerificationStatus(_ context.Context, _ uuid.UUID) (*models.RepositoryVerificationStatus, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.status, nil
}

func setupVerificationsTestRouter(store VerificationStore, trigger VerificationTrigger, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewVerificationsHandler(store, trigger, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

// newTestVerification creates a Verification with populated fields for testing.
func newTestVerification(repoID uuid.UUID) *models.Verification {
	now := time.Now()
	completed := now.Add(5 * time.Second)
	dur := int64(5000)
	return &models.Verification{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Type:         models.VerificationTypeCheck,
		SnapshotID:   "abc123",
		StartedAt:    now,
		CompletedAt:  &completed,
		Status:       models.VerificationStatusPassed,
		DurationMs:   &dur,
		CreatedAt:    now,
	}
}

// newTestSchedule creates a VerificationSchedule with populated fields for testing.
func newTestSchedule(repoID uuid.UUID) *models.VerificationSchedule {
	now := time.Now()
	return &models.VerificationSchedule{
		ID:             uuid.New(),
		RepositoryID:   repoID,
		Type:           models.VerificationTypeCheck,
		CronExpression: "0 2 * * *",
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// --- List Tests ---

func TestVerificationsList(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockVerificationStore{user: dbUser}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success returns empty list", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["verifications"]; !ok {
			t.Fatal("expected 'verifications' key in response")
		}
		var verifications []VerificationResponse
		if err := json.Unmarshal(resp["verifications"], &verifications); err != nil {
			t.Fatalf("failed to unmarshal verifications: %v", err)
		}
		if len(verifications) != 0 {
			t.Fatalf("expected empty list, got %d items", len(verifications))
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- Get Tests ---

func TestVerificationsGet(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	ver := newTestVerification(repoID)

	store := &mockVerificationStore{
		user:         dbUser,
		repo:         repo,
		verification: ver,
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/"+ver.ID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ID != ver.ID.String() {
			t.Fatalf("expected id %s, got %s", ver.ID.String(), resp.ID)
		}
		if resp.RepositoryID != repoID.String() {
			t.Fatalf("expected repository_id %s, got %s", repoID.String(), resp.RepositoryID)
		}
		if resp.Type != string(models.VerificationTypeCheck) {
			t.Fatalf("expected type 'check', got %s", resp.Type)
		}
		if resp.Status != string(models.VerificationStatusPassed) {
			t.Fatalf("expected status 'passed', got %s", resp.Status)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo access denied user not found", func(t *testing.T) {
		// User exists in session but not found in the store => verifyRepoAccess fails
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/"+ver.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo access denied repo not found", func(t *testing.T) {
		// Verification references a repo that doesn't exist in the store
		otherRepoID := uuid.New()
		orphanVer := newTestVerification(otherRepoID)
		orphanStore := &mockVerificationStore{
			user:         dbUser,
			repo:         repo, // repo has different ID than verification's RepositoryID
			verification: orphanVer,
		}
		r := setupVerificationsTestRouter(orphanStore, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/"+orphanVer.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verifications/"+ver.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- ListByRepository Tests ---

func TestVerificationsListByRepository(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	ver1 := newTestVerification(repoID)
	ver2 := newTestVerification(repoID)

	store := &mockVerificationStore{
		user:          dbUser,
		repo:          repo,
		verifications: []*models.Verification{ver1, ver2},
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verifications", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["verifications"]; !ok {
			t.Fatal("expected 'verifications' key in response")
		}
		var verifications []VerificationResponse
		if err := json.Unmarshal(resp["verifications"], &verifications); err != nil {
			t.Fatalf("failed to unmarshal verifications: %v", err)
		}
		if len(verifications) != 2 {
			t.Fatalf("expected 2 verifications, got %d", len(verifications))
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/bad-uuid/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+uuid.New().String()+"/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		errStore := &mockVerificationStore{
			user:       dbUser,
			repo:       repo,
			listVerErr: errors.New("db error"),
		}
		r := setupVerificationsTestRouter(errStore, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verifications", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- GetStatus Tests ---

func TestVerificationsGetStatus(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}

	lastVer := newTestVerification(repoID)
	nextAt := time.Now().Add(24 * time.Hour)

	status := &models.RepositoryVerificationStatus{
		RepositoryID:     repoID,
		LastVerification: lastVer,
		NextScheduledAt:  &nextAt,
		ConsecutiveFails: 2,
	}

	store := &mockVerificationStore{
		user: dbUser,
		repo: repo,
	}
	trigger := &mockVerificationTrigger{status: status}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-status", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.RepositoryID != repoID.String() {
			t.Fatalf("expected repository_id %s, got %s", repoID.String(), resp.RepositoryID)
		}
		if resp.ConsecutiveFails != 2 {
			t.Fatalf("expected consecutive_fails 2, got %d", resp.ConsecutiveFails)
		}
		if resp.LastVerification == nil {
			t.Fatal("expected last_verification to be set")
		}
		if resp.NextScheduledAt == "" {
			t.Fatal("expected next_scheduled_at to be set")
		}
	})

	t.Run("success without optional fields", func(t *testing.T) {
		minimalStatus := &models.RepositoryVerificationStatus{
			RepositoryID:     repoID,
			ConsecutiveFails: 0,
		}
		minTrigger := &mockVerificationTrigger{status: minimalStatus}
		r := setupVerificationsTestRouter(store, minTrigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-status", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.LastVerification != nil {
			t.Fatal("expected last_verification to be nil")
		}
		if resp.NextScheduledAt != "" {
			t.Fatalf("expected next_scheduled_at to be empty, got %s", resp.NextScheduledAt)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/bad-uuid/verification-status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("status error", func(t *testing.T) {
		errTrigger := &mockVerificationTrigger{statusErr: errors.New("service error")}
		r := setupVerificationsTestRouter(store, errTrigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-status", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- TriggerVerification Tests ---

func TestVerificationsTrigger(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	triggered := newTestVerification(repoID)

	store := &mockVerificationStore{
		user: dbUser,
		repo: repo,
	}
	trigger := &mockVerificationTrigger{verification: triggered}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ID != triggered.ID.String() {
			t.Fatalf("expected id %s, got %s", triggered.ID.String(), resp.ID)
		}
	})

	t.Run("success check_read_data type", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check_read_data"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("success test_restore type", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"test_restore"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/bad-uuid/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body missing type", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body bad type value", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"invalid_type"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body malformed json", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{bad json`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("trigger error", func(t *testing.T) {
		errTrigger := &mockVerificationTrigger{triggerErr: errors.New("trigger failed")}
		r := setupVerificationsTestRouter(store, errTrigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verifications", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- ListSchedules Tests ---

func TestVerificationsListSchedules(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	sched1 := newTestSchedule(repoID)
	sched2 := newTestSchedule(repoID)

	store := &mockVerificationStore{
		user:      dbUser,
		repo:      repo,
		schedules: []*models.VerificationSchedule{sched1, sched2},
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["schedules"]; !ok {
			t.Fatal("expected 'schedules' key in response")
		}
		var schedules []VerificationScheduleResponse
		if err := json.Unmarshal(resp["schedules"], &schedules); err != nil {
			t.Fatalf("failed to unmarshal schedules: %v", err)
		}
		if len(schedules) != 2 {
			t.Fatalf("expected 2 schedules, got %d", len(schedules))
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/bad-uuid/verification-schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		errStore := &mockVerificationStore{
			user:            dbUser,
			repo:            repo,
			listScheduleErr: errors.New("db error"),
		}
		r := setupVerificationsTestRouter(errStore, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- CreateSchedule Tests ---

func TestVerificationsCreateSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}

	store := &mockVerificationStore{
		user: dbUser,
		repo: repo,
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationScheduleResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.RepositoryID != repoID.String() {
			t.Fatalf("expected repository_id %s, got %s", repoID.String(), resp.RepositoryID)
		}
		if resp.Type != "check" {
			t.Fatalf("expected type 'check', got %s", resp.Type)
		}
		if resp.CronExpression != "0 2 * * *" {
			t.Fatalf("expected cron '0 2 * * *', got %s", resp.CronExpression)
		}
		if !resp.Enabled {
			t.Fatal("expected enabled to be true by default")
		}
	})

	t.Run("success with all fields", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check_read_data","cron_expression":"0 3 * * 0","enabled":false,"read_data_subset":"5%"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationScheduleResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.Enabled {
			t.Fatal("expected enabled to be false")
		}
		if resp.ReadDataSubset != "5%" {
			t.Fatalf("expected read_data_subset '5%%', got %s", resp.ReadDataSubset)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/bad-uuid/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body missing type", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body missing cron", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body bad type value", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"bad_type","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		body := `{"type":"check","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		errStore := &mockVerificationStore{
			user:              dbUser,
			repo:              repo,
			createScheduleErr: errors.New("db error"),
		}
		r := setupVerificationsTestRouter(errStore, trigger, user)
		w := httptest.NewRecorder()
		body := `{"type":"check","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		body := `{"type":"check","cron_expression":"0 2 * * *"}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/verification-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- GetSchedule Tests ---

func TestVerificationsGetSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	schedule := newTestSchedule(repoID)

	store := &mockVerificationStore{
		user:     dbUser,
		repo:     repo,
		schedule: schedule,
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationScheduleResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ID != schedule.ID.String() {
			t.Fatalf("expected id %s, got %s", schedule.ID.String(), resp.ID)
		}
		if resp.CronExpression != schedule.CronExpression {
			t.Fatalf("expected cron %s, got %s", schedule.CronExpression, resp.CronExpression)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verification-schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verification-schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- UpdateSchedule Tests ---

func TestVerificationsUpdateSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	schedule := newTestSchedule(repoID)

	store := &mockVerificationStore{
		user:     dbUser,
		repo:     repo,
		schedule: schedule,
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success update cron", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp VerificationScheduleResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ID != schedule.ID.String() {
			t.Fatalf("expected id %s, got %s", schedule.ID.String(), resp.ID)
		}
	})

	t.Run("success update enabled", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("success update read_data_subset", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"read_data_subset":"10%"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body malformed json", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{invalid`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		errStore := &mockVerificationStore{
			user:              dbUser,
			repo:              repo,
			schedule:          schedule,
			updateScheduleErr: errors.New("db error"),
		}
		r := setupVerificationsTestRouter(errStore, trigger, user)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		body := `{"cron_expression":"0 4 * * *"}`
		req, _ := http.NewRequest("PUT", "/api/v1/verification-schedules/"+schedule.ID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- DeleteSchedule Tests ---

func TestVerificationsDeleteSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "test-repo"}
	schedule := newTestSchedule(repoID)

	store := &mockVerificationStore{
		user:     dbUser,
		repo:     repo,
		schedule: schedule,
	}
	trigger := &mockVerificationTrigger{}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["message"] != "verification schedule deleted" {
			t.Fatalf("expected deletion message, got %s", resp["message"])
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("repo access denied", func(t *testing.T) {
		otherUserID := uuid.New()
		otherSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: orgID}
		r := setupVerificationsTestRouter(store, trigger, otherSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		errStore := &mockVerificationStore{
			user:              dbUser,
			repo:              repo,
			schedule:          schedule,
			deleteScheduleErr: errors.New("db error"),
		}
		r := setupVerificationsTestRouter(errStore, trigger, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupVerificationsTestRouter(store, trigger, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-schedules/"+schedule.ID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

// --- toVerificationResponse Tests ---

func TestToVerificationResponse(t *testing.T) {
	repoID := uuid.New()

	t.Run("with completed_at", func(t *testing.T) {
		ver := newTestVerification(repoID)
		resp := toVerificationResponse(ver)

		if resp.ID != ver.ID.String() {
			t.Fatalf("expected id %s, got %s", ver.ID.String(), resp.ID)
		}
		if resp.CompletedAt == "" {
			t.Fatal("expected completed_at to be set")
		}
		if resp.SnapshotID != ver.SnapshotID {
			t.Fatalf("expected snapshot_id %s, got %s", ver.SnapshotID, resp.SnapshotID)
		}
		if resp.DurationMs == nil {
			t.Fatal("expected duration_ms to be set")
		}
	})

	t.Run("without completed_at", func(t *testing.T) {
		ver := &models.Verification{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Type:         models.VerificationTypeCheck,
			StartedAt:    time.Now(),
			Status:       models.VerificationStatusRunning,
			CreatedAt:    time.Now(),
		}
		resp := toVerificationResponse(ver)

		if resp.CompletedAt != "" {
			t.Fatalf("expected empty completed_at, got %s", resp.CompletedAt)
		}
		if resp.DurationMs != nil {
			t.Fatal("expected nil duration_ms")
		}
	})

	t.Run("with details", func(t *testing.T) {
		ver := newTestVerification(repoID)
		ver.Details = &models.VerificationDetails{
			ErrorsFound:   []string{"error1"},
			FilesRestored: 10,
			BytesRestored: 1024,
		}
		resp := toVerificationResponse(ver)

		if resp.Details == nil {
			t.Fatal("expected details to be set")
		}
		if len(resp.Details.ErrorsFound) != 1 {
			t.Fatalf("expected 1 error, got %d", len(resp.Details.ErrorsFound))
		}
	})

	t.Run("with error message", func(t *testing.T) {
		ver := newTestVerification(repoID)
		ver.ErrorMessage = "something went wrong"
		resp := toVerificationResponse(ver)

		if resp.ErrorMessage != "something went wrong" {
			t.Fatalf("expected error message, got %s", resp.ErrorMessage)
		}
	})
}

// --- toVerificationScheduleResponse Tests ---

func TestToVerificationScheduleResponse(t *testing.T) {
	repoID := uuid.New()
	sched := newTestSchedule(repoID)
	sched.ReadDataSubset = "2.5%"

	resp := toVerificationScheduleResponse(sched)

	if resp.ID != sched.ID.String() {
		t.Fatalf("expected id %s, got %s", sched.ID.String(), resp.ID)
	}
	if resp.RepositoryID != repoID.String() {
		t.Fatalf("expected repository_id %s, got %s", repoID.String(), resp.RepositoryID)
	}
	if resp.Type != string(models.VerificationTypeCheck) {
		t.Fatalf("expected type 'check', got %s", resp.Type)
	}
	if resp.CronExpression != "0 2 * * *" {
		t.Fatalf("expected cron '0 2 * * *', got %s", resp.CronExpression)
	}
	if !resp.Enabled {
		t.Fatal("expected enabled to be true")
	}
	if resp.ReadDataSubset != "2.5%" {
		t.Fatalf("expected read_data_subset '2.5%%', got %s", resp.ReadDataSubset)
	}
	if resp.CreatedAt == "" {
		t.Fatal("expected created_at to be set")
	}
	if resp.UpdatedAt == "" {
		t.Fatal("expected updated_at to be set")
	}
}
