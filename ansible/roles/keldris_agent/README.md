# Keldris Agent Ansible Role

An Ansible role to install and configure the Keldris backup agent on Linux and macOS systems.

## Requirements

- Ansible 2.10 or higher
- Target systems: Linux (systemd) or macOS 12+
- Network access to download agent binary and connect to Keldris server

## Role Variables

### Required Variables

| Variable | Description |
|----------|-------------|
| `keldris_server_url` | URL of the Keldris server (e.g., `https://backup.example.com`) |
| `keldris_api_key` | API key for authentication with the server |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `keldris_agent_version` | `latest` | Version to install |
| `keldris_download_url` | `https://releases.keldris.io/agent` | Base URL for binary downloads |
| `keldris_install_dir` | `/usr/local/bin` | Installation directory |
| `keldris_binary_name` | `keldris-agent` | Binary filename |
| `keldris_config_dir_linux` | `/etc/keldris` | Config directory (Linux) |
| `keldris_config_dir_macos` | `~/.config/keldris` | Config directory (macOS) |
| `keldris_auto_check_update` | `true` | Enable automatic update checks |
| `keldris_hostname` | `{{ ansible_hostname }}` | Hostname to register |
| `keldris_service_enabled` | `true` | Enable service at boot |
| `keldris_service_started` | `true` | Start service after install |
| `keldris_register_agent` | `true` | Register with server |
| `keldris_force_registration` | `false` | Force re-registration |
| `keldris_remove_quarantine` | `true` | Remove quarantine attribute (macOS) |

## Dependencies

None.

## Example Playbook

```yaml
- hosts: backup_clients
  roles:
    - role: keldris_agent
      vars:
        keldris_server_url: "https://backup.example.com"
        keldris_api_key: "your-api-key"
        keldris_agent_version: "latest"
```

## Platform Support

- **Linux**: Ubuntu 20.04+, Debian 11+, RHEL/CentOS 8+ (systemd)
- **macOS**: 12 (Monterey) through 15 (Sequoia) (launchd)

## Service Management

### Linux (systemd)

```bash
sudo systemctl status keldris-agent
sudo systemctl start keldris-agent
sudo systemctl stop keldris-agent
sudo journalctl -u keldris-agent -f
```

### macOS (launchd)

```bash
launchctl list | grep keldris
launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
tail -f ~/.config/keldris/agent.log
```

## License

AGPL-3.0

## Author

MacJediWizard
