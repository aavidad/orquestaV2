#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"

if [[ ! -x "$ORQUESTA_BIN" ]]; then
  echo "No encuentro el binario ejecutable en: $ORQUESTA_BIN" >&2
  exit 1
fi

usage() {
  cat <<'EOF'
Uso:
  scripts/cargar_agentes.sh [--ejecutar] <fichero.plan>

Formato del fichero: una línea por agente
  agente|rol|ruta_proyecto|titulo_tarea|prioridad|modulo|nota_asignacion|conector|comando_runtime

Ejemplo:
  Codex2|programador|/home/alberto/Trabajo/PlataformaMunicipal/orquestador|Locks y worktrees|alta|orquestador|locks y worktrees|codex-cli|codex

Comportamiento:
  - registra o reactiva el agente
  - descubre el proyecto por ruta
  - activa la asignación del agente al proyecto
  - crea una tarea ya asignada al agente

Por defecto trabaja en modo seco y solo muestra lo que haría.
Usa --ejecutar para aplicar cambios reales.
EOF
}

MODE="dry-run"
PLAN_FILE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ejecutar)
      MODE="apply"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -z "$PLAN_FILE" ]]; then
        PLAN_FILE="$1"
        shift
      else
        echo "Argumento inesperado: $1" >&2
        usage
        exit 1
      fi
      ;;
  esac
done

if [[ -z "$PLAN_FILE" ]]; then
  usage
  exit 1
fi

if [[ ! -f "$PLAN_FILE" ]]; then
  echo "No existe el fichero: $PLAN_FILE" >&2
  exit 1
fi

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

run_or_echo() {
  if [[ "$MODE" == "apply" ]]; then
    "$@"
  else
    printf '[SECO] '
    printf '%q ' "$@"
    printf '\n'
  fi
}

declare -A DISCOVERED=()
LINE_NO=0

while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  LINE_NO=$((LINE_NO + 1))
  line="$(trim "$raw_line")"

  if [[ -z "$line" || "${line:0:1}" == "#" ]]; then
    continue
  fi

  IFS='|' read -r agente rol ruta_proyecto titulo prioridad modulo nota conector comando_runtime extra <<<"$line"

  agente="$(trim "${agente:-}")"
  rol="$(trim "${rol:-programador}")"
  ruta_proyecto="$(trim "${ruta_proyecto:-}")"
  titulo="$(trim "${titulo:-}")"
  prioridad="$(trim "${prioridad:-media}")"
  modulo="$(trim "${modulo:-orquestador}")"
  nota="$(trim "${nota:-}")"
  conector="$(trim "${conector:-}")"
  comando_runtime="$(trim "${comando_runtime:-}")"
  extra="$(trim "${extra:-}")"

  if [[ -n "$extra" ]]; then
    echo "Línea $LINE_NO inválida: sobran columnas" >&2
    exit 1
  fi

  if [[ -z "$agente" || -z "$ruta_proyecto" || -z "$titulo" ]]; then
    echo "Línea $LINE_NO inválida: agente, ruta_proyecto y titulo_tarea son obligatorios" >&2
    exit 1
  fi

  if [[ -z "$nota" ]]; then
    nota="$titulo"
  fi

  echo
  echo "==> $agente :: $titulo"

  run_or_echo "$ORQUESTA_BIN" config agente-nuevo "$agente" "$rol"

  if [[ -z "${DISCOVERED[$ruta_proyecto]+x}" ]]; then
    run_or_echo "$ORQUESTA_BIN" proyecto descubrir "$ruta_proyecto"
    DISCOVERED["$ruta_proyecto"]=1
  fi

  run_or_echo "$ORQUESTA_BIN" asignacion activar "$agente" "$ruta_proyecto" --nota "$nota"
  run_or_echo "$ORQUESTA_BIN" tarea nueva --titulo "$titulo" --proyecto "$ruta_proyecto" --prioridad "$prioridad" --modulo "$modulo" --agente "$agente" --por alberto
done < "$PLAN_FILE"

echo
if [[ "$MODE" == "apply" ]]; then
  echo "Plan aplicado."
else
  echo "Modo seco completado. Usa --ejecutar para aplicar cambios."
fi
