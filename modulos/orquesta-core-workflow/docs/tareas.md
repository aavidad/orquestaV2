# Tareas locales: orquesta-core-workflow

Formato obligatorio:

```text
ID:
Objetivo:
Write-set:
Contrato:
Validacion:
Bloqueos:
Estado:
```

## Backlog inicial

```text
ID: NCW-070
Objetivo: Documentar que `WorkflowTaskV0` es una unidad de trabajo de granularidad adaptativa y que `CreateMicrotask` no obliga a partir siempre al minimo.
Write-set: docs/contratos_microtareas.md, docs/decisiones.md, docs/tareas.md, docs globales.
Contrato: WorkflowTaskV0, CreateMicrotask, MicrotaskCreated.
Validacion: 2026-05-10, ok, cambio documental; suite local del nucleo ya pasa tras el smoke real.
Bloqueos: No renombra eventos durables v0 para evitar migracion innecesaria; si se crea v1 podra usar nombres `WorkItemCreated`.
Estado: completada local
```

```text
ID: NCW-076
Objetivo: Propagar `work_profile_kind` desde `WorkProfileV0` a `WorkflowTaskV0` para que el scheduler pueda resolver rol/capacidad sin parsear textos.
Write-set: work_items_*.go, work_profile_task_v0.go, work_profile_v0_test.go, docs locales.
Contrato: WorkflowTaskV0, WorkProfileV0.
Validacion: 2026-05-22, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core.
Bloqueos: La resolucion concreta de scheduler queda en `orquesta-orchestration-core`; el core solo valida el campo neutral opcional.
Estado: completada local
```

```text
ID: NCW-075
Objetivo: Definir perfiles neutrales de trabajo para estudio de codigo, implementacion, refactor, pruebas, documentacion, revision y trabajo de dominio sin crear un modulo paralelo.
Write-set: work_profile_*.go, work_profile_v0_test.go, docs/contratos.md, docs/contratos_microtareas.md, docs/pruebas.md, docs/tareas.md, docs/decisiones.md.
Contrato: WorkProfileV0, WorkflowTaskFromWorkProfileV0, WorkflowTaskV0.
Validacion: 2026-05-22, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: La seleccion de capacidad/rol en scheduler y la inyeccion desde conectores quedan para el siguiente corte; este corte solo cierra el contrato neutral reutilizable.
Estado: completada local
```

```text
ID: NCW-069
Objetivo: Permitir `RecordReplanDecision` desde `AgentWorkAssessed` como fuente durable en fase `programacion`.
Write-set: replan_decision_flow_v0.go, replan_decision_projection_v0.go, replan_decision_agent_assessment_v0_test.go, docs locales.
Contrato: RecordReplanDecision, ReplanDecisionRecorded, AgentWorkAssessed, OrchestrationRunV0.ReplanDecisions.
Validacion: 2026-05-09, ok, pruebas focalizadas de replan desde AgentWorkAssessed.
Bloqueos: No relanza agentes, no pide capacidad, no crea tareas ni convierte la evaluacion en rework artificial; los followups entran por comandos separados.
Estado: completada local
```

```text
ID: NCW-068
Objetivo: Crear politica pura para detectar quality gates bloqueantes pendientes por subject.
Write-set: quality_gate_blocking_v0.go, quality_gate_blocking_v0_test.go, docs/tareas.md, docs/pruebas.md.
Contrato: OrchestrationRunV0.QualityGates, QualityGateRecorded projection, PendingBlockingQualityGateRefsForSubjectV0.
Validacion: 2026-05-07, ok, go test -count=1 . desde orquesta-core-workflow; git diff --check -- .
Bloqueos: No toca DB, runtime, provider, modelo, HOME, OAuth, cierre de fase ni parsea fuera de QualityGates del run.
Estado: completada local
```

```text
ID: NCW-068
Objetivo: Registrar artefactos compactos de fases no-programacion generados por un agente ya arrancado, sin convertirlos en entregas de codigo.
Write-set: phase_artifact_*.go, commands_v0.go, events_v0.go, command_event_catalog_v0.go, command_router_v0.go, event_router_v0.go, run_state_v0.go, reducer_clone_v0.go, run_state_validation_v0.go, tests y docs locales.
Contrato: RegisterPhaseArtifact, PhaseArtifactRegistered, OrchestrationRunV0.PhaseArtifacts.
Validacion: 2026-05-09, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: El core no lee ACKs ni runtimes. El cableado automatico queda fuera del core y ya se implementa por `DeliveryCandidateProviderV0` + scheduler.
Estado: completada local
```

```text
ID: NCW-067
Objetivo: Documentar y cerrar localmente `RecordQualityGate -> QualityGateRecorded` como gate durable compacto de calidad.
Write-set: docs/contratos_quality_gates.md, docs/pruebas_quality_gates.md, docs/contratos.md, docs/contratos_estado_fases.md, docs/contratos_revisiones.md, docs/pruebas.md, docs/tareas.md, docs/decisiones.md, docs/roadmap_cierre_nucleo.md.
Contrato: RecordQualityGate, QualityGateRecorded, OrchestrationRunV0.QualityGates, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-07, ok, pruebas focalizadas `TestRecordQualityGate*`, replay de `QualityGateRecorded` y validacion de `quality_gates`.
Bloqueos: No materializa review result, rework, bloqueo, cierre, scheduler ni outbox; esos efectos quedan para comandos separados.
Estado: completada local
```

```text
ID: NCW-065
Objetivo: Aplicar identidad fuerte a `OpenPhase` sin impedir reaperturas reales de fase.
Write-set: handler_v0.go, reducer_run_lifecycle_v0.go, run_lifecycle_effects_v0.go, handler_v0_test.go, docs locales/globales.
Contrato: PhaseOpened, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de apertura de fase; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: La identidad de `PhaseOpened` es por `event_id`, no por `phase_id`; si el producto quiere prohibir reaperturas debe ser una decision de workflow separada.
Estado: completada
```

```text
ID: NCW-064
Objetivo: Aplicar identidad fuerte a `StartRun`, `BlockRun` y `ClosePhase` para impedir lifecycle ambiguo por refs ya reflejadas.
Write-set: handler_v0.go, reducer_run_lifecycle_v0.go, phase_close_v0.go, run_lifecycle_effects_v0.go, tests asociados, docs locales/globales.
Contrato: RunStarted, RunBlocked, PhaseClosed, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de lifecycle run/fase; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: `OpenPhase` permite reapertura de fases y no se endurece en este corte para no cambiar la semantica del workflow sin decision separada.
Estado: completada
```

