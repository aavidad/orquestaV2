#!/usr/bin/env bash
set -euo pipefail

plan_file="scripts/agentes.orquestador.plan"
confirm=0
dry_run=1

usage() {
  cat <<'USAGE'
Uso: scripts/cargar_agentes.sh [--plan FICHERO] [--confirm]

Carga asistida de agentes manuales. Por defecto solo lista el plan y no arranca
flotas legacy por seed. Con --confirm delega cada agente en inicio_agente.sh,
que exige servidor residente y CLI/API publica.
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

printf 'manual_agent_ops_no_auto_fleet_seed: contrato=recuperacion_asistida plan=%s modo=%s\n' "$plan_file" "$([ "$confirm" -eq 1 ] && printf confirm || printf dry-run)"

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ""|\#*) continue ;;
  esac
  agent="${line%%|*}"
  [ -n "$agent" ] || continue
  if [ "$dry_run" -eq 1 ]; then
    printf 'manual_agent_ops_dry_run: contrato=recuperacion_asistida agente=%s accion=inicio_asistido\n' "$agent"
  else
    scripts/inicio_agente.sh "$agent" --no-auto
  fi
done < "$plan_file"
