# Contratos: orquesta-app-gateway

## `NewHTTPHandlerV0`

Construye un `http.Handler` publico sin abrir servidor.

Rutas montadas:

- `/nueva-app`: handler web HTML.
- `/app-change`: handler web JSON/HTML de cambios sobre app existente.
- `/director-stats`: handler web JSON para panel de estadisticas.
- `/run-control`: handler web JSON para pausar, reanudar, parar o cancelar
  runs.
- `/run-queue`: handler web JSON para cola multiapp y cambio de prioridad.
- `/api/v0/apps/spec`: REST de factory.
- `/api/v0/apps/director`: bridge REST de MCP para arrancar director.
- `/api/v0/apps/{app_ref}/changes`: bridge REST de MCP para cambios de app.
- `/api/v0/director/stats`: bridge REST de MCP para estadisticas; devuelve el
  `DirectorRunStatsV0` canonico dentro de `stats`, con tareas, agentes, rework,
  replan, progreso y cierre. Tambien devuelve `decision_context`
  `DirectorDecisionContextV0`, con progreso por fase/tarea/agente, procesos y
  sesiones opacas cuando se solicitan, actividad reciente, bloqueos, cierre,
  rework/replan, duraciones y quietud.
- `/api/v0/runs/control`: bridge REST de MCP para pausa, reanudacion, cancelado
  y parada por `run_ref`.
- `/api/v0/runs/queue/priority`: bridge REST de MCP para ranking de cola global
  y cambio de prioridad.
- `/api/v0/runs/supervise`: bridge REST de MCP para pedir una pasada acotada
  de supervision sobre una run concreta o sobre la cola inyectada.
- `/api/v0/autoprogramming/validate-request`: bridge REST de validacion de
  peticiones de autoprogramacion.
- `/api/v0/autoprogramming/prepare-run`: bridge REST de MCP para preparar una
  run de autoprogramacion continuable por executor inyectado.
- `/api/v0/autoprogramming/status`: bridge REST de MCP para consultar estado de
  cola/run y diagnostico compacto usando puertos ya inyectados.
- `/api/v0/autoprogramming/supervise`: bridge REST de MCP para ejecutar una
  pasada puntual de supervision por el supervisor inyectado.
- `/api/v0/server/shutdown`: bridge REST de MCP para cierre controlado.
- `/api/v0/domain-work`: bridge REST de MCP para crear trabajo de dominio
  externo o entregar artefactos mediante un executor inyectado.
- `/api/v0/external-work/run`: bridge REST de MCP para crear un run operativo de
  trabajo externo ya definido.

## `ConfigV0`

Entrada de composicion:

- `Clock`: reloj opcional para factory.
- `ArrancarDirector`: executor MCP inyectado.
- `RequestAppChange`: executor MCP inyectado para cambios de app.
- `DirectorLimits`: limites web para arranque de director.
- `DirectorStats`: executor MCP inyectado.
- `RunControl`: executor MCP inyectado para control de runs.
- `RunQueuePriority`: executor MCP inyectado para cola multiapp.
- `RunSupervisor`: executor MCP inyectado para supervision acotada de runs.
- `AutoprogrammingPrepareRun`: executor MCP inyectado para preparar runs de
  autoprogramacion.
- Estado y supervision de autoprogramacion reutilizan `RunQueuePriority`,
  `DirectorStats` y `RunSupervisor`; este modulo no crea casos de uso nuevos.
- `ServerShutdown`: executor MCP inyectado para cierre controlado.
- `DomainWork`: executor MCP inyectado para trabajo de dominio externo.
- `ExternalWorkRun`: executor MCP inyectado para crear runs de trabajo externo.
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
- `/run-queue` consulta `/api/v0/runs/queue/priority` por cliente REST
  in-process; el gateway no interpreta ranking, score, aging ni estado de cola.
- `/run-control` consulta `/api/v0/runs/control` por cliente REST in-process;
  el gateway no interpreta estados, checkpoint, parada fisica ni runtime.
- `/api/v0/runs/supervise` delega en `orquesta-mcp`; este modulo no sabe si el
  executor usa Codex, trabajo de dominio u otra composicion.
- `/api/v0/autoprogramming/prepare-run` delega en `orquesta-mcp`; este modulo no
  conoce `PrepareAutoprogrammingRunV0`, stores, runtime ni Codex.
- `/api/v0/autoprogramming/status` y `/api/v0/autoprogramming/supervise`
  delegan en `orquesta-mcp`; este modulo no interpreta cola, stats, procesos ni
  scheduler.
- `/api/v0/domain-work` delega en `orquesta-mcp`; este modulo no interpreta
  `DomainWorkJobRequestV0`, no importa OPES y no crea conectores reales.
