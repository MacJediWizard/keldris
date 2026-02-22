package handlers

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/airgap"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// AirGapHandler handles air-gap mode status endpoints.
type AirGapHandler struct {
	logger zerolog.Logger
}

// NewAirGapHandler creates a new AirGapHandler.
func NewAirGapHandler(logger zerolog.Logger) *AirGapHandler {
	return &AirGapHandler{
		logger: logger.With().Str("component", "airgap_handler").Logger(),
	}
}

// RegisterRoutes registers air-gap routes on the given router group.
func (h *AirGapHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/system/airgap", h.GetStatus)
}

// AirGapStatusResponse is the response for the air-gap status endpoint.
type AirGapStatusResponse struct {
	Enabled          bool                     `json:"enabled"`
	DisabledFeatures []airgap.DisabledFeature `json:"disabled_features"`
}

// GetStatus returns the current air-gap mode status.
func (h *AirGapHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	enabled := airgap.IsAirGapMode()

	resp := AirGapStatusResponse{
		Enabled: enabled,
	}

	if enabled {
		resp.DisabledFeatures = airgap.DisabledFeatures()
	}

	c.JSON(http.StatusOK, resp)
}
