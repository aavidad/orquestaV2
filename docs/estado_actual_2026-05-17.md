> Estado documental: histórico — runtime Orquesta pre-rebuild.
>
> No es autoridad de producto, backlog ejecutable ni evidencia de cierre para
> Orquesta V2. La autoridad V2 es `AGENTS.md` →
> `docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md` → `product/roadmap.json` →
> `product/capabilities.json` y `product/evidence/` →
> `docs/reconstruccion/ruta_total_100.md`. Para el estado operativo, consultar
> `docs/reconstruccion/HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md`.

# Estado actual de Orquesta - 2026-05-17

Este documento es la foto vigente para orientar trabajo nuevo. Los documentos
anteriores siguen siendo contexto historico o tecnico, pero no todos describen
la frontera actual del proyecto.

## Orden de autoridad documental

1. `AGENTS.md` y este documento fijan la foto vigente del repo y la frontera del
   nucleo.
2. `docs/guia_nucleo_orquestacion_2026-05-17.md`,
   `docs/corte_cierre_generico_director_operativo_2026-05-17.md` y
   `docs/matriz_pruebas_reales_y_smoke_2026-05-17.md` fijan handoff operativo,
   pruebas y estado de smokes.
3. `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` es backlog
   ejecutable. Debe respetar los cierres ya documentados en la matriz y abrir
   tareas nuevas solo por huecos o regresiones verificables.
4. Docs locales de modulo acotan trabajo del modulo, pero si contradicen esta
   foto deben marcarse como historicos/stale o sincronizarse. Las referencias
   obligatorias a `docs/reinicio_orquesta_v2/*` son historicas si el arbol no
   existe en la foto vigente; no deben inventarse ni restaurarse para cumplir
   contexto.
5. Docs historicos sirven como contexto y no como fuente canonica sin enlace a
   una fuente vigente.

## Estado vigente

Orquesta ya no debe leerse como una app cerrada de programacion. La direccion
vigente es convertirla en un nucleo reutilizable de orquestacion de agentes para
apps externas.

La regla central es: Orquesta aporta juicio mediante director y agentes; las
apps de dominio aportan datos, reglas, validadores, persistencia, UI/API y
ensamblado final.
OPES no es el producto base del nucleo: es una composicion consumidora. Cualquier
dominio externo, incluido OPES, entra por puertos, conectores y refs opacas, sin
compartir DB/filesystem interno ni mover reglas de producto al core.

Hasta nueva orden, rige una regla operativa sin filtros: Orquesta no debe
limitar artificialmente agentes ni cortar trabajo recuperable por rails,
cinturones, strings sueltos o heuristicas de logs. Cada Codex padre puede usar
hasta 6 subagentes y no hay limite global artificial de Codex padres o agentes
vivos salvo frontera dura del proveedor/runtime/OS o instruccion explicita del
operador. Alias, nombres cercanos, formato recuperable, contexto omitido por
presupuesto o palabras como `capacity` no justifican `stop_agent`, `failed`,
`garbage`, `capacity_limited` ni rechazo automatico. El Director conserva el
trabajo y ordena normalizacion, review, rework o replan. Solo quedan cortes
fuertes por seguridad, causalidad, refs imposibles, datos sensibles o efectos
externos no autorizados.
Los rails de detalle operativo/local son advisory tambien en produccion:
referencias a runtime, provider, modelo, ficheros de control, HOME como
diagnostico sin valor sensible o evidencia ref-only se conservan como evidencia
o nota, pero no paran agentes ni rechazan entregas. Si un rail de ese tipo
vuelve a bloquear trabajo valido, debe retirarse del camino de ejecucion y
quedar solo como observabilidad/revision.
La autoridad de decision no pertenece al rail: el Director o el agente
orquestador decide si una regla blanda se ignora, se elimina, se conserva como
diagnostico o se convierte en tarea de mejora. El codigo no debe convertir
heuristicas blandas en veto automatico.
Si un texto o artefacto no cumple el uso para el que se pidio, no se descarta
por defecto: se revisa para aprovecharlo total o parcialmente como otro
artefacto, borrador, insumo documental, evidencia, nota de revision o nueva
tarea derivada. Esta regla aplica especialmente a OPES, pero es criterio general
de dominios consumidores.

Esto implica:

- el core gobierna runs, fases, tareas, artefactos, revision, evidencias,
  capacidad, handoff, shutdown logico y supervision;
- la autovigilancia de CPU sostenida sin progreso pertenece a la composicion
  residente del servidor por puertos de telemetria y shutdown cooperativo; sus
  umbrales canonicos se publican en `effective_config` de `cmd/orquesta-server`
  y no son rails de contenido ni filtros para entregas de agentes;
  rework OrquestaV2 del 2026-06-11 la revalida con pruebas obligatorias pasadas,
  sin drenar OPES/TCAE ni mover la politica al core;
- la seleccion de modelo, runtime, cuotas y proveedor pertenece a composiciones
  y adaptadores, no al core puro;
