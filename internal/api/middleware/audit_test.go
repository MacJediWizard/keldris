package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockAuditStore implements AuditStore for testing.
type mockAuditStore struct {
	mu   sync.Mutex
	logs []*models.AuditLog
	user *models.User
}

func (m *mockAuditStore) CreateAuditLog(_ context.Context, log *models.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, nil
}

func (m *mockAuditStore) getLogs() []*models.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.AuditLog, len(m.logs))
	copy(result, m.logs)
	return result
}

func newMockAuditStore(user *models.User) *mockAuditStore {
	return &mockAuditStore{user: user}
}

func TestAuditMiddleware_LogsRequest(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Email: "audit@example.com",
		Name:  "Audit User",
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	sessionUser := &auth.SessionUser{
		ID:              userID,
		Email:           user.Email,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	r := gin.New()
	// Inject user into context (simulates AuthMiddleware)
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), sessionUser)
		c.Next()
	})
	r.Use(AuditMiddleware(store, zerolog.Nop()))
	r.GET("/api/v1/agents", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
	req.Header.Set("User-Agent", "test-client/1.0")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Wait for async audit log goroutine
	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	log := logs[0]
	if log.Action != models.AuditActionRead {
		t.Fatalf("expected action 'read', got %q", log.Action)
	}
	if log.ResourceType != "agent" {
		t.Fatalf("expected resource type 'agent', got %q", log.ResourceType)
	}
	if log.Result != models.AuditResultSuccess {
		t.Fatalf("expected result 'success', got %q", log.Result)
	}
	if log.OrgID != orgID {
		t.Fatalf("expected org_id %s, got %s", orgID, log.OrgID)
	}
}

func TestAuditMiddleware_LogsResponse(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Email: "audit@example.com",
		Role:  models.UserRoleAdmin,
	}

	sessionUser := &auth.SessionUser{
		ID:              userID,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	// Test POST (create) with success
	t.Run("POST creates success log", func(t *testing.T) {
		store := newMockAuditStore(user)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(UserContextKey), sessionUser)
			c.Next()
		})
		r.Use(AuditMiddleware(store, zerolog.Nop()))
		r.POST("/api/v1/agents", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"id": "new-agent"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/agents", nil)
		r.ServeHTTP(w, req)

		time.Sleep(50 * time.Millisecond)

		logs := store.getLogs()
		if len(logs) != 1 {
			t.Fatalf("expected 1 audit log, got %d", len(logs))
		}
		if logs[0].Action != models.AuditActionCreate {
			t.Fatalf("expected action 'create', got %q", logs[0].Action)
		}
		if logs[0].Result != models.AuditResultSuccess {
			t.Fatalf("expected result 'success' for 201, got %q", logs[0].Result)
		}
	})

	// Test DELETE with 404 (failure)
	t.Run("DELETE with 404 creates failure log", func(t *testing.T) {
		store := newMockAuditStore(user)
		agentID := uuid.New()
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(UserContextKey), sessionUser)
			c.Next()
		})
		r.Use(AuditMiddleware(store, zerolog.Nop()))
		r.DELETE("/api/v1/agents/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		time.Sleep(50 * time.Millisecond)

		logs := store.getLogs()
		if len(logs) != 1 {
			t.Fatalf("expected 1 audit log, got %d", len(logs))
		}
		if logs[0].Action != models.AuditActionDelete {
			t.Fatalf("expected action 'delete', got %q", logs[0].Action)
		}
		if logs[0].Result != models.AuditResultFailure {
			t.Fatalf("expected result 'failure' for 404, got %q", logs[0].Result)
		}
	})
}

func TestAuditMiddleware_CapturesUser(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Email: "capture@example.com",
		Role:  models.UserRoleUser,
	}
	store := newMockAuditStore(user)

	sessionUser := &auth.SessionUser{
		ID:              userID,
		Email:           "capture@example.com",
		Name:            "Capture User",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), sessionUser)
		c.Next()
	})
	r.Use(AuditMiddleware(store, zerolog.Nop()))

	agentID := uuid.New()
	r.PUT("/api/v1/agents/"+agentID.String(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/agents/"+agentID.String(), nil)
	req.Header.Set("User-Agent", "keldris-test/1.0")
	r.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	log := logs[0]
	if log.UserID == nil {
		t.Fatal("expected user_id to be set")
	}
	if *log.UserID != userID {
		t.Fatalf("expected user_id %s, got %s", userID, *log.UserID)
	}
	if log.ResourceID == nil {
		t.Fatal("expected resource_id to be set")
	}
	if *log.ResourceID != agentID {
		t.Fatalf("expected resource_id %s, got %s", agentID, *log.ResourceID)
	}
	if log.Action != models.AuditActionUpdate {
		t.Fatalf("expected action 'update', got %q", log.Action)
	}
	if log.UserAgent != "keldris-test/1.0" {
		t.Fatalf("expected user agent 'keldris-test/1.0', got %q", log.UserAgent)
	}
}

func TestAuditMiddleware_SkipsHealthEndpoint(t *testing.T) {
	user := &models.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	sessionUser := &auth.SessionUser{
		ID:              user.ID,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    user.OrgID,
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), sessionUser)
		c.Next()
	})
	r.Use(AuditMiddleware(store, zerolog.Nop()))
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Health endpoint should be skipped
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w2, req2)

	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 0 {
		t.Fatalf("expected 0 audit logs for health endpoints, got %d", len(logs))
	}
}

