package monitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// mockAlertStore implements AlertStore for testing.
type mockAlertStore struct {
	alerts          map[uuid.UUID]*models.Alert
	alertsByOrg     map[uuid.UUID][]*models.Alert
	alertRules      map[uuid.UUID]*models.AlertRule
	alertRulesByOrg map[uuid.UUID][]*models.AlertRule
	resourceAlert   *models.Alert

	createErr       error
	updateErr       error
	getErr          error
	listErr         error
	activeListErr   error
	activeCountErr  error
	resourceAlertErr error
	resolveErr      error
	createRuleErr   error
	updateRuleErr   error
	deleteRuleErr   error
	getRuleErr      error
	listRulesErr    error
	enabledRulesErr error
}

func newMockAlertStore() *mockAlertStore {
	return &mockAlertStore{
		alerts:          make(map[uuid.UUID]*models.Alert),
		alertsByOrg:     make(map[uuid.UUID][]*models.Alert),
		alertRules:      make(map[uuid.UUID]*models.AlertRule),
		alertRulesByOrg: make(map[uuid.UUID][]*models.AlertRule),
	}
}

func (m *mockAlertStore) CreateAlert(_ context.Context, alert *models.Alert) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.alerts[alert.ID] = alert
	m.alertsByOrg[alert.OrgID] = append(m.alertsByOrg[alert.OrgID], alert)
	return nil
}

func (m *mockAlertStore) UpdateAlert(_ context.Context, alert *models.Alert) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockAlertStore) GetAlertByID(_ context.Context, id uuid.UUID) (*models.Alert, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if a, ok := m.alerts[id]; ok {
		return a, nil
	}
	return nil, errors.New("alert not found")
}

func (m *mockAlertStore) GetAlertsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.alertsByOrg[orgID], nil
}

func (m *mockAlertStore) GetActiveAlertsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	if m.activeListErr != nil {
		return nil, m.activeListErr
	}
	var active []*models.Alert
	for _, a := range m.alertsByOrg[orgID] {
		if a.Status == models.AlertStatusActive {
			active = append(active, a)
		}
	}
	return active, nil
}

func (m *mockAlertStore) GetActiveAlertCountByOrgID(_ context.Context, orgID uuid.UUID) (int, error) {
	if m.activeCountErr != nil {
		return 0, m.activeCountErr
	}
	count := 0
	for _, a := range m.alertsByOrg[orgID] {
		if a.Status == models.AlertStatusActive {
			count++
		}
	}
	return count, nil
}

func (m *mockAlertStore) GetAlertByResourceAndType(_ context.Context, _ uuid.UUID, _ models.ResourceType, _ uuid.UUID, _ models.AlertType) (*models.Alert, error) {
	if m.resourceAlertErr != nil {
		return nil, m.resourceAlertErr
	}
	if m.resourceAlert != nil {
		return m.resourceAlert, nil
	}
	return nil, pgx.ErrNoRows
}

func (m *mockAlertStore) ResolveAlertsByResource(_ context.Context, _ models.ResourceType, _ uuid.UUID) error {
	return m.resolveErr
}

func (m *mockAlertStore) GetAlertRulesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	if m.listRulesErr != nil {
		return nil, m.listRulesErr
	}
	return m.alertRulesByOrg[orgID], nil
}

func (m *mockAlertStore) GetEnabledAlertRulesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	if m.enabledRulesErr != nil {
		return nil, m.enabledRulesErr
	}
	var enabled []*models.AlertRule
	for _, r := range m.alertRulesByOrg[orgID] {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	return enabled, nil
}

func (m *mockAlertStore) GetAlertRuleByID(_ context.Context, id uuid.UUID) (*models.AlertRule, error) {
	if m.getRuleErr != nil {
		return nil, m.getRuleErr
	}
	if r, ok := m.alertRules[id]; ok {
		return r, nil
	}
	return nil, errors.New("rule not found")
}

func (m *mockAlertStore) CreateAlertRule(_ context.Context, rule *models.AlertRule) error {
	if m.createRuleErr != nil {
		return m.createRuleErr
	}
	m.alertRules[rule.ID] = rule
	m.alertRulesByOrg[rule.OrgID] = append(m.alertRulesByOrg[rule.OrgID], rule)
	return nil
}

func (m *mockAlertStore) UpdateAlertRule(_ context.Context, rule *models.AlertRule) error {
	if m.updateRuleErr != nil {
		return m.updateRuleErr
	}
	m.alertRules[rule.ID] = rule
	return nil
}

func (m *mockAlertStore) DeleteAlertRule(_ context.Context, id uuid.UUID) error {
	if m.deleteRuleErr != nil {
		return m.deleteRuleErr
	}
	delete(m.alertRules, id)
	return nil
}

// mockNotifier implements NotificationSender for testing.
type mockNotifier struct {
	sent    []*models.Alert
	sendErr error
}

func (n *mockNotifier) SendAlertNotification(_ context.Context, alert *models.Alert) error {
	if n.sendErr != nil {
		return n.sendErr
	}
	n.sent = append(n.sent, alert)
	return nil
}

