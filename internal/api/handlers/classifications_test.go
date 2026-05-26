package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockClassificationStore struct {
	rules        []*models.PathClassificationRule
	ruleByID     map[uuid.UUID]*models.PathClassificationRule
	schedules    []*models.Schedule
	scheduleByID map[uuid.UUID]*models.Schedule
	agent        *models.Agent
	scheduleCls  *models.ScheduleClassification
	backupCls    *models.BackupClassification
	backups      []*models.Backup
	summary      *models.ClassificationSummary
	err          error
}

func (m *mockClassificationStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.scheduleByID[id], nil
}

func (m *mockClassificationStore) GetSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.err
}

func (m *mockClassificationStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockClassificationStore) GetPathClassificationRulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.PathClassificationRule, error) {
	return m.rules, m.err
}

func (m *mockClassificationStore) GetPathClassificationRuleByID(_ context.Context, id uuid.UUID) (*models.PathClassificationRule, error) {
	if m.ruleByID == nil {
		return nil, m.err
	}
	return m.ruleByID[id], m.err
}

func (m *mockClassificationStore) CreatePathClassificationRule(_ context.Context, r *models.PathClassificationRule) error {
	if m.err != nil {
		return m.err
	}
	if m.ruleByID == nil {
		m.ruleByID = map[uuid.UUID]*models.PathClassificationRule{}
	}
	m.ruleByID[r.ID] = r
	m.rules = append(m.rules, r)
	return nil
}

func (m *mockClassificationStore) UpdatePathClassificationRule(_ context.Context, _ *models.PathClassificationRule) error {
	return m.err
}

func (m *mockClassificationStore) DeletePathClassificationRule(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockClassificationStore) GetScheduleClassification(_ context.Context, _ uuid.UUID) (*models.ScheduleClassification, error) {
	return m.scheduleCls, m.err
}

func (m *mockClassificationStore) SetScheduleClassification(_ context.Context, _ *models.ScheduleClassification) error {
	return m.err
}

func (m *mockClassificationStore) UpdateScheduleClassificationLevel(_ context.Context, _ uuid.UUID, _ string, _ []string) error {
	return m.err
}

func (m *mockClassificationStore) GetSchedulesByClassificationLevel(_ context.Context, _ uuid.UUID, _ string) ([]*models.Schedule, error) {
	return m.schedules, m.err
}

func (m *mockClassificationStore) GetBackupClassification(_ context.Context, _ uuid.UUID) (*models.BackupClassification, error) {
	return m.backupCls, m.err
}

func (m *mockClassificationStore) GetBackupsByClassificationLevel(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.Backup, error) {
	return m.backups, m.err
}

func (m *mockClassificationStore) GetClassificationSummary(_ context.Context, _ uuid.UUID) (*models.ClassificationSummary, error) {
	return m.summary, m.err
}

func setupClassificationsTestRouter(store ClassificationStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewClassificationsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestClassificationsListLevels(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockClassificationStore{}
	r := setupClassificationsTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/classifications/levels"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestClassificationsListDataTypes(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockClassificationStore{}
	r := setupClassificationsTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/classifications/data-types"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestClassificationsListRules(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns empty list", func(t *testing.T) {
		store := &mockClassificationStore{}
		r := setupClassificationsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/classifications/rules"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns existing rules", func(t *testing.T) {
		store := &mockClassificationStore{rules: []*models.PathClassificationRule{{ID: uuid.New(), OrgID: orgID, Pattern: "/srv/*"}}}
		r := setupClassificationsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/classifications/rules"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}
