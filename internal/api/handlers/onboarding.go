package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// OnboardingStore defines the interface for onboarding persistence operations.
type OnboardingStore interface {
	GetOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error)
	GetOrCreateOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error)
	UpdateOnboardingProgress(ctx context.Context, progress *models.OnboardingProgress) error
	SkipOnboarding(ctx context.Context, orgID uuid.UUID) error
}

// OnboardingHandler handles onboarding-related HTTP endpoints.
type OnboardingHandler struct {
	store  OnboardingStore
	logger zerolog.Logger
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(store OnboardingStore, logger zerolog.Logger) *OnboardingHandler {
	return &OnboardingHandler{
		store:  store,
		logger: logger.With().Str("component", "onboarding_handler").Logger(),
	}
}

// RegisterRoutes registers onboarding routes on the given router group.
func (h *OnboardingHandler) RegisterRoutes(r *gin.RouterGroup) {
	onboarding := r.Group("/onboarding")
	{
		onboarding.GET("/status", h.GetStatus)
		onboarding.POST("/step/:step", h.CompleteStep)
		onboarding.POST("/skip", h.Skip)
	}
}

// OnboardingStatusResponse is the response for the onboarding status endpoint.
type OnboardingStatusResponse struct {
	NeedsOnboarding bool                    `json:"needs_onboarding"`
	CurrentStep     models.OnboardingStep   `json:"current_step"`
	CompletedSteps  []models.OnboardingStep `json:"completed_steps"`
	Skipped         bool                    `json:"skipped"`
	IsComplete      bool                    `json:"is_complete"`
	NeedsOnboarding bool                     `json:"needs_onboarding"`
	CurrentStep     models.OnboardingStep    `json:"current_step"`
	CompletedSteps  []models.OnboardingStep  `json:"completed_steps"`
	Skipped         bool                     `json:"skipped"`
	IsComplete      bool                     `json:"is_complete"`
}

// GetStatus returns the onboarding status for the current organization.
// GET /api/v1/onboarding/status
func (h *OnboardingHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	progress, err := h.store.GetOrCreateOnboardingProgress(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get onboarding progress"})
		return
	}

	c.JSON(http.StatusOK, OnboardingStatusResponse{
		NeedsOnboarding: !progress.IsComplete(),
		CurrentStep:     progress.CurrentStep,
		CompletedSteps:  progress.CompletedSteps,
		Skipped:         progress.Skipped,
		IsComplete:      progress.IsComplete(),
	})
}

// CompleteStep marks a step as completed.
// POST /api/v1/onboarding/step/:step
func (h *OnboardingHandler) CompleteStep(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	step := models.OnboardingStep(c.Param("step"))

	// Validate step
	validStep := false
	for _, s := range models.OnboardingSteps {
		if s == step {
			validStep = true
			break
		}
	}
	if !validStep {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid step"})
		return
	}

	progress, err := h.store.GetOrCreateOnboardingProgress(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get onboarding progress"})
		return
	}

	progress.CompleteStep(step)

	if err := h.store.UpdateOnboardingProgress(c.Request.Context(), progress); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update onboarding progress"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("step", string(step)).
		Msg("completed onboarding step")

	c.JSON(http.StatusOK, OnboardingStatusResponse{
		NeedsOnboarding: !progress.IsComplete(),
		CurrentStep:     progress.CurrentStep,
		CompletedSteps:  progress.CompletedSteps,
		Skipped:         progress.Skipped,
		IsComplete:      progress.IsComplete(),
	})
}

// Skip marks the onboarding as skipped.
// POST /api/v1/onboarding/skip
func (h *OnboardingHandler) Skip(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Ensure progress record exists
	progress, err := h.store.GetOrCreateOnboardingProgress(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get onboarding progress"})
		return
	}

	if err := h.store.SkipOnboarding(c.Request.Context(), user.CurrentOrgID); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to skip onboarding")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to skip onboarding"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Msg("skipped onboarding")

	// Return updated status
	progress.Skip()
	c.JSON(http.StatusOK, OnboardingStatusResponse{
		NeedsOnboarding: false,
		CurrentStep:     progress.CurrentStep,
		CompletedSteps:  progress.CompletedSteps,
		Skipped:         true,
		IsComplete:      true,
	})
}
