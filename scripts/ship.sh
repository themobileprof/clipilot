#!/bin/bash
# Deploy the registry from this machine. GitHub Actions cannot start jobs
# while the account is locked for billing, so this is the production path.
# The server keeps ~/clio-registry/env. Nothing in this script needs Actions secrets.

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOST="${DEPLOY_HOST:-samuel@157.230.148.144}"
STAGING="$(mktemp -d)"
BUNDLE="$(mktemp --suffix=.tar.gz)"

cleanup() {
  rm -rf "$STAGING" "$BUNDLE"
}
trap cleanup EXIT

echo "Building registry for linux/amd64..."
cd "$REPO_DIR"
mkdir -p "$STAGING/server"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$STAGING/registry" ./cmd/registry
cp -r server/static server/templates "$STAGING/server/"
cp -r modules "$STAGING/modules"
cp deploy/clio-registry.user.service "$STAGING/"
cp scripts/remote-deploy.sh "$STAGING/"
chmod +x "$STAGING/remote-deploy.sh"
tar czf "$BUNDLE" -C "$STAGING" .

echo "Copying bundle to $HOST..."
scp "$BUNDLE" "$HOST:/tmp/clio-deploy.tar.gz"

echo "Installing on the server..."
ssh "$HOST" 'bash -s' <<'EOF'
set -euo pipefail
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
rm -rf /tmp/clio-deploy
mkdir -p /tmp/clio-deploy
tar xzf /tmp/clio-deploy.tar.gz -C /tmp/clio-deploy
DEPLOY_MODE=user \
  DEPLOY_STAGING_DIR=/tmp/clio-deploy \
  bash /tmp/clio-deploy/remote-deploy.sh
rm -rf /tmp/clio-deploy /tmp/clio-deploy.tar.gz
EOF

echo "Deployed."
