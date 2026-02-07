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

// SLAStore defines the interface for SLA persistence operations.
type SLAStore interface {
	GetSLADefinitionByID(ctx context.Context, id uuid.UUID) (*models.SLADefinition, error)
	ListSLADefinitionsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLADefinition, error)
	ListSLADefinitionsWithAssignments(ctx context.Context, orgID uuid.UUID) ([]*models.SLAWithAssignments, error)
	CreateSLADefinition(ctx context.Context, s *models.SLADefinition) error
	UpdateSLADefinition(ctx context.Context, s *models.SLADefinition) error
	DeleteSLADefinition(ctx context.Context, id uuid.UUID) error

	GetSLAAssignmentByID(ctx context.Context, id uuid.UUID) (*models.SLAAssignment, error)
	ListSLAAssignmentsBySLA(ctx context.Context, slaID uuid.UUID) ([]*models.SLAAssignment, error)
	CreateSLAAssignment(ctx context.Context, a *models.SLAAssignment) error
	DeleteSLAAssignment(ctx context.Context, id uuid.UUID) error

	ListSLAComplianceBySLA(ctx context.Context, slaID uuid.UUID, limit int) ([]*models.SLACompliance, error)
	ListSLAComplianceByOrg(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.SLACompliance, error)

	GetSLABreachByID(ctx context.Context, id uuid.UUID) (*models.SLABreach, error)
	ListSLABreachesByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.SLABreach, error)
	ListActiveSLABreachesByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLABreach, error)
	ListSLABreachesBySLA(ctx context.Context, slaID uuid.UUID, limit int) ([]*models.SLABreach, error)
	UpdateSLABreach(ctx context.Context, b *models.SLABreach) error

	GetSLADashboardStats(ctx context.Context, orgID uuid.UUID) (*models.SLADashboardStats, error)
}

// SLAHandler handles SLA HTTP endpoints.
type SLAHandler struct {
	store  SLAStore
	logger zerolog.Logger
}

// NewSLAHandler creates a new SLAHandler.
func NewSLAHandler(store SLAStore, logger zerolog.Logger) *SLAHandler {
	return &SLAHandler{
		store:  store,
		logger: logger.With().Str("component", "sla_handler").Logger(),
	}
}

// RegisterRoutes registers SLA routes on the given router group.
func (h *SLAHandler) RegisterRoutes(r *gin.RouterGroup) {
	slas := r.Group("/slas")
	{
		// SLA Definitions
		slas.GET("", h.ListSLAs)
		slas.POST("", h.CreateSLA)
		slas.GET("/:id", h.GetSLA)
		slas.PUT("/:id", h.UpdateSLA)
		slas.DELETE("/:id", h.DeleteSLA)

		// SLA Assignments
		slas.GET("/:id/assignments", h.ListAssignments)
		slas.POST("/:id/assignments", h.CreateAssignment)
		slas.DELETE("/:id/assignments/:assignmentId", h.DeleteAssignment)

		// SLA Compliance
		slas.GET("/:id/compliance", h.GetCompliance)

		// SLA Breaches
		slas.GET("/:id/breaches", h.ListBreachesBySLA)
	}

	// Organization-wide endpoints
	r.GET("/sla-dashboard", h.GetDashboard)
	r.GET("/sla-breaches", h.ListBreaches)
	r.GET("/sla-breaches/active", h.ListActiveBreaches)
	r.GET("/sla-breaches/:id", h.GetBreach)
	r.POST("/sla-breaches/:id/acknowledge", h.AcknowledgeBreach)
	r.POST("/sla-breaches/:id/resolve", h.ResolveBreach)
	r.GET("/sla-compliance", h.ListOrgCompliance)
	r.GET("/sla-report", h.GetReport)
}

// ListSLAs returns all SLA definitions for the organization.
// GET /api/v1/slas
func (h *SLAHandler) ListSLAs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	slas, err := h.store.ListSLADefinitionsWithAssignments(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list SLAs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list SLAs"})
		return
	}

	c.JSON(http.StatusOK, models.SLADefinitionsResponse{SLAs: toSLAWithAssignmentsSlice(slas)})
}

