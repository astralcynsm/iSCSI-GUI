#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
BIN_SRC="$ROOT_DIR/bin/iscsi-agent"
BIN_DST="/usr/bin/iscsi-agent"
SERVICE_SRC="$ROOT_DIR/agent/packaging/systemd/iscsi-agent.service"
SERVICE_DST="/etc/systemd/system/iscsi-agent.service"

if [[ ! -f "$BIN_SRC" ]]; then
  echo "missing binary: $BIN_SRC"
  echo "build first: make build-agent"
  exit 1
fi

echo "installing binary to $BIN_DST"
install -m 0755 "$BIN_SRC" "$BIN_DST"

echo "installing service file to $SERVICE_DST"
install -m 0644 "$SERVICE_SRC" "$SERVICE_DST"

echo "reloading systemd"
systemctl daemon-reload

echo "enabling and restarting iscsi-agent"
systemctl enable iscsi-agent
systemctl restart iscsi-agent

echo "done"
