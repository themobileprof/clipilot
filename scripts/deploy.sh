#!/bin/bash
# Manual deployment for CLIPilot Registry (user systemd, no sudo).
# Run on the production server as your SSH user (same as GitHub SSH_USERNAME).

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGING_DIR="${STAGING_DIR:-/tmp/clipilot-deploy}"
ENV_SOURCE="${ENV_SOURCE:-$REPO_DIR/.env.production}"

if [ ! -f "$ENV_SOURCE" ] && [ -f "$REPO_DIR/.env" ]; then
  ENV_SOURCE="$REPO_DIR/.env"
fi

if [ ! -f "$ENV_SOURCE" ]; then
  echo "Error: env file not found. Set ENV_SOURCE or create .env.production"
  exit 1
fi

echo "Building registry binary..."
cd "$REPO_DIR"
mkdir -p "$STAGING_DIR/server"
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$STAGING_DIR/registry" ./cmd/registry
cp -r server/static server/templates "$STAGING_DIR/server/"
cp deploy/clipilot-registry.user.service "$STAGING_DIR/"
cp scripts/remote-deploy.sh "$STAGING_DIR/"
chmod +x "$STAGING_DIR/remote-deploy.sh"

DEPLOY_MODE=user \
  DEPLOY_STAGING_DIR="$STAGING_DIR" \
  DEPLOY_ENV_FILE="$ENV_SOURCE" \
  bash "$STAGING_DIR/remote-deploy.sh"

rm -rf "$STAGING_DIR"
