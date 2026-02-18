package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	pkgmodels "github.com/MacJediWizard/keldris/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// mockAgentAPIStore implements AgentAPIStore for testing.
type mockAgentAPIStore struct {
	updateErr        error
	healthHistoryErr error
	createAlertErr   error
	getAlertErr      error
	existingAlert    *models.Alert
	resolveErr       error
	updatedAgent     *models.Agent
	createdHistory   *models.AgentHealthHistory
	createdAlert     *models.Alert
	resolvedResource bool
}

func (m *mockAgentAPIStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return nil, nil // not used in ReportHealth
}

func (m *mockAgentAPIStore) UpdateAgent(_ context.Context, agent *models.Agent) error {
	m.updatedAgent = agent
	return m.updateErr
}

func (m *mockAgentAPIStore) CreateAgentHealthHistory(_ context.Context, history *models.AgentHealthHistory) error {
	m.createdHistory = history
	return m.healthHistoryErr
}

func (m *mockAgentAPIStore) CreateAlert(_ context.Context, alert *models.Alert) error {
	m.createdAlert = alert
	return m.createAlertErr
}

func (m *mockAgentAPIStore) GetAlertByResourceAndType(_ context.Context, _ uuid.UUID, _ models.ResourceType, _ uuid.UUID, _ models.AlertType) (*models.Alert, error) {
	if m.getAlertErr != nil {
		return nil, m.getAlertErr
	}
	return m.existingAlert, nil
}

func (m *mockAgentAPIStore) ResolveAlertsByResource(_ context.Context, _ models.ResourceType, _ uuid.UUID) error {
	m.resolvedResource = true
	return m.resolveErr
}

func (m *mockAgentAPIStore) GetSchedulesByAgentID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return nil, nil
}

func (m *mockAgentAPIStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return nil, nil
}

func (m *mockAgentAPIStore) GetRepositoryKeyByRepositoryID(_ context.Context, _ uuid.UUID) (*models.RepositoryKey, error) {
	return nil, nil
}

func (m *mockAgentAPIStore) CreateBackup(_ context.Context, _ *models.Backup) error {
	return nil
}

func (m *mockAgentAPIStore) UpdateBackup(_ context.Context, _ *models.Backup) error {
	return nil
}

func (m *mockAgentAPIStore) GetBackupsByAgentID(_ context.Context, _ uuid.UUID) ([]*models.Backup, error) {
	return nil, nil
}

// InjectAgent returns gin middleware that injects an Agent into context.
func InjectAgent(agent *models.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		if agent != nil {
			c.Set(string(middleware.AgentContextKey), agent)
		}
		c.Next()
	}
}

func setupAgentAPITestRouter(store AgentAPIStore, agent *models.Agent) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(InjectAgent(agent))
	handler := NewAgentAPIHandler(store, nil, zerolog.Nop())
	agentGroup := r.Group("/api/v1/agent")
	handler.RegisterRoutes(agentGroup)
	return r
}

func TestReportHealthy(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:           agentID,
		OrgID:        orgID,
		Hostname:     "healthy-agent",
		Status:       models.AgentStatusActive,
		HealthStatus: models.HealthStatusHealthy,
	}

	store := &mockAgentAPIStore{
		getAlertErr: pgx.ErrNoRows, // no existing alert
	}

	t.Run("healthy status", func(t *testing.T) {
		testAgent := *agent
		r := setupAgentAPITestRouter(store, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"healthy"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp pkgmodels.HeartbeatResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.Acknowledged {
			t.Fatal("expected acknowledged to be true")
		}
		if resp.AgentID != agentID.String() {
			t.Fatalf("expected agent_id %s, got %s", agentID.String(), resp.AgentID)
		}
	})

	t.Run("unhealthy status", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"unhealthy"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("degraded status", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"degraded"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestReportHealthWithMetrics(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:           agentID,
		OrgID:        orgID,
		Hostname:     "metrics-agent",
		Status:       models.AgentStatusActive,
		HealthStatus: models.HealthStatusHealthy,
	}

	t.Run("with metrics", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{
			"status":"healthy",
			"metrics":{
				"cpu_usage":25.5,
				"memory_usage":60.0,
				"disk_usage":45.0,
				"disk_free_bytes":100000000000,
				"disk_total_bytes":200000000000,
				"network_up":true,
				"uptime_seconds":86400,
				"restic_version":"0.16.4",
				"restic_available":true
			}
		}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		if s.updatedAgent == nil {
			t.Fatal("expected agent to be updated")
		}
		if s.updatedAgent.HealthMetrics == nil {
			t.Fatal("expected health metrics to be set")
		}
		if s.updatedAgent.HealthMetrics.CPUUsage != 25.5 {
			t.Fatalf("expected CPU usage 25.5, got %f", s.updatedAgent.HealthMetrics.CPUUsage)
		}
	})

	t.Run("with os info", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{
			"status":"healthy",
			"os_info":{"os":"linux","arch":"amd64","hostname":"test","version":"Ubuntu 22.04"}
		}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		if s.updatedAgent == nil {
			t.Fatal("expected agent to be updated")
		}
		if s.updatedAgent.OSInfo == nil {
			t.Fatal("expected OS info to be set")
		}
		if s.updatedAgent.OSInfo.OS != "linux" {
			t.Fatalf("expected OS 'linux', got %q", s.updatedAgent.OSInfo.OS)
		}
	})
}

