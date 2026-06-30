# Contratos: orquesta-app-codex-stack

## `CodexAppStackConfigV0`

Contrato de configuracion previsto para la composition externa.

Campos conceptuales:

- `enabled`: opt-in explicito;
- `gateway`: handler web/API/MCP ya construido por `orquesta-app-gateway`;
- `director_service`: servicio `StartAppDirectorV0`;
- `director_ports`: `StartAppDirectorPortsV0` reales;
- `codex_runtime`: comando, HOME, CODEX_HOME, PATH, sandbox y approval policy;
- `delivery_store`: store externo para descriptors ACK;
- `progress_store`: store externo para estado compacto anti-bucle;
- `process_registry`: registro externo de procesos/sesiones;
- `worktree`: resolutores de proyecto, runtime dir, baseline y verificacion;
- `review_gate`: evidencia de ficheros, politica de tamano y estado de fallo;
- `capacity`: decision externa de capacidad/proveedor/modelo;
- `domain_work`: executor opt-in para trabajo de dominio externo, por ejemplo
  OPES, recibido como puerto generico ya construido;
- `persistence`: referencia externa a persistencia operacional, si aplica.

Invariantes:

- `enabled` debe ser verdadero para arrancar Codex real.
- No existe default de proveedor, modelo, DB, HOME, CODEX_HOME, puerto HTTP ni
  path de runtime.
- El modelo puede venir en `capacity` o configuracion equivalente, pero nunca se
  fija dentro de este modulo.
- La DB puede ser SQLite, Postgres, filesystem, broker o memoria segun el
  operador, pero este modulo solo recibe puertos/refs ya configurados.
- Los errores publicos no incluyen paths absolutos, tokens, prompts,
  transcripts, HOME, provider ni modelo salvo refs opacas aptas para auditoria.

## `StartAppDirectorPortsV0` reales

El stack debe inyectar el mismo contrato que consume
`orquesta-app-director-service`.

Puertos requeridos por composicion:

- store de run;
- event sink;
- outbox ledger;
- dispatchers del loop;
- fuentes opcionales de entrega, progreso, leases, replan y decisiones;
- store de microtareas de director cuando una decision crea trabajo nuevo;
- writer/store de `OperationalDirectorPlanStateV0` si la composicion quiere
  estado vivo del plan durable y reentrada por `operational_director_plan_ref`.
  Si no se pasan explicitos, el stack intenta usar el writer cuando tambien lee
  y despues el `TaskStore` cuando implementa ese puerto.

Reglas:

- REST, MCP y web no construyen estos puertos directamente.
- El gateway solo monta handlers; la composition externa decide los conectores.
- Los puertos de Codex delivery/progress se conectan como observadores del
  servicio, no como dependencias del core.
- Si falta un puerto real requerido, el arranque falla antes de lanzar agentes.
- El bridge `domain_work` solo se activa si el borde superior inyecta un
  executor; este stack no importa OPES ni crea clientes REST de dominio.

## Supervisor de agente Codex V0

Contrato interno de composicion para avanzar un agente Codex gestionado por
Orquesta:

```text
SuperviseCodexV0
  tick 1 -> AgentLifecycle.LaunchV0(ctx)
  tick 2..N -> AgentLifecycle.ContinueV0(ctx, "sigue")
  stop -> done | failed | stop_pending | stopped | max_ticks | context_done | runtime_error
```

Invariantes:

- el primer tick siempre lanza la sesion;
- los ticks siguientes empujan la sesion con `continue_message`, por defecto
  `sigue`;
- `done`, `completed` y `complete` cierran como `done`;
- `failed`, `error` y `errored` cierran como `failed`;
- `stop_pending`, `stop_requested` y `run_stop_requested` cierran el supervisor
  como `stop_pending`: el run ya tiene parada solicitada, pero todavia no se
  declara parado;
- `stopped`, `blocked`, `paused` y `needs_replan` cierran como `stopped` para
  compatibilidad del contrato de ciclo de vida;
- `pending` y `running` no son terminales para esta pieza: se siguen empujando
  hasta `done`, error, stop pendiente, stop confirmado, cancelacion de contexto
  o `max_ticks`;
