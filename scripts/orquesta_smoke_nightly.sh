#!/usr/bin/env bash
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -z "${ORQUESTA_NIGHTLY_RESULTS_DIR:-}" && -z "${HOME:-}" ]]; then
  echo "ORQUESTA_NIGHTLY_RESULTS_DIR o HOME son necesarios para escribir resultado JSON" >&2
  exit 2
fi

results_dir="${ORQUESTA_NIGHTLY_RESULTS_DIR:-$HOME/.orquesta-nightly}"
logs_dir="$results_dir/logs"
date_id="${ORQUESTA_NIGHTLY_DATE_ID:-$(date -u +%Y%m%d)}"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_epoch="$(date -u +%s)"
run_id="nightly-$date_id-$(date -u +%H%M%S)"
result_file="$results_dir/resultado_${date_id}.json"
log_file="$logs_dir/${run_id}.log"
retention_days="${ORQUESTA_NIGHTLY_RETENTION_DAYS:-30}"
smoke_script="${ORQUESTA_NIGHTLY_SMOKE_SCRIPT:-$repo_root/scripts/smoke_goal_first_app_server_real.sh}"
code_audit_enabled="${ORQUESTA_NIGHTLY_CODE_AUDIT:-1}"
code_audit_script="${ORQUESTA_NIGHTLY_CODE_AUDIT_SCRIPT:-$repo_root/scripts/orquesta_auditoria_codigo.sh}"
code_audit_dir="$results_dir/code_audit"
code_audit_json=""
code_audit_sqlite=""
mode="preflight"
phase="init"
finalized="0"

if [[ "${ORQUESTA_NIGHTLY_REAL_CONFIRM:-0}" == "1" ]]; then
  mode="real"
fi

mkdir -p "$results_dir" "$logs_dir" || {
  echo "no se pudo crear directorio nightly: $results_dir" >&2
  exit 2
}

if [[ "$retention_days" =~ ^[0-9]+$ ]]; then
  find "$results_dir" -maxdepth 1 -type f -name 'resultado_*.json' -mtime +"$retention_days" -delete 2>/dev/null || true
  find "$logs_dir" -maxdepth 1 -type f -name 'nightly-*.log' -mtime +"$retention_days" -delete 2>/dev/null || true
fi

json_result_status() {
  local code="$1"
  if [[ "$code" == "0" ]]; then
    printf '%s' "ok"
  elif [[ "$code" == "2" ]]; then
    printf '%s' "blocked"
  else
    printf '%s' "failed"
  fi
}

