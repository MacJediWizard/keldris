// Package docker provides Docker network configuration backup and restore functionality.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// NetworkDriver represents the Docker network driver type.
type NetworkDriver string

const (
	// NetworkDriverBridge is the default bridge driver.
	NetworkDriverBridge NetworkDriver = "bridge"
	// NetworkDriverHost is the host driver (shares host networking).
	NetworkDriverHost NetworkDriver = "host"
	// NetworkDriverOverlay is for multi-host networking (Swarm).
	NetworkDriverOverlay NetworkDriver = "overlay"
	// NetworkDriverMacvlan allows assigning MAC addresses to containers.
	NetworkDriverMacvlan NetworkDriver = "macvlan"
	// NetworkDriverIPvlan is similar to macvlan but shares MAC address.
	NetworkDriverIPvlan NetworkDriver = "ipvlan"
	// NetworkDriverNone disables networking.
	NetworkDriverNone NetworkDriver = "none"
)

// ErrNetworkNotFound is returned when a network cannot be found.
var ErrNetworkNotFound = errors.New("network not found")

// ErrNetworkConflict is returned when a network conflicts with an existing one.
var ErrNetworkConflict = errors.New("network conflict detected")

// NetworkIPAMSubnet represents a single IPAM subnet configuration for a network.
type NetworkIPAMSubnet struct {
	Subnet     string            `json:"subnet,omitempty"`
	IPRange    string            `json:"ip_range,omitempty"`
	Gateway    string            `json:"gateway,omitempty"`
	AuxAddress map[string]string `json:"aux_addresses,omitempty"`
}

// NetworkIPAM represents the IP Address Management settings for a network.
type NetworkIPAM struct {
	Driver  string              `json:"driver,omitempty"`
	Config  []NetworkIPAMSubnet `json:"config,omitempty"`
	Options map[string]string   `json:"options,omitempty"`
}

