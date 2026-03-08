package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/portal/portalctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// --- Mock store ---

type mockStore struct {
	customers map[string]*models.Customer // keyed by email
	sessions  map[string]*portalctx.Session // keyed by token hash

	// Tracking calls
	incrementFailedCalled bool
	lockCalled            bool
	lockUntil             time.Time
	resetFailedCalled     bool
	createSessionCalled   bool
	lastLoginUpdated      bool
	lastLoginIP           string
	passwordUpdated       bool
	newPasswordHash       string
	sessionsDeleted       bool
	resetTokenSaved       bool
	resetTokenValue       string
	resetTokenExpiry      time.Time

	// Error injection
	getByEmailErr         error
	createCustomerErr     error
	incrementFailedErr    error
	lockAccountErr        error
	resetFailedErr        error
	createSessionErr      error
	updateLastLoginErr    error
	updatePasswordErr     error
	deleteSessionsErr     error
	updateResetTokenErr   error
}

func newMockStore() *mockStore {
	return &mockStore{
		customers: make(map[string]*models.Customer),
		sessions:  make(map[string]*portalctx.Session),
	}
}

func (m *mockStore) GetCustomerByID(_ context.Context, id uuid.UUID) (*models.Customer, error) {
	for _, c := range m.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("customer not found")
}

func (m *mockStore) GetCustomerByEmail(_ context.Context, email string) (*models.Customer, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	c, ok := m.customers[email]
	if !ok {
		return nil, fmt.Errorf("customer not found")
	}
	return c, nil
}

func (m *mockStore) CreateCustomer(_ context.Context, customer *models.Customer) error {
	if m.createCustomerErr != nil {
		return m.createCustomerErr
	}
	m.customers[customer.Email] = customer
	return nil
}

func (m *mockStore) UpdateCustomer(_ context.Context, _ *models.Customer) error { return nil }

func (m *mockStore) UpdateCustomerPassword(_ context.Context, _ uuid.UUID, hash string) error {
	if m.updatePasswordErr != nil {
		return m.updatePasswordErr
	}
	m.passwordUpdated = true
	m.newPasswordHash = hash
	return nil
}

func (m *mockStore) UpdateCustomerResetToken(_ context.Context, _ uuid.UUID, token string, expiresAt time.Time) error {
	if m.updateResetTokenErr != nil {
		return m.updateResetTokenErr
	}
	m.resetTokenSaved = true
	m.resetTokenValue = token
	m.resetTokenExpiry = expiresAt
	return nil
}

func (m *mockStore) ClearCustomerResetToken(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockStore) IncrementCustomerFailedLogin(_ context.Context, _ uuid.UUID) error {
	if m.incrementFailedErr != nil {
		return m.incrementFailedErr
	}
	m.incrementFailedCalled = true
	return nil
}

func (m *mockStore) ResetCustomerFailedLogin(_ context.Context, _ uuid.UUID) error {
	if m.resetFailedErr != nil {
		return m.resetFailedErr
	}
	m.resetFailedCalled = true
	return nil
}

func (m *mockStore) LockCustomerAccount(_ context.Context, _ uuid.UUID, until time.Time) error {
	if m.lockAccountErr != nil {
		return m.lockAccountErr
	}
	m.lockCalled = true
	m.lockUntil = until
	return nil
}

func (m *mockStore) UpdateCustomerLastLogin(_ context.Context, _ uuid.UUID, ip string) error {
	if m.updateLastLoginErr != nil {
		return m.updateLastLoginErr
	}
	m.lastLoginUpdated = true
	m.lastLoginIP = ip
	return nil
}

func (m *mockStore) CreateSession(_ context.Context, session *portalctx.Session) error {
	if m.createSessionErr != nil {
		return m.createSessionErr
	}
	m.createSessionCalled = true
	m.sessions[session.TokenHash] = session
	return nil
}

