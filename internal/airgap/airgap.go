// Package airgap provides air-gapped deployment support for Keldris.
// Air-gap mode disables features that require internet access (webhooks, updates, telemetry).
// License verification in air-gap mode uses local Ed25519 key validation.
package airgap

import (
	"os"
	"strings"
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

