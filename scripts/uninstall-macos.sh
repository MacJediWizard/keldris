#!/bin/bash
# Keldris Agent Uninstaller for macOS
# Stops the launchd service, removes the binary, and optionally removes config/data

set -euo pipefail

BINARY_NAME="keldris-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/keldris"
PLIST_DIR="$HOME/Library/LaunchAgents"
SERVICE_LABEL="io.keldris.agent"

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

remove_all=false

usage() {
    echo "Usage: $0 [--purge]"
    echo ""
    echo "Options:"
    echo "  --purge    Also remove configuration, data, and log files"
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
    log_info "Uninstalling Keldris Agent..."

    local plist_file="${PLIST_DIR}/${SERVICE_LABEL}.plist"

    # Unload launchd service
    if [[ -f "$plist_file" ]]; then
        log_info "Stopping and unloading launchd service..."
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

    # Remove config and data if --purge
    if [[ "$remove_all" == true ]]; then
        if [[ -d "$CONFIG_DIR" ]]; then
            log_info "Removing configuration directory ${CONFIG_DIR}..."
            rm -rf "$CONFIG_DIR"
        fi

        # Also clean up ~/.keldris if it exists
        if [[ -d "$HOME/.keldris" ]]; then
            log_info "Removing $HOME/.keldris..."
            rm -rf "$HOME/.keldris"
        fi
    else
        if [[ -d "$CONFIG_DIR" ]]; then
            log_warn "Configuration directory ${CONFIG_DIR} was preserved."
            log_warn "Run with --purge to remove all configuration and data."
        fi
    fi

    log_info "Keldris Agent has been uninstalled."
}

main
