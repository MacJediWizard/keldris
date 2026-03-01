package handlers

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// OrganizationStore defines the interface for organization persistence operations.
type OrganizationStore interface {
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error)
	CreateOrganization(ctx context.Context, org *models.Organization) error
	UpdateOrganization(ctx context.Context, org *models.Organization) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
	GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error)
	GetMembershipsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMembershipWithUser, error)
	CreateMembership(ctx context.Context, m *models.OrgMembership) error
	UpdateMembership(ctx context.Context, m *models.OrgMembership) error
	DeleteMembership(ctx context.Context, userID, orgID uuid.UUID) error
	CreateInvitation(ctx context.Context, inv *models.OrgInvitation) error
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*models.OrgInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*models.OrgInvitation, error)
	GetPendingInvitationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.OrgInvitationWithDetails, error)
	GetPendingInvitationsByEmail(ctx context.Context, email string) ([]*models.OrgInvitationWithDetails, error)
	AcceptInvitation(ctx context.Context, id uuid.UUID) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	UpdateInvitationResent(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

// OrganizationsHandler handles organization-related HTTP endpoints.
type OrganizationsHandler struct {
	store    OrganizationStore
	sessions *auth.SessionStore
	rbac     *auth.RBAC
	checker  *license.FeatureChecker
	logger   zerolog.Logger
}

// NewOrganizationsHandler creates a new OrganizationsHandler.
func NewOrganizationsHandler(store OrganizationStore, sessions *auth.SessionStore, rbac *auth.RBAC, checker *license.FeatureChecker, logger zerolog.Logger) *OrganizationsHandler {
	return &OrganizationsHandler{
		store:    store,
		sessions: sessions,
		rbac:     rbac,
		checker:  checker,
		logger:   logger.With().Str("component", "organizations_handler").Logger(),
	}
}

// RegisterRoutes registers organization routes available on all tiers.
// These allow free-tier users to access and manage their default organization.
func (h *OrganizationsHandler) RegisterRoutes(r *gin.RouterGroup) {
	orgs := r.Group("/organizations")
	{
		orgs.GET("", h.List)
		orgs.GET("/current", h.GetCurrent)
		orgs.POST("/switch", h.Switch)
		orgs.GET("/:id", h.Get)
		orgs.PUT("/:id", h.Update)
		orgs.GET("/:id/members", h.ListMembers)
	}
}

// RegisterMultiOrgRoutes registers organization management routes that require the multi_org feature.
// These include creating, deleting orgs, managing members, and invitations.
func (h *OrganizationsHandler) RegisterMultiOrgRoutes(r *gin.RouterGroup, createMiddleware ...gin.HandlerFunc) {
	orgs := r.Group("/organizations")
	{
		createChain := append(createMiddleware, h.Create)
		orgs.POST("", createChain...)
		orgs.DELETE("/:id", h.Delete)

		// Member management
		orgs.PUT("/:id/members/:user_id", h.UpdateMember)
		orgs.DELETE("/:id/members/:user_id", h.RemoveMember)

		// Invitations
		orgs.GET("/:id/invitations", h.ListInvitations)
		orgs.POST("/:id/invitations", h.CreateInvitation)
		orgs.POST("/:id/invitations/bulk", h.BulkInvite)
		orgs.POST("/:id/invitations/:invitation_id/resend", h.ResendInvitation)
		orgs.DELETE("/:id/invitations/:invitation_id", h.DeleteInvitation)
	}

	// Invitation acceptance
	r.POST("/invitations/accept", h.AcceptInvitation)
	// Public endpoint to get invitation details by token
	r.GET("/invitations/:token", h.GetInvitationByToken)
}

// CreateOrgRequest is the request body for creating an organization.
type CreateOrgRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
	Slug string `json:"slug" binding:"required,min=1,max=255,alphanum"`
}

// UpdateOrgRequest is the request body for updating an organization.
type UpdateOrgRequest struct {
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// SwitchOrgRequest is the request body for switching organizations.
type SwitchOrgRequest struct {
	OrgID string `json:"org_id" binding:"required,uuid"`
}

// InviteMemberRequest is the request body for inviting a member.
type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin member readonly"`
}

// UpdateMemberRequest is the request body for updating a member.
type UpdateMemberRequest struct {
	Role string `json:"role" binding:"required,oneof=owner admin member readonly"`
}

// AcceptInvitationRequest is the request body for accepting an invitation.
type AcceptInvitationRequest struct {
	Token string `json:"token" binding:"required"`
}

