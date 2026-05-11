# Contratos: orquesta-app-gateway

## `NewHTTPHandlerV0`

Construye un `http.Handler` publico sin abrir servidor.

Rutas montadas:

- `/nueva-app`: handler web HTML.
- `/director-stats`: handler web JSON para panel de estadisticas.
- `/api/v0/apps/spec`: REST de factory.
- `/api/v0/apps/director`: bridge REST de MCP para arrancar director.
- `/api/v0/director/stats`: bridge REST de MCP para estadisticas; devuelve el
  `DirectorRunStatsV0` canonico dentro de `stats`, con tareas, agentes, rework,
  replan, progreso y cierre. Tambien devuelve `decision_context`
  `DirectorDecisionContextV0`, con progreso por fase/tarea/agente, procesos y
  sesiones opacas cuando se solicitan, actividad reciente, bloqueos, cierre,
  rework/replan, duraciones y quietud.

## `ConfigV0`

Entrada de composicion:

- `Clock`: reloj opcional para factory.
- `ArrancarDirector`: executor MCP inyectado.
- `DirectorStats`: executor MCP inyectado.
- `HTTPClient`: cliente opcional para que web llame a APIs REST.
- `Timeout`: timeout de clientes REST creados por defecto.

## Invariantes

- Web consume REST, no core.
- MCP recibe executors por puerto, no crea stores ni runtime.
- Factory recibe reloj por puerto, no configuracion global.
- Sin DB, runtime real, procesos, proveedor, HOME, OAuth, flags ni servidor real.
- `/director-stats` consulta `/api/v0/director/stats` por cliente REST
  in-process; el gateway no interpreta ni recorta `stats` ni
  `decision_context`.
