# Pruebas locales: orquesta-director

Registra pruebas obligatorias del modulo.

## Pruebas previstas

```text
Caso: DIR-P022 rework split crea microtareas
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director -run TestBuildReplanFollowupsV0ReviewReworkSplitConstruyeMicrotareas
Evidencia esperada: BuildReplanFollowupsV0 recibe `source_kind=review_rework`, `split_task`, OpenPhase y candidates CreateMicrotask explicitos; devuelve RecordReplanDecision, OpenPhase y dos CreateMicrotask sin AskDirector ni capacidad/agente inventados.
Ultima ejecucion: 2026-05-10, ok.
Riesgos: no aplica comandos ni persiste detalle completo de tarea; esa cobertura vive en orquestacionnucleoapp.
```

```text
Caso: DIR-P001 bootstrap desde AppSpec valida
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: AppSpecV0 + BacklogInicialPropuestoV0 validos producen registro aceptado compacto, StartRun y RunStarted sin DB/runtime.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-factory
Riesgos: acoplar director a internals de factory/core/workflow.
```

```text
Caso: DIR-P002 bootstrap rechaza idempotency_key vacia
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: error publico `director_bootstrap_invalido` o propagado sin panics ni efectos externos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-factory
Riesgos: crear runs duplicados o no deterministas.
```

```text
Caso: DIR-P003 salida compacta sin detalles prohibidos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: JSON del resultado no contiene DB, SQL, runtime, provider, OAuth, HOME, token ni secretos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-core ./modulos/orquesta-core-workflow ./modulos/orquesta-factory
Riesgos: filtrar AppSpec/backlog completos hacia workflow o outbox.
```

```text
Caso: DIR-P004 smoke app simple hasta fase descubrimiento
Tipo: smoke
Comando: go test -count=1 -v ./modulos/orquesta-director -run TestSmokeOrquestaAppSimpleV0ArrancaRunYAbreDescubrimiento
Evidencia esperada: una app simple "Lista simple de tareas" genera AppSpec valida, backlog, registro en borrador, StartRun, replay durable y OpenPhase(descubrimiento) sin outbox ni efectos externos.
Ultima ejecucion: 2026-05-04, ok; salida: estado=borrador, fase=descubrimiento, fases=5, microtareas=5, contratos=5, eventos=2, outbox=0.
Riesgos: esto no programa codigo real; solo demuestra arranque durable y fase inicial.
```

```text
Caso: DIR-P005 gobierno progresivo de capacidad y agentes logicos
Tipo: integration_fake_runtime
Comando: go test -count=1 -v ./modulos/orquesta-director -run TestProgressiveOrquestaV0GobiernaCapacidadArrancaYParaAgentesLogicos
Evidencia esperada: el run pasa por brainstorming, voto, decision, planificacion, publica FunctionContract, crea microtarea, abre `programacion`, solicita capacidad low y luego xhigh, registra `CapacityDecided` para cada solicitud, pide dos agentes logicos, despacha outbox LaunchRuntimeAgent con dispatcher fake local, valida envelope OutboxMessage v0, valida AgentLauncherInbound v0 y RuntimeFakeLifecycleV0, valida AgentProgressReport loop_detected, registra AssessAgentWork, despacha StopRuntimeAgent contra AgentStopperInbound v0 y termina el fake como stopped.
Ultima ejecucion: 2026-05-05, ok; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-e2e; salida progresiva: fase `programacion` abierta antes de capacidad/agentes, con decision durable antes de launch.
Riesgos: paralelismo real no existe aun; el workflow serializa eventos durables aunque represente dos agentes logicos en vuelo, una evaluacion de trabajo, una parada logica y un lifecycle fake en memoria.
```

```text
Caso: DIR-P006 parada de proceso real pendiente
Tipo: pendiente_conector
Comando: no aplica todavia
Evidencia esperada: un corte futuro debera usar un adaptador runtime con handle, heartbeat, ACK de parada efectiva y proceso externo controlado.
Ultima ejecucion: no aplica; `StopAgent` logico y RuntimeFakeLifecycleV0 ya estan cubiertos por DIR-P005.
Riesgos: sin conector runtime real solo se valida contrato y fake en memoria, no se detiene un proceso externo.
```

