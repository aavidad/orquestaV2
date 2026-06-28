# Tareas: orquesta-app-codex-stack

## APP-CODEX-STACK-049

Objetivo: que el bridge residente OPES no proyecte como trabajo vivo una entrega
durable sin proceso vivo.

Estado: hecho local.

Write-set aplicado:

- la lectura de stats del Director acepta contadores/refs de entregas
  (`tasks_delivered`, `agents_delivered`, `deliveries`);
- si no hay started/in-flight ni detalle de agente iniciado, pero si entrega
  durable, el bridge devuelve `needs_reconcile` con
  `stop_reason=stale_lock_no_process`;
- `needs_reconcile` cuenta como observacion valida del dispatch para no crear
  un bucle de reenvio;
- la evidencia prioriza refs de delivery y despues tareas/agentes entregados.

Validacion:

- `go test -count=1 ./cmd/orquesta-server -run 'TestOPESBridgeSupervisionFromDirectorStatsV0'`

## APP-CODEX-STACK-048

Objetivo: que evidencias opacas del supervisor Codex salgan como diagnosticos
publicos accionables en MCP/HTTP.

Estado: hecho local.

Write-set aplicado:

- `evidence-ref-warning-stream-fd` se proyecta como
  `codex_runtime_stream_fd_warning` con accion de seguir observando sin
  relanzar solo por ese aviso;
- evidencias de cuota (`provider-quota-exhausted`,
  `provider-usage-limit-retry-after`, `codex-usage-quota-*`) se proyectan como
  `codex_provider_quota_exhausted`;
- evidencias de capacidad limitada se proyectan como
  `codex_provider_capacity_limited`;
- la traduccion aplica tanto a resultados OK como a errores con snapshot
  parcial, sin cambiar el estado del run ni bloquear entregas.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRunSupervisor.*Diagnostics|TestCodexStackRunSupervisorErrorResultMCPV0IncluyeDiagnosticosPorEvidencia'`

## APP-CODEX-STACK-047

Objetivo: que una supervision global `no_execution` con runs `ready` no quede
muda para el operador.

Estado: hecho local.

Write-set aplicado:

- el diagnostico `run_supervisor_queue_no_execution_with_ready_candidates`
  incluye `executions=0`;
- publica limites efectivos de supervision: `max_ticks`, `max_runs_per_tick`,
  `max_executions`, `max_dispatches_per_wait` y `max_outbox_per_cycle`;
- publica `top_candidates` con run ref, status y prioridad de los primeros
  candidatos visibles;
- mantiene la accion `supervise_with_resident_mode_or_run_ref` sin relanzar ni
  mutar la cola desde el diagnostico.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponeNoExecutionConReady|TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponePresionWaitingOutbox'`

## APP-CODEX-STACK-046

Objetivo: que `autoprogramming/status` no marque `running_stale` cuando hay
procesos Codex vivos ni marque vivo un `process_ref` antiguo.

Estado: hecho local.

Write-set aplicado:

- `orquesta.director.stats.v0` recibe `ProcessSnapshot` desde
  `config.Codex.SnapshotSource`;
- el contrato neutral `DirectorAgentProcessStatsV0` publica `process.status`
  cuando el snapshot existe;
- `autoprogramming/status` solo cuenta liveness por progreso reciente o por
  `process.status=running|stopping`;
- refs de proceso sin status verificado quedan como
  `running_without_recent_stats`;
- `process.status=stopped` queda disponible para `running_stale_no_process`.

Validacion:

- `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`

## APP-CODEX-STACK-045

Objetivo: no declarar una run parada mientras queden procesos Codex sin
confirmacion real de parada.

Estado: hecho local.

Write-set aplicado:

- `SuperviseCodexV0` y el lifecycle del stack distinguen `stop_pending` de
  `stopped`;
- `run_stop_requested`, `stop_requested` y `stop_pending` ya salen como
  `stop_pending`, no como parada terminal;
- si `ProcessAgentStopperV0` no recibe snapshot `stopped`, `/runs/supervise`
  devuelve `stop_pending` y conserva la parada como pendiente;
- la reconciliacion queued de run-control ya no considera `StoppedAgents` como
  confirmacion suficiente;
- antes de completar `RunControl` como `stopped`, el stack verifica
  `ProcessRegistry + SnapshotV0` y bloquea si un proceso registrado sigue
  `running`/`stopping` o no cuadra su identidad;
- el resultado MCP publica siguientes acciones para volver a supervisar y no
  marcar parado sin confirmacion;
- `orquesta.director.stats.v0` recibe `RunControl` desde el stack y publica
  `stop_control` con `stop_requested`, `stop_propagated`, `stop_pending` o
  `stop_confirmed`, mas checkpoint/forced/evidencias compactas.

Validacion:

- `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-app-codex-stack`.

Pendiente:

- smoke Codex real opt-in que demuestre cero procesos `codex exec` vivos tras
  stop forzado.

## APP-CODEX-STACK-044

Objetivo: reconciliar `RunQueue` al observar un goal-first terminal sin
reintroducir el loop legacy.

Estado: hecho.

Write-set aplicado:

- `StackV0.ObserveAppDirectorGoalV0` envuelve
  `orquesta-app-director-service.ObserveAppDirectorGoalV0`;
- si el servicio deja el run `cerrada`, la cola queda `closed`;
- si el servicio deja el run `bloqueada`, la cola queda `stopped` porque
  `RunQueue` no tiene estado `blocked`;
- la sincronizacion conserva prioridad, app, fairness, grupos de intento,
  parent/supersedes, rescue reason, workset claims y evidencias cuando ya habia
  candidato;
- si goal-first no se habia encolado al arrancar, se crea solo una traza
  terminal no ejecutable al observar el cierre/bloqueo.
- `CodexStackObserveAppDirectorGoalExecutorV0` expone esa observacion por el
  binding MCP/REST del stack y llama al wrapper anterior, no al servicio directo.
- `TestObserveAppDirectorGoalV0ReanudaTrasRestartDesdeStateFile` fija que un
  stack reconstruido con `orquesta-state-file` puede cargar `GoalWorkStateV0`,
  observar el goal terminal, cerrar la run persistida y crear traza terminal de
  cola sin reentrar al loop legacy.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'CodexStackObserveAppDirectorGoalExecutor|TestObserveAppDirectorGoalV0SincronizaCola|TestQueuedArrancarDirectorExecutorV0NoEncolaGoalFirst|TestBuildDirectorPortsV0CableaAppGoalLauncher'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run TestObserveAppDirectorGoalV0ReanudaTrasRestartDesdeStateFile`

Pendiente siguiente:

- ampliar UI/operacion para refresco periodico si el operador necesita polling
  automatico; la accion publica manual ya existe por
  `/api/v0/apps/director/goal/observe`.

## APP-CODEX-STACK-043

Objetivo: recuperar decisiones de capacidad duraderas no proyectadas para que
una run OPES no deje tareas sin agente.

Estado: hecho.

Write-set aplicado:

- `DrainRunV0` ejecuta `reconcileDurableCapacityDecisionsForRunV0` antes de la
  recuperacion de launches reclamados;
- el reconciliador lee eventos duraderos del run y localiza
  `CapacityDecided` por `capacity_ref`;
- solo actua si la solicitud de capacidad sigue existiendo en la proyeccion
  actual del run;
- aplica el evento `CapacityDecided` sobre la proyeccion actual, conserva
  `LastEventID`/`LastSequence` coherentes y guarda el run;
- no interpreta texto OPES, logs ni summaries de agentes.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestReconcileDurableCapacityDecisionsForRunV0ProyectaDecisionPerdida|TestReconcile(OrphanCapacity|ClaimedLaunch)'`

