#!/usr/bin/env bash
# Copyright (C) 2026 Alberto Avidad Fernandez - Oficina de Software Libre (OSL) - Diputacion de Granada
# Este programa es software libre: puede redistribuirlo y/o modificarlo bajo los terminos de la
# Licencia Publica General de GNU publicada por la Free Software Foundation, version 3.
# Este programa se distribuye con la esperanza de que sea util, pero SIN NINGUNA GARANTIA.
# Consulte la GNU General Public License para mas detalles: <https://www.gnu.org/licenses/>.
#
# Uso:
#   scripts/verificar_bd.sh [--db <ruta>] [--full] [--scan-tables]
#
# Descripcion:
#   Verifica la integridad de una base SQLite de Orquesta y avisa cuando
#   detecta ficheros WAL/SHM activos, situacion en la que copiar solo
#   orquesta.db con cp/scp no es una exportacion segura.
#   Esta es una utilidad de rescate solo-SQLite.
#   La verificacion operativa normal es: `./orquesta persistencia verificar`
#   Este script es solo para SQLite. Para otros backends debe usarse una
#   verificacion especifica del motor.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_PATH="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"
CHECK_KIND="quick_check"
SCAN_TABLES=0

resolve_driver() {
  local raw="${ORQUESTA_DB_DRIVER:-${ORQUESTA_DB_BACKEND:-sqlite}}"
  raw="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    ""|sqlite|sqlite3) printf 'sqlite\n' ;;
    *) printf '%s\n' "$raw" ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --db) DB_PATH="$2"; shift 2 ;;
    --full) CHECK_KIND="integrity_check"; shift ;;
    --scan-tables) SCAN_TABLES=1; shift ;;
    -h|--help)
      sed -n '/^# Uso/,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
      exit 0 ;;
    *) echo "Argumento desconocido: $1" >&2; exit 1 ;;
  esac
done

DRIVER="$(resolve_driver)"
if [[ "$DRIVER" != "sqlite" ]]; then
  echo "[ERROR] scripts/verificar_bd.sh solo cubre el backend sqlite; backend actual: $DRIVER" >&2
  echo "[ERROR] Usa una verificacion especifica del motor o una ruta server-first dedicada." >&2
  exit 2
fi

if [[ ! -f "$DB_PATH" ]]; then
  echo "[ERROR] No se encuentra la BD en: $DB_PATH" >&2
  exit 1
fi

if ! command -v sqlite3 &>/dev/null; then
  echo "[ERROR] sqlite3 es obligatorio para verificar la BD." >&2
  exit 1
fi

echo "[INFO] BD: $DB_PATH"
echo "[INFO] Tamano: $(stat -c '%s bytes' "$DB_PATH")"

if [[ -f "${DB_PATH}-wal" || -f "${DB_PATH}-shm" ]]; then
  echo "[WARN] Hay WAL/SHM activos junto a la BD."
  echo "[WARN] No copies solo $(basename "$DB_PATH") con cp/scp; usa .backup, VACUUM INTO o scripts/backup_bd.sh."
fi

JOURNAL_MODE="$(sqlite3 "$DB_PATH" 'PRAGMA journal_mode;' 2>/dev/null || true)"
PAGE_SIZE="$(sqlite3 "$DB_PATH" 'PRAGMA page_size;' 2>/dev/null || true)"
PAGE_COUNT="$(sqlite3 "$DB_PATH" 'PRAGMA page_count;' 2>/dev/null || true)"
FREELIST_COUNT="$(sqlite3 "$DB_PATH" 'PRAGMA freelist_count;' 2>/dev/null || true)"

[[ -n "$JOURNAL_MODE" ]] && echo "[INFO] journal_mode=$JOURNAL_MODE"
[[ -n "$PAGE_SIZE" ]] && echo "[INFO] page_size=$PAGE_SIZE"
[[ -n "$PAGE_COUNT" ]] && echo "[INFO] page_count=$PAGE_COUNT"
[[ -n "$FREELIST_COUNT" ]] && echo "[INFO] freelist_count=$FREELIST_COUNT"

CHECK_OUTPUT="$(sqlite3 "$DB_PATH" "PRAGMA $CHECK_KIND;" 2>&1)" || {
  echo "[ERROR] sqlite3 no pudo ejecutar PRAGMA $CHECK_KIND sobre $DB_PATH" >&2
  echo "$CHECK_OUTPUT" >&2
  exit 2
}

if [[ "$CHECK_OUTPUT" != "ok" ]]; then
  echo "[ERROR] PRAGMA $CHECK_KIND ha detectado corrupcion en $DB_PATH" >&2
  echo "$CHECK_OUTPUT" >&2
  exit 2
fi

echo "[OK] PRAGMA $CHECK_KIND: ok"

if (( SCAN_TABLES )); then
  TABLE_LIST="$(sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name;")"
  for table_name in $TABLE_LIST; do
    if count="$(sqlite3 "$DB_PATH" "SELECT count(*) FROM \"$table_name\";" 2>/dev/null)"; then
      echo "[OK] tabla=$table_name filas=$count"
      continue
    fi
    echo "[ERROR] No se puede leer la tabla: $table_name" >&2
    exit 3
  done
fi

echo "[OK] Verificacion completada"
