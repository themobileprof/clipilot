#!/bin/bash
# Clio Uninstall Script
# Removes Clio binary, data, and configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║            Clio Uninstall Utility                   ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# Detect environment
IS_TERMUX=false
if [ -n "$TERMUX_VERSION" ] || [ -n "$PREFIX" ]; then
    IS_TERMUX=true
fi

# Determine installation paths
if [ "$IS_TERMUX" = true ]; then
    INSTALL_DIR="$PREFIX/bin"
else
    # Check common installation locations
    if [ -f "/usr/local/bin/clipilot" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -f "$HOME/.local/bin/clipilot" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    elif [ -f "$HOME/bin/clipilot" ]; then
        INSTALL_DIR="$HOME/bin"
    else
        INSTALL_DIR=""
    fi
fi

DATA_DIR="$HOME/.clipilot"
BINARY_NAME="clipilot"

# Show what will be removed
echo -e "${BLUE}📋 Found the following Clio components:${NC}"
echo ""

FOUND_BINARY=false
FOUND_DATA=false

if [ -n "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo -e "  ${GREEN}✓${NC} Binary: $INSTALL_DIR/$BINARY_NAME"
    FOUND_BINARY=true
else
    echo -e "  ${YELLOW}○${NC} Binary: Not found in standard locations"
fi

if [ -d "$DATA_DIR" ]; then
    SIZE=$(du -sh "$DATA_DIR" 2>/dev/null | cut -f1)
    echo -e "  ${GREEN}✓${NC} Data directory: $DATA_DIR ($SIZE)"
    FOUND_DATA=true
else
    echo -e "  ${YELLOW}○${NC} Data directory: Not found"
fi

echo ""

# Check if anything was found
if [ "$FOUND_BINARY" = false ] && [ "$FOUND_DATA" = false ]; then
    echo -e "${YELLOW}⚠️  Clio does not appear to be installed.${NC}"
    echo ""
    echo "If you installed Clio in a custom location, you'll need to remove it manually."
    exit 0
fi

# Confirm uninstallation
echo -e "${YELLOW}⚠️  This will permanently delete:${NC}"
if [ "$FOUND_BINARY" = true ]; then
    echo "   • Clio binary ($BINARY_NAME)"
fi
if [ "$FOUND_DATA" = true ]; then
    echo "   • All your modules and settings"
    echo "   • Command index and search history"
    echo "   • Execution logs"
fi
echo ""
echo -e "${RED}This action cannot be undone!${NC}"
echo ""
read -p "Continue with uninstallation? [y/N] " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Uninstallation cancelled.${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}🗑️  Uninstalling Clio...${NC}"
echo ""

# Remove binary
if [ "$FOUND_BINARY" = true ]; then
    if rm -f "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Removed binary from $INSTALL_DIR"
    else
        # Try with sudo if permission denied
        if command -v sudo >/dev/null 2>&1; then
            if sudo rm -f "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null; then
                echo -e "${GREEN}✓${NC} Removed binary from $INSTALL_DIR (using sudo)"
            else
                echo -e "${RED}✗${NC} Failed to remove binary from $INSTALL_DIR"
                echo "   You may need to remove it manually with: sudo rm $INSTALL_DIR/$BINARY_NAME"
            fi
        else
            echo -e "${RED}✗${NC} Failed to remove binary from $INSTALL_DIR"
            echo "   You may need to remove it manually"
        fi
    fi
fi

# Remove data directory
if [ "$FOUND_DATA" = true ]; then
    if rm -rf "$DATA_DIR" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Removed data directory $DATA_DIR"
    else
        echo -e "${RED}✗${NC} Failed to remove data directory"
        echo "   You may need to remove it manually with: rm -rf $DATA_DIR"
    fi
fi

echo ""
echo -e "${GREEN}✅ Clio has been uninstalled!${NC}"
echo ""
echo -e "${BLUE}📝 Note:${NC} The following were ${YELLOW}not${NC} removed:"
echo "   • Packages installed using Clio modules"
echo "   • System commands indexed by Clio"
echo "   • Any files created using Clio"
echo ""
echo "Thank you for trying Clio! 👋"
echo ""
echo "To reinstall later, visit: https://github.com/themobileprof/clipilot"
echo ""
