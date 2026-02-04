# Agent Deployment Guide

This guide covers deploying Keldris agents across your infrastructure, from single machines to large fleets.

## Quick Start

### Single Machine Installation

**Linux:**
```bash
curl -sSL https://releases.keldris.io/install-linux.sh | sudo bash
```

**macOS:**
```bash
curl -sSL https://releases.keldris.io/install-macos.sh | bash
```

**Windows (PowerShell as Administrator):**
```powershell
irm https://releases.keldris.io/install-windows.ps1 | iex
```

After installation, register the agent:

```bash
keldris-agent register --server https://your-keldris-server.com
```

## Fleet Deployment

For deploying agents across many machines, Keldris supports several automation methods.

### Pre-Registration

Generate pre-authorized agent configurations from the web interface:

1. Go to **Agents > Fleet Deployment**
2. Click **Generate Deployment Package**
3. Select agent count and expiration
4. Download the deployment script

The script includes:
- Agent binary download
- Pre-configured server URL
- Pre-authorized API key
- Automatic service installation

### CSV Import

For bulk agent registration:

1. Go to **Agents > Import**
2. Upload a CSV with agent details:

```csv
name,hostname,tags,group
web-server-01,web01.example.com,"production,web",web-servers
web-server-02,web02.example.com,"production,web",web-servers
db-server-01,db01.example.com,"production,database",database
```

3. Download pre-configured installation scripts

### Ansible

Use the provided Ansible playbook for automated deployment:

```yaml
# playbook.yml
---
- hosts: backup_targets
  become: yes
  vars:
    keldris_server_url: "https://keldris.example.com"
    keldris_api_key: "{{ vault_keldris_api_key }}"

  tasks:
    - name: Download agent installer
      get_url:
        url: "{{ keldris_server_url }}/downloads/install-linux.sh"
        dest: /tmp/install-keldris.sh
        mode: '0755'

    - name: Run installer
      command: /tmp/install-keldris.sh
      args:
        creates: /usr/local/bin/keldris-agent

    - name: Configure agent
      template:
        src: agent.yaml.j2
        dest: /etc/keldris/agent.yaml
        mode: '0600'
      notify: restart keldris-agent

    - name: Start agent service
      systemd:
        name: keldris-agent
        state: started
        enabled: yes

  handlers:
    - name: restart keldris-agent
      systemd:
        name: keldris-agent
        state: restarted
```

### Terraform

Using the Keldris Terraform provider:

```hcl
provider "keldris" {
  server_url = "https://keldris.example.com"
  api_key    = var.keldris_api_key
}

resource "keldris_agent" "web_servers" {
  count = 3

  name     = "web-server-${count.index + 1}"
  hostname = "web${count.index + 1}.example.com"
  tags     = ["production", "web"]
  group_id = keldris_agent_group.web.id
}

resource "keldris_agent_group" "web" {
  name        = "Web Servers"
  description = "Production web server fleet"
}
```

### Puppet

```puppet
class keldris_agent (
  String $server_url,
  String $api_key,
) {
  package { 'keldris-agent':
    ensure   => installed,
    source   => "${server_url}/downloads/keldris-agent.deb",
    provider => dpkg,
  }

  file { '/etc/keldris/agent.yaml':
    ensure  => file,
    content => template('keldris_agent/agent.yaml.erb'),
    require => Package['keldris-agent'],
    notify  => Service['keldris-agent'],
  }

  service { 'keldris-agent':
    ensure  => running,
    enable  => true,
    require => File['/etc/keldris/agent.yaml'],
  }
}
```

### Chef

```ruby
# recipes/default.rb
remote_file '/tmp/install-keldris.sh' do
  source "#{node['keldris']['server_url']}/downloads/install-linux.sh"
  mode '0755'
  action :create
end

execute 'install-keldris-agent' do
  command '/tmp/install-keldris.sh'
  creates '/usr/local/bin/keldris-agent'
end

template '/etc/keldris/agent.yaml' do
  source 'agent.yaml.erb'
  mode '0600'
  notifies :restart, 'service[keldris-agent]'
end

service 'keldris-agent' do
  action [:enable, :start]
end
```

## Platform-Specific Details

### Linux (systemd)

The installer creates `/etc/systemd/system/keldris-agent.service`.

**Service Management:**
```bash
sudo systemctl start keldris-agent
sudo systemctl stop keldris-agent
sudo systemctl status keldris-agent
sudo journalctl -u keldris-agent -f
```

**Configuration:** `/etc/keldris/agent.yaml`

**Logs:** `journalctl -u keldris-agent`

### macOS (launchd)

The installer creates `~/Library/LaunchAgents/io.keldris.agent.plist`.

**Service Management:**
```bash
launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
launchctl list | grep keldris
```

**Configuration:** `~/.config/keldris/agent.yaml`

**Logs:** `~/.config/keldris/agent.log`

### Windows (Windows Service)

The installer creates the "KeldrisAgent" Windows Service.

**Service Management (PowerShell as Admin):**
```powershell
Start-Service KeldrisAgent
Stop-Service KeldrisAgent
Get-Service KeldrisAgent
```

**Configuration:** `C:\ProgramData\Keldris\agent.yaml`

**Install Directory:** `C:\Program Files\Keldris`

## Agent Groups

Organize agents into groups for easier management:

1. Go to **Agents > Groups**
2. Create groups like "Production", "Development", etc.
3. Assign agents to groups during registration or later

Groups enable:
- Bulk schedule assignment
- Group-level policies
- Organized dashboards

## Agent Tags

Tags provide flexible categorization:

```yaml
# agent.yaml
agent:
  name: web-server-01
  tags:
    - production
    - web
    - us-east-1
    - nginx
```

Tags can be used in:
- Search and filtering
- Schedule targeting
- Alert routing
- Report grouping

## Health Monitoring

Agents report health status every 60 seconds (configurable).

**Monitored metrics:**
- CPU usage
- Memory usage
- Disk space
- Network connectivity
- Backup job status

**Configure health reporting:**
```yaml
# agent.yaml
health:
  interval: 60s
  timeout: 30s
```

## Uninstallation

### Linux
```bash
sudo ./install-linux.sh uninstall
# Or manually:
sudo systemctl stop keldris-agent
sudo systemctl disable keldris-agent
sudo rm /etc/systemd/system/keldris-agent.service
sudo rm /usr/local/bin/keldris-agent
sudo rm -rf /etc/keldris
```

### macOS
```bash
./install-macos.sh uninstall
# Or manually:
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
rm ~/Library/LaunchAgents/io.keldris.agent.plist
sudo rm /usr/local/bin/keldris-agent
rm -rf ~/.config/keldris
```

### Windows (PowerShell as Admin)
```powershell
.\install-windows.ps1 -Action Uninstall
# Or manually:
Stop-Service KeldrisAgent
sc.exe delete KeldrisAgent
Remove-Item "C:\Program Files\Keldris" -Recurse
Remove-Item "C:\ProgramData\Keldris" -Recurse
```

## Security Considerations

1. **API Key Security:**
   - Store API keys securely (vault, secrets manager)
   - Rotate keys regularly
   - Use per-agent keys when possible

2. **Network Security:**
   - Agents only need outbound HTTPS
   - No inbound ports required
   - Consider using a proxy for air-gapped networks

3. **File Permissions:**
   - Config files should be readable only by root/admin
   - API keys must not be world-readable

4. **Agent Updates:**
   - Keep agents updated
   - Use automatic updates in production

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md#agent-issues) for common agent issues.