- REST, MCP, web y CLI son adaptadores sobre puertos, no el dominio;
- la superficie preferente para que una IA maneje Orquesta es MCP:
  `orquesta.domain_work.v0`, tools de director/supervisor y resources compactos;
  el bridge HTTP local existe como adaptador fino, y un servidor MCP/MCPO real
  productivo sigue siendo adaptador opt-in, no logica de core;
- para OPES, cualquier efecto real por bridge sigue limitado por guardas de
  composicion: confirmacion explicita de instancia, destino sin credenciales ni
  query, loopback tratado solo como local no como temporal confirmado,
  productivo solo con opt-in y evidence ref compacta, filtro por `job_ref`,
  `job_type` o scope duro, limite bajo y ledger idempotente. La secuencia
  `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE` no habilita efectos por si sola:
  debe ir acotada por `program_id`, `topic_id`, `correlation_id` o filtro
  equivalente para no tocar jobs ajenos;
- desde el 2026-05-24, `orquesta.project.roadmap.v0` y
  `orquesta.contracts.shared.v0` incluyen freshness y refs vivas al backlog para
  no presentar estados `pendiente_*` historicos como backlog actual;
  el cierre T25 queda reflejado en el backlog con verificacion focal
  `go test -count=1 ./modulos/orquesta-mcp`;
- programacion, OPES y otros dominios son consumidores/composiciones;
- los conectores traducen contratos de dominio a trabajo orquestable y devuelven
  artefactos al dominio propietario;
- los conectores externos son puertos/adaptadores de composicion; si una fuente
  real de dominio o cierre no existe, se marca como pendiente verificable;
- los perfiles de trabajo (`code_study`, `implementation`, `refactor`,
  `required_tests`, `documentation`, `review`, `domain_work`) son neutrales y se
  transportan como `WorkProfileV0`/`WorkflowTaskV0.work_profile_kind`;
- las skills operativas viajan desde el corte 2026-06-08 como `SkillRefs`
  neutrales y opacas derivadas por perfil/tarea/composicion hasta runtime,
  packet y prompt. No meten proveedor, modelo, rutas locales, HOME, OAuth,
  token ni contenido completo de la skill dentro del core;
- ninguna app externa debe copiar internals, compartir DB/filesystem interno ni
  decidir plan, runtime, modelo o paralelismo sin director de Orquesta.
- la espina `DirectorCycleStepV0 -> director-runner -> director-scheduler ->
  core-workflow -> director-cycle-outbox` es la fuente de verdad documental para
  un tick acotado del Director V2: consulta outbox pendiente, construye tick
  compacto, obtiene comandos del scheduler, aplica workflow por puerto y
  registra nueva outbox. No es el loop progresivo historico de
  `app-director-service`, no es daemon y no demuestra por si sola composicion
  residente, smoke real, proveedor real ni OPES temporal.
- desde el corte 2026-06-25 se abre el camino goal-first para adelgazar el loop
  residente cuando el runtime ya aporta `goal` persistente. En ese modo, Codex
  Goal actua como Director operativo interno; Orquesta compila
  `GoalWorkSpecV0`, lanza/observa por adaptador opt-in, conserva receipt y
  valida cierre por evidencias. El status publico solo expone resumen compacto
  de spec/receipt/result/closure. El loop historico
  `app-director-service`/`PlanState` no se reabre para nuevas rutas goal-first
  salvo compatibilidad o regresion. Desde el 2026-06-27,
  `orquesta.apps.arrancar_director.v0` y `StartAppDirectorV0` normalizan modo
  vacio a `goal_first`: si falta backend Goal, devuelven
  `goal_backend_unavailable` y no caen al loop historico; la compatibilidad
  antigua exige `director_execution_mode=legacy_director_loop` explicito.
- para `AppSpecV0`, la ruta publica operativa preferente es
  `orquesta.apps.arrancar_director.v0`, que delega en
  `orquesta-app-director-service`. `orquesta.apps.preparar_orquestacion.v0` y
  `orquesta.apps.ejecutar_orquestacion.v0` quedan como preview/compatibilidad
  sobre `orquesta-app-runner` y publican `route_policy` en resultados y
  descriptores compactos; si el caller declara que necesita Director V2, el
  runner bloquea con `director_v2_required`.

## Nucleo neutral

El nucleo neutral vigente es el plano comun de orquestacion:

- contratos genericos de trabajo externo y artefactos;
- director, agentes, scheduler, fases, tareas y evidencias;
- seleccion de capacidad y contratos para que composiciones externas conecten
  runtime, proveedor, modelo y cuotas sin contaminar el core;
- control de progreso, cuota, bloqueos, loops, handoff y cierre;
- registro de entregas y estadisticas compactas.

La composicion Codex/programacion no define el nucleo. Es una app consumidora que
usa Orquesta para crear, modificar, auditar, migrar o desplegar software. Debe
entrar por contratos como `AppSpecV0` o trabajo de dominio equivalente.
El stack Codex puede ejecutar perfiles neutrales, pero no define su taxonomia.

