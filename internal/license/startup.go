package license

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog/log"
)

// StartupConfig contains configuration for license validation at startup.
type StartupConfig struct {
	PublicKey        string
	WarningDays      int
	AllowUnlicensed  bool
	GracePeriodDays  int
}

// DefaultStartupConfig returns the default startup configuration.
func DefaultStartupConfig() StartupConfig {
	return StartupConfig{
		WarningDays:      30,
		AllowUnlicensed:  true,
		GracePeriodDays:  models.GracePeriodDays,
	}
}

// LicenseStore defines the interface for license storage operations.
type LicenseStore interface {
	GetActiveLicense(ctx context.Context) (*models.License, error)
	CreateLicenseAuditLog(ctx context.Context, licenseID *string, action string, details map[string]interface{}) error
}

// StartupValidator validates licenses at server startup.
type StartupValidator struct {
	config  StartupConfig
	manager *Manager
	store   LicenseStore
}

// NewStartupValidator creates a new startup validator.
func NewStartupValidator(config StartupConfig, store LicenseStore) (*StartupValidator, error) {
	if config.PublicKey == "" {
		if config.AllowUnlicensed {
			log.Warn().Msg("No license public key configured, running in unlicensed mode")
			return &StartupValidator{
				config: config,
				store:  store,
			}, nil
		}
		return nil, fmt.Errorf("license public key is required")
	}

	manager, err := NewManagerFromBase64(config.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("create license manager: %w", err)
	}

	return &StartupValidator{
		config:  config,
		manager: manager,
		store:   store,
	}, nil
}

// StartupResult contains the result of startup license validation.
type StartupResult struct {
	Valid          bool
	Status         models.LicenseStatus
	Tier           models.LicenseTier
	Message        string
	Warnings       []string
	DaysRemaining  int
	License        *models.License
}

// Validate performs license validation at startup.
func (sv *StartupValidator) Validate(ctx context.Context) (*StartupResult, error) {
	result := &StartupResult{
		Valid:    false,
		Status:   models.LicenseStatusInvalid,
		Tier:     models.LicenseTierCommunity,
		Warnings: []string{},
	}

	// Check if we have a manager (public key configured)
	if sv.manager == nil {
		if sv.config.AllowUnlicensed {
			result.Valid = true
			result.Status = models.LicenseStatusActive
			result.Message = "Running in unlicensed community mode"
			log.Info().Msg("Server starting in unlicensed community mode")
			return result, nil
		}
		return result, fmt.Errorf("license validation required but no public key configured")
	}

	// Get the active license from storage
	license, err := sv.store.GetActiveLicense(ctx)
	if err != nil {
		if sv.config.AllowUnlicensed {
			result.Valid = true
			result.Status = models.LicenseStatusActive
			result.Message = "No license found, running in community mode"
			log.Warn().Err(err).Msg("No license found, running in community mode")
			return result, nil
		}
		return result, fmt.Errorf("get active license: %w", err)
	}

	if license == nil {
		if sv.config.AllowUnlicensed {
			result.Valid = true
			result.Status = models.LicenseStatusActive
			result.Message = "No license configured, running in community mode"
			log.Warn().Msg("No license configured, running in community mode")
			return result, nil
		}
		return result, fmt.Errorf("no active license found")
	}

	// Validate the license key
	validationResult, err := sv.manager.SetLicense(license)
	if err != nil && err != ErrLicenseExpired {
		sv.logAuditEvent(ctx, license, "validation_failed", map[string]interface{}{
			"error": err.Error(),
		})
		return result, fmt.Errorf("validate license: %w", err)
	}

	result.License = license
	result.Tier = validationResult.Tier
	result.Status = validationResult.Status
	result.DaysRemaining = validationResult.DaysRemaining

	switch validationResult.Status {
	case models.LicenseStatusActive:
		result.Valid = true
		result.Message = fmt.Sprintf("License valid for %s tier", validationResult.Tier)

		// Check for expiry warnings
		if validationResult.DaysRemaining <= sv.config.WarningDays {
			warning := fmt.Sprintf("License expires in %d days", validationResult.DaysRemaining)
			result.Warnings = append(result.Warnings, warning)
			log.Warn().
				Str("tier", string(validationResult.Tier)).
				Int("days_remaining", validationResult.DaysRemaining).
				Msg("License expiring soon")
		} else {
			log.Info().
				Str("tier", string(validationResult.Tier)).
				Int("days_remaining", validationResult.DaysRemaining).
				Msg("License validated successfully")
		}

		sv.logAuditEvent(ctx, license, "validation_success", map[string]interface{}{
			"tier":           validationResult.Tier,
			"days_remaining": validationResult.DaysRemaining,
		})

	case models.LicenseStatusGracePeriod:
		result.Valid = true
		result.Message = fmt.Sprintf("License expired, %d days remaining in grace period", validationResult.DaysRemaining)
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("LICENSE EXPIRED: Your license has expired but is in the %d-day grace period.", models.GracePeriodDays),
			fmt.Sprintf("You have %d days remaining to renew your license.", validationResult.DaysRemaining),
			"After the grace period, the system will revert to community tier limits.",
		)

		log.Warn().
			Str("tier", string(validationResult.Tier)).
			Int("grace_days_remaining", validationResult.DaysRemaining).
			Msg("License is in grace period")

		sv.logAuditEvent(ctx, license, "grace_period_active", map[string]interface{}{
			"tier":                  validationResult.Tier,
			"grace_days_remaining": validationResult.DaysRemaining,
		})

	case models.LicenseStatusExpired:
		if sv.config.AllowUnlicensed {
			result.Valid = true
			result.Status = models.LicenseStatusExpired
			result.Tier = models.LicenseTierCommunity
			result.Message = "License expired, running in community mode"
			result.Warnings = append(result.Warnings,
				"LICENSE EXPIRED: Your license has expired and the grace period has ended.",
				"The system is now running in community tier with limited features.",
				"Please contact sales to renew your license.",
			)

			log.Warn().Msg("License expired past grace period, running in community mode")

			sv.logAuditEvent(ctx, license, "license_expired", map[string]interface{}{
				"fallback_tier": "community",
			})
		} else {
			result.Message = "License expired and grace period has ended"
			sv.logAuditEvent(ctx, license, "license_expired_blocked", nil)
			return result, ErrLicenseExpired
		}

	default:
		result.Message = "Invalid license"
		sv.logAuditEvent(ctx, license, "validation_invalid", nil)
		if !sv.config.AllowUnlicensed {
			return result, fmt.Errorf("invalid license status: %s", validationResult.Status)
		}
	}

	return result, nil
}

// logAuditEvent logs a license audit event.
func (sv *StartupValidator) logAuditEvent(ctx context.Context, license *models.License, action string, details map[string]interface{}) {
	if sv.store == nil {
		return
	}

	var licenseID *string
	if license != nil {
		id := license.ID.String()
		licenseID = &id
	}

	if details == nil {
		details = map[string]interface{}{}
	}
	details["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	if err := sv.store.CreateLicenseAuditLog(ctx, licenseID, action, details); err != nil {
		log.Error().Err(err).Str("action", action).Msg("Failed to create license audit log")
	}
}

// GetManager returns the license manager for limit enforcement.
func (sv *StartupValidator) GetManager() *Manager {
	if sv.manager == nil {
		// Return a manager with community defaults
		return &Manager{}
	}
	return sv.manager
}

// PrintStartupBanner prints license information on startup.
func PrintStartupBanner(result *StartupResult) {
	log.Info().Msg("=== License Information ===")
	log.Info().
		Str("status", string(result.Status)).
		Str("tier", string(result.Tier)).
		Msg(result.Message)

	if len(result.Warnings) > 0 {
		log.Warn().Msg("=== License Warnings ===")
		for _, warning := range result.Warnings {
			log.Warn().Msg(warning)
		}
	}

	if result.License != nil && result.DaysRemaining > 0 {
		log.Info().
			Int("days_remaining", result.DaysRemaining).
			Time("expires_at", result.License.ExpiresAt).
			Msg("License expiry information")
	}
	log.Info().Msg("===========================")
}
