#!/usr/bin/env bash
# Deploy atomico local del binario orquesta-server.
# No toca remoto: opera sobre repo/worktree/binario recibidos por env y arranca
# siempre via scripts/orquesta_server_ctl.sh.

set -Eeuo pipefail

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
DEPLOY_TELEGRAM_CONFIG="${ORQUESTA_DEPLOY_TELEGRAM_CONFIG:-${ORQUESTA_CTL_CONFIG:-}}"

tmp_root=""
target_sha=""
binary_sha=""
deploy_remote_url=""
status_text=""
phase="init"
receipt_written="0"
writing_receipt="0"
notification_status="not_attempted"
notification_receipt_ref=""
notification_reason=""

cleanup() {
  if [ -n "$tmp_root" ] && [ -d "$tmp_root" ]; then
    rm -rf "$tmp_root"
  fi
}
trap cleanup EXIT

on_error() {
  exit_code="$?"
  if [ "${receipt_written:-0}" != "1" ] && [ "${writing_receipt:-0}" != "1" ]; then
    capture_deploy_notification "failed" "deploy_unhandled_failure" "phase=$phase exit_code=$exit_code"
    write_receipt "failed" "deploy_unhandled_failure" "phase=$phase exit_code=$exit_code"
  fi
  exit "$exit_code"
}
trap on_error ERR

json_string() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

write_receipt() {
  status="$1"
  reason_code="${2:-}"
  message="${3:-}"
  writing_receipt="1"
  mkdir -p "$DEPLOY_STATE_DIR"
  tmp_receipt="$DEPLOY_RECEIPT.tmp.$$"
  python3 - "$status" "$reason_code" "$message" "$DEPLOY_REF" "$target_sha" "$binary_sha" "$DEPLOY_BINARY" "$deploy_remote_url" "$status_text" "$notification_status" "$notification_receipt_ref" "$notification_reason" >"$tmp_receipt" <<'PY'
import json
import os
import sys
import time

(
    status,
    reason,
    message,
    ref,
    git_sha,
    binary_sha,
    binary,
    remote_url,
    status_text,
    notification_status,
    notification_receipt_ref,
    notification_reason,
) = sys.argv[1:13]
receipt = {
    "schema_version": "orquesta_server_deploy_receipt.v0",
    "status": status,
    "reason_code": reason,
    "message": message,
    "deployed_ref": ref,
    "git_sha": git_sha,
    "binary_sha256": binary_sha,
    "binary_path": binary,
    "remote_url": remote_url,
    "ctl_status": status_text[:4000],
    "notification": {
        "status": notification_status,
        "receipt_ref": notification_receipt_ref,
        "reason": notification_reason,
    },
    "created_at_unix": int(time.time()),
}
print(json.dumps(receipt, sort_keys=True))
PY
  mv "$tmp_receipt" "$DEPLOY_RECEIPT"
  receipt_written="1"
  writing_receipt="0"
}

fail() {
  code="$1"
  shift || true
  message="$*"
  capture_deploy_notification "failed" "$code" "$message"
  write_receipt "failed" "$code" "$message"
  echo "orquesta_server_deploy: reason_code=$code $message" >&2
  exit 1
}

resolve_telegram_config() {
  if [ -n "$DEPLOY_TELEGRAM_CONFIG" ]; then
    return 0
  fi
  if [ -n "${ORQUESTA_CTL_WORKDIR:-}" ] && [ -f "${ORQUESTA_CTL_WORKDIR:-}/orquesta.config.json" ]; then
    DEPLOY_TELEGRAM_CONFIG="${ORQUESTA_CTL_WORKDIR:-}/orquesta.config.json"
    return 0
  fi
  if [ -f "$DEPLOY_WORKTREE/orquesta.config.json" ]; then
    DEPLOY_TELEGRAM_CONFIG="$DEPLOY_WORKTREE/orquesta.config.json"
  fi
}

