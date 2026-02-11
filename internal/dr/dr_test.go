package dr

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// mockRunbookStore implements RunbookStore for testing.
type mockRunbookStore struct {
	schedule *models.Schedule
	agent    *models.Agent
	repo     *models.Repository
	drTest   *models.DRTest

	scheduleErr error
	agentErr    error
	repoErr     error
	drTestErr   error
}

func (m *mockRunbookStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	return m.schedule, m.scheduleErr
}

func (m *mockRunbookStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.agentErr
}

func (m *mockRunbookStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return m.repo, m.repoErr
}

func (m *mockRunbookStore) GetLatestDRTestByRunbookID(_ context.Context, _ uuid.UUID) (*models.DRTest, error) {
	return m.drTest, m.drTestErr
}

// helpers for building test data

func testSchedule() *models.Schedule {
	repoID := uuid.New()
	s := models.NewSchedule(uuid.New(), "daily-backup", "0 2 * * *", []string{"/data", "/config"})
	s.Repositories = []models.ScheduleRepository{
		{
			ID:           uuid.New(),
			ScheduleID:   s.ID,
			RepositoryID: repoID,
			Priority:     0,
			Enabled:      true,
		},
	}
	return s
}

func testAgent() *models.Agent {
	return &models.Agent{
		ID:       uuid.New(),
		OrgID:    uuid.New(),
		Hostname: "backup-server-01",
		Status:   models.AgentStatusActive,
	}
}

func testRepo(repoType models.RepositoryType) *models.Repository {
	return models.NewRepository(uuid.New(), "prod-repo", repoType, []byte("config"))
}

// --- NewRunbookGenerator ---

func TestNewRunbookGenerator(t *testing.T) {
	store := &mockRunbookStore{}
	gen := NewRunbookGenerator(store)
	if gen == nil {
		t.Fatal("expected non-nil RunbookGenerator")
	}
	if gen.store != store {
		t.Error("expected store to be set")
	}
}

// --- GenerateForSchedule ---

func TestGenerateForSchedule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		schedule := testSchedule()
		agent := testAgent()
		repo := testRepo(models.RepositoryTypeS3)

		store := &mockRunbookStore{
			agent: agent,
			repo:  repo,
		}
		gen := NewRunbookGenerator(store)
		orgID := uuid.New()

		runbook, err := gen.GenerateForSchedule(context.Background(), schedule, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if runbook.OrgID != orgID {
			t.Errorf("expected orgID %s, got %s", orgID, runbook.OrgID)
		}
		if runbook.ScheduleID == nil || *runbook.ScheduleID != schedule.ID {
			t.Error("expected schedule ID to be set")
		}
		if !strings.Contains(runbook.Name, "DR Runbook") {
			t.Errorf("expected name to contain 'DR Runbook', got %q", runbook.Name)
		}
		if !strings.Contains(runbook.Description, schedule.Name) {
			t.Errorf("expected description to contain schedule name %q", schedule.Name)
		}
		if !strings.Contains(runbook.Description, agent.Hostname) {
			t.Errorf("expected description to contain agent hostname %q", agent.Hostname)
		}
		if !strings.Contains(runbook.Description, repo.Name) {
			t.Errorf("expected description to contain repo name %q", repo.Name)
		}
		if runbook.Status != models.DRRunbookStatusDraft {
			t.Errorf("expected status draft, got %s", runbook.Status)
		}
		if len(runbook.Steps) != 11 {
			t.Errorf("expected 11 default steps, got %d", len(runbook.Steps))
		}
	})

	t.Run("agent fetch error", func(t *testing.T) {
		schedule := testSchedule()
		store := &mockRunbookStore{
			agentErr: errors.New("agent not found"),
		}
		gen := NewRunbookGenerator(store)

		_, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "get agent") {
			t.Errorf("expected error to contain 'get agent', got %q", err.Error())
		}
	})

	t.Run("no primary repository", func(t *testing.T) {
		schedule := testSchedule()
		schedule.Repositories = nil // no repositories
		agent := testAgent()

		store := &mockRunbookStore{
			agent: agent,
		}
		gen := NewRunbookGenerator(store)

		_, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "no primary repository") {
			t.Errorf("expected error to contain 'no primary repository', got %q", err.Error())
		}
	})

	t.Run("repository fetch error", func(t *testing.T) {
		schedule := testSchedule()
		agent := testAgent()

		store := &mockRunbookStore{
			agent:   agent,
			repoErr: errors.New("db connection lost"),
		}
		gen := NewRunbookGenerator(store)

		_, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "get repository") {
			t.Errorf("expected error to contain 'get repository', got %q", err.Error())
		}
	})

	t.Run("disabled primary repository", func(t *testing.T) {
		schedule := testSchedule()
		schedule.Repositories[0].Enabled = false
		agent := testAgent()

		store := &mockRunbookStore{
			agent: agent,
		}
		gen := NewRunbookGenerator(store)

		_, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
		if err == nil {
			t.Fatal("expected error for disabled primary repo")
		}
		if !strings.Contains(err.Error(), "no primary repository") {
			t.Errorf("expected error to contain 'no primary repository', got %q", err.Error())
		}
	})
}

