#!/bin/bash
# Keldris Agent Installer for macOS
# Downloads the agent binary, installs to /usr/local/bin, and configures launchd

set -euo pipefail

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/keldris"
PLIST_DIR="$HOME/Library/LaunchAgents"
SERVICE_LABEL="io.keldris.agent"
BINARY_NAME="keldris-agent"
GITHUB_REPO="MacJediWizard/keldris"
DOWNLOAD_BASE_URL="${KELDRIS_DOWNLOAD_URL:-https://github.com/${GITHUB_REPO}/releases/latest/download}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Check macOS version
check_macos() {
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "This script is for macOS only"
        exit 1
    fi

    local version
    version=$(sw_vers -productVersion)
    log_info "Detected macOS version: $version"
}

# Download binary
download_binary() {
    local arch="$1"
    local download_url="${DOWNLOAD_BASE_URL}/keldris-agent-darwin-${arch}"
    local tmp_file="/tmp/${BINARY_NAME}"

    log_info "Downloading Keldris Agent (darwin/${arch})..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "$tmp_file" "$download_url" || {
            log_error "Failed to download binary from $download_url"
            exit 1
        }
    else
        log_error "curl not found. Please install curl."
        exit 1
    fi

    echo "$tmp_file"
}

# Install binary
install_binary() {
    local tmp_file="$1"
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"

    log_info "Installing binary to ${install_path}..."

    # Create install directory if it doesn't exist
    if [[ ! -d "$INSTALL_DIR" ]]; then
        sudo mkdir -p "$INSTALL_DIR"
    fi

    # Move binary to install location (requires sudo for /usr/local/bin)
    sudo mv "$tmp_file" "$install_path"

    # Set permissions
    sudo chmod 755 "$install_path"

    # Remove quarantine attribute if present (from downloaded file)
    sudo xattr -d com.apple.quarantine "$install_path" 2>/dev/null || true

    log_info "Binary installed successfully"
}

# Install restic backup engine
install_restic() {
    if command -v restic &> /dev/null; then
        local current_version
        current_version=$(restic version 2>/dev/null | awk '{print $2}')
        log_info "Restic already installed (${current_version})"
        return 0
    fi

    log_info "Installing restic..."

    local arch
    arch=$(uname -m)
    local restic_arch
    case "$arch" in
        x86_64)  restic_arch="amd64" ;;
        arm64)   restic_arch="arm64" ;;
        *) log_warn "Unsupported architecture for restic: $arch"; return 0 ;;
    esac

    # Get latest version tag
    local latest_tag
    latest_tag=$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/restic/restic/releases/latest | grep -o '[^/]*$')
    local version="${latest_tag#v}"
    local restic_url="https://github.com/restic/restic/releases/download/${latest_tag}/restic_${version}_darwin_${restic_arch}.bz2"
    local tmp_restic="/tmp/restic.bz2"

    curl -fsSL -o "$tmp_restic" "$restic_url" || {
        log_warn "Failed to download restic. Install manually: brew install restic"
        return 0
    }

    bunzip2 -f "$tmp_restic"
    if [[ ! -f "/tmp/restic" ]]; then
        log_warn "Failed to decompress restic. Install manually: brew install restic"
        return 0
    fi

    sudo mv "/tmp/restic" "${INSTALL_DIR}/restic"
    sudo chmod 755 "${INSTALL_DIR}/restic"
    sudo xattr -d com.apple.quarantine "${INSTALL_DIR}/restic" 2>/dev/null || true

    local installed_version
    installed_version=$(restic version 2>/dev/null | awk '{print $2}')
    log_info "Restic ${installed_version} installed successfully"
}

# Create config directory
create_config_dir() {
    log_info "Creating configuration directory..."
    mkdir -p "$CONFIG_DIR"
    chmod 700 "$CONFIG_DIR"
}

# Create launchd plist
create_launchd_plist() {
    local plist_file="${PLIST_DIR}/${SERVICE_LABEL}.plist"

    log_info "Creating launchd plist..."

    # Create LaunchAgents directory if it doesn't exist
    mkdir -p "$PLIST_DIR"

    cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_LABEL}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BINARY_NAME}</string>
        <string>start</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>NetworkState</key>
        <true/>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>StandardOutPath</key>
    <string>${CONFIG_DIR}/agent.log</string>

    <key>StandardErrorPath</key>
    <string>${CONFIG_DIR}/agent.error.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>KELDRIS_CONFIG_DIR</key>
        <string>${CONFIG_DIR}</string>
    </dict>

    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>
EOF

    chmod 644 "$plist_file"

    log_info "Launchd plist created at ${plist_file}"
}

# Load and start service
start_service() {
    local plist_file="${PLIST_DIR}/${SERVICE_LABEL}.plist"

    log_info "Loading launchd service..."

    # Unload if already loaded
    launchctl unload "$plist_file" 2>/dev/null || true

    # Load the service
    launchctl load -w "$plist_file"

    # Give it a moment to start
    sleep 2

    # Check if running
    if launchctl list | grep -q "$SERVICE_LABEL"; then
        log_info "Service started successfully"
    else
        log_warn "Service may not have started correctly. Check logs at ${CONFIG_DIR}/agent.log"
    fi
}

