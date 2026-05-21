# Pruebas locales: orquesta-core-workflow

## Plantilla

```text
Caso:
Tipo: unit | contract | replay | idempotency | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```

Detalle quality gates: `docs/pruebas_quality_gates.md`.

## Pruebas previstas

```text
Caso: work_profile_to_workflow_task_neutral
Tipo: unit | contract | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestWorkflowTaskFromWorkProfileV0|TestValidateWorkProfileV0|TestNormalizeWorkProfileKindV0'
Evidencia esperada: `WorkProfileV0` normaliza alias, aplica fase y criterios base, exige contratos de funcion, exige pruebas para implementacion/refactor/pruebas, conserva linaje y rechaza scope inseguro a traves de `WorkflowTaskV0`.
Ultima ejecucion: 2026-05-22, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: No decide proveedor, modelo, proceso real ni reglas internas de conectores; solo materializa contrato neutral hacia `WorkflowTaskV0`.
```

```text
Caso: quality_gate_blocking_policy_por_subject
Tipo: unit | contract | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestPendingBlockingQualityGateRefsForSubjectV0
Evidencia esperada: blocked/rework_required/ask_director dejan blockers pendientes por subject; accepted posterior del mismo subject los resuelve; accepted de otro subject no los resuelve; proyeccion invalida se ignora sin panico.
Ultima ejecucion: 2026-05-07, ok, go test -count=1 . desde orquesta-core-workflow.
Riesgos: Politica pura no cierra fases ni ejecuta rework, scheduler, runtime, provider, DB, modelo, HOME u OAuth.
```

```text
Caso: record_quality_gate_durable
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRecordQualityGate|TestReplayDurableEventsV0AcceptsQualityGateRecordedV0|TestValidateOrchestrationRunV0RejectsInvalidQualityGateProjection'
Evidencia esperada: `RecordQualityGate` emite solo `QualityGateRecorded`, outbox vacio, proyeccion `quality_gates`, validacion de issue_refs para decisiones no aceptadas, replay de duplicado exacto y rechazo de proyeccion invalida.
Ultima ejecucion: 2026-05-07, ok, pruebas focalizadas de quality gate.
Riesgos: No ejecuta rework, bloqueo, cierre, scheduler ni pregunta al director; esos efectos siguen como comandos separados.
```

```text
Caso: apertura_fase_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestHandleOpenPhase.*|TestApplyPhaseOpened.*'
Evidencia esperada: PhaseOpened registra CommandEffects por event_id; repetir la misma apertura es no-op aunque el run haya avanzado, cambiar payload para el mismo event_id se rechaza y otra apertura de la misma fase con otro idempotency key queda como evento nuevo.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de apertura de fase; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No prohibe reaperturas; conserva el modelo actual y solo elimina el no-op ambiguo por fase activa.
```

```text
Caso: lifecycle_run_fase_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestHandle(StartRun|BlockRun|ClosePhase).*|TestApply(RunBlocked|PhaseClosed).*|TestReplayDurableEventsV0AcceptsDuplicatePhaseClosed'
Evidencia esperada: RunStarted, RunBlocked y PhaseClosed registran CommandEffects por run_id, blocker_id y closure_ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de lifecycle run/fase; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: OpenPhase conserva el modelo actual de reapertura de fases; si se quiere identidad fuerte por cada apertura necesita contrato nuevo de instancia de fase.
```

```text
Caso: supervision_gobierno_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'Test.*(ConcurrencyGate|AgentLeaseExpired|DirectorQuestionAnswered|AnswerDirectorQuestion).*|TestReplayDurableEventsV0Accepts(ConcurrencyGateRecorded|AgentLeaseExpired)'
Evidencia esperada: ConcurrencyGateRecorded, AgentLeaseExpired y DirectorQuestionAnswered registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de supervision/gobierno; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No calcula concurrencia, no ejecuta timers, no para agentes y no entrega preguntas al director; esos efectos pertenecen a director/adaptadores.
```

```text
Caso: programacion_operativa_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'Test.*(CapacityDecision|CapacityDecided|AgentStarted|AgentFailed|AgentStopConfirmed|DeliveryRegistered|RegisterDelivery).*|TestReplayDurableEventsV0Accepts(CapacityDecided|AgentStopConfirmed|DeliveryRegistered)'
Evidencia esperada: CapacityDecided, AgentStarted, AgentFailed, AgentStopConfirmed y DeliveryRegistered registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de programacion operativa; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No ejecuta runtime ni confirma procesos reales; esos efectos pertenecen a conectores/adaptadores.
```

```text
Caso: cierre_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'Test.*(CloseTask|TaskClosed|FinalValidation|RunClosed|CloseRun).*|TestReplayDurableEventsV0Accepts(TaskClosed|FinalValidationRegistered|RunClosed)'
Evidencia esperada: TaskClosed, FinalValidationRegistered y RunClosed registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de cierre; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No cierra fases automaticamente; `ClosePhase` sigue siendo transicion separada.
```