// GetSLA returns a specific SLA by ID.
// GET /api/v1/slas/:id
func (h *SLAHandler) GetSLA(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	c.JSON(http.StatusOK, sla)
}

// CreateSLA creates a new SLA definition.
// POST /api/v1/slas
func (h *SLAHandler) CreateSLA(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate at least one target is defined
	if req.RPOMinutes == nil && req.RTOMinutes == nil && req.UptimePercentage == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one target (RPO, RTO, or Uptime) must be defined"})
		return
	}

	// Validate uptime percentage range
	if req.UptimePercentage != nil && (*req.UptimePercentage < 0 || *req.UptimePercentage > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uptime percentage must be between 0 and 100"})
		return
	}

	sla := models.NewSLADefinition(user.CurrentOrgID, req.Name, req.Scope)
	sla.Description = req.Description
	sla.RPOMinutes = req.RPOMinutes
	sla.RTOMinutes = req.RTOMinutes
	sla.UptimePercentage = req.UptimePercentage
	sla.CreatedBy = &user.ID

	if req.Active != nil {
		sla.Active = *req.Active
	}

	if err := h.store.CreateSLADefinition(c.Request.Context(), sla); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to create SLA")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create SLA"})
		return
	}

	h.logger.Info().
		Str("sla_id", sla.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("name", sla.Name).
		Msg("SLA created")

	c.JSON(http.StatusCreated, sla)
}

// UpdateSLA updates an existing SLA definition.
// PUT /api/v1/slas/:id
func (h *SLAHandler) UpdateSLA(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	var req models.UpdateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if req.Name != nil {
		sla.Name = *req.Name
	}
	if req.Description != nil {
		sla.Description = *req.Description
	}
	if req.RPOMinutes != nil {
		sla.RPOMinutes = req.RPOMinutes
	}
	if req.RTOMinutes != nil {
		sla.RTOMinutes = req.RTOMinutes
	}
	if req.UptimePercentage != nil {
		if *req.UptimePercentage < 0 || *req.UptimePercentage > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "uptime percentage must be between 0 and 100"})
			return
		}
		sla.UptimePercentage = req.UptimePercentage
	}
	if req.Scope != nil {
		sla.Scope = *req.Scope
	}
	if req.Active != nil {
		sla.Active = *req.Active
	}

	if err := h.store.UpdateSLADefinition(c.Request.Context(), sla); err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to update SLA")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update SLA"})
		return
	}

	h.logger.Info().
		Str("sla_id", sla.ID.String()).
		Str("name", sla.Name).
		Msg("SLA updated")

	c.JSON(http.StatusOK, sla)
}

// DeleteSLA deletes an SLA definition.
// DELETE /api/v1/slas/:id
func (h *SLAHandler) DeleteSLA(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if err := h.store.DeleteSLADefinition(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to delete SLA")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete SLA"})
		return
	}

	h.logger.Info().
		Str("sla_id", id.String()).
		Str("name", sla.Name).
		Msg("SLA deleted")

	c.JSON(http.StatusOK, gin.H{"message": "SLA deleted"})
}

// ListAssignments returns all assignments for an SLA.
// GET /api/v1/slas/:id/assignments
func (h *SLAHandler) ListAssignments(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	assignments, err := h.store.ListSLAAssignmentsBySLA(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to list assignments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assignments"})
		return
	}

	c.JSON(http.StatusOK, models.SLAAssignmentsResponse{Assignments: toSLAAssignmentSlice(assignments)})
}

// CreateAssignment creates a new SLA assignment.
// POST /api/v1/slas/:id/assignments
func (h *SLAHandler) CreateAssignment(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	var req models.AssignSLARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate exactly one target
	if (req.AgentID == nil && req.RepositoryID == nil) || (req.AgentID != nil && req.RepositoryID != nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exactly one of agent_id or repository_id must be provided"})
		return
	}

	assignment := models.NewSLAAssignment(user.CurrentOrgID, id)
	assignment.AgentID = req.AgentID
	assignment.RepositoryID = req.RepositoryID
	assignment.AssignedBy = &user.ID

	if err := h.store.CreateSLAAssignment(c.Request.Context(), assignment); err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to create assignment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create assignment"})
		return
	}

	h.logger.Info().
		Str("sla_id", id.String()).
		Str("assignment_id", assignment.ID.String()).
		Msg("SLA assignment created")

	c.JSON(http.StatusCreated, assignment)
}

