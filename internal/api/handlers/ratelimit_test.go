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

type mockRateLimitStore struct {
	membership *models.OrgMembership
	err        error
}

func (m *mockRateLimitStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.membership != nil {
		return m.membership, nil
	}
	return &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin}, nil
}

func setupRateLimitTestRouter(store RateLimitStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewRateLimitHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestRateLimitGetDashboardStats(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin without manager returns 500", func(t *testing.T) {
		store := &mockRateLimitStore{}
		r := setupRateLimitTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/rate-limits"))
		// Without GlobalRateLimitManager initialized, handler returns 500
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (no manager), got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		store := &mockRateLimitStore{
			membership: &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleMember},
		}
		r := setupRateLimitTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/rate-limits"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}
