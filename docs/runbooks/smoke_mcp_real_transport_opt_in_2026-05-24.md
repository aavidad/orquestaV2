# Smoke MCP real opt-in - 2026-05-24

## Alcance

Smoke temporal del transporte MCP real de composicion para herramientas de
operador y automejora. El transporte vive en `cmd/orquesta-server`, envuelve el
registro puro de `orquesta-mcp` y habla JSON-RPC por HTTP en loopback. No se
activa en `run` por defecto.

## Guardas

- Requiere confirmacion explicita: `ORQUESTA_MCP_REAL_SMOKE_CONFIRM=1`.
- Usa solo `127.0.0.1:0` y un servidor temporal.
- No lee stores, runtime, Git/worktrees, HOME, DB, OPES, Codex, proveedor,
  secretos ni paths locales.
- Si falta operador real, el error esperado es publico:
  `operator_mcp_port_unavailable`.

## Comando

```bash
ORQUESTA_MCP_REAL_SMOKE_CONFIRM=1 \
GOCACHE=/tmp/orquesta-go-build-cache \
go run ./cmd/orquesta-server mcp-real-smoke
```

## Criterio de exito

La salida JSON debe incluir:

```json
{
  "schema_version": "mcp_real_transport_smoke.v0",
  "status": "completed",
  "resource_read": true,
  "self_improvement": true,
  "operator_error_code": "operator_mcp_port_unavailable"
}
```

El smoke prueba una lectura de `orquesta.operator.operations.v0`, una propuesta
no destructiva de `orquesta.autoprogramming.self_improvement.propose.v0` y un
error de operador normalizado.

## Regresion automatica

```bash
GOCACHE=/tmp/orquesta-go-build-cache \
go test -count=1 ./cmd/orquesta-server -run TestMCPRealTransportSmokeOptInV0
```
