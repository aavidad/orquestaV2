#!/usr/bin/env bash
set -euo pipefail

MCP_URL="${ORQUESTA_MCP_URL:-http://127.0.0.1:16543/api/mcp}"
SUPERVISOR="${SUPERVISOR:-OpenClaw}"
OUTER_SLEEP="${OUTER_SLEEP:-15}"
EXIT_ON_COMPLETE="${EXIT_ON_COMPLETE:-true}"
LOG_FILE="${LOG_FILE:-/tmp/orquesta-openclaw-autoloop.log}"
MAX_SAFE_ACTIONS_PER_TICK="${MAX_SAFE_ACTIONS_PER_TICK:-1}"
MCP_TIMEOUT="${MCP_TIMEOUT:-8}"
SAFE_ACTION_WHITELIST="${SAFE_ACTION_WHITELIST:-asignar_tarea_libre,reservar_tarea_libre,replanificar_por_cuota,rebalancear_reserva,seguir_guidance_durable}"
REJECT_COOLDOWN_SECS="${REJECT_COOLDOWN_SECS:-120}"
REJECT_STATE_FILE="${REJECT_STATE_FILE:-/tmp/orquesta-openclaw-autoloop.rejects}"

mkdir -p "$(dirname "$LOG_FILE")"

call_tool() {
  local tool_name="$1"
  local arguments_json="${2:-{}}"
  local request
  request="$(cat <<JSON
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"${tool_name}","arguments":${arguments_json}}}
JSON
)"
  curl -fsS \
    --max-time "$MCP_TIMEOUT" \
    -H 'Content-Type: application/json' \
    -d "$request" \
    "$MCP_URL"
}

action_allowed() {
  local action_name="$1"
  local candidate
  IFS=',' read -r -a whitelist <<< "$SAFE_ACTION_WHITELIST"
  for candidate in "${whitelist[@]}"; do
    if [[ "$(printf '%s' "$candidate" | xargs)" == "$action_name" ]]; then
      return 0
    fi
  done
  return 1
}

reject_cooldown_active() {
  local signature="$1"
  local now_ts last_ts
  now_ts="$(date +%s)"
  [[ -f "$REJECT_STATE_FILE" ]] || return 1
  last_ts="$(awk -F'|' -v sig="$signature" '$1 == sig {print $2}' "$REJECT_STATE_FILE" | tail -n 1)"
  [[ -n "$last_ts" ]] || return 1
  (( now_ts - last_ts < REJECT_COOLDOWN_SECS ))
}

record_reject() {
  local signature="$1"
  local now_ts tmp_file
  now_ts="$(date +%s)"
  tmp_file="${REJECT_STATE_FILE}.tmp"
  if [[ -f "$REJECT_STATE_FILE" ]]; then
    awk -F'|' -v sig="$signature" '$1 != sig {print $0}' "$REJECT_STATE_FILE" > "$tmp_file" || true
  else
    : > "$tmp_file"
  fi
  printf '%s|%s\n' "$signature" "$now_ts" >> "$tmp_file"
  mv "$tmp_file" "$REJECT_STATE_FILE"
}

