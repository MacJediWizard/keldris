package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// EmailVerificationStore defines the interface for email verification persistence.
type EmailVerificationStore interface {
	CreateEmailVerificationToken(ctx context.Context, token *auth.EmailVerificationToken) error
	GetEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (*auth.EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	SetUserEmailVerified(ctx context.Context, userID uuid.UUID) error
	InvalidateUserVerificationTokens(ctx context.Context, userID uuid.UUID) error
	IsUserEmailVerified(ctx context.Context, userID uuid.UUID) (bool, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByIDForVerification(ctx context.Context, userID uuid.UUID) (auth.VerifiableUser, error)
	AdminSetUserEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error
	GetSecuritySettings(ctx context.Context, orgID uuid.UUID) (*settings.SecuritySettings, error)
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// EmailVerificationHandler handles email verification HTTP endpoints.
type EmailVerificationHandler struct {
	store        EmailVerificationStore
	sessions     *auth.SessionStore
	emailService *notifications.EmailService
	baseURL      string
	logger       zerolog.Logger
}

// NewEmailVerificationHandler creates a new email verification handler.
func NewEmailVerificationHandler(
	store EmailVerificationStore,
	sessions *auth.SessionStore,
	emailService *notifications.EmailService,
	baseURL string,
	logger zerolog.Logger,
) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		store:        store,
		sessions:     sessions,
		emailService: emailService,
		baseURL:      baseURL,
		logger:       logger.With().Str("component", "email_verification_handler").Logger(),
	}
}

// RegisterRoutes registers email verification routes on the given router group.
func (h *EmailVerificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/verify", h.VerifyEmail)
	r.POST("/resend", h.ResendVerification)
	r.GET("/status", h.GetVerificationStatus)
}

// RegisterAdminRoutes registers admin routes for email verification.
func (h *EmailVerificationHandler) RegisterAdminRoutes(r *gin.RouterGroup) {
	r.POST("/users/:id/verify-email", h.AdminVerifyEmail)
}

// VerifyEmailRequest is the request body for email verification.
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmailResponse is the response for email verification.
type VerifyEmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VerifyEmail verifies a user's email address using a token.
//
//	@Summary		Verify email address
//	@Description	Verifies a user's email address using the token sent via email
//	@Tags			Email Verification
//	@Accept			json
//	@Produce		json
//	@Param			body	body		VerifyEmailRequest	true	"Verification token"
//	@Success		200		{object}	VerifyEmailResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/auth/email/verify [post]
func (h *EmailVerificationHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the token for lookup
	tokenHash := auth.HashToken(req.Token)

	// Get the token from the database
	token, err := h.store.GetEmailVerificationTokenByHash(c.Request.Context(), tokenHash)
	if err != nil {
		h.logger.Debug().Err(err).Msg("verification token not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid or expired verification token"})
		return
	}

	// Check if token is already used
	if token.IsUsed() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification token has already been used"})
		return
	}

	// Check if token is expired
	if token.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification token has expired"})
		return
	}

	// Mark token as used
	if err := h.store.MarkEmailVerificationTokenUsed(c.Request.Context(), token.ID); err != nil {
		h.logger.Error().Err(err).Str("token_id", token.ID.String()).Msg("failed to mark token used")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	// Mark user as verified
	if err := h.store.SetUserEmailVerified(c.Request.Context(), token.UserID); err != nil {
		h.logger.Error().Err(err).Str("user_id", token.UserID.String()).Msg("failed to set user verified")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	// Invalidate all other tokens for this user
	if err := h.store.InvalidateUserVerificationTokens(c.Request.Context(), token.UserID); err != nil {
		h.logger.Warn().Err(err).Str("user_id", token.UserID.String()).Msg("failed to invalidate other tokens")
	}

	h.logger.Info().
		Str("user_id", token.UserID.String()).
		Msg("email verified successfully")

	c.JSON(http.StatusOK, VerifyEmailResponse{
		Success: true,
		Message: "Email verified successfully. You can now log in.",
	})
}

// ResendVerificationResponse is the response for resending verification.
type ResendVerificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ResendVerification resends the verification email to the current user.
//
//	@Summary		Resend verification email
//	@Description	Resends the verification email to the authenticated user
//	@Tags			Email Verification
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ResendVerificationResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/email/resend [post]
func (h *EmailVerificationHandler) ResendVerification(c *gin.Context) {
	// Get the authenticated user
	sessionUser, err := h.sessions.GetUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Check if user is already verified
	verified, err := h.store.IsUserEmailVerified(c.Request.Context(), sessionUser.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", sessionUser.ID.String()).Msg("failed to check verification status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check verification status"})
		return
	}

	if verified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is already verified"})
		return
	}

	// Get security settings for token expiration
	securitySettings, err := h.store.GetSecuritySettings(c.Request.Context(), sessionUser.CurrentOrgID)
	if err != nil {
		securitySettings = &settings.SecuritySettings{EmailVerificationTokenHours: 24}
	}

	// Invalidate existing tokens
	if err := h.store.InvalidateUserVerificationTokens(c.Request.Context(), sessionUser.ID); err != nil {
		h.logger.Warn().Err(err).Str("user_id", sessionUser.ID.String()).Msg("failed to invalidate existing tokens")
	}

	// Generate new token
	tokenExpiration := time.Duration(securitySettings.EmailVerificationTokenHours) * time.Hour
	token, rawToken, err := auth.NewEmailVerificationToken(sessionUser.ID, tokenExpiration)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate verification token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verification token"})
		return
	}

	// Store the token
	if err := h.store.CreateEmailVerificationToken(c.Request.Context(), token); err != nil {
		h.logger.Error().Err(err).Msg("failed to store verification token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate verification token"})
		return
	}

	// Send verification email
	verificationURL := auth.BuildVerificationURL(h.baseURL, rawToken)
	if h.emailService != nil {
		emailData := notifications.EmailVerificationData{
			UserName:        sessionUser.Name,
			Email:           sessionUser.Email,
			VerificationURL: verificationURL,
			ExpiresAt:       token.ExpiresAt,
		}
		if err := h.emailService.SendEmailVerification(sessionUser.Email, emailData); err != nil {
			h.logger.Error().Err(err).Str("email", sessionUser.Email).Msg("failed to send verification email")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification email"})
			return
		}
	}

	h.logger.Info().
		Str("user_id", sessionUser.ID.String()).
		Str("email", sessionUser.Email).
		Msg("verification email resent")

	c.JSON(http.StatusOK, ResendVerificationResponse{
		Success: true,
		Message: "Verification email sent. Please check your inbox.",
	})
}

