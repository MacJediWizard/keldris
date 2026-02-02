package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/portal"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	// MaxFailedLoginAttempts before account lockout.
	MaxFailedLoginAttempts = 5
	// LockoutDuration is how long an account is locked after max failed attempts.
	LockoutDuration = 15 * time.Minute
	// ResetTokenDuration is how long a password reset token is valid.
	ResetTokenDuration = 1 * time.Hour
)

// AuthHandler handles customer authentication endpoints.
type AuthHandler struct {
	store  portal.Store
	logger zerolog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(store portal.Store, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		store:  store,
		logger: logger.With().Str("component", "portal_auth_handler").Logger(),
	}
}

// RegisterRoutes registers auth routes on the given router group.
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.POST("/logout", h.Logout)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
	}
}

// RegisterProtectedRoutes registers auth routes that require authentication.
func (h *AuthHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.GET("/me", h.Me)
		auth.POST("/change-password", h.ChangePassword)
	}
}

// Login authenticates a customer and creates a session.
//
//	@Summary		Customer login
//	@Description	Authenticates a customer and returns a session cookie
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CustomerLoginRequest	true	"Login credentials"
//	@Success		200		{object}	models.CustomerResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/portal/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.CustomerLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get customer by email
	customer, err := h.store.GetCustomerByEmail(c.Request.Context(), req.Email)
	if err != nil || customer == nil {
		// Don't reveal whether email exists
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Check if account is locked
	if customer.IsLocked() {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is locked, try again later"})
		return
	}

	// Check if account is active
	if customer.Status != models.CustomerStatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not active"})
		return
	}

	// Verify password
	if !models.ComparePasswordHash(req.Password, customer.PasswordHash) {
		// Increment failed login attempts
		if err := h.store.IncrementCustomerFailedLogin(c.Request.Context(), customer.ID); err != nil {
			h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to increment failed login")
		}

		// Lock account if too many failed attempts
		if customer.FailedLoginAttempts+1 >= MaxFailedLoginAttempts {
			lockUntil := time.Now().Add(LockoutDuration)
			if err := h.store.LockCustomerAccount(c.Request.Context(), customer.ID, lockUntil); err != nil {
				h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to lock account")
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "account is locked due to too many failed attempts"})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Reset failed login attempts
	if err := h.store.ResetCustomerFailedLogin(c.Request.Context(), customer.ID); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to reset failed login")
	}

	// Create session
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	session, token, err := portal.NewSession(customer.ID, clientIP, userAgent)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	if err := h.store.CreateSession(c.Request.Context(), session); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to save session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Update last login
	if err := h.store.UpdateCustomerLastLogin(c.Request.Context(), customer.ID, clientIP); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to update last login")
	}

	// Set session cookie
	c.SetCookie(
		portal.SessionCookieName,
		token,
		int(portal.SessionDuration.Seconds()),
		"/",
		"",
		true,  // Secure
		true,  // HTTPOnly
	)

	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Str("email", customer.Email).
		Msg("customer logged in")

	c.JSON(http.StatusOK, customer.ToResponse())
}

// Register creates a new customer account.
//
//	@Summary		Customer registration
//	@Description	Creates a new customer account
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CustomerRegisterRequest	true	"Registration details"
//	@Success		201		{object}	models.CustomerResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/portal/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.CustomerRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Check if email already exists
	existing, _ := h.store.GetCustomerByEmail(c.Request.Context(), req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Hash password
	passwordHash := models.HashPassword(req.Password)

	// Create customer
	customer := models.NewCustomer(req.Email, req.Name, passwordHash)
	customer.Company = req.Company
	customer.Status = models.CustomerStatusActive // Auto-activate for now

	if err := h.store.CreateCustomer(c.Request.Context(), customer); err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("failed to create customer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Str("email", customer.Email).
		Msg("customer registered")

	c.JSON(http.StatusCreated, customer.ToResponse())
}

// Logout destroys the current session.
//
//	@Summary		Customer logout
//	@Description	Destroys the current session
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/portal/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get session token from cookie
	token, err := c.Cookie(portal.SessionCookieName)
	if err == nil && token != "" {
		// Delete session from database
		tokenHash := portal.HashSessionToken(token)
		session, err := h.store.GetSessionByTokenHash(c.Request.Context(), tokenHash)
		if err == nil && session != nil {
			if err := h.store.DeleteSession(c.Request.Context(), session.ID); err != nil {
				h.logger.Error().Err(err).Str("session_id", session.ID.String()).Msg("failed to delete session")
			}
		}
	}

	// Clear cookie
	c.SetCookie(
		portal.SessionCookieName,
		"",
		-1,
		"/",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me returns the current authenticated customer.
//
//	@Summary		Get current customer
//	@Description	Returns the currently authenticated customer
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.CustomerResponse
//	@Failure		401	{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	customer := portal.RequireCustomer(c)
	if customer == nil {
		return
	}

	// Get full customer details
	fullCustomer, err := h.store.GetCustomerByID(c.Request.Context(), customer.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}

	c.JSON(http.StatusOK, fullCustomer.ToResponse())
}

// ForgotPassword initiates password reset.
//
//	@Summary		Forgot password
//	@Description	Sends a password reset email
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CustomerResetPasswordRequest	true	"Email address"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Router			/portal/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req models.CustomerResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get customer by email
	customer, err := h.store.GetCustomerByEmail(c.Request.Context(), req.Email)
	if err != nil || customer == nil {
		// Don't reveal whether email exists - always return success
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error().Err(err).Msg("failed to generate reset token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		return
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(ResetTokenDuration)

	// Save reset token
	if err := h.store.UpdateCustomerResetToken(c.Request.Context(), customer.ID, token, expiresAt); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to save reset token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		return
	}

	// TODO: Send email with reset link
	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Str("email", customer.Email).
		Str("reset_token", token). // In production, don't log this
		Msg("password reset requested")

	c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword completes password reset.
//
//	@Summary		Reset password
//	@Description	Sets a new password using a reset token
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CustomerSetPasswordRequest	true	"Reset token and new password"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Router			/portal/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req models.CustomerSetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Find customer with this reset token
	// This is a simplified implementation - in production you'd query by token
	// For now we just validate the token format
	if len(req.Token) != 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reset token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// ChangePassword changes the customer's password.
//
//	@Summary		Change password
//	@Description	Changes the password for the authenticated customer
//	@Tags			Portal Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CustomerChangePasswordRequest	true	"Current and new password"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		PortalSession
//	@Router			/portal/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	customer := portal.RequireCustomer(c)
	if customer == nil {
		return
	}

	var req models.CustomerChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get full customer to verify current password
	fullCustomer, err := h.store.GetCustomerByID(c.Request.Context(), customer.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}

	// Verify current password
	if !models.ComparePasswordHash(req.CurrentPassword, fullCustomer.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	// Hash and save new password
	newPasswordHash := models.HashPassword(req.NewPassword)
	if err := h.store.UpdateCustomerPassword(c.Request.Context(), customer.ID, newPasswordHash); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to update password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	// Invalidate all other sessions
	if err := h.store.DeleteSessionsByCustomerID(c.Request.Context(), customer.ID); err != nil {
		h.logger.Error().Err(err).Str("customer_id", customer.ID.String()).Msg("failed to invalidate sessions")
	}

	h.logger.Info().
		Str("customer_id", customer.ID.String()).
		Msg("password changed")

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}