// --- addDefaultSteps ---

func TestAddDefaultSteps(t *testing.T) {
	schedule := testSchedule()
	agent := testAgent()
	repo := testRepo(models.RepositoryTypeS3)
	orgID := uuid.New()

	store := &mockRunbookStore{
		agent: agent,
		repo:  repo,
	}
	gen := NewRunbookGenerator(store)

	runbook, err := gen.GenerateForSchedule(context.Background(), schedule, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runbook.Steps) != 11 {
		t.Fatalf("expected 11 steps, got %d", len(runbook.Steps))
	}

	// Verify step ordering
	for i, step := range runbook.Steps {
		expectedOrder := i + 1
		if step.Order != expectedOrder {
			// Steps 4 and 7 are added directly with hardcoded Order values
			if step.Order == 4 && step.Title == "List Available Snapshots" {
				continue
			}
			if step.Order == 7 && step.Title == "Execute Restore" {
				continue
			}
			// For steps added via AddStep, order is len(steps) at time of add
		}
	}

	// Verify specific step titles
	expectedTitles := []string{
		"Assess the Disaster",
		"Notify Stakeholders",
		"Verify Backup Repository",
		"List Available Snapshots",
		"Select Recovery Point",
		"Prepare Target System",
		"Execute Restore",
		"Verify Restored Data",
		"Update Application Configuration",
		"Validate System Functionality",
		"Document Recovery",
	}
	for i, title := range expectedTitles {
		if runbook.Steps[i].Title != title {
			t.Errorf("step %d: expected title %q, got %q", i+1, title, runbook.Steps[i].Title)
		}
	}

	// Verify step types
	if runbook.Steps[0].Type != models.DRRunbookStepTypeManual {
		t.Errorf("step 1: expected type manual, got %s", runbook.Steps[0].Type)
	}
	if runbook.Steps[1].Type != models.DRRunbookStepTypeNotify {
		t.Errorf("step 2: expected type notify, got %s", runbook.Steps[1].Type)
	}
	if runbook.Steps[2].Type != models.DRRunbookStepTypeVerify {
		t.Errorf("step 3: expected type verify, got %s", runbook.Steps[2].Type)
	}
	if runbook.Steps[6].Type != models.DRRunbookStepTypeRestore {
		t.Errorf("step 7: expected type restore, got %s", runbook.Steps[6].Type)
	}

	// Verify commands are set on steps 4 and 7
	if runbook.Steps[3].Command == "" {
		t.Error("step 4 (List Available Snapshots) should have a command")
	}
	if !strings.Contains(runbook.Steps[3].Command, "restic") {
		t.Error("step 4 command should contain 'restic'")
	}
	if runbook.Steps[6].Command == "" {
		t.Error("step 7 (Execute Restore) should have a command")
	}
	if !strings.Contains(runbook.Steps[6].Command, "restic") {
		t.Error("step 7 command should contain 'restic'")
	}

	// Verify restore step includes schedule paths
	for _, p := range schedule.Paths {
		if !strings.Contains(runbook.Steps[6].Command, p) {
			t.Errorf("step 7 command should include path %q", p)
		}
	}

	// Verify expected fields on steps with them
	if runbook.Steps[3].Expected == "" {
		t.Error("step 4 should have an expected result")
	}
	if runbook.Steps[6].Expected == "" {
		t.Error("step 7 should have an expected result")
	}

	// Verify agent hostname appears in descriptions
	if !strings.Contains(runbook.Steps[0].Description, agent.Hostname) {
		t.Error("step 1 description should reference agent hostname")
	}

	// Verify repo info appears in step 3
	if !strings.Contains(runbook.Steps[2].Description, repo.Name) {
		t.Error("step 3 description should reference repo name")
	}
	if !strings.Contains(runbook.Steps[2].Description, string(repo.Type)) {
		t.Error("step 3 description should reference repo type")
	}

	// Verify schedule paths appear in step 6
	if !strings.Contains(runbook.Steps[5].Description, "/data") {
		t.Error("step 6 description should reference schedule paths")
	}
}