write_result_json() {
  local exit_code="$1"
  local fallback_phase="$2"
  local status
  status="$(json_result_status "$exit_code")"
  local finished_at finished_epoch duration_seconds
  finished_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  finished_epoch="$(date -u +%s)"
  duration_seconds="$((finished_epoch - started_epoch))"

  CODE_AUDIT_ENABLED="$code_audit_enabled" \
  CODE_AUDIT_JSON_FILE="$code_audit_json" \
  python3 - \
    "$result_file" \
    "$log_file" \
    "$date_id" \
    "$run_id" \
    "$mode" \
    "$status" \
    "$exit_code" \
    "$fallback_phase" \
    "$started_at" \
    "$finished_at" \
    "$duration_seconds" \
    "$smoke_script" <<'PY'
import json
import os
import re
import sys

(
    result_file,
    log_file,
    date_id,
    run_id,
    mode,
    status,
    exit_code,
    fallback_phase,
    started_at,
    finished_at,
    duration_seconds,
    smoke_script,
) = sys.argv[1:13]

phases = []
refs = {}

def add_phase(name):
    if name and name not in phases:
        phases.append(name)

line_patterns = (
    (re.compile(r"^run_ref=(.+)$"), "run_ref"),
    (re.compile(r"^goal_ref=(.+)$"), "goal_ref"),
    (re.compile(r"^external_goal_ref=(.+)$"), "external_goal_ref"),
    (re.compile(r"^smoke_root=(.+)$"), "smoke_root"),
    (re.compile(r"^artifact_refs=(.+)$"), "artifact_refs"),
    (re.compile(r"^evidence_refs=(.+)$"), "evidence_refs"),
)

try:
    with open(log_file, encoding="utf-8", errors="replace") as fh:
        lines = fh.readlines()
except FileNotFoundError:
    lines = []

for raw in lines:
    line = raw.strip()
    if not line:
        continue
    if "smoke_goal_first_app_server_preflight=ok" in line:
        add_phase("preflight_ok")
    if line.startswith("code_audit_json="):
        add_phase("code_audit_reported")
    if line.startswith("code_audit_ratchet=ok"):
        add_phase("code_audit_ratchet_ok")
    if line.startswith("code_audit_ratchet=failed"):
        add_phase("code_audit_ratchet_failed")
    if "smoke_goal_first_app_server_preflight=blocked" in line:
        add_phase("preflight_blocked")
    if "Codex app-server preparado" in line or "daemon Codex app-server accesible" in line:
        add_phase("codex_app_server_ready")
    if "compilando servidor temporal" in line:
        add_phase("server_build")
    if "arrancando orquesta-server" in line:
        add_phase("server_start")
    if line.startswith("servidor listo:"):
        add_phase("server_ready")
    if "POST /api/v0/apps/director -> HTTP" in line:
        add_phase("director_start_posted")
    if line.startswith("run_ref="):
        add_phase("run_ref_observed")
    if line.startswith("goal_ref="):
        add_phase("goal_ref_observed")
    if line.startswith("external_goal_ref="):
        add_phase("external_goal_ref_observed")
    if line.startswith("poll="):
        add_phase("goal_observe_poll")
    if "closure_status=accepted" in line and "closure_accepted=true" in line:
        add_phase("closure_accepted")
    if "smoke_goal_first_app_server_real=ok" in line:
        add_phase("real_ok")
    if "app_server_tmux_shutdown_ready=true" in line:
        add_phase("shutdown_verified")
    for pattern, key in line_patterns:
        match = pattern.match(line)
        if match:
            refs[key] = match.group(1).strip()

if not phases:
    add_phase(fallback_phase or "init")

payload = {
    "schema_version": "orquesta_smoke_nightly.v0",
    "result_ref": f"orquesta-nightly-{date_id}",
    "run_id": run_id,
    "date_utc": date_id,
    "started_at_utc": started_at,
    "finished_at_utc": finished_at,
    "duration_seconds": int(duration_seconds),
    "mode": mode,
    "real_confirmed": os.environ.get("ORQUESTA_NIGHTLY_REAL_CONFIRM", "0") == "1",
    "status": status,
    "exit_code": int(exit_code),
    "phase_reached": phases[-1],
    "phases": phases,
    "refs": refs,
    "command": os.path.relpath(smoke_script, os.getcwd()) if os.path.isabs(smoke_script) else smoke_script,
    "log_file": log_file,
    "notes": [
        "default mode is preflight and must not consume provider quota",
        "real mode requires ORQUESTA_NIGHTLY_REAL_CONFIRM=1 in the caller environment",
        "OPES environment is blocked by this wrapper",
        "the wrapper never commits repository changes",
    ],
}

code_audit = {"enabled": os.environ.get("CODE_AUDIT_ENABLED", "1") == "1"}
code_audit_json = os.environ.get("CODE_AUDIT_JSON_FILE", "").strip()
if code_audit_json:
    code_audit["json_file"] = code_audit_json
    try:
        with open(code_audit_json, encoding="utf-8") as fh:
            audit_payload = json.load(fh)
        code_audit["schema_version"] = audit_payload.get("schema_version", "")
        code_audit["deadcode_source"] = (audit_payload.get("deadcode") or {}).get("source", "")
        code_audit["metrics"] = audit_payload.get("metrics") or {}
    except (OSError, json.JSONDecodeError) as exc:
        code_audit["load_error"] = str(exc)
if code_audit["enabled"] or code_audit_json:
    payload["code_audit"] = code_audit

tmp = result_file + ".tmp"
with open(tmp, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, indent=2, sort_keys=True)
    fh.write("\n")
os.replace(tmp, result_file)
PY
}