```text
Caso: DIR-P007 flujo completo app simple hasta cierre
Tipo: integration_workflow_fake
Comando: go test -count=1 -v ./modulos/orquesta-director -run TestProgressiveOrquestaV0FlujoCompletoAppSimpleHastaCierre
Evidencia esperada: una app simple prepara microtarea, solicita capacidad compacta, registra `CapacityDecided`, pide agente logico, despacha launch runtime fake, registra `AgentStarted`, registra entrega, solicita revision, acepta revision, cierra tarea, registra validacion final y cierra el run. Los comandos de lifecycle/entrega/cierre emiten un evento durable cada uno, no emiten outbox y proyectan refs compactas en `started_agents`, `deliveries`, `reviews`, `accepted_reviews`, `closed_tasks`, `validations` y `closures`.
Ultima ejecucion: 2026-05-06, ok; salida esperada: eventos=23, status=cerrada, fase=cierre, entrega=delivery-ref-full-flow-001, revision=accepted-review-ref-full-flow-001, validacion=validation-ref-full-flow-001, cierre=closure-ref-full-flow-001. El flujo registra `AgentStarted` antes de `RegisterDelivery` y `ReviewResultRecorded(accepted)` antes de `ReviewAccepted`.
Riesgos: no programa codigo real, no persiste y no ejecuta runtime; solo valida la composicion durable e in-memory hasta `RunClosed`.
```

```text
Caso: DIR-P008 supervisor puro de progreso de agente
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow
Evidencia esperada: BuildAgentProgressSupervisionV0 valida AgentProgressReportV0 con runtime, loop_detected genera AssessAgentWork stop_agent y al aplicarlo core emite AgentWorkAssessed + AgentStopRequested + outbox StopRuntimeAgent; stalled genera AssessAgentWork ask_director y AskDirector separado que emite SendDirectorQuestion; stopped/progressing quedan acceptable/continue/low; reporte invalido devuelve error local y JSON de salida no filtra detalles prohibidos.

Caso: agent_progress_supervision_budget_operativo
Comando: go test -count=1 ./modulos/orquesta-director -run 'TestBuildAgentProgressSupervisionV0Tiempo'
Ultima ejecucion: 2026-05-15, ok
Evidencia esperada: `over_budget_no_activity` genera `timeout/stop_agent/high` aunque el reporte venga como `stalled` y exista artefacto sin ACK; `over_budget_but_active` genera `needs_revision/ask_director/high` sin parada directa.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow
Riesgos: la decision es pura y compacta; no observa procesos reales, no persiste y no resuelve la parada efectiva fuera del outbox contractual.
```

```text
Caso: DIR-P009 tamano saneado de supervisor de progreso
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow
Evidencia esperada: BuildAgentProgressSupervisionV0 conserva salida, errores, JSON tags y tests tras separar tipos, validacion, helpers y entrada publica en ficheros pequenos.
Ultima ejecucion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados bajo 250 lineas.
Riesgos: refactor interno sin cambio contractual; el riesgo principal es perder imports o mover logica, cubierto por tests existentes y diff-check.
```

```text
Caso: DIR-P010 tamano saneado de dispatcher fake de outbox runtime
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow
Evidencia esperada: progressiveRuntimeFakeOutboxDispatcherV0 conserva el despacho LaunchRuntimeAgent/StopRuntimeAgent, conversion outbox -> inbound runtime, must helpers, errores codificados y rechazo de mensaje no soportado tras separar el harness en ficheros pequenos.
Ultima ejecucion: 2026-05-05, ok, gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados 106/48/154 lineas.
Riesgos: refactor de test sin cambio contractual; el riesgo principal es romper nombres usados por otros tests, cubierto por compilacion de director y tests de runtime/core-workflow.
```

