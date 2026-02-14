package models

// DockerContainer represents a Docker container on an agent.
type DockerContainer struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Status  string   `json:"status"`
	State   string   `json:"state"`
	Created string   `json:"created"`
	Ports   []string `json:"ports,omitempty"`
}

// DockerVolume represents a Docker volume on an agent.
type DockerVolume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	SizeBytes  int64  `json:"size_bytes"`
	Created    string `json:"created"`
}

// DockerDaemonStatus represents Docker daemon status on an agent.
type DockerDaemonStatus struct {
	Available      bool   `json:"available"`
	Version        string `json:"version,omitempty"`
	ContainerCount int    `json:"container_count"`
	VolumeCount    int    `json:"volume_count"`
	ServerOS       string `json:"server_os,omitempty"`
	DockerRootDir  string `json:"docker_root_dir,omitempty"`
	StorageDriver  string `json:"storage_driver,omitempty"`
}

// DockerBackupParams holds the parameters for creating a Docker backup.
type DockerBackupParams struct {
	AgentID      string   `json:"agent_id"`
	RepositoryID string   `json:"repository_id"`
	ContainerIDs []string `json:"container_ids,omitempty"`
	VolumeNames  []string `json:"volume_names,omitempty"`
}

// DockerBackupResult holds the result of a triggered Docker backup.
type DockerBackupResult struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}
