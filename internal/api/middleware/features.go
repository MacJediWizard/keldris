package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// LicenseContextKey is the context key for the current license.
const LicenseContextKey ContextKey = "license"

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
func FeatureMiddleware(feature license.Feature, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().
		Str("component", "feature_middleware").
		Str("feature", string(feature)).
		Logger()

	return func(c *gin.Context) {
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

// DynamicLicenseMiddleware reads the current license from the validator on each
// request instead of using a static license. This ensures license downgrades
// (from revocation or grace period expiry) take effect immediately.
func DynamicLicenseMiddleware(validator *license.Validator, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "dynamic_license_middleware").Logger()

	return func(c *gin.Context) {
		lic := validator.GetLicense()
		if lic == nil {
			lic = license.FreeLicense()
		}
		c.Set(string(LicenseContextKey), lic)
		log.Debug().
			Str("tier", string(lic.Tier)).
			Str("path", c.Request.URL.Path).
			Msg("dynamic license context set")
		c.Next()
	}
}
