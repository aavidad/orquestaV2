# Contratos locales: orquesta-director

`BootstrapProyectoDesdeAppSpec v0` queda promovido como contrato compartido minimo en `../CONTRATOS.md`. Este documento conserva el detalle canonico local.

## Plantilla

```text
Nombre:
Tipo: caso_uso | dto | error | puerto
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas:
Estado:
```

## `BootstrapProyectoDesdeAppSpec v0`

```text
Nombre: BootstrapProyectoDesdeAppSpec v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: web/MCP/CLI futuros mediante adaptadores finos
Entrada:
  - AppSpecV0 validada
  - BacklogInicialPropuestoV0 compatible
  - idempotency_key
  - requested_by
  - occurred_at
  - request_id/correlation_id opcionales
Salida:
  - resumen compacto del RegistroProyectoAceptadoV0
  - project_ref y app_spec_ref opacas derivadas del resultado de core
  - OrchestrationCommandV0 StartRun
  - OrchestrationCommandResultV0 con RunStarted
Invariantes:
  - Usa solo contratos publicos de factory, core y core-workflow.
  - No persiste, no publica eventos reales y no arranca runtime.
  - El workflow recibe refs opacas, no AppSpec completa ni backlog completo.
  - El resultado debe ser determinista para la misma idempotency_key y entrada.
  - No contiene DB, SQL, runtime, provider, OAuth, HOME ni secretos.
Errores:
  - director_bootstrap_invalido
  - errores publicos propagados de core/core-workflow
Pruebas:
  - caso feliz desde AppSpec valida genera registro y RunStarted.
  - idempotency_key vacia falla.
  - salida compacta no contiene detalles prohibidos.
Estado: implementado_local_promovido_global_minimo
```

## `BuildAgentProgressSupervision v0`

```text
Nombre: BuildAgentProgressSupervision v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: supervisores/adaptadores futuros del director
Entrada:
  - OrchestrationCommandMetaV0
  - AgentProgressReportV0 validado por contrato publico de orquesta-runtime
  - phase_id
  - task_ref opcional
  - delivery_ref opcional
  - assessment_ref
  - question_id requerido cuando status=stalled
Salida:
  - OrchestrationCommandV0 AssessAgentWork siempre
  - OrchestrationCommandV0 AskDirector opcional y separado para stalled
Invariantes:
  - Usa solo contratos publicos de orquesta-runtime y orquesta-core-workflow.
  - No persiste, no ejecuta runtime real, no lee DB, no llama proveedores ni adaptadores.
  - progressing produce acceptable/continue/low.
  - stalled produce needs_revision/ask_director/medium o high segun contadores y pregunta compacta no bloqueante al director.
  - loop_detected produce loop_detected/stop_agent/critical; la parada efectiva queda en outbox de core-workflow.
  - stopped produce acceptable/continue/low con resumen compacto de parada observada.
  - No propaga summary bruto, transcripts, proveedor, DB, HOME, OAuth ni contexto grande.
Errores:
  - director_agent_progress_supervision_invalida
  - errores publicos propagados de orquesta-core-workflow al construir comandos
Pruebas:
  - loop_detected genera AssessAgentWork stop_agent y core emite AgentWorkAssessed, AgentStopRequested y StopRuntimeAgent.
  - stalled genera AssessAgentWork ask_director y AskDirector no bloqueante; core emite SendDirectorQuestion sin RunBlocked.
  - reporte invalido devuelve error publico local.
  - JSON de salida/comandos no contiene detalles prohibidos.
Estado: implementado_local
```

## `BuildPostLeaseAction v0`

```text
Nombre: BuildPostLeaseAction v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: adaptadores/supervisores futuros del director
Entrada:
  - OrchestrationCommandMetaV0
  - run_ref
  - agent_request_id
  - lease_ref
  - reason_code
  - observed_at
  - recommended_action
  - evidence_refs opcionales
  - question_id requerido cuando recommended_action=ask_director
Salida:
  - OrchestrationCommandV0 RegisterAgentLeaseExpired siempre
  - OrchestrationCommandV0 StopAgent opcional y separado cuando recommended_action=stop_agent
  - OrchestrationCommandV0 AskDirector opcional y separado cuando recommended_action=ask_director
  - followup_status `unsupported/needs_director` para acciones sin efecto local
Invariantes:
  - Usa solo contratos publicos de orquesta-core-workflow.
  - No persiste, no ejecuta runtime real, no lee DB, no llama proveedores ni adaptadores.
  - RegisterAgentLeaseExpired se construye primero y no inventa efectos secundarios.
  - StopAgent y AskDirector son comandos separados; retry/mark_failed/mark_stopped/replan_task/alert_only quedan sin comando secundario.
  - No propaga transcripts, proveedor, DB, HOME, OAuth ni contexto grande.
Errores:
  - director_post_lease_action_invalida
  - errores publicos propagados de orquesta-core-workflow al construir comandos
Pruebas:
  - stop_agent genera RegisterAgentLeaseExpired y StopAgent separados.
  - ask_director genera RegisterAgentLeaseExpired y AskDirector separado.
  - retry queda como unsupported/needs_director sin outbox ni comando secundario.
  - input invalido devuelve error publico local y JSON de salida no contiene detalles prohibidos.
Estado: implementado_local
```

## `RunOutboxDispatchCycle v0`

```text
Nombre: RunOutboxDispatchCycle v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: adaptadores/supervisores futuros del director
Entrada:
  - OutboxLedgerPortV0 inyectado
  - OutboxDispatcherPortV0 inyectado
  - run_id
  - target_port
  - dispatched_at determinista para ACK
  - mensajes OutboxMessageV0 opcionales para guardar antes de listar
Salida:
  - contadores compactos de save/pending/dispatched/failed
  - snapshots publicos de ACK registrados
  - issues publicos locales cuando haya rechazo, ledger issues o fallo de dispatcher
Puertos:
  - OutboxLedgerPortV0: SavePending, ListPending, MarkDispatched con DTOs locales del director.
  - OutboxDispatcherPortV0: DispatchOutboxMessageV0 recibe OutboxMessageV0 y devuelve dispatch_ref/evidence_refs.
Invariantes:
  - Usa solo OutboxMessageV0 de orquesta-core-workflow como contrato externo productivo.
  - No importa orquesta-persistence, orquesta-runtime ni implementaciones concretas en codigo productivo.
  - No persiste por si mismo, no abre DB, no ejecuta runtime real, no usa procesos, colas, goroutines, sleeps, provider, HOME ni OAuth.
  - dispatch_ref es obligatorio en cada intento y lo aporta el dispatcher/adaptador; el caso de uso no inventa refs de proveedor.
  - El ciclo guarda mensajes opcionales, lista pendientes por run_id/target_port y procesa en orden serial.
  - ACK dispatched retira el pendiente segun ledger.
  - Si el dispatcher falla, registra ACK failed con error_code compacto, devuelve snapshot publico y detiene el ciclo; ese ACK failed es terminal para el message_id.
Errores:
  - director_outbox_dispatch_cycle_invalido
  - director_outbox_dispatch_cycle_ledger
  - director_outbox_dispatch_cycle_dispatch_failed
  - dispatch_ref_requerido como issue compacto
Pruebas:
  - caso feliz guarda/lista/despacha/ackea y deja pendientes en cero.
  - fallo de dispatcher registra ACK failed y retira el mensaje fallido de pendientes.
  - input sin ledger/dispatcher/run_id/target_port falla con error publico local.
  - JSON de resultado/error no contiene detalles prohibidos.
Estado: implementado_local
```

## `BuildConcurrencyGateAgentRequests v0`

