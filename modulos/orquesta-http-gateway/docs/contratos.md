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
- `DirectorStatsPage`;
- `AppSpec`;
- `AppDirector`;
- `DirectorStats`.

## Rutas estables

- `RouteNuevaAppV0`: `/nueva-app`;
- `RouteDirectorStatsPageV0`: `/director-stats`;
- `RouteAppSpecV0`: `/api/v0/apps/spec`;
- `RouteAppDirectorV0`: `/api/v0/apps/director`;
- `RouteDirectorStatsV0`: `/api/v0/director/stats`.

`RouteDirectorStatsPageV0` y `RouteDirectorStatsV0` son rutas separadas: la
primera apunta al handler web inyectado y la segunda al contrato REST que
transporta `DirectorRunStatsV0` completo. El gateway no inspecciona campos de
tareas, agentes, rework, replan, progreso, cierre ni `decision_context`.

## Invariantes

- el codigo productivo solo depende de libreria estandar;
- el gateway no importa paquetes de `cmd`, `db`, `codex`, `runtime`,
  `orquesta-web`, `orquesta-mcp` ni `orquesta-factory`;
- una ruta sin handler inyectado responde como no configurada.
