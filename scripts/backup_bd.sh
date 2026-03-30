#!/usr/bin/env bash
# Copyright (C) 2026 Alberto Avidad Fernandez - Oficina de Software Libre (OSL) - Diputacion de Granada
# Este programa es software libre: puede redistribuirlo y/o modificarlo bajo los terminos de la
# Licencia Publica General de GNU publicada por la Free Software Foundation, version 3.
# Este programa se distribuye con la esperanza de que sea util, pero SIN NINGUNA GARANTIA.
# Consulte la GNU General Public License para mas detalles: <https://www.gnu.org/licenses/>.
#
# Uso:
#   scripts/backup_bd.sh [--ruta <dir>] [--max-diarios <n>] [--max-semanales <n>] [--max-mensuales <n>]
#
# Descripcion:
#   Genera una copia de seguridad verificada del backend SQLite de Orquesta y
#   aplica la politica de retencion definida:
#   - Diarios:   7 ficheros (1 cada 24h)
#   - Semanales: 4 ficheros (1 cada lunes)
#   - Mensuales: 3 ficheros (1 cada dia 1 de mes)
#   La copia se almacena fuera del directorio de trabajo para protegerla de borrados accidentales.
#   Esta es una utilidad de rescate/automatizacion solo-SQLite.
#   El camino oficial interactivo desde Orquesta es: `./orquesta respaldo bd`
#   Este script es solo para SQLite. Para otros backends debe usarse un adaptador
#   de backup especifico del motor.
set -euo pipefail

# ── Configuracion por defecto ─────────────────────────────────────────────
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_SRC="${ORQUESTA_DB:-$ROOT_DIR/orquesta.db}"
BACKUP_DIR="${ORQUESTA_BACKUP_DIR:-$HOME/Trabajo/backups/orquestador}"
MAX_DIARIOS="${MAX_BACKUPS_DIARIOS:-7}"
MAX_SEMANALES="${MAX_BACKUPS_SEMANALES:-4}"
MAX_MENSUALES="${MAX_BACKUPS_MENSUALES:-3}"
VERIFY_SCRIPT="$ROOT_DIR/scripts/verificar_bd.sh"

resolve_driver() {
  local raw="${ORQUESTA_DB_DRIVER:-${ORQUESTA_DB_BACKEND:-sqlite}}"
  raw="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    ""|sqlite|sqlite3) printf 'sqlite\n' ;;
    *) printf '%s\n' "$raw" ;;
  esac
}

# ── Argumentos ────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --ruta)          BACKUP_DIR="$2"; shift 2 ;;
    --max-diarios)   MAX_DIARIOS="$2"; shift 2 ;;
    --max-semanales) MAX_SEMANALES="$2"; shift 2 ;;
    --max-mensuales) MAX_MENSUALES="$2"; shift 2 ;;
    -h|--help)
      sed -n '/^# Uso/,/^set /p' "$0" | grep '^#' | sed 's/^# \?//'
      exit 0 ;;
    *) echo "Argumento desconocido: $1" >&2; exit 1 ;;
  esac
done

DRIVER="$(resolve_driver)"
if [[ "$DRIVER" != "sqlite" ]]; then
  echo "[ERROR] scripts/backup_bd.sh solo cubre el backend sqlite; backend actual: $DRIVER" >&2
  echo "[ERROR] Usa un adaptador de backup especifico del motor o una ruta server-first dedicada." >&2
  exit 2
fi

# ── Validaciones ──────────────────────────────────────────────────────────
if [[ ! -f "$DB_SRC" ]]; then
  echo "[ERROR] No se encuentra la BD en: $DB_SRC" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

# ── Nombre del fichero de backup ──────────────────────────────────────────
TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
BACKUP_FILE="$BACKUP_DIR/${TIMESTAMP}_orquesta.db.bak"
BACKUP_TMP="$BACKUP_DIR/.${TIMESTAMP}_orquesta.db.bak.tmp"

cleanup_tmp() {
  rm -f "$BACKUP_TMP"
}
trap cleanup_tmp EXIT

# ── Copia segura y verificada ─────────────────────────────────────────────
# En WAL mode no es seguro copiar solo el fichero principal con cp/scp.
# Exigimos sqlite3 para generar un snapshot consistente y verificable.
if ! command -v sqlite3 &>/dev/null; then
  echo "[ERROR] sqlite3 es obligatorio para crear respaldos seguros de Orquesta." >&2
  exit 1
fi

rm -f "$BACKUP_TMP"
sqlite3 "$DB_SRC" ".timeout 15000" ".backup '$BACKUP_TMP'"
bash "$VERIFY_SCRIPT" --db "$BACKUP_TMP" --full >/dev/null
mv "$BACKUP_TMP" "$BACKUP_FILE"
trap - EXIT

echo "[OK] Backup creado y verificado: $BACKUP_FILE"

# ── Politica de retencion ─────────────────────────────────────────────────
# Clasificamos cada backup segun su timestamp en: diario, semanal, mensual.
# - Mensual : primer backup del mes (dia 01)
# - Semanal : primer backup del lunes de cada semana
# - Diario  : el resto
#
# Estrategia de purga: eliminar los mas antiguos que superen el maximo.

purgar_exceso() {
  local patron="$1"
  local max="$2"
  local ficheros=()
  # shellcheck disable=SC2012
  mapfile -t ficheros < <(ls -1t "$BACKUP_DIR"/$patron 2>/dev/null)
  local total="${#ficheros[@]}"
  if (( total > max )); then
    local a_borrar=$(( total - max ))
    for (( i = total - 1; i >= total - a_borrar; i-- )); do
      rm -f "${ficheros[$i]}"
      echo "[PURGADO] ${ficheros[$i]}"
    done
  fi
}

# Identificar el dia actual
DIA_MES=$(date +%d)
DIA_SEM=$(date +%u)  # 1=lunes ... 7=domingo

if [[ "$DIA_MES" == "01" ]]; then
  # Backup mensual: mover a prefijo mensual
  MENSUAL_FILE="$BACKUP_DIR/mensual_${TIMESTAMP}_orquesta.db.bak"
  mv "$BACKUP_FILE" "$MENSUAL_FILE"
  echo "[INFO] Backup mensual registrado: $MENSUAL_FILE"
  purgar_exceso "mensual_*_orquesta.db.bak" "$MAX_MENSUALES"
elif [[ "$DIA_SEM" == "1" ]]; then
  # Backup semanal (lunes)
  SEMANAL_FILE="$BACKUP_DIR/semanal_${TIMESTAMP}_orquesta.db.bak"
  mv "$BACKUP_FILE" "$SEMANAL_FILE"
  echo "[INFO] Backup semanal registrado: $SEMANAL_FILE"
  purgar_exceso "semanal_*_orquesta.db.bak" "$MAX_SEMANALES"
else
  # Backup diario normal
  purgar_exceso "${TIMESTAMP:0:10}*_orquesta.db.bak" 1
  purgar_exceso "????-??-??_*_orquesta.db.bak" "$MAX_DIARIOS"
fi

echo "[OK] Politica de retencion aplicada (diarios=$MAX_DIARIOS, semanales=$MAX_SEMANALES, mensuales=$MAX_MENSUALES)"
