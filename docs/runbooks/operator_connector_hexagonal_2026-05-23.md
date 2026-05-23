# Runbook: conector operador hexagonal

Objetivo: permitir que una IA operadora invoque estado, burst supervisado,
outbox pendiente y consulta dirigida por MCP sin conocer internals de Orquesta.

## Frontera

- `orquesta-operator-mcp` define DTOs, puertos y un conector simulado local.
- `orquesta-mcp` registra resource/tools y delega en puertos inyectados.
- Hermes, OpenClaw u otro operador real deben vivir como adaptadores externos
  opt-in que implementen `OperatorMCPConnectorV0`.

## Uso

1. Descubrir `orquesta.operator.operations.v0`.
2. Invocar tools con refs opacas: `request_ref`, `run_ref`, `subject_ref` y
   `*_connector_ref`.
3. Si falta conector, tratar `operator_mcp_port_unavailable` como modo sin
   adaptador y pedir wiring al director.
4. Para pruebas offline, inyectar `NewOperatorMCPSimulatedConnectorV0`.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-operator-mcp ./modulos/orquesta-mcp
```

El contrato no lee DB, outbox real, runtime, filesystem productivo, HOME,
OAuth, proveedor, modelo, prompts ni transcripts.
