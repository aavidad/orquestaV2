#!/usr/bin/env bash
# Deploy atomico local del binario orquesta-server.
# No toca remoto: opera sobre repo/worktree/binario recibidos por env y arranca
# siempre via scripts/orquesta_server_ctl.sh.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_REPO="${ORQUESTA_DEPLOY_REPO:-$ROOT}"
DEPLOY_WORKTREE="${ORQUESTA_DEPLOY_WORKTREE:-$DEPLOY_REPO}"
DEPLOY_REF="${ORQUESTA_DEPLOY_REF:-HEAD}"
DEPLOY_CTL="${ORQUESTA_DEPLOY_CTL:-$ROOT/scripts/orquesta_server_ctl.sh}"
DEPLOY_BINARY="${ORQUESTA_DEPLOY_BINARY:-${ORQUESTA_CTL_BINARY:-/srv/orquesta-self/runtime/orquesta-server-claude}}"
DEPLOY_STATE_DIR="${ORQUESTA_DEPLOY_STATE_DIR:-${ORQUESTA_CTL_HOME:-/srv/orquesta-self/claude-director-20260705}/state}"
DEPLOY_RECEIPT="${ORQUESTA_DEPLOY_RECEIPT:-$DEPLOY_STATE_DIR/orquesta_server_deploy_receipt_v0.json}"
DEPLOY_REQUIRE_CONFIG="${ORQUESTA_DEPLOY_REQUIRE_CONFIG:-0}"
DEPLOY_BUILD_CMD="${ORQUESTA_DEPLOY_BUILD_CMD:-}"
DEPLOY_STATUS_URL="${ORQUESTA_DEPLOY_STATUS_URL:-}"
DEPLOY_READINESS_URL="${ORQUESTA_DEPLOY_READINESS_URL:-}"
DEPLOY_SUPERVISOR_URL="${ORQUESTA_DEPLOY_SUPERVISOR_URL:-}"

tmp_root=""
target_sha=""
binary_sha=""
status_text=""

cleanup() {
  if [ -n "$tmp_root" ] && [ -d "$tmp_root" ]; then
    rm -rf "$tmp_root"
  fi
}
trap cleanup EXIT

json_string() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

write_receipt() {
  status="$1"
  reason_code="${2:-}"
  message="${3:-}"
  mkdir -p "$DEPLOY_STATE_DIR"
  tmp_receipt="$DEPLOY_RECEIPT.tmp.$$"
  python3 - "$status" "$reason_code" "$message" "$DEPLOY_REF" "$target_sha" "$binary_sha" "$DEPLOY_BINARY" "$status_text" >"$tmp_receipt" <<'PY'
import json
import os
import sys
import time

status, reason, message, ref, git_sha, binary_sha, binary, status_text = sys.argv[1:9]
receipt = {
    "schema_version": "orquesta_server_deploy_receipt.v0",
    "status": status,
    "reason_code": reason,
    "message": message,
    "deployed_ref": ref,
    "git_sha": git_sha,
    "binary_sha256": binary_sha,
    "binary_path": binary,
    "ctl_status": status_text[:4000],
    "created_at_unix": int(time.time()),
}
print(json.dumps(receipt, sort_keys=True))
PY
  mv "$tmp_receipt" "$DEPLOY_RECEIPT"
}

