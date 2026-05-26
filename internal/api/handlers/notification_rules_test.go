package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockNotificationRuleStore struct {
	rules   []*models.NotificationRule
	rule    *models.NotificationRule
	channel *models.NotificationChannel
	events  []*models.NotificationRuleEvent
	execs   []*models.NotificationRuleExecution
	err     error
}

func (m *mockNotificationRuleStore) GetNotificationRulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.NotificationRule, error) {
	return m.rules, m.err
}

func (m *mockNotificationRuleStore) GetNotificationRuleByID(_ context.Context, _ uuid.UUID) (*models.NotificationRule, error) {
	return m.rule, m.err
}

func (m *mockNotificationRuleStore) GetEnabledRulesByTriggerType(_ context.Context, _ uuid.UUID, _ models.RuleTriggerType) ([]*models.NotificationRule, error) {
	return m.rules, m.err
}

func (m *mockNotificationRuleStore) CreateNotificationRule(_ context.Context, _ *models.NotificationRule) error {
	return m.err
}

func (m *mockNotificationRuleStore) UpdateNotificationRule(_ context.Context, _ *models.NotificationRule) error {
	return m.err
}

func (m *mockNotificationRuleStore) DeleteNotificationRule(_ context.Context, _, _ uuid.UUID) error {
	return m.err
}

func (m *mockNotificationRuleStore) GetRecentEventsForRule(_ context.Context, _ uuid.UUID, _ int) ([]*models.NotificationRuleEvent, error) {
	return m.events, m.err
}

func (m *mockNotificationRuleStore) GetRecentExecutionsForRule(_ context.Context, _ uuid.UUID, _ int) ([]*models.NotificationRuleExecution, error) {
	return m.execs, m.err
}

func (m *mockNotificationRuleStore) GetNotificationChannelByID(_ context.Context, _ uuid.UUID) (*models.NotificationChannel, error) {
	return m.channel, m.err
}

func (m *mockNotificationRuleStore) CreateNotificationRuleEvent(_ context.Context, _ *models.NotificationRuleEvent) error {
	return nil
}

func (m *mockNotificationRuleStore) CountEventsInTimeWindow(_ context.Context, _ uuid.UUID, _ models.RuleTriggerType, _ *uuid.UUID, _ time.Time) (int, error) {
	return 0, nil
}

func (m *mockNotificationRuleStore) CreateNotificationRuleExecution(_ context.Context, _ *models.NotificationRuleExecution) error {
	return nil
}

func setupNotificationRulesTestRouter(store NotificationRuleStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewNotificationRulesHandler(store, newTestKeyManager(&testing.T{}), zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestNotificationRulesListRules(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockNotificationRuleStore{rules: []*models.NotificationRule{{ID: uuid.New(), OrgID: orgID, Name: "test"}}}
	r := setupNotificationRulesTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/notification-rules"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestNotificationRulesGetRule(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockNotificationRuleStore{}
		r := setupNotificationRulesTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/notification-rules/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("returns rule when found and in org", func(t *testing.T) {
		ruleID := uuid.New()
		store := &mockNotificationRuleStore{rule: &models.NotificationRule{ID: ruleID, OrgID: orgID, Name: "test"}}
		r := setupNotificationRulesTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/notification-rules/"+ruleID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestNotificationRulesDeleteRule(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	ruleID := uuid.New()

	t.Run("deletes rule in own org", func(t *testing.T) {
		store := &mockNotificationRuleStore{rule: &models.NotificationRule{ID: ruleID, OrgID: orgID, Name: "x"}}
		r := setupNotificationRulesTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/notification-rules/"+ruleID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("404 when rule belongs to other org", func(t *testing.T) {
		otherOrg := uuid.New()
		store := &mockNotificationRuleStore{rule: &models.NotificationRule{ID: ruleID, OrgID: otherOrg, Name: "x"}}
		r := setupNotificationRulesTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/notification-rules/"+ruleID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}
