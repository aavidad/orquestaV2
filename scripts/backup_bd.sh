#!/usr/bin/env bash
# Copyright (C) 2026 Alberto Avidad Fernandez - Oficina de Software Libre (OSL) - Diputacion de Granada
# Este programa es software libre: puede redistribuirlo y/o modificarlo bajo los terminos de la
# Licencia Publica General de GNU publicada por la Free Software Foundation, version 3.
# Este programa se distribuye con la esperanza de que sea util, pero SIN NINGUNA GARANTIA.
# Consulte la GNU General Public License para mas detalles: <https://www.gnu.org/licenses/>.
#
# Uso:
#   scripts/backup_bd.sh [--ruta <dir>] [--retener <n>] [--etiqueta <txt>]
#
# Descripcion:
#   Wrapper server-first del backup de persistencia de Orquesta.
#   No asume SQLite ni accede directamente al motor. Delegacion canonica:
#     ./orquesta respaldo bd
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DESTINO=""
RETENER=0
ETIQUETA=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ruta|--destino) DESTINO="$2"; shift 2 ;;
    --retener) RETENER="$2"; shift 2 ;;
    --etiqueta) ETIQUETA="$2"; shift 2 ;;
    --max-diarios|--max-semanales|--max-mensuales)
      echo "[WARN] $1 ya no se gestiona en este wrapper; usa --retener o una politica externa." >&2
      shift 2 ;;
    -h|--help)
      sed -n '/^# Uso/,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
      exit 0 ;;
    *)
      echo "Argumento desconocido: $1" >&2
      exit 1 ;;
  esac
done

CMD=("$ROOT_DIR/orquesta" "respaldo" "bd")
if [[ -n "$DESTINO" ]]; then
  CMD+=("--destino" "$DESTINO")
fi
if [[ "$RETENER" -gt 0 ]]; then
  CMD+=("--retener" "$RETENER")
fi
if [[ -n "$ETIQUETA" ]]; then
  CMD+=("--etiqueta" "$ETIQUETA")
fi

exec "${CMD[@]}"