send_deploy_notification() {
  local status="$1"
  local reason="$2"
  local message="$3"
  resolve_telegram_config
  python3 - \
    "$DEPLOY_TELEGRAM_CONFIG" \
    "$status" \
    "$reason" \
    "$message" \
    "$phase" \
    "$DEPLOY_REF" \
    "$target_sha" \
    "$binary_sha" \
    "$DEPLOY_RECEIPT" <<'PY'
import hashlib
import json
import os
import sys
import urllib.error
import urllib.request

config_path, status, reason, message, phase, deploy_ref, git_sha, binary_sha, receipt_path = sys.argv[1:10]

def line(status_value, reason_value="", receipt=""):
    print("deploy_notification_status=" + status_value)
    if reason_value:
        print("deploy_notification_reason=" + reason_value)
    if receipt:
        print("deploy_notification_receipt_ref=" + receipt)

if not config_path:
    line("disabled", "config_missing")
    raise SystemExit(0)

try:
    with open(config_path, encoding="utf-8") as fh:
        config = json.load(fh)
except FileNotFoundError:
    line("disabled", "config_missing")
    raise SystemExit(0)
except (OSError, json.JSONDecodeError):
    line("blocked", "config_invalid")
    raise SystemExit(0)

if str(config.get("schema_version") or "").strip() != "orquesta_config.v0":
    line("blocked", "config_schema_unsupported")
    raise SystemExit(0)

telegram = config.get("telegram_operator") or {}
if not telegram.get("enabled", False):
    line("disabled", "telegram_operator_disabled")
    raise SystemExit(0)

token = str(telegram.get("token") or "").strip()
target = str(telegram.get("notification_target_ref") or "").strip()
if not target:
    chats = telegram.get("authorized_chat_refs") or []
    if isinstance(chats, list) and chats:
        target = str(chats[0] or "").strip()

missing = []
if not token:
    missing.append("token")
if not target.startswith("telegram:") or not target.removeprefix("telegram:").strip():
    missing.append("notification_target_ref")
if missing:
    line("blocked", "telegram_config_incomplete:" + ",".join(missing))
    raise SystemExit(0)

short_sha = git_sha[:12] if git_sha else "unknown"
short_binary = binary_sha[:12] if binary_sha else "unknown"
text = (
    "Orquesta deploy | estado=" + status +
    " | fase=" + (phase or "unknown") +
    " | ref=" + (deploy_ref or "HEAD") +
    " | git=" + short_sha +
    " | binario=" + short_binary +
    " | receipt=" + receipt_path
)
if reason:
    text += " | motivo=" + reason
if message:
    text += " | detalle=" + message[:700]

payload = {
    "chat_id": target.removeprefix("telegram:").strip(),
    "text": text[:3500],
    "disable_web_page_preview": True,
}
base_url = os.environ.get("ORQUESTA_DEPLOY_TELEGRAM_BOT_API_BASE_URL", "https://api.telegram.org").rstrip("/")
request = urllib.request.Request(
    base_url + "/bot" + token + "/sendMessage",
    data=json.dumps(payload).encode("utf-8"),
    headers={"Content-Type": "application/json"},
    method="POST",
)
try:
    with urllib.request.urlopen(request, timeout=8) as response:
        if response.status < 200 or response.status >= 300:
            line("failed", "telegram_bot_api_send_rejected")
            raise SystemExit(0)
except urllib.error.HTTPError:
    line("failed", "telegram_bot_api_send_rejected")
    raise SystemExit(0)
except Exception:
    line("failed", "telegram_bot_api_send_failed")
    raise SystemExit(0)

dedupe = "|".join([status, deploy_ref, git_sha, binary_sha, reason])
receipt = "evidence-ref-deploy-telegram-send-" + hashlib.sha1(dedupe.encode("utf-8")).hexdigest()[:12]
line("sent", receipt=receipt)
PY
}

capture_deploy_notification() {
  local output line
  output="$(send_deploy_notification "$1" "$2" "$3" 2>&1 || true)"
  printf '%s\n' "$output"
  while IFS= read -r line; do
    case "$line" in
      deploy_notification_status=*)
        notification_status="${line#deploy_notification_status=}"
        ;;
      deploy_notification_reason=*)
        notification_reason="${line#deploy_notification_reason=}"
        ;;
      deploy_notification_receipt_ref=*)
        notification_receipt_ref="${line#deploy_notification_receipt_ref=}"
        ;;
    esac
  done <<EOF
