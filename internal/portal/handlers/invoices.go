package handlers

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/portal/portalctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// InvoicesHandler handles invoice-related endpoints for the portal.
type InvoicesHandler struct {
	store  portalctx.Store
	logger zerolog.Logger
}

// NewInvoicesHandler creates a new InvoicesHandler.
func NewInvoicesHandler(store portalctx.Store, logger zerolog.Logger) *InvoicesHandler {
	return &InvoicesHandler{
		store:  store,
		logger: logger.With().Str("component", "portal_invoices_handler").Logger(),
	}
}

// RegisterRoutes registers invoice routes on the given router group.
func (h *InvoicesHandler) RegisterRoutes(r *gin.RouterGroup) {
	invoices := r.Group("/invoices")
	{
		invoices.GET("", h.List)
		invoices.GET("/:id", h.Get)
		invoices.GET("/:id/download", h.Download)
	}
}

// List returns all invoices for the authenticated customer.
//
//	@Summary		List invoices
//	@Description	Returns all invoices for the customer
//	@Tags			Portal Invoices
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]models.Invoice
//	@Failure		401	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/invoices [get]
func (h *InvoicesHandler) List(c *gin.Context) {
	customer := portalctx.RequireCustomer(c)
	if customer == nil {
		return
	}

	invoices, err := h.store.GetInvoicesByCustomerID(c.Request.Context(), customer.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to list invoices")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

// Get returns a specific invoice by ID.
//
//	@Summary		Get invoice
//	@Description	Returns a specific invoice by ID with line items
//	@Tags			Portal Invoices
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Invoice ID"
//	@Success		200	{object}	models.InvoiceWithItems
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/invoices/{id} [get]
func (h *InvoicesHandler) Get(c *gin.Context) {
	customer := portalctx.RequireCustomer(c)
	if customer == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	invoice, err := h.store.GetInvoiceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// Verify invoice belongs to customer
	if invoice.CustomerID != customer.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// Get invoice items
	items, err := h.store.GetInvoiceItems(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("invoice_id", id.String()).Msg("failed to get invoice items")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoice": invoice,
		"items":   items,
	})
}

// Download returns the invoice for PDF download.
//
//	@Summary		Download invoice
//	@Description	Returns the invoice details for PDF generation
//	@Tags			Portal Invoices
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Invoice ID"
//	@Success		200	{object}	models.InvoiceDownloadResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/invoices/{id}/download [get]
func (h *InvoicesHandler) Download(c *gin.Context) {
	customer := portalctx.RequireCustomer(c)
	if customer == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice ID"})
		return
	}

	invoice, err := h.store.GetInvoiceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// Verify invoice belongs to customer
	if invoice.CustomerID != customer.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// Get invoice items
	items, err := h.store.GetInvoiceItems(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("invoice_id", id.String()).Msg("failed to get invoice items")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
		return
	}

	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Str("invoice_id", invoice.ID.String()).
		Msg("invoice downloaded")

	// Convert items
	var invoiceItems []interface{}
	for _, item := range items {
		invoiceItems = append(invoiceItems, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"invoice":          invoice,
		"items":            items,
		"customer_name":    customer.Name,
		"customer_email":   customer.Email,
		"customer_company": customer.Company,
	})
}
