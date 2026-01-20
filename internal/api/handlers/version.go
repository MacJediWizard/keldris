package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// VersionInfo contains server version information.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
}

// VersionHandler handles version-related HTTP endpoints.
type VersionHandler struct {
	info   VersionInfo
	logger zerolog.Logger
}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler(version, commit, buildDate string, logger zerolog.Logger) *VersionHandler {
	return &VersionHandler{
		info: VersionInfo{
			Version:   version,
			Commit:    commit,
			BuildDate: buildDate,
		},
		logger: logger.With().Str("component", "version_handler").Logger(),
	}
}

// RegisterRoutes registers version routes on the given router group.
func (h *VersionHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/version", h.Get)
}

// RegisterPublicRoutes registers version routes that don't require authentication.
func (h *VersionHandler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/version", h.Get)
}

// Get returns the server version information.
// GET /api/v1/version or GET /version
func (h *VersionHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, h.info)
}
