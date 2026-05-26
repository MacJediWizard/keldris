package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func injectUser(user *auth.SessionUser) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user != nil {
			c.Set(string(UserContextKey), user)
		}
		c.Next()
	}
}

func TestSuperuserMiddleware_NoUser401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", SuperuserMiddleware(nil, zerolog.Nop()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSuperuserMiddleware_NonSuperuser403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(injectUser(&auth.SessionUser{ID: uuid.New()}))
	r.GET("/x", SuperuserMiddleware(nil, zerolog.Nop()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestSuperuserMiddleware_SuperuserPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(injectUser(&auth.SessionUser{ID: uuid.New(), IsSuperuser: true}))
	r.GET("/x", SuperuserMiddleware(nil, zerolog.Nop()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireSuperuser_NoUser401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		if RequireSuperuser(c) == nil {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireSuperuser_NonSuperuser403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(injectUser(&auth.SessionUser{ID: uuid.New()}))
	r.GET("/x", func(c *gin.Context) {
		if RequireSuperuser(c) == nil {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestIsSuperuser_True(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(injectUser(&auth.SessionUser{ID: uuid.New(), IsSuperuser: true}))
	r.GET("/x", func(c *gin.Context) {
		if !IsSuperuser(c) {
			c.Status(http.StatusForbidden)
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestIsImpersonating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("false when not impersonating", func(t *testing.T) {
		r := gin.New()
		r.Use(injectUser(&auth.SessionUser{ID: uuid.New()}))
		r.GET("/x", func(c *gin.Context) {
			if IsImpersonating(c) {
				c.Status(http.StatusForbidden)
				return
			}
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/x", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 when not impersonating, got %d", w.Code)
		}
	})

	t.Run("true when ImpersonatingID is set", func(t *testing.T) {
		r := gin.New()
		impID := uuid.New()
		r.Use(injectUser(&auth.SessionUser{ID: uuid.New(), ImpersonatingID: impID}))
		r.GET("/x", func(c *gin.Context) {
			if !IsImpersonating(c) {
				c.Status(http.StatusForbidden)
				return
			}
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/x", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 when impersonating, got %d", w.Code)
		}
	})
}
