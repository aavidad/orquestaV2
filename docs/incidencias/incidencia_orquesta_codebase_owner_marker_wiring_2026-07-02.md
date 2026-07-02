# Incidencia: codebase status sin wiring automatico de owner marker

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-CODEBASE-20260702-001`.

## Sintoma residual

`orquesta.codebase.status.v0` ya aceptaba `observations[]`, pero la composicion
del servidor no las derivaba automaticamente desde los owner markers file-based.
Un agente podia obtener un status correcto solo si aportaba manualmente la
observacion compacta; si llamaba al status publico sin esa observacion, un lease
expirado con CPU alta podia seguir apareciendo como `request_stop`.

## Cambio

- `codeContextBrokerWiringV0` conserva el observador file-based de owner markers
  cuando `ORQUESTA_CODEBASE_BROKER_STATE_DIR` esta configurado.
- `serverCodebaseStatusOwnerMarkerExecutorV0` envuelve el executor de status y
  anade observaciones de `active_requests`, CPU, heartbeat y evidencias desde el
  marker antes de delegar en el contrato MCP existente.
- `buildStackFromEnvWithGoalBackendV0` sustituye el binding
  `CodebaseStatus` por el wrapper enriquecido.
- `buildServerAppHandlerV0` publica la ruta exacta
  `/api/v0/codebase/status` con el binding enriquecido, de forma que el status
  HTTP publico no depende de que el agente envie `observations[]`.
- No se toca `modulos/orquesta-mcp`; la parada real sigue en watchdog/servidor.

## Evidencia

Tests focales:

```bash
go test -count=1 ./cmd/orquesta-server -run 'Test(ServerCodebaseStatusOwnerMarkerExecutor|BuildServerAppHandlerV0CodebaseStatusPublicoUsaOwnerMarkers|ServerCodeContextToolWatchdogV0UsaOwnerMarkerConPeticionesActivas)'
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPCodebaseStatus'
```

Bateria de servidor:

```bash
go test -count=1 ./cmd/orquesta-server
```

Casos nuevos:

- `TestServerCodebaseStatusOwnerMarkerExecutorV0InyectaPeticionesActivas`
- `TestBuildServerAppHandlerV0CodebaseStatusPublicoUsaOwnerMarkersFileBased`

## Cierre

El status publico ya incorpora owner markers file-based sin llamada MCP directa
de cada agente y publica `continue/active_requests` cuando hay peticiones
activas observadas.