- el resultado conserva historial compacto de tick, accion y snapshot;
- el contrato es un puerto del borde Codex, no del core;
- `CodexSupervisorRuntimePortV0` queda como alias compatible, pero el concepto
  correcto es `CodexSupervisorAgentLifecyclePortV0`: ciclo de vida de agente
  Orquesta, no stdin ni runtime interactivo.

Manejo real de agentes en Orquesta:

- las tareas se convierten en outbox `LaunchRuntimeAgent`;
- `agentBatchDispatcherV0` ejecuta `ExternalProcessAgentBatchExecutorV0`;
- antes de reclamar un batch de `LaunchRuntimeAgent`, el stack inyecta
  `LiveProcessCapacityGateV0`: `MaxConcurrency` de la config Codex significa
  limite global de procesos Codex vivos. Si el limite esta completo, el outbox
  queda pendiente y no se pierde la tarea; si hay hueco parcial, solo se lanza
  ese numero de agentes. Los outbox de parada no usan este gate: deben poder
  cerrar o reconciliar agentes aunque la capacidad de lanzamiento este llena;
- el launcher resuelve spec Codex, arranca proceso, registra `AgentStarted` y
  escribe `AgentProcessRegistryRecordV0`;
- si la tarea viene de `WorkflowTaskStore`, el paquete neutral conserva linaje
  parent/cohorte/ola/profundidad/fanout/hijos para que el agente conozca su
  posicion causal sin recibir proveedor, HOME, modelo ni control directo de
  spawn;
- si la tarea es de consejo residente (`architecture_proposal`,
  `architecture_critique`, `architecture_vote` o `task-council-p/c/v-*`), el
  paquete se materializa como area `decision_council`, conserva rol,
  assignment, `agent_ref`, `family_ref`, artefacto esperado, gate, deps, cohorte
  y ola en `decision_council_context.v0`, y no cae en objetivo de director
  generico ni exige `director_decisions.json`;
- `CodexReceiptRecordingSpecResolverV0` deja descriptor ACK por agente;
- `CodexDeliveryObservationSourceV0` observa `agent_ack.json` y devuelve
  `AgentDeliveryObservationV0`;
- los `director_decisions.json` libres que pueda dejar un agente de trabajo se
  leen en modo tolerante: si no cumplen el contrato
  `director_agent_decisions_file.v0`, se ignoran y no bloquean el run completo;
- `DrainRunV0` aplica ACKs, reentra `ContinueAppDirectorV0` y despacha nuevas
  decisiones;
- una parada solicitada no puede cerrar `RunControl` como `stopped` solo porque
  el run tenga `StoppedAgents`: el stack exige `ConfirmedStoppedAgents` o
  terminalidad equivalente y, si hay registro de procesos, verifica que ningun
  `process_ref + session_ref` siga `running` o `stopping`;
- si `ProcessAgentStopperV0` devuelve `process_runtime.status` no confirmado
  como `stopped`, `/api/v0/runs/supervise` proyecta `stop_pending` y deja el
  control como `stop_requested`, con siguiente accion de volver a supervisar
  hasta confirmacion real;
- si `RunControl` ya esta en `stop_requested`/`cancel_requested` pero el run o
  el ultimo snapshot muestran dispatch iniciado, agentes arrancados o proceso
  vivo, `runs/supervise` publica el diagnostico advisory
  `stop_pending_but_dispatch_in_progress`: el operador debe observar agentes y
  confirmar stop cooperativo, no matar entregas utiles por la carrera;
- progreso parado/lento entra por `ProgressSupervisionCandidateProviderV0` y el
  replan por `AssessmentReplanSourceV0`, que puede pedir `replace_agent`.

`CodexSupervisorStackLifecycleV0` es el adaptador de stack para
`ContinueV0("sigue")`: con `RunRef` reentra por `DrainRunV0`; sin `RunRef`
ejecuta `RunGlobalSupervisorV0`. En ambos casos el avance real sigue siendo tick
de supervisor/drain/replan/outbox. Si hace falta otro Codex, debe salir por
`LaunchRuntimeAgent` y el dispatcher existente, no por un canal paralelo.

Superficie publica de app:

```text
POST /api/v0/autoprogramming/prepare-run
  input: autoprogramming_request, occurred_at?, requested_by?, limites?
  output: estado, accepted, run_ref?, workflow_task_refs?, wait_agent_refs?,
          goal_spec_summaries?, goal?, goals[]?, continue?

POST /api/v0/autoprogramming/goal/observe
  input: run_ref, occurred_at?, requested_by?
  output: estado, run_ref, run_status?, goal_ref, goal_status,
          closure_status?, closure_accepted?, evidence_refs?

POST /api/v0/runs/supervise
  input: run_ref?, queue_ref?, max_ticks?, continue_message?, limites?
  output: estado, run_ref, stop_reason, ticks, last, history?, evidence_refs
```

`prepare-run` es una entrada opt-in de esta composicion: adapta el contrato MCP
`orquesta.autoprogramming.prepare_run.v0` a `PrepareAutoprogrammingRunV0`.
En modo legacy guarda `WorkflowTaskV0`/run por los stores del stack y devuelve
un `continue` acotado.
Cuando el stack tiene backend Goal completo, marca la request con
`goal_migration:goal-first` y las capacidades Goal antes de compilar el trabajo,
salvo que la request declare `goal_migration:legacy-required` o
`goal_migration:covered`. Si no hay backend Goal disponible, solo trata como
goal-first las requests que ya llegan clasificadas por
`orquesta-autoprogramming`. Cuando `orquesta-autoprogramming` clasifica la
request como `goal_ready`, el resultado depende de los puertos inyectados. Si no
hay backend Goal disponible, conserva `GoalWorkSpecV0` como handoff interno y
la API publica solo `goal_spec_summaries[]` con hash/refs/cuentas, sin
materializar ni encolar un run legacy; `run_ref`, `workflow_task_refs`,
`wait_agent_refs` y `continue` quedan vacios para que otra composicion haga
handoff. Si existen `GoalLauncher` y
`GoalObserver`, `GoalClosureValidator` y `GoalStateStore`, el stack crea un run
contenedor sin `WorkflowTaskV0` ni contratos legacy, completa
`GoalWorkSpecV0.RunRef`, lanza el goal, persiste `GoalWorkStateV0` y devuelve
`goal{run_ref,goal_ref,external_goal_ref,goal_status,evidence_refs}` para un
solo goal o `goals[]` para lotes. En batch no existe run agregado: cada
`GoalWorkSpecV0` recibe un `run_ref` derivado `request_ref-goal-XX`,
`run_ref` superior apunta al primer goal por compatibilidad, `goal` singular se
omite y cada goal debe observarse por su `goals[i].run_ref`. Si hay launcher u
observer pero falta alguna pieza del bundle, devuelve issue publico
`autoprogramming_goal_backend_incomplete` y no materializa run ni loop legacy.
Cada run contenedor goal-first no se marca `ready` en RunQueue, por lo que no entra en
`runs/supervise` ni en el drain legacy.
La reentrada es idempotente solo si el `GoalWorkStateV0.Spec` persistido
coincide con el `GoalWorkSpecV0` esperado; si cambia el contrato bajo el mismo
`request_ref`, devuelve `autoprogramming_goal_state_spec_mismatch` y no
reutiliza ni relanza. Si el launcher o el state store fallan, devuelve issue
goal-first y conserva la regla de no caer al loop legacy.
Cuando la clasificacion queda `covered_by_goal_first` o
`blocked_by_goal_capability`, tampoco se programa loop legacy salvo que el
contrato marque explicitamente `legacy_loop_required`. No arranca agentes ni
goals por si misma fuera de esos puertos opt-in. Las tareas explicitas preservan
objetivo, contexto, criterios, tests y reglas compactas hasta el paquete del
agente sin convertir `worktree_ref` ni `branch_ref` en rutas o nombres Git. La
ruta de supervision es neutral de runs. El gateway y `orquesta-mcp` no conocen
Codex; este stack inyecta
`CodexStackAutoprogrammingPrepareRunExecutorV0` y
`CodexStackAutoprogrammingObserveGoalExecutorV0` junto a
`CodexStackRunSupervisorExecutorV0`.
En la ruta legacy de `runs/supervise`, si una run external-work vuelve terminal
`done` sin agentes pedidos, agentes arrancados, agentes en vuelo ni entregas,
el stack expone el diagnostico
`external_work_accepted_no_agent_materialized` y acciones para relanzar o
replantear el trabajo con error causal explicito, en vez de presentar el cierre
como correcto.

Cuando `RunGlobalTickV0` drena una run y el loop del nucleo devuelve la run
cerrada, el stack informa `queue_status=closed` al coordinador. La cola global
persiste ese estado mediante su puerto y deja de rankear esa run en ticks
posteriores.
Si la composicion tiene `DomainWorkDeliveryBridgeConfigV0.JobRecords`
inyectado, una run quiescent no se proyecta como `stopped` ni `delivered`
mientras exista un `DomainWorkJobRecordV0` aceptado con `external_ref
run_ref=<run>` y no exista un receipt aceptado correlado en el ledger de
submissions. Esa guarda es causal y opt-in: no convierte cualquier cola vacia
en trabajo vivo, solo jobs DomainWork aceptados sin consolidacion de artefacto.

Para goal-first, `QueuedArrancarDirectorExecutorV0` no encola la run al
arrancar si el resultado `ok` trae `run_ref` y `goal_ref`: Codex Goal ocupa el
loop automatico. La sincronizacion de cola sucede al observar el goal por
`StackV0.ObserveAppDirectorGoalV0`: primero delega en
`orquesta-app-director-service.ObserveAppDirectorGoalV0` y, si el run queda
terminal, actualiza `RunQueue` como `closed` para `cerrada` o `stopped` para
`bloqueada`. Como la cola no tiene estado `blocked`, `stopped` es el estado
terminal no ejecutable usado por la composicion. Si habia candidato previo, se
conservan prioridad, app, fairness, grupo de intento, parent/supersedes,
rescue reason, claims y evidencias; si no lo habia, se crea solo una traza
terminal no ejecutable. El binding MCP/REST del stack usa
`CodexStackObserveAppDirectorGoalExecutorV0`, que llama a
`StackV0.ObserveAppDirectorGoalV0` para no saltarse esa reconciliacion.
Autoprogramacion expone la misma observacion con tool/ruta propios:
`orquesta.autoprogramming.observe_goal.v0` y
`/api/v0/autoprogramming/goal/observe`. El batch
`orquesta.autoprogramming.observe_active_goals.v0` /
`/api/v0/autoprogramming/goals/observe-active` lista estados goal por el store
inyectado y delega cada `run_ref` en ese mismo wrapper, sin saltarse la
reconciliacion de cola.
Si un operador llama `runs.supervisor` con un `run_ref` que ya tiene
`GoalWorkStateV0`, el executor no drena el loop legacy: devuelve
`stop_reason=goal_first_observe_required`, diagnostico
`run_supervisor_goal_first_not_legacy` y `next_actions` que apuntan a observar
el goal por la ruta/tool goal-first.
Cuando un `GoalWorkSpecV0` exige `ClosurePolicy.RequireDomainReceipt`, el
wrapper de cierre del stack solo acepta receipts DomainWork persistidos en el
ledger si cubren los contratos requeridos y el record asociado declara
`complete_job=true`. Un receipt aceptado pero incompleto bloquea el cierre con
`domain_work_receipt_artifact_incomplete` y rework; no puede cerrar un trabajo
goal-first por error.
Para OPES, el wrapper aplica gates editoriales de cierre sobre receipts
aceptados: un `visual_asset` SVG no cierra como arte visual profesional final, y
una entrega HTML/final/ready con `visual_count=0` no cierra si declara assets
visuales comunes/reutilizables pendientes de importar, copiar o insertar. La
ausencia de visuales solo se acepta con justificacion explicita de no
aplicabilidad o evidencia equivalente.

## Bridge de entregas a dominio externo

Contrato opt-in:

```text
ACK Codex validado
  -> DeliveryRegistered en workflow
  -> DomainWorkArtifactSubmissionV0 construido por builder inyectado/default
  -> DomainWork submit_artifact
  -> ledger idempotente por delivery_ref
```

Invariantes:

- vive solo en `orquesta-app-codex-stack`; el core no conoce OPES ni
  `DomainWork`;
- solo actua si `DomainWorkDeliveryBridgeConfigV0.Enabled=true` y existe
  executor `DomainWork`;
- usa `external_work.job_ref` para devolver artefactos, sin inferir IDs desde
  prefijos de OPES;
- si una delivery ya estaba registrada y el submit no ocurrio, el siguiente
  `DrainRunV0` reintenta mediante ledger e `idempotency_key` deterministica;
- el builder default mapea `draft_content_block` a `content_block`,
  `generate_visual_asset` a `visual_asset`, revisiones a `block_revision` y
  trabajos de fuentes a `source`;
- el builder default mapea `plan_tema`, `plan_temario` y `plan_documento` a
  `document_plan`, desenvuelve envelopes compatibles y rechaza payloads que no
  validen como `DomainDocumentPlanV0`;
- los payloads de dominio salen de `external_work.input_fields` y del fichero
  permitido por el ACK, no del core ni de internals de OPES.
- para `visual_asset`, el builder conserva `asset_type`, `format`, `title`,
  `caption`, `alt_text`, `placement`, `language_code` y refs enviadas por OPES;
  el fichero del ACK se proyecta como `body` y `format=svg` usa
  `content_type=image/svg+xml`.

## Integracion de producto externo

Si un trabajo externo legacy materializa un padre con contrato
`ApplyExternalDomainWorkV0`, hijos causales y `AllowedWriteSet` de producto,
pero el padre durable queda estrechado solo a rutas de coordinacion, el stack
puede crear una microtarea integradora generica:

- no depende de OPES, nombres de interfaz ni `subroles_required`;
- exige `ExternalWork`, write-set autorizado, hijos causales y contrato de
  dominio externo en el padre;
- usa `AllowedWriteSet` mas `/<coordinacion>` como write-set del integrador;
- conserva `parent_task_ref`, cohortes/olas si existen y dependencias del padre
  y de los hijos;
- no cierra el job externo como producto consolidado hasta que esa integracion
  entregue evidencia o bloqueo causal.

## Shutdown cooperativo de agentes Codex

Contrato interno del stack:

```text
PrepareAgentShutdownPortV0
  -> stats del run por puertos
  -> descriptors de agentes Codex por ReceiptStore
  -> request de checkpoint en runtime_work_dir
  -> ACK de checkpoint validado por runtime-codex
  -> RecordRunCheckpointV0 solo si todos responden
```

Invariantes:

- El stack no conoce detalles internos del core; usa `RunStore`,
  `ReceiptStore`, `ProcessRegistry`, progreso y uso por puertos.
- Si no hay agentes en vuelo, registra checkpoint conservador como antes.
- Si hay agentes en vuelo, escribe una request por agente y exige ACK
  `codex_shutdown_checkpoint_ack.v0`.
- Si falta descriptor, runtime dir o ACK valido, devuelve
  `checkpoint_recorded=false`, `pending_agent_refs` y evidencia compacta; el
  caso de uso de shutdown lo proyecta como `pending_checkpoint_agent_refs`,
  `checkpoint_evidence_refs` y `checkpoint_agents_pending` para REST/MCP/web.
- No se exponen rutas de runtime, HOME, modelo, proveedor ni DB en el contrato
  publico; solo refs compactas.

## Ruta `/nueva-app` opt-in

Contrato funcional:

```text
AppSpecRequestV0 validada
  -> StartAppDirectorRequestV0
  -> StartAppDirectorV0
  -> resultado compacto para web/API/MCP
```

Invariantes:

- La superficie publica no conoce scheduler, outbox, runtime Codex, DB ni
  filesystem.
- El resultado puede exponer `director_tasks`, estado del loop, agentes
  arrancados y evidence refs compactas.
- La continuidad tras `wait_external` se resuelve por observadores y reentrada
  acotada del servicio, no por polling ad hoc desde web.

## Solicitar cambio sobre app existente

Contrato de entrada conceptual: `AppChangeRequestV0`.

Campos:

