#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Uso:
  scripts/custom_terminal_launcher.sh <window|tab> <titulo> <cwd> <bash_cmd> [index]

Contrato del launcher custom para scripts/launch_agentes.sh.
Debe abrir una consola con:
  - título sugerido
  - directorio de trabajo
  - comando bash -lc ya construido

Backends intentados, por orden:
  1. gnome-terminal
  2. xfce4-terminal
  3. x-terminal-emulator
EOF
}

if [[ $# -lt 4 ]]; then
  usage
  exit 1
fi

MODE="$1"
TITLE="$2"
CWD_TRABAJO="$3"
BASH_CMD="$4"
INDEX="${5:-0}"

if [[ ! -d "$CWD_TRABAJO" ]]; then
  echo "No existe el directorio de trabajo: $CWD_TRABAJO" >&2
  exit 1
fi

if [[ -z "${DISPLAY:-}" && -z "${WAYLAND_DISPLAY:-}" ]]; then
  echo "No detecto DISPLAY ni WAYLAND_DISPLAY; no puedo abrir un terminal gráfico." >&2
  exit 1
fi

launch_gnome_terminal() {
  local mode="$1"
  local title="$2"
  local cwd="$3"
  local bash_cmd="$4"

  if [[ "$mode" == "tab" ]]; then
    gnome-terminal --tab --title="$title" --working-directory="$cwd" -- bash -lc "$bash_cmd"
  else
    gnome-terminal --title="$title" --working-directory="$cwd" -- bash -lc "$bash_cmd"
  fi
}

launch_xfce4_terminal() {
  local mode="$1"
  local title="$2"
  local cwd="$3"
  local bash_cmd="$4"

  if [[ "$mode" == "tab" ]]; then
    xfce4-terminal --tab --title="$title" --working-directory="$cwd" --command "bash -lc $(printf '%q' "$bash_cmd")"
  else
    xfce4-terminal --title="$title" --working-directory="$cwd" --command "bash -lc $(printf '%q' "$bash_cmd")"
  fi
}

launch_x_terminal_emulator() {
  local _mode="$1"
  local title="$2"
  local cwd="$3"
  local bash_cmd="$4"

  x-terminal-emulator -T "$title" -e bash -lc "cd $(printf '%q' "$cwd") && $bash_cmd"
}

if command -v gnome-terminal >/dev/null 2>&1; then
  launch_gnome_terminal "$MODE" "$TITLE" "$CWD_TRABAJO" "$BASH_CMD"
  exit 0
fi

if command -v xfce4-terminal >/dev/null 2>&1; then
  launch_xfce4_terminal "$MODE" "$TITLE" "$CWD_TRABAJO" "$BASH_CMD"
  exit 0
fi

if command -v x-terminal-emulator >/dev/null 2>&1; then
  if [[ "$MODE" == "tab" && "$INDEX" != "0" ]]; then
    echo "El backend x-terminal-emulator no soporta pestañas; abro una ventana nueva." >&2
  fi
  launch_x_terminal_emulator "$MODE" "$TITLE" "$CWD_TRABAJO" "$BASH_CMD"
  exit 0
fi

echo "No encuentro un terminal gráfico compatible para el launcher custom." >&2
exit 1
