package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDatabaseConnectionStore struct {
	conn  *models.DatabaseConnection
	conns []*models.DatabaseConnection
	err   error
}

func (m *mockDatabaseConnectionStore) GetDatabaseConnectionsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.DatabaseConnection, error) {
	return m.conns, m.err
}

func (m *mockDatabaseConnectionStore) GetDatabaseConnectionsByAgentID(_ context.Context, _ uuid.UUID) ([]*models.DatabaseConnection, error) {
	return m.conns, m.err
}

func (m *mockDatabaseConnectionStore) GetDatabaseConnectionByID(_ context.Context, _ uuid.UUID) (*models.DatabaseConnection, error) {
	return m.conn, m.err
}

func (m *mockDatabaseConnectionStore) CreateDatabaseConnection(_ context.Context, _ *models.DatabaseConnection) error {
	return m.err
}

func (m *mockDatabaseConnectionStore) UpdateDatabaseConnection(_ context.Context, _ *models.DatabaseConnection) error {
	return m.err
}

func (m *mockDatabaseConnectionStore) UpdateDatabaseConnectionCredentials(_ context.Context, _ uuid.UUID, _ []byte) error {
	return m.err
}

func (m *mockDatabaseConnectionStore) DeleteDatabaseConnection(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockDatabaseConnectionStore) UpdateDatabaseConnectionHealth(_ context.Context, _ uuid.UUID, _ models.DatabaseConnectionHealthStatus, _ *string, _ *string) error {
	return m.err
}

func setupDatabaseConnectionsTestRouter(store DatabaseConnectionStore, user *auth.SessionUser, km *crypto.KeyManager) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDatabaseConnectionsHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDatabaseConnectionsListConnections(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)
	store := &mockDatabaseConnectionStore{conns: []*models.DatabaseConnection{}}
	r := setupDatabaseConnectionsTestRouter(store, user, km)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/database-connections"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDatabaseConnectionsListTypes(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)
	store := &mockDatabaseConnectionStore{}
	r := setupDatabaseConnectionsTestRouter(store, user, km)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/database-connections/types"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDatabaseConnectionsGetConnection(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockDatabaseConnectionStore{}
		r := setupDatabaseConnectionsTestRouter(store, user, km)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/database-connections/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
