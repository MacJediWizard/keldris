package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/rs/zerolog"
)

func TestNewVolumeBackup(t *testing.T) {
	docker := NewDockerClient(zerolog.Nop())
	restic := backup.NewRestic(zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	if vb.docker != docker {
		t.Error("docker client not set correctly")
	}
	if vb.restic != restic {
		t.Error("restic not set correctly")
	}
}

func TestGetVolumePath(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		exitCode int
		want     string
		wantErr  bool
	}{
		{
			name:     "valid mountpoint",
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
			want:     "/var/lib/docker/volumes/mydata/_data",
		},
		{
			name:     "empty mountpoint",
			stdout:   `""`,
			exitCode: 0,
			wantErr:  true,
		},
		{
			name:     "volume not found",
			stdout:   "Error: No such volume: nonexistent",
			exitCode: 1,
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			stdout:   "not json",
			exitCode: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, tt.stdout, tt.exitCode)
			docker := NewDockerClientWithBinary(binary, zerolog.Nop())
			restic := backup.NewRestic(zerolog.Nop())
			vb := NewVolumeBackup(docker, restic, zerolog.Nop())

			got, err := vb.GetVolumePath(context.Background(), "mydata")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVolumePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("GetVolumePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListVolumeContents(t *testing.T) {
	tests := []struct {
		name      string
		stdout    string
		exitCode  int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "multiple files",
			stdout:    "/data/file1.txt\n/data/file2.txt\n/data/subdir/file3.log\n",
			exitCode:  0,
			wantCount: 3,
		},
		{
			name:      "single file",
			stdout:    "/data/only-file.db\n",
			exitCode:  0,
			wantCount: 1,
		},
		{
			name:      "empty volume",
			stdout:    "",
			exitCode:  0,
			wantCount: 0,
		},
		{
			name:      "output with blank lines",
			stdout:    "/data/file1.txt\n\n\n/data/file2.txt\n\n",
			exitCode:  0,
			wantCount: 2,
		},
		{
			name:     "command fails",
			stdout:   "Unable to find image 'alpine:latest' locally",
			exitCode: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := fakeDockerBinary(t, tt.stdout, tt.exitCode)
			docker := NewDockerClientWithBinary(binary, zerolog.Nop())
			restic := backup.NewRestic(zerolog.Nop())
			vb := NewVolumeBackup(docker, restic, zerolog.Nop())

			files, err := vb.ListVolumeContents(context.Background(), "testvolume")
			if (err != nil) != tt.wantErr {
				t.Errorf("ListVolumeContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(files) != tt.wantCount {
				t.Errorf("ListVolumeContents() returned %d files, want %d", len(files), tt.wantCount)
			}
		})
	}
}

func TestListVolumeContents_FileNames(t *testing.T) {
	stdout := "/data/config.yml\n/data/db/data.sql\n"
	binary := fakeDockerBinary(t, stdout, 0)
	docker := NewDockerClientWithBinary(binary, zerolog.Nop())
	restic := backup.NewRestic(zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	files, err := vb.ListVolumeContents(context.Background(), "appdata")
	if err != nil {
		t.Fatalf("ListVolumeContents() unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "/data/config.yml" {
		t.Errorf("files[0] = %q, want %q", files[0], "/data/config.yml")
	}
	if files[1] != "/data/db/data.sql" {
		t.Errorf("files[1] = %q, want %q", files[1], "/data/db/data.sql")
	}
}

// fakeResticBinary creates a shell script that mimics the restic binary.
func fakeResticBinary(t *testing.T, stdout string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()

	outFile := filepath.Join(dir, "stdout.txt")
	if err := os.WriteFile(outFile, []byte(stdout), 0644); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(dir, "restic")
	content := fmt.Sprintf("#!/bin/sh\ncat '%s'\nexit %d\n", outFile, exitCode)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	return script
}

// fakeDockerSubcmd creates a fake docker binary that matches on $1 (subcommand).
// "volume" subcommand further matches on $2 for "inspect" vs "ls".
func fakeDockerSubcmd(t *testing.T, responses map[string]fakeResponse) string {
	t.Helper()
	dir := t.TempDir()

	var scriptContent string
	scriptContent = "#!/bin/sh\n"

	for subcmd, resp := range responses {
		outFile := filepath.Join(dir, subcmd+".txt")
		if err := os.WriteFile(outFile, []byte(resp.stdout), 0644); err != nil {
			t.Fatal(err)
		}

		switch subcmd {
		case "volume_inspect":
			scriptContent += fmt.Sprintf("if [ \"$1\" = \"volume\" ] && [ \"$2\" = \"inspect\" ]; then\n  cat '%s'\n  exit %d\nfi\n", outFile, resp.exitCode)
		case "volume_ls":
			scriptContent += fmt.Sprintf("if [ \"$1\" = \"volume\" ] && [ \"$2\" = \"ls\" ]; then\n  cat '%s'\n  exit %d\nfi\n", outFile, resp.exitCode)
		default:
			scriptContent += fmt.Sprintf("if [ \"$1\" = \"%s\" ]; then\n  cat '%s'\n  exit %d\nfi\n", subcmd, outFile, resp.exitCode)
		}
	}
	scriptContent += "exit 0\n"

	scriptPath := filepath.Join(dir, "docker")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	return scriptPath
}

func TestBackupVolume_Success(t *testing.T) {
	// Fake docker: volume inspect returns mountpoint, ps returns no containers
	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   "",
			exitCode: 0,
		},
	})

	// Fake restic: backup returns a valid summary
	resticOutput := `{"message_type":"status","percent_done":1.0}
{"message_type":"summary","snapshot_id":"snap123abc","files_new":5,"files_changed":2,"files_unmodified":10,"data_added":2048000}`
	resticBinary := fakeResticBinary(t, resticOutput, 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName: "mydata",
		Tags:       []string{"daily"},
	}

	stats, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BackupVolume() unexpected error: %v", err)
	}
	if stats.SnapshotID != "snap123abc" {
		t.Errorf("SnapshotID = %q, want %q", stats.SnapshotID, "snap123abc")
	}
	if stats.FilesNew != 5 {
		t.Errorf("FilesNew = %d, want 5", stats.FilesNew)
	}
	if stats.FilesChanged != 2 {
		t.Errorf("FilesChanged = %d, want 2", stats.FilesChanged)
	}
	if stats.SizeBytes != 2048000 {
		t.Errorf("SizeBytes = %d, want 2048000", stats.SizeBytes)
	}
}

