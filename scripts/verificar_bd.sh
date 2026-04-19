#!/usr/bin/env bash
# Copyright (C) 2026 Alberto Avidad Fernandez - Oficina de Software Libre (OSL) - Diputacion de Granada
# Este programa es software libre: puede redistribuirlo y/o modificarlo bajo los terminos de la
# Licencia Publica General de GNU publicada por la Free Software Foundation, version 3.
# Este programa se distribuye con la esperanza de que sea util, pero SIN NINGUNA GARANTIA.
# Consulte la GNU General Public License para mas detalles: <https://www.gnu.org/licenses/>.
#
# Uso:
#   scripts/verificar_bd.sh
#
# Descripcion:
#   Wrapper canonico de verificacion de persistencia.
#   No inspecciona SQLite/Postgres de forma directa; delega en Orquesta:
#     ./orquesta persistencia verificar
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ $# -gt 0 ]]; then
  case "${1:-}" in
    -h|--help)
      sed -n '/^# Uso/,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
      exit 0 ;;
    *)
      echo "[WARN] Este wrapper ya no acepta inspeccion directa por fichero ni flags de motor; se delega en Orquesta." >&2 ;;
  esac
fi

exec "$ROOT_DIR/orquesta" persistencia verificar