```text
Caso: revision_resultado_rework_replan_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'Test.*(ReviewResult|Rework|Replan).*|TestReplayDurableEventsV0Accepts(ReviewResultRecorded|ReworkRequested|ReplanDecisionRecorded)'
Evidencia esperada: ReviewResultRecorded, ReworkRequested y ReplanDecisionRecorded registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de review result/rework/replan; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No materializa followups ni relanza agentes; esos efectos pertenecen a director/scheduler por comandos separados.
```

```text
Caso: revision_solicitud_aceptacion_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'Test.*Review.*(Requested|Accepted|Request|Accept)|TestReplayDurableEventsV0AcceptsReview'
Evidencia esperada: ReviewRequested y ReviewAccepted registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de revision/aceptacion; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No valida la calidad del review ni materializa rework/cierre; esos cortes siguen separados.
```

```text
Caso: brainstorming_votacion_identidad_fuerte
Tipo: contract | replay | idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestHandleRequestBrainstormCommandV0|TestApplyBrainstormRequestedV0|TestReplayDurableEventsV0AcceptsBrainstormRequested|TestHandleRequestVoteCommandV0|TestApplyVoteRequestedV0|TestReplayDurableEventsV0AcceptsVoteRequested'
Evidencia esperada: BrainstormRequested y VoteRequested registran CommandEffects por ref; repetir el mismo comando es no-op, pero cambiar payload, idempotency key, command_id o event_id para la misma ref se rechaza.
Ultima ejecucion: 2026-05-06, ok, tests focalizados de brainstorming/votacion; go test -count=1 ./modulos/orquesta-core-workflow; go test -race -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh; ./scripts/verificar_producto_orquesta_v2.sh.
Riesgos: No valida calidad semantica del brainstorming ni de la votacion; esa evaluacion pertenece a agentes/director externos.
```

```text
Caso: record_replan_decision_desde_agent_failed
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RecordReplanDecision` y `ReplanDecisionRecorded` aceptan `source_ref` si apunta a `failed_agents` y la fase actual es `programacion`; fuera de programacion se rechaza; `ValidateOrchestrationRunV0` acepta la proyeccion.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: No crea capacidad ni agente de reemplazo; esos followups entran por comandos separados.
```

```text
Caso: record_replan_decision_desde_agent_work_assessed
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRecordReplanDecisionCommandV0.*AgentAssessment|TestValidateOrchestrationRunV0AcceptsReplanDecisionFromAgentAssessment'
Evidencia esperada: `RecordReplanDecision` y `ReplanDecisionRecorded` aceptan `source_ref` si apunta a `agent_assessments` y la fase actual es `programacion`; fuera de programacion se rechaza; `ValidateOrchestrationRunV0` acepta la proyeccion.
Ultima ejecucion: 2026-05-09, ok, pruebas focalizadas de replan desde `AgentWorkAssessed`.
Riesgos: No crea capacidad ni agente de reemplazo; esos followups entran por comandos separados para que el director/scheduler mantengan control explicito.
```

```text
Caso: stop_agent_rechaza_agente_fallido
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `StopAgent` y `AgentStopRequested` rechazan `agent_request_id` si ya existe en `failed_agents`; no se emite `StopRuntimeAgent` para un lanzamiento que ya fallo.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh.
Riesgos: No impide parada de agentes arrancados ni cancelacion de agentes solo solicitados.
```

```text
Caso: lifecycle_agente_no_permite_outcomes_contradictorios
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RegisterAgentStarted` y `AgentStarted` rechazan agentes ya fallidos o parados; `RegisterAgentFailed` y `AgentFailed` rechazan agentes ya arrancados o parados; los duplicados del outcome ya reflejado siguen siendo idempotentes.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; ./scripts/verificar_nucleo_orquesta_v2.sh.
Riesgos: No representa fallo de ejecucion despues de start; ese caso debe modelarse con contrato separado.
```

```text
Caso: register_delivery_exige_agente_arrancado_y_no_fallido
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RegisterDelivery` y `DeliveryRegistered` rechazan `agent_ref` si no existe `AgentStarted` previo o si el agente ya esta en `failed_agents`; las historias durables validas insertan `AgentStarted` antes de cualquier entrega.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: No relanza ni sustituye agentes fallidos; esas acciones pertenecen a replan/director.
```

```text
Caso: register_delivery_rechaza_agente_parado
Tipo: contract | replay | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RegisterDelivery` y `DeliveryRegistered` rechazan `agent_ref` si ese agente ya esta en `stopped_agents`; el E2E de programacion basura verifica que no se puede registrar delivery despues de `AssessAgentWork(garbage, stop_agent)`.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow; go test -count=1 ./modulos/orquesta-e2e -run TestE2EProgramacionTrabajoBasuraSolicitaParadaSinEntregaV0.
Riesgos: No relanza agentes ni replanifica; esas acciones pertenecen a replan/director.
```

