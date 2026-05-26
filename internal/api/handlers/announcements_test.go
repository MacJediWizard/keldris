package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockAnnouncementStore struct {
	announcement *models.Announcement
	list         []*models.Announcement
	getErr       error
	listErr      error
	createErr    error
	updateErr    error
	deleteErr    error
	dismissErr   error
}

func (m *mockAnnouncementStore) GetAnnouncementByID(_ context.Context, _ uuid.UUID) (*models.Announcement, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.announcement, nil
}

func (m *mockAnnouncementStore) ListAnnouncementsByOrg(_ context.Context, _ uuid.UUID) ([]*models.Announcement, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.list, nil
}

func (m *mockAnnouncementStore) ListActiveAnnouncements(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) ([]*models.Announcement, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.list, nil
}

func (m *mockAnnouncementStore) CreateAnnouncement(_ context.Context, a *models.Announcement) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.announcement = a
	return nil
}

func (m *mockAnnouncementStore) UpdateAnnouncement(_ context.Context, a *models.Announcement) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.announcement = a
	return nil
}

func (m *mockAnnouncementStore) DeleteAnnouncement(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockAnnouncementStore) CreateAnnouncementDismissal(_ context.Context, _ *models.AnnouncementDismissal) error {
	return m.dismissErr
}

func setupAnnouncementsTestRouter(store AnnouncementStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAnnouncementsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAnnouncementsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns announcements for admin", func(t *testing.T) {
		ann := models.NewAnnouncement(orgID, "Welcome", models.AnnouncementTypeInfo)
		store := &mockAnnouncementStore{list: []*models.Announcement{ann}}
		r := setupAnnouncementsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result models.AnnouncementsResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(result.Announcements) != 1 {
			t.Fatalf("expected 1 announcement, got %d", len(result.Announcements))
		}
		if result.Announcements[0].Title != "Welcome" {
			t.Errorf("expected title Welcome, got %s", result.Announcements[0].Title)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, nonAdmin)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockAnnouncementStore{listErr: errors.New("db down")}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsGetActive(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns active list", func(t *testing.T) {
		store := &mockAnnouncementStore{list: []*models.Announcement{models.NewAnnouncement(orgID, "Hi", models.AnnouncementTypeInfo)}}
		r := setupAnnouncementsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements/active"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements/active"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	annID := uuid.New()

	t.Run("returns announcement", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: orgID, Title: "X", Type: models.AnnouncementTypeInfo}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements/"+annID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("wrong org returns 404", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: uuid.New(), Title: "X", Type: models.AnnouncementTypeInfo}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/announcements/"+annID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates successfully", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, user)
		body := `{"title":"New","type":"info"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/announcements", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, nonAdmin)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/announcements", `{"title":"x","type":"info"}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/announcements", `{"type":"info"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("end before start returns 400", func(t *testing.T) {
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, user)
		body := `{"title":"x","type":"info","starts_at":"2030-01-02T00:00:00Z","ends_at":"2030-01-01T00:00:00Z"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/announcements", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	annID := uuid.New()

	t.Run("updates successfully", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: orgID, Title: "Old", Type: models.AnnouncementTypeInfo}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/announcements/"+annID.String(), `{"title":"New"}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, nonAdmin)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/announcements/"+annID.String(), `{}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	annID := uuid.New()

	t.Run("deletes successfully", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: orgID, Title: "X", Type: models.AnnouncementTypeInfo}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/announcements/"+annID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockAnnouncementStore{}
		r := setupAnnouncementsTestRouter(store, nonAdmin)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/announcements/"+annID.String()))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestAnnouncementsDismiss(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	annID := uuid.New()

	t.Run("dismisses dismissible announcement", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: orgID, Title: "X", Type: models.AnnouncementTypeInfo, Dismissible: true}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("POST", "/api/v1/announcements/"+annID.String()+"/dismiss"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-dismissible returns 400", func(t *testing.T) {
		ann := &models.Announcement{ID: annID, OrgID: orgID, Title: "X", Type: models.AnnouncementTypeInfo, Dismissible: false}
		store := &mockAnnouncementStore{announcement: ann}
		r := setupAnnouncementsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("POST", "/api/v1/announcements/"+annID.String()+"/dismiss"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
