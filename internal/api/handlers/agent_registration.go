package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RegistrationCodeStore defines the interface for registration code persistence operations.
type RegistrationCodeStore interface {
	CreateRegistrationCode(ctx context.Context, code *models.RegistrationCode) error
	GetRegistrationCodeByCode(ctx context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error)
	GetPendingRegistrationCodes(ctx context.Context, orgID uuid.UUID) ([]*models.RegistrationCode, error)
	GetPendingRegistrationsWithCreator(ctx context.Context, orgID uuid.UUID) ([]*models.PendingRegistration, error)
	MarkRegistrationCodeUsed(ctx context.Context, codeID, agentID uuid.UUID) error
	DeleteExpiredRegistrationCodes(ctx context.Context) error
	DeleteRegistrationCode(ctx context.Context, id uuid.UUID) error
	CreateAgent(ctx context.Context, agent *models.Agent) error
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// AgentRegistrationHandler handles agent registration code endpoints.
type AgentRegistrationHandler struct {
	store    RegistrationCodeStore
	agentMFA *auth.AgentMFA
	logger   zerolog.Logger
}

// NewAgentRegistrationHandler creates a new AgentRegistrationHandler.
func NewAgentRegistrationHandler(store RegistrationCodeStore, logger zerolog.Logger) *AgentRegistrationHandler {
	agentMFA := auth.NewAgentMFA(store, logger)
	return &AgentRegistrationHandler{
		store:    store,
		agentMFA: agentMFA,
		logger:   logger.With().Str("component", "agent_registration_handler").Logger(),
	}
}

// RegisterRoutes registers agent registration routes on the given router group.
func (h *AgentRegistrationHandler) RegisterRoutes(r *gin.RouterGroup) {
	codes := r.Group("/agent-registration-codes")
	{
		codes.POST("", h.CreateCode)
		codes.GET("", h.ListPendingCodes)
		codes.DELETE("/:id", h.DeleteCode)
	}

	// Public endpoint for agents to register with a code (requires no auth - agent uses code)
	// This will be called by the agent binary
}

// RegisterPublicRoutes registers public agent registration routes that don't require session auth.
func (h *AgentRegistrationHandler) RegisterPublicRoutes(r *gin.Engine) {
	// Agent registration with code - no session auth required, code is the auth
	r.POST("/api/v1/agents/register", h.RegisterWithCode)
}

// CreateCode creates a new registration code for an organization.
// POST /api/v1/agent-registration-codes
func (h *AgentRegistrationHandler) CreateCode(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.CreateRegistrationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Optional hostname - if provided, agent must match
	var hostname *string
	if req.Hostname != "" {
		hostname = &req.Hostname
	}

	regCode, err := h.agentMFA.GenerateCode(c.Request.Context(), user.CurrentOrgID, user.ID, hostname)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create registration code")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create registration code"})
		return
	}

	// Log the audit event
	h.logAuditEvent(c, user.CurrentOrgID, user.ID, models.AuditActionCreate, "agent_registration_code",
		&regCode.ID, models.AuditResultSuccess, "registration code created")

	h.logger.Info().
		Str("code_id", regCode.ID.String()).
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("registration code created")

	c.JSON(http.StatusCreated, models.CreateRegistrationCodeResponse{
		ID:        regCode.ID,
		Code:      regCode.Code,
		Hostname:  regCode.Hostname,
		ExpiresAt: regCode.ExpiresAt,
	})
}

// ListPendingCodes returns all pending registration codes for the organization.
// GET /api/v1/agent-registration-codes
func (h *AgentRegistrationHandler) ListPendingCodes(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	registrations, err := h.store.GetPendingRegistrationsWithCreator(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list pending registrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pending registrations"})
		return
	}

	// Ensure we return an empty array instead of null
	if registrations == nil {
		registrations = []*models.PendingRegistration{}
	}

	c.JSON(http.StatusOK, gin.H{"registrations": registrations})
}

// DeleteCode deletes a registration code by ID.
// DELETE /api/v1/agent-registration-codes/:id
func (h *AgentRegistrationHandler) DeleteCode(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registration code ID"})
		return
	}

	if err := h.store.DeleteRegistrationCode(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("code_id", id.String()).Msg("failed to delete registration code")
		c.JSON(http.StatusNotFound, gin.H{"error": "registration code not found or already used"})
		return
	}

	// Log the audit event
	h.logAuditEvent(c, user.CurrentOrgID, user.ID, models.AuditActionDelete, "agent_registration_code",
		&id, models.AuditResultSuccess, "registration code deleted")

	h.logger.Info().Str("code_id", id.String()).Msg("registration code deleted")
	c.JSON(http.StatusOK, gin.H{"message": "registration code deleted"})
}

