package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// PasswordPolicyStore defines the interface for password policy persistence operations.
type PasswordPolicyStore interface {
	GetPasswordPolicyByOrgID(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error)
	GetOrCreatePasswordPolicy(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error)
	UpdatePasswordPolicy(ctx context.Context, policy *models.PasswordPolicy) error
	GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*models.PasswordHistory, error)
	CreatePasswordHistory(ctx context.Context, history *models.PasswordHistory) error
	CleanupPasswordHistory(ctx context.Context, userID uuid.UUID, keepCount int) error
	GetUserPasswordInfo(ctx context.Context, userID uuid.UUID) (*models.UserPasswordInfo, error)
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string, expiresAt *time.Time) error
}

// PasswordPoliciesHandler handles password policy HTTP endpoints.
type PasswordPoliciesHandler struct {
	store     PasswordPolicyStore
	validator *auth.PasswordValidator
	logger    zerolog.Logger
}

// NewPasswordPoliciesHandler creates a new PasswordPoliciesHandler.
func NewPasswordPoliciesHandler(store PasswordPolicyStore, logger zerolog.Logger) *PasswordPoliciesHandler {
	return &PasswordPoliciesHandler{
		store:     store,
		validator: auth.NewPasswordValidator(store),
		logger:    logger.With().Str("component", "password_policies_handler").Logger(),
	}
}

// RegisterRoutes registers password policy routes on the given router group.
func (h *PasswordPoliciesHandler) RegisterRoutes(r *gin.RouterGroup) {
	policies := r.Group("/password-policies")
	{
		policies.GET("", h.Get)
		policies.PUT("", h.Update)
		policies.GET("/requirements", h.GetRequirements)
		policies.POST("/validate", h.ValidatePassword)
	}

	// Password management endpoints
	password := r.Group("/password")
	{
		password.POST("/change", h.ChangePassword)
		password.GET("/expiration", h.GetExpirationInfo)
	}
}

// Get returns the password policy for the current organization.
// GET /api/v1/password-policies
func (h *PasswordPoliciesHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	policy, err := h.store.GetOrCreatePasswordPolicy(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get password policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password policy"})
		return
	}

	c.JSON(http.StatusOK, models.PasswordPolicyResponse{
		Policy:       *policy,
		Requirements: policy.GetRequirements(),
	})
}

// Update updates the password policy for the current organization.
// PUT /api/v1/password-policies
func (h *PasswordPoliciesHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can update password policies
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.UpdatePasswordPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing policy or create default
	policy, err := h.store.GetOrCreatePasswordPolicy(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get password policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password policy"})
		return
	}

	// Apply updates
	if req.MinLength != nil {
		policy.MinLength = *req.MinLength
	}
	if req.RequireUppercase != nil {
		policy.RequireUppercase = *req.RequireUppercase
	}
	if req.RequireLowercase != nil {
		policy.RequireLowercase = *req.RequireLowercase
	}
	if req.RequireNumber != nil {
		policy.RequireNumber = *req.RequireNumber
	}
	if req.RequireSpecial != nil {
		policy.RequireSpecial = *req.RequireSpecial
	}
	if req.MaxAgeDays != nil {
		if *req.MaxAgeDays == 0 {
			policy.MaxAgeDays = nil // 0 means disable expiration
		} else {
			policy.MaxAgeDays = req.MaxAgeDays
		}
	}
	if req.HistoryCount != nil {
		policy.HistoryCount = *req.HistoryCount
	}

	// Validate policy settings
	if policy.MinLength < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum password length must be at least 6"})
		return
	}
	if policy.MinLength > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimum password length cannot exceed 128"})
		return
	}

	if err := h.store.UpdatePasswordPolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update password policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password policy"})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("password policy updated")

	c.JSON(http.StatusOK, models.PasswordPolicyResponse{
		Policy:       *policy,
		Requirements: policy.GetRequirements(),
	})
}

// GetRequirements returns the password requirements in a user-friendly format.
// GET /api/v1/password-policies/requirements
func (h *PasswordPoliciesHandler) GetRequirements(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	policy, err := h.store.GetOrCreatePasswordPolicy(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get password policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password requirements"})
		return
	}

	c.JSON(http.StatusOK, policy.GetRequirements())
}

// ValidatePasswordRequest is the request for password validation.
type ValidatePasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// ValidatePassword validates a password against the organization's policy.
// POST /api/v1/password-policies/validate
func (h *PasswordPoliciesHandler) ValidatePassword(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req ValidatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.validator.ValidatePassword(c.Request.Context(), user.CurrentOrgID, req.Password)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to validate password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate password"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ChangePassword handles password change requests.
// POST /api/v1/password/change
func (h *PasswordPoliciesHandler) ChangePassword(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user's current password info
	passwordInfo, err := h.store.GetUserPasswordInfo(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user password info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user information"})
		return
	}

	// Verify user has password auth configured
	if passwordInfo.PasswordHash == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password authentication is not configured for this user"})
		return
	}

	// Verify current password
	if err := auth.VerifyPassword(req.CurrentPassword, *passwordInfo.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	// Validate new password against policy and history
	result, err := h.validator.ValidatePasswordWithHistory(c.Request.Context(), user.CurrentOrgID, user.ID, req.NewPassword)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to validate new password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate password"})
		return
	}

	if !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "password does not meet requirements",
			"errors": result.Errors,
		})
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to hash new password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	// Calculate expiration date based on policy
	policy, err := h.store.GetOrCreatePasswordPolicy(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get password policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	expiresAt := policy.CalculateExpirationDate(time.Now())

	// Update password
	if err := h.store.UpdateUserPassword(c.Request.Context(), user.ID, newHash, expiresAt); err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to update password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	// Record in history
	if err := h.validator.RecordPasswordChange(c.Request.Context(), user.CurrentOrgID, user.ID, newHash); err != nil {
		h.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to record password history")
		// Don't fail the password change for this
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Msg("password changed successfully")

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// GetExpirationInfo returns password expiration information for the current user.
// GET /api/v1/password/expiration
func (h *PasswordPoliciesHandler) GetExpirationInfo(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	passwordInfo, err := h.store.GetUserPasswordInfo(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user password info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get password information"})
		return
	}

	// If user doesn't have password auth, return empty info
	if passwordInfo.PasswordHash == nil {
		c.JSON(http.StatusOK, models.PasswordExpirationInfo{
			IsExpired:     false,
			MustChangeNow: false,
		})
		return
	}

	info := models.GetExpirationInfo(passwordInfo.PasswordExpiresAt, passwordInfo.MustChangePassword)
	c.JSON(http.StatusOK, info)
}
