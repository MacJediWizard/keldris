package handlers

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/portal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LicensesHandler handles license-related endpoints for the portal.
type LicensesHandler struct {
	store  portal.Store
	logger zerolog.Logger
}

// NewLicensesHandler creates a new LicensesHandler.
func NewLicensesHandler(store portal.Store, logger zerolog.Logger) *LicensesHandler {
	return &LicensesHandler{
		store:  store,
		logger: logger.With().Str("component", "portal_licenses_handler").Logger(),
	}
}

// RegisterRoutes registers license routes on the given router group.
func (h *LicensesHandler) RegisterRoutes(r *gin.RouterGroup) {
	licenses := r.Group("/licenses")
	{
		licenses.GET("", h.List)
		licenses.GET("/:id", h.Get)
		licenses.GET("/:id/download", h.Download)
	}
}

// List returns all licenses for the authenticated customer.
//
//	@Summary		List licenses
//	@Description	Returns all licenses owned by the customer
//	@Tags			Portal Licenses
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.License
//	@Failure		401	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/licenses [get]
func (h *LicensesHandler) List(c *gin.Context) {
	customer := portal.RequireCustomer(c)
	if customer == nil {
		return
	}

	licenses, err := h.store.GetLicensesByCustomerID(c.Request.Context(), customer.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to list licenses")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list licenses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"licenses": licenses})
}

// Get returns a specific license by ID.
//
//	@Summary		Get license
//	@Description	Returns a specific license by ID
//	@Tags			Portal Licenses
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"License ID"
//	@Success		200	{object}	models.License
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/licenses/{id} [get]
func (h *LicensesHandler) Get(c *gin.Context) {
	customer := portal.RequireCustomer(c)
	if customer == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license ID"})
		return
	}

	license, err := h.store.GetLicenseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Verify license belongs to customer
	if license.CustomerID != customer.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	c.JSON(http.StatusOK, license)
}

// Download returns the license key for download.
//
//	@Summary		Download license
//	@Description	Returns the license key details for download
//	@Tags			Portal Licenses
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"License ID"
//	@Success		200	{object}	models.LicenseDownloadResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/licenses/{id}/download [get]
func (h *LicensesHandler) Download(c *gin.Context) {
	customer := portal.RequireCustomer(c)
	if customer == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license ID"})
		return
	}

	license, err := h.store.GetLicenseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Verify license belongs to customer
	if license.CustomerID != customer.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Check if license is valid
	if !license.IsValid() {
		c.JSON(http.StatusForbidden, gin.H{"error": "license is not active"})
		return
	}

	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Str("license_id", license.ID.String()).
		Msg("license downloaded")

	c.JSON(http.StatusOK, license.ToDownloadResponse())
}

// AdminLicensesHandler handles admin license operations.
type AdminLicensesHandler struct {
	store  portal.Store
	logger zerolog.Logger
}

// NewAdminLicensesHandler creates a new AdminLicensesHandler.
func NewAdminLicensesHandler(store portal.Store, logger zerolog.Logger) *AdminLicensesHandler {
	return &AdminLicensesHandler{
		store:  store,
		logger: logger.With().Str("component", "admin_licenses_handler").Logger(),
	}
}

// RegisterRoutes registers admin license routes on the given router group.
func (h *AdminLicensesHandler) RegisterRoutes(r *gin.RouterGroup) {
	licenses := r.Group("/licenses")
	{
		licenses.GET("", h.List)
		licenses.POST("", h.Create)
		licenses.GET("/:id", h.Get)
		licenses.PUT("/:id", h.Update)
		licenses.POST("/:id/revoke", h.Revoke)
	}
}

