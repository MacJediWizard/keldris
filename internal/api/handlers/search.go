package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/db"
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
	store  SearchStore
	logger zerolog.Logger
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(store SearchStore, logger zerolog.Logger) *SearchHandler {
	return &SearchHandler{
		store:  store,
		logger: logger.With().Str("component", "search_handler").Logger(),
	}
}

// RegisterRoutes registers search routes on the given router group.
func (h *SearchHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/search", h.Search)
}

// Search performs a global search across resources.
// GET /api/v1/search
// Query params:
//   - q: search query (required)
//   - types: comma-separated list of types to search (agent, backup, schedule, repository)
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
