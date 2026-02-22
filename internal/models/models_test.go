package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Constructor Tests ---

func TestNewOrganization(t *testing.T) {
	org := NewOrganization("Acme Corp", "acme-corp")

	if org.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if org.Name != "Acme Corp" {
		t.Errorf("expected Name 'Acme Corp', got %s", org.Name)
	}
	if org.Slug != "acme-corp" {
		t.Errorf("expected Slug 'acme-corp', got %s", org.Slug)
	}
	if org.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewUser(t *testing.T) {
	orgID := uuid.New()
	user := NewUser(orgID, "oidc|123", "user@example.com", "Test User", UserRoleAdmin)

	if user.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if user.OrgID != orgID {
		t.Errorf("expected OrgID %v, got %v", orgID, user.OrgID)
	}
	if user.OIDCSubject != "oidc|123" {
		t.Errorf("expected OIDCSubject 'oidc|123', got %s", user.OIDCSubject)
	}
	if user.Email != "user@example.com" {
		t.Errorf("expected Email 'user@example.com', got %s", user.Email)
	}
	if user.Role != UserRoleAdmin {
		t.Errorf("expected Role admin, got %s", user.Role)
	}
}

func TestNewRepository(t *testing.T) {
	orgID := uuid.New()
	repo := NewRepository(orgID, "prod-backups", RepositoryTypeS3, []byte("config"))

	if repo.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if repo.OrgID != orgID {
		t.Errorf("expected OrgID %v, got %v", orgID, repo.OrgID)
	}
	if repo.Name != "prod-backups" {
		t.Errorf("expected Name 'prod-backups', got %s", repo.Name)
	}
	if repo.Type != RepositoryTypeS3 {
		t.Errorf("expected Type 's3', got %s", repo.Type)
	}
	if string(repo.ConfigEncrypted) != "config" {
		t.Error("expected ConfigEncrypted to be set")
	}
}

func TestNewNotificationChannel(t *testing.T) {
	orgID := uuid.New()
	ch := NewNotificationChannel(orgID, "slack-ops", ChannelTypeSlack, []byte("config"))

	if ch.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if ch.Name != "slack-ops" {
		t.Errorf("expected Name 'slack-ops', got %s", ch.Name)
	}
	if ch.Type != ChannelTypeSlack {
		t.Errorf("expected Type 'slack', got %s", ch.Type)
	}
	if !ch.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestNewNotificationPreference(t *testing.T) {
	orgID := uuid.New()
	channelID := uuid.New()
	pref := NewNotificationPreference(orgID, channelID, EventBackupFailed)

	if pref.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if pref.ChannelID != channelID {
		t.Errorf("expected ChannelID %v, got %v", channelID, pref.ChannelID)
	}
	if pref.EventType != EventBackupFailed {
		t.Errorf("expected EventType 'backup_failed', got %s", pref.EventType)
	}
	if !pref.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestNewBackupScript(t *testing.T) {
	scheduleID := uuid.New()
	script := NewBackupScript(scheduleID, BackupScriptTypePreBackup, "#!/bin/bash\necho start")

	if script.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if script.ScheduleID != scheduleID {
		t.Errorf("expected ScheduleID %v, got %v", scheduleID, script.ScheduleID)
	}
	if script.Type != BackupScriptTypePreBackup {
		t.Errorf("expected Type 'pre_backup', got %s", script.Type)
	}
	if script.TimeoutSeconds != 300 {
		t.Errorf("expected TimeoutSeconds 300 (default), got %d", script.TimeoutSeconds)
	}
	if script.FailOnError {
		t.Error("expected FailOnError to be false by default")
	}
	if !script.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestNewAuditLog(t *testing.T) {
	orgID := uuid.New()
	log := NewAuditLog(orgID, AuditActionCreate, "agent", AuditResultSuccess)

	if log.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if log.OrgID != orgID {
		t.Errorf("expected OrgID %v, got %v", orgID, log.OrgID)
	}
	if log.Action != AuditActionCreate {
		t.Errorf("expected Action 'create', got %s", log.Action)
	}
	if log.Result != AuditResultSuccess {
		t.Errorf("expected Result 'success', got %s", log.Result)
	}
}

func TestNewVerificationSchedule(t *testing.T) {
	repoID := uuid.New()
	vs := NewVerificationSchedule(repoID, VerificationTypeCheck, "0 3 * * *")

	if vs.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if vs.RepositoryID != repoID {
		t.Errorf("expected RepositoryID %v, got %v", repoID, vs.RepositoryID)
	}
	if vs.CronExpression != "0 3 * * *" {
		t.Errorf("expected CronExpression '0 3 * * *', got %s", vs.CronExpression)
	}
	if !vs.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestNewDRTestSchedule(t *testing.T) {
	runbookID := uuid.New()
	ts := NewDRTestSchedule(runbookID, "0 0 * * 0")

	if ts.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if ts.RunbookID != runbookID {
		t.Errorf("expected RunbookID %v, got %v", runbookID, ts.RunbookID)
	}
	if !ts.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestNewStorageStats(t *testing.T) {
	repoID := uuid.New()
	stats := NewStorageStats(repoID)

	if stats.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if stats.RepositoryID != repoID {
		t.Errorf("expected RepositoryID %v, got %v", repoID, stats.RepositoryID)
	}
	if stats.CollectedAt.IsZero() {
		t.Error("expected CollectedAt to be set")
	}
}

// --- Status Transitions ---

func TestRestore_StatusTransitions(t *testing.T) {
	t.Run("pending to running", func(t *testing.T) {
		restore := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		if restore.Status != RestoreStatusPending {
			t.Errorf("expected Status pending, got %s", restore.Status)
		}

		restore.Start()
		if restore.Status != RestoreStatusRunning {
			t.Errorf("expected Status running, got %s", restore.Status)
		}
		if restore.StartedAt == nil {
			t.Fatal("expected StartedAt to be set")
		}
	})

	t.Run("running to completed", func(t *testing.T) {
		restore := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		restore.Start()
		restore.Complete()

		if restore.Status != RestoreStatusCompleted {
			t.Errorf("expected Status completed, got %s", restore.Status)
		}
		if restore.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}
	})

	t.Run("running to failed", func(t *testing.T) {
		restore := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		restore.Start()
		restore.Fail("disk error")

		if restore.Status != RestoreStatusFailed {
			t.Errorf("expected Status failed, got %s", restore.Status)
		}
		if restore.ErrorMessage != "disk error" {
			t.Errorf("expected ErrorMessage 'disk error', got %s", restore.ErrorMessage)
		}
	})

	t.Run("to canceled", func(t *testing.T) {
		restore := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		restore.Cancel()

		if restore.Status != RestoreStatusCanceled {
			t.Errorf("expected Status canceled, got %s", restore.Status)
		}
	})
}

func TestRestore_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   RestoreStatus
		complete bool
	}{
		{"pending", RestoreStatusPending, false},
		{"running", RestoreStatusRunning, false},
		{"completed", RestoreStatusCompleted, true},
		{"failed", RestoreStatusFailed, true},
		{"canceled", RestoreStatusCanceled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Restore{Status: tt.status}
			if got := r.IsComplete(); got != tt.complete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.complete)
			}
		})
	}
}

