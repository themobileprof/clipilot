#!/bin/bash
# Server-side deployment for CLIPilot Registry (user systemd, no sudo).
# Invoked by CI after files are copied to DEPLOY_STAGING_DIR.

set -euo pipefail

DEPLOY_STAGING_DIR="${DEPLOY_STAGING_DIR:-/tmp/clipilot-deploy}"
SERVICE_NAME="${SERVICE_NAME:-clipilot-registry}"
DEPLOY_MODE="${DEPLOY_MODE:-user}"

if [ "$DEPLOY_MODE" != "user" ]; then
  echo "Error: only DEPLOY_MODE=user is supported (no sudo required)."
  echo "Set DEPLOY_MODE=user or omit it."
  exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-$HOME/clipilot-registry}"
DATA_DIR="${DATA_DIR:-$HOME/clipilot-data}"
ENV_FILE="${ENV_FILE:-$INSTALL_DIR/env}"
STATIC_DIR="${STATIC_DIR:-$INSTALL_DIR/server/static}"
TEMPLATE_DIR="${TEMPLATE_DIR:-$INSTALL_DIR/server/templates}"
SERVICE_UNIT="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/${SERVICE_NAME}.service"
SYSTEMCTL=(systemctl --user)

echo "=== CLIPilot Registry Deployment ==="
echo "User:        $(whoami)"
echo "Install dir: $INSTALL_DIR"
echo "Data dir:    $DATA_DIR"
echo ""

ensure_user_systemd() {
  export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

  if [ ! -d "$XDG_RUNTIME_DIR" ]; then
    echo "Error: user systemd session is not available."
    echo "Ask an admin to run once as root:"
    echo "  loginctl enable-linger $(whoami)"
    exit 1
  fi

  if ! loginctl show-user "$(whoami)" -p Linger 2>/dev/null | grep -q 'yes'; then
    echo "Warning: systemd linger may not be enabled for $(whoami)."
    echo "If deploy fails, run as root: loginctl enable-linger $(whoami)"
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
  grep -v -E '^(DATA_DIR|STATIC_DIR|TEMPLATE_DIR)=' "$DEPLOY_ENV_FILE" > "$ENV_FILE" || true
  {
    echo "DATA_DIR=$DATA_DIR"
    echo "STATIC_DIR=$STATIC_DIR"
    echo "TEMPLATE_DIR=$TEMPLATE_DIR"
  } >> "$ENV_FILE"
  chmod 600 "$ENV_FILE"
}

install_files() {
  if [ ! -f "$DEPLOY_STAGING_DIR/registry" ]; then
    echo "Error: Binary not found in $DEPLOY_STAGING_DIR"
    exit 1
  fi

  if [ ! -f "$DEPLOY_STAGING_DIR/clipilot-registry.user.service" ]; then
    echo "Error: clipilot-registry.user.service not found in bundle"
    exit 1
  fi

  mkdir -p "$INSTALL_DIR/server/static" "$INSTALL_DIR/server/templates" "$DATA_DIR/uploads"
  install -m 755 "$DEPLOY_STAGING_DIR/registry" "$INSTALL_DIR/registry"
  cp -r "$DEPLOY_STAGING_DIR/server/static/." "$INSTALL_DIR/server/static/"
  cp -r "$DEPLOY_STAGING_DIR/server/templates/." "$INSTALL_DIR/server/templates/"
  mkdir -p "$(dirname "$SERVICE_UNIT")"
  install -m 644 "$DEPLOY_STAGING_DIR/clipilot-registry.user.service" "$SERVICE_UNIT"
  chmod 750 "$DATA_DIR"
}

deploy_clio_install_script() {
  if [ ! -f "$DATA_DIR/registry.db" ] || ! command -v sqlite3 >/dev/null 2>&1; then
    echo "Skipping Clio install script sync (database not ready or sqlite3 missing)"
    return 0
  fi

  local script_version checksum filename file_path upload_dir tmp_script
  tmp_script="/tmp/clio-install-$$.sh"

  curl -fsSL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh -o "$tmp_script" 2>/dev/null || return 0
  [ -f "$tmp_script" ] || return 0

  script_version=$(grep -m1 '^VERSION=' "$tmp_script" | cut -d'=' -f2 | tr -d '"' || echo "auto")
  checksum=$(sha256sum "$tmp_script" | awk '{print $1}')
  filename="install-${script_version}-${checksum:0:8}.sh"
  upload_dir="$DATA_DIR/uploads/install_scripts"
  file_path="$upload_dir/$filename"

  mkdir -p "$upload_dir"
  cp "$tmp_script" "$file_path"
  chmod 644 "$file_path"

  sqlite3 "$DATA_DIR/registry.db" <<EOF
UPDATE install_scripts SET is_active = 0 WHERE is_active = 1;
INSERT INTO install_scripts (version, file_path, checksum_sha256, size_bytes, uploaded_by, is_active, uploaded_at)
VALUES ('$script_version', '$file_path', '$checksum', $(wc -c < "$tmp_script"), NULL, 1, CURRENT_TIMESTAMP);
EOF

  rm -f "$tmp_script"
  echo "Clio install script deployed (v${script_version})"
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

ensure_user_systemd
write_env_file
install_files

"${SYSTEMCTL[@]}" daemon-reload
"${SYSTEMCTL[@]}" enable "$SERVICE_NAME"
"${SYSTEMCTL[@]}" restart "$SERVICE_NAME"

verify_health
deploy_clio_install_script

echo ""
echo "Deployment complete"
echo "Service: $SERVICE_NAME (user systemd)"
echo "URL:     ${BASE_URL:-http://127.0.0.1:${PORT:-8080}}"
