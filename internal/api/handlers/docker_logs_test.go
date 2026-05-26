package handlers

import (
	"testing"
)

// DockerLogsHandler depends on a concrete *docker.LogBackupService that performs
// real filesystem and agent operations. Tests that exercise the handler require
// a constructable backup service; skip until a service interface exists.
func TestDockerLogsHandler(t *testing.T) {
	t.Skip("requires real *docker.LogBackupService - no clean interface available")
}
