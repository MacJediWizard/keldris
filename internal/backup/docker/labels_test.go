package docker

import (
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

func TestLabelParser_ParseConfig(t *testing.T) {
	parser := NewLabelParser()
	agentID := uuid.New()

	tests := []struct {
		name           string
		container      ContainerInfo
		expectNil      bool
		expectEnabled  bool
		expectSchedule models.DockerBackupSchedule
		expectExcludes []string
		expectPreHook  string
	}{
		{
			name: "basic enabled container",
			container: ContainerInfo{
				ID:    "abc123",
				Name:  "test-container",
				Image: "postgres:15",
				Labels: map[string]string{
					"keldris.backup": "true",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleDaily,
		},
		{
			name: "disabled container",
			container: ContainerInfo{
				ID:    "abc123",
				Name:  "test-container",
				Image: "postgres:15",
				Labels: map[string]string{
					"keldris.backup": "false",
				},
			},
			expectNil: true,
		},
		{
			name: "container without labels",
			container: ContainerInfo{
				ID:     "abc123",
				Name:   "test-container",
				Image:  "postgres:15",
				Labels: map[string]string{},
			},
			expectNil: true,
		},
		{
			name: "container with full config",
			container: ContainerInfo{
				ID:    "def456",
				Name:  "postgres-db",
				Image: "postgres:15",
				Labels: map[string]string{
					"keldris.backup":          "true",
					"keldris.backup.schedule": "hourly",
					"keldris.backup.exclude":  "/tmp,/var/cache,*.log",
					"keldris.backup.pre-hook": "pg_dump -U postgres mydb > /backup/dump.sql",
					"keldris.backup.volumes":  "true",
					"keldris.backup.stop":     "false",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleHourly,
			expectExcludes: []string{"/tmp", "/var/cache", "*.log"},
			expectPreHook:  "pg_dump -U postgres mydb > /backup/dump.sql",
		},
		{
			name: "container with custom cron",
			container: ContainerInfo{
				ID:    "ghi789",
				Name:  "custom-schedule",
				Image: "redis:7",
				Labels: map[string]string{
					"keldris.backup":      "true",
					"keldris.backup.cron": "0 */4 * * *",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleCustom,
		},
		{
			name: "container with weekly schedule",
			container: ContainerInfo{
				ID:    "weekly123",
				Name:  "weekly-backup",
				Image: "nginx:latest",
				Labels: map[string]string{
					"keldris.backup":          "true",
					"keldris.backup.schedule": "weekly",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleWeekly,
		},
		{
			name: "enabled with yes",
			container: ContainerInfo{
				ID:    "yes123",
				Name:  "yes-container",
				Image: "alpine:latest",
				Labels: map[string]string{
					"keldris.backup": "yes",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleDaily,
		},
		{
			name: "enabled with 1",
			container: ContainerInfo{
				ID:    "one123",
				Name:  "one-container",
				Image: "alpine:latest",
				Labels: map[string]string{
					"keldris.backup": "1",
				},
			},
			expectNil:      false,
			expectEnabled:  true,
			expectSchedule: models.DockerBackupScheduleDaily,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := parser.ParseConfig(agentID, tt.container)

			if tt.expectNil {
				if config != nil {
					t.Errorf("expected nil config, got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("expected non-nil config, got nil")
			}

			if config.Enabled != tt.expectEnabled {
				t.Errorf("expected Enabled=%v, got %v", tt.expectEnabled, config.Enabled)
			}

			if config.Schedule != tt.expectSchedule {
				t.Errorf("expected Schedule=%v, got %v", tt.expectSchedule, config.Schedule)
			}

			if tt.expectExcludes != nil {
				if len(config.Excludes) != len(tt.expectExcludes) {
					t.Errorf("expected %d excludes, got %d", len(tt.expectExcludes), len(config.Excludes))
				} else {
					for i, expected := range tt.expectExcludes {
						if config.Excludes[i] != expected {
							t.Errorf("exclude[%d]: expected %q, got %q", i, expected, config.Excludes[i])
						}
					}
				}
			}

			if tt.expectPreHook != "" && config.PreHook != tt.expectPreHook {
				t.Errorf("expected PreHook=%q, got %q", tt.expectPreHook, config.PreHook)
			}

			if config.AgentID != agentID {
				t.Errorf("expected AgentID=%v, got %v", agentID, config.AgentID)
			}

			if config.ContainerID != tt.container.ID {
				t.Errorf("expected ContainerID=%v, got %v", tt.container.ID, config.ContainerID)
			}
		})
	}
}

func TestLabelParser_ParseRetentionPolicy(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name           string
		labels         map[string]string
		expectDefault  bool
		expectKeepLast int
		expectKeepDaily int
	}{
		{
			name:          "empty labels",
			labels:        map[string]string{},
			expectDefault: true,
		},
		{
			name: "custom retention",
			labels: map[string]string{
				"keldris.backup.retention.keep-last":  "10",
				"keldris.backup.retention.keep-daily": "14",
			},
			expectDefault:   false,
			expectKeepLast:  10,
			expectKeepDaily: 14,
		},
		{
			name: "partial retention",
			labels: map[string]string{
				"keldris.backup.retention.keep-last": "3",
			},
			expectDefault:  false,
			expectKeepLast: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := parser.ParseRetentionPolicy(tt.labels)

			if tt.expectDefault {
				defaultPolicy := models.DefaultRetentionPolicy()
				if policy.KeepLast != defaultPolicy.KeepLast {
					t.Errorf("expected default KeepLast=%d, got %d", defaultPolicy.KeepLast, policy.KeepLast)
				}
				return
			}

			if policy.KeepLast != tt.expectKeepLast {
				t.Errorf("expected KeepLast=%d, got %d", tt.expectKeepLast, policy.KeepLast)
			}

			if tt.expectKeepDaily > 0 && policy.KeepDaily != tt.expectKeepDaily {
				t.Errorf("expected KeepDaily=%d, got %d", tt.expectKeepDaily, policy.KeepDaily)
			}
		})
	}
}

func TestLabelParser_ValidateLabels(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name        string
		labels      map[string]string
		expectValid bool
		expectError string
	}{
		{
			name: "valid labels",
			labels: map[string]string{
				"keldris.backup":          "true",
				"keldris.backup.schedule": "daily",
			},
			expectValid: true,
		},
		{
			name: "invalid schedule",
			labels: map[string]string{
				"keldris.backup":          "true",
				"keldris.backup.schedule": "invalid",
			},
			expectValid: false,
			expectError: "invalid schedule value",
		},
		{
			name: "custom schedule without cron",
			labels: map[string]string{
				"keldris.backup":          "true",
				"keldris.backup.schedule": "custom",
			},
			expectValid: false,
			expectError: "custom schedule requires",
		},
		{
			name: "custom schedule with cron",
			labels: map[string]string{
				"keldris.backup":          "true",
				"keldris.backup.schedule": "custom",
				"keldris.backup.cron":     "0 2 * * *",
			},
			expectValid: true,
		},
		{
			name: "invalid boolean",
			labels: map[string]string{
				"keldris.backup": "maybe",
			},
			expectValid: false,
			expectError: "invalid boolean value",
		},
		{
			name: "invalid integer",
			labels: map[string]string{
				"keldris.backup":                      "true",
				"keldris.backup.retention.keep-last": "not-a-number",
			},
			expectValid: false,
			expectError: "invalid integer value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parser.ValidateLabels(tt.labels)

			if tt.expectValid {
				if len(errors) > 0 {
					t.Errorf("expected valid labels, got errors: %v", errors)
				}
				return
			}

			if len(errors) == 0 {
				t.Error("expected validation errors, got none")
				return
			}

			found := false
			for _, err := range errors {
				if strings.Contains(err, tt.expectError) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got %v", tt.expectError, errors)
			}
		})
	}
}

func TestLabelParser_HasBackupLabel(t *testing.T) {
	parser := NewLabelParser()

	tests := []struct {
		name   string
		labels map[string]string
		expect bool
	}{
		{
			name:   "no labels",
			labels: map[string]string{},
			expect: false,
		},
		{
			name: "other labels",
			labels: map[string]string{
				"app": "test",
			},
			expect: false,
		},
		{
			name: "has backup label",
			labels: map[string]string{
				"keldris.backup": "true",
			},
			expect: true,
		},
		{
			name: "has schedule label only",
			labels: map[string]string{
				"keldris.backup.schedule": "daily",
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.HasBackupLabel(tt.labels)
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestDockerContainerConfig_GetEffectiveCronExpression(t *testing.T) {
	tests := []struct {
		name     string
		schedule models.DockerBackupSchedule
		cron     string
		expect   string
	}{
		{
			name:     "hourly",
			schedule: models.DockerBackupScheduleHourly,
			expect:   "0 0 * * * *",
		},
		{
			name:     "daily",
			schedule: models.DockerBackupScheduleDaily,
			expect:   "0 0 2 * * *",
		},
		{
			name:     "weekly",
			schedule: models.DockerBackupScheduleWeekly,
			expect:   "0 0 2 * * 0",
		},
		{
			name:     "monthly",
			schedule: models.DockerBackupScheduleMonthly,
			expect:   "0 0 2 1 * *",
		},
		{
			name:     "custom with cron",
			schedule: models.DockerBackupScheduleCustom,
			cron:     "0 4 * * *",
			expect:   "0 4 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.DockerContainerConfig{
				Schedule:       tt.schedule,
				CronExpression: tt.cron,
			}
			result := config.GetEffectiveCronExpression()
			if result != tt.expect {
				t.Errorf("expected %q, got %q", tt.expect, result)
			}
		})
	}
}

func TestDockerContainerConfig_ApplyOverrides(t *testing.T) {
	enabled := false
	schedule := models.DockerBackupScheduleHourly
	preHook := "custom-hook"

	config := &models.DockerContainerConfig{
		Enabled:   true,
		Schedule:  models.DockerBackupScheduleDaily,
		PreHook:   "original-hook",
		Overrides: &models.ContainerOverrides{
			Enabled:  &enabled,
			Schedule: &schedule,
			PreHook:  &preHook,
		},
	}

	config.ApplyOverrides()

	if config.Enabled != false {
		t.Errorf("expected Enabled=false after override, got %v", config.Enabled)
	}
	if config.Schedule != models.DockerBackupScheduleHourly {
		t.Errorf("expected Schedule=hourly after override, got %v", config.Schedule)
	}
	if config.PreHook != "custom-hook" {
		t.Errorf("expected PreHook=%q after override, got %q", "custom-hook", config.PreHook)
	}
}

func TestLabelParser_GenerateLabelDocs(t *testing.T) {
	parser := NewLabelParser()
	docs := parser.GenerateLabelDocs()

	if len(docs.Labels) == 0 {
		t.Error("expected non-empty label docs")
	}

	if docs.GeneratedAt.After(time.Now()) {
		t.Error("generated_at should not be in the future")
	}

	// Check for required label
	found := false
	for _, label := range docs.Labels {
		if label.Label == "keldris.backup" {
			found = true
			if !label.Required {
				t.Error("keldris.backup should be marked as required")
			}
			break
		}
	}
	if !found {
		t.Error("expected keldris.backup label in docs")
	}
}

func TestLabelParser_GenerateDockerComposeExample(t *testing.T) {
	parser := NewLabelParser()
	example := parser.GenerateDockerComposeExample()

	if example == "" {
		t.Error("expected non-empty Docker Compose example")
	}

	if !strings.Contains(example, "keldris.backup=true") {
		t.Error("expected Docker Compose example to contain keldris.backup=true")
	}

	if !strings.Contains(example, "postgres") {
		t.Error("expected Docker Compose example to contain postgres service")
	}
}

func TestLabelParser_GenerateDockerRunExample(t *testing.T) {
	parser := NewLabelParser()
	example := parser.GenerateDockerRunExample()

	if example == "" {
		t.Error("expected non-empty docker run example")
	}

	if !strings.Contains(example, "--label") {
		t.Error("expected docker run example to contain --label")
	}

	if !strings.Contains(example, "keldris.backup=true") {
		t.Error("expected docker run example to contain keldris.backup=true")
	}
}