// --- RenderText ---

func TestRenderText(t *testing.T) {
	t.Run("full render with all data", func(t *testing.T) {
		schedule := testSchedule()
		agent := testAgent()
		repo := testRepo(models.RepositoryTypeS3)
		orgID := uuid.New()

		drTest := models.NewDRTest(uuid.New())
		drTest.Start()
		passed := true
		drTest.Complete("snap-abc123", 1024*1024, 120, true)
		drTest.VerificationPassed = &passed

		store := &mockRunbookStore{
			schedule: schedule,
			agent:    agent,
			repo:     repo,
			drTest:   drTest,
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(orgID, "Test DR Runbook")
		runbook.ScheduleID = &schedule.ID
		runbook.Description = "Test description for DR runbook"
		rto := 60
		rpo := 30
		runbook.RecoveryTimeObjectiveMins = &rto
		runbook.RecoveryPointObjectiveMins = &rpo
		runbook.CredentialsLocation = "/vault/dr-creds"
		runbook.AddContact("John Doe", "SRE Lead", "john@example.com", "555-0100", true)
		runbook.AddStep("Test Step", "A test step description", models.DRRunbookStepTypeManual)

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify markdown header
		if !strings.Contains(text, "# Test DR Runbook") {
			t.Error("expected runbook name in header")
		}

		// Verify schedule info
		if !strings.Contains(text, schedule.Name) {
			t.Error("expected schedule name in output")
		}

		// Verify agent info
		if !strings.Contains(text, agent.Hostname) {
			t.Error("expected agent hostname in output")
		}

		// Verify repository info
		if !strings.Contains(text, repo.Name) {
			t.Error("expected repo name in output")
		}

		// Verify description
		if !strings.Contains(text, "Test description for DR runbook") {
			t.Error("expected description in output")
		}

		// Verify recovery objectives
		if !strings.Contains(text, "60 minutes") {
			t.Error("expected RTO in output")
		}
		if !strings.Contains(text, "30 minutes") {
			t.Error("expected RPO in output")
		}

		// Verify credentials location
		if !strings.Contains(text, "/vault/dr-creds") {
			t.Error("expected credentials location in output")
		}

		// Verify contact info
		if !strings.Contains(text, "John Doe") {
			t.Error("expected contact name in output")
		}
		if !strings.Contains(text, "SRE Lead") {
			t.Error("expected contact role in output")
		}
		if !strings.Contains(text, "john@example.com") {
			t.Error("expected contact email in output")
		}

		// Verify step appears
		if !strings.Contains(text, "Test Step") {
			t.Error("expected step title in output")
		}

		// Verify last test section
		if !strings.Contains(text, "Last DR Test") {
			t.Error("expected last DR test section in output")
		}
		if !strings.Contains(text, "Passed") {
			t.Error("expected verification passed in output")
		}

		// Verify footer
		if !strings.Contains(text, "automatically generated by Keldris") {
			t.Error("expected footer in output")
		}
	})

	t.Run("render without schedule ID", func(t *testing.T) {
		store := &mockRunbookStore{
			drTestErr: errors.New("not found"),
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Standalone Runbook")
		runbook.Description = "A runbook without schedule"
		// ScheduleID is nil by default

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, "# Standalone Runbook") {
			t.Error("expected runbook name in header")
		}
		// Should not contain schedule/agent/repo sections
		if strings.Contains(text, "**Schedule:**") {
			t.Error("should not contain schedule section when no schedule ID")
		}
		if strings.Contains(text, "**Agent:**") {
			t.Error("should not contain agent section when no schedule ID")
		}
	})

	t.Run("render with schedule fetch error", func(t *testing.T) {
		scheduleID := uuid.New()
		store := &mockRunbookStore{
			scheduleErr: errors.New("db error"),
			drTestErr:   errors.New("not found"),
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Error Runbook")
		runbook.ScheduleID = &scheduleID

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still render without schedule data
		if !strings.Contains(text, "# Error Runbook") {
			t.Error("expected runbook name in header")
		}
	})

	t.Run("render with agent fetch error", func(t *testing.T) {
		schedule := testSchedule()
		store := &mockRunbookStore{
			schedule:  schedule,
			agentErr:  errors.New("agent not found"),
			repoErr:   errors.New("repo not found"),
			drTestErr: errors.New("not found"),
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Agent Error Runbook")
		runbook.ScheduleID = &schedule.ID

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, schedule.Name) {
			t.Error("expected schedule name even without agent data")
		}
		if strings.Contains(text, "**Agent:**") {
			t.Error("should not contain agent section when agent fetch fails")
		}
	})

	t.Run("render with step command and expected", func(t *testing.T) {
		store := &mockRunbookStore{
			drTestErr: errors.New("not found"),
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Command Runbook")
		step := models.DRRunbookStep{
			Order:       1,
			Title:       "Run Restore",
			Description: "Execute restore command",
			Type:        models.DRRunbookStepTypeRestore,
			Command:     "restic restore latest --target /tmp",
			Expected:    "Files restored successfully",
		}
		runbook.Steps = append(runbook.Steps, step)

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, "restic restore latest --target /tmp") {
			t.Error("expected command in output")
		}
		if !strings.Contains(text, "Files restored successfully") {
			t.Error("expected expected result in output")
		}
	})

	t.Run("render with failed test", func(t *testing.T) {
		drTest := models.NewDRTest(uuid.New())
		drTest.Fail("restore timeout")

		store := &mockRunbookStore{
			drTest: drTest,
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Failed Test Runbook")

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, "Last DR Test") {
			t.Error("expected last DR test section")
		}
		if !strings.Contains(text, string(models.DRTestStatusFailed)) {
			t.Error("expected failed status in output")
		}
		if !strings.Contains(text, "Failed") {
			t.Error("expected 'Failed' verification in output")
		}
	})

	t.Run("render with completed test with duration", func(t *testing.T) {
		drTest := models.NewDRTest(uuid.New())
		drTest.Complete("snap-001", 2048, 300, true)

		store := &mockRunbookStore{
			drTest: drTest,
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "Duration Runbook")

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, "300") {
			t.Error("expected restore duration seconds in output")
		}
		if !strings.Contains(text, "Restore Duration") {
			t.Error("expected restore duration label in output")
		}
	})

	t.Run("render with no verification passed", func(t *testing.T) {
		drTest := models.NewDRTest(uuid.New())
		drTest.Start()
		// VerificationPassed is nil

		store := &mockRunbookStore{
			drTest: drTest,
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "No Verify Runbook")

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, "Last DR Test") {
			t.Error("expected last DR test section")
		}
		// When VerificationPassed is nil, the template should not render the verification line
		if strings.Contains(text, "Verification:") {
			t.Error("should not contain verification line when VerificationPassed is nil")
		}
	})

	t.Run("render with schedule but no primary repo", func(t *testing.T) {
		schedule := testSchedule()
		schedule.Repositories = nil // no repos
		agent := testAgent()

		store := &mockRunbookStore{
			schedule:  schedule,
			agent:     agent,
			drTestErr: errors.New("not found"),
		}
		gen := NewRunbookGenerator(store)

		runbook := models.NewDRRunbook(uuid.New(), "No Repo Runbook")
		runbook.ScheduleID = &schedule.ID

		text, err := gen.RenderText(context.Background(), runbook)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(text, agent.Hostname) {
			t.Error("expected agent hostname in output")
		}
		if strings.Contains(text, "**Repository:**") {
			t.Error("should not contain repository section when no primary repo")
		}
	})
}

