#!/bin/bash
# Manual production deployment for CLIPilot Registry (native binary + systemd).
# Run on the production server with sudo.

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/clipilot-registry}"
DATA_DIR="${DATA_DIR:-/var/lib/clipilot-registry}"
ENV_FILE="${ENV_FILE:-/etc/clipilot-registry/env}"
STAGING_DIR="${STAGING_DIR:-/tmp/clipilot-deploy}"

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root: sudo $0"
  exit 1
fi

if [ ! -f "$ENV_FILE" ] && [ -f "$REPO_DIR/.env.production" ]; then
  mkdir -p "$(dirname "$ENV_FILE")"
  cp "$REPO_DIR/.env.production" "$ENV_FILE"
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: $ENV_FILE not found. Create it from .env.example first."
  exit 1
fi

echo "Building registry binary..."
cd "$REPO_DIR"
mkdir -p "$STAGING_DIR"
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$STAGING_DIR/registry" ./cmd/registry

mkdir -p "$STAGING_DIR/server"
cp -r server/static server/templates "$STAGING_DIR/server/"
cp deploy/clipilot-registry.service "$STAGING_DIR/"
cp scripts/remote-deploy.sh "$STAGING_DIR/"
chmod +x "$STAGING_DIR/remote-deploy.sh"

DEPLOY_STAGING_DIR="$STAGING_DIR" \
  INSTALL_DIR="$INSTALL_DIR" \
  DATA_DIR="$DATA_DIR" \
  ENV_FILE="$ENV_FILE" \
  bash "$STAGING_DIR/remote-deploy.sh"

rm -rf "$STAGING_DIR"
