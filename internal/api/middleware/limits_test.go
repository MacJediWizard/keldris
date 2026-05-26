package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type stubCounter struct {
	agents, users, orgs int
	err                 error
}

func (s *stubCounter) CountAgentsByOrgID(_ context.Context, _ uuid.UUID) (int, error) {
	return s.agents, s.err
}

func (s *stubCounter) CountUsersByOrgID(_ context.Context, _ uuid.UUID) (int, error) {
	return s.users, s.err
}

func (s *stubCounter) CountOrganizations(_ context.Context) (int, error) {
	return s.orgs, s.err
}

func setupLimitRouter(counter ResourceCounter, resource string, user *auth.SessionUser, lic *license.License) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(UserContextKey), user)
		}
		if lic != nil {
			c.Set(string(LicenseContextKey), lic)
		}
		c.Next()
	})
	r.POST("/x", LimitMiddleware(counter, resource, zerolog.Nop()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestLimitMiddleware_NoUserPasses(t *testing.T) {
	counter := &stubCounter{agents: 999}
	r := setupLimitRouter(counter, "agents", nil, license.FreeLicense())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when no user, got %d", w.Code)
	}
}

func TestLimitMiddleware_UnlimitedPasses(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierEnterprise, Limits: license.TierLimits{MaxAgents: -1}}
	counter := &stubCounter{agents: 999999}
	r := setupLimitRouter(counter, "agents", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unlimited, got %d", w.Code)
	}
}

func TestLimitMiddleware_UnderLimitPasses(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierFree, Limits: license.TierLimits{MaxAgents: 5}}
	counter := &stubCounter{agents: 3}
	r := setupLimitRouter(counter, "agents", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 under limit, got %d", w.Code)
	}
}

func TestLimitMiddleware_AtLimitBlocked(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierFree, Limits: license.TierLimits{MaxAgents: 5}}
	counter := &stubCounter{agents: 5}
	r := setupLimitRouter(counter, "agents", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 at limit, got %d", w.Code)
	}
}

func TestLimitMiddleware_UsersResource(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierFree, Limits: license.TierLimits{MaxUsers: 2}}
	counter := &stubCounter{users: 2}
	r := setupLimitRouter(counter, "users", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for users limit, got %d", w.Code)
	}
}

func TestLimitMiddleware_OrganizationsResource(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierFree, Limits: license.TierLimits{MaxOrgs: 1}}
	counter := &stubCounter{orgs: 1}
	r := setupLimitRouter(counter, "organizations", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for orgs limit, got %d", w.Code)
	}
}

func TestLimitMiddleware_UnknownResourcePassesThrough(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := license.FreeLicense()
	counter := &stubCounter{}
	r := setupLimitRouter(counter, "unknown", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown resource, got %d", w.Code)
	}
}

func TestLimitMiddleware_CounterErrorPassesThrough(t *testing.T) {
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
	lic := &license.License{Tier: license.TierFree, Limits: license.TierLimits{MaxAgents: 5}}
	counter := &stubCounter{err: context.DeadlineExceeded}
	r := setupLimitRouter(counter, "agents", user, lic)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on counter error (fail-open), got %d", w.Code)
	}
}
