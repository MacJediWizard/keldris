package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockRegistrationCodeStore implements RegistrationCodeStore for testing.
type mockRegistrationCodeStore struct {
	mu               sync.Mutex
	codes            map[uuid.UUID]*models.RegistrationCode // keyed by code ID
	createErr        error
	getErr           error
	markUsedErr      error
	getPendingErr    error
	deleteExpiredErr error
}

func newMockRegistrationCodeStore() *mockRegistrationCodeStore {
	return &mockRegistrationCodeStore{
		codes: make(map[uuid.UUID]*models.RegistrationCode),
	}
}

func (m *mockRegistrationCodeStore) CreateRegistrationCode(_ context.Context, code *models.RegistrationCode) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[code.ID] = code
	return nil
}

func (m *mockRegistrationCodeStore) GetRegistrationCodeByCode(_ context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.codes {
		if c.OrgID == orgID && c.Code == code {
			return c, nil
		}
	}
	return nil, errors.New("code not found")
}

func (m *mockRegistrationCodeStore) GetPendingRegistrationCodes(_ context.Context, orgID uuid.UUID) ([]*models.RegistrationCode, error) {
	if m.getPendingErr != nil {
		return nil, m.getPendingErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var pending []*models.RegistrationCode
	for _, c := range m.codes {
		if c.OrgID == orgID && c.IsValid() {
			pending = append(pending, c)
		}
	}
	return pending, nil
}

func (m *mockRegistrationCodeStore) MarkRegistrationCodeUsed(_ context.Context, codeID, agentID uuid.UUID) error {
	if m.markUsedErr != nil {
		return m.markUsedErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.codes[codeID]
	if !ok {
		return errors.New("not found")
	}
	c.MarkUsed(agentID)
	return nil
}

func (m *mockRegistrationCodeStore) DeleteExpiredRegistrationCodes(_ context.Context) error {
	if m.deleteExpiredErr != nil {
		return m.deleteExpiredErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.codes {
		if c.IsExpired() {
			delete(m.codes, id)
		}
	}
	return nil
}

func newTestAgentMFA(t *testing.T) (*AgentMFA, *mockRegistrationCodeStore) {
	t.Helper()
	store := newMockRegistrationCodeStore()
	return NewAgentMFA(store, zerolog.Nop()), store
}

func TestGenerateRandomCode(t *testing.T) {
	t.Run("length matches request", func(t *testing.T) {
		code, err := generateRandomCode(RegistrationCodeLength)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(code) != RegistrationCodeLength {
			t.Errorf("len=%d, want %d", len(code), RegistrationCodeLength)
		}
	})

	t.Run("characters from charset", func(t *testing.T) {
		code, err := generateRandomCode(32)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, c := range code {
			if !strings.ContainsRune(RegistrationCodeChars, c) {
				t.Errorf("char %q not in charset", c)
			}
		}
	})

	t.Run("two codes differ", func(t *testing.T) {
		a, _ := generateRandomCode(16)
		b, _ := generateRandomCode(16)
		if a == b {
			t.Error("two random codes were identical")
		}
	})
}

func TestAgentMFA_GenerateCode(t *testing.T) {
	mfa, store := newTestAgentMFA(t)
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("creates and stores code", func(t *testing.T) {
		hostname := "host1"
		regCode, err := mfa.GenerateCode(context.Background(), orgID, userID, &hostname)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if regCode.OrgID != orgID {
			t.Errorf("OrgID = %v, want %v", regCode.OrgID, orgID)
		}
		if regCode.CreatedBy != userID {
			t.Errorf("CreatedBy = %v, want %v", regCode.CreatedBy, userID)
		}
		if len(regCode.Code) != RegistrationCodeLength {
			t.Errorf("Code length = %d, want %d", len(regCode.Code), RegistrationCodeLength)
		}
		if regCode.ExpiresAt.Before(time.Now()) {
			t.Error("ExpiresAt should be in the future")
		}
		// Should be in store
		store.mu.Lock()
		_, ok := store.codes[regCode.ID]
		store.mu.Unlock()
		if !ok {
			t.Error("code not stored")
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		failingStore := newMockRegistrationCodeStore()
		failingStore.createErr = errors.New("db failure")
		failingMFA := NewAgentMFA(failingStore, zerolog.Nop())
		_, err := failingMFA.GenerateCode(context.Background(), orgID, userID, nil)
		if err == nil {
			t.Error("expected error from store")
		}
	})
}

func TestAgentMFA_VerifyCode(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("valid code succeeds", func(t *testing.T) {
		mfa, _ := newTestAgentMFA(t)
		regCode, err := mfa.GenerateCode(ctx, orgID, userID, nil)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		got, err := mfa.VerifyCode(ctx, orgID, regCode.Code)
		if err != nil {
			t.Fatalf("VerifyCode: %v", err)
		}
		if got.ID != regCode.ID {
			t.Errorf("ID mismatch: got %v want %v", got.ID, regCode.ID)
		}
	})

	t.Run("normalizes lowercase + whitespace", func(t *testing.T) {
		mfa, _ := newTestAgentMFA(t)
		regCode, err := mfa.GenerateCode(ctx, orgID, userID, nil)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		// Codes are already uppercase from charset, so lowercase the code and add spaces
		mangled := "  " + strings.ToLower(regCode.Code) + "  "
		got, err := mfa.VerifyCode(ctx, orgID, mangled)
		if err != nil {
			t.Fatalf("VerifyCode (normalized): %v", err)
		}
		if got.ID != regCode.ID {
			t.Errorf("ID mismatch: got %v want %v", got.ID, regCode.ID)
		}
	})

	t.Run("unknown code returns error", func(t *testing.T) {
		mfa, _ := newTestAgentMFA(t)
		_, err := mfa.VerifyCode(ctx, orgID, "NOTFOUND")
		if err == nil {
			t.Error("expected error for unknown code")
		}
	})

	t.Run("used code rejected", func(t *testing.T) {
		mfa, store := newTestAgentMFA(t)
		regCode, _ := mfa.GenerateCode(ctx, orgID, userID, nil)
		// Mark used
		store.mu.Lock()
		now := time.Now()
		store.codes[regCode.ID].UsedAt = &now
		store.mu.Unlock()
		_, err := mfa.VerifyCode(ctx, orgID, regCode.Code)
		if err == nil {
			t.Error("expected error for used code")
		}
	})

	t.Run("expired code rejected", func(t *testing.T) {
		mfa, store := newTestAgentMFA(t)
		regCode, _ := mfa.GenerateCode(ctx, orgID, userID, nil)
		// Backdate expiry
		store.mu.Lock()
		store.codes[regCode.ID].ExpiresAt = time.Now().Add(-1 * time.Hour)
		store.mu.Unlock()
		_, err := mfa.VerifyCode(ctx, orgID, regCode.Code)
		if err == nil {
			t.Error("expected error for expired code")
		}
	})
}

func TestAgentMFA_MarkCodeUsed(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	agentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mfa, store := newTestAgentMFA(t)
		regCode, _ := mfa.GenerateCode(ctx, orgID, userID, nil)
		if err := mfa.MarkCodeUsed(ctx, regCode.ID, agentID); err != nil {
			t.Fatalf("MarkCodeUsed: %v", err)
		}
		store.mu.Lock()
		used := store.codes[regCode.ID].IsUsed()
		store.mu.Unlock()
		if !used {
			t.Error("expected code to be marked used")
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		store := newMockRegistrationCodeStore()
		store.markUsedErr = errors.New("db error")
		mfa := NewAgentMFA(store, zerolog.Nop())
		err := mfa.MarkCodeUsed(ctx, uuid.New(), agentID)
		if err == nil {
			t.Error("expected error from store")
		}
	})
}

func TestAgentMFA_GetPendingCodes(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	mfa, _ := newTestAgentMFA(t)

	// Add several codes
	for i := 0; i < 3; i++ {
		if _, err := mfa.GenerateCode(ctx, orgID, userID, nil); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	pending, err := mfa.GetPendingCodes(ctx, orgID)
	if err != nil {
		t.Fatalf("GetPendingCodes: %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("len(pending) = %d, want 3", len(pending))
	}
}

func TestAgentMFA_CleanupExpiredCodes(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()
	mfa, store := newTestAgentMFA(t)

	// Create one valid, one expired
	valid, _ := mfa.GenerateCode(ctx, orgID, userID, nil)
	expired, _ := mfa.GenerateCode(ctx, orgID, userID, nil)
	store.mu.Lock()
	store.codes[expired.ID].ExpiresAt = time.Now().Add(-1 * time.Hour)
	store.mu.Unlock()

	if err := mfa.CleanupExpiredCodes(ctx); err != nil {
		t.Fatalf("CleanupExpiredCodes: %v", err)
	}

	store.mu.Lock()
	_, validExists := store.codes[valid.ID]
	_, expiredExists := store.codes[expired.ID]
	store.mu.Unlock()

	if !validExists {
		t.Error("valid code should remain")
	}
	if expiredExists {
		t.Error("expired code should be deleted")
	}
}
