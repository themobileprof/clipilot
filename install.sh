#!/bin/bash
# CLIPilot Installation Script
# Installs CLIPilot and initializes the database with default modules

set -e  # Exit on error

REPO_OWNER="themobileprof"
REPO_NAME="clipilot"
BINARY_URL="https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/main/bin/clipilot"
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

# Download binary
echo "📥 Downloading CLIPilot binary..."
if command -v curl &> /dev/null; then
    curl -fsSL "$BINARY_URL" -o "${INSTALL_DIR}/clipilot"
elif command -v wget &> /dev/null; then
    wget -q "$BINARY_URL" -O "${INSTALL_DIR}/clipilot"
else
    echo -e "${RED}Error: curl or wget is required for installation${NC}"
    exit 1
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
