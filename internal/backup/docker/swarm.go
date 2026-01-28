// Package docker provides Docker Swarm backup and restore functionality.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Error definitions for Swarm backup operations.
var (
	ErrNotSwarmManager   = errors.New("node is not a swarm manager")
	ErrServiceNotFound   = errors.New("service not found")
	ErrStackNotFound     = errors.New("stack not found")
	ErrBackupFailed      = errors.New("backup failed")
	ErrRestoreFailed     = errors.New("restore failed")
	ErrDependencyCycle   = errors.New("circular dependency detected")
	ErrInvalidBackupData = errors.New("invalid backup data")
)

// SwarmBackupConfig holds configuration for Swarm backup operations.
type SwarmBackupConfig struct {
	// DockerHost is the Docker daemon socket (default: unix:///var/run/docker.sock)
	DockerHost string `json:"docker_host,omitempty"`
	// IncludeSecrets determines whether to backup secret metadata (not values)
	IncludeSecrets bool `json:"include_secrets,omitempty"`
	// IncludeConfigs determines whether to backup Docker configs
	IncludeConfigs bool `json:"include_configs,omitempty"`
	// IncludeNetworks determines whether to backup custom networks
	IncludeNetworks bool `json:"include_networks,omitempty"`
	// IncludeVolumes determines whether to backup volume metadata
	IncludeVolumes bool `json:"include_volumes,omitempty"`
	// StackFilter limits backup to specific stacks (empty = all)
	StackFilter []string `json:"stack_filter,omitempty"`
	// ServiceFilter limits backup to specific services (empty = all)
	ServiceFilter []string `json:"service_filter,omitempty"`
	// BackupPath is the directory where backups are stored
	BackupPath string `json:"backup_path,omitempty"`
}

// DefaultSwarmBackupConfig returns a config with sensible defaults.
func DefaultSwarmBackupConfig() *SwarmBackupConfig {
	return &SwarmBackupConfig{
		DockerHost:      "unix:///var/run/docker.sock",
		IncludeSecrets:  true,
		IncludeConfigs:  true,
		IncludeNetworks: true,
		IncludeVolumes:  true,
		BackupPath:      "/var/lib/keldris/swarm-backups",
	}
}

// NodeRole represents the role of a node in the Swarm cluster.
type NodeRole string

const (
	NodeRoleManager NodeRole = "manager"
	NodeRoleWorker  NodeRole = "worker"
)

// NodeState represents the current state of a Swarm node.
type NodeState string

const (
	NodeStateReady        NodeState = "ready"
	NodeStateDown         NodeState = "down"
	NodeStateDisconnected NodeState = "disconnected"
)

// SwarmNode represents a node in the Docker Swarm cluster.
type SwarmNode struct {
	ID           string            `json:"id"`
	Hostname     string            `json:"hostname"`
	Role         NodeRole          `json:"role"`
	State        NodeState         `json:"state"`
	Availability string            `json:"availability"`
	Address      string            `json:"address"`
	Labels       map[string]string `json:"labels,omitempty"`
	EngineVersion string           `json:"engine_version,omitempty"`
	IsLeader     bool              `json:"is_leader"`
}

// SwarmService represents a service definition in the Swarm cluster.
type SwarmService struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Image         string                 `json:"image"`
	Replicas      int                    `json:"replicas"`
	Mode          string                 `json:"mode"` // "replicated" or "global"
	Ports         []ServicePort          `json:"ports,omitempty"`
	Mounts        []ServiceMount         `json:"mounts,omitempty"`
	Networks      []string               `json:"networks,omitempty"`
	Constraints   []string               `json:"constraints,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Env           []string               `json:"env,omitempty"`
	Secrets       []ServiceSecret        `json:"secrets,omitempty"`
	Configs       []SwarmServiceConfig        `json:"configs,omitempty"`
	DependsOn     []string               `json:"depends_on,omitempty"`
	StackName     string                 `json:"stack_name,omitempty"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Spec          map[string]interface{} `json:"spec,omitempty"` // Raw service spec for full restore
}

// ServicePort represents a port mapping for a service.
type ServicePort struct {
	Protocol      string `json:"protocol"`
	TargetPort    uint32 `json:"target_port"`
	PublishedPort uint32 `json:"published_port"`
	PublishMode   string `json:"publish_mode"`
}

