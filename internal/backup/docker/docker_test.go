package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

// fakeDockerBinary creates a shell script that outputs the given stdout and exits with the given code.
func fakeDockerBinary(t *testing.T, stdout string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()

	outFile := filepath.Join(dir, "stdout.txt")
	if err := os.WriteFile(outFile, []byte(stdout), 0644); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(dir, "docker")
	content := fmt.Sprintf("#!/bin/sh\ncat '%s'\nexit %d\n", outFile, exitCode)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	return script
}

// fakeResponse holds a response for a fake docker subcommand.
type fakeResponse struct {
	stdout   string
	exitCode int
}

// fakeDockerMulti creates a shell script that returns different outputs based on the first argument.
func fakeDockerMulti(t *testing.T, responses map[string]fakeResponse) string {
	t.Helper()
	dir := t.TempDir()

	var script string
	script = "#!/bin/sh\n"
	for subcmd, resp := range responses {
		outFile := filepath.Join(dir, subcmd+".txt")
		if err := os.WriteFile(outFile, []byte(resp.stdout), 0644); err != nil {
			t.Fatal(err)
		}
		script += fmt.Sprintf("if [ \"$1\" = \"%s\" ]; then\n  cat '%s'\n  exit %d\nfi\n", subcmd, outFile, resp.exitCode)
	}
	script += "exit 0\n"

	scriptPath := filepath.Join(dir, "docker")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	return scriptPath
}

func TestNewDockerClient(t *testing.T) {
	logger := zerolog.Nop()
	client := NewDockerClient(logger)

	if client.binary != "docker" {
		t.Errorf("binary = %q, want %q", client.binary, "docker")
	}
}

func TestNewDockerClientWithBinary(t *testing.T) {
	logger := zerolog.Nop()
	client := NewDockerClientWithBinary("/usr/local/bin/docker", logger)

	if client.binary != "/usr/local/bin/docker" {
		t.Errorf("binary = %q, want %q", client.binary, "/usr/local/bin/docker")
	}
}

func TestDockerClient_ListContainers(t *testing.T) {
	tests := []struct {
		name      string
		stdout    string
		exitCode  int
		wantCount int
		wantErr   bool
	}{
		{
			name: "single container",
			stdout: `{"ID":"abc123","Names":"/myapp","Image":"nginx:latest","State":"running","Status":"Up 2 hours","Labels":"env=prod,team=backend","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`,
			exitCode:  0,
			wantCount: 1,
		},
		{
			name: "multiple containers",
			stdout: `{"ID":"abc123","Names":"/myapp","Image":"nginx:latest","State":"running","Status":"Up 2 hours","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
{"ID":"def456","Names":"/mydb","Image":"postgres:15","State":"exited","Status":"Exited (0) 3 hours ago","Labels":"","Mounts":"","CreatedAt":"2024-06-14 08:00:00 -0700 MST"}
`,
			exitCode:  0,
			wantCount: 2,
		},
		{
			name:      "empty output",
			stdout:    "",
			exitCode:  0,
			wantCount: 0,
		},
		{
			name:      "only whitespace",
			stdout:    "\n\n  \n",
			exitCode:  0,
			wantCount: 0,
		},
		{
			name:     "docker command fails",
			stdout:   "Cannot connect to the Docker daemon",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, tt.stdout, tt.exitCode)
			client := NewDockerClientWithBinary(binary, zerolog.Nop())

			containers, err := client.ListContainers(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListContainers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(containers) != tt.wantCount {
				t.Errorf("ListContainers() returned %d containers, want %d", len(containers), tt.wantCount)
			}
		})
	}
}

