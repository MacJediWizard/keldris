// Package airgap provides air-gapped deployment support for Keldris.
// Air-gap mode disables features that require internet access (webhooks, updates, telemetry).
// License verification in air-gap mode uses local Ed25519 key validation.
package airgap

import (
	"os"
	"strings"

	"github.com/MacJediWizard/keldris/internal/license"
)

// DisabledFeature describes a feature that is disabled in air-gap mode.
type DisabledFeature struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// IsAirGapMode returns true if the server is running in air-gap mode.
func IsAirGapMode() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("AIR_GAP_MODE")))
	return val == "true" || val == "1" || val == "yes"
}

// DisabledFeatures returns the list of features disabled in air-gap mode.
func DisabledFeatures() []DisabledFeature {
	return []DisabledFeature{
		{Name: "auto_update", Reason: "Automatic updates require internet access"},
		{Name: "external_webhooks", Reason: "External webhooks require internet access"},
		{Name: "telemetry", Reason: "Telemetry reporting requires internet access"},
		{Name: "cloud_storage_validation", Reason: "Cloud storage endpoint validation requires internet access"},
	}
}

// IsFeatureDisabled returns true if the given feature name is disabled in air-gap mode.
func IsFeatureDisabled(feature string) bool {
	if !IsAirGapMode() {
		return false
	}
	for _, f := range DisabledFeatures() {
		if f.Name == feature {
			return true
		}
	}
	return false
}

// ValidateAirGapLicense validates an offline license for air-gap deployments.
// It delegates to the license package's offline validation using Ed25519.
func ValidateAirGapLicense(licenseData []byte, publicKey []byte) (*license.License, error) {
	return license.ValidateOfflineLicense(licenseData, publicKey)
}