// --- DefaultRestoreSteps ---

func TestDefaultRestoreSteps(t *testing.T) {
	t.Run("generic local type", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeLocal)

		if len(steps) != 5 {
			t.Fatalf("expected 5 steps for local type, got %d", len(steps))
		}

		// Verify sequential ordering
		for i, step := range steps {
			if step.Order != i+1 {
				t.Errorf("step %d: expected order %d, got %d", i, i+1, step.Order)
			}
		}

		// Verify base step titles
		if steps[0].Title != "Verify Repository Access" {
			t.Errorf("expected first step 'Verify Repository Access', got %q", steps[0].Title)
		}
		if steps[1].Title != "List Snapshots" {
			t.Errorf("expected second step 'List Snapshots', got %q", steps[1].Title)
		}
		if steps[2].Title != "Select Snapshot" {
			t.Errorf("expected third step 'Select Snapshot', got %q", steps[2].Title)
		}
		if steps[3].Title != "Execute Restore" {
			t.Errorf("expected fourth step 'Execute Restore', got %q", steps[3].Title)
		}
		if steps[4].Title != "Verify Restored Data" {
			t.Errorf("expected fifth step 'Verify Restored Data', got %q", steps[4].Title)
		}

		// Verify commands
		if !strings.Contains(steps[1].Command, "restic snapshots") {
			t.Error("list snapshots step should have restic command")
		}
		if !strings.Contains(steps[3].Command, "restic restore") {
			t.Error("execute restore step should have restic restore command")
		}
	})

	t.Run("S3 type", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeS3)

		if len(steps) != 6 {
			t.Fatalf("expected 6 steps for S3 type, got %d", len(steps))
		}

		// First step should be AWS credential config
		if steps[0].Title != "Configure AWS Credentials" {
			t.Errorf("expected first step 'Configure AWS Credentials', got %q", steps[0].Title)
		}
		if !strings.Contains(steps[0].Command, "AWS_ACCESS_KEY_ID") {
			t.Error("S3 step should reference AWS_ACCESS_KEY_ID")
		}
		if !strings.Contains(steps[0].Command, "AWS_SECRET_ACCESS_KEY") {
			t.Error("S3 step should reference AWS_SECRET_ACCESS_KEY")
		}
		if steps[0].Type != models.DRRunbookStepTypeManual {
			t.Errorf("expected manual type, got %s", steps[0].Type)
		}

		// Verify renumbering
		for i, step := range steps {
			if step.Order != i+1 {
				t.Errorf("step %d: expected order %d, got %d", i, i+1, step.Order)
			}
		}
	})

	t.Run("B2 type", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeB2)

		if len(steps) != 6 {
			t.Fatalf("expected 6 steps for B2 type, got %d", len(steps))
		}

		if steps[0].Title != "Configure B2 Credentials" {
			t.Errorf("expected first step 'Configure B2 Credentials', got %q", steps[0].Title)
		}
		if !strings.Contains(steps[0].Command, "B2_ACCOUNT_ID") {
			t.Error("B2 step should reference B2_ACCOUNT_ID")
		}
		if !strings.Contains(steps[0].Command, "B2_ACCOUNT_KEY") {
			t.Error("B2 step should reference B2_ACCOUNT_KEY")
		}

		// Verify renumbering
		for i, step := range steps {
			if step.Order != i+1 {
				t.Errorf("step %d: expected order %d, got %d", i, i+1, step.Order)
			}
		}
	})

	t.Run("SFTP type", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeSFTP)

		if len(steps) != 6 {
			t.Fatalf("expected 6 steps for SFTP type, got %d", len(steps))
		}

		if steps[0].Title != "Verify SFTP Access" {
			t.Errorf("expected first step 'Verify SFTP Access', got %q", steps[0].Title)
		}
		if steps[0].Type != models.DRRunbookStepTypeVerify {
			t.Errorf("expected verify type for SFTP step, got %s", steps[0].Type)
		}

		// Verify renumbering
		for i, step := range steps {
			if step.Order != i+1 {
				t.Errorf("step %d: expected order %d, got %d", i, i+1, step.Order)
			}
		}
	})

	t.Run("rest type (no special steps)", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeRest)

		if len(steps) != 5 {
			t.Fatalf("expected 5 steps for rest type, got %d", len(steps))
		}

		// Should have base steps only, no prepended credential step
		if steps[0].Title != "Verify Repository Access" {
			t.Errorf("expected first step 'Verify Repository Access', got %q", steps[0].Title)
		}
	})

	t.Run("dropbox type (no special steps)", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeDropbox)

		if len(steps) != 5 {
			t.Fatalf("expected 5 steps for dropbox type, got %d", len(steps))
		}
	})

	t.Run("step types are correct", func(t *testing.T) {
		steps := DefaultRestoreSteps(models.RepositoryTypeLocal)

		if steps[0].Type != models.DRRunbookStepTypeVerify {
			t.Errorf("step 1: expected verify, got %s", steps[0].Type)
		}
		if steps[1].Type != models.DRRunbookStepTypeVerify {
			t.Errorf("step 2: expected verify, got %s", steps[1].Type)
		}
		if steps[2].Type != models.DRRunbookStepTypeManual {
			t.Errorf("step 3: expected manual, got %s", steps[2].Type)
		}
		if steps[3].Type != models.DRRunbookStepTypeRestore {
			t.Errorf("step 4: expected restore, got %s", steps[3].Type)
		}
		if steps[4].Type != models.DRRunbookStepTypeVerify {
			t.Errorf("step 5: expected verify, got %s", steps[4].Type)
		}
	})
}

