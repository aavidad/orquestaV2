#!/usr/bin/env bash
# Software libre bajo licencia GNU GPL v3
# Proyecto: PlataformaMunicipal — Orquesta
# Autor: Alberto Avidad Fernandez (OSL - Diputacion de Granada)
#
# Instala Orquesta como servicio systemd para el usuario actual.
# Uso: bash scripts/instalar_servicio.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
USUARIO="${USER}"
UNIT_DIR="/etc/systemd/system"
UNIT_ORQUESTA="orquesta.service"
UNIT_BACKUP="orquesta-backup.service"
UNIT_BACKUP_TIMER="orquesta-backup.timer"
UNIT_TEMPLATE_ORQUESTA="$(dirname "$0")/orquesta.service"
UNIT_TEMPLATE_BACKUP="$(dirname "$0")/orquesta-backup.service"
UNIT_TEMPLATE_BACKUP_TIMER="$(dirname "$0")/orquesta-backup.timer"

if [[ ! -f "${ROOT_DIR}/orquesta" ]]; then
  echo "❌ No encuentro el binario '${ROOT_DIR}/orquesta'. Compila primero con 'go build ./...'." >&2
  exit 1
fi

render_unit() {
  local src="$1"
  local dest="$2"
  local escaped_root
  escaped_root="$(printf '%s' "${ROOT_DIR}" | sed 's/[\/&]/\\&/g')"
  sudo bash -c "sed -e 's/__USER__/${USUARIO}/g' -e 's/__ROOT_DIR__/${escaped_root}/g' '${src}' > '${dest}'"
  sudo chmod 644 "${dest}"
}

echo "📦 Instalando servicios systemd en ${UNIT_DIR}"
render_unit "${UNIT_TEMPLATE_ORQUESTA}" "${UNIT_DIR}/${UNIT_ORQUESTA}"
BACKEND_DRIVER="$("${ROOT_DIR}/orquesta" persistencia info 2>/dev/null | awk -F': ' '/^Storage driver:/ {print $2; exit}')"
BACKUP_SUPPORT="$("${ROOT_DIR}/orquesta" persistencia info 2>/dev/null | awk -F': ' '/^Backup support:/ {print $2; exit}')"
if [[ "${BACKUP_SUPPORT}" == "true" ]]; then
  render_unit "${UNIT_TEMPLATE_BACKUP}" "${UNIT_DIR}/${UNIT_BACKUP}"
  render_unit "${UNIT_TEMPLATE_BACKUP_TIMER}" "${UNIT_DIR}/${UNIT_BACKUP_TIMER}"
fi

echo "🔄 Recargando systemd..."
sudo systemctl daemon-reload

echo "✅ Habilitando e iniciando ${UNIT_ORQUESTA}..."
sudo systemctl enable "${UNIT_ORQUESTA}"
sudo systemctl restart "${UNIT_ORQUESTA}"

if [[ "${BACKUP_SUPPORT}" == "true" ]]; then
  echo "🕒 Habilitando temporizador de respaldo..."
  sudo systemctl enable --now "${UNIT_BACKUP_TIMER}"
else
  echo "⚠️  No se instala temporizador de backup para backend '${BACKEND_DRIVER}' porque el conector de persistencia no declara soporte de backup."
fi

echo ""
echo "📊 Estado del servicio:"
sudo systemctl status "${UNIT_ORQUESTA}" --no-pager -l | head -20

echo ""
echo "✅ Orquesta instalada como servicio."
echo "   Ver logs:   journalctl -u ${UNIT_ORQUESTA} -f"
echo "   Health:     curl -fsS http://127.0.0.1:16543/api/status"
if [[ "${BACKUP_SUPPORT}" == "true" ]]; then
  echo "   Backup diario: sudo systemctl status ${UNIT_BACKUP_TIMER}"
else
  echo "   Backup diario: no configurado por este instalador para backend ${BACKEND_DRIVER}"
fi
echo "   Detener:    sudo systemctl stop ${UNIT_ORQUESTA}"
echo "   Deshabilitar: sudo systemctl disable ${UNIT_ORQUESTA}"
