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

type mockDockerBackupStore struct {
	containers    []models.DockerContainer
	volumes       []models.DockerVolume
	daemonStatus  *models.DockerDaemonStatus
	backupResp    *models.DockerBackupResult
	containersErr error
	volumesErr    error
	statusErr     error
	backupErr     error
}

func (m *mockDockerBackupStore) GetDockerContainers(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.DockerContainer, error) {
	if m.containersErr != nil {
		return nil, m.containersErr
	}
	return m.containers, nil
}

func (m *mockDockerBackupStore) GetDockerVolumes(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.DockerVolume, error) {
	if m.volumesErr != nil {
		return nil, m.volumesErr
	}
	return m.volumes, nil
}

func (m *mockDockerBackupStore) GetDockerDaemonStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.DockerDaemonStatus, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.daemonStatus, nil
}

func (m *mockDockerBackupStore) CreateDockerBackup(_ context.Context, _ uuid.UUID, _ *models.DockerBackupParams) (*models.DockerBackupResult, error) {
	if m.backupErr != nil {
		return nil, m.backupErr
	}
	return m.backupResp, nil
}

func setupDockerBackupTestRouter(store DockerBackupStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})

	handler := NewDockerBackupHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListDockerContainers(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	store := &mockDockerBackupStore{
		containers: []models.DockerContainer{
			{ID: "c1", Name: "app", Image: "nginx:latest", Status: "Up", State: "running"},
		},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["containers"]; !ok {
			t.Fatal("expected 'containers' key in response")
		}
	})

	t.Run("missing agent_id", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers?agent_id=invalid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupDockerBackupTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockDockerBackupStore{containersErr: errors.New("db error")}
		r := setupDockerBackupTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/containers?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestListDockerVolumes(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	store := &mockDockerBackupStore{
		volumes: []models.DockerVolume{
			{Name: "data", Driver: "local", SizeBytes: 1024},
		},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/volumes?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["volumes"]; !ok {
			t.Fatal("expected 'volumes' key in response")
		}
	})

	t.Run("missing agent_id", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/volumes", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/volumes?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockDockerBackupStore{volumesErr: errors.New("db error")}
		r := setupDockerBackupTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/volumes?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestDockerDaemonStatus(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	store := &mockDockerBackupStore{
		daemonStatus: &models.DockerDaemonStatus{
			Available:      true,
			Version:        "24.0.7",
			ContainerCount: 5,
			VolumeCount:    3,
		},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/status?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DockerDaemonStatus
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.Available {
			t.Fatal("expected daemon to be available")
		}
	})

	t.Run("missing agent_id", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/status", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/status?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockDockerBackupStore{statusErr: errors.New("connection refused")}
		r := setupDockerBackupTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/docker/status?agent_id="+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestTriggerDockerBackup(t *testing.T) {
	orgID := uuid.New()

	store := &mockDockerBackupStore{
		backupResp: &models.DockerBackupResult{
			ID:        uuid.New().String(),
			Status:    "queued",
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_id":"` + uuid.New().String() + `","repository_id":"` + uuid.New().String() + `","container_ids":["c1"],"volume_names":["v1"]}`
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d: %s", w.Code, w.Body.String())
		}

		var resp models.DockerBackupResult
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "queued" {
			t.Fatalf("expected status 'queued', got '%s'", resp.Status)
		}
	})

	t.Run("no items selected", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"agent_id":"` + uuid.New().String() + `","repository_id":"` + uuid.New().String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupDockerBackupTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"agent_id":"` + uuid.New().String() + `","repository_id":"` + uuid.New().String() + `","container_ids":["c1"]}`
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupDockerBackupTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"agent_id":"` + uuid.New().String() + `","repository_id":"` + uuid.New().String() + `","container_ids":["c1"]}`
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockDockerBackupStore{backupErr: errors.New("backup failed")}
		r := setupDockerBackupTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"agent_id":"` + uuid.New().String() + `","repository_id":"` + uuid.New().String() + `","container_ids":["c1"]}`
		req, _ := http.NewRequest("POST", "/api/v1/docker/backup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}
