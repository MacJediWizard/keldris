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
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockNotificationStore struct {
	channels    []*models.NotificationChannel
	channelByID map[uuid.UUID]*models.NotificationChannel
	preferences []*models.NotificationPreference
	logs        []*models.NotificationLog
	user        *models.User
	createChanErr error
	updateChanErr error
	deleteChanErr error
	createPrefErr error
	updatePrefErr error
	deletePrefErr error
	listChanErr   error
	listPrefErr   error
	listLogErr    error
}

func (m *mockNotificationStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockNotificationStore) GetNotificationChannelsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.NotificationChannel, error) {
	if m.listChanErr != nil {
		return nil, m.listChanErr
	}
	var result []*models.NotificationChannel
	for _, ch := range m.channels {
		if ch.OrgID == orgID {
			result = append(result, ch)
		}
	}
	return result, nil
}

func (m *mockNotificationStore) GetNotificationChannelByID(_ context.Context, id uuid.UUID) (*models.NotificationChannel, error) {
	if ch, ok := m.channelByID[id]; ok {
		return ch, nil
	}
	return nil, errors.New("channel not found")
}

func (m *mockNotificationStore) CreateNotificationChannel(_ context.Context, _ *models.NotificationChannel) error {
	return m.createChanErr
}

func (m *mockNotificationStore) UpdateNotificationChannel(_ context.Context, _ *models.NotificationChannel) error {
	return m.updateChanErr
}

func (m *mockNotificationStore) DeleteNotificationChannel(_ context.Context, _ uuid.UUID) error {
	return m.deleteChanErr
}

