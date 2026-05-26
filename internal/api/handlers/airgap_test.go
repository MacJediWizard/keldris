package handlers

import "testing"

func TestAirGap(t *testing.T) {
	t.Skip("airgap handler depends on concrete license.AirGapManager + filesystem (update packages, docs); covered by license_airgap_test.go in internal/license")
}
