package handlers

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LicenseInfoResponse is the JSON response for the license info endpoint.
type LicenseInfoResponse struct {
	Tier       string            `json:"tier"`
	CustomerID string            `json:"customer_id"`
	ExpiresAt  string            `json:"expires_at"`
	IssuedAt   string            `json:"issued_at"`
	Features   []string          `json:"features"`
	Limits     license.TierLimits `json:"limits"`
}

// LicenseInfoHandler handles license information endpoints.
type LicenseInfoHandler struct {
	logger zerolog.Logger
}

// NewLicenseInfoHandler creates a new LicenseInfoHandler.
func NewLicenseInfoHandler(logger zerolog.Logger) *LicenseInfoHandler {
	return &LicenseInfoHandler{
		logger: logger.With().Str("component", "license_info_handler").Logger(),
	}
}

// RegisterRoutes registers license info routes on the given router group.
func (h *LicenseInfoHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/license", h.Get)
}

// Get returns the current license information.
func (h *LicenseInfoHandler) Get(c *gin.Context) {
	lic := middleware.GetLicense(c)
	if lic == nil {
		lic = license.FreeLicense()
	}

	features := license.FeaturesForTier(lic.Tier)
	featureStrings := make([]string, len(features))
	for i, f := range features {
		featureStrings[i] = string(f)
	}

	c.JSON(http.StatusOK, LicenseInfoResponse{
		Tier:       string(lic.Tier),
		CustomerID: lic.CustomerID,
		ExpiresAt:  lic.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		IssuedAt:   lic.IssuedAt.Format("2006-01-02T15:04:05Z"),
		Features:   featureStrings,
		Limits:     lic.Limits,
	})
}
