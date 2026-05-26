package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockBackupQueueStore struct {
	entries      []*models.BackupQueueEntryWithDetails
	summary      *models.ConcurrencyQueueSummary
	org          *models.Organization
	agent        *models.Agent
	runningOrg   int
	runningAgent int
	queuedAgent  int
	listErr      error
	summaryErr   error
	deleteErr    error
	updateOrgErr error
	updateAgErr  error
	getOrgErr    error
	getAgentErr  error
}

func (m *mockBackupQueueStore) GetQueuedBackupsWithDetails(_ context.Context, _ uuid.UUID) ([]*models.BackupQueueEntryWithDetails, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.entries, nil
}

func (m *mockBackupQueueStore) GetConcurrencyQueueSummary(_ context.Context, _ uuid.UUID) (*models.ConcurrencyQueueSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	if m.summary == nil {
		return &models.ConcurrencyQueueSummary{}, nil
	}
	return m.summary, nil
}

func (m *mockBackupQueueStore) DeleteBackupQueueEntry(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockBackupQueueStore) GetRunningBackupsCountByOrg(_ context.Context, _ uuid.UUID) (int, error) {
	return m.runningOrg, nil
}

func (m *mockBackupQueueStore) GetRunningBackupsCountByAgent(_ context.Context, _ uuid.UUID) (int, error) {
	return m.runningAgent, nil
}

func (m *mockBackupQueueStore) GetQueuedBackupsCountByAgent(_ context.Context, _ uuid.UUID) (int, error) {
	return m.queuedAgent, nil
}

func (m *mockBackupQueueStore) UpdateOrganizationConcurrencyLimit(_ context.Context, _ uuid.UUID, _ *int) error {
	return m.updateOrgErr
}

func (m *mockBackupQueueStore) UpdateAgentConcurrencyLimit(_ context.Context, _ uuid.UUID, _ *int) error {
	return m.updateAgErr
}

func (m *mockBackupQueueStore) GetOrganizationByIDWithConcurrency(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
	if m.getOrgErr != nil {
		return nil, m.getOrgErr
	}
	return m.org, nil
}

func (m *mockBackupQueueStore) GetAgentByIDWithConcurrency(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.getAgentErr != nil {
		return nil, m.getAgentErr
	}
	return m.agent, nil
}

type mockBackupQueueMembershipStore struct{}

func (m *mockBackupQueueMembershipStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	return &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner}, nil
}

func (m *mockBackupQueueMembershipStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	return nil, nil
}

func setupBackupQueueTestRouter(store BackupQueueStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := SetupTestRouter(user)
	rbac := auth.NewRBAC(&mockBackupQueueMembershipStore{})
	handler := NewBackupQueueHandler(store, rbac, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestBackupQueueListQueue(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns queue", func(t *testing.T) {
		store := &mockBackupQueueStore{entries: []*models.BackupQueueEntryWithDetails{}}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-queue"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-queue"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockBackupQueueStore{listErr: errors.New("db down")}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-queue"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestBackupQueueGetSummary(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns summary", func(t *testing.T) {
		store := &mockBackupQueueStore{summary: &models.ConcurrencyQueueSummary{TotalQueued: 5}}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-queue/summary"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-queue/summary"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBackupQueueCancelQueuedBackup(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("cancels entry", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/backup-queue/"+uuid.New().String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/backup-queue/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBackupQueueGetOrgConcurrency(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns concurrency", func(t *testing.T) {
		store := &mockBackupQueueStore{org: &models.Organization{ID: orgID}}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/organizations/"+orgID.String()+"/concurrency"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/organizations/not-a-uuid/concurrency"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBackupQueueUpdateOrgConcurrency(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("updates limit", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/concurrency", `{"max_concurrent_backups":10}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("negative limit returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/concurrency", `{"max_concurrent_backups":-1}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBackupQueueGetAgentConcurrency(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	agentID := uuid.New()

	t.Run("returns concurrency", func(t *testing.T) {
		store := &mockBackupQueueStore{agent: &models.Agent{ID: agentID, OrgID: orgID}}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/concurrency"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockBackupQueueStore{}
		r := setupBackupQueueTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/not-a-uuid/concurrency"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
