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
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockReportStore struct {
	schedules    []*models.ReportSchedule
	scheduleByID map[uuid.UUID]*models.ReportSchedule
	history      []*models.ReportHistory
	historyByID  map[uuid.UUID]*models.ReportHistory
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
}
