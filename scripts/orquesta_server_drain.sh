#!/usr/bin/env bash
# Drain gobernado de orquesta-server para mantenimiento/deploy local.
# Inventaria, protege uso-app, hace backup y solo despues intenta parada
# cooperativa de procesos identificados por perfil.

set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DRAIN_RUNTIME_ROOT="${ORQUESTA_DRAIN_RUNTIME_ROOT:-/srv/orquesta-self/runtime}"
DRAIN_STATE_ROOT="${ORQUESTA_DRAIN_STATE_ROOT:-/srv/orquesta-self/claude-director-20260705}"
DRAIN_PROTECTED_APP_ROOT="${ORQUESTA_DRAIN_PROTECTED_APP_ROOT:-/srv/orquesta-self/uso-app}"
DRAIN_RECEIPT="${ORQUESTA_DRAIN_RECEIPT:-$DRAIN_STATE_ROOT/state/orquesta_server_drain_receipt_v0.json}"
DRAIN_BACKUP_DIR="${ORQUESTA_DRAIN_BACKUP_DIR:-$DRAIN_STATE_ROOT/backups/drain-$(date -u +%Y%m%dT%H%M%SZ)}"
DRAIN_PS_COMMAND="${ORQUESTA_DRAIN_PS_COMMAND:-ps -eo pid=,ppid=,user=,args=}"
DRAIN_KILL_COMMAND="${ORQUESTA_DRAIN_KILL_COMMAND:-kill}"
DRAIN_CURL_COMMAND="${ORQUESTA_DRAIN_CURL_COMMAND:-curl}"
DRAIN_TMUX_COMMAND="${ORQUESTA_DRAIN_TMUX_COMMAND:-tmux}"
DRAIN_TMUX_LIST_COMMAND="${ORQUESTA_DRAIN_TMUX_LIST_COMMAND:-}"
DRAIN_BASE_URL_FILE="${ORQUESTA_DRAIN_BASE_URL_FILE:-$DRAIN_STATE_ROOT/runtime/base_url.txt}"
DRAIN_CONFIRM="${ORQUESTA_DRAIN_CONFIRM:-}"
DRAIN_ACTION=0
DRAIN_DRY_RUN=1
DRAIN_WAIT_SECONDS="${ORQUESTA_DRAIN_WAIT_SECONDS:-8}"
DRAIN_REASON="${ORQUESTA_DRAIN_REASON:-orquesta_server_drain}"

usage() {
  cat >&2 <<'USAGE'
uso: scripts/orquesta_server_drain.sh [--dry-run]
     scripts/orquesta_server_drain.sh --drain --confirm-drain orquesta-server-drain

Env principal:
  ORQUESTA_DRAIN_RUNTIME_ROOT       runtime Orquesta gobernado
  ORQUESTA_DRAIN_STATE_ROOT         raiz de estado Orquesta gobernada
  ORQUESTA_DRAIN_PROTECTED_APP_ROOT raiz de uso-app, protegida siempre
  ORQUESTA_DRAIN_RECEIPT            recibo JSON durable
  ORQUESTA_DRAIN_BACKUP_DIR         destino de backup previo a parada
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRAIN_DRY_RUN=1
      shift
      ;;
    --drain)
      DRAIN_ACTION=1
      shift
      ;;
    --confirm-drain)
      if [ "$#" -lt 2 ]; then
        echo "orquesta_server_drain: falta token de confirmacion" >&2
        usage
        exit 2
      fi
      DRAIN_CONFIRM="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "orquesta_server_drain: argumento no soportado: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [ "$DRAIN_ACTION" != "1" ] || [ "$DRAIN_CONFIRM" != "orquesta-server-drain" ]; then
  DRAIN_DRY_RUN=1
else
  DRAIN_DRY_RUN=0
fi

json_escape() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1"
}

run_ps_inventory() {
  bash -c "$DRAIN_PS_COMMAND"
}