// NetworkDefinition represents a Docker network definition captured during backup.
type NetworkDefinition struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     NetworkDriver     `json:"driver"`
	Scope      string            `json:"scope"`
	EnableIPv6 bool              `json:"enable_ipv6"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Ingress    bool              `json:"ingress"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	IPAM       *NetworkIPAM      `json:"ipam,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// ContainerEndpoint represents a container's connection to a network.
type ContainerEndpoint struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	EndpointID    string `json:"endpoint_id"`
	MacAddress    string `json:"mac_address,omitempty"`
	IPv4Address   string `json:"ipv4_address,omitempty"`
	IPv6Address   string `json:"ipv6_address,omitempty"`
	IsStatic      bool   `json:"is_static"` // True if IP was statically assigned
}

// NetworkAssignment represents container assignments to a network.
type NetworkAssignment struct {
	NetworkID   string              `json:"network_id"`
	NetworkName string              `json:"network_name"`
	Endpoints   []ContainerEndpoint `json:"endpoints"`
}

// NetworkBackup represents a complete backup of Docker network configuration.
type NetworkBackup struct {
	Networks     []NetworkDefinition `json:"networks"`
	Assignments  []NetworkAssignment `json:"assignments"`
	BackupTime   time.Time           `json:"backup_time"`
	DockerHost   string              `json:"docker_host,omitempty"`
	DockerAPIVer string              `json:"docker_api_version,omitempty"`
}

// NetworkConflict represents a detected conflict during restore.
type NetworkConflict struct {
	NetworkName   string `json:"network_name"`
	ConflictType  string `json:"conflict_type"` // "name", "subnet", "gateway"
	ExistingValue string `json:"existing_value"`
	BackupValue   string `json:"backup_value"`
	Resolution    string `json:"resolution,omitempty"`
}

// NetworkRestoreOptions configures the network restore operation.
type NetworkRestoreOptions struct {
	// DryRun only previews what would be restored without making changes.
	DryRun bool
	// SkipExisting skips networks that already exist.
	SkipExisting bool
	// Force recreates networks even if they exist (removes and recreates).
	Force bool
	// RestoreAssignments attempts to reconnect containers to networks.
	RestoreAssignments bool
	// RestoreStaticIPs attempts to restore static IP assignments.
	RestoreStaticIPs bool
	// NetworkFilter only restores networks matching these names (empty = all).
	NetworkFilter []string
}

// NetworkRestoreResult contains the results of a network restore operation.
type NetworkRestoreResult struct {
	NetworksRestored    []string          `json:"networks_restored"`
	NetworksSkipped     []string          `json:"networks_skipped"`
	NetworksFailed      []string          `json:"networks_failed"`
	AssignmentsRestored int               `json:"assignments_restored"`
	AssignmentsFailed   int               `json:"assignments_failed"`
	Conflicts           []NetworkConflict `json:"conflicts,omitempty"`
	Errors              []string          `json:"errors,omitempty"`
}

// Networks provides Docker network backup and restore functionality.
type Networks struct {
	logger     zerolog.Logger
	dockerPath string
}

// NewNetworks creates a new Networks instance.
func NewNetworks(logger zerolog.Logger) *Networks {
	return &Networks{
		logger:     logger.With().Str("component", "docker_networks").Logger(),
		dockerPath: "docker",
	}
}

// NewNetworksWithPath creates a new Networks instance with a custom Docker path.
func NewNetworksWithPath(dockerPath string, logger zerolog.Logger) *Networks {
	return &Networks{
		logger:     logger.With().Str("component", "docker_networks").Logger(),
		dockerPath: dockerPath,
	}
}

// CheckDockerAvailable verifies that Docker is available and accessible.
func (n *Networks) CheckDockerAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, n.dockerPath, "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return ErrDockerNotAvailable
	}
	return nil
}

// Backup creates a complete backup of Docker network configurations.
func (n *Networks) Backup(ctx context.Context) (*NetworkBackup, error) {
	n.logger.Info().Msg("starting Docker network backup")

	if err := n.CheckDockerAvailable(ctx); err != nil {
		return nil, err
	}

	// Get all networks
	networks, err := n.listNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}

	// Get detailed configuration for each network
	var configs []NetworkDefinition
	var assignments []NetworkAssignment

	for _, netName := range networks {
		// Skip default networks that are always present
		if isDefaultNetwork(netName) {
			n.logger.Debug().Str("network", netName).Msg("skipping default network")
			continue
		}

		config, err := n.inspectNetwork(ctx, netName)
		if err != nil {
			n.logger.Warn().Err(err).Str("network", netName).Msg("failed to inspect network")
			continue
		}

		configs = append(configs, *config)

		// Get container assignments for this network
		assignment, err := n.getNetworkAssignments(ctx, config)
		if err != nil {
			n.logger.Warn().Err(err).Str("network", netName).Msg("failed to get assignments")
			continue
		}

		if len(assignment.Endpoints) > 0 {
			assignments = append(assignments, *assignment)
		}
	}

	// Get Docker host and API version info
	dockerHost, apiVersion := n.getDockerInfo(ctx)

	backup := &NetworkBackup{
		Networks:     configs,
		Assignments:  assignments,
		BackupTime:   time.Now(),
		DockerHost:   dockerHost,
		DockerAPIVer: apiVersion,
	}

	n.logger.Info().
		Int("networks", len(configs)).
		Int("assignments", len(assignments)).
		Msg("Docker network backup completed")

	return backup, nil
}

// listNetworks returns all Docker network names.
func (n *Networks) listNetworks(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, n.dockerPath, "network", "ls", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker network ls: %w", err)
	}

	var networks []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			networks = append(networks, line)
		}
	}

	return networks, nil
}

// inspectNetwork returns detailed configuration for a network.
func (n *Networks) inspectNetwork(ctx context.Context, name string) (*NetworkDefinition, error) {
	cmd := exec.CommandContext(ctx, n.dockerPath, "network", "inspect", name, "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker network inspect: %w", err)
	}

	// Docker inspect returns a raw JSON object
	var raw dockerNetworkInspect
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("parse network inspect: %w", err)
	}

	config := &NetworkDefinition{
		ID:         raw.ID,
		Name:       raw.Name,
		Driver:     NetworkDriver(raw.Driver),
		Scope:      raw.Scope,
		EnableIPv6: raw.EnableIPv6,
		Internal:   raw.Internal,
		Attachable: raw.Attachable,
		Ingress:    raw.Ingress,
		Labels:     raw.Labels,
		Options:    raw.Options,
	}

	// Parse IPAM configuration
	if raw.IPAM.Driver != "" || len(raw.IPAM.Config) > 0 {
		config.IPAM = &NetworkIPAM{
			Driver:  raw.IPAM.Driver,
			Options: raw.IPAM.Options,
		}
		for _, ipamCfg := range raw.IPAM.Config {
			config.IPAM.Config = append(config.IPAM.Config, NetworkIPAMSubnet{
				Subnet:     ipamCfg.Subnet,
				IPRange:    ipamCfg.IPRange,
				Gateway:    ipamCfg.Gateway,
				AuxAddress: ipamCfg.AuxiliaryAddresses,
			})
		}
	}

	// Parse creation time
	if raw.Created != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.Created); err == nil {
			config.CreatedAt = t
		}
	}

	return config, nil
}

// dockerNetworkInspect represents the raw Docker network inspect output.
type dockerNetworkInspect struct {
	ID         string                         `json:"Id"`
	Name       string                         `json:"Name"`
	Created    string                         `json:"Created"`
	Scope      string                         `json:"Scope"`
	Driver     string                         `json:"Driver"`
	EnableIPv6 bool                           `json:"EnableIPv6"`
	IPAM       dockerIPAM                     `json:"IPAM"`
	Internal   bool                           `json:"Internal"`
	Attachable bool                           `json:"Attachable"`
	Ingress    bool                           `json:"Ingress"`
	Options    map[string]string              `json:"Options"`
	Labels     map[string]string              `json:"Labels"`
	Containers map[string]dockerContainer     `json:"Containers"`
}

type dockerIPAM struct {
	Driver  string             `json:"Driver"`
	Options map[string]string  `json:"Options"`
	Config  []dockerIPAMConfig `json:"Config"`
}

type dockerIPAMConfig struct {
	Subnet             string            `json:"Subnet"`
	IPRange            string            `json:"IPRange"`
	Gateway            string            `json:"Gateway"`
	AuxiliaryAddresses map[string]string `json:"AuxiliaryAddresses"`
}

type dockerContainer struct {
	Name        string `json:"Name"`
	EndpointID  string `json:"EndpointID"`
	MacAddress  string `json:"MacAddress"`
	IPv4Address string `json:"IPv4Address"`
	IPv6Address string `json:"IPv6Address"`
}

// getNetworkAssignments returns container assignments for a network.
func (n *Networks) getNetworkAssignments(ctx context.Context, config *NetworkDefinition) (*NetworkAssignment, error) {
	cmd := exec.CommandContext(ctx, n.dockerPath, "network", "inspect", config.Name, "--format", "{{json .Containers}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker network inspect containers: %w", err)
	}

	var containers map[string]dockerContainer
	if err := json.Unmarshal(output, &containers); err != nil {
		// Empty or null containers
		return &NetworkAssignment{
			NetworkID:   config.ID,
			NetworkName: config.Name,
			Endpoints:   []ContainerEndpoint{},
		}, nil
	}

	var endpoints []ContainerEndpoint
	for containerID, container := range containers {
		endpoint := ContainerEndpoint{
			ContainerID:   containerID,
			ContainerName: strings.TrimPrefix(container.Name, "/"),
			EndpointID:    container.EndpointID,
			MacAddress:    container.MacAddress,
			IPv4Address:   stripCIDR(container.IPv4Address),
			IPv6Address:   stripCIDR(container.IPv6Address),
		}

		// Determine if IP was statically assigned
		endpoint.IsStatic = n.isStaticIP(ctx, containerID, config.Name)

		endpoints = append(endpoints, endpoint)
	}

	return &NetworkAssignment{
		NetworkID:   config.ID,
		NetworkName: config.Name,
		Endpoints:   endpoints,
	}, nil
}

// isStaticIP checks if a container's IP on a network was statically assigned.
func (n *Networks) isStaticIP(ctx context.Context, containerID, networkName string) bool {
	// Inspect container to check network settings
	cmd := exec.CommandContext(ctx, n.dockerPath, "inspect", containerID,
		"--format", fmt.Sprintf("{{index .NetworkSettings.Networks \"%s\"}}", networkName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If IPAMConfig is set, it was a static assignment
	return strings.Contains(string(output), "IPAMConfig")
}

// stripCIDR removes the CIDR notation from an IP address.
func stripCIDR(addr string) string {
	if idx := strings.Index(addr, "/"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// getDockerInfo returns Docker host and API version.
func (n *Networks) getDockerInfo(ctx context.Context) (host, apiVersion string) {
	cmd := exec.CommandContext(ctx, n.dockerPath, "info", "--format", "{{.Name}}")
	if output, err := cmd.Output(); err == nil {
		host = strings.TrimSpace(string(output))
	}

	cmd = exec.CommandContext(ctx, n.dockerPath, "version", "--format", "{{.Server.APIVersion}}")
	if output, err := cmd.Output(); err == nil {
		apiVersion = strings.TrimSpace(string(output))
	}

	return host, apiVersion
}

// isDefaultNetwork returns true if the network is a Docker default.
func isDefaultNetwork(name string) bool {
	defaults := []string{"bridge", "host", "none"}
	for _, d := range defaults {
		if name == d {
			return true
		}
	}
	return false
}

// Restore restores Docker networks from a backup.
func (n *Networks) Restore(ctx context.Context, backup *NetworkBackup, opts NetworkRestoreOptions) (*NetworkRestoreResult, error) {
	n.logger.Info().
		Bool("dry_run", opts.DryRun).
		Bool("force", opts.Force).
		Bool("restore_assignments", opts.RestoreAssignments).
		Msg("starting Docker network restore")

	if err := n.CheckDockerAvailable(ctx); err != nil {
		return nil, err
	}

	result := &NetworkRestoreResult{
		NetworksRestored: []string{},
		NetworksSkipped:  []string{},
		NetworksFailed:   []string{},
	}

	// Detect conflicts before restore
	conflicts, err := n.DetectConflicts(ctx, backup)
	if err != nil {
		n.logger.Warn().Err(err).Msg("failed to detect conflicts")
	}
	result.Conflicts = conflicts

	// Restore networks
	for _, netDef := range backup.Networks {
		// Apply network filter if specified
		if len(opts.NetworkFilter) > 0 && !contains(opts.NetworkFilter, netDef.Name) {
			n.logger.Debug().Str("network", netDef.Name).Msg("skipping filtered network")
			result.NetworksSkipped = append(result.NetworksSkipped, netDef.Name)
			continue
		}

		// Check if network exists
		exists, err := n.networkExists(ctx, netDef.Name)
		if err != nil {
			n.logger.Warn().Err(err).Str("network", netDef.Name).Msg("failed to check network existence")
		}

		if exists {
			if opts.SkipExisting {
				n.logger.Info().Str("network", netDef.Name).Msg("skipping existing network")
				result.NetworksSkipped = append(result.NetworksSkipped, netDef.Name)
				continue
			}

			if opts.Force {
				if !opts.DryRun {
					if err := n.removeNetwork(ctx, netDef.Name); err != nil {
						n.logger.Error().Err(err).Str("network", netDef.Name).Msg("failed to remove existing network")
						result.NetworksFailed = append(result.NetworksFailed, netDef.Name)
						result.Errors = append(result.Errors, fmt.Sprintf("remove %s: %v", netDef.Name, err))
						continue
					}
				}
				n.logger.Info().Str("network", netDef.Name).Msg("removed existing network for recreation")
			} else {
				n.logger.Info().Str("network", netDef.Name).Msg("network exists, skipping")
				result.NetworksSkipped = append(result.NetworksSkipped, netDef.Name)
				continue
			}
		}

		// Create the network
		if opts.DryRun {
			n.logger.Info().Str("network", netDef.Name).Msg("[dry-run] would create network")
			result.NetworksRestored = append(result.NetworksRestored, netDef.Name)
		} else {
			if err := n.createNetwork(ctx, &netDef); err != nil {
				n.logger.Error().Err(err).Str("network", netDef.Name).Msg("failed to create network")
				result.NetworksFailed = append(result.NetworksFailed, netDef.Name)
				result.Errors = append(result.Errors, fmt.Sprintf("create %s: %v", netDef.Name, err))
				continue
			}
			n.logger.Info().Str("network", netDef.Name).Msg("created network")
			result.NetworksRestored = append(result.NetworksRestored, netDef.Name)
		}
	}

	// Restore container assignments if requested
	if opts.RestoreAssignments {
		for _, assignment := range backup.Assignments {
			for _, endpoint := range assignment.Endpoints {
				if opts.DryRun {
					n.logger.Info().
						Str("container", endpoint.ContainerName).
						Str("network", assignment.NetworkName).
						Msg("[dry-run] would connect container")
					result.AssignmentsRestored++
					continue
				}

				err := n.connectContainer(ctx, assignment.NetworkName, endpoint, opts.RestoreStaticIPs)
				if err != nil {
					n.logger.Warn().Err(err).
						Str("container", endpoint.ContainerName).
						Str("network", assignment.NetworkName).
						Msg("failed to connect container")
					result.AssignmentsFailed++
					result.Errors = append(result.Errors, fmt.Sprintf("connect %s to %s: %v",
						endpoint.ContainerName, assignment.NetworkName, err))
					continue
				}
				result.AssignmentsRestored++
			}
		}
	}

	n.logger.Info().
		Int("restored", len(result.NetworksRestored)).
		Int("skipped", len(result.NetworksSkipped)).
		Int("failed", len(result.NetworksFailed)).
		Int("assignments_restored", result.AssignmentsRestored).
		Msg("Docker network restore completed")

	return result, nil
}

// networkExists checks if a network with the given name exists.
func (n *Networks) networkExists(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, n.dockerPath, "network", "inspect", name)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// removeNetwork removes a Docker network.
func (n *Networks) removeNetwork(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, n.dockerPath, "network", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// createNetwork creates a Docker network from configuration.
func (n *Networks) createNetwork(ctx context.Context, config *NetworkDefinition) error {
	args := []string{"network", "create"}

	// Driver
	args = append(args, "--driver", string(config.Driver))

	// Network options
	if config.Internal {
		args = append(args, "--internal")
	}
	if config.Attachable {
		args = append(args, "--attachable")
	}
	if config.EnableIPv6 {
		args = append(args, "--ipv6")
	}

	// Labels
	for key, value := range config.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Driver options
	for key, value := range config.Options {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", key, value))
	}

	// IPAM configuration
	if config.IPAM != nil {
		if config.IPAM.Driver != "" && config.IPAM.Driver != "default" {
			args = append(args, "--ipam-driver", config.IPAM.Driver)
		}

		for _, ipamCfg := range config.IPAM.Config {
			if ipamCfg.Subnet != "" {
				args = append(args, "--subnet", ipamCfg.Subnet)
			}
			if ipamCfg.IPRange != "" {
				args = append(args, "--ip-range", ipamCfg.IPRange)
			}
			if ipamCfg.Gateway != "" {
				args = append(args, "--gateway", ipamCfg.Gateway)
			}
			for name, addr := range ipamCfg.AuxAddress {
				args = append(args, "--aux-address", fmt.Sprintf("%s=%s", name, addr))
			}
		}

		for key, value := range config.IPAM.Options {
			args = append(args, "--ipam-opt", fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Network name
	args = append(args, config.Name)

	cmd := exec.CommandContext(ctx, n.dockerPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
}

// connectContainer connects a container to a network.
func (n *Networks) connectContainer(ctx context.Context, networkName string, endpoint ContainerEndpoint, restoreStaticIP bool) error {
	// Check if container exists
	cmd := exec.CommandContext(ctx, n.dockerPath, "inspect", endpoint.ContainerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container %s not found", endpoint.ContainerName)
	}

	args := []string{"network", "connect"}

	// Restore static IP if requested and it was originally static
	if restoreStaticIP && endpoint.IsStatic && endpoint.IPv4Address != "" {
		args = append(args, "--ip", endpoint.IPv4Address)
	}

	// Restore IPv6 address if it was static
	if restoreStaticIP && endpoint.IsStatic && endpoint.IPv6Address != "" {
		args = append(args, "--ip6", endpoint.IPv6Address)
	}

	args = append(args, networkName, endpoint.ContainerName)

	cmd = exec.CommandContext(ctx, n.dockerPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errStr := stderr.String()
		// Ignore if already connected
		if strings.Contains(errStr, "already exists") {
			return nil
		}
		return fmt.Errorf("%w: %s", err, errStr)
	}

	return nil
}

// DetectConflicts checks for conflicts between backup and existing networks.
func (n *Networks) DetectConflicts(ctx context.Context, backup *NetworkBackup) ([]NetworkConflict, error) {
	n.logger.Debug().Msg("detecting network conflicts")

	var conflicts []NetworkConflict

	// Get existing networks
	existingNetworks, err := n.listNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list existing networks: %w", err)
	}

	// Build a map of existing network configurations
	existingConfigs := make(map[string]*NetworkDefinition)
	for _, name := range existingNetworks {
		if isDefaultNetwork(name) {
			continue
		}
		config, err := n.inspectNetwork(ctx, name)
		if err != nil {
			continue
		}
		existingConfigs[name] = config
	}

	// Check each backup network for conflicts
	for _, backupNet := range backup.Networks {
		// Check name conflict
		if existing, ok := existingConfigs[backupNet.Name]; ok {
			// Same name exists - check for deeper conflicts
			conflicts = append(conflicts, NetworkConflict{
				NetworkName:   backupNet.Name,
				ConflictType:  "name",
				ExistingValue: existing.ID,
				BackupValue:   backupNet.ID,
				Resolution:    "Network with same name already exists",
			})

			// Check subnet conflicts
			if backupNet.IPAM != nil && existing.IPAM != nil {
				for _, backupIPAM := range backupNet.IPAM.Config {
					for _, existingIPAM := range existing.IPAM.Config {
						if backupIPAM.Subnet != "" && existingIPAM.Subnet != "" {
							if subnetOverlaps(backupIPAM.Subnet, existingIPAM.Subnet) && backupIPAM.Subnet != existingIPAM.Subnet {
								conflicts = append(conflicts, NetworkConflict{
									NetworkName:   backupNet.Name,
									ConflictType:  "subnet",
									ExistingValue: existingIPAM.Subnet,
									BackupValue:   backupIPAM.Subnet,
									Resolution:    "Subnet configuration differs",
								})
							}
						}
					}
				}
			}
		}

		// Check for subnet conflicts with other existing networks
		if backupNet.IPAM != nil {
			for existingName, existing := range existingConfigs {
				if existingName == backupNet.Name {
					continue
				}
				if existing.IPAM == nil {
					continue
				}

				for _, backupIPAM := range backupNet.IPAM.Config {
					for _, existingIPAM := range existing.IPAM.Config {
						if backupIPAM.Subnet != "" && existingIPAM.Subnet != "" {
							if subnetOverlaps(backupIPAM.Subnet, existingIPAM.Subnet) {
								conflicts = append(conflicts, NetworkConflict{
									NetworkName:   backupNet.Name,
									ConflictType:  "subnet_overlap",
									ExistingValue: fmt.Sprintf("%s uses %s", existingName, existingIPAM.Subnet),
									BackupValue:   backupIPAM.Subnet,
									Resolution:    "Subnet overlaps with existing network",
								})
							}
						}
					}
				}
			}
		}
	}

	n.logger.Debug().Int("conflicts", len(conflicts)).Msg("conflict detection completed")
	return conflicts, nil
}

// subnetOverlaps checks if two CIDR subnets overlap.
func subnetOverlaps(cidr1, cidr2 string) bool {
	_, net1, err1 := net.ParseCIDR(cidr1)
	_, net2, err2 := net.ParseCIDR(cidr2)

	if err1 != nil || err2 != nil {
		return false
	}

	return net1.Contains(net2.IP) || net2.Contains(net1.IP)
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// ExportJSON exports the backup to JSON format.
func (n *Networks) ExportJSON(backup *NetworkBackup) ([]byte, error) {
	return json.MarshalIndent(backup, "", "  ")
}

// ImportJSON imports a backup from JSON format.
func (n *Networks) ImportJSON(data []byte) (*NetworkBackup, error) {
	var backup NetworkBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("parse backup JSON: %w", err)
	}
	return &backup, nil
}

// ListNetworksByDriver returns networks filtered by driver type.
func (n *Networks) ListNetworksByDriver(ctx context.Context, driver NetworkDriver) ([]NetworkDefinition, error) {
	backup, err := n.Backup(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []NetworkDefinition
	for _, netDef := range backup.Networks {
		if netDef.Driver == driver {
			filtered = append(filtered, netDef)
		}
	}

	return filtered, nil
}

// GetNetworkDefinition returns the definition for a specific network.
func (n *Networks) GetNetworkDefinition(ctx context.Context, name string) (*NetworkDefinition, error) {
	exists, err := n.networkExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNetworkNotFound
	}

	return n.inspectNetwork(ctx, name)
}

// ValidateBackup validates a backup for potential restore issues.
func (n *Networks) ValidateBackup(backup *NetworkBackup) []string {
	var issues []string

	for _, netDef := range backup.Networks {
		// Validate driver
		switch netDef.Driver {
		case NetworkDriverBridge, NetworkDriverHost, NetworkDriverOverlay,
			NetworkDriverMacvlan, NetworkDriverIPvlan, NetworkDriverNone:
			// Valid driver
		default:
			issues = append(issues, fmt.Sprintf("network %s has unknown driver: %s", netDef.Name, netDef.Driver))
		}

		// Validate IPAM configuration
		if netDef.IPAM != nil {
			for _, cfg := range netDef.IPAM.Config {
				if cfg.Subnet != "" {
					if _, _, err := net.ParseCIDR(cfg.Subnet); err != nil {
						issues = append(issues, fmt.Sprintf("network %s has invalid subnet: %s", netDef.Name, cfg.Subnet))
					}
				}
				if cfg.Gateway != "" {
					if ip := net.ParseIP(cfg.Gateway); ip == nil {
						issues = append(issues, fmt.Sprintf("network %s has invalid gateway: %s", netDef.Name, cfg.Gateway))
					}
				}
			}
		}

		// Validate network names
		if netDef.Name == "" {
			issues = append(issues, "network has empty name")
		}
	}

	return issues
}
