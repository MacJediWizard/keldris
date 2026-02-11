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
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockAuditLogStore struct {
	logs      []*models.AuditLog
	logByID   map[uuid.UUID]*models.AuditLog
	count     int64
	user      *models.User
	createErr error
	listErr   error
	countErr  error
}

func (m *mockAuditLogStore) GetAuditLogsByOrgID(_ context.Context, orgID uuid.UUID, _ db.AuditLogFilter) ([]*models.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*models.AuditLog
	for _, l := range m.logs {
		if l.OrgID == orgID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockAuditLogStore) GetAuditLogByID(_ context.Context, id uuid.UUID) (*models.AuditLog, error) {
	if l, ok := m.logByID[id]; ok {
		return l, nil
	}
	return nil, errors.New("audit log not found")
}

func (m *mockAuditLogStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return m.createErr
}

func (m *mockAuditLogStore) CountAuditLogsByOrgID(_ context.Context, _ uuid.UUID, _ db.AuditLogFilter) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func (m *mockAuditLogStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func setupAuditLogTestRouter(store AuditLogStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewAuditLogsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListAuditLogs(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	logID := uuid.New()

	auditLog := &models.AuditLog{ID: logID, OrgID: orgID, Action: models.AuditActionCreate, ResourceType: "agent", Result: models.AuditResultSuccess}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAuditLogStore{
		logs:    []*models.AuditLog{auditLog},
		logByID: map[uuid.UUID]*models.AuditLog{logID: auditLog},
		count:   1,
		user:    dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp AuditLogListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.TotalCount != 1 {
			t.Fatalf("expected total_count 1, got %d", resp.TotalCount)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?action=create&resource_type=agent&limit=10&offset=0", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockAuditLogStore{
			logs:    []*models.AuditLog{auditLog},
			logByID: map[uuid.UUID]*models.AuditLog{logID: auditLog},
			count:   1,
			user:    nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupAuditLogTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAuditLogStore{
			logs:    []*models.AuditLog{auditLog},
			logByID: map[uuid.UUID]*models.AuditLog{logID: auditLog},
			count:   1,
			user:    dbUser,
			listErr: errors.New("db error"),
		}
		r := setupAuditLogTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("with date filters RFC3339", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("with date filters date-only", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?start_date=2024-01-01&end_date=2024-12-31", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("with result and search", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?result=success&search=test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("with invalid limit", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?limit=abc", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (should use default limit)", w.Code)
		}
	})

	t.Run("with invalid date format", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs?start_date=not-a-date", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (invalid date should be ignored)", w.Code)
		}
	})

	t.Run("count error", func(t *testing.T) {
		countErrStore := &mockAuditLogStore{
			logs:     []*models.AuditLog{auditLog},
			logByID:  map[uuid.UUID]*models.AuditLog{logID: auditLog},
			count:    0,
			user:     dbUser,
			countErr: errors.New("count db error"),
		}
		r := setupAuditLogTestRouter(countErrStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetAuditLog(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	logID := uuid.New()

	auditLog := &models.AuditLog{ID: logID, OrgID: orgID, Action: models.AuditActionCreate, ResourceType: "agent"}
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAuditLogStore{
		logByID: map[uuid.UUID]*models.AuditLog{logID: auditLog},
		user:    dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/"+logID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockAuditLogStore{
			logByID: map[uuid.UUID]*models.AuditLog{logID: auditLog},
			user:    otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupAuditLogTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/"+logID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/"+logID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestExportAuditLogsCSV(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAuditLogStore{
		logs: []*models.AuditLog{},
		user: dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		ct := w.Header().Get("Content-Type")
		if ct != "text/csv" {
			t.Fatalf("expected Content-Type text/csv, got %s", ct)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockAuditLogStore{
			logs: []*models.AuditLog{},
			user: nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupAuditLogTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("with data", func(t *testing.T) {
		csvLogID := uuid.New()
		dataStore := &mockAuditLogStore{
			logs: []*models.AuditLog{
				{
					ID:           csvLogID,
					OrgID:        orgID,
					Action:       models.AuditActionCreate,
					ResourceType: "agent",
					Result:       models.AuditResultSuccess,
					IPAddress:    "127.0.0.1",
					UserAgent:    "test-agent",
					Details:      "created agent",
					CreatedAt:    time.Now(),
				},
			},
			user: dbUser,
		}
		r := setupAuditLogTestRouter(dataStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "create") {
			t.Fatal("expected CSV to contain action data")
		}
		if !strings.Contains(body, "agent") {
			t.Fatal("expected CSV to contain resource type data")
		}
	})

	t.Run("with filters", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv?action=create&start_date=2024-01-01", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAuditLogStore{
			user:    dbUser,
			listErr: errors.New("db error"),
		}
		r := setupAuditLogTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/csv", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestExportAuditLogsJSON(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	dbUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}
	store := &mockAuditLogStore{
		logs: []*models.AuditLog{},
		user: dbUser,
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %s", ct)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("with data", func(t *testing.T) {
		jsonLogID := uuid.New()
		dataStore := &mockAuditLogStore{
			logs: []*models.AuditLog{
				{
					ID:           jsonLogID,
					OrgID:        orgID,
					Action:       models.AuditActionCreate,
					ResourceType: "agent",
					Result:       models.AuditResultSuccess,
					CreatedAt:    time.Now(),
				},
			},
			user: dbUser,
		}
		r := setupAuditLogTestRouter(dataStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "create") {
			t.Fatal("expected JSON to contain action data")
		}
	})

	t.Run("with filters", func(t *testing.T) {
		r := setupAuditLogTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json?action=create&end_date=2024-12-31T23:59:59Z", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAuditLogStore{
			user:    dbUser,
			listErr: errors.New("db error"),
		}
		r := setupAuditLogTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		noUserStore := &mockAuditLogStore{
			logs: []*models.AuditLog{},
			user: nil,
		}
		randomUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}
		r := setupAuditLogTestRouter(noUserStore, randomUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit-logs/export/json", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestUuidPtrToString(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		result := uuidPtrToString(nil)
		if result != "" {
			t.Fatalf("expected empty string for nil, got %q", result)
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		id := uuid.New()
		result := uuidPtrToString(&id)
		if result != id.String() {
			t.Fatalf("expected %q, got %q", id.String(), result)
		}
	})
}