// DeleteAssignment deletes an SLA assignment.
// DELETE /api/v1/slas/:id/assignments/:assignmentId
func (h *SLAHandler) DeleteAssignment(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	slaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	assignmentID, err := uuid.Parse(c.Param("assignmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), slaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	assignment, err := h.store.GetSLAAssignmentByID(c.Request.Context(), assignmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	if assignment.SLAID != slaID {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	if err := h.store.DeleteSLAAssignment(c.Request.Context(), assignmentID); err != nil {
		h.logger.Error().Err(err).Str("assignment_id", assignmentID.String()).Msg("failed to delete assignment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete assignment"})
		return
	}

	h.logger.Info().
		Str("assignment_id", assignmentID.String()).
		Msg("SLA assignment deleted")

	c.JSON(http.StatusOK, gin.H{"message": "assignment deleted"})
}

// GetCompliance returns compliance records for an SLA.
// GET /api/v1/slas/:id/compliance
func (h *SLAHandler) GetCompliance(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	compliance, err := h.store.ListSLAComplianceBySLA(c.Request.Context(), id, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to list compliance")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list compliance"})
		return
	}

	c.JSON(http.StatusOK, models.SLAComplianceResponse{Compliance: toSLAComplianceSlice(compliance)})
}

// GetDashboard returns SLA dashboard statistics.
// GET /api/v1/sla-dashboard
func (h *SLAHandler) GetDashboard(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	stats, err := h.store.GetSLADashboardStats(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get dashboard stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, models.SLADashboardResponse{Stats: *stats})
}

// ListBreaches returns all breaches for the organization.
// GET /api/v1/sla-breaches
func (h *SLAHandler) ListBreaches(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	breaches, err := h.store.ListSLABreachesByOrg(c.Request.Context(), user.CurrentOrgID, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list breaches")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list breaches"})
		return
	}

	c.JSON(http.StatusOK, models.SLABreachesResponse{Breaches: toSLABreachSlice(breaches)})
}

// ListActiveBreaches returns active (unresolved) breaches for the organization.
// GET /api/v1/sla-breaches/active
func (h *SLAHandler) ListActiveBreaches(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	breaches, err := h.store.ListActiveSLABreachesByOrg(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list active breaches")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list active breaches"})
		return
	}

	c.JSON(http.StatusOK, models.SLABreachesResponse{Breaches: toSLABreachSlice(breaches)})
}

// ListBreachesBySLA returns breaches for a specific SLA.
// GET /api/v1/slas/:id/breaches
func (h *SLAHandler) ListBreachesBySLA(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SLA ID"})
		return
	}

	sla, err := h.store.GetSLADefinitionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	if sla.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA not found"})
		return
	}

	breaches, err := h.store.ListSLABreachesBySLA(c.Request.Context(), id, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("sla_id", id.String()).Msg("failed to list breaches")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list breaches"})
		return
	}

	c.JSON(http.StatusOK, models.SLABreachesResponse{Breaches: toSLABreachSlice(breaches)})
}

// GetBreach returns a specific breach by ID.
// GET /api/v1/sla-breaches/:id
func (h *SLAHandler) GetBreach(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid breach ID"})
		return
	}

	breach, err := h.store.GetSLABreachByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	if breach.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	c.JSON(http.StatusOK, breach)
}