// RegisterWithCode registers a new agent using a registration code.
// POST /api/v1/agents/register
// This endpoint doesn't require session auth - the code serves as authentication.
func (h *AgentRegistrationHandler) RegisterWithCode(c *gin.Context) {
	var req models.RegisterWithCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// We need to find the organization by trying each org's codes
	// For now, we'll require the org_id to be passed in the request
	// This is a simplification - in production you might want a different approach
	orgIDStr := c.GetHeader("X-Org-ID")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Org-ID header required"})
		return
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Verify the registration code
	regCode, err := h.agentMFA.VerifyCode(c.Request.Context(), orgID, req.Code)
	if err != nil {
		h.logger.Warn().
			Str("org_id", orgID.String()).
			Str("code", req.Code).
			Str("hostname", req.Hostname).
			Err(err).
			Msg("agent registration failed - invalid code")

		// Log failed registration attempt
		h.logAuditEvent(c, orgID, uuid.Nil, models.AuditActionCreate, "agent",
			nil, models.AuditResultFailure, "agent registration failed: "+err.Error())

		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// If code has a hostname, verify it matches
	if regCode.Hostname != nil && *regCode.Hostname != "" && *regCode.Hostname != req.Hostname {
		h.logger.Warn().
			Str("code_id", regCode.ID.String()).
			Str("expected_hostname", *regCode.Hostname).
			Str("actual_hostname", req.Hostname).
			Msg("agent registration failed - hostname mismatch")

		// Log failed registration attempt
		h.logAuditEvent(c, orgID, uuid.Nil, models.AuditActionCreate, "agent",
			nil, models.AuditResultFailure, "agent registration failed: hostname mismatch")

		c.JSON(http.StatusBadRequest, gin.H{"error": "hostname does not match registration code"})
		return
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}

	// Hash the API key for storage
	apiKeyHash := hashAPIKey(apiKey)

	// Create the agent
	agent := models.NewAgent(orgID, req.Hostname, apiKeyHash)

	if err := h.store.CreateAgent(c.Request.Context(), agent); err != nil {
		h.logger.Error().Err(err).Str("hostname", req.Hostname).Msg("failed to create agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}

	// Mark the registration code as used
	if err := h.agentMFA.MarkCodeUsed(c.Request.Context(), regCode.ID, agent.ID); err != nil {
		h.logger.Error().Err(err).
			Str("code_id", regCode.ID.String()).
			Str("agent_id", agent.ID.String()).
			Msg("failed to mark registration code as used")
		// Don't fail the request - agent was created successfully
	}

	// Log successful registration
	h.logAuditEvent(c, orgID, regCode.CreatedBy, models.AuditActionCreate, "agent",
		&agent.ID, models.AuditResultSuccess, "agent registered via registration code")

	h.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("hostname", req.Hostname).
		Str("org_id", orgID.String()).
		Str("code_id", regCode.ID.String()).
		Msg("agent registered with code")

	c.JSON(http.StatusCreated, models.RegisterWithCodeResponse{
		ID:       agent.ID,
		Hostname: agent.Hostname,
		APIKey:   apiKey,
	})
}

// logAuditEvent logs an audit event for agent registration actions.
func (h *AgentRegistrationHandler) logAuditEvent(c *gin.Context, orgID, userID uuid.UUID, action models.AuditAction, resourceType string, resourceID *uuid.UUID, result models.AuditResult, details string) {
	auditLog := models.NewAuditLog(orgID, action, resourceType, result).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent()).
		WithDetails(details)

	if userID != uuid.Nil {
		auditLog.WithUser(userID)
	}

	if resourceID != nil {
		auditLog.WithResource(*resourceID)
	}

	go func() {
		if err := h.store.CreateAuditLog(context.Background(), auditLog); err != nil {
			h.logger.Error().Err(err).
				Str("action", string(action)).
				Str("resource_type", resourceType).
				Msg("failed to create audit log")
		}
	}()
}
