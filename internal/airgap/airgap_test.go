package airgap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAirGapMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{name: "true", envValue: "true", expected: true},
		{name: "TRUE", envValue: "TRUE", expected: true},
		{name: "1", envValue: "1", expected: true},
		{name: "yes", envValue: "yes", expected: true},
		{name: "false", envValue: "false", expected: false},
		{name: "0", envValue: "0", expected: false},
		{name: "no", envValue: "no", expected: false},
		{name: "empty", envValue: "", expected: false},
		{name: "invalid", envValue: "maybe", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AIR_GAP_MODE", tt.envValue)
			defer os.Unsetenv("AIR_GAP_MODE")
			assert.Equal(t, tt.expected, IsAirGapMode())
		})
	}
}

func TestDisabledFeatures(t *testing.T) {
	features := DisabledFeatures()
	assert.NotEmpty(t, features)

	names := make(map[string]bool)
	for _, f := range features {
		assert.NotEmpty(t, f.Name)
		assert.NotEmpty(t, f.Reason)
		names[f.Name] = true
	}

	assert.True(t, names["auto_update"])
	assert.True(t, names["external_webhooks"])
	assert.True(t, names["telemetry"])
}

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAirGapMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{name: "true", envValue: "true", expected: true},
		{name: "TRUE", envValue: "TRUE", expected: true},
		{name: "1", envValue: "1", expected: true},
		{name: "yes", envValue: "yes", expected: true},
		{name: "false", envValue: "false", expected: false},
		{name: "0", envValue: "0", expected: false},
		{name: "no", envValue: "no", expected: false},
		{name: "empty", envValue: "", expected: false},
		{name: "invalid", envValue: "maybe", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AIR_GAP_MODE", tt.envValue)
			defer os.Unsetenv("AIR_GAP_MODE")
			assert.Equal(t, tt.expected, IsAirGapMode())
		})
	}
}

func TestDisabledFeatures(t *testing.T) {
	features := DisabledFeatures()
	assert.NotEmpty(t, features)

	names := make(map[string]bool)
	for _, f := range features {
		assert.NotEmpty(t, f.Name)
		assert.NotEmpty(t, f.Reason)
		names[f.Name] = true
	}

	assert.True(t, names["auto_update"])
	assert.True(t, names["external_webhooks"])
	assert.True(t, names["telemetry"])
}

func TestIsFeatureDisabled(t *testing.T) {
	// Not in air-gap mode
	os.Unsetenv("AIR_GAP_MODE")
	assert.False(t, IsFeatureDisabled("auto_update"))

	// In air-gap mode
	os.Setenv("AIR_GAP_MODE", "true")
	defer os.Unsetenv("AIR_GAP_MODE")

	assert.True(t, IsFeatureDisabled("auto_update"))
	assert.True(t, IsFeatureDisabled("external_webhooks"))
	assert.False(t, IsFeatureDisabled("nonexistent_feature"))
}
