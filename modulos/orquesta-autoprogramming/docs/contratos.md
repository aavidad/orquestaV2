# Contratos: orquesta-autoprogramming

## AutoprogrammingRequestV0

Entrada pura para validar que una peticion de autoprogramacion es pequena,
aislada y ejecutable por adaptadores externos.

Campos obligatorios:

- `request_ref`, `project_ref`, `worktree_ref`, `branch_ref`;
- `worktree_isolated=true`;
- `tasks` con `task_ref` y `area`; opcionalmente cada tarea puede aportar
  `title`, `objective`, `context`, `context_refs`, `acceptance_criteria`,
  `required_tests` y `compact_rules` para que el worker reciba un contrato
  explicito y no tenga que inferir semantica desde `task_ref`;
- `write_set` relativo y compacto;
- `required_tests`.
- opcionalmente `area_aliases` para normalizar nombres seguros de area y
  `live_works` para declarar trabajos vivos con refs opacas y `write_set`.
- opcionalmente limites de ola: `max_task_refs`, `max_areas`,
  `max_write_set_entries`, `max_delegation_depth`,
  `max_subagents_per_agent` y `max_recursive_agents`.

Limites por defecto:

- 10 tareas padre;
- 10 areas;
- 10 entradas de `write_set`;
- profundidad programable por defecto 1;
- hasta 6 subagentes por padre;
- presupuesto recursivo por defecto 60 agentes derivados.

Limites ampliados por contrato explicito:

- `max_task_refs`, `max_areas` y `max_write_set_entries` pueden subir hasta 40
  cuando la composicion/director declara el contrato en la request;
- si no vienen declarados, siguen aplicando los limites por defecto de 10;
- valores negativos o superiores a 40 se rechazan como contrato invalido;
- hints en texto libre, nombres de dominio como OPES o frases de apps grandes no
  amplian limites por si solos.

Errores frecuentes:

- `scopes`, `scope` u otros terminos de programacion no activan por si solos
  politica documental OPES ni `xhigh`;
- para reducir una ola bajo 10 padres, declarar limites explicitos mas bajos en
  la request;
- para ampliar por encima de 10 padres, 10 areas o 10 entradas de `write_set`,
  declarar los campos `max_*` explicitos dentro del contrato; los subagentes por
  padre siguen limitados a 6 salvo cambio de contrato propio.

Invariantes:

- No ejecuta agentes, tests, comandos, Git ni filesystem.
- No elige DB, runtime, proveedor, modelo ni HOME.
- No importa el nucleo de orquestacion.

## AutoprogrammingRequestSourceV0

Envoltura pura para solicitudes reales que entran por una superficie publica
como web, CLI o MCP. El modulo no ejecuta transporte ni importa adaptadores: solo
valida que el origen publico queda como evidencia durable y que la request
interna sigue siendo `AutoprogrammingRequestV0`.

Campos de evidencia:

- `schema_version=autoprogramming_request_source.v0`, `source_ref`,
  `source_surface`, `transport`, `request_id`, `correlation_id`,
  `requested_by` y `priority_score`;
- metadatos opcionales como `endpoint`, `tool_name` o `resource_uri` se
  transportan como datos opacos, sin semantica de red dentro del modulo;
- cada tarea debe conservar `context_refs` con `source_surface:*`,
  `source_ref:*`, `source_transport:*`, `source_request_id:*`,
  `source_correlation_id:*` y `source_requested_by:*`.

Asi APG-003 queda cerrado por contrato verificable, no solo por fixtures:
`ValidateAutoprogrammingRequestSourceV0` valida la envoltura y despues delega en
`ValidateAutoprogrammingRequestV0`.

## AutoprogrammingReviewGateInputV0

Entrada pura para decidir si una entrega de codigo puede aceptarse:

- ACK presente y `status=completed`;
- tests obligatorios ejecutados y pasados;
- ficheros dentro del `write_set`;
- limite de lineas por fichero, por defecto 300.

El resultado devuelve `accepted` e issues publicos. Un adaptador externo decide
como convertirlo en observaciones de review, rework o aceptacion.

El resultado tambien expone:

- `preserve_output`: la salida puede conservarse como evidencia reutilizable;
- `requires_followup`: la entrega se conserva como aceptada con rail dudoso y
  debe quedar como evidencia para revision o tarea posterior dirigida;
- `recommended_action`: `accept`, `request_followup_review`, `block_closure` o
  `request_changes`.

Si una entrega tiene ACK completado, tests obligatorios verdes y solo trae rails
dudosos de codigo (`file_too_large`, `file_outside_write_set` o destino faltante
del `write_set`), el gate devuelve `accepted=true`, conserva la salida y adjunta
`requires_followup=true` como evidencia para review/rework posterior. Asi un
arreglo util no queda bloqueado por falsos positivos, pero la evidencia no se
pierde.

Los rails de presupuesto de snapshot (`worktree_snapshot_file_too_large`,
`worktree_snapshot_too_many_files`, `worktree_snapshot_too_large` y
`worktree_snapshot_unreadable`) siguen la misma politica advisory cuando llegan
desde un adaptador de review: no autorizan efectos externos ni ignoran tests,
pero tampoco convierten una limitacion de observabilidad en veto automatico.

Las refs `ack-pending-rail:*` se tratan igual: son evidencia de detector dudoso,
no razon suficiente para relanzar otro agente si el ACK y los tests estan
verdes.

ACK ausente, ACK no completado, tests obligatorios ausentes o tests fallidos
siguen bloqueando cierre con issues compactos y
`recommended_action=block_closure`.

## BuildAutoprogrammingProgrammableWorkV0

Funcion pura que valida `AutoprogrammingRequestV0` y construye trabajo
programable compatible con el nucleo:

- devuelve `WorkProfileV0` y `WorkflowTaskV0` por grupo;
- conserva `write_set`, `required_tests`, `worktree_ref` y `branch_ref`;
- conserva objetivo, contexto, criterios, tests y reglas compactas declaradas
  por tarea dentro de `WorkflowTaskV0`/`WorkProfileV0`;
- propaga limites de delegacion gobernada a cada `WorkflowTaskV0` sin lanzar
  agentes ni elegir runtime: profundidad, hijos por padre y presupuesto total;
- compacta `title`, `summary`, objetivo y criterios antes de crear
  `WorkflowTaskV0`; el contexto largo debe viajar por refs opacas, no como
  payload durable masivo;
- usa refs opacas como `ContextRefs`, sin leer repositorios ni ejecutar nada;
- para varios grupos, particiona `write_set` por area normalizada y rechaza
  rutas sin area;
- si una ruta coincide con varias areas compatibles, devuelve
  `partition.repairs` y secuencia las tareas con `depends_on`;
- si un trabajo vivo activo solapa el `write_set`, mantiene la tarea generada
  pero la marca como `postponed` mediante `depends_on` al ref vivo;
- corta refs vivos imposibles y rutas inseguras en `live_works.write_set`.
- si el core rechaza un `WorkProfileV0`/`WorkflowTaskV0`, conserva el subcampo
  causal en el issue publico para que el director pueda reparar la forma.
- devuelve `goal_migration` como evidencia pura de migracion goal-first:
  `legacy_loop_compatible`, `goal_ready`,
  `blocked_by_goal_capability`, `covered_by_goal_first` o
  `legacy_loop_required`.
- si `goal_migration.status=goal_ready`, devuelve `goal_specs[]` como
  `GoalWorkSpecV0` neutral, uno por grupo programable, con `director_kind`
  `codex_goal`, `write_set`, pruebas requeridas, criterios, refs de contexto y
  politica de cierre por tests. No lanza ni observa ningun goal.
- si una tarea declara refs explicitas de contrato/incidencia conocidas, puede
  inyectar invariantes y pruebas de regresion obligatorias antes de entregar el
  trabajo. Primer guard soportado: `contract:codex-startup-lock:v0`,
  `bug:BUG-ORQ-20260701-092` o las incidencias startup-lock asociadas anaden
  los tests focales de `codex_wrapper` y criterios para enumerar
  `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS`,
  `ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS` y
  `ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS`.
- una composicion externa puede consumir esos `goal_specs[]` para lanzar y
  persistir `GoalWorkStateV0` si tiene backend Goal opt-in; esa decision no
  vive en este modulo puro.

### Clasificacion goal-first

La clasificacion no lanza goals, no observa runtime y no importa Codex ni
servidor. Solo lee `tasks.context_refs` para que el planner residente o un
adaptador externo decidan si una tarea de autoprogramacion debe seguir por el
loop historico o esperar/correr por Goal.

Refs reconocidas:

- `goal_migration:goal-first`: la tarea es candidata a ruta Goal.
- `goal_migration:covered`: el trabajo ya esta cubierto por una ruta goal-first
  y no debe reprogramarse en el loop legacy.
- `goal_migration:legacy-required`: el caso necesita el loop historico por
  compatibilidad, smoke existente o ausencia de runtime Goal.
- `goal_capability:starter`: existe capacidad de arrancar goals por puerto.
- `goal_capability:observer`: existe capacidad de observar `running`,
  `complete` o `blocked`.
- `goal_capability:closure-validator`: existe validacion de cierre por
  evidencias, tests y artefactos.

Reglas:

- sin marcadores, la tarea conserva `legacy_loop_compatible`;
- `legacy-required` gana sobre `goal-first`;
- `covered` gana sobre `goal-first` y devuelve
  `do_not_schedule_legacy_loop`, porque ya hay una ruta goal-first cubriendo el
  trabajo y no debe duplicarse;
- `goal-first` sin las tres capacidades queda
  `blocked_by_goal_capability`;
- `goal-first` con las tres capacidades queda `goal_ready`, pero el lanzamiento
  real sigue perteneciendo a composicion/adaptador.

### Goal specs

`BuildAutoprogrammingGoalWorkSpecsV0` compila `GoalWorkSpecV0` solo desde un
`AutoprogrammingProgrammableWorkV0` ya aceptado y clasificado como `goal_ready`.
Para cualquier otro estado devuelve lista vacia. Si el spec resultante no pasa
`ValidateGoalWorkSpecV0`, devuelve issue publico `goal_work_spec_invalid` con el
subcampo causal.

Invariantes:

- no importa runtime, Codex, MCP, servidor, HTTP, DB, VCS ni filesystem;
- conserva paths relativos del `write_set`, pruebas requeridas y criterios del
  `WorkflowTaskV0`;
- conserva/incluye guards de contrato como `rule_refs` hard cuando el task trae
  una ref explicita conocida y los tests de regresion quedan en
  `required_tests`;
- usa refs opacas de request/proyecto/worktree/branch y contexto compacto de la
  tarea;
- `complete` de un goal no se considera cierre por este modulo; la validacion
  de cierre queda en `orquesta-goal` y en la composicion.

## Automejora idle v0

`ResolveAutoprogrammingIdleSelfImprovementConfigV0` normaliza la configuracion
que una composicion ya leyo desde su entorno. El modulo no llama a `os.Getenv`.

Variables canonicas transportadas como contrato:

- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`: por defecto 60; valor
  `0` desactiva el disparo por reloj idle, sin apagar el relleno de cola por
  capacidad libre.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE`: por defecto 1; permite
  rellenar cola secundaria cuando hay capacidad libre y la cola visible esta por
  debajo del objetivo.

`DecideAutoprogrammingIdleSelfImprovementV0` devuelve `prepare=true` si:

- el servidor lleva al menos `after_seconds` sin ejecuciones; o
- hay `free_capacity > 0` y `queue_size < target_queue`.

`PlanAutoprogrammingBacklogSelfImprovementV0` opera solo con entradas ya
proyectadas por la composicion:

- salta tareas ya visibles en cola por `task_ref` o `section_ref`;
- filtra secciones narrativas y las conserva como `skipped` con razon
  `narrative_section`, sin convertirlas en runs;
- puede emitir una tarea scanner `Escaneo backlog nuevos` para descubrir huecos
  nuevos, preservando `backlog_scan_epoch`, reservas y `backlog_scan_doc` con
  linea/hash.

`ProjectAutoprogrammingExternalWorkV0` publica un estado compacto y distinguible
para trabajo externo: `outbox_pending`, `wait_external`,
`external_process_verified` o `no_external_work`. La verificacion de proceso
externo requiere refs externas y bandera `external_process_verified=true`.

Invariantes:

- no importa servidor, HTTP, runtime, DB, VCS, shell ni filesystem;
- no decide modelos, proveedores, puertos ni ejecucion real;
- las refs del scanner son opacas y compactas para que el ACK cite epoch, ref,
  lineas y hashes sin transportar documentos completos.

## AutoprogrammingRequestV1

Versiona la solicitud `v0` para composiciones que ya tienen perfiles de trabajo
por tipo de app. No cambia el contrato `v0`: si no hay `work_profiles`, el
trabajo programable conserva el perfil `implementation`.

Campos nuevos:

- `schema_version=autoprogramming_request.v1`;
- `app_kind`, normalizado como ref opaca compacta;
- `work_profiles`, con `app_kind` opcional, selector por `task_ref`, selector
  por `area` o perfil por defecto de app, `profile_kind` del catalogo neutral
  de `orquesta-core-workflow` y refs de contrato de funcion opcionales.

Reglas:

- si hay `work_profiles`, `app_kind` es obligatorio;
- `profile_kind` debe existir en el catalogo neutral del core;
- prioridad de seleccion: `task_ref`, despues `area`, despues perfil por
  defecto de app, despues fallback `implementation`;
- el resultado `BuildAutoprogrammingProgrammableWorkV1` devuelve una envoltura
  `v1` con `base` compatible con el trabajo programable `v0` y
  `profile_bindings` para evidencia.

## BuildAutoprogrammingSelfImprovementRequestV0

Entrada pura para que un director o agente convierta un fallo observado en una
automejora programable de segundo plano.

- exige `project_ref`, resumen del fallo, worktree/rama opacas, worktree
  aislada, `suggested_write_set` propio y tests requeridos;
- devuelve `AutoprogrammingRequestV0` validada y `priority_score=10` por
  defecto, salvo prioridad explicita;
- conserva `source_run_ref`, `source_task_ref`, `context_refs` y
  `evidence_refs` como contexto opaco;
- si faltan refs o tests, no descarta la evidencia: devuelve acciones para
  completar worktree aislada, estudiar write-set o definir pruebas antes de
  preparar run.

Invariantes:

- no ejecuta agentes, no encola, no lee repositorios y no toca disco;
- no bloquea el trabajo principal que produjo la observacion;
- la preparacion real queda en el adaptador que ya usa
  `orquesta.autoprogramming.prepare_run.v0`.
