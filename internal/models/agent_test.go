package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAgent(t *testing.T) {
	orgID := uuid.New()
	hostname := "test-host"
	apiKeyHash := "hash123"

	agent := NewAgent(orgID, hostname, apiKeyHash)

	if agent.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if agent.OrgID != orgID {
		t.Errorf("expected OrgID %v, got %v", orgID, agent.OrgID)
	}
	if agent.Hostname != hostname {
		t.Errorf("expected Hostname %s, got %s", hostname, agent.Hostname)
	}
	if agent.APIKeyHash != apiKeyHash {
		t.Errorf("expected APIKeyHash %s, got %s", apiKeyHash, agent.APIKeyHash)
	}
	if agent.Status != AgentStatusPending {
		t.Errorf("expected Status %s, got %s", AgentStatusPending, agent.Status)
	}
	if agent.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if agent.LastSeen != nil {
		t.Error("expected LastSeen to be nil")
	}
}

func TestAgent_SetOSInfo_OSInfoJSON(t *testing.T) {
	agent := NewAgent(uuid.New(), "test-host", "hash123")

	osInfo := OSInfo{
		OS:       "linux",
		Arch:     "amd64",
		Hostname: "test-host",
		Version:  "22.04",
	}

	data, err := json.Marshal(osInfo)
	if err != nil {
		t.Fatalf("failed to marshal OSInfo: %v", err)
	}

	t.Run("round trip", func(t *testing.T) {
		if err := agent.SetOSInfo(data); err != nil {
			t.Fatalf("SetOSInfo failed: %v", err)
		}

		retrieved, err := agent.OSInfoJSON()
		if err != nil {
			t.Fatalf("OSInfoJSON failed: %v", err)
		}

		var got OSInfo
		if err := json.Unmarshal(retrieved, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got.OS != osInfo.OS || got.Arch != osInfo.Arch {
			t.Errorf("round trip mismatch: got %+v, want %+v", got, osInfo)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		if err := a.SetOSInfo(nil); err != nil {
			t.Errorf("SetOSInfo(nil) should not error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		if err := a.SetOSInfo([]byte("invalid")); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("nil OSInfo", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		data, err := a.OSInfoJSON()
		if err != nil {
			t.Errorf("OSInfoJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})
}

func TestAgent_IsOnline(t *testing.T) {
	tests := []struct {
		name      string
		lastSeen  *time.Time
		threshold time.Duration
		expected  bool
	}{
		{"nil last seen", nil, 5 * time.Minute, false},
		{"within threshold", timePtr(time.Now().Add(-2 * time.Minute)), 5 * time.Minute, true},
		{"outside threshold", timePtr(time.Now().Add(-10 * time.Minute)), 5 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(uuid.New(), "test", "hash")
			agent.LastSeen = tt.lastSeen

			if got := agent.IsOnline(tt.threshold); got != tt.expected {
				t.Errorf("IsOnline() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgent_MarkSeen(t *testing.T) {
	agent := NewAgent(uuid.New(), "test-host", "hash123")

	if agent.LastSeen != nil {
		t.Error("expected LastSeen to be nil initially")
	}

	before := time.Now()
	agent.MarkSeen()
	after := time.Now()

	if agent.LastSeen == nil {
		t.Fatal("expected LastSeen to be set")
	}
	if agent.LastSeen.Before(before) || agent.LastSeen.After(after) {
		t.Errorf("LastSeen %v not within expected range", agent.LastSeen)
	}
	if agent.Status != AgentStatusActive {
		t.Errorf("expected Status %s, got %s", AgentStatusActive, agent.Status)
	}
}

func TestAgent_NetworkMounts(t *testing.T) {
	agent := NewAgent(uuid.New(), "test-host", "hash123")

	mounts := []NetworkMount{
		{Path: "/mnt/share1", Type: MountTypeNFS, Remote: "192.168.1.100:/share", Status: MountStatusConnected, LastChecked: time.Now()},
		{Path: "/mnt/share2", Type: MountTypeSMB, Remote: "//server/share", Status: MountStatusDisconnected, LastChecked: time.Now()},
	}

	data, _ := json.Marshal(mounts)
	if err := agent.SetNetworkMounts(data); err != nil {
		t.Fatalf("SetNetworkMounts failed: %v", err)
	}

	t.Run("round trip", func(t *testing.T) {
		retrieved, err := agent.NetworkMountsJSON()
		if err != nil {
			t.Fatalf("NetworkMountsJSON failed: %v", err)
		}
		var got []NetworkMount
		if err := json.Unmarshal(retrieved, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 mounts, got %d", len(got))
		}
	})

	t.Run("nil mounts returns empty array", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		data, err := a.NetworkMountsJSON()
		if err != nil {
			t.Errorf("NetworkMountsJSON failed: %v", err)
		}
		if string(data) != "[]" {
			t.Errorf("expected [], got %s", string(data))
		}
	})
}


func TestAgent_HealthMetrics(t *testing.T) {
	agent := NewAgent(uuid.New(), "test-host", "hash123")

	metrics := HealthMetrics{
		CPUUsage:        45.5,
		MemoryUsage:     60.2,
		DiskUsage:       75.8,
		DiskFreeBytes:   100000000000,
		DiskTotalBytes:  500000000000,
		NetworkUp:       true,
		UptimeSeconds:   86400,
		ResticVersion:   "0.16.0",
		ResticAvailable: true,
	}

	data, _ := json.Marshal(metrics)

	t.Run("round trip", func(t *testing.T) {
		if err := agent.SetHealthMetrics(data); err != nil {
			t.Fatalf("SetHealthMetrics failed: %v", err)
		}

		retrieved, err := agent.HealthMetricsJSON()
		if err != nil {
			t.Fatalf("HealthMetricsJSON failed: %v", err)
		}

		var got HealthMetrics
		if err := json.Unmarshal(retrieved, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got.CPUUsage != metrics.CPUUsage {
			t.Errorf("expected CPUUsage %f, got %f", metrics.CPUUsage, got.CPUUsage)
		}
	})

	t.Run("nil metrics", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		data, err := a.HealthMetricsJSON()
		if err != nil {
			t.Errorf("HealthMetricsJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		a := NewAgent(uuid.New(), "test", "hash")
		if err := a.SetHealthMetrics([]byte("invalid")); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}
