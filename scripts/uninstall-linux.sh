#!/bin/bash
# Keldris Agent Uninstaller for Linux
# Stops the service, removes the binary, and optionally removes config/data

set -euo pipefail

SERVICE_NAME="keldris-agent"
BINARY_NAME="keldris-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/keldris"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

remove_all=false

usage() {
    echo "Usage: $0 [--purge]"
    echo ""
    echo "Options:"
    echo "  --purge    Also remove configuration and data files"
    echo ""
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --purge)
            remove_all=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

main() {
    check_root

    log_info "Uninstalling Keldris Agent..."

    # Stop and disable systemd service
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
        log_info "Removing systemd service file..."
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        systemctl daemon-reload
        systemctl reset-failed "$SERVICE_NAME" 2>/dev/null || true
    fi

    # Remove binary
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_info "Removing binary..."
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    # Remove restic if installed by us (only in /usr/local/bin, not system package)
    if [[ "$remove_all" == true && -f "${INSTALL_DIR}/restic" ]]; then
        log_info "Removing restic..."
        rm -f "${INSTALL_DIR}/restic"
    fi

    # Remove config and data if --purge
    if [[ "$remove_all" == true ]]; then
        if [[ -d "$CONFIG_DIR" ]]; then
            log_info "Removing configuration directory ${CONFIG_DIR}..."
            rm -rf "$CONFIG_DIR"
        fi

        # Also clean up root's home config if it exists
        if [[ -d "/root/.keldris" ]]; then
            log_info "Removing /root/.keldris..."
            rm -rf "/root/.keldris"
        fi

        # Unmount any active FUSE mounts before cleanup
        if [[ -d "/tmp/keldris-mounts" ]]; then
            log_info "Unmounting FUSE mounts..."
            for mnt in /tmp/keldris-mounts/*/; do
                [[ -d "$mnt" ]] || continue
                fusermount -u "$mnt" 2>/dev/null || umount "$mnt" 2>/dev/null || true
            done
        fi

        log_info "Removing temporary files..."
        rm -rf /tmp/keldris-* 2>/dev/null || true
        rm -f /tmp/restic-compressed-* /tmp/restic-download-* 2>/dev/null || true

        # Remove log files
        rm -f /var/log/keldris* 2>/dev/null || true
    else
        if [[ -d "$CONFIG_DIR" ]]; then
            log_warn "Configuration directory ${CONFIG_DIR} was preserved."
            log_warn "Run with --purge to remove all configuration and data."
        fi
    fi

    log_info "Keldris Agent has been uninstalled."
}

main
