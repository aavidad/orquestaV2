#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
PLAN_FILE=""
MODE="dry-run"

usage() {
  cat <<'EOF'
Uso:
  scripts/preparar_lanzamiento_agentes.sh [--ejecutar] <fichero.plan>

Prepara el lanzamiento de agentes y emite una línea TSV por agente con:
  agente  ruta_proyecto  cwd_trabajo  conector  titulo  bash_cmd

La salida está pensada para adaptadores de terminal como Terminator, tmux o PTY.
EOF
}

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

if [[ -z "$PLAN_FILE" || ! -f "$PLAN_FILE" ]]; then
  usage
  exit 1
fi

if [[ "$MODE" == "apply" && ! -x "$ORQUESTA_BIN" ]]; then
  echo "No encuentro el binario ejecutable: $ORQUESTA_BIN" >&2
  exit 1
fi

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

declare -A PROJECT_COUNT=()
declare -a PLAN_LINES=()

while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line="$(trim "$raw_line")"
  if [[ -z "$line" || "${line:0:1}" == "#" ]]; then
    continue
  fi
  PLAN_LINES+=("$line")
  IFS='|' read -r _agente _rol ruta_proyecto _titulo _prioridad _modulo _nota _conector _runtime _extra <<<"$line"
  ruta_proyecto="$(trim "${ruta_proyecto:-}")"
  PROJECT_COUNT["$ruta_proyecto"]=$(( ${PROJECT_COUNT["$ruta_proyecto"]:-0} + 1 ))
done < "$PLAN_FILE"

existing_worktree_path() {
  local agente="$1"
  local ruta_proyecto="$2"
  "$ORQUESTA_BIN" worktree resolver "$agente" "$ruta_proyecto" 2>/dev/null || true
}

ensure_worktree() {
  local agente="$1"
  local ruta_proyecto="$2"
  local path
  path="$(existing_worktree_path "$agente" "$ruta_proyecto")"
  if [[ -n "$path" ]]; then
    printf '%s' "$path"
    return 0
  fi

  "$ORQUESTA_BIN" worktree crear "$agente" "$ruta_proyecto" --motivo "launcher adapter para $agente" >&2
  path="$(existing_worktree_path "$agente" "$ruta_proyecto")"
  if [[ -z "$path" ]]; then
    echo "No he podido resolver la ruta del worktree para $agente" >&2
    exit 1
  fi
  printf '%s' "$path"
}

build_bash_command() {
  local agente="$1"
  local ruta_proyecto="$2"
  local cwd_trabajo="$3"
  local conector="$4"
  local runtime_cmd="$5"
  local titulo="$6"

  printf '%q ' "$ROOT_DIR/scripts/agente_console.sh" "$agente" "$ruta_proyecto" "$cwd_trabajo" "$conector" "$runtime_cmd" "$titulo"
}

if [[ "$MODE" == "apply" ]]; then
  "$ROOT_DIR/scripts/cargar_agentes.sh" --ejecutar "$PLAN_FILE" >&2
else
  "$ROOT_DIR/scripts/cargar_agentes.sh" "$PLAN_FILE" >&2
fi

for line in "${PLAN_LINES[@]}"; do
  IFS='|' read -r agente rol ruta_proyecto titulo prioridad modulo nota conector runtime_cmd extra <<<"$line"
  agente="$(trim "${agente:-}")"
  ruta_proyecto="$(trim "${ruta_proyecto:-}")"
  conector="$(trim "${conector:-codex-cli}")"
  runtime_cmd="$(trim "${runtime_cmd:-codex}")"

  cwd_trabajo="$ruta_proyecto"
  if [[ ${PROJECT_COUNT["$ruta_proyecto"]:-0} -gt 1 ]]; then
    if [[ "$MODE" == "apply" ]]; then
      cwd_trabajo="$(ensure_worktree "$agente" "$ruta_proyecto")"
    else
      cwd_trabajo="$ruta_proyecto/.orquesta-worktrees/$(basename "$ruta_proyecto")-${agente,,}"
    fi
  fi

  titulo_tab="${agente} · $(basename "$ruta_proyecto")"
  bash_cmd="$(build_bash_command "$agente" "$ruta_proyecto" "$cwd_trabajo" "$conector" "$runtime_cmd" "$titulo_tab")"

  printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$agente" \
    "$ruta_proyecto" \
    "$cwd_trabajo" \
    "$conector" \
    "$titulo_tab" \
    "$bash_cmd"
done
