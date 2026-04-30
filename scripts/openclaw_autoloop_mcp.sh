#!/usr/bin/env bash
set -euo pipefail

MCP_URL="${ORQUESTA_MCP_URL:-http://127.0.0.1:16543/api/mcp}"
SUPERVISOR="${SUPERVISOR:-OpenClaw}"
OUTER_SLEEP="${OUTER_SLEEP:-20}"
IDLE_SLEEP="${IDLE_SLEEP:-60}"
QUIET_SLEEP="${QUIET_SLEEP:-90}"
PASSIVE_RECOVERY_SLEEP="${PASSIVE_RECOVERY_SLEEP:-300}"
MANUAL_QUEUE_SLEEP="${MANUAL_QUEUE_SLEEP:-180}"
EXIT_ON_COMPLETE="${EXIT_ON_COMPLETE:-true}"
LOG_FILE="${LOG_FILE:-/tmp/orquesta-openclaw-autoloop.log}"
MAX_SAFE_ACTIONS_PER_TICK="${MAX_SAFE_ACTIONS_PER_TICK:-1}"
MCP_TIMEOUT="${MCP_TIMEOUT:-20}"
SAFE_ACTION_WHITELIST="${SAFE_ACTION_WHITELIST:-asignar_tarea_libre,reservar_tarea_libre,replanificar_por_cuota,rebalancear_reserva,seguir_guidance_durable}"
REJECT_COOLDOWN_SECS="${REJECT_COOLDOWN_SECS:-120}"
REJECT_STATE_FILE="${REJECT_STATE_FILE:-/tmp/orquesta-openclaw-autoloop.rejects}"
SELF_HEAL_COOLDOWN_SECS="${SELF_HEAL_COOLDOWN_SECS:-20}"
SELF_HEAL_STATE_FILE="${SELF_HEAL_STATE_FILE:-/tmp/orquesta-openclaw-autoloop.self_heal}"

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

self_heal_cooldown_active() {
  local now_ts last_ts
  now_ts="$(date +%s)"
  [[ -f "$SELF_HEAL_STATE_FILE" ]] || return 1
  last_ts="$(cat "$SELF_HEAL_STATE_FILE" 2>/dev/null || true)"
  [[ -n "$last_ts" ]] || return 1
  (( now_ts - last_ts < SELF_HEAL_COOLDOWN_SECS ))
}

record_self_heal() {
  date +%s > "$SELF_HEAL_STATE_FILE"
}

