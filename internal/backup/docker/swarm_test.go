package docker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestDefaultSwarmBackupConfig(t *testing.T) {
	config := DefaultSwarmBackupConfig()

	if config.DockerHost != "unix:///var/run/docker.sock" {
		t.Errorf("expected default docker host, got %s", config.DockerHost)
	}

	if !config.IncludeSecrets {
		t.Error("expected IncludeSecrets to be true by default")
	}

	if !config.IncludeConfigs {
		t.Error("expected IncludeConfigs to be true by default")
	}

	if !config.IncludeNetworks {
		t.Error("expected IncludeNetworks to be true by default")
	}

	if !config.IncludeVolumes {
		t.Error("expected IncludeVolumes to be true by default")
	}
}

func TestDefaultRestoreOptions(t *testing.T) {
	opts := DefaultRestoreOptions()

	if opts.DryRun {
		t.Error("expected DryRun to be false by default")
	}

	if opts.Force {
		t.Error("expected Force to be false by default")
	}

	if !opts.IncludeNetworks {
		t.Error("expected IncludeNetworks to be true by default")
	}

	if !opts.IncludeVolumes {
		t.Error("expected IncludeVolumes to be true by default")
	}

	if !opts.IncludeConfigs {
		t.Error("expected IncludeConfigs to be true by default")
	}

	if !opts.RespectDependencies {
		t.Error("expected RespectDependencies to be true by default")
	}
}

func TestNewSwarmBackupManager(t *testing.T) {
	logger := zerolog.Nop()

	// Test with nil config
	mgr := NewSwarmBackupManager(nil, logger)
	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
	if mgr.config == nil {
		t.Error("expected default config to be set")
	}

	// Test with custom config
	config := &SwarmBackupConfig{
		DockerHost:   "tcp://localhost:2375",
		StackFilter:  []string{"mystack"},
	}
	mgr = NewSwarmBackupManager(config, logger)
	if mgr.config.DockerHost != "tcp://localhost:2375" {
		t.Errorf("expected custom docker host, got %s", mgr.config.DockerHost)
	}
}

func TestNewSwarmBackupManagerWithBinary(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultSwarmBackupConfig()

	mgr := NewSwarmBackupManagerWithBinary(config, "/usr/local/bin/docker", logger)
	if mgr.binary != "/usr/local/bin/docker" {
		t.Errorf("expected custom binary path, got %s", mgr.binary)
	}
}

func TestResolveRestoreOrder(t *testing.T) {
	logger := zerolog.Nop()
	mgr := NewSwarmBackupManager(nil, logger)

	tests := []struct {
		name        string
		services    []SwarmService
		respectDeps bool
		wantOrder   []string
		wantErr     bool
	}{
		{
			name: "no dependencies",
			services: []SwarmService{
				{Name: "svc1"},
				{Name: "svc2"},
				{Name: "svc3"},
			},
			respectDeps: true,
			wantOrder:   []string{"svc1", "svc2", "svc3"},
			wantErr:     false,
		},
		{
			name: "simple dependency chain",
			services: []SwarmService{
				{Name: "frontend", DependsOn: []string{"backend"}},
				{Name: "backend", DependsOn: []string{"database"}},
				{Name: "database"},
			},
			respectDeps: true,
			wantOrder:   []string{"database", "backend", "frontend"},
			wantErr:     false,
		},
		{
			name: "multiple dependencies",
			services: []SwarmService{
				{Name: "app", DependsOn: []string{"cache", "db"}},
				{Name: "cache"},
				{Name: "db"},
			},
			respectDeps: true,
			wantOrder:   []string{"cache", "db", "app"},
			wantErr:     false,
		},
		{
			name: "circular dependency",
			services: []SwarmService{
				{Name: "a", DependsOn: []string{"b"}},
				{Name: "b", DependsOn: []string{"c"}},
				{Name: "c", DependsOn: []string{"a"}},
			},
			respectDeps: true,
			wantErr:     true,
		},
		{
			name: "ignore dependencies when disabled",
			services: []SwarmService{
				{Name: "frontend", DependsOn: []string{"backend"}},
				{Name: "backend"},
			},
			respectDeps: false,
			wantOrder:   []string{"frontend", "backend"},
			wantErr:     false,
		},
		{
			name: "dependency on non-existent service",
			services: []SwarmService{
				{Name: "app", DependsOn: []string{"missing"}},
			},
			respectDeps: true,
			wantOrder:   []string{"app"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mgr.resolveRestoreOrder(tt.services, tt.respectDeps)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.wantOrder) {
				t.Errorf("expected %d services, got %d", len(tt.wantOrder), len(result))
				return
			}

			for i, name := range tt.wantOrder {
				if result[i].Name != name {
					t.Errorf("position %d: expected %s, got %s", i, name, result[i].Name)
				}
			}
		})
	}
}

