#!/usr/bin/env bash
set -euo pipefail

SERVICE_DST="/etc/systemd/system/iscsi-agent.service"
BIN_DST="/usr/bin/iscsi-agent"

echo "stopping and disabling iscsi-agent"
systemctl disable --now iscsi-agent || true

echo "removing service and binary"
rm -f "$SERVICE_DST"
rm -f "$BIN_DST"

echo "reloading systemd"
systemctl daemon-reload

echo "done"