La composicion OPES tampoco define el nucleo. OPES conserva oposiciones,
programas, temas, taxonomia, fuentes, reglas pedagogicas, validadores,
persistencia y ensamblado. Cuando necesita planificacion documental, pide trabajo
a Orquesta (`plan_temario`, `plan_tema`, `plan_documento`, etc.) y valida despues
los artefactos recibidos.
OPES aporta el dominio; el nucleo conserva el perfil neutral y la orquestacion.

## Evidencia y pruebas documentadas

La evidencia operativa mas reciente revisada es la prueba limpia OPES-Orquesta
`plan_tema` documentada el 2026-05-15:

- OPES envio un job externo neutral `plan_tema`.
- Orquesta arranco un agente Codex real (`gpt-5.5`, `xhigh`).
- Orquesta recupero/canonicalizo un `document_plan`.
- OPES recibio exactamente un artefacto `document_plan`.
- El job paso a `completed`.
- OPES creo 21 jobs derivados: 10 `draft_content_block`, 5
  `generate_visual_asset`, 1 `validate_topic`, 1 `review_legal`, 1
  `review_pedagogical`, 2 `review_quality` y 1 `assemble_topic`.

Pruebas documentadas como pasadas en ese corte:

- `go test -count=1 ./modulos/orquesta-domain-work`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCanonicalDomainWorkDeliveryPayloadBodyV0NormalizaPlanOPESReal|TestDefaultDomainWorkArtifactSubmissionBuilderV0CanonicalizaDocumentPlanAliasesOPES|TestDefaultDomainWorkArtifactSubmissionBuilderV0IdempotenciaEstablePorJobYArtefacto'`
- `go test -count=1 ./modulos/orquesta-opes-bridge`
- `go test -count=1 ./modulos/orquesta-runtime-codex -run 'TestCodexExecResolverV0PromptUsaControlFilesDelRuntime'`
- `go test -count=1 ./...`

Nota: esas pruebas son evidencia del corte documentado, no una reejecucion de
este cambio de documentacion. El arbol de trabajo actual contiene muchos cambios
concurrentes.

El corte del 2026-05-18 anade cobertura focal para `plan_temario` de OPES como
`document_plan`: mapper, policy editorial OPES, `xhigh`, drain fake por REST,
flujo local con Codex fake y envio por `DomainWork`. Ese mismo dia se cerro un
smoke real acotado de Operario por REST contra OPES temporal: OPES importo el
programa real, creo `job_ref=9784a562f074769a08043707fdd79eb2`, Orquesta lo
dreno por bridge residente, lanzo Codex real `gpt-5.5` con `xhigh`, envio
`document_plan` a OPES, OPES completo el job y creo 20 derivados pendientes.
No se ejecutaron los derivados en ese smoke. Despues de ese corte se anadio
automatizacion de pases por `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`: el bridge
residente consulta tipos en orden y solo drena la primera fase que OPES siga
mostrando como `pending`, usando el ledger para no relanzar jobs ya enviados.

## Estado real tras el primer corte del 2026-05-17

El primer corte ya no esta solo en documentos:

- `modulos/orquesta-director-operativo` contiene el contrato puro del Director
  Operativo V1. Modela `OperationalDirectorRequestV0`,
  `OperationalDirectorPlanV0`, `OperationalDirectorWaveWorkV0`, modos
  `programming` y `domain_work`, contexto insuficiente, presupuestos,
  write-set, tests requeridos y delegacion recursiva gobernada.
- `modulos/orquesta-director-operativo/README.md`,
  `modulos/orquesta-director-operativo/docs/contratos.md` y
  `modulos/orquesta-director-operativo/docs/pruebas.md` son la referencia local
  para ese contrato.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  es el materializador actual: toma un plan `ready`, construye olas con
  `BuildOperationalDirectorWaveWorkV0`, selecciona items `launch_subagents`,
  guarda `WorkflowTaskV0` y emite comandos `CreateMicrotask` por el workflow.
- `modulos/orquesta-orchestration-core/operational_director_materializer_v0_test.go`
  demuestra dos guardas basicas: crea microtarea accionable para un plan listo y
  no lanza un plan bloqueado por contexto insuficiente.
- `WaitAgentRefs` existe en los loops progresivos y en
  `modulos/orquesta-app-codex-stack` para esperar agentes externos concretos.
  Ya puede elevarse a contrato de cohorte/ola mediante
  `WorkflowTaskWaitStateV0`, que guarda causa, refs de tasks, agentes objetivo y
  pendientes. `ContinueAppDirectorV0` ya puede recuperar ese wait state
  persistido si esta en `waiting` y no ampliar la espera a toda la ola/cohorte;
  cuando `wait_subagents` se consume, el estado de espera se marca `continued`
  por puerto de forma idempotente.
- `StartAppDirectorV0` tambien puede recibir un `OperationalDirectorPlanV0`
  inicial listo, siempre que la composicion aporte contratos funcionales
  explicitos. En ese modo publica causalmente voto, decision, fase y contratos
  antes de materializar `launch_subagents`, deriva wait de ola/cohorte y
  persiste `PlanState`/wait state por puertos. Desde el corte Goal, el modo
  vacio normaliza a `goal_first`: si falta backend Goal, la composicion debe
  devolver `goal_backend_unavailable`. El bootstrap historico sin plan directo
  sigue vigente solo para `legacy_director_loop` explicito o composiciones no
  migradas a Goal; esto no mete runtime, Codex ni producto en el servicio.
