#!/bin/bash
# Server-side deployment for Clio registry (user systemd, no sudo).
# Invoked by CI after files are copied to DEPLOY_STAGING_DIR.

set -euo pipefail

DEPLOY_STAGING_DIR="${DEPLOY_STAGING_DIR:-/tmp/clio-deploy}"
SERVICE_NAME="${SERVICE_NAME:-clio-registry}"
DEPLOY_MODE="${DEPLOY_MODE:-user}"

if [ "$DEPLOY_MODE" != "user" ]; then
  echo "Error: only DEPLOY_MODE=user is supported (no sudo required)."
  exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-$HOME/clio-registry}"
DATA_DIR="${DATA_DIR:-$HOME/clio-data}"
ENV_FILE="${ENV_FILE:-$INSTALL_DIR/env}"
STATIC_DIR="${STATIC_DIR:-$INSTALL_DIR/server/static}"
TEMPLATE_DIR="${TEMPLATE_DIR:-$INSTALL_DIR/server/templates}"
SERVICE_UNIT="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/${SERVICE_NAME}.service"
SYSTEMCTL=(systemctl --user)

echo "=== Clio registry deployment ==="
echo "User:        $(whoami)"
echo "Install dir: $INSTALL_DIR"
echo "Data dir:    $DATA_DIR"
echo ""

migrate_legacy_paths() {
  # One-time move from older home-dir layout, if present.
  local old_unit="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/clipilot-registry.service"
  if "${SYSTEMCTL[@]}" list-unit-files 2>/dev/null | grep -q '^clipilot-registry.service'; then
    "${SYSTEMCTL[@]}" stop clipilot-registry 2>/dev/null || true
    "${SYSTEMCTL[@]}" disable clipilot-registry 2>/dev/null || true
  fi
  rm -f "$old_unit" 2>/dev/null || true

  if [ ! -d "$INSTALL_DIR" ] && [ -d "$HOME/clipilot-registry" ]; then
    echo "Migrating $HOME/clipilot-registry → $INSTALL_DIR"
    mv "$HOME/clipilot-registry" "$INSTALL_DIR"
  fi
  if [ ! -d "$DATA_DIR" ] && [ -d "$HOME/clipilot-data" ]; then
    echo "Migrating $HOME/clipilot-data → $DATA_DIR"
    mv "$HOME/clipilot-data" "$DATA_DIR"
  fi
}

ensure_user_systemd() {
  export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

  if [ ! -d "$XDG_RUNTIME_DIR" ]; then
    echo "Error: user systemd session is not available."
    echo "Ask an admin to run once as root: loginctl enable-linger $(whoami)"
    exit 1
  fi

  if ! loginctl show-user "$(whoami)" -p Linger 2>/dev/null | grep -q 'yes'; then
    echo "Warning: systemd linger may not be enabled for $(whoami)."
  fi
}

write_env_file() {
  if [ ! -f "${DEPLOY_ENV_FILE:-}" ]; then
    echo "Error: DEPLOY_ENV_FILE is required"
    exit 1
  fi

  # shellcheck disable=SC1090
  source "$DEPLOY_ENV_FILE"

  if [ -z "${ADMIN_PASSWORD:-}" ]; then
    echo "Error: ADMIN_PASSWORD must be set in env file"
    exit 1
  fi

  mkdir -p "$(dirname "$ENV_FILE")"
  # Strip leading whitespace so systemd EnvironmentFile accepts every line.
  # Drop a leftover .env in the install dir — it can override PORT after migration.
  rm -f "$INSTALL_DIR/.env"
  {
    grep -v -E '^[[:space:]]*(DATA_DIR|STATIC_DIR|TEMPLATE_DIR|PORT)=' "$DEPLOY_ENV_FILE" | sed 's/^[[:space:]]*//' || true
    echo "DATA_DIR=$DATA_DIR"
    echo "STATIC_DIR=$STATIC_DIR"
    echo "TEMPLATE_DIR=$TEMPLATE_DIR"
  } > "$ENV_FILE"

  # Normalize PORT: strip quotes/whitespace; default 8082 (8080 is often Apache).
  port_val=$(grep -E '^[[:space:]]*PORT=' "$DEPLOY_ENV_FILE" | tail -1 | sed 's/^[[:space:]]*PORT=//' | tr -d ' \t\r"'"'" || true)
  if ! [[ "$port_val" =~ ^[0-9]+$ ]]; then
    port_val=8082
  fi
  echo "PORT=$port_val" >> "$ENV_FILE"
  echo "Listen PORT=$port_val"
  chmod 600 "$ENV_FILE"
}