// ServiceMount represents a mount point for a service.
type ServiceMount struct {
	Type     string `json:"type"` // "bind", "volume", "tmpfs"
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// ServiceSecret represents a secret used by a service.
type ServiceSecret struct {
	SecretID   string `json:"secret_id"`
	SecretName string `json:"secret_name"`
	FileName   string `json:"file_name,omitempty"`
	UID        string `json:"uid,omitempty"`
	GID        string `json:"gid,omitempty"`
	Mode       uint32 `json:"mode,omitempty"`
}

// SwarmServiceConfig represents a config used by a service.
type SwarmServiceConfig struct {
	ConfigID   string `json:"config_id"`
	ConfigName string `json:"config_name"`
	FileName   string `json:"file_name,omitempty"`
	UID        string `json:"uid,omitempty"`
	GID        string `json:"gid,omitempty"`
	Mode       uint32 `json:"mode,omitempty"`
}

// SwarmStack represents a Docker stack deployment.
type SwarmStack struct {
	Name           string                 `json:"name"`
	Services       []string               `json:"services"`
	Networks       []string               `json:"networks,omitempty"`
	Volumes        []string               `json:"volumes,omitempty"`
	Secrets        []string               `json:"secrets,omitempty"`
	Configs        []string               `json:"configs,omitempty"`
	ComposeFile    string                 `json:"compose_file,omitempty"` // Original compose file content if available
	ComposeVersion string                 `json:"compose_version,omitempty"`
	Labels         map[string]string      `json:"labels,omitempty"`
}

// SwarmSecret represents a Docker secret (metadata only, not the value).
type SwarmSecret struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SwarmConfig represents a Docker config.
type SwarmConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Data      string            `json:"data,omitempty"` // Base64 encoded
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SwarmNetwork represents a Docker network.
type SwarmNetwork struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Ingress    bool              `json:"ingress"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	SwarmIPAMConfig []SwarmIPAMConfig      `json:"ipam_config,omitempty"`
}

// SwarmIPAMConfig represents IP address management configuration.
type SwarmIPAMConfig struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	IPRange string `json:"ip_range,omitempty"`
}

// SwarmVolume represents a Docker volume.
type SwarmVolume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	Mountpoint string            `json:"mountpoint,omitempty"`
}

// ClusterState represents the complete state of a Swarm cluster.
type ClusterState struct {
	ClusterID   string    `json:"cluster_id"`
	CreatedAt   time.Time `json:"created_at"`
	BackupTime  time.Time `json:"backup_time"`
	Version     string    `json:"version"` // Backup format version
	ManagerNode string    `json:"manager_node"`
}

// SwarmBackup represents a complete Swarm cluster backup.
type SwarmBackup struct {
	Metadata     BackupMetadata  `json:"metadata"`
	ClusterState ClusterState    `json:"cluster_state"`
	Nodes        []SwarmNode     `json:"nodes"`
	Services     []SwarmService  `json:"services"`
	Stacks       []SwarmStack    `json:"stacks"`
	Secrets      []SwarmSecret   `json:"secrets,omitempty"`
	Configs      []SwarmConfig   `json:"configs,omitempty"`
	Networks     []SwarmNetwork  `json:"networks,omitempty"`
	Volumes      []SwarmVolume   `json:"volumes,omitempty"`
}

// BackupMetadata contains information about the backup itself.
type BackupMetadata struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Version      string    `json:"version"`
	AgentID      string    `json:"agent_id,omitempty"`
	Hostname     string    `json:"hostname"`
	Description  string    `json:"description,omitempty"`
	ServiceCount int       `json:"service_count"`
	StackCount   int       `json:"stack_count"`
	NodeCount    int       `json:"node_count"`
}

// SwarmRestoreOptions configures how services are restored.
type SwarmRestoreOptions struct {
	// DryRun performs a dry run without making changes
	DryRun bool `json:"dry_run"`
	// Force removes existing services before restore
	Force bool `json:"force"`
	// IncludeNetworks restores custom networks
	IncludeNetworks bool `json:"include_networks"`
	// IncludeVolumes restores volume definitions
	IncludeVolumes bool `json:"include_volumes"`
	// IncludeConfigs restores Docker configs
	IncludeConfigs bool `json:"include_configs"`
	// StackFilter limits restore to specific stacks
	StackFilter []string `json:"stack_filter,omitempty"`
	// ServiceFilter limits restore to specific services
	ServiceFilter []string `json:"service_filter,omitempty"`
	// RespectDependencies ensures services are restored in dependency order
	RespectDependencies bool `json:"respect_dependencies"`
}

// DefaultSwarmRestoreOptions returns restore options with sensible defaults.
func DefaultSwarmRestoreOptions() *SwarmRestoreOptions {
	return &SwarmRestoreOptions{
		DryRun:              false,
		Force:               false,
		IncludeNetworks:     true,
		IncludeVolumes:      true,
		IncludeConfigs:      true,
		RespectDependencies: true,
	}
}

// SwarmRestoreResult contains the results of a restore operation.
type SwarmRestoreResult struct {
	Success          bool                    `json:"success"`
	ServicesRestored []string                `json:"services_restored"`
	ServicesFailed   []ServiceRestoreFailure `json:"services_failed,omitempty"`
	NetworksCreated  []string                `json:"networks_created,omitempty"`
	VolumesCreated   []string                `json:"volumes_created,omitempty"`
	ConfigsCreated   []string                `json:"configs_created,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	Duration         time.Duration           `json:"duration"`
}

// ServiceRestoreFailure represents a failed service restore.
type ServiceRestoreFailure struct {
	ServiceName string `json:"service_name"`
	Error       string `json:"error"`
}

// SwarmBackupManager handles Swarm backup and restore operations.
type SwarmBackupManager struct {
	config *SwarmBackupConfig
	logger zerolog.Logger
	binary string // docker binary path
}

// NewSwarmBackupManager creates a new SwarmBackupManager.
func NewSwarmBackupManager(config *SwarmBackupConfig, logger zerolog.Logger) *SwarmBackupManager {
	if config == nil {
		config = DefaultSwarmBackupConfig()
	}
	return &SwarmBackupManager{
		config: config,
		logger: logger.With().Str("component", "swarm-backup").Logger(),
		binary: "docker",
	}
}

// NewSwarmBackupManagerWithBinary creates a manager with a custom docker binary path.
func NewSwarmBackupManagerWithBinary(config *SwarmBackupConfig, binary string, logger zerolog.Logger) *SwarmBackupManager {
	mgr := NewSwarmBackupManager(config, logger)
	mgr.binary = binary
	return mgr
}

// IsSwarmManager checks if the current node is a Swarm manager.
func (m *SwarmBackupManager) IsSwarmManager(ctx context.Context) (bool, error) {
	output, err := m.runDocker(ctx, "info", "--format", "{{.Swarm.LocalNodeState}}")
	if err != nil {
		return false, fmt.Errorf("check swarm status: %w", err)
	}

	state := strings.TrimSpace(string(output))
	if state != "active" {
		return false, nil
	}

	// Check if manager
	output, err = m.runDocker(ctx, "info", "--format", "{{.Swarm.ControlAvailable}}")
	if err != nil {
		return false, fmt.Errorf("check manager status: %w", err)
	}

	return strings.TrimSpace(string(output)) == "true", nil
}

