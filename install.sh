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

# Check if release exists and has binaries
DOWNLOAD_URL=""
if [ -n "$RELEASE_DATA" ] && ! echo "$RELEASE_DATA" | grep -q "Not Found"; then
    # Extract download URL for the binary
    DOWNLOAD_URL=$(echo "$RELEASE_DATA" | grep "browser_download_url.*${BINARY_NAME}.tar.gz\"" | cut -d '"' -f 4)
fi

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${YELLOW}⚠️  No pre-built binary found for ${OS}-${ARCH}${NC}"
    echo "Building from source instead..."
    echo ""
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is required to build from source${NC}"
        echo ""
        echo "Option 1: Install Go"
        echo "  https://golang.org/dl/"
        echo ""
        echo "Option 2: Wait for binaries to be published"
        echo "  https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
        echo ""
        echo "Option 3: Build manually in the repo directory:"
        echo "  git clone https://github.com/${REPO_OWNER}/${REPO_NAME}.git"
        echo "  cd ${REPO_NAME}"
        echo "  go build -o clipilot ./cmd/clipilot"
        echo "  sudo mv clipilot /usr/local/bin/"
        exit 1
    fi
    
    # Clone and build
    TMP_DIR=$(mktemp -d)
    echo "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" "$TMP_DIR" 2>&1
    cd "$TMP_DIR"
    echo "Downloading dependencies..."
    go mod download
    echo "Building binary..."
    go build -ldflags="-s -w" -o clipilot ./cmd/clipilot
    mv clipilot "${INSTALL_DIR}/clipilot"
    
    # Copy modules from cloned repo
    CLONED_MODULES_DIR="$TMP_DIR/modules"
    cd - > /dev/null
    
    echo -e "${GREEN}✓ Binary built and installed${NC}"
else
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
    echo -e "${GREEN}✓ Binary installed${NC}"
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

# Check if we have a cloned repo with modules
if [ -n "$CLONED_MODULES_DIR" ] && [ -d "$CLONED_MODULES_DIR" ]; then
    echo "Using modules from cloned repository..."
    cp "$CLONED_MODULES_DIR"/*.yaml "${CONFIG_DIR}/modules/"
    rm -rf "$TMP_DIR"
else
    # Download from GitHub
    for module in "${MODULES[@]}"; do
        echo "  - $module"
        if command -v curl &> /dev/null; then
            curl -fsSL "${MODULES_BASE_URL}/${module}" -o "${CONFIG_DIR}/modules/${module}" 2>/dev/null || echo "    (skipped)"
        else
            wget -q "${MODULES_BASE_URL}/${module}" -O "${CONFIG_DIR}/modules/${module}" 2>/dev/null || echo "    (skipped)"
        fi
    done
    rm -rf "$TMP_DIR"
fi
echo -e "${GREEN}✓ Modules downloaded${NC}"

# Initialize database
echo ""
echo "🗄️  Initializing database..."
"${INSTALL_DIR}/clipilot" --init --load="${CONFIG_DIR}/modules"
echo -e "${GREEN}✓ Database initialized${NC}"

# Check if install dir is in PATH
echo ""
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo -e "${YELLOW}⚠️  Adding ${INSTALL_DIR} to PATH...${NC}"
    
    # Detect shell and add to appropriate RC file
    SHELL_RC=""
    if [ -n "$BASH_VERSION" ]; then
        SHELL_RC="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
    elif [ -f "$HOME/.profile" ]; then
        SHELL_RC="$HOME/.profile"
    fi
    
    if [ -n "$SHELL_RC" ]; then
        # Check if already in RC file
        if ! grep -q "export PATH.*${INSTALL_DIR}" "$SHELL_RC" 2>/dev/null; then
            echo "" >> "$SHELL_RC"
            echo "# Added by CLIPilot installer" >> "$SHELL_RC"
            echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$SHELL_RC"
            echo -e "${GREEN}✓ Added to $SHELL_RC${NC}"
            echo "  Run: source $SHELL_RC"
        fi
        # Add to current session
        export PATH="$PATH:${INSTALL_DIR}"
    else
        echo "Please add this to your shell config:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
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

# Offer to start CLIPilot interactively
echo -e "${YELLOW}Would you like to start CLIPilot now? [Y/n]:${NC} "
read -r response
response=${response,,}  # Convert to lowercase
if [ -z "$response" ] || [ "$response" = "y" ] || [ "$response" = "yes" ]; then
    echo ""
    echo "Starting CLIPilot..."
    echo ""
    "${INSTALL_DIR}/clipilot"
fi
