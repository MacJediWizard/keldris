// Package docker provides Docker container and volume backup support.
package docker

import "time"

// Container represents a Docker container.
type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Labels  map[string]string `json:"labels,omitempty"`
	Mounts  []Mount           `json:"mounts,omitempty"`
	Created time.Time         `json:"created"`
}

// Mount represents a container mount point.
type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only"`
}

// Volume represents a Docker volume.
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Labels     map[string]string `json:"labels,omitempty"`
	Scope      string            `json:"scope"`
	CreatedAt  string            `json:"created_at"`
}

// ContainerInfo contains detailed information about a container.
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	State      ContainerState    `json:"state"`
	Config     ContainerConfig   `json:"config"`
	Mounts     []Mount           `json:"mounts,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Created    time.Time         `json:"created"`
	RestartKey string            `json:"restart_policy,omitempty"`
}

// ContainerState represents the running state of a container.
type ContainerState struct {
	Status     string     `json:"status"`
	Running    bool       `json:"running"`
	Paused     bool       `json:"paused"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ExitCode   int        `json:"exit_code"`
}

// ContainerConfig holds the container's configuration.
type ContainerConfig struct {
	Hostname string   `json:"hostname"`
	Env      []string `json:"env,omitempty"`
	Cmd      []string `json:"cmd,omitempty"`
}

// DockerInfo contains information about the Docker installation.
type DockerInfo struct {
	ServerVersion  string `json:"server_version"`
	StorageDriver  string `json:"storage_driver"`
	DockerRootDir  string `json:"docker_root_dir"`
	Containers     int    `json:"containers"`
	ContRunning    int    `json:"containers_running"`
	ContPaused     int    `json:"containers_paused"`
	ContStopped    int    `json:"containers_stopped"`
	Images         int    `json:"images"`
	OperatingSystem string `json:"operating_system"`
	Architecture   string `json:"architecture"`
}

// dockerInspectOutput maps the JSON output from docker inspect.
type dockerInspectOutput struct {
	ID      string `json:"Id"`
	Name    string `json:"Name"`
	Image   string `json:"Image"`
	Created string `json:"Created"`
	State   struct {
		Status     string `json:"Status"`
		Running    bool   `json:"Running"`
		Paused     bool   `json:"Paused"`
		StartedAt  string `json:"StartedAt"`
		FinishedAt string `json:"FinishedAt"`
		ExitCode   int    `json:"ExitCode"`
	} `json:"State"`
	Config struct {
		Hostname string   `json:"Hostname"`
		Env      []string `json:"Env"`
		Cmd      []string `json:"Cmd"`
		Labels   map[string]string `json:"Labels"`
		Image    string   `json:"Image"`
	} `json:"Config"`
	HostConfig struct {
		RestartPolicy struct {
			Name string `json:"Name"`
		} `json:"RestartPolicy"`
	} `json:"HostConfig"`
	Mounts []struct {
		Type        string `json:"Type"`
		Name        string `json:"Name"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
}

// dockerPSOutput maps the JSON output from docker ps --format.
type dockerPSOutput struct {
	ID      string `json:"ID"`
	Names   string `json:"Names"`
	Image   string `json:"Image"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Labels  string `json:"Labels"`
	Mounts  string `json:"Mounts"`
	Created string `json:"CreatedAt"`
}

// dockerVolumeOutput maps the JSON output from docker volume ls --format.
type dockerVolumeOutput struct {
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Mountpoint string `json:"Mountpoint"`
	Labels     string `json:"Labels"`
	Scope      string `json:"Scope"`
	CreatedAt  string `json:"CreatedAt"`
}

// dockerInfoOutput maps the JSON output from docker info --format.
type dockerInfoOutput struct {
	ServerVersion   string `json:"ServerVersion"`
	Driver          string `json:"Driver"`
	DockerRootDir   string `json:"DockerRootDir"`
	Containers      int    `json:"Containers"`
	ContainersRunning int  `json:"ContainersRunning"`
	ContainersPaused  int  `json:"ContainersPaused"`
	ContainersStopped int  `json:"ContainersStopped"`
	Images          int    `json:"Images"`
	OperatingSystem string `json:"OperatingSystem"`
	Architecture    string `json:"Architecture"`
}
