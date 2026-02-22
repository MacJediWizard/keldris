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

func TestUnlimitedConstant(t *testing.T) {
	if Unlimited != -1 {
		t.Errorf("Unlimited constant = %d, want -1", Unlimited)
	}
}

func TestLimits_CheckAgentLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 3 agents", TierFree, 3},
		{"pro tier allows 25 agents", TierPro, 25},
		{"enterprise tier has unlimited agents", TierEnterprise, Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := GetLimits(tt.tier)
			if limits.MaxAgents != tt.wantLimit {
				t.Errorf("MaxAgents = %d, want %d", limits.MaxAgents, tt.wantLimit)
			}
		})
	}

	t.Run("enterprise agents are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxAgents) {
			t.Error("enterprise MaxAgents should be unlimited")
		}
	})

	t.Run("free agents are not unlimited", func(t *testing.T) {
		limits := GetLimits(TierFree)
		if IsUnlimited(limits.MaxAgents) {
			t.Error("free MaxAgents should not be unlimited")
		}
	})
}

func TestLimits_CheckUserLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 3 users", TierFree, 3},
		{"pro tier allows 10 users", TierPro, 10},
		{"enterprise tier has unlimited users", TierEnterprise, Unlimited},
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
			limits := GetLimits(tt.tier)
			if limits.MaxUsers != tt.wantLimit {
				t.Errorf("MaxUsers = %d, want %d", limits.MaxUsers, tt.wantLimit)
			}
		})
	}

	t.Run("enterprise users are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxUsers) {
			t.Error("enterprise MaxUsers should be unlimited")
		}
	})

	t.Run("pro users are not unlimited", func(t *testing.T) {
		limits := GetLimits(TierPro)
		if IsUnlimited(limits.MaxUsers) {
			t.Error("pro MaxUsers should not be unlimited")
		}
	})
}

func TestLimits_CheckOrgLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 1 org", TierFree, 1},
		{"pro tier allows 3 orgs", TierPro, 3},
		{"enterprise tier has unlimited orgs", TierEnterprise, Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := GetLimits(tt.tier)
			if limits.MaxOrgs != tt.wantLimit {
				t.Errorf("MaxOrgs = %d, want %d", limits.MaxOrgs, tt.wantLimit)
			}
		})
	}

	t.Run("free tier is restricted to single org", func(t *testing.T) {
		limits := GetLimits(TierFree)
		if limits.MaxOrgs != 1 {
			t.Errorf("free MaxOrgs = %d, want 1", limits.MaxOrgs)
		}
	})

	t.Run("enterprise orgs are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxOrgs) {
			t.Error("enterprise MaxOrgs should be unlimited")
		}
	})
}

func TestLimits_Exceeded(t *testing.T) {
	t.Run("free tier agent limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 4
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			// Limit is exceeded - this is expected
		} else {
			t.Error("4 agents should exceed free tier limit of 3")
		}
	})

	t.Run("free tier agent limit not exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 2
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			t.Error("2 agents should not exceed free tier limit of 3")
		}
	})

	t.Run("pro tier user limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierPro)
		currentUsers := 11
		if !IsUnlimited(limits.MaxUsers) && currentUsers > limits.MaxUsers {
			// Limit is exceeded - this is expected
		} else {
			t.Error("11 users should exceed pro tier limit of 10")
		}
	})

	t.Run("enterprise tier never exceeded", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		exceeded := !IsUnlimited(limits.MaxAgents) && 1000 > limits.MaxAgents
		if exceeded {
			t.Error("enterprise tier should never be exceeded for agents")
		}
		exceeded = !IsUnlimited(limits.MaxUsers) && 1000 > limits.MaxUsers
		if exceeded {
			t.Error("enterprise tier should never be exceeded for users")
		}
		exceeded = !IsUnlimited(limits.MaxOrgs) && 1000 > limits.MaxOrgs
		if exceeded {
			t.Error("enterprise tier should never be exceeded for orgs")
		}
	})

	t.Run("free tier at exact limit is not exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 3
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			t.Error("3 agents should not exceed free tier limit of 3 (at limit, not over)")
		}
	})
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

func TestLimits_CheckAgentLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 3 agents", TierFree, 3},
		{"pro tier allows 25 agents", TierPro, 25},
		{"enterprise tier has unlimited agents", TierEnterprise, Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := GetLimits(tt.tier)
			if limits.MaxAgents != tt.wantLimit {
				t.Errorf("MaxAgents = %d, want %d", limits.MaxAgents, tt.wantLimit)
			}
		})
	}

	t.Run("enterprise agents are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxAgents) {
			t.Error("enterprise MaxAgents should be unlimited")
		}
	})

	t.Run("free agents are not unlimited", func(t *testing.T) {
		limits := GetLimits(TierFree)
		if IsUnlimited(limits.MaxAgents) {
			t.Error("free MaxAgents should not be unlimited")
		}
	})
}

