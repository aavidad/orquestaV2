#!/usr/bin/env bash
# Software libre bajo licencia GNU GPL v3
# Proyecto: PlataformaMunicipal — Orquesta
# Autor: Alberto Avidad Fernandez (OSL - Diputacion de Granada)
#
# Instala Orquesta como servicio systemd para el usuario actual.
# Uso: bash scripts/instalar_servicio.sh

set -euo pipefail

USUARIO="${USER}"
UNIT="orquesta@${USUARIO}.service"
UNIT_TEMPLATE_SRC="$(dirname "$0")/orquesta.service"
UNIT_DEST="/etc/systemd/system/${UNIT}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ ! -f "${ROOT_DIR}/orquesta" ]]; then
  echo "❌ No encuentro el binario '${ROOT_DIR}/orquesta'. Compila primero con 'go build ./...'." >&2
  exit 1
fi

echo "📦 Instalando servicio systemd: ${UNIT}"

# Copiar con interpolación del usuario
sudo bash -c "sed 's/%i/${USUARIO}/g' '${UNIT_TEMPLATE_SRC}' > '${UNIT_DEST}'"
sudo chmod 644 "${UNIT_DEST}"

echo "🔄 Recargando systemd..."
sudo systemctl daemon-reload

echo "✅ Habilitando e iniciando ${UNIT}..."
sudo systemctl enable "${UNIT}"
sudo systemctl restart "${UNIT}"

echo ""
echo "📊 Estado del servicio:"
sudo systemctl status "${UNIT}" --no-pager -l | head -20

echo ""
echo "✅ Orquesta instalada como servicio."
echo "   Ver logs:   journalctl -u ${UNIT} -f"
echo "   Detener:    sudo systemctl stop ${UNIT}"
echo "   Deshabilitar: sudo systemctl disable ${UNIT}"