- P1 WaitAgentRefs queda cerrado para el stack Codex: el mismo scope limita
  pending, wait e ingesta de ACK/deliveries. `DrainRunRequestV0.WaitAgentRefs`
  llega al request de observaciones, el `DeliverySource` queda envuelto/filtrado
  y `DrainRunV0` vuelve a filtrar antes de aplicar observaciones. El source
  Codex acota descriptors antes de leer ACKs, y `domain_work` filtra submit y
  recovery antes de producir efectos. La evidencia focal incluye
  `TestRunProgressiveLoopV0PropagaWaitAgentRefsAObservationSource`,
  `TestCodexDeliveryObservationSourceV0WaitAgentRefs*`,
  `TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope` y
  `TestDomainWork*WaitAgentRefs`; el modo legacy con `WaitAgentRefs` vacio sigue
  observando el run completo.
- `docs/corte_director_funcionando_tarde_2026-05-17.md` deja el handoff de tarde:
  que ruta puede usarse hoy, que comandos ejecutar, que P0 queda cerrado y que
  queda pendiente despues del cierre de scope.
- `docs/corte_cierre_generico_director_operativo_2026-05-17.md` deja el handoff
  siguiente: P1 `WaitAgentRefs` queda cerrado y el foco pasa a cierre causal
  offline generico del Director Operativo. Lo que no tenga implementacion y test
  claro se trata como pendiente verificable.
- El cierre operativo generico ya tiene un tramo offline en codigo:
  `OperationalDirectorClosureV0` valida entrega registrada, review aceptada y
  evidencias de tests requeridos antes de cerrar task, abrir/registrar
  validacion final y cerrar run. Tambien valida por historial de eventos que la
  entrega, review request, resultado aceptado y accepted review pertenezcan a la
  misma cadena causal. `ContinueAppDirectorV0` lo invoca despues del loop cuando
  queda quiescent y la composicion inyecta `OperationalClosureSource`.
- `RequiredTestEvidenceV0` ya existe en `orquesta-orchestration-core` como
  comprobante durable minimo de tests requeridos. Guarda `run_ref`, `task_ref`,
  comando exacto, status `passed`/`failed` y refs causales de entrega, review
  request, review result y accepted review. `OperationalDirectorClosureV0`
  acepta solo evidencias `passed` que coinciden con cada
  `WorkflowTaskV0.RequiredTests` y con la cadena causal del cierre; un summary
  textual no cuenta como test ejecutado. La evidencia debe incluir artefactos en
  `evidence_refs`, su ref debe estar pedida por el cierre y el cierre exige
  `EventSink` para no mutar runs sin eventos persistidos.
- `orquesta-state-file` ya persiste `RequiredTestEvidenceV0`; el stack Codex lo
  usa como fuente de refs para cierre desde `RequiredTestEvidenceRefs` del
  request o, por compatibilidad, cuando el review result contiene esas
  evidencias, ignorando refs mixtas que no correspondan a evidencia de test.
  El guardado es idempotente solo si el payload coincide; la misma ref con
  payload distinto es conflicto. El `PlanState` ya consume esas evidencias en
  `run_required_tests`: con `passed` causal avanza a `replan_or_close`, con
  `failed` bloquea como `required-tests-failed`, y si falta evidencia causal sin
  runner efectivo registra `QualityGateRecorded(blocked)` idempotente por
  `required-tests-evidence-missing` sin crear replan automatico. Las refs
  aceptadas se pasan al cierre. Esto no equivale a exigir replan ante latencia
  o infra: la correccion puede ser aportar evidencia causal posterior.
- `orquesta-app-codex-stack` ya cablea `OperationalClosureSource` para el caso
  Director Operativo: lee `WorkflowTaskStore` y `LoadRunEventsV0`, respeta el
  scope `WaitAgentRefs`, exige cadena delivery/review requested/review result
  accepted/accepted review y prueba deliveries en orden estable. Si el cierre
  produce issues, `ContinueAppDirectorV0` los expone en
  `operational_closure_issues` en vez de tragarlos silenciosamente.
  scope resuelto (`WaitAgentRefs`) y solo actua sobre tasks con metadatos
  `operational_director`/contrato funcional, para no cerrar flujos legacy.
- `orquesta-app-codex-stack` incorpora desde el 2026-05-18 un supervisor Codex
  unitario: `SuperviseCodexV0` lanza en el primer tick y llama
  `ContinueV0(ctx, "sigue")` en ticks posteriores hasta `done`, `failed`,
  cancelacion/error o `max_ticks`. Este contrato no toca el core y debe leerse
  como ciclo de vida de agente: el adaptador real tiene que avanzar
  supervisor/drain/replan/outbox `LaunchRuntimeAgent`, registry, ACK y progreso
  ya existentes.