func TestRestore_Duration(t *testing.T) {
	t.Run("not started", func(t *testing.T) {
		r := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		if d := r.Duration(); d != 0 {
			t.Errorf("expected 0 duration, got %v", d)
		}
	})

	t.Run("started but not completed", func(t *testing.T) {
		r := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		r.Start()
		if d := r.Duration(); d != 0 {
			t.Errorf("expected 0 duration, got %v", d)
		}
	})

	t.Run("completed", func(t *testing.T) {
		r := NewRestore(uuid.New(), uuid.New(), "snap-1", "/restore", nil, nil)
		r.Start()
		time.Sleep(10 * time.Millisecond)
		r.Complete()

		if d := r.Duration(); d <= 0 {
			t.Errorf("expected positive duration, got %v", d)
		}
	})
}

func TestVerification_StatusTransitions(t *testing.T) {
	t.Run("running to passed", func(t *testing.T) {
		v := NewVerification(uuid.New(), VerificationTypeCheck)
		if v.Status != VerificationStatusRunning {
			t.Errorf("expected Status running, got %s", v.Status)
		}

		details := &VerificationDetails{ErrorsFound: []string{}}
		v.Pass(details)

		if v.Status != VerificationStatusPassed {
			t.Errorf("expected Status passed, got %s", v.Status)
		}
		if v.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}
		if v.DurationMs == nil {
			t.Fatal("expected DurationMs to be set")
		}
		if v.Details == nil {
			t.Error("expected Details to be set")
		}
	})

	t.Run("running to failed", func(t *testing.T) {
		v := NewVerification(uuid.New(), VerificationTypeCheckReadData)

		details := &VerificationDetails{ErrorsFound: []string{"corrupt block"}}
		v.Fail("integrity check failed", details)

		if v.Status != VerificationStatusFailed {
			t.Errorf("expected Status failed, got %s", v.Status)
		}
		if v.ErrorMessage != "integrity check failed" {
			t.Errorf("expected ErrorMessage, got %s", v.ErrorMessage)
		}
		if v.Details == nil || len(v.Details.ErrorsFound) != 1 {
			t.Error("expected Details with errors")
		}
	})
}

func TestVerification_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   VerificationStatus
		complete bool
	}{
		{"pending", VerificationStatusPending, false},
		{"running", VerificationStatusRunning, false},
		{"passed", VerificationStatusPassed, true},
		{"failed", VerificationStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Verification{Status: tt.status}
			if got := v.IsComplete(); got != tt.complete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.complete)
			}
		})
	}
}

