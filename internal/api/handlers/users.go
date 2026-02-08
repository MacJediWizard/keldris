package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UsersStore defines the interface for user persistence operations.
type UsersStore interface {
	ListUsers(ctx context.Context, orgID uuid.UUID) ([]*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// UsersHandler handles user-related HTTP endpoints.
type UsersHandler struct {
	store  UsersStore
	rbac   *auth.RBAC
	logger zerolog.Logger
}

// NewUsersHandler creates a new UsersHandler.
func NewUsersHandler(store UsersStore, rbac *auth.RBAC, logger zerolog.Logger) *UsersHandler {
	return &UsersHandler{
		store:  store,
		rbac:   rbac,
		logger: logger.With().Str("component", "users_handler").Logger(),
	}
}

// RegisterRoutes registers user routes on the given router group.
func (h *UsersHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.List)
		users.GET("/:id", h.Get)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", h.Delete)
	}
}

// List returns all users for the current organization.
// GET /api/v1/users
func (h *UsersHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermMemberRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	users, err := h.store.ListUsers(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Get returns a single user by ID.
// GET /api/v1/users/:id
func (h *UsersHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermMemberRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	target, err := h.store.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if target.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, target)
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Role  *string `json:"role,omitempty"`
}

// Update updates a user's details.
// PUT /api/v1/users/:id
func (h *UsersHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermMemberUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	target, err := h.store.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if target.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Name != nil {
		target.Name = *req.Name
	}
	if req.Email != nil {
		target.Email = *req.Email
	}
	if req.Role != nil {
		role := models.UserRole(strings.TrimSpace(*req.Role))
		target.Role = role
	}

	if err := h.store.UpdateUser(c.Request.Context(), target); err != nil {
		h.logger.Error().Err(err).Str("user_id", id.String()).Msg("failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	h.logger.Info().
		Str("target_user_id", id.String()).
		Str("actor_id", user.ID.String()).
		Msg("user updated")

	c.JSON(http.StatusOK, target)
}

// Delete removes a user.
// DELETE /api/v1/users/:id
func (h *UsersHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermMemberRemove); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	target, err := h.store.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if target.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.store.DeleteUser(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "last owner") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last owner of an organization"})
			return
		}
		h.logger.Error().Err(err).Str("user_id", id.String()).Msg("failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	h.logger.Info().
		Str("target_user_id", id.String()).
		Str("actor_id", user.ID.String()).
		Msg("user deleted")

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}
