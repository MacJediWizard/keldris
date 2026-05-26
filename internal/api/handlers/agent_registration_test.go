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

type mockRegistrationCodeStore struct {
	codes       []*models.RegistrationCode
	codeByValue *models.RegistrationCode
	pending     []*models.RegistrationCode
	pendingWith []*models.PendingRegistration
	err         error
	createErr   error
}

func (m *mockRegistrationCodeStore) CreateRegistrationCode(_ context.Context, c *models.RegistrationCode) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.codes = append(m.codes, c)
	return nil
}

func (m *mockRegistrationCodeStore) GetRegistrationCodeByCode(_ context.Context, _ uuid.UUID, _ string) (*models.RegistrationCode, error) {
	return m.codeByValue, m.err
}

func (m *mockRegistrationCodeStore) GetPendingRegistrationCodes(_ context.Context, _ uuid.UUID) ([]*models.RegistrationCode, error) {
	return m.pending, m.err
}

func (m *mockRegistrationCodeStore) GetPendingRegistrationsWithCreator(_ context.Context, _ uuid.UUID) ([]*models.PendingRegistration, error) {
	return m.pendingWith, m.err
}

func (m *mockRegistrationCodeStore) MarkRegistrationCodeUsed(_ context.Context, _, _ uuid.UUID) error {
	return m.err
}

func (m *mockRegistrationCodeStore) DeleteExpiredRegistrationCodes(_ context.Context) error {
	return m.err
}

func (m *mockRegistrationCodeStore) DeleteRegistrationCode(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockRegistrationCodeStore) CreateAgent(_ context.Context, _ *models.Agent) error {
	return m.err
}

func (m *mockRegistrationCodeStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func setupAgentRegistrationTestRouter(store RegistrationCodeStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAgentRegistrationHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAgentRegistrationCreateCode(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates code", func(t *testing.T) {
		store := &mockRegistrationCodeStore{}
		r := setupAgentRegistrationTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-registration-codes", `{}`))
		if resp.Code != http.StatusOK && resp.Code != http.StatusCreated {
			t.Fatalf("expected 200/201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockRegistrationCodeStore{}
		r := setupAgentRegistrationTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-registration-codes", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAgentRegistrationListPendingCodes(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns pending registrations", func(t *testing.T) {
		store := &mockRegistrationCodeStore{pendingWith: []*models.PendingRegistration{}}
		r := setupAgentRegistrationTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-registration-codes"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockRegistrationCodeStore{}
		r := setupAgentRegistrationTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-registration-codes"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAgentRegistrationDeleteCode(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockRegistrationCodeStore{}
		r := setupAgentRegistrationTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-registration-codes/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