// EmailVerificationStatusResponse is the response for email verification status.
type EmailVerificationStatusResponse struct {
	Email           string     `json:"email"`
	IsVerified      bool       `json:"is_verified"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	IsOIDCUser      bool       `json:"is_oidc_user"`
	RequiresEmail   bool       `json:"requires_email"`
}

// GetVerificationStatus returns the email verification status for the current user.
//
//	@Summary		Get verification status
//	@Description	Returns the email verification status for the authenticated user
//	@Tags			Email Verification
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	EmailVerificationStatusResponse
//	@Failure		401	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/auth/email/status [get]
func (h *EmailVerificationHandler) GetVerificationStatus(c *gin.Context) {
	sessionUser, err := h.sessions.GetUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	user, err := h.store.GetUserByIDForVerification(c.Request.Context(), sessionUser.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", sessionUser.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get verification status"})
		return
	}

	c.JSON(http.StatusOK, EmailVerificationStatusResponse{
		Email:         user.GetEmail(),
		IsVerified:    user.IsEmailVerified(),
		IsOIDCUser:    user.IsOIDCUser(),
		RequiresEmail: !user.IsEmailVerified() && !user.IsOIDCUser(),
	})
}

// AdminVerifyEmailRequest is the request body for admin email verification.
type AdminVerifyEmailRequest struct {
	Verified bool `json:"verified"`
}

// AdminVerifyEmail allows an admin to manually verify or unverify a user's email.
//
//	@Summary		Admin verify user email
//	@Description	Allows an admin to manually set a user's email verification status
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"User ID"
//	@Param			body	body		AdminVerifyEmailRequest	true	"Verification status"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/admin/users/{id}/verify-email [post]
func (h *EmailVerificationHandler) AdminVerifyEmail(c *gin.Context) {
	// Get the admin user
	sessionUser, err := h.sessions.GetUser(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Check if admin bypass is allowed
	securitySettings, err := h.store.GetSecuritySettings(c.Request.Context(), sessionUser.CurrentOrgID)
	if err != nil {
		securitySettings = &settings.SecuritySettings{AllowAdminVerificationBypass: true}
	}

	if !securitySettings.AllowAdminVerificationBypass {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin email verification bypass is disabled"})
		return
	}

	// Parse user ID
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req AdminVerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the user exists
	targetUser, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Set verification status
	if err := h.store.AdminSetUserEmailVerified(c.Request.Context(), userID, req.Verified); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to set verification status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update verification status"})
		return
	}

	// Create audit log
	action := "email verified"
	if !req.Verified {
		action = "email unverified"
	}
	auditLog := models.NewAuditLog(sessionUser.CurrentOrgID, models.AuditActionUpdate, "user", models.AuditResultSuccess).
		WithUser(sessionUser.ID).
		WithResource(userID).
		WithDetails("Admin " + action + " for: " + targetUser.Email)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("admin_id", sessionUser.ID.String()).
		Str("user_id", userID.String()).
		Bool("verified", req.Verified).
		Msg("admin set email verification status")

	status := "verified"
	if !req.Verified {
		status = "unverified"
	}
	c.JSON(http.StatusOK, gin.H{"message": "User email marked as " + status})
}