func TestVerification_Duration(t *testing.T) {
	t.Run("not completed", func(t *testing.T) {
		v := NewVerification(uuid.New(), VerificationTypeCheck)
		if d := v.Duration(); d != 0 {
			t.Errorf("expected 0 duration, got %v", d)
		}
	})

	t.Run("completed", func(t *testing.T) {
		v := NewVerification(uuid.New(), VerificationTypeCheck)
		time.Sleep(10 * time.Millisecond)
		v.Pass(nil)

		if d := v.Duration(); d <= 0 {
			t.Errorf("expected positive duration, got %v", d)
		}
	})
}

func TestAlert_StatusTransitions(t *testing.T) {
	t.Run("active to acknowledged", func(t *testing.T) {
		alert := NewAlert(uuid.New(), AlertTypeAgentOffline, AlertSeverityCritical, "Agent Down", "Server offline")
		if alert.Status != AlertStatusActive {
			t.Errorf("expected Status active, got %s", alert.Status)
		}

		userID := uuid.New()
		alert.Acknowledge(userID)

		if alert.Status != AlertStatusAcknowledged {
			t.Errorf("expected Status acknowledged, got %s", alert.Status)
		}
		if alert.AcknowledgedBy == nil || *alert.AcknowledgedBy != userID {
			t.Error("expected AcknowledgedBy to match user")
		}
		if alert.AcknowledgedAt == nil {
			t.Error("expected AcknowledgedAt to be set")
		}
	})

	t.Run("to resolved", func(t *testing.T) {
		alert := NewAlert(uuid.New(), AlertTypeBackupSLA, AlertSeverityWarning, "SLA Breach", "24h since backup")
		alert.Resolve()

		if alert.Status != AlertStatusResolved {
			t.Errorf("expected Status resolved, got %s", alert.Status)
		}
		if alert.ResolvedAt == nil {
			t.Error("expected ResolvedAt to be set")
		}
	})
}

func TestAlert_SetResource(t *testing.T) {
	alert := NewAlert(uuid.New(), AlertTypeAgentOffline, AlertSeverityCritical, "test", "test")
	resourceID := uuid.New()
	alert.SetResource(ResourceTypeAgent, resourceID)

	if alert.ResourceType == nil || *alert.ResourceType != ResourceTypeAgent {
		t.Error("expected ResourceType to be agent")
	}
	if alert.ResourceID == nil || *alert.ResourceID != resourceID {
		t.Error("expected ResourceID to match")
	}
}

func TestAlert_SetRuleID(t *testing.T) {
	alert := NewAlert(uuid.New(), AlertTypeBackupSLA, AlertSeverityInfo, "test", "test")
	ruleID := uuid.New()
	alert.SetRuleID(ruleID)

	if alert.RuleID == nil || *alert.RuleID != ruleID {
		t.Error("expected RuleID to match")
	}
}

func TestNotificationLog_StatusTransitions(t *testing.T) {
	t.Run("queued to sent", func(t *testing.T) {
		log := NewNotificationLog(uuid.New(), nil, "backup_success", "admin@example.com", "Backup OK")
		if log.Status != NotificationStatusQueued {
			t.Errorf("expected Status queued, got %s", log.Status)
		}

		log.MarkSent()
		if log.Status != NotificationStatusSent {
			t.Errorf("expected Status sent, got %s", log.Status)
		}
		if log.SentAt == nil {
			t.Error("expected SentAt to be set")
		}
	})

	t.Run("queued to failed", func(t *testing.T) {
		log := NewNotificationLog(uuid.New(), nil, "backup_failed", "admin@example.com", "Backup Failed")
		log.MarkFailed("SMTP connection refused")

		if log.Status != NotificationStatusFailed {
			t.Errorf("expected Status failed, got %s", log.Status)
		}
		if log.ErrorMessage != "SMTP connection refused" {
			t.Errorf("expected ErrorMessage, got %s", log.ErrorMessage)
		}
	})
}

func TestDRRunbook_StatusTransitions(t *testing.T) {
	t.Run("draft to active", func(t *testing.T) {
		rb := NewDRRunbook(uuid.New(), "DR Plan A")
		if rb.Status != DRRunbookStatusDraft {
			t.Errorf("expected Status draft, got %s", rb.Status)
		}

		rb.Activate()
		if rb.Status != DRRunbookStatusActive {
			t.Errorf("expected Status active, got %s", rb.Status)
		}
	})

	t.Run("active to archived", func(t *testing.T) {
		rb := NewDRRunbook(uuid.New(), "DR Plan B")
		rb.Activate()
		rb.Archive()

		if rb.Status != DRRunbookStatusArchived {
			t.Errorf("expected Status archived, got %s", rb.Status)
		}
	})
}

