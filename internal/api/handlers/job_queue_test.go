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

type mockJobQueueStore struct {
	job     *models.Job
	jobs    []*models.JobWithDetails
	summary *models.JobQueueSummary
	user    *models.User
	err     error
}

func (m *mockJobQueueStore) GetJobByID(_ context.Context, _ uuid.UUID) (*models.Job, error) {
	return m.job, m.err
}

func (m *mockJobQueueStore) GetJobsWithDetails(_ context.Context, _ uuid.UUID, _ *models.JobStatus, _ int) ([]*models.JobWithDetails, error) {
	return m.jobs, m.err
}

func (m *mockJobQueueStore) GetJobQueueSummary(_ context.Context, _ uuid.UUID) (*models.JobQueueSummary, error) {
	return m.summary, m.err
}

func (m *mockJobQueueStore) UpdateJob(_ context.Context, _ *models.Job) error {
	return m.err
}

func (m *mockJobQueueStore) DeleteJob(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockJobQueueStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

type mockMembershipStore struct {
	membership  *models.OrgMembership
	memberships []*models.OrgMembership
}

func (m *mockMembershipStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	if m.membership != nil {
		return m.membership, nil
	}
	return &models.OrgMembership{UserID: userID, OrgID: orgID, Role: "admin"}, nil
}

func (m *mockMembershipStore) GetMembershipsByUserID(_ context.Context, userID uuid.UUID) ([]*models.OrgMembership, error) {
	if m.memberships != nil {
		return m.memberships, nil
	}
	return []*models.OrgMembership{{UserID: userID, Role: "admin"}}, nil
}

func setupJobQueueTestRouter(store JobQueueStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	rbac := auth.NewRBAC(&mockMembershipStore{})
	handler := NewJobQueueHandler(store, rbac, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestJobQueueListJobs(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockJobQueueStore{jobs: []*models.JobWithDetails{}}
	r := setupJobQueueTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/job-queue"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestJobQueueGetSummary(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockJobQueueStore{summary: &models.JobQueueSummary{}}
	r := setupJobQueueTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/job-queue/summary"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestJobQueueGetJob(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockJobQueueStore{}
		r := setupJobQueueTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/job-queue/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
