package handlers

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockUserStore implements UserStore for auth handler testing.
type mockUserStore struct {
	user            *models.User
	org             *models.Organization
	memberships     []*models.OrgMembership
	ssoGroups       *models.UserSSOGroups
	groupMappings   []*models.SSOGroupMapping
	membershipByOrg map[uuid.UUID]*models.OrgMembership

	getUserErr          error
	createUserErr       error
	getOrCreateOrgErr   error
	getMembershipsErr   error
	createMembershipErr error
	getSSOGroupsErr     error
	createAuditLogErr   error
}

func (m *mockUserStore) GetUserByOIDCSubject(_ context.Context, subject string) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	if m.user != nil && m.user.OIDCSubject == subject {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserStore) CreateUser(_ context.Context, user *models.User) error {
	if m.createUserErr != nil {
		return m.createUserErr
	}
	m.user = user
	return nil
}

func (m *mockUserStore) GetOrCreateDefaultOrg(_ context.Context) (*models.Organization, error) {
	if m.getOrCreateOrgErr != nil {
		return nil, m.getOrCreateOrgErr
	}
	if m.org != nil {
		return m.org, nil
	}
	return models.NewOrganization("Default", "default"), nil
}

func (m *mockUserStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	if m.getMembershipsErr != nil {
		return nil, m.getMembershipsErr
	}
	return m.memberships, nil
}

func (m *mockUserStore) CreateMembership(_ context.Context, membership *models.OrgMembership) error {
	if m.createMembershipErr != nil {
		return m.createMembershipErr
	}
	m.memberships = append(m.memberships, membership)
	return nil
}

func (m *mockUserStore) GetSSOGroupMappingsByGroupNames(_ context.Context, _ []string) ([]*models.SSOGroupMapping, error) {
	return m.groupMappings, nil
}

func (m *mockUserStore) GetSSOGroupMappingsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.SSOGroupMapping, error) {
	return m.groupMappings, nil
}

func (m *mockUserStore) GetMembershipByUserAndOrg(_ context.Context, _ uuid.UUID, orgID uuid.UUID) (*models.OrgMembership, error) {
	if m.membershipByOrg != nil {
		if mem, ok := m.membershipByOrg[orgID]; ok {
			return mem, nil
		}
	}
	return nil, errors.New("membership not found")
}

func (m *mockUserStore) UpdateMembershipRole(_ context.Context, _ uuid.UUID, _ models.OrgRole) error {
	return nil
}

func (m *mockUserStore) UpsertUserSSOGroups(_ context.Context, _ uuid.UUID, _ []string) error {
	return nil
}

func (m *mockUserStore) GetUserSSOGroups(_ context.Context, _ uuid.UUID) (*models.UserSSOGroups, error) {
	if m.getSSOGroupsErr != nil {
		return nil, m.getSSOGroupsErr
	}
	return m.ssoGroups, nil
}

func (m *mockUserStore) GetOrganizationByID(_ context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.org != nil && m.org.ID == id {
		return m.org, nil
	}
	return nil, errors.New("organization not found")
}

func (m *mockUserStore) GetOrganizationSSOSettings(_ context.Context, _ uuid.UUID) (*string, bool, error) {
	return nil, false, nil
}

func (m *mockUserStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return m.createAuditLogErr
}

func (m *mockUserStore) CreateUserSession(_ context.Context, _ *models.UserSession) error {
	return nil
}

func (m *mockUserStore) RevokeUserSession(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockUserStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserStore) GetUserPasswordInfo(_ context.Context, _ uuid.UUID) (*models.UserPasswordInfo, error) {
	return nil, errors.New("not implemented")
}

// newMockOIDCServer creates a mock OIDC provider HTTP server for handler testing.
// Supported authorization codes: "valid-code" returns a valid token; unknown codes return an error.
func newMockOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	var server *httptest.Server
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		discovery := map[string]interface{}{
			"issuer":                                server.URL,
			"authorization_endpoint":                server.URL + "/authorize",
			"token_endpoint":                        server.URL + "/token",
			"jwks_uri":                              server.URL + "/jwks",
			"userinfo_endpoint":                     server.URL + "/userinfo",
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discovery)
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": "test-key",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")

		var resp map[string]interface{}
		switch code {
		case "valid-code":
			idToken := createTestIDToken(t, key, server.URL, "test-client-id",
				"sub-12345", "user@example.com", "Test User", time.Now().Add(time.Hour))
			resp = map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     idToken,
			}
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"sub":   "sub-12345",
			"email": "user@example.com",
			"name":  "Test User",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server = httptest.NewServer(mux)
	return server
}

