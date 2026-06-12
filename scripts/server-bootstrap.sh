#!/bin/bash
# One-time server bootstrap (run as root).
# Enables user-level systemd deploys without sudo for your existing SSH user.

set -euo pipefail

DEPLOY_USER="${1:-}"

if [ "$(id -u)" -ne 0 ]; then
  echo "Run as root: sudo $0 <ssh-username>"
  exit 1
fi

if [ -z "$DEPLOY_USER" ]; then
  echo "Usage: sudo $0 <ssh-username>"
  echo "Example: sudo $0 samuel"
  echo ""
  echo "Use the same username as GitHub secret SSH_USERNAME."
  exit 1
fi

if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  echo "Creating user $DEPLOY_USER..."
  adduser --disabled-password --gecos "" "$DEPLOY_USER"
fi

echo "Enabling systemd linger for $DEPLOY_USER..."
loginctl enable-linger "$DEPLOY_USER"

echo "Creating home directories..."
sudo -u "$DEPLOY_USER" mkdir -p "/home/$DEPLOY_USER/clipilot-registry/server"
sudo -u "$DEPLOY_USER" mkdir -p "/home/$DEPLOY_USER/clipilot-data"

echo ""
echo "Bootstrap complete for user: $DEPLOY_USER"
echo ""
echo "Next steps:"
echo "  1. Ensure GitHub secret SSH_USERNAME=$DEPLOY_USER"
echo "  2. Set ENV_FILE secret (ADMIN_PASSWORD, BASE_URL, OAuth keys, etc.)"
echo "  3. Push to main to deploy"