```text
Caso: DIR-P011 ledger outbox antes de dispatcher runtime fake
Tipo: integration_fake_runtime
Comando: go test -count=1 -v ./modulos/orquesta-director -run TestProgressiveOrquestaV0OutboxLedgerAntesDeDispatchRuntimeFake
Evidencia esperada: el harness progresivo genera LaunchRuntimeAgent y StopRuntimeAgent, guarda ambos en InMemoryOutboxLedgerV0, lista pendientes por run_id y target_port agent_launcher, despacha desde pendientes con el dispatcher runtime fake, registra ACK dispatched con dispatch_ref opaca, retira solo el ACKed y mantiene el otro pendiente hasta su ACK. Tambien guarda y ACKea un SendDirectorQuestion generado via BuildAgentProgressSupervision stalled contra target director.
Ultima ejecucion: 2026-05-05, ok; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow ./modulos/orquesta-runtime; git diff --check -- modulos/orquesta-director; wc -l Go tocado DIR-006: 215 lineas.
Riesgos: sigue siendo integracion fake/in-memory; no prueba DB real, dispatcher productivo, runtime real, procesos ni entrega efectiva a UI/MCP/CLI.
```

```text
Caso: DIR-P012 ciclo reutilizable de dispatch outbox
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow
Evidencia esperada: RunOutboxDispatchCycleV0 usa puertos locales del director, guarda mensajes opcionales, lista pendientes por run_id/target_port, despacha cada pendiente con dispatcher fake, registra ACK dispatched y deja pendientes en cero. En fallo del dispatcher registra ACK failed terminal para ese message_id, devuelve error publico con snapshot y deja pendiente solo lo no procesado. Rechaza input sin ledger/dispatcher/run_id/target_port y el JSON de resultado/error no filtra DB, SQL, runtime real, provider, HOME, OAuth, transcripts ni prompts.
Ultima ejecucion: 2026-05-05, ok; gofmt sobre Go tocados; go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-director; wc -l Go tocados 117/161/236/81/85/221 lineas.
Riesgos: sigue siendo fake/in-memory; el dispatch productivo, la persistencia real y runtime real quedan en adaptadores futuros.
```

```text
Caso: DIR-P013 nucleo con procesos reales, ledger durable y escalado de capacidad
Tipo: integration_real_connector
Comando: go test -count=1 -v ./modulos/orquesta-e2e -run TestE2ERealGobiernaDosProcesosParaBasuraYEscalaCapacidadV0
Evidencia esperada: el flujo usa contratos publicos de core-workflow/director/runtime/persistence/capacity; registra `CapacityDecided` antes de cada launch; arranca dos procesos locales controlados mediante ProcessRuntimeConnectorV0; guarda outbox en FileOutboxLedgerV0 con ruta explicita de test; ACKea LaunchRuntimeAgent; detecta trabajo basura/bucle con AssessAgentWork; emite StopRuntimeAgent; para solo el proceso afectado; mantiene el segundo proceso vivo; solicita capacidad low y despues xhigh; deja el ledger sin pendientes para agent_launcher.
Ultima ejecucion: 2026-05-05, ok; prueba focal 0.033s; go test -count=1 ./modulos/orquesta-e2e ok; go test -race -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-e2e ok.
Riesgos: el proceso real es un binario de test controlado, no un proveedor externo ni Codex real; no cubre HOME/OAuth/multi-cuenta, cuotas de proveedor, web, CLI, MCP productivo ni empaquetado.
```

```text
Caso: DIR-P014 contexto pequeno para LaunchRuntimeAgent
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-context
Evidencia esperada: BuildLaunchContextBundleV0 valida OutboxMessage LaunchRuntimeAgent, decodifica payload compacto, construye ContextBundleV0 con AGENTS/README/docs locales, pruebas de fase programacion, write_set, contract_refs y cross_module_refs; rechaza outbox que no es launch y rechaza bundle sin write_set.
Ultima ejecucion: 2026-05-05, ok.
Riesgos: no materializa refs por filesystem/MCP ni arranca runtime; solo fija la frontera director -> context -> runtime.
```

```text
Caso: DIR-P015 gate de concurrencia antes de RequestAgent
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: BuildConcurrencyGateAgentRequestsV0 evalua WorksetClaimV0 con EvaluateConcurrencyGateV0, construye RecordConcurrencyGate para allow/block, y solo en allow construye RequestAgent para los candidates dados por subject claim. Con write-set solapado no construye RequestAgent; con allow y candidate ausente devuelve error local.
Ultima ejecucion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director.
Riesgos: no aplica comandos ni prueba CapacityDecided en reducer; solo compone comandos publicos puros.
```