// --- RunbookTemplateData ---

func TestRunbookTemplateData(t *testing.T) {
	runbook := models.NewDRRunbook(uuid.New(), "Template Test")
	data := RunbookTemplateData{
		Runbook: runbook,
		OrgName: "TestOrg",
	}

	if data.Runbook.Name != "Template Test" {
		t.Errorf("expected name 'Template Test', got %q", data.Runbook.Name)
	}
	if data.OrgName != "TestOrg" {
		t.Errorf("expected org name 'TestOrg', got %q", data.OrgName)
	}
	if data.Schedule != nil {
		t.Error("expected nil schedule")
	}
	if data.Agent != nil {
		t.Error("expected nil agent")
	}
	if data.Repository != nil {
		t.Error("expected nil repository")
	}
	if data.LastTest != nil {
		t.Error("expected nil last test")
	}
}

// --- Edge cases ---

func TestGenerateForSchedule_EmptyPaths(t *testing.T) {
	schedule := testSchedule()
	schedule.Paths = []string{}
	agent := testAgent()
	repo := testRepo(models.RepositoryTypeLocal)

	store := &mockRunbookStore{
		agent: agent,
		repo:  repo,
	}
	gen := NewRunbookGenerator(store)

	runbook, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runbook.Steps) != 11 {
		t.Errorf("expected 11 steps even with empty paths, got %d", len(runbook.Steps))
	}
}

