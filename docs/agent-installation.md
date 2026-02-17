# Keldris Agent Installation Guide

This guide covers how to install and configure the Keldris backup agent on Linux, macOS, and Windows.

## Quick Start

### Linux (Debian, Ubuntu, RHEL, etc.)

```bash
curl -sSL https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-linux.sh | sudo bash
```

Or download and run manually:

```bash
curl -O https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-linux.sh
chmod +x install-linux.sh
sudo ./install-linux.sh
```

### macOS

```bash
curl -sSL https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-macos.sh | bash
```

Or download and run manually:

```bash
curl -O https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-macos.sh
chmod +x install-macos.sh
./install-macos.sh
```

### Windows (PowerShell as Administrator)

```powershell
irm https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-windows.ps1 | iex
```

Or download and run manually:

```powershell
Invoke-WebRequest -Uri https://raw.githubusercontent.com/MacJediWizard/keldris/main/scripts/install-windows.ps1 -OutFile install-windows.ps1
.\install-windows.ps1 -Action Install
```

### Docker

```bash
docker pull ghcr.io/macjediwizard/keldris-agent:1.0.0-beta.3
docker run -d --name keldris-agent \
  -e KELDRIS_SERVER_URL=https://your-keldris-server.com \
  -v /etc/keldris:/etc/keldris \
  ghcr.io/macjediwizard/keldris-agent:1.0.0-beta.3
```

## Post-Installation Setup

After installation, register the agent with your Keldris server:

```bash
keldris-agent register --server https://your-keldris-server.com
```

You will be prompted for an API key. Generate one from the Keldris web interface under **Settings > API Keys**.

Verify the connection:

```bash
keldris-agent status
```

## Manual Installation

If you prefer to install manually without the installer scripts:

### 1. Download the Binary