func TestDockerClient_ListContainers_Fields(t *testing.T) {
	stdout := `{"ID":"abc123full","Names":"/web-server","Image":"nginx:1.25","State":"running","Status":"Up 5 minutes","Labels":"app=web,env=staging","Mounts":"vol1,vol2","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`
	binary := fakeDockerBinary(t, stdout, 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers() unexpected error: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	c := containers[0]
	if c.ID != "abc123full" {
		t.Errorf("ID = %q, want %q", c.ID, "abc123full")
	}
	if c.Name != "web-server" {
		t.Errorf("Name = %q, want %q", c.Name, "web-server")
	}
	if c.Image != "nginx:1.25" {
		t.Errorf("Image = %q, want %q", c.Image, "nginx:1.25")
	}
	if c.State != "running" {
		t.Errorf("State = %q, want %q", c.State, "running")
	}
	if c.Status != "Up 5 minutes" {
		t.Errorf("Status = %q, want %q", c.Status, "Up 5 minutes")
	}
	if c.Labels["app"] != "web" {
		t.Errorf("Labels[app] = %q, want %q", c.Labels["app"], "web")
	}
	if c.Labels["env"] != "staging" {
		t.Errorf("Labels[env] = %q, want %q", c.Labels["env"], "staging")
	}
}

func TestDockerClient_ListContainers_InvalidJSON(t *testing.T) {
	stdout := `{"ID":"good","Names":"/ok","Image":"nginx","State":"running","Status":"Up","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
not valid json
{"ID":"good2","Names":"/ok2","Image":"redis","State":"running","Status":"Up","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`
	binary := fakeDockerBinary(t, stdout, 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	containers, err := client.ListContainers(context.Background())
	if err != nil {
		t.Fatalf("ListContainers() unexpected error: %v", err)
	}
	// Invalid JSON lines are skipped, valid ones are kept
	if len(containers) != 2 {
		t.Errorf("expected 2 containers (skipping invalid), got %d", len(containers))
	}
}

func TestDockerClient_ListVolumes(t *testing.T) {
	tests := []struct {
		name      string
		stdout    string
		exitCode  int
		wantCount int
		wantErr   bool
	}{
		{
			name: "single volume",
			stdout: `{"Name":"mydata","Driver":"local","Mountpoint":"/var/lib/docker/volumes/mydata/_data","Labels":"","Scope":"local","CreatedAt":"2024-06-15T10:00:00Z"}
`,
			exitCode:  0,
			wantCount: 1,
		},
		{
			name: "multiple volumes",
			stdout: `{"Name":"vol1","Driver":"local","Mountpoint":"/var/lib/docker/volumes/vol1/_data","Labels":"backup=true","Scope":"local","CreatedAt":"2024-06-15T10:00:00Z"}
{"Name":"vol2","Driver":"local","Mountpoint":"/var/lib/docker/volumes/vol2/_data","Labels":"","Scope":"local","CreatedAt":"2024-06-14T08:00:00Z"}
`,
			exitCode:  0,
			wantCount: 2,
		},
		{
			name:      "empty output",
			stdout:    "",
			exitCode:  0,
			wantCount: 0,
		},
		{
			name:     "command fails",
			stdout:   "",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, tt.stdout, tt.exitCode)
			client := NewDockerClientWithBinary(binary, zerolog.Nop())

			volumes, err := client.ListVolumes(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ListVolumes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(volumes) != tt.wantCount {
				t.Errorf("ListVolumes() returned %d volumes, want %d", len(volumes), tt.wantCount)
			}
		})
	}
}

func TestDockerClient_ListVolumes_Fields(t *testing.T) {
	stdout := `{"Name":"pgdata","Driver":"local","Mountpoint":"/var/lib/docker/volumes/pgdata/_data","Labels":"backup=yes,tier=db","Scope":"local","CreatedAt":"2024-06-10T14:30:00Z"}
`
	binary := fakeDockerBinary(t, stdout, 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	volumes, err := client.ListVolumes(context.Background())
	if err != nil {
		t.Fatalf("ListVolumes() unexpected error: %v", err)
	}
	if len(volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(volumes))
	}

	v := volumes[0]
	if v.Name != "pgdata" {
		t.Errorf("Name = %q, want %q", v.Name, "pgdata")
	}
	if v.Driver != "local" {
		t.Errorf("Driver = %q, want %q", v.Driver, "local")
	}
	if v.Mountpoint != "/var/lib/docker/volumes/pgdata/_data" {
		t.Errorf("Mountpoint = %q, want %q", v.Mountpoint, "/var/lib/docker/volumes/pgdata/_data")
	}
	if v.Labels["backup"] != "yes" {
		t.Errorf("Labels[backup] = %q, want %q", v.Labels["backup"], "yes")
	}
	if v.Labels["tier"] != "db" {
		t.Errorf("Labels[tier] = %q, want %q", v.Labels["tier"], "db")
	}
	if v.Scope != "local" {
		t.Errorf("Scope = %q, want %q", v.Scope, "local")
	}
}

func TestDockerClient_ListVolumes_InvalidJSON(t *testing.T) {
	stdout := `{"Name":"good","Driver":"local","Mountpoint":"/data","Labels":"","Scope":"local","CreatedAt":"2024-01-01T00:00:00Z"}
{invalid
`
	binary := fakeDockerBinary(t, stdout, 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	volumes, err := client.ListVolumes(context.Background())
	if err != nil {
		t.Fatalf("ListVolumes() unexpected error: %v", err)
	}
	if len(volumes) != 1 {
		t.Errorf("expected 1 volume (skipping invalid), got %d", len(volumes))
	}
}

func TestDockerClient_InspectContainer(t *testing.T) {
	inspectJSON := `{
		"Id": "abc123def456",
		"Name": "/my-container",
		"Image": "sha256:deadbeef",
		"Created": "2024-06-15T10:30:00.123456789Z",
		"State": {
			"Status": "running",
			"Running": true,
			"Paused": false,
			"StartedAt": "2024-06-15T10:30:01.000000000Z",
			"FinishedAt": "0001-01-01T00:00:00Z",
			"ExitCode": 0
		},
		"Config": {
			"Hostname": "abc123",
			"Env": ["PATH=/usr/bin", "APP_ENV=prod"],
			"Cmd": ["nginx", "-g", "daemon off;"],
			"Labels": {"app": "web"},
			"Image": "nginx:latest"
		},
		"HostConfig": {
			"RestartPolicy": {
				"Name": "always"
			}
		},
		"Mounts": [
			{
				"Type": "volume",
				"Name": "mydata",
				"Source": "/var/lib/docker/volumes/mydata/_data",
				"Destination": "/data",
				"RW": true
			},
			{
				"Type": "bind",
				"Name": "",
				"Source": "/host/config",
				"Destination": "/etc/app",
				"RW": false
			}
		]
	}`

	binary := fakeDockerBinary(t, inspectJSON, 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	info, err := client.InspectContainer(context.Background(), "abc123def456")
	if err != nil {
		t.Fatalf("InspectContainer() unexpected error: %v", err)
	}

	if info.ID != "abc123def456" {
		t.Errorf("ID = %q, want %q", info.ID, "abc123def456")
	}
	if info.Name != "my-container" {
		t.Errorf("Name = %q, want %q", info.Name, "my-container")
	}
	if info.Image != "nginx:latest" {
		t.Errorf("Image = %q, want %q", info.Image, "nginx:latest")
	}
	if !info.State.Running {
		t.Error("State.Running = false, want true")
	}
	if info.State.Status != "running" {
		t.Errorf("State.Status = %q, want %q", info.State.Status, "running")
	}
	if info.State.Paused {
		t.Error("State.Paused = true, want false")
	}
	if info.State.ExitCode != 0 {
		t.Errorf("State.ExitCode = %d, want 0", info.State.ExitCode)
	}
	if info.State.StartedAt == nil {
		t.Error("State.StartedAt = nil, want non-nil")
	}
	if info.Config.Hostname != "abc123" {
		t.Errorf("Config.Hostname = %q, want %q", info.Config.Hostname, "abc123")
	}
	if len(info.Config.Env) != 2 {
		t.Errorf("Config.Env length = %d, want 2", len(info.Config.Env))
	}
	if len(info.Config.Cmd) != 3 {
		t.Errorf("Config.Cmd length = %d, want 3", len(info.Config.Cmd))
	}
	if info.RestartKey != "always" {
		t.Errorf("RestartKey = %q, want %q", info.RestartKey, "always")
	}
	if info.Labels["app"] != "web" {
		t.Errorf("Labels[app] = %q, want %q", info.Labels["app"], "web")
	}
	if len(info.Mounts) != 2 {
		t.Fatalf("Mounts length = %d, want 2", len(info.Mounts))
	}

	// First mount: volume, RW=true -> ReadOnly=false
	if info.Mounts[0].Type != "volume" {
		t.Errorf("Mounts[0].Type = %q, want %q", info.Mounts[0].Type, "volume")
	}
	if info.Mounts[0].Name != "mydata" {
		t.Errorf("Mounts[0].Name = %q, want %q", info.Mounts[0].Name, "mydata")
	}
	if info.Mounts[0].ReadOnly {
		t.Error("Mounts[0].ReadOnly = true, want false (RW=true)")
	}

	// Second mount: bind, RW=false -> ReadOnly=true
	if info.Mounts[1].Type != "bind" {
		t.Errorf("Mounts[1].Type = %q, want %q", info.Mounts[1].Type, "bind")
	}
	if !info.Mounts[1].ReadOnly {
		t.Error("Mounts[1].ReadOnly = false, want true (RW=false)")
	}
}

func TestDockerClient_InspectContainer_Error(t *testing.T) {
	binary := fakeDockerBinary(t, "Error: No such container: bad-id", 1)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	_, err := client.InspectContainer(context.Background(), "bad-id")
	if err == nil {
		t.Error("InspectContainer() expected error for non-existent container")
	}
}

func TestDockerClient_InspectContainer_InvalidJSON(t *testing.T) {
	binary := fakeDockerBinary(t, "not json at all", 0)
	client := NewDockerClientWithBinary(binary, zerolog.Nop())

	_, err := client.InspectContainer(context.Background(), "abc123")
	if err == nil {
		t.Error("InspectContainer() expected error for invalid JSON")
	}
}

func TestDockerClient_PauseContainer(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{
			name:     "success",
			exitCode: 0,
			wantErr:  false,
		},
		{
			name:     "error",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, "", tt.exitCode)
			client := NewDockerClientWithBinary(binary, zerolog.Nop())

			err := client.PauseContainer(context.Background(), "test-id")
			if (err != nil) != tt.wantErr {
				t.Errorf("PauseContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDockerClient_UnpauseContainer(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{
			name:     "success",
			exitCode: 0,
			wantErr:  false,
		},
		{
			name:     "error",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, "", tt.exitCode)
			client := NewDockerClientWithBinary(binary, zerolog.Nop())

			err := client.UnpauseContainer(context.Background(), "test-id")
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpauseContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDockerClient_NotRunning(t *testing.T) {
	// Use a binary that doesn't exist
	client := NewDockerClientWithBinary("/nonexistent/docker-binary", zerolog.Nop())

	t.Run("list containers", func(t *testing.T) {
		_, err := client.ListContainers(context.Background())
		if err == nil {
			t.Error("ListContainers() expected error when binary not found")
		}
	})

	t.Run("list volumes", func(t *testing.T) {
		_, err := client.ListVolumes(context.Background())
		if err == nil {
			t.Error("ListVolumes() expected error when binary not found")
		}
	})

	t.Run("inspect container", func(t *testing.T) {
		_, err := client.InspectContainer(context.Background(), "test")
		if err == nil {
			t.Error("InspectContainer() expected error when binary not found")
		}
	})

	t.Run("pause container", func(t *testing.T) {
		err := client.PauseContainer(context.Background(), "test")
		if err == nil {
			t.Error("PauseContainer() expected error when binary not found")
		}
	})

	t.Run("unpause container", func(t *testing.T) {
		err := client.UnpauseContainer(context.Background(), "test")
		if err == nil {
			t.Error("UnpauseContainer() expected error when binary not found")
		}
	})
}

func TestDockerClient_ContextCanceled(t *testing.T) {
	// Script sleeps for 10s — context will cancel first
	dir := t.TempDir()
	script := filepath.Join(dir, "docker")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 10\n"), 0755); err != nil {
		t.Fatal(err)
	}

	client := NewDockerClientWithBinary(script, zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.ListContainers(ctx)
	if err == nil {
		t.Error("ListContainers() expected error on canceled context")
	}
}

func TestParseLabels(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   map[string]string
		wantNil bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:  "single label",
			input: "app=web",
			want:  map[string]string{"app": "web"},
		},
		{
			name:  "multiple labels",
			input: "app=web,env=prod,team=backend",
			want: map[string]string{
				"app":  "web",
				"env":  "prod",
				"team": "backend",
			},
		},
		{
			name:  "labels with spaces",
			input: " app = web , env = staging ",
			want: map[string]string{
				"app": "web",
				"env": "staging",
			},
		},
		{
			name:  "label value with equals sign",
			input: "config=key=value",
			want:  map[string]string{"config": "key=value"},
		},
		{
			name:  "malformed label without equals",
			input: "noequalssign",
			want:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLabels(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("parseLabels(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("parseLabels(%q) returned %d entries, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseLabels(%q)[%q] = %q, want %q", tt.input, k, got[k], v)
				}
			}
		})
	}
}
