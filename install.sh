#!/bin/bash
# CLIPilot Installation Script
# Installs CLIPilot and initializes the database with default modules

set -e  # Exit on error

REPO_OWNER="themobileprof"
REPO_NAME="clipilot"
GITHUB_API="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
MODULES_BASE_URL="https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/main/modules"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================"
echo "  CLIPilot Installation Script"
echo "========================================"
echo ""

# Detect installation directory
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -n "$PREFIX" ]; then
    # Termux support
    INSTALL_DIR="$PREFIX/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo -e "${YELLOW}Installing to: ${INSTALL_DIR}${NC}"
echo ""

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="armv7"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
BINARY_NAME="clipilot-${OS}-${ARCH}"

echo "Detected platform: ${OS}-${ARCH}"
echo ""

# Get latest release info
echo "📥 Fetching latest release..."
RELEASE_DATA=""
if command -v curl &> /dev/null; then
    RELEASE_DATA=$(curl -fsSL "$GITHUB_API" 2>/dev/null || echo "")
elif command -v wget &> /dev/null; then
    RELEASE_DATA=$(wget -qO- "$GITHUB_API" 2>/dev/null || echo "")
else
    echo -e "${RED}Error: curl or wget is required for installation${NC}"
    exit 1
fi

# Check if release exists
if [ -z "$RELEASE_DATA" ] || echo "$RELEASE_DATA" | grep -q "Not Found"; then
    echo -e "${YELLOW}⚠️  No release found. Building from source instead...${NC}"
    echo ""
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is required to build from source${NC}"
        echo "Please install Go from https://golang.org/dl/"
        echo "Or wait for a release to be published at:"
        echo "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
        exit 1
    fi
    
    # Clone and build
    TMP_DIR=$(mktemp -d)
    echo "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" "$TMP_DIR"
    cd "$TMP_DIR"
    echo "Building binary..."
    go build -ldflags="-s -w" -o clipilot ./cmd/clipilot
    mv clipilot "${INSTALL_DIR}/clipilot"
    cd -
    rm -rf "$TMP_DIR"
    echo -e "${GREEN}✓ Binary built and installed${NC}"
    
    MODULES_DIR="${TMP_DIR}/modules"
else
    # Extract download URL for the binary
    DOWNLOAD_URL=$(echo "$RELEASE_DATA" | grep "browser_download_url.*${BINARY_NAME}.tar.gz\"" | cut -d '"' -f 4)

    if [ -z "$DOWNLOAD_URL" ]; then
        echo -e "${RED}Error: Could not find binary for ${OS}-${ARCH}${NC}"
        echo "Available releases:"
        echo "$RELEASE_DATA" | grep "browser_download_url" | cut -d '"' -f 4
        exit 1
    fi

    echo "Downloading from: $DOWNLOAD_URL"

    # Download and extract binary
    TMP_DIR=$(mktemp -d)
    if command -v curl &> /dev/null; then
        curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/clipilot.tar.gz"
    elif command -v wget &> /dev/null; then
        wget -q "$DOWNLOAD_URL" -O "${TMP_DIR}/clipilot.tar.gz"
    fi

    tar -xzf "${TMP_DIR}/clipilot.tar.gz" -C "$TMP_DIR"
    mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/clipilot"
    rm -rf "$TMP_DIR"
fi

# Make executable
chmod +x "${INSTALL_DIR}/clipilot"
echo -e "${GREEN}✓ Binary installed${NC}"

# Create config directory
CONFIG_DIR="$HOME/.clipilot"
mkdir -p "$CONFIG_DIR"
mkdir -p "$CONFIG_DIR/modules"

# Download default modules
echo ""
echo "📦 Downloading default modules..."
MODULES=("detect_os.yaml" "git_setup.yaml" "docker_install.yaml")

for module in "${MODULES[@]}"; do
    echo "  - $module"
    if command -v curl &> /dev/null; then
        curl -fsSL "${MODULES_BASE_URL}/${module}" -o "${CONFIG_DIR}/modules/${module}"
    else
        wget -q "${MODULES_BASE_URL}/${module}" -O "${CONFIG_DIR}/modules/${module}"
    fi
done
echo -e "${GREEN}✓ Modules downloaded${NC}"

# Initialize database
echo ""
echo "🗄️  Initializing database..."
"${INSTALL_DIR}/clipilot" --init --load="${CONFIG_DIR}/modules"
echo -e "${GREEN}✓ Database initialized${NC}"

# Check if install dir is in PATH
echo ""
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo -e "${YELLOW}⚠️  ${INSTALL_DIR} is not in your PATH${NC}"
    echo ""
    echo "Add it to your PATH by adding this line to your ~/.bashrc or ~/.zshrc:"
    echo ""
    echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "Then reload your shell with: source ~/.bashrc"
else
    echo -e "${GREEN}✓ Installation directory is in PATH${NC}"
fi

echo ""
echo "========================================"
echo -e "${GREEN}✓ CLIPilot installed successfully!${NC}"
echo "========================================"
echo ""
echo "Quick Start:"
echo "  clipilot              - Start interactive mode"
echo "  clipilot help         - Show available commands"
echo "  clipilot search git   - Search for modules"
echo "  clipilot run git_setup - Run a specific module"
echo ""
echo "For more information, visit:"
echo "  https://github.com/${REPO_OWNER}/${REPO_NAME}"
echo ""
