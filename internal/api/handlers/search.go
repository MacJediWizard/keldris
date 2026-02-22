package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/search"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SearchStore defines the interface for search persistence operations.
type SearchStore interface {
	Search(ctx context.Context, orgID uuid.UUID, filter db.SearchFilter) ([]db.SearchResult, error)
}

// SearchHandler handles search-related HTTP endpoints.
type SearchHandler struct {
	store    SearchStore
	searcher *search.GlobalSearcher
	logger   zerolog.Logger
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(store SearchStore, logger zerolog.Logger) *SearchHandler {
	return &SearchHandler{
		store:  store,
		logger: logger.With().Str("component", "search_handler").Logger(),
	}
}

// SetGlobalSearcher sets the global searcher for enhanced search features.
func (h *SearchHandler) SetGlobalSearcher(searcher *search.GlobalSearcher) {
	h.searcher = searcher
}

// RegisterRoutes registers search routes on the given router group.
func (h *SearchHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/search", h.Search)
	r.GET("/search/grouped", h.SearchGrouped)
	r.GET("/search/suggestions", h.GetSuggestions)
	r.GET("/search/recent", h.GetRecentSearches)
	r.POST("/search/recent", h.SaveRecentSearch)
	r.DELETE("/search/recent/:id", h.DeleteRecentSearch)
	r.DELETE("/search/recent", h.ClearRecentSearches)
}

// Search performs a global search across resources.
// GET /api/v1/search
// Query params:
//   - q: search query (required)
//   - types: comma-separated list of types to search (agent, backup, snapshot, schedule, repository)
//   - status: filter by status
//   - tag_ids: comma-separated list of tag IDs to filter by
//   - date_from: filter by date range start (RFC3339)
//   - date_to: filter by date range end (RFC3339)
//   - size_min: filter by minimum size in bytes
//   - size_max: filter by maximum size in bytes
//   - limit: max results per type (default 10)
func (h *SearchHandler) Search(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}
	if len(query) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query too long (max 1000 characters)"})
		return
	}

	filter := db.SearchFilter{
		Query: query,
	}

	// Parse types filter
	if typesStr := c.Query("types"); typesStr != "" {
		filter.Types = strings.Split(typesStr, ",")
	}

	// Parse status filter
	filter.Status = c.Query("status")

	// Parse tag IDs filter
	if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
		tagIDStrings := strings.Split(tagIDsStr, ",")
		for _, idStr := range tagIDStrings {
			id, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
				return
			}
			filter.TagIDs = append(filter.TagIDs, id)
		}
	}

	// Parse date range filters
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format (use RFC3339)"})
			return
		}
		filter.DateFrom = &dateFrom
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format (use RFC3339)"})
			return
		}
		filter.DateTo = &dateTo
	}

	// Parse size filters
	if sizeMinStr := c.Query("size_min"); sizeMinStr != "" {
		sizeMin, err := strconv.ParseInt(sizeMinStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_min"})
			return
		}
		filter.SizeMin = &sizeMin
	}

	if sizeMaxStr := c.Query("size_max"); sizeMaxStr != "" {
		sizeMax, err := strconv.ParseInt(sizeMaxStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_max"})
			return
		}
		filter.SizeMax = &sizeMax
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit (1-100)"})
			return
		}
		filter.Limit = limit
	}

	results, err := h.store.Search(c.Request.Context(), user.CurrentOrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("search failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"query":   query,
		"total":   len(results),
	})
}

