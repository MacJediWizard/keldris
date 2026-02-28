package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAgent_JSONMarshal(t *testing.T) {
	agent := NewAgent(uuid.New(), "test-host", "secret-hash")

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal Agent: %v", err)
	}

	// APIKeyHash should not appear in JSON (json:"-")
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := raw["api_key_hash"]; exists {
		t.Error("APIKeyHash should not be in JSON output")
	}

	if raw["hostname"] != "test-host" {
		t.Errorf("expected hostname 'test-host', got %v", raw["hostname"])
	}

	if raw["status"] != string(AgentStatusPending) {
		t.Errorf("expected status 'pending', got %v", raw["status"])
	}

	// last_seen should be null
	if raw["last_seen"] != nil {
		t.Errorf("expected last_seen to be null, got %v", raw["last_seen"])
	}
}

func TestAgent_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"org_id": "660e8400-e29b-41d4-a716-446655440000",
		"hostname": "prod-server",
		"status": "active",
		"last_seen": "2024-01-15T10:30:00Z",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-15T10:30:00Z"
	}`

	var agent Agent
	if err := json.Unmarshal([]byte(jsonStr), &agent); err != nil {
		t.Fatalf("failed to unmarshal Agent: %v", err)
	}

	if agent.Hostname != "prod-server" {
		t.Errorf("expected hostname 'prod-server', got %s", agent.Hostname)
	}
	if agent.Status != AgentStatusActive {
		t.Errorf("expected status active, got %s", agent.Status)
	}
	if agent.LastSeen == nil {
		t.Fatal("expected last_seen to be set")
	}
}

func TestRepository_JSONSensitiveFields(t *testing.T) {
	repo := NewRepository(uuid.New(), "my-repo", RepositoryTypeS3, []byte("secret-config"))

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("failed to marshal Repository: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, exists := raw["config_encrypted"]; exists {
		t.Error("ConfigEncrypted should not be in JSON output")
	}

	if raw["name"] != "my-repo" {
		t.Errorf("expected name 'my-repo', got %v", raw["name"])
	}

	if raw["type"] != string(RepositoryTypeS3) {
		t.Errorf("expected type 's3', got %v", raw["type"])
	}
}

func TestRepositoryKey_JSONSensitiveFields(t *testing.T) {
	rk := NewRepositoryKey(uuid.New(), []byte("encrypted-key"), true, []byte("escrow-key"))

	data, err := json.Marshal(rk)
	if err != nil {
		t.Fatalf("failed to marshal RepositoryKey: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, exists := raw["encrypted_key"]; exists {
		t.Error("EncryptedKey should not be in JSON output")
	}
	if _, exists := raw["escrow_encrypted_key"]; exists {
		t.Error("EscrowEncryptedKey should not be in JSON output")
	}
	if raw["escrow_enabled"] != true {
		t.Errorf("expected escrow_enabled to be true, got %v", raw["escrow_enabled"])
	}
}

func TestNotificationChannel_JSONSensitiveFields(t *testing.T) {
	ch := NewNotificationChannel(uuid.New(), "email-alerts", ChannelTypeEmail, []byte("smtp-config"))

	data, err := json.Marshal(ch)
	if err != nil {
		t.Fatalf("failed to marshal NotificationChannel: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, exists := raw["config_encrypted"]; exists {
		t.Error("ConfigEncrypted should not be in JSON output")
	}
	if raw["name"] != "email-alerts" {
		t.Errorf("expected name 'email-alerts', got %v", raw["name"])
	}
	if raw["enabled"] != true {
		t.Errorf("expected enabled true, got %v", raw["enabled"])
	}
}

func TestOrgInvitation_JSONSensitiveFields(t *testing.T) {
	invite := NewOrgInvitation(uuid.New(), "test@example.com", OrgRoleMember, "secret-token", uuid.New(), time.Now().Add(24*time.Hour))

	data, err := json.Marshal(invite)
	if err != nil {
		t.Fatalf("failed to marshal OrgInvitation: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if _, exists := raw["token"]; exists {
		t.Error("Token should not be in JSON output")
	}
	if raw["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", raw["email"])
	}
}

func TestBackup_JSONOmitempty(t *testing.T) {
	repoID := uuid.New()
	backup := NewBackup(uuid.New(), uuid.New(), &repoID)

	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("failed to marshal Backup: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// Running backup should not have completed_at
	if raw["completed_at"] != nil {
		t.Errorf("expected completed_at to be nil for running backup, got %v", raw["completed_at"])
	}

	// Complete the backup and check fields appear
	backup.Complete("snap-1", 1024, 10, 5)

	data, err = json.Marshal(backup)
	if err != nil {
		t.Fatalf("failed to marshal completed Backup: %v", err)
	}

	json.Unmarshal(data, &raw)
	if raw["completed_at"] == nil {
		t.Error("expected completed_at to be set for completed backup")
	}
	if raw["snapshot_id"] != "snap-1" {
		t.Errorf("expected snapshot_id 'snap-1', got %v", raw["snapshot_id"])
	}
}

func TestRestore_JSONOmitempty(t *testing.T) {
	restore := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore/path", []string{"/data"}, nil)

	data, err := json.Marshal(restore)
	if err != nil {
		t.Fatalf("failed to marshal Restore: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// Check omitempty fields
	if raw["started_at"] != nil {
		t.Error("expected started_at to be nil for pending restore")
	}
	if raw["error_message"] != nil {
		t.Errorf("expected error_message to be omitted, got %v", raw["error_message"])
	}

	// include_paths should be present (non-nil slice)
	if raw["include_paths"] == nil {
		t.Error("expected include_paths to be present")
	}
	// exclude_paths should be omitted (nil slice)
	if raw["exclude_paths"] != nil {
		t.Errorf("expected exclude_paths to be omitted, got %v", raw["exclude_paths"])
	}
}

func TestVerification_JSONRoundTrip(t *testing.T) {
	v := NewVerification(uuid.New(), VerificationTypeCheck)

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal Verification: %v", err)
	}

	var v2 Verification
	if err := json.Unmarshal(data, &v2); err != nil {
		t.Fatalf("failed to unmarshal Verification: %v", err)
	}

	if v2.ID != v.ID {
		t.Errorf("expected ID %v, got %v", v.ID, v2.ID)
	}
	if v2.Type != VerificationTypeCheck {
		t.Errorf("expected Type 'check', got %s", v2.Type)
	}
	if v2.Status != VerificationStatusRunning {
		t.Errorf("expected Status 'running', got %s", v2.Status)
	}
}

func TestAlert_JSONRoundTrip(t *testing.T) {
	alert := NewAlert(uuid.New(), AlertTypeAgentOffline, AlertSeverityCritical, "Agent Down", "Server-1 is offline")
	resourceID := uuid.New()
	alert.SetResource(ResourceTypeAgent, resourceID)

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("failed to marshal Alert: %v", err)
	}

	var alert2 Alert
	if err := json.Unmarshal(data, &alert2); err != nil {
		t.Fatalf("failed to unmarshal Alert: %v", err)
	}

	if alert2.Title != "Agent Down" {
		t.Errorf("expected Title 'Agent Down', got %s", alert2.Title)
	}
	if alert2.Severity != AlertSeverityCritical {
		t.Errorf("expected Severity 'critical', got %s", alert2.Severity)
	}
	if alert2.ResourceType == nil || *alert2.ResourceType != ResourceTypeAgent {
		t.Error("expected ResourceType to be agent")
	}
	if alert2.ResourceID == nil || *alert2.ResourceID != resourceID {
		t.Error("expected ResourceID to match")
	}
}

func TestAlert_MetadataJSON(t *testing.T) {
	alert := NewAlert(uuid.New(), AlertTypeStorageUsage, AlertSeverityWarning, "Storage High", "85% used")

	t.Run("nil metadata", func(t *testing.T) {
		data, err := alert.MetadataJSON()
		if err != nil {
			t.Fatalf("MetadataJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("set and retrieve metadata", func(t *testing.T) {
		metaJSON := []byte(`{"usage_percent": 85, "threshold": 80}`)
		if err := alert.SetMetadata(metaJSON); err != nil {
			t.Fatalf("SetMetadata failed: %v", err)
		}

		data, err := alert.MetadataJSON()
		if err != nil {
			t.Fatalf("MetadataJSON failed: %v", err)
		}

		var got map[string]interface{}
		json.Unmarshal(data, &got)
		if got["threshold"] != float64(80) {
			t.Errorf("expected threshold 80, got %v", got["threshold"])
		}
	})

	t.Run("empty data", func(t *testing.T) {
		a := NewAlert(uuid.New(), AlertTypeBackupSLA, AlertSeverityInfo, "test", "test")
		if err := a.SetMetadata(nil); err != nil {
			t.Errorf("SetMetadata(nil) should not error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		a := NewAlert(uuid.New(), AlertTypeBackupSLA, AlertSeverityInfo, "test", "test")
		if err := a.SetMetadata([]byte("invalid")); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestAlertRule_ConfigJSON(t *testing.T) {
	config := AlertRuleConfig{
		OfflineThresholdMinutes: 10,
		MaxHoursSinceBackup:     24,
		StorageUsagePercent:     85,
	}
	rule := NewAlertRule(uuid.New(), "offline-check", AlertTypeAgentOffline, config)

	t.Run("round trip", func(t *testing.T) {
		data, err := rule.ConfigJSON()
		if err != nil {
			t.Fatalf("ConfigJSON failed: %v", err)
		}

		rule2 := &AlertRule{}
		if err := rule2.SetConfig(data); err != nil {
			t.Fatalf("SetConfig failed: %v", err)
		}

		if rule2.Config.OfflineThresholdMinutes != 10 {
			t.Errorf("expected OfflineThresholdMinutes 10, got %d", rule2.Config.OfflineThresholdMinutes)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		r := &AlertRule{}
		if err := r.SetConfig(nil); err != nil {
			t.Errorf("SetConfig(nil) should not error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		r := &AlertRule{}
		if err := r.SetConfig([]byte("invalid")); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestVerification_DetailsJSON(t *testing.T) {
	v := NewVerification(uuid.New(), VerificationTypeTestRestore)

	t.Run("nil details", func(t *testing.T) {
		data, err := v.DetailsJSON()
		if err != nil {
			t.Fatalf("DetailsJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("set and retrieve details", func(t *testing.T) {
		detailsJSON := []byte(`{"files_restored": 100, "bytes_restored": 5242880}`)
		if err := v.SetDetails(detailsJSON); err != nil {
			t.Fatalf("SetDetails failed: %v", err)
		}

		data, err := v.DetailsJSON()
		if err != nil {
			t.Fatalf("DetailsJSON failed: %v", err)
		}

		var got VerificationDetails
		json.Unmarshal(data, &got)
		if got.FilesRestored != 100 {
			t.Errorf("expected FilesRestored 100, got %d", got.FilesRestored)
		}
		if got.BytesRestored != 5242880 {
			t.Errorf("expected BytesRestored 5242880, got %d", got.BytesRestored)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		v2 := NewVerification(uuid.New(), VerificationTypeCheck)
		if err := v2.SetDetails(nil); err != nil {
			t.Errorf("SetDetails(nil) should not error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		v2 := NewVerification(uuid.New(), VerificationTypeCheck)
		if err := v2.SetDetails([]byte("not json")); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestDRRunbook_StepsJSON(t *testing.T) {
	rb := NewDRRunbook(uuid.New(), "test-runbook")
	rb.AddStep("Verify backup", "Check backup exists", DRRunbookStepTypeVerify)
	rb.AddStep("Restore data", "Restore from snapshot", DRRunbookStepTypeRestore)

	t.Run("round trip", func(t *testing.T) {
		data, err := rb.StepsJSON()
		if err != nil {
			t.Fatalf("StepsJSON failed: %v", err)
		}

		rb2 := NewDRRunbook(uuid.New(), "other")
		if err := rb2.SetSteps(data); err != nil {
			t.Fatalf("SetSteps failed: %v", err)
		}

		if len(rb2.Steps) != 2 {
			t.Fatalf("expected 2 steps, got %d", len(rb2.Steps))
		}
		if rb2.Steps[0].Title != "Verify backup" {
			t.Errorf("expected step 0 title 'Verify backup', got %s", rb2.Steps[0].Title)
		}
		if rb2.Steps[1].Order != 2 {
			t.Errorf("expected step 1 order 2, got %d", rb2.Steps[1].Order)
		}
	})

	t.Run("nil steps", func(t *testing.T) {
		rb2 := &DRRunbook{}
		data, err := rb2.StepsJSON()
		if err != nil {
			t.Fatalf("StepsJSON failed: %v", err)
		}
		if string(data) != "[]" {
			t.Errorf("expected [], got %s", string(data))
		}
	})

	t.Run("empty data resets to empty", func(t *testing.T) {
		rb2 := NewDRRunbook(uuid.New(), "test")
		rb2.AddStep("step", "desc", DRRunbookStepTypeManual)
		if err := rb2.SetSteps(nil); err != nil {
			t.Fatalf("SetSteps(nil) failed: %v", err)
		}
		if len(rb2.Steps) != 0 {
			t.Errorf("expected 0 steps, got %d", len(rb2.Steps))
		}
	})
}

func TestDRRunbook_ContactsJSON(t *testing.T) {
	rb := NewDRRunbook(uuid.New(), "test-runbook")
	rb.AddContact("John Doe", "DBA", "john@example.com", "+1234567890", true)

	t.Run("round trip", func(t *testing.T) {
		data, err := rb.ContactsJSON()
		if err != nil {
			t.Fatalf("ContactsJSON failed: %v", err)
		}

		rb2 := NewDRRunbook(uuid.New(), "other")
		if err := rb2.SetContacts(data); err != nil {
			t.Fatalf("SetContacts failed: %v", err)
		}

		if len(rb2.Contacts) != 1 {
			t.Fatalf("expected 1 contact, got %d", len(rb2.Contacts))
		}
		if rb2.Contacts[0].Name != "John Doe" {
			t.Errorf("expected Name 'John Doe', got %s", rb2.Contacts[0].Name)
		}
		if !rb2.Contacts[0].Notify {
			t.Error("expected Notify to be true")
		}
	})

	t.Run("nil contacts", func(t *testing.T) {
		rb2 := &DRRunbook{}
		data, err := rb2.ContactsJSON()
		if err != nil {
			t.Fatalf("ContactsJSON failed: %v", err)
		}
		if string(data) != "[]" {
			t.Errorf("expected [], got %s", string(data))
		}
	})

	t.Run("empty data resets to empty", func(t *testing.T) {
		rb2 := NewDRRunbook(uuid.New(), "test")
		rb2.AddContact("Test", "role", "", "", false)
		if err := rb2.SetContacts(nil); err != nil {
			t.Fatalf("SetContacts(nil) failed: %v", err)
		}
		if len(rb2.Contacts) != 0 {
			t.Errorf("expected 0 contacts, got %d", len(rb2.Contacts))
		}
	})
}

func TestPolicy_JSONSerialization(t *testing.T) {
	policy := NewPolicy(uuid.New(), "standard-policy")
	policy.Paths = []string{"/data", "/home"}
	policy.Excludes = []string{"*.tmp"}
	policy.RetentionPolicy = &RetentionPolicy{KeepLast: 5, KeepDaily: 7}
	bw := 1024
	policy.BandwidthLimitKB = &bw
	policy.ExcludedHours = []int{9, 10, 11}

	t.Run("paths round trip", func(t *testing.T) {
		data, err := policy.PathsJSON()
		if err != nil {
			t.Fatalf("PathsJSON failed: %v", err)
		}
		p2 := NewPolicy(uuid.New(), "test")
		if err := p2.SetPaths(data); err != nil {
			t.Fatalf("SetPaths failed: %v", err)
		}
		if len(p2.Paths) != 2 {
			t.Errorf("expected 2 paths, got %d", len(p2.Paths))
		}
	})

	t.Run("nil paths", func(t *testing.T) {
		p := NewPolicy(uuid.New(), "test")
		data, err := p.PathsJSON()
		if err != nil {
			t.Fatalf("PathsJSON failed: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %v", data)
		}
	})

	t.Run("excludes round trip", func(t *testing.T) {
		data, err := policy.ExcludesJSON()
		if err != nil {
			t.Fatalf("ExcludesJSON failed: %v", err)
		}
		p2 := NewPolicy(uuid.New(), "test")
		if err := p2.SetExcludes(data); err != nil {
			t.Fatalf("SetExcludes failed: %v", err)
		}
		if len(p2.Excludes) != 1 {
			t.Errorf("expected 1 exclude, got %d", len(p2.Excludes))
		}
	})

	t.Run("retention round trip", func(t *testing.T) {
		data, err := policy.RetentionPolicyJSON()
		if err != nil {
			t.Fatalf("RetentionPolicyJSON failed: %v", err)
		}
		p2 := NewPolicy(uuid.New(), "test")
		if err := p2.SetRetentionPolicy(data); err != nil {
			t.Fatalf("SetRetentionPolicy failed: %v", err)
		}
		if p2.RetentionPolicy == nil || p2.RetentionPolicy.KeepLast != 5 {
			t.Error("expected retention policy with KeepLast 5")
		}
	})

	t.Run("excluded hours round trip", func(t *testing.T) {
		data, err := policy.ExcludedHoursJSON()
		if err != nil {
			t.Fatalf("ExcludedHoursJSON failed: %v", err)
		}
		p2 := NewPolicy(uuid.New(), "test")
		if err := p2.SetExcludedHours(data); err != nil {
			t.Fatalf("SetExcludedHours failed: %v", err)
		}
		if len(p2.ExcludedHours) != 3 {
			t.Errorf("expected 3 excluded hours, got %d", len(p2.ExcludedHours))
		}
	})

	t.Run("backup window", func(t *testing.T) {
		start := "02:00"
		end := "06:00"
		p := NewPolicy(uuid.New(), "test")
		p.SetBackupWindow(&start, &end)
		if p.BackupWindow == nil {
			t.Fatal("expected BackupWindow to be set")
		}
		if p.BackupWindow.Start != "02:00" || p.BackupWindow.End != "06:00" {
			t.Errorf("unexpected window: %+v", p.BackupWindow)
		}
	})

	t.Run("backup window nil", func(t *testing.T) {
		p := NewPolicy(uuid.New(), "test")
		p.BackupWindow = &BackupWindow{Start: "01:00", End: "05:00"}
		p.SetBackupWindow(nil, nil)
		if p.BackupWindow != nil {
			t.Error("expected BackupWindow to be nil after SetBackupWindow(nil, nil)")
		}
	})

	t.Run("empty data for set methods", func(t *testing.T) {
		p := NewPolicy(uuid.New(), "test")
		if err := p.SetPaths(nil); err != nil {
			t.Errorf("SetPaths(nil) error: %v", err)
		}
		if err := p.SetExcludes(nil); err != nil {
			t.Errorf("SetExcludes(nil) error: %v", err)
		}
		if err := p.SetRetentionPolicy(nil); err != nil {
			t.Errorf("SetRetentionPolicy(nil) error: %v", err)
		}
		if err := p.SetExcludedHours(nil); err != nil {
			t.Errorf("SetExcludedHours(nil) error: %v", err)
		}
	})
}

func TestNotificationLog_JSONRoundTrip(t *testing.T) {
	channelID := uuid.New()
	log := NewNotificationLog(uuid.New(), &channelID, "backup_success", "admin@example.com", "Backup Complete")

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal NotificationLog: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if raw["status"] != string(NotificationStatusQueued) {
		t.Errorf("expected status 'queued', got %v", raw["status"])
	}
	if raw["sent_at"] != nil {
		t.Error("expected sent_at to be omitted for queued notification")
	}

	// Mark sent and verify
	log.MarkSent()
	data, _ = json.Marshal(log)
	json.Unmarshal(data, &raw)

	if raw["status"] != string(NotificationStatusSent) {
		t.Errorf("expected status 'sent', got %v", raw["status"])
	}
	if raw["sent_at"] == nil {
		t.Error("expected sent_at to be set after MarkSent")
	}
}

func TestNotificationLog_NilChannelID(t *testing.T) {
	log := NewNotificationLog(uuid.New(), nil, "agent_offline", "ops@example.com", "Agent Offline")

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// channel_id should be omitted when nil
	if raw["channel_id"] != nil {
		t.Errorf("expected channel_id to be nil, got %v", raw["channel_id"])
	}
}

func TestOnboardingProgress_JSONRoundTrip(t *testing.T) {
	progress := NewOnboardingProgress(uuid.New())
	progress.CompleteStep(OnboardingStepWelcome)
	progress.CompleteStep(OnboardingStepOrganization)

	data, err := json.Marshal(progress)
	if err != nil {
		t.Fatalf("failed to marshal OnboardingProgress: %v", err)
	}

	var p2 OnboardingProgress
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("failed to unmarshal OnboardingProgress: %v", err)
	}

	if len(p2.CompletedSteps) != 2 {
		t.Errorf("expected 2 completed steps, got %d", len(p2.CompletedSteps))
	}
	if p2.CurrentStep != OnboardingStepOIDC {
		t.Errorf("expected current step 'oidc', got %s", p2.CurrentStep)
	}
}

func TestUUID_JSONRoundTrip(t *testing.T) {
	originalID := uuid.New()
	agent := NewAgent(originalID, "test-host", "hash")
	agent.ID = originalID

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var agent2 Agent
	if err := json.Unmarshal(data, &agent2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if agent2.ID != originalID {
		t.Errorf("UUID round-trip failed: expected %v, got %v", originalID, agent2.ID)
	}
	if agent2.OrgID != originalID {
		t.Errorf("OrgID round-trip failed: expected %v, got %v", originalID, agent2.OrgID)
	}
}

func TestTime_JSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	org := &Organization{
		ID:        uuid.New(),
		Name:      "Test Org",
		Slug:      "test-org",
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(org)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var org2 Organization
	if err := json.Unmarshal(data, &org2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !org2.CreatedAt.Equal(now) {
		t.Errorf("time round-trip failed: expected %v, got %v", now, org2.CreatedAt)
	}
}

func TestEmailChannelConfig_JSON(t *testing.T) {
	config := EmailChannelConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
		TLS:      true,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var config2 EmailChannelConfig
	if err := json.Unmarshal(data, &config2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if config2.Host != "smtp.example.com" {
		t.Errorf("expected host 'smtp.example.com', got %s", config2.Host)
	}
	if config2.Port != 587 {
		t.Errorf("expected port 587, got %d", config2.Port)
	}
	if !config2.TLS {
		t.Error("expected TLS to be true")
	}
}

func TestSlackChannelConfig_JSON(t *testing.T) {
	config := SlackChannelConfig{WebhookURL: "https://hooks.slack.com/test"}
	data, _ := json.Marshal(config)

	var config2 SlackChannelConfig
	json.Unmarshal(data, &config2)

	if config2.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("expected webhook URL, got %s", config2.WebhookURL)
	}
}

func TestWebhookChannelConfig_JSON(t *testing.T) {
	config := WebhookChannelConfig{URL: "https://example.com/webhook", Secret: "secret123"}
	data, _ := json.Marshal(config)

	var config2 WebhookChannelConfig
	json.Unmarshal(data, &config2)

	if config2.URL != "https://example.com/webhook" {
		t.Errorf("expected URL, got %s", config2.URL)
	}
	if config2.Secret != "secret123" {
		t.Errorf("expected secret, got %s", config2.Secret)
	}
}

func TestDRTest_JSONRoundTrip(t *testing.T) {
	test := NewDRTest(uuid.New())
	scheduleID := uuid.New()
	agentID := uuid.New()
	test.SetSchedule(scheduleID)
	test.SetAgent(agentID)

	data, err := json.Marshal(test)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var test2 DRTest
	if err := json.Unmarshal(data, &test2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if test2.ScheduleID == nil || *test2.ScheduleID != scheduleID {
		t.Error("expected ScheduleID to match")
	}
	if test2.AgentID == nil || *test2.AgentID != agentID {
		t.Error("expected AgentID to match")
	}
	if test2.Status != DRTestStatusScheduled {
		t.Errorf("expected status 'scheduled', got %s", test2.Status)
	}
}

func TestMaintenanceWindow_JSONRoundTrip(t *testing.T) {
	starts := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
	ends := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)
	mw := NewMaintenanceWindow(uuid.New(), "DB Migration", starts, ends)

	data, err := json.Marshal(mw)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var mw2 MaintenanceWindow
	if err := json.Unmarshal(data, &mw2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if mw2.Title != "DB Migration" {
		t.Errorf("expected title 'DB Migration', got %s", mw2.Title)
	}
	if mw2.NotifyBeforeMinutes != 60 {
		t.Errorf("expected NotifyBeforeMinutes 60, got %d", mw2.NotifyBeforeMinutes)
	}
}

func TestReplicationStatus_JSONRoundTrip(t *testing.T) {
	rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())

	data, err := json.Marshal(rs)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	if raw["status"] != string(ReplicationStatusPending) {
		t.Errorf("expected status 'pending', got %v", raw["status"])
	}
	// Optional fields should be omitted
	if raw["last_snapshot_id"] != nil {
		t.Error("expected last_snapshot_id to be omitted")
	}
	if raw["error_message"] != nil {
		t.Error("expected error_message to be omitted")
	}
}
