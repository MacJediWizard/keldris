package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/branding"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BrandingStore defines the interface for branding persistence operations.
type BrandingStore interface {
	GetBrandingSettings(ctx context.Context, orgID uuid.UUID) (*models.BrandingSettings, error)
	UpsertBrandingSettings(ctx context.Context, b *models.BrandingSettings) error
	DeleteBrandingSettings(ctx context.Context, orgID uuid.UUID) error
}

// BrandingHandler handles branding-related HTTP endpoints.
// BrandingStore defines the interface for branding settings persistence operations.
type BrandingStore interface {
	GetBrandingSettings(ctx context.Context, orgID uuid.UUID) (*branding.BrandingSettings, error)
	UpdateBrandingSettings(ctx context.Context, orgID uuid.UUID, b *branding.BrandingSettings) error
	GetPublicBrandingSettings(ctx context.Context, orgSlug string) (*branding.PublicBrandingSettings, error)
	CreateSettingsAuditLog(ctx context.Context, log *settings.SettingsAuditLog) error
	HasFeatureFlag(ctx context.Context, orgID uuid.UUID, flag string) (bool, error)
}

// BrandingHandler handles branding settings HTTP endpoints.
type BrandingHandler struct {
	store  BrandingStore
	logger zerolog.Logger
}

// NewBrandingHandler creates a new BrandingHandler.
func NewBrandingHandler(store BrandingStore, logger zerolog.Logger) *BrandingHandler {
	return &BrandingHandler{
		store:  store,
		logger: logger.With().Str("component", "branding_handler").Logger(),
	}
}

// RegisterRoutes registers branding routes on the given router group.
func (h *BrandingHandler) RegisterRoutes(r *gin.RouterGroup) {
	b := r.Group("/branding")
	{
		b.GET("", h.Get)
		b.PUT("", h.Update)
		b.DELETE("", h.Reset)
	}
}

// UpdateBrandingRequest is the request body for updating branding settings.
type UpdateBrandingRequest struct {
	LogoURL        *string `json:"logo_url,omitempty"`
	FaviconURL     *string `json:"favicon_url,omitempty"`
	ProductName    *string `json:"product_name,omitempty"`
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	SupportURL     *string `json:"support_url,omitempty"`
	CustomCSS      *string `json:"custom_css,omitempty"`
}

// Get returns the branding settings for the current organization.
// RegisterRoutes registers branding settings routes on the given router group.
func (h *BrandingHandler) RegisterRoutes(r *gin.RouterGroup) {
	brandingGroup := r.Group("/branding")
	{
		brandingGroup.GET("", h.Get)
		brandingGroup.PUT("", h.Update)
	}
}

// RegisterPublicRoutes registers public branding routes (no auth required).
func (h *BrandingHandler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/branding/:orgSlug", h.GetPublic)
}

// Get returns branding settings for the organization.
// GET /api/v1/branding
func (h *BrandingHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	settings, err := branding.LoadBranding(c.Request.Context(), h.store, user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to load branding")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load branding settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// Update creates or updates branding settings for the current organization.
	// Only admins/owners can view branding settings
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Check if organization has custom_branding feature flag
	hasBranding, err := h.store.HasFeatureFlag(c.Request.Context(), user.CurrentOrgID, "custom_branding")
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to check feature flag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature access"})
		return
	}

	if !hasBranding {
		c.JSON(http.StatusForbidden, gin.H{"error": "custom branding is an Enterprise feature"})
		return
	}

	b, err := h.store.GetBrandingSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get branding settings"})
		return
	}

	c.JSON(http.StatusOK, b)
}

