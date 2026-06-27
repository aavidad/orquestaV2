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
- `/api/v0/apps/director/goal/observe`: bridge REST de MCP para observar un
  goal-first ya lanzado por `run_ref`.
- `/api/v0/apps/intake/guided-turn`: endpoint JSON puro de intake guiado de
  nueva app; calcula decisiones/followups de wizard y sesion parcial sin
  persistir estado ni arrancar trabajo.
- `/api/v0/apps/vcs`: overlay REST de MCP para AppVCS; se monta como ruta
  exacta antes del fallback `/api/v0/apps/{app_ref}/changes`.
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
  de supervision sobre una run concreta o sobre la cola inyectada. Solo aplica
  a runs legacy/resident sin `GoalWorkStateV0`; si existe estado Goal, la accion
  segura es `observe_goal`.
- `/api/v0/autoprogramming/validate-request`: bridge REST de validacion de
  peticiones de autoprogramacion.
- `/api/v0/autoprogramming/self-improvement`: bridge REST de automejora de baja
  prioridad desde evidencia de fallo y refs opacas.
- `/api/v0/autoprogramming/prepare-run`: bridge REST de MCP para preparar una
  run de autoprogramacion continuable por executor inyectado.
- `/api/v0/autoprogramming/status`: bridge REST de MCP para consultar estado de
  cola/run y diagnostico compacto usando puertos ya inyectados.
- `/api/v0/autoprogramming/supervise`: bridge REST de MCP para ejecutar una
  pasada puntual de supervision por el supervisor inyectado.
- `/api/v0/governance/catalog/query`: REST read-only de gobernanza con request
  `{request_id, correlation_id, filters}` y respuesta `{request_id,
  correlation_id, effective, counters}`. Si falta provider, devuelve error
  publico recuperable `governance_catalog_source_unavailable`.
- `/api/v0/server/shutdown`: bridge REST de MCP para cierre controlado.
- `/api/v0/domain-work`: bridge REST de MCP para crear trabajo de dominio
  externo o entregar artefactos mediante un executor inyectado.
- `/api/v0/external-work/run`: bridge REST de MCP para crear un run operativo de
  trabajo externo ya definido.

## `ConfigV0`

Entrada de composicion:

- `Clock`: reloj opcional para factory.
- `ArrancarDirector`: executor MCP inyectado.
- `ObserveDirectorGoal`: executor MCP inyectado para observar goal-first de
  nueva app.
- `RequestAppChange`: executor MCP inyectado para cambios de app.
- `AppVCS`: executor MCP inyectado para preparar, revisar, commitear o publicar
  repos por refs opacas; nil deja el overlay sin montar.
- `DirectorLimits`: limites web para arranque de director.
- `DirectorStats`: executor MCP inyectado.
- `RunControl`: executor MCP inyectado para control de runs.
- `RunQueuePriority`: executor MCP inyectado para cola multiapp.
- `RunSupervisor`: executor MCP inyectado para supervision acotada de runs.
- `AutoprogrammingPrepareRun`: executor MCP inyectado para preparar runs de
  autoprogramacion.
- Estado y supervision de autoprogramacion reutilizan `RunQueuePriority`,
  `DirectorStats` y `RunSupervisor`; este modulo no crea casos de uso nuevos.
- `GovernanceCatalog`: provider inyectado para consulta read-only del catalogo
  publico; nil conserva ruta con error publico, sin fallback documental.
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
- `/api/v0/apps/vcs` debe preceder al prefijo `/api/v0/apps/` y no llegar al
  handler catch-all de app-change.
- `/api/v0/apps/intake/guided-turn` debe preceder al prefijo `/api/v0/apps/`,
  no llegar al handler catch-all de app-change y no conocer Director, runtime,
  DB, proveedores ni LLM real.
- `/api/v0/apps/director/goal/observe` debe preceder al prefijo
  `/api/v0/apps/` y delega en `orquesta-mcp`; este modulo no interpreta
  `GoalWorkResultV0`, no valida cierre, no toca cola y no conoce runtime goal,
  Codex, DB, filesystem ni proveedor.
- `/api/v0/runs/supervise` delega en `orquesta-mcp`; este modulo no sabe si el
  executor usa Codex, trabajo de dominio u otra composicion.
- `/api/v0/autoprogramming/prepare-run` delega en `orquesta-mcp`; este modulo no
  conoce `PrepareAutoprogrammingRunV0`, stores, runtime ni Codex, y conserva
  `goal?` y `goal_specs?` si el executor los devuelve.
- `/api/v0/autoprogramming/goal/observe` delega en `orquesta-mcp`; este modulo
  no observa goals, no valida cierre, no toca cola y no conoce runtime goal,
  Codex, DB, filesystem ni proveedor.
- `/api/v0/autoprogramming/goals/observe-active` delega en `orquesta-mcp`;
  este modulo no lista estados goal, no ejecuta supervision legacy y no decide
  cierre.
- `/api/v0/autoprogramming/self-improvement` delega en `orquesta-mcp`; este
  modulo no decide prioridad, cola, runtime ni preparacion salvo puerto
  inyectado.
- `/api/v0/autoprogramming/status` y `/api/v0/autoprogramming/supervise`
  delegan en `orquesta-mcp`; este modulo no interpreta cola, stats, procesos ni
  scheduler. Las lecturas lentas de status se resuelven en el handler MCP con
  JSON publico de timeout; el gateway no deja la conexion indefinida ni crea
  un flujo paralelo.
- `/api/v0/governance/catalog/query` delega en `orquesta-governance`; este
  modulo no activa historicos, no lee DB v1, no muta permisos y no inventa
  catalogos si falta provider.
- `/api/v0/domain-work` delega en `orquesta-mcp`; este modulo no interpreta
  `DomainWorkJobRequestV0`, no importa OPES y no crea conectores reales.