fail() {
  code="$1"
  shift || true
  message="$*"
  write_receipt "failed" "$code" "$message"
  echo "orquesta_server_deploy: reason_code=$code $message" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

resolve_config() {
  if [ "$DEPLOY_REQUIRE_CONFIG" != "1" ]; then
    return 0
  fi
  config="${ORQUESTA_CTL_CONFIG:-}"
  if [ -z "$config" ]; then
    config="${ORQUESTA_CTL_WORKDIR:-$DEPLOY_WORKTREE}/orquesta.config.json"
  fi
  [ -r "$config" ] || fail "deploy_config_missing" "config canonica no legible: $config"
  export ORQUESTA_CTL_CONFIG="$config"
}

sync_worktree_ff_only() {
  git -C "$DEPLOY_REPO" rev-parse --is-inside-work-tree >/dev/null
  mkdir -p "$DEPLOY_WORKTREE"
  if [ ! -d "$DEPLOY_WORKTREE/.git" ]; then
    git clone --no-checkout "$DEPLOY_REPO" "$DEPLOY_WORKTREE" >/dev/null 2>&1
  fi
  git -C "$DEPLOY_WORKTREE" fetch --quiet "$DEPLOY_REPO" "$DEPLOY_REF"
  target_sha="$(git -C "$DEPLOY_WORKTREE" rev-parse FETCH_HEAD)"
  current_sha="$(git -C "$DEPLOY_WORKTREE" rev-parse --verify HEAD 2>/dev/null || true)"
  if [ -n "$current_sha" ] && ! git -C "$DEPLOY_WORKTREE" merge-base --is-ancestor "$current_sha" "$target_sha"; then
    fail "deploy_not_fast_forward" "HEAD=$current_sha target=$target_sha"
  fi
  git -C "$DEPLOY_WORKTREE" checkout --quiet "$target_sha"
}

build_from_tree() {
  tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-deploy.XXXXXX")"
  src="$tmp_root/src"
  out="$tmp_root/orquesta-server"
  mkdir -p "$src"
  git -C "$DEPLOY_WORKTREE" archive "$target_sha" | tar -x -C "$src"
  if [ -n "$DEPLOY_BUILD_CMD" ]; then
    ORQUESTA_DEPLOY_BUILD_SRC="$src" ORQUESTA_DEPLOY_BUILD_OUT="$out" bash -c "$DEPLOY_BUILD_CMD"
  else
    (cd "$src" && go build -o "$out" ./cmd/orquesta-server)
  fi
  [ -x "$out" ] || fail "deploy_build_failed" "no se genero binario ejecutable"
  binary_sha="$(sha256_file "$out")"
}

swap_binary() {
  install_dir="$(dirname "$DEPLOY_BINARY")"
  mkdir -p "$install_dir"
  if [ -e "$DEPLOY_BINARY" ]; then
    cp -p "$DEPLOY_BINARY" "$DEPLOY_BINARY.bak.$(date +%Y%m%d%H%M%S)"
  fi
  tmp_target="$DEPLOY_BINARY.tmp.$$"
  cp "$tmp_root/orquesta-server" "$tmp_target"
  chmod +x "$tmp_target"
  mv "$tmp_target" "$DEPLOY_BINARY"
}

verify_json_bool_if_present() {
  body="$1"
  field="$2"
  expected="$3"
  python3 - "$body" "$field" "$expected" <<'PY'
import json, sys
body, field, expected = sys.argv[1:4]
try:
    data = json.loads(body)
except Exception:
    sys.exit(0)
if field in data and str(data[field]).lower() != expected:
    sys.exit(1)
PY
}

verify_runtime_identity() {
  status_text="$("$DEPLOY_CTL" status 2>&1 || true)"
  combined="$status_text"
  for url in "$DEPLOY_STATUS_URL" "$DEPLOY_READINESS_URL" "$DEPLOY_SUPERVISOR_URL"; do
    if [ -n "$url" ] && command -v curl >/dev/null 2>&1; then
      body="$(curl -fsS -m 8 "$url" 2>/dev/null || true)"
      combined="$combined
$body"
      if [ -n "$body" ]; then
        verify_json_bool_if_present "$body" "readiness" "true" || fail "deploy_readiness_failed" "$url readiness=false"
        verify_json_bool_if_present "$body" "ready" "true" || fail "deploy_readiness_failed" "$url ready=false"
        verify_json_bool_if_present "$body" "supervisor" "true" || fail "deploy_supervisor_failed" "$url supervisor=false"
        verify_json_bool_if_present "$body" "supervisor_ok" "true" || fail "deploy_supervisor_failed" "$url supervisor_ok=false"
      fi
    fi
  done
  status_text="$combined"
  observed_sha="$(python3 - "$combined" <<'PY'
import json, re, sys
text = sys.argv[1]
for raw in text.splitlines():
    raw = raw.strip()
    if not raw.startswith("{"):
        continue
    try:
        data = json.loads(raw)
    except Exception:
        continue
    for key in ("binary_sha256", "runtime_binary_sha256", "orquesta_server_sha256"):
        value = data.get(key)
        if isinstance(value, str) and value:
            print(value)
            raise SystemExit
match = re.search(r"(?:binary_sha256|runtime_binary_sha256|orquesta_server_sha256)=([0-9a-fA-F]{64})", text)
if match:
    print(match.group(1))
PY
)"
  if [ -n "$observed_sha" ] && [ "$observed_sha" != "$binary_sha" ]; then
    fail "deploy_runtime_identity_mismatch" "esperado=$binary_sha observado=$observed_sha"
  fi
}

main() {
  resolve_config
  sync_worktree_ff_only
  build_from_tree
  swap_binary
  ORQUESTA_CTL_BINARY="$DEPLOY_BINARY" "$DEPLOY_CTL" start
  verify_runtime_identity
  write_receipt "ok" "" ""
  echo "orquesta_server_deploy=ok ref=$DEPLOY_REF git_sha=$target_sha binary_sha256=$binary_sha receipt=$DEPLOY_RECEIPT"
}

main "$@"
