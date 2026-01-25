# Keldris Agent Ansible Examples

Example playbooks for deploying and managing Keldris backup agents.

## Quick Start

1. Copy the example inventory:
   ```bash
   cp inventory.example.yml inventory.yml
   ```

2. Edit `inventory.yml` with your hosts and credentials:
   ```yaml
   all:
     vars:
       keldris_server_url: "https://your-keldris-server.com"
       keldris_api_key: "your-api-key"
   ```

3. Run the deployment playbook:
   ```bash
   ansible-playbook -i inventory.yml deploy-agent.yml
   ```

## Playbooks

### deploy-agent.yml

Deploys the Keldris agent to all hosts in the `backup_clients` group.

```bash
# Using environment variables
export KELDRIS_SERVER_URL="https://backup.example.com"
export KELDRIS_API_KEY="your-api-key"
ansible-playbook -i inventory.yml deploy-agent.yml

# Or pass variables directly
ansible-playbook -i inventory.yml deploy-agent.yml \
  -e keldris_server_url="https://backup.example.com" \
  -e keldris_api_key="your-api-key"
```

### upgrade-agents.yml

Performs a rolling upgrade of agents across all hosts.

```bash
# Upgrade to latest version
ansible-playbook -i inventory.yml upgrade-agents.yml

# Upgrade to specific version
ansible-playbook -i inventory.yml upgrade-agents.yml -e target_version="v1.2.3"
```

## Directory Structure

```
examples/ansible/
├── README.md                 # This file
├── deploy-agent.yml          # Initial deployment playbook
├── inventory.example.yml     # Example inventory
└── upgrade-agents.yml        # Upgrade playbook
```

## Using with Ansible Galaxy

Once the role is published to Ansible Galaxy, you can install it with:

```bash
ansible-galaxy install macjediwizard.keldris_agent
```

Then reference it in your playbooks:

```yaml
- hosts: backup_clients
  roles:
    - role: macjediwizard.keldris_agent
```

## Security Notes

- Store API keys in Ansible Vault or use environment variables
- Never commit `inventory.yml` with real credentials to version control
- The agent config file (containing API key) is created with mode 0600

Example using Ansible Vault:

```bash
# Create encrypted vars file
ansible-vault create vault.yml

# Add to playbook
vars_files:
  - vault.yml

# Run with vault
ansible-playbook -i inventory.yml deploy-agent.yml --ask-vault-pass
```
