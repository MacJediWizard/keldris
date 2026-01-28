package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/MacJediWizard/keldris/internal/activity"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ActivityStore defines the interface for activity event persistence operations.
type ActivityStore interface {
	GetActivityEvents(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) ([]*models.ActivityEvent, error)
	GetActivityEventCount(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) (int, error)
	GetRecentActivityEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ActivityEvent, error)
	GetActivityCategories(ctx context.Context, orgID uuid.UUID) (map[string]int, error)
	SearchActivityEvents(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]*models.ActivityEvent, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// ActivityHandler handles activity-related HTTP endpoints.
type ActivityHandler struct {
	store  ActivityStore
	feed   *activity.Feed
	logger zerolog.Logger
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(store ActivityStore, feed *activity.Feed, logger zerolog.Logger) *ActivityHandler {
	return &ActivityHandler{
		store:  store,
		feed:   feed,
		logger: logger.With().Str("component", "activity_handler").Logger(),
	}
}

// RegisterRoutes registers activity routes on the given router group.
func (h *ActivityHandler) RegisterRoutes(r *gin.RouterGroup) {
	activity := r.Group("/activity")
	{
		activity.GET("", h.List)
		activity.GET("/recent", h.Recent)
		activity.GET("/count", h.Count)
		activity.GET("/categories", h.Categories)
		activity.GET("/search", h.Search)
	}
}

// RegisterWebSocketRoute registers the WebSocket route for real-time activity feed.
func (h *ActivityHandler) RegisterWebSocketRoute(r *gin.Engine, authMiddleware gin.HandlerFunc) {
	r.GET("/ws/activity", authMiddleware, h.WebSocket)
}

// List returns activity events for the authenticated user's organization.
//
//	@Summary		List activity events
//	@Description	Returns activity events for the current organization with optional filtering
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Param			category	query		string	false	"Filter by category"
//	@Param			type		query		string	false	"Filter by event type"
//	@Param			user_id		query		string	false	"Filter by user ID"
//	@Param			agent_id	query		string	false	"Filter by agent ID"
//	@Param			start_time	query		string	false	"Filter events after this time (RFC3339)"
//	@Param			end_time	query		string	false	"Filter events before this time (RFC3339)"
//	@Param			limit		query		int		false	"Maximum number of events to return"	default(50)
//	@Param			offset		query		int		false	"Number of events to skip"				default(0)
//	@Success		200			{object}	map[string]interface{}
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity [get]
func (h *ActivityHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	filter := models.ActivityEventFilter{}

	// Parse query parameters
	if category := c.Query("category"); category != "" {
		cat := models.ActivityEventCategory(category)
		filter.Category = &cat
	}

	if eventType := c.Query("type"); eventType != "" {
		t := models.ActivityEventType(eventType)
		filter.Type = &t
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if agentIDStr := c.Query("agent_id"); agentIDStr != "" {
		if agentID, err := uuid.Parse(agentIDStr); err == nil {
			filter.AgentID = &agentID
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	events, err := h.store.GetActivityEvents(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list activity events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list activity events"})
		return
	}

	if events == nil {
		events = []*models.ActivityEvent{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// Recent returns the most recent activity events for the authenticated user's organization.
//
//	@Summary		Get recent activity events
//	@Description	Returns the most recent activity events for the current organization
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Maximum number of events to return"	default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity/recent [get]
func (h *ActivityHandler) Recent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := h.store.GetRecentActivityEvents(c.Request.Context(), dbUser.OrgID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get recent activity events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recent activity events"})
		return
	}

	if events == nil {
		events = []*models.ActivityEvent{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// Count returns the count of activity events for the authenticated user's organization.
//
//	@Summary		Count activity events
//	@Description	Returns the count of activity events for the current organization
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Param			category	query		string	false	"Filter by category"
//	@Param			type		query		string	false	"Filter by event type"
//	@Success		200			{object}	map[string]int
//	@Failure		401			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity/count [get]
func (h *ActivityHandler) Count(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	filter := models.ActivityEventFilter{}

	if category := c.Query("category"); category != "" {
		cat := models.ActivityEventCategory(category)
		filter.Category = &cat
	}

	if eventType := c.Query("type"); eventType != "" {
		t := models.ActivityEventType(eventType)
		filter.Type = &t
	}

	count, err := h.store.GetActivityEventCount(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to count activity events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count activity events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// Categories returns activity event categories with their counts.
//
//	@Summary		Get activity categories
//	@Description	Returns all activity event categories with their event counts
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity/categories [get]
func (h *ActivityHandler) Categories(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	categories, err := h.store.GetActivityCategories(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get activity categories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// Search searches activity events by title or description.
//
//	@Summary		Search activity events
//	@Description	Searches activity events by title, description, user name, agent name, or resource name
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Param			q		query		string	true	"Search query"
//	@Param			limit	query		int		false	"Maximum number of events to return"	default(50)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/activity/search [get]
func (h *ActivityHandler) Search(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := h.store.SearchActivityEvents(c.Request.Context(), dbUser.OrgID, query, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to search activity events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search activity events"})
		return
	}

	if events == nil {
		events = []*models.ActivityEvent{}
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// WebSocket handles the WebSocket connection for real-time activity feed.
//
//	@Summary		WebSocket activity feed
//	@Description	Establishes a WebSocket connection for real-time activity events
//	@Tags			Activity
//	@Success		101	"Switching Protocols"
//	@Failure		401	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/ws/activity [get]
func (h *ActivityHandler) WebSocket(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	h.feed.HandleWebSocket(c.Writer, c.Request, dbUser.OrgID, user.ID)
}