```text
Caso: register_agent_stop_confirmed_despues_de_stop
Tipo: contract | replay | idempotency | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RegisterAgentStopConfirmed` solo progresa si existe `AgentStopRequested`, produce `AgentStopConfirmed`, no emite outbox, proyecta `confirmed_stopped_agents`, replay acepta duplicado exacto y el E2E real registra confirmacion despues de parar un proceso controlado.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp; go test -count=1 ./modulos/orquesta-e2e -run TestE2ERealGobiernaDosProcesosParaBasuraYEscalaCapacidadV0.
Riesgos: No confirma proceso productivo ni proveedor real; la senal entra por adaptador externo compacto.
```

```text
Caso: record_concurrency_gate_sin_scheduler
Tipo: contract | replay | idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RecordConcurrencyGate` produce solo `ConcurrencyGateRecorded`, outbox vacio, proyecta `concurrency_gates` compacto, es idempotente por `gate_ref`, valida consistencia de allow/block/ask_director y rechaza conflictos o detalles prohibidos.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: El workflow no calcula conflictos ni impide por si solo `RequestAgent`; el director debe llamar al gate antes de lanzar agentes.
```

```text
Caso: register_agent_lease_expired_observado
Tipo: contract | replay | idempotency | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RegisterAgentLeaseExpired` produce solo `AgentLeaseExpired`, outbox vacio, proyecta `agent_lease_expirations` compacto, exige agente solicitado y `run_ref` correcto, rechaza `continue`, tiempo invalido, refs prohibidas y conflictos por `lease_ref`.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Riesgos: La accion recomendada no ejecuta parada, fallo ni replan; esos efectos quedan para comandos posteriores.
```

```text
Caso: record_replan_decision_desde_rework
Tipo: contract | replay | idempotency | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RecordReplanDecision` produce solo `ReplanDecisionRecorded`, outbox vacio, proyecta `replan_decisions` compacto, exige `ReworkRequested` previo como `source_ref`, exige `task_ref` existente, es idempotente por `replan_ref` y rechaza conflictos.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Riesgos: `followup_refs` no materializa tareas, capacidad ni agentes; esos efectos quedan para comandos posteriores.
```

```text
Caso: request_rework_changes_rejected_replay_idempotencia
Tipo: contract | replay | idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RequestRework` produce solo `ReworkRequested`, outbox vacio, proyecta `rework_requests` compacto, acepta solo `ReviewResultRecorded(status=changes_requested|rejected)` para la misma revision/entrega, es idempotente por `rework_request_ref` y rechaza conflictos.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: `ReworkRequested` no relanza agentes ni registra replan; `RecordReplanDecision` sigue siendo un corte separado.
```

```text
Caso: accept_review_exige_review_result_accepted
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `AcceptReview` y `ReviewAccepted` rechazan aceptar si no existe `ReviewResultRecorded(status=accepted)` para el mismo `review_request_id` y `delivery_ref`; `changes_requested` no habilita aceptacion; los flujos validos registran resultado aceptado antes de aceptar.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: `changes_requested` y `rejected` siguen sin rework automatico; esa promocion pertenece a `RequestRework`/`RecordReplanDecision`.
```

```text
Caso: record_review_result_replay_idempotencia_invariantes
Tipo: contract | replay | idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RecordReviewResult` acepta `accepted|changes_requested|rejected`, produce solo `ReviewResultRecorded`, outbox vacio, replay deduplica duplicados exactos, `ReviewResults` proyecta ref/status/review/delivery sin duplicar y rechaza fase no `revision`, review inexistente, entrega inexistente y conflictos por `review_result_ref`.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: El rework y la decision de aceptar/cerrar tareas quedan para comandos futuros; este corte solo registra resultado compacto.
```

```text
Caso: ask_director_retry_outbox_y_no_rebloqueo
Tipo: idempotency | durability | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Si `DirectorQuestionRaised` ya esta proyectado y falta entrega externa, repetir `AskDirector` reconstruye solo `SendDirectorQuestion`; si la pregunta ya tiene `DirectorQuestionAnswered`, repetir `AskDirector` es no-op y no vuelve a crear `RunBlocked`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: El ACK de entrega a UI/MCP/CLI pertenece a persistence/dispatcher; el core no confirma lectura humana/IA.
```

```text
Caso: retry_outbox_pendiente_capacity_agent
Tipo: idempotency | durability
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Si `CapacityRequested` ya esta proyectado pero todavia no hay `CapacityDecided`, repetir `RequestCapacity` no emite eventos y reconstruye solo `RequestCapacityDecision`; si `AgentRequested` ya esta proyectado pero no hay `AgentStarted`/`AgentFailed`/`AgentStopRequested`, repetir `RequestAgent` no emite eventos y reconstruye solo `LaunchRuntimeAgent`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: Persistence debe deduplicar message_id/idempotency_key y registrar ACK; el core no sabe si un dispatcher externo ya entrego el mensaje.
```

```text
Caso: capacity_decided_barrera_lanzamiento_agente
Tipo: contract | replay | idempotency | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-e2e
Evidencia esperada: `RegisterCapacityDecision` produce `CapacityDecided` sin outbox y proyecta `capacity_decisions`; `RequestAgent`/`AgentRequested` rechazan `capacity_request_ref` si no hay decision durable previa; director y e2e registran decision antes de lanzar agentes.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-e2e.
Riesgos: La decision compacta no sustituye al contrato rico de `orquesta-capacity`; solo guarda refs y niveles seguros.
```