func TestDRRunbook_AddStep(t *testing.T) {
	rb := NewDRRunbook(uuid.New(), "test")

	rb.AddStep("First", "Do first thing", DRRunbookStepTypeManual)
	rb.AddStep("Second", "Do second thing", DRRunbookStepTypeRestore)
	rb.AddStep("Third", "Do third thing", DRRunbookStepTypeVerify)

	if len(rb.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(rb.Steps))
	}

	// Check auto-incrementing order
	for i, step := range rb.Steps {
		if step.Order != i+1 {
			t.Errorf("expected step %d order %d, got %d", i, i+1, step.Order)
		}
	}

	if rb.Steps[0].Type != DRRunbookStepTypeManual {
		t.Errorf("expected step 0 type manual, got %s", rb.Steps[0].Type)
	}
	if rb.Steps[1].Type != DRRunbookStepTypeRestore {
		t.Errorf("expected step 1 type restore, got %s", rb.Steps[1].Type)
	}
}

func TestDRRunbook_AddContact(t *testing.T) {
	rb := NewDRRunbook(uuid.New(), "test")

	rb.AddContact("Alice", "DBA", "alice@example.com", "+1111111111", true)
	rb.AddContact("Bob", "DevOps", "bob@example.com", "", false)

	if len(rb.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(rb.Contacts))
	}
	if rb.Contacts[0].Name != "Alice" {
		t.Errorf("expected first contact 'Alice', got %s", rb.Contacts[0].Name)
	}
	if rb.Contacts[1].Phone != "" {
		t.Errorf("expected empty phone for Bob, got %s", rb.Contacts[1].Phone)
	}
}

func TestDRTest_StatusTransitions(t *testing.T) {
	t.Run("scheduled to running", func(t *testing.T) {
		test := NewDRTest(uuid.New())
		if test.Status != DRTestStatusScheduled {
			t.Errorf("expected Status scheduled, got %s", test.Status)
		}

		test.Start()
		if test.Status != DRTestStatusRunning {
			t.Errorf("expected Status running, got %s", test.Status)
		}
		if test.StartedAt == nil {
			t.Error("expected StartedAt to be set")
		}
	})

	t.Run("running to completed", func(t *testing.T) {
		test := NewDRTest(uuid.New())
		test.Start()
		test.Complete("snap-123", 1024*1024*100, 120, true)

		if test.Status != DRTestStatusCompleted {
			t.Errorf("expected Status completed, got %s", test.Status)
		}
		if test.CompletedAt == nil {
			t.Fatal("expected CompletedAt to be set")
		}
		if test.SnapshotID != "snap-123" {
			t.Errorf("expected SnapshotID 'snap-123', got %s", test.SnapshotID)
		}
		if test.RestoreSizeBytes == nil || *test.RestoreSizeBytes != 1024*1024*100 {
			t.Error("expected RestoreSizeBytes to match")
		}
		if test.RestoreDurationSeconds == nil || *test.RestoreDurationSeconds != 120 {
			t.Error("expected RestoreDurationSeconds to match")
		}
		if test.VerificationPassed == nil || !*test.VerificationPassed {
			t.Error("expected VerificationPassed to be true")
		}
	})

	t.Run("running to failed", func(t *testing.T) {
		test := NewDRTest(uuid.New())
		test.Start()
		test.Fail("restore timed out")

		if test.Status != DRTestStatusFailed {
			t.Errorf("expected Status failed, got %s", test.Status)
		}
		if test.ErrorMessage != "restore timed out" {
			t.Errorf("expected ErrorMessage 'restore timed out', got %s", test.ErrorMessage)
		}
		if test.VerificationPassed == nil || *test.VerificationPassed != false {
			t.Error("expected VerificationPassed to be false on failure")
		}
	})

	t.Run("canceled", func(t *testing.T) {
		test := NewDRTest(uuid.New())
		test.Cancel("no longer needed")

		if test.Status != DRTestStatusCanceled {
			t.Errorf("expected Status canceled, got %s", test.Status)
		}
		if test.Notes != "no longer needed" {
			t.Errorf("expected Notes, got %s", test.Notes)
		}
	})
}

func TestDRTest_SetScheduleAndAgent(t *testing.T) {
	test := NewDRTest(uuid.New())
	scheduleID := uuid.New()
	agentID := uuid.New()

	test.SetSchedule(scheduleID)
	test.SetAgent(agentID)

	if test.ScheduleID == nil || *test.ScheduleID != scheduleID {
		t.Error("expected ScheduleID to match")
	}
	if test.AgentID == nil || *test.AgentID != agentID {
		t.Error("expected AgentID to match")
	}
}

