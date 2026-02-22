package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestSessionStore(t *testing.T) *auth.SessionStore {
	t.Helper()
	cfg := auth.DefaultSessionConfig([]byte("test-secret-that-is-at-least-32-bytes-long!"), false, 0, 0)
	store, err := auth.NewSessionStore(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}
	return store
}

// setSessionCookies creates a session for the given user and returns the cookies.
func setSessionCookies(t *testing.T, sessions *auth.SessionStore, user *auth.SessionUser) []*http.Cookie {
	t.Helper()
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	if err := sessions.SetUser(req, w, user); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}
	return w.Result().Cookies()
}

// mockAgentStore implements auth.AgentStore for testing.
type mockAgentStore struct {
	agents map[string]*models.Agent // hash -> agent
}

func (m *mockAgentStore) GetAgentByAPIKeyHash(_ context.Context, hash string) (*models.Agent, error) {
	agent, ok := m.agents[hash]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	return agent, nil
}

// testAPIKey is a deterministic valid API key for tests.
const testAPIKey = "kld_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func newTestAgent() *models.Agent {
	return &models.Agent{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Hostname: "test-agent-01",
		Status:   models.AgentStatusActive,
	}
}

func newTestValidator(agents map[string]*models.Agent) *auth.APIKeyValidator {
	return auth.NewAPIKeyValidator(&mockAgentStore{agents: agents}, zerolog.Nop())
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	userID := uuid.New()
	orgID := uuid.New()
	sessionUser := &auth.SessionUser{
		ID:              userID,
		OIDCSubject:     "subject-123",
		Email:           "test@example.com",
		Name:            "Test User",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": user.ID.String()})
	})

	cookies := setSessionCookies(t, sessions, sessionUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	// Create a session with one secret, then try to validate with a different secret.
	// This simulates an expired/rotated session.
	oldStore := newTestSessionStore(t)
	sessionUser := &auth.SessionUser{
		ID:              uuid.New(),
		OIDCSubject:     "subject-expired",
		Email:           "expired@example.com",
		Name:            "Expired User",
		AuthenticatedAt: time.Now().Add(-48 * time.Hour),
		CurrentOrgID:    uuid.New(),
	}
	cookies := setSessionCookies(t, oldStore, sessionUser)

	// Create a new session store with a different secret (simulates secret rotation)
	cfg := auth.DefaultSessionConfig([]byte("different-secret-that-is-at-least-32-bytes!!"), false, 0, 0)
	newStore, err := auth.NewSessionStore(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("failed to create new session store: %v", err)
	}

	mw := AuthMiddleware(newStore, zerolog.Nop())
	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for expired/rotated session, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// Send a garbage cookie that doesn't decode to a valid session
	req.AddCookie(&http.Cookie{
		Name:  "keldris_session",
		Value: "completely-invalid-garbage-data",
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for invalid session, got %d", w.Code)
	}
}

func TestAuthMiddleware_RefreshSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	sessionUser := &auth.SessionUser{
		ID:              uuid.New(),
		OIDCSubject:     "subject-refresh",
		Email:           "refresh@example.com",
		Name:            "Refresh User",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    uuid.New(),
		CurrentOrgRole:  "admin",
	}

	cookies := setSessionCookies(t, sessions, sessionUser)

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_id": user.ID.String(),
			"email":   user.Email,
			"org_id":  user.CurrentOrgID.String(),
		})
	})

	// First request with session cookie
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req1.AddCookie(cookie)
	}
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected status 200, got %d", w1.Code)
	}

	// Second request with the same session cookie should also work
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second request: expected status 200, got %d", w2.Code)
	}

	// Verify user data is preserved across requests
	var resp map[string]string
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["user_id"] != sessionUser.ID.String() {
		t.Fatalf("expected user_id %s, got %s", sessionUser.ID, resp["user_id"])
	}
	if resp["email"] != sessionUser.Email {
		t.Fatalf("expected email %s, got %s", sessionUser.Email, resp["email"])
	}
}

