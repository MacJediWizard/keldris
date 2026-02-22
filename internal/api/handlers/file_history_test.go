package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockFileHistoryStore struct {
	user       *models.User
	agent      *models.Agent
	repo       *models.Repository
	backups    []*models.Backup
	schedule   *models.Schedule
	getUserErr error
	getAgentErr error
	getRepoErr  error
	getBackupsErr error
	getScheduleErr error
}

func (m *mockFileHistoryStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	return m.user, nil
}

func (m *mockFileHistoryStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.getAgentErr != nil {
		return nil, m.getAgentErr
	}
	if m.agent != nil && m.agent.ID == id {
		return m.agent, nil
	}
	return nil, errors.New("not found")
}

func (m *mockFileHistoryStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockFileHistoryStore) GetBackupsByAgentID(_ context.Context, _ uuid.UUID) ([]*models.Backup, error) {
	if m.getBackupsErr != nil {
		return nil, m.getBackupsErr
	}
	return m.backups, nil
}

func (m *mockFileHistoryStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.getScheduleErr != nil {
		return nil, m.getScheduleErr
	}
	return m.schedule, nil
}

func setupFileHistoryTestRouter(store FileHistoryStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewFileHistoryHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestFileHistoryGetHistory(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	agentID := uuid.New()
	repoID := uuid.New()
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host1"}
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "repo1"}
	scheduleID := uuid.New()

	t.Run("success with versions", func(t *testing.T) {
		schedule := &models.Schedule{
			ID:      scheduleID,
			AgentID: agentID,
			Paths:   []string{"/home/data"},
			Repositories: []models.ScheduleRepository{
				{RepositoryID: repoID},
			},
		}
		backups := []*models.Backup{
			{
				ID:         uuid.New(),
				ScheduleID: scheduleID,
				AgentID:    agentID,
				SnapshotID: "abcdef1234567890",
				Status:     models.BackupStatusCompleted,
				StartedAt:  time.Now().Add(-1 * time.Hour),
			},
		}
		store := &mockFileHistoryStore{
			user:     dbUser,
			agent:    agent,
			repo:     repo,
			backups:  backups,
			schedule: schedule,
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/home/data/file.txt&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body FileHistoryResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body.Versions) != 1 {
			t.Fatalf("expected 1 version, got %d", len(body.Versions))
		}
		if body.Versions[0].ShortID != "abcdef12" {
			t.Fatalf("expected short id 'abcdef12', got %q", body.Versions[0].ShortID)
		}
	})

	t.Run("no versions", func(t *testing.T) {
		store := &mockFileHistoryStore{
			user:    dbUser,
			agent:   agent,
			repo:    repo,
			backups: []*models.Backup{},
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/home/data/file.txt&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}

		var body FileHistoryResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Message == "" {
			t.Fatal("expected message for no versions")
		}
	})

	t.Run("missing path", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?agent_id="+agentID.String()+"&repository_id="+repoID.String()))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing agent_id", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?path=/test&repository_id="+repoID.String()))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing repository_id", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?path=/test&agent_id="+agentID.String()))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?path=/test&agent_id=bad&repository_id="+repoID.String()))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid repository_id", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?path=/test&agent_id="+agentID.String()+"&repository_id=bad"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockFileHistoryStore{getUserErr: errors.New("not found")}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		store := &mockFileHistoryStore{
			user:        dbUser,
			getAgentErr: errors.New("not found"),
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: agentID, OrgID: uuid.New(), Hostname: "other"}
		store := &mockFileHistoryStore{
			user:  dbUser,
			agent: otherAgent,
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		store := &mockFileHistoryStore{
			user:       dbUser,
			agent:      agent,
			getRepoErr: errors.New("not found"),
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("repo wrong org", func(t *testing.T) {
		otherRepo := &models.Repository{ID: repoID, OrgID: uuid.New(), Name: "other"}
		store := &mockFileHistoryStore{
			user:  dbUser,
			agent: agent,
			repo:  otherRepo,
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("backups error", func(t *testing.T) {
		store := &mockFileHistoryStore{
			user:          dbUser,
			agent:         agent,
			repo:          repo,
			getBackupsErr: errors.New("db error"),
		}
		r := setupFileHistoryTestRouter(store, user)

		path := "/api/v1/files/history?path=/test&agent_id=" + agentID.String() + "&repository_id=" + repoID.String()
		resp := DoRequest(r, AuthenticatedRequest("GET", path))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockFileHistoryStore{}
		r := setupFileHistoryTestRouter(store, nil)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/files/history?path=/test&agent_id="+agentID.String()+"&repository_id="+repoID.String()))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}
