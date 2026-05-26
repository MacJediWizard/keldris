package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/logs"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockServerLogStore struct {
	membership *models.OrgMembership
	err        error
}

func (m *mockServerLogStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.membership != nil {
		return m.membership, nil
	}
	return &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin}, nil
}

func setupServerLogsTestRouter(store ServerLogStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	buffer := logs.NewLogBuffer(logs.DefaultConfig())
	handler := NewServerLogsHandler(store, buffer, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestServerLogsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockServerLogStore{}
	r := setupServerLogsTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/logs"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestServerLogsListComponents(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockServerLogStore{}
	r := setupServerLogsTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/logs/components"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestServerLogsNonAdminForbidden(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockServerLogStore{
		membership: &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleMember},
	}
	r := setupServerLogsTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/logs"))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}
