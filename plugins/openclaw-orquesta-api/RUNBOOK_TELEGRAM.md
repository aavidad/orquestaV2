# Runbook Telegram para OpenClaw

## Objetivo

Usar Telegram como canal conversacional de OpenClaw sin romper la soberanía de Orquesta.

## Reglas

- Telegram no decide ni persiste estado del proyecto.
- El estado canónico sale de Orquesta:
  - `/api/openclaw/operator`
  - `/api/notificaciones`
  - `/api/mcp`
- Las acciones se ejecutan por MCP/API del servidor, no por scripts laterales.

## Preflight

1. `./orquesta server start`
2. `./orquesta server doctor`
3. `./plugins/openclaw-orquesta-api/scripts/smoke_openclaw_orquesta.sh`
4. Configura, si aplica:
   - `telegram_token`
   - `telegram_chat_id`
   - `openclaw_gateway_url`
   - `openclaw_gateway_token`

## Flujo recomendado

1. OpenClaw pide briefing del supervisor.
2. OpenClaw consulta cola segura y `queue_summary`.
3. Si la acción es segura, actúa por MCP.
4. Si requiere arbitraje, pide confirmación humana en Telegram.
5. Toda confirmación humana se traduce otra vez a acción MCP/API sobre Orquesta.

## Errores típicos

- `404` en MCP:
  - el endpoint correcto es `http://127.0.0.1:16543/api/mcp`
- `Health RPC` no OK:
  - reinicia con `./orquesta server stop && ./orquesta server start`
- Cola vacía pero guidance viva:
  - revisar `/api/openclaw/operator` y `status.mailboxPendiente`