# Register agent with Keldris server using env vars
REGISTRATION_SUCCESS=""
register_agent() {
    local server="${KELDRIS_SERVER:-}"
    local code="${KELDRIS_CODE:-}"
    local org_id="${KELDRIS_ORG_ID:-}"

    # Skip if any required env var is missing
    if [[ -z "$server" || -z "$code" || -z "$org_id" ]]; then
        return 0
    fi

    log_info "Registering agent with Keldris server..."

    local hostname
    hostname=$(hostname)

    local response
    response=$(curl -fsSL -X POST "${server}/api/v1/agents/register" \
        -H "Content-Type: application/json" \
        -H "X-Org-ID: ${org_id}" \
        -d "{\"code\":\"${code}\",\"hostname\":\"${hostname}\"}") || {
        log_warn "Agent registration failed. You can register manually later."
        return 0
    }

    # Parse api_key and agent_id from JSON response (no jq dependency)
    local api_key agent_id
    api_key=$(echo "$response" | grep -o '"api_key":"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')
    agent_id=$(echo "$response" | grep -o '"id":"[^"]*"' | grep -o '"[^"]*"$' | tr -d '"')

    if [[ -z "$api_key" || -z "$agent_id" ]]; then
        log_warn "Could not parse registration response. You can register manually later."
        return 0
    fi

    # Write config file
    cat > "${CONFIG_DIR}/config.yml" << CONF
server_url: ${server}
api_key: ${api_key}
agent_id: ${agent_id}
hostname: ${hostname}
CONF

    chmod 600 "${CONFIG_DIR}/config.yml"

    log_info "Agent registered successfully (agent_id: ${agent_id})"
    REGISTRATION_SUCCESS="true"
}

# Print post-install instructions
print_instructions() {
    echo ""
    echo "=============================================="
    echo "  Keldris Agent Installation Complete"
    echo "=============================================="
    echo ""

    if [[ -n "$REGISTRATION_SUCCESS" ]]; then
        echo "  Agent is registered and running!"
        echo ""
        echo "  Check agent status:"
        echo "     keldris-agent status"
    else
        echo "Next steps:"
        echo "  1. Register the agent with your Keldris server:"
        echo "     keldris-agent register --server https://your-server.com"
        echo ""
        echo "  2. Check agent status:"
        echo "     keldris-agent status"
    fi

    echo ""
    echo "Service management:"
    echo "  Start:  launchctl load ~/Library/LaunchAgents/${SERVICE_LABEL}.plist"
    echo "  Stop:   launchctl unload ~/Library/LaunchAgents/${SERVICE_LABEL}.plist"
    echo "  Status: launchctl list | grep keldris"
    echo "  Logs:   tail -f ${CONFIG_DIR}/agent.log"
    echo ""
    echo "Configuration directory: ${CONFIG_DIR}"
    echo ""
}

# Uninstall function
uninstall() {
    local purge=false
    for arg in "$@"; do
        case "$arg" in
            --purge) purge=true ;;
        esac
    done

    local plist_file="${PLIST_DIR}/${SERVICE_LABEL}.plist"

    log_info "Uninstalling Keldris Agent..."

    # Unload service
    if [[ -f "$plist_file" ]]; then
        log_info "Stopping and unloading service..."
        launchctl unload "$plist_file" 2>/dev/null || true
        rm -f "$plist_file"
        log_info "Launchd plist removed."
    fi

    # Remove binary (may need sudo for /usr/local/bin)
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_info "Removing binary..."
        if [[ -w "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
            rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        else
            sudo rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        fi
    fi

    # Remove restic if installed by us (only in /usr/local/bin, not Homebrew)
    if [[ "$purge" == true && -f "${INSTALL_DIR}/restic" ]]; then
        if ! brew list restic &>/dev/null 2>&1; then
            log_info "Removing restic..."
            if [[ -w "${INSTALL_DIR}/restic" ]]; then
                rm -f "${INSTALL_DIR}/restic"
            else
                sudo rm -f "${INSTALL_DIR}/restic"
            fi
        else
            log_info "Restic is managed by Homebrew, skipping"
        fi
    fi

    # Remove config and data if --purge
    if [[ "$purge" == true ]]; then
        if [[ -d "$CONFIG_DIR" ]]; then
            log_info "Removing configuration directory ${CONFIG_DIR}..."
            rm -rf "$CONFIG_DIR"
        fi

        # Also clean up ~/.keldris if it exists
        if [[ -d "$HOME/.keldris" ]]; then
            log_info "Removing $HOME/.keldris..."
            rm -rf "$HOME/.keldris"
        fi

        # Unmount any active FUSE mounts before cleanup
        if [[ -d "/tmp/keldris-mounts" ]]; then
            log_info "Unmounting FUSE mounts..."
            for mnt in /tmp/keldris-mounts/*/; do
                [[ -d "$mnt" ]] || continue
                umount "$mnt" 2>/dev/null || true
            done
        fi

        log_info "Removing temporary files..."
        rm -rf /tmp/keldris-* 2>/dev/null || true
        rm -f /tmp/restic-compressed-* /tmp/restic-download-* 2>/dev/null || true
    else
        if [[ -d "$CONFIG_DIR" ]]; then
            log_warn "Configuration directory ${CONFIG_DIR} was preserved."
            log_warn "Run with --purge to remove all configuration and data."
        fi
    fi

    log_info "Keldris Agent has been uninstalled."
}

# Main installation function
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            log_info "Starting Keldris Agent installation..."

            check_macos

            local arch
            arch=$(detect_arch)
            log_info "Detected architecture: $arch"

            local tmp_file
            tmp_file=$(download_binary "$arch")

            install_binary "$tmp_file"
            install_restic
            create_config_dir
            create_launchd_plist
            register_agent
            start_service
            print_instructions
            ;;
        uninstall)
            uninstall "${@:2}"
            ;;
        *)
            echo "Usage: $0 [install|uninstall [--purge]]"
            exit 1
            ;;
    esac
}

main "$@"
