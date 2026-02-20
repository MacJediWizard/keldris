package handlers

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LicenseManageHandler handles license activation/deactivation from the GUI.
type LicenseManageHandler struct {
	validator *license.Validator
	publicKey ed25519.PublicKey
	logger    zerolog.Logger
}

// NewLicenseManageHandler creates a new LicenseManageHandler.
func NewLicenseManageHandler(validator *license.Validator, publicKey ed25519.PublicKey, logger zerolog.Logger) *LicenseManageHandler {
	return &LicenseManageHandler{
		validator: validator,
		publicKey: publicKey,
		logger:    logger.With().Str("component", "license_manage_handler").Logger(),
	}
}

// RegisterRoutes registers license management routes.
func (h *LicenseManageHandler) RegisterRoutes(r *gin.RouterGroup) {
	system := r.Group("/system/license")
	system.POST("/activate", h.Activate)
	system.POST("/deactivate", h.Deactivate)
	system.GET("/plans", h.GetPlans)
}

type activateLicenseRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
}

// Activate validates and activates a license key from the GUI.
func (h *LicenseManageHandler) Activate(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "license management not available in air-gap mode"})
		return
	}

	var req activateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the key format (Ed25519 signature check)
	if len(h.publicKey) == ed25519.PublicKeySize {
		_, err := license.ParseLicenseKeyEd25519(req.LicenseKey, h.publicKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license key format"})
			return
		}
	}

	// Store key and trigger activation
	if err := h.validator.SetLicenseKey(c.Request.Context(), req.LicenseKey); err != nil {
		h.logger.Error().Err(err).Msg("failed to activate license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate license"})
		return
	}

	// Return current license info
	lic := h.validator.GetLicense()
	features := license.FeaturesForTier(lic.Tier)
	featureStrings := make([]string, len(features))
	for i, f := range features {
		featureStrings[i] = string(f)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "activated",
		"tier":       string(lic.Tier),
		"expires_at": lic.ExpiresAt.Format(time.RFC3339),
		"features":   featureStrings,
		"limits":     lic.Limits,
	})
}

// Deactivate removes the license key and reverts to free tier.
func (h *LicenseManageHandler) Deactivate(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "license management not available in air-gap mode"})
		return
	}

	// Deactivate with the license server first
	ctx := c.Request.Context()
	key := h.validator.GetLicenseKey()
	if key != "" {
		// Stop background loops and deactivate with license server
		h.validator.Stop(ctx)
		// Reinitialize so the validator can accept a new license later
		h.validator.Restart()
	}

	// Clear the key from DB
	if err := h.validator.ClearLicenseKey(ctx); err != nil {
		h.logger.Error().Err(err).Msg("failed to deactivate license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate license"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "deactivated",
		"tier":   "free",
	})
}

// GetPlans proxies pricing plans from the license server.
func (h *LicenseManageHandler) GetPlans(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not available in air-gap mode"})
		return
	}

	// Proxy request to license server
	serverURL := fmt.Sprintf("%s/api/v1/products/keldris/pricing", h.validator.GetServerURL())
	resp, err := http.Get(serverURL) //nolint:gosec // URL is from trusted server config
	if err != nil {
		// Return empty plans on error
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var plans []json.RawMessage
	if err := json.Unmarshal(body, &plans); err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	c.JSON(http.StatusOK, plans)
}