```text
ID: NCW-063
Objetivo: Aplicar identidad fuerte a senales de supervision y gobierno: gate de concurrencia, expiracion de lease y respuesta del director.
Write-set: concurrency_gate_v0.go, agent_lease_expired_v0.go, director_answer_v0.go, reducer_projections_v0.go, tests asociados, docs locales/globales.
Contrato: ConcurrencyGateRecorded, AgentLeaseExpired, DirectorQuestionAnswered, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de supervision/gobierno; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No calcula scheduler, no ejecuta runtime, no para agentes y no pregunta al director automaticamente; solo protege identidad de senales ya reflejadas.
Estado: completada
```

```text
ID: NCW-062
Objetivo: Aplicar identidad fuerte a capacidad decidida, lifecycle de agente, confirmacion de parada y entrega registrada.
Write-set: capacity_decision_*.go, agent_lifecycle_*.go, agent_stop_confirm_*.go, delivery_register_v0.go, tests asociados, docs locales/globales.
Contrato: CapacityDecided, AgentStarted, AgentFailed, AgentStopConfirmed, DeliveryRegistered, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de programacion operativa; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No cambia barreras ni outbox; solo protege identidad de evidencia operativa ya reflejada.
Estado: completada
```

```text
ID: NCW-061
Objetivo: Aplicar identidad fuerte a `CloseTask`, `RegisterFinalValidation` y `CloseRun` para impedir cierres ambiguos con refs reutilizadas.
Write-set: close_task_v0.go, final_validation_v0.go, close_run_v0.go, tests asociados, docs locales/globales.
Contrato: TaskClosed, FinalValidationRegistered, RunClosed, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de cierre; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No cierra fases automaticamente ni emite outbox; solo protege identidad de cierre ya reflejada.
Estado: completada
```

```text
ID: NCW-060
Objetivo: Aplicar identidad fuerte a `RecordReviewResult`, `RequestRework` y `RecordReplanDecision` para impedir que sus proyecciones compactas oculten payloads distintos.
Write-set: review_result_record_v0.go, review_rework_v0.go, replan_decision_v0.go, tests asociados, docs locales/globales.
Contrato: ReviewResultRecorded, ReworkRequested, ReplanDecisionRecorded, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de review result/rework/replan; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No materializa rework, followups ni outbox; solo protege identidad de refs ya reflejadas.
Estado: completada
```

```text
ID: NCW-059
Objetivo: Aplicar identidad fuerte a `RequestReview`/`AcceptReview` para impedir reutilizar refs de revision con otro comando o payload.
Write-set: review_request_v0.go, review_accept_v0.go, review_request_v0_test.go, review_accept_v0_test.go, docs locales/globales.
Contrato: RequestReview, ReviewRequested, AcceptReview, ReviewAccepted, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de revision/aceptacion; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No cambia resultados, rework, cierre ni outbox; solo protege proyecciones de revision ya reflejadas.
Estado: completada
```

```text
ID: NCW-058
Objetivo: Aplicar identidad fuerte a `RequestBrainstorm`/`RequestVote` para impedir reutilizar refs de arquitectura con otro comando o payload.
Write-set: brainstorm_request_v0.go, vote_request_v0.go, brainstorm_request_v0_test.go, vote_request_v0_test.go, docs locales/globales.
Contrato: RequestBrainstorm, BrainstormRequested, RequestVote, VoteRequested, OrchestrationRunV0.CommandEffects.
Validacion: 2026-05-06, ok, tests focalizados de brainstorming/votacion; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Bloqueos: No cambia la fase ni emite outbox; solo protege proyecciones de arquitectura ya reflejadas.
Estado: completada
```

```text
ID: NCW-054
Objetivo: Permitir `RecordReplanDecision` desde `AgentFailed` como fuente durable en fase `programacion`.
Write-set: replan_decision_flow_v0.go, replan_decision_projection_v0.go, replan_decision_agent_failed_v0_test.go, docs locales.
Contrato: RecordReplanDecision, ReplanDecisionRecorded, AgentFailed.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No materializa reemplazo ni capacidad; solo abre decision durable trazada a `failed_agents`.
Estado: completada
```

```text
ID: NCW-053
Objetivo: Impedir `StopAgent` si el agente ya tiene `AgentFailed` durable.
Write-set: agent_stop_handler_v0.go, agent_stop_invariant_v0_test.go, docs locales.
Contrato: StopAgent, AgentStopRequested, AgentFailed.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh.
Bloqueos: No cambia la parada de agentes arrancados ni la cancelacion de agentes solicitados; solo bloquea parada posterior a fallo de lanzamiento.
Estado: completada
```

```text
ID: NCW-052
Objetivo: Impedir lifecycle contradictorio de agente: start despues de fallo/parada y fallo despues de start/parada.
Write-set: agent_lifecycle_handler_v0.go, agent_lifecycle_reducer_v0.go, agent_lifecycle_terminal_v0.go, agent_lifecycle_invariant_v0_test.go, docs locales.
Contrato: RegisterAgentStarted, AgentStarted, RegisterAgentFailed, AgentFailed, AgentStopRequested.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh.
Bloqueos: No modela fallo runtime posterior a un agente arrancado; esa senal debe entrar por assessment/lease/replan u otro contrato futuro.
Estado: completada
```

```text
ID: NCW-051
Objetivo: Impedir entregas de agentes no arrancados o ya fallidos.
Write-set: delivery_register_flow_v0.go, delivery_register_v0_test.go, delivery_register_invariant_v0_test.go, tests de replay dependientes y docs locales.
Contrato: RegisterDelivery, DeliveryRegistered, AgentStarted, AgentFailed.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No decide relanzamiento ni replan; solo endurece la frontera de entrega con lifecycle durable.
Estado: completada
```

```text
ID: NCW-050
Objetivo: Impedir `RegisterDelivery` cuando el agente citado ya tiene parada solicitada.
Write-set: delivery_register_flow_v0.go, delivery_register_invariant_v0_test.go, docs locales.
Contrato: RegisterDelivery, DeliveryRegistered, AgentStopRequested.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; E2E programacion basura rechaza delivery posterior.
Bloqueos: No decide recuperacion ni replan; solo protege la frontera de entregas.
Estado: completada
```

