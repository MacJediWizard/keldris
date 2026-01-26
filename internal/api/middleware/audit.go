package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AuditStore defines the interface for audit log persistence operations.
type AuditStore interface {
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// AuditMiddleware returns a Gin middleware that logs all API actions for compliance.
func AuditMiddleware(store AuditStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "audit_middleware").Logger()

	return func(c *gin.Context) {
		// Skip audit log endpoints to avoid recursion
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/audit-logs") {
			c.Next()
			return
		}

		// Skip health checks and other non-auditable endpoints
		if c.Request.URL.Path == "/api/v1/health" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get user from context (set by AuthMiddleware)
		user := GetUser(c)

		// Process request first
		c.Next()

		// After request processing, create audit log
		// Only audit authenticated requests
		if user == nil {
			return
		}

		// Determine action from HTTP method
		action := mapMethodToAction(c.Request.Method)
		if action == "" {
			return
		}

		// Get user's org ID
		dbUser, err := store.GetUserByID(c.Request.Context(), user.ID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user for audit")
			return
		}

		// Extract resource type and ID from path
		resourceType, resourceID := parseResourceFromPath(c.Request.URL.Path)
		if resourceType == "" {
			return
		}

		// Determine result based on response status
		result := models.AuditResultSuccess
		if c.Writer.Status() >= 400 {
			result = models.AuditResultFailure
		}
		if c.Writer.Status() == http.StatusForbidden || c.Writer.Status() == http.StatusUnauthorized {
			result = models.AuditResultDenied
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Create audit log entry
		auditLog := models.NewAuditLog(dbUser.OrgID, action, resourceType, result).
			WithUser(user.ID).
			WithRequestInfo(clientIP, c.Request.UserAgent())

		if resourceID != uuid.Nil {
			auditLog.WithResource(resourceID)
		}

		// Save audit log asynchronously to not block the response
		go func(ctx context.Context, entry *models.AuditLog) {
			if err := store.CreateAuditLog(ctx, entry); err != nil {
				log.Error().Err(err).
					Str("action", string(entry.Action)).
					Str("resource_type", entry.ResourceType).
					Msg("failed to create audit log")
			}
		}(context.Background(), auditLog)
	}
}

// mapMethodToAction maps HTTP methods to audit actions.
func mapMethodToAction(method string) models.AuditAction {
	switch method {
	case http.MethodGet:
		return models.AuditActionRead
	case http.MethodPost:
		return models.AuditActionCreate
	case http.MethodPut, http.MethodPatch:
		return models.AuditActionUpdate
	case http.MethodDelete:
		return models.AuditActionDelete
	default:
		return ""
	}
}

// parseResourceFromPath extracts the resource type and ID from the API path.
func parseResourceFromPath(path string) (string, uuid.UUID) {
	// Remove /api/v1/ prefix
	path = strings.TrimPrefix(path, "/api/v1/")
	path = strings.TrimPrefix(path, "/")

	// Split by /
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", uuid.Nil
	}

	resourceType := parts[0]

	// Check for resource ID
	var resourceID uuid.UUID
	if len(parts) >= 2 {
		if id, err := uuid.Parse(parts[1]); err == nil {
			resourceID = id
		}
	}

	// Map resource types to singular form
	switch resourceType {
	case "agents":
		return "agent", resourceID
	case "agent-registration-codes":
		return "agent_registration_code", resourceID
	case "repositories":
		return "repository", resourceID
	case "schedules":
		return "schedule", resourceID
	case "backups":
		return "backup", resourceID
	case "users":
		return "user", resourceID
	case "organizations":
		return "organization", resourceID
	case "ip-allowlists":
		return "ip_allowlist", resourceID
	case "ip-allowlist-settings":
		return "ip_allowlist_settings", uuid.Nil
	case "ip-blocked-attempts":
		return "ip_blocked_attempt", uuid.Nil
	case "auth":
		// Handle auth endpoints
		if len(parts) >= 2 {
			switch parts[1] {
			case "login", "callback":
				return "session", uuid.Nil
			case "logout":
				return "session", uuid.Nil
			}
		}
		return "auth", uuid.Nil
	default:
		return resourceType, resourceID
	}
}

// LogAuditEvent is a helper function to manually log audit events from handlers.
func LogAuditEvent(store AuditStore, logger zerolog.Logger, orgID, userID uuid.UUID, action models.AuditAction, resourceType string, resourceID *uuid.UUID, result models.AuditResult, details string) {
	auditLog := models.NewAuditLog(orgID, action, resourceType, result).
		WithUser(userID).
		WithDetails(details)

	if resourceID != nil {
		auditLog.WithResource(*resourceID)
	}

	if err := store.CreateAuditLog(context.Background(), auditLog); err != nil {
		logger.Error().Err(err).
			Str("action", string(action)).
			Str("resource_type", resourceType).
			Msg("failed to create audit log")
	}
}