func TestLimits_CheckUserLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 3 users", TierFree, 3},
		{"pro tier allows 10 users", TierPro, 10},
		{"enterprise tier has unlimited users", TierEnterprise, Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := GetLimits(tt.tier)
			if limits.MaxUsers != tt.wantLimit {
				t.Errorf("MaxUsers = %d, want %d", limits.MaxUsers, tt.wantLimit)
			}
		})
	}

	t.Run("enterprise users are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxUsers) {
			t.Error("enterprise MaxUsers should be unlimited")
		}
	})

	t.Run("pro users are not unlimited", func(t *testing.T) {
		limits := GetLimits(TierPro)
		if IsUnlimited(limits.MaxUsers) {
			t.Error("pro MaxUsers should not be unlimited")
		}
	})
}

func TestLimits_CheckOrgLimit(t *testing.T) {
	tests := []struct {
		name      string
		tier      LicenseTier
		wantLimit int
	}{
		{"free tier allows 1 org", TierFree, 1},
		{"pro tier allows 3 orgs", TierPro, 3},
		{"enterprise tier has unlimited orgs", TierEnterprise, Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := GetLimits(tt.tier)
			if limits.MaxOrgs != tt.wantLimit {
				t.Errorf("MaxOrgs = %d, want %d", limits.MaxOrgs, tt.wantLimit)
			}
		})
	}

	t.Run("free tier is restricted to single org", func(t *testing.T) {
		limits := GetLimits(TierFree)
		if limits.MaxOrgs != 1 {
			t.Errorf("free MaxOrgs = %d, want 1", limits.MaxOrgs)
		}
	})

	t.Run("enterprise orgs are unlimited", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		if !IsUnlimited(limits.MaxOrgs) {
			t.Error("enterprise MaxOrgs should be unlimited")
		}
	})
}

func TestLimits_Exceeded(t *testing.T) {
	t.Run("free tier agent limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 4
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			// Limit is exceeded - this is expected
		} else {
			t.Error("4 agents should exceed free tier limit of 3")
		}
	})

	t.Run("free tier agent limit not exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 2
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			t.Error("2 agents should not exceed free tier limit of 3")
		}
	})

	t.Run("pro tier user limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierPro)
		currentUsers := 11
		if !IsUnlimited(limits.MaxUsers) && currentUsers > limits.MaxUsers {
			// Limit is exceeded - this is expected
		} else {
			t.Error("11 users should exceed pro tier limit of 10")
		}
	})

	t.Run("enterprise tier never exceeded", func(t *testing.T) {
		limits := GetLimits(TierEnterprise)
		exceeded := !IsUnlimited(limits.MaxAgents) && 1000 > limits.MaxAgents
		if exceeded {
			t.Error("enterprise tier should never be exceeded for agents")
		}
		exceeded = !IsUnlimited(limits.MaxUsers) && 1000 > limits.MaxUsers
		if exceeded {
			t.Error("enterprise tier should never be exceeded for users")
		}
		exceeded = !IsUnlimited(limits.MaxOrgs) && 1000 > limits.MaxOrgs
		if exceeded {
			t.Error("enterprise tier should never be exceeded for orgs")
		}
		exceeded = !IsStorageUnlimited(limits.MaxStorage) && int64(1000*1024*1024*1024) > limits.MaxStorage
		if exceeded {
			t.Error("enterprise tier should never be exceeded for storage")
		}
	})

	t.Run("free tier storage limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentStorage := int64(11 * 1024 * 1024 * 1024) // 11 GB
		if !IsStorageUnlimited(limits.MaxStorage) && currentStorage > limits.MaxStorage {
			// Limit is exceeded - this is expected
		} else {
			t.Error("11 GB should exceed free tier storage limit of 10 GB")
		}
	})

	t.Run("free tier at exact limit is not exceeded", func(t *testing.T) {
		limits := GetLimits(TierFree)
		currentAgents := 3
		if !IsUnlimited(limits.MaxAgents) && currentAgents > limits.MaxAgents {
			t.Error("3 agents should not exceed free tier limit of 3 (at limit, not over)")
		}
	})

	t.Run("pro tier storage limit exceeded", func(t *testing.T) {
		limits := GetLimits(TierPro)
		currentStorage := int64(101 * 1024 * 1024 * 1024) // 101 GB
		if !IsStorageUnlimited(limits.MaxStorage) && currentStorage > limits.MaxStorage {
			// Limit is exceeded - this is expected
		} else {
			t.Error("101 GB should exceed pro tier storage limit of 100 GB")
		}
	})
}
