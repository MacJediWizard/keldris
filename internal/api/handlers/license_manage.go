package handlers

import (
	"bytes"
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
	system.POST("/trial/start", h.StartTrial)
	system.GET("/trial/check", h.CheckTrial)
}

type activateLicenseRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
}

// Activate validates and activates a license key from the GUI.
func (h *LicenseManageHandler) Activate(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "license management not available without a license server connection"})
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
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "license management not available without a license server connection"})
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
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not available without a license server connection"})
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

type startTrialRequest struct {
	Email string `json:"email" binding:"required"`
	Tier  string `json:"tier" binding:"required"`
}

// StartTrial proxies a trial start request to the license server.
func (h *LicenseManageHandler) StartTrial(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not available without a license server connection"})
		return
	}

	var req startTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Proxy to license server
	serverURL := fmt.Sprintf("%s/api/v1/trials/start", h.validator.GetServerURL())
	payload := map[string]string{
		"product": "keldris",
		"email":   req.Email,
		"tier":    req.Tier,
	}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(serverURL, "application/json", io.NopCloser(bytes.NewReader(payloadBytes))) //nolint:gosec // URL is from trusted server config
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to start trial via license server")
		c.JSON(http.StatusBadGateway, gin.H{"error": "license server unreachable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid response from license server"})
		return
	}

	if resp.StatusCode != http.StatusCreated {
		errMsg, _ := result["error"].(string)
		if errMsg == "" {
			errMsg = "failed to start trial"
		}
		c.JSON(resp.StatusCode, gin.H{"error": errMsg})
		return
	}

	// Auto-activate the trial license key
	if licenseKey, ok := result["license_key"].(string); ok && licenseKey != "" {
		if err := h.validator.SetLicenseKey(c.Request.Context(), licenseKey); err != nil {
			h.logger.Error().Err(err).Msg("failed to activate trial license key")
		}
	}

	lic := h.validator.GetLicense()
	features := license.FeaturesForTier(lic.Tier)
	featureStrings := make([]string, len(features))
	for i, f := range features {
		featureStrings[i] = string(f)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              "trial_started",
		"tier":                result["tier"],
		"expires_at":          result["expires_at"],
		"trial_duration_days": result["trial_duration_days"],
		"features":            featureStrings,
		"limits":              lic.Limits,
	})
}

// CheckTrial checks trial availability for an email via the license server.
func (h *LicenseManageHandler) CheckTrial(c *gin.Context) {
	if h.validator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not available without a license server connection"})
		return
	}

	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query param required"})
		return
	}

	serverURL := fmt.Sprintf("%s/api/v1/trials/check?email=%s&product=keldris", h.validator.GetServerURL(), email)
	resp, err := http.Get(serverURL) //nolint:gosec // URL is from trusted server config
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"has_trial": false})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"has_trial": false})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"has_trial": false})
		return
	}

	c.JSON(http.StatusOK, result)
}
