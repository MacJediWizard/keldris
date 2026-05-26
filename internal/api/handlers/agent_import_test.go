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

type mockAgentImportStore struct {
	agents     []*models.Agent
	groups     []*models.AgentGroup
	regCode    *models.RegistrationCode
	agentsErr  error
	groupsErr  error
	regCodeErr error
}

func (m *mockAgentImportStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	if m.agentsErr != nil {
		return nil, m.agentsErr
	}
	return m.agents, nil
}

func (m *mockAgentImportStore) GetAgentGroupsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.AgentGroup, error) {
	if m.groupsErr != nil {
		return nil, m.groupsErr
	}
	return m.groups, nil
}

func (m *mockAgentImportStore) CreateAgentGroup(_ context.Context, _ *models.AgentGroup) error {
	return nil
}

func (m *mockAgentImportStore) CreateAgent(_ context.Context, _ *models.Agent) error {
	return nil
}

func (m *mockAgentImportStore) AddAgentToGroup(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockAgentImportStore) CreateRegistrationCode(_ context.Context, _ *models.RegistrationCode) error {
	return nil
}

func (m *mockAgentImportStore) GetRegistrationCodeByCode(_ context.Context, _ uuid.UUID, _ string) (*models.RegistrationCode, error) {
	if m.regCodeErr != nil {
		return nil, m.regCodeErr
	}
	return m.regCode, nil
}

func (m *mockAgentImportStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func setupAgentImportTestRouter(store AgentImportStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAgentImportHandler(store, "https://test.example.com", zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAgentImportTemplate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns JSON template", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/import/template"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns CSV template", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/import/template?format=csv"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		if ct := resp.Header().Get("Content-Type"); ct != "text/csv" {
			t.Errorf("expected text/csv, got %s", ct)
		}
	})
}

func TestAgentImportPreviewNoFile(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("missing file returns 400", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agents/import/preview", ""))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agents/import/preview", ""))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAgentImportExportTokens(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("missing results param returns 400", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/import/tokens/export"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid results format returns 400", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/import/tokens/export?results=not-json"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("valid empty results returns CSV", func(t *testing.T) {
		store := &mockAgentImportStore{}
		r := setupAgentImportTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/import/tokens/export?results=%5B%5D"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}
