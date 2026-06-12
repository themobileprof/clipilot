#!/bin/bash
# Server-side deployment script for CLIPilot Registry (native binary + systemd).
# Invoked by CI after files are copied to DEPLOY_STAGING_DIR.

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/clipilot-registry}"
DATA_DIR="${DATA_DIR:-/var/lib/clipilot-registry}"
SERVICE_NAME="${SERVICE_NAME:-clipilot-registry}"
ENV_FILE="${ENV_FILE:-/etc/clipilot-registry/env}"
DEPLOY_STAGING_DIR="${DEPLOY_STAGING_DIR:-/tmp/clipilot-deploy}"
SERVICE_USER="${SERVICE_USER:-clipilot}"

echo "=== CLIPilot Registry Deployment ==="
echo "Install dir: $INSTALL_DIR"
echo "Data dir:    $DATA_DIR"
echo ""

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: Environment file not found at $ENV_FILE"
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

if [ -z "${ADMIN_PASSWORD:-}" ]; then
  echo "Error: ADMIN_PASSWORD must be set in $ENV_FILE"
  exit 1
fi

PORT="${PORT:-8080}"

stop_legacy_docker() {
  if command -v docker >/dev/null 2>&1; then
    docker stop clipilot-registry 2>/dev/null || true
    docker rm clipilot-registry 2>/dev/null || true
  fi
}

migrate_docker_volume() {
  if [ -f "$DATA_DIR/registry.db" ]; then
    return
  fi

  if ! command -v docker >/dev/null 2>&1; then
    return
  fi

  local volume_path
  volume_path=$(docker volume inspect clipilot-registry-data --format '{{.Mountpoint}}' 2>/dev/null || true)
  if [ -n "$volume_path" ] && [ -f "$volume_path/registry.db" ]; then
    echo "Migrating data from Docker volume..."
    mkdir -p "$DATA_DIR"
    cp -a "$volume_path/." "$DATA_DIR/"
  fi
}

fix_legacy_docker_paths() {
  if [ ! -f "$DATA_DIR/registry.db" ] || ! command -v sqlite3 >/dev/null 2>&1; then
    return
  fi

  sqlite3 "$DATA_DIR/registry.db" <<EOF
UPDATE modules SET file_path = REPLACE(file_path, '/app/data', '$DATA_DIR') WHERE file_path LIKE '/app/data/%';
UPDATE install_scripts SET file_path = REPLACE(file_path, '/app/data', '$DATA_DIR') WHERE file_path LIKE '/app/data/%';
EOF
}

ensure_service_user() {
  if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --home-dir "$INSTALL_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
  fi
}

install_files() {
  if [ ! -f "$DEPLOY_STAGING_DIR/registry" ]; then
    echo "Error: Binary not found in $DEPLOY_STAGING_DIR"
    exit 1
  fi

  mkdir -p "$INSTALL_DIR/server/static" "$INSTALL_DIR/server/templates" "$DATA_DIR/uploads"
  install -m 755 "$DEPLOY_STAGING_DIR/registry" "$INSTALL_DIR/registry"
  cp -r "$DEPLOY_STAGING_DIR/server/static/." "$INSTALL_DIR/server/static/"
  cp -r "$DEPLOY_STAGING_DIR/server/templates/." "$INSTALL_DIR/server/templates/"
  install -m 644 "$DEPLOY_STAGING_DIR/clipilot-registry.service" "/etc/systemd/system/${SERVICE_NAME}.service"
}

configure_permissions() {
  chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR" "$DATA_DIR"
  chmod 750 "$DATA_DIR"
  chmod 640 "$ENV_FILE"
  chown root:"$SERVICE_USER" "$ENV_FILE"
}

deploy_clio_install_script() {
  local script_version checksum filename file_path upload_dir
  local tmp_script="/tmp/clio-install.sh"

  curl -fsSL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh -o "$tmp_script" 2>/dev/null || return 0
  [ -f "$tmp_script" ] || return 0

  if ! command -v sqlite3 >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y sqlite3 >/dev/null 2>&1 || return 0
  fi

  script_version=$(grep -m1 '^VERSION=' "$tmp_script" | cut -d'=' -f2 | tr -d '"' || echo "auto")
  checksum=$(sha256sum "$tmp_script" | awk '{print $1}')
  filename="install-${script_version}-${checksum:0:8}.sh"
  upload_dir="$DATA_DIR/uploads/install_scripts"
  file_path="$upload_dir/$filename"

  mkdir -p "$upload_dir"
  cp "$tmp_script" "$file_path"
  chmod 644 "$file_path"
  chown "$SERVICE_USER:$SERVICE_USER" "$file_path"

  sqlite3 "$DATA_DIR/registry.db" <<EOF
UPDATE install_scripts SET is_active = 0 WHERE is_active = 1;
INSERT INTO install_scripts (version, file_path, checksum_sha256, size_bytes, uploaded_by, is_active, uploaded_at)
VALUES ('$script_version', '$file_path', '$checksum', $(wc -c < "$file_path"), NULL, 1, CURRENT_TIMESTAMP);
EOF

  rm -f "$tmp_script"
  echo "Clio install script deployed (v${script_version})"
}

verify_health() {
  local attempts=0
  local max_attempts=15

  while [ "$attempts" -lt "$max_attempts" ]; do
    if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
      echo "Health check passed"
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 2
  done

  echo "Health check failed"
  journalctl -u "$SERVICE_NAME" -n 50 --no-pager || true
  return 1
}

stop_legacy_docker
migrate_docker_volume
fix_legacy_docker_paths
ensure_service_user
install_files
configure_permissions

mkdir -p "$(dirname "$ENV_FILE")"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

deploy_clio_install_script
verify_health

echo ""
echo "Deployment complete"
echo "Service: $SERVICE_NAME"
echo "URL:     ${BASE_URL:-http://127.0.0.1:${PORT}}"
