# Pruebas locales: orquesta-director-scheduler

```text
Caso: scheduler_tick_progress_no_ack_interrupted_capacity_causal
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0(InterruptedWithoutAckStopsBeforeWork|CapacityLimitedStopPrecedesReplanFollowup|CapacityLimitedReemitsStopBeforeReplan|StoppedProgressYieldsToReplanAfterStopReflected|StoppedProgressDoesNotRepeatAfterStopReflected)'
Evidencia esperada: no_ack/interrupted produce assessment garbage/stop_agent;
capacity_limited produce assessment capacity_limited/stop_agent; un replan
explicito no se procesa hasta que la parada logica este reflejada en
stopped_agents; despues de stopped_agents no se duplica assessment y se permite
RecordReplanDecision -> RequestCapacity.
Estado: completada en SCH-014
```

```text
Caso: scheduler_tick_progress_stop_reconstruction_capacity_limited
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0(CapacityLimitedReconstructsMissingStop|StoppedWithoutAckReconstructsMissingStop)'
Evidencia esperada: un progress candidate `capacity_limited` o `stopped` con
assessment_ref ya durable y sin stopped_agent vuelve a emitir AssessAgentWork
stop_agent para que el workflow materialice la parada/outbox pendiente.
Estado: completada en SCH-013
```

```text
Caso: scheduler_tick_replan_split_create_microtasks
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run TestBuildDirectorSchedulerTickV0ReplanSplitCreatesMicrotasksAfterOpenPhase
Evidencia esperada: un ReplanFollowupCandidate `split_task` de `review_rework` en revision emite RecordReplanDecision, OpenPhase y CreateMicrotask para cada microtarea candidate; no inventa capacidad ni agente.
Estado: completada en SCH-012
```

```text
Caso: scheduler_tick_review_gate_accepted_flow
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run TestBuildDirectorSchedulerTickV0ReviewGateStagesReviewCommands
Evidencia esperada: un `ReviewGateCandidate` en fase `revision` emite primero
`RequestReview`; con revision durable emite `RecordReviewResult`; con resultado
durable `accepted` emite `AcceptReview`; con accepted ya reflejado queda
`quiescent`.
Estado: completada
```

```text
Caso: scheduler_tick_review_gate_rework_flow
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run TestBuildDirectorSchedulerTickV0ReviewGateStagesReworkCommand
Evidencia esperada: un `ReviewGateCandidate` con resultado
`changes_requested` emite `RequestReview`, despues `RecordReviewResult` y,
cuando el resultado no aceptado ya es durable, emite `RequestRework`; con
`rework_request` ya reflejado no duplica el comando.
Estado: completada
```

```text
Caso: scheduler_tick_review_gate_dependencies
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0(ReviewGateWaitsForDelivery|RejectsForeignRunReviewGateCandidate|RejectsForeignRunReworkCandidate)'
Evidencia esperada: el candidate espera si falta entrega durable y rechaza
comandos de revision o rework cuyo `run_id` no coincide con el run del tick.
Estado: completada
```

```text
Caso: scheduler_tick_inflight_no_bloquea_candidate_nuevo
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0(SchedulesReadyCandidateWhileOtherAgentInFlight|WaitsWhenStartedAgentHasNoDelivery|BuildsAgentsForMultipleDecidedCandidates)' -v
Evidencia esperada: un agente arrancado sin delivery mantiene espera externa
si no hay candidates nuevos, pero no bloquea un candidate listo y no
conflictivo. El tick produce gate+RequestAgent para el nuevo candidate.
Estado: completada el 2026-05-10
```

```text
Caso: scheduler_tick_anti_detalles_no_rechaza_refs_opacas
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0(AcceptsOpaque.*Refs|RejectsForbidden|ProgressRejectsForbidden)'
Evidencia esperada: CandidateRef/ClaimRef/AgentRef y evidence_refs con slugs opacos tipo db-admin/model-viewer/runtime/process/session no fallan por el filtro anti-detalles; summaries y reasons con detalles operativos reales siguen rechazandose como payload invalido.
Estado: completada
```

```text
Caso: scheduler_tick_work_claims_compartidos_detectan_conflictos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run TestBuildDirectorSchedulerTickV0UsaWorkClaimsCompartidosParaConflictos
Evidencia esperada: dos candidates con claims locales pequenos usan
`work_claims` compartido para evaluar el gate; si sus write-set se solapan se
registran gates y no se lanza ningun agente.
Estado: completada
```

```text
Caso: scheduler_tick_payload_admite_cohorte_inicial_pequena
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-app-director-intake ./modulos/orquesta-app-director-service
Evidencia esperada: una solicitud de autonomia alta puede preparar 4 candidates
de director, crear sus solicitudes de capacidad, lanzar sus agentes y quedar en
wait_external esperando artefactos sin partir la cohorte por el limite compacto.
Estado: completada
```

```text
Caso: scheduler_tick_phase_artifact_candidate_registers_artifact
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0.*PhaseArtifact'
Evidencia esperada: PhaseArtifactCandidates produce RegisterPhaseArtifact en fase no-programacion cuando el agente existe y esta arrancado; no repite artefacto ya registrado; espera si falta started_agent; rechaza run ajeno.
Estado: completada
```

```text
Caso: scheduler_tick_delivery_candidate_registers_delivery
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0.*Delivery|TestBuildDirectorSchedulerTickV0RejectsForeignRunDeliveryCandidate'
Evidencia esperada: DeliveryCandidates produce RegisterDelivery cuando task/agente/started_agent existen, no repite delivery ya registrada, espera si el agente no esta arrancado y rechaza run ajeno.
Estado: completada
```

```text
Caso: scheduler_tick_quality_gate_blocked_requires_replan_before_phase_advance
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: quality gate bloqueante en programacion bloquea work de documentacion/avance si no hay followup cuyo source_ref apunte al gate; con ReplanFollowupCandidates valido procesa replan antes que work.
Estado: completada en SCH-009
```

```text
Caso: scheduler_tick_quality_gate_replan_capacity_states
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: con replan_ref ya registrado pero sin capacity_request durable vuelve a emitir RequestCapacity; con capacity_request pendiente espera CapacityDecided; con capacidad decidida puede pedir agente retry/replacement desde candidates explicitos.
Estado: completada
```

```text
Caso: scheduler_tick_capacity_first
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: claim listo sin capacidad solicitada produce solo RequestCapacity.
Estado: completada en SCH-001
```

```text
Caso: scheduler_tick_gate_agent_after_capacity
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: con CapacityDecided y claims seguros produce RecordConcurrencyGate y RequestAgent.
Estado: completada en SCH-001
```

```text
Caso: scheduler_tick_waits
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: outbox pendiente, capacidad pendiente o agente ya solicitado no duplican comandos.
Estado: completada en SCH-001
```

```text
Caso: scheduler_tick_waits_for_inflight_agent_delivery
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run 'TestBuildDirectorSchedulerTickV0.*StartedAgent'
Evidencia esperada: agente arrancado sin delivery ni phase_artifact deja el tick en waiting/agent_delivery_pending; si ya hay entrega queda quiescent.
Estado: completada
```

```text
Caso: scheduler_tick_blocks_conflicts
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: conflicto de write-set registra gate pero no produce RequestAgent.
Estado: completada en SCH-001
```

```text
Caso: scheduler_tick_e2e_workflow
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-e2e -run TestE2EDirectorSchedulerTickProgramaAgenteSinRuntimeRealV0
Evidencia esperada: primer tick produce RequestCapacity y outbox RequestCapacityDecision; tras CapacityDecided externo, segundo tick produce RecordConcurrencyGate y RequestAgent; workflow emite LaunchRuntimeAgent. No hay runtime real.
Estado: completada en SCH-002
```

```text
Caso: scheduler_tick_e2e_lease_stop_workflow
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-e2e -run TestE2EDirectorSchedulerTickLeaseStopAgentSinRuntimeRealV0
Evidencia esperada: LeaseActionCandidates alimenta BuildPostLeaseActionV0; workflow registra AgentLeaseExpired sin outbox y StopAgent separado emite StopRuntimeAgent. No hay runtime real.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_progress_loop_priority_over_work
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: ProgressSupervisionCandidates con loop_detected usa BuildAgentProgressSupervisionV0, produce AssessAgentWork stop_agent antes que work y el workflow posterior emite StopRuntimeAgent.
Estado: completada en SCH-006/SCH-008
```

```text
Caso: scheduler_tick_progress_stalled_question_dedupe
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: stalled produce AssessAgentWork + AskDirector separado; pregunta pendiente deja waiting y pregunta respondida queda quiescent.
Estado: completada en SCH-005/SCH-006
```

```text
Caso: scheduler_tick_progress_does_not_repeat_assessment
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: assessment_ref ya presente en agent_assessments no repite AssessAgentWork, pero reconstruye StopAgent si falta la parada logica.
Estado: completada en SCH-005/SCH-006
```

```text
Caso: scheduler_tick_stopped_progress_candidate_keeps_inflight_wait
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler -run TestBuildDirectorSchedulerTickV0StoppedProgressCandidateKeepsInFlightWait
Evidencia esperada: un candidate de progreso obsoleto para un agente parado no oculta que otro agente arrancado sigue pendiente de entrega.
Estado: completada
```

```text
Caso: scheduler_tick_lease_priority_over_progress
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: lease candidate se procesa antes que ProgressSupervisionCandidates y work.
Estado: completada en SCH-008
```

```text
Caso: scheduler_tick_multi_capacity_requests
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: dos candidates sin capacidad solicitada/decidida producen dos RequestCapacity ordenados y sin refs inventadas.
Estado: completada en SCH-003
```

```text
Caso: scheduler_tick_lease_stop_agent_priority
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: lease stop_agent produce RegisterAgentLeaseExpired + StopAgent y no procesa work candidate.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_lease_ask_director
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: lease ask_director produce RegisterAgentLeaseExpired + AskDirector.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_lease_unsupported
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: retry/replan_task/alert_only quedan needs_director con solo RegisterAgentLeaseExpired.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_lease_expired_quiescent
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: lease_ref ya presente en expired_lease_refs no repite RegisterAgentLeaseExpired.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_pending_outbox_blocks_leases
Tipo: unit
Comando: go test -count=1 .
Evidencia esperada: outbox pendiente mantiene waiting y no procesa leases.
Estado: completada en SCH-004
```

```text
Caso: scheduler_tick_multi_agents_after_capacity
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: dos candidates con capacidad decidida producen gate+agent por candidate, sin repetir gates grabados ni agentes solicitados.
Estado: completada en SCH-003
```

```text
Caso: scheduler_tick_replan_requests_capacity_before_agent
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: ReplanFollowupCandidates usa BuildReplanFollowupsV0; primero registra decision de replan y RequestCapacity, sin RequestAgent hasta CapacityDecided externo.
Estado: completada en SCH-007
```

```text
Caso: scheduler_tick_replan_replacement_dedupe
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: replan_ref ya presente no repite RecordReplanDecision; agent refs ya solicitadas, fallidas, paradas o bloqueadas no se reutilizan.
Estado: completada en SCH-005/SCH-007 y revalidada con stopped_agents.
```

```text
Caso: scheduler_tick_replan_question_and_priority
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: ask_director de replan produce pregunta separada, respeta preguntas pendientes/respondidas y tiene prioridad sobre work.
Estado: completada en SCH-007/SCH-008
```

```text
Caso: scheduler_tick_pending_outbox_blocks_progress_and_replan
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: pending_outbox_refs deja waiting y bloquea progress, replan y work sin listar ni despachar outbox.
Estado: completada en SCH-008
```

```text
Caso: scheduler_tick_pending_candidate_does_not_stop_ready_candidate
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: un candidate con capacidad pendiente no produce comandos ni bloquea gate+agent de otro candidate listo.
Estado: completada en SCH-003
```

```text
Caso: scheduler_tick_conflict_does_not_stop_safe_candidate
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-director-scheduler
Evidencia esperada: un candidate con gate block registra su gate y blocked_refs, mientras otro candidate seguro produce RequestAgent.
Estado: completada en SCH-003
```
