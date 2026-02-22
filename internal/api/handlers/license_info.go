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
	Tier             string             `json:"tier"`
	CustomerID       string             `json:"customer_id"`
	CustomerName     string             `json:"customer_name,omitempty"`
	ExpiresAt        string             `json:"expires_at"`
	IssuedAt         string             `json:"issued_at"`
	Features         []string           `json:"features"`
	Limits           license.TierLimits `json:"limits"`
	LicenseKeySource string             `json:"license_key_source"`
	IsTrial          bool               `json:"is_trial"`
	TrialDaysLeft    int                `json:"trial_days_left,omitempty"`
}

// LicenseInfoHandler handles license information endpoints.
type LicenseInfoHandler struct {
	validator *license.Validator
	logger    zerolog.Logger
}

// NewLicenseInfoHandler creates a new LicenseInfoHandler.
func NewLicenseInfoHandler(validator *license.Validator, logger zerolog.Logger) *LicenseInfoHandler {
	return &LicenseInfoHandler{
		validator: validator,
		logger:    logger.With().Str("component", "license_info_handler").Logger(),
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

	// Use entitlement features if available, otherwise fall back to tier features
	var featureStrings []string
	if ent := middleware.GetEntitlement(c); ent != nil && !ent.IsExpired() {
		featureStrings = make([]string, len(ent.Features))
		for i, f := range ent.Features {
			featureStrings[i] = string(f)
		}
	} else {
		features := license.FeaturesForTier(lic.Tier)
		featureStrings = make([]string, len(features))
		for i, f := range features {
			featureStrings[i] = string(f)
		}
	}

	keySource := "none"
	if h.validator != nil {
		keySource = h.validator.LicenseKeySource()
	}

	c.JSON(http.StatusOK, LicenseInfoResponse{
		Tier:             string(lic.Tier),
		CustomerID:       lic.CustomerID,
		CustomerName:     lic.CustomerName,
		ExpiresAt:        lic.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		IssuedAt:         lic.IssuedAt.Format("2006-01-02T15:04:05Z"),
		Features:         featureStrings,
		Limits:           lic.Limits,
		LicenseKeySource: keySource,
		IsTrial:          lic.IsTrial,
		TrialDaysLeft:    lic.TrialDaysLeft(),
	})
}
