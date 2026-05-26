package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/metadata"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockMetadataStore struct {
	schemas    []*metadata.Schema
	schema     *metadata.Schema
	agent      *models.Agent
	repository *models.Repository
	schedule   *models.Schedule
	ids        []uuid.UUID

	listErr     error
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	updateMDErr error
	searchErr   error
	entityErr   error
}

func (m *mockMetadataStore) GetMetadataSchemasByOrgAndEntity(_ context.Context, _ uuid.UUID, _ metadata.EntityType) ([]*metadata.Schema, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.schemas, nil
}

func (m *mockMetadataStore) GetMetadataSchemaByID(_ context.Context, _ uuid.UUID) (*metadata.Schema, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.schema, nil
}

func (m *mockMetadataStore) CreateMetadataSchema(_ context.Context, s *metadata.Schema) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.schema = s
	return nil
}

func (m *mockMetadataStore) UpdateMetadataSchema(_ context.Context, _ *metadata.Schema) error {
	return m.updateErr
}

func (m *mockMetadataStore) DeleteMetadataSchema(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockMetadataStore) UpdateAgentMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return m.updateMDErr
}

func (m *mockMetadataStore) UpdateRepositoryMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return m.updateMDErr
}

func (m *mockMetadataStore) UpdateScheduleMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return m.updateMDErr
}

func (m *mockMetadataStore) SearchAgentsByMetadata(_ context.Context, _ uuid.UUID, _, _ string) ([]uuid.UUID, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.ids, nil
}

func (m *mockMetadataStore) SearchRepositoriesByMetadata(_ context.Context, _ uuid.UUID, _, _ string) ([]uuid.UUID, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.ids, nil
}

func (m *mockMetadataStore) SearchSchedulesByMetadata(_ context.Context, _ uuid.UUID, _, _ string) ([]uuid.UUID, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.ids, nil
}

func (m *mockMetadataStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.agent, nil
}

func (m *mockMetadataStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.repository, nil
}

func (m *mockMetadataStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.schedule, nil
}

func setupMetadataTestRouter(store MetadataStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewMetadataHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestMetadataListSchemas(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns schemas", func(t *testing.T) {
		store := &mockMetadataStore{schemas: []*metadata.Schema{{ID: uuid.New(), OrgID: orgID}}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas?entity_type=agent"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing entity_type returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid entity_type returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas?entity_type=bogus"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestMetadataGetSchema(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("returns own schema", func(t *testing.T) {
		store := &mockMetadataStore{schema: &metadata.Schema{ID: id, OrgID: orgID}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org schema returns 404", func(t *testing.T) {
		store := &mockMetadataStore{schema: &metadata.Schema{ID: id, OrgID: uuid.New()}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestMetadataCreateSchema(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates schema", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		body := `{"entity_type":"agent","name":"Department","field_key":"department","field_type":"text"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/metadata/schemas", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid entity_type returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		body := `{"entity_type":"bogus","name":"x","field_key":"x","field_type":"text"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/metadata/schemas", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid field_type returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		body := `{"entity_type":"agent","name":"x","field_key":"x","field_type":"bogus"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/metadata/schemas", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestMetadataDeleteSchema(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes own schema", func(t *testing.T) {
		store := &mockMetadataStore{schema: &metadata.Schema{ID: id, OrgID: orgID}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/metadata/schemas/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org schema returns 404", func(t *testing.T) {
		store := &mockMetadataStore{schema: &metadata.Schema{ID: id, OrgID: uuid.New()}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/metadata/schemas/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("delete store error returns 500", func(t *testing.T) {
		store := &mockMetadataStore{schema: &metadata.Schema{ID: id, OrgID: orgID}, deleteErr: errors.New("db down")}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/metadata/schemas/"+id.String()))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestMetadataListFieldAndEntityTypes(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("lists field types", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas/types"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("lists entity types", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/schemas/entities"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}

func TestMetadataUpdateAgentMetadata(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("updates agent metadata", func(t *testing.T) {
		store := &mockMetadataStore{agent: &models.Agent{ID: id, OrgID: orgID}}
		r := setupMetadataTestRouter(store, user)
		body := `{"metadata":{"department":"engineering"}}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agents/"+id.String()+"/metadata", body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("agent not found returns 404", func(t *testing.T) {
		store := &mockMetadataStore{entityErr: errors.New("not found")}
		r := setupMetadataTestRouter(store, user)
		body := `{"metadata":{"department":"engineering"}}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agents/"+id.String()+"/metadata", body))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		body := `{"metadata":{}}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agents/not-a-uuid/metadata", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestMetadataSearchByMetadata(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("searches agents", func(t *testing.T) {
		store := &mockMetadataStore{ids: []uuid.UUID{uuid.New()}}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/search?entity_type=agent&key=department&value=engineering"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing params returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/search?entity_type=agent"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid entity_type returns 400", func(t *testing.T) {
		store := &mockMetadataStore{}
		r := setupMetadataTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/metadata/search?entity_type=bogus&key=x&value=y"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