```text
Caso: DIR-P016 acciones posteriores a lease expirado
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: BuildPostLeaseActionV0 construye primero RegisterAgentLeaseExpired; con recommended_action=stop_agent construye StopAgent separado que core traduce a StopRuntimeAgent; con ask_director construye AskDirector separado que emite SendDirectorQuestion; con retry no construye comando secundario y marca unsupported/needs_director; input invalido devuelve error publico local y la salida no filtra HOME/transcript/provider/OAuth.
Ultima ejecucion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director.
Riesgos: las acciones no implementadas quedan deliberadamente sin efecto local hasta que exista un contrato explicito; este corte no persiste ni llama runtime real.
```

```text
Caso: DIR-P018 replan followups no reutiliza agente bloqueado
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director -run 'TestBuildReplanFollowupsV0RejectsBlockedAgentCandidate|TestBuildReplanFollowupsV0AllowsReplacementAgentWhenBlockedDiffers'
Evidencia esperada: si `agent_candidate.payload.agent_request_id` aparece en `blocked_agent_refs`, `BuildReplanFollowupsV0` devuelve `director_replan_followups_invalido`; si el reemplazo usa otra ref, construye `RequestAgent`.
Ultima ejecucion: 2026-05-06, ok.
Riesgos: no genera refs nuevas ni consulta estado del workflow; consume lista bloqueada proporcionada por el director superior.
```

```text
Caso: DIR-P017 replan followups puros
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director
Evidencia esperada: BuildReplanFollowupsV0 construye siempre RecordReplanDecision; retry_task con candidates dados construye RequestCapacity y RequestAgent; replace_agent sin candidates no inventa comandos; escalate_capacity construye solo RequestCapacity aunque venga candidate de agente; ask_director construye AskDirector solo con candidate explicito; split_task queda needs_director/unsupported sin efectos.
Ultima ejecucion: 2026-05-06, ok; go test -count=1 ./modulos/orquesta-director; git diff --check -- modulos/orquesta-director.
Riesgos: no aplica los comandos ni valida precondiciones durables de ReworkRequested/CapacityDecided; solo compone comandos publicos.
```

```text
Caso: DIR-P019 replan por quality gate bloqueado
Tipo: unit_contract
Comando: go test -count=1 .
Evidencia esperada: BuildReplanFollowupsV0 recibe source_kind `quality_gate_blocked` con source_ref opaca y produce RecordReplanDecision y RequestCapacity trazables para retry aplicable en programacion; no construye OpenPhase/documentacion ni comandos de otra fase. Para split_task usa AskDirector con candidate explicito.
Ultima ejecucion: 2026-05-07, ok; go test -count=1 .
Riesgos: no aplica comandos ni valida estado durable de quality_gates; esa cobertura vive en e2e/core.
```

```text
Caso: DIR-P020 stalled no bloquea entregas tardias
Tipo: unit_contract + smoke_real
Comando: go test -count=1 ./modulos/orquesta-director -run TestBuildAgentProgressSupervisionV0StalledPreguntaNoBloqueanteAlDirector; go test -count=1 ./modulos/orquesta-app-codex-stack -run TestCodexStackV0ProgressStalledProtegeDirectorInicial
Evidencia esperada: BuildAgentProgressSupervisionV0 construye AssessAgentWork ask_director y AskDirector con blocking=false; core emite SendDirectorQuestion y no emite RunBlocked. En stack, el director inicial con progreso stalled queda protegido, registra assessment/pregunta y no emite AgentStopRequested ni AgentStopConfirmed.
Ultima ejecucion: 2026-05-13, ok.
Riesgos: la prueba real Codex stack completa ya cubre decision files tardios; este caso focal no ejecuta Codex real para mantenerlo rapido y determinista.
```

```text
Caso: DIR-P021 replan de revision reabre programacion
Tipo: unit_contract
Comando: go test -count=1 ./modulos/orquesta-director ./modulos/orquesta-director-scheduler
Evidencia esperada: BuildReplanFollowupsV0 construye RecordReplanDecision + OpenPhase + RequestCapacity cuando recibe OpenPhaseCandidate explicito; el scheduler ordena RecordReplanDecision, OpenPhase y RequestCapacity, y no permite followups de otra fase sin ese candidate.
Ultima ejecucion: 2026-05-10, ok.
Riesgos: no aplica workflow completo; la cobertura progresiva vive en orquestacionnucleoapp.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