```text
ID: NCW-049
Objetivo: Registrar confirmacion durable de parada de agente tras ejecutar el outbox `StopRuntimeAgent`.
Write-set: agent_stop_confirm*_v0.go, commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, catalogos, tests y docs locales.
Contrato: RegisterAgentStopConfirmed, AgentStopConfirmed, OrchestrationRunV0.ConfirmedStoppedAgents.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp; E2E real de dos procesos registra confirmacion tras parada controlada.
Bloqueos: No ejecuta runtime, no consulta proceso ni ACK de persistence; solo acepta una senal compacta de adaptador despues de `AgentStopRequested`.
Estado: completada
```

```text
ID: NCW-048
Objetivo: Publicar catalogo soportado de comandos/eventos del workflow para que adaptadores como MCP no queden desfasados.
Write-set: command_event_catalog_v0.go, command_event_catalog_v0_test.go, commands_validation_v0.go, events_validation_v0.go, docs locales.
Contrato: SupportedOrchestrationCommandTypesV0, SupportedOrchestrationEventTypesV0.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp.
Bloqueos: No cambia payloads ni transiciones; solo elimina duplicacion de catalogo y conserva core sin DB/runtime/provider/HOME/OAuth.
Estado: completada
```

```text
ID: NCW-047
Objetivo: Implementar `RecordConcurrencyGate -> ConcurrencyGateRecorded` como gate durable compacto.
Write-set: concurrency_gate*_v0.go, concurrency_gate*_v0_test.go, commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, reducer_clone_v0.go, docs locales.
Contrato: RecordConcurrencyGate, ConcurrencyGateRecorded, OrchestrationRunV0.ConcurrencyGates.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No evalua scopes dentro del workflow, no crea scheduler, no lanza agentes y no bloquea todo el run; director/runtime materializan despues.
Estado: completada
```

```text
ID: NCW-046
Objetivo: Implementar `RegisterAgentLeaseExpired -> AgentLeaseExpired` como expiracion durable observada.
Write-set: agent_lease_expired*_v0.go, agent_lease_expired*_v0_test.go, commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, reducer_clone_v0.go, docs locales.
Contrato: RegisterAgentLeaseExpired, AgentLeaseExpired, OrchestrationRunV0.AgentLeaseExpirations.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; go test -count=1 ./modulos/orquesta-e2e.
Bloqueos: No para procesos, no emite outbox, no marca failed/stopped y no replanifica; StopAgent/AskDirector/replan quedan como comandos separados.
Estado: completada
```

```text
ID: NCW-045
Objetivo: Implementar `RecordReplanDecision -> ReplanDecisionRecorded` desde `ReworkRequested`.
Write-set: replan_decision*_v0.go, replan_decision*_v0_test.go, commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, reducer_clone_v0.go, docs locales.
Contrato: RecordReplanDecision, ReplanDecisionRecorded, OrchestrationRunV0.ReplanDecisions, ReworkRequested.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core-replanner ./modulos/orquesta-core-leases ./modulos/orquesta-core-concurrency ./modulos/orquesta-director ./modulos/orquesta-runtime ./modulos/orquesta-persistence ./modulos/orquesta-capacity ./modulos/orquesta-context ./modulos/orquesta-e2e.
Bloqueos: No crea tareas, no pide capacidad, no relanza agentes, no emite outbox y no materializa followups; esos efectos quedan para comandos separados.
Estado: completada
```

```text
ID: NCW-044
Objetivo: Implementar `RequestRework -> ReworkRequested` para resultados de revision `changes_requested`/`rejected`.
Write-set: review_rework*_v0.go, review_rework*_v0_test.go, commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, docs locales.
Contrato: RequestRework, ReworkRequested, OrchestrationRunV0.ReworkRequests, ReviewResultRecorded.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No outbox, no relanza agentes, no registra replan, no cierra tareas/fases/run y no toca runtime/persistence/capacity/director/e2e.
Estado: completada
```

```text
ID: NCW-043
Objetivo: Endurecer `AcceptReview` para exigir `ReviewResultRecorded(status=accepted)` previo.
Write-set: review_accept*_v0.go, review_result_record_projection_v0.go, tests de replay/invariantes/cierre, docs locales.
Contrato: AcceptReview, ReviewAccepted, ReviewResultRecorded, OrchestrationRunV0.ReviewResults.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No dispara cierre de tarea, merge, rework, replan, outbox, conectores ni decisiones de runtime; solo valida la evidencia durable minima.
Estado: completada
```

```text
ID: NCW-042
Objetivo: Implementar `RecordReviewResult -> ReviewResultRecorded` usando `ReviewResultV0` existente.
Write-set: review_result_record*_v0.go, review_result_record*_v0_test.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, docs locales.
Contrato: RecordReviewResult, ReviewResultRecorded, OrchestrationRunV0.ReviewResults.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No dispara outbox, cierre de tarea, `AcceptReview`, rework automatico, leases, concurrency ni replan completo; esos cortes quedan separados.
Estado: completada
```

```text
ID: NCW-041
Objetivo: Hacer seguro el reintento de `AskDirector` sin perder outbox ni re-bloquear preguntas ya respondidas.
Write-set: director_question_*.go, reducer_projections_v0.go, run_state_*.go, tests y docs locales/globales.
Contrato: AskDirector, DirectorQuestionRaised, DirectorQuestionAnswered, OrchestrationRunV0.DirectorAnsweredQuestions.
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: La entrega real a UI/MCP/CLI sigue fuera; el core solo reconstruye outbox compacto y conserva respuesta durable.
Estado: completada
```

```text
ID: NCW-040
Objetivo: Reemitir outbox pendiente si se persistio el evento pero se perdio el mensaje externo en `RequestCapacity` o `RequestAgent`.
Write-set: capacity_command_handler_v0.go, agent_request_handler_v0.go, capacity_command_v0_test.go, agent_request_v0_test.go, docs locales.
Contrato: RequestCapacity, RequestCapacityDecision, RequestAgent, LaunchRuntimeAgent.
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: La unicidad productiva y ACK siguen perteneciendo a persistence/dispatcher; el core solo reconstruye el outbox determinista.
Estado: completada
```

