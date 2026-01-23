package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/support"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// SupportBundleResponse is the response containing bundle metadata.
type SupportBundleResponse struct {
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SupportHandler handles support-related HTTP endpoints.
type SupportHandler struct {
	version   string
	commit    string
	buildDate string
	logDir    string
	logger    zerolog.Logger
}

// NewSupportHandler creates a new SupportHandler.
func NewSupportHandler(version, commit, buildDate, logDir string, logger zerolog.Logger) *SupportHandler {
	return &SupportHandler{
		version:   version,
		commit:    commit,
		buildDate: buildDate,
		logDir:    logDir,
		logger:    logger.With().Str("component", "support_handler").Logger(),
	}
}

// RegisterRoutes registers support routes that require authentication.
func (h *SupportHandler) RegisterRoutes(r *gin.RouterGroup) {
	support := r.Group("/support")
	{
		support.POST("/bundle", h.GenerateBundle)
	}
}

// GenerateBundle generates and downloads a support bundle.
// POST /api/v1/support/bundle
// @Summary Generate support bundle
// @Description Generates a diagnostic bundle containing sanitized logs, configuration, and system information
// @Tags Support
// @Produce application/zip
// @Success 200 {file} binary "Support bundle zip file"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security SessionAuth
// @Router /support/bundle [post]
func (h *SupportHandler) GenerateBundle(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	opts := support.DefaultBundleOptions()
	opts.IncludeServerInfo = true
	opts.LogDir = h.logDir

	generator := support.NewGenerator(h.logger, opts)

	bundleData := support.BundleData{
		ServerInfo: &support.ServerInfo{
			Version:   h.version,
			Commit:    h.commit,
			BuildDate: h.buildDate,
		},
		Config:     nil, // Server config not included for security
		CustomData: make(map[string]any),
	}

	data, info, err := generator.Generate(ctx, bundleData)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate support bundle")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate support bundle"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+info.Filename)
	c.Header("Content-Type", "application/zip")
	c.Data(http.StatusOK, "application/zip", data)
}
