package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UserSessionStore defines the interface for user session persistence operations.
type UserSessionStore interface {
	ListActiveUserSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error)
	GetUserSessionByID(ctx context.Context, id uuid.UUID) (*models.UserSession, error)
	RevokeUserSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) (int64, error)
}

// UserSessionsHandler handles user session HTTP endpoints.
type UserSessionsHandler struct {
	store  UserSessionStore
	logger zerolog.Logger
}

// NewUserSessionsHandler creates a new UserSessionsHandler.
func NewUserSessionsHandler(store UserSessionStore, logger zerolog.Logger) *UserSessionsHandler {
	return &UserSessionsHandler{
		store:  store,
		logger: logger.With().Str("component", "user_sessions_handler").Logger(),
	}
}

// RegisterRoutes registers user session routes on the given router group.
func (h *UserSessionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	sessions := r.Group("/users/me/sessions")
	{
		sessions.GET("", h.List)
		sessions.DELETE("/:id", h.Revoke)
		sessions.DELETE("", h.RevokeAll)
	}
}

// List returns all active sessions for the current user.
//
//	@Summary		List user sessions
//	@Description	Returns all active sessions for the currently authenticated user
//	@Tags			User Sessions
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.UserSessionsResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/me/sessions [get]
func (h *UserSessionsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get current session ID from context
	currentSessionID := middleware.GetCurrentSessionID(c)

	sessions, err := h.store.ListActiveUserSessionsByUserID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to list user sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sessions"})
		return
	}

	// Convert to response format and mark current session
	response := models.UserSessionsResponse{
		Sessions: make([]models.UserSession, len(sessions)),
	}
	for i, s := range sessions {
		response.Sessions[i] = *s
		// Clear sensitive data
		response.Sessions[i].SessionTokenHash = ""
		// Mark if this is the current session
		if currentSessionID != uuid.Nil && s.ID == currentSessionID {
			response.Sessions[i].IsCurrent = true
		}
	}

	c.JSON(http.StatusOK, response)
}

// Revoke revokes a specific session.
//
//	@Summary		Revoke a session
//	@Description	Revokes a specific session for the current user
//	@Tags			User Sessions
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Session ID"
//	@Success		200	{object}	models.RevokeSessionsResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/me/sessions/{id} [delete]
func (h *UserSessionsHandler) Revoke(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Check if trying to revoke current session
	currentSessionID := middleware.GetCurrentSessionID(c)
	if currentSessionID != uuid.Nil && sessionID == currentSessionID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot revoke current session, use logout instead"})
		return
	}

	err = h.store.RevokeUserSession(c.Request.Context(), sessionID, user.ID)
	if err != nil {
		h.logger.Error().Err(err).
			Str("user_id", user.ID.String()).
			Str("session_id", sessionID.String()).
			Msg("failed to revoke session")
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("session_id", sessionID.String()).
		Msg("session revoked")

	c.JSON(http.StatusOK, models.RevokeSessionsResponse{Message: "session revoked"})
}

// RevokeAll revokes all sessions for the current user except the current one.
//
//	@Summary		Revoke all sessions
//	@Description	Revokes all sessions for the current user except the current session
//	@Tags			User Sessions
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	models.RevokeSessionsResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/users/me/sessions [delete]
func (h *UserSessionsHandler) RevokeAll(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get current session ID to exclude it
	currentSessionID := middleware.GetCurrentSessionID(c)
	var exceptID *uuid.UUID
	if currentSessionID != uuid.Nil {
		exceptID = &currentSessionID
	}

	count, err := h.store.RevokeAllUserSessions(c.Request.Context(), user.ID, exceptID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to revoke all sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Int64("revoked_count", count).
		Msg("all other sessions revoked")

	c.JSON(http.StatusOK, models.RevokeSessionsResponse{
		Message:      "all other sessions revoked",
		RevokedCount: int(count),
	})
}