print_phase_diff_if_failed() {
  local exit_code="$1"
  if [[ "$exit_code" == "0" ]]; then
    return 0
  fi
  python3 - "$results_dir" "$result_file" <<'PY'
import difflib
import glob
import json
import os
import sys

results_dir, current_file = sys.argv[1:3]

try:
    with open(current_file, encoding="utf-8") as fh:
        current = json.load(fh)
except (OSError, json.JSONDecodeError):
    return_code = 0
    raise SystemExit(return_code)

current_path = os.path.realpath(current_file)
last_green = None
for path in sorted(glob.glob(os.path.join(results_dir, "resultado_*.json"))):
    if os.path.realpath(path) == current_path:
        continue
    try:
        with open(path, encoding="utf-8") as fh:
            payload = json.load(fh)
    except (OSError, json.JSONDecodeError):
        continue
    if payload.get("status") == "ok":
        last_green = (path, payload)

if not last_green:
    print("nightly_phase_diff_against_last_green=unavailable")
    return_code = 0
    raise SystemExit(return_code)

green_path, green = last_green
green_phases = [str(item) for item in green.get("phases") or []]
current_phases = [str(item) for item in current.get("phases") or []]
print("nightly_phase_diff_against_last_green=begin")
for line in difflib.unified_diff(
    green_phases,
    current_phases,
    fromfile="last_green:" + os.path.basename(green_path),
    tofile="failed:" + os.path.basename(current_file),
    lineterm="",
):
    print(line)
print("nightly_phase_diff_against_last_green=end")
PY
}

finish() {
  local exit_code="$1"
  write_result_json "$exit_code" "$phase" || true
  echo "nightly_result_json=$result_file"
  print_phase_diff_if_failed "$exit_code" || true
  finalized="1"
  exit "$exit_code"
}

trap 'if [[ "$finalized" != "1" ]]; then write_result_json "$?" "$phase" || true; fi' EXIT
trap 'phase="interrupted"; exit 130' INT TERM

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 es requerido para escribir resultado JSON" >&2
  finish 127
fi

if [[ ! -x "$smoke_script" ]]; then
  echo "smoke base no ejecutable: $smoke_script" >&2
  phase="base_smoke_unavailable"
  finish 2
fi

if [[ -n "${ORQUESTA_OPES_BASE_URL:-}" ||
  -n "${OPES_BASE_URL:-}" ||
  "${ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM:-0}" == "1" ||
  "${ORQUESTA_OPES_PRODUCTIVE_CONFIRM:-0}" == "1" ]]; then
  echo "nightly bloqueado: OPES/productivo debe estar vacio para este smoke" >&2
  phase="blocked_opes_environment"
  finish 2
fi

run_code_audit_phase() {
  if [[ "$code_audit_enabled" != "1" ]]; then
    echo "code_audit=disabled" | tee -a "$log_file"
    return 0
  fi
  if [[ ! -x "$code_audit_script" ]]; then
    echo "code_audit_unavailable=$code_audit_script" | tee -a "$log_file" >&2
    phase="code_audit_unavailable"
    return 2
  fi

  phase="code_audit_started"
  mkdir -p "$code_audit_dir" || {
    echo "code_audit_results_dir_unavailable=$code_audit_dir" | tee -a "$log_file" >&2
    return 2
  }

  local audit_output audit_status line
  audit_output="$(
    ORQUESTA_AUDIT_RUN_ID="$run_id" \
      "$code_audit_script" --out-dir "$code_audit_dir" 2>&1
  )"
  audit_status=$?
  printf '%s\n' "$audit_output" | tee -a "$log_file"
  while IFS= read -r line; do
    case "$line" in
      code_audit_json=*)
        code_audit_json="${line#code_audit_json=}"
        ;;
      code_audit_sqlite=*)
        code_audit_sqlite="${line#code_audit_sqlite=}"
        ;;
    esac
  done <<<"$audit_output"
  if [[ "$audit_status" != "0" ]]; then
    phase="code_audit_failed"
    return "$audit_status"
  fi
  if [[ -z "$code_audit_json" || ! -f "$code_audit_json" ]]; then
    echo "code_audit_json_missing=$code_audit_json" | tee -a "$log_file" >&2
    phase="code_audit_json_missing"
    return 1
  fi

  phase="code_audit_ratchet"
  python3 - "$results_dir" "$result_file" "$code_audit_json" <<'PY' | tee -a "$log_file"
