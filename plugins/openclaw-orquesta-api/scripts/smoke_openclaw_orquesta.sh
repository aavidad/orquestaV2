#!/usr/bin/env bash
set -euo pipefail

base_url="${ORQUESTA_SERVER_URL:-http://127.0.0.1:16543}"

echo "[1/4] MCP endpoint"
curl -fsS "${base_url}/api/mcp" | jq '{name,endpoint,protocol_version}' >/dev/null

echo "[2/4] OpenClaw operator"
curl -fsS "${base_url}/api/openclaw/operator" | jq '{queue_summary,next_action,next_safe_action}' >/dev/null

echo "[3/4] Notificaciones"
curl -fsS "${base_url}/api/notificaciones" | jq '.canales' >/dev/null

echo "[4/4] Runtime status"
curl -fsS "${base_url}/api/status" | jq '{generado,agentesActivos,tareasPorEstado}' >/dev/null

echo "OK: OpenClaw/Orquesta server-first"
