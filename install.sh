#!/bin/bash
#
# Honeybadger CLI Installation Script
#
# This script installs the Honeybadger CLI (hb) and optionally configures it
# to run as a systemd service for continuous metrics reporting.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash
#   curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash -s -- --api-key YOUR_API_KEY
#   curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash -s -- --version v1.0.0
#
# Options:
#   --api-key KEY       Honeybadger API key for the agent
#   --version VERSION   Specific version to install (default: latest)
#   --interval SECONDS  Metrics reporting interval (default: 60)
#   --no-service        Install binary only, skip systemd service setup
#   --help              Show this help message
#

set -e

# Configuration
GITHUB_REPO="honeybadger-io/cli"
BINARY_NAME="hb"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="honeybadger-agent"
DEFAULT_INTERVAL=60

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script options
API_KEY=""
VERSION="latest"
INTERVAL=$DEFAULT_INTERVAL
INSTALL_SERVICE=true

#######################################
# Print colored output
#######################################
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

#######################################
# Show usage information
#######################################
usage() {
    cat << EOF
Honeybadger CLI Installation Script

Usage:
  $0 [options]

Options:
  --api-key KEY       Honeybadger API key for the agent
  --version VERSION   Specific version to install (default: latest)
  --interval SECONDS  Metrics reporting interval in seconds (default: 60)
  --no-service        Install binary only, skip systemd service setup
  --help              Show this help message

Examples:
  # Install latest version and configure as systemd service
  curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash

  # Install with API key provided
  curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash -s -- --api-key YOUR_API_KEY

  # Install specific version
  curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash -s -- --version v1.0.0

  # Install binary only (no systemd service)
  curl -sSL https://raw.githubusercontent.com/honeybadger-io/cli/main/install.sh | bash -s -- --no-service

EOF
    exit 0
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --api-key)
                API_KEY="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --interval)
                INTERVAL="$2"
                shift 2
                ;;
            --no-service)
                INSTALL_SERVICE=false
                shift
                ;;
            --help|-h)
                usage
                ;;
            *)
                error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

#######################################
# Check if running as root
#######################################
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
        exit 1
    fi
}

#######################################
# Check for required dependencies
#######################################
check_dependencies() {
    local missing=()

    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi

    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing[*]}"
        error "Please install them and try again."
        exit 1
    fi
}

#######################################
# Detect operating system
#######################################
detect_os() {
    local os
    os=$(uname -s)

    case "$os" in
        Linux)
            echo "Linux"
            ;;
        Darwin)
            echo "Darwin"
            ;;
        *)
            error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

#######################################
# Detect CPU architecture
#######################################
detect_arch() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64)
            echo "x86_64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

#######################################
# Get the latest release version from GitHub
#######################################
get_latest_version() {
    local version
    version=$(curl -sS "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version from GitHub"
        exit 1
    fi

    echo "$version"
}

#######################################
# Download and install the binary
#######################################
install_binary() {
    local os=$1
    local arch=$2
    local version=$3

    # Remove 'v' prefix if present for the download URL
    local version_number="${version#v}"

    # Construct download URL
    local archive_name="cli_${os}_${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${archive_name}"

    info "Downloading Honeybadger CLI ${version} for ${os}/${arch}..."

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download archive
    if ! curl -sSL -o "${tmp_dir}/${archive_name}" "$download_url"; then
        error "Failed to download from: $download_url"
        exit 1
    fi

    # Extract archive
    info "Extracting archive..."
    if ! tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"; then
        error "Failed to extract archive"
        exit 1
    fi

    # Install binary
    info "Installing binary to ${INSTALL_DIR}/${BINARY_NAME}..."
    if ! install -m 755 "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"; then
        error "Failed to install binary"
        exit 1
    fi

    success "Binary installed successfully"
}

#######################################
# Prompt for API key if not provided
#######################################
prompt_api_key() {
    if [[ -z "$API_KEY" ]]; then
        echo ""
        echo -e "${YELLOW}Honeybadger API Key Required${NC}"
        echo "You can find your API key in your Honeybadger project settings."
        echo ""
        read -rp "Enter your Honeybadger API key: " API_KEY

        if [[ -z "$API_KEY" ]]; then
            error "API key is required for the agent service"
            exit 1
        fi
    fi
}

#######################################
# Check if systemd is available
#######################################
check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        warn "systemd is not available on this system"
        warn "Skipping service installation. You can run the agent manually with:"
        echo "  ${INSTALL_DIR}/${BINARY_NAME} agent --api-key YOUR_API_KEY"
        return 1
    fi
    return 0
}