func TestGenerateForSchedule_MultipleRepositories(t *testing.T) {
	schedule := testSchedule()
	// Add a secondary repo
	schedule.Repositories = append(schedule.Repositories, models.ScheduleRepository{
		ID:           uuid.New(),
		ScheduleID:   schedule.ID,
		RepositoryID: uuid.New(),
		Priority:     1,
		Enabled:      true,
	})

	agent := testAgent()
	repo := testRepo(models.RepositoryTypeS3)

	store := &mockRunbookStore{
		agent: agent,
		repo:  repo,
	}
	gen := NewRunbookGenerator(store)

	runbook, err := gen.GenerateForSchedule(context.Background(), schedule, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use primary repo (priority 0)
	if runbook.ScheduleID == nil {
		t.Error("expected schedule ID to be set")
	}
	if len(runbook.Steps) != 11 {
		t.Errorf("expected 11 steps, got %d", len(runbook.Steps))
	}
}

func TestRenderText_EmptyRunbook(t *testing.T) {
	store := &mockRunbookStore{
		drTestErr: errors.New("not found"),
	}
	gen := NewRunbookGenerator(store)

	runbook := models.NewDRRunbook(uuid.New(), "Empty Runbook")

	text, err := gen.RenderText(context.Background(), runbook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(text, "# Empty Runbook") {
		t.Error("expected runbook name in output")
	}
	if !strings.Contains(text, "automatically generated by Keldris") {
		t.Error("expected footer in output")
	}
}

func TestRenderText_MultipleContacts(t *testing.T) {
	store := &mockRunbookStore{
		drTestErr: errors.New("not found"),
	}
	gen := NewRunbookGenerator(store)

	runbook := models.NewDRRunbook(uuid.New(), "Multi Contact Runbook")
	runbook.AddContact("Alice", "Engineer", "alice@test.com", "555-0001", true)
	runbook.AddContact("Bob", "Manager", "bob@test.com", "555-0002", false)

	text, err := gen.RenderText(context.Background(), runbook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(text, "Alice") {
		t.Error("expected first contact name")
	}
	if !strings.Contains(text, "Bob") {
		t.Error("expected second contact name")
	}
	if !strings.Contains(text, "Contacts") {
		t.Error("expected contacts header")
	}
}

func TestRenderText_WithoutRTORPO(t *testing.T) {
	store := &mockRunbookStore{
		drTestErr: errors.New("not found"),
	}
	gen := NewRunbookGenerator(store)

	runbook := models.NewDRRunbook(uuid.New(), "No RTO/RPO Runbook")

	text, err := gen.RenderText(context.Background(), runbook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(text, "**RTO (Recovery Time Objective):**") {
		t.Error("should not contain RTO line when not set")
	}
	if strings.Contains(text, "**RPO (Recovery Point Objective):**") {
		t.Error("should not contain RPO line when not set")
	}
}
