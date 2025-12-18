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

# Detect Termux environment early
IS_TERMUX=false
if [ -n "$TERMUX_VERSION" ] || [ -n "$PREFIX" ]; then
    IS_TERMUX=true
    echo -e "${YELLOW}📱 Termux environment detected${NC}"
    echo -e "${GREEN}Termux is a first-class platform for CLIPilot!${NC}"
    echo ""
fi

# Detect installation directory
if [ "$IS_TERMUX" = true ]; then
    INSTALL_DIR="$PREFIX/bin"
    CONFIG_DIR="$HOME/.clipilot"
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    CONFIG_DIR="$HOME/.clipilot"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    CONFIG_DIR="$HOME/.clipilot"
else
    INSTALL_DIR="$HOME/bin"
    CONFIG_DIR="$HOME/.clipilot"
    mkdir -p "$INSTALL_DIR"
fi

echo -e "${YELLOW}Installing to: ${INSTALL_DIR}${NC}"
if [ "$IS_TERMUX" = true ]; then
    echo -e "${GREEN}Config directory: ${CONFIG_DIR}${NC}"
fi
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
    armv7l|armv8l)
        ARCH="armv7"
        ;;
    *)
        echo -e "${YELLOW}⚠️  Unknown architecture: $ARCH${NC}"
        if [ "$IS_TERMUX" = true ]; then
            echo "Attempting to use ARM64 binary..."
            ARCH="arm64"
        else
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
        fi
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
        
        if [ "$IS_TERMUX" = true ]; then
            echo "📱 Termux Installation Instructions:"
            echo ""
            echo "1. Install Go in Termux:"
            echo "   pkg update && pkg install golang git"
            echo ""
            echo "2. Then run this installer again:"
            echo "   bash <(curl -fsSL https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/main/install.sh)"
            echo ""
            echo "Or install manually:"
            echo "   git clone https://github.com/${REPO_OWNER}/${REPO_NAME}.git"
            echo "   cd ${REPO_NAME}"
            echo "   go build -o clipilot ./cmd/clipilot"
            echo "   mv clipilot \$PREFIX/bin/"
        else
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
        fi
        exit 1
    fi
    
    # Clone and build
    TMP_DIR=$(mktemp -d)
    echo "Building from source..."
    git clone --depth 1 -q "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" "$TMP_DIR" || {
        echo -e "${RED}✗ Failed to clone repository${NC}"
        exit 1
    }
    cd "$TMP_DIR"
    
    # For Termux, ensure we have necessary build tools
    if [ "$IS_TERMUX" = true ]; then
        if ! command -v clang &> /dev/null; then
            pkg update -y >/dev/null 2>&1
            pkg install -y clang >/dev/null 2>&1
        fi
    fi
    
    go mod download >/dev/null 2>&1 || {
        echo -e "${RED}✗ Failed to download dependencies${NC}"
        exit 1
    }
    
    if [ "$IS_TERMUX" = true ]; then
        echo "Compiling (2-5 minutes)..."
    fi
    
    CGO_ENABLED=0 go build -ldflags="-s -w" -o clipilot ./cmd/clipilot >/dev/null 2>&1 || {
        echo -e "${RED}✗ Build failed. Try: pkg install golang git${NC}"
        exit 1
    }
    
    mv clipilot "${INSTALL_DIR}/clipilot" || {
        echo -e "${RED}✗ Failed to install binary${NC}"
        exit 1
    }
    
    # Copy modules from cloned repo
    CLONED_MODULES_DIR="$TMP_DIR/modules"
    cd - > /dev/null
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
if [ -n "$CLONED_MODULES_DIR" ] && [ -d "$CLONED_MODULES_DIR" ]; then
    cp "$CLONED_MODULES_DIR"/*.yaml "${CONFIG_DIR}/modules/" 2>/dev/null || true
    rm -rf "$TMP_DIR"
    MODULE_COUNT=$(ls -1 "${CONFIG_DIR}/modules/"*.yaml 2>/dev/null | wc -l)
    echo -e "${GREEN}✓ Installed $MODULE_COUNT modules${NC}"
else
    MODULES=("detect_os.yaml" "git_setup.yaml" "docker_install.yaml" "nginx_setup.yaml" "nodejs_setup.yaml" "python_dev_setup.yaml")
    for module in "${MODULES[@]}"; do
        if command -v curl &> /dev/null; then
            curl -fsSL "${MODULES_BASE_URL}/${module}" -o "${CONFIG_DIR}/modules/${module}" 2>/dev/null || true
        else
            wget -q "${MODULES_BASE_URL}/${module}" -O "${CONFIG_DIR}/modules/${module}" 2>/dev/null || true
        fi
    done
    rm -rf "$TMP_DIR"
    MODULE_COUNT=$(ls -1 "${CONFIG_DIR}/modules/"*.yaml 2>/dev/null | wc -l)
    echo -e "${GREEN}✓ Installed $MODULE_COUNT core modules${NC}"
fi

# Verify binary works before initializing
if ! "${INSTALL_DIR}/clipilot" --version >/dev/null 2>&1; then
    echo -e "${RED}✗ Binary verification failed${NC}"
    
    # Show diagnostic information
    echo ""
    echo "Diagnostic information:"
    echo "  OS: ${OS}"
    echo "  Arch: ${ARCH}"
    echo "  Binary: ${INSTALL_DIR}/clipilot"
    echo "  File type: $(file "${INSTALL_DIR}/clipilot" 2>/dev/null || echo 'file command not available')"
    echo ""
    echo "Attempting to run binary with error output:"
    "${INSTALL_DIR}/clipilot" --version 2>&1 | head -5
    echo ""
    
    if [ "$IS_TERMUX" = true ]; then
        echo "Building from source..."
        
        # Install Go if not present
        if ! command -v go &> /dev/null; then
            pkg update >/dev/null 2>&1
            pkg install -y golang git clang
        fi
        
        # Build from source
        TMP_DIR=$(mktemp -d)
        cd "$TMP_DIR"
        git clone --depth 1 "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" . || exit 1
        
        # Copy modules before building
        echo "Copying modules..."
        mkdir -p "$CONFIG_DIR/modules"
        cp modules/*.yaml "$CONFIG_DIR/modules/" 2>/dev/null || true
        
        go mod download || exit 1
        CGO_ENABLED=1 go build -ldflags="-s -w" -o clipilot ./cmd/clipilot || exit 1
        mv clipilot "${INSTALL_DIR}/clipilot"
        chmod +x "${INSTALL_DIR}/clipilot"
        cd - > /dev/null
        rm -rf "$TMP_DIR"
        
        echo -e "${GREEN}✓ Built from source${NC}"
        echo -e "${GREEN}✓ Modules copied${NC}"
    else
        echo "Please try building from source:"
        echo "  git clone https://github.com/${REPO_OWNER}/${REPO_NAME}.git"
        echo "  cd ${REPO_NAME}"
        echo "  go build -o clipilot ./cmd/clipilot"
        exit 1
    fi
fi
echo -e "${GREEN}✓ Binary verified${NC}"

# Initialize database
"${INSTALL_DIR}/clipilot" --init --load="${CONFIG_DIR}/modules" >/dev/null 2>&1
echo -e "${GREEN}✓ Database initialized${NC}"

# Sync with registry to get full module library
echo "📦 Syncing with module registry..."
if "${INSTALL_DIR}/clipilot" sync >/dev/null 2>&1; then
    SYNCED_COUNT=$("${INSTALL_DIR}/clipilot" modules list --available 2>/dev/null | wc -l || echo "0")
    echo -e "${GREEN}✓ Registry synced (${SYNCED_COUNT} modules available)${NC}"
else
    echo -e "${YELLOW}⚠️  Registry sync failed (you can run 'clipilot sync' later)${NC}"
fi

# Check if install dir is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
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
        if ! grep -q "export PATH.*${INSTALL_DIR}" "$SHELL_RC" 2>/dev/null; then
            echo "" >> "$SHELL_RC"
            echo "# Added by CLIPilot installer" >> "$SHELL_RC"
            echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$SHELL_RC"
        fi
        export PATH="$PATH:${INSTALL_DIR}"
    fi
fi

echo ""
echo -e "${GREEN}✓ CLIPilot installed successfully!${NC}"
echo ""
if [ "$IS_TERMUX" = true ]; then
    echo "Try: clipilot run termux_setup"
else
    echo "Try: clipilot"
fi
echo ""

# Offer to start CLIPilot interactively (with proper prompt handling)
if [ -t 0 ]; then
    echo -n "Start CLIPilot now? [Y/n]: "
    read -r response
    response=${response,,}
    if [ -z "$response" ] || [ "$response" = "y" ] || [ "$response" = "yes" ]; then
        echo ""
        "${INSTALL_DIR}/clipilot"
    fi
fi