```text
Caso: agent_lifecycle_started_failed_opaco
Tipo: contract | replay | e2e
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: `RegisterAgentStarted` produce `AgentStarted` y proyecta `started_agents`; `RegisterAgentFailed` produce `AgentFailed` y proyecta `failed_agents`; ambos exigen `agent_request_id` ya solicitado y rechazan runtime/provider/HOME/DB/secretos. E2E real registra `AgentStarted` tras ACK de launch.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Riesgos: El detalle de proceso/ACK real vive en runtime/persistence; el core solo conserva refs opacas.
```

```text
Caso: gates_fase_activa_capacidad_agente_evaluacion
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director
Evidencia esperada: `RequestCapacity`, `RequestAgent`, `AssessAgentWork`, `CapacityRequested`, `AgentRequested` y `AgentWorkAssessed` rechazan `phase_id` si no coincide con `current_phase`; el smoke progresivo abre `programacion` antes de capacidad/agentes.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director ./modulos/orquesta-persistence ./modulos/orquesta-runtime ./modulos/orquesta-capacity ./modulos/orquesta-e2e.
Riesgos: Las decisiones ricas de capacidad siguen fuera del core; el core solo conserva la barrera durable compacta.
```

```text
Caso: assess_agent_work_reintenta_parada_pendiente
Tipo: idempotency | durability
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Si `AgentWorkAssessed` se aplico pero `AgentStopRequested`/outbox no llegaron a persistirse, repetir `AssessAgentWork action=stop_agent` emite solo `AgentStopRequested` y `StopRuntimeAgent`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: La unicidad productiva de outbox sigue perteneciendo a persistence/dispatcher.
```

```text
Caso: answer_director_question_desbloquea_blocker_asociado
Tipo: contract | replay | idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `AnswerDirectorQuestion` valido produce `DirectorQuestionAnswered`, no emite outbox, proyecta respuesta compacta por `question_id` con refs opacas y desbloquea solo si el blocker activo es `director-question-<question_id>`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: La entrega de la pregunta al director sigue fuera del core; el desbloqueo no debe limpiar blockers de otra causa.
```

```text
Caso: answer_director_question_sin_detalles_prohibidos
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Comando, evento y JSON rechazan secretos, provider/proveedor, HOME, DB, runtime, prompts, transcripts, adaptadores concretos y contexto masivo; `decision` y `evidence_refs` se conservan como datos compactos/opacos.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: La lista negativa debe mantenerse sincronizada con nuevos conectores.
```

```text
Caso: review_result_v0_dto_puro
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `NewReviewResultV0` acepta `accepted`; `changes_requested` y `rejected` no habilitan cierre de tarea; rechaza detalles prohibidos, payload masivo, refs vacias y status desconocido; JSON valido sin provider/HOME/OAuth/DB/prompts/transcripts.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow; git diff --check -- modulos/orquesta-core-workflow.
Riesgos: El DTO no esta conectado a `RequestReview`/`AcceptReview`; un comando durable futuro debe versionar la integracion sin cerrar tareas implicitamente.
```

```text
Caso: capacity_command_split_sin_cambio_funcional
Tipo: contract | replay | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director
Evidencia esperada: La division de `capacity_command_v0.go` conserva `RequestCapacity` -> `CapacityRequested` -> outbox `RequestCapacityDecision`, validacion de detalles prohibidos, replay/idempotencia de `capacity_requests` y compatibilidad con `orquesta-director`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director.
Riesgos: Refactor mecanico; los helpers de validacion siguen compartidos con `capacity_outbox_v0.go`.
```

```text
Caso: agent_stop_split_sin_cambio_funcional
Tipo: contract | replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director
Evidencia esperada: La division de `agent_stop_v0.go` conserva `StopAgent` -> `AgentStopRequested` -> outbox `StopRuntimeAgent`, replay/idempotencia de `stopped_agents` y la integracion del director progresivo sin runtime real.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-director.
Riesgos: Refactor mecanico; el riesgo principal es mover helpers compartidos por `AssessAgentWork` sin cambiar firmas ni errores.
```

```text
Caso: assess_agent_work_continue_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `AssessAgentWork` con verdict acceptable/action continue produce `AgentWorkAssessed`, proyecta `assessment_ref` y no emite outbox.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La calidad de la evaluacion la decide un consumidor externo; el core solo valida y registra el contrato compacto.
```

```text
Caso: assess_agent_work_stop_agent
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `AssessAgentWork` con verdict loop_detected/action stop_agent produce `AgentWorkAssessed`, `AgentStopRequested` y outbox `StopRuntimeAgent` con target `agent_launcher`.

Caso: assess_agent_work_timeout_stop_agent
Modulo: orquesta-core-workflow
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestHandleAssessAgentWorkCommandV0TimeoutPuedePararAgente
Ultima ejecucion: 2026-05-15, ok
Evidencia esperada: `AssessAgentWork` con verdict timeout/action stop_agent produce `AgentWorkAssessed`, `AgentStopRequested` y `StopRuntimeAgent` con reason_code `timeout`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La parada real pertenece al adaptador del puerto `agent_launcher`.
```