func TestGetUser_NoUser(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user != nil {
			t.Fatal("expected nil user")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUser_WrongType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), "not-a-session-user")
		c.Next()
	})
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user != nil {
			t.Fatal("expected nil user for wrong type")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRequireUser(t *testing.T) {
	t.Run("with user", func(t *testing.T) {
		r := gin.New()
		sessionUser := &auth.SessionUser{
			ID:           uuid.New(),
			CurrentOrgID: uuid.New(),
		}
		r.Use(func(c *gin.Context) {
			c.Set(string(UserContextKey), sessionUser)
			c.Next()
		})
		r.GET("/test", func(c *gin.Context) {
			user := RequireUser(c)
			if user == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"user_id": user.ID.String()})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("without user", func(t *testing.T) {
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			user := RequireUser(c)
			if user == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestGetAgent_NoAgent(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		agent := GetAgent(c)
		if agent != nil {
			t.Fatal("expected nil agent")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetAgent_WrongType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AgentContextKey), "not-an-agent")
		c.Next()
	})
	r.GET("/test", func(c *gin.Context) {
		agent := GetAgent(c)
		if agent != nil {
			t.Fatal("expected nil agent for wrong type")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRequireAgent(t *testing.T) {
	t.Run("with agent", func(t *testing.T) {
		r := gin.New()
		agent := newTestAgent()
		r.Use(func(c *gin.Context) {
			c.Set(string(AgentContextKey), agent)
			c.Next()
		})
		r.GET("/test", func(c *gin.Context) {
			a := RequireAgent(c)
			if a == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"agent_id": a.ID.String()})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("without agent", func(t *testing.T) {
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			agent := RequireAgent(c)
			if agent == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestSessionStore(t *testing.T) *auth.SessionStore {
	t.Helper()
	cfg := auth.DefaultSessionConfig([]byte("test-secret-that-is-at-least-32-bytes-long!"), false)
	store, err := auth.NewSessionStore(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}
	return store
}

// setSessionCookies creates a session for the given user and returns the cookies.
func setSessionCookies(t *testing.T, sessions *auth.SessionStore, user *auth.SessionUser) []*http.Cookie {
	t.Helper()
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	if err := sessions.SetUser(req, w, user); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}
	return w.Result().Cookies()
}

// mockAgentStore implements auth.AgentStore for testing.
type mockAgentStore struct {
	agents map[string]*models.Agent // hash -> agent
}

func (m *mockAgentStore) GetAgentByAPIKeyHash(_ context.Context, hash string) (*models.Agent, error) {
	agent, ok := m.agents[hash]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	return agent, nil
}

// testAPIKey is a deterministic valid API key for tests.
const testAPIKey = "kld_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func newTestAgent() *models.Agent {
	return &models.Agent{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Hostname: "test-agent-01",
		Status:   models.AgentStatusActive,
	}
}

func newTestValidator(agents map[string]*models.Agent) *auth.APIKeyValidator {
	return auth.NewAPIKeyValidator(&mockAgentStore{agents: agents}, zerolog.Nop())
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	userID := uuid.New()
	orgID := uuid.New()
	sessionUser := &auth.SessionUser{
		ID:              userID,
		OIDCSubject:     "subject-123",
		Email:           "test@example.com",
		Name:            "Test User",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    orgID,
	}

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": user.ID.String()})
	})

	cookies := setSessionCookies(t, sessions, sessionUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	// Create a session with one secret, then try to validate with a different secret.
	// This simulates an expired/rotated session.
	oldStore := newTestSessionStore(t)
	sessionUser := &auth.SessionUser{
		ID:              uuid.New(),
		OIDCSubject:     "subject-expired",
		Email:           "expired@example.com",
		Name:            "Expired User",
		AuthenticatedAt: time.Now().Add(-48 * time.Hour),
		CurrentOrgID:    uuid.New(),
	}
	cookies := setSessionCookies(t, oldStore, sessionUser)

	// Create a new session store with a different secret (simulates secret rotation)
	cfg := auth.DefaultSessionConfig([]byte("different-secret-that-is-at-least-32-bytes!!"), false)
	newStore, err := auth.NewSessionStore(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("failed to create new session store: %v", err)
	}

	mw := AuthMiddleware(newStore, zerolog.Nop())
	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for expired/rotated session, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// Send a garbage cookie that doesn't decode to a valid session
	req.AddCookie(&http.Cookie{
		Name:  "keldris_session",
		Value: "completely-invalid-garbage-data",
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for invalid session, got %d", w.Code)
	}
}

func TestAuthMiddleware_RefreshSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	sessionUser := &auth.SessionUser{
		ID:              uuid.New(),
		OIDCSubject:     "subject-refresh",
		Email:           "refresh@example.com",
		Name:            "Refresh User",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    uuid.New(),
		CurrentOrgRole:  "admin",
	}

	cookies := setSessionCookies(t, sessions, sessionUser)

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_id": user.ID.String(),
			"email":   user.Email,
			"org_id":  user.CurrentOrgID.String(),
		})
	})

	// First request with session cookie
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req1.AddCookie(cookie)
	}
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected status 200, got %d", w1.Code)
	}

	// Second request with the same session cookie should also work
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second request: expected status 200, got %d", w2.Code)
	}

	// Verify user data is preserved across requests
	var resp map[string]string
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["user_id"] != sessionUser.ID.String() {
		t.Fatalf("expected user_id %s, got %s", sessionUser.ID, resp["user_id"])
	}
	if resp["email"] != sessionUser.Email {
		t.Fatalf("expected email %s, got %s", sessionUser.Email, resp["email"])
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := OptionalAuthMiddleware(sessions, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user != nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	t.Run("no session proceeds", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("with valid session loads user", func(t *testing.T) {
		sessionUser := &auth.SessionUser{
			ID:              uuid.New(),
			Email:           "optional@example.com",
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    uuid.New(),
		}
		cookies := setSessionCookies(t, sessions, sessionUser)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp map[string]bool
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp["authenticated"] {
			t.Fatal("expected authenticated to be true")
		}
	})
}

func TestGetUser_NoUser(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user != nil {
			t.Fatal("expected nil user")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUser_WrongType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), "not-a-session-user")
		c.Next()
	})
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user != nil {
			t.Fatal("expected nil user for wrong type")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRequireUser(t *testing.T) {
	t.Run("with user", func(t *testing.T) {
		r := gin.New()
		sessionUser := &auth.SessionUser{
			ID:           uuid.New(),
			CurrentOrgID: uuid.New(),
		}
		r.Use(func(c *gin.Context) {
			c.Set(string(UserContextKey), sessionUser)
			c.Next()
		})
		r.GET("/test", func(c *gin.Context) {
			user := RequireUser(c)
			if user == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"user_id": user.ID.String()})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("without user", func(t *testing.T) {
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			user := RequireUser(c)
			if user == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestGetAgent_NoAgent(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		agent := GetAgent(c)
		if agent != nil {
			t.Fatal("expected nil agent")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetAgent_WrongType(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AgentContextKey), "not-an-agent")
		c.Next()
	})
	r.GET("/test", func(c *gin.Context) {
		agent := GetAgent(c)
		if agent != nil {
			t.Fatal("expected nil agent for wrong type")
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRequireAgent(t *testing.T) {
	t.Run("with agent", func(t *testing.T) {
		r := gin.New()
		agent := newTestAgent()
		r.Use(func(c *gin.Context) {
			c.Set(string(AgentContextKey), agent)
			c.Next()
		})
		r.GET("/test", func(c *gin.Context) {
			a := RequireAgent(c)
			if a == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"agent_id": a.ID.String()})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("without agent", func(t *testing.T) {
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			agent := RequireAgent(c)
			if agent == nil {
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestSessionOrAPIKeyMiddleware_SessionAuth(t *testing.T) {
	sessions := newTestSessionStore(t)
	agent := newTestAgent()
	keyHash := auth.HashAPIKey(testAPIKey)
	validator := newTestValidator(map[string]*models.Agent{keyHash: agent})

	mw := SessionOrAPIKeyMiddleware(sessions, validator, zerolog.Nop())

	sessionUser := &auth.SessionUser{
		ID:              uuid.New(),
		Email:           "dual@example.com",
		AuthenticatedAt: time.Now(),
		CurrentOrgID:    uuid.New(),
	}
	cookies := setSessionCookies(t, sessions, sessionUser)

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no auth"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"auth": "session", "user_id": user.ID.String()})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionOrAPIKeyMiddleware_APIKeyAuth(t *testing.T) {
	sessions := newTestSessionStore(t)
	agent := newTestAgent()
	keyHash := auth.HashAPIKey(testAPIKey)
	validator := newTestValidator(map[string]*models.Agent{keyHash: agent})

	mw := SessionOrAPIKeyMiddleware(sessions, validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		a := GetAgent(c)
		if a == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no auth"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"auth": "apikey", "agent_id": a.ID.String()})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSessionOrAPIKeyMiddleware_NoAuth(t *testing.T) {
	sessions := newTestSessionStore(t)
	validator := newTestValidator(map[string]*models.Agent{})

	mw := SessionOrAPIKeyMiddleware(sessions, validator, zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
