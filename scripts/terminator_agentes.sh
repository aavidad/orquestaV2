#!/usr/bin/env bash
set -euo pipefail

plan_file="scripts/agentes.orquestador.plan"
confirm=0
dry_run=1

usage() {
  cat <<'USAGE'
Uso: scripts/terminator_agentes.sh [--plan FICHERO] [--confirm]

Adaptador grafico opcional. Terminator no es contrato del nucleo; este wrapper
solo abre consolas tras confirmacion y cada consola delega en inicio_agente.sh
para pasar por servidor residente y CLI/API publica.
USAGE
}

fail_public() {
  printf 'manual_agent_ops_blocked: %s\n' "$1" >&2
  exit 2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --plan)
      [ "$#" -ge 2 ] || fail_public "plan_sin_valor"
      plan_file="$2"
      shift 2
      ;;
    --confirm)
      confirm=1
      dry_run=0
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
      fail_public "argumento_incompatible:$1"
      ;;
  esac
done

[ -f "$plan_file" ] || fail_public "plan_no_encontrado:$plan_file"

if [ "$confirm" -ne 1 ]; then
  printf 'manual_agent_ops_no_auto_fleet_seed: contrato=recuperacion_asistida terminal=terminator dry-run plan=%s\n' "$plan_file"
  dry_run=1
fi

[ "$dry_run" -eq 1 ] || command -v terminator >/dev/null 2>&1 || fail_public "terminator_no_disponible"

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ""|\#*) continue ;;
  esac
  agent="${line%%|*}"
  [ -n "$agent" ] || continue
  if [ "$dry_run" -eq 1 ]; then
    printf 'manual_agent_ops_dry_run: contrato=recuperacion_asistida agente=%s terminal=terminator\n' "$agent"
  else
    terminator --title "orquesta $agent" -x bash -lc "scripts/inicio_agente.sh '$agent' --no-auto; exec bash" &
  fi
done < "$plan_file"