func TestBackupVolume_WithPause(t *testing.T) {
	// Container using the volume
	containerPS := `{"ID":"cont1","Names":"/webapp","Image":"nginx","State":"running","Status":"Up","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`
	inspectJSON := `{
		"Id": "cont1",
		"Name": "/webapp",
		"Image": "sha256:abc",
		"Created": "2024-06-15T10:30:00Z",
		"State": {"Status":"running","Running":true,"Paused":false,"StartedAt":"2024-06-15T10:30:01Z","FinishedAt":"0001-01-01T00:00:00Z","ExitCode":0},
		"Config": {"Hostname":"web","Env":[],"Cmd":["nginx"],"Labels":{},"Image":"nginx"},
		"HostConfig": {"RestartPolicy":{"Name":"always"}},
		"Mounts": [{"Type":"volume","Name":"mydata","Source":"/var/lib/docker/volumes/mydata/_data","Destination":"/data","RW":true}]
	}`

	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   containerPS,
			exitCode: 0,
		},
		"inspect": {
			stdout:   inspectJSON,
			exitCode: 0,
		},
		"pause": {
			stdout:   "cont1",
			exitCode: 0,
		},
		"unpause": {
			stdout:   "cont1",
			exitCode: 0,
		},
	})

	resticOutput := `{"message_type":"summary","snapshot_id":"snap456","files_new":1,"files_changed":0,"files_unmodified":0,"data_added":512}`
	resticBinary := fakeResticBinary(t, resticOutput, 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName:      "mydata",
		PauseContainers: true,
	}

	stats, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BackupVolume() with pause unexpected error: %v", err)
	}
	if stats.SnapshotID != "snap456" {
		t.Errorf("SnapshotID = %q, want %q", stats.SnapshotID, "snap456")
	}
}

func TestBackupVolume_VolumeNotFound(t *testing.T) {
	dockerBinary := fakeDockerBinary(t, "Error: No such volume: missing", 1)
	resticBinary := fakeResticBinary(t, "", 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName: "missing",
	}

	_, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err == nil {
		t.Error("BackupVolume() expected error when volume not found")
	}
}

