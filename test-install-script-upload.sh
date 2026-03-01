#!/bin/bash
# Test script for uploading Clio install script to registry
# This mimics what the CI/CD workflow does

set -e

echo "🧪 Testing Clio install script upload process..."
echo ""

# Check if running as root (needed for volume access)
if [ "$EUID" -ne 0 ] && [ ! -w "$(docker volume inspect clipilot-registry-data --format '{{.Mountpoint}}' 2>/dev/null || echo '/tmp')" ]; then 
  echo "⚠️  This script needs root access to write to Docker volumes"
  echo "   Run with: sudo $0"
  exit 1
fi

# Download Clio install script
echo "📥 Fetching Clio install script from GitHub..."
curl -fsSL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh -o /tmp/clio-install.sh

if [ ! -f /tmp/clio-install.sh ]; then
  echo "❌ Failed to download install script"
  exit 1
fi

echo "📤 Uploading install script to registry..."

# Extract version from script
SCRIPT_VERSION=$(grep -m1 '^VERSION=' /tmp/clio-install.sh | cut -d'=' -f2 | tr -d '"' || echo "auto")
echo "   Version detected: $SCRIPT_VERSION"

# Get volume mount point
VOLUME_PATH=$(docker volume inspect clipilot-registry-data --format '{{.Mountpoint}}')
echo "   Volume path: $VOLUME_PATH"

# Ensure sqlite3 is installed
if ! command -v sqlite3 &> /dev/null; then
  echo "📦 Installing sqlite3..."
  apt-get update -qq && apt-get install -y sqlite3 > /dev/null 2>&1 || {
    echo "❌ Could not install sqlite3"
    exit 1
  }
fi

# Create uploads directory
mkdir -p "$VOLUME_PATH/uploads"

# Copy script to uploads directory
TIMESTAMP=$(date +%s)
DEST_FILE="$VOLUME_PATH/uploads/install-${TIMESTAMP}.sh"
cp /tmp/clio-install.sh "$DEST_FILE"
chmod 644 "$DEST_FILE"

# Calculate checksum
CHECKSUM=$(sha256sum "$DEST_FILE" | awk '{print $1}')

# Insert into database
echo "💾 Updating database..."
sqlite3 "$VOLUME_PATH/registry.db" <<EOF
UPDATE install_scripts SET is_active = 0 WHERE is_active = 1;
INSERT INTO install_scripts (file_path, version, checksum_sha256, uploaded_at, is_active)
VALUES ('/app/data/uploads/install-${TIMESTAMP}.sh', '$SCRIPT_VERSION', '$CHECKSUM', datetime('now'), 1);
EOF

if [ $? -eq 0 ]; then
  echo ""
  echo "✅ Install script deployed successfully"
  echo "   Version: $SCRIPT_VERSION"
  echo "   Checksum: $CHECKSUM"
  echo "   File: $DEST_FILE"
  echo ""
  echo "📍 Test it: curl http://localhost:8080/clio"
  echo "   (Server must be running on port 8080)"
else
  echo "❌ Database update failed"
  exit 1
fi

# Cleanup
rm -f /tmp/clio-install.sh

echo ""
echo "🎉 Upload test completed!"
