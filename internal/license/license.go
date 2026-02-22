// Package license provides license management and feature gating for Keldris.
package license

import (
	"errors"
	"fmt"
	"time"
)

// LicenseTier represents the subscription level.
type LicenseTier string

const (
	// TierFree is the default tier with basic functionality.
	TierFree LicenseTier = "free"
	// TierPro unlocks advanced features like OIDC and audit logs.
	TierPro LicenseTier = "pro"
	// TierEnterprise unlocks all features including multi-org.
	TierEnterprise LicenseTier = "enterprise"
)

// ValidTiers returns all valid license tiers.
func ValidTiers() []LicenseTier {
	return []LicenseTier{TierFree, TierPro, TierEnterprise}
}

// IsValid checks if the tier is a recognized value.
func (t LicenseTier) IsValid() bool {
	for _, valid := range ValidTiers() {
		if t == valid {
			return true
		}
	}
	return false
}

// License represents a Keldris license with tier, limits, and expiry.
type License struct {
	Tier              LicenseTier `json:"tier"`
	CustomerID        string      `json:"customer_id"`
	CustomerName      string      `json:"customer_name,omitempty"`
	ExpiresAt         time.Time   `json:"expires_at"`
	IssuedAt          time.Time   `json:"issued_at"`
	Limits            TierLimits  `json:"limits"`
	IsTrial           bool        `json:"is_trial"`
	TrialDurationDays int         `json:"trial_duration_days,omitempty"`
	TrialStartedAt    time.Time   `json:"trial_started_at,omitempty"`
}

// TrialDaysLeft returns the number of days remaining in the trial, or 0 if not a trial.
func (l *License) TrialDaysLeft() int {
	if !l.IsTrial {
		return 0
	}
	days := int(time.Until(l.ExpiresAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// ValidateLicense checks that a license is still valid (not expired).
func ValidateLicense(license *License) error {
	if license == nil {
		return errors.New("nil license")
	}

	if !license.Tier.IsValid() {
		return fmt.Errorf("unknown license tier: %s", license.Tier)
	}

	if time.Now().After(license.ExpiresAt) {
		return errors.New("license has expired")
	}

	if license.CustomerID == "" {
		return errors.New("missing customer ID")
	}

	return nil
}

// FreeLicense returns a default free-tier license with no expiry constraint.
func FreeLicense() *License {
	return &License{
		Tier:       TierFree,
		CustomerID: "free",
		ExpiresAt:  time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC),
		IssuedAt:   time.Now(),
		Limits:     GetLimits(TierFree),
	}
}