Pendiente siguiente:

- revisar la eficiencia del supervisor cuando hay agentes vivos pero aun queda
  concurrencia disponible para lanzar otras tareas.

## APP-CODEX-STACK-042

Objetivo: especializar el paquete Codex de las tareas del consejo residente.

Estado: hecho.

Write-set aplicado:

- `codexAreaV0` clasifica `architecture_proposal`, `architecture_critique`,
  `architecture_vote` y refs `task-council-p/c/v-*` como `decision_council`,
  no como `director`;
- el resolver carga `WorkflowTaskV0` para tareas de consejo aunque la fase sea
  `brainstorming_arquitectura` o `votacion_y_decision`;
- `councilTaskV0` conserva titulo, write-set, skills, cohorte, ola, linaje y
  criterios durables, y genera objetivo especifico para propuesta, critica o
  voto;
- el contexto del packet incluye `decision_council_context.v0` con
  `task_ref`, rol, `assignment_ref`, `agent_ref`, `family_ref`,
  `expected_artifact`, `gate_ref`, `context_refs`, `depends_on`, cohorte y ola;
- la deteccion usa refs estructuradas `decision-council-role-p/c/v`,
  `payload.Role` enum y prefijo de task como fallback, sin parsear summaries,
  logs ni texto libre.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexLaunchSpecResolverV0MaterializaPacketConsejoPropuestaCriticaVoto|TestCodexLaunchSpecResolverV0DetectaConsejoPorMetadataSinRolPayloadV0'`

Pendiente siguiente:

- fuente real de `architecture_vote.v0` desde entregas/artefactos Codex;
- smoke opt-in `Director residente + Codex real` con servidor temporal.

## APP-CODEX-STACK-041

Objetivo: cerrar el consejo residente desde votos estructurados hasta
`AcceptDecision`.

Estado: hecho.

Write-set aplicado:

- `ConfigV0.DecisionCouncil` y `StackV0.DecisionCouncil` permiten inyectar una
  fuente de votos por composicion;
- cuando las tareas `task-council-v-*` estan entregadas, el briefing residente
  propone `accept_decision_council_result` solo si existe `VoteSource`
  inyectado para no entrar en un pendiente externo repetitivo;
- el handler pide votos estructurados al puerto `VoteSource`, hidrata
  `TaskRef`/`VoteRef`, fuerza `VoterRef` desde `agent_ref` y `FamilyRef` desde
  `family_ref` en tareas/plan durables, evalua quorum con
  `orquesta-decision-council` y aplica `AcceptDecision` por el workflow;
- la ruta queda idempotente si la decision ya esta reflejada;
- no se parsean summaries, logs ni texto libre de agentes.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'ResidentCouncil|ResidentDirector'`
- `go test -count=1 ./modulos/orquesta-orchestration-core -run 'DecisionCouncil|WorkflowTaskProfile|AcceptDecision'`
- `go test -count=1 ./modulos/orquesta-decision-council`

Pendiente siguiente:

- fuente real de `architecture_vote.v0` desde entregas/artefactos Codex;
- smoke opt-in `Director residente + Codex real` con servidor temporal.

## APP-CODEX-STACK-040

Objetivo: hacer que el consejo residente materializado sea trabajo vivo del
scheduler, no solo tareas persistidas.

Estado: hecho.

Write-set aplicado:

- `materialize_decision_council` abre `brainstorming_arquitectura` tras crear
  `task-council-*`;
- la fuente residente detecta propuestas y criticas ya entregadas y propone
  `open_decision_council_vote_phase`;
- el handler abre `votacion_y_decision` y conserva idempotencia por comando
  `OpenPhase`;
- tests fake validan candidatos de propuesta y voto por `WorkflowTaskStore`.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-decision-council ./modulos/orquesta-app-director-service`

Pendiente siguiente:

- fuente real de `architecture_vote.v0` desde entregas/artefactos Codex;
- smoke opt-in `Director residente + Codex real`.

## APP-CODEX-STACK-001

Objetivo: crear contexto local del mini-proyecto exterior sin tocar core ni
app-gateway productivo.

Estado: hecho.

Validacion:

- `git diff --check -- modulos/orquesta-app-codex-stack`

## APP-CODEX-STACK-002

Objetivo: documentar el contrato de composition para inyectar
`StartAppDirectorPortsV0` reales desde web/API/MCP.

Estado: hecho.

Validacion:

- `docs/contratos.md` define config, puertos e invariantes opt-in.

## APP-CODEX-STACK-003

Objetivo: preparar un arranque manual guardado para smokes Codex reales.

Estado: hecho.

Validacion:

- `arrancar_codex.sh` exige `ORQUESTA_CODEX_STACK_OPT_IN=1`;
- no define proveedor, modelo, DB, HOME, CODEX_HOME, PATH ni comando por
  defecto.

## APP-CODEX-STACK-004

Objetivo: implementar composition Go real del stack externo.

Estado: hecho.

Write-set aplicado:

- `ConfigV0` con puertos/stores/runtime/capacidad inyectados;
- `BuildStackV0` como composition root HTTP para web/API/MCP;
- resolutor de launch spec y waiter de ACK por puertos;
- test de API y web con runtime fake inyectado.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no activar esta ruta desde `cmd` ni app-gateway productivo;
- no crear defaults de DB/proveedor/modelo;
- no duplicar logica de `orquesta-app-director-service`;
- no filtrar nombre de conector real en payloads, evidence refs ni refs
  publicas del core.

## APP-CODEX-STACK-022

Objetivo: cablear `domain_work` como executor opt-in en el stack HTTP.

Estado: hecho.

Write-set aplicado:

- `ConfigV0.DomainWork` como puerto MCP generico;
- `BuildStackV0` pasa ese executor a `orquesta-app-gateway`;
- test de `/api/v0/domain-work` con executor fake inyectado.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no importar OPES ni conectores REST en este modulo;
- no crear DB, runtime, proveedor ni modelo por defecto;
- si no hay executor inyectado, el endpoint queda apagado por opt-in.