// Update updates branding settings for the organization.
// PUT /api/v1/branding
func (h *BrandingHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate colors
	if req.PrimaryColor != nil {
		if err := branding.ValidateColor(*req.PrimaryColor); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.SecondaryColor != nil {
		if err := branding.ValidateColor(*req.SecondaryColor); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Validate URLs
	if req.LogoURL != nil {
		if err := branding.ValidateURL(*req.LogoURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.FaviconURL != nil {
		if err := branding.ValidateURL(*req.FaviconURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.SupportURL != nil {
		if err := branding.ValidateURL(*req.SupportURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Load existing or create new
	ctx := c.Request.Context()
	existing, err := h.store.GetBrandingSettings(ctx, user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load branding settings"})
		return
	}

	settings := existing
	if settings == nil {
		settings = models.NewBrandingSettings(user.CurrentOrgID)
	}

	// Apply updates
	if req.LogoURL != nil {
		settings.LogoURL = *req.LogoURL
	}
	if req.FaviconURL != nil {
		settings.FaviconURL = *req.FaviconURL
	}
	if req.ProductName != nil {
		settings.ProductName = *req.ProductName
	}
	if req.PrimaryColor != nil {
		settings.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		settings.SecondaryColor = *req.SecondaryColor
	}
	if req.SupportURL != nil {
		settings.SupportURL = *req.SupportURL
	}
	if req.CustomCSS != nil {
		settings.CustomCSS = *req.CustomCSS
	}

	if err := h.store.UpsertBrandingSettings(ctx, settings); err != nil {
		h.logger.Error().Err(err).Msg("failed to update branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update branding settings"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Msg("branding settings updated")

	c.JSON(http.StatusOK, settings)
}

// Reset removes all custom branding, reverting to defaults.
// DELETE /api/v1/branding
func (h *BrandingHandler) Reset(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if err := h.store.DeleteBrandingSettings(c.Request.Context(), user.CurrentOrgID); err != nil {
		h.logger.Error().Err(err).Msg("failed to reset branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset branding settings"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Msg("branding settings reset to defaults")

	c.JSON(http.StatusOK, gin.H{"message": "branding settings reset to defaults"})
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Check if organization has custom_branding feature flag
	hasBranding, err := h.store.HasFeatureFlag(c.Request.Context(), user.CurrentOrgID, "custom_branding")
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to check feature flag")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature access"})
		return
	}

	if !hasBranding {
		c.JSON(http.StatusForbidden, gin.H{"error": "custom branding is an Enterprise feature"})
		return
	}

	var req branding.UpdateBrandingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current settings for audit log and merging
	current, err := h.store.GetBrandingSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get current branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current settings"})
		return
	}

	// Store old value for audit
	oldValue, _ := json.Marshal(current)

	// Apply updates
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.ProductName != nil {
		current.ProductName = *req.ProductName
	}
	if req.CompanyName != nil {
		current.CompanyName = *req.CompanyName
	}
	if req.LogoURL != nil {
		current.LogoURL = *req.LogoURL
	}
	if req.LogoDarkURL != nil {
		current.LogoDarkURL = *req.LogoDarkURL
	}
	if req.FaviconURL != nil {
		current.FaviconURL = *req.FaviconURL
	}
	if req.PrimaryColor != nil {
		current.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		current.SecondaryColor = *req.SecondaryColor
	}
	if req.AccentColor != nil {
		current.AccentColor = *req.AccentColor
	}
	if req.SupportURL != nil {
		current.SupportURL = *req.SupportURL
	}
	if req.SupportEmail != nil {
		current.SupportEmail = *req.SupportEmail
	}
	if req.PrivacyURL != nil {
		current.PrivacyURL = *req.PrivacyURL
	}
	if req.TermsURL != nil {
		current.TermsURL = *req.TermsURL
	}
	if req.FooterText != nil {
		current.FooterText = *req.FooterText
	}
	if req.LoginTitle != nil {
		current.LoginTitle = *req.LoginTitle
	}
	if req.LoginSubtitle != nil {
		current.LoginSubtitle = *req.LoginSubtitle
	}
	if req.LoginBgURL != nil {
		current.LoginBgURL = *req.LoginBgURL
	}
	if req.HidePoweredBy != nil {
		current.HidePoweredBy = *req.HidePoweredBy
	}
	if req.CustomCSS != nil {
		current.CustomCSS = *req.CustomCSS
	}

	// Validate
	if err := current.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save
	if err := h.store.UpdateBrandingSettings(c.Request.Context(), user.CurrentOrgID, current); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update branding settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	// Create audit log
	newValue, _ := json.Marshal(current)

	auditLog := settings.NewSettingsAuditLog(
		user.CurrentOrgID,
		"branding",
		oldValue,
		newValue,
		user.ID,
		getClientIP(c),
	)
	if err := h.store.CreateSettingsAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create branding audit log")
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("branding settings updated")

	c.JSON(http.StatusOK, current)
}

// GetPublic returns public branding settings for the login page.
// This endpoint does not require authentication.
// GET /api/public/branding/:orgSlug
func (h *BrandingHandler) GetPublic(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	if orgSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization slug is required"})
		return
	}

	b, err := h.store.GetPublicBrandingSettings(c.Request.Context(), orgSlug)
	if err != nil {
		h.logger.Error().Err(err).Str("org_slug", orgSlug).Msg("failed to get public branding settings")
		// Return default branding on error
		defaults := branding.DefaultBrandingSettings()
		c.JSON(http.StatusOK, defaults.ToPublic())
		return
	}

	c.JSON(http.StatusOK, b)
}