```text
ID: NCW-039
Objetivo: Registrar decision durable de capacidad antes de permitir lanzamiento logico de agentes.
Write-set: capacity_decision_*.go, agent_request_*.go, commands_v0.go, events_v0.go, handler_v0.go, reducer_*.go, run_state_*.go, tests, director/e2e y docs locales/globales.
Contrato: RegisterCapacityDecision, CapacityDecided, OrchestrationRunV0.CapacityDecisions, RequestAgent.capacity_request_ref obligatorio.
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-e2e.
Bloqueos: No registra provider/model/HOME/cuota real; esos detalles quedan en capacity/runtime por conectores.
Estado: completada
```

```text
ID: NCW-038
Objetivo: Registrar ciclo logico de agente arrancado/fallido sin reinterpretar `run.agents`.
Write-set: agent_lifecycle_*.go, commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, run_state_*.go, tests, E2E y docs locales/globales.
Contrato: RegisterAgentStarted, AgentStarted, RegisterAgentFailed, AgentFailed, OrchestrationRunV0.StartedAgents, OrchestrationRunV0.FailedAgents.
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Bloqueos: No registra PID/proceso/runtime real; solo refs opacas de launch/ack/readiness o fallo.
Estado: completada
```

```text
ID: NCW-037
Objetivo: Endurecer orquestacion por fase activa y reintento parcial de parada desde `AssessAgentWork`.
Write-set: capacity_command_handler_v0.go, agent_request_handler_v0.go, agent_work_assessment_*.go, reducer_projections_v0.go, reducer_helpers_v0.go, tests y docs locales.
Contrato: RequestCapacity, CapacityRequested, RequestAgent, AgentRequested, AssessAgentWork, AgentWorkAssessed.
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-runtime ./modulos/orquesta-capacity ./modulos/orquesta-e2e.
Bloqueos: Resuelto en NCW-039: `RequestAgent` exige ahora `CapacityDecided`.
Estado: completada
```

```text
ID: NCW-036
Objetivo: Implementar `AnswerDirectorQuestion` -> `DirectorQuestionAnswered` como respuesta compacta del director, sin outbox, con desbloqueo acotado de `director-question-<question_id>`.
Write-set: answer_director_question_*.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_*.go, run_state_*.go, tests y docs locales.
Contrato: AnswerDirectorQuestion, DirectorQuestionAnswered, OrchestrationRunV0.
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow; respuesta valida proyectada; outbox vacio; refs opacas; rechazo de secretos/provider/HOME/DB/runtime; desbloqueo solo del blocker asociado.
Bloqueos: No entrega respuestas por UI/MCP/CLI ni resuelve adaptadores; solo registra decision del director y desbloqueo durable.
Estado: completada
```

```text
ID: NCW-000
Objetivo: Crear miniproyecto nuevo del nucleo durable, documentar analisis forense, decisiones, contratos y pruebas.
Write-set: modulos/orquesta-core-workflow/**, modulos/README.md, docs/reinicio_orquesta_v2/registro_reutilizacion.md, docs/reinicio_orquesta_v2/roadmap_operativo.md
Contrato: N/A
Validacion: git diff --check; docs locales presentes; snapshot del core actual creado.
Bloqueos: ninguno.
Estado: completada
```

```text
ID: NCW-001
Objetivo: Implementar DTOs puros de `OrchestrationRunV0`, `OrchestrationPhaseV0`, estados y fases v0.
Write-set: modulos/orquesta-core-workflow/run_state_v0.go, modulos/orquesta-core-workflow/run_state_v0_test.go, docs/tareas.md, docs/pruebas.md
Contrato: OrchestrationRunV0, OrchestrationPhaseV0
Validacion: go test -count=1 ./modulos/orquesta-core-workflow
Bloqueos: ninguno.
Estado: completada
```

```text
ID: NCW-002
Objetivo: Implementar DTO puro `OrchestrationEventV0`, constantes iniciales `RunStarted`, `PhaseOpened` y `RunBlocked`, constructores y validadores sin depender de NCW-001.
Write-set: modulos/orquesta-core-workflow/events_v0.go, modulos/orquesta-core-workflow/events_v0_test.go, docs/tareas.md, docs/pruebas.md
Contrato: OrchestrationEventV0
Validacion: serializacion de eventos no contiene secretos, transcripts ni detalles prohibidos; evento desconocido devuelve error publico.
Bloqueos: ninguno; reducer inicial cerrado en NCW-002R.
Estado: completada
```

```text
ID: NCW-002R
Objetivo: Implementar reducer puro inicial `ApplyEventV0` para `RunStarted`, `PhaseOpened`, `RunBlocked` y helper secuencial `ReplayEventsV0`.
Write-set: modulos/orquesta-core-workflow/reducer_v0.go, modulos/orquesta-core-workflow/reducer_v0_test.go, docs/tareas.md, docs/pruebas.md
Contrato: OrchestrationRunV0, OrchestrationEventV0
Validacion: go test -count=1 ./modulos/orquesta-core-workflow
Bloqueos: ninguno.
Estado: completada
```

```text
ID: NCW-003
Objetivo: Implementar comandos y handler puro inicial para `StartRun`, `OpenPhase` y `BlockRun`.
Write-set: modulos/orquesta-core-workflow/commands_v0.go, modulos/orquesta-core-workflow/handler_v0.go, modulos/orquesta-core-workflow/handler_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md
Contrato: OrchestrationCommandV0, OrchestrationCommandResultV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow; handler devuelve eventos sin ejecutar efectos externos; comandos repetidos ya reflejados no duplican progreso.
Bloqueos: NCW-002 completada; idempotencia durable completa queda para NCW-005.
Estado: completada
```

```text
ID: NCW-004
Objetivo: Definir outbox inicial y consulta al director. DTOs puros, validadores, constructor `SendDirectorQuestion` e integracion durable de `AskDirector`.
Write-set: modulos/orquesta-core-workflow/outbox_v0.go, modulos/orquesta-core-workflow/director_question_v0.go, modulos/orquesta-core-workflow/outbox_v0_test.go, modulos/orquesta-core-workflow/director_question_command_v0.go, modulos/orquesta-core-workflow/director_question_command_v0_test.go, modulos/orquesta-core-workflow/commands_v0.go, modulos/orquesta-core-workflow/events_v0.go, modulos/orquesta-core-workflow/handler_v0.go, modulos/orquesta-core-workflow/reducer_v0.go, modulos/orquesta-core-workflow/run_state_v0.go, docs/contratos.md, docs/pruebas.md, docs/tareas.md
Contrato: OutboxMessageV0, DirectorQuestionV0, AskDirector, DirectorQuestionRaised
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; mensajes no contienen adaptadores concretos ni secretos; tipo desconocido rechaza; pregunta requiere summary/source; payload no contiene contexto masivo; `HandleCommandV0` acepta `AskDirector`; `DirectorQuestionRaised` entra en reducer/replay y `RunBlocked` se emite cuando `blocking=true`.
Bloqueos: ninguno local; entrega real de outbox queda para adaptadores MCP/web/CLI/director.
Estado: completada durable local
```

