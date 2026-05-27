#!/usr/bin/env bash
set -euo pipefail

server_url="${ORQUESTA_SERVER_URL:-http://127.0.0.1:8787}"
orquesta_bin="${ORQUESTA_BIN:-orquesta}"
agent=""
task_ref=""
project_ref=""
no_auto=0
dry_run=0

usage() {
  cat <<'USAGE'
Uso: scripts/inicio_agente.sh <agente> [--tarea REF] [--proyecto REF] [--no-auto] [--dry-run]

Wrapper manual de compatibilidad. Exige servidor residente listo y solo delega
en CLI/API publica; no muta stores, worktrees ni runtime por fuera del control
plane. Contrato T121: recuperacion u operacion asistida, nunca runtime
paralelo para agentes gobernados por OrquestaV2.
USAGE
}

fail_public() {
  printf 'manual_agent_ops_blocked: %s\n' "$1" >&2
  exit 2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --tarea)
      [ "$#" -ge 2 ] || fail_public "tarea_sin_valor"
      task_ref="$2"
      shift 2
      ;;
    --proyecto)
      [ "$#" -ge 2 ] || fail_public "proyecto_sin_valor"
      project_ref="$2"
      shift 2
      ;;
    --no-auto)
      no_auto=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*)
      fail_public "opcion_incompatible:$1"
      ;;
    *)
      [ -z "$agent" ] || fail_public "agente_duplicado"
      agent="$1"
      shift
      ;;
  esac
done

[ -n "$agent" ] || fail_public "agente_requerido"

check_readiness() {
  if [ "${ORQUESTA_MANUAL_AGENT_ALLOW_OFFLINE:-0}" = "1" ]; then
    fail_public "modo_offline_bloqueado: use servidor residente o decision explicita del director"
  fi
  command -v curl >/dev/null 2>&1 || fail_public "curl_no_disponible"
  readiness="$(curl -fsS --max-time 2 "$server_url/api/v0/server/readiness" 2>/dev/null || true)"
  [ -n "$readiness" ] || fail_public "servidor_no_disponible: arranque go run ./cmd/orquesta-server run"
  printf '%s' "$readiness" | grep -q '"ready"[[:space:]]*:[[:space:]]*true' || fail_public "servidor_no_ready"
}

project_args=()
[ -z "$project_ref" ] || project_args=(--proyecto "$project_ref")

if [ "$dry_run" -eq 1 ]; then
  printf 'manual_agent_ops_dry_run: contrato=recuperacion_asistida readiness=%s agente=%s tarea=%s proyecto=%s no_auto=%s\n' \
    "$server_url/api/v0/server/readiness" "$agent" "${task_ref:-none}" "${project_ref:-none}" "$no_auto"
  exit 0
fi

check_readiness
command -v "$orquesta_bin" >/dev/null 2>&1 || fail_public "cli_no_disponible:$orquesta_bin"

"$orquesta_bin" sesion inicio "$agent" "${project_args[@]}"
"$orquesta_bin" tarea listar "$agent" "${project_args[@]}"

if [ -n "$task_ref" ]; then
  "$orquesta_bin" tarea iniciar "$task_ref" --agente "$agent" "${project_args[@]}"
elif [ "$no_auto" -eq 0 ]; then
  fail_public "auto_inicio_sin_tarea_bloqueado: indique --tarea o use --no-auto"
fi
