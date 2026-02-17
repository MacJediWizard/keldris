package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/airgap"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AirGapStore defines the interface for air-gap license persistence operations.
type AirGapStore interface {
	CreateOfflineLicense(ctx context.Context, license *models.OfflineLicense) error
	GetLatestOfflineLicense(ctx context.Context, orgID uuid.UUID) (*models.OfflineLicense, error)
}

// AirGapHandler handles air-gap mode HTTP endpoints.
type AirGapHandler struct {
	store     AirGapStore
	publicKey []byte
	logger    zerolog.Logger
}

// NewAirGapHandler creates a new AirGapHandler.
func NewAirGapHandler(store AirGapStore, publicKey []byte, logger zerolog.Logger) *AirGapHandler {
	return &AirGapHandler{
		store:     store,
		publicKey: publicKey,
		logger:    logger.With().Str("component", "airgap_handler").Logger(),
	}
}

// RegisterRoutes registers air-gap routes on the given router group.
func (h *AirGapHandler) RegisterRoutes(r *gin.RouterGroup) {
	system := r.Group("/system")
	{
		system.GET("/airgap", h.GetStatus)
		system.POST("/license", h.UploadLicense)
	}
}

// AirGapStatusResponse is the response for the air-gap status endpoint.
type AirGapStatusResponse struct {
	Enabled          bool                     `json:"enabled"`
	DisabledFeatures []airgap.DisabledFeature `json:"disabled_features"`
	License          *AirGapLicenseInfo       `json:"license,omitempty"`
}

// AirGapLicenseInfo contains information about the current offline license.
type AirGapLicenseInfo struct {
	CustomerID string    `json:"customer_id"`
	Tier       string    `json:"tier"`
	ExpiresAt  time.Time `json:"expires_at"`
	IssuedAt   time.Time `json:"issued_at"`
	Valid      bool      `json:"valid"`
}

// GetStatus returns the current air-gap mode status.
// @Summary Get air-gap status
// @Description Returns whether air-gap mode is enabled and which features are disabled
// @Tags system
// @Produce json
// @Success 200 {object} AirGapStatusResponse
// @Router /api/v1/system/airgap [get]
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

	// Check for existing offline license
	lic, err := h.store.GetLatestOfflineLicense(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get offline license")
	}
	if lic != nil {
		resp.License = &AirGapLicenseInfo{
			CustomerID: lic.CustomerID,
			Tier:       lic.Tier,
			ExpiresAt:  lic.ExpiresAt,
			IssuedAt:   lic.IssuedAt,
			Valid:      time.Now().Before(lic.ExpiresAt),
		}
	}

	c.JSON(http.StatusOK, resp)
}

// UploadLicense handles offline license file upload.
// @Summary Upload offline license
// @Description Validates and stores an offline license file for air-gap deployments
// @Tags system
// @Accept octet-stream
// @Produce json
// @Success 200 {object} AirGapLicenseInfo
// @Failure 400 {object} map[string]string
// @Router /api/v1/system/license [post]
func (h *AirGapHandler) UploadLicense(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 1MB limit
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read license data"})
		return
	}
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty license data"})
		return
	}

	// Validate the license
	lic, err := license.ValidateOfflineLicense(body, h.publicKey)
	if err != nil {
		h.logger.Warn().Err(err).Msg("invalid offline license uploaded")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid license: " + err.Error()})
		return
	}

	// Store in database
	offlineLicense := &models.OfflineLicense{
		ID:          uuid.New(),
		OrgID:       user.CurrentOrgID,
		CustomerID:  lic.CustomerID,
		Tier:        string(lic.Tier),
		LicenseData: body,
		ExpiresAt:   lic.ExpiresAt,
		IssuedAt:    lic.IssuedAt,
		UploadedBy:  &user.ID,
		CreatedAt:   time.Now(),
	}

	if err := h.store.CreateOfflineLicense(c.Request.Context(), offlineLicense); err != nil {
		h.logger.Error().Err(err).Msg("failed to store offline license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store license"})
		return
	}

	h.logger.Info().
		Str("customer_id", lic.CustomerID).
		Str("tier", string(lic.Tier)).
		Time("expires_at", lic.ExpiresAt).
		Msg("offline license uploaded successfully")

	c.JSON(http.StatusOK, AirGapLicenseInfo{
		CustomerID: lic.CustomerID,
		Tier:       string(lic.Tier),
		ExpiresAt:  lic.ExpiresAt,
		IssuedAt:   lic.IssuedAt,
		Valid:      true,
	})
}