```text
Caso: assess_agent_work_refs_idempotencia_y_accion
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Rechaza agente inexistente, `delivery_ref` inexistente y `stop_agent` con trabajo aceptable; `ReplayDurableEventsV0` acepta `AgentWorkAssessed` y duplicado exacto; repetir assessment ya reflejado no duplica eventos.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: `AgentAssessments` es proyeccion compacta; el detalle de evaluacion vive en el evento.
```

```text
Caso: assess_agent_work_sin_detalles_prohibidos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: El comando/evento rechaza payloads con DB, runtime, proveedor, modelo, HOME, OAuth, adaptadores concretos o secretos.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La lista negativa debe mantenerse sincronizada con nuevos puertos.
```

```text
Caso: assess_agent_work_ask_director_boundary
Tipo: contract | replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `AssessAgentWork` con verdict needs_revision/action ask_director emite solo `AgentWorkAssessed` y outbox vacio; `AskDirector` separado emite `DirectorQuestionRaised`, `SendDirectorQuestion` y, si bloquea, `RunBlocked`; replay/aplicar eventos proyecta `agent_assessments` y `director_questions`.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La consulta real sigue fuera del nucleo; el core solo conserva eventos y outbox logico.
```

```text
Caso: stop_agent_emite_evento_y_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `StopAgent` sobre un agente ya solicitado produce `AgentStopRequested` y outbox `StopRuntimeAgent` con target `agent_launcher`, payload compacto y sin proveedor/modelo/HOME/OAuth.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La ejecucion real de parada pertenece al puerto `agent_launcher`.
```

```text
Caso: stop_agent_replay_idempotente
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ApplyEventV0` proyecta `stopped_agents` una sola vez; `ReplayDurableEventsV0` acepta `AgentStopRequested` y un duplicado exacto sin avanzar secuencia ni duplicar refs.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La unicidad durable final depende de persistence/outbox futuros.
```

```text
Caso: stop_agent_rechaza_agente_inexistente
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `StopAgent` devuelve `transicion_invalida` si `agent_request_id` no existe en `run.agents`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: `Agents` sigue siendo una proyeccion compacta; datos operativos del agente viven fuera del core.
```

```text
Caso: command_event_catalog_v0_publicado_para_adaptadores
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `SupportedOrchestrationCommandTypesV0` y `SupportedOrchestrationEventTypesV0` devuelven copias defensivas con 29 entradas cada una y todos los valores son aceptados por los validadores del core.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-mcp
Riesgos: Nuevos comandos/eventos deben entrar por el catalogo antes de exponerlos en adaptadores MCP/web/CLI.
```

```text
Caso: run_state_serializable_sin_adaptadores
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: OrchestrationRunV0 serializado no contiene DSN, SQL, HOME, OAuth, tmux, Docker, proveedor LLM, transcript ni token.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La lista negativa debe ampliarse cuando aparezcan nuevos conectores.
```

```text
Caso: fases_v0_catalogo_cerrado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Todas las fases v0 permitidas se reconocen y una fase desconocida se rechaza con error publico.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Cambios de fases requieren version o consulta.
```

```text
Caso: eventos_v0_serializables_sin_detalles_prohibidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RunStarted`, `PhaseOpened` y `RunBlocked` serializan como JSON valido sin secretos, transcripts, SQL, DSN, HOME, OAuth, tmux, Docker, Git, proveedor ni modelo.
Ultima ejecucion: 2026-05-04, pasa.
Riesgos: La lista negativa debe revisarse al promover eventos nuevos.
```

```text
Caso: evento_desconocido_rechazado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ValidateOrchestrationEventV0` devuelve `evento_no_soportado` para un `event_type` no incluido en el contrato inicial.
Ultima ejecucion: 2026-05-04, pasa.
Riesgos: Debe mantenerse compatible con eventos versionados futuros.
```

```text
Caso: reducer_inicial_aplica_eventos_minimos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ApplyEventV0` crea run activo con catalogo v0 para `RunStarted`, deja una sola fase activa para `PhaseOpened` y bloquea con ref compacto para `RunBlocked`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Las transiciones invalidas completas quedan para handler/NCW-003.
```

```text
Caso: reducer_evento_desconocido_no_muta_estado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ApplyEventV0` devuelve error publico `evento_no_soportado` y conserva intacto el estado de entrada.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Nuevos eventos deben agregarse explicitamente al reducer.
```

```text
Caso: replay_inicial_reconstruye_estado_minimo
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ReplayEventsV0` usa replay durable estricto, aplica `RunStarted`, `PhaseOpened` y `RunBlocked` y reconstruye estado bloqueado con fase actual, blocker compacto y `last_sequence` correcta.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Nuevos eventos deben declarar su semantica de secuencia e idempotencia.
```

