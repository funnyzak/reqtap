#!/usr/bin/env bash

# ReqTap Installation & Management Script
# GitHub: https://github.com/funnyzak/reqtap
# Usage: curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash
#        Or download and run: ./install.sh [command] [options]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="funnyzak/reqtap"
BINARY_NAME="reqtap"
INSTALL_DIR=""
VERSION=""
SKIP_CHECKSUM=false
FORCE_INSTALL=false
COMMAND=""

# Display functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

title() {
    echo -e "${BOLD}${CYAN}$1${NC}"
}

# Print banner
print_banner() {
    cat << "EOF"
    ____             ______            
   / __ \___  ____ _/_  __/___ _____  
  / /_/ / _ \/ __ `// / / __ `/ __ \ 
 / _, _/  __/ /_/ // / / /_/ / /_/ / 
/_/ |_|\___/\__, //_/  \__,_/ .___/  
              /_/          /_/        
EOF
    echo ""
    title "ReqTap Installation & Management Tool"
    echo -e "${CYAN}GitHub: https://github.com/${GITHUB_REPO}${NC}"
    echo ""
}

# Print usage
usage() {
    cat << EOF
Usage: 
    $0 [COMMAND] [OPTIONS]

Commands:
    install         Install ReqTap (default if no command specified)
    uninstall       Uninstall ReqTap
    update          Update to the latest version
    check           Check installed version and available updates
    list            List all available versions
    help            Show this help message

Options:
    -v, --version VERSION    Specify version to install (default: latest)
    -d, --dir DIRECTORY      Installation directory (default: auto-detect)
    -f, --force             Force installation (overwrite existing)
    --skip-checksum          Skip SHA256 checksum verification
    -y, --yes               Skip confirmation prompts
    -h, --help              Show this help message

Examples:
    # Interactive mode
    $0

    # Install latest version
    $0 install

    # Install specific version
    $0 install -v v1.0.0

    # Install to custom directory
    $0 install -d /opt/bin

    # Update to latest version
    $0 update

    # Check for updates
    $0 check

    # Uninstall
    $0 uninstall

    # Quick install (one-liner)
    curl -fsSL https://raw.githubusercontent.com/funnyzak/reqtap/main/scripts/install.sh | bash

EOF
}

# Parse command line arguments
parse_args() {
    # First argument might be a command
    if [[ $# -gt 0 ]] && [[ ! "$1" =~ ^- ]]; then
        COMMAND="$1"
        shift
    fi

    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -f|--force)
                FORCE_INSTALL=true
                shift
                ;;
            --skip-checksum)
                SKIP_CHECKSUM=true
                shift
                ;;
            -y|--yes)
                SKIP_CONFIRMATION=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1\nUse -h or --help for usage information"
                ;;
        esac
    done
}

# Detect OS
detect_os() {
    local os
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)          error "Unsupported operating system: $(uname -s)" ;;
    esac
    echo "$os"
}

# Detect architecture
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        armv7l|armv6l)  arch="arm" ;;
        s390x)          arch="s390x" ;;
        ppc64le)        arch="ppc64le" ;;
        riscv64)        arch="riscv64" ;;
        *)              error "Unsupported architecture: $(uname -m)" ;;
    esac
    echo "$arch"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check required dependencies
check_dependencies() {
    local missing_deps=()
    local os="$1"

    # Check for download tool
    if ! command_exists curl && ! command_exists wget; then
        missing_deps+=("curl or wget")
    fi

    # Check for tar (not needed on Windows with unzip)
    if [[ "$os" != "windows" ]] && ! command_exists tar; then
        missing_deps+=("tar")
    fi

    # Check for unzip on Windows
    if [[ "$os" == "windows" ]] && ! command_exists unzip; then
        missing_deps+=("unzip")
    fi

    # Check for checksum tool
    if [[ "$SKIP_CHECKSUM" == false ]]; then
        if ! command_exists sha256sum && ! command_exists shasum; then
            warn "sha256sum/shasum not found, checksum verification will be skipped"
            SKIP_CHECKSUM=true
        fi
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing_deps[*]}\nPlease install them and try again."
    fi
}

# Download file using curl or wget
download_file() {
    local url="$1"
    local output="$2"

    if command_exists curl; then
        curl -fsSL --progress-bar -o "$output" "$url" 2>&1 || return 1
    elif command_exists wget; then
        wget -q --show-progress -O "$output" "$url" 2>&1 || return 1
    else
        return 1
    fi
    return 0
}

# Get latest version from GitHub
get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local version

    if command_exists curl; then
        version=$(curl -fsSL "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists wget; then
        version=$(wget -qO- "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    if [[ -z "$version" ]]; then
        return 1
    fi

    echo "$version"
}

# Get all available versions
get_all_versions() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases"
    local versions

    if command_exists curl; then
        versions=$(curl -fsSL "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists wget; then
        versions=$(wget -qO- "$api_url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    echo "$versions"
}

# Verify SHA256 checksum
verify_checksum() {
    local file="$1"
    local expected_hash="$2"
    local actual_hash

    if [[ "$SKIP_CHECKSUM" == true ]]; then
        warn "Skipping checksum verification"
        return 0
    fi

    info "Verifying SHA256 checksum..."

    if command_exists sha256sum; then
        actual_hash=$(sha256sum "$file" | awk '{print $1}')
    elif command_exists shasum; then
        actual_hash=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "Cannot verify checksum: sha256sum/shasum not found"
        return 0
    fi

    if [[ "$actual_hash" != "$expected_hash" ]]; then
        error "Checksum verification failed!\nExpected: $expected_hash\nActual: $actual_hash"
    fi

    success "Checksum verified"
}

# Get checksum from checksums file
get_checksum() {
    local filename="$1"
    local checksums_file="$2"
    
    if [[ ! -f "$checksums_file" ]]; then
        return 1
    fi

    grep "$filename" "$checksums_file" 2>/dev/null | awk '{print $1}'
}

# Find installed binary
find_installed_binary() {
    local locations=(
        "/usr/local/bin/$BINARY_NAME"
        "$HOME/.local/bin/$BINARY_NAME"
        "$HOME/bin/$BINARY_NAME"
        "/opt/bin/$BINARY_NAME"
    )

    # Also check in custom INSTALL_DIR if specified
    if [[ -n "$INSTALL_DIR" ]]; then
        locations=("$INSTALL_DIR/$BINARY_NAME" "${locations[@]}")
    fi

    for loc in "${locations[@]}"; do
        if [[ -f "$loc" ]]; then
            echo "$loc"
            return 0
        fi
    done

    return 1
}

# Get installed version
get_installed_version() {
    local binary_path
    if binary_path=$(find_installed_binary); then
        # Try to get version from binary
        if "$binary_path" --version 2>/dev/null | head -n 1; then
            return 0
        elif "$binary_path" -v 2>/dev/null | head -n 1; then
            return 0
        else
            echo "unknown"
            return 0
        fi
    fi
    return 1
}

# Determine installation directory
determine_install_dir() {
    if [[ -n "$INSTALL_DIR" ]]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Try common installation directories
    local dirs=(
        "/usr/local/bin"
        "$HOME/.local/bin"
        "$HOME/bin"
    )

    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]] && [[ -w "$dir" ]]; then
            echo "$dir"
            return
        fi
    done

    # If /usr/local/bin exists but not writable, try with sudo
    if [[ -d "/usr/local/bin" ]] && command_exists sudo; then
        echo "/usr/local/bin"
        return
    fi

    # Fallback to user's home bin
    local fallback="$HOME/.local/bin"
    if [[ ! -d "$fallback" ]]; then
        mkdir -p "$fallback"
    fi
    echo "$fallback"
}

# Extract archive
extract_archive() {
    local archive="$1"
    local dest_dir="$2"

    info "Extracting archive..."

    mkdir -p "$dest_dir"

    if [[ "$archive" == *.tar.gz ]]; then
        tar -xzf "$archive" -C "$dest_dir" || error "Failed to extract archive"
    elif [[ "$archive" == *.zip ]]; then
        unzip -q "$archive" -d "$dest_dir" || error "Failed to extract archive"
    else
        error "Unsupported archive format: $archive"
    fi

    success "Archive extracted"
}

