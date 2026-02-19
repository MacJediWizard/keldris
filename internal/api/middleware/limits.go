package middleware

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ResourceCounter provides counts of resources for limit enforcement.
type ResourceCounter interface {
	CountAgentsByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	CountUsersByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	CountOrganizations(ctx context.Context) (int, error)
}

// LimitMiddleware returns a Gin middleware that enforces resource limits based on
// the current license tier. It checks the count of the specified resource type
// against the tier limits before allowing creation.
func LimitMiddleware(counter ResourceCounter, resource string, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().
		Str("component", "limit_middleware").
		Str("resource", resource).
		Logger()

	return func(c *gin.Context) {
		lic := GetLicense(c)
		if lic == nil {
			lic = license.FreeLicense()
		}

		user := GetUser(c)
		if user == nil {
			c.Next()
			return
		}

		var current int
		var limit int
		var err error

		switch resource {
		case "agents":
			limit = lic.Limits.MaxAgents
			if license.IsUnlimited(limit) {
				c.Next()
				return
			}
			current, err = counter.CountAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
		case "users":
			limit = lic.Limits.MaxUsers
			if license.IsUnlimited(limit) {
				c.Next()
				return
			}
			current, err = counter.CountUsersByOrgID(c.Request.Context(), user.CurrentOrgID)
		case "organizations":
			limit = lic.Limits.MaxOrgs
			if license.IsUnlimited(limit) {
				c.Next()
				return
			}
			current, err = counter.CountOrganizations(c.Request.Context())
		default:
			log.Warn().Str("resource", resource).Msg("unknown resource type for limit check")
			c.Next()
			return
		}

		if err != nil {
			log.Error().Err(err).Msg("failed to count resources for limit check")
			c.Next()
			return
		}

		if current >= limit {
			log.Info().
				Str("tier", string(lic.Tier)).
				Int("current", current).
				Int("limit", limit).
				Msg("resource limit exceeded")
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":    "limit_exceeded",
				"resource": resource,
				"current":  current,
				"limit":    limit,
				"tier":     string(lic.Tier),
				"message":  "You have reached the maximum number of " + resource + " for your plan. Please upgrade to add more.",
			})
			return
		}

		c.Next()
	}
}