```text
Caso: replay_durable_ordenado
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ReplayDurableEventsV0` acepta `RunStarted`, `PhaseOpened` y `RunBlocked` con sequence 1, 2 y 3 y reconstruye el estado bloqueado sin duplicar fase ni blocker.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Nuevos eventos deben agregar su huella de progreso si mutan campos adicionales del estado.
```

```text
Caso: replay_durable_secuencia_rota
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Una lista que no empieza en `RunStarted`, cambia de `run_id` o salta `sequence` falla con error publico `secuencia_invalida`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La secuencia se valida sobre eventos de progreso unico; duplicados idempotentes no avanzan contador.
```

```text
Caso: replay_durable_duplicado_exacto
Tipo: idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Repetir el mismo evento por `event_id` o `idempotency_key` no duplica fase activa ni blocker si conserva la misma huella durable.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La deduplicacion es conservadora y no sustituye unicidad durable en persistence futura.
```

```text
Caso: replay_durable_duplicado_conflictivo
Tipo: idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Reusar una `idempotency_key` para otro progreso, payload o metadato durable (`sequence`, `causation_id`, `occurred_at`) falla con error publico `evento_conflictivo`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La politica de conflicto debe versionarse si nuevos eventos necesitan equivalencias mas ricas.
```

```text
Caso: outbox_send_director_question_serializable_sin_adaptadores
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `SendDirectorQuestion` serializa como JSON valido sin DB, HTTP, CLI, MCP, HOME, OAuth, tmux, Docker, Git, proveedores, secretos ni transcripts.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La lista negativa debe ampliarse al incorporar nuevos puertos de salida.
```

```text
Caso: outbox_tipo_desconocido_rechazado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ValidateOutboxMessageV0` devuelve `outbox_tipo_no_soportado` para un `message_type` fuera del catalogo v0.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Nuevos mensajes deben agregarse de forma explicita junto con su `target_port`.
```

```text
Caso: director_question_requiere_summary_source
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `NewDirectorQuestionV0` rechaza preguntas sin `summary` o sin `source_group`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: El handler futuro debe propagar el mismo error publico sin envolverlo en detalles de adaptador.
```

```text
Caso: outbox_payload_sin_contexto_masivo
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `ValidateOutboxMessageV0` rechaza payloads con claves de transcript, prompt, completion, raw text, full context o contexto masivo.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Los limites compactos pueden requerir version si el director necesita preguntas mas ricas.
```

```text
Caso: handler_start_run_puro
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `StartRun` devuelve eventos y outbox; no llama a DB, runtime, capacity, MCP ni filesystem.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La pureza se valida por estructura y tests; los adaptadores reales se prueban aparte.
```

```text
Caso: handler_open_phase_soportada
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `OpenPhase` sobre un run activo produce un unico `PhaseOpened` para una fase del catalogo v0; fase desconocida devuelve error publico `fase_no_soportada`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Las reglas completas de entrada/salida de fase quedan para cortes posteriores.
```

```text
Caso: handler_block_run_puro
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `BlockRun` produce un unico `RunBlocked`, outbox vacio y el reducer reconstruye estado `bloqueada` con blocker compacto.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Desbloqueo y multiples causas con politica completa quedan fuera de NCW-003.
```

```text
Caso: comando_desconocido_rechazado
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `HandleCommandV0` devuelve error publico `comando_no_soportado` para un `command_type` fuera del catalogo inicial.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Nuevos comandos deben agregarse explicitamente junto con payload y pruebas.
```

```text
Caso: idempotencia_start_run
Tipo: idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Repetir `StartRun` con la misma `idempotency_key` sobre estado que ya refleja el arranque no duplica `RunStarted`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Es idempotencia logica v0 sobre estado proyectado; la idempotencia productiva requiere persistence/outbox con claves unicas.
```

```text
Caso: idempotencia_open_phase_block_run
Tipo: idempotency
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Repetir `OpenPhase` o `BlockRun` con la misma `idempotency_key` sobre estado que ya refleja el progreso no emite nuevos eventos ni outbox.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Es idempotencia logica v0; la relacion durable comando-evento queda para NCW-005.
```

```text
Caso: ask_director_emite_evento_y_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `HandleCommandV0` acepta `AskDirector`; `HandleAskDirectorCommandV0` produce `DirectorQuestionRaised`, emite `SendDirectorQuestion` con target `director`, conserva payload compacto, envuelve errores de payload como errores de comando cuando entra por el contrato generico y, si `blocking=true`, emite tambien `RunBlocked` para reflejar el bloqueo sin ejecutar efectos externos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La entrega real de la pregunta pertenece a MCP/web/CLI/director; persistence/outbox aun debe imponer claves unicas en adaptadores.
```

```text
Caso: appspec_bridge_start_run_local
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `StartRunFromAppSpecV0` convierte un draft compacto en comando `StartRun` valido, con mapping determinista, refs opacas y sin imports de `orquesta-core` ni `orquesta-factory`.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: El puente real con `RegistrarProyectoDesdeAppSpec v0` queda bloqueado hasta promocion de contrato global.
```