// OrgResponse is the response for organization endpoints.
type OrgResponse struct {
	Organization *models.Organization `json:"organization"`
	Role         string               `json:"role"`
}

// List returns all organizations the user belongs to.
//
//	@Summary		List organizations
//	@Description	Returns all organizations the current user is a member of
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/organizations [get]
// GET /api/v1/organizations
func (h *OrganizationsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgs, err := h.store.GetUserOrganizations(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to list organizations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list organizations"})
		return
	}

	// Include role for each org
	type orgWithRole struct {
		*models.Organization
		Role string `json:"role"`
	}

	result := make([]orgWithRole, 0, len(orgs))
	for _, org := range orgs {
		membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, org.ID)
		if err != nil {
			continue
		}
		result = append(result, orgWithRole{
			Organization: org,
			Role:         string(membership.Role),
		})
	}

	c.JSON(http.StatusOK, gin.H{"organizations": result})
}

// Create creates a new organization.
//
//	@Summary		Create organization
//	@Description	Creates a new organization with the current user as owner
//	@Tags			Organizations
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateOrgRequest	true	"Organization details"
//	@Success		201		{object}	OrgResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/organizations [post]
// POST /api/v1/organizations
func (h *OrganizationsHandler) Create(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureMultiOrg) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Check if slug is already taken
	_, err := h.store.GetOrganizationBySlug(c.Request.Context(), strings.ToLower(req.Slug))
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "organization slug already exists"})
		return
	}

	org := models.NewOrganization(req.Name, strings.ToLower(req.Slug))
	if err := h.store.CreateOrganization(c.Request.Context(), org); err != nil {
		h.logger.Error().Err(err).Str("slug", req.Slug).Msg("failed to create organization")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	// Add creator as owner
	membership := models.NewOrgMembership(user.ID, org.ID, models.OrgRoleOwner)
	if err := h.store.CreateMembership(c.Request.Context(), membership); err != nil {
		h.logger.Error().Err(err).Str("org_id", org.ID.String()).Msg("failed to create owner membership")
		// Rollback org creation
		_ = h.store.DeleteOrganization(c.Request.Context(), org.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	h.logger.Info().
		Str("org_id", org.ID.String()).
		Str("slug", org.Slug).
		Str("user_id", user.ID.String()).
		Msg("organization created")

	c.JSON(http.StatusCreated, OrgResponse{
		Organization: org,
		Role:         string(models.OrgRoleOwner),
	})
}

// GetCurrent returns the current organization.
// GET /api/v1/organizations/current
func (h *OrganizationsHandler) GetCurrent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no organization selected"})
		return
	}

	org, err := h.store.GetOrganizationByID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, OrgResponse{
		Organization: org,
		Role:         user.CurrentOrgRole,
	})
}

// Switch switches to a different organization.
// POST /api/v1/organizations/switch
func (h *OrganizationsHandler) Switch(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req SwitchOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Verify user is a member
	membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, orgID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this organization"})
		return
	}

	org, err := h.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	// Update session
	if err := h.sessions.SetCurrentOrg(c.Request, c.Writer, orgID, string(membership.Role)); err != nil {
		h.logger.Error().Err(err).Msg("failed to update session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to switch organization"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("org_id", orgID.String()).
		Msg("switched organization")

	c.JSON(http.StatusOK, OrgResponse{
		Organization: org,
		Role:         string(membership.Role),
	})
}

// Get returns a specific organization.
// GET /api/v1/organizations/:id
func (h *OrganizationsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Verify user is a member
	membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, orgID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this organization"})
		return
	}

	org, err := h.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, OrgResponse{
		Organization: org,
		Role:         string(membership.Role),
	})
}

// Update updates an organization.
// PUT /api/v1/organizations/:id
func (h *OrganizationsHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	org, err := h.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Slug != "" {
		newSlug := strings.ToLower(req.Slug)
		if newSlug != org.Slug {
			// Check if new slug is taken
			existing, err := h.store.GetOrganizationBySlug(c.Request.Context(), newSlug)
			if err == nil && existing.ID != org.ID {
				c.JSON(http.StatusConflict, gin.H{"error": "organization slug already exists"})
				return
			}
			org.Slug = newSlug
		}
	}

	if err := h.store.UpdateOrganization(c.Request.Context(), org); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to update organization")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update organization"})
		return
	}

	membership, _ := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, orgID)
	role := ""
	if membership != nil {
		role = string(membership.Role)
	}

	c.JSON(http.StatusOK, OrgResponse{
		Organization: org,
		Role:         role,
	})
}