- Desde el corte del 2026-05-22 el conector Codex especializa el objetivo de
  `AgentStartTaskV0` segun `WorkflowTaskV0.WorkProfileKind`: `code_study`,
  `implementation`, `refactor` y `required_tests` mantienen `write_set`,
  `required_tests` y linaje, pero no reciben todos el mismo texto generico de
  "implementar".
- Desde el corte posterior del 2026-05-22, la entrada
  `POST /api/v0/autoprogramming/prepare-run` en `orquesta-app-codex-stack`
  persiste la run/tareas y tambien la deja como candidato de la cola global del
  stack Codex cuando la clasificacion requiere loop legacy. Desde el 2026-06-26,
  si `goal_migration.status=goal_ready` y la composicion tiene
  `GoalLauncher` + `GoalStateStore`, crea un run contenedor sin tareas legacy,
  lanza/persiste `GoalWorkStateV0`, publica `goal{...}` y no encola `ready`;
  sin backend conserva specs completas solo como handoff interno y publica
  `goal_spec_summaries[]` sin `run_ref`. El supervisor residente del servidor o
  `POST /api/v0/runs/supervise` solo deben arrancar la rama legacy; esto sigue
  siendo politica de composicion Codex, no contrato del nucleo ni de MCP/gateway.
- Desde el 2026-06-08, `cmd/orquesta-server` puede inyectar por opt-in un
  Director residente real sobre el stack Codex mediante `ResidentDirectorPortV0`
  (`8944ca9f`, afinado en `f619e899`): ranking/cola neutral,
  `BuildContinueAppDirectorLoopRuntimeV0`, briefing reentrable y cierre externo
  delegado a `app-director-service`. Esto no declara smoke largo real ni wiring
  live de consejo/votacion; ambos siguen pendientes con evidencia propia.
- Tambien existe harness de ola/cohorte amplia en el stack Codex:
  `TestCodexStackOperationalWaveFakeRuntimeV0` cierra una ola de tres tasks con
  runtime fake, `WaitAgentRefs`, review causal, runner de tests y cierre; el
  modo real opt-in `TestCodexStackRealOperationalWaveOptInV0` queda documentado
  como `CODEX-WAVE-REAL` y ya se ejecuto con proveedor, guardando evidencia de
  varios agentes Codex vivos, review causal, runner y cierre.

Lo pendiente no debe confundirse con lo hecho:

- en la espina `DirectorCycleStepV0 -> runner -> scheduler -> workflow ->
  cycle-outbox`, el codigo offline ya cubre el tick neutral con outbox pendiente
  como corte de seguridad; lo pendiente para esa linea se clasifica aparte:
  composicion residente/restart (`T65`), smoke neutral de proceso real (`T66`)
  y proveedor real si aplica. OPES temporal real de derivados/cierre quedo
  cerrado despues por la ruta goal-first documentada en
  `docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`;
  no se debe reabrir como "pendiente generico" review/tests/rework/cierre ya
  cubiertos por PlanState y pruebas offline;
- el `PlanState` ya cubre la salida positiva
  `review_deliveries -> run_required_tests/replan_or_close` cuando existe la
  cadena causal `DeliveryRegistered -> ReviewRequested ->
  ReviewResultRecorded(accepted) -> ReviewAccepted`, y `run_required_tests`
  consume `RequiredTestEvidenceV0` causal para avanzar o bloquear; el codigo
  local observa review negativa con `ReworkRequested` y
  `ReplanDecisionRecorded` en el `PlanState` con prueba focal, y reabre
  `wait_subagents` en una reentrada posterior cuando los followups causales ya
  estan materializados; el cierre
  operacional ya marca el state como `closed` o `blocked` con
  `closure_reason`, y si `replan_or_close/running` queda quiescent con outbox
  pendiente bloquea `operational-closure-outbox-pending` sin invocar la fuente
  de cierre; el runner/adaptador de tests por puerto existe y tiene replay focal
  sin reejecucion externa; el cierre bloqueado por
  `required_test_evidence_refs` ya puede reabrir `wait_subagents` cuando aparece
  un followup causal reflejado, y el cierre insuficiente causal con
  `closure_ref`/`validation_ref` ya emite quality gate + replan retry y reabre
  solo el followup causal reflejado, tambien en ola multitarea cuando el fallo
  identifica una unica task causal del scope; el smoke Codex real acotado con
  runner ya quedo cerrado por `CODEX-REQTEST-REAL-E2E` con un agente vivo,
  review causal, `RequiredTestEvidenceV0` y cierre de plan/run; el smoke
  no-OPES temporal `EXT-NO-OPES` ya cerro app HTTP/file externa con submitter
  real opt-in, `codex-fake`, review, tests requeridos y plan cerrado;
  `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` ya cerraron Codex real amplio y
  recursivo; OPES temporal real de derivados/cierre quedo cerrado despues por
  goal-first hasta `completed_syllabus_package`, con pendiente residual solo de
  revalidar optimizaciones posteriores de contexto/coste y QA editorial fuera
  del smoke;