func TestReplicationStatus_Transitions(t *testing.T) {
	t.Run("pending to syncing", func(t *testing.T) {
		rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())
		if rs.Status != ReplicationStatusPending {
			t.Errorf("expected Status pending, got %s", rs.Status)
		}

		rs.MarkSyncing()
		if rs.Status != ReplicationStatusSyncing {
			t.Errorf("expected Status syncing, got %s", rs.Status)
		}
	})

	t.Run("syncing to synced", func(t *testing.T) {
		rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())
		rs.MarkSyncing()
		rs.MarkSynced("snap-abc")

		if rs.Status != ReplicationStatusSynced {
			t.Errorf("expected Status synced, got %s", rs.Status)
		}
		if rs.LastSnapshotID == nil || *rs.LastSnapshotID != "snap-abc" {
			t.Error("expected LastSnapshotID to match")
		}
		if rs.LastSyncAt == nil {
			t.Error("expected LastSyncAt to be set")
		}
		if rs.ErrorMessage != nil {
			t.Error("expected ErrorMessage to be cleared")
		}
	})

	t.Run("syncing to failed", func(t *testing.T) {
		rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())
		rs.MarkSyncing()
		rs.MarkFailed("connection refused")

		if rs.Status != ReplicationStatusFailed {
			t.Errorf("expected Status failed, got %s", rs.Status)
		}
		if rs.ErrorMessage == nil || *rs.ErrorMessage != "connection refused" {
			t.Error("expected ErrorMessage to match")
		}
	})

	t.Run("IsSynced", func(t *testing.T) {
		rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())
		if rs.IsSynced() {
			t.Error("expected IsSynced false for pending")
		}

		rs.MarkSynced("snap-1")
		if !rs.IsSynced() {
			t.Error("expected IsSynced true after MarkSynced")
		}
	})
}

func TestReplicationStatus_SyncClearsError(t *testing.T) {
	rs := NewReplicationStatus(uuid.New(), uuid.New(), uuid.New())
	rs.MarkFailed("error 1")
	rs.MarkSyncing()
	rs.MarkSynced("snap-2")

	if rs.ErrorMessage != nil {
		t.Errorf("expected ErrorMessage to be nil after sync, got %v", rs.ErrorMessage)
	}
}

// --- MaintenanceWindow Tests ---

func TestMaintenanceWindow_IsActive(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now.Add(-1*time.Hour),
		now.Add(1*time.Hour),
	)

	if !mw.IsActive(now) {
		t.Error("expected IsActive true during window")
	}

	if mw.IsActive(now.Add(-2 * time.Hour)) {
		t.Error("expected IsActive false before window")
	}

	if mw.IsActive(now.Add(2 * time.Hour)) {
		t.Error("expected IsActive false after window")
	}
}

func TestMaintenanceWindow_IsUpcoming(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now.Add(30*time.Minute),
		now.Add(90*time.Minute),
	)

	if !mw.IsUpcoming(now, 1*time.Hour) {
		t.Error("expected IsUpcoming true within notify window")
	}

	if mw.IsUpcoming(now, 10*time.Minute) {
		t.Error("expected IsUpcoming false when notify window too small")
	}

	if mw.IsUpcoming(now.Add(60*time.Minute), 1*time.Hour) {
		t.Error("expected IsUpcoming false after start")
	}
}

func TestMaintenanceWindow_IsPast(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now.Add(-2*time.Hour),
		now.Add(-1*time.Hour),
	)

	if !mw.IsPast(now) {
		t.Error("expected IsPast true after window")
	}

	if mw.IsPast(now.Add(-3 * time.Hour)) {
		t.Error("expected IsPast false before window")
	}
}

func TestMaintenanceWindow_Duration(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now,
		now.Add(2*time.Hour),
	)

	if d := mw.Duration(); d != 2*time.Hour {
		t.Errorf("expected 2h duration, got %v", d)
	}
}

func TestMaintenanceWindow_TimeUntilStartEnd(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now.Add(1*time.Hour),
		now.Add(3*time.Hour),
	)

	startDelta := mw.TimeUntilStart(now)
	if startDelta < 59*time.Minute || startDelta > 61*time.Minute {
		t.Errorf("expected ~1h until start, got %v", startDelta)
	}

	endDelta := mw.TimeUntilEnd(now)
	if endDelta < 2*time.Hour+59*time.Minute || endDelta > 3*time.Hour+time.Minute {
		t.Errorf("expected ~3h until end, got %v", endDelta)
	}
}

func TestMaintenanceWindow_ShouldNotify(t *testing.T) {
	now := time.Now()
	mw := NewMaintenanceWindow(uuid.New(), "Maintenance",
		now.Add(30*time.Minute),
		now.Add(90*time.Minute),
	)

	// Default notify is 60 minutes before
	if !mw.ShouldNotify(now) {
		t.Error("expected ShouldNotify true when upcoming and not notified")
	}

	// After notification sent
	mw.NotificationSent = true
	if mw.ShouldNotify(now) {
		t.Error("expected ShouldNotify false when already notified")
	}
}

