# OpenClaw Orquesta API

Plugin local para conectar OpenClaw con Orquesta sin segunda fuente de verdad.

## Contrato canónico

- API operativa: `http://127.0.0.1:16543/api/openclaw/operator`
- MCP HTTP: `http://127.0.0.1:16543/api/mcp`
- Notificaciones: `http://127.0.0.1:16543/api/notificaciones`

## Uso

1. Arranca el daemon oficial:
   - `./orquesta server start`
2. Verifica salud:
   - `./orquesta server doctor`
3. Usa el servidor MCP configurado en [`.mcp.json`](./.mcp.json).
4. Para smoke local:
   - [`scripts/smoke_openclaw_orquesta.sh`](./scripts/smoke_openclaw_orquesta.sh)

## Principios

- Siempre `server-first`.
- Nunca leer SQLite directa.
- OpenClaw consume estado vivo por API/MCP y actúa por herramientas canónicas del servidor.
- La vista operativa humana equivalente es [`/openclaw`](http://127.0.0.1:16543/openclaw).
