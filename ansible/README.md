# Keldris Ansible Roles

This directory contains Ansible roles for deploying and managing Keldris components.

## Available Roles

### keldris_agent

Installs and configures the Keldris backup agent on Linux and macOS systems.

**Features:**
- Downloads and installs the agent binary
- Configures the agent with server URL and API key
- Sets up systemd service (Linux) or launchd service (macOS)
- Supports multiple versions and rolling upgrades

See [roles/keldris_agent/README.md](roles/keldris_agent/README.md) for detailed documentation.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/MacJediWizard/keldris.git
cd keldris

# Set up your inventory
cp examples/ansible/inventory.example.yml examples/ansible/inventory.yml
# Edit inventory.yml with your hosts and credentials

# Deploy agents
cd examples/ansible
ansible-playbook -i inventory.yml deploy-agent.yml
```

## Requirements

- Ansible 2.10 or higher
- Python 3.8+ on control node
- SSH access to target hosts

## Role Structure

```
ansible/
├── README.md                          # This file
└── roles/
    └── keldris_agent/
        ├── defaults/main.yml          # Default variables
        ├── handlers/main.yml          # Service restart handlers
        ├── meta/main.yml              # Galaxy metadata
        ├── tasks/
        │   ├── main.yml               # Main entry point
        │   ├── install.yml            # Binary installation
        │   ├── configure.yml          # Configuration
        │   ├── systemd.yml            # Linux service setup
        │   ├── launchd.yml            # macOS service setup
        │   └── register.yml           # Server registration
        ├── templates/
        │   ├── config.yml.j2          # Agent config template
        │   ├── keldris-agent.service.j2   # Systemd service
        │   └── io.keldris.agent.plist.j2  # Launchd plist
        ├── vars/
        │   ├── Debian.yml             # Debian/Ubuntu vars
        │   ├── RedHat.yml             # RHEL/CentOS vars
        │   └── Darwin.yml             # macOS vars
        └── README.md                  # Role documentation
```

## Future Plans

- Publish to Ansible Galaxy as `macjediwizard.keldris_agent`
- Add `keldris_server` role for server deployment
- Collection packaging for easier distribution