func (m *mockStore) GetSessionByTokenHash(_ context.Context, tokenHash string) (*portalctx.Session, error) {
	s, ok := m.sessions[tokenHash]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

func (m *mockStore) DeleteSession(_ context.Context, id uuid.UUID) error {
	for k, s := range m.sessions {
		if s.ID == id {
			delete(m.sessions, k)
			return nil
		}
	}
	return nil
}

func (m *mockStore) DeleteSessionsByCustomerID(_ context.Context, _ uuid.UUID) error {
	if m.deleteSessionsErr != nil {
		return m.deleteSessionsErr
	}
	m.sessionsDeleted = true
	return nil
}

func (m *mockStore) CleanupExpiredSessions(_ context.Context) (int64, error) { return 0, nil }

// Stub license and invoice operations to satisfy Store interface.
func (m *mockStore) GetLicensesByCustomerID(_ context.Context, _ uuid.UUID) ([]*models.PortalLicense, error) {
	return nil, nil
}
func (m *mockStore) GetLicenseByID(_ context.Context, _ uuid.UUID) (*models.PortalLicense, error) {
	return nil, nil
}
func (m *mockStore) GetLicenseByKey(_ context.Context, _ string) (*models.PortalLicense, error) {
	return nil, nil
}
func (m *mockStore) CreateLicense(_ context.Context, _ *models.PortalLicense) error { return nil }
func (m *mockStore) UpdateLicense(_ context.Context, _ *models.PortalLicense) error { return nil }
func (m *mockStore) GetInvoicesByCustomerID(_ context.Context, _ uuid.UUID) ([]*models.Invoice, error) {
	return nil, nil
}
func (m *mockStore) GetInvoiceByID(_ context.Context, _ uuid.UUID) (*models.Invoice, error) {
	return nil, nil
}
func (m *mockStore) GetInvoiceByNumber(_ context.Context, _ string) (*models.Invoice, error) {
	return nil, nil
}
func (m *mockStore) GetInvoiceItems(_ context.Context, _ uuid.UUID) ([]*models.InvoiceItem, error) {
	return nil, nil
}
func (m *mockStore) CreateInvoice(_ context.Context, _ *models.Invoice) error      { return nil }
func (m *mockStore) CreateInvoiceItem(_ context.Context, _ *models.InvoiceItem) error { return nil }
func (m *mockStore) UpdateInvoice(_ context.Context, _ *models.Invoice) error      { return nil }
func (m *mockStore) ListCustomers(_ context.Context, _, _ int) ([]*models.Customer, int, error) {
	return nil, 0, nil
}
func (m *mockStore) ListLicenses(_ context.Context, _, _ int) ([]*models.PortalLicenseWithCustomer, int, error) {
	return nil, 0, nil
}
func (m *mockStore) ListInvoices(_ context.Context, _, _ int) ([]*models.InvoiceWithCustomer, int, error) {
	return nil, 0, nil
}
func (m *mockStore) GenerateInvoiceNumber(_ context.Context) (string, error) { return "INV-0001", nil }

// --- Helper functions ---

func newTestHandler(store *mockStore) *AuthHandler {
	logger := zerolog.Nop()
	return NewAuthHandler(store, logger)
}

func addTestCustomer(store *mockStore, email, password string, status models.CustomerStatus) *models.Customer {
	c := models.NewCustomer(email, "Test User", models.HashPassword(password))
	c.Status = status
	store.customers[email] = c
	return c
}

func performRequest(handler gin.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return w
}

func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response body: %v (body: %s)", err, w.Body.String())
	}
	return result
}

// --- Tests ---

