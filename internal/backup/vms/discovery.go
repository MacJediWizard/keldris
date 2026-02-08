package vms

import (
	"context"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// ProxmoxDiscovery handles discovery of Proxmox VMs and containers.
type ProxmoxDiscovery struct {
	logger zerolog.Logger
}

// NewProxmoxDiscovery creates a new Proxmox discovery service.
func NewProxmoxDiscovery(logger zerolog.Logger) *ProxmoxDiscovery {
	return &ProxmoxDiscovery{
		logger: logger.With().Str("component", "proxmox_discovery").Logger(),
	}
}

// DiscoveryResult contains the results of a Proxmox discovery operation.
type DiscoveryResult struct {
	Success      bool
	ProxmoxInfo  *models.ProxmoxInfo
	ErrorMessage string
}

// Discover connects to a Proxmox server and discovers all VMs and containers.
func (d *ProxmoxDiscovery) Discover(ctx context.Context, client *ProxmoxClient, connectionID string) *DiscoveryResult {
	result := &DiscoveryResult{
		Success: true,
	}

	// Get version info
	version, err := client.GetVersion(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to get Proxmox version")
		result.Success = false
		result.ErrorMessage = err.Error()
		result.ProxmoxInfo = &models.ProxmoxInfo{
			Available: false,
			Error:     err.Error(),
		}
		return result
	}

	d.logger.Info().
		Str("version", version.Version).
		Str("release", version.Release).
		Msg("connected to Proxmox")

	// Get all VMs and containers
	vms, err := client.ListAll(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to list VMs")
		result.Success = false
		result.ErrorMessage = err.Error()
		result.ProxmoxInfo = &models.ProxmoxInfo{
			Available: true,
			Version:   version.Version,
			Error:     err.Error(),
		}
		return result
	}

	d.logger.Info().
		Int("total", len(vms)).
		Msg("discovered Proxmox VMs and containers")

	// Convert to model info
	result.ProxmoxInfo = ToModelInfo(
		vms,
		client.config.Host,
		client.config.Node,
		version.Version,
		connectionID,
	)

	return result
}

// DiscoverFromConnection performs discovery using a ProxmoxConnection model.
func (d *ProxmoxDiscovery) DiscoverFromConnection(
	ctx context.Context,
	conn *models.ProxmoxConnection,
	tokenSecret string,
) *DiscoveryResult {
	client := NewProxmoxClientFromConnection(conn, tokenSecret, d.logger)
	return d.Discover(ctx, client, conn.ID.String())
}

// RefreshAgent updates an agent's Proxmox information.
func (d *ProxmoxDiscovery) RefreshAgent(
	ctx context.Context,
	agent *models.Agent,
	conn *models.ProxmoxConnection,
	tokenSecret string,
) error {
	result := d.DiscoverFromConnection(ctx, conn, tokenSecret)

	if result.ProxmoxInfo != nil {
		agent.ProxmoxInfo = result.ProxmoxInfo
	}

	return nil
}
