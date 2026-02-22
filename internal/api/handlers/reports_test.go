package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/reports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockReportStore struct {
	schedules    []*models.ReportSchedule
	scheduleByID map[uuid.UUID]*models.ReportSchedule
	history      []*models.ReportHistory
	historyByID  map[uuid.UUID]*models.ReportHistory
	user             *models.User
	createErr        error
	updateErr        error
	deleteErr        error
	listSchedulesErr error
	listHistoryErr   error
	user         *models.User
	createErr    error
	updateErr    error
	deleteErr    error
}

func (m *mockReportStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockReportStore) GetReportSchedulesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.ReportSchedule, error) {
	if m.listSchedulesErr != nil {
		return nil, m.listSchedulesErr
	}
	var result []*models.ReportSchedule
	for _, s := range m.schedules {
		if s.OrgID == orgID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockReportStore) GetReportScheduleByID(_ context.Context, id uuid.UUID) (*models.ReportSchedule, error) {
	if s, ok := m.scheduleByID[id]; ok {
		return s, nil
	}
	return nil, errors.New("report schedule not found")
}

func (m *mockReportStore) CreateReportSchedule(_ context.Context, _ *models.ReportSchedule) error {
	return m.createErr
}

func (m *mockReportStore) UpdateReportSchedule(_ context.Context, _ *models.ReportSchedule) error {
	return m.updateErr
}

func (m *mockReportStore) DeleteReportSchedule(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockReportStore) GetReportHistoryByOrgID(_ context.Context, orgID uuid.UUID, _ int) ([]*models.ReportHistory, error) {
	if m.listHistoryErr != nil {
		return nil, m.listHistoryErr
	}
	var result []*models.ReportHistory
	for _, h := range m.history {
		if h.OrgID == orgID {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockReportStore) GetReportHistoryByID(_ context.Context, id uuid.UUID) (*models.ReportHistory, error) {
	if h, ok := m.historyByID[id]; ok {
		return h, nil
	}
	return nil, errors.New("report history not found")
}

func setupReportTestRouter(store ReportStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	// Pass nil scheduler - basic CRUD tests don't need it
	handler := NewReportsHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListReportSchedules(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	scheduleID := uuid.New()

	schedule := &models.ReportSchedule{
		ID:         scheduleID,
		OrgID:      orgID,
		Name:       "weekly-report",
		Frequency:  models.ReportFrequencyWeekly,
		Recipients: []string{"admin@example.com"},
	}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		schedules:    []*models.ReportSchedule{schedule},
		scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
		user:         dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["schedules"]; !ok {
			t.Fatal("expected 'schedules' key")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockReportStore{
			schedules:    []*models.ReportSchedule{schedule},
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupReportTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockReportStore{
			schedules:        []*models.ReportSchedule{schedule},
			scheduleByID:     map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:             dbUser,
			listSchedulesErr: errors.New("db error"),
		}
		r := setupReportTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetReportSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	scheduleID := uuid.New()

	schedule := &models.ReportSchedule{ID: scheduleID, OrgID: orgID, Name: "weekly-report"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
		user:         dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupReportTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestCreateReportSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		scheduleByID: map[uuid.UUID]*models.ReportSchedule{},
		user:         dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"daily-report","frequency":"daily","recipients":["admin@example.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"daily","recipients":["a@b.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid frequency", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"bad","frequency":"hourly","recipients":["a@b.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{},
			user:         dbUser,
			createErr:    errors.New("db error"),
		}
		r := setupReportTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"fail","frequency":"daily","recipients":["a@b.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"test","frequency":"daily","recipients":["a@b.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{},
			user:         nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupReportTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		body := `{"name":"test","frequency":"daily","recipients":["a@b.com"]}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("with timezone and channel", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		channelID := uuid.New().String()
		body := `{"name":"tz-report","frequency":"weekly","recipients":["a@b.com"],"timezone":"America/New_York","channel_id":"` + channelID + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestUpdateReportSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	scheduleID := uuid.New()

	schedule := &models.ReportSchedule{ID: scheduleID, OrgID: orgID, Name: "weekly-report", Frequency: models.ReportFrequencyWeekly}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
		user:         dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated-report"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupReportTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		body := `{"name":"nope"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		errStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         dbUser,
			updateErr:    errors.New("db error"),
		}
		r := setupReportTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"updated"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("update with frequency", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"daily"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update invalid frequency", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"hourly"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("update with channel_id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":"` + uuid.New().String() + `"}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update clear channel_id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"channel_id":""}`
		req, _ := http.NewRequest("PUT", "/api/v1/reports/schedules/"+scheduleID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDeleteReportSchedule(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	scheduleID := uuid.New()

	schedule := &models.ReportSchedule{ID: scheduleID, OrgID: orgID, Name: "weekly-report"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
		user:         dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         dbUser,
			deleteErr:    errors.New("db error"),
		}
		r := setupReportTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockReportStore{
			scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
			user:         otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupReportTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/reports/schedules/"+scheduleID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestListReportHistory(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		history: []*models.ReportHistory{},
		user:    dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockReportStore{
			history: []*models.ReportHistory{},
			user:    nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupReportTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockReportStore{
			history:        []*models.ReportHistory{},
			user:           dbUser,
			listHistoryErr: errors.New("db error"),
		}
		r := setupReportTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetReportHistoryEntry(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	historyID := uuid.New()

	entry := &models.ReportHistory{ID: historyID, OrgID: orgID, ReportType: "daily", PeriodStart: time.Now(), PeriodEnd: time.Now()}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockReportStore{
		historyByID: map[uuid.UUID]*models.ReportHistory{historyID: entry},
		user:        dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history/"+historyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockReportStore{
			historyByID: map[uuid.UUID]*models.ReportHistory{historyID: entry},
			user:        otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupReportTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history/"+historyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupReportTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history/"+historyID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupReportTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/history/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

// mockReportSchedulerStore implements both ReportStore and reports.SchedulerStore for testing SendReport/PreviewReport.
type mockReportSchedulerStore struct {
	mockReportStore
	org              *models.Organization
	lastSentErr      error
	historyCreateErr error
}

func (m *mockReportSchedulerStore) GetBackupsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Backup, error) {
	return []*models.Backup{}, nil
}

func (m *mockReportSchedulerStore) GetEnabledSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return []*models.Schedule{}, nil
}

func (m *mockReportSchedulerStore) GetStorageStatsSummary(_ context.Context, _ uuid.UUID) (*models.StorageStatsSummary, error) {
	return &models.StorageStatsSummary{}, nil
}

func (m *mockReportSchedulerStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return []*models.Agent{}, nil
}

func (m *mockReportSchedulerStore) GetAlertsByOrgIDAndDateRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.Alert, error) {
	return []*models.Alert{}, nil
}

func (m *mockReportSchedulerStore) GetEnabledReportSchedules(_ context.Context) ([]*models.ReportSchedule, error) {
	return nil, nil
}

func (m *mockReportSchedulerStore) UpdateReportScheduleLastSent(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return m.lastSentErr
}

func (m *mockReportSchedulerStore) CreateReportHistory(_ context.Context, _ *models.ReportHistory) error {
	return m.historyCreateErr
}

func (m *mockReportSchedulerStore) GetNotificationChannelByID(_ context.Context, _ uuid.UUID) (*models.NotificationChannel, error) {
	return nil, errors.New("not found")
}

func (m *mockReportSchedulerStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
	if m.org != nil {
		return m.org, nil
	}
	return &models.Organization{Name: "Test"}, nil
}

func setupReportTestRouterWithScheduler(store *mockReportSchedulerStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	scheduler := reports.NewScheduler(store, reports.DefaultSchedulerConfig(), zerolog.Nop())
	handler := NewReportsHandler(store, scheduler, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestSendReport(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	scheduleID := uuid.New()

	schedule := &models.ReportSchedule{
		ID:         scheduleID,
		OrgID:      orgID,
		Name:       "test-report",
		Frequency:  models.ReportFrequencyDaily,
		Recipients: []string{"admin@example.com"},
	}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	t.Run("success", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{
				scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
				user:         dbUser,
			},
			org: &models.Organization{ID: orgID, Name: "Test"},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/"+scheduleID.String()+"/send", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		// SendReport will fail with "no email channel configured" since there's no channel
		// but the handler returns 500 with the error message, which is expected behavior
		// when no SMTP is configured. Preview mode bypasses this.
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (no email channel), got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("schedule not found", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{
				scheduleByID: map[uuid.UUID]*models.ReportSchedule{},
				user:         dbUser,
			},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/"+uuid.New().String()+"/send", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{
				scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
				user:         otherUser,
			},
		}
		user := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/"+scheduleID.String()+"/send", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/bad-uuid/send", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
		}
		r := setupReportTestRouterWithScheduler(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/"+scheduleID.String()+"/send", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("preview mode", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{
				scheduleByID: map[uuid.UUID]*models.ReportSchedule{scheduleID: schedule},
				user:         dbUser,
			},
			org: &models.Organization{ID: orgID, Name: "Test"},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{"preview":true}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/schedules/"+scheduleID.String()+"/send", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestPreviewReport(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	t.Run("success", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
			org:             &models.Organization{ID: orgID, Name: "Test"},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"daily"}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid frequency", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"hourly"}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing frequency", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
		}
		r := setupReportTestRouterWithScheduler(store, nil)
		w := httptest.NewRecorder()
		body := `{"frequency":"daily"}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("with timezone", func(t *testing.T) {
		store := &mockReportSchedulerStore{
			mockReportStore: mockReportStore{user: dbUser},
			org:             &models.Organization{ID: orgID, Name: "Test"},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupReportTestRouterWithScheduler(store, user)
		w := httptest.NewRecorder()
		body := `{"frequency":"weekly","timezone":"America/New_York"}`
		req, _ := http.NewRequest("POST", "/api/v1/reports/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}