// createTestIDToken creates a signed JWT ID token for testing.
func createTestIDToken(t *testing.T, key *rsa.PrivateKey, issuer, audience, subject, email, name string, expiry time.Time) string {
	t.Helper()

	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": "test-key",
	}
	claims := map[string]interface{}{
		"iss":   issuer,
		"sub":   subject,
		"aud":   audience,
		"email": email,
		"name":  name,
		"iat":   time.Now().Unix(),
		"exp":   expiry.Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	h := crypto.SHA256.New()
	h.Write([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// parseURL is a test helper that parses a URL string.
func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

// newAuthTestSessionStore creates a session store for testing.
func newAuthTestSessionStore(t *testing.T) *auth.SessionStore {
	t.Helper()
	secret := []byte("test-secret-must-be-32-bytes-lon")
	cfg := auth.SessionConfig{
		Secret:      secret,
		MaxAge:      86400,
		IdleTimeout: 0, // disable idle timeout for tests
		Secure:      false,
		HTTPOnly:    true,
		SameSite:    http.SameSiteLaxMode,
		CookiePath:  "/",
	}
	store, err := auth.NewSessionStore(cfg, zerolog.Nop())
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}
	return store
}

// newAuthTestOIDCProvider creates a mock OIDC server and provider for handler testing.
func newAuthTestOIDCProvider(t *testing.T) (*auth.OIDC, *httptest.Server) {
	t.Helper()
	server := newMockOIDCServer(t)

	cfg := auth.OIDCConfig{
		Issuer:       server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/auth/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	oidcProvider, err := auth.NewOIDC(context.Background(), cfg, zerolog.Nop())
	if err != nil {
		server.Close()
		t.Fatalf("failed to create OIDC provider: %v", err)
	}
	return oidcProvider, server
}

// setupAuthTestRouter creates a Gin router with the auth handler registered.
func setupAuthTestRouter(handler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authGroup := r.Group("/auth")
	handler.RegisterRoutes(authGroup)
	return r
}

func TestAuthHandler_Login(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	t.Run("redirects to OIDC provider", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/login", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Fatalf("expected status 302, got %d", w.Code)
		}

		location := w.Header().Get("Location")
		if location == "" {
			t.Fatal("expected Location header")
		}

		// Should contain OIDC parameters
		u, err := parseURL(location)
		if err != nil {
			t.Fatalf("failed to parse redirect URL: %v", err)
		}
		if u.Query().Get("state") == "" {
			t.Fatal("expected state parameter in redirect URL")
		}
		if u.Query().Get("client_id") != "test-client-id" {
			t.Fatalf("expected client_id 'test-client-id', got %q", u.Query().Get("client_id"))
		}
		if u.Query().Get("response_type") != "code" {
			t.Fatalf("expected response_type 'code', got %q", u.Query().Get("response_type"))
		}
	})
}

func TestAuthHandler_Callback(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	}

	t.Run("exchanges code and creates session", func(t *testing.T) {
		oidcProvider, server := newAuthTestOIDCProvider(t)
		defer server.Close()

		sessions := newAuthTestSessionStore(t)
		membership := models.NewOrgMembership(uuid.New(), orgID, models.OrgRoleOwner)
		store := &mockUserStore{
			org:         org,
			memberships: []*models.OrgMembership{membership},
		}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		// Initiate login to set the state in session
		loginW := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("GET", "/auth/login", nil)
		r.ServeHTTP(loginW, loginReq)

		cookies := loginW.Result().Cookies()

		location := loginW.Header().Get("Location")
		if location == "" {
			t.Fatal("expected redirect from login")
		}

		redirectURL, err := parseURL(location)
		if err != nil {
			t.Fatalf("failed to parse redirect URL: %v", err)
		}
		state := redirectURL.Query().Get("state")
		if state == "" {
			t.Fatal("expected state in redirect URL")
		}

		// Call callback with valid code and matching state
		callbackW := httptest.NewRecorder()
		callbackReq, _ := http.NewRequest("GET", "/auth/callback?code=valid-code&state="+url.QueryEscape(state), nil)
		for _, c := range cookies {
			callbackReq.AddCookie(c)
		}
		r.ServeHTTP(callbackW, callbackReq)

		if callbackW.Code != http.StatusFound {
			t.Fatalf("expected status 302, got %d: %s", callbackW.Code, callbackW.Body.String())
		}

		callbackLocation := callbackW.Header().Get("Location")
		if callbackLocation != "/" {
			t.Fatalf("expected redirect to /, got %q", callbackLocation)
		}
	})

	t.Run("OIDC provider error", func(t *testing.T) {
		oidcProvider, server := newAuthTestOIDCProvider(t)
		defer server.Close()

		sessions := newAuthTestSessionStore(t)
		store := &mockUserStore{org: org}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?error=access_denied&error_description=user+denied", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "access_denied" {
			t.Fatalf("expected error 'access_denied', got %q", resp["error"])
		}
	})
}

