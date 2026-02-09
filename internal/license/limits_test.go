package license

import (
	"testing"
)

func TestGetLimits(t *testing.T) {
	t.Run("free tier limits", func(t *testing.T) {
		limits := GetLimits(TierFree)

		if limits.MaxAgents != 3 {
			t.Errorf("MaxAgents = %d, want 3", limits.MaxAgents)
		}
		if limits.MaxUsers != 3 {
			t.Errorf("MaxUsers = %d, want 3", limits.MaxUsers)
		}
		if limits.MaxOrgs != 1 {
			t.Errorf("MaxOrgs = %d, want 1", limits.MaxOrgs)
		}
		if limits.MaxStorage != 10*1024*1024*1024 {
			t.Errorf("MaxStorage = %d, want %d", limits.MaxStorage, int64(10*1024*1024*1024))
		}
	})

	t.Run("pro tier limits", func(t *testing.T) {
		limits := GetLimits(TierPro)

		if limits.MaxAgents != 25 {
			t.Errorf("MaxAgents = %d, want 25", limits.MaxAgents)
		}
		if limits.MaxUsers != 10 {
			t.Errorf("MaxUsers = %d, want 10", limits.MaxUsers)
		}
		if limits.MaxOrgs != 3 {
			t.Errorf("MaxOrgs = %d, want 3", limits.MaxOrgs)
		}
		if limits.MaxStorage != 100*1024*1024*1024 {
			t.Errorf("MaxStorage = %d, want %d", limits.MaxStorage, int64(100*1024*1024*1024))
		}
	})

	t.Run("enterprise tier limits", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)

		if limits.MaxAgents != -1 {
			t.Errorf("MaxAgents = %d, want -1 (unlimited)", limits.MaxAgents)
		}
		if limits.MaxUsers != -1 {
			t.Errorf("MaxUsers = %d, want -1 (unlimited)", limits.MaxUsers)
		}
		if limits.MaxOrgs != -1 {
			t.Errorf("MaxOrgs = %d, want -1 (unlimited)", limits.MaxOrgs)
		}
		if limits.MaxStorage != -1 {
			t.Errorf("MaxStorage = %d, want -1 (unlimited)", limits.MaxStorage)
		}
	})

	t.Run("unknown tier returns free limits", func(t *testing.T) {
		limits := GetLimits(LicenseTier("unknown"))

		if limits.MaxAgents != 3 {
			t.Errorf("MaxAgents = %d, want 3 (free tier default)", limits.MaxAgents)
		}
		if limits.MaxUsers != 3 {
			t.Errorf("MaxUsers = %d, want 3 (free tier default)", limits.MaxUsers)
		}
	})
}

func TestIsUnlimited(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		unlimited bool
	}{
		{"negative one is unlimited", -1, true},
		{"zero is limited", 0, false},
		{"positive number is limited", 10, false},
		{"negative two is limited", -2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnlimited(tt.limit); got != tt.unlimited {
				t.Errorf("IsUnlimited(%d) = %v, want %v", tt.limit, got, tt.unlimited)
			}
		})
	}
}

func TestIsStorageUnlimited(t *testing.T) {
	tests := []struct {
		name      string
		limit     int64
		unlimited bool
	}{
		{"negative one is unlimited", -1, true},
		{"zero is limited", 0, false},
		{"positive number is limited", 10 * 1024 * 1024 * 1024, false},
		{"negative two is limited", -2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStorageUnlimited(tt.limit); got != tt.unlimited {
				t.Errorf("IsStorageUnlimited(%d) = %v, want %v", tt.limit, got, tt.unlimited)
			}
		})
	}
}

func TestUnlimitedConstant(t *testing.T) {
	if Unlimited != -1 {
		t.Errorf("Unlimited constant = %d, want -1", Unlimited)
	}
}
