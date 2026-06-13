#!/bin/bash
# Test script for uploading Clio install script to registry

set -euo pipefail

DATA_DIR="${DATA_DIR:-$HOME/clipilot-data}"
DB_PATH="$DATA_DIR/registry.db"

if [ ! -f "$DB_PATH" ]; then
  echo "Error: Database not found at $DB_PATH"
  echo "Start the registry service first, or set DATA_DIR."
  exit 1
fi

echo "Testing Clio install script upload process..."
echo ""

curl -fsSL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh -o /tmp/clio-install.sh

if [ ! -f /tmp/clio-install.sh ]; then
  echo "Failed to download install script"
  exit 1
fi

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "Installing sqlite3..."
  apt-get update -qq && apt-get install -y sqlite3 >/dev/null 2>&1
fi

SCRIPT_VERSION=$(grep -m1 '^VERSION=' /tmp/clio-install.sh | cut -d'=' -f2 | tr -d '"' || echo "auto")
CHECKSUM=$(sha256sum /tmp/clio-install.sh | awk '{print $1}')
FILENAME="install-${SCRIPT_VERSION}-${CHECKSUM:0:8}.sh"
UPLOAD_DIR="$DATA_DIR/uploads/install_scripts"
FILE_PATH="$UPLOAD_DIR/$FILENAME"

mkdir -p "$UPLOAD_DIR"
cp /tmp/clio-install.sh "$FILE_PATH"
chmod 644 "$FILE_PATH"

sqlite3 "$DB_PATH" <<EOF
UPDATE install_scripts SET is_active = 0 WHERE is_active = 1;
INSERT INTO install_scripts (version, file_path, checksum_sha256, size_bytes, uploaded_by, is_active, uploaded_at)
VALUES ('$SCRIPT_VERSION', '$FILE_PATH', '$CHECKSUM', $(wc -c < "$FILE_PATH"), NULL, 1, CURRENT_TIMESTAMP);
EOF

rm -f /tmp/clio-install.sh

echo "Install script deployed successfully"
echo "Version:  $SCRIPT_VERSION"
echo "Checksum: $CHECKSUM"
echo "File:     $FILE_PATH"
echo ""
echo "Test: curl http://localhost:${PORT:-8080}/clio"