func TestAuthHandler_Callback_InvalidState(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	t.Run("missing state parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?code=valid-code", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "missing state parameter" {
			t.Fatalf("expected 'missing state parameter', got %q", resp["error"])
		}
	})

	t.Run("state mismatch", func(t *testing.T) {
		// Login to set a valid state
		loginW := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("GET", "/auth/login", nil)
		r.ServeHTTP(loginW, loginReq)
		cookies := loginW.Result().Cookies()

		// Use a completely different state in callback
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?code=valid-code&state=wrong-state", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "state mismatch" {
			t.Fatalf("expected 'state mismatch', got %q", resp["error"])
		}
	})

	t.Run("no session state", func(t *testing.T) {
		// Request without session cookie — no state stored
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?code=valid-code&state=some-state", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "invalid session state" {
			t.Fatalf("expected 'invalid session state', got %q", resp["error"])
		}
	})
}

func TestAuthHandler_Callback_InvalidCode(t *testing.T) {
	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	}

	t.Run("missing code", func(t *testing.T) {
		oidcProvider, server := newAuthTestOIDCProvider(t)
		defer server.Close()

		sessions := newAuthTestSessionStore(t)
		store := &mockUserStore{org: org}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		// Login to set state
		loginW := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("GET", "/auth/login", nil)
		r.ServeHTTP(loginW, loginReq)
		cookies := loginW.Result().Cookies()

		location := loginW.Header().Get("Location")
		redirectURL, _ := parseURL(location)
		state := redirectURL.Query().Get("state")

		// Callback without code parameter
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?state="+url.QueryEscape(state), nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "missing authorization code" {
			t.Fatalf("expected 'missing authorization code', got %q", resp["error"])
		}
	})

	t.Run("invalid code rejected by provider", func(t *testing.T) {
		oidcProvider, server := newAuthTestOIDCProvider(t)
		defer server.Close()

		sessions := newAuthTestSessionStore(t)
		store := &mockUserStore{org: org}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		// Login to set state
		loginW := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("GET", "/auth/login", nil)
		r.ServeHTTP(loginW, loginReq)
		cookies := loginW.Result().Cookies()

		location := loginW.Header().Get("Location")
		redirectURL, _ := parseURL(location)
		state := redirectURL.Query().Get("state")

		// Callback with code the mock server will reject
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/callback?code=invalid-code&state="+url.QueryEscape(state), nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["error"] != "authentication failed" {
			t.Fatalf("expected 'authentication failed', got %q", resp["error"])
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	t.Run("clears session", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp["message"] != "logged out successfully" {
			t.Fatalf("expected 'logged out successfully', got %q", resp["message"])
		}

		// Verify the session cookie is cleared (MaxAge=-1)
		for _, c := range w.Result().Cookies() {
			if c.Name == "keldris_session" && c.MaxAge > 0 {
				t.Fatal("expected session cookie to be cleared")
			}
		}
	})

	t.Run("succeeds even without active session", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Me(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	orgID := uuid.New()
	userID := uuid.New()
	syncedAt := time.Now()

	t.Run("returns current user", func(t *testing.T) {
		sessions := newAuthTestSessionStore(t)
		ssoGroups := &models.UserSSOGroups{
			ID:         uuid.New(),
			UserID:     userID,
			OIDCGroups: []string{"engineering", "admin"},
			SyncedAt:   syncedAt,
		}
		store := &mockUserStore{ssoGroups: ssoGroups}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		// Set up an authenticated session
		sessionUser := &auth.SessionUser{
			ID:              userID,
			OIDCSubject:     "sub-12345",
			Email:           "user@example.com",
			Name:            "Test User",
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    orgID,
			CurrentOrgRole:  "admin",
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/me", nil)
		if err := sessions.SetUser(req, w, sessionUser); err != nil {
			t.Fatalf("failed to set session user: %v", err)
		}

		// Issue new request with the session cookie
		meW := httptest.NewRecorder()
		meReq, _ := http.NewRequest("GET", "/auth/me", nil)
		for _, c := range w.Result().Cookies() {
			meReq.AddCookie(c)
		}
		r.ServeHTTP(meW, meReq)

		if meW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", meW.Code, meW.Body.String())
		}

		var resp MeResponse
		if err := json.Unmarshal(meW.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ID != userID {
			t.Fatalf("expected user ID %s, got %s", userID, resp.ID)
		}
		if resp.Email != "user@example.com" {
			t.Fatalf("expected email 'user@example.com', got %q", resp.Email)
		}
		if resp.Name != "Test User" {
			t.Fatalf("expected name 'Test User', got %q", resp.Name)
		}
		if resp.CurrentOrgID != orgID {
			t.Fatalf("expected org ID %s, got %s", orgID, resp.CurrentOrgID)
		}
		if resp.CurrentOrgRole != "admin" {
			t.Fatalf("expected role 'admin', got %q", resp.CurrentOrgRole)
		}
		if len(resp.SSOGroups) != 2 {
			t.Fatalf("expected 2 SSO groups, got %d", len(resp.SSOGroups))
		}
		if resp.SSOGroupsSyncedAt == nil {
			t.Fatal("expected non-nil SSO groups synced at")
		}
	})

	t.Run("returns user without SSO groups", func(t *testing.T) {
		sessions := newAuthTestSessionStore(t)
		store := &mockUserStore{getSSOGroupsErr: errors.New("no groups")}
		handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
		r := setupAuthTestRouter(handler)

		sessionUser := &auth.SessionUser{
			ID:              userID,
			OIDCSubject:     "sub-12345",
			Email:           "user@example.com",
			Name:            "Test User",
			AuthenticatedAt: time.Now(),
			CurrentOrgID:    orgID,
			CurrentOrgRole:  "admin",
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/auth/me", nil)
		if err := sessions.SetUser(req, w, sessionUser); err != nil {
			t.Fatalf("failed to set session user: %v", err)
		}

		meW := httptest.NewRecorder()
		meReq, _ := http.NewRequest("GET", "/auth/me", nil)
		for _, c := range w.Result().Cookies() {
			meReq.AddCookie(c)
		}
		r.ServeHTTP(meW, meReq)

		if meW.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", meW.Code, meW.Body.String())
		}

		var resp MeResponse
		if err := json.Unmarshal(meW.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.Email != "user@example.com" {
			t.Fatalf("expected email 'user@example.com', got %q", resp.Email)
		}
		if resp.SSOGroups != nil {
			t.Fatalf("expected nil SSO groups, got %v", resp.SSOGroups)
		}
		if resp.SSOGroupsSyncedAt != nil {
			t.Fatal("expected nil SSO groups synced at")
		}
	})
}

func TestAuthHandler_Me_Unauthenticated(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["error"] != "not authenticated" {
		t.Fatalf("expected 'not authenticated', got %q", resp["error"])
	}
}

func TestAuth_AuthStatus_NoOIDC(t *testing.T) {
	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	// Unconfigured OIDC provider (nil inner provider)
	handler := NewAuthHandler(auth.NewOIDCProvider(nil, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AuthStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.OIDCEnabled {
		t.Fatal("expected oidc_enabled to be false when OIDC is not configured")
	}
	if !resp.PasswordEnabled {
		t.Fatal("expected password_enabled to be true")
	}
}

func TestAuth_AuthStatus_WithOIDC(t *testing.T) {
	oidcProvider, server := newAuthTestOIDCProvider(t)
	defer server.Close()

	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	handler := NewAuthHandler(auth.NewOIDCProvider(oidcProvider, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AuthStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.OIDCEnabled {
		t.Fatal("expected oidc_enabled to be true when OIDC is configured")
	}
	if !resp.PasswordEnabled {
		t.Fatal("expected password_enabled to be true")
	}
}

func TestAuth_Login_NoOIDCProvider(t *testing.T) {
	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	// Unconfigured OIDC provider (nil inner provider)
	handler := NewAuthHandler(auth.NewOIDCProvider(nil, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/login", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "SSO not configured" {
		t.Fatalf("expected error 'SSO not configured', got %q", resp["error"])
	}
}

func TestAuth_Callback_NoOIDCProvider(t *testing.T) {
	sessions := newAuthTestSessionStore(t)
	store := &mockUserStore{}
	// Unconfigured OIDC provider (nil inner provider)
	handler := NewAuthHandler(auth.NewOIDCProvider(nil, zerolog.Nop()), sessions, store, zerolog.Nop())
	r := setupAuthTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/callback?code=test-code&state=test-state", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["error"] != "SSO not configured" {
		t.Fatalf("expected error 'SSO not configured', got %q", resp["error"])
	}
}
