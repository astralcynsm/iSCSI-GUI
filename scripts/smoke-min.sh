#!/usr/bin/env bash
set -euo pipefail

# Minimal smoke checks for iSCSIGUI on ZimaOS target host.
# Non-intrusive by default: only creates/deletes a temporary target.

SUDO_BIN="${SUDO_BIN:-sudo}"
SOCKET_PATH="${SOCKET_PATH:-/run/iscsi-agent/agent.sock}"
GATEWAY_BASE="${GATEWAY_BASE:-http://127.0.0.1:18081}"
TARGET_IQN="${1:-iqn.$(date +%Y-%m).local.iscsi:smoke-$(date +%s)}"

HAS_JQ=0
if command -v jq >/dev/null 2>&1; then
  HAS_JQ=1
fi

PASS_COUNT=0
FAIL_COUNT=0

log() { printf '[INFO] %s\n' "$*"; }
pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf '[PASS] %s\n' "$*"
}
fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf '[FAIL] %s\n' "$*"
}

run_check() {
  local name="$1"
  shift
  if "$@"; then
    pass "$name"
    return 0
  fi
  fail "$name"
  return 1
}

json_field() {
  local json="$1"
  local field="$2"
  if [[ "$HAS_JQ" -eq 1 ]]; then
    printf '%s' "$json" | jq -r ".$field // empty"
  else
    printf '%s' "$json" | sed -nE "s/.*\"$field\"[[:space:]]*:[[:space:]]*\"?([^\",}]*)\"?.*/\1/p" | head -n1
  fi
}

http_unix() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  if [[ -n "$body" ]]; then
    $SUDO_BIN curl -sS --fail --unix-socket "$SOCKET_PATH" -X "$method" \
      -H 'Content-Type: application/json' \
      -d "$body" "http://localhost$path"
  else
    $SUDO_BIN curl -sS --fail --unix-socket "$SOCKET_PATH" -X "$method" "http://localhost$path"
  fi
}

http_tcp() {
  local path="$1"
  curl -sS --fail "$GATEWAY_BASE$path"
}

cleanup() {
  # best-effort cleanup for temp target
  $SUDO_BIN curl -sS --unix-socket "$SOCKET_PATH" -X DELETE \
    "http://localhost/api/v1/targets?iqn=$TARGET_IQN" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "smoke target iqn: $TARGET_IQN"

run_check "service iscsi-agent active" $SUDO_BIN systemctl is-active --quiet iscsi-agent
run_check "service iscsi-web-gateway active" $SUDO_BIN systemctl is-active --quiet iscsi-web-gateway

HEALTH_JSON="$(http_unix GET /health)"
[[ "$(json_field "$HEALTH_JSON" status)" == "ok" ]]
pass "agent /health status=ok"

SYS_HEALTH_JSON="$(http_unix GET /api/v1/system/health)"
SYS_STATUS="$(json_field "$SYS_HEALTH_JSON" status)"
if [[ "$SYS_STATUS" == "ok" || "$SYS_STATUS" == "degraded" ]]; then
  pass "agent /api/v1/system/health status=$SYS_STATUS"
else
  fail "agent /api/v1/system/health unexpected status=$SYS_STATUS"
fi

GW_HEALTH_JSON="$(http_tcp /health)"
[[ "$(json_field "$GW_HEALTH_JSON" status)" == "ok" ]]
pass "gateway /health status=ok"

GW_SYS_JSON="$(http_tcp /api/v1/system/health)"
GW_SYS_STATUS="$(json_field "$GW_SYS_JSON" status)"
if [[ "$GW_SYS_STATUS" == "ok" || "$GW_SYS_STATUS" == "degraded" ]]; then
  pass "gateway /api/v1/system/health status=$GW_SYS_STATUS"
else
  fail "gateway /api/v1/system/health unexpected status=$GW_SYS_STATUS"
fi

CREATE_TARGET_JSON="$(http_unix POST /api/v1/targets "{\"iqn\":\"$TARGET_IQN\"}")"
if [[ "$(json_field "$CREATE_TARGET_JSON" changed)" == "true" ]]; then
  pass "create target changed=true"
else
  fail "create target expected changed=true"
fi

CREATE_TARGET_AGAIN_JSON="$(http_unix POST /api/v1/targets "{\"iqn\":\"$TARGET_IQN\"}")"
if [[ "$(json_field "$CREATE_TARGET_AGAIN_JSON" changed)" == "false" ]]; then
  pass "create target again idempotent changed=false"
else
  fail "create target again expected changed=false"
fi

CHAP_GET_JSON="$(http_unix GET "/api/v1/auth/chap?target_iqn=$TARGET_IQN")"
CHAP_STATUS="$(json_field "$CHAP_GET_JSON" status)"
if [[ "$CHAP_STATUS" == "ok" ]]; then
  pass "chap get status=ok"
else
  fail "chap get expected status=ok"
fi

AUDIT_JSON="$(http_unix GET "/api/v1/audit/logs?limit=10&target_iqn=$TARGET_IQN")"
AUDIT_STATUS="$(json_field "$AUDIT_JSON" status)"
if [[ "$AUDIT_STATUS" == "ok" ]]; then
  pass "audit logs status=ok"
else
  fail "audit logs expected status=ok"
fi

DELETE_TARGET_JSON="$(http_unix DELETE "/api/v1/targets?iqn=$TARGET_IQN")"
if [[ "$(json_field "$DELETE_TARGET_JSON" changed)" == "true" ]]; then
  pass "delete target changed=true"
else
  fail "delete target expected changed=true"
fi

DELETE_TARGET_AGAIN_JSON="$(http_unix DELETE "/api/v1/targets?iqn=$TARGET_IQN")"
if [[ "$(json_field "$DELETE_TARGET_AGAIN_JSON" changed)" == "false" ]]; then
  pass "delete target again idempotent changed=false"
else
  fail "delete target again expected changed=false"
fi

echo
echo "Summary: PASS=$PASS_COUNT FAIL=$FAIL_COUNT"
if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi

