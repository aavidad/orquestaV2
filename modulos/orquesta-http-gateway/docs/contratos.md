# Contratos: orquesta-http-gateway

## `NewAppGatewayMuxV0`

Constructor publico:

```go
func NewAppGatewayMuxV0(handlers RouteHandlersV0) http.Handler
```

Reglas:

- devuelve siempre un `http.Handler`;
- registra una ruta solo si el handler correspondiente no es `nil`;
- no envuelve ni modifica la request;
- no restringe metodo HTTP;
- no inyecta dependencias de negocio.

## `RouteHandlersV0`

Contenedor de handlers `net/http` ya construidos por otro borde de composicion.

Campos:

- `NuevaApp`;
- `AppChangePage`;
- `DirectorStatsPage`;
- `RunControlPage`;
- `RunQueuePage`;
- `AppSpec`;
- `AppDirector`;
- `AppChange`;
- `DirectorStats`;
- `RunControl`;
- `RunQueuePriority`;
- `ServerShutdown`;
- `AutoprogrammingValidateRequest`.
- `DomainWork`.
- `ExternalWorkRun`.

## Rutas estables

- `RouteNuevaAppV0`: `/nueva-app`;
- `RouteAppChangePageV0`: `/app-change`;
- `RouteDirectorStatsPageV0`: `/director-stats`;
- `RouteRunControlPageV0`: `/run-control`;
- `RouteRunQueuePageV0`: `/run-queue`;
- `RouteAppSpecV0`: `/api/v0/apps/spec`;
- `RouteAppDirectorV0`: `/api/v0/apps/director`;
- `RouteAppChangeV0`: `/api/v0/apps/`;
- `RouteDirectorStatsV0`: `/api/v0/director/stats`.
- `RouteRunControlV0`: `/api/v0/runs/control`;
- `RouteRunQueuePriorityV0`: `/api/v0/runs/queue/priority`;
- `RouteAutoprogrammingValidateRequestV0`: `/api/v0/autoprogramming/validate-request`;
- `RouteServerShutdownV0`: `/api/v0/server/shutdown`.
- `RouteDomainWorkV0`: `/api/v0/domain-work`.
- `RouteExternalWorkRunV0`: `/api/v0/external-work/run`.

`RouteDirectorStatsPageV0` y `RouteDirectorStatsV0` son rutas separadas: la
primera apunta al handler web inyectado y la segunda al contrato REST que
transporta `DirectorRunStatsV0` completo. El gateway no inspecciona campos de
tareas, agentes, rework, replan, progreso, cierre ni `decision_context`.

`RouteRunQueuePageV0` y `RouteRunQueuePriorityV0` son rutas separadas: la
primera apunta al panel web inyectado y la segunda al contrato REST/MCP de
cola. El gateway no conoce ranking, prioridad, aging, fairness ni stores.

`RouteRunControlPageV0` y `RouteRunControlV0` son rutas separadas: la primera
apunta al panel web inyectado y la segunda al contrato REST/MCP de control. El
gateway no conoce pausa, parada, checkpoint, procesos ni runtime.

`RouteDomainWorkV0` apunta al contrato REST/MCP de trabajo de dominio externo.
El gateway solo registra el handler inyectado; no conoce OPES, contratos de
dominio, conectores REST, DB, runtime ni proveedores.

`RouteExternalWorkRunV0` apunta al contrato REST/MCP que crea un run operativo
para un trabajo externo ya definido. El gateway no crea runs, no abre fases y no
encola por si mismo; solo monta el handler inyectado.

## Invariantes

- el codigo productivo solo depende de libreria estandar;
- el gateway no importa paquetes de `cmd`, `db`, `codex`, `runtime`,
  `orquesta-web`, `orquesta-mcp` ni `orquesta-factory`;
- una ruta sin handler inyectado responde como no configurada.