while true; do
  timestamp="$(date -Iseconds)"
  if ! operator_response="$(call_tool "orquesta.openclaw.operator" "{\"supervisor\":\"${SUPERVISOR}\",\"rich\":false}")"; then
    printf '%s operator_error\n' "$timestamp" >> "$LOG_FILE"
    sleep "${OUTER_SLEEP}"
    continue
  fi
  printf '%s operator %s\n' "$timestamp" "$operator_response" >> "$LOG_FILE"

  mapfile -t summary < <(python3 - "$operator_response" <<'PY'
import json, sys
resp = json.loads(sys.argv[1])
result = resp.get("result", {})
payload = result.get("structuredContent", {}) or {}
server = payload.get("server_operational") or {}
plan = payload.get("next_recovery_plan") or {}
next_safe = payload.get("next_safe_action") or {}
queue = payload.get("queue_summary") or {}
print("state=" + str(server.get("state", "")))
print("reason=" + str(server.get("reason", "")))
print("operational=" + str(server.get("operational", False)).lower())
print("tasks_active=" + str(server.get("tasksInProgress", 0)))
print("tasks_reserved=" + str(server.get("reservedTasks", 0)))
print("compaction_debt=" + str(server.get("compactionDebtTasks", 0)))
print("next_recovery_action=" + str(plan.get("action", "")))
print("next_recovery_auto=" + str(plan.get("autoExecutable", False)).lower())
print("next_recovery_requires_rearm=" + str(plan.get("requiresRearm", False)).lower())
print("next_safe_action=" + str(next_safe.get("action", "")))
print("next_safe_target=" + str(next_safe.get("target", "")))
print("next_safe_assignee=" + str(next_safe.get("assignee", "")))
print("safe_queue_total=" + str(queue.get("safe", 0)))
PY
)

  declare -A kv=()
  for line in "${summary[@]}"; do
    key="${line%%=*}"
    value="${line#*=}"
    kv["$key"]="$value"
  done

  printf '%s operational=%s tasks=%s reserved=%s compaction_debt=%s next_safe=%s safe_queue=%s next_recovery=%s auto=%s rearm=%s state=%s reason=%s\n' \
    "$timestamp" \
    "${kv[operational]}" \
    "${kv[tasks_active]}" \
    "${kv[tasks_reserved]}" \
    "${kv[compaction_debt]}" \
    "${kv[next_safe_action]}" \
    "${kv[safe_queue_total]}" \
    "${kv[next_recovery_action]}" \
    "${kv[next_recovery_auto]}" \
    "${kv[next_recovery_requires_rearm]}" \
    "${kv[state]}" \
    "${kv[reason]}"

  actions_applied=0
  while [[ "${actions_applied}" -lt "${MAX_SAFE_ACTIONS_PER_TICK}" && "${kv[operational]}" == "true" && "${kv[next_safe_action]}" != "" ]]; do
    if ! action_allowed "${kv[next_safe_action]}"; then
      printf '%s apply_skipped action=%s target=%s reason=not_whitelisted\n' \
        "$timestamp" "${kv[next_safe_action]}" "${kv[next_safe_target]}" >> "$LOG_FILE"
      break
    fi
    reject_signature="${kv[next_safe_action]}|${kv[next_safe_target]}|${kv[next_safe_assignee]}"
    if reject_cooldown_active "$reject_signature"; then
      printf '%s apply_skipped action=%s target=%s assignee=%s reason=reject_cooldown\n' \
        "$timestamp" "${kv[next_safe_action]}" "${kv[next_safe_target]}" "${kv[next_safe_assignee]}" >> "$LOG_FILE"
      break
    fi
    if ! apply_response="$(call_tool "orquesta.supervision.acciones.aplicar_siguiente" "{\"supervisor\":\"${SUPERVISOR}\"}")"; then
      printf '%s apply_error action=%s target=%s\n' "$timestamp" "${kv[next_safe_action]}" "${kv[next_safe_target]}" >> "$LOG_FILE"
      record_reject "$reject_signature"
      break
    fi
    printf '%s apply_next %s\n' "$timestamp" "$apply_response" >> "$LOG_FILE"
    actions_applied=$((actions_applied + 1))

    mapfile -t apply_summary < <(python3 - "$apply_response" <<'PY'
import json, sys
resp = json.loads(sys.argv[1])
result = resp.get("result", {})
if resp.get("error") or resp.get("isError") or result.get("isError"):
    print("apply_ok=false")
    print("verified_operational=false")
    print("verified_state=")
    print("verified_reason=apply_error")
    print("verified_tasks_active=0")
    print("verified_tasks_reserved=0")
    print("verified_next_safe_action=")
    print("verified_safe_queue_total=0")
    raise SystemExit(0)
payload = result.get("structuredContent", {}) or {}
verification = payload.get("verification") or {}
server = verification.get("server_operational") or {}
queue = verification.get("queue_summary") or {}
next_safe = verification.get("next_safe_action") or {}
print("apply_ok=true")
print("verified_operational=" + str(server.get("operational", False)).lower())
print("verified_state=" + str(server.get("state", "")))
print("verified_reason=" + str(server.get("reason", "")))
print("verified_tasks_active=" + str(server.get("tasksInProgress", 0)))
print("verified_tasks_reserved=" + str(server.get("reservedTasks", 0)))
print("verified_next_safe_action=" + str(next_safe.get("action", "")))
print("verified_safe_queue_total=" + str(queue.get("safe", 0)))
PY
)
    declare -A apply_kv=()
    for line in "${apply_summary[@]}"; do
      key="${line%%=*}"
      value="${line#*=}"
      apply_kv["$key"]="$value"
    done
    if [[ "${apply_kv[apply_ok]}" != "true" ]]; then
      printf '%s apply_rejected state=%s reason=%s\n' \
        "$timestamp" "${apply_kv[verified_state]}" "${apply_kv[verified_reason]}" >> "$LOG_FILE"
      record_reject "$reject_signature"
      break
    fi
    printf '%s apply_verified operational=%s tasks=%s reserved=%s next_safe=%s safe_queue=%s state=%s reason=%s\n' \
      "$timestamp" \
      "${apply_kv[verified_operational]}" \
      "${apply_kv[verified_tasks_active]}" \
      "${apply_kv[verified_tasks_reserved]}" \
      "${apply_kv[verified_next_safe_action]}" \
      "${apply_kv[verified_safe_queue_total]}" \
      "${apply_kv[verified_state]}" \
      "${apply_kv[verified_reason]}"

    kv[operational]="${apply_kv[verified_operational]}"
    kv[state]="${apply_kv[verified_state]}"
    kv[reason]="${apply_kv[verified_reason]}"
    kv[tasks_active]="${apply_kv[verified_tasks_active]}"
    kv[tasks_reserved]="${apply_kv[verified_tasks_reserved]}"
    kv[next_safe_action]="${apply_kv[verified_next_safe_action]}"
    kv[safe_queue_total]="${apply_kv[verified_safe_queue_total]}"
  done

  if [[ "${kv[next_recovery_action]}" == "server_rearm" && "${kv[next_recovery_auto]}" == "true" && "${kv[next_recovery_requires_rearm]}" == "true" ]]; then
    if ! rearm_response="$(call_tool "orquesta.server.rearm" "{\"supervisor\":\"${SUPERVISOR}\"}")"; then
      printf '%s server_rearm_error\n' "$timestamp" >> "$LOG_FILE"
      sleep "${OUTER_SLEEP}"
      continue
    fi
    printf '%s server_rearm %s\n' "$timestamp" "$rearm_response" >> "$LOG_FILE"
  fi

  if [[ "${EXIT_ON_COMPLETE}" == "true" && "${kv[operational]}" == "true" && "${kv[tasks_active]}" == "0" && "${kv[tasks_reserved]}" == "0" ]]; then
    break
  fi
  sleep "${OUTER_SLEEP}"
done