```text
ID: NCW-005
Objetivo: Completar replay durable e idempotencia minima sobre el reducer inicial ya disponible.
Write-set: modulos/orquesta-core-workflow/replay_v0.go, modulos/orquesta-core-workflow/idempotency_v0.go, modulos/orquesta-core-workflow/replay_v0_test.go, docs/tareas.md, docs/pruebas.md
Contrato: OrchestrationRunV0, OrchestrationEventV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; replay durable valida sequence estricta desde 1, deduplica eventos ya reflejados por event_id/idempotency_key solo con misma huella durable y rechaza conflictos de payload o metadatos con error publico.
Bloqueos: NCW-003 para idempotencia de comandos; replay secuencial inicial cubierto en NCW-002R.
Estado: completada parte ejecutable sin tocar handler
```

```text
ID: NCW-006
Objetivo: Documentar e implementar candidato local desde draft compacto AppSpec/Proyecto hacia `StartRun`, sin promover contrato global.
Write-set: modulos/orquesta-core-workflow/appspec_bridge_v0.go, modulos/orquesta-core-workflow/appspec_bridge_v0_test.go, docs/contratos.md, docs/decisiones.md, docs/tareas.md, docs/pruebas.md
Contrato: StartRunFromAppSpecV0 candidato
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow; mapping determinista; rechaza refs vacios y detalles DB/runtime/proveedor/HOME; produce comando valido `StartRun`; no importa `orquesta-factory` ni `orquesta-core`.
Bloqueos: puente real desde contratos globales queda pendiente de promocion y decision del director.
Estado: completada local sin promocion global
```

```text
ID: NCW-007
Objetivo: Promover contratos globales minimos si los tests NCW-001..NCW-005 estabilizan la forma.
Write-set: ../../CONTRATOS.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: OrchestrationRunV0, OutboxMessageV0, DirectorQuestionV0
Validacion: 2026-05-04, ok, git diff --check; go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: ninguno; NCW-001..NCW-005 ya tienen evidencia local y el director autorizo esta promocion en la microtarea.
Estado: completada global
```

```text
ID: NCW-008
Objetivo: Definir DTOs puros y validadores para work items/microtareas del workflow durable, con refs opacas a contratos de funcion.
Write-set: modulos/orquesta-core-workflow/work_items_v0.go, modulos/orquesta-core-workflow/work_items_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: WorkflowTaskV0, WorkflowFunctionContractRefV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow; caso valido, fase desconocida, write_set vacio, criterio vacio, detalles prohibidos y serializacion sin adaptadores/secretos.
Bloqueos: La integracion con comandos/eventos `CreateMicrotask` queda fuera para no mezclar el DTO puro con otra transicion durable en el mismo corte.
Estado: completada local
```

```text
ID: NCW-066
Objetivo: Dividir tests del workflow que habian pasado zona amarilla para mantener contexto pequeno sin cambiar comportamiento.
Write-set: director_question_command_v0_test.go, director_question_command_conflicts_v0_test.go, handler_v0_test.go, handler_idempotency_validation_v0_test.go, agent_request_v0_test.go, agent_request_validation_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: AskDirector, HandleCommandV0, RequestAgent; sin cambio funcional.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: ninguno; si nuevos tests vuelven a superar zona amarilla, partir por contrato antes de anadir escenarios.
Estado: completada local
```

```text
ID: NCW-021
Objetivo: Dividir `events_v0.go` antes de anadir mas eventos para mantener ficheros manejables.
Write-set: events_v0.go, events_validation_v0.go, docs/tareas.md, docs/decisiones.md
Contrato: sin cambios funcionales
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: Ninguno; prepara el terreno para `AcceptReview`.
Estado: completada local
```

```text
ID: NCW-022
Objetivo: Implementar aceptacion durable de revision como `AcceptReview` -> `ReviewAccepted`.
Write-set: review_accept_*.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, tests y docs locales
Contrato: AcceptReview, ReviewAccepted, OrchestrationRunV0.AcceptedReviews
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; referencias compactas; sin conectores hardcodeados; sin detalles de runtime, DB, proveedor, OAuth ni HOME salvo como prohibiciones.
Bloqueos: Cierre de tarea, cierre de fase y validacion final quedan como microtareas posteriores.
Estado: completada local
```

```text
ID: NCW-023
Objetivo: Implementar cierre durable de tarea como `CloseTask` -> `TaskClosed`.
Write-set: close_task_*.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, tests y docs locales
Contrato: CloseTask, TaskClosed, OrchestrationRunV0.ClosedTasks
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; `CloseTask` compacto sin outbox; exige fase `revision` activa, `task_id` en `tasks`, `delivery_ref` en `deliveries` y `accepted_review_ref` en `accepted_reviews`; `TaskClosed` proyecta `task_id` en `closed_tasks`; no cierra fase ni run; sin conectores ni detalles de runtime, DB, proveedor, OAuth o HOME salvo como prohibiciones.
Bloqueos: Cierre de fase, validacion final y cierre de run quedan como microtareas posteriores.
Estado: completada local
```

```text
ID: NCW-024
Objetivo: Implementar validacion final durable como `RegisterFinalValidation` -> `FinalValidationRegistered`.
Write-set: final_validation_*.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, tests y docs locales
Contrato: RegisterFinalValidation, FinalValidationRegistered, OrchestrationRunV0.Validations
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; refs compactas; sin outbox; exige fase `validacion_final` activa y `closed_task_ref` en `closed_tasks`; `FinalValidationRegistered` proyecta `validation_ref` en `validations`; no cierra fase ni run; sin conectores ni detalles de runtime, DB, proveedor, OAuth o HOME salvo como prohibiciones.
Bloqueos: Cierre de fase y cierre de run quedan como microtareas posteriores.
Estado: completada local
```

