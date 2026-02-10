package license

import (
	"errors"
	"time"
)

// DefaultTrialDuration is the default trial period (14 days).
const DefaultTrialDuration = 14 * 24 * time.Hour

// Trial represents a time-limited trial of a license tier.
type Trial struct {
	License   *License
	StartedAt time.Time
	Duration  time.Duration
}

// StartTrial creates a new trial for the given tier and customer.
func StartTrial(tier LicenseTier, customerID string) (*Trial, error) {
	if !tier.IsValid() {
		return nil, errors.New("invalid license tier")
	}
	if customerID == "" {
		return nil, errors.New("missing customer ID")
	}
	if tier == TierFree {
		return nil, errors.New("cannot start trial for free tier")
	}

	now := time.Now()
	return &Trial{
		License: &License{
			Tier:       tier,
			CustomerID: customerID,
			ExpiresAt:  now.Add(DefaultTrialDuration),
			IssuedAt:   now,
			Limits:     GetLimits(tier),
		},
		StartedAt: now,
		Duration:  DefaultTrialDuration,
	}, nil
}

// IsActive returns true if the trial has not yet expired.
func (t *Trial) IsActive() bool {
	if t == nil || t.License == nil {
		return false
	}
	return time.Now().Before(t.License.ExpiresAt)
}

// IsExpired returns true if the trial period has ended.
func (t *Trial) IsExpired() bool {
	return !t.IsActive()
}

// Convert creates a paid license from a trial with the specified tier.
func (t *Trial) Convert(tier LicenseTier) (*License, error) {
	if t == nil || t.License == nil {
		return nil, errors.New("nil trial")
	}
	if !tier.IsValid() {
		return nil, errors.New("invalid license tier")
	}
	if tier == TierFree {
		return nil, errors.New("cannot convert trial to free tier")
	}

	now := time.Now()
	return &License{
		Tier:       tier,
		CustomerID: t.License.CustomerID,
		ExpiresAt:  now.Add(365 * 24 * time.Hour),
		IssuedAt:   now,
		Limits:     GetLimits(tier),
	}, nil
}