# Install binary
install_binary() {
    local src="$1"
    local dest_dir="$2"
    local dest="$dest_dir/$BINARY_NAME"

    info "Installing to: $dest"

    # Backup existing binary if exists
    if [[ -f "$dest" ]] && [[ "$FORCE_INSTALL" == false ]]; then
        local backup="$dest.backup.$(date +%s)"
        info "Backing up existing binary to: $backup"
        if [[ -w "$dest_dir" ]]; then
            cp "$dest" "$backup"
        else
            if command_exists sudo; then
                sudo cp "$dest" "$backup"
            fi
        fi
    fi

    # Install new binary
    if [[ ! -w "$dest_dir" ]]; then
        if command_exists sudo; then
            sudo cp "$src" "$dest" || error "Installation failed"
            sudo chmod +x "$dest" || error "Failed to set executable permission"
        else
            error "No write permission to $dest_dir and sudo is not available"
        fi
    else
        cp "$src" "$dest" || error "Installation failed"
        chmod +x "$dest" || error "Failed to set executable permission"
    fi

    success "Binary installed successfully"
}

# Check if directory is in PATH
is_in_path() {
    local dir="$1"
    [[ ":$PATH:" == *":$dir:"* ]]
}

# Show post-installation message
show_post_install() {
    local install_dir="$1"
    local installed_version="$2"

    echo ""
    success "ReqTap has been installed!"
    echo ""
    info "Installed version: ${BOLD}${installed_version}${NC}"
    info "Installation path: ${BOLD}${install_dir}/$BINARY_NAME${NC}"
    echo ""
    
    # Check if installation directory is in PATH
    if ! is_in_path "$install_dir"; then
        warn "Installation directory is not in your PATH"
        echo ""
        echo "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo -e "    ${CYAN}export PATH=\"$install_dir:\$PATH\"${NC}"
        echo ""
    fi

    echo "Run the following command to verify installation:"
    echo ""
    echo -e "    ${CYAN}$BINARY_NAME --version${NC}"
    echo ""
}

# Install ReqTap
cmd_install() {
    echo ""
    title "Installing ReqTap"
    echo ""

    # Detect platform
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "Detected platform: ${BOLD}$os/$arch${NC}"

    # Check dependencies
    check_dependencies "$os"

    # Check if already installed
    local existing_binary existing_version
    if existing_binary=$(find_installed_binary); then
        existing_version=$(get_installed_version) || existing_version="unknown"
        warn "ReqTap is already installed: $existing_binary"
        info "Current version: $existing_version"
        
        if [[ "$FORCE_INSTALL" == false ]]; then
            echo ""
            read -p "Do you want to reinstall/upgrade? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                info "Installation cancelled"
                return 0
            fi
        fi
        echo ""
    fi

    # Get version
    if [[ -z "$VERSION" ]]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version) || error "Failed to fetch latest version. Please specify version manually with -v option."
    fi
    info "Target version: ${BOLD}${VERSION}${NC}"

    # Build filename and URLs
    local archive_name="${BINARY_NAME}-${os}-${arch}"
    local archive_ext=".tar.gz"
    if [[ "$os" == "windows" ]]; then
        archive_ext=".zip"
    fi
    local archive_file="${archive_name}${archive_ext}"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${archive_file}"
    local checksums_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/checksums.txt"

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '$tmp_dir'" EXIT

    # Download archive
    info "Downloading from GitHub..."
    echo ""
    if ! download_file "$download_url" "$tmp_dir/$archive_file"; then
        error "Failed to download: $download_url"
    fi
    echo ""
    success "Download completed"

    # Download and verify checksums
    local checksum=""
    if [[ "$SKIP_CHECKSUM" == false ]]; then
        if download_file "$checksums_url" "$tmp_dir/checksums.txt" 2>/dev/null; then
            checksum=$(get_checksum "$archive_file" "$tmp_dir/checksums.txt")
            if [[ -n "$checksum" ]]; then
                verify_checksum "$tmp_dir/$archive_file" "$checksum"
            else
                warn "Checksum not found in checksums file, skipping verification"
            fi
        else
            warn "Checksums file not available, skipping verification"
        fi
    fi

    # Extract archive
    extract_archive "$tmp_dir/$archive_file" "$tmp_dir/extracted"

    # Find binary in extracted files
    local binary_path
    binary_path=$(find "$tmp_dir/extracted" -name "$BINARY_NAME" -type f 2>/dev/null | head -n 1)
    
    if [[ -z "$binary_path" ]]; then
        # Try with .exe extension for Windows
        binary_path=$(find "$tmp_dir/extracted" -name "${BINARY_NAME}.exe" -type f 2>/dev/null | head -n 1)
    fi

    if [[ -z "$binary_path" ]]; then
        error "Binary not found in archive"
    fi

    # Determine installation directory
    if [[ -z "$INSTALL_DIR" ]]; then
        INSTALL_DIR=$(determine_install_dir)
    fi

    # Install binary
    install_binary "$binary_path" "$INSTALL_DIR"

    # Show post-installation message
    show_post_install "$INSTALL_DIR" "$VERSION"
}

