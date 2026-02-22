package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestDefaultSessionConfig(t *testing.T) {
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, true, 0, -1)

	if cfg.MaxAge != 86400 {
		t.Errorf("expected MaxAge 86400, got %d", cfg.MaxAge)
	}
	if cfg.IdleTimeout != 1800 {
		t.Errorf("expected IdleTimeout 1800, got %d", cfg.IdleTimeout)
	}
	if !cfg.Secure {
		t.Error("expected Secure to be true")
	}
	if !cfg.HTTPOnly {
		t.Error("expected HTTPOnly to be true")
	}
	if cfg.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", cfg.SameSite)
	}
	if cfg.CookiePath != "/" {
		t.Errorf("expected CookiePath '/', got %s", cfg.CookiePath)
	}
}

func TestDefaultSessionConfig_CustomValues(t *testing.T) {
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, true, 3600, 900)

	if cfg.MaxAge != 3600 {
		t.Errorf("expected MaxAge 3600, got %d", cfg.MaxAge)
	}
	if cfg.IdleTimeout != 900 {
		t.Errorf("expected IdleTimeout 900, got %d", cfg.IdleTimeout)
	}
}

func TestDefaultSessionConfig_Insecure(t *testing.T) {
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false, 0, 0)

	if cfg.Secure {
		t.Error("expected Secure to be false for insecure config")
	}
	if !cfg.HTTPOnly {
		t.Error("expected HTTPOnly to still be true")
	}
}

func newTestSessionStore(t *testing.T) *SessionStore {
	t.Helper()
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false, 0, 0)
	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store
}

func newTestSessionStoreWithIdleTimeout(t *testing.T, idleTimeout int) *SessionStore {
	t.Helper()
	logger := zerolog.Nop()
	cfg := SessionConfig{
		Secret:      []byte("test-secret-that-is-at-least-32-bytes-long"),
		MaxAge:      86400,
		IdleTimeout: idleTimeout,
		Secure:      false,
		HTTPOnly:    true,
		SameSite:    http.SameSiteLaxMode,
		CookiePath:  "/",
	}
	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store
}

func TestSession_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newTestSessionStore(t)
		if store == nil {
			t.Fatal("expected non-nil store")
		}
	})

	t.Run("secret too short", func(t *testing.T) {
		logger := zerolog.Nop()
		cfg := SessionConfig{
			Secret:   []byte("short"),
			MaxAge:   3600,
			Secure:   false,
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		}

		_, err := NewSessionStore(cfg, logger)
		if err == nil {
			t.Error("expected error for short secret")
		}
	})

	t.Run("exactly 32 bytes", func(t *testing.T) {
		logger := zerolog.Nop()
		cfg := SessionConfig{
			Secret:   []byte("12345678901234567890123456789012"),
			MaxAge:   3600,
			Secure:   false,
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		}

		store, err := NewSessionStore(cfg, logger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store == nil {
			t.Fatal("expected non-nil store")
		}
	})
}

func TestSession_Get(t *testing.T) {
	store := newTestSessionStore(t)

	t.Run("get from fresh request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		session, err := store.Get(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
	})

	t.Run("get user from empty session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_, err := store.GetUser(req)
		if err == nil {
			t.Error("expected error for missing user in session")
		}
	})

	t.Run("get OIDC state from empty session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		_, err := store.GetOIDCState(req, w)
		if err == nil {
			t.Error("expected error for missing state in session")
		}
	})
}

func TestSessionStore_OIDCState(t *testing.T) {
	store := newTestSessionStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	testState := "test-state-12345"
	if err := store.SetOIDCState(req, w, testState); err != nil {
		t.Fatalf("failed to set state: %v", err)
	}

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()

	state, err := store.GetOIDCState(req2, w2)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}
	if state != testState {
		t.Errorf("expected state %s, got %s", testState, state)
	}

	// State should be cleared after retrieval
	resp2 := w2.Result()
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp2.Cookies() {
		req3.AddCookie(cookie)
	}
	w3 := httptest.NewRecorder()

	_, err = store.GetOIDCState(req3, w3)
	if err == nil {
		t.Error("expected error when state has been cleared")
	}
}

