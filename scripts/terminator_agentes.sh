#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORQUESTA_BIN="$ROOT_DIR/orquesta"
ORQUESTA_DB_PATH="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"
PLAN_FILE=""
MODE="dry-run"
LAUNCH_MODE="windows"
TERMINATOR_WRAPPER="${ORQUESTA_TERMINATOR_WRAPPER:-}"

usage() {
  cat <<'EOF'
Uso:
  scripts/terminator_agentes.sh [--ejecutar] [--tabs|--ventanas] <fichero.plan>

Abre Terminator con una consola por agente usando el mismo fichero .plan.
Si varios agentes comparten proyecto, crea/reutiliza un worktree por agente.

Por defecto trabaja en modo seco.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ejecutar)
      MODE="apply"
      shift
      ;;
    --tabs)
      LAUNCH_MODE="tabs"
      shift
      ;;
    --ventanas)
      LAUNCH_MODE="windows"
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
        exit 1
      fi
      ;;
  esac
done

if [[ -z "$PLAN_FILE" || ! -f "$PLAN_FILE" ]]; then
  usage
  exit 1
fi

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

sql_escape() {
  printf "%s" "$1" | sed "s/'/''/g"
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
  sqlite3 "$ORQUESTA_DB_PATH" "
    SELECT w.ruta_abs
    FROM worktrees w
    JOIN proyectos p ON p.id = w.proyecto_id
    WHERE lower(w.agente) = lower('$(sql_escape "$agente")')
      AND p.ruta_abs = '$(sql_escape "$ruta_proyecto")'
      AND w.estado = 'activa'
    ORDER BY w.id DESC
    LIMIT 1;
  " 2>/dev/null | head -n 1
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

  "$ORQUESTA_BIN" worktree crear "$agente" "$ruta_proyecto" --motivo "consola terminator para $agente" >&2
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

preflight_apply() {
  if ! command -v terminator >/dev/null 2>&1; then
    echo "No encuentro 'terminator' en PATH." >&2
    exit 1
  fi
  if [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
    echo "No detecto DISPLAY ni WAYLAND_DISPLAY; no puedo abrir Terminator." >&2
    exit 1
  fi
}

launch_window() {
  local titulo_tab="$1"
  local cwd_trabajo="$2"
  local bash_cmd="$3"
  if [[ -n "$TERMINATOR_WRAPPER" ]]; then
    bash -lc "$TERMINATOR_WRAPPER terminator -u -T \"\$1\" --working-directory=\"\$2\" -x bash -lc \"\$3\"" _ "$titulo_tab" "$cwd_trabajo" "$bash_cmd" &
  else
    terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd" &
  fi
}

launch_tab() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  if [[ $tab_index -eq 0 ]]; then
    if [[ -n "$TERMINATOR_WRAPPER" ]]; then
      bash -lc "$TERMINATOR_WRAPPER terminator -u -T \"\$1\" --working-directory=\"\$2\" -x bash -lc \"\$3\"" _ "$titulo_tab" "$cwd_trabajo" "$bash_cmd" &
    else
      terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd" &
    fi
    sleep 1
  else
    if [[ -n "$TERMINATOR_WRAPPER" ]]; then
      bash -lc "$TERMINATOR_WRAPPER terminator --new-tab -T \"\$1\" --working-directory=\"\$2\" -x bash -lc \"\$3\"" _ "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
    else
      terminator --new-tab -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
    fi
    sleep 0.3
  fi
}

show_dry_run_launch() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  if [[ "$LAUNCH_MODE" == "tabs" ]]; then
    if [[ $tab_index -eq 0 ]]; then
      run_or_echo terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
    else
      run_or_echo terminator --new-tab -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
    fi
  else
    run_or_echo terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
  fi
}

if [[ "$MODE" == "apply" ]]; then
  preflight_apply
  "$ROOT_DIR/scripts/cargar_agentes.sh" --ejecutar "$PLAN_FILE"
else
  "$ROOT_DIR/scripts/cargar_agentes.sh" "$PLAN_FILE"
fi

tab_index=0

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

  if [[ "$MODE" == "apply" ]]; then
    if [[ "$LAUNCH_MODE" == "tabs" ]]; then
      launch_tab "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
    else
      launch_window "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
      sleep 0.3
    fi
  else
    show_dry_run_launch "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
  fi

  tab_index=$((tab_index + 1))
done

if [[ "$MODE" == "apply" ]]; then
  echo "Terminator lanzado."
else
  echo "Modo seco completado. Usa --ejecutar para abrir Terminator."
fi