- si una composicion real distinta del stack Codex todavia no inyecta una fuente
  `OperationalClosureSource`, el cierre queda pendiente de wiring real en esa
  composicion;
- `wait_subagents` ya tiene scope operativo, estado durable de espera, plan
  state inicial, reentrada por `operational_director_plan_ref`, avance a
  `review_deliveries` cuando el wait queda consumido y avance de review positiva
  a tests/replan por eventos causales del scope; tests requeridos durables ya
  avanzan o bloquean por evidencia causal; la observacion negativa de review y
  el cierre/bloqueo posterior del PlanState estan cubiertos offline; el cierre
  de ola multitarea ya progresa por reentradas sin bloquear por
  `run.open_tasks`; cierre insuficiente causal ya tiene replan retry acotado,
  incluso con una unica task fallida dentro de ola multitarea; la ola/cohorte
  amplia Codex real esta cerrada por `CODEX-WAVE-REAL` y la recursion real por
  `CODEX-RECURSION-REAL`. El caso Codex real acotado con runner esta cerrado
  por `CODEX-REQTEST-REAL-E2E`, y el caso no-OPES temporal con runtime fake esta
  cerrado por `EXT-NO-OPES`;
- la recursion Codex real ya tiene contrato unitario `launch -> sigue -> done`,
  adaptador de stack para supervisor/drain, prueba fake que empuja un arbol
  recursivo sin llamadas manuales por nivel y smoke real con proveedor Codex,
  ACK/entregas vivas, hijos/nietos con parent/child refs, presupuesto global,
  profundidad/fanout y review causal;
- OPES tiene evidencia de `plan_tema` y `plan_temario`; el bridge ya automatiza
  derivados por fases con `JOB_TYPE_SEQUENCE`, pero los derivados de review,
  validacion y ensamblado aun necesitan smoke real completo con guardas;
- no hay que borrar piezas antiguas sin revisar referencias. Si un documento o
  modulo queda historico, marcarlo como historico y enlazar al documento vigente.

## Documentos vigentes de referencia

- `principio_orquesta_piensa_director.md`: decision transversal vigente sobre
  reparto de juicio entre Orquesta y apps de dominio.
- `guia_nucleo_orquestacion_2026-05-17.md`: mapa operativo de piezas,
  invariantes y reglas para futuros agentes.
- `director_operativo_v1_2026-05-17.md`: contrato vigente del Director
  Operativo V1, corte razonable, espera por cohortes/oleadas y recursion
  gobernada.
- `corte_director_funcionando_tarde_2026-05-17.md`: handoff operativo para
  dejar Orquesta usable hoy sin confundir lo probado con la recursion completa.
- `corte_cierre_generico_director_operativo_2026-05-17.md`: handoff vigente del
  siguiente corte del Director: review por ola, tests durables, replan/close y
  plan state completo como ciclo causal offline generico.
- `matriz_pruebas_reales_y_smoke_2026-05-17.md`: matriz viva de smokes reales,
  guardas opt-in y pruebas focales.
- `autoprogramacion_orquesta_pendientes_2026-05-23.md`: backlog ejecutable de
  automejora; queda subordinado a esta foto y a la matriz para no relanzar casos
  ya cerrados con evidencia.
- `corte_opes_como_consumidor_orquesta_2026-05-18.md`: corte vigente de OPES
  como consumidor generico de Orquesta, con `plan_temario -> document_plan`,
  policy editorial OPES y ruta REST/MCP generica aclarada.
- `runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`: runbook acotado
  para probar `plan_temario` de operadores contra OPES temporal.
- `resultado_prueba_opes_orquesta_plan_temario_operario_2026-05-18.md`:
  evidencia real acotada de OPES temporal, Orquesta REST bridge, Codex real
  `xhigh`, `document_plan` entregado y derivados creados.
- `corte_supervisor_codex_director_2026-05-18.md`: corte vigente de la pieza
  `launch -> sigue -> done` para sesiones Codex supervisadas por Orquesta.
- `orquesta_goal_first_codex_2026-06-25.md`: corte vigente para Codex Goal como
  Director operativo interno y Orquesta como plano de gobierno externo.
- `arquitectura_plataforma_agentes_2026-05-13.md`: decision de evolucion hacia
  plataforma/nucleo de agentes con dominios consumidores.
- `resultado_prueba_opes_orquesta_plan_limpia_2026-05-15.md`: evidencia viva de
  integracion OPES-Orquesta para `plan_tema`.
- `informe_revision_hexagonal.md`: diagnostico historico util para detectar
  patrones de deuda, pero no es snapshot actual. El repo activo ya no conserva
  varias piezas antiguas que analiza.
- `db_hexagonal_refactor_plan.md`: plan historico util para entender la
  intencion de extraer policy de persistencia, no prueba de que exista hoy un
  `db/` activo en la raiz.

## Documentos historicos o stale

Leer con cuidado:

- `orquesta_v1_vision.md`, `orquesta_v1_roadmap.md` y `BIBLIA_APP_ORQUESTA.md`
  son utiles como historia y vision, pero cualquier lectura centrada en
  "app de programacion" queda subordinada al nucleo neutral reutilizable. Desde
  el 2026-05-26, `BIBLIA_APP_ORQUESTA.md` debe autoidentificarse en cabecera
  como `doc_estado=historico-stale` y enlazar a `AGENTS.md`, `README.md`, esta
  foto, la guia de nucleo y el backlog vivo como sustitutos vigentes.
  Sus secciones internas V1 llamadas "fuentes de verdad", "doctrina" o
  "canonico" son contexto historico; no reabren autoridad documental ni
  autorizan planificacion automatica sin enlace nuevo a esta foto.
- `integracion_opes_orquesta_2026-05-13.md`,
  `estado_integracion_opes_orquesta_2026-05-13.md`,
  `corte_integracion_opes_orquesta_2026-05-13.md` y
  `resultado_prueba_opes_orquesta_plan_2026-05-14.md` quedan superados, para
  estado OPES, por la prueba limpia `plan_tema` del 2026-05-15 y el corte
  OPES-consumidor del 2026-05-18.
- `manual_programador.md` contiene deriva documentada: referencias tecnicas y
  URLs que no deben usarse como fuente canonica sin revision.
- `uso_actual_app_orquesta.md` conserva valor operativo, pero debe leerse junto
  con la politica servidor-primero y con este estado actual.
- Los recuentos de deuda en informes de arquitectura son utiles para priorizar,
  pero no deben tratarse como cifras actuales si no se recalculan.
- Cualquier documento que hable del antiguo `cmd/db/internal`, de
  `ensureLocalDB` o de rutas SQLite acopladas pertenece al periodo previo al
  saneamiento modular y debe verificarse contra el arbol actual antes de usarse.
- Los snapshots SQLite/DB v1, incluido el ref relativo
  `backups/legacy-sqlite-20260422/orquesta.db`, son evidencia forense en
  cuarentena. No son prerequisito operativo, fuente viva de persistencia ni
  entrada programable para agentes sin decision explicita del director.

## Riesgos abiertos

- La arquitectura hexagonal no esta cerrada: `cmd/` sigue concentrando
  composicion de producto y quedan riesgos de policy dentro de adaptadores de
  runtime/persistencia. No hay que reintroducir el antiguo `db/` por merge.
- El modo servidor-primero existe como direccion, pero aun convive con rutas
  locales/fallback que deben quedar como recuperacion explicita.
- OPES ya valida la composicion de planificacion, pero faltan controles finos
  para pruebas focales y arranque completo de temas derivados.
