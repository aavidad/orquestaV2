# Runbook: conector operador hexagonal

Objetivo: permitir que una IA operadora invoque estado, burst supervisado,
outbox pendiente y consulta dirigida por MCP sin conocer internals de Orquesta.

## Frontera

- `orquesta-operator-mcp` define DTOs, puertos y un conector simulado local.
- `orquesta-mcp` registra resource/tools y delega en puertos inyectados.
- `orquesta-operator-mcp-client` implementa un conector externo opt-in sobre un
  cliente MCP generico, sin importar el transporte local.
- Hermes, OpenClaw u otro operador real deben vivir como adaptadores externos
  opt-in que implementen `OperatorMCPConnectorV0`.
- Hermes se consume por API/MCP. No usar CLI de proveedor, wrappers de proceso
  local ni rutas internas como sustituto de su conector.

## Uso

1. Descubrir `orquesta.operator.operations.v0`.
2. Invocar tools con refs opacas: `request_ref`, `run_ref`, `subject_ref` y
   `*_connector_ref`.
3. Si falta conector, tratar `operator_mcp_port_unavailable` como modo sin
   adaptador y pedir wiring al director.
4. Para pruebas offline, inyectar `NewOperatorMCPSimulatedConnectorV0` o un
   cliente MCP fake en `NewOperatorMCPClientConnectorV0`.
5. Para Hermes u OpenClaw, configurar nombres de tool y refs de conector en
   `OperatorMCPClientConfigV0`; Orquesta trata esas refs como opacas.

## Hermes API-only

- Usar el transporte MCP JSON-RPC opt-in en `/mcp` o la API publica versionada
  `/api/v0/*` cuando aplique.
- No usar `/api/mcp`, `/api/*` legacy, OpenClaw V1 ni CLI de proveedor para
  cerrar evidencias nuevas.
- Si faltan `HERMES_BASE_URL` o credenciales opt-in, reportar
  `operator_mcp_connector_unavailable` o dejar el smoke bloqueado; no leer DB,
  filesystem productivo ni outbox interno como alternativa.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp-client
```

El contrato no lee DB, outbox real, runtime, filesystem productivo, HOME,
OAuth, proveedor, modelo, prompts ni transcripts.