// AcknowledgeBreach marks a breach as acknowledged.
// POST /api/v1/sla-breaches/:id/acknowledge
func (h *SLAHandler) AcknowledgeBreach(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid breach ID"})
		return
	}

	breach, err := h.store.GetSLABreachByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	if breach.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	if breach.Acknowledged {
		c.JSON(http.StatusBadRequest, gin.H{"error": "breach already acknowledged"})
		return
	}

	var req models.AcknowledgeBreachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Notes are optional, so continue even if binding fails
		req.Notes = ""
	}

	breach.Acknowledge(user.ID)
	if req.Notes != "" {
		breach.Description = req.Notes
	}

	if err := h.store.UpdateSLABreach(c.Request.Context(), breach); err != nil {
		h.logger.Error().Err(err).Str("breach_id", id.String()).Msg("failed to acknowledge breach")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to acknowledge breach"})
		return
	}

	h.logger.Info().
		Str("breach_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("breach acknowledged")

	c.JSON(http.StatusOK, breach)
}

// ResolveBreach marks a breach as resolved.
// POST /api/v1/sla-breaches/:id/resolve
func (h *SLAHandler) ResolveBreach(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid breach ID"})
		return
	}

	breach, err := h.store.GetSLABreachByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	if breach.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "breach not found"})
		return
	}

	if breach.Resolved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "breach already resolved"})
		return
	}

	breach.Resolve()

	if err := h.store.UpdateSLABreach(c.Request.Context(), breach); err != nil {
		h.logger.Error().Err(err).Str("breach_id", id.String()).Msg("failed to resolve breach")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve breach"})
		return
	}

	h.logger.Info().
		Str("breach_id", id.String()).
		Msg("breach resolved")

	c.JSON(http.StatusOK, breach)
}

// ListOrgCompliance returns compliance records for the organization.
// GET /api/v1/sla-compliance
func (h *SLAHandler) ListOrgCompliance(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Default to last 30 days
	now := time.Now()
	periodStart := now.AddDate(0, 0, -30)
	periodEnd := now

	compliance, err := h.store.ListSLAComplianceByOrg(c.Request.Context(), user.CurrentOrgID, periodStart, periodEnd)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list compliance")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list compliance"})
		return
	}

	c.JSON(http.StatusOK, models.SLAComplianceResponse{Compliance: toSLAComplianceSlice(compliance)})
}

// GetReport returns a monthly SLA report.
// GET /api/v1/sla-report?month=2024-01
func (h *SLAHandler) GetReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Parse month parameter (default to current month)
	monthStr := c.Query("month")
	var reportMonth time.Time
	if monthStr != "" {
		var err error
		reportMonth, err = time.Parse("2006-01", monthStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month format, use YYYY-MM"})
			return
		}
	} else {
		now := time.Now()
		reportMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	// Generate a simple report from current data
	stats, err := h.store.GetSLADashboardStats(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get report data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	report := models.SLAReport{
		OrgID:         user.CurrentOrgID,
		ReportMonth:   reportMonth,
		GeneratedAt:   time.Now(),
		TotalBreaches: stats.ActiveBreaches,
		SLASummaries:  []models.SLAComplianceSummary{},
	}

	c.JSON(http.StatusOK, models.SLAReportResponse{Report: report})
}

// Helper functions to convert slices
func toSLAWithAssignmentsSlice(slas []*models.SLAWithAssignments) []models.SLAWithAssignments {
	if slas == nil {
		return []models.SLAWithAssignments{}
	}
	result := make([]models.SLAWithAssignments, len(slas))
	for i, s := range slas {
		result[i] = *s
	}
	return result
}

func toSLAAssignmentSlice(assignments []*models.SLAAssignment) []models.SLAAssignment {
	if assignments == nil {
		return []models.SLAAssignment{}
	}
	result := make([]models.SLAAssignment, len(assignments))
	for i, a := range assignments {
		result[i] = *a
	}
	return result
}

func toSLAComplianceSlice(compliance []*models.SLACompliance) []models.SLACompliance {
	if compliance == nil {
		return []models.SLACompliance{}
	}
	result := make([]models.SLACompliance, len(compliance))
	for i, c := range compliance {
		result[i] = *c
	}
	return result
}

func toSLABreachSlice(breaches []*models.SLABreach) []models.SLABreach {
	if breaches == nil {
		return []models.SLABreach{}
	}
	result := make([]models.SLABreach, len(breaches))
	for i, b := range breaches {
		result[i] = *b
	}
	return result
}