classify_inventory() {
  local out="$1"
  run_ps_inventory | python3 -c '
import json
import sys

runtime_root, state_root, protected_root = sys.argv[1:4]
processes = []
targets = []
protected = []
skipped = []

for raw in sys.stdin:
    raw = raw.rstrip("\n")
    parts = raw.split(None, 3)
    if len(parts) < 4:
        continue
    pid, ppid, user, args = parts
    proc = {"pid": pid, "ppid": ppid, "user": user, "args": args}
    is_protected = protected_root and protected_root in args
    is_orquesta_server = "orquesta-server" in args and " run" in args
    identity_match = (
        (runtime_root and runtime_root in args)
        or (state_root and state_root in args)
        or "ORQUESTA_SERVER_STATE_DIR=" in args
        or "ORQUESTA_CODEX_RUNTIME_WORKDIR=" in args
    )
    proc["protected"] = bool(is_protected)
    proc["identity_match"] = bool(identity_match)
    proc["target"] = bool(is_orquesta_server and identity_match and not is_protected)
    if proc["target"]:
        targets.append(proc)
    elif is_protected:
        proc["skip_reason"] = "protected_uso_app"
        protected.append(proc)
    elif is_orquesta_server:
        proc["skip_reason"] = "identity_not_managed"
        skipped.append(proc)
    processes.append(proc)

print(json.dumps({
    "schema_version": "orquesta_server_drain_inventory.v0",
    "processes": processes,
    "targets": targets,
    "protected": protected,
    "skipped": skipped,
}, sort_keys=True))
' \
    "$DRAIN_RUNTIME_ROOT" \
    "$DRAIN_STATE_ROOT" \
    "$DRAIN_PROTECTED_APP_ROOT" >"$out"
}

prepare_backup() {
  mkdir -p "$DRAIN_BACKUP_DIR"
  local manifest="$DRAIN_BACKUP_DIR/manifest.json"
  local copied="$DRAIN_BACKUP_DIR/copied_paths.txt"
  : >"$copied"
  for candidate in \
    "$DRAIN_STATE_ROOT/state" \
    "$DRAIN_STATE_ROOT/server.pid" \
    "$DRAIN_STATE_ROOT/logs" \
    "$DRAIN_STATE_ROOT/runtime/base_url.txt"
  do
    if [ -e "$candidate" ]; then
      cp -a "$candidate" "$DRAIN_BACKUP_DIR/" 2>/dev/null || true
      printf '%s\n' "$candidate" >>"$copied"
    fi
  done
  python3 - "$DRAIN_BACKUP_DIR" "$copied" >"$manifest" <<'PY'
import json
import sys
import time

backup_dir, copied_file = sys.argv[1:3]
with open(copied_file, encoding="utf-8") as fh:
    copied = [line.strip() for line in fh if line.strip()]
print(json.dumps({
    "schema_version": "orquesta_server_drain_backup.v0",
    "backup_dir": backup_dir,
    "copied_paths": copied,
    "created_at_unix": int(time.time()),
}, sort_keys=True))
PY
}

json_array_field() {
  local file="$1"
  local field="$2"
  python3 - "$file" "$field" <<'PY'
import json
import sys
with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
for item in data.get(sys.argv[2], []):
    print(item.get("pid", ""))
PY
}

process_alive() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null
}

wait_for_exit() {
  local pid="$1"
  local deadline="$((SECONDS + DRAIN_WAIT_SECONDS))"
  while [ "$SECONDS" -lt "$deadline" ]; do
    if ! process_alive "$pid"; then
      return 0
    fi
    sleep 1
  done
  return 1
}

stop_targets() {
  local inventory="$1"
  local actions="$2"
  : >"$actions"
  request_shutdown_http "$actions"
  while IFS= read -r pid; do
    [ -n "$pid" ] || continue
    if [ "$DRAIN_DRY_RUN" = "1" ]; then
      printf 'dry_run_skip pid=%s signal=INT\n' "$pid" >>"$actions"
      continue
    fi
    printf 'send pid=%s signal=INT\n' "$pid" >>"$actions"
    "$DRAIN_KILL_COMMAND" -INT "$pid" 2>/dev/null || true
    if wait_for_exit "$pid"; then
      printf 'exited pid=%s after=INT\n' "$pid" >>"$actions"
      continue
    fi
    printf 'send pid=%s signal=TERM\n' "$pid" >>"$actions"
    "$DRAIN_KILL_COMMAND" -TERM "$pid" 2>/dev/null || true
    if wait_for_exit "$pid"; then
      printf 'exited pid=%s after=TERM\n' "$pid" >>"$actions"
    else
      printf 'still_alive pid=%s after=TERM\n' "$pid" >>"$actions"
    fi
  done < <(json_array_field "$inventory" targets)
}

request_shutdown_http() {
  local actions="$1"
  if [ ! -s "$DRAIN_BASE_URL_FILE" ]; then
    printf 'skip action=shutdown_http reason=base_url_missing
' >>"$actions"
    return 0
  fi
  local base_url
  base_url="$(head -n 1 "$DRAIN_BASE_URL_FILE" | tr -d '[:space:]')"
  if [ -z "$base_url" ]; then
    printf 'skip action=shutdown_http reason=base_url_empty
' >>"$actions"
    return 0
  fi
  if [ "$DRAIN_DRY_RUN" = "1" ]; then
    printf 'dry_run_skip action=shutdown_http endpoint=%s/api/v0/server/shutdown
' "$base_url" >>"$actions"
    return 0
  fi
  local request_id="orquesta-drain-shutdown-$(date -u +%Y%m%dT%H%M%SZ)"
  local payload
  payload="{"request_id":"$request_id","idempotency_key":"$request_id","requested_by":"orquesta-drain","cleanup_goal_backends":true}"
  if "$DRAIN_CURL_COMMAND" -fsS -m 20 -X POST "$base_url/api/v0/server/shutdown"     -H 'Content-Type: application/json'     --data-binary "$payload" >/dev/null 2>&1; then
    printf 'ok action=shutdown_http endpoint=%s/api/v0/server/shutdown
' "$base_url" >>"$actions"
  else
    printf 'error action=shutdown_http endpoint=%s/api/v0/server/shutdown
' "$base_url" >>"$actions"
  fi
}

