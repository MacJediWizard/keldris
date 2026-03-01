package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
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
	// OIDC settings for onboarding step
	GetOIDCSettings(ctx context.Context, orgID uuid.UUID) (*settings.OIDCSettings, error)
	UpdateOIDCSettings(ctx context.Context, orgID uuid.UUID, oidc *settings.OIDCSettings) error
	EnsureSystemSettingsExist(ctx context.Context, orgID uuid.UUID) error
}

// OnboardingHandler handles onboarding-related HTTP endpoints.
type OnboardingHandler struct {
	store        OnboardingStore
	checker      *license.FeatureChecker
	oidcProvider *auth.OIDCProvider
	logger       zerolog.Logger
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(store OnboardingStore, checker *license.FeatureChecker, oidcProvider *auth.OIDCProvider, logger zerolog.Logger) *OnboardingHandler {
	return &OnboardingHandler{
		store:        store,
		checker:      checker,
		oidcProvider: oidcProvider,
		logger:       logger.With().Str("component", "onboarding_handler").Logger(),
	}
}

// RegisterRoutes registers onboarding routes on the given router group.
func (h *OnboardingHandler) RegisterRoutes(r *gin.RouterGroup) {
	onboarding := r.Group("/onboarding")
	{
		onboarding.GET("/status", h.GetStatus)
		onboarding.POST("/step/:step", h.CompleteStep)
		onboarding.POST("/skip", h.Skip)
		onboarding.POST("/test-oidc", h.TestOIDC)
	}
}

// OnboardingStatusResponse is the response for the onboarding status endpoint.
type OnboardingStatusResponse struct {
	NeedsOnboarding bool                    `json:"needs_onboarding"`
	CurrentStep     models.OnboardingStep   `json:"current_step"`
	CompletedSteps  []models.OnboardingStep `json:"completed_steps"`
	Skipped         bool                    `json:"skipped"`
	IsComplete      bool                    `json:"is_complete"`
	LicenseTier     string                  `json:"license_tier,omitempty"`
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

	// Get license tier from gin context (set by DynamicLicenseMiddleware)
	var licenseTier string
	if lic := middleware.GetLicense(c); lic != nil {
		licenseTier = string(lic.Tier)
	}

	c.JSON(http.StatusOK, OnboardingStatusResponse{
		NeedsOnboarding: !progress.IsComplete(),
		CurrentStep:     progress.CurrentStep,
		CompletedSteps:  progress.CompletedSteps,
		Skipped:         progress.Skipped,
		IsComplete:      progress.IsComplete(),
		LicenseTier:     licenseTier,
	})
}

// OIDCOnboardingRequest is the request body for the OIDC onboarding step.
type OIDCOnboardingRequest struct {
	Issuer       string `json:"issuer" binding:"required,url"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	RedirectURL  string `json:"redirect_url" binding:"required,url"`
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

	// Handle OIDC step specially — only require feature gating when
	// actually configuring OIDC (request has a body). A skip (no body)
	// just marks the step complete without feature checks.
	if step == models.OnboardingStepOIDC && c.Request.ContentLength > 0 {
		h.completeOIDCStep(c, user)
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

// completeOIDCStep handles the OIDC onboarding step with feature gating and provider hot-reload.
func (h *OnboardingHandler) completeOIDCStep(c *gin.Context, user *auth.SessionUser) {
	// All 3 security layers enforced via RequireFeature:
	//   Layer 1: Org tier from DB (via FeatureChecker.CheckFeatureWithInfo)
	//   Layer 2: Entitlement nonce (proves license server issued a token)
	//   Layer 3: Refresh token (proves recent heartbeat)
	// The DB tier is updated by LicenseManageHandler.Activate/StartTrial
	// before the user reaches this step. Layers 2+3 are delivered by the
	// activation and heartbeat calls in SetLicenseKey.
	if !middleware.RequireFeature(c, h.checker, license.FeatureOIDC) {
		return
	}

	// Parse OIDC settings from request body
	var req OIDCOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build OIDC settings for storage
	oidcSettings := &settings.OIDCSettings{
		Enabled:      true,
		Issuer:       req.Issuer,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURL:  req.RedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		DefaultRole:  "member",
	}

	// Validate
	if err := oidcSettings.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Ensure system settings row exists for this org
	if err := h.store.EnsureSystemSettingsExist(ctx, user.CurrentOrgID); err != nil {
		h.logger.Warn().Err(err).Msg("failed to ensure system settings exist")
	}

	// Save OIDC settings to DB
	if err := h.store.UpdateOIDCSettings(ctx, user.CurrentOrgID, oidcSettings); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to save OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save OIDC settings"})
		return
	}

	// Hot-reload the OIDC provider
	if h.oidcProvider != nil {
		oidcCfg := auth.DefaultOIDCConfig(req.Issuer, req.ClientID, req.ClientSecret, req.RedirectURL)
		if err := h.oidcProvider.Update(ctx, oidcCfg); err != nil {
			h.logger.Error().Err(err).Msg("failed to hot-reload OIDC provider (settings saved, restart may be needed)")
			// Don't fail the step — settings are saved, provider will load on restart
		} else {
			h.logger.Info().Str("issuer", req.Issuer).Msg("OIDC provider hot-reloaded from onboarding")
		}
	}

	// Mark step complete
	progress, err := h.store.GetOrCreateOnboardingProgress(ctx, user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get onboarding progress"})
		return
	}

	progress.CompleteStep(models.OnboardingStepOIDC)

	if err := h.store.UpdateOnboardingProgress(ctx, progress); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update onboarding progress")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update onboarding progress"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("issuer", req.Issuer).
		Msg("completed OIDC onboarding step")

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

// TestOIDC tests an OIDC configuration by performing provider discovery.
// POST /api/v1/onboarding/test-oidc
func (h *OnboardingHandler) TestOIDC(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// All 3 security layers enforced via RequireFeature (same as completeOIDCStep).
	if !middleware.RequireFeature(c, h.checker, license.FeatureOIDC) {
		return
	}

	var req OIDCOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the settings struct (set defaults for fields not in the request)
	oidcSettings := &settings.OIDCSettings{
		Enabled:      true,
		Issuer:       req.Issuer,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		RedirectURL:  req.RedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		DefaultRole:  "member",
	}
	if err := oidcSettings.Validate(); err != nil {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: "Configuration validation failed: " + err.Error(),
		})
		return
	}

	// Perform OIDC discovery against the issuer
	discoveryURL := strings.TrimRight(req.Issuer, "/") + "/.well-known/openid-configuration"
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(discoveryURL) //nolint:gosec // Issuer URL is user-provided for OIDC discovery
	if err != nil {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to reach OIDC provider: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: fmt.Sprintf("OIDC discovery returned status %d", resp.StatusCode),
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: "Failed to read OIDC discovery response",
		})
		return
	}

	var discovery struct {
		Issuer   string `json:"issuer"`
		AuthURL  string `json:"authorization_endpoint"`
		TokenURL string `json:"token_endpoint"`
	}
	if err := json.Unmarshal(body, &discovery); err != nil {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: "Invalid OIDC discovery document",
		})
		return
	}

	if discovery.Issuer == "" || discovery.AuthURL == "" {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: "OIDC discovery document missing required fields (issuer, authorization_endpoint)",
		})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("issuer", req.Issuer).
		Msg("OIDC test connection successful during onboarding")

	c.JSON(http.StatusOK, settings.TestOIDCResponse{
		Success:       true,
		Message:       "OIDC provider discovery successful",
		ProviderName:  discovery.Issuer,
		AuthURL:       discovery.AuthURL,
		SupportedFlow: "authorization_code",
	})
}
