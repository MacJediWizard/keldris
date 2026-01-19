package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestDefaultSessionConfig(t *testing.T) {
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, true)

	if cfg.MaxAge != 86400 {
		t.Errorf("expected MaxAge 86400, got %d", cfg.MaxAge)
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

func TestNewSessionStore_SecretTooShort(t *testing.T) {
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
}

func TestNewSessionStore_Success(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false)

	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestSessionStore_OIDCState(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false)

	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create test request and response
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Set state
	testState := "test-state-12345"
	if err := store.SetOIDCState(req, w, testState); err != nil {
		t.Fatalf("failed to set state: %v", err)
	}

	// Copy cookies from response to new request
	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()

	// Get state
	state, err := store.GetOIDCState(req2, w2)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}
	if state != testState {
		t.Errorf("expected state %s, got %s", testState, state)
	}
}

func TestSessionStore_User(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false)

	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create test request and response
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Test user
	testUser := &SessionUser{
		ID:          uuid.New(),
		OIDCSubject: "sub-12345",
		Email:       "test@example.com",
		Name:        "Test User",
	}

	// Set user
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Copy cookies from response to new request
	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}

	// Get user
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
}

func TestSessionStore_IsAuthenticated(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false)

	store, err := NewSessionStore(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Test unauthenticated
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if store.IsAuthenticated(req) {
		t.Error("expected IsAuthenticated to be false for new request")
	}

	// Set user
	w := httptest.NewRecorder()
	testUser := &SessionUser{
		ID:          uuid.New(),
		OIDCSubject: "sub-12345",
		Email:       "test@example.com",
	}
	if err := store.SetUser(req, w, testUser); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	// Test authenticated
	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	if !store.IsAuthenticated(req2) {
		t.Error("expected IsAuthenticated to be true after setting user")
	}
}

func TestSessionStore_ClearUser(t *testing.T) {
	logger := zerolog.Nop()
	secret := []byte("test-secret-that-is-at-least-32-bytes-long")
	cfg := DefaultSessionConfig(secret, false)

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