# Uninstall ReqTap
cmd_uninstall() {
    echo ""
    title "Uninstalling ReqTap"
    echo ""

    # Find binary
    local binary_path
    if ! binary_path=$(find_installed_binary); then
        warn "ReqTap is not installed or not found in common locations"
        echo ""
        echo "If installed in a custom location, please remove it manually"
        return 1
    fi

    local version
    version=$(get_installed_version) || version="unknown"
    
    info "Found installation: ${BOLD}$binary_path${NC}"
    info "Version: $version"
    echo ""

    # Ask for confirmation
    if [[ "${SKIP_CONFIRMATION:-false}" != true ]]; then
        read -p "Are you sure you want to uninstall? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Uninstall cancelled"
            return 0
        fi
    fi

    # Remove binary
    local dir
    dir=$(dirname "$binary_path")
    if [[ -w "$dir" ]]; then
        rm -f "$binary_path"
    else
        if command_exists sudo; then
            sudo rm -f "$binary_path"
        else
            error "No permission to remove $binary_path and sudo is not available"
        fi
    fi

    # Remove backup files if any
    local backups
    backups=$(find "$dir" -name "${BINARY_NAME}.backup.*" 2>/dev/null)
    if [[ -n "$backups" ]]; then
        info "Found backup files:"
        echo "$backups"
        echo ""
        read -p "Do you want to remove backup files too? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if [[ -w "$dir" ]]; then
                find "$dir" -name "${BINARY_NAME}.backup.*" -delete
            else
                sudo find "$dir" -name "${BINARY_NAME}.backup.*" -delete
            fi
            success "Backup files removed"
        fi
    fi

    echo ""
    success "ReqTap has been uninstalled successfully"
}

# Update ReqTap
cmd_update() {
    echo ""
    title "Updating ReqTap"
    echo ""

    # Check if installed
    local binary_path current_version latest_version
    if ! binary_path=$(find_installed_binary); then
        warn "ReqTap is not installed"
        echo ""
        read -p "Do you want to install it now? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            cmd_install
        fi
        return
    fi

    current_version=$(get_installed_version) || current_version="unknown"
    info "Current version: ${BOLD}$current_version${NC}"

    info "Checking for updates..."
    latest_version=$(get_latest_version) || error "Failed to check for updates"
    info "Latest version: ${BOLD}$latest_version${NC}"

    if [[ "$current_version" == *"$latest_version"* ]]; then
        success "You already have the latest version installed"
        return 0
    fi

    echo ""
    info "A new version is available!"
    echo ""

    if [[ "${SKIP_CONFIRMATION:-false}" != true ]]; then
        read -p "Do you want to update? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Update cancelled"
            return 0
        fi
    fi

    # Set version and force install
    VERSION="$latest_version"
    FORCE_INSTALL=true
    
    # Use the existing installation directory
    INSTALL_DIR=$(dirname "$binary_path")

    cmd_install
}

