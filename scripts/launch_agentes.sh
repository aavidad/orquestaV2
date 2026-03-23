#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREP_SCRIPT="$ROOT_DIR/scripts/preparar_lanzamiento_agentes.sh"
PLAN_FILE=""
MODE="dry-run"
LAUNCH_MODE="windows"
TERMINAL_BACKEND="${ORQUESTA_TERMINAL_BACKEND:-terminator}"
TERMINAL_LAUNCHER="${ORQUESTA_TERMINAL_LAUNCHER:-}"
TERMINAL_WRAPPER="${ORQUESTA_TERMINAL_WRAPPER:-${ORQUESTA_TERMINATOR_WRAPPER:-}}"
TMUX_SESSION_NAME="${ORQUESTA_TMUX_SESSION_NAME:-orquesta-agentes}"
TMUX_ATTACH="${ORQUESTA_TMUX_ATTACH:-1}"
declare -a TMUX_CREATED_SESSIONS=()

usage() {
  cat <<'EOF'
Uso:
  scripts/launch_agentes.sh [--ejecutar] [--tabs|--ventanas] <fichero.plan>

Lanza consolas de agentes usando un backend terminal configurable.
La preparación neutra de worktrees y comandos se delega a:
  scripts/preparar_lanzamiento_agentes.sh

Variables de entorno:
  ORQUESTA_TERMINAL_BACKEND=terminator|tmux|custom
  ORQUESTA_TERMINAL_LAUNCHER=/ruta/a/launcher
  ORQUESTA_TERMINAL_WRAPPER='comando envoltorio'
  ORQUESTA_TMUX_SESSION_NAME=orquesta-agentes
  ORQUESTA_TMUX_ATTACH=1

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

if [[ ! -x "$PREP_SCRIPT" ]]; then
  echo "No encuentro el script de preparación: $PREP_SCRIPT" >&2
  exit 1
fi

run_or_echo() {
  if [[ "$MODE" == "apply" ]]; then
    "$@"
  else
    printf '[SECO] '
    printf '%q ' "$@"
    printf '\n'
  fi
}

terminal_run_with_wrapper() {
  local terminal_cmd=("$@")
  if [[ -n "$TERMINAL_WRAPPER" ]]; then
    bash -lc "$TERMINAL_WRAPPER \"\$@\"" _ "${terminal_cmd[@]}"
  else
    "${terminal_cmd[@]}"
  fi
}

terminal_backend_preflight() {
  case "$TERMINAL_BACKEND" in
    terminator)
      if ! command -v terminator >/dev/null 2>&1; then
        echo "No encuentro 'terminator' en PATH." >&2
        exit 1
      fi
      if [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
        echo "No detecto DISPLAY ni WAYLAND_DISPLAY; no puedo abrir Terminator." >&2
        exit 1
      fi
      ;;
    tmux)
      if ! command -v tmux >/dev/null 2>&1; then
        echo "No encuentro 'tmux' en PATH." >&2
        exit 1
      fi
      ;;
    custom)
      if [[ -z "$TERMINAL_LAUNCHER" ]]; then
        echo "Con ORQUESTA_TERMINAL_BACKEND=custom debes indicar ORQUESTA_TERMINAL_LAUNCHER." >&2
        exit 1
      fi
      if [[ ! -x "$TERMINAL_LAUNCHER" ]]; then
        echo "No encuentro launcher ejecutable: $TERMINAL_LAUNCHER" >&2
        exit 1
      fi
      ;;
    *)
      echo "Backend terminal no soportado: $TERMINAL_BACKEND" >&2
      exit 1
      ;;
  esac
}

terminal_backend_launch_window() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  local tmux_session
  local tmux_window
  local session_suffix
  session_suffix=$(printf '%02d' "$tab_index")
  case "$TERMINAL_BACKEND" in
    terminator)
      terminal_run_with_wrapper terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd" &
      ;;
    tmux)
      tmux_session="${TMUX_SESSION_NAME}-${session_suffix}"
      tmux_window="$(printf '%s' "$titulo_tab" | tr ' ' '_' | tr -cd '[:alnum:]_.:-')"
      if [[ -z "$tmux_window" ]]; then
        tmux_window="agente-${session_suffix}"
      fi
      tmux new-session -d -s "$tmux_session" -n "$tmux_window" -c "$cwd_trabajo" bash -lc "$bash_cmd"
      TMUX_CREATED_SESSIONS+=("$tmux_session")
      ;;
    custom)
      "$TERMINAL_LAUNCHER" window "$titulo_tab" "$cwd_trabajo" "$bash_cmd" 0
      ;;
  esac
}

terminal_backend_launch_tab() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  case "$TERMINAL_BACKEND" in
    terminator)
      if [[ $tab_index -eq 0 ]]; then
        terminal_run_with_wrapper terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd" &
        sleep 1
      else
        terminal_run_with_wrapper terminator --new-tab -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
        sleep 0.3
      fi
      ;;
    tmux)
      launch_tmux_tab "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
      ;;
    custom)
      "$TERMINAL_LAUNCHER" tab "$titulo_tab" "$cwd_trabajo" "$bash_cmd" "$tab_index"
      ;;
  esac
}