// --- Onboarding Tests ---

func TestOnboardingProgress_CompleteStep(t *testing.T) {
	p := NewOnboardingProgress(uuid.New())

	if p.CurrentStep != OnboardingStepWelcome {
		t.Errorf("expected initial step 'welcome', got %s", p.CurrentStep)
	}

	p.CompleteStep(OnboardingStepWelcome)
	if p.CurrentStep != OnboardingStepLicense {
		t.Errorf("expected step 'license' after welcome, got %s", p.CurrentStep)
	if p.CurrentStep != OnboardingStepOrganization {
		t.Errorf("expected step 'organization' after welcome, got %s", p.CurrentStep)
	}
	if !p.HasCompletedStep(OnboardingStepWelcome) {
		t.Error("expected welcome to be completed")
	}

	// Complete multiple steps
	p.CompleteStep(OnboardingStepLicense)
	p.CompleteStep(OnboardingStepOrganization)
	p.CompleteStep(OnboardingStepSMTP)
	p.CompleteStep(OnboardingStepRepository)
	p.CompleteStep(OnboardingStepAgent)
	p.CompleteStep(OnboardingStepSchedule)

	if p.CurrentStep != OnboardingStepVerify {
		t.Errorf("expected step 'verify', got %s", p.CurrentStep)
	}

	// Completing verify should mark as complete
	p.CompleteStep(OnboardingStepVerify)
	if p.CurrentStep != OnboardingStepComplete {
		t.Errorf("expected step 'complete', got %s", p.CurrentStep)
	}
	if p.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestOnboardingProgress_IsComplete(t *testing.T) {
	t.Run("not complete", func(t *testing.T) {
		p := NewOnboardingProgress(uuid.New())
		if p.IsComplete() {
			t.Error("expected IsComplete false initially")
		}
	})

	t.Run("complete via steps", func(t *testing.T) {
		p := NewOnboardingProgress(uuid.New())
		p.CurrentStep = OnboardingStepComplete
		if !p.IsComplete() {
			t.Error("expected IsComplete true at complete step")
		}
	})

	t.Run("complete via skip", func(t *testing.T) {
		p := NewOnboardingProgress(uuid.New())
		p.Skip()
		if !p.IsComplete() {
			t.Error("expected IsComplete true after skip")
		}
	})

	t.Run("complete via CompletedAt", func(t *testing.T) {
		p := NewOnboardingProgress(uuid.New())
		now := time.Now()
		p.CompletedAt = &now
		if !p.IsComplete() {
			t.Error("expected IsComplete true with CompletedAt set")
		}
	})
}

func TestOnboardingProgress_HasCompletedStep(t *testing.T) {
	p := NewOnboardingProgress(uuid.New())
	p.CompleteStep(OnboardingStepWelcome)
	p.CompleteStep(OnboardingStepOrganization)

	if !p.HasCompletedStep(OnboardingStepWelcome) {
		t.Error("expected HasCompletedStep true for welcome")
	}
	if !p.HasCompletedStep(OnboardingStepOrganization) {
		t.Error("expected HasCompletedStep true for organization")
	}
	if p.HasCompletedStep(OnboardingStepSMTP) {
		t.Error("expected HasCompletedStep false for smtp")
	}
}

func TestOnboardingProgress_Skip(t *testing.T) {
	p := NewOnboardingProgress(uuid.New())
	p.Skip()

	if !p.Skipped {
		t.Error("expected Skipped to be true")
	}
	if p.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	if !p.IsComplete() {
		t.Error("expected IsComplete true after skip")
	}
}

func TestOnboardingProgress_DuplicateStep(t *testing.T) {
	p := NewOnboardingProgress(uuid.New())
	p.CompleteStep(OnboardingStepWelcome)
	p.CompleteStep(OnboardingStepWelcome) // duplicate

	count := 0
	for _, s := range p.CompletedSteps {
		if s == OnboardingStepWelcome {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected welcome to appear once in completed steps, got %d", count)
	}
}

// --- AuditLog Builder Pattern ---

func TestAuditLog_BuilderPattern(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	agentID := uuid.New()
	resourceID := uuid.New()

	log := NewAuditLog(orgID, AuditActionCreate, "agent", AuditResultSuccess).
		WithUser(userID).
		WithAgent(agentID).
		WithResource(resourceID).
		WithRequestInfo("192.168.1.1", "Mozilla/5.0").
		WithDetails("Created new agent")

	if log.UserID == nil || *log.UserID != userID {
		t.Error("expected UserID to match")
	}
	if log.AgentID == nil || *log.AgentID != agentID {
		t.Error("expected AgentID to match")
	}
	if log.ResourceID == nil || *log.ResourceID != resourceID {
		t.Error("expected ResourceID to match")
	}
	if log.IPAddress != "192.168.1.1" {
		t.Errorf("expected IPAddress '192.168.1.1', got %s", log.IPAddress)
	}
	if log.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent 'Mozilla/5.0', got %s", log.UserAgent)
	}
	if log.Details != "Created new agent" {
		t.Errorf("expected Details 'Created new agent', got %s", log.Details)
	}
}

func TestAuditLog_IsSuccess(t *testing.T) {
	tests := []struct {
		name    string
		result  AuditResult
		success bool
	}{
		{"success", AuditResultSuccess, true},
		{"failure", AuditResultFailure, false},
		{"denied", AuditResultDenied, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewAuditLog(uuid.New(), AuditActionLogin, "session", tt.result)
			if got := log.IsSuccess(); got != tt.success {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.success)
			}
		})
	}
}

