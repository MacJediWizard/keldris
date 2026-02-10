package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// withFakeDockerInPath creates a fake "docker" script in a temp directory and
// prepends it to PATH. Returns a cleanup function that restores the original PATH.
// NOTE: Not safe for parallel tests since PATH is global state.
func withFakeDockerInPath(t *testing.T, stdout string, exitCode int) func() {
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

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+oldPath)

	return func() {
		os.Setenv("PATH", oldPath)
	}
}

// withFakeDockerMultiInPath creates a fake "docker" in PATH that returns different
// outputs based on the subcommand (first argument).
func withFakeDockerMultiInPath(t *testing.T, responses map[string]fakeResponse) func() {
	t.Helper()
	dir := t.TempDir()

	var scriptContent string
	scriptContent = "#!/bin/sh\n"
	for subcmd, resp := range responses {
		outFile := filepath.Join(dir, subcmd+".txt")
		if err := os.WriteFile(outFile, []byte(resp.stdout), 0644); err != nil {
			t.Fatal(err)
		}
		scriptContent += fmt.Sprintf("if [ \"$1\" = \"%s\" ]; then\n  cat '%s'\n  exit %d\nfi\n", subcmd, outFile, resp.exitCode)
	}
	scriptContent += "exit 1\n"

	script := filepath.Join(dir, "docker")
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+oldPath)

	return func() {
		os.Setenv("PATH", oldPath)
	}
}

func TestDetectDocker_Available(t *testing.T) {
	cleanup := withFakeDockerInPath(t, "24.0.7", 0)
	defer cleanup()

	available, version := DetectDocker()
	if !available {
		t.Error("DetectDocker() available = false, want true")
	}
	if version != "24.0.7" {
		t.Errorf("DetectDocker() version = %q, want %q", version, "24.0.7")
	}
}

func TestDetectDocker_VersionWithWhitespace(t *testing.T) {
	cleanup := withFakeDockerInPath(t, "  25.0.1\n", 0)
	defer cleanup()

	available, version := DetectDocker()
	if !available {
		t.Error("DetectDocker() available = false, want true")
	}
	if version != "25.0.1" {
		t.Errorf("DetectDocker() version = %q, want %q", version, "25.0.1")
	}
}

func TestDetectDocker_NotInstalled(t *testing.T) {
	// Set PATH to an empty temp dir so "docker" can't be found
	dir := t.TempDir()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", oldPath)

	available, version := DetectDocker()
	if available {
		t.Error("DetectDocker() available = true, want false")
	}
	if version != "" {
		t.Errorf("DetectDocker() version = %q, want empty", version)
	}
}

func TestDetectDocker_NotRunning(t *testing.T) {
	cleanup := withFakeDockerInPath(t, "Cannot connect to the Docker daemon", 1)
	defer cleanup()

	available, version := DetectDocker()
	if available {
		t.Error("DetectDocker() available = true, want false when daemon not running")
	}
	if version != "" {
		t.Errorf("DetectDocker() version = %q, want empty", version)
	}
}

func TestDetectDocker_EmptyVersion(t *testing.T) {
	cleanup := withFakeDockerInPath(t, "", 0)
	defer cleanup()

	available, version := DetectDocker()
	if available {
		t.Error("DetectDocker() available = true, want false when version is empty")
	}
	if version != "" {
		t.Errorf("DetectDocker() version = %q, want empty", version)
	}
}

func TestIsDockerRunning_True(t *testing.T) {
	cleanup := withFakeDockerMultiInPath(t, map[string]fakeResponse{
		"info": {stdout: "24.0.7", exitCode: 0},
	})
	defer cleanup()

	if !IsDockerRunning() {
		t.Error("IsDockerRunning() = false, want true")
	}
}

func TestIsDockerRunning_False(t *testing.T) {
	cleanup := withFakeDockerMultiInPath(t, map[string]fakeResponse{
		"info": {stdout: "", exitCode: 1},
	})
	defer cleanup()

	if IsDockerRunning() {
		t.Error("IsDockerRunning() = true, want false when daemon not running")
	}
}

func TestIsDockerRunning_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", oldPath)

	if IsDockerRunning() {
		t.Error("IsDockerRunning() = true, want false when docker not installed")
	}
}

func TestGetDockerInfo(t *testing.T) {
	infoJSON := `{
		"ServerVersion": "24.0.7",
		"Driver": "overlay2",
		"DockerRootDir": "/var/lib/docker",
		"Containers": 10,
		"ContainersRunning": 5,
		"ContainersPaused": 1,
		"ContainersStopped": 4,
		"Images": 25,
		"OperatingSystem": "Ubuntu 22.04.3 LTS",
		"Architecture": "x86_64"
	}`

	cleanup := withFakeDockerMultiInPath(t, map[string]fakeResponse{
		"info": {stdout: infoJSON, exitCode: 0},
	})
	defer cleanup()

	info, err := GetDockerInfo()
	if err != nil {
		t.Fatalf("GetDockerInfo() unexpected error: %v", err)
	}

	if info.ServerVersion != "24.0.7" {
		t.Errorf("ServerVersion = %q, want %q", info.ServerVersion, "24.0.7")
	}
	if info.StorageDriver != "overlay2" {
		t.Errorf("StorageDriver = %q, want %q", info.StorageDriver, "overlay2")
	}
	if info.DockerRootDir != "/var/lib/docker" {
		t.Errorf("DockerRootDir = %q, want %q", info.DockerRootDir, "/var/lib/docker")
	}
	if info.Containers != 10 {
		t.Errorf("Containers = %d, want 10", info.Containers)
	}
	if info.ContRunning != 5 {
		t.Errorf("ContRunning = %d, want 5", info.ContRunning)
	}
	if info.ContPaused != 1 {
		t.Errorf("ContPaused = %d, want 1", info.ContPaused)
	}
	if info.ContStopped != 4 {
		t.Errorf("ContStopped = %d, want 4", info.ContStopped)
	}
	if info.Images != 25 {
		t.Errorf("Images = %d, want 25", info.Images)
	}
	if info.OperatingSystem != "Ubuntu 22.04.3 LTS" {
		t.Errorf("OperatingSystem = %q, want %q", info.OperatingSystem, "Ubuntu 22.04.3 LTS")
	}
	if info.Architecture != "x86_64" {
		t.Errorf("Architecture = %q, want %q", info.Architecture, "x86_64")
	}
}

func TestGetDockerInfo_Error(t *testing.T) {
	cleanup := withFakeDockerMultiInPath(t, map[string]fakeResponse{
		"info": {stdout: "Cannot connect to the Docker daemon", exitCode: 1},
	})
	defer cleanup()

	_, err := GetDockerInfo()
	if err == nil {
		t.Error("GetDockerInfo() expected error when daemon not running")
	}
}

func TestGetDockerInfo_InvalidJSON(t *testing.T) {
	cleanup := withFakeDockerMultiInPath(t, map[string]fakeResponse{
		"info": {stdout: "not json", exitCode: 0},
	})
	defer cleanup()

	_, err := GetDockerInfo()
	if err == nil {
		t.Error("GetDockerInfo() expected error for invalid JSON")
	}
}

func TestGetDockerInfo_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir)
	defer os.Setenv("PATH", oldPath)

	_, err := GetDockerInfo()
	if err == nil {
		t.Error("GetDockerInfo() expected error when docker not installed")
	}
}
