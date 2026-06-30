# orquesta-mcp

Responsabilidad: superficie MCP para que una IA gobierne Orquesta.

Incluye:

- resources;
- prompts;
- tools;
- proyecciones compactas;
- validacion de entrada/salida.

MCP es adaptador inbound. No es el cerebro del sistema.

Estado vigente:

- `orquesta.autoprogramming.prepare_run.v0` prepara, por executor inyectado, una
  salida legacy con `run_ref` continuable o un handoff goal-first con
  `goal_spec_summaries[]` sin run legacy. Goal-first es la ruta preferente
  cuando el executor devuelve `goal` o resumenes de specs; `continue` queda como
  compatibilidad legacy. Preparar una salida legacy requiere opt-in de
  composicion y `director_execution_mode=legacy_director_loop`. No arranca
  agentes por si mismo; la supervision con `run_ref` aplica solo a la rama
  legacy. La composicion puede conservar `GoalWorkSpecV0` como handoff interno,
  pero MCP/HTTP no publican objective, write-set, contexto ni tests completos.
- `orquesta.autoprogramming.self_improvement.propose.v0` convierte fallos
  observados en requests de automejora de segundo plano con prioridad baja; no
  sustituye la ruta goal-first ni el estado operativo.
- `orquesta.director.human_work.review_plan.v0` convierte ordenes humanas
  amplias en planes revisables y, si procede, en una request para prepare-run.
- `orquesta.domain_work.v0` es el tool generico para que una IA cree trabajos de
  dominio, entregue artefactos y evalue capabilities externas requeridas sin
  conocer OPES, DB ni runtime.
- `/api/v0/autoprogramming/prepare-run` es el bridge HTTP local del executor de
  preparacion cuando una composicion lo inyecta.
- `/api/v0/autoprogramming/status` y `/api/v0/autoprogramming/supervise`
  exponen gestion fina de autoprogramacion sobre puertos inyectados de cola,
  stats y supervisor. `status` es la observacion preferente; `supervise` queda
  como compatibilidad legacy/resident y no debe usarse para runs goal-first. No
  ejecutan runtime ni leen estado concreto por si mismos. El bridge HTTP de
  `supervise`, igual que `/api/v0/runs/supervise`, devuelve
  `accepted_background` si el executor sigue vivo y deduplica una operacion
  activa por `operation_ref`; el progreso se consulta por status/stats, no
  relanzando la misma supervision.
- `/api/v0/autoprogramming/goals/observe-active` observa en lote goals
  goal-first activos listados por `GoalWorkStateListPortV0`; reutiliza el
  executor `observe_goal` por cada `run_ref` para conservar cierre y
  reconciliacion de la composicion.
- `orquesta.director.stats.v0` puede publicar un bloque `goal` goal-first si la
  composicion inyecta un `GoalStateStore`; ese bloque solo proyecta estado ya
  persistido por `run_ref`, no observa ni cierra el Goal.
- `/api/v0/domain-work` es el bridge HTTP local del mismo executor.
- `orquesta.apps.arrancar_director.v0` es la entrada operativa preferente para
  apps nuevas desde `AppSpecV0`; `orquesta.apps.preparar_orquestacion.v0` y
  `orquesta.apps.ejecutar_orquestacion.v0` quedan como preview/compatibilidad
  sobre `orquesta-app-runner` y publican `route_policy`.
- OPES se conecta hoy inyectando su cliente REST como adaptador de dominio; si se
  usa MCPO o servidor MCP real, debe envolver estos tools como transporte opt-in,
  no duplicar logica en el nucleo.
- Los resources registrados publican `descriptor_source` verificable. La fuente
  canonica de esta regla es T198 del backlog: `orquesta-mcp` declara el envelope
  y cada owner aporta DTO/validador/freshness sin exponer rutas locales,
  secretos, prompts, transcripts ni payloads de dominio.
- La reconciliacion `agent-ref-task-autoprogramming-c3678e9bc306-g01` conserva
  ese cierre como documental/stale: no reabre codigo, transporte ni owners.
