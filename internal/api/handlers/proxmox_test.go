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

type mockProxmoxStore struct {
	conn  *models.ProxmoxConnection
	conns []*models.ProxmoxConnection
	err   error
}

func (m *mockProxmoxStore) CreateProxmoxConnection(_ context.Context, c *models.ProxmoxConnection) error {
	if m.err != nil {
		return m.err
	}
	m.conn = c
	return nil
}

func (m *mockProxmoxStore) GetProxmoxConnectionByID(_ context.Context, _ uuid.UUID) (*models.ProxmoxConnection, error) {
	return m.conn, m.err
}

func (m *mockProxmoxStore) GetProxmoxConnectionsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.ProxmoxConnection, error) {
	return m.conns, m.err
}

func (m *mockProxmoxStore) UpdateProxmoxConnection(_ context.Context, _ *models.ProxmoxConnection) error {
	return m.err
}

func (m *mockProxmoxStore) DeleteProxmoxConnection(_ context.Context, _ uuid.UUID) error {
	return m.err
}

type noopEncryption struct{}

func (n *noopEncryption) Encrypt(p []byte) ([]byte, error) { return p, nil }
func (n *noopEncryption) Decrypt(c []byte) ([]byte, error) { return c, nil }

func setupProxmoxTestRouter(store ProxmoxStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewProxmoxHandler(store, &noopEncryption{}, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestProxmoxListConnections(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockProxmoxStore{conns: []*models.ProxmoxConnection{}}
	r := setupProxmoxTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/proxmox/connections"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestProxmoxGetConnection(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockProxmoxStore{}
		r := setupProxmoxTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/proxmox/connections/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestProxmoxCreateConnection(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid json returns 400", func(t *testing.T) {
		store := &mockProxmoxStore{}
		r := setupProxmoxTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/proxmox/connections", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