```text
Nombre: BuildConcurrencyGateAgentRequests v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: planificador/scheduler futuro del director
Entrada:
  - WorksetClaimV0 ya candidatos para un run
  - subject_claim_refs a evaluar
  - OrchestrationCommandMetaV0 para RecordConcurrencyGate
  - CandidateAgentRequestV0 con meta/payload RequestAgent ya dados por claim_ref
Salida:
  - ConcurrencyGateEvaluationV0
  - OrchestrationCommandV0 RecordConcurrencyGate siempre que la entrada sea valida
  - OrchestrationCommandV0 RequestAgent solo cuando decision=allow_request_agent
Invariantes:
  - Usa solo contratos publicos de orquesta-core-concurrency y orquesta-core-workflow.
  - No persiste, no aplica comandos, no despacha outbox y no arranca runtime.
  - RecordConcurrencyGate se construye para allow/block/ask_director.
  - RequestAgent se construye solo para candidates cuyos claim_ref estan en subject_claim_refs permitidos.
  - Block/ask_director no construyen RequestAgent aunque existan candidates.
Errores:
  - director_concurrency_gate_agent_requests_invalido
  - errores publicos propagados de core-workflow al construir comandos
Pruebas:
  - allow produce RecordConcurrencyGate y RequestAgent por cada subject candidato.
  - write-set solapado produce block y no produce RequestAgent.
  - allow sin candidate por subject devuelve error local.
Estado: implementado_local
```

## `BuildReplanFollowups v0`

```text
Nombre: BuildReplanFollowups v0
Tipo: caso_uso
Version: v0
Propietario: orquesta-director
Consumidores: replanner/scheduler futuro del director
Entrada:
  - OrchestrationCommandMetaV0 para RecordReplanDecision
  - RecordReplanDecisionCommandPayloadV0 explicito
  - source_kind opcional; `quality_gate_blocked` activa followups aplicables en programacion
  - candidatos CreateMicrotask opcionales con meta/payload explicitos
  - candidato RequestCapacity opcional con meta/payload explicitos
  - candidato RequestAgent opcional con meta/payload explicitos
  - candidato AskDirector opcional con meta/payload explicitos
  - blocked_agent_refs opcional con agentes que no se pueden reutilizar
Salida:
  - OrchestrationCommandV0 RecordReplanDecision siempre que la entrada sea valida
  - OrchestrationCommandV0 OpenPhase opcional
  - OrchestrationCommandV0 CreateMicrotask opcional para split de review_rework
  - OrchestrationCommandV0 RequestCapacity opcional
  - OrchestrationCommandV0 RequestAgent opcional
  - OrchestrationCommandV0 AskDirector opcional
  - followup_status compacto
Invariantes:
  - Usa solo contratos publicos de orquesta-core-workflow.
  - No persiste, no aplica comandos, no despacha outbox y no arranca runtime.
  - Siempre construye RecordReplanDecision antes de cualquier followup.
  - retry_task y replace_agent pueden construir RequestCapacity y RequestAgent solo desde candidates dados.
  - RequestAgent se rechaza si su `agent_request_id` aparece en `blocked_agent_refs`.
  - escalate_capacity puede construir RequestCapacity solo desde candidate dado.
  - ask_director puede construir AskDirector solo desde candidate dado.
  - review_rework con split_task puede construir CreateMicrotask solo desde
    candidates dados; no guarda el detalle ni aplica comandos.
  - quality_gate_blocked no crea microtareas ni rework automatico en programacion porque esos comandos pertenecen a planificacion o revision.
  - quality_gate_blocked puede construir RequestCapacity y, tras capacidad decidida, RequestAgent desde candidates dados.
  - quality_gate_blocked construye AskDirector para split_task, abort_task o ausencia de followup aplicable con candidate explicito de consulta.
  - split_task, abort_task o acciones sin candidate quedan en `needs_director/unsupported` sin comandos secundarios.
Errores:
  - director_replan_followups_invalido
  - errores publicos propagados de orquesta-core-workflow al construir comandos
Pruebas:
  - retry_task con candidates dados produce decision, capacidad y agente.
  - candidate de agente bloqueado se rechaza antes de construir RequestAgent.
  - replace_agent sin candidates no inventa comandos.
  - escalate_capacity ignora agent candidate y construye solo capacidad.
  - ask_director construye consulta solo si candidate viene dado.
  - quality_gate_blocked genera replan/capacidad aplicables o consulta trazable sin abrir documentacion.
Estado: implementado_local
```