// List returns all licenses (admin).
//
//	@Summary		List all licenses (admin)
//	@Description	Returns all licenses with customer details
//	@Tags			Admin Licenses
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"
//	@Param			offset	query		int	false	"Offset"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/api/v1/admin/licenses [get]
func (h *AdminLicensesHandler) List(c *gin.Context) {
	limit := 50
	offset := 0
	// Parse query params (simplified)

	licenses, total, err := h.store.ListLicenses(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list licenses")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list licenses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"licenses": licenses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// Create creates a new license (admin).
//
//	@Summary		Create license (admin)
//	@Description	Creates a new license for a customer
//	@Tags			Admin Licenses
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CreateLicenseRequest	true	"License details"
//	@Success		201		{object}	models.License
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/api/v1/admin/licenses [post]
func (h *AdminLicensesHandler) Create(c *gin.Context) {
	var req models.CreateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify customer exists
	customer, err := h.store.GetCustomerByID(c.Request.Context(), req.CustomerID)
	if err != nil || customer == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer not found"})
		return
	}

	// Create license
	license := models.NewLicense(req.CustomerID, req.LicenseType, req.ProductName)
	license.MaxAgents = req.MaxAgents
	license.MaxRepos = req.MaxRepos
	license.MaxStorage = req.MaxStorage
	license.Features = req.Features
	license.ExpiresAt = req.ExpiresAt
	license.Notes = req.Notes

	if err := h.store.CreateLicense(c.Request.Context(), license); err != nil {
		h.logger.Error().Err(err).Str("customer_id", req.CustomerID.String()).Msg("failed to create license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create license"})
		return
	}

	h.logger.Info().
		Str("license_id", license.ID.String()).
		Str("customer_id", req.CustomerID.String()).
		Str("license_type", string(req.LicenseType)).
		Msg("license created")

	c.JSON(http.StatusCreated, license)
}

// Get returns a specific license (admin).
//
//	@Summary		Get license (admin)
//	@Description	Returns a specific license by ID
//	@Tags			Admin Licenses
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"License ID"
//	@Success		200	{object}	models.License
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/api/v1/admin/licenses/{id} [get]
func (h *AdminLicensesHandler) Get(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license ID"})
		return
	}

	license, err := h.store.GetLicenseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	c.JSON(http.StatusOK, license)
}

// Update updates a license (admin).
//
//	@Summary		Update license (admin)
//	@Description	Updates a license
//	@Tags			Admin Licenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"License ID"
//	@Param			request	body		models.UpdateLicenseRequest	true	"License updates"
//	@Success		200		{object}	models.License
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/api/v1/admin/licenses/{id} [put]
func (h *AdminLicensesHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license ID"})
		return
	}

	var req models.UpdateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	license, err := h.store.GetLicenseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Apply updates
	if req.Status != nil {
		license.Status = *req.Status
	}
	if req.MaxAgents != nil {
		license.MaxAgents = req.MaxAgents
	}
	if req.MaxRepos != nil {
		license.MaxRepos = req.MaxRepos
	}
	if req.MaxStorage != nil {
		license.MaxStorage = req.MaxStorage
	}
	if req.Features != nil {
		license.Features = req.Features
	}
	if req.ExpiresAt != nil {
		license.ExpiresAt = req.ExpiresAt
	}
	if req.Notes != nil {
		license.Notes = *req.Notes
	}

	if err := h.store.UpdateLicense(c.Request.Context(), license); err != nil {
		h.logger.Error().Err(err).Str("license_id", id.String()).Msg("failed to update license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update license"})
		return
	}

	h.logger.Info().
		Str("license_id", id.String()).
		Msg("license updated")

	c.JSON(http.StatusOK, license)
}

// Revoke revokes a license (admin).
//
//	@Summary		Revoke license (admin)
//	@Description	Revokes a license, making it invalid
//	@Tags			Admin Licenses
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"License ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/api/v1/admin/licenses/{id}/revoke [post]
func (h *AdminLicensesHandler) Revoke(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid license ID"})
		return
	}

	license, err := h.store.GetLicenseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	license.Status = models.LicenseStatusRevoked
	if err := h.store.UpdateLicense(c.Request.Context(), license); err != nil {
		h.logger.Error().Err(err).Str("license_id", id.String()).Msg("failed to revoke license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke license"})
		return
	}

	h.logger.Info().
		Str("license_id", id.String()).
		Msg("license revoked")

	c.JSON(http.StatusOK, gin.H{"message": "license revoked"})
}
