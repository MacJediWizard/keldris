package migration

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMigrationVersionConstant(t *testing.T) {
	if MigrationVersion == "" {
		t.Error("expected non-empty MigrationVersion")
	}
}

func TestMigrationExport_JSONRoundtrip(t *testing.T) {
	original := MigrationExport{
		Metadata: MigrationMetadata{
			Version:    MigrationVersion,
			ExportedAt: time.Now().UTC().Truncate(time.Second),
			Encrypted:  false,
		},
		Organizations: []OrganizationExport{},
		Users:         []UserExport{},
		Agents:        []AgentExport{},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded MigrationExport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Metadata.Version != MigrationVersion {
		t.Errorf("expected version %s, got %s", MigrationVersion, decoded.Metadata.Version)
	}
}

func TestMigrationMetadata_RequiredFields(t *testing.T) {
	meta := MigrationMetadata{
		Version:        MigrationVersion,
		ExportedAt:     time.Now(),
		Encrypted:      true,
		SecretsOmitted: false,
	}

	if !meta.Encrypted {
		t.Error("expected Encrypted=true")
	}
	if meta.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestChecksums_NonNegative(t *testing.T) {
	c := Checksums{
		Organizations: 3,
		Users:         5,
		Agents:        10,
		Repositories:  2,
		Schedules:     7,
		Policies:      1,
	}

	if c.Organizations != 3 {
		t.Errorf("Organizations: got %d", c.Organizations)
	}
	if c.Users != 5 {
		t.Errorf("Users: got %d", c.Users)
	}
}
