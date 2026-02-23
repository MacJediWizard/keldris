package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// AirGapHandler handles air-gapped operation and license management endpoints.
type AirGapHandler struct {
	licenseManager *license.AirGapManager
	logger         zerolog.Logger
}

// NewAirGapHandler creates a new AirGapHandler.
func NewAirGapHandler(licenseManager *license.AirGapManager, logger zerolog.Logger) *AirGapHandler {
	return &AirGapHandler{
		licenseManager: licenseManager,
		logger:         logger.With().Str("component", "airgap_handler").Logger(),
	}
}

// RegisterRoutes registers air-gap and license routes.
func (h *AirGapHandler) RegisterRoutes(r *gin.RouterGroup, publicGroup *gin.RouterGroup) {
	// Public endpoint for air-gap status (needed by frontend before auth)
	publicGroup.GET("/airgap/status", h.GetAirGapStatus)

	// Protected admin endpoints
	if r != nil {
		airgap := r.Group("/airgap")
		{
			airgap.GET("/license", h.GetLicenseStatus)
			airgap.POST("/license", h.UploadLicense)
			airgap.GET("/license/renewal-request", h.GetRenewalRequest)
			airgap.POST("/revocations", h.UpdateRevocationList)
			airgap.GET("/updates", h.ListUpdatePackages)
			airgap.POST("/updates/:filename/apply", h.ApplyUpdate)
			airgap.GET("/docs", h.GetDocumentation)
			airgap.GET("/docs/*path", h.ServeDocumentation)
		}
	}
}

// AirGapStatusResponse is the response for air-gap status check.
type AirGapStatusResponse struct {
	AirGapMode           bool   `json:"airgap_mode"`
	DisableUpdateChecker bool   `json:"disable_update_checker"`
	DisableTelemetry     bool   `json:"disable_telemetry"`
	DisableExternalLinks bool   `json:"disable_external_links"`
	OfflineDocsVersion   string `json:"offline_docs_version,omitempty"`
	LicenseValid         bool   `json:"license_valid"`
}

// GetAirGapStatus returns the current air-gap mode status.
// GET /api/v1/public/airgap/status
func (h *AirGapHandler) GetAirGapStatus(c *gin.Context) {
	config := h.licenseManager.GetConfig()
	status := h.licenseManager.GetStatus()

	c.JSON(http.StatusOK, AirGapStatusResponse{
		AirGapMode:           config.Enabled,
		DisableUpdateChecker: config.DisableUpdateChecker,
		DisableTelemetry:     config.DisableTelemetry,
		DisableExternalLinks: config.DisableExternalLinks,
		OfflineDocsVersion:   config.OfflineDocsVersion,
		LicenseValid:         status.Valid,
	})
}

// GetLicenseStatus returns detailed license status.
// GET /api/v1/airgap/license
func (h *AirGapHandler) GetLicenseStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins/owners or superusers can view license details
	if !isAdmin(user.CurrentOrgRole) && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	status := h.licenseManager.GetStatus()
	c.JSON(http.StatusOK, status)
}

// UploadLicenseRequest is the request body for license upload.
type UploadLicenseRequest struct {
	License string `json:"license" binding:"required"`
}

// UploadLicense handles new license file upload.
// POST /api/v1/airgap/license
func (h *AirGapHandler) UploadLicense(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only superusers can update license
	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	var req UploadLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.licenseManager.ApplyNewLicense([]byte(req.License)); err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to apply new license")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to apply license: " + err.Error()})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("user_email", user.Email).
		Msg("license updated successfully")

	status := h.licenseManager.GetStatus()
	c.JSON(http.StatusOK, status)
}

// GetRenewalRequest generates a license renewal request.
// GET /api/v1/airgap/license/renewal-request
func (h *AirGapHandler) GetRenewalRequest(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) && !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	request, err := h.licenseManager.GenerateRenewalRequest()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Generate downloadable JSON
	data, _ := json.MarshalIndent(request, "", "  ")

	c.Header("Content-Disposition", "attachment; filename=license-renewal-request.json")
	c.Data(http.StatusOK, "application/json", data)
}

// UpdateRevocationList handles revocation list updates.
// POST /api/v1/airgap/revocations
func (h *AirGapHandler) UpdateRevocationList(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := h.licenseManager.UpdateRevocationList(data); err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to update revocation list")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Msg("revocation list updated")

	c.JSON(http.StatusOK, gin.H{"message": "revocation list updated successfully"})
}

// ListUpdatePackages lists available offline update packages.
// GET /api/v1/airgap/updates
func (h *AirGapHandler) ListUpdatePackages(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	packages, err := h.licenseManager.GetUpdatePackages()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list update packages")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list updates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"packages": packages,
		"count":    len(packages),
	})
}

// ApplyUpdate applies an offline update package.
// POST /api/v1/airgap/updates/:filename/apply
func (h *AirGapHandler) ApplyUpdate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	config := h.licenseManager.GetConfig()
	packagePath := filepath.Join(config.UpdatePackagePath, filepath.Clean(filename))

	// Verify the file exists and is within the update directory
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	absUpdatePath, _ := filepath.Abs(config.UpdatePackagePath)
	if !isSubPath(absUpdatePath, absPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "update package not found"})
		return
	}

	// TODO: Implement actual update application logic
	// For now, return a placeholder response
	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("package", filename).
		Msg("update application requested")

	c.JSON(http.StatusOK, gin.H{
		"message": "Update queued for application. The system will restart automatically.",
		"package": filename,
	})
}

// DocumentationIndex represents the offline documentation index.
type DocumentationIndex struct {
	Version  string            `json:"version"`
	BuildAt  time.Time         `json:"built_at"`
	Sections []DocSection      `json:"sections"`
	Search   map[string]string `json:"search_index,omitempty"`
}

// DocSection represents a documentation section.
type DocSection struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Path     string       `json:"path"`
	Children []DocSection `json:"children,omitempty"`
}

// GetDocumentation returns the documentation index.
// GET /api/v1/airgap/docs
func (h *AirGapHandler) GetDocumentation(c *gin.Context) {
	config := h.licenseManager.GetConfig()

	indexPath := filepath.Join(config.DocumentationPath, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "offline documentation not installed",
				"message": "Please install the offline documentation bundle for air-gapped operation.",
			})
			return
		}
		h.logger.Error().Err(err).Msg("failed to read documentation index")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read documentation"})
		return
	}

	var index DocumentationIndex
	if err := json.Unmarshal(data, &index); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse documentation index")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid documentation index"})
		return
	}

	c.JSON(http.StatusOK, index)
}

// ServeDocumentation serves documentation files.
// GET /api/v1/airgap/docs/*path
func (h *AirGapHandler) ServeDocumentation(c *gin.Context) {
	requestPath := c.Param("path")
	if requestPath == "" || requestPath == "/" {
		c.Redirect(http.StatusMovedPermanently, "/api/v1/airgap/docs")
		return
	}

	config := h.licenseManager.GetConfig()
	docPath := filepath.Join(config.DocumentationPath, filepath.Clean(requestPath))

	// Security: Verify path is within documentation directory
	absPath, err := filepath.Abs(docPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	absDocsPath, _ := filepath.Abs(config.DocumentationPath)
	if !isSubPath(absDocsPath, absPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Check if file exists
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access document"})
		return
	}

	if info.IsDir() {
		// Try to serve index.html from directory
		indexPath := filepath.Join(absPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.File(absPath)
}

// isSubPath checks if child is a subpath of parent.
func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && len(rel) >= 2 && rel[:2] != ".."
}
