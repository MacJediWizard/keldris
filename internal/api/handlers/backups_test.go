package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockBackupStore implements BackupStore for testing.
type mockBackupStore struct {
	backupsBySchedule map[uuid.UUID][]*models.Backup
	backupsByAgent    map[uuid.UUID][]*models.Backup
	backupByID        map[uuid.UUID]*models.Backup
	agentByID         map[uuid.UUID]*models.Agent
	agentsByOrg       map[uuid.UUID][]*models.Agent
	scheduleByID      map[uuid.UUID]*models.Schedule
}

func newMockBackupStore() *mockBackupStore {
	return &mockBackupStore{
		backupsBySchedule: make(map[uuid.UUID][]*models.Backup),
		backupsByAgent:    make(map[uuid.UUID][]*models.Backup),
		backupByID:        make(map[uuid.UUID]*models.Backup),
		agentByID:         make(map[uuid.UUID]*models.Agent),
		agentsByOrg:       make(map[uuid.UUID][]*models.Agent),
		scheduleByID:      make(map[uuid.UUID]*models.Schedule),
	}
}

func (m *mockBackupStore) GetBackupsByScheduleID(_ context.Context, scheduleID uuid.UUID) ([]*models.Backup, error) {
	return m.backupsBySchedule[scheduleID], nil
}

func (m *mockBackupStore) GetBackupsByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	return m.backupsByAgent[agentID], nil
}

func (m *mockBackupStore) GetBackupByID(_ context.Context, id uuid.UUID) (*models.Backup, error) {
	if b, ok := m.backupByID[id]; ok {
		return b, nil
	}
	return nil, errors.New("backup not found")
}

func (m *mockBackupStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if a, ok := m.agentByID[id]; ok {
		return a, nil
	}
	return nil, errors.New("agent not found")
}

func (m *mockBackupStore) GetAgentsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	return m.agentsByOrg[orgID], nil
}

func (m *mockBackupStore) GetScheduleByID(_ context.Context, id uuid.UUID) (*models.Schedule, error) {
	if s, ok := m.scheduleByID[id]; ok {
		return s, nil
	}
	return nil, errors.New("schedule not found")
}

func setupBackupTestRouter(store BackupStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})

	handler := NewBackupsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListBackups(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	scheduleID := uuid.New()
	backupID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID, Name: "daily"}
	backup := &models.Backup{ID: backupID, ScheduleID: scheduleID, AgentID: agentID, Status: models.BackupStatusCompleted}

	store := newMockBackupStore()
	store.agentByID[agentID] = agent
	store.agentsByOrg[orgID] = []*models.Agent{agent}
	store.scheduleByID[scheduleID] = schedule
	store.backupsBySchedule[scheduleID] = []*models.Backup{backup}
	store.backupsByAgent[agentID] = []*models.Backup{backup}

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("list all", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["backups"]; !ok {
			t.Fatal("expected 'backups' key")
		}
	})

	t.Run("filter by agent_id", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("filter by schedule_id", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?schedule_id="+scheduleID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?status=completed", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?agent_id=bad", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid schedule_id", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?schedule_id=bad", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: uuid.New(), OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockBackupStore()
		store2.agentByID[otherAgent.ID] = otherAgent

		r := setupBackupTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups?agent_id="+otherAgent.ID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupBackupTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}

func TestGetBackup(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	backupID := uuid.New()

	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "test-host"}
	backup := &models.Backup{ID: backupID, AgentID: agentID, Status: models.BackupStatusCompleted}

	store := newMockBackupStore()
	store.agentByID[agentID] = agent
	store.backupByID[backupID] = backup

	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+backupID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/not-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongOrgAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store2 := newMockBackupStore()
		store2.agentByID[agentID] = wrongOrgAgent
		store2.backupByID[backupID] = backup

		r := setupBackupTestRouter(store2, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+backupID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})
}

func TestFilterByStatus(t *testing.T) {
	backups := []*models.Backup{
		{ID: uuid.New(), Status: models.BackupStatusCompleted},
		{ID: uuid.New(), Status: models.BackupStatusFailed},
		{ID: uuid.New(), Status: models.BackupStatusRunning},
		{ID: uuid.New(), Status: models.BackupStatusCompleted},
	}

	t.Run("no filter", func(t *testing.T) {
		result := filterByStatus(backups, "")
		if len(result) != 4 {
			t.Fatalf("expected 4 backups, got %d", len(result))
		}
	})

	t.Run("filter completed", func(t *testing.T) {
		result := filterByStatus(backups, "completed")
		if len(result) != 2 {
			t.Fatalf("expected 2 completed backups, got %d", len(result))
		}
	})

	t.Run("filter failed", func(t *testing.T) {
		result := filterByStatus(backups, "failed")
		if len(result) != 1 {
			t.Fatalf("expected 1 failed backup, got %d", len(result))
		}
	})

	t.Run("filter nonexistent status", func(t *testing.T) {
		result := filterByStatus(backups, "nonexistent")
		if len(result) != 0 {
			t.Fatalf("expected 0 backups, got %d", len(result))
		}
	})
}
