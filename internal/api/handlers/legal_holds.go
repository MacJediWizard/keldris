package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LegalHoldStore defines the interface for legal hold persistence operations.
type LegalHoldStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateLegalHold(ctx context.Context, hold *models.LegalHold) error
	GetLegalHoldByID(ctx context.Context, id uuid.UUID) (*models.LegalHold, error)
	GetLegalHoldBySnapshotID(ctx context.Context, snapshotID string, orgID uuid.UUID) (*models.LegalHold, error)
	GetLegalHoldsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LegalHold, error)
	DeleteLegalHold(ctx context.Context, id uuid.UUID) error
	IsSnapshotOnHold(ctx context.Context, snapshotID string, orgID uuid.UUID) (bool, error)
	GetSnapshotHoldStatus(ctx context.Context, snapshotIDs []string, orgID uuid.UUID) (map[string]bool, error)
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	// Audit logging
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// LegalHoldsHandler handles legal hold HTTP endpoints.
type LegalHoldsHandler struct {
	store  LegalHoldStore
	logger zerolog.Logger
}

// NewLegalHoldsHandler creates a new LegalHoldsHandler.
func NewLegalHoldsHandler(store LegalHoldStore, logger zerolog.Logger) *LegalHoldsHandler {
	return &LegalHoldsHandler{
		store:  store,
		logger: logger.With().Str("component", "legal_holds_handler").Logger(),
	}
}

// RegisterRoutes registers legal hold routes on the given router group.
func (h *LegalHoldsHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Legal holds list endpoint
	legalHolds := r.Group("/legal-holds")
	{
		legalHolds.GET("", h.ListLegalHolds)
	}

	// Snapshot hold endpoints (nested under snapshots)
	snapshots := r.Group("/snapshots")
	{
		snapshots.POST("/:id/hold", h.CreateLegalHold)
		snapshots.DELETE("/:id/hold", h.DeleteLegalHold)
		snapshots.GET("/:id/hold", h.GetLegalHold)
	}
}

// LegalHoldResponse represents a legal hold in API responses.
type LegalHoldResponse struct {
	ID           string `json:"id"`
	SnapshotID   string `json:"snapshot_id"`
	Reason       string `json:"reason"`
	PlacedBy     string `json:"placed_by"`
	PlacedByName string `json:"placed_by_name"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (h *LegalHoldsHandler) toLegalHoldResponse(hold *models.LegalHold, placedByUser *models.User) LegalHoldResponse {
	placedByName := ""
	if placedByUser != nil {
		placedByName = placedByUser.Name
	}
	return LegalHoldResponse{
		ID:           hold.ID.String(),
		SnapshotID:   hold.SnapshotID,
		Reason:       hold.Reason,
		PlacedBy:     hold.PlacedBy.String(),
		PlacedByName: placedByName,
		CreatedAt:    hold.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    hold.UpdatedAt.Format(time.RFC3339),
	}
}

// ListLegalHolds returns all legal holds for the authenticated user's organization.
//
//	@Summary		List legal holds
//	@Description	Returns all legal holds for the current organization (admin only)
//	@Tags			Legal Holds
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]LegalHoldResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/legal-holds [get]
func (h *LegalHoldsHandler) ListLegalHolds(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	holds, err := h.store.GetLegalHoldsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list legal holds")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list legal holds"})
		return
	}

	// Build user cache for response enrichment
	userCache := make(map[uuid.UUID]*models.User)
	var responses []LegalHoldResponse
	for _, hold := range holds {
		var placedByUser *models.User
		if cached, ok := userCache[hold.PlacedBy]; ok {
			placedByUser = cached
		} else {
			placedByUser, _ = h.store.GetUserByID(c.Request.Context(), hold.PlacedBy)
			userCache[hold.PlacedBy] = placedByUser
		}
		responses = append(responses, h.toLegalHoldResponse(hold, placedByUser))
	}

	c.JSON(http.StatusOK, gin.H{"legal_holds": responses})
}

// CreateLegalHold places a legal hold on a snapshot.
//
//	@Summary		Create legal hold
//	@Description	Places a legal hold on a snapshot to prevent deletion (admin only)
//	@Tags			Legal Holds
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Snapshot ID"
//	@Param			request	body		models.CreateLegalHoldRequest	true	"Hold details"
//	@Success		201		{object}	LegalHoldResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/hold [post]
func (h *LegalHoldsHandler) CreateLegalHold(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	var req models.CreateLegalHoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Verify the snapshot exists and user has access
	backup, err := h.store.GetBackupBySnapshotID(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}

	// Check if hold already exists
	existing, _ := h.store.GetLegalHoldBySnapshotID(c.Request.Context(), snapshotID, dbUser.OrgID)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "legal hold already exists for this snapshot"})
		return
	}

	hold := models.NewLegalHold(dbUser.OrgID, snapshotID, req.Reason, dbUser.ID)

	if err := h.store.CreateLegalHold(c.Request.Context(), hold); err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to create legal hold")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create legal hold"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(dbUser.OrgID, models.AuditActionCreate, "legal_hold", models.AuditResultSuccess).
		WithUser(dbUser.ID).
		WithResource(hold.ID).
		WithDetails("Legal hold placed on snapshot " + snapshotID + ": " + req.Reason)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("hold_id", hold.ID.String()).
		Str("snapshot_id", snapshotID).
		Str("placed_by", dbUser.ID.String()).
		Msg("legal hold created")

	c.JSON(http.StatusCreated, h.toLegalHoldResponse(hold, dbUser))
}

// GetLegalHold returns the legal hold for a specific snapshot.
//
//	@Summary		Get legal hold
//	@Description	Returns the legal hold for a specific snapshot
//	@Tags			Legal Holds
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Snapshot ID"
//	@Success		200	{object}	LegalHoldResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/hold [get]
func (h *LegalHoldsHandler) GetLegalHold(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	hold, err := h.store.GetLegalHoldBySnapshotID(c.Request.Context(), snapshotID, dbUser.OrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no legal hold on this snapshot"})
		return
	}

	placedByUser, _ := h.store.GetUserByID(c.Request.Context(), hold.PlacedBy)
	c.JSON(http.StatusOK, h.toLegalHoldResponse(hold, placedByUser))
}

// DeleteLegalHold removes a legal hold from a snapshot.
//
//	@Summary		Delete legal hold
//	@Description	Removes the legal hold from a snapshot (admin only)
//	@Tags			Legal Holds
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Snapshot ID"
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/hold [delete]
func (h *LegalHoldsHandler) DeleteLegalHold(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	hold, err := h.store.GetLegalHoldBySnapshotID(c.Request.Context(), snapshotID, dbUser.OrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no legal hold on this snapshot"})
		return
	}

	if err := h.store.DeleteLegalHold(c.Request.Context(), hold.ID); err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to delete legal hold")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete legal hold"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(dbUser.OrgID, models.AuditActionDelete, "legal_hold", models.AuditResultSuccess).
		WithUser(dbUser.ID).
		WithResource(hold.ID).
		WithDetails("Legal hold removed from snapshot " + snapshotID)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("hold_id", hold.ID.String()).
		Str("snapshot_id", snapshotID).
		Str("removed_by", dbUser.ID.String()).
		Msg("legal hold removed")

	c.JSON(http.StatusOK, gin.H{"message": "legal hold removed"})
}