// Delete deletes an organization.
// DELETE /api/v1/organizations/:id
func (h *OrganizationsHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Only owner can delete
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermOrgDelete); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the owner can delete an organization"})
		return
	}

	if err := h.store.DeleteOrganization(c.Request.Context(), orgID); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to delete organization")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete organization"})
		return
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("user_id", user.ID.String()).
		Msg("organization deleted")

	c.JSON(http.StatusOK, gin.H{"message": "organization deleted"})
}

// ListMembers returns all members of an organization.
// GET /api/v1/organizations/:id/members
func (h *OrganizationsHandler) ListMembers(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	members, err := h.store.GetMembershipsByOrgID(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to list members")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// UpdateMember updates a member's role.
// PUT /api/v1/organizations/:id/members/:user_id
func (h *OrganizationsHandler) UpdateMember(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Check permission to manage this member
	canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, orgID)
	if err != nil || !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Check if user can assign this role
	newRole := models.OrgRole(req.Role)
	canAssign, err := h.rbac.CanAssignRole(c.Request.Context(), user.ID, orgID, newRole)
	if err != nil || !canAssign {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot assign this role"})
		return
	}

	membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), targetUserID, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}

	membership.Role = newRole
	if err := h.store.UpdateMembership(c.Request.Context(), membership); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Str("user_id", targetUserID.String()).Msg("failed to update member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update member"})
		return
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("target_user_id", targetUserID.String()).
		Str("new_role", req.Role).
		Str("actor_id", user.ID.String()).
		Msg("member role updated")

	c.JSON(http.StatusOK, gin.H{"message": "member updated"})
}

// RemoveMember removes a member from an organization.
// DELETE /api/v1/organizations/:id/members/:user_id
func (h *OrganizationsHandler) RemoveMember(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// Users can remove themselves (leave org)
	if user.ID != targetUserID {
		// Check permission to manage this member
		canManage, err := h.rbac.CanManageMember(c.Request.Context(), user.ID, targetUserID, orgID)
		if err != nil || !canManage {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
	}

	// Don't allow removing the last owner
	targetMembership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), targetUserID, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}

	if targetMembership.Role == models.OrgRoleOwner {
		// Check if there are other owners
		members, err := h.store.GetMembershipsByOrgID(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check membership"})
			return
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == models.OrgRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the last owner"})
			return
		}
	}

	if err := h.store.DeleteMembership(c.Request.Context(), targetUserID, orgID); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Str("user_id", targetUserID.String()).Msg("failed to remove member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("target_user_id", targetUserID.String()).
		Str("actor_id", user.ID.String()).
		Msg("member removed")

	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// ListInvitations returns pending invitations for an organization.
// GET /api/v1/organizations/:id/invitations
func (h *OrganizationsHandler) ListInvitations(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	invitations, err := h.store.GetPendingInvitationsByOrgID(c.Request.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to list invitations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invitations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitations": invitations})
}

// CreateInvitation creates a new invitation.
// POST /api/v1/organizations/:id/invitations
func (h *OrganizationsHandler) CreateInvitation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberInvite); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Check if user can assign this role
	newRole := models.OrgRole(req.Role)
	canAssign, err := h.rbac.CanAssignRole(c.Request.Context(), user.ID, orgID, newRole)
	if err != nil || !canAssign {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot invite with this role"})
		return
	}

	// Generate secure token
	token, err := generateInviteToken()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate invite token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}

	// Create invitation (expires in 7 days)
	inv := models.NewOrgInvitation(
		orgID,
		strings.ToLower(req.Email),
		newRole,
		token,
		user.ID,
		time.Now().Add(7*24*time.Hour),
	)

	if err := h.store.CreateInvitation(c.Request.Context(), inv); err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Str("email", req.Email).Msg("failed to create invitation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Str("email", req.Email).
		Str("role", req.Role).
		Str("inviter_id", user.ID.String()).
		Msg("invitation created")

	c.JSON(http.StatusCreated, gin.H{
		"message": "invitation created",
		"token":   token, // Return token so it can be shared
	})
}

// DeleteInvitation deletes a pending invitation.
// DELETE /api/v1/organizations/:id/invitations/:invitation_id
func (h *OrganizationsHandler) DeleteInvitation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberInvite); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	if err := h.store.DeleteInvitation(c.Request.Context(), invitationID); err != nil {
		h.logger.Error().Err(err).Str("invitation_id", invitationID.String()).Msg("failed to delete invitation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete invitation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invitation deleted"})
}