func TestLogin_ValidCredentials(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	customer := addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)

	body := `{"email":"user@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["email"] != customer.Email {
		t.Errorf("expected email %q, got %q", customer.Email, resp["email"])
	}

	if !store.resetFailedCalled {
		t.Error("expected failed login counter to be reset")
	}
	if !store.createSessionCalled {
		t.Error("expected session to be created")
	}
	if !store.lastLoginUpdated {
		t.Error("expected last login to be updated")
	}

	// Verify session cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == portalctx.SessionCookieName {
			found = true
			if c.Value == "" {
				t.Error("session cookie value should not be empty")
			}
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
			if !c.Secure {
				t.Error("session cookie should be Secure")
			}
		}
	}
	if !found {
		t.Error("session cookie not found in response")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)

	body := `{"email":"user@example.com","password":"wrongpassword"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["error"] != "invalid email or password" {
		t.Errorf("unexpected error message: %v", resp["error"])
	}

	if !store.incrementFailedCalled {
		t.Error("expected failed login counter to be incremented")
	}
}

func TestLogin_NonexistentEmail(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"email":"nobody@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	resp := parseBody(t, w)
	// Should not reveal whether email exists
	if resp["error"] != "invalid email or password" {
		t.Errorf("expected generic error, got: %v", resp["error"])
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"email":"not-an-email","password":"short"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_EmptyBody(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	c := addTestCustomer(store, "locked@example.com", "password123", models.CustomerStatusActive)
	lockTime := time.Now().Add(15 * time.Minute)
	c.LockedUntil = &lockTime

	body := `{"email":"locked@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["error"] != "account is locked, try again later" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestLogin_LockoutAfterMaxFailedAttempts(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	c := addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)
	// Set failed attempts to one less than max so next failure triggers lockout
	c.FailedLoginAttempts = MaxFailedLoginAttempts - 1

	body := `{"email":"user@example.com","password":"wrongpassword"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["error"] != "account is locked due to too many failed attempts" {
		t.Errorf("unexpected error: %v", resp["error"])
	}

	if !store.lockCalled {
		t.Error("expected LockCustomerAccount to be called")
	}
	if !store.incrementFailedCalled {
		t.Error("expected IncrementCustomerFailedLogin to be called")
	}

	// Lock duration should be approximately LockoutDuration from now
	expectedLockTime := time.Now().Add(LockoutDuration)
	if store.lockUntil.Before(expectedLockTime.Add(-1*time.Second)) || store.lockUntil.After(expectedLockTime.Add(1*time.Second)) {
		t.Errorf("lock until time %v not within expected range of %v", store.lockUntil, expectedLockTime)
	}
}

func TestLogin_LockoutExpiry(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	c := addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)

	// Lockout that expired in the past
	pastLock := time.Now().Add(-1 * time.Minute)
	c.LockedUntil = &pastLock

	body := `{"email":"user@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	// Should succeed because lock has expired
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 (lock expired), got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InactiveAccount(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	addTestCustomer(store, "disabled@example.com", "password123", models.CustomerStatusDisabled)

	body := `{"email":"disabled@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["error"] != "account is not active" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestLogin_SessionCreationFailure(t *testing.T) {
	store := newMockStore()
	store.createSessionErr = fmt.Errorf("db down")
	h := newTestHandler(store)
	addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)

	body := `{"email":"user@example.com","password":"password123"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_Success(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"email":"new@example.com","name":"New User","password":"securepassword123"}`
	w := performRequest(h.Register, http.MethodPost, "/api/v1/auth/register", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["email"] != "new@example.com" {
		t.Errorf("expected email new@example.com, got %v", resp["email"])
	}
	if resp["name"] != "New User" {
		t.Errorf("expected name 'New User', got %v", resp["name"])
	}

	// Verify customer was stored
	if _, ok := store.customers["new@example.com"]; !ok {
		t.Error("customer was not stored in mock")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	addTestCustomer(store, "existing@example.com", "password123", models.CustomerStatusActive)

	body := `{"email":"existing@example.com","name":"Dup User","password":"securepassword123"}`
	w := performRequest(h.Register, http.MethodPost, "/api/v1/auth/register", body)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["error"] != "email already registered" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
}

func TestRegister_InvalidRequest(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	tests := []struct {
		name string
		body string
	}{
		{"missing email", `{"name":"User","password":"securepassword123"}`},
		{"missing name", `{"email":"u@example.com","password":"securepassword123"}`},
		{"missing password", `{"email":"u@example.com","name":"User"}`},
		{"short password", `{"email":"u@example.com","name":"User","password":"short"}`},
		{"invalid email", `{"email":"not-email","name":"User","password":"securepassword123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(h.Register, http.MethodPost, "/api/v1/auth/register", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRegister_StoreError(t *testing.T) {
	store := newMockStore()
	store.createCustomerErr = fmt.Errorf("db write error")
	h := newTestHandler(store)

	body := `{"email":"new@example.com","name":"New User","password":"securepassword123"}`
	w := performRequest(h.Register, http.MethodPost, "/api/v1/auth/register", body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogout_WithSession(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	// Create a session
	sessionID := uuid.New()
	token := "test-session-token"
	tokenHash := portalctx.HashSessionToken(token)
	store.sessions[tokenHash] = &portalctx.Session{
		ID:         sessionID,
		CustomerID: uuid.New(),
		TokenHash:  tokenHash,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		CreatedAt:  time.Now(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	c.Request.AddCookie(&http.Cookie{Name: portalctx.SessionCookieName, Value: token})
	h.Logout(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify session was deleted
	if _, ok := store.sessions[tokenHash]; ok {
		t.Error("expected session to be deleted from store")
	}

	// Verify cookie was cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == portalctx.SessionCookieName {
			if cookie.MaxAge >= 0 {
				t.Error("expected session cookie to be cleared (negative MaxAge)")
			}
		}
	}
}

func TestLogout_WithoutSession(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	w := performRequest(h.Logout, http.MethodPost, "/api/v1/auth/logout", "")

	// Logout should succeed even without a session
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestMe_Authenticated(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	customer := addTestCustomer(store, "me@example.com", "password123", models.CustomerStatusActive)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)

	// Inject authenticated customer into context
	sessionUser := &portalctx.SessionUser{
		ID:    customer.ID,
		Email: customer.Email,
		Name:  customer.Name,
	}
	c.Set(string(portalctx.CustomerContextKey), sessionUser)

	h.Me(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["email"] != "me@example.com" {
		t.Errorf("expected email me@example.com, got %v", resp["email"])
	}
}

func TestMe_Unauthenticated(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	w := performRequest(h.Me, http.MethodGet, "/api/v1/auth/me", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestForgotPassword_ExistingEmail(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	addTestCustomer(store, "forgot@example.com", "password123", models.CustomerStatusActive)

	body := `{"email":"forgot@example.com"}`
	w := performRequest(h.ForgotPassword, http.MethodPost, "/api/v1/auth/forgot-password", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseBody(t, w)
	if resp["message"] != "if the email exists, a reset link has been sent" {
		t.Errorf("unexpected message: %v", resp["message"])
	}

	if !store.resetTokenSaved {
		t.Error("expected reset token to be saved")
	}
	if len(store.resetTokenValue) != 64 {
		t.Errorf("expected 64-char hex token, got length %d", len(store.resetTokenValue))
	}
	if store.resetTokenExpiry.Before(time.Now()) {
		t.Error("reset token expiry should be in the future")
	}
}

func TestForgotPassword_NonexistentEmail(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"email":"nobody@example.com"}`
	w := performRequest(h.ForgotPassword, http.MethodPost, "/api/v1/auth/forgot-password", body)

	// Should return 200 even for nonexistent emails (don't reveal existence)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseBody(t, w)
	if resp["message"] != "if the email exists, a reset link has been sent" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestResetPassword_ValidToken(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	// 64-character hex token (32 bytes)
	token := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	body := fmt.Sprintf(`{"token":"%s","new_password":"newpassword123"}`, token)
	w := performRequest(h.ResetPassword, http.MethodPost, "/api/v1/auth/reset-password", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"token":"tooshort","new_password":"newpassword123"}`
	w := performRequest(h.ResetPassword, http.MethodPost, "/api/v1/auth/reset-password", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_Success(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	customer := addTestCustomer(store, "change@example.com", "oldpassword1", models.CustomerStatusActive)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		strings.NewReader(`{"current_password":"oldpassword1","new_password":"newpassword1"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	sessionUser := &portalctx.SessionUser{
		ID:    customer.ID,
		Email: customer.Email,
		Name:  customer.Name,
	}
	c.Set(string(portalctx.CustomerContextKey), sessionUser)

	h.ChangePassword(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if !store.passwordUpdated {
		t.Error("expected password to be updated")
	}
	if !store.sessionsDeleted {
		t.Error("expected other sessions to be invalidated")
	}

	// Verify the new password hash is correct
	expectedHash := models.HashPassword("newpassword1")
	if store.newPasswordHash != expectedHash {
		t.Error("stored password hash does not match expected hash for new password")
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)
	customer := addTestCustomer(store, "change@example.com", "oldpassword1", models.CustomerStatusActive)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password",
		strings.NewReader(`{"current_password":"wrongpassword","new_password":"newpassword1"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	sessionUser := &portalctx.SessionUser{
		ID:    customer.ID,
		Email: customer.Email,
		Name:  customer.Name,
	}
	c.Set(string(portalctx.CustomerContextKey), sessionUser)

	h.ChangePassword(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}

	if store.passwordUpdated {
		t.Error("password should NOT be updated when current password is wrong")
	}
}

func TestChangePassword_Unauthenticated(t *testing.T) {
	store := newMockStore()
	h := newTestHandler(store)

	body := `{"current_password":"old","new_password":"newpassword1"}`
	w := performRequest(h.ChangePassword, http.MethodPost, "/api/v1/auth/change-password", body)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "mysecretpassword"
	hash := models.HashPassword(password)

	t.Run("correct password matches", func(t *testing.T) {
		if !models.ComparePasswordHash(password, hash) {
			t.Error("correct password should match hash")
		}
	})

	t.Run("wrong password does not match", func(t *testing.T) {
		if models.ComparePasswordHash("wrongpassword", hash) {
			t.Error("wrong password should not match hash")
		}
	})

	t.Run("hash is deterministic", func(t *testing.T) {
		hash2 := models.HashPassword(password)
		if hash != hash2 {
			t.Error("same password should produce same hash")
		}
	})

	t.Run("different passwords produce different hashes", func(t *testing.T) {
		hash2 := models.HashPassword("differentpassword")
		if hash == hash2 {
			t.Error("different passwords should produce different hashes")
		}
	})
}

func TestSessionTokenGeneration(t *testing.T) {
	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := portalctx.GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		token2, err := portalctx.GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token1 == token2 {
			t.Error("two generated tokens should not be identical")
		}
	})

	t.Run("token has correct length", func(t *testing.T) {
		token, err := portalctx.GenerateSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 32 bytes = 64 hex chars
		if len(token) != 64 {
			t.Errorf("expected token length 64, got %d", len(token))
		}
	})

	t.Run("hash is deterministic", func(t *testing.T) {
		token := "test-token-value"
		hash1 := portalctx.HashSessionToken(token)
		hash2 := portalctx.HashSessionToken(token)
		if hash1 != hash2 {
			t.Error("hash should be deterministic")
		}
	})

	t.Run("different tokens produce different hashes", func(t *testing.T) {
		h1 := portalctx.HashSessionToken("token-a")
		h2 := portalctx.HashSessionToken("token-b")
		if h1 == h2 {
			t.Error("different tokens should produce different hashes")
		}
	})
}

func TestNewSession(t *testing.T) {
	customerID := uuid.New()
	session, token, err := portalctx.NewSession(customerID, "127.0.0.1", "TestAgent/1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.CustomerID != customerID {
		t.Errorf("expected customer ID %s, got %s", customerID, session.CustomerID)
	}
	if session.IPAddress != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", session.IPAddress)
	}
	if session.UserAgent != "TestAgent/1.0" {
		t.Errorf("expected user agent TestAgent/1.0, got %s", session.UserAgent)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
	if session.TokenHash == "" {
		t.Error("token hash should not be empty")
	}

	// Token hash should match what we get from hashing the token
	expectedHash := portalctx.HashSessionToken(token)
	if session.TokenHash != expectedHash {
		t.Error("stored token hash does not match hash of returned token")
	}

	// Session should not be expired
	if session.IsExpired() {
		t.Error("new session should not be expired")
	}

	// ExpiresAt should be roughly SessionDuration from now
	expectedExpiry := time.Now().Add(portalctx.SessionDuration)
	if session.ExpiresAt.Before(expectedExpiry.Add(-2*time.Second)) || session.ExpiresAt.After(expectedExpiry.Add(2*time.Second)) {
		t.Errorf("expiry %v not within expected range of %v", session.ExpiresAt, expectedExpiry)
	}
}

func TestSessionIsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		s := &portalctx.Session{ExpiresAt: time.Now().Add(1 * time.Hour)}
		if s.IsExpired() {
			t.Error("session should not be expired")
		}
	})

	t.Run("expired", func(t *testing.T) {
		s := &portalctx.Session{ExpiresAt: time.Now().Add(-1 * time.Hour)}
		if !s.IsExpired() {
			t.Error("session should be expired")
		}
	})
}

func TestCustomerIsLocked(t *testing.T) {
	t.Run("not locked (nil)", func(t *testing.T) {
		c := &models.Customer{}
		if c.IsLocked() {
			t.Error("customer with nil LockedUntil should not be locked")
		}
	})

	t.Run("locked (future)", func(t *testing.T) {
		future := time.Now().Add(10 * time.Minute)
		c := &models.Customer{LockedUntil: &future}
		if !c.IsLocked() {
			t.Error("customer locked until future should be locked")
		}
	})

	t.Run("unlocked (past)", func(t *testing.T) {
		past := time.Now().Add(-10 * time.Minute)
		c := &models.Customer{LockedUntil: &past}
		if c.IsLocked() {
			t.Error("customer with past LockedUntil should not be locked")
		}
	})
}

func TestLogin_FailedLoginProgressionToLockout(t *testing.T) {
	// Test that incrementing failed logins from 0 to max-1 does NOT lock,
	// then at max-1 a failure DOES lock.
	store := newMockStore()
	h := newTestHandler(store)
	c := addTestCustomer(store, "user@example.com", "password123", models.CustomerStatusActive)

	// Attempt with 0 failed attempts
	c.FailedLoginAttempts = 0
	body := `{"email":"user@example.com","password":"wrongpassword"}`
	w := performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if store.lockCalled {
		t.Error("should NOT lock on first failed attempt")
	}

	// Reset tracking
	store.lockCalled = false
	store.incrementFailedCalled = false

	// Attempt at max-1 (triggers lockout)
	c.FailedLoginAttempts = MaxFailedLoginAttempts - 1
	w = performRequest(h.Login, http.MethodPost, "/api/v1/auth/login", body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if !store.lockCalled {
		t.Error("should lock after reaching max failed attempts")
	}
}

func TestConstants(t *testing.T) {
	if MaxFailedLoginAttempts <= 0 {
		t.Error("MaxFailedLoginAttempts should be positive")
	}
	if LockoutDuration <= 0 {
		t.Error("LockoutDuration should be positive")
	}
	if ResetTokenDuration <= 0 {
		t.Error("ResetTokenDuration should be positive")
	}
}