```text
Caso: appspec_bridge_rechaza_detalles_infra
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: El draft local rechaza refs vacios y detalles DB/runtime/proveedor/HOME; el comando serializado no contiene esos detalles.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: La lista negativa debe versionarse si aparecen nuevos adaptadores o nombres de infraestructura.
```

```text
Caso: ncw_007_contratos_globales_minimos_workflow
Tipo: contract
Comando: git diff --check; go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `../../CONTRATOS.md` promueve `OrchestrationRun v0`, `OutboxMessage v0` y `DirectorQuestion v0` con propietario, consumidores, DTOs, invariantes, errores publicos y prohibiciones de DB/runtime/proveedor/HOME/OAuth/transcripts/contexto masivo; no cambia codigo Go.
Ultima ejecucion: 2026-05-04, ok, git diff --check; go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: Si un consumidor necesita payloads ricos, debe versionar contrato o elevar `CONSULTA AL DIRECTOR`.
```

## Microtareas

Detalle: `docs/pruebas_microtareas.md`.

```text
Caso: request_capacity_handler_emite_evento_y_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RequestCapacity`, valida payload compacto, emite `CapacityRequested` y outbox `RequestCapacityDecision` con target `capacity`, sin ejecutar efectos externos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La entrega real y la seleccion de capacidad quedan fuera del nucleo.
```

```text
Caso: capacity_requested_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `capacity_request_id` en `capacity_requests` sin duplicar; `ReplayDurableEventsV0` acepta `CapacityRequested` con sequence estricta y duplicado exacto idempotente.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: El evento guarda la solicitud compacta, no la decision final de capacidad.
```

```text
Caso: request_capacity_rechaza_detalles_prohibidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: Payload con proveedor/modelo/runtime/HOME/Codex/Claude/Ollama/vLLM se rechaza con error publico; outbox valido no contiene esos detalles.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La lista negativa debe ampliarse si aparecen nuevos adaptadores concretos.
```

```text
Caso: request_capacity_outbox_payload_tipado
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ValidateOutboxMessageV0` rechaza `RequestCapacityDecision` con payload sin campos obligatorios.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La entrega real del outbox sigue fuera del nucleo.
```

```text
Caso: request_agent_handler_emite_evento_y_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RequestAgent`, valida payload compacto, emite `AgentRequested` y outbox `LaunchRuntimeAgent` con target `agent_launcher`, sin ejecutar efectos externos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La entrega real y la seleccion de runtime quedan fuera del nucleo.
```

```text
Caso: agent_requested_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `agent_request_id` en `agents` sin duplicar; `ReplayDurableEventsV0` acepta `AgentRequested` con sequence estricta y duplicado exacto idempotente.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: El evento guarda la solicitud compacta, no la ejecucion real.
```

```text
Caso: request_agent_rechaza_detalles_prohibidos
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: Comando y replay de evento con proveedor/modelo/runtime/HOME/OAuth/Codex/Claude/Ollama/vLLM se rechazan con error publico; outbox valido no contiene esos detalles.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La lista negativa debe ampliarse si aparecen nuevos adaptadores concretos.
```

```text
Caso: launch_runtime_agent_outbox_payload_tipado
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ValidateOutboxMessageV0` rechaza `LaunchRuntimeAgent` con payload sin campos obligatorios.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: El nombre historico del mensaje contiene `Runtime`, pero el target y payload se mantienen logicos y compactos.
```

```text
Caso: close_phase_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `ClosePhase` para el run activo y la fase actual activa, emite un unico `PhaseClosed`, no emite outbox y rechaza fase no actual o payload con detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: La idempotencia del handler usa el evento de cierre ya reflejado en la proyeccion actual.
```

