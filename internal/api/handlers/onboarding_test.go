package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockOnboardingStore struct {
	progress  *models.OnboardingProgress
	getErr    error
	updateErr error
	skipErr   error
}

func (m *mockOnboardingStore) GetOnboardingProgress(_ context.Context, _ uuid.UUID) (*models.OnboardingProgress, error) {
	return m.progress, m.getErr
}

func (m *mockOnboardingStore) GetOrCreateOnboardingProgress(_ context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.progress != nil {
		return m.progress, nil
	}
	return models.NewOnboardingProgress(orgID), nil
}

func (m *mockOnboardingStore) UpdateOnboardingProgress(_ context.Context, _ *models.OnboardingProgress) error {
	return m.updateErr
}

func (m *mockOnboardingStore) SkipOnboarding(_ context.Context, _ uuid.UUID) error {
	return m.skipErr
}

func setupOnboardingTestRouter(store OnboardingStore, orgID uuid.UUID) *gin.Engine {
	user := TestUser(orgID)
	r := SetupTestRouter(user)
	handler := NewOnboardingHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestOnboardingGetStatus(t *testing.T) {
	orgID := uuid.New()

	t.Run("new org needs onboarding", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=true for new org")
		}
		if resp.CurrentStep != models.OnboardingStepWelcome {
			t.Fatalf("expected welcome step, got %q", resp.CurrentStep)
		}
	})

	t.Run("completed onboarding", func(t *testing.T) {
		progress := models.NewOnboardingProgress(orgID)
		progress.Skip()
		store := &mockOnboardingStore{progress: progress}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=false")
		}
		if !resp.IsComplete {
			t.Fatal("expected is_complete=true")
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := SetupTestRouter(nil)
		handler := NewOnboardingHandler(store, zerolog.Nop())
		api := r.Group("/api/v1")
		handler.RegisterRoutes(api)

		w := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/onboarding/status"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestOnboardingCompleteStep(t *testing.T) {
	orgID := uuid.New()

	t.Run("valid step", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.CurrentStep != models.OnboardingStepLicense {
			t.Fatalf("expected step to advance to license, got %q", resp.CurrentStep)
		}
	})

	t.Run("invalid step", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/bogus", "{}"))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store get error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store update error", func(t *testing.T) {
		store := &mockOnboardingStore{updateErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/step/welcome", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestOnboardingSkip(t *testing.T) {
	orgID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockOnboardingStore{}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp OnboardingStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.Skipped {
			t.Fatal("expected skipped=true")
		}
		if resp.NeedsOnboarding {
			t.Fatal("expected needs_onboarding=false after skip")
		}
	})

	t.Run("store get error", func(t *testing.T) {
		store := &mockOnboardingStore{getErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})

	t.Run("store skip error", func(t *testing.T) {
		store := &mockOnboardingStore{skipErr: errors.New("db error")}
		r := setupOnboardingTestRouter(store, orgID)

		w := DoRequest(r, JSONRequest("POST", "/api/v1/onboarding/skip", "{}"))

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}
