package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
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

func TestAuthMiddleware_ValidSession(t *testing.T) {
	sessions := newTestSessionStore(t)
	mw := AuthMiddleware(sessions, zerolog.Nop())

	// Set up authenticated session
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

	// Create request and set session
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

	// First, create a request to set the session
	setReq, _ := http.NewRequest("GET", "/test", nil)
	setW := httptest.NewRecorder()
	if err := sessions.SetUser(setReq, setW, sessionUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Get cookie from the response
	cookies := setW.Result().Cookies()

	// Now make request with the session cookie
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