# Check version and updates
cmd_check() {
    echo ""
    title "Checking ReqTap Installation"
    echo ""

    # Check if installed
    local binary_path current_version
    if binary_path=$(find_installed_binary); then
        current_version=$(get_installed_version) || current_version="unknown"
        success "ReqTap is installed"
        info "Location: ${BOLD}$binary_path${NC}"
        info "Current version: ${BOLD}$current_version${NC}"
    else
        warn "ReqTap is not installed"
        echo ""
        return 1
    fi

    # Check for updates
    echo ""
    info "Checking for updates..."
    local latest_version
    if latest_version=$(get_latest_version); then
        info "Latest version: ${BOLD}$latest_version${NC}"
        
        if [[ "$current_version" == *"$latest_version"* ]]; then
            echo ""
            success "You have the latest version installed"
        else
            echo ""
            warn "A new version is available: $latest_version"
            info "Run '$0 update' to upgrade"
        fi
    else
        warn "Failed to check for updates"
    fi

    # Check if in PATH
    local dir
    dir=$(dirname "$binary_path")
    echo ""
    if is_in_path "$dir"; then
        success "Installation directory is in PATH"
    else
        warn "Installation directory is not in PATH"
        echo ""
        echo "Add the following line to your shell profile:"
        echo -e "    ${CYAN}export PATH=\"$dir:\$PATH\"${NC}"
    fi
}

# List all available versions
cmd_list() {
    echo ""
    title "Available Versions"
    echo ""

    info "Fetching version list from GitHub..."
    local versions
    if ! versions=$(get_all_versions); then
        error "Failed to fetch version list"
    fi

    if [[ -z "$versions" ]]; then
        warn "No versions found"
        return 1
    fi

    echo ""
    echo "$versions" | while read -r version; do
        echo "  • $version"
    done
    echo ""
    
    # Show current version if installed
    local current_version
    if current_version=$(get_installed_version) 2>/dev/null; then
        info "Currently installed: ${BOLD}$current_version${NC}"
    fi
}

# Interactive menu
interactive_menu() {
    print_banner

    # Check installation status
    local is_installed=false
    local installed_version=""
    if binary_path=$(find_installed_binary); then
        is_installed=true
        installed_version=$(get_installed_version) || installed_version="unknown"
    fi

    while true; do
        echo ""
        title "Please select an option:"
        echo ""
        
        if [[ "$is_installed" == true ]]; then
            echo -e "  ${GREEN}[Installed]${NC} Version: $installed_version"
            echo ""
        fi

        echo "  1) Install ReqTap"
        echo "  2) Uninstall ReqTap"
        echo "  3) Update ReqTap"
        echo "  4) Check installation status"
        echo "  5) List available versions"
        echo "  6) Exit"
        echo ""
        read -p "Enter your choice [1-6]: " choice

        case $choice in
            1)
                cmd_install
                is_installed=true
                installed_version=$(get_installed_version) || installed_version="unknown"
                ;;
            2)
                if cmd_uninstall; then
                    is_installed=false
                    installed_version=""
                fi
                ;;
            3)
                cmd_update
                installed_version=$(get_installed_version) || installed_version="unknown"
                ;;
            4)
                cmd_check
                ;;
            5)
                cmd_list
                ;;
            6)
                echo ""
                info "Goodbye!"
                exit 0
                ;;
            *)
                warn "Invalid choice. Please enter a number between 1 and 6."
                ;;
        esac

        echo ""
        read -p "Press Enter to continue..." dummy
        clear
        print_banner
    done
}

# Main function
main() {
    # If run as piped script (e.g., curl | bash), default to install
    if [[ ! -t 0 ]]; then
        parse_args "$@"
        
        # Default command is install if none specified
        if [[ -z "$COMMAND" ]]; then
            COMMAND="install"
        fi
    else
        parse_args "$@"
        
        # If no command and no args, show interactive menu
        if [[ -z "$COMMAND" ]] && [[ $# -eq 0 ]]; then
            interactive_menu
            exit 0
        fi
        
        # Default to install if no command
        if [[ -z "$COMMAND" ]]; then
            COMMAND="install"
        fi
    fi

    # Execute command
    case "$COMMAND" in
        install)
            cmd_install
            ;;
        uninstall|remove)
            cmd_uninstall
            ;;
        update|upgrade)
            cmd_update
            ;;
        check|status)
            cmd_check
            ;;
        list|versions)
            cmd_list
            ;;
        help|--help|-h)
            print_banner
            usage
            ;;
        *)
            error "Unknown command: $COMMAND\nUse 'help' for usage information"
            ;;
    esac
}

# Run main function
main "$@"
