#!/usr/bin/env bash
# Reacredita SQLite V16 -> V19 sobre una copia privada y unidades efímeras.

set -euo pipefail
umask 077
PATH=/usr/bin:/bin
export PATH

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
readonly SCRIPT_DIR
readonly DRIVER="$SCRIPT_DIR/lib/orquesta_sqlite_v23_reaccredit.py"

if [ ! -f "$DRIVER" ] || [ -L "$DRIVER" ]; then
  printf '%s\n' \
    'orquesta_sqlite_v23_reaccredit: status=error reason_code=driver_invalid' >&2
  exit 1
fi

exec /usr/bin/python3 "$DRIVER" "$@"