func TestReportHealthInvalidBody(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "invalid-agent",
		Status:   models.AgentStatusActive,
	}

	store := &mockAgentAPIStore{}

	t.Run("invalid body", func(t *testing.T) {
		testAgent := *agent
		r := setupAgentAPITestRouter(store, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"invalid_value"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing status", func(t *testing.T) {
		testAgent := *agent
		r := setupAgentAPITestRouter(store, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		testAgent := *agent
		r := setupAgentAPITestRouter(store, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", ``)
		w := DoRequest(r, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}

func TestReportHealthNoAgent(t *testing.T) {
	store := &mockAgentAPIStore{}

	t.Run("no agent auth", func(t *testing.T) {
		r := setupAgentAPITestRouter(store, nil)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"healthy"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestReportHealthStoreErrors(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:           agentID,
		OrgID:        orgID,
		Hostname:     "error-agent",
		Status:       models.AgentStatusActive,
		HealthStatus: models.HealthStatusHealthy,
	}

	t.Run("update agent error", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{
			updateErr: errors.New("db error"),
		}
		r := setupAgentAPITestRouter(s, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"healthy"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("health history error still succeeds", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{
			healthHistoryErr: errors.New("db error"),
			getAlertErr:      pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, &testAgent)
		req := JSONRequest("POST", "/api/v1/agent/health", `{"status":"healthy"}`)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 (health history error is non-fatal), got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestReportHealthThresholds(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:           agentID,
		OrgID:        orgID,
		Hostname:     "threshold-agent",
		Status:       models.AgentStatusActive,
		HealthStatus: models.HealthStatusHealthy,
	}

	t.Run("critical disk >= 90", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":92,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusCritical {
			t.Fatalf("expected critical health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("warning disk >= 80", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":85,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("critical memory >= 95", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":97,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusCritical {
			t.Fatalf("expected critical health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("warning memory >= 85", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":88,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("critical CPU >= 95", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":97,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusCritical {
			t.Fatalf("expected critical health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("warning CPU >= 80", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":82,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning health status, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("network down", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":50,"network_up":false,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning health status for network down, got %q", s.updatedAgent.HealthStatus)
		}
	})

	t.Run("restic not available", func(t *testing.T) {
		testAgent := *agent
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, &testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":false}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning health status for restic unavailable, got %q", s.updatedAgent.HealthStatus)
		}
	})
}

func TestReportHealthStatusChanges(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	t.Run("status change from healthy to critical creates alert", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "alert-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy, // previous status
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows, // no existing alert
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":95,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.createdAlert == nil {
			t.Fatal("expected alert to be created on status change to critical")
		}
	})

	t.Run("status change from healthy to warning creates alert", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "warn-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy,
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":82,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.createdAlert == nil {
			t.Fatal("expected alert to be created on status change to warning")
		}
	})

	t.Run("status change to healthy resolves alerts", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "resolve-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusCritical, // was critical
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if !s.resolvedResource {
			t.Fatal("expected alerts to be resolved on status change to healthy")
		}
	})

	t.Run("existing alert found no duplicate", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "dup-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy, // was healthy
		}
		existingAlert := models.NewAlert(orgID, models.AlertTypeAgentHealthCritical, models.AlertSeverityCritical, "Existing", "msg")
		s := &mockAgentAPIStore{
			existingAlert: existingAlert, // alert already exists
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":95,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.createdAlert != nil {
			t.Fatal("expected no duplicate alert to be created when one already exists")
		}
	})

	t.Run("no status change no alert", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "stable-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy,
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.createdAlert != nil {
			t.Fatal("expected no alert when status does not change")
		}
		if s.resolvedResource {
			t.Fatal("expected no resolve when status does not change")
		}
	})

	t.Run("status change from empty previous to healthy no alert", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "new-agent",
			Status:       models.AgentStatusPending,
			HealthStatus: "", // empty previous
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy","metrics":{"cpu_usage":10,"memory_usage":50,"disk_usage":50,"network_up":true,"restic_available":true}}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		// When transitioning from empty to healthy, it resolves alerts (no-op) since it's a status change
		// No critical alert should be created
		if s.createdAlert != nil {
			t.Fatal("expected no alert when transitioning to healthy")
		}
	})

	t.Run("unhealthy reported status becomes critical", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "unhealthy-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy,
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"unhealthy"}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusCritical {
			t.Fatalf("expected critical status for unhealthy report, got %q", s.updatedAgent.HealthStatus)
		}
		if s.createdAlert == nil {
			t.Fatal("expected alert to be created for status change")
		}
	})

	t.Run("degraded status with no metrics becomes warning", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "degraded-agent",
			Status:       models.AgentStatusActive,
			HealthStatus: models.HealthStatusHealthy,
		}
		s := &mockAgentAPIStore{
			getAlertErr: pgx.ErrNoRows,
		}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"degraded"}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.HealthStatus != models.HealthStatusWarning {
			t.Fatalf("expected warning status for degraded report, got %q", s.updatedAgent.HealthStatus)
		}
	})
}

func TestReportHealthAgentFieldsUpdated(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	t.Run("agent marked active and seen", func(t *testing.T) {
		testAgent := &models.Agent{
			ID:           agentID,
			OrgID:        orgID,
			Hostname:     "update-agent",
			Status:       models.AgentStatusPending,
			HealthStatus: models.HealthStatusHealthy,
		}
		s := &mockAgentAPIStore{getAlertErr: pgx.ErrNoRows}
		r := setupAgentAPITestRouter(s, testAgent)
		body := `{"status":"healthy"}`
		req := JSONRequest("POST", "/api/v1/agent/health", body)
		w := DoRequest(r, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if s.updatedAgent.Status != models.AgentStatusActive {
			t.Fatalf("expected agent status 'active', got %q", s.updatedAgent.Status)
		}
		if s.updatedAgent.LastSeen == nil {
			t.Fatal("expected last_seen to be set")
		}
		if s.updatedAgent.HealthCheckedAt == nil {
			t.Fatal("expected health_checked_at to be set")
		}
	})
}
