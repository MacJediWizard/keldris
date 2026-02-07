// Package handlers provides HTTP handlers for the Keldris API.
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// PasswordResetHandler handles password reset HTTP endpoints.
type PasswordResetHandler struct {
	resetService *auth.PasswordResetService
	emailService *notifications.EmailService
	serverURL    string
	logger       zerolog.Logger
}

// PasswordResetStore extends the store interface for password reset operations.
type PasswordResetStore interface {
	auth.PasswordResetStore
}

// NewPasswordResetHandler creates a new PasswordResetHandler.
func NewPasswordResetHandler(
	store PasswordResetStore,
	emailService *notifications.EmailService,
	serverURL string,
	logger zerolog.Logger,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		resetService: auth.NewPasswordResetService(store, logger),
		emailService: emailService,
		serverURL:    serverURL,
		logger:       logger.With().Str("component", "password_reset_handler").Logger(),
	}
}

// RegisterPublicRoutes registers the password reset routes that don't require authentication.
func (h *PasswordResetHandler) RegisterPublicRoutes(r *gin.Engine) {
	resetGroup := r.Group("/auth/reset-password")
	{
		resetGroup.POST("/request", h.RequestReset)
		resetGroup.POST("/reset", h.ResetPassword)
		resetGroup.GET("/validate/:token", h.ValidateToken)
	}
}

// RequestReset handles password reset requests.
//
//	@Summary		Request password reset
//	@Description	Initiates a password reset flow by sending a reset email to the user.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.PasswordResetRequestPayload	true	"Email address"
//	@Success		200		{object}	models.PasswordResetResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		429		{object}	map[string]string	"Rate limit exceeded"
//	@Router			/auth/reset-password/request [post]
func (h *PasswordResetHandler) RequestReset(c *gin.Context) {
	var req models.PasswordResetRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Request password reset
	resetRequest, err := h.resetService.RequestReset(c.Request.Context(), req.Email, ipAddress, userAgent)
	if err != nil {
		if err == auth.ErrResetRateLimited {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many password reset requests. Please try again later.",
			})
			return
		}
		h.logger.Error().Err(err).Msg("failed to request password reset")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Always return success to prevent email enumeration
	// Only send email if we got a valid reset request (user exists and has password auth)
	if resetRequest != nil && h.emailService != nil {
		// Generate reset URL
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", h.serverURL, resetRequest.Token)

		emailData := notifications.PasswordResetRequestData{
			UserName:  resetRequest.UserName,
			UserEmail: resetRequest.UserEmail,
			ResetURL:  resetURL,
			ExpiresAt: resetRequest.ExpiresAt,
		}

		// Send email in background to not block response
		go func() {
			if err := h.emailService.SendPasswordResetRequest([]string{resetRequest.UserEmail}, emailData); err != nil {
				h.logger.Error().
					Err(err).
					Str("email", resetRequest.UserEmail).
					Msg("failed to send password reset email")
			}
		}()
	}

	c.JSON(http.StatusOK, models.PasswordResetResponse{
		Message: "If an account exists with that email, you will receive a password reset link.",
	})
}

// ValidateToken validates a password reset token.
//
//	@Summary		Validate reset token
//	@Description	Validates a password reset token without consuming it.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			token	path		string	true	"Reset token"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Router			/auth/reset-password/validate/{token} [get]
func (h *PasswordResetHandler) ValidateToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	user, err := h.resetService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		switch err {
		case auth.ErrResetTokenExpired:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "This password reset link has expired. Please request a new one.",
				"code":  "token_expired",
			})
		case auth.ErrResetTokenUsed:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "This password reset link has already been used.",
				"code":  "token_used",
			})
		case auth.ErrResetTokenInvalid:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid password reset link.",
				"code":  "token_invalid",
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid password reset link.",
				"code":  "token_invalid",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"email": user.Email,
	})
}

// ResetPassword resets the user's password.
//
//	@Summary		Reset password
//	@Description	Resets the user's password using a valid reset token.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.PasswordResetPayload	true	"Reset token and new password"
//	@Success		200		{object}	models.PasswordResetResponse
//	@Failure		400		{object}	map[string]string
//	@Router			/auth/reset-password/reset [post]
func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req models.PasswordResetPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Validate the token first to get the user for email notification
	user, err := h.resetService.ValidateToken(c.Request.Context(), req.Token)
	if err != nil {
		switch err {
		case auth.ErrResetTokenExpired:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "This password reset link has expired. Please request a new one.",
				"code":  "token_expired",
			})
		case auth.ErrResetTokenUsed:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "This password reset link has already been used.",
				"code":  "token_used",
			})
		case auth.ErrResetTokenInvalid:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid password reset link.",
				"code":  "token_invalid",
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid password reset link.",
				"code":  "token_invalid",
			})
		}
		return
	}

	// Reset the password
	if err := h.resetService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword, ipAddress, userAgent); err != nil {
		h.logger.Error().Err(err).Msg("failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Send success notification email
	if h.emailService != nil {
		emailData := notifications.PasswordResetSuccessData{
			UserName:  user.Name,
			UserEmail: user.Email,
			ChangedAt: time.Now(),
		}

		go func() {
			if err := h.emailService.SendPasswordResetSuccess([]string{user.Email}, emailData); err != nil {
				h.logger.Error().
					Err(err).
					Str("email", user.Email).
					Msg("failed to send password reset success email")
			}
		}()
	}

	c.JSON(http.StatusOK, models.PasswordResetResponse{
		Message: "Your password has been reset successfully. You can now log in with your new password.",
	})
}
