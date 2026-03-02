#!/bin/bash
# Keldris Agent Installer for Linux
# Downloads the agent binary, installs to /usr/local/bin, and configures systemd

set -euo pipefail

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/keldris"
SERVICE_NAME="keldris-agent"
BINARY_NAME="keldris-agent"
GITHUB_REPO="MacJediWizard/keldris"
DOWNLOAD_BASE_URL="${KELDRIS_DOWNLOAD_URL:-https://github.com/${GITHUB_REPO}/releases/latest/download}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Detect OS
detect_os() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS: /etc/os-release not found"
        exit 1
    fi

    # shellcheck source=/dev/null
    . /etc/os-release
    echo "$ID"
}

# Download binary
download_binary() {
    local arch="$1"
    local download_url="${DOWNLOAD_BASE_URL}/keldris-agent-linux-${arch}"
    local tmp_file="/tmp/${BINARY_NAME}"

    log_info "Downloading Keldris Agent (linux/${arch})..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "$tmp_file" "$download_url" || {
            log_error "Failed to download binary from $download_url"
            exit 1
        }
    elif command -v wget &> /dev/null; then
        wget -q -O "$tmp_file" "$download_url" || {
            log_error "Failed to download binary from $download_url"
            exit 1
        }
    else
        log_error "Neither curl nor wget found. Please install one of them."
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
    mkdir -p "$INSTALL_DIR"

    # Move binary to install location
    mv "$tmp_file" "$install_path"

    # Set permissions
    chmod 755 "$install_path"
    chown root:root "$install_path"

    log_info "Binary installed successfully"
}

# Create config directory
create_config_dir() {
    log_info "Creating configuration directory..."
    mkdir -p "$CONFIG_DIR"
    chmod 750 "$CONFIG_DIR"
}

# Create systemd service
create_systemd_service() {
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"

    log_info "Creating systemd service..."

    cat > "$service_file" << 'EOF'
[Unit]
Description=Keldris Backup Agent
Documentation=https://keldris.io/docs
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/keldris-agent daemon
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=keldris-agent

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=yes
ReadWritePaths=/etc/keldris /var/log

# Environment
Environment=KELDRIS_CONFIG_DIR=/etc/keldris

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$service_file"

    log_info "Systemd service created at ${service_file}"
}

# Enable and start service
enable_service() {
    log_info "Reloading systemd daemon..."
    systemctl daemon-reload

    log_info "Enabling ${SERVICE_NAME} service..."
    systemctl enable "$SERVICE_NAME"

    log_info "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"

    # Check service status
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "Service started successfully"
    else
        log_warn "Service may not have started correctly. Check 'systemctl status ${SERVICE_NAME}'"
    fi
}

# Print post-install instructions
print_instructions() {
    echo ""
    echo "=============================================="
    echo "  Keldris Agent Installation Complete"
    echo "=============================================="
    echo ""
    echo "Next steps:"
    echo "  1. Register the agent with your Keldris server:"
    echo "     sudo keldris-agent register --server https://your-server.com"
    echo ""
    echo "  2. Check agent status:"
    echo "     sudo keldris-agent status"
    echo ""
    echo "Service management:"
    echo "  Start:   sudo systemctl start ${SERVICE_NAME}"
    echo "  Stop:    sudo systemctl stop ${SERVICE_NAME}"
    echo "  Status:  sudo systemctl status ${SERVICE_NAME}"
    echo "  Logs:    sudo journalctl -u ${SERVICE_NAME} -f"
    echo ""
    echo "Configuration directory: ${CONFIG_DIR}"
    echo ""
}

# Uninstall function
uninstall() {
    log_info "Uninstalling Keldris Agent..."

    # Stop and disable service
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        log_info "Stopping service..."
        systemctl stop "$SERVICE_NAME"
    fi

    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        log_info "Disabling service..."
        systemctl disable "$SERVICE_NAME"
    fi

    # Remove service file
    if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
        log_info "Removing systemd service..."
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        systemctl daemon-reload
    fi

    # Remove binary
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_info "Removing binary..."
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    log_info "Uninstall complete"
    log_warn "Configuration directory ${CONFIG_DIR} was not removed. Delete manually if needed."
}

# Main installation function
main() {
    local action="${1:-install}"

    case "$action" in
        install)
            check_root

            log_info "Starting Keldris Agent installation..."

            local arch
            arch=$(detect_arch)
            log_info "Detected architecture: $arch"

            local os
            os=$(detect_os)
            log_info "Detected OS: $os"

            local tmp_file
            tmp_file=$(download_binary "$arch")

            install_binary "$tmp_file"
            create_config_dir
            create_systemd_service
            enable_service
            print_instructions
            ;;
        uninstall)
            check_root
            uninstall
            ;;
        *)
            echo "Usage: $0 [install|uninstall]"
            exit 1
            ;;
    esac
}

main "$@"