func TestBackupVolume_ResticFails(t *testing.T) {
	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   "",
			exitCode: 0,
		},
	})

	// Restic fails
	resticBinary := fakeResticBinary(t, "Fatal: unable to open repository", 1)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName: "mydata",
	}

	_, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err == nil {
		t.Error("BackupVolume() expected error when restic fails")
	}
}

func TestBackupVolume_WithOptions(t *testing.T) {
	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/appvol/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   "",
			exitCode: 0,
		},
	})

	resticOutput := `{"message_type":"summary","snapshot_id":"snap789","files_new":3,"files_changed":0,"files_unmodified":0,"data_added":1024}`
	resticBinary := fakeResticBinary(t, resticOutput, 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	bwLimit := 500
	compression := "max"

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName:       "appvol",
		Tags:             []string{"manual", "critical"},
		Excludes:         []string{"*.tmp", "*.log"},
		BandwidthLimitKB: &bwLimit,
		CompressionLevel: &compression,
	}

	stats, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BackupVolume() with options unexpected error: %v", err)
	}
	if stats.SnapshotID != "snap789" {
		t.Errorf("SnapshotID = %q, want %q", stats.SnapshotID, "snap789")
	}
}

func TestBackupVolume_PauseFails(t *testing.T) {
	containerPS := `{"ID":"cont1","Names":"/webapp","Image":"nginx","State":"running","Status":"Up","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`
	inspectJSON := `{
		"Id": "cont1",
		"Name": "/webapp",
		"Image": "sha256:abc",
		"Created": "2024-06-15T10:30:00Z",
		"State": {"Status":"running","Running":true,"Paused":false,"StartedAt":"2024-06-15T10:30:01Z","FinishedAt":"0001-01-01T00:00:00Z","ExitCode":0},
		"Config": {"Hostname":"web","Env":[],"Cmd":["nginx"],"Labels":{},"Image":"nginx"},
		"HostConfig": {"RestartPolicy":{"Name":"always"}},
		"Mounts": [{"Type":"volume","Name":"mydata","Source":"/var/lib/docker/volumes/mydata/_data","Destination":"/data","RW":true}]
	}`

	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   containerPS,
			exitCode: 0,
		},
		"inspect": {
			stdout:   inspectJSON,
			exitCode: 0,
		},
		"pause": {
			stdout:   "Error: Cannot pause",
			exitCode: 1,
		},
		"unpause": {
			stdout:   "",
			exitCode: 0,
		},
	})

	resticBinary := fakeResticBinary(t, "", 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName:      "mydata",
		PauseContainers: true,
	}

	_, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err == nil {
		t.Error("BackupVolume() expected error when pause fails")
	}
}

func TestBackupVolume_NoPauseSkipsStoppedContainers(t *testing.T) {
	// Stopped container should be ignored during pause
	containerPS := `{"ID":"stopped1","Names":"/stopped-app","Image":"nginx","State":"exited","Status":"Exited (0)","Labels":"","Mounts":"","CreatedAt":"2024-06-15 10:30:00 -0700 MST"}
`
	dockerBinary := fakeDockerSubcmd(t, map[string]fakeResponse{
		"volume_inspect": {
			stdout:   `"/var/lib/docker/volumes/mydata/_data"`,
			exitCode: 0,
		},
		"ps": {
			stdout:   containerPS,
			exitCode: 0,
		},
	})

	resticOutput := `{"message_type":"summary","snapshot_id":"snapABC","files_new":1,"files_changed":0,"files_unmodified":0,"data_added":100}`
	resticBinary := fakeResticBinary(t, resticOutput, 0)

	docker := NewDockerClientWithBinary(dockerBinary, zerolog.Nop())
	restic := backup.NewResticWithBinary(resticBinary, zerolog.Nop())
	vb := NewVolumeBackup(docker, restic, zerolog.Nop())

	cfg := backends.ResticConfig{
		Repository: "/tmp/test-repo",
		Password:   "testpassword",
	}
	opts := BackupVolumeOptions{
		VolumeName:      "mydata",
		PauseContainers: true,
	}

	stats, err := vb.BackupVolume(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BackupVolume() unexpected error: %v", err)
	}
	if stats.SnapshotID != "snapABC" {
		t.Errorf("SnapshotID = %q, want %q", stats.SnapshotID, "snapABC")
	}
}
