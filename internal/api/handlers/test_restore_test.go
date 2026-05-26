package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockTestRestoreStore struct {
	settings *models.TestRestoreSettings
	result   *models.TestRestoreResult
	results  []*models.TestRestoreResult
	summary  *db.TestRestoreSummary
	repo     *models.Repository
	consec   int
	err      error
}

func (m *mockTestRestoreStore) GetTestRestoreSettingsByID(_ context.Context, _ uuid.UUID) (*models.TestRestoreSettings, error) {
	return m.settings, m.err
}

func (m *mockTestRestoreStore) GetTestRestoreSettingsByRepoID(_ context.Context, _ uuid.UUID) (*models.TestRestoreSettings, error) {
	return m.settings, m.err
}

func (m *mockTestRestoreStore) CreateTestRestoreSettings(_ context.Context, _ *models.TestRestoreSettings) error {
	return m.err
}

func (m *mockTestRestoreStore) UpdateTestRestoreSettings(_ context.Context, _ *models.TestRestoreSettings) error {
	return m.err
}

func (m *mockTestRestoreStore) DeleteTestRestoreSettings(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockTestRestoreStore) GetTestRestoreResultsByRepoID(_ context.Context, _ uuid.UUID, _ int) ([]*models.TestRestoreResult, error) {
	return m.results, m.err
}

func (m *mockTestRestoreStore) GetTestRestoreResultByID(_ context.Context, _ uuid.UUID) (*models.TestRestoreResult, error) {
	return m.result, m.err
}

func (m *mockTestRestoreStore) GetLatestTestRestoreResultByRepoID(_ context.Context, _ uuid.UUID) (*models.TestRestoreResult, error) {
	return m.result, m.err
}

func (m *mockTestRestoreStore) GetConsecutiveFailedTestRestores(_ context.Context, _ uuid.UUID) (int, error) {
	return m.consec, m.err
}

func (m *mockTestRestoreStore) GetTestRestoreSummaryByOrgID(_ context.Context, _ uuid.UUID) (*db.TestRestoreSummary, error) {
	if m.summary != nil {
		return m.summary, m.err
	}
	return &db.TestRestoreSummary{}, m.err
}

func (m *mockTestRestoreStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return m.repo, m.err
}

type stubTestRestoreTrigger struct{}

func (s *stubTestRestoreTrigger) TriggerTestRestore(_ context.Context, _ uuid.UUID, _ int) (*models.TestRestoreResult, error) {
	return &models.TestRestoreResult{}, nil
}

func (s *stubTestRestoreTrigger) GetRepositoryTestRestoreStatus(_ context.Context, _ uuid.UUID) (*models.TestRestoreStatus, error) {
	return &models.TestRestoreStatus{}, nil
}

func setupTestRestoreTestRouter(store TestRestoreStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewTestRestoreHandler(store, &stubTestRestoreTrigger{}, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestTestRestoreGetSummary(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockTestRestoreStore{}
	r := setupTestRestoreTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard/test-restore-summary"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestTestRestoreGetResult(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockTestRestoreStore{}
		r := setupTestRestoreTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/test-restore-results/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
