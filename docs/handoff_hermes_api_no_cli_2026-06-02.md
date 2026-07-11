# Handoff: Hermes API/MCP, no CLI - 2026-06-02

## Estado

Correccion vigente: Hermes debe entrar como conector externo por API/MCP. No es
un proveedor CLI y no debe confundirse con el adaptador Gemini CLI opt-in.

La revision externa se limito a superficies publicas:

- `GET http://127.0.0.1:16543/api/v0/server/status`
- `POST http://127.0.0.1:16543/mcp` con JSON-RPC `resources/list`
- variables de entorno presentes o ausentes, sin imprimir secretos

Resultado: no hay servidor local escuchando en `127.0.0.1:16543` y no existen
`ORQUESTA_HERMES_BASE_URL`, `ORQUESTA_HERMES_API_KEY`, `GEMINI_API_KEY` ni `GOOGLE_API_KEY` en el
entorno de esta sesion. Por tanto no se puede cerrar un smoke real API-only de
Hermes ni de Gemini API sin inventar accesos, arrancar caminos por CLI o tocar
persistencia interna.

Actualizacion posterior del mismo dia: se implemento
`modulos/orquesta-operator-mcp-hermes` y el wiring opt-in `ORQUESTA_HERMES_*` en
`cmd/orquesta-server`. La prueba real externa sigue pendiente por falta de
endpoint/token reales.

## Frontera confirmada

- `modulos/orquesta-operator-mcp` define los contratos puros de operador:
  `OperatorMCPConnectorV0`, estado, burst supervisado, outbox y consulta
  dirigida.
- `modulos/orquesta-operator-mcp-client` ya implementa un cliente MCP generico
  opt-in sobre esos contratos. Sirve como base para Hermes si Hermes expone
  tools MCP compatibles.
- Hermes, OpenClaw u otros operadores reales deben vivir fuera del nucleo como
  adaptadores externos. No deben importar core privado ni leer DB, outbox,
  filesystem productivo, HOME, OAuth, proveedor/modelo ni rutas locales.
- El transporte real se consume por refs opacas, timeouts, cancelacion y errores
  publicos. La falta de conector se reporta como
  `operator_mcp_connector_unavailable`, no como fallback a internals.

## Hallazgos

- Ya existe un adaptador concreto `orquesta-operator-mcp-hermes`. Desde
  2026-07-11 la configuracion canonica es `hermes_operator.*` en
  `orquesta.config.json`; `ORQUESTA_HERMES_*` queda como override deprecated.
- El paquete `modulos/orquesta-runtime-gemini` creado para infografias es CLI
  headless. Puede quedar como proveedor Gemini CLI opt-in, pero no representa a
  Hermes.
- Si la decision de producto es "Gemini tambien por API", hace falta otro
  adaptador de Gemini API o refactor de transporte; no debe renombrarse el CLI
  como Hermes.
- Un smoke real de operador debe usar API publica: status, MCP `resources/list`,
  `tools/list` y `tools/call`, con endpoint/credenciales opt-in y sin acceso a
  bases de datos.

## Tareas

- `HERMES-API-001`: hecho offline. Crear adaptador externo Hermes sobre
  `OperatorMCPConnectorV0`. Entrada: `ORQUESTA_HERMES_BASE_URL`, token opt-in, nombres de
  tools, refs opacas, timeout y context factory. Salida: errores publicos
  estables y payload compacto.
- `HERMES-API-002`: hecho offline. Cablear configuracion opt-in en composicion/servidor sin
  tocar core. La configuracion efectiva debe aparecer redacted y canonica.
- `HERMES-API-003`: anadir smoke API-only con instancia temporal de Hermes:
  `resources/list`, `tools/list`, `status`, `pending_outbox`,
  `supervised_burst` acotado y `directed_query`. No usar CLI ni DB.
- `HERMES-API-004`: documentar runbook de operador externo para que Hermes
  actue como vigilante: descubrir capacidades, llamar tools con refs opacas,
  reportar fallos y abrir tareas sin forzar acceso.
- `GEMINI-API-001`: si Gemini debe abandonar CLI, implementar proveedor Gemini
  API separado del runtime CLI actual y rutearlo por rol/capacidad desde
  composicion, no desde OPES.

## Verificacion pendiente

Cuando existan endpoint y credenciales:

```bash
curl -fsS "$ORQUESTA_API_BASE/api/v0/server/status"
curl -fsS "$ORQUESTA_API_BASE/mcp" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":"hermes-api-smoke","method":"resources/list","params":{}}'
```

El smoke completo solo queda cerrado si las llamadas anteriores pasan por API y
las llamadas de tool devuelven refs/evidencia publica sin detalles internos.