func TestAuditMiddleware_SkipsAuditLogEndpoint(t *testing.T) {
	user := &models.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	sessionUser := &auth.SessionUser{
		ID:              user.ID,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    user.OrgID,
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), sessionUser)
		c.Next()
	})
	r.Use(AuditMiddleware(store, zerolog.Nop()))
	r.GET("/api/v1/audit-logs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"logs": []string{}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit-logs", nil)
	r.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 0 {
		t.Fatalf("expected 0 audit logs for audit-logs endpoint, got %d", len(logs))
	}
}

func TestAuditMiddleware_SkipsUnauthenticatedRequests(t *testing.T) {
	user := &models.User{
		ID:    uuid.New(),
		OrgID: uuid.New(),
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	r := gin.New()
	// No auth middleware - user is nil
	r.Use(AuditMiddleware(store, zerolog.Nop()))
	r.GET("/api/v1/agents", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
	r.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 0 {
		t.Fatalf("expected 0 audit logs for unauthenticated request, got %d", len(logs))
	}
}

func TestAuditMiddleware_DeniedResult(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Role:  models.UserRoleViewer,
	}
	store := newMockAuditStore(user)

	sessionUser := &auth.SessionUser{
		ID:              userID,
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	agentID := uuid.New()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), sessionUser)
		c.Next()
	})
	r.Use(AuditMiddleware(store, zerolog.Nop()))
	r.DELETE("/api/v1/agents/:id", func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+agentID.String(), nil)
	r.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	logs := store.getLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Result != models.AuditResultDenied {
		t.Fatalf("expected result 'denied' for 403, got %q", logs[0].Result)
	}
}

func TestMapMethodToAction(t *testing.T) {
	tests := []struct {
		method string
		action models.AuditAction
	}{
		{"GET", models.AuditActionRead},
		{"POST", models.AuditActionCreate},
		{"PUT", models.AuditActionUpdate},
		{"PATCH", models.AuditActionUpdate},
		{"DELETE", models.AuditActionDelete},
		{"HEAD", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := mapMethodToAction(tt.method)
			if got != tt.action {
				t.Fatalf("expected %q, got %q", tt.action, got)
			}
		})
	}
}

func TestParseResourceFromPath(t *testing.T) {
	agentID := uuid.New()

	tests := []struct {
		name         string
		path         string
		resourceType string
		hasID        bool
		expectedID   uuid.UUID
	}{
		{"agents list", "/api/v1/agents", "agent", false, uuid.Nil},
		{"agent by id", "/api/v1/agents/" + agentID.String(), "agent", true, agentID},
		{"repositories", "/api/v1/repositories", "repository", false, uuid.Nil},
		{"schedules", "/api/v1/schedules", "schedule", false, uuid.Nil},
		{"backups", "/api/v1/backups", "backup", false, uuid.Nil},
		{"users", "/api/v1/users", "user", false, uuid.Nil},
		{"organizations", "/api/v1/organizations", "organization", false, uuid.Nil},
		{"auth login", "/api/v1/auth/login", "session", false, uuid.Nil},
		{"auth callback", "/api/v1/auth/callback", "session", false, uuid.Nil},
		{"auth logout", "/api/v1/auth/logout", "session", false, uuid.Nil},
		{"auth base", "/api/v1/auth", "auth", false, uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceType, resourceID := parseResourceFromPath(tt.path)
			if resourceType != tt.resourceType {
				t.Fatalf("expected resource type %q, got %q", tt.resourceType, resourceType)
			}
			if tt.hasID {
				if resourceID != tt.expectedID {
					t.Fatalf("expected resource ID %s, got %s", tt.expectedID, resourceID)
				}
			} else {
				if resourceID != uuid.Nil {
					t.Fatalf("expected nil resource ID, got %s", resourceID)
				}
			}
		})
	}
}

func TestLogAuditEvent(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	resourceID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	LogAuditEvent(
		store,
		zerolog.Nop(),
		orgID,
		userID,
		models.AuditActionCreate,
		"agent",
		&resourceID,
		models.AuditResultSuccess,
		"created agent via API",
	)

	logs := store.getLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}

	log := logs[0]
	if log.OrgID != orgID {
		t.Fatalf("expected org_id %s, got %s", orgID, log.OrgID)
	}
	if log.UserID == nil || *log.UserID != userID {
		t.Fatal("expected user_id to be set correctly")
	}
	if log.ResourceID == nil || *log.ResourceID != resourceID {
		t.Fatal("expected resource_id to be set correctly")
	}
	if log.Action != models.AuditActionCreate {
		t.Fatalf("expected action 'create', got %q", log.Action)
	}
	if log.Details != "created agent via API" {
		t.Fatalf("expected details 'created agent via API', got %q", log.Details)
	}
}

func TestLogAuditEvent_NilResourceID(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		OrgID: orgID,
		Role:  models.UserRoleAdmin,
	}
	store := newMockAuditStore(user)

	LogAuditEvent(
		store,
		zerolog.Nop(),
		orgID,
		userID,
		models.AuditActionLogin,
		"session",
		nil,
		models.AuditResultSuccess,
		"user logged in",
	)

	logs := store.getLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].ResourceID != nil {
		t.Fatal("expected nil resource_id")
	}
}