- Los smokes OPES deben seguir con guardas opt-in, instancia temporal y scope
  por tipo de job o `job_ref` solo para no tocar productivo ni jobs ajenos. Ese
  scope no es filtro de agentes, capacidad o entregas. El caso peligroso es
  drenar una cola OPES real sin `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni
  `ORQUESTA_OPES_BRIDGE_JOB_REF`.
- La espera por agentes ya puede derivar `WaitAgentRefs` desde cohortes u olas
  declaradas, persistirse como `WorkflowTaskWaitStateV0` y acotar la ingesta de
  ACK/deliveries en el stack Codex. El ciclo durable offline ya cubre wait
  consumido, wait expirado, review positiva/negativa, tests requeridos,
  replan/cierre y varios replays; esa garantia ya se repitio con Codex real en
  ola/cohorte amplia y recursion. Para nuevas regresiones, los smokes reales
  quedan opt-in y no deben mezclarse con OPES.
  Esperar todos los procesos vivos del run vuelve a introducir cuelgues falsos.
- El cierre operativo desde `ContinueAppDirectorV0` depende de que el loop haya
  quedado quiescent y de una `OperationalClosureSource` real. Sin esa fuente, no
  hay que anunciar cierre productivo aunque exista el cerrador offline generico.
- La delegacion recursiva con Codex aun no esta cerrada end-to-end. No anunciar
  "director recursivo completo" hasta tener evidencia real con hijos, nietos,
  limites y review.
- El contrato i18n y su cargador activo tenian brechas historicas documentadas.
  Corte T75 del 2026-05-25: `orquesta-i18n-docs` queda declarado owner activo
  de bundles, loader shape y documentacion generada; factory, web y MCP deben
  consumir su proyeccion `ActiveI18nDocsCompositionOwnerV0` o el plan
  `AppI18nDocsPlanV0` en vez de crear otro owner paralelo.
- Seguridad operativa, multi-tenant, TLS/mTLS, RBAC y auditoria fuerte siguen
  siendo frentes de cierre antes de considerar la plataforma lista para despliegue
  amplio.
- Hay cambios concurrentes en curso; cualquier nuevo corte debe tener write-set
  explicito y no pisar hotspots ajenos.

## Siguiente incremento

El cierre causal offline generico ya tiene evidencia para los caminos listados
abajo. `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` ya cubren Codex real amplio y
recursivo en la matriz; el frente no cubierto por este bloque sigue siendo OPES
temporal real de derivados/cierre o cualquier regresion nueva con prueba propia:

- [x] Review por ola/cohorte: `review_deliveries` usa el scope activo y cadena
  causal de task/delivery/review aceptada. No avanza una ola por entregas de
  otra ola ni por cualquier ACK del run.
- [x] Tests durables por evidencia: en modo `programming`,
  `run_required_tests` consume `RequiredTestEvidenceV0` causal antes de cierre,
  avanza con evidencias `passed`, bloquea con `required-tests-failed` y no acepta
  refs de otra review. El runner/adaptador por puerto ya existe como opt-in; si
  falta evidencia y no hay runner efectivo, el `PlanState` bloquea con
  `required-tests-evidence-missing` y puede reentrar cuando aparezca evidencia
  causal. En `domain_work`, usar artefactos y validadores de dominio sin inventar
  tests de programacion.
- [x] Replan negativo: review negativa, tests fallidos de un task y reentrada
  tardia a followups ya materializados estan cubiertos. Tambien queda cubierta
  la reentrada de cierre bloqueado por `required_test_evidence_refs` cuando
  aparece despues un followup causal reflejado, y cierre insuficiente causal ya
  produce replan retry con refs de task/delivery/review aceptada, tambien si el
  scope contiene varias tasks pero el cierre fallido identifica una unica task.
  Outbox pendiente ya bloquea el cierre con causa durable sin llamar la fuente.
  Un blocker nuevo sin decision causal debe documentarse como caso nuevo con
  prueba propia, no como pendiente abierto de este corte.
- [x] Plan state inicial: persistir estado vivo del plan con step activo
  `wait_subagents`, ola/cohorte activa, tasks, agentes, pendientes y `wait_ref`.
- [x] Plan state reentrada inicial: `ContinueAppDirectorV0` lee
  `operational_director_plan_ref` y recupera scope de wait sin mirar agentes
  vivos globales; si hay `WaitStateStore` y el step trae `wait_ref`, prefiere el
  `WorkflowTaskWaitStateV0` persistido validado antes de recomputar scope. El
  paso a review marca ese wait state como `continued`.
- [x] Plan state wait expirado: si `ManagedProgressiveLoopV0` agota
  `MaxExternalWaits` con `wait_external`, `ContinueAppDirectorV0` transporta esa
  senial, marca el wait state `expired` por puerto cuando existe y bloquea el
  `PlanState`/step con `external-wait-exhausted` sin ampliar scope.
- [x] Plan state wait limpiado: si el cierre operativo exitoso cierra el
  `PlanState`, cualquier `WorkflowTaskWaitStateV0` antiguo aun `waiting`
  referenciado por el plan se marca `cleared` por puerto, sin tocar waits ya
  `continued` o `expired`.
- [x] Plan state wait consumido: pasar de `wait_subagents` a
  `review_deliveries` cuando los agentes pendientes entregaron.
- [x] Plan state restante: review negativa observada, intentos de replan,
  reentrada tardia a followups causales ya materializados, bloqueo por evidencia
  de test faltante, razon de cierre/bloqueo, reentrada de cierre bloqueado por
  falta de evidencia requerida, replan de cierre insuficiente causal, bloqueo
  por outbox pendiente y tests durables ya quedan persistidos en el ciclo
  fake-runtime probado.
- [x] Evento idempotente del ciclo probado: review, tests, cierre, bloqueo de
  cierre y replan causal cubierto no duplican refs/eventos en las pruebas
  focales y replays listados en la matriz.

La evidencia debe quedar reflejada en
`docs/matriz_pruebas_reales_y_smoke_2026-05-17.md`. Si una composicion no tiene
`OperationalClosureSource` o validador real, documentarlo como pendiente
verificable para esa composicion.

## Siguiente ruta

1. Mantener la frontera conceptual: core neutral primero, composiciones despues.
2. Mantener cerrado el frente OPES temporal real de derivados/cierre por
   goal-first y no relanzarlo salvo regresion demostrada. El smoke largo del
   2026-06-28 cerro 24/24 jobs hasta `completed_syllabus_package`; queda como
   residual reejecutar con optimizaciones posteriores de contexto/coste si se
   necesita medir eficiencia. El smoke Codex real acotado con runner ya esta
   cerrado en `CODEX-REQTEST-REAL-E2E`, el no-OPES temporal con runtime fake en
   `EXT-NO-OPES`, la ola/cohorte amplia en `CODEX-WAVE-REAL` y el arbol
   recursivo en `CODEX-RECURSION-REAL`.
3. Consolidar `orquesta-domain-work` y los contratos de artefactos como entrada
   comun para OPES, programacion y futuros dominios.
4. Seguir vaciando policy de `cmd` y adaptadores concretos hacia
   servicios/puertos de aplicacion.
5. Endurecer server-first: CLI/web/MCP como clientes finos del control plane.
6. Reforzar pruebas de frontera: contratos de dominio, no duplicacion de
   artefactos, idempotencia, canonicalizacion y ausencia de regresion
   arquitectonica.
7. Separar en documentacion lo vigente de lo historico antes de usar cualquier
   manual como guia operativa.
