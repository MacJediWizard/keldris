// Package dr provides disaster recovery runbook generation and management.
package dr

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// RunbookStore defines the interface for runbook data access.
type RunbookStore interface {
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetLatestDRTestByRunbookID(ctx context.Context, runbookID uuid.UUID) (*models.DRTest, error)
}

// RunbookGenerator generates DR runbooks for schedules.
type RunbookGenerator struct {
	store RunbookStore
}

// NewRunbookGenerator creates a new runbook generator.
func NewRunbookGenerator(store RunbookStore) *RunbookGenerator {
	return &RunbookGenerator{store: store}
}

// GenerateForSchedule creates a DR runbook for a backup schedule.
func (g *RunbookGenerator) GenerateForSchedule(ctx context.Context, schedule *models.Schedule, orgID uuid.UUID) (*models.DRRunbook, error) {
	// Get related entities
	agent, err := g.store.GetAgentByID(ctx, schedule.AgentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	repo, err := g.store.GetRepositoryByID(ctx, schedule.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	runbook := models.NewDRRunbook(orgID, fmt.Sprintf("DR Runbook: %s", schedule.Name))
	runbook.ScheduleID = &schedule.ID
	runbook.Description = fmt.Sprintf(
		"Disaster recovery runbook for %s backup schedule on agent %s to %s repository.",
		schedule.Name, agent.Hostname, repo.Name,
	)

	// Add default steps based on repository type
	g.addDefaultSteps(runbook, schedule, agent, repo)

	return runbook, nil
}

// addDefaultSteps adds default recovery steps based on the configuration.
func (g *RunbookGenerator) addDefaultSteps(runbook *models.DRRunbook, schedule *models.Schedule, agent *models.Agent, repo *models.Repository) {
	// Step 1: Assess the situation
	runbook.AddStep(
		"Assess the Disaster",
		fmt.Sprintf("Determine the scope of data loss or system failure on %s. Document what systems are affected and what data needs to be recovered.", agent.Hostname),
		models.DRRunbookStepTypeManual,
	)

	// Step 2: Notify stakeholders
	runbook.AddStep(
		"Notify Stakeholders",
		"Notify relevant team members and stakeholders about the disaster recovery operation. Ensure all necessary personnel are available.",
		models.DRRunbookStepTypeNotify,
	)

	// Step 3: Verify backup availability
	runbook.AddStep(
		"Verify Backup Repository",
		fmt.Sprintf("Verify access to the %s repository (%s). Ensure credentials are available and the repository is accessible.", repo.Name, repo.Type),
		models.DRRunbookStepTypeVerify,
	)

	// Step 4: List available snapshots
	step := models.DRRunbookStep{
		Order:       4,
		Title:       "List Available Snapshots",
		Description: "List all available snapshots to identify the most recent valid backup.",
		Type:        models.DRRunbookStepTypeVerify,
		Command:     "restic -r <REPOSITORY_URL> snapshots --json",
		Expected:    "JSON output listing all available snapshots with dates and IDs",
	}
	runbook.Steps = append(runbook.Steps, step)

	// Step 5: Select snapshot
	runbook.AddStep(
		"Select Recovery Point",
		"Select the appropriate snapshot to restore based on the recovery point objective (RPO) and the timestamp of data loss.",
		models.DRRunbookStepTypeManual,
	)

	// Step 6: Prepare target system
	runbook.AddStep(
		"Prepare Target System",
		fmt.Sprintf("Ensure the target system is ready for restoration. Original paths: %v", schedule.Paths),
		models.DRRunbookStepTypeManual,
	)

	// Step 7: Perform restore
	pathsStr := ""
	for _, p := range schedule.Paths {
		pathsStr += p + " "
	}
	restoreStep := models.DRRunbookStep{
		Order:       7,
		Title:       "Execute Restore",
		Description: "Execute the restore operation to recover data from the selected snapshot.",
		Type:        models.DRRunbookStepTypeRestore,
		Command:     fmt.Sprintf("restic -r <REPOSITORY_URL> restore <SNAPSHOT_ID> --target /restore --include %s", pathsStr),
		Expected:    "Successful restore completion with file count and size",
	}
	runbook.Steps = append(runbook.Steps, restoreStep)

	// Step 8: Verify restored data
	runbook.AddStep(
		"Verify Restored Data",
		"Verify the integrity and completeness of the restored data. Check file permissions, ownership, and compare file counts with the backup manifest.",
		models.DRRunbookStepTypeVerify,
	)

	// Step 9: Update services
	runbook.AddStep(
		"Update Application Configuration",
		"Update any application configurations to point to the restored data. Restart necessary services.",
		models.DRRunbookStepTypeManual,
	)

	// Step 10: Validate recovery
	runbook.AddStep(
		"Validate System Functionality",
		"Perform functional testing to ensure all systems are operating correctly with the restored data.",
		models.DRRunbookStepTypeVerify,
	)

	// Step 11: Document and close
	runbook.AddStep(
		"Document Recovery",
		"Document the recovery process, any issues encountered, and lessons learned. Update the runbook if necessary.",
		models.DRRunbookStepTypeManual,
	)
}

// RunbookTemplateData contains data for runbook template rendering.
type RunbookTemplateData struct {
	Runbook      *models.DRRunbook
	Schedule     *models.Schedule
	Agent        *models.Agent
	Repository   *models.Repository
	LastTest     *models.DRTest
	GeneratedAt  time.Time
	OrgName      string
}

// runbookTextTemplate is the template for text/markdown runbook output.
const runbookTextTemplate = `# {{ .Runbook.Name }}

**Generated:** {{ .GeneratedAt.Format "2006-01-02 15:04:05 MST" }}
{{- if .Schedule }}
**Schedule:** {{ .Schedule.Name }}
**Backup Paths:** {{ range .Schedule.Paths }}{{ . }} {{ end }}
{{- end }}
{{- if .Agent }}
**Agent:** {{ .Agent.Hostname }}
{{- end }}
{{- if .Repository }}
**Repository:** {{ .Repository.Name }} ({{ .Repository.Type }})
{{- end }}

## Description

{{ .Runbook.Description }}

## Recovery Objectives

{{- if .Runbook.RecoveryTimeObjectiveMins }}
- **RTO (Recovery Time Objective):** {{ .Runbook.RecoveryTimeObjectiveMins }} minutes
{{- end }}
{{- if .Runbook.RecoveryPointObjectiveMins }}
- **RPO (Recovery Point Objective):** {{ .Runbook.RecoveryPointObjectiveMins }} minutes
{{- end }}

{{- if .Runbook.CredentialsLocation }}

## Credentials

Repository credentials are stored at: {{ .Runbook.CredentialsLocation }}
{{- end }}

{{- if .Runbook.Contacts }}

## Contacts

| Name | Role | Email | Phone |
|------|------|-------|-------|
{{- range .Runbook.Contacts }}
| {{ .Name }} | {{ .Role }} | {{ .Email }} | {{ .Phone }} |
{{- end }}
{{- end }}

## Recovery Steps

{{- range .Runbook.Steps }}

### Step {{ .Order }}: {{ .Title }}

**Type:** {{ .Type }}

{{ .Description }}

{{- if .Command }}

**Command:**
` + "```" + `
{{ .Command }}
` + "```" + `
{{- end }}

{{- if .Expected }}
**Expected Result:** {{ .Expected }}
{{- end }}

{{- end }}

{{- if .LastTest }}

## Last DR Test

- **Date:** {{ .LastTest.CreatedAt.Format "2006-01-02 15:04:05 MST" }}
- **Status:** {{ .LastTest.Status }}
{{- if .LastTest.VerificationPassed }}
- **Verification:** {{ if (eq (deref .LastTest.VerificationPassed) true) }}Passed{{ else }}Failed{{ end }}
{{- end }}
{{- if .LastTest.RestoreDurationSeconds }}
- **Restore Duration:** {{ .LastTest.RestoreDurationSeconds }} seconds
{{- end }}
{{- end }}

---

*This runbook was automatically generated by Keldris. Review and customize as needed.*
`

// RenderText renders the runbook as markdown text.
func (g *RunbookGenerator) RenderText(ctx context.Context, runbook *models.DRRunbook) (string, error) {
	data := RunbookTemplateData{
		Runbook:     runbook,
		GeneratedAt: time.Now(),
	}

	// Load related data if available
	if runbook.ScheduleID != nil {
		schedule, err := g.store.GetScheduleByID(ctx, *runbook.ScheduleID)
		if err == nil {
			data.Schedule = schedule

			agent, err := g.store.GetAgentByID(ctx, schedule.AgentID)
			if err == nil {
				data.Agent = agent
			}

			repo, err := g.store.GetRepositoryByID(ctx, schedule.RepositoryID)
			if err == nil {
				data.Repository = repo
			}
		}
	}

	// Get last test
	lastTest, err := g.store.GetLatestDRTestByRunbookID(ctx, runbook.ID)
	if err == nil {
		data.LastTest = lastTest
	}

	funcMap := template.FuncMap{
		"deref": func(b *bool) bool {
			if b == nil {
				return false
			}
			return *b
		},
	}

	tmpl, err := template.New("runbook").Funcs(funcMap).Parse(runbookTextTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// DefaultRestoreSteps returns default restore steps for a given repository type.
func DefaultRestoreSteps(repoType models.RepositoryType) []models.DRRunbookStep {
	steps := []models.DRRunbookStep{
		{
			Order:       1,
			Title:       "Verify Repository Access",
			Description: "Ensure you have access to the backup repository and credentials.",
			Type:        models.DRRunbookStepTypeVerify,
		},
		{
			Order:       2,
			Title:       "List Snapshots",
			Description: "List available snapshots to identify the recovery point.",
			Type:        models.DRRunbookStepTypeVerify,
			Command:     "restic snapshots",
		},
		{
			Order:       3,
			Title:       "Select Snapshot",
			Description: "Choose the appropriate snapshot based on the desired recovery point.",
			Type:        models.DRRunbookStepTypeManual,
		},
		{
			Order:       4,
			Title:       "Execute Restore",
			Description: "Run the restore command to recover data.",
			Type:        models.DRRunbookStepTypeRestore,
			Command:     "restic restore <snapshot-id> --target /path/to/restore",
		},
		{
			Order:       5,
			Title:       "Verify Restored Data",
			Description: "Check that all expected files are present and accessible.",
			Type:        models.DRRunbookStepTypeVerify,
		},
	}

	// Add type-specific steps
	switch repoType {
	case models.RepositoryTypeS3:
		steps = append([]models.DRRunbookStep{
			{
				Order:       0,
				Title:       "Configure AWS Credentials",
				Description: "Ensure AWS credentials are configured for S3 access.",
				Type:        models.DRRunbookStepTypeManual,
				Command:     "export AWS_ACCESS_KEY_ID=<key>\nexport AWS_SECRET_ACCESS_KEY=<secret>",
			},
		}, steps...)
	case models.RepositoryTypeB2:
		steps = append([]models.DRRunbookStep{
			{
				Order:       0,
				Title:       "Configure B2 Credentials",
				Description: "Ensure Backblaze B2 credentials are configured.",
				Type:        models.DRRunbookStepTypeManual,
				Command:     "export B2_ACCOUNT_ID=<account>\nexport B2_ACCOUNT_KEY=<key>",
			},
		}, steps...)
	case models.RepositoryTypeSFTP:
		steps = append([]models.DRRunbookStep{
			{
				Order:       0,
				Title:       "Verify SFTP Access",
				Description: "Ensure SSH keys or credentials are available for SFTP access.",
				Type:        models.DRRunbookStepTypeVerify,
			},
		}, steps...)
	}

	// Renumber steps
	for i := range steps {
		steps[i].Order = i + 1
	}

	return steps
}