func TestSession_Destroy(t *testing.T) {
	store := newTestSessionStore(t)

	// Set user
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	testUser := &SessionUser{
		ID:          uuid.New(),
		OIDCSubject: "sub-12345",
		Email:       "test@example.com",
	}
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Create request with session cookie
	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()

	// Clear user
	if err := store.ClearUser(req2, w2); err != nil {
		t.Fatalf("failed to clear user: %v", err)
	}

	// Verify cookie is set to expire
	resp2 := w2.Result()
	cookies := resp2.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	if cookies[0].MaxAge >= 0 {
		t.Errorf("expected MaxAge < 0 to delete cookie, got %d", cookies[0].MaxAge)
	}
}

func TestSessionStore_User(t *testing.T) {
	store := newTestSessionStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	testUser := &SessionUser{
		ID:              uuid.New(),
		OIDCSubject:     "sub-12345",
		Email:           "test@example.com",
		Name:            "Test User",
		AuthenticatedAt: time.Now().Truncate(time.Second),
		CurrentOrgID:    uuid.New(),
		CurrentOrgRole:  "admin",
	}

	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}

	user, err := store.GetUser(req2)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user.ID != testUser.ID {
		t.Errorf("expected ID %s, got %s", testUser.ID, user.ID)
	}
	if user.OIDCSubject != testUser.OIDCSubject {
		t.Errorf("expected subject %s, got %s", testUser.OIDCSubject, user.OIDCSubject)
	}
	if user.Email != testUser.Email {
		t.Errorf("expected email %s, got %s", testUser.Email, user.Email)
	}
	if user.Name != testUser.Name {
		t.Errorf("expected name %s, got %s", testUser.Name, user.Name)
	}
	if user.CurrentOrgID != testUser.CurrentOrgID {
		t.Errorf("expected org ID %s, got %s", testUser.CurrentOrgID, user.CurrentOrgID)
	}
	if user.CurrentOrgRole != testUser.CurrentOrgRole {
		t.Errorf("expected org role %s, got %s", testUser.CurrentOrgRole, user.CurrentOrgRole)
	}
}

func TestSession_Refresh(t *testing.T) {
	store := newTestSessionStore(t)

	// First set a user
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	testUser := &SessionUser{
		ID:             uuid.New(),
		OIDCSubject:    "sub-12345",
		Email:          "test@example.com",
		CurrentOrgID:   uuid.New(),
		CurrentOrgRole: "member",
	}
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Update the current org
	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()

	newOrgID := uuid.New()
	if err := store.SetCurrentOrg(req2, w2, newOrgID, "admin"); err != nil {
		t.Fatalf("failed to set current org: %v", err)
	}

	// Verify the updated org
	resp2 := w2.Result()
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp2.Cookies() {
		req3.AddCookie(cookie)
	}

	user, err := store.GetUser(req3)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user.CurrentOrgID != newOrgID {
		t.Errorf("expected org ID %s, got %s", newOrgID, user.CurrentOrgID)
	}
	if user.CurrentOrgRole != "admin" {
		t.Errorf("expected org role admin, got %s", user.CurrentOrgRole)
	}
	// User ID should remain unchanged
	if user.ID != testUser.ID {
		t.Errorf("expected user ID %s to remain, got %s", testUser.ID, user.ID)
	}
}

func TestSession_Expired(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := SessionConfig{
		Secret:     secret,
		MaxAge:     1, // 1 second
		Secure:     false,
		HTTPOnly:   true,
		SameSite:   http.SameSiteLaxMode,
		CookiePath: "/",
	}

	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Set user
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	testUser := &SessionUser{
		ID:          uuid.New(),
		OIDCSubject: "sub-12345",
		Email:       "test@example.com",
	}
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Verify cookie has the expected MaxAge
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	if cookies[0].MaxAge != 1 {
		t.Errorf("expected MaxAge 1, got %d", cookies[0].MaxAge)
	}
}

func TestSessionStore_IsAuthenticated(t *testing.T) {
	store := newTestSessionStore(t)

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if store.IsAuthenticated(req) {
			t.Error("expected IsAuthenticated to be false for new request")
		}
	})

	t.Run("authenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}
		if !store.IsAuthenticated(req2) {
			t.Error("expected IsAuthenticated to be true after setting user")
		}
	})

	t.Run("after clear", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}
		w2 := httptest.NewRecorder()

		if err := store.ClearUser(req2, w2); err != nil {
			t.Fatalf("failed to clear user: %v", err)
		}

		// After clearing, a new request without cookies shouldn't be authenticated
		req3 := httptest.NewRequest(http.MethodGet, "/", nil)
		if store.IsAuthenticated(req3) {
			t.Error("expected IsAuthenticated to be false after clear (no cookie)")
		}
	})
}

