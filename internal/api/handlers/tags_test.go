package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockTagStore struct {
	tags         []*models.Tag
	tagByID      map[uuid.UUID]*models.Tag
	backupTags   []*models.Tag
	backup       *models.Backup
	agent        *models.Agent
	createErr    error
	updateErr    error
	deleteErr    error
	setTagsErr   error
}

func (m *mockTagStore) GetTagsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Tag, error) {
	var result []*models.Tag
	for _, t := range m.tags {
		if t.OrgID == orgID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTagStore) GetTagByID(_ context.Context, id uuid.UUID) (*models.Tag, error) {
	if t, ok := m.tagByID[id]; ok {
		return t, nil
	}
	return nil, errors.New("tag not found")
}

func (m *mockTagStore) CreateTag(_ context.Context, _ *models.Tag) error {
	return m.createErr
}

func (m *mockTagStore) UpdateTag(_ context.Context, _ *models.Tag) error {
	return m.updateErr
}

func (m *mockTagStore) DeleteTag(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockTagStore) GetTagsByBackupID(_ context.Context, _ uuid.UUID) ([]*models.Tag, error) {
	return m.backupTags, nil
}

func (m *mockTagStore) SetBackupTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return m.setTagsErr
}

func (m *mockTagStore) GetBackupByID(_ context.Context, id uuid.UUID) (*models.Backup, error) {
	if m.backup != nil && m.backup.ID == id {
		return m.backup, nil
	}
	return nil, errors.New("backup not found")
}

func (m *mockTagStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.agent != nil && m.agent.ID == id {
		return m.agent, nil
	}
	return nil, errors.New("agent not found")
}

func setupTagTestRouter(store TagStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewTagsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListTags(t *testing.T) {
	orgID := uuid.New()
	tagID := uuid.New()
	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "production", Color: "#ff0000"}
	store := &mockTagStore{
		tags:    []*models.Tag{tag},
		tagByID: map[uuid.UUID]*models.Tag{tagID: tag},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["tags"]; !ok {
			t.Fatal("expected 'tags' key in response")
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupTagTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupTagTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestGetTag(t *testing.T) {
	orgID := uuid.New()
	tagID := uuid.New()
	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "production"}
	store := &mockTagStore{
		tagByID: map[uuid.UUID]*models.Tag{tagID: tag},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags/"+tagID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags/not-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupTagTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/tags/"+tagID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestCreateTag(t *testing.T) {
	orgID := uuid.New()
	store := &mockTagStore{tagByID: map[uuid.UUID]*models.Tag{}}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"staging","color":"#00ff00"}`
		req, _ := http.NewRequest("POST", "/api/v1/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"color":"#00ff00"}`
		req, _ := http.NewRequest("POST", "/api/v1/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockTagStore{tagByID: map[uuid.UUID]*models.Tag{}, createErr: errors.New("db error")}
		r := setupTagTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail"}`
		req, _ := http.NewRequest("POST", "/api/v1/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestUpdateTag(t *testing.T) {
	orgID := uuid.New()
	tagID := uuid.New()
	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "production"}
	store := &mockTagStore{
		tagByID: map[uuid.UUID]*models.Tag{tagID: tag},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"staging"}`
		req, _ := http.NewRequest("PUT", "/api/v1/tags/"+tagID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"staging"}`
		req, _ := http.NewRequest("PUT", "/api/v1/tags/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupTagTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		body := `{"name":"staging"}`
		req, _ := http.NewRequest("PUT", "/api/v1/tags/"+tagID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockTagStore{
			tagByID:   map[uuid.UUID]*models.Tag{tagID: tag},
			updateErr: errors.New("db error"),
		}
		r := setupTagTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"staging"}`
		req, _ := http.NewRequest("PUT", "/api/v1/tags/"+tagID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDeleteTag(t *testing.T) {
	orgID := uuid.New()
	tagID := uuid.New()
	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "production"}
	store := &mockTagStore{
		tagByID: map[uuid.UUID]*models.Tag{tagID: tag},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/tags/"+tagID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/tags/bad-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/tags/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupTagTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/tags/"+tagID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockTagStore{
			tagByID:   map[uuid.UUID]*models.Tag{tagID: tag},
			deleteErr: errors.New("db error"),
		}
		r := setupTagTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/tags/"+tagID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetBackupTags(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	backupID := uuid.New()
	tagID := uuid.New()

	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "important"}
	backup := &models.Backup{ID: backupID, AgentID: agentID}
	agent := &models.Agent{ID: agentID, OrgID: orgID}

	store := &mockTagStore{
		tagByID:    map[uuid.UUID]*models.Tag{tagID: tag},
		backupTags: []*models.Tag{tag},
		backup:     backup,
		agent:      agent,
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+backupID.String()+"/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("backup not found", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+uuid.New().String()+"/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupTagTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/backups/"+backupID.String()+"/tags", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestSetBackupTags(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	backupID := uuid.New()
	tagID := uuid.New()

	tag := &models.Tag{ID: tagID, OrgID: orgID, Name: "important"}
	backup := &models.Backup{ID: backupID, AgentID: agentID}
	agent := &models.Agent{ID: agentID, OrgID: orgID}

	store := &mockTagStore{
		tagByID:    map[uuid.UUID]*models.Tag{tagID: tag},
		backupTags: []*models.Tag{tag},
		backup:     backup,
		agent:      agent,
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"tag_ids":["` + tagID.String() + `"]}`
		req, _ := http.NewRequest("POST", "/api/v1/backups/"+backupID.String()+"/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid tag id in request", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		// Tag that doesn't exist in our org
		body := `{"tag_ids":["` + uuid.New().String() + `"]}`
		req, _ := http.NewRequest("POST", "/api/v1/backups/"+backupID.String()+"/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("backup not found", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"tag_ids":["` + tagID.String() + `"]}`
		req, _ := http.NewRequest("POST", "/api/v1/backups/"+uuid.New().String()+"/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("missing body", func(t *testing.T) {
		r := setupTagTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/backups/"+backupID.String()+"/tags", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}