func TestAlertManager_Evaluate(t *testing.T) {
	orgID := uuid.New()

	t.Run("creates alert and stores it", func(t *testing.T) {
		store := newMockAlertStore()
		notifier := &mockNotifier{}
		svc := NewAlertService(store, notifier, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Agent down", "Agent X is offline")
		err := svc.CreateAlert(context.Background(), alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := store.alerts[alert.ID]; !ok {
			t.Error("alert not stored")
		}
	})

	t.Run("sends notification on create", func(t *testing.T) {
		store := newMockAlertStore()
		notifier := &mockNotifier{}
		svc := NewAlertService(store, notifier, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Agent down", "Details")
		err := svc.CreateAlert(context.Background(), alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifier.sent) != 1 {
			t.Errorf("expected 1 notification, got %d", len(notifier.sent))
		}
	})

	t.Run("create succeeds even if notification fails", func(t *testing.T) {
		store := newMockAlertStore()
		notifier := &mockNotifier{sendErr: errors.New("email failed")}
		svc := NewAlertService(store, notifier, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "SLA breach", "Details")
		err := svc.CreateAlert(context.Background(), alert)
		if err != nil {
			t.Fatalf("expected no error despite notification failure, got: %v", err)
		}
	})

	t.Run("create fails on store error", func(t *testing.T) {
		store := newMockAlertStore()
		store.createErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		err := svc.CreateAlert(context.Background(), alert)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("create with nil notification sender", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		err := svc.CreateAlert(context.Background(), alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("HasActiveAlert returns true when alert exists", func(t *testing.T) {
		store := newMockAlertStore()
		existingAlert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Active", "Alert")
		store.resourceAlert = existingAlert
		svc := NewAlertService(store, nil, zerolog.Nop())

		agentID := uuid.New()
		has, err := svc.HasActiveAlert(context.Background(), orgID, models.ResourceTypeAgent, agentID, models.AlertTypeAgentOffline)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected active alert to exist")
		}
	})

	t.Run("HasActiveAlert returns false when no alert", func(t *testing.T) {
		store := newMockAlertStore()
		store.resourceAlertErr = pgx.ErrNoRows
		svc := NewAlertService(store, nil, zerolog.Nop())

		agentID := uuid.New()
		has, err := svc.HasActiveAlert(context.Background(), orgID, models.ResourceTypeAgent, agentID, models.AlertTypeAgentOffline)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected no active alert")
		}
	})

	t.Run("HasActiveAlert returns error on db failure", func(t *testing.T) {
		store := newMockAlertStore()
		store.resourceAlertErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		_, err := svc.HasActiveAlert(context.Background(), orgID, models.ResourceTypeAgent, uuid.New(), models.AlertTypeAgentOffline)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAlertManager_Trigger(t *testing.T) {
	orgID := uuid.New()

	t.Run("get alert by ID", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		store.alerts[alert.ID] = alert

		got, err := svc.GetAlert(context.Background(), alert.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != alert.ID {
			t.Errorf("expected alert %s, got %s", alert.ID, got.ID)
		}
	})

	t.Run("list alerts for org", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		a1 := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Alert 1", "Msg")
		a2 := models.NewAlert(orgID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "Alert 2", "Msg")
		store.alertsByOrg[orgID] = []*models.Alert{a1, a2}

		alerts, err := svc.ListAlerts(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alerts) != 2 {
			t.Errorf("expected 2 alerts, got %d", len(alerts))
		}
	})

	t.Run("list active alerts", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		active := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Active", "Msg")
		resolved := models.NewAlert(orgID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "Resolved", "Msg")
		resolved.Resolve()

		store.alertsByOrg[orgID] = []*models.Alert{active, resolved}

		alerts, err := svc.ListActiveAlerts(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(alerts) != 1 {
			t.Errorf("expected 1 active alert, got %d", len(alerts))
		}
	})

	t.Run("get active alert count", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		a1 := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "A1", "Msg")
		a2 := models.NewAlert(orgID, models.AlertTypeBackupSLA, models.AlertSeverityCritical, "A2", "Msg")
		store.alertsByOrg[orgID] = []*models.Alert{a1, a2}

		count, err := svc.GetActiveAlertCount(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2, got %d", count)
		}
	})

	t.Run("acknowledge alert", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		store.alerts[alert.ID] = alert

		userID := uuid.New()
		err := svc.AcknowledgeAlert(context.Background(), alert.ID, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alert.Status != models.AlertStatusAcknowledged {
			t.Errorf("expected acknowledged, got %s", alert.Status)
		}
		if alert.AcknowledgedBy == nil || *alert.AcknowledgedBy != userID {
			t.Error("expected acknowledged_by to be set")
		}
	})

	t.Run("cannot acknowledge resolved alert", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		alert.Resolve()
		store.alerts[alert.ID] = alert

		err := svc.AcknowledgeAlert(context.Background(), alert.ID, uuid.New())
		if err == nil {
			t.Fatal("expected error acknowledging resolved alert")
		}
	})

	t.Run("acknowledge fails on get error", func(t *testing.T) {
		store := newMockAlertStore()
		store.getErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		err := svc.AcknowledgeAlert(context.Background(), uuid.New(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("acknowledge fails on update error", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		store.alerts[alert.ID] = alert
		store.updateErr = errors.New("db error")

		err := svc.AcknowledgeAlert(context.Background(), alert.ID, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAlertManager_Resolve(t *testing.T) {
	orgID := uuid.New()

	t.Run("resolves active alert", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		store.alerts[alert.ID] = alert

		err := svc.ResolveAlert(context.Background(), alert.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alert.Status != models.AlertStatusResolved {
			t.Errorf("expected resolved, got %s", alert.Status)
		}
		if alert.ResolvedAt == nil {
			t.Error("expected resolved_at to be set")
		}
	})

	t.Run("resolving already resolved alert is idempotent", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		alert.Resolve()
		store.alerts[alert.ID] = alert

		err := svc.ResolveAlert(context.Background(), alert.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resolve fails on get error", func(t *testing.T) {
		store := newMockAlertStore()
		store.getErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		err := svc.ResolveAlert(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("resolve fails on update error", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		alert := models.NewAlert(orgID, models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")
		store.alerts[alert.ID] = alert
		store.updateErr = errors.New("db error")

		err := svc.ResolveAlert(context.Background(), alert.ID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("resolve alerts by resource", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		agentID := uuid.New()
		err := svc.ResolveAlertsByResource(context.Background(), models.ResourceTypeAgent, agentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resolve by resource fails on store error", func(t *testing.T) {
		store := newMockAlertStore()
		store.resolveErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		err := svc.ResolveAlertsByResource(context.Background(), models.ResourceTypeAgent, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAlertRuleCRUD(t *testing.T) {
	orgID := uuid.New()

	t.Run("create alert rule", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "Agent offline rule", models.AlertTypeAgentOffline, models.AlertRuleConfig{
			OfflineThresholdMinutes: 5,
		})
		err := svc.CreateAlertRule(context.Background(), rule)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := store.alertRules[rule.ID]; !ok {
			t.Error("rule not stored")
		}
	})

	t.Run("create rule fails on store error", func(t *testing.T) {
		store := newMockAlertStore()
		store.createRuleErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "Test", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		err := svc.CreateAlertRule(context.Background(), rule)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("get alert rule", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "Test rule", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		store.alertRules[rule.ID] = rule

		got, err := svc.GetAlertRule(context.Background(), rule.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != rule.ID {
			t.Errorf("expected %s, got %s", rule.ID, got.ID)
		}
	})

	t.Run("list alert rules", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		r1 := models.NewAlertRule(orgID, "Rule 1", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		r2 := models.NewAlertRule(orgID, "Rule 2", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		store.alertRulesByOrg[orgID] = []*models.AlertRule{r1, r2}

		rules, err := svc.ListAlertRules(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}
	})

	t.Run("list enabled alert rules", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		r1 := models.NewAlertRule(orgID, "Enabled", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		r2 := models.NewAlertRule(orgID, "Disabled", models.AlertTypeBackupSLA, models.AlertRuleConfig{})
		r2.Enabled = false
		store.alertRulesByOrg[orgID] = []*models.AlertRule{r1, r2}

		rules, err := svc.ListEnabledAlertRules(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 1 {
			t.Errorf("expected 1 enabled rule, got %d", len(rules))
		}
	})

	t.Run("update alert rule", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "Original", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		store.alertRules[rule.ID] = rule

		rule.Name = "Updated"
		err := svc.UpdateAlertRule(context.Background(), rule)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.alertRules[rule.ID].Name != "Updated" {
			t.Errorf("expected name 'Updated', got %q", store.alertRules[rule.ID].Name)
		}
	})

	t.Run("update rule fails on store error", func(t *testing.T) {
		store := newMockAlertStore()
		store.updateRuleErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "Test", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		err := svc.UpdateAlertRule(context.Background(), rule)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("delete alert rule", func(t *testing.T) {
		store := newMockAlertStore()
		svc := NewAlertService(store, nil, zerolog.Nop())

		rule := models.NewAlertRule(orgID, "To delete", models.AlertTypeAgentOffline, models.AlertRuleConfig{})
		store.alertRules[rule.ID] = rule

		err := svc.DeleteAlertRule(context.Background(), rule.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := store.alertRules[rule.ID]; ok {
			t.Error("rule should have been deleted")
		}
	})

	t.Run("delete rule fails on store error", func(t *testing.T) {
		store := newMockAlertStore()
		store.deleteRuleErr = errors.New("db error")
		svc := NewAlertService(store, nil, zerolog.Nop())

		err := svc.DeleteAlertRule(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoOpNotificationSender(t *testing.T) {
	sender := &NoOpNotificationSender{}
	alert := models.NewAlert(uuid.New(), models.AlertTypeAgentOffline, models.AlertSeverityWarning, "Title", "Msg")

	err := sender.SendAlertNotification(context.Background(), alert)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}