- `app_ref`: identificador opaco de la app existente;
- `change_ref`: identificador idempotente de la solicitud de cambio;
- `actor_ref`: identidad opaca del solicitante, si aplica;
- `locale`: etiqueta de idioma de la interaccion de usuario;
- `user_intent`: descripcion del cambio escrita por el usuario;
- `current_state_refs`: refs compactas a estado, specs, entregas o runs previos;
- `scope`: modulos, areas o capacidades que pueden cambiar;
- `acceptance_criteria`: criterios visibles y verificables del cambio;
- `constraints`: limites de seguridad, compatibilidad, datos y despliegue;
- `allowed_write_set`: ficheros o directorios permitidos para la ejecucion;
- `metadata_refs`: refs opacas adicionales para auditoria o trazabilidad.
- `external_work`: refs opacas de trabajo de dominio externo cuando Orquesta
  coordina otra app sin importar su nucleo.

Invariantes:

- `app_ref`, `change_ref` y `current_state_refs` no son rutas, DSN, IDs de DB ni
  nombres de proveedor.
- El contrato no hereda `AppSpecRequestV0`; crear app y cambiar app existente
  son intenciones distintas.
- Los textos visibles usan `locale`, pero enums, refs y DTOs internos permanecen
  estables y no localizados.
- Los criterios de aceptacion deben poder convertirse en pruebas, checks o
  evidencia compacta antes de cerrar el cambio.
- Ningun transporte puede ampliar el `allowed_write_set` despues de validar la
  solicitud.
- `external_work` solo puede contener `project_ref`, `job_ref`, `interface_refs`,
  `work_kind`, `work_refs` e `input_fields` compactos; no contiene rutas reales,
  DB, proveedor, token, prompt ni contrato interno de la app propietaria.
- `input_fields` es el paquete de dominio curado para el agente. Puede usar
  `value`, `values` o `value_json`; el stack lo materializa en el contexto del
  agente y lo reenvia como payload de artefacto cuando corresponde. Para
  trabajos largos, el dominio puede indicar `context_budget_profile`,
  `context_profile`, `work_granularity` o `editorial_granularity` con valores
  compactos `compact`, `standard` o `large`; Orquesta usa esos hints solo para
  dimensionar la ventana de contexto, no para decidir el dominio.

Flujo por transportes:

```text
web / cambio de app existente
  -> cliente REST interno
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta

REST /api/v0/apps/{app_ref}/changes
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta

MCP app_change_request_v0
  -> AppChangeRequestV0 validada
  -> servicio de director con puertos reales
  -> run de cambio y respuesta compacta
```

Papel del director:

- reconstruye contexto solo desde puertos y `current_state_refs`;
- decide si falta informacion y puede emitir pregunta durable al usuario;
- abre fases y microtareas con linaje `change_ref`;
- compara entregas contra `acceptance_criteria`, write-set y estadisticas;
- rechaza o replanifica cuando hay tests fallidos, alcance incompleto,
  exceso de tamano o ausencia real de progreso;
- publica resultado compacto para web/API/MCP sin exponer prompts, rutas,
  credenciales, proveedor, modelo ni detalles de DB.

Reglas de replanificacion:

- replanificar crea nuevas decisiones y microtareas; no muta la solicitud
  original;
- cada nueva tarea conserva `app_ref`, `change_ref`, refs de evidencia y
  write-set acotado;
- el POST inicial solo arranca o registra el run de cambio; la continuidad se
  resuelve por `DrainRunV0` o worker equivalente;
- las reentradas consumen ACKs, estadisticas y decisiones persistidas por
  puertos hexagonales.

Estado implementado:

- `orquesta-app-change` valida `AppChangeRequestV0`;
- MCP expone `orquesta.apps.request_change.v0`;
- REST acepta `POST /api/v0/apps/change` y
  `POST /api/v0/apps/{app_ref}/changes`;
- web expone `/app-change`;
- `orquesta-app-codex-stack` registra el cambio como `AskDirector` durable en
  el workflow y deja outbox de director pendiente;
- `orquesta-app-change-director-source` lee cambios concretos por puerto y
  emite decisiones ejecutables para replanificar una microtarea de cambio;
- cuando la entrega del cambio ya existe y la programacion esta cubierta, la
  fuente abre `revision` y el review gate generico valida la entrega.

Reglas de arquitectura:

- web, REST y MCP son transportes finos; no leen filesystem, runtime ni DB para
  inferir estado de la app;
- la persistencia concreta se inyecta por puertos: SQLite, Postgres,
  filesystem, broker o memoria son decisiones del operador;
