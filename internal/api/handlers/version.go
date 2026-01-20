package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// VersionInfo contains server version information.
type VersionInfo struct {
	Version   string `json:"version" example:"1.0.0"`
	Commit    string `json:"commit,omitempty" example:"abc1234"`
	BuildDate string `json:"build_date,omitempty" example:"2024-01-15T10:30:00Z"`
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
//
//	@Summary		Get version
//	@Description	Returns server version, commit hash, and build date
//	@Tags			Version
//	@Produce		json
//	@Success		200	{object}	VersionInfo
//	@Router			/version [get]
func (h *VersionHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, h.info)
}
