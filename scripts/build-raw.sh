#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
RAW_DIR="$ROOT_DIR/packaging/raw"
OUT_FILE="$ROOT_DIR/iscsi_gui.raw"
AGENT_BIN="$ROOT_DIR/bin/iscsi-agent"
AGENT_DST="$RAW_DIR/usr/bin/iscsi-agent"
GATEWAY_BIN="$ROOT_DIR/bin/iscsi-web-gateway"
GATEWAY_DST="$RAW_DIR/usr/bin/iscsi-web-gateway"
AGENT_SERVICE_SRC="$ROOT_DIR/agent/packaging/systemd/iscsi-agent.service"
AGENT_SERVICE_DST="$RAW_DIR/usr/lib/systemd/system/iscsi-agent.service"
GATEWAY_SERVICE_SRC="$ROOT_DIR/web/gateway/packaging/systemd/iscsi-web-gateway.service"
GATEWAY_SERVICE_DST="$RAW_DIR/usr/lib/systemd/system/iscsi-web-gateway.service"

if ! command -v mksquashfs >/dev/null 2>&1; then
  echo "missing dependency: mksquashfs"
  exit 1
fi

if [[ ! -f "$AGENT_BIN" ]]; then
  echo "missing binary: $AGENT_BIN"
  echo "run: make build-agent"
  exit 1
fi

if [[ ! -f "$GATEWAY_BIN" ]]; then
  echo "missing binary: $GATEWAY_BIN"
  echo "run: make build-gateway"
  exit 1
fi

if [[ ! -f "$AGENT_SERVICE_SRC" ]]; then
  echo "missing service file: $AGENT_SERVICE_SRC"
  exit 1
fi

if [[ ! -f "$GATEWAY_SERVICE_SRC" ]]; then
  echo "missing service file: $GATEWAY_SERVICE_SRC"
  exit 1
fi

echo "staging agent binary"
install -m 0755 "$AGENT_BIN" "$AGENT_DST"

echo "staging gateway binary"
install -m 0755 "$GATEWAY_BIN" "$GATEWAY_DST"

echo "staging systemd units"
install -m 0644 "$AGENT_SERVICE_SRC" "$AGENT_SERVICE_DST"
install -m 0644 "$GATEWAY_SERVICE_SRC" "$GATEWAY_SERVICE_DST"

echo "building raw package: $OUT_FILE"
rm -f "$OUT_FILE"
mksquashfs "$RAW_DIR" "$OUT_FILE" -noappend >/dev/null

echo "done: $OUT_FILE"
ls -lh "$OUT_FILE"
