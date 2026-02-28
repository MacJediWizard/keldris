// Package middleware provides HTTP middleware for the Keldris API.
package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LicenseContextKey is the context key for the current license.
const LicenseContextKey ContextKey = "license"

// EntitlementContextKey is the context key for the current entitlement.
const EntitlementContextKey ContextKey = "entitlement"

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

// LicenseMiddleware stores the current license in the Gin context so downstream
// middleware and handlers can access it.
func LicenseMiddleware(lic *license.License, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "license_middleware").Logger()

	return func(c *gin.Context) {
		c.Set(string(LicenseContextKey), lic)
		log.Debug().
			Str("tier", string(lic.Tier)).
			Str("path", c.Request.URL.Path).
			Msg("license context set")
		c.Next()
	}
}

// FeatureMiddleware returns a Gin middleware that gates access to a feature.
// If the current license does not include the required feature, the request
// is rejected with HTTP 402 Payment Required.
// When a validator is provided, it uses the entitlement token for server-side
// gating and tracks feature usage for telemetry.
func FeatureMiddleware(feature license.Feature, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().
		Str("component", "feature_middleware").
		Str("feature", string(feature)).
		Logger()

	return func(c *gin.Context) {
		// Try entitlement-based gating first (server-side)
		if ent := GetEntitlement(c); ent != nil {
			if ent.IsExpired() {
				log.Info().Str("path", c.Request.URL.Path).Msg("entitlement token expired")
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error":   "entitlement expired, please reconnect to license server",
					"feature": string(feature),
				})
				return
			}
			if !ent.HasFeature(feature) {
				log.Info().
					Str("tier", string(ent.Tier)).
					Str("path", c.Request.URL.Path).
					Msg("feature not available in entitlement")
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error":   "feature not available on your current plan",
					"feature": string(feature),
					"tier":    string(ent.Tier),
				})
				return
			}

			// Layer 2: Entitlement nonce verification
			if ent.Nonce == "" {
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error":   "valid entitlement required",
					"feature": string(feature),
				})
				return
			}

			// Layer 3: Refresh token verification
			if validator := getValidator(c); validator != nil {
				if !validator.HasValidRefreshToken() {
					c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
						"error":   "service connection required",
						"feature": string(feature),
					})
					return
				}
			}

			// Track feature usage for telemetry
			if tracker, exists := c.Get("feature_usage_tracker"); exists {
				if t, ok := tracker.(*license.FeatureUsageTracker); ok {
					t.Record(string(feature))
				}
			}

			c.Next()
			return
		}

		// Fall back to tier-based gating (backward compat / no entitlement)
		lic := GetLicense(c)
		if lic == nil {
			log.Warn().Str("path", c.Request.URL.Path).Msg("no license in context")
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "license required",
				"feature": string(feature),
			})
			return
		}

		if !license.HasFeature(lic.Tier, feature) {
			log.Info().
				Str("tier", string(lic.Tier)).
				Str("path", c.Request.URL.Path).
				Msg("feature not available for tier")
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "feature not available on your current plan",
				"feature": string(feature),
				"tier":    string(lic.Tier),
			})
			return
		}

		// Layer 3: Refresh token verification (tier-based fallback path)
		if validator := getValidator(c); validator != nil {
			if !validator.HasValidRefreshToken() {
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error":   "service connection required",
					"feature": string(feature),
				})
				return
			}
		}

		// Track feature usage for telemetry
		if tracker, exists := c.Get("feature_usage_tracker"); exists {
			if t, ok := tracker.(*license.FeatureUsageTracker); ok {
				t.Record(string(feature))
			}
		}

		c.Next()
	}
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

		// Layer 2: Entitlement nonce verification
		if ent := GetEntitlement(c); ent != nil && ent.Nonce == "" {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "valid entitlement required",
				"feature": string(feature),
			})
			return
		}

		// Layer 3: Refresh token verification
		if validator := getValidator(c); validator != nil {
			if !validator.HasValidRefreshToken() {
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
					"error":   "service connection required",
					"feature": string(feature),
				})
				return
			}
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

// GetLicense retrieves the license from the Gin context.
// Returns nil if no license is set.
func GetLicense(c *gin.Context) *license.License {
	val, exists := c.Get(string(LicenseContextKey))
	if !exists {
		return nil
	}
	lic, ok := val.(*license.License)
	if !ok {
		return nil
	}
	return lic
}

// GetEntitlement retrieves the entitlement from the Gin context.
// Returns nil if no entitlement is set.
func GetEntitlement(c *gin.Context) *license.Entitlement {
	val, exists := c.Get(string(EntitlementContextKey))
	if !exists {
		return nil
	}
	ent, ok := val.(*license.Entitlement)
	if !ok {
		return nil
	}
	return ent
}

// DynamicLicenseMiddleware reads the current license from the validator on each
// request instead of using a static license. This ensures license downgrades
// (from revocation or grace period expiry) take effect immediately.
// It also sets the entitlement token and feature usage tracker in the context.
func DynamicLicenseMiddleware(validator *license.Validator, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "dynamic_license_middleware").Logger()

	return func(c *gin.Context) {
		// Check if killed
		if validator.IsKilled() {
			lic := license.FreeLicense()
			c.Set(string(LicenseContextKey), lic)
			log.Debug().Str("path", c.Request.URL.Path).Msg("instance killed - free tier only")
			c.Next()
			return
		}

		lic := validator.GetLicense()
		if lic == nil {
			lic = license.FreeLicense()
		}
		c.Set(string(LicenseContextKey), lic)

		// Set entitlement if available
		if ent := validator.GetEntitlement(); ent != nil {
			c.Set(string(EntitlementContextKey), ent)
		}

		// Set feature usage tracker
		c.Set("feature_usage_tracker", validator.GetFeatureUsageTracker())

		// Set validator reference for refresh token checks
		c.Set(string(ValidatorContextKey), validator)

		log.Debug().
			Str("tier", string(lic.Tier)).
			Str("path", c.Request.URL.Path).
			Msg("dynamic license context set")
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
// Performs three layers of verification: entitlement nonce, refresh token, and
// organization tier check.
func RequireFeature(c *gin.Context, checker *license.FeatureChecker, feature license.Feature) bool {
	// Layers 2+3 only apply when validator is active (phone-home mode).
	if validator := getValidator(c); validator != nil {
		// Layer 2: Entitlement nonce check
		ent := GetEntitlement(c)
		if ent == nil || ent.Nonce == "" {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "valid entitlement required",
				"feature": string(feature),
			})
			return false
		}

		// Layer 3: Refresh token check
		if !validator.HasValidRefreshToken() {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "service connection required",
				"feature": string(feature),
			})
			return false
		}
	}

	// If no checker is provided, skip org-tier check (test/air-gap).
	if checker == nil {
		return true
	}

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

// ValidatorContextKey is the context key for the license validator.
const ValidatorContextKey ContextKey = "license_validator"

// getValidator retrieves the license validator from the Gin context.
func getValidator(c *gin.Context) *license.Validator {
	val, exists := c.Get(string(ValidatorContextKey))
	if !exists {
		return nil
	}
	v, ok := val.(*license.Validator)
	if !ok {
		return nil
	}
	return v
}

// FeatureGateGroup applies feature gating to a route group.
func FeatureGateGroup(group *gin.RouterGroup, checker *license.FeatureChecker, feature license.Feature, logger zerolog.Logger) {
	group.Use(FeatureGateMiddleware(checker, feature, logger))
}