install_files() {
  if [ ! -f "$DEPLOY_STAGING_DIR/registry" ]; then
    echo "Error: Binary not found in $DEPLOY_STAGING_DIR"
    exit 1
  fi

  if [ ! -f "$DEPLOY_STAGING_DIR/clio-registry.user.service" ]; then
    echo "Error: clio-registry.user.service not found in bundle"
    exit 1
  fi

  mkdir -p "$INSTALL_DIR/server/static" "$INSTALL_DIR/server/templates" "$INSTALL_DIR/modules" "$DATA_DIR/uploads"
  install -m 755 "$DEPLOY_STAGING_DIR/registry" "$INSTALL_DIR/registry"
  cp -r "$DEPLOY_STAGING_DIR/server/static/." "$INSTALL_DIR/server/static/"
  cp -r "$DEPLOY_STAGING_DIR/server/templates/." "$INSTALL_DIR/server/templates/"
  if [ -d "$DEPLOY_STAGING_DIR/modules" ]; then
    rm -rf "$INSTALL_DIR/modules"
    cp -r "$DEPLOY_STAGING_DIR/modules" "$INSTALL_DIR/modules"
  fi
  mkdir -p "$(dirname "$SERVICE_UNIT")"
  install -m 644 "$DEPLOY_STAGING_DIR/clio-registry.user.service" "$SERVICE_UNIT"
  chmod 750 "$DATA_DIR"
}

deploy_clio_install_script() {
  local port="${PORT:-8080}"
  local tmp_script="/tmp/clio-install-$$.sh"
  local cookie_jar="/tmp/clio-cookies-$$.txt"
  local script_version

  if ! curl -fsSL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh -o "$tmp_script"; then
    echo "Warning: could not download Clio install script"
    return 0
  fi

  script_version=$(grep -m1 '^VERSION=' "$tmp_script" | cut -d'=' -f2 | tr -d '"' || echo "auto")

  if curl -fsS -c "$cookie_jar" -b "$cookie_jar" \
    -X POST "http://127.0.0.1:${port}/login" \
    -d "username=${ADMIN_USER:-admin}&password=${ADMIN_PASSWORD}" \
    -o /dev/null; then
    if curl -fsS -b "$cookie_jar" \
      -X POST "http://127.0.0.1:${port}/api/install-script/upload" \
      -F "file=@${tmp_script};filename=install.sh" \
      -F "version=${script_version}" >/dev/null; then
      echo "Clio install script uploaded via API (v${script_version})"
      rm -f "$tmp_script" "$cookie_jar"
      return 0
    fi
  fi

  echo "Warning: API upload failed; server will bootstrap /clio on startup"
  rm -f "$tmp_script" "$cookie_jar"
}

verify_health() {
  local port="${PORT:-8080}"
  local attempts=0
  local max_attempts=20

  while [ "$attempts" -lt "$max_attempts" ]; do
    if curl -fsS "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
      echo "Health check passed"
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 2
  done

  echo "Health check failed"
  "${SYSTEMCTL[@]}" status "$SERVICE_NAME" --no-pager || true
  journalctl --user -u "$SERVICE_NAME" -n 50 --no-pager || true
  return 1
}

verify_clio_endpoint() {
  local port="${PORT:-8080}"
  if curl -fsS "http://127.0.0.1:${port}/clio" | head -1 | grep -q '#!/'; then
    echo "Clio install endpoint ready"
    return 0
  fi
  echo "Warning: /clio not serving install script yet"
  return 0
}

ensure_user_systemd
migrate_legacy_paths
write_env_file
install_files

"${SYSTEMCTL[@]}" daemon-reload
"${SYSTEMCTL[@]}" enable "$SERVICE_NAME"
"${SYSTEMCTL[@]}" restart "$SERVICE_NAME"

verify_health
deploy_clio_install_script
verify_clio_endpoint

echo ""
echo "Deployment complete"
echo "Service: $SERVICE_NAME (user systemd)"
echo "URL:     ${BASE_URL:-http://127.0.0.1:${PORT:-8080}}"
