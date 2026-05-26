package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDockerRegistryStore struct {
	getErr error
}

func (m *mockDockerRegistryStore) GetDockerRegistriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.DockerRegistry, error) {
	return nil, nil
}

func (m *mockDockerRegistryStore) GetDockerRegistryByID(_ context.Context, _ uuid.UUID) (*models.DockerRegistry, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return nil, errors.New("not found")
}

func (m *mockDockerRegistryStore) GetDefaultDockerRegistry(_ context.Context, _ uuid.UUID) (*models.DockerRegistry, error) {
	return nil, nil
}

func (m *mockDockerRegistryStore) CreateDockerRegistry(_ context.Context, _ *models.DockerRegistry) error {
	return nil
}

func (m *mockDockerRegistryStore) UpdateDockerRegistry(_ context.Context, _ *models.DockerRegistry) error {
	return nil
}

func (m *mockDockerRegistryStore) DeleteDockerRegistry(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDockerRegistryStore) UpdateDockerRegistryHealth(_ context.Context, _ uuid.UUID, _ models.DockerRegistryHealthStatus, _ *string) error {
	return nil
}

func (m *mockDockerRegistryStore) UpdateDockerRegistryCredentials(_ context.Context, _ uuid.UUID, _ []byte, _ *time.Time) error {
	return nil
}

func (m *mockDockerRegistryStore) SetDefaultDockerRegistry(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockDockerRegistryStore) GetDockerRegistriesWithExpiringCredentials(_ context.Context, _ uuid.UUID, _ time.Time) ([]*models.DockerRegistry, error) {
	return nil, nil
}

func (m *mockDockerRegistryStore) CreateDockerRegistryAuditLog(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID, _ string, _ map[string]interface{}, _, _ string) error {
	return nil
}

func setupDockerRegistriesTestRouter(store DockerRegistryStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	// Provide a real KeyManager - constructor only fails on bad master key length.
	masterKey := make([]byte, 32)
	km, err := crypto.NewKeyManager(masterKey)
	if err != nil {
		panic(err)
	}
	handler := NewDockerRegistriesHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDockerRegistriesListRegistries(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns empty list", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestDockerRegistriesListRegistryTypes(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns built-in types", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries/types"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["types"]; !ok {
			t.Fatal("expected 'types' key in response")
		}
	})
}

func TestDockerRegistriesGetRegistry(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestDockerRegistriesListExpiringCredentials(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns empty", func(t *testing.T) {
		store := &mockDockerRegistryStore{}
		r := setupDockerRegistriesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-registries/expiring"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}