// AcceptInvitation accepts an invitation.
// POST /api/v1/invitations/accept
func (h *OrganizationsHandler) AcceptInvitation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	inv, err := h.store.GetInvitationByToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	if inv.IsExpired() {
		c.JSON(http.StatusGone, gin.H{"error": "invitation has expired"})
		return
	}

	if inv.IsAccepted() {
		c.JSON(http.StatusConflict, gin.H{"error": "invitation has already been accepted"})
		return
	}

	// Verify email matches
	if !strings.EqualFold(user.Email, inv.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "invitation is for a different email address"})
		return
	}

	// Create membership
	membership := models.NewOrgMembership(user.ID, inv.OrgID, inv.Role)
	if err := h.store.CreateMembership(c.Request.Context(), membership); err != nil {
		// Check if already a member
		if strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "already a member of this organization"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Str("org_id", inv.OrgID.String()).Msg("failed to create membership")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join organization"})
		return
	}

	// Mark invitation as accepted
	if err := h.store.AcceptInvitation(c.Request.Context(), inv.ID); err != nil {
		h.logger.Error().Err(err).Str("invitation_id", inv.ID.String()).Msg("failed to mark invitation as accepted")
	}

	// Switch to new org
	if err := h.sessions.SetCurrentOrg(c.Request, c.Writer, inv.OrgID, string(inv.Role)); err != nil {
		h.logger.Error().Err(err).Msg("failed to update session after accepting invitation")
	}

	org, _ := h.store.GetOrganizationByID(c.Request.Context(), inv.OrgID)

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("org_id", inv.OrgID.String()).
		Str("role", string(inv.Role)).
		Msg("invitation accepted")

	c.JSON(http.StatusOK, gin.H{
		"message":      "joined organization",
		"organization": org,
		"role":         string(inv.Role),
	})
}

// generateInviteToken generates a secure random token for invitations.
func generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BulkInviteRequest is the structure for each invitation in a bulk request.
type BulkInviteEntry struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin member readonly"`
}

// BulkInviteResponse is the response for a bulk invitation request.
type BulkInviteResponse struct {
	Successful []BulkInviteResult `json:"successful"`
	Failed     []BulkInviteError  `json:"failed"`
	Total      int                `json:"total"`
}

// BulkInviteResult represents a successful invitation.
type BulkInviteResult struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	Token string `json:"token,omitempty"`
}

// BulkInviteError represents a failed invitation.
type BulkInviteError struct {
	Email string `json:"email"`
	Error string `json:"error"`
}