#######################################
# Create systemd service file
#######################################
create_systemd_service() {
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"

    info "Creating systemd service..."

    cat > "$service_file" << EOF
[Unit]
Description=Honeybadger Agent - System Metrics Reporter
Documentation=https://github.com/honeybadger-io/cli
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BINARY_NAME} agent --interval ${INTERVAL}
Restart=always
RestartSec=10
Environment="HONEYBADGER_API_KEY=${API_KEY}"

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

    # Secure the service file (contains API key)
    chmod 600 "$service_file"

    success "Systemd service created at ${service_file}"
}

#######################################
# Enable and start the service
#######################################
start_service() {
    info "Reloading systemd daemon..."
    systemctl daemon-reload

    info "Enabling ${SERVICE_NAME} service..."
    systemctl enable "$SERVICE_NAME"

    info "Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"

    # Wait a moment and check status
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        success "Service is running"
    else
        warn "Service may not have started correctly. Check with: systemctl status ${SERVICE_NAME}"
    fi
}

#######################################
# Print installation summary
#######################################
print_summary() {
    echo ""
    echo "================================================"
    echo -e "${GREEN}Honeybadger CLI Installation Complete${NC}"
    echo "================================================"
    echo ""
    echo "Binary installed: ${INSTALL_DIR}/${BINARY_NAME}"
    echo "Version: ${VERSION}"

    if [[ "$INSTALL_SERVICE" == true ]] && check_systemd; then
        echo ""
        echo "Systemd Service: ${SERVICE_NAME}"
        echo ""
        echo "Useful commands:"
        echo "  systemctl status ${SERVICE_NAME}   # Check service status"
        echo "  systemctl restart ${SERVICE_NAME}  # Restart the service"
        echo "  systemctl stop ${SERVICE_NAME}     # Stop the service"
        echo "  journalctl -u ${SERVICE_NAME} -f   # View logs"
    else
        echo ""
        echo "Run the agent manually:"
        echo "  ${BINARY_NAME} agent --api-key YOUR_API_KEY"
    fi

    echo ""
    echo "CLI Usage:"
    echo "  ${BINARY_NAME} --help                 # Show all commands"
    echo "  ${BINARY_NAME} agent --help           # Agent help"
    echo "  ${BINARY_NAME} deploy --help          # Deploy reporting help"
    echo ""
}

#######################################
# Main installation flow
#######################################
main() {
    echo ""
    echo "================================================"
    echo "  Honeybadger CLI Installer"
    echo "================================================"
    echo ""

    # Parse command line arguments
    parse_args "$@"

    # Check prerequisites
    check_root
    check_dependencies

    # Detect system
    local os
    local arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected system: ${os}/${arch}"

    # Determine version to install
    if [[ "$VERSION" == "latest" ]]; then
        VERSION=$(get_latest_version)
    fi
    info "Version to install: ${VERSION}"

    # Install binary
    install_binary "$os" "$arch" "$VERSION"

    # Verify installation
    if ! "${INSTALL_DIR}/${BINARY_NAME}" --version &> /dev/null; then
        warn "Binary installed but version check failed"
    fi

    # Setup systemd service if requested
    if [[ "$INSTALL_SERVICE" == true ]]; then
        if check_systemd; then
            prompt_api_key
            create_systemd_service
            start_service
        fi
    fi

    # Print summary
    print_summary
}

# Run main function
main "$@"