// Backup performs a complete backup of the Swarm cluster state.
func (m *SwarmBackupManager) Backup(ctx context.Context) (*SwarmBackup, error) {
	m.logger.Info().Msg("starting swarm backup")
	start := time.Now()

	// Verify we're on a manager node
	isManager, err := m.IsSwarmManager(ctx)
	if err != nil {
		return nil, err
	}
	if !isManager {
		return nil, ErrNotSwarmManager
	}

	backup := &SwarmBackup{
		Metadata: BackupMetadata{
			ID:        fmt.Sprintf("swarm-backup-%d", time.Now().Unix()),
			Timestamp: time.Now(),
			Version:   "1.0",
		},
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	backup.Metadata.Hostname = hostname

	// Backup cluster state
	clusterState, err := m.backupClusterState(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup cluster state: %w", err)
	}
	backup.ClusterState = *clusterState

	// Backup nodes
	nodes, err := m.backupNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup nodes: %w", err)
	}
	backup.Nodes = nodes
	backup.Metadata.NodeCount = len(nodes)

	// Backup services
	services, err := m.backupServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("backup services: %w", err)
	}
	backup.Services = m.filterServices(services)
	backup.Metadata.ServiceCount = len(backup.Services)

	// Backup stacks
	stacks, err := m.backupStacks(ctx)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to backup stacks, continuing")
	} else {
		backup.Stacks = m.filterStacks(stacks)
		backup.Metadata.StackCount = len(backup.Stacks)
	}

	// Backup secrets (metadata only)
	if m.config.IncludeSecrets {
		secrets, err := m.backupSecrets(ctx)
		if err != nil {
			m.logger.Warn().Err(err).Msg("failed to backup secrets, continuing")
		} else {
			backup.Secrets = secrets
		}
	}

	// Backup configs
	if m.config.IncludeConfigs {
		configs, err := m.backupConfigs(ctx)
		if err != nil {
			m.logger.Warn().Err(err).Msg("failed to backup configs, continuing")
		} else {
			backup.Configs = configs
		}
	}

	// Backup networks
	if m.config.IncludeNetworks {
		networks, err := m.backupNetworks(ctx)
		if err != nil {
			m.logger.Warn().Err(err).Msg("failed to backup networks, continuing")
		} else {
			backup.Networks = networks
		}
	}

	// Backup volumes
	if m.config.IncludeVolumes {
		volumes, err := m.backupVolumes(ctx)
		if err != nil {
			m.logger.Warn().Err(err).Msg("failed to backup volumes, continuing")
		} else {
			backup.Volumes = volumes
		}
	}

	m.logger.Info().
		Int("services", backup.Metadata.ServiceCount).
		Int("stacks", backup.Metadata.StackCount).
		Int("nodes", backup.Metadata.NodeCount).
		Dur("duration", time.Since(start)).
		Msg("swarm backup completed")

	return backup, nil
}

// backupClusterState retrieves the cluster state information.
func (m *SwarmBackupManager) backupClusterState(ctx context.Context) (*ClusterState, error) {
	output, err := m.runDocker(ctx, "info", "--format", "{{json .Swarm}}")
	if err != nil {
		return nil, err
	}

	var swarmInfo struct {
		NodeID           string `json:"NodeID"`
		LocalNodeState   string `json:"LocalNodeState"`
		ControlAvailable bool   `json:"ControlAvailable"`
		Cluster          struct {
			ID        string    `json:"ID"`
			CreatedAt time.Time `json:"CreatedAt"`
		} `json:"Cluster"`
	}

	if err := json.Unmarshal(output, &swarmInfo); err != nil {
		return nil, fmt.Errorf("parse swarm info: %w", err)
	}

	hostname, _ := os.Hostname()

	return &ClusterState{
		ClusterID:   swarmInfo.Cluster.ID,
		CreatedAt:   swarmInfo.Cluster.CreatedAt,
		BackupTime:  time.Now(),
		Version:     "1.0",
		ManagerNode: hostname,
	}, nil
}

