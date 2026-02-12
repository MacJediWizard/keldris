package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/branding"
	"github.com/MacJediWizard/keldris/internal/models"
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
}