Download the appropriate binary for your platform from [GitHub Releases](https://github.com/MacJediWizard/keldris/releases):

| Platform | Architecture | Download |
|----------|-------------|----------|
| Linux | x86_64 | [keldris-agent-linux-amd64](https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-linux-amd64) |
| Linux | ARM64 | [keldris-agent-linux-arm64](https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-linux-arm64) |
| macOS | Intel | [keldris-agent-darwin-amd64](https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-darwin-amd64) |
| macOS | Apple Silicon | [keldris-agent-darwin-arm64](https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-darwin-arm64) |
| Windows | x86_64 | [keldris-agent-windows-amd64.exe](https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-windows-amd64.exe) |

Example download with curl:

```bash
# Linux x86_64
curl -Lo keldris-agent https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-linux-amd64

# macOS Apple Silicon
curl -Lo keldris-agent https://github.com/MacJediWizard/keldris/releases/latest/download/keldris-agent-darwin-arm64
```

### 2. Install the Binary

**Linux/macOS:**
```bash
sudo mv keldris-agent-* /usr/local/bin/keldris-agent
sudo chmod +x /usr/local/bin/keldris-agent
```

**Windows:**
Move the executable to `C:\Program Files\Keldris\keldris-agent.exe` and add the directory to your PATH.

### 3. Configure Auto-Start (Optional)

See the platform-specific sections below for setting up automatic startup.

## Build from Source

Requires Go 1.25.7+ installed.

```bash
git clone https://github.com/MacJediWizard/keldris.git
cd keldris
go build -o keldris-agent ./cmd/keldris-agent
sudo mv keldris-agent /usr/local/bin/
```

To cross-compile for a different platform:

```bash
# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o keldris-agent-linux-arm64 ./cmd/keldris-agent

# Windows
GOOS=windows GOARCH=amd64 go build -o keldris-agent-windows-amd64.exe ./cmd/keldris-agent
```

## Platform-Specific Details

### Linux (systemd)

The installer creates a systemd service at `/etc/systemd/system/keldris-agent.service`.

**Service Management:**

```bash
# Start the service
sudo systemctl start keldris-agent

# Stop the service
sudo systemctl stop keldris-agent

# Check status
sudo systemctl status keldris-agent

# View logs
sudo journalctl -u keldris-agent -f

# Enable auto-start on boot
sudo systemctl enable keldris-agent

# Disable auto-start
sudo systemctl disable keldris-agent
```

**Configuration Directory:** `/etc/keldris`

**Manual systemd Service Setup:**

Create `/etc/systemd/system/keldris-agent.service`:

```ini
[Unit]
Description=Keldris Backup Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/keldris-agent daemon
Restart=always
RestartSec=10
Environment=KELDRIS_CONFIG_DIR=/etc/keldris

[Install]
WantedBy=multi-user.target
```

Then enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable keldris-agent
sudo systemctl start keldris-agent
```

### macOS (launchd)

The installer creates a launch agent at `~/Library/LaunchAgents/io.keldris.agent.plist`.

**Service Management:**

```bash
# Load/start the service
launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist

# Unload/stop the service
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist

# Check if running
launchctl list | grep keldris

# View logs
tail -f ~/.config/keldris/agent.log
```

**Configuration Directory:** `~/.config/keldris`

**Manual launchd Setup:**

Create `~/Library/LaunchAgents/io.keldris.agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.keldris.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/keldris-agent</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Then load:

```bash
launchctl load ~/Library/LaunchAgents/io.keldris.agent.plist
```

### Windows (Windows Service)

The installer creates a Windows Service named "KeldrisAgent".

**Service Management (PowerShell as Administrator):**

```powershell
# Start the service
Start-Service KeldrisAgent

# Stop the service
Stop-Service KeldrisAgent

# Check status
Get-Service KeldrisAgent

# View service details
sc.exe query KeldrisAgent
```

**Configuration Directory:** `C:\ProgramData\Keldris`

**Install Directory:** `C:\Program Files\Keldris`

**Manual Windows Service Setup:**

```powershell
# Create the service
sc.exe create KeldrisAgent binPath= "\"C:\Program Files\Keldris\keldris-agent.exe\" daemon" start= auto DisplayName= "Keldris Backup Agent"

# Set description
sc.exe description KeldrisAgent "Keldris backup agent for automated system backups"

# Configure recovery (restart on failure)
sc.exe failure KeldrisAgent reset= 86400 actions= restart/10000/restart/30000/restart/60000

# Start the service
Start-Service KeldrisAgent
```

## Environment Variables

The agent supports the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KELDRIS_CONFIG_DIR` | Configuration directory path | Platform-specific |
| `KELDRIS_SERVER_URL` | Server URL (overrides config) | - |
| `KELDRIS_LOG_LEVEL` | Log level (debug, info, warn, error) | info |

## Uninstallation

### Docker

```bash
docker stop keldris-agent
docker rm keldris-agent
docker rmi ghcr.io/macjediwizard/keldris-agent:1.0.0-beta.3
```

### Linux

If you used the install script:

```bash
sudo ./install-linux.sh uninstall
```

Or manually:

```bash
sudo systemctl stop keldris-agent
sudo systemctl disable keldris-agent
sudo rm /etc/systemd/system/keldris-agent.service
sudo systemctl daemon-reload
sudo rm /usr/local/bin/keldris-agent
# Optionally remove config: sudo rm -rf /etc/keldris
```

### macOS

If you used the install script:

```bash
./install-macos.sh uninstall
```

Or manually:

```bash
launchctl unload ~/Library/LaunchAgents/io.keldris.agent.plist
rm ~/Library/LaunchAgents/io.keldris.agent.plist
sudo rm /usr/local/bin/keldris-agent
# Optionally remove config: rm -rf ~/.config/keldris
```

### Windows (PowerShell as Administrator)

If you used the install script:

```powershell
.\install-windows.ps1 -Action Uninstall
```

Or manually:

```powershell
Stop-Service KeldrisAgent
sc.exe delete KeldrisAgent
Remove-Item "C:\Program Files\Keldris" -Recurse -Force
# Optionally remove config: Remove-Item "C:\ProgramData\Keldris" -Recurse -Force
```

## Troubleshooting

### Agent won't start

1. Check the logs for errors
2. Verify the configuration file exists and is valid
3. Ensure the agent is registered with a server
4. Check network connectivity to the Keldris server

### Connection refused

1. Verify the server URL is correct
2. Check firewall rules allow outbound HTTPS (port 443)
3. Ensure the API key is valid

### Permission denied

1. Linux: Run commands with `sudo`
2. macOS: Ensure binary has execute permissions
3. Windows: Run PowerShell as Administrator

### Service fails to start

1. Check for port conflicts
2. Verify the binary path is correct in the service configuration
3. Review system logs for detailed error messages
