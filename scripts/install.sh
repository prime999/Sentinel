#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/sentinel}"
DATA_DIR="${DATA_DIR:-/var/lib/sentinel}"
SERVICE_USER="${SERVICE_USER:-sentinel}"
BINARY_SRC="${BINARY_SRC:-}"

if [[ $EUID -ne 0 ]]; then
  echo "Please run as root (sudo)."
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  GOARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "==> Creating service user"
if ! id "$SERVICE_USER" &>/dev/null; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

echo "==> Installing binary"
if [[ -n "$BINARY_SRC" && -f "$BINARY_SRC" ]]; then
  install -m 0755 "$BINARY_SRC" "$INSTALL_DIR/sentinel"
elif [[ -f "./bin/sentinel" ]]; then
  install -m 0755 "./bin/sentinel" "$INSTALL_DIR/sentinel"
else
  echo "Binary not found. Build first with 'make build' or set BINARY_SRC."
  exit 1
fi

echo "==> Creating directories"
mkdir -p "$CONFIG_DIR" "$DATA_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"

if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
  echo "==> Installing default config"
  if [[ -f "./config.example.yaml" ]]; then
    cp ./config.example.yaml "$CONFIG_DIR/config.yaml"
  else
    cat > "$CONFIG_DIR/config.yaml" <<EOF
server:
  listen: "0.0.0.0:8082"
  workers: 10
  retention_days: 90
  dashboard_url: "http://$(hostname -I | awk '{print $1}'):8082"
auth:
  username: admin
  password: changeme
smtp:
  host: ""
  port: 587
  username: ""
  password: ""
  from: ""
  tls: true
database:
  path: $DATA_DIR/sentinel.db
EOF
  fi
  sed -i "s|path: ./data/sentinel.db|path: $DATA_DIR/sentinel.db|" "$CONFIG_DIR/config.yaml" 2>/dev/null || \
    sed -i '' "s|path: ./data/sentinel.db|path: $DATA_DIR/sentinel.db|" "$CONFIG_DIR/config.yaml"
fi

echo "==> Installing systemd service"
install -m 0644 ./deploy/sentinel.service /etc/systemd/system/sentinel.service
systemctl daemon-reload
systemctl enable sentinel
systemctl restart sentinel

IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost')"
echo ""
echo "Sentinel installed successfully!"
echo "  URL:      http://${IP}:8082"
echo "  Config:   $CONFIG_DIR/config.yaml"
echo "  Data:     $DATA_DIR"
echo "  Login:    admin / changeme  (change password in config!)"
echo ""
echo "  systemctl status sentinel"