```text
Caso: phase_closed_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` marca la fase actual como `cerrada`, conserva `opened_at`, asigna `closed_at` desde `occurred_at`, actualiza `last_event_id/last_sequence` y `ReplayDurableEventsV0` acepta duplicado exacto.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: `ClosePhase` no avanza automaticamente a la siguiente fase; esa transicion sigue perteneciendo a `OpenPhase`.
```

```text
Caso: open_phase_sigue_funcionando
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `OpenPhase` mantiene el contrato previo tras registrar `ClosePhase` en los routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Riesgos: Las reglas completas entre fases se versionaran en cortes posteriores.
```

## Fases operativas

Detalle: `docs/pruebas_brainstorm.md`, `docs/pruebas_votaciones.md` y `docs/pruebas_decisiones.md`.
Detalle contratos de funcion: `docs/pruebas_contratos_funcion.md`.
Detalle entregas: `docs/pruebas_entregas.md`.
Detalle revisiones: `docs/pruebas_revisiones.md`; NCW-022 anade casos de `AcceptReview`.
Detalle cierre de tareas: `docs/pruebas_cierre_tareas.md`; NCW-023 anade casos de `CloseTask`.
Detalle validacion final: `docs/pruebas_validacion_final.md`; NCW-024 anade casos de `RegisterFinalValidation`.
Detalle cierre de run: `docs/pruebas_cierre_run.md`; NCW-025 anade casos de `CloseRun`; NCW-030 anade composicion por comandos hasta cierre.
Detalle artefactos de fase: `docs/pruebas_artefactos_fase.md`; NCW-068 anade `RegisterPhaseArtifact`.

```text
Caso: ncw_027_refactor_sin_cambio_funcional
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: La division de `reducer_v0_test.go`, `commands_v0.go`, `outbox_v0.go` y `director_question_command_v0.go` conserva comandos, eventos, errores publicos, invariantes y expectations existentes.
Ultima ejecucion: 2026-05-04, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: El cambio es mecanico; si una futura division requiere mover contratos o semantica debe elevar `CONSULTA AL DIRECTOR`.
```

```text
Caso: ncw_032_reducer_v0_dividido_sin_cambio_funcional
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core ./modulos/orquesta-director
Evidencia esperada: `ApplyEventV0` y `ReplayEventsV0` conservan contratos publicos; la division de lifecycle, proyecciones, helpers y clone mantiene eventos, errores, replay e invariantes existentes.
Ultima ejecucion: 2026-05-05, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core ./modulos/orquesta-director; git diff --check -- modulos/orquesta-core-workflow.
Riesgos: Refactor mecanico; nuevas proyecciones deben agregarse al fichero de responsabilidad adecuada para no volver a inflar el dispatch principal.
```

```text
Caso: ncw_055_idempotencia_efectos_outbox
Tipo: invariant
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: RequestCapacity, RequestAgent, StopAgent, AssessAgentWork y AskDirector aceptan retry exacto para reparar outbox pendiente, pero rechazan misma ref con otra idempotency key o payload. Los reducers rechazan eventos con mismo subject ref y distinta huella.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: Las refs sin outbox todavia requieren revision contrato a contrato antes de aplicar identidad fuerte.
```

```text
Caso: ncw_056_router_catalogo_sincronizado
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Cada comando/evento del catalogo publico tiene ruta interna y no existen rutas fuera del catalogo; `HandleCommandV0` y `ApplyEventV0` conservan errores publicos y comportamiento.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: El corte no migra validadores de payload; si se mueven despues, deben conservar orden de validacion y errores.
```

```text
Caso: ncw_057_identidad_planificacion
Tipo: invariant
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: AcceptDecision, PublishFunctionContract y CreateMicrotask aceptan retry exacto, pero rechazan misma ref con otra idempotency key o payload. Los reducers rechazan eventos con mismo subject ref y distinta huella.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow
Riesgos: No cubre todavia brainstorm, votacion, revisiones ni cierre; se deben cerrar como cortes independientes.
```

```text
Caso: ncw_066_tests_workflow_divididos
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: La division de tests de AskDirector, handler general y RequestAgent conserva comportamiento, errores publicos, idempotencia, conflictos reflejados y validaciones de seguridad.
Ultima ejecucion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: Refactor mecanico; no cambia codigo productivo. Los siguientes escenarios deben agregarse al fichero de responsabilidad adecuada.
```

```text
Caso: ncw_068_artefactos_fase_director
Tipo: contrato
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: `RegisterPhaseArtifact` registra artefactos compactos en fases no-programacion para agentes arrancados, rechaza `programacion`, valida replay durable e impide proyecciones huerfanas.
Ultima ejecucion: 2026-05-09, ok, go test -count=1 ./modulos/orquesta-core-workflow.
Riesgos: El core no guarda contenido ni lee ACKs; el cableado de receipts vive en adaptadores externos y queda cubierto por `orquesta-runtime-codex-delivery`.
```

```text
Caso: ncw_073_agent_stop_reason_projection
Tipo: contrato
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core
Evidencia esperada: `AgentStopRequested` proyecta `agent_stop_requests` con `agent_request_id` y `reason_code`; replay no duplica; razones con separador inseguro se rechazan; `DirectorRunStatsV0` expone `stop_reason_code`, `stop_reason_source` y `stop_reason_ref` por agente parado.
Ultima ejecucion: 2026-05-13, ok, go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core.
Riesgos: La proyeccion es compacta y no reemplaza al payload historico del event log; los conectores externos siguen siendo responsables de evidencias detalladas.
```

```text
Caso: ncw_074_artefacto_fase_tardio_agente
Tipo: contrato
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRegisterPhaseArtifactCommandV0AcceptsLateArtifactForAgentLaunchPhase|TestRegisterPhaseArtifactCommandV0RejectsLateArtifactForOtherAgentPhase|TestReplayDurableEventsV0AcceptsLatePhaseArtifactForAgentLaunchPhase'
Evidencia esperada: `AgentRequested` proyecta la fase de arranque del agente; `RegisterPhaseArtifact` acepta un artefacto tardio de esa fase aunque el run ya haya abierto otra fase, y rechaza el mismo agente si declara una fase distinta.
Ultima ejecucion: 2026-05-21, ok, comando focal anterior.
Riesgos: No relaja identidad de agente ni `programacion` como fase de artefacto; solo evita que una transicion posterior invalide ACKs tardios legitimos.
```