```text
ID: NCW-025
Objetivo: Implementar cierre durable de run como `CloseRun` -> `RunClosed`.
Write-set: close_run_*.go, commands_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, tests y docs locales
Contrato: CloseRun, RunClosed, OrchestrationRunV0.Closures
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; `CloseRun` compacto sin outbox; exige fase `cierre` activa y `validation_ref` ya proyectado en `validations`; `RunClosed` proyecta `closure_ref` en `closures` y marca el run como `cerrado`; no cierra fase automaticamente; sin conectores ni detalles de runtime, DB, provider/proveedor, OAuth o HOME salvo como prohibiciones.
Bloqueos: Si producto exige cerrar la fase `cierre`, debe seguir usando `ClosePhase` separado.
Estado: completada local
```

```text
ID: NCW-026
Objetivo: Dividir `run_state_v0.go` en estado, catalogo, validacion y helpers para mantener contexto pequeno.
Write-set: run_state_v0.go, run_state_catalog_v0.go, run_state_validation_v0.go, run_state_helpers_v0.go, docs/tareas.md, docs/decisiones.md
Contrato: sin cambios funcionales
Validacion: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow; ficheros resultantes bajo 400 lineas.
Bloqueos: Ninguno; prepara el terreno para nuevas fases sin inflar el contexto base.
Estado: completada local
```

```text
ID: NCW-027
Objetivo: Sanear zona amarilla de core-workflow dividiendo ficheros grandes por fase, escenario y responsabilidad sin cambiar comportamiento.
Write-set: reducer_v0_test.go, reducer_*_v0_test.go, commands_v0.go, commands_*_v0.go, outbox_v0.go, outbox_*_v0.go, director_question_command_v0.go, director_question_*_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: sin cambios funcionales; conserva nombres publicos, eventos, comandos, errores publicos, invariantes, fixtures y expectations.
Validacion: 2026-05-04, ok, gofmt; go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow; wc -l de ficheros Go tocados.
Bloqueos: ninguno; no requiere cambio de contrato ni semantica.
Estado: completada local
```

```text
ID: NCW-028
Objetivo: Promover contrato puro `StopAgent` como comando -> evento -> outbox logico hacia `agent_launcher`.
Write-set: commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, outbox_v0.go, outbox_validation_v0.go, run_state_v0.go, run_state_validation_v0.go, agent_stop_v0.go, agent_stop_v0_test.go, docs/contratos.md, docs/contratos_agentes.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: StopAgent, AgentStopRequested, StopRuntimeAgentRequestV0, OrchestrationRunV0.StoppedAgents
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No detiene runtimes reales ni decide proveedor/modelo; la ejecucion pertenece al puerto `agent_launcher`.
Estado: completada local
```

```text
ID: NCW-029
Objetivo: Promover microtarea de evaluacion de trabajo/progreso de agente como `AssessAgentWork` -> `AgentWorkAssessed`, con parada logica opcional.
Write-set: agent_work_assessment_v0.go, agent_work_assessment_v0_test.go, commands_v0.go, commands_validation_v0.go, events_v0.go, events_validation_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, run_state_validation_v0.go, docs/contratos.md, docs/contratos_agentes.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: AssessAgentWork, AgentWorkAssessed, OrchestrationRunV0.AgentAssessments
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No evalua con modelos ni ejecuta parada real; solo emite evento y outbox logico reutilizando `agent_launcher`.
Estado: completada local
```

```text
ID: NCW-030
Objetivo: Endurecer prueba de composicion del flujo desde `RegisterDelivery` hasta `CloseRun`.
Write-set: modulos/orquesta-core-workflow/close_run_flow_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/pruebas_cierre_run.md
Contrato: RegisterDelivery, RequestReview, AcceptReview, CloseTask, RegisterFinalValidation, CloseRun
Validacion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: ninguno; no cambia runtime, DB, outbox dispatcher ni adaptadores.
Estado: completada local
```

```text
ID: NCW-031
Objetivo: Probar y documentar que `AssessAgentWork` con action `ask_director` no emite consulta ni outbox; `AskDirector` queda como comando separado.
Write-set: modulos/orquesta-core-workflow/agent_work_assessment_director_question_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: AssessAgentWork, AgentWorkAssessed, AskDirector, DirectorQuestionRaised, SendDirectorQuestion, RunBlocked
Validacion: 2026-05-05, ok, gofmt; go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: ninguno; no cambia handler, reducer ni outbox.
Estado: completada local
```

```text
ID: NCW-032
Objetivo: Sanear tamano de `reducer_v0.go` dividiendo el reducer por dispatch, ciclo de vida, proyecciones, helpers y clonado sin cambiar comportamiento.
Write-set: reducer_v0.go, reducer_run_lifecycle_v0.go, reducer_projections_v0.go, reducer_helpers_v0.go, reducer_clone_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: sin cambios funcionales; conserva `ApplyEventV0`, `ReplayEventsV0`, helpers existentes compatibles, eventos, errores publicos y reglas de replay.
Validacion: 2026-05-05, ok, gofmt; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core ./modulos/orquesta-director; git diff --check -- modulos/orquesta-core-workflow; wc -l de Go tocados, todos bajo 250 lineas.
Bloqueos: ninguno.
Estado: completada local
```

```text
ID: NCW-033
Objetivo: Sanear tamano de `agent_stop_v0.go` dividiendo tipos, handler/reducer, outbox, validacion y helpers sin cambiar comportamiento.
Write-set: agent_stop_v0.go, agent_stop_handler_v0.go, agent_stop_outbox_v0.go, agent_stop_validation_v0.go, agent_stop_helpers_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: sin cambios funcionales; conserva `StopAgent`, `AgentStopRequested`, `StopRuntimeAgentRequestV0`, outbox `StopRuntimeAgent`, errores publicos e idempotencia.
Validacion: 2026-05-05, ok, gofmt; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director; git diff --check -- modulos/orquesta-core-workflow; wc -l de Go tocados, todos bajo 250 lineas.
Bloqueos: ninguno.
Estado: completada local
```

```text
ID: NCW-034
Objetivo: Sanear tamano de `capacity_command_v0.go` dividiendo tipos/constructores, handler/idempotencia y validacion/helpers sin cambiar comportamiento.
Write-set: capacity_command_v0.go, capacity_command_handler_v0.go, capacity_command_validation_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: sin cambios funcionales; conserva `RequestCapacity`, `CapacityRequested`, JSON tags, errores publicos, idempotencia y outbox `RequestCapacityDecision`.
Validacion: 2026-05-05, ok, gofmt; go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director; git diff --check -- modulos/orquesta-core-workflow; wc -l de Go tocados, todos bajo 250 lineas.
Bloqueos: ninguno.
Estado: completada local
```

