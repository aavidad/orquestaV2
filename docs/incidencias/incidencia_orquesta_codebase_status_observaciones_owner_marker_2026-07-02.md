# Incidencia: codebase status sin observaciones de owner marker

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-CODEBASE-20260702-001`.

## Sintoma

`orquesta.codebase.status.v0` y `POST /api/v0/codebase/status` listaban leases y
evaluaban TTL/CPU, pero no aceptaban observaciones compactas del owner marker.
Un lease expirado podia publicarse como `request_stop` aunque el marcador
externo hubiese observado `active_requests > 0`.

## Riesgo

El status publico podia recomendar parada cooperativa de un proceso de
`codebase-memory-mcp` que seguia atendiendo una consulta valida del broker
central. La parada real sigue fuera de MCP, pero la proyeccion incorrecta podia
alimentar watchdogs u operadores con una decision demasiado agresiva.

## Cambio

- El input de `orquesta.codebase.status.v0` acepta `observations[]` compactas
  con `lease_ref`, `observed_at`, `cpu_percent`, `active_requests`,
  `cpu_high_percent` y `evidence_refs`.
- El endpoint HTTP transporta esas observaciones al contrato neutral
  `CodeContextToolingStatusRequestV0`.
- Si una observacion indica peticiones activas, el status publica
  `decision=continue`, `reason_code=active_requests` y no incluye
  `stop_expired_code_context_tool_lease`.
- MCP sigue sin arrancar ni parar procesos; solo evalua y publica estado.

## Evidencia

Test focal:

```bash
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPCodebaseStatus'
```

Caso nuevo:

- `TestMCPCodebaseStatusHTTPHandlerV0ObservacionesEvitanParadaConPeticionesActivas`

## Residual

El wiring automatico desde owner markers file-based al status HTTP sigue en la
composicion/servidor. Este cierre garantiza que el endpoint publico ya puede
recibir esas observaciones sin inducir paradas falsas.