list_goal_tmux_sessions() {
  if [ -n "$DRAIN_TMUX_LIST_COMMAND" ]; then
    bash -c "$DRAIN_TMUX_LIST_COMMAND"
    return 0
  fi
  command -v "$DRAIN_TMUX_COMMAND" >/dev/null 2>&1 || return 0
  "$DRAIN_TMUX_COMMAND" list-sessions -F '#{session_name}' 2>/dev/null || true
}

stop_goal_tmux_sessions() {
  local actions="$1"
  while IFS= read -r session; do
    case "$session" in
      orquesta-goal-*) ;;
      *) continue ;;
    esac
    if [ "$DRAIN_DRY_RUN" = "1" ]; then
      printf 'dry_run_skip session=%s action=tmux_kill_session\n' "$session" >>"$actions"
      continue
    fi
    printf 'send session=%s action=tmux_kill_session\n' "$session" >>"$actions"
    "$DRAIN_TMUX_COMMAND" kill-session -t "$session" >/dev/null 2>&1 || true
  done < <(list_goal_tmux_sessions)
}

drain_status_from_after_inventory() {
  local after="$1"
  if [ "$DRAIN_DRY_RUN" = "1" ]; then
    printf 'refused\n'
    return 0
  fi
  python3 - "$after" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    data = json.load(fh)
print("clean" if len(data.get("targets", [])) == 0 else "residual")
PY
}

write_receipt() {
  local status="$1"
  local reason="$2"
  local before="$3"
  local after="$4"
  local actions="$5"
  local drain_status
  drain_status="$(drain_status_from_after_inventory "$after")"
  mkdir -p "$(dirname "$DRAIN_RECEIPT")"
  python3 - \
    "$status" \
    "$reason" \
    "$DRAIN_REASON" \
    "$DRAIN_DRY_RUN" \
    "$DRAIN_RUNTIME_ROOT" \
    "$DRAIN_STATE_ROOT" \
    "$DRAIN_PROTECTED_APP_ROOT" \
    "$DRAIN_BACKUP_DIR" \
    "$drain_status" \
    "$before" \
    "$after" \
    "$actions" >"$DRAIN_RECEIPT.tmp.$$" <<'PY'
import json
import sys
import time

(
    status, reason, drain_reason, dry_run, runtime_root, state_root,
    protected_root, backup_dir, drain_status, before_path, after_path, actions_path,
) = sys.argv[1:13]

def load(path):
    with open(path, encoding="utf-8") as fh:
        return json.load(fh)

with open(actions_path, encoding="utf-8") as fh:
    actions = [line.strip() for line in fh if line.strip()]

receipt = {
    "schema_version": "orquesta_server_drain_receipt.v0",
    "status": status,
    "reason_code": reason,
    "reason": drain_reason,
    "dry_run": dry_run == "1",
    "drain_status": drain_status,
    "runtime_root": runtime_root,
    "state_root": state_root,
    "protected_app_root": protected_root,
    "backup_dir": backup_dir,
    "backup_prepared_before_stop": True,
    "inventory_before": load(before_path),
    "inventory_after": load(after_path),
    "actions": actions,
    "created_at_unix": int(time.time()),
}
print(json.dumps(receipt, sort_keys=True))
PY
  mv "$DRAIN_RECEIPT.tmp.$$" "$DRAIN_RECEIPT"
}

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-drain.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

before_inventory="$tmp_dir/inventory_before.json"
after_inventory="$tmp_dir/inventory_after.json"
actions_file="$tmp_dir/actions.log"

classify_inventory "$before_inventory"
prepare_backup
stop_targets "$before_inventory" "$actions_file"
stop_goal_tmux_sessions "$actions_file"
classify_inventory "$after_inventory"
write_receipt "ok" "drain_completed" "$before_inventory" "$after_inventory" "$actions_file"

printf 'orquesta_server_drain=ok receipt=%s backup=%s\n' "$DRAIN_RECEIPT" "$DRAIN_BACKUP_DIR"