func TestSessionStore_Save(t *testing.T) {
	store := newTestSessionStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	session, err := store.Get(req)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	session.Values["test_key"] = "test_value"
	if err := store.Save(req, w, session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// Verify cookie was set
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set after save")
	}
}

func TestSessionStore_IdleTimeout(t *testing.T) {
	t.Run("session expires after idle timeout", func(t *testing.T) {
		// Create store with 1-second idle timeout
		store := newTestSessionStoreWithIdleTimeout(t, 1)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		// Wait for idle timeout to expire
		time.Sleep(1100 * time.Millisecond)

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}

		// GetUser should fail due to idle timeout
		_, err := store.GetUser(req2)
		if err == nil {
			t.Error("expected error for idle timeout expired session")
		}

		// IsAuthenticated should also return false
		if store.IsAuthenticated(req2) {
			t.Error("expected IsAuthenticated to be false after idle timeout")
		}
	})

	t.Run("session valid within idle timeout", func(t *testing.T) {
		store := newTestSessionStoreWithIdleTimeout(t, 3600)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}

		user, err := store.GetUser(req2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != testUser.ID {
			t.Errorf("expected user ID %s, got %s", testUser.ID, user.ID)
		}
	})

	t.Run("idle timeout disabled with zero", func(t *testing.T) {
		store := newTestSessionStoreWithIdleTimeout(t, 0)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}

		// Should succeed regardless - idle timeout is disabled
		user, err := store.GetUser(req2)
		if err != nil {
			t.Fatalf("unexpected error with idle timeout disabled: %v", err)
		}
		if user.ID != testUser.ID {
			t.Errorf("expected user ID %s, got %s", testUser.ID, user.ID)
		}
	})
}

func TestSessionStore_TouchSession(t *testing.T) {
	t.Run("touch updates last activity", func(t *testing.T) {
		store := newTestSessionStoreWithIdleTimeout(t, 2)

		// Set user
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		testUser := &SessionUser{
			ID:          uuid.New(),
			OIDCSubject: "sub-12345",
			Email:       "test@example.com",
		}
		if err := store.SetUser(req, w, testUser); err != nil {
			t.Fatalf("failed to set user: %v", err)
		}

		// Wait, then touch the session to keep it alive
		time.Sleep(1 * time.Second)

		resp := w.Result()
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp.Cookies() {
			req2.AddCookie(cookie)
		}
		w2 := httptest.NewRecorder()

		if err := store.TouchSession(req2, w2); err != nil {
			t.Fatalf("failed to touch session: %v", err)
		}

		// Wait again, but total time since touch should be within timeout
		time.Sleep(1 * time.Second)

		resp2 := w2.Result()
		req3 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range resp2.Cookies() {
			req3.AddCookie(cookie)
		}

		// Session should still be valid because we touched it
		user, err := store.GetUser(req3)
		if err != nil {
			t.Fatalf("unexpected error after touch: %v", err)
		}
		if user.ID != testUser.ID {
			t.Errorf("expected user ID %s, got %s", testUser.ID, user.ID)
		}
	})

	t.Run("touch is noop when idle timeout disabled", func(t *testing.T) {
		store := newTestSessionStoreWithIdleTimeout(t, 0)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		// Should succeed without error even with no session data
		if err := store.TouchSession(req, w); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSessionStore_IdleTimeout_IsAuthenticated(t *testing.T) {
	store := newTestSessionStoreWithIdleTimeout(t, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	testUser := &SessionUser{
		ID:          uuid.New(),
		OIDCSubject: "sub-12345",
		Email:       "test@example.com",
	}
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}

	// Should be authenticated initially
	if !store.IsAuthenticated(req2) {
		t.Error("expected IsAuthenticated to be true immediately after login")
	}

	// Wait for idle timeout
	time.Sleep(1100 * time.Millisecond)

	// Should no longer be authenticated
	if store.IsAuthenticated(req2) {
		t.Error("expected IsAuthenticated to be false after idle timeout")
	}
}
