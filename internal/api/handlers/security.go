package handlers

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// SecurityHeadersTestResponse contains the security headers currently applied.
type SecurityHeadersTestResponse struct {
	Status  string                        `json:"status" example:"ok"`
	Headers middleware.SecurityHeadersInfo `json:"headers"`
	Message string                        `json:"message" example:"Security headers are configured correctly"`
}

// SecurityHandler handles security-related HTTP endpoints.
type SecurityHandler struct {
	logger zerolog.Logger
}

// NewSecurityHandler creates a new SecurityHandler.
func NewSecurityHandler(logger zerolog.Logger) *SecurityHandler {
	return &SecurityHandler{
		logger: logger.With().Str("component", "security_handler").Logger(),
	}
}

// RegisterRoutes registers security routes on the given router group.
func (h *SecurityHandler) RegisterRoutes(r *gin.RouterGroup) {
	security := r.Group("/security")
	{
		security.GET("/headers/test", h.TestHeaders)
	}
}

// RegisterPublicRoutes registers public security routes for verification.
func (h *SecurityHandler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/security/headers/test", h.TestHeaders)
}

// TestHeaders returns the current security headers for testing and verification.
//
//	@Summary		Test security headers
//	@Description	Returns all security headers currently applied to responses
//	@Tags			Security
//	@Produce		json
//	@Success		200	{object}	SecurityHeadersTestResponse
//	@Router			/security/headers/test [get]
func (h *SecurityHandler) TestHeaders(c *gin.Context) {
	headers := middleware.GetSecurityHeadersFromContext(c)

	response := SecurityHeadersTestResponse{
		Status:  "ok",
		Headers: headers,
		Message: "Security headers are configured correctly",
	}

	// Validate essential headers are present
	issues := []string{}
	if headers.XFrameOptions == "" {
		issues = append(issues, "X-Frame-Options is not set")
	}
	if headers.XContentTypeOptions == "" {
		issues = append(issues, "X-Content-Type-Options is not set")
	}
	if headers.ContentSecurityPolicy == "" {
		issues = append(issues, "Content-Security-Policy is not set")
	}

	if len(issues) > 0 {
		response.Status = "warning"
		response.Message = "Some security headers are missing"
		h.logger.Warn().Strs("issues", issues).Msg("Security headers validation found issues")
	}

	c.JSON(http.StatusOK, response)
}