// SearchGrouped performs a global search and returns results grouped by type.
// GET /api/v1/search/grouped
// Query params: same as Search
func (h *SearchHandler) SearchGrouped(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "grouped search not available"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	filter := search.Filter{
		Query: query,
	}

	// Parse types filter
	if typesStr := c.Query("types"); typesStr != "" {
		filter.Types = strings.Split(typesStr, ",")
	}

	// Parse status filter
	filter.Status = c.Query("status")

	// Parse tag IDs filter
	if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
		tagIDStrings := strings.Split(tagIDsStr, ",")
		for _, idStr := range tagIDStrings {
			id, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag ID"})
				return
			}
			filter.TagIDs = append(filter.TagIDs, id)
		}
	}

	// Parse date range filters
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format (use RFC3339)"})
			return
		}
		filter.DateFrom = &dateFrom
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format (use RFC3339)"})
			return
		}
		filter.DateTo = &dateTo
	}

	// Parse size filters
	if sizeMinStr := c.Query("size_min"); sizeMinStr != "" {
		sizeMin, err := strconv.ParseInt(sizeMinStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_min"})
			return
		}
		filter.SizeMin = &sizeMin
	}

	if sizeMaxStr := c.Query("size_max"); sizeMaxStr != "" {
		sizeMax, err := strconv.ParseInt(sizeMaxStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid size_max"})
			return
		}
		filter.SizeMax = &sizeMax
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit (1-100)"})
			return
		}
		filter.Limit = limit
	}

	results, err := h.searcher.Search(c.Request.Context(), user.CurrentOrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("grouped search failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agents":       results.Agents,
		"backups":      results.Backups,
		"snapshots":    results.Snapshots,
		"schedules":    results.Schedules,
		"repositories": results.Repositories,
		"query":        query,
		"total":        results.Total,
	})
}

// GetSuggestions returns autocomplete suggestions for a partial query.
// GET /api/v1/search/suggestions
// Query params:
//   - q: search prefix (required)
//   - limit: max suggestions (default 10, max 20)
func (h *SearchHandler) GetSuggestions(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "suggestions not available"})
		return
	}

	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusOK, gin.H{"suggestions": []search.Suggestion{}})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	suggestions, err := h.searcher.GetSuggestions(c.Request.Context(), user.CurrentOrgID, prefix, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("prefix", prefix).Msg("failed to get suggestions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get suggestions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// GetRecentSearches returns the user's recent search history.
// GET /api/v1/search/recent
// Query params:
//   - limit: max results (default 10, max 20)
func (h *SearchHandler) GetRecentSearches(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recent searches not available"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	searches, err := h.searcher.GetRecentSearches(c.Request.Context(), user.ID, user.CurrentOrgID, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get recent searches")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recent searches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recent_searches": searches})
}

// SaveRecentSearchRequest is the request body for saving a recent search.
type SaveRecentSearchRequest struct {
	Query string   `json:"query" binding:"required"`
	Types []string `json:"types"`
}

// SaveRecentSearch saves a search query to the user's recent search history.
// POST /api/v1/search/recent
func (h *SearchHandler) SaveRecentSearch(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recent searches not available"})
		return
	}

	var req SaveRecentSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Query) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query too long (max 500 characters)"})
		return
	}

	err := h.searcher.SaveRecentSearch(c.Request.Context(), user.ID, user.CurrentOrgID, req.Query, req.Types)
	if err != nil {
		h.logger.Error().Err(err).Str("query", req.Query).Msg("failed to save recent search")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save recent search"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "search saved"})
}

// DeleteRecentSearch deletes a specific recent search.
// DELETE /api/v1/search/recent/:id
func (h *SearchHandler) DeleteRecentSearch(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recent searches not available"})
		return
	}

	searchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search ID"})
		return
	}

	err = h.searcher.DeleteRecentSearch(c.Request.Context(), user.ID, user.CurrentOrgID, searchID)
	if err != nil {
		h.logger.Error().Err(err).Str("search_id", searchID.String()).Msg("failed to delete recent search")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recent search"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "search deleted"})
}

// ClearRecentSearches clears all recent searches for the user.
// DELETE /api/v1/search/recent
func (h *SearchHandler) ClearRecentSearches(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if h.searcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recent searches not available"})
		return
	}

	err := h.searcher.ClearRecentSearches(c.Request.Context(), user.ID, user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to clear recent searches")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear recent searches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recent searches cleared"})
}