func TestFilterServices(t *testing.T) {
	logger := zerolog.Nop()

	services := []SwarmService{
		{Name: "mystack_web", StackName: "mystack"},
		{Name: "mystack_db", StackName: "mystack"},
		{Name: "other_api", StackName: "other"},
		{Name: "standalone"},
	}

	tests := []struct {
		name          string
		stackFilter   []string
		serviceFilter []string
		wantCount     int
	}{
		{
			name:      "no filter",
			wantCount: 4,
		},
		{
			name:        "filter by stack",
			stackFilter: []string{"mystack"},
			wantCount:   2,
		},
		{
			name:          "filter by service name",
			serviceFilter: []string{"mystack_web", "other_api"},
			wantCount:     2,
		},
		{
			name:          "filter by both",
			stackFilter:   []string{"mystack"},
			serviceFilter: []string{"mystack_web"},
			wantCount:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SwarmBackupConfig{
				StackFilter:   tt.stackFilter,
				ServiceFilter: tt.serviceFilter,
			}
			mgr := NewSwarmBackupManager(config, logger)

			result := mgr.filterServices(services)
			if len(result) != tt.wantCount {
				t.Errorf("expected %d services, got %d", tt.wantCount, len(result))
			}
		})
	}
}

func TestFilterServicesForRestore(t *testing.T) {
	logger := zerolog.Nop()
	mgr := NewSwarmBackupManager(nil, logger)

	services := []SwarmService{
		{Name: "web", StackName: "app"},
		{Name: "api", StackName: "app"},
		{Name: "db", StackName: "data"},
	}

	opts := &RestoreOptions{
		StackFilter: []string{"app"},
	}

	result := mgr.filterServicesForRestore(services, opts)
	if len(result) != 2 {
		t.Errorf("expected 2 services, got %d", len(result))
	}
}