```text
ID: NCW-035
Objetivo: Crear `ReviewResultV0` como DTO/validador puro para representar una revision compacta antes de comandos durables existentes.
Write-set: review_result_v0.go, review_result_v0_test.go, docs/contratos_revisiones.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: ReviewResultV0
Validacion: gofmt; go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No cambia `RequestReview`/`AcceptReview`, no agrega handler/evento/reducer/outbox y no toca conectores, DB, runtime, provider, HOME, OAuth, prompts ni transcripts.
Estado: completada local
```

```text
ID: NCW-020
Objetivo: Implementar solicitud durable de revision como `RequestReview` -> `ReviewRequested`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, review_request_*.go, delivery_register_v0_test.go, docs/contratos.md, docs/contratos_revisiones.md, docs/tareas.md, docs/pruebas.md, docs/pruebas_revisiones.md, docs/decisiones.md
Contrato: RequestReview, ReviewRequested, OrchestrationRunV0.Reviews
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No acepta revision ni ejecuta agentes; requiere fase `revision` y entrega ya proyectada.
Estado: completada local
```

```text
ID: NCW-019
Objetivo: Implementar registro durable de entrega como `RegisterDelivery` -> `DeliveryRegistered`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, delivery_register_*.go, run_state_v0_test.go, docs/contratos.md, docs/contratos_entregas.md, docs/contratos_estado_fases.md, docs/tareas.md, docs/pruebas.md, docs/pruebas_entregas.md, docs/decisiones.md
Contrato: RegisterDelivery, DeliveryRegistered, OrchestrationRunV0.Deliveries
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No solicita revision ni guarda implementacion; requiere fase `programacion`, tarea y agente ya proyectados.
Estado: completada local
```

```text
ID: NCW-018
Objetivo: Reducir indices locales de contratos y pruebas moviendo detalle estable a documentos tematicos.
Write-set: docs/contratos.md, docs/contratos_estado_fases.md, docs/pruebas.md, docs/pruebas_microtareas.md, docs/tareas.md, docs/decisiones.md
Contrato: documentacion local del modulo
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: Ninguno; no cambia codigo ni contratos ejecutables.
Estado: completada local
```

```text
ID: NCW-017
Objetivo: Endurecer `CreateMicrotask` para exigir planificacion activa y contratos de funcion publicados.
Write-set: work_items_command_v0.go, work_items_command_flow_v0.go, work_items_command_v0_test.go, work_items_command_invariant_v0_test.go, work_items_v0_test.go, reducer_v0.go, docs/contratos.md, docs/contratos_microtareas.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: CreateMicrotask, MicrotaskCreated, WorkflowTaskV0, OrchestrationRunV0.FunctionContracts
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No ejecuta tareas ni decide runtime; solo valida la relacion durable entre planificacion, contratos publicados y microtareas.
Estado: completada local
```

```text
ID: NCW-016
Objetivo: Implementar publicacion durable de contrato de funcion como `PublishFunctionContract` -> `FunctionContractPublished`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, function_contract_publish_*.go, docs/contratos.md, docs/contratos_funcion.md, docs/tareas.md, docs/pruebas.md, docs/pruebas_contratos_funcion.md, docs/decisiones.md
Contrato: PublishFunctionContract, FunctionContractPublished, OrchestrationRunV0.Decisions, OrchestrationRunV0.FunctionContracts
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No crea microtareas ni ejecuta codigo; requiere decision aceptada y fase `planificacion_microtareas` activa.
Estado: completada local
```

```text
ID: NCW-009
Objetivo: Integrar `WorkflowTaskV0` como transicion durable `CreateMicrotask` sin runtime, DB, proveedor, HOME ni adaptadores.
Write-set: modulos/orquesta-core-workflow/commands_v0.go, modulos/orquesta-core-workflow/events_v0.go, modulos/orquesta-core-workflow/handler_v0.go, modulos/orquesta-core-workflow/reducer_v0.go, modulos/orquesta-core-workflow/work_items_command_v0.go, modulos/orquesta-core-workflow/work_items_command_v0_test.go, docs/contratos.md, docs/contratos_microtareas.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: CreateMicrotask, MicrotaskCreated, WorkflowTaskV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check; `MicrotaskCreated` usa refs compactas `[]string` y el comando rechaza proyecciones de evento demasiado grandes.
Bloqueos: ninguno local; ejecucion real de microtareas pertenece a adaptadores/runtime futuros.
Estado: completada local
```

```text
ID: NCW-010
Objetivo: Integrar solicitud de capacidad dinamica como transicion durable `RequestCapacity` sin decidir proveedor/modelo ni usar runtime, DB, HOME, OAuth, Codex, Claude, Ollama, vLLM ni adaptadores.
Write-set: modulos/orquesta-core-workflow/commands_v0.go, modulos/orquesta-core-workflow/events_v0.go, modulos/orquesta-core-workflow/handler_v0.go, modulos/orquesta-core-workflow/reducer_v0.go, modulos/orquesta-core-workflow/outbox_v0.go, modulos/orquesta-core-workflow/capacity_command_v0.go, modulos/orquesta-core-workflow/capacity_outbox_v0.go, modulos/orquesta-core-workflow/capacity_command_v0_test.go, modulos/orquesta-core-workflow/docs/contratos.md, modulos/orquesta-core-workflow/docs/contratos_capacidad.md, modulos/orquesta-core-workflow/docs/tareas.md, modulos/orquesta-core-workflow/docs/pruebas.md, modulos/orquesta-core-workflow/docs/decisiones.md
Contrato: RequestCapacity, CapacityRequested, CapacityDecisionRequestV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check.
Bloqueos: ninguno local; la decision real de capacidad pertenece al puerto `capacity`.
Estado: completada local
```

```text
ID: NCW-011
Objetivo: Integrar solicitud de agente como transicion durable `RequestAgent` sin decidir ni transportar runtime, proveedor, modelo, HOME, OAuth, Codex, Claude, Ollama, vLLM, adaptadores ni secretos.
Write-set: modulos/orquesta-core-workflow/commands_v0.go, modulos/orquesta-core-workflow/events_v0.go, modulos/orquesta-core-workflow/handler_v0.go, modulos/orquesta-core-workflow/reducer_v0.go, modulos/orquesta-core-workflow/outbox_v0.go, modulos/orquesta-core-workflow/agent_request_v0.go, modulos/orquesta-core-workflow/agent_outbox_v0.go, modulos/orquesta-core-workflow/agent_request_v0_test.go, docs/contratos.md, docs/contratos_agentes.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: RequestAgent, AgentRequested, LaunchRuntimeAgentRequestV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check.
Bloqueos: ninguno local; el lanzamiento real pertenece al puerto `agent_launcher`.
Estado: completada local
```

