// Package middleware provides HTTP middleware for the Keldris API.
package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// FeatureContextKey is the context key for feature access info.
const FeatureContextKey ContextKey = "feature_access"

// OrgTierContextKey is the context key for the organization's license tier.
const OrgTierContextKey ContextKey = "org_tier"

// FeatureAccess contains information about feature access for the current request.
type FeatureAccess struct {
	OrgID   uuid.UUID       `json:"org_id"`
	Tier    license.Tier    `json:"tier"`
	Feature license.Feature `json:"feature,omitempty"`
	Enabled bool            `json:"enabled"`
}

// FeatureGateMiddleware returns a Gin middleware that blocks requests if the organization
// doesn't have access to the specified feature. Returns 402 Payment Required with upgrade info.
func FeatureGateMiddleware(checker *license.FeatureChecker, feature license.Feature, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "feature_gate_middleware").Str("feature", string(feature)).Logger()

	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		result, err := checker.CheckFeatureWithInfo(c.Request.Context(), user.CurrentOrgID, feature)
		if err != nil {
			log.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to check feature access")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature access"})
			return
		}

		// Store feature access info in context
		c.Set(string(FeatureContextKey), &FeatureAccess{
			OrgID:   user.CurrentOrgID,
			Tier:    result.CurrentTier,
			Feature: feature,
			Enabled: result.Enabled,
		})
		c.Set(string(OrgTierContextKey), result.CurrentTier)

		if !result.Enabled {
			log.Debug().
				Str("org_id", user.CurrentOrgID.String()).
				Str("current_tier", string(result.CurrentTier)).
				Str("required_tier", string(result.RequiredTier)).
				Msg("feature access denied")

			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":         "feature not available",
				"feature":       string(feature),
				"current_tier":  string(result.CurrentTier),
				"required_tier": string(result.RequiredTier),
				"upgrade_info":  result.UpgradeInfo,
			})
			return
		}

		log.Debug().
			Str("org_id", user.CurrentOrgID.String()).
			Str("tier", string(result.CurrentTier)).
			Msg("feature access granted")

		c.Next()
	}
}

// LoadTierMiddleware returns a Gin middleware that loads the organization's license tier
// into the context without blocking the request. Useful for endpoints that need tier info
// but don't require a specific feature.
func LoadTierMiddleware(checker *license.FeatureChecker, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "load_tier_middleware").Logger()

	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.Next()
			return
		}

		tier, err := checker.GetOrgTier(c.Request.Context(), user.CurrentOrgID)
		if err != nil {
			log.Warn().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to load org tier, defaulting to free")
			tier = license.TierFree
		}

		c.Set(string(OrgTierContextKey), tier)
		c.Next()
	}
}

// GetOrgTier retrieves the organization's license tier from the Gin context.
// Returns TierFree if not set.
func GetOrgTier(c *gin.Context) license.Tier {
	tier, exists := c.Get(string(OrgTierContextKey))
	if !exists {
		return license.TierFree
	}
	t, ok := tier.(license.Tier)
	if !ok {
		return license.TierFree
	}
	return t
}

// GetFeatureAccess retrieves the feature access info from the Gin context.
// Returns nil if not set.
func GetFeatureAccess(c *gin.Context) *FeatureAccess {
	access, exists := c.Get(string(FeatureContextKey))
	if !exists {
		return nil
	}
	fa, ok := access.(*FeatureAccess)
	if !ok {
		return nil
	}
	return fa
}

// RequireFeature is a helper that checks feature access inline within a handler.
// Returns true if the feature is accessible, false and aborts if not.
func RequireFeature(c *gin.Context, checker *license.FeatureChecker, feature license.Feature) bool {
	user := GetUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return false
	}

	result, err := checker.CheckFeatureWithInfo(c.Request.Context(), user.CurrentOrgID, feature)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check feature access"})
		return false
	}

	if !result.Enabled {
		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":         "feature not available",
			"feature":       string(feature),
			"current_tier":  string(result.CurrentTier),
			"required_tier": string(result.RequiredTier),
			"upgrade_info":  result.UpgradeInfo,
		})
		return false
	}

	return true
}

// FeatureGateGroup applies feature gating to a route group.
func FeatureGateGroup(group *gin.RouterGroup, checker *license.FeatureChecker, feature license.Feature, logger zerolog.Logger) {
	group.Use(FeatureGateMiddleware(checker, feature, logger))
}