terminal_backend_show_dry_run() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  case "$TERMINAL_BACKEND" in
    terminator)
      if [[ "$LAUNCH_MODE" == "tabs" ]]; then
        if [[ $tab_index -eq 0 ]]; then
          run_or_echo terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
        else
          run_or_echo terminator --new-tab -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
        fi
      else
        run_or_echo terminator -u -T "$titulo_tab" --working-directory="$cwd_trabajo" -x bash -lc "$bash_cmd"
      fi
      ;;
    tmux)
      if [[ "$LAUNCH_MODE" == "tabs" ]]; then
        if [[ $tab_index -eq 0 ]]; then
          run_or_echo tmux new-session -d -s "$TMUX_SESSION_NAME" -n "$titulo_tab" -c "$cwd_trabajo" bash -lc "$bash_cmd"
        else
          run_or_echo tmux new-window -t "$TMUX_SESSION_NAME:" -n "$titulo_tab" -c "$cwd_trabajo" bash -lc "$bash_cmd"
        fi
      else
        run_or_echo tmux new-session -d -s "${TMUX_SESSION_NAME}-$(printf '%02d' "$tab_index")" -n "$titulo_tab" -c "$cwd_trabajo" bash -lc "$bash_cmd"
      fi
      ;;
    custom)
      run_or_echo "$TERMINAL_LAUNCHER" "$LAUNCH_MODE" "$titulo_tab" "$cwd_trabajo" "$bash_cmd" "$tab_index"
      ;;
  esac
}

tmux_safe_name() {
  local raw="$1"
  local fallback="$2"
  local safe
  safe="$(printf '%s' "$raw" | tr ' ' '_' | tr -cd '[:alnum:]_.:-')"
  if [[ -z "$safe" ]]; then
    safe="$fallback"
  fi
  printf '%s' "$safe"
}

launch_tmux_tab() {
  local tab_index="$1"
  local titulo_tab="$2"
  local cwd_trabajo="$3"
  local bash_cmd="$4"
  local tmux_window
  tmux_window="$(tmux_safe_name "$titulo_tab" "agente-$(printf '%02d' "$tab_index")")"
  if [[ $tab_index -eq 0 ]]; then
    tmux new-session -d -s "$TMUX_SESSION_NAME" -n "$tmux_window" -c "$cwd_trabajo" bash -lc "$bash_cmd"
    TMUX_CREATED_SESSIONS+=("$TMUX_SESSION_NAME")
  else
    tmux new-window -t "$TMUX_SESSION_NAME:" -n "$tmux_window" -c "$cwd_trabajo" bash -lc "$bash_cmd"
  fi
}

if [[ "$MODE" == "apply" ]]; then
  terminal_backend_preflight
fi

declare -a LAUNCH_ROWS=()
if [[ "$MODE" == "apply" ]]; then
  mapfile -t LAUNCH_ROWS < <("$PREP_SCRIPT" --ejecutar "$PLAN_FILE")
else
  mapfile -t LAUNCH_ROWS < <("$PREP_SCRIPT" "$PLAN_FILE")
fi

tab_index=0
for row in "${LAUNCH_ROWS[@]}"; do
  IFS=$'\t' read -r agente ruta_proyecto cwd_trabajo conector titulo_tab bash_cmd <<<"$row"
  if [[ "$MODE" == "apply" ]]; then
    if [[ "$LAUNCH_MODE" == "tabs" ]]; then
      terminal_backend_launch_tab "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
    else
      terminal_backend_launch_window "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
      sleep 0.3
    fi
  else
    terminal_backend_show_dry_run "$tab_index" "$titulo_tab" "$cwd_trabajo" "$bash_cmd"
  fi
  tab_index=$((tab_index + 1))
done

if [[ "$MODE" == "apply" ]]; then
  echo "Launch completado con backend terminal: $TERMINAL_BACKEND."
  if [[ "$TERMINAL_BACKEND" == "tmux" ]]; then
    if [[ "$LAUNCH_MODE" == "tabs" ]]; then
      echo "Adjunta con: tmux attach -t $TMUX_SESSION_NAME"
      if [[ "$TMUX_ATTACH" == "1" && -t 0 && -t 1 ]]; then
        exec tmux attach -t "$TMUX_SESSION_NAME"
      fi
    elif [[ ${#TMUX_CREATED_SESSIONS[@]} -gt 0 ]]; then
      echo "Sesiones tmux creadas:"
      for session_name in "${TMUX_CREATED_SESSIONS[@]}"; do
        echo "  - $session_name"
      done
    fi
  fi
else
  echo "Modo seco completado. Usa --ejecutar para lanzar el backend terminal."
fi
