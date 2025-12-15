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
    echo "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO_OWNER}/${REPO_NAME}.git" "$TMP_DIR" 2>&1 || {
        echo -e "${RED}Error: Failed to clone repository${NC}"
        exit 1
    }
    cd "$TMP_DIR"
    
    # For Termux, ensure we have necessary build tools
    if [ "$IS_TERMUX" = true ]; then
        echo "Checking Termux build dependencies..."
        MISSING_DEPS=""
        
        if ! command -v clang &> /dev/null; then
            MISSING_DEPS="$MISSING_DEPS clang"
        fi
        
        if [ -n "$MISSING_DEPS" ]; then
            echo -e "${YELLOW}Installing required dependencies:$MISSING_DEPS${NC}"
            pkg update -y 2>&1 | grep -v "dpkg: warning" || true
            pkg install -y $MISSING_DEPS 2>&1 | grep -v "dpkg: warning" || true
            echo -e "${GREEN}✓ Dependencies installed${NC}"
        else
            echo -e "${GREEN}✓ All dependencies present${NC}"
        fi
    fi
    
    echo "Downloading dependencies..."
    go mod download || {
        echo -e "${RED}Error: Failed to download Go dependencies${NC}"
        exit 1
    }
    
    if [ "$IS_TERMUX" = true ]; then
        echo -e "${YELLOW}Building binary for Termux (this may take 2-5 minutes)...${NC}"
        echo "Tip: Keep your device plugged in and screen awake for faster builds"
    else
        echo "Building binary (this may take a few minutes)..."
    fi
    
    CGO_ENABLED=1 go build -ldflags="-s -w" -o clipilot ./cmd/clipilot || {
        echo -e "${RED}Error: Build failed${NC}"
        if [ "$IS_TERMUX" = true ]; then
            echo ""
            echo -e "${YELLOW}Termux troubleshooting:${NC}"
            echo "1. Ensure you have enough storage space (at least 500MB free)"
            echo "2. Install required packages: pkg install golang git clang"
            echo "3. Try clearing Go cache: go clean -cache -modcache"
            echo "4. Check for package updates: pkg update && pkg upgrade"
        fi
        exit 1
    }
    
    mv clipilot "${INSTALL_DIR}/clipilot" || {
        echo -e "${RED}Error: Failed to move binary to ${INSTALL_DIR}${NC}"
        exit 1
    }
    
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
    echo "Copying all modules from repository..."
    MODULE_COUNT=$(ls -1 "$CLONED_MODULES_DIR"/*.yaml 2>/dev/null | wc -l)
    cp "$CLONED_MODULES_DIR"/*.yaml "${CONFIG_DIR}/modules/" 2>/dev/null || true
    rm -rf "$TMP_DIR"
    echo -e "${GREEN}✓ $MODULE_COUNT modules installed${NC}"
    if [ "$IS_TERMUX" = true ]; then
        echo -e "${GREEN}✓ All modules are Termux-compatible!${NC}"
    fi
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
    echo -e "${GREEN}✓ Modules downloaded${NC}"
fi

# Verify binary works before initializing
echo ""
echo "🔍 Verifying binary..."
if ! "${INSTALL_DIR}/clipilot" --version &>/dev/null; then
    echo -e "${RED}✗ Binary verification failed${NC}"
    
    if [ "$IS_TERMUX" = true ]; then
        echo ""
        echo "The downloaded binary is not compatible with your device."
        echo "This usually means:"
        echo "  • Wrong architecture detected"
        echo "  • Your device architecture: $(uname -m)"
        echo ""
        echo "Installing Go and building from source..."
        echo ""
        
        # Install Go if not present
        if ! command -v go &> /dev/null; then
            echo "Installing Go and build dependencies..."
            pkg update
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

if [ "$IS_TERMUX" = true ]; then
    echo -e "${GREEN}📱 Termux-Optimized Installation Complete!${NC}"
    echo ""
    echo "🎯 Recommended first steps for Termux:"
    echo "  clipilot run termux_setup     - Complete Termux environment setup"
    echo "  clipilot run setup_development_environment - Install dev tools"
    echo ""
    echo "💡 Quick Start:"
    echo "  clipilot                      - Start interactive mode"
    echo "  clipilot search phone         - Find phone-related modules"
    echo "  clipilot run <module>         - Run a specific module"
    echo ""
    echo "📚 Popular Termux modules available:"
    echo "  • termux_setup - Configure Termux environment"
    echo "  • git_setup - Install and configure Git"
    echo "  • python_dev_setup - Python development environment"
    echo "  • nodejs_setup - Node.js development environment"
    echo "  • database_clients_install - Database tools"
    echo "  • modern_cli_tools_install - Modern CLI utilities"
    echo ""
    echo "💾 Storage tip: Run 'termux-setup-storage' to access phone storage"
else
    echo "Quick Start:"
    echo "  clipilot              - Start interactive mode"
    echo "  clipilot help         - Show available commands"
    echo "  clipilot search git   - Search for modules"
    echo "  clipilot run git_setup - Run a specific module"
    echo ""
fi

echo "For more information, visit:"
echo "  https://github.com/${REPO_OWNER}/${REPO_NAME}"
if [ "$IS_TERMUX" = true ]; then
    echo "  Termux guide: https://github.com/${REPO_OWNER}/${REPO_NAME}/blob/main/docs/TERMUX.md"
fi
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
