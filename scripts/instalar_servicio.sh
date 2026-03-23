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
UNIT_VIGILANTE="orquesta-vigilante.service"
UNIT_TEMPLATE_ORQUESTA="$(dirname "$0")/orquesta.service"
UNIT_TEMPLATE_VIGILANTE="$(dirname "$0")/orquesta-vigilante.service"

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
render_unit "${UNIT_TEMPLATE_VIGILANTE}" "${UNIT_DIR}/${UNIT_VIGILANTE}"

echo "🔄 Recargando systemd..."
sudo systemctl daemon-reload

echo "✅ Habilitando e iniciando ${UNIT_ORQUESTA}..."
sudo systemctl enable "${UNIT_ORQUESTA}"
sudo systemctl restart "${UNIT_ORQUESTA}"

echo ""
echo "📊 Estado del servicio:"
sudo systemctl status "${UNIT_ORQUESTA}" --no-pager -l | head -20

echo ""
echo "✅ Orquesta instalada como servicio."
echo "   Ver logs:   journalctl -u ${UNIT_ORQUESTA} -f"
echo "   Vigilante:  sudo systemctl enable --now ${UNIT_VIGILANTE}"
echo "   Detener:    sudo systemctl stop ${UNIT_ORQUESTA}"
echo "   Deshabilitar: sudo systemctl disable ${UNIT_ORQUESTA}"