## APP-CODEX-STACK-005

Objetivo: smoke opt-in desde `/nueva-app` con Codex real, ACK de director y
registro durable de artefacto/fase.

Estado: hecho.

Validacion:

- `ORQUESTA_CODEX_STACK_OPT_IN=1` con `arrancar_codex.sh`;
- `ORQUESTA_CODEX_STACK_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealOptInV0 -count=1 -timeout 300s -v`;
- modelo configurado por operador: `ORQUESTA_CODEX_MODEL=gpt-5.5`;
- evidencia local: `/tmp/orquesta-smokes/app-codex-stack-real-2/project`;
- resultado: PASS en 88.09s;
- repeticion tras modularizar tests: PASS en 82.09s con evidencia local
  `/tmp/orquesta-smokes/app-codex-stack-real-3/project`;
- ACK real escrito en `agent_ack.json`;
- documentos creados por el agente: `docs/arquitectura.md` y
  `docs/plan_microtareas.md`;
- `PhaseArtifactRegistered` observado en el sink del run;
- proceso real detenido al cerrar la prueba.

## APP-CODEX-STACK-006

Objetivo: validar `/nueva-app` con cohorte multiagente real lanzada por
Orquesta, no por coordinacion manual.

Estado: hecho.

Validacion:

- `ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealMultiagentOptInV0 -count=1 -timeout 420s -v`;
- comando ejecutado mediante `arrancar_codex.sh` con opt-in explicito;
- modelo configurado por operador: `ORQUESTA_CODEX_MODEL=gpt-5.5`;
- resultado: PASS en 148.207s;
- evidencia local:
  `/tmp/orquesta-smokes/app-codex-stack-multiagent-real-12/project`;
- Orquesta arranco 4 Codex reales en paralelo: `director`, `web`, `api` y
  `persistencia`;
- cada agente escribio `agent_ack.json` en runtime aislado;
- documentos creados: `docs/arquitectura.md`, `docs/plan_microtareas.md`,
  `docs/web.md`, `docs/api.md`, `docs/persistencia.md`;
- el drenaje final registro los ACKs como artefactos de fase y la prueba cerro
  procesos sin dejar sesiones vivas.

Regla cerrada:

- una firma de runtime sin cambios no equivale a bucle; para no parar agentes
  reales que estan pensando, `loop_detected` solo debe salir de una senal real
  de accion repetida, no de ausencia temporal de ACK.

## APP-CODEX-STACK-007

Objetivo: cerrar el salto director-documentacion -> decisiones ejecutables ->
programacion.

Estado: hecho.

Write-set aplicado:

- contrato de decisiones del director principal en helper separado;
- criterio de cierre que exige `director_decisions.json` solo al area
  `director`;
- clasificacion de rol `implementacion` como area `programacion`;
- test fake que simula un director escribiendo decisiones y valida que Orquesta
  abre `programacion` y arranca el agente de microtarea.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexAreaV0|TestDirectorTaskV0|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`

Reglas cerradas:

- el director no decide por memoria de Codex sino por fichero de control
  consumido por puerto;
- las areas especializadas no toman control del workflow;
- programacion no hereda identidad ni contrato de director.

## APP-CODEX-STACK-008

Objetivo: endurecer el contrato textual de `director_decisions.json` tras
prueba real con JSON no aplicable.

Estado: hecho.

Write-set aplicado:

- prompt del director con campos obligatorios `decision_ref`, `command_type`,
  `phase_id`, `command_ref`, `summary` y `evidence_refs`;
- campos internos obligatorios por payload tipado, incluido `task_id` frente a
  `task_ref`;
- prohibicion explicita de aliases genericos `action`, `decision_id`,
  `payload` y `refs`;
- prohibicion explicita de `evidence_refs` con espacios, slash, rutas o texto
  humano;
- IDs encadenados exactos entre `request_vote`, `accept_decision`,
  `publish_function_contract` y `create_microtask`;
- vocabulario seguro para campos de decision: `datos sensibles` en vez de
  terminos prohibidos literales;
- enum literal de capacidad: `low`, `medium`, `high`, `xhigh`; la votacion
  inicial usa `high`;
- test del objetivo del director para fijar esos nombres.

Validacion:

- local focal ok tras repetir suite con la restriccion de refs compactas:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDirectorTaskV0|TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion|TestCodexStackV0WebDrenaACKMultiagenteTardio|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`;
- primera repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-6/project`;
- segunda repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-7/project`
  por refs encadenadas inconsistentes y vocabulario prohibido en decision.
- tercera repeticion real fallo en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-8/project`
  por capacidad localizada como `alta` en vez del enum `high`.
- cuarta repeticion real ok en
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project`.

Reglas cerradas:

- el stack no convierte formatos inventados por el agente;
- la autonomia se mantiene por DTO estricto y puerto de decisiones.

## APP-CODEX-STACK-009

Objetivo: hacer que el drenaje de un run existente consuma decisiones tardias
del director y lance agentes de programacion.

Estado: hecho.

Write-set aplicado:

- `DrainRunV0` reentra por `ContinueAppDirectorV0`;
- espera externa acotada en el borde del stack;
- dispatcher fino para `SendDirectorQuestion` no bloqueante;
- test de decision file tardio que exige abrir `programacion` y arrancar
  agente de microtarea;
- smoke real multiagente ahora exige al menos un descriptor de programacion.

Validacion:

- local focal ok:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion|TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion|TestCodexStackV0WebDrenaACKMultiagenteTardio|TestNuevaAppWebCodexStackRealMultiagentOptInV0' -v`;
- integrada ok:
  `go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./orquestacionnucleoapp ./modulos/orquesta-app-director-service ./modulos/orquesta-app-director-intake ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`;
- smoke real opt-in con Codex ok en 194.108s:
  `/tmp/orquesta-smokes/app-codex-stack-director-decisions-real-10/project`.

Reglas cerradas:

- no se continua la orquestacion desde la sesion principal;
- Orquesta reentra sola usando puertos hexagonales y decisiones persistidas.

## APP-CODEX-STACK-010

Objetivo: cerrar el ciclo completo de programacion real, no solo el lanzamiento
de agentes de programacion.

Estado: hecho.

Write-set aplicado:

- `DrainRunV0` corta cuando lanza una nueva ola de agentes externos y deja la
  continuacion a otra reentrada;
- refs de entrega unicas por agente para evitar colisiones de ACK, mailbox y
  readiness;
- smoke real que espera ACKs de la ola de programacion;
- segundo drain que registra entregas de programacion;
- verificador de write-set compatible con ficheros y directorios;
- guarda de tamano para ficheros Go generados: maximo 300 lineas.

Validacion:

- local focal ok:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRealSmokeVerifyProjectFileV0AceptaDirectorioConContenido|TestCodexStackRealSmokeVerifyGoFileSizesV0AceptaFicheroManejable|TestAgentPacketV0UsaDeliveryRefsUnicasPorAgente|TestNuevaAppWebCodexStackRealMultiagentOptInV0' -v`;
- integrada ok:
  `go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./orquestacionnucleoapp ./modulos/orquesta-app-director-service ./modulos/orquesta-app-director-intake ./modulos/orquesta-web ./modulos/orquesta-app-gateway ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack`;
- smoke real `real-11`: fallo por bug del verificador, no por contrato de
  orquestacion;
- smoke real `real-12`: PASS en 500.099s con app Go/API/web generada,
  ACKs de programacion, pruebas Go de la app y cierre limpio de procesos.

Reglas cerradas:

- no meter bases de datos concretas por defecto en el core;
- no esperar programacion completa dentro del POST inicial;
- mantener microtareas y ficheros Go por debajo del tamano manejable;
- no continuar programacion desde la sesion principal: solo desde reentradas
  de Orquesta.

## APP-CODEX-STACK-011

Objetivo: convertir los resultados de programacion en estadisticas y decisiones
automaticas del director.

Estado: hecho para exposicion de estadisticas por puertos; pendiente separar
una politica productiva de rechazo/replanificacion automatica por entregas
invalidas.

## APP-CODEX-STACK-012

Objetivo: conectar el review gate de entregas de programacion al stack exterior
para que el director pueda aceptar o pedir cambios desde Orquesta.

Estado: hecho.

Write-set aplicado:

- `ConfigV0` exige `ReviewGate.FileEvidence` como puerto explicito;
- `buildDirectorPortsV0` inyecta `ReviewGateSource`;
- `reviewGateSourceV0` compone `CodexReviewGateObservationSourceV0`;
- el runtime fake de tests materializa ficheros reales declarados en ACK;
- test de stack valida aceptacion con fichero real y rail blando por fichero de
  mas de 300 lineas, sin convertirlo en rework duro.

Validacion:

- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el stack no decide internamente si una entrega vale: compone puertos;
- la lectura de ficheros queda en adaptador externo inyectado;
- no hay default de DB, proveedor, modelo, HOME ni path global;
- la revision no filtra detalles operacionales hacia el core.

Write-set aplicado:

- `stack_v0.go` inyecta `RunStore`, `ProcessRegistry` y `ProgressSource` en
  `orquesta.director.stats.v0`;
- `stack_flow_progress_stats_v0_test.go` prueba que el stack expone agentes con
  control de parada y progreso sin senal por web/API;
- `orquestacionnucleoapp` y `orquesta-mcp` publican `DirectorRunStatsV0` con
  `progress` y `closure`.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0DirectorStatsIncluyeProcesoYProgresoPorPuertos|TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado' -v`;
- `go test -count=1 ./orquestacionnucleoapp ./modulos/orquesta-mcp`.

Pendiente separado:

- permitir que el director rechace/replanifique entregas con tests fallidos,
  write-sets incompletos, exceso de tamano o ausencia real de progreso;
- definir politica separada para specs no Go, por ejemplo OpenAPI largo.

Reglas de entrada:

- mantener hexagonalidad: estadisticas como puerto, no como lectura directa de
  ficheros del runtime desde web/MCP;
- no hardcodear DB, proveedor ni modelo;
- no crear ficheros grandes de agregacion.

## APP-CODEX-STACK-012

Objetivo: permitir que el usuario solicite cambios sobre una app existente y
que el director reciba la senal por Orquesta.

Estado: hecho.

Write-set aplicado:

- nuevo mini-proyecto `modulos/orquesta-app-change`;
- herramienta MCP `orquesta.apps.request_change.v0`;
- REST `POST /api/v0/apps/change` y
  `POST /api/v0/apps/{app_ref}/changes`;
- web `/app-change`;
- gateway con ruta dinamica sin romper `/api/v0/apps/spec` ni
  `/api/v0/apps/director`;
- stack Codex traduce el cambio a `AskDirector` y guarda outbox del director;
- store de cambios inyectado por puerto, sin DB concreta;
- fuente compuesta de decisiones que convierte cambios concretos en respuesta al
  director, votacion, contrato funcional, microtarea y vuelta a `programacion`;
- fuente de cambios abre `revision` cuando la entrega del cambio ya existe y el
  review gate generico acepta o pide rework con evidencia real.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change ./modulos/orquesta-app-change-director-source ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack`
- `TestCodexStackV0CambioProgresivoPasaReviewGate`.

## APP-CODEX-STACK-013

Objetivo: cerrar presupuestos por agente/tarea/ACK para que una app grande no
dependa de un timeout global.

Estado: hecho para helper/test local de stack; pendiente repetir smoke real
largo con Codex real.

Contexto real:

- el smoke real `/tmp/orquesta-smokes/multiagent-20260511143049/project`
  genero una app Go/API/web parcial que compila y pasa `go test ./...`;
- fallo por timeout global antes de que el tercer agente escribiera ACK;
- la tarea `HTTP/web/i18n` era demasiado amplia para el presupuesto de smoke.

Trabajo minimo:

- definir contrato de presupuesto por tarea: tiempo maximo esperado, tiempo sin
  actividad, tiempo desde ultimo cambio de fichero, tiempo desde ultimo ACK y
  severidad;
- separar en stats/MCP: `working`, `stalled`, `over_budget_but_active`,
  `over_budget_no_activity`, `ack_registered_cleanup`;
- hacer que el director pueda decidir split/replan cuando una tarea activa se
  pasa de presupuesto;
- exigir que el stack no mate directores protegidos por presupuesto automatico;
- mantener cleanup terminal tras ACK como operacion runtime, no como stop de
  workflow.

Trabajo aplicado en stack:

- el helper de app completa capea cada pasada de `DrainRunV0` para no consumir
  el timeout global en una sola espera externa;
- el criterio de progreso incluye secuencia, tareas, entregas, artefactos,
  assessments, descriptores y ACKs observables;
- si un descriptor de programacion tiene write-set completo, el proyecto Go
  compila y no hay ACK registrable, el smoke falla temprano con
  `project_compiles_but_ack_missing`;
- el smoke real multiagente exige una ola paralela de programacion con varios
  `AgentStarted` antes del primer `DeliveryRegistered`;
- `TestCodexStackRealSmokeDetectaACKFaltanteConProyectoCompilable` reproduce
  localmente el caso sin lanzar Codex real ni depender de intervencion manual.

Validacion esperada:

- test local de agente activo que supera tiempo pero modifica archivos: no se
  mata, se marca `over_budget_but_active`;
- test local de agente sin actividad: assessment y accion segun fase;
- smoke real acotado con tarea HTTP/web/i18n partida en microtareas menores;
- `go test ./modulos/orquesta-app-codex-stack -count=1`;
- bateria del nucleo/director/runtime/dispatch.

## APP-CODEX-STACK-014

Objetivo: exponer uso de recursos por agente para director, MCP y web.

Estado: cerrado para puerto, acumulado por run y reporte runtime redactado
T209. Sigue fuera de alcance inventar cuota restante real si Codex no la
expone.

Trabajo aplicado:

- `AgentUsageStatsProviderPortV0` en el nucleo;
- `DirectorAgentStatsV0.Usage` con capacidad, cuota y tokens agregados;
- MCP `orquesta.director.stats.v0` acepta `include_agent_usage`;
- web proyecta `capacity_level`, `quota_status`, `quota_remaining`,
  `quota_limit` y `total_tokens`;
- stack Codex publica capacidad desde el paquete de agente; sin fuente de
  metricas queda `not_configured`, y con fuente runtime redactada puede quedar
  `unknown`, `available`, `limited` o `exhausted`.
- `CodexStackAgentUsageMetricsProviderPortV0` permite inyectar metricas por
  agente sin que el stack conozca proveedor, HOME, OAuth ni API remota;
- `DirectorRunStatsV0.UsageSummary` acumula agentes observados, cuota y tokens
  por run;
- web proyecta el resumen en `resumen.usage_*`.
- `CodexStackRuntimeUsageMetricsSourceV0` lee
  `codex_usage_accounting.json` desde descriptors del agente y rechaza reportes
  con provider, modelo, cuenta, HOME, coste, rutas, prompts, transcripts o
  completions.
- reporte ausente/no redactado queda como `unknown` con
  `quota_observed_unavailable`, no como `not_configured`.

Pendiente:

- cuota real restante de proveedor si Codex no la publica de forma segura;
- politica final de privacidad para aliases publicos si una composicion quiere
  exponer metadatos de proveedor fuera del contrato neutral.

## APP-CODEX-STACK-015

Objetivo: soportar multiples apps simultaneas con cola global por prioridad.

Estado: hecho en contratos, MCP/REST, conector de memoria y runner global
interno.

Trabajo minimo:

- contrato `RunQueueReaderPortV0`, `RunQueuePriorityWriterPortV0` y
  `RunSchedulingCandidateV0`;
- ranking puro por `priority_score` descendente, aging/fairness y `updated_at`;
- filtro de runs no ejecutables: `paused`, `canceled`, `stopped`, `closed`;
- MCP/REST `orquesta.run_queue.priority.v0` con acciones `rank` y
  `set_priority`;
- `orquesta-run-memory` como conector de memoria thread-safe para pruebas;
- `orquesta-run-coordinator` como runner global puro que selecciona por ranking,
  respeta `RunControl` y drena por puerto;
- el stack registra automaticamente cada run arrancada en la cola global;
- `RunGlobalTickV0` en el stack ejecuta la run prioritaria sin conocer detalles
  internos de cada app;
- tests de stack que validan alta en cola, prioridad y pausa.

Cerrado separado en `orquesta-web` y gateway:

- vista web `/run-queue` para ranking de cola global y cambio de prioridad;
- progreso profundo por run sigue en `/director-stats` para no duplicar el
  contrato de estadisticas.

## APP-CODEX-STACK-017

Objetivo: cerrar el tick global multiapp sin abrir aun superficie MCP nueva.

Estado: hecho.

Trabajo aplicado:

- adaptador `RunGlobalTickV0` que compone `orquesta-run-coordinator` con
  `StackV0.DrainRunV0`;
- `QueuedArrancarDirectorExecutorV0` que encola la run al arrancar director;
- `RunQueueConfigV0` con `queue_ref`, prioridad inicial, limite de cola y
  ejecuciones maximas por tick;
- comandos de prioridad con `queue_ref` y `updated_at`;
- presupuesto conservador por tick para evitar que una run bloquee a las demas.
- propagacion de `queue_status=closed` cuando el loop del nucleo ya devuelve la
  run cerrada, para que la cola no repita trabajos terminales historicos.

Validacion:

- `go test -count=1 ./modulos/orquesta-run-queue ./modulos/orquesta-run-memory ./modulos/orquesta-run-coordinator`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el coordinador global no importa stack, runtime, HTTP, MCP ni DB;
- el stack exterior traduce entre puertos, no mete politica global dentro del
  scheduler interno de cada run;
- la cola global queda alimentada desde el arranque real de app, no solo por
  llamadas manuales a `set_priority`.
- una run cerrada por el nucleo se marca terminal en cola por puerto y no se
  re-rankea en ticks posteriores.

## APP-CODEX-STACK-018

Objetivo: ejecutar una pasada supervisada de la cola global multiapp.

Estado: hecho.

Trabajo aplicado:

- nuevo mini-proyecto puro `orquesta-run-supervisor`;
- contrato `SuperviseRunsV0` con `MaxTicks`, `MaxExecutions`,
  `MaxRunsPerTick`, `StopOnNoExecution` y exclusion temporal de runs ya
  ejecutadas;
- `StackV0.RunGlobalSupervisorV0` compone el supervisor con
  `RunGlobalTickV0`;
- test de stack que arranca dos apps, ajusta prioridades y valida que la pasada
  ejecuta primero la de mayor prioridad y despues la siguiente, sin repetir la
  primera dentro de la misma pasada.

Validacion:

- `go test -count=1 ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el supervisor no es daemon, no duerme y no crea goroutines;
- todo bucle tiene presupuesto explicito;
- el mini-proyecto puro no importa DB, HTTP, MCP, runtime ni stack;
- la repetition policy no muta la cola global.

## APP-CODEX-STACK-016

Objetivo: control humano/orquestador de una app completa: pause, resume, stop y
cancel.

Estado: hecho en contratos, MCP/REST, stack y gate del nucleo; pendiente
checkpoint/cierre terminal de estado para stop/cancel.

Trabajo minimo:

- mini-proyecto `orquesta-run-control` con estados `running`, `paused`,
  `stop_requested`, `stopped`, `cancel_requested`, `canceled`;
- MCP/REST `orquesta.runs.control.v0` con acciones `pause`, `resume`, `stop`
  y `cancel`;
- `orquesta-run-memory` como conector de memoria thread-safe;
- el nucleo consulta `RunControlReaderPortV0` antes de planificar y antes de
  despachar;
- sin estado de control tipado como `RunControlStateNotFoundErrorV0` se asume
  `running` por defecto;
- test de stack pausa una app por API y lee el estado por puerto.

Cerrado despues:

- escritura terminal `stopped/canceled` mediante
  `RunControlTerminalWriterPortV0` cuando todos los agentes vivos han sido
  confirmados o no habia agentes que parar;
- el cierre terminal queda bloqueado si todavia existe outbox
  `StopRuntimeAgent` pendiente.
- la politica autonoma del nucleo acepta `DirectorRunStatsV0` y usa senales de
  progreso/presupuesto para subir capacidad y acotar paralelismo.
- checkpoint real antes de stop/cancel no forzado queda cubierto en
  `orquesta-server-shutdown` por
  `TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado` y
  `TestShutdownServerV0NoPideStopSiCheckpointNoEstaListo`; el stack mantiene el
  puerto real con `TestStackShutdownCheckpointV0SolicitaAckSiHayAgenteEnVuelo`
  y `TestStackShutdownCheckpointV0RegistraCuandoTodosLosAgentesResponden`.

Pendiente separado:

- UI web de control queda cubierta por `/run-control`; cambio de prioridad por
  `/run-queue`.

## APP-CODEX-STACK-020

Objetivo: permitir que OPES y otras apps externas reciban artefactos producidos
por agentes de Orquesta.

Estado: hecho en bridge opt-in inicial.

Trabajo aplicado:

- `DomainWorkDeliveryBridgeConfigV0` con builder y ledger hexagonales;
- `DrainRunV0` intenta enviar artefactos tras observar ACKs y tambien reintenta
  deliveries ya registradas sin ledger;
- builder default usa `external_work.job_ref`, `input_fields`, ACK validado y
  fichero permitido por el write-set;
- mapeo generico de `draft_content_block -> content_block`,
  revisiones -> `block_revision` y fuentes -> `source`;
- cmd server activa el bridge solo si `ORQUESTA_OPES_BASE_URL` esta definido.

Validacion:

- `TestCodexStackV0OPESExternalWorkDeliveryEnviaArtefactoDomainWork`;
- `TestCodexLaunchSpecResolverV0MaterializaContextoDominioExterno`;
- `go test -count=1 ./modulos/orquesta-app-codex-stack`.

## APP-CODEX-STACK-021

Objetivo: permitir que OPES consulte progreso por job externo sin conocer la
estructura interna de la run.

Estado: hecho como proyeccion opt-in de stats.

Trabajo aplicado:

- `CodexStackExternalJobStatsSourceV0` resuelve `external_job_ref` desde
  `AppChangeStore`, deriva `task_ref/agent_ref` y proyecta status compacto;
- en la proyeccion de status, `failed`/`lost` prevalecen sobre restos en
  `started` para no exponer como vivo un agente ya reconciliado como perdido;
- si el padre ya entrego ACK pero los hijos causales siguen abiertos, el job
  externo se expone como `parent_ack_received` con
  `status_reason=cohort_open`;
- cuando `WorkflowTaskStore` muestra que los subroles hijos ya estan resueltos
  pero el padre sigue sin entrega/cierre, el job externo se expone como
  `integration_required` con `status_reason`, `issue_refs` y `diagnostics`;
- `/api/v0/director/stats` acepta `external_job_ref` y puede resolver `run_ref`
  mediante puerto MCP;
- `RunFileStoreV0` conserva `external_work.job_ref` e `input_fields` al
  persistir y reabrir estado;
- `orquesta.runs.control.v0` puede resolver `external_job_ref` y controlar la
  run asociada;
- el servidor usa ledger de artefactos persistente en
  `ORQUESTA_DOMAIN_DELIVERY_LEDGER_PATH` o en `StateDir` por defecto.

Validacion:

- `TestMCPDirectorStatsToolExecutorV0ResuelveRunPorJobExterno`;
- `TestCodexStackExternalJobStatsSourceV0MarcaIntegracionPendienteTrasSubroles`;
- `TestCodexStackExternalJobStatsSourceV0ExponeNarrowingLegacyComoRazon`;
- `TestCodexStackExternalJobStatsSourceV0PriorizaLostSobreStarted`;
- `TestCodexStackV0OPESExternalWorkRESTCreaMicrotareaSinWriteSetLocal`;
- `TestMCPRunControlExecutorV0ResuelveRunPorJobExterno`;
- `TestRunFileStoreAppChangePersistsAfterRecreateAndReplacesV0`;
- `TestFileDomainWorkArtifactSubmissionLedgerV0PersisteYRecupera`.

## APP-CODEX-STACK-019

Objetivo: preparar smoke opt-in de shutdown cooperativo real de Codex.

Estado: hecho como prueba opt-in; pendiente ejecucion manual con Codex real por
operador.

Trabajo aplicado:

- nuevo `TestCodexStackRealShutdownCheckpointOptInV0`, desactivado salvo
  `ORQUESTA_CODEX_STACK_SHUTDOWN_SMOKE=1`;
- el smoke lanza `/nueva-app`, exige proceso Codex vivo, llama
  `POST /api/v0/server/shutdown` con `forced=false` y valida
  `orquesta_shutdown_request.json`;
- el agente recibe instrucciones en el `AGENTS.md` temporal para esperar la
  request y responder con `agent_shutdown_checkpoint_ack.json`;
- una segunda llamada de shutdown exige checkpoint registrado o falla con
  diagnostico de pending/checkpoint y logs compactos;
- cleanup final usa el conector de proceso real ya inyectado.

Validacion:

- `go test ./modulos/orquesta-app-codex-stack -run Test.*Shutdown.* -count=1`

Reglas cerradas:

- no lanza Codex real por defecto;
- no introduce defaults de modelo, HOME, CODEX_HOME, PATH ni runtime;
- el endpoint REST/MCP sigue siendo el borde; el core solo ve refs compactas.

## APP-CODEX-STACK-023

Objetivo: introducir la pieza minima para que Orquesta pueda supervisar una
sesion Codex y empujarla con `sigue` sin intervencion manual.

Estado: hecho como contrato unitario, adaptador de stack, API HTTP generica
`POST /api/v0/runs/supervise` sobre el ciclo legacy de agentes Orquesta, prueba
fake recursiva 1->2->4 sin llamadas manuales por nivel y prueba offline donde
el arbol se genera por `review -> split_task -> WorkflowTaskStore -> launch`.
Desde el corte Goal, esta tarea queda como historica/compatibilidad: una run
`goal_first` no debe pasar por este loop, sino por observacion de
`GoalWorkStateV0`.
Pendiente smoke real recursivo padre/hijo/nieto con proveedor y, si se quiere,
cliente CLI fino contra esa API.

Trabajo aplicado:

- `CodexSupervisorAgentLifecyclePortV0` con `LaunchV0` y `ContinueV0`;
- alias compatible `CodexSupervisorRuntimePortV0` para no romper el test/contrato
  inicial;
- `SuperviseCodexV0` con tick 1 de launch, ticks posteriores de continue y
  mensaje por defecto `sigue`;
- estados terminales `done`/`failed`, corte por contexto, error de runtime o
  `max_ticks`;
- historial compacto para que el borde superior pueda auditar que Orquesta
  empujo la sesion;