func TestAuditLog_IsUserAction(t *testing.T) {
	log := NewAuditLog(uuid.New(), AuditActionLogin, "session", AuditResultSuccess)
	if log.IsUserAction() {
		t.Error("expected IsUserAction false without user")
	}

	log.WithUser(uuid.New())
	if !log.IsUserAction() {
		t.Error("expected IsUserAction true with user")
	}
}

func TestAuditLog_IsAgentAction(t *testing.T) {
	log := NewAuditLog(uuid.New(), AuditActionBackup, "backup", AuditResultSuccess)
	if log.IsAgentAction() {
		t.Error("expected IsAgentAction false without agent")
	}

	log.WithAgent(uuid.New())
	if !log.IsAgentAction() {
		t.Error("expected IsAgentAction true with agent")
	}
}

// --- StorageStats Tests ---

func TestStorageStats_SetStats(t *testing.T) {
	stats := NewStorageStats(uuid.New())

	t.Run("normal dedup", func(t *testing.T) {
		stats.SetStats(100, 50, 100, 500, 10)

		if stats.TotalSize != 100 {
			t.Errorf("expected TotalSize 100, got %d", stats.TotalSize)
		}
		if stats.TotalFileCount != 50 {
			t.Errorf("expected TotalFileCount 50, got %d", stats.TotalFileCount)
		}
		if stats.SnapshotCount != 10 {
			t.Errorf("expected SnapshotCount 10, got %d", stats.SnapshotCount)
		}
		if stats.DedupRatio != 5.0 {
			t.Errorf("expected DedupRatio 5.0, got %f", stats.DedupRatio)
		}
		if stats.SpaceSaved != 400 {
			t.Errorf("expected SpaceSaved 400, got %d", stats.SpaceSaved)
		}
		if stats.SpaceSavedPct != 80.0 {
			t.Errorf("expected SpaceSavedPct 80.0, got %f", stats.SpaceSavedPct)
		}
	})

	t.Run("zero raw data", func(t *testing.T) {
		s := NewStorageStats(uuid.New())
		s.SetStats(0, 0, 0, 0, 0)

		if s.DedupRatio != 0 {
			t.Errorf("expected DedupRatio 0 for zero raw data, got %f", s.DedupRatio)
		}
		if s.SpaceSaved != 0 {
			t.Errorf("expected SpaceSaved 0 for zero raw data, got %d", s.SpaceSaved)
		}
	})
}

// --- OrgInvitation Tests ---

func TestOrgInvitation_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		inv := NewOrgInvitation(uuid.New(), "test@example.com", OrgRoleMember, "token", uuid.New(), time.Now().Add(24*time.Hour))
		if inv.IsExpired() {
			t.Error("expected IsExpired false for future expiry")
		}
	})

	t.Run("expired", func(t *testing.T) {
		inv := NewOrgInvitation(uuid.New(), "test@example.com", OrgRoleMember, "token", uuid.New(), time.Now().Add(-1*time.Hour))
		if !inv.IsExpired() {
			t.Error("expected IsExpired true for past expiry")
		}
	})
}

func TestOrgInvitation_IsAccepted(t *testing.T) {
	t.Run("not accepted", func(t *testing.T) {
		inv := NewOrgInvitation(uuid.New(), "test@example.com", OrgRoleMember, "token", uuid.New(), time.Now().Add(24*time.Hour))
		if inv.IsAccepted() {
			t.Error("expected IsAccepted false")
		}
	})

	t.Run("accepted", func(t *testing.T) {
		inv := NewOrgInvitation(uuid.New(), "test@example.com", OrgRoleMember, "token", uuid.New(), time.Now().Add(24*time.Hour))
		now := time.Now()
		inv.AcceptedAt = &now
		if !inv.IsAccepted() {
			t.Error("expected IsAccepted true")
		}
	})
}

// --- Policy ApplyToSchedule ---