func (m *mockNotificationStore) GetNotificationPreferencesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.NotificationPreference, error) {
	if m.listPrefErr != nil {
		return nil, m.listPrefErr
	}
	var result []*models.NotificationPreference
	for _, p := range m.preferences {
		if p.OrgID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockNotificationStore) GetNotificationPreferencesByChannelID(_ context.Context, channelID uuid.UUID) ([]*models.NotificationPreference, error) {
	var result []*models.NotificationPreference
	for _, p := range m.preferences {
		if p.ChannelID == channelID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockNotificationStore) CreateNotificationPreference(_ context.Context, _ *models.NotificationPreference) error {
	return m.createPrefErr
}

func (m *mockNotificationStore) UpdateNotificationPreference(_ context.Context, _ *models.NotificationPreference) error {
	return m.updatePrefErr
}

func (m *mockNotificationStore) DeleteNotificationPreference(_ context.Context, _ uuid.UUID) error {
	return m.deletePrefErr
}

func (m *mockNotificationStore) GetNotificationLogsByOrgID(_ context.Context, _ uuid.UUID, _ int) ([]*models.NotificationLog, error) {
	if m.listLogErr != nil {
		return nil, m.listLogErr
	}
	return m.logs, nil
}

func setupNotificationTestRouter(store NotificationStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		// Set an enterprise license so feature gate checks pass in tests
		c.Set("license", &license.License{Tier: license.TierEnterprise})
		c.Next()
	})
	km, _ := crypto.NewKeyManager(make([]byte, 32))
	handler := NewNotificationsHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListChannels(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()

	channel := &models.NotificationChannel{ID: channelID, OrgID: orgID, Name: "slack", Type: models.ChannelTypeSlack, Enabled: true}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channels:    []*models.NotificationChannel{channel},
		channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["channels"]; !ok {
			t.Fatal("expected 'channels' key")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockNotificationStore{user: nil}
		wrongSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupNotificationTestRouter(noUserStore, wrongSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			user:        dbUser,
			listChanErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetChannel(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()

	channel := &models.NotificationChannel{ID: channelID, OrgID: orgID, Name: "slack"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockNotificationStore{
			channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:        otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupNotificationTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestCreateChannel(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channelByID: map[uuid.UUID]*models.NotificationChannel{},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test-slack","type":"slack","config":{"webhook_url":"https://hooks.slack.com/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"type":"slack","config":{}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid channel type", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"bad","type":"invalid_type","config":{}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			channelByID:   map[uuid.UUID]*models.NotificationChannel{},
			user:          dbUser,
			createChanErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail","type":"slack","config":{"webhook_url":"https://x"}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"test","type":"slack","config":{"webhook_url":"https://x"}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockNotificationStore{user: nil}
		wrongSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupNotificationTestRouter(noUserStore, wrongSession)
		w := httptest.NewRecorder()
		body := `{"name":"test","type":"slack","config":{"webhook_url":"https://x"}}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/channels", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestUpdateChannel(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()

	channel := &models.NotificationChannel{ID: channelID, OrgID: orgID, Name: "slack", Enabled: true}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated-slack"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/"+channelID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"test"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/"+channelID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrgID := uuid.New()
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: otherOrgID, Role: models.UserRoleAdmin}
		wrongStore := &mockNotificationStore{
			channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:        otherUser,
		}
		wrongSession := &auth.SessionUser{ID: otherUserID, CurrentOrgID: otherOrgID}
		r := setupNotificationTestRouter(wrongStore, wrongSession)
		w := httptest.NewRecorder()
		body := `{"name":"test"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/"+channelID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			channelByID:   map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:          dbUser,
			updateChanErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail"}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/channels/"+channelID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDeleteChannel(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()

	channel := &models.NotificationChannel{ID: channelID, OrgID: orgID, Name: "slack"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			channelByID:   map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:          dbUser,
			deleteChanErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrgID := uuid.New()
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: otherOrgID, Role: models.UserRoleAdmin}
		wrongStore := &mockNotificationStore{
			channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:        otherUser,
		}
		wrongSession := &auth.SessionUser{ID: otherUserID, CurrentOrgID: otherOrgID}
		r := setupNotificationTestRouter(wrongStore, wrongSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/channels/"+channelID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestListPreferences(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{user: dbUser}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/preferences", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/preferences", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockNotificationStore{user: nil}
		wrongSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupNotificationTestRouter(noUserStore, wrongSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/preferences", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestCreatePreference(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()

	channel := &models.NotificationChannel{ID: channelID, OrgID: orgID, Name: "slack"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: channel},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid event type", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"invalid_event","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("channel not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + uuid.New().String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("store create error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			channelByID:   map[uuid.UUID]*models.NotificationChannel{channelID: channel},
			user:          dbUser,
			createPrefErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockNotificationStore{user: nil}
		wrongSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupNotificationTestRouter(noUserStore, wrongSession)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("channel wrong org", func(t *testing.T) {
		otherOrgID := uuid.New()
		otherChannel := &models.NotificationChannel{ID: channelID, OrgID: otherOrgID, Name: "slack"}
		wrongOrgStore := &mockNotificationStore{
			channelByID: map[uuid.UUID]*models.NotificationChannel{channelID: otherChannel},
			user:        dbUser,
		}
		r := setupNotificationTestRouter(wrongOrgStore, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + channelID.String() + `","event_type":"backup_success","enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/notifications/preferences", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestUpdatePreference(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	prefID := uuid.New()

	pref := &models.NotificationPreference{ID: prefID, OrgID: orgID, Enabled: true}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		preferences: []*models.NotificationPreference{pref},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences/"+prefID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences/"+prefID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			preferences:   []*models.NotificationPreference{pref},
			user:          dbUser,
			updatePrefErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"enabled":false}`
		req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences/"+prefID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestDeletePreference(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	prefID := uuid.New()

	pref := &models.NotificationPreference{ID: prefID, OrgID: orgID, Enabled: true}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{
		preferences: []*models.NotificationPreference{pref},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/preferences/"+prefID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/preferences/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/preferences/"+prefID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/preferences/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store delete error", func(t *testing.T) {
		errStore := &mockNotificationStore{
			preferences:   []*models.NotificationPreference{pref},
			user:          dbUser,
			deletePrefErr: errors.New("db error"),
		}
		r := setupNotificationTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/notifications/preferences/"+prefID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestListLogs(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockNotificationStore{user: dbUser, logs: []*models.NotificationLog{}}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupNotificationTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupNotificationTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockNotificationStore{user: nil, logs: []*models.NotificationLog{}}
		wrongSession := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupNotificationTestRouter(noUserStore, wrongSession)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications/logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}