- `CodexSupervisorStackLifecycleV0`, que implementa
  `CodexSupervisorAgentLifecyclePortV0` sin stdin ni canal paralelo:
  - con `RunRef` legacy usa `DrainRunV0` para avanzar una run existente;
  - sin `RunRef` legacy usa `RunGlobalSupervisorV0` para avanzar la cola global;
  - ambos caminos legacy reutilizan `DrainRunV0` / `ContinueAppDirectorV0` / replan /
    outbox `LaunchRuntimeAgent` / dispatcher existente;
  - con contenedor `goal_first`, el supervisor no entra al drain: si existe
    `GoalWorkStateV0` devuelve `goal_first_observe_required`; si falta, devuelve
    `goal_first_state_missing` para reparacion explicita;
  - el snapshot traduce estados del loop del nucleo a
    `pending|running|stopped|done|failed` para que `SuperviseCodexV0` pueda
    seguir empujando hasta terminal o `max_ticks`;
- contrato publico neutral `orquesta.runs.supervisor.v0` en `orquesta-mcp`;
- ruta neutral `POST /api/v0/runs/supervise` en `orquesta-http-gateway` y
  `orquesta-app-gateway`;
- executor `CodexStackRunSupervisorExecutorV0` en este stack, que adapta el
  contrato neutral al supervisor Codex del borde sin filtrar Codex al gateway.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexSupervisorV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexSupervisor(StackLifecycle|RuntimeState)|TestCodexSupervisorV0'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunSupervisor|TestRegisterMCPTransportV0ExponeOperacionesExistentes'`
- `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway -run 'TestNewAppGatewayMuxV0RegistersConfiguredRoutes|TestRunControlYRunQueueAPIDeleganEnMCPPortsV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackRunSupervisorAPI|TestCodexSupervisorStackLifecycle|TestCodexSupervisorV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run '^TestCodexSupervisorStackLifecycleV0AvanzaArbolRecursivoFakeSinManualPorNivelV0$'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no tocar el core para esta pieza;
- `ContinueV0("sigue")` no es stdin ni runtime interactivo;
- el adaptador de stack avanza el flujo legacy: `RunGlobalSupervisorV0` /
  `DrainRunV0` / `ContinueAppDirectorV0`, observaciones ACK/progreso, replan y
  outbox `LaunchRuntimeAgent`;
- el flujo normal goal-first se conserva fuera del drain legacy y exige
  observacion/cierre por estado Goal;
- la API publica es de runs (`/api/v0/runs/supervise`), no de Codex; Codex vive
  solo en el executor de este stack;
- si hace falta arrancar otro Codex, se hace por `ExternalProcessAgentBatchExecutorV0`
  y `ProcessRegistry`, no por un lanzador paralelo;
- no declarar cerrada la recursion real hasta ejecutar el smoke con proveedor
  vivo; el fake de supervisor recursivo ya cubre parent/child refs, presupuesto
  global, profundidad/fanout, waits acotados y review/cierre causal sin Codex
  real.

## APP-CODEX-STACK-024

Objetivo: cerrar el smoke Codex real acotado con runner de tests requeridos y
cierre operativo causal.

Estado: hecho como `CODEX-REQTEST-REAL-E2E`; no CI por defecto.

Trabajo aplicado:

- smoke opt-in `./scripts/smoke_codex_real_required_test_runner.sh`;
- prueba directa `TestCodexStackRealRequiredTestRunnerEndToEndOptInV0`;
- una task del Director Operativo con `RequiredTests=go test ./...`;
- un unico agente Codex real bajo `WaitAgentRefs`;
- ACK/entrega, review causal aceptada, `RequiredTestRunner`,
  `RequiredTestEvidenceV0` durable y cierre de plan/run.

Validacion:

- `ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM=1`;
- `ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_EXECUTE_CODEX=1`;
- `ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CODEX_EXECUTION_CONFIRMED=1`;
- `./scripts/smoke_codex_real_required_test_runner.sh`;
- resultado documentado: PASS el 2026-05-22 en 64.49s.

Reglas cerradas:

- no lanza Codex real por defecto;
- no toca OPES ni core;
- no declara cerrado OPES real.

## APP-CODEX-STACK-025

Objetivo: inyectar un sanitizador local opt-in antes de enviar contexto a
agentes Codex premium/remotos.

Estado: completada localmente.

Trabajo aplicado:

- `CodexRuntimeConfigV0.ContextSanitizer`;
- `LocalSensitiveDataSanitizerV0`;
- policies publicas de packet cuando hay evidencia o revision de saneamiento;
- guardas de cierre cuando la evidencia exige revision humana/director;
- tests de no fuga de token, secreto, HOME ni material privado ambiguo.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- no hay sanitizador activo si la composicion no lo inyecta;
- no se elige proveedor, modelo, HOME ni transporte desde el adaptador;
- la evidencia no conserva el dato sensible.

## APP-CODEX-STACK-026

Objetivo: crear un indice/mapa local del stack para orientar estructura,
owners, estado de cierres reales y plan de secciones sin reabrir frentes ya
cerrados por la foto vigente.

Estado: hecho como documentacion local.

Trabajo aplicado:

- `docs/indice_mapa_2026-05-26.md` resume frontera, flujo, tabla de areas,
  estado operativo local y plan de secciones;
- `README.md` enlaza el mapa desde el estado actual;
- `docs/pruebas.md` deja claro que `CODEX-WAVE-REAL`,
  `CODEX-REQTEST-REAL-E2E` y `CODEX-RECURSION-REAL` estan cerrados salvo
  regresion demostrada.

Validacion:

- `git diff --check -- modulos/orquesta-app-codex-stack`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`

Reglas cerradas:

- el mapa no declara nuevos smokes cerrados sin evidencia;
- los huecos siguen clasificados como verificables, especialmente OPES temporal
  real de derivados/cierre;
- no toca core, runtime, servidor ni adaptadores de dominio.

## APP-CODEX-STACK-027

Objetivo: historico. La validacion de calidad de `domain_work` que bloqueaba
entregas por contenido se retiro del camino de ejecucion.

Estado: sustituida por politica de conservacion.

Trabajo aplicado:

- `domain_work` no debe parar agentes ni descartar entregas por palabras,
  longitud, placeholders, paths locales, nombres de campos o texto recuperable;
- una entrega que no sirva se conserva como borrador/evidencia/insumo y pasa a
  review/rework/director;
- los tests de bloqueo `domain_work_artifact_quality_gate_failed` fueron
  eliminados.

Validacion:

- `GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestDomainWorkDeliveryArtifactIntakeV0|TestDefaultDomainWorkArtifactSubmissionBuilderV0'`
- `GOCACHE=/tmp/orquesta-go-cache go test -count=1 ./modulos/orquesta-domain-work`

Reglas cerradas:

- no reintroducir rails de contenido en `domain_work`;
- no toca core, OPES bridge, runtime, servidor ni conectores externos;
- no declara cerrado el smoke real OPES de derivados/cierre.

## APP-CODEX-STACK-028

Objetivo: exponer adaptador real para el Director residente del servidor.

Estado: hecho local.

Trabajo aplicado:

- `RunCodexStackResidentDirectorV0` coordina runs ejecutables desde `RunQueue`,
  respeta el ranking neutral y acota el lote por
  `MaxRunsPerTick`/`MaxExecutions`;
- reutiliza `BuildContinueAppDirectorLoopRuntimeV0` para no duplicar el camino
  de `ContinueAppDirectorV0`;
- ejecuta `RunResidentDirectorBriefingLoopV0` con una fuente reentrable basada
  en outbox vivo;
- materializa consejos `DecisionCouncilPlanMaterializerV0` como accion externa
  idempotente cuando el run trae estado durable, contratos publicados y fase
  compatible con `CreateMicrotask`;
- `cmd/orquesta-server` inyecta el puerto solo con opt-in del servidor.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'CodexStackResidentDirector'`
- `go test -count=1 ./cmd/orquesta-server -run 'ResidentDirector|ServerConfigFromEnvV0ConfiguraResidentDirector'`

Reglas cerradas:

- no mete Codex, servidor ni OPES en el core;
- no convierte `close_or_idle` en cierre inventado del tick residente: delega
  en `ContinueAppDirectorV0`;
- no convierte `stop_max_steps` interno en rechazo de trabajo: el siguiente
  pulso reevalua outbox/stores vivos y sigue o espera;
- no dispara consejos por texto, logs ni `needs_director` generico: usa
  `Brainstorms`, `Votes`, contratos funcionales y tareas `task-council-*`
  persistidas;
- no usa filtros de contenido para decidir si un agente vale: conserva el
  trabajo en stores/outbox y deja rework/cierre al Director.

## APP-CODEX-STACK-029

Objetivo: centralizar runtime por proveedor y egress sanitizer en la
composicion, dejando OpenAI Privacy Filter como modelo local opt-in.

Estado: hecho local.

Trabajo aplicado:

- `EgressSanitizerConfigV0`;
- `PrivacyFilterModelConfigV0`;
- `RuntimeProviderConfigV0`;
- `CanonicalRuntimeProviderConfigsV0`;
- inyeccion automatica de `LocalSensitiveDataSanitizerV0` solo si la config
  canonica esta activa y no hay puerto explicito;
- proyeccion de providers con flags/refs compactas, sin filtrar rutas crudas;
- runbooks de operacion para configuracion y saneamiento.

Validacion:

- `go test -count=1 ./modulos/orquesta-context`;
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestLocalSensitiveDataSanitizerV0|Test.*Sanitizer|Test.*Egress'`.

Reglas cerradas:

- no mueve proveedor, modelo, HOME, transporte ni OpenAI al nucleo;
- no convierte el sanitizer en rail de contenido o bloqueo por palabras;
- no toca OPES productivo ni procesos externos.

## APP-CODEX-STACK-030

Objetivo: reconciliar lanzamientos Codex reclamados cuando `ProcessRegistry`
existe pero la proyeccion del run no tiene el agente iniciado.

Estado: hecho local.

Trabajo aplicado:

- `DrainRunV0` ejecuta `reconcileClaimedLaunchOutboxForProcessRegistryV0`
  despues de reconciliar capacidad huerfana y antes de regenerar outbox de
  lanzamiento faltante;
- el reconciliador lista `LaunchRuntimeAgent` pendientes/reclamados del ledger;
- reproyecta `AgentRequested` desde eventos durables si el run no lo refleja;
- reproyecta `AgentStarted` si el evento ya existe y falta en la proyeccion;
- si no hay `AgentStarted` pero si registro de proceso, registra
  `AgentStarted` desde `ProcessRegistry` y ACKea el outbox supersedido;
- conserva el cierre posterior en el flujo normal de ACK, perdida o replan.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestReconcile(ClaimedLaunchOutbox|OrphanCapacity)'`

Reglas cerradas:

- no toca `orquesta-core-workflow`;
- no interpreta texto ni logs de agente;
- no inventa entregas ni marca tareas como cerradas;
- convierte la inconsistencia en estado causal recuperable para que la
  supervision normal decida si el agente termino, se perdio o requiere rework.

## APP-CODEX-STACK-031

Objetivo: permitir replacements acotados cuando un followup de assessment ya
termino sin entrega.

Estado: hecho local.

Trabajo aplicado:

- `AssessmentReplanSourceV0` deja de bloquear una tarea por el primer followup
  terminal fallido;
- si el followup anterior sigue vivo, solicitado o iniciado, no duplica agente;
- si el followup anterior ya esta perdido/parado/fallido sin delivery, permite
  generar otro replacement;
- corta el bucle tras `3` followups terminales fallidos por tarea.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestAssessmentReplanSourceV0(ReintentaSiFollowupFalloTerminal|CortaBucleTrasTresFollowupsFallidos|NoEncadenaReemplazosParaMismaTarea)'`

Reglas cerradas:

- no reabre reemplazos mientras exista un agente materializado que aun pueda
  entregar;
- no deja el plan bloqueado esperando una entrega imposible;
- no mete reglas OPES en el core.

## APP-CODEX-STACK-032

Objetivo: impedir que un receipt OPES aceptado cierre paquetes finales de temario
sin evidencia editorial mínima.

Estado: hecho local.

Trabajo aplicado:

- el cierre operativo de `domain_work` OPES bloquea `finalize_topic_package`,
  `finalize_temario_package`, `close_temario_package` y equivalentes si el
  receipt no trae evidencia de extensión mínima y comunes/no aplicabilidad;
- la causalidad de artefactos usa `expected_artifact_type` cuando el bridge OPES
  lo declara;
- un paquete final necesita evidencia tipo `opes-extension-minima-passed` o
  `informe_extension_temario`, y evidencia de `matriz_reutilizacion_comunes` o
  `opes-common-master-not-applicable`.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestOperationalClosureSourceV0(NoCierraOPESFinalSinMinimosYComunes|CierraOPESFinalConMinimosYComunes)'`.

## APP-CODEX-STACK-033

Objetivo: impedir que el cierre goal-first acepte receipts DomainWork parciales.

Estado: hecho local 2026-06-28.

Trabajo aplicado:

- `domainWorkGoalReceiptClosureValidatorV0` exige `CompleteJob=true` en el
  record aceptado que cubre cada contrato de artefacto requerido;
- si solo existe receipt aceptado con `complete_job=false`, la closure bloquea
  con `domain_work_receipt_artifact_incomplete` y `NeedsRework=true`;
- el helper positivo de external-work goal-first marca el receipt completo para
  no esconder el requisito.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0ExternalWorkGoalFirst(CierraConReceiptAceptadoEnLedger|NoCierraReceiptDomainWorkIncomplete|BloqueaReceiptInventadoSinLedger)'`.

Reglas cerradas:

- no mete mínimos concretos en el núcleo genérico;
- solo afecta a OPES y paquetes finales;
- no bloquea borradores, fuentes, visuales, tests ni entregas parciales
  recuperables.
