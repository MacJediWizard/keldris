package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LicenseContextKey is the context key for the current license.
const LicenseContextKey ContextKey = "license"

// EntitlementContextKey is the context key for the current entitlement.
const EntitlementContextKey ContextKey = "entitlement"

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

		// Track feature usage for telemetry
		if tracker, exists := c.Get("feature_usage_tracker"); exists {
			if t, ok := tracker.(*license.FeatureUsageTracker); ok {
				t.Record(string(feature))
			}
		}

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

		log.Debug().
			Str("tier", string(lic.Tier)).
			Str("path", c.Request.URL.Path).
			Msg("dynamic license context set")
		c.Next()
	}
}
