package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// testUser creates a SessionUser for testing with a given org.
func testUser(orgID uuid.UUID) *auth.SessionUser {
	return &auth.SessionUser{
		ID:             uuid.New(),
		Email:          "test@example.com",
		Name:           "Test User",
		CurrentOrgID:   orgID,
		CurrentOrgRole: "admin",
	}
}

// testUserNoOrg creates a SessionUser with no organization selected.
func testUserNoOrg() *auth.SessionUser {
	return &auth.SessionUser{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}
}

// InjectUser returns gin middleware that injects a SessionUser into context.
func InjectUser(user *auth.SessionUser) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	}
}

// JSONRequest creates an HTTP request with JSON content type.
func JSONRequest(method, path, body string) *http.Request {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// AuthenticatedRequest creates a GET request (no body needed).
func AuthenticatedRequest(method, path string) *http.Request {
	req, _ := http.NewRequest(method, path, nil)
	return req
}

// SetupTestRouter creates a test gin engine with optional user injection.
func SetupTestRouter(user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(InjectUser(user))
	return r
}

// DoRequest performs a request and returns the recorder.
func DoRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