$output
EOF
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
  deploy_remote_url="$(git -C "$DEPLOY_WORKTREE" remote get-url origin 2>/dev/null || git -C "$DEPLOY_REPO" remote get-url origin 2>/dev/null || true)"
  current_sha="$(git -C "$DEPLOY_WORKTREE" rev-parse --verify HEAD 2>/dev/null || true)"
  if [ -n "$current_sha" ] && ! git -C "$DEPLOY_WORKTREE" merge-base --is-ancestor "$current_sha" "$target_sha"; then
    fail "deploy_not_fast_forward" "HEAD=$current_sha target=$target_sha"
  fi
  if [ "$DEPLOY_REF" != "HEAD" ] && git check-ref-format --branch "$DEPLOY_REF" >/dev/null 2>&1; then
    git -C "$DEPLOY_WORKTREE" checkout --quiet -B "$DEPLOY_REF" "$target_sha"
  else
    git -C "$DEPLOY_WORKTREE" checkout --quiet "$target_sha"
  fi
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

stop_existing_server() {
  status_text="$(ORQUESTA_CTL_BINARY="$DEPLOY_BINARY" "$DEPLOY_CTL" stop 2>&1)" || fail "deploy_stop_failed" "ctl stop fallo: $status_text"
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
  status_text="$("$DEPLOY_CTL" status 2>&1)" || fail "deploy_status_failed" "ctl status fallo"
  combined="$status_text"
  for url in "$DEPLOY_STATUS_URL" "$DEPLOY_READINESS_URL" "$DEPLOY_SUPERVISOR_URL"; do
    if [ -n "$url" ] && command -v curl >/dev/null 2>&1; then
      body="$(curl -fsS -m 8 "$url" 2>/dev/null)" || fail "deploy_readiness_unreachable" "$url no responde"
      [ -n "$body" ] || fail "deploy_readiness_unreachable" "$url respuesta vacia"
      combined="$combined
$body"
      if [ -n "$body" ]; then
        verify_json_bool_if_present "$body" "readiness" "true" || fail "deploy_readiness_failed" "$url readiness=false"
        verify_json_bool_if_present "$body" "ready" "true" || fail "deploy_readiness_failed" "$url ready=false"
        verify_json_bool_if_present "$body" "supervisor" "true" || fail "deploy_supervisor_failed" "$url supervisor=false"
        verify_json_bool_if_present "$body" "supervisor_ok" "true" || fail "deploy_supervisor_failed" "$url supervisor_ok=false"
      fi
    elif [ -n "$url" ]; then
      fail "deploy_curl_missing" "curl requerido para verificar $url"
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
    runtime_identity = data.get("runtime_identity")
    if isinstance(runtime_identity, dict):
        value = runtime_identity.get("binary_sha256")
        if isinstance(value, str) and value:
            print(value)
            raise SystemExit
match = re.search(r"(?:binary_sha256|runtime_binary_sha256|orquesta_server_sha256)=([0-9a-fA-F]{64})", text)
if match:
    print(match.group(1))
PY
)"
  if [ -z "$observed_sha" ]; then
    fail "deploy_runtime_identity_missing" "status/readiness no exponen sha256 del binario"
  fi
  if [ -n "$observed_sha" ] && [ "$observed_sha" != "$binary_sha" ]; then
    fail "deploy_runtime_identity_mismatch" "esperado=$binary_sha observado=$observed_sha"
  fi
}

main() {
  phase="resolve_config"
  resolve_config
  phase="sync_worktree"
  sync_worktree_ff_only
  phase="build"
  build_from_tree
  phase="stop"
  stop_existing_server
  phase="swap_binary"
  swap_binary
  phase="start"
  ORQUESTA_CTL_BINARY="$DEPLOY_BINARY" "$DEPLOY_CTL" start
  phase="verify_runtime_identity"
  verify_runtime_identity
  phase="notify"
  capture_deploy_notification "ok" "" "deploy completado"
  phase="write_receipt_ok"
  write_receipt "ok" "" ""
  phase="done"
  echo "orquesta_server_deploy=ok ref=$DEPLOY_REF git_sha=$target_sha binary_sha256=$binary_sha receipt=$DEPLOY_RECEIPT"
}

main "$@"