```text
ID: NCW-012
Objetivo: Implementar cierre durable de fase como transicion pura con `ClosePhase` y `PhaseClosed`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, phase_close_v0.go, phase_close_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md, docs/contratos_fases.md, docs/contratos.md
Contrato: ClosePhase, PhaseClosed, OrchestrationPhaseV0
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; handler y reducer no emiten outbox para cierre de fase.
Bloqueos: ninguno local; el cierre de run completo queda fuera de esta microtarea.
Estado: completada local
```

```text
ID: NCW-013
Objetivo: Implementar solicitud durable de brainstorming como transicion pura `RequestBrainstorm` -> `BrainstormRequested`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, brainstorm_request_v0.go, brainstorm_request_v0_test.go, run_state_v0_test.go, docs/contratos.md, docs/contratos_brainstorm.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: RequestBrainstorm, BrainstormRequested, OrchestrationRunV0.Brainstorms
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No lanza agentes ni pide capacidad; esos efectos se modelan con RequestCapacity y RequestAgent en cortes separados.
Estado: completada local
```

```text
ID: NCW-014
Objetivo: Implementar solicitud durable de votacion como transicion pura `RequestVote` -> `VoteRequested`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, run_state_v0.go, vote_request_v0.go, vote_request_v0_test.go, run_state_v0_test.go, docs/contratos.md, docs/contratos_votaciones.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Contrato: RequestVote, VoteRequested, OrchestrationRunV0.Votes
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No acepta decision final; `AcceptDecision` queda para un corte posterior.
Estado: completada local
```

```text
ID: NCW-015
Objetivo: Implementar aceptacion durable de decision de arquitectura como `AcceptDecision` -> `ArchitectureDecisionAccepted`.
Write-set: commands_v0.go, events_v0.go, handler_v0.go, reducer_v0.go, decision_accept_v0.go, decision_accept_validation_v0.go, decision_accept_v0_test.go, docs/contratos.md, docs/contratos_decisiones.md, docs/tareas.md, docs/pruebas.md, docs/pruebas_decisiones.md, docs/decisiones.md
Contrato: AcceptDecision, ArchitectureDecisionAccepted, OrchestrationRunV0.Votes, OrchestrationRunV0.Decisions
Validacion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core; git diff --check -- modulos/orquesta-core-workflow.
Bloqueos: No abre planificacion ni crea microtareas; requiere voto proyectado previamente y esas transiciones quedan separadas.
Estado: completada local
```

```text
ID: NCW-055
Objetivo: Endurecer idempotencia de efectos para que una ref ya reflejada no pueda reconstruir outbox con otro comando o payload.
Write-set: command_effects_*.go, run_state_v0.go, reducer_*_v0.go, capacity_command_handler_v0.go, agent_request_handler_v0.go, agent_stop_handler_v0.go, agent_work_assessment_*_v0.go, director_question_*_v0.go, tests locales y docs.
Contrato: CommandEffects, CapacityRequested, AgentRequested, AgentStopRequested, AgentWorkAssessed, DirectorQuestionRaised, outbox pendiente.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: Extender identidad fuerte a refs sin outbox queda pendiente por contrato; no se toca DB, runtime, proveedor, HOME ni OAuth.
Estado: completada local
```

```text
ID: NCW-056
Objetivo: Separar dispatch interno de comandos y eventos en routers pequenos sincronizados con el catalogo publico.
Write-set: command_router_v0.go, event_router_v0.go, handler_v0.go, reducer_v0.go, command_event_catalog_v0_test.go, docs locales.
Contrato: HandleCommandV0, ApplyEventV0, catalogo de comandos/eventos soportados.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: No mueve validadores de payload ni cambia errores publicos; ese corte debe ser otra microtarea si aporta valor.
Estado: completada local
```

```text
ID: NCW-057
Objetivo: Aplicar identidad durable fuerte a la cadena `AcceptDecision` -> `PublishFunctionContract` -> `CreateMicrotask`.
Write-set: decision_accept_v0.go, function_contract_publish_v0.go, work_items_command_v0.go, reducer_projections_v0.go, tests locales y docs.
Contrato: CommandEffects, ArchitectureDecisionAccepted, FunctionContractPublished, MicrotaskCreated.
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Bloqueos: Brainstorm/votacion/revision/cierre quedan para cortes separados por contrato, no mezclados en esta microtarea.
Estado: completada local
```

```text
ID: NCW-073
Objetivo: Exponer motivo historico compacto de parada por agente para director/web sin depender de payloads largos ni adaptadores.
Write-set: agent_stop_projection_v0.go, agent_stop_handler_v0.go, agent_stop_validation_v0.go, run_state_v0.go, run_state_validation_v0.go, director_agent_stop_reason_v0.go, director_stats_*_v0.go, tests y docs locales.
Contrato: AgentStopRequested, OrchestrationRunV0.AgentStopRequests, DirectorRunStatsV0.Agents[].stop_reason_*.
Validacion: 2026-05-13, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core.
Bloqueos: No sustituye evidencias completas ni parada fisica; esas piezas siguen en event log, outbox y puertos runtime.
Estado: completada local
```

```text
ID: NCW-074
Objetivo: Permitir artefactos de fase tardios cuando pertenecen al agente y fase donde fue arrancado.
Write-set: agent_phase_projection_v0.go, run_state_v0.go, reducer_projections_v0.go, phase_artifact_flow_v0.go, tests y docs locales.
Contrato: AgentRequested, OrchestrationRunV0.AgentPhaseRefs, RegisterPhaseArtifact, PhaseArtifactRegistered.
Validacion: 2026-05-21, ok, go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRegisterPhaseArtifactCommandV0AcceptsLateArtifactForAgentLaunchPhase|TestRegisterPhaseArtifactCommandV0RejectsLateArtifactForOtherAgentPhase|TestReplayDurableEventsV0AcceptsLatePhaseArtifactForAgentLaunchPhase'.
Bloqueos: No guarda contenido de ACK ni conoce runtime; la observacion y validacion de archivos sigue en adaptadores externos.
Estado: completada local
```
