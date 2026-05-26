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

type mockWebhooksStore struct {
	endpoints  []*models.WebhookEndpoint
	deliveries []*models.WebhookDelivery
	err        error
}

func (m *mockWebhooksStore) GetWebhookEndpointsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.WebhookEndpoint, error) {
	return m.endpoints, m.err
}

func (m *mockWebhooksStore) GetWebhookEndpointByID(_ context.Context, id uuid.UUID) (*models.WebhookEndpoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.endpoints {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (m *mockWebhooksStore) GetEnabledWebhookEndpointsForEvent(_ context.Context, _ uuid.UUID, _ models.WebhookEventType) ([]*models.WebhookEndpoint, error) {
	return m.endpoints, m.err
}

func (m *mockWebhooksStore) CreateWebhookEndpoint(_ context.Context, e *models.WebhookEndpoint) error {
	if m.err != nil {
		return m.err
	}
	m.endpoints = append(m.endpoints, e)
	return nil
}

func (m *mockWebhooksStore) UpdateWebhookEndpoint(_ context.Context, _ *models.WebhookEndpoint) error {
	return m.err
}

func (m *mockWebhooksStore) DeleteWebhookEndpoint(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockWebhooksStore) GetWebhookDeliveriesByOrgID(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.WebhookDelivery, int, error) {
	return m.deliveries, len(m.deliveries), m.err
}

func (m *mockWebhooksStore) GetWebhookDeliveriesByEndpointID(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.WebhookDelivery, int, error) {
	return m.deliveries, len(m.deliveries), m.err
}

func (m *mockWebhooksStore) GetWebhookDeliveryByID(_ context.Context, _ uuid.UUID) (*models.WebhookDelivery, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.deliveries) > 0 {
		return m.deliveries[0], nil
	}
	return nil, nil
}

func (m *mockWebhooksStore) CreateWebhookDelivery(_ context.Context, _ *models.WebhookDelivery) error {
	return m.err
}

func (m *mockWebhooksStore) UpdateWebhookDelivery(_ context.Context, _ *models.WebhookDelivery) error {
	return m.err
}

func (m *mockWebhooksStore) GetPendingWebhookDeliveries(_ context.Context, _ int) ([]*models.WebhookDelivery, error) {
	return m.deliveries, m.err
}

func newTestKeyManager(t *testing.T) *crypto.KeyManager {
	t.Helper()
	km, err := crypto.NewKeyManager(make([]byte, 32))
	if err != nil {
		t.Fatalf("new key manager: %v", err)
	}
	return km
}

func setupWebhooksTestRouter(store WebhooksStore, user *auth.SessionUser, km *crypto.KeyManager) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewWebhooksHandler(store, km, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestWebhooksListEventTypes(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockWebhooksStore{}
	r := setupWebhooksTestRouter(store, user, newTestKeyManager(t))

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/webhooks/event-types"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestWebhooksListEndpoints(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	km := newTestKeyManager(t)

	t.Run("returns endpoints", func(t *testing.T) {
		store := &mockWebhooksStore{endpoints: []*models.WebhookEndpoint{{ID: uuid.New(), OrgID: orgID, URL: "https://example.com/hook"}}}
		r := setupWebhooksTestRouter(store, user, km)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/webhooks/endpoints"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockWebhooksStore{}
		r := setupWebhooksTestRouter(store, testUserNoOrg(), km)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/webhooks/endpoints"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestWebhooksListDeliveries(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	km := newTestKeyManager(t)

	t.Run("returns paginated deliveries", func(t *testing.T) {
		store := &mockWebhooksStore{deliveries: []*models.WebhookDelivery{{ID: uuid.New(), OrgID: orgID}}}
		r := setupWebhooksTestRouter(store, user, km)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/webhooks/deliveries"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockWebhooksStore{}
		r := setupWebhooksTestRouter(store, testUserNoOrg(), km)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/webhooks/deliveries"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
