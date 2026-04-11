#!/usr/bin/env bash
# Software libre bajo licencia GNU GPL v3
# Proyecto: PlataformaMunicipal — Orquesta
# Autor: Alberto Avidad Fernandez (OSL - Diputacion de Granada)
#
# Bucle de vigilancia del agente antigravity (el "vigilante" del sistema).
# Este script se ejecuta como servicio systemd y se auto-reanuda si cae.
# Se encarga de:
#   1. Enviar heartbeats periódicos a Orquesta (agente tick)
#   2. Comprobar si hay runtime_orders pendientes para antigravity
#   3. Verificar el estado de los demás agentes y reportar anomalías
#
# Uso: bash scripts/vigilante.sh [proyecto]

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}") /.." && pwd)"
ORQUESTA="${ROOT_DIR}/orquesta"
PROYECTO="${1:-orquestador}"
AGENTE="antigravity"
INTERVALO=60  # segundos entre ciclos de vigilancia

if [[ ! -x "${ORQUESTA}" ]]; then
  echo "❌ No encuentro el binario: ${ORQUESTA}" >&2
  exit 1
fi

echo "🛡️  [Vigilante] Iniciando bucle de vigilancia para ${AGENTE} (proyecto: ${PROYECTO})"
echo "🛡️  [Vigilante] Intervalo de ciclo: ${INTERVALO}s"

# Registrar inicio de sesión
"${ORQUESTA}" sesion inicio "${AGENTE}" --proyecto "${PROYECTO}" --conector cli --cwd "${ROOT_DIR}" 2>/dev/null || true

cleanup() {
  echo "🛡️  [Vigilante] Señal de parada recibida. Cerrando sesión..."
  "${ORQUESTA}" sesion fin "${AGENTE}" 2>/dev/null || true
  exit 0
}
trap cleanup SIGTERM SIGINT

ciclo=0
while true; do
  ciclo=$((ciclo + 1))
  echo "🔄 [Vigilante] Ciclo #${ciclo} — $(date '+%H:%M:%S')"

  # 1. Heartbeat: actualizar estado operativo
  tick_output=$("${ORQUESTA}" agente tick "${AGENTE}" --proyecto "${PROYECTO}" 2>&1) || true
  echo "💓 [Vigilante] Tick: ${tick_output}"

  # 2. Comprobar estado de todos los agentes (resumen)
  estado=$("${ORQUESTA}" status 2>&1 | head -20) || true
  echo "📊 [Vigilante] Estado global:"
  echo "${estado}"

  # 3. Esperar hasta el próximo ciclo
  echo "💤 [Vigilante] Próximo ciclo en ${INTERVALO}s..."
  sleep "${INTERVALO}"
done