// backupNodes retrieves all nodes in the cluster.
func (m *SwarmBackupManager) backupNodes(ctx context.Context) ([]SwarmNode, error) {
	output, err := m.runDocker(ctx, "node", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}

	var nodes []SwarmNode
	lines := bytes.Split(output, []byte("\n"))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var nodeInfo struct {
			ID           string `json:"ID"`
			Hostname     string `json:"Hostname"`
			Status       string `json:"Status"`
			Availability string `json:"Availability"`
			ManagerStatus string `json:"ManagerStatus"`
			EngineVersion string `json:"EngineVersion"`
		}

		if err := json.Unmarshal(line, &nodeInfo); err != nil {
			m.logger.Warn().Err(err).Msg("failed to parse node info")
			continue
		}

		node := SwarmNode{
			ID:            nodeInfo.ID,
			Hostname:      nodeInfo.Hostname,
			Availability:  nodeInfo.Availability,
			EngineVersion: nodeInfo.EngineVersion,
		}

		// Determine role and state
		if nodeInfo.ManagerStatus != "" {
			node.Role = NodeRoleManager
			node.IsLeader = nodeInfo.ManagerStatus == "Leader"
		} else {
			node.Role = NodeRoleWorker
		}

		switch nodeInfo.Status {
		case "Ready":
			node.State = NodeStateReady
		case "Down":
			node.State = NodeStateDown
		default:
			node.State = NodeStateDisconnected
		}

		// Get detailed node info including labels
		detailOutput, err := m.runDocker(ctx, "node", "inspect", "--format", "{{json .}}", nodeInfo.ID)
		if err == nil {
			var detail struct {
				Spec struct {
					Labels map[string]string `json:"Labels"`
				} `json:"Spec"`
				Status struct {
					Addr string `json:"Addr"`
				} `json:"Status"`
			}
			if json.Unmarshal(detailOutput, &detail) == nil {
				node.Labels = detail.Spec.Labels
				node.Address = detail.Status.Addr
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// backupServices retrieves all service definitions.
func (m *SwarmBackupManager) backupServices(ctx context.Context) ([]SwarmService, error) {
	output, err := m.runDocker(ctx, "service", "ls", "--format", "{{.ID}}")
	if err != nil {
		return nil, err
	}

	var services []SwarmService
	serviceIDs := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, id := range serviceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		service, err := m.inspectService(ctx, id)
		if err != nil {
			m.logger.Warn().Err(err).Str("service_id", id).Msg("failed to inspect service")
			continue
		}

		services = append(services, *service)
	}

	return services, nil
}

// inspectService retrieves detailed information about a service.
func (m *SwarmBackupManager) inspectService(ctx context.Context, serviceID string) (*SwarmService, error) {
	output, err := m.runDocker(ctx, "service", "inspect", "--format", "{{json .}}", serviceID)
	if err != nil {
		return nil, err
	}

	var spec struct {
		ID   string `json:"ID"`
		Spec struct {
			Name   string            `json:"Name"`
			Labels map[string]string `json:"Labels"`
			TaskTemplate struct {
				ContainerSpec struct {
					Image   string   `json:"Image"`
					Env     []string `json:"Env"`
					Mounts  []struct {
						Type     string `json:"Type"`
						Source   string `json:"Source"`
						Target   string `json:"Target"`
						ReadOnly bool   `json:"ReadOnly"`
					} `json:"Mounts"`
					Secrets []struct {
						SecretID   string `json:"SecretID"`
						SecretName string `json:"SecretName"`
						File       struct {
							Name string `json:"Name"`
							UID  string `json:"UID"`
							GID  string `json:"GID"`
							Mode uint32 `json:"Mode"`
						} `json:"File"`
					} `json:"Secrets"`
					Configs []struct {
						ConfigID   string `json:"ConfigID"`
						ConfigName string `json:"ConfigName"`
						File       struct {
							Name string `json:"Name"`
							UID  string `json:"UID"`
							GID  string `json:"GID"`
							Mode uint32 `json:"Mode"`
						} `json:"File"`
					} `json:"Configs"`
				} `json:"ContainerSpec"`
				Networks []struct {
					Target string `json:"Target"`
				} `json:"Networks"`
				Placement struct {
					Constraints []string `json:"Constraints"`
				} `json:"Placement"`
			} `json:"TaskTemplate"`
			Mode struct {
				Replicated *struct {
					Replicas uint64 `json:"Replicas"`
				} `json:"Replicated"`
				Global *struct{} `json:"Global"`
			} `json:"Mode"`
			EndpointSpec *struct {
				Ports []struct {
					Protocol      string `json:"Protocol"`
					TargetPort    uint32 `json:"TargetPort"`
					PublishedPort uint32 `json:"PublishedPort"`
					PublishMode   string `json:"PublishMode"`
				} `json:"Ports"`
			} `json:"EndpointSpec"`
		} `json:"Spec"`
		UpdatedAt time.Time `json:"UpdatedAt"`
	}

	if err := json.Unmarshal(output, &spec); err != nil {
		return nil, fmt.Errorf("parse service spec: %w", err)
	}

	service := &SwarmService{
		ID:          spec.ID,
		Name:        spec.Spec.Name,
		Image:       spec.Spec.TaskTemplate.ContainerSpec.Image,
		Labels:      spec.Spec.Labels,
		Env:         spec.Spec.TaskTemplate.ContainerSpec.Env,
		Constraints: spec.Spec.TaskTemplate.Placement.Constraints,
		UpdatedAt:   spec.UpdatedAt,
	}

	// Determine mode and replicas
	if spec.Spec.Mode.Global != nil {
		service.Mode = "global"
		service.Replicas = 0
	} else if spec.Spec.Mode.Replicated != nil {
		service.Mode = "replicated"
		service.Replicas = int(spec.Spec.Mode.Replicated.Replicas)
	}

	// Extract stack name from labels
	if stackName, ok := spec.Spec.Labels["com.docker.stack.namespace"]; ok {
		service.StackName = stackName
	}

	// Extract ports
	if spec.Spec.EndpointSpec != nil {
		for _, p := range spec.Spec.EndpointSpec.Ports {
			service.Ports = append(service.Ports, ServicePort{
				Protocol:      p.Protocol,
				TargetPort:    p.TargetPort,
				PublishedPort: p.PublishedPort,
				PublishMode:   p.PublishMode,
			})
		}
	}

	// Extract mounts
	for _, m := range spec.Spec.TaskTemplate.ContainerSpec.Mounts {
		service.Mounts = append(service.Mounts, ServiceMount{
			Type:     m.Type,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// Extract networks
	for _, n := range spec.Spec.TaskTemplate.Networks {
		service.Networks = append(service.Networks, n.Target)
	}

	// Extract secrets
	for _, s := range spec.Spec.TaskTemplate.ContainerSpec.Secrets {
		service.Secrets = append(service.Secrets, ServiceSecret{
			SecretID:   s.SecretID,
			SecretName: s.SecretName,
			FileName:   s.File.Name,
			UID:        s.File.UID,
			GID:        s.File.GID,
			Mode:       s.File.Mode,
		})
	}

	// Extract configs
	for _, c := range spec.Spec.TaskTemplate.ContainerSpec.Configs {
		service.Configs = append(service.Configs, SwarmServiceConfig{
			ConfigID:   c.ConfigID,
			ConfigName: c.ConfigName,
			FileName:   c.File.Name,
			UID:        c.File.UID,
			GID:        c.File.GID,
			Mode:       c.File.Mode,
		})
	}

	// Store raw spec for complete restore
	var rawSpec map[string]interface{}
	if err := json.Unmarshal(output, &rawSpec); err == nil {
		service.Spec = rawSpec
	}

	// Infer dependencies from labels
	if deps, ok := spec.Spec.Labels["com.docker.compose.depends_on"]; ok {
		service.DependsOn = strings.Split(deps, ",")
	}

	return service, nil
}

// backupStacks retrieves all stack definitions.
func (m *SwarmBackupManager) backupStacks(ctx context.Context) ([]SwarmStack, error) {
	output, err := m.runDocker(ctx, "stack", "ls", "--format", "{{.Name}}")
	if err != nil {
		// Stack command might not be available
		return nil, err
	}

	var stacks []SwarmStack
	stackNames := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, name := range stackNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		stack, err := m.inspectStack(ctx, name)
		if err != nil {
			m.logger.Warn().Err(err).Str("stack", name).Msg("failed to inspect stack")
			continue
		}

		stacks = append(stacks, *stack)
	}

	return stacks, nil
}

// inspectStack retrieves detailed information about a stack.
func (m *SwarmBackupManager) inspectStack(ctx context.Context, stackName string) (*SwarmStack, error) {
	// Get services in the stack
	output, err := m.runDocker(ctx, "stack", "services", stackName, "--format", "{{.Name}}")
	if err != nil {
		return nil, err
	}

	stack := &SwarmStack{
		Name:     stackName,
		Services: []string{},
	}

	services := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, svc := range services {
		svc = strings.TrimSpace(svc)
		if svc != "" {
			stack.Services = append(stack.Services, svc)
		}
	}

	return stack, nil
}

// backupSecrets retrieves all secret metadata.
func (m *SwarmBackupManager) backupSecrets(ctx context.Context) ([]SwarmSecret, error) {
	output, err := m.runDocker(ctx, "secret", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}

	var secrets []SwarmSecret
	lines := bytes.Split(output, []byte("\n"))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var secretInfo struct {
			ID        string    `json:"ID"`
			Name      string    `json:"Name"`
			CreatedAt string    `json:"CreatedAt"`
			UpdatedAt string    `json:"UpdatedAt"`
		}

		if err := json.Unmarshal(line, &secretInfo); err != nil {
			continue
		}

		secret := SwarmSecret{
			ID:   secretInfo.ID,
			Name: secretInfo.Name,
		}

		// Get detailed info
		detailOutput, err := m.runDocker(ctx, "secret", "inspect", "--format", "{{json .}}", secretInfo.ID)
		if err == nil {
			var detail struct {
				Spec struct {
					Labels map[string]string `json:"Labels"`
				} `json:"Spec"`
				CreatedAt time.Time `json:"CreatedAt"`
				UpdatedAt time.Time `json:"UpdatedAt"`
			}
			if json.Unmarshal(detailOutput, &detail) == nil {
				secret.Labels = detail.Spec.Labels
				secret.CreatedAt = detail.CreatedAt
				secret.UpdatedAt = detail.UpdatedAt
			}
		}

		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// backupConfigs retrieves all Docker configs.
func (m *SwarmBackupManager) backupConfigs(ctx context.Context) ([]SwarmConfig, error) {
	output, err := m.runDocker(ctx, "config", "ls", "--format", "{{.ID}}")
	if err != nil {
		return nil, err
	}

	var configs []SwarmConfig
	configIDs := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, id := range configIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		detailOutput, err := m.runDocker(ctx, "config", "inspect", "--format", "{{json .}}", id)
		if err != nil {
			continue
		}

		var detail struct {
			ID   string `json:"ID"`
			Spec struct {
				Name   string            `json:"Name"`
				Labels map[string]string `json:"Labels"`
				Data   string            `json:"Data"`
			} `json:"Spec"`
			CreatedAt time.Time `json:"CreatedAt"`
			UpdatedAt time.Time `json:"UpdatedAt"`
		}

		if err := json.Unmarshal(detailOutput, &detail); err != nil {
			continue
		}

		configs = append(configs, SwarmConfig{
			ID:        detail.ID,
			Name:      detail.Spec.Name,
			Data:      detail.Spec.Data,
			Labels:    detail.Spec.Labels,
			CreatedAt: detail.CreatedAt,
			UpdatedAt: detail.UpdatedAt,
		})
	}

	return configs, nil
}

// backupNetworks retrieves all custom networks.
func (m *SwarmBackupManager) backupNetworks(ctx context.Context) ([]SwarmNetwork, error) {
	output, err := m.runDocker(ctx, "network", "ls", "--format", "{{json .}}", "--filter", "scope=swarm")
	if err != nil {
		return nil, err
	}

	var networks []SwarmNetwork
	lines := bytes.Split(output, []byte("\n"))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var netInfo struct {
			ID     string `json:"ID"`
			Name   string `json:"Name"`
			Driver string `json:"Driver"`
			Scope  string `json:"Scope"`
		}

		if err := json.Unmarshal(line, &netInfo); err != nil {
			continue
		}

		// Skip built-in networks
		if netInfo.Name == "ingress" || netInfo.Name == "docker_gwbridge" {
			continue
		}

		network := SwarmNetwork{
			ID:     netInfo.ID,
			Name:   netInfo.Name,
			Driver: netInfo.Driver,
			Scope:  netInfo.Scope,
		}

		// Get detailed info
		detailOutput, err := m.runDocker(ctx, "network", "inspect", "--format", "{{json .}}", netInfo.ID)
		if err == nil {
			var detail struct {
				Labels     map[string]string `json:"Labels"`
				Options    map[string]string `json:"Options"`
				Internal   bool              `json:"Internal"`
				Attachable bool              `json:"Attachable"`
				Ingress    bool              `json:"Ingress"`
				IPAM       struct {
					Config []struct {
						Subnet  string `json:"Subnet"`
						Gateway string `json:"Gateway"`
						IPRange string `json:"IPRange"`
					} `json:"Config"`
				} `json:"IPAM"`
			}
			if json.Unmarshal(detailOutput, &detail) == nil {
				network.Labels = detail.Labels
				network.Options = detail.Options
				network.Internal = detail.Internal
				network.Attachable = detail.Attachable
				network.Ingress = detail.Ingress

				for _, cfg := range detail.IPAM.Config {
					network.SwarmIPAMConfig = append(network.SwarmIPAMConfig, SwarmIPAMConfig{
						Subnet:  cfg.Subnet,
						Gateway: cfg.Gateway,
						IPRange: cfg.IPRange,
					})
				}
			}
		}

		networks = append(networks, network)
	}

	return networks, nil
}

// backupVolumes retrieves all volume definitions.
func (m *SwarmBackupManager) backupVolumes(ctx context.Context) ([]SwarmVolume, error) {
	output, err := m.runDocker(ctx, "volume", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}

	var volumes []SwarmVolume
	lines := bytes.Split(output, []byte("\n"))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var volInfo struct {
			Name   string `json:"Name"`
			Driver string `json:"Driver"`
		}

		if err := json.Unmarshal(line, &volInfo); err != nil {
			continue
		}

		volume := SwarmVolume{
			Name:   volInfo.Name,
			Driver: volInfo.Driver,
		}

		// Get detailed info
		detailOutput, err := m.runDocker(ctx, "volume", "inspect", "--format", "{{json .}}", volInfo.Name)
		if err == nil {
			var detail struct {
				Labels     map[string]string `json:"Labels"`
				Options    map[string]string `json:"Options"`
				Mountpoint string            `json:"Mountpoint"`
			}
			if json.Unmarshal(detailOutput, &detail) == nil {
				volume.Labels = detail.Labels
				volume.Options = detail.Options
				volume.Mountpoint = detail.Mountpoint
			}
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// Restore restores services from a backup in dependency order.
func (m *SwarmBackupManager) Restore(ctx context.Context, backup *SwarmBackup, opts *SwarmRestoreOptions) (*SwarmRestoreResult, error) {
	if opts == nil {
		opts = DefaultSwarmRestoreOptions()
	}

	m.logger.Info().
		Bool("dry_run", opts.DryRun).
		Bool("force", opts.Force).
		Bool("respect_deps", opts.RespectDependencies).
		Msg("starting swarm restore")

	start := time.Now()
	result := &SwarmRestoreResult{
		Success:          true,
		ServicesRestored: []string{},
		ServicesFailed:   []ServiceRestoreFailure{},
		NetworksCreated:  []string{},
		VolumesCreated:   []string{},
		ConfigsCreated:   []string{},
		Warnings:         []string{},
	}

	// Verify we're on a manager node
	isManager, err := m.IsSwarmManager(ctx)
	if err != nil {
		return nil, err
	}
	if !isManager {
		return nil, ErrNotSwarmManager
	}

	// Restore networks first
	if opts.IncludeNetworks && len(backup.Networks) > 0 {
		for _, network := range backup.Networks {
			if opts.DryRun {
				result.NetworksCreated = append(result.NetworksCreated, network.Name+" (dry-run)")
				continue
			}

			if err := m.restoreNetwork(ctx, &network); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to restore network %s: %v", network.Name, err))
			} else {
				result.NetworksCreated = append(result.NetworksCreated, network.Name)
			}
		}
	}

	// Restore volumes
	if opts.IncludeVolumes && len(backup.Volumes) > 0 {
		for _, volume := range backup.Volumes {
			if opts.DryRun {
				result.VolumesCreated = append(result.VolumesCreated, volume.Name+" (dry-run)")
				continue
			}

			if err := m.restoreVolume(ctx, &volume); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to restore volume %s: %v", volume.Name, err))
			} else {
				result.VolumesCreated = append(result.VolumesCreated, volume.Name)
			}
		}
	}

	// Restore configs
	if opts.IncludeConfigs && len(backup.Configs) > 0 {
		for _, config := range backup.Configs {
			if opts.DryRun {
				result.ConfigsCreated = append(result.ConfigsCreated, config.Name+" (dry-run)")
				continue
			}

			if err := m.restoreConfig(ctx, &config); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to restore config %s: %v", config.Name, err))
			} else {
				result.ConfigsCreated = append(result.ConfigsCreated, config.Name)
			}
		}
	}

	// Filter services if needed
	services := m.filterServicesForRestore(backup.Services, opts)

	// Determine restore order
	orderedServices, err := m.resolveRestoreOrder(services, opts.RespectDependencies)
	if err != nil {
		return nil, fmt.Errorf("resolve restore order: %w", err)
	}

	// Restore services in order
	for _, service := range orderedServices {
		if opts.DryRun {
			result.ServicesRestored = append(result.ServicesRestored, service.Name+" (dry-run)")
			continue
		}

		// Remove existing service if force is enabled
		if opts.Force {
			_ = m.removeService(ctx, service.Name)
		}

		if err := m.restoreService(ctx, &service); err != nil {
			result.Success = false
			result.ServicesFailed = append(result.ServicesFailed, ServiceRestoreFailure{
				ServiceName: service.Name,
				Error:       err.Error(),
			})
			m.logger.Error().Err(err).Str("service", service.Name).Msg("failed to restore service")
		} else {
			result.ServicesRestored = append(result.ServicesRestored, service.Name)
			m.logger.Info().Str("service", service.Name).Msg("service restored")
		}
	}

	result.Duration = time.Since(start)

	m.logger.Info().
		Int("services_restored", len(result.ServicesRestored)).
		Int("services_failed", len(result.ServicesFailed)).
		Dur("duration", result.Duration).
		Msg("swarm restore completed")

	return result, nil
}

// resolveRestoreOrder determines the order to restore services based on dependencies.
func (m *SwarmBackupManager) resolveRestoreOrder(services []SwarmService, respectDeps bool) ([]SwarmService, error) {
	if !respectDeps {
		return services, nil
	}

	// Build dependency graph
	serviceMap := make(map[string]*SwarmService)
	for i := range services {
		serviceMap[services[i].Name] = &services[i]
	}

	// Topological sort using Kahn's algorithm
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	for _, svc := range services {
		if _, exists := inDegree[svc.Name]; !exists {
			inDegree[svc.Name] = 0
		}
		for _, dep := range svc.DependsOn {
			if _, exists := serviceMap[dep]; exists {
				graph[dep] = append(graph[dep], svc.Name)
				inDegree[svc.Name]++
			}
		}
	}

	// Find all nodes with no incoming edges
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort queue for deterministic ordering
	sort.Strings(queue)

	var result []SwarmService
	for len(queue) > 0 {
		// Take first element
		name := queue[0]
		queue = queue[1:]

		if svc, exists := serviceMap[name]; exists {
			result = append(result, *svc)
		}

		// Reduce in-degree for dependent services
		dependents := graph[name]
		sort.Strings(dependents)
		for _, dependent := range dependents {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for cycles
	if len(result) != len(services) {
		return nil, ErrDependencyCycle
	}

	return result, nil
}

// restoreService restores a single service.
func (m *SwarmBackupManager) restoreService(ctx context.Context, service *SwarmService) error {
	args := []string{"service", "create", "--name", service.Name}

	// Add mode
	if service.Mode == "global" {
		args = append(args, "--mode", "global")
	} else {
		args = append(args, "--replicas", fmt.Sprintf("%d", service.Replicas))
	}

	// Add ports
	for _, port := range service.Ports {
		portSpec := fmt.Sprintf("%d:%d", port.PublishedPort, port.TargetPort)
		if port.Protocol != "" && port.Protocol != "tcp" {
			portSpec += "/" + port.Protocol
		}
		args = append(args, "--publish", portSpec)
	}

	// Add mounts
	for _, mount := range service.Mounts {
		mountSpec := fmt.Sprintf("type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target)
		if mount.ReadOnly {
			mountSpec += ",readonly"
		}
		args = append(args, "--mount", mountSpec)
	}

	// Add networks
	for _, network := range service.Networks {
		args = append(args, "--network", network)
	}

	// Add constraints
	for _, constraint := range service.Constraints {
		args = append(args, "--constraint", constraint)
	}

	// Add environment variables
	for _, env := range service.Env {
		args = append(args, "--env", env)
	}

	// Add labels
	for k, v := range service.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add secrets (by name, assuming they exist)
	for _, secret := range service.Secrets {
		secretSpec := secret.SecretName
		if secret.FileName != "" {
			secretSpec += ",target=" + secret.FileName
		}
		args = append(args, "--secret", secretSpec)
	}

	// Add configs (by name, assuming they exist)
	for _, config := range service.Configs {
		configSpec := config.ConfigName
		if config.FileName != "" {
			configSpec += ",target=" + config.FileName
		}
		args = append(args, "--config", configSpec)
	}

	// Add image
	args = append(args, service.Image)

	_, err := m.runDocker(ctx, args...)
	return err
}

// restoreNetwork restores a network.
func (m *SwarmBackupManager) restoreNetwork(ctx context.Context, network *SwarmNetwork) error {
	// Check if network already exists
	_, err := m.runDocker(ctx, "network", "inspect", network.Name)
	if err == nil {
		return nil // Already exists
	}

	args := []string{"network", "create", "--driver", network.Driver, "--scope", "swarm"}

	if network.Attachable {
		args = append(args, "--attachable")
	}

	if network.Internal {
		args = append(args, "--internal")
	}

	for k, v := range network.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range network.Options {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", k, v))
	}

	for _, ipam := range network.SwarmIPAMConfig {
		if ipam.Subnet != "" {
			args = append(args, "--subnet", ipam.Subnet)
		}
		if ipam.Gateway != "" {
			args = append(args, "--gateway", ipam.Gateway)
		}
		if ipam.IPRange != "" {
			args = append(args, "--ip-range", ipam.IPRange)
		}
	}

	args = append(args, network.Name)

	_, err = m.runDocker(ctx, args...)
	return err
}

// restoreVolume restores a volume.
func (m *SwarmBackupManager) restoreVolume(ctx context.Context, volume *SwarmVolume) error {
	// Check if volume already exists
	_, err := m.runDocker(ctx, "volume", "inspect", volume.Name)
	if err == nil {
		return nil // Already exists
	}

	args := []string{"volume", "create"}

	if volume.Driver != "" && volume.Driver != "local" {
		args = append(args, "--driver", volume.Driver)
	}

	for k, v := range volume.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range volume.Options {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, volume.Name)

	_, err = m.runDocker(ctx, args...)
	return err
}

// restoreConfig restores a Docker config.
func (m *SwarmBackupManager) restoreConfig(ctx context.Context, config *SwarmConfig) error {
	// Check if config already exists
	_, err := m.runDocker(ctx, "config", "inspect", config.Name)
	if err == nil {
		return nil // Already exists
	}

	// Create temporary file with config data
	tmpFile, err := os.CreateTemp("", "docker-config-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Config data is base64 encoded
	if _, err := tmpFile.WriteString(config.Data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write config data: %w", err)
	}
	tmpFile.Close()

	args := []string{"config", "create"}

	for k, v := range config.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, config.Name, tmpFile.Name())

	_, err = m.runDocker(ctx, args...)
	return err
}

// removeService removes an existing service.
func (m *SwarmBackupManager) removeService(ctx context.Context, serviceName string) error {
	_, err := m.runDocker(ctx, "service", "rm", serviceName)
	return err
}

// filterServices filters services based on config filters.
func (m *SwarmBackupManager) filterServices(services []SwarmService) []SwarmService {
	if len(m.config.StackFilter) == 0 && len(m.config.ServiceFilter) == 0 {
		return services
	}

	var filtered []SwarmService
	for _, svc := range services {
		// Check stack filter
		if len(m.config.StackFilter) > 0 {
			stackMatch := false
			for _, stack := range m.config.StackFilter {
				if svc.StackName == stack {
					stackMatch = true
					break
				}
			}
			if !stackMatch {
				continue
			}
		}

		// Check service filter
		if len(m.config.ServiceFilter) > 0 {
			svcMatch := false
			for _, name := range m.config.ServiceFilter {
				if svc.Name == name {
					svcMatch = true
					break
				}
			}
			if !svcMatch {
				continue
			}
		}

		filtered = append(filtered, svc)
	}

	return filtered
}

// filterStacks filters stacks based on config filters.
func (m *SwarmBackupManager) filterStacks(stacks []SwarmStack) []SwarmStack {
	if len(m.config.StackFilter) == 0 {
		return stacks
	}

	var filtered []SwarmStack
	for _, stack := range stacks {
		for _, name := range m.config.StackFilter {
			if stack.Name == name {
				filtered = append(filtered, stack)
				break
			}
		}
	}

	return filtered
}

// filterServicesForRestore filters services for restore based on options.
func (m *SwarmBackupManager) filterServicesForRestore(services []SwarmService, opts *SwarmRestoreOptions) []SwarmService {
	if len(opts.StackFilter) == 0 && len(opts.ServiceFilter) == 0 {
		return services
	}

	var filtered []SwarmService
	for _, svc := range services {
		// Check stack filter
		if len(opts.StackFilter) > 0 {
			stackMatch := false
			for _, stack := range opts.StackFilter {
				if svc.StackName == stack {
					stackMatch = true
					break
				}
			}
			if !stackMatch {
				continue
			}
		}

		// Check service filter
		if len(opts.ServiceFilter) > 0 {
			svcMatch := false
			for _, name := range opts.ServiceFilter {
				if svc.Name == name {
					svcMatch = true
					break
				}
			}
			if !svcMatch {
				continue
			}
		}

		filtered = append(filtered, svc)
	}

	return filtered
}

// SaveBackup saves a backup to a file.
func (m *SwarmBackupManager) SaveBackup(backup *SwarmBackup, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	m.logger.Info().Str("path", path).Msg("backup saved to file")
	return nil
}

// LoadBackup loads a backup from a file.
func (m *SwarmBackupManager) LoadBackup(path string) (*SwarmBackup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read backup file: %w", err)
	}

	var backup SwarmBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("parse backup file: %w", err)
	}

	// Validate backup
	if backup.Metadata.Version == "" {
		return nil, ErrInvalidBackupData
	}

	return &backup, nil
}

// runDocker executes a docker command and returns the output.
func (m *SwarmBackupManager) runDocker(ctx context.Context, args ...string) ([]byte, error) {
	// Add host if configured
	if m.config.DockerHost != "" && m.config.DockerHost != "unix:///var/run/docker.sock" {
		args = append([]string{"-H", m.config.DockerHost}, args...)
	}

	cmd := exec.CommandContext(ctx, m.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	m.logger.Debug().
		Str("command", m.binary).
		Strs("args", args).
		Msg("executing docker command")

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	return stdout.Bytes(), nil
}

// SwarmAgentMode represents a Swarm-specific agent operating mode.
type SwarmAgentMode struct {
	manager *SwarmBackupManager
	logger  zerolog.Logger
	config  *SwarmBackupConfig
}

// NewSwarmAgentMode creates a new Swarm agent mode handler.
func NewSwarmAgentMode(config *SwarmBackupConfig, logger zerolog.Logger) *SwarmAgentMode {
	if config == nil {
		config = DefaultSwarmBackupConfig()
	}
	return &SwarmAgentMode{
		manager: NewSwarmBackupManager(config, logger),
		logger:  logger.With().Str("mode", "swarm").Logger(),
		config:  config,
	}
}

// IsAvailable checks if Swarm mode is available on this node.
func (m *SwarmAgentMode) IsAvailable(ctx context.Context) bool {
	isManager, err := m.manager.IsSwarmManager(ctx)
	if err != nil {
		m.logger.Debug().Err(err).Msg("swarm mode not available")
		return false
	}
	return isManager
}

// PerformBackup performs a Swarm cluster backup.
func (m *SwarmAgentMode) PerformBackup(ctx context.Context) (*SwarmBackup, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.Backup(ctx)
}

// PerformRestore performs a Swarm cluster restore.
func (m *SwarmAgentMode) PerformRestore(ctx context.Context, backup *SwarmBackup, opts *SwarmRestoreOptions) (*SwarmRestoreResult, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.Restore(ctx, backup, opts)
}

// GetClusterInfo returns information about the current Swarm cluster.
func (m *SwarmAgentMode) GetClusterInfo(ctx context.Context) (*ClusterState, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.backupClusterState(ctx)
}

// ListServices returns all services in the cluster.
func (m *SwarmAgentMode) ListServices(ctx context.Context) ([]SwarmService, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.backupServices(ctx)
}

// ListStacks returns all stacks in the cluster.
func (m *SwarmAgentMode) ListStacks(ctx context.Context) ([]SwarmStack, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.backupStacks(ctx)
}

// ListNodes returns all nodes in the cluster.
func (m *SwarmAgentMode) ListNodes(ctx context.Context) ([]SwarmNode, error) {
	if !m.IsAvailable(ctx) {
		return nil, ErrNotSwarmManager
	}

	return m.manager.backupNodes(ctx)
}