while true; do
  timestamp="$(date -Iseconds)"
  if ! operator_response="$(call_tool "orquesta.openclaw.operator" "{\"supervisor\":\"${SUPERVISOR}\",\"rich\":false}")"; then
    printf '%s operator_error\n' "$timestamp" >> "$LOG_FILE"
    sleep "${OUTER_SLEEP}"
    continue
  fi
  mapfile -t summary < <(python3 - "$operator_response" <<'PY'
import json, sys
resp = json.loads(sys.argv[1])
result = resp.get("result", {})
payload = result.get("structuredContent", {}) or {}
server = payload.get("server_operational") or {}
plan = payload.get("next_recovery_plan") or {}
next_safe = payload.get("next_safe_action") or {}
next_action = payload.get("next_action") or {}
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
print("next_quota_reset_at=" + str(server.get("nextQuotaResetAt", "")))
print("next_action=" + str(next_action.get("action", "")))
print("next_action_target=" + str(next_action.get("target", "")))
print("next_safe_action=" + str(next_safe.get("action", "")))
print("next_safe_target=" + str(next_safe.get("target", "")))
print("next_safe_assignee=" + str(next_safe.get("assignee", "")))
print("safe_queue_total=" + str(queue.get("safe", 0)))
print("manual_queue_total=" + str(queue.get("manual", 0)))
print("queue_total=" + str(queue.get("total", 0)))
PY
)

  declare -A kv=()
  for line in "${summary[@]}"; do
    key="${line%%=*}"
    value="${line#*=}"
    kv["$key"]="$value"
  done

  printf '%s operational=%s tasks=%s reserved=%s compaction_debt=%s next_action=%s next_safe=%s safe_queue=%s queue_total=%s next_recovery=%s auto=%s rearm=%s state=%s reason=%s\n' \
    "$timestamp" \
    "${kv[operational]}" \
    "${kv[tasks_active]}" \
    "${kv[tasks_reserved]}" \
    "${kv[compaction_debt]}" \
    "${kv[next_action]}" \
    "${kv[next_safe_action]}" \
    "${kv[safe_queue_total]}" \
    "${kv[queue_total]}" \
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

  should_self_heal="false"
  if [[ "${kv[operational]}" != "true" ]]; then
    should_self_heal="true"
  elif [[ "${kv[next_recovery_action]}" != "" ]]; then
    should_self_heal="true"
  fi
  if [[ "${kv[next_recovery_action]}" == "wait_quota_reset" ]]; then
    should_self_heal="false"
  fi

  if [[ "$should_self_heal" == "true" && ! self_heal_cooldown_active ]]; then
    if self_heal_response="$(call_tool "orquesta.server.self_heal" "{\"supervisor\":\"${SUPERVISOR}\"}")"; then
      printf '%s self_heal %s\n' "$timestamp" "$self_heal_response" >> "$LOG_FILE"
      record_self_heal
      mapfile -t heal_summary < <(python3 - "$self_heal_response" <<'PY'
import json, sys
resp = json.loads(sys.argv[1])
result = resp.get("result", {})
if resp.get("error") or resp.get("isError") or result.get("isError"):
    print("heal_ok=false")
    print("heal_operational=false")
    print("heal_state=")
    print("heal_reason=self_heal_error")
    print("heal_tasks_active=0")
    print("heal_tasks_reserved=0")
    print("heal_next_recovery_action=")
    raise SystemExit(0)
payload = result.get("structuredContent", {}) or {}
server = payload.get("operational") or {}
plan = server.get("nextRecoveryPlan") or {}
print("heal_ok=" + str(payload.get("ok", False)).lower())
print("heal_operational=" + str(server.get("operational", False)).lower())
print("heal_state=" + str(server.get("state", "")))
print("heal_reason=" + str(server.get("reason", "")))
print("heal_tasks_active=" + str(server.get("tasksInProgress", 0)))
print("heal_tasks_reserved=" + str(server.get("reservedTasks", 0)))
print("heal_next_recovery_action=" + str(plan.get("action", "")))
print("heal_next_recovery_auto=" + str(plan.get("autoExecutable", False)).lower())
print("heal_next_recovery_requires_rearm=" + str(plan.get("requiresRearm", False)).lower())
PY
)
      declare -A heal_kv=()
      for line in "${heal_summary[@]}"; do
        key="${line%%=*}"
        value="${line#*=}"
        heal_kv["$key"]="$value"
      done
      printf '%s self_heal_verified ok=%s operational=%s tasks=%s reserved=%s next_recovery=%s state=%s reason=%s\n' \
        "$timestamp" \
        "${heal_kv[heal_ok]}" \
        "${heal_kv[heal_operational]}" \
        "${heal_kv[heal_tasks_active]}" \
        "${heal_kv[heal_tasks_reserved]}" \
        "${heal_kv[heal_next_recovery_action]}" \
        "${heal_kv[heal_state]}" \
        "${heal_kv[heal_reason]}"
      kv[operational]="${heal_kv[heal_operational]}"
      kv[state]="${heal_kv[heal_state]}"
      kv[reason]="${heal_kv[heal_reason]}"
      kv[tasks_active]="${heal_kv[heal_tasks_active]}"
      kv[tasks_reserved]="${heal_kv[heal_tasks_reserved]}"
      kv[next_recovery_action]="${heal_kv[heal_next_recovery_action]}"
      kv[next_recovery_auto]="${heal_kv[heal_next_recovery_auto]}"
      kv[next_recovery_requires_rearm]="${heal_kv[heal_next_recovery_requires_rearm]}"
    else
      printf '%s self_heal_error\n' "$timestamp" >> "$LOG_FILE"
      record_self_heal
    fi
  fi

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
  sleep_for="${OUTER_SLEEP}"
  if [[ "${kv[next_recovery_action]}" == "wait_quota_reset" && "${kv[safe_queue_total]}" == "0" ]]; then
    sleep_for="${PASSIVE_RECOVERY_SLEEP}"
    if [[ -n "${kv[next_quota_reset_at]}" ]]; then
      now_ts="$(date +%s)"
      reset_ts="$(date -d "${kv[next_quota_reset_at]}" +%s 2>/dev/null || echo 0)"
      if [[ "$reset_ts" -gt "$now_ts" ]]; then
        delta="$((reset_ts - now_ts))"
        if [[ "$delta" -lt "$sleep_for" ]]; then
          sleep_for="$delta"
        fi
        if [[ "$sleep_for" -lt 60 ]]; then
          sleep_for=60
        fi
      fi
    fi
  fi
  if [[ "${kv[operational]}" == "true" && "${kv[next_recovery_action]}" == "" && "${kv[safe_queue_total]}" == "0" && "${kv[manual_queue_total]}" != "0" ]]; then
    sleep_for="${MANUAL_QUEUE_SLEEP}"
  fi
  if [[ "${kv[operational]}" == "true" && "${kv[next_recovery_action]}" == "" && "${kv[safe_queue_total]}" == "0" && "${kv[manual_queue_total]}" == "0" ]]; then
    sleep_for="${IDLE_SLEEP}"
    if [[ "${kv[next_action]}" == "" ]]; then
      sleep_for="${QUIET_SLEEP}"
    fi
  fi
  sleep "${sleep_for}"
done