func TestPolicy_ApplyToSchedule(t *testing.T) {
	policy := NewPolicy(uuid.New(), "standard")
	policy.Paths = []string{"/data", "/home"}
	policy.Excludes = []string{"*.tmp", "*.log"}
	policy.RetentionPolicy = &RetentionPolicy{KeepLast: 10, KeepDaily: 14, KeepWeekly: 8}
	bw := 2048
	policy.BandwidthLimitKB = &bw
	policy.BackupWindow = &BackupWindow{Start: "02:00", End: "06:00"}
	policy.ExcludedHours = []int{9, 17}
	policy.CronExpression = "0 3 * * *"

	schedule := NewSchedule(uuid.New(), "test-schedule", "0 * * * *", []string{"/old"})
	policy.ApplyToSchedule(schedule)

	if schedule.PolicyID == nil || *schedule.PolicyID != policy.ID {
		t.Error("expected PolicyID to be set")
	}

	// Verify deep copy of paths
	if len(schedule.Paths) != 2 || schedule.Paths[0] != "/data" {
		t.Errorf("expected Paths copied, got %v", schedule.Paths)
	}
	policy.Paths[0] = "/modified"
	if schedule.Paths[0] == "/modified" {
		t.Error("expected Paths to be a deep copy, not a reference")
	}

	// Verify deep copy of excludes
	if len(schedule.Excludes) != 2 {
		t.Errorf("expected 2 Excludes, got %d", len(schedule.Excludes))
	}

	// Verify deep copy of retention
	if schedule.RetentionPolicy == nil || schedule.RetentionPolicy.KeepLast != 10 {
		t.Error("expected RetentionPolicy copied")
	}
	policy.RetentionPolicy.KeepLast = 99
	if schedule.RetentionPolicy.KeepLast == 99 {
		t.Error("expected RetentionPolicy to be a deep copy")
	}

	// Verify bandwidth
	if schedule.BandwidthLimitKB == nil || *schedule.BandwidthLimitKB != 2048 {
		t.Error("expected BandwidthLimitKB copied")
	}

	// Verify backup window copy
	if schedule.BackupWindow == nil || schedule.BackupWindow.Start != "02:00" {
		t.Error("expected BackupWindow copied")
	}

	// Verify excluded hours copy
	if len(schedule.ExcludedHours) != 2 {
		t.Errorf("expected 2 ExcludedHours, got %d", len(schedule.ExcludedHours))
	}

	// Verify cron expression
	if schedule.CronExpression != "0 3 * * *" {
		t.Errorf("expected CronExpression '0 3 * * *', got %s", schedule.CronExpression)
	}
}

func TestPolicy_ApplyToSchedule_NilFields(t *testing.T) {
	policy := NewPolicy(uuid.New(), "minimal")
	schedule := NewSchedule(uuid.New(), "test", "0 * * * *", []string{"/data"})
	originalCron := schedule.CronExpression

	policy.ApplyToSchedule(schedule)

	// Original schedule values should be preserved for nil policy fields
	if schedule.CronExpression != originalCron {
		t.Errorf("expected CronExpression unchanged, got %s", schedule.CronExpression)
	}
}

// --- AgentGroup Tests ---

func TestNewAgentGroup(t *testing.T) {
	orgID := uuid.New()
	group := NewAgentGroup(orgID, "production", "Production servers", "#ff0000")

	if group.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if group.OrgID != orgID {
		t.Errorf("expected OrgID %v, got %v", orgID, group.OrgID)
	}
	if group.Name != "production" {
		t.Errorf("expected Name 'production', got %s", group.Name)
	}
	if group.Description != "Production servers" {
		t.Errorf("expected Description 'Production servers', got %s", group.Description)
	}
	if group.Color != "#ff0000" {
		t.Errorf("expected Color '#ff0000', got %s", group.Color)
	}
}

func TestNewAgentGroupMember(t *testing.T) {
	agentID := uuid.New()
	groupID := uuid.New()
	member := NewAgentGroupMember(agentID, groupID)

	if member.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if member.AgentID != agentID {
		t.Errorf("expected AgentID %v, got %v", agentID, member.AgentID)
	}
	if member.GroupID != groupID {
		t.Errorf("expected GroupID %v, got %v", groupID, member.GroupID)
	}
}

func TestNewTag(t *testing.T) {
	orgID := uuid.New()
	tag := NewTag(orgID, "important", "#ff0000")

	if tag.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if tag.Name != "important" {
		t.Errorf("expected Name 'important', got %s", tag.Name)
	}
	if tag.Color != "#ff0000" {
		t.Errorf("expected Color '#ff0000', got %s", tag.Color)
	}
}

func TestNewTag_DefaultColor(t *testing.T) {
	tag := NewTag(uuid.New(), "test", "")

	if tag.Color != "#6366f1" {
		t.Errorf("expected default Color '#6366f1', got %s", tag.Color)
	}
}
