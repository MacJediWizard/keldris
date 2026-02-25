package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LicenseStore defines the interface for license persistence operations.
type LicenseStore interface {
	GetOrgTier(ctx context.Context, orgID uuid.UUID) (license.Tier, error)
}

// LicenseHandler handles license and feature-related HTTP endpoints.
type LicenseHandler struct {
	store   LicenseStore
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewLicenseHandler creates a new LicenseHandler.
func NewLicenseHandler(store LicenseStore, checker *license.FeatureChecker, logger zerolog.Logger) *LicenseHandler {
	return &LicenseHandler{
		store:   store,
		checker: checker,
		logger:  logger.With().Str("component", "license_handler").Logger(),
	}
}

// RegisterRoutes registers license routes on the given router group.
// Note: GET /license is handled by LicenseInfoHandler (registered separately).
func (h *LicenseHandler) RegisterRoutes(r *gin.RouterGroup) {
	lic := r.Group("/license")
	{
		lic.GET("/features", h.ListFeatures)
		lic.GET("/features/:feature/check", h.CheckFeature)
		lic.GET("/tiers", h.ListTiers)
	}
}

// LicenseInfo represents the license information for an organization.
type LicenseInfo struct {
	OrgID    uuid.UUID         `json:"org_id"`
	Tier     license.Tier      `json:"tier"`
	Features []license.Feature `json:"features"`
}

// LicenseResponse is the response for the GetLicense endpoint.
type LicenseResponse struct {
	License LicenseInfo `json:"license"`
}

// GetLicense returns the current organization's license information.
// GET /api/v1/license
func (h *LicenseHandler) GetLicense(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	tier, err := h.checker.GetOrgTier(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get org tier")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve license"})
		return
	}

	features := license.GetTierFeatures(tier)

	c.JSON(http.StatusOK, LicenseResponse{
		License: LicenseInfo{
			OrgID:    user.CurrentOrgID,
			Tier:     tier,
			Features: features,
		},
	})
}

// FeaturesResponse is the response for the ListFeatures endpoint.
type FeaturesResponse struct {
	Features []license.FeatureInfo `json:"features"`
}

// ListFeatures returns all available features with their tier requirements.
// GET /api/v1/license/features
func (h *LicenseHandler) ListFeatures(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	features := license.GetAllFeatureInfo()
	c.JSON(http.StatusOK, FeaturesResponse{
		Features: features,
	})
}

// FeatureCheckResponse is the response for the CheckFeature endpoint.
type FeatureCheckResponse struct {
	Result *license.FeatureCheckResult `json:"result"`
}

// CheckFeature checks if the organization has access to a specific feature.
// GET /api/v1/license/features/:feature/check
func (h *LicenseHandler) CheckFeature(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	featureName := c.Param("feature")
	feature := license.Feature(featureName)

	// Validate feature name
	validFeature := false
	for _, f := range license.AllFeatures() {
		if f == feature {
			validFeature = true
			break
		}
	}
	if !validFeature {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature name"})
		return
	}

	result, err := h.checker.CheckFeatureWithInfo(c.Request.Context(), user.CurrentOrgID, feature)
	if err != nil {
		h.logger.Error().Err(err).
			Str("org_id", user.CurrentOrgID.String()).
			Str("feature", featureName).
			Msg("failed to check feature")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature"})
		return
	}

	c.JSON(http.StatusOK, FeatureCheckResponse{
		Result: result,
	})
}

// TiersResponse is the response for the ListTiers endpoint.
type TiersResponse struct {
	Tiers []license.TierInfo `json:"tiers"`
}

// ListTiers returns all available tiers with their features.
// GET /api/v1/license/tiers
func (h *LicenseHandler) ListTiers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	tiers := license.GetAllTierInfo()
	c.JSON(http.StatusOK, TiersResponse{
		Tiers: tiers,
	})
}