// BulkInvite creates multiple invitations from a JSON array or CSV file.
// POST /api/v1/organizations/:id/invitations/bulk
func (h *OrganizationsHandler) BulkInvite(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberInvite); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var invites []BulkInviteEntry

	// Check content type to determine if JSON or CSV
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle CSV file upload
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file upload required"})
			return
		}
		defer file.Close()

		invites, err = parseCSVInvites(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Handle JSON array
		var req struct {
			Invites []BulkInviteEntry `json:"invites" binding:"required,dive"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		invites = req.Invites
	}

	if len(invites) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no invitations provided"})
		return
	}

	if len(invites) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 100 invitations per batch"})
		return
	}

	response := BulkInviteResponse{
		Successful: make([]BulkInviteResult, 0),
		Failed:     make([]BulkInviteError, 0),
		Total:      len(invites),
	}

	for _, invite := range invites {
		email := strings.ToLower(strings.TrimSpace(invite.Email))
		role := models.OrgRole(invite.Role)

		// Check if user can assign this role
		canAssign, err := h.rbac.CanAssignRole(c.Request.Context(), user.ID, orgID, role)
		if err != nil || !canAssign {
			response.Failed = append(response.Failed, BulkInviteError{
				Email: email,
				Error: "cannot invite with this role",
			})
			continue
		}

		// Check if already a member
		existingUser, err := h.store.GetUserByEmail(c.Request.Context(), email)
		if err == nil && existingUser != nil {
			membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), existingUser.ID, orgID)
			if err == nil && membership != nil {
				response.Failed = append(response.Failed, BulkInviteError{
					Email: email,
					Error: "user is already a member",
				})
				continue
			}
		}

		// Check for existing pending invitation
		pending, err := h.store.GetPendingInvitationsByEmail(c.Request.Context(), email)
		if err == nil {
			hasExisting := false
			for _, inv := range pending {
				if inv.OrgID == orgID && time.Now().Before(inv.ExpiresAt) && inv.AcceptedAt == nil {
					hasExisting = true
					break
				}
			}
			if hasExisting {
				response.Failed = append(response.Failed, BulkInviteError{
					Email: email,
					Error: "pending invitation already exists",
				})
				continue
			}
		}

		// Generate token
		token, err := generateInviteToken()
		if err != nil {
			response.Failed = append(response.Failed, BulkInviteError{
				Email: email,
				Error: "failed to generate token",
			})
			continue
		}

		// Create invitation
		inv := models.NewOrgInvitation(
			orgID,
			email,
			role,
			token,
			user.ID,
			time.Now().Add(7*24*time.Hour),
		)

		if err := h.store.CreateInvitation(c.Request.Context(), inv); err != nil {
			response.Failed = append(response.Failed, BulkInviteError{
				Email: email,
				Error: "failed to create invitation",
			})
			continue
		}

		response.Successful = append(response.Successful, BulkInviteResult{
			Email: email,
			Role:  invite.Role,
			Token: token,
		})
	}

	h.logger.Info().
		Str("org_id", orgID.String()).
		Int("total", response.Total).
		Int("successful", len(response.Successful)).
		Int("failed", len(response.Failed)).
		Str("inviter_id", user.ID.String()).
		Msg("bulk invitations created")

	c.JSON(http.StatusCreated, response)
}

// parseCSVInvites parses a CSV file containing invite data (email,role format).
func parseCSVInvites(reader io.Reader) ([]BulkInviteEntry, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	var invites []BulkInviteEntry
	lineNum := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		lineNum++

		// Skip header row if present
		if lineNum == 1 && (strings.EqualFold(record[0], "email") || strings.EqualFold(record[0], "e-mail")) {
			continue
		}

		if len(record) < 1 {
			continue
		}

		email := strings.TrimSpace(record[0])
		if email == "" {
			continue
		}

		// Default role is member
		role := "member"
		if len(record) >= 2 {
			roleStr := strings.TrimSpace(strings.ToLower(record[1]))
			switch roleStr {
			case "admin":
				role = "admin"
			case "member":
				role = "member"
			case "readonly", "read-only", "viewer":
				role = "readonly"
			default:
				role = "member"
			}
		}

		invites = append(invites, BulkInviteEntry{
			Email: email,
			Role:  role,
		})
	}

	return invites, nil
}

// ResendInvitation resends an invitation email.
// POST /api/v1/organizations/:id/invitations/:invitation_id/resend
func (h *OrganizationsHandler) ResendInvitation(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, orgID, auth.PermMemberInvite); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Get invitation
	inv, err := h.store.GetInvitationByID(c.Request.Context(), invitationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	// Verify invitation belongs to this org
	if inv.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	if inv.IsAccepted() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation has already been accepted"})
		return
	}

	if inv.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation has expired"})
		return
	}

	// Update resent timestamp
	if err := h.store.UpdateInvitationResent(c.Request.Context(), invitationID); err != nil {
		h.logger.Warn().Err(err).Msg("failed to update resent timestamp")
	}

	h.logger.Info().
		Str("invitation_id", invitationID.String()).
		Str("email", inv.Email).
		Str("resent_by", user.ID.String()).
		Msg("invitation resent")

	c.JSON(http.StatusOK, gin.H{
		"message": "invitation resent",
		"token":   inv.Token,
	})
}

// GetInvitationByToken returns invitation details by token (public endpoint).
// GET /api/v1/invitations/:token
func (h *OrganizationsHandler) GetInvitationByToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	inv, err := h.store.GetInvitationByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation not found"})
		return
	}

	if inv.IsExpired() {
		c.JSON(http.StatusGone, gin.H{"error": "invitation has expired"})
		return
	}

	if inv.IsAccepted() {
		c.JSON(http.StatusConflict, gin.H{"error": "invitation has already been accepted"})
		return
	}

	// Get organization details
	org, err := h.store.GetOrganizationByID(c.Request.Context(), inv.OrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get organization"})
		return
	}

	// Get inviter details
	inviterName := "A team member"
	inviter, err := h.store.GetUserByID(c.Request.Context(), inv.InvitedBy)
	if err == nil && inviter != nil {
		inviterName = inviter.Name
		if inviterName == "" {
			inviterName = inviter.Email
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"invitation": gin.H{
			"id":           inv.ID,
			"org_id":       inv.OrgID,
			"org_name":     org.Name,
			"email":        inv.Email,
			"role":         inv.Role,
			"inviter_name": inviterName,
			"expires_at":   inv.ExpiresAt,
			"created_at":   inv.CreatedAt,
		},
	})
}
