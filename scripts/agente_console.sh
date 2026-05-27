#!/usr/bin/env bash
set -euo pipefail

agent=""
project_ref=""
task_ref=""
dry_run=0

usage() {
  cat <<'USAGE'
Uso: scripts/agente_console.sh <agente> [--proyecto REF] [--tarea REF] [--dry-run]

Consola manual de compatibilidad. No lanza runtime directo: verifica/delega la
sesion por inicio_agente.sh y deja el control al operador. Contrato T121:
recuperacion asistida sobre servidor residente.
USAGE
}

fail_public() {
  printf 'manual_agent_ops_blocked: %s\n' "$1" >&2
  exit 2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --proyecto)
      [ "$#" -ge 2 ] || fail_public "proyecto_sin_valor"
      project_ref="$2"
      shift 2
      ;;
    --tarea)
      [ "$#" -ge 2 ] || fail_public "tarea_sin_valor"
      task_ref="$2"
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --runtime-cmd|--comando-runtime)
      fail_public "runtime_directo_bloqueado: use runtime gobernado por OrquestaV2"
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

args=("$agent" --no-auto)
[ -z "$project_ref" ] || args+=(--proyecto "$project_ref")
[ -z "$task_ref" ] || args+=(--tarea "$task_ref")
[ "$dry_run" -eq 0 ] || args+=(--dry-run)

scripts/inicio_agente.sh "${args[@]}"