- no hay DB, provider, modelo, idioma ni path hardcodeado en este modulo;
- i18n pertenece a los bordes visibles; el core recibe claves, locale o texto
  validado, no literales de UI mezclados con enums internos.

## ACK/progreso Codex

El stack reutiliza los contratos de `orquesta-runtime-codex-delivery`.

Reglas:

- cada agente real debe tener descriptor ACK registrable antes de `AgentStarted`;
- el ACK se observa por descriptor externo, no por escaneo desde el core;
- el progreso sin ACK se reduce a senales compactas `stalled`,
  `loop_detected` o `stopped`;
- la parada de proceso usa `process_ref + session_ref` registrado, no nombre ni
  PID visible;
- la verificacion de write-set se hace fuera del core y antes de aceptar el ACK.

## Director residente del stack

Contrato de composicion:

- `RunCodexStackResidentDirectorV0` es el adaptador real para el puerto
  `orquestaserver.ResidentDirectorPortV0` usado por `cmd/orquesta-server`.
- Coordina runs ejecutables desde `RunQueue`, ordenados por
  `RankRunCandidatesV0` y acotados por `MaxRunsPerTick`/`MaxExecutions`; no
  inventa runs ni escanea todos los agentes vivos.
- Antes de ejecutar el briefing loop reutiliza
  `BuildContinueAppDirectorLoopRuntimeV0`; por tanto conserva materializacion
  de plan operativo, waits por refs/cohorte/ola, required tests, review phase,
  stores y limites del `ContinueAppDirectorV0` existente.
- La fuente de briefing es reentrante: si hay outbox pendiente emite
  `wait_outbox`; si no hay outbox emite `continue`; si el burst anterior ya
  dejo `FinalBriefing`, lo reaprovecha para no perder causalidad del paso,
  excepto cuando ese briefing representa un corte interno de presupuesto
  (`stop_max_steps`/`stop_budget_exhausted`). Ese caso no es rail de contenido:
  el siguiente pulso vuelve a mirar stores/outbox vivos y continua o espera.
- Si el run ya tiene `Brainstorms`, `Votes`, contratos funcionales publicados
  y esta en una fase donde el core permite `CreateMicrotask`, la fuente puede
  emitir la accion externa `materialize_decision_council`. El handler del stack
  convierte esa accion en tareas `task-council-*` mediante
  `DecisionCouncilPlanMaterializerV0`, usando solo `RunStore`, `EventSink` y
  `DirectorTaskStore`. La deteccion es estructural por estado durable; no lee
  prompts, logs ni palabras sueltas.
- La materializacion del consejo es idempotente: si ya existen tareas
  `task-council-*` en el run o en `DirectorTaskStore`, el handler responde como
  aplicada y no duplica trabajo. Si faltan contratos funcionales publicados o
  puertos requeridos, queda pendiente con evidencia compacta y no crea tareas.
- Las tareas resultantes tienen metadata suficiente para paquetes Codex de
  propuesta, critica y voto: refs `decision-council-role-p/c/v`,
  `assignment_ref`, `agent_ref`, `family_ref`, artefacto esperado, gate, deps,
  cohorte y ola.
- El dispatch usa los `Dispatchers`/`BatchDispatchers` ya inyectados en el
  stack. Codex real sigue viviendo solo en el batch executor y el runtime
  configurado.
- `close_or_idle` se aplica por handler externo que delega en
  `ContinueAppDirectorV0`; el adaptador no muta estados terminales a mano ni
  sustituye las fuentes reales de cierre causal.

Riesgos:

- reentrada: el servidor evita solape de ticks; el stack debe seguir usando
  stores persistentes y no rutas paralelas de dispatch.
- idempotencia: outbox ledger, run queue, run store y wait state son la barrera
  real. No sustituirla por filtros de texto ni sleeps.
- presupuesto: `MaxActions` limita el numero de acciones por tick; si se agota,
  el siguiente tick debe continuar desde eventos/outbox persistidos.

## Review gate de programacion

Contrato de composicion:

```text
delivery registrada en programacion
  -> fase revision abierta por director
  -> ReviewGateSource inyectado
  -> RequestReview + RecordReviewResult
  -> AcceptReview solo si status=accepted
```

Puertos requeridos:

- `CodexReceiptDescriptorStorePortV0` para localizar el ACK correlado;
- `CodexReviewGateFileEvidenceResultProviderPortV0` para comprobar ficheros
  reales sin que el core conozca filesystem;
- politica de limite de lineas, si el operador quiere sobrescribir el default
  del core.

Invariantes:

- `BuildStackV0` falla si no se inyecta evidencia de review gate;
- el stack solo compone el source; no reimplementa scheduler ni workflow;
- una entrega aceptada genera `accepted_review_ref`;
- una entrega invalida queda como `changes_requested` y deja evidencia compacta;
- rework/replan posterior pertenece al director y a los puertos de workflow, no
  al adaptador de revision.

## Shutdown controlado de servidor

El stack publica el binding productivo del caso de uso
`orquesta-server-shutdown` para REST/MCP/CLI.

Contrato de composicion:

```text
POST /api/v0/server/shutdown
  -> tool MCP orquesta.server.shutdown.v0
  -> orquesta-server-shutdown.ShutdownServerV0
  -> RunQueueReaderPortV0 lista runs no terminales
  -> PrepareAgentShutdownPortV0 pide checkpoint si forced=false
  -> RunControlCheckpointWriterPortV0 registra ACK durable si todos responden
  -> RunControlWriterPortV0 solicita stop despues del checkpoint
  -> RunGlobalSupervisorV0 drena stop/confirmaciones
  -> stats de director calculan readiness
```

Puertos usados:

- `RunQueueReaderPortV0` desde el store de cola;
- `RunControlReaderPortV0` y `RunControlWriterPortV0` desde el store de control;
- `RunControlCheckpointWriterPortV0` desde el store de control para dejar ACK
  durable cuando el shutdown no es forzado;
- `PrepareAgentShutdownPortV0` implementado por el stack: si no hay agentes en
  vuelo registra checkpoint conservador; si los hay, pide ACK cooperativo por
  runtime dir de cada agente Codex;
- `RunGlobalSupervisorV0` del propio stack como supervisor hexagonal;
- stats de shutdown calculadas desde `RunStore`, telemetria, progreso y usage
  inyectados en `StackConfigV0`.

Invariantes:

- el stack no mata procesos del servidor;
- el stack no lee DB, runtime, HOME, OAuth, proveedor ni modelo fuera de los
  puertos ya inyectados;
- `forced=false` devuelve `waiting_checkpoint` mientras existan agentes en
  vuelo o no haya ACK durable de checkpoint;
- `forced=true` drena por supervisor y exige stats sin agentes en vuelo antes
  de que CLI pueda enviar la senal final al servidor;
- la decision de cerrar el proceso servidor pertenece al borde operativo
  (`cmd/orquesta-server` o futuro runtime de servidor), no al caso de uso.

## Sanitizador local de contexto sensible

`CodexRuntimeConfigV0.ContextSanitizer` permite inyectar un
`ContextSanitizerPortV0` antes de construir el packet del agente. El adaptador
local `LocalSensitiveDataSanitizerV0` es determinista y opt-in: sustituye
tokens, claves, rutas privadas, URLs y material no publicable por refs opacas y
adjunta `ContextSanitizationEvidenceV0`.

`EgressSanitizerConfigV0` es la superficie canonica de la composicion para
activar el saneamiento de salida. `PrivacyFilterModelConfigV0` declara el modelo
local `openai_privacy_filter_local` como metadata opt-in sin mover proveedor,
modelo, transporte ni HOME al nucleo. `CanonicalRuntimeProviderConfigsV0`
proyecta Codex, Gemini, Claude y egress sanitizer como proveedores de runtime
uniformes para observabilidad/configuracion de composicion, usando flags/refs
compactas en vez de rutas crudas.

Invariantes:

- si no se inyecta sanitizador, el stack conserva el comportamiento anterior;
- el adaptador no elige proveedor, modelo, HOME ni transporte;
- la evidencia incluye categorias y contador, no el dato sensible;
- si hay duda, el contexto requerido se reduce a refs y el task exige revision
  por director/humano antes del cierre.
- si existe un `ContextSanitizerPortV0` explicito, prevalece sobre la config
  canonica;
- palabras blandas de operacion como provider, model, runtime o capacity no son
  veto automatico.