import glob
import json
import os
import sys

results_dir, result_file, current_json = sys.argv[1:4]

with open(current_json, encoding="utf-8") as fh:
    current = json.load(fh)
current_metrics = current.get("metrics") or {}

if current.get("schema_version") != "orquesta_code_audit.v0":
    print("code_audit_ratchet=invalid")
    print("code_audit_invalid_reason=schema_version")
    raise SystemExit(1)

deadcode_source = (current.get("deadcode") or {}).get("source", "")
if deadcode_source in {"snapshot_file", "unavailable", "deadcode_tool_failed"}:
    print("code_audit_ratchet=invalid")
    print("code_audit_invalid_reason=deadcode_source:" + str(deadcode_source))
    raise SystemExit(1)

required_keys = ("deadcode_candidates", "helper_duplicate_definitions")
missing = [key for key in required_keys if key not in current_metrics]
if missing:
    print("code_audit_ratchet=invalid")
    print("code_audit_invalid_reason=missing_metrics:" + ",".join(missing))
    raise SystemExit(1)

try:
    current_values = {key: int(current_metrics[key]) for key in required_keys}
except (TypeError, ValueError):
    print("code_audit_ratchet=invalid")
    print("code_audit_invalid_reason=non_integer_metrics")
    raise SystemExit(1)

last_green = None
current_result_path = os.path.realpath(result_file)
for path in sorted(glob.glob(os.path.join(results_dir, "resultado_*.json"))):
    if os.path.realpath(path) == current_result_path:
        continue
    try:
        with open(path, encoding="utf-8") as fh:
            payload = json.load(fh)
    except (OSError, json.JSONDecodeError):
        continue
    if payload.get("status") != "ok":
        continue
    metrics = (payload.get("code_audit") or {}).get("metrics") or {}
    if "deadcode_candidates" in metrics and "helper_duplicate_definitions" in metrics:
        last_green = (path, metrics)

if not last_green:
    print("code_audit_ratchet=unavailable")
    raise SystemExit(0)

green_path, green_metrics = last_green
regressions = []
for key in required_keys:
    current_value = current_values[key]
    green_value = int(green_metrics.get(key, 0))
    if current_value > green_value:
        regressions.append((key, green_value, current_value))

if regressions:
    print("code_audit_ratchet=failed")
    print("code_audit_ratchet_baseline=" + os.path.basename(green_path))
    for key, green_value, current_value in regressions:
        print(f"code_audit_ratchet_regression={key}:{green_value}->{current_value}")
    raise SystemExit(1)

print("code_audit_ratchet=ok")
print("code_audit_ratchet_baseline=" + os.path.basename(green_path))
PY
  local ratchet_status=${PIPESTATUS[0]}
  if [[ "$ratchet_status" != "0" ]]; then
    phase="code_audit_ratchet_failed"
    return "$ratchet_status"
  fi
  phase="code_audit_completed"
  return 0
}

echo "orquesta_smoke_nightly_mode=$mode"
echo "orquesta_smoke_nightly_result=$result_file"
echo "orquesta_smoke_nightly_log=$log_file"

: >"$log_file"
run_code_audit_phase
audit_status=$?
if [[ "$audit_status" != "0" ]]; then
  finish "$audit_status"
fi

if [[ "$mode" == "real" ]]; then
  phase="real_started"
  (
    export ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=0
    export ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1
    export ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1
    export ORQUESTA_OPES_BASE_URL=""
    export OPES_BASE_URL=""
    "$smoke_script"
  ) > >(tee -a "$log_file") 2>&1
  smoke_status=$?
else
  phase="preflight_started"
  (
    export ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1
    export ORQUESTA_OPES_BASE_URL=""
    export OPES_BASE_URL=""
    "$smoke_script"
  ) > >(tee -a "$log_file") 2>&1
  smoke_status=$?
fi

finish "$smoke_status"