func TestSaveAndLoadBackup(t *testing.T) {
	logger := zerolog.Nop()
	mgr := NewSwarmBackupManager(nil, logger)

	// Create a test backup
	backup := &SwarmBackup{
		Metadata: BackupMetadata{
			ID:           "test-backup-123",
			Timestamp:    time.Now(),
			Version:      "1.0",
			Hostname:     "test-host",
			ServiceCount: 2,
			StackCount:   1,
			NodeCount:    1,
		},
		ClusterState: ClusterState{
			ClusterID:   "cluster-abc",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			BackupTime:  time.Now(),
			Version:     "1.0",
			ManagerNode: "manager1",
		},
		Nodes: []SwarmNode{
			{
				ID:       "node1",
				Hostname: "manager1",
				Role:     NodeRoleManager,
				State:    NodeStateReady,
				IsLeader: true,
			},
		},
		Services: []SwarmService{
			{
				ID:       "svc1",
				Name:     "web",
				Image:    "nginx:latest",
				Mode:     "replicated",
				Replicas: 3,
			},
			{
				ID:        "svc2",
				Name:      "api",
				Image:     "myapp:v1",
				Mode:      "replicated",
				Replicas:  2,
				DependsOn: []string{"web"},
			},
		},
		Stacks: []SwarmStack{
			{
				Name:     "myapp",
				Services: []string{"web", "api"},
			},
		},
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "swarm-backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupPath := filepath.Join(tmpDir, "backup.json")

	// Save backup
	if err := mgr.SaveBackup(backup, backupPath); err != nil {
		t.Fatalf("failed to save backup: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup file was not created")
	}

	// Load backup
	loaded, err := mgr.LoadBackup(backupPath)
	if err != nil {
		t.Fatalf("failed to load backup: %v", err)
	}

	// Verify loaded data
	if loaded.Metadata.ID != backup.Metadata.ID {
		t.Errorf("ID mismatch: expected %s, got %s", backup.Metadata.ID, loaded.Metadata.ID)
	}

	if loaded.Metadata.Version != backup.Metadata.Version {
		t.Errorf("version mismatch: expected %s, got %s", backup.Metadata.Version, loaded.Metadata.Version)
	}

	if len(loaded.Services) != len(backup.Services) {
		t.Errorf("services count mismatch: expected %d, got %d", len(backup.Services), len(loaded.Services))
	}

	if len(loaded.Nodes) != len(backup.Nodes) {
		t.Errorf("nodes count mismatch: expected %d, got %d", len(backup.Nodes), len(loaded.Nodes))
	}
}

func TestLoadBackupInvalidData(t *testing.T) {
	logger := zerolog.Nop()
	mgr := NewSwarmBackupManager(nil, logger)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "swarm-backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with invalid JSON
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = mgr.LoadBackup(invalidPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Test with valid JSON but missing version
	missingVersionPath := filepath.Join(tmpDir, "no-version.json")
	data, _ := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{}})
	if err := os.WriteFile(missingVersionPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = mgr.LoadBackup(missingVersionPath)
	if err != ErrInvalidBackupData {
		t.Errorf("expected ErrInvalidBackupData, got %v", err)
	}
}

func TestSwarmAgentMode(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultSwarmBackupConfig()

	mode := NewSwarmAgentMode(config, logger)
	if mode == nil {
		t.Fatal("expected agent mode to be created")
	}

	if mode.manager == nil {
		t.Error("expected manager to be set")
	}

	if mode.config == nil {
		t.Error("expected config to be set")
	}
}

func TestNodeRoleConstants(t *testing.T) {
	if NodeRoleManager != "manager" {
		t.Errorf("expected NodeRoleManager to be 'manager', got %s", NodeRoleManager)
	}
	if NodeRoleWorker != "worker" {
		t.Errorf("expected NodeRoleWorker to be 'worker', got %s", NodeRoleWorker)
	}
}

func TestNodeStateConstants(t *testing.T) {
	if NodeStateReady != "ready" {
		t.Errorf("expected NodeStateReady to be 'ready', got %s", NodeStateReady)
	}
	if NodeStateDown != "down" {
		t.Errorf("expected NodeStateDown to be 'down', got %s", NodeStateDown)
	}
	if NodeStateDisconnected != "disconnected" {
		t.Errorf("expected NodeStateDisconnected to be 'disconnected', got %s", NodeStateDisconnected)
	}
}

func TestErrorDefinitions(t *testing.T) {
	errors := []error{
		ErrNotSwarmManager,
		ErrSwarmNotActive,
		ErrServiceNotFound,
		ErrStackNotFound,
		ErrBackupFailed,
		ErrRestoreFailed,
		ErrDependencyCycle,
		ErrInvalidBackupData,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("error should not be nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

func TestServicePortJSON(t *testing.T) {
	port := ServicePort{
		Protocol:      "tcp",
		TargetPort:    80,
		PublishedPort: 8080,
		PublishMode:   "ingress",
	}

	data, err := json.Marshal(port)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ServicePort
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Protocol != port.Protocol {
		t.Errorf("protocol mismatch")
	}
	if decoded.TargetPort != port.TargetPort {
		t.Errorf("target port mismatch")
	}
	if decoded.PublishedPort != port.PublishedPort {
		t.Errorf("published port mismatch")
	}
}

func TestRestoreResultSuccess(t *testing.T) {
	result := &RestoreResult{
		Success:          true,
		ServicesRestored: []string{"web", "api", "db"},
		Duration:         5 * time.Second,
	}

	if !result.Success {
		t.Error("expected success to be true")
	}

	if len(result.ServicesRestored) != 3 {
		t.Errorf("expected 3 services restored, got %d", len(result.ServicesRestored))
	}
}

func TestFilterStacks(t *testing.T) {
	logger := zerolog.Nop()

	stacks := []SwarmStack{
		{Name: "app1"},
		{Name: "app2"},
		{Name: "monitoring"},
	}

	tests := []struct {
		name        string
		stackFilter []string
		wantCount   int
	}{
		{
			name:      "no filter",
			wantCount: 3,
		},
		{
			name:        "single stack",
			stackFilter: []string{"app1"},
			wantCount:   1,
		},
		{
			name:        "multiple stacks",
			stackFilter: []string{"app1", "monitoring"},
			wantCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SwarmBackupConfig{
				StackFilter: tt.stackFilter,
			}
			mgr := NewSwarmBackupManager(config, logger)

			result := mgr.filterStacks(stacks)
			if len(result) != tt.wantCount {
				t.Errorf("expected %d stacks, got %d", tt.wantCount, len(result))
			}
		})
	}
}

// BenchmarkResolveRestoreOrder benchmarks the dependency resolution algorithm.
func BenchmarkResolveRestoreOrder(b *testing.B) {
	logger := zerolog.Nop()
	mgr := NewSwarmBackupManager(nil, logger)

	// Create a chain of 100 services with dependencies
	services := make([]SwarmService, 100)
	for i := 0; i < 100; i++ {
		services[i] = SwarmService{
			Name: "service" + string(rune('0'+i)),
		}
		if i > 0 {
			services[i].DependsOn = []string{services[i-1].Name}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.resolveRestoreOrder(services, true)
	}
}

// Integration test helper - skipped unless DOCKER_TEST is set
func TestSwarmBackupIntegration(t *testing.T) {
	if os.Getenv("DOCKER_TEST") == "" {
		t.Skip("skipping integration test; set DOCKER_TEST=1 to run")
	}

	ctx := context.Background()
	logger := zerolog.New(os.Stdout)
	mgr := NewSwarmBackupManager(nil, logger)

	// Check if we're on a swarm manager
	isManager, err := mgr.IsSwarmManager(ctx)
	if err != nil {
		t.Fatalf("failed to check swarm status: %v", err)
	}

	if !isManager {
		t.Skip("not running on a swarm manager node")
	}

	// Perform backup
	backup, err := mgr.Backup(ctx)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	t.Logf("Backup completed: %d services, %d stacks, %d nodes",
		backup.Metadata.ServiceCount,
		backup.Metadata.StackCount,
		backup.Metadata.NodeCount)
}
