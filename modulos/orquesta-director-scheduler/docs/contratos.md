# Contratos locales: orquesta-director-scheduler

## `BuildDirectorSchedulerTick v0`

```text
Tipo: caso_uso puro
Estado: candidato implementado
Entrada:
  - tick_ref
  - run_ref
  - occurred_at externo
  - snapshot compacto del run
  - lease_action_candidates explicitos del caller
  - phase_artifact_candidates explicitos del caller
  - delivery_candidates explicitos del caller
  - review_gate_candidates explicitos del caller
  - progress_supervision_candidates explicitos del caller
  - replan_followup_candidates explicitos del caller
  - work_claims compartidos de la ola de trabajo
  - work_candidates con claims y comandos candidates explicitos
Salida:
  - status: commands_ready | waiting | blocked | needs_director | quiescent
  - commands ordenados de orquesta-core-workflow
  - waiting_reasons
  - blocked_refs
  - summary compacto
Invariantes:
  - No aplica comandos ni eventos.
  - No persiste, no despacha outbox y no arranca runtime.
  - No inventa refs nuevas; capacity, gate y agent llegan como candidates.
  - Puede procesar varios `work_candidates` en un tick, manteniendo orden de entrada y orden interno capacity -> gate -> agent por candidate.
  - No salta CapacityDecided antes de RequestAgent.
  - No lanza agente sin gate de concurrencia evaluado o comando de gate en el mismo plan.
  - Si hay outbox pendiente, espera.
  - Prioridad fija: outbox pendiente > lease > phase_artifact > delivery > review_gate > progress > replan > work.
  - Si hay `lease_action_candidates`, procesa como maximo el primero y no evalua delivery, progress, replan ni work.
  - Si hay `phase_artifact_candidates`, procesa como maximo el primero y no evalua delivery, review_gate, progress, replan ni work.
  - Si hay `delivery_candidates`, procesa como maximo el primero y no evalua review_gate, progress, replan ni work.
  - Si hay `review_gate_candidates`, procesa como maximo el primero y no evalua progress, replan ni work.
  - Un `ReviewGateCandidate` solo es accionable en fase `revision`.
  - Si hay `progress_supervision_candidates`, procesa como maximo el primero usando `BuildAgentProgressSupervisionV0` y no evalua replan ni work.
  - Si el primer progress candidate apunta a un agente ya observado en
    `stopped_agents`, y hay `replan_followup_candidates` explicitos, el
    scheduler puede tratar esa supervision como ya materializada y evaluar
    replan en el mismo tick; no evalua work por esa excepcion.
  - Si el assessment del progress candidate ya es durable pero la parada
    logica aun no aparece en `stopped_agents`, reemite `AssessAgentWork`
    antes de cualquier replan para que el workflow materialice `StopAgent`.
  - Si hay `replan_followup_candidates`, usa `BuildReplanFollowupsV0` antes que work.
  - Para `split_task` de `review_rework`, puede ordenar
    `RecordReplanDecision -> OpenPhase -> CreateMicrotask` desde candidates
    explicitos y deduplicar por `snapshot.tasks`.
  - Si `snapshot.blocking_quality_gate_refs` trae refs y no hay `replan_followup_candidates` cuyo `decision_payload.source_ref` coincida con una de esas refs opacas, bloquea antes de evaluar work.
  - Un quality gate bloqueante no prepara candidates de documentacion/avance de fase como trabajo normal; requiere followup de replan explicito para ese gate o bloqueo contractual.
  - No evalua politicas de lease ni parsea expiraciones del workflow; solo consume candidates explicitos.
  - Si `lease_ref` ya esta en `expired_lease_refs`, no repite `RegisterAgentLeaseExpired` y queda `quiescent`.
  - Si el agente del lease no existe en `agents` ni `started_agents`, devuelve `needs_director/candidate_missing`.
  - Si el agente ya esta solicitado o arrancado, no repite RequestAgent.
  - No repite `AssessAgentWork` si `assessment_ref` ya esta en `agent_assessments`.
  - No repite `AskDirector` si `question_id` ya esta en `director_questions`; si ya esta en `director_answered_questions`, queda `quiescent`.
  - No repite `RecordReplanDecision` si `replan_ref` ya esta en `replan_refs`.
  - Si `replan_ref` ya existe pero no hay `capacity_request` durable ni planificada, puede emitir `RequestCapacity`; solo espera `CapacityDecided` cuando la solicitud de capacidad ya esta observada o planificada.
  - Si no hay candidates accionables pero existen agentes arrancados sin entrega
    o artefacto de fase observado, devuelve `waiting/agent_delivery_pending`,
    no `quiescent`.
  - Un progress candidate ya inservible porque el agente observado esta parado
    no debe ocultar otros agentes arrancados que siguen pendientes de entrega.
  - Un candidate waiting o blocked no impide comandos seguros para otros candidates listos.
  - No transporta DB, runtime, provider, modelo, HOME, OAuth, secretos, prompts ni transcripts.
```

## `RunSchedulingSnapshotV0`

```text
Tipo: DTO compacto
Estado: candidato implementado
Campos:
  - run_ref
  - current_phase_id
  - capacity_requests
  - capacity_decisions
  - concurrency_gates
  - agents
  - started_agents
  - failed_agents
  - stopped_agents
  - tasks
  - phase_artifacts
  - deliveries
  - reviews
  - review_results
  - accepted_reviews
  - rework_requests
  - expired_lease_refs
  - agent_assessments
  - director_questions
  - director_answered_questions
  - replan_refs
  - blocking_quality_gate_refs
  - pending_outbox_refs
Invariantes:
  - Solo refs opacas y fase actual.
  - Las refs compactas de assessment, preguntas respondidas/no respondidas y replan se usan solo para dedupe.
  - `reviews`, `review_results`, `accepted_reviews` y `rework_requests` se usan para dedupe y para verificar dependencias de revision sin payloads completos.
  - `blocking_quality_gate_refs` son refs opacas aportadas por el caller; el scheduler no parsea la proyeccion del gate.
  - `expired_lease_refs` son refs opacas aportadas por el caller; el scheduler no parsea proyecciones.
  - No contiene AppSpec, backlog, payloads completos, paths, provider, HOME ni runtime.
```

## `SchedulableReviewGateCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - request_review opcional
  - record_review_result opcional
  - accept_review opcional
  - request_rework opcional
  - evidence_refs
Invariantes:
  - El scheduler no evalua la calidad ni lee ACKs, tests, ficheros, logs,
    transcripts, DB, runtime ni proveedor.
  - El caller aporta candidates compactos ya calculados para `RequestReview`,
    `RecordReviewResult`, `AcceptReview` o `RequestRework`.
  - El scheduler emite como maximo el primer comando pendiente del candidate,
    en orden: `RequestReview`, `RecordReviewResult`, `AcceptReview`,
    `RequestRework`.
  - `RequestReview` exige entrega durable en `snapshot.deliveries`.
  - `RecordReviewResult` exige revision solicitada y entrega durable.
  - `AcceptReview` exige un resultado durable `accepted` para la misma revision
    y entrega; no acepta resultados `changes_requested` ni `rejected`.
  - `RequestRework` exige un resultado durable `changes_requested` o `rejected`
    para la misma revision y entrega; no se emite para resultados `accepted`.
  - `accepted_reviews` y `rework_requests` deduplican comandos ya durables.
  - Todos los `command_meta.run_id` deben coincidir con `run_ref`.
```

## `SchedulableDeliveryCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - command_meta para RegisterDelivery
  - payload RegisterDeliveryCommandPayloadV0
  - evidence_refs
Invariantes:
  - El scheduler no lee ACKs, archivos, logs, transcripts, DB ni runtime.
  - El caller debe convertir un receipt externo en payload compacto.
  - `command_meta.run_id` debe coincidir con `run_ref`.
  - `task_id` debe existir en `snapshot.tasks`.
  - `agent_ref` debe existir en `snapshot.agents` y `snapshot.started_agents`.
  - No registra entrega si el agente esta en `failed_agents` o `stopped_agents`.
  - No repite `RegisterDelivery` si `delivery_ref` ya esta en `snapshot.deliveries`
    o ya fue planificada en el mismo tick.
```

## `SchedulableLeaseActionCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - post_lease_action_input de orquesta-director.PostLeaseActionInputV0
  - evidence_refs
Invariantes:
  - El scheduler no crea ni evalua leases; consume candidates explicitos del caller.
  - `post_lease_action_input.run_ref` y `command_meta.run_id` deben coincidir con `run_ref`.
  - `agent_request_id` debe estar observado en `agents` o `started_agents`.
  - `BuildPostLeaseActionV0` construye RegisterAgentLeaseExpired, StopAgent o AskDirector.
  - Acciones sin followup local quedan como `needs_director` con solo RegisterAgentLeaseExpired.
```

## `SchedulableWorkCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - subject_claim_refs
  - claims locales del candidato; si `work_claims` existe, no contienen la ola completa
  - capacity_candidate opcional
  - agent_candidate opcional
  - gate_command_meta
  - gate_evidence_refs
  - evidence_refs
Invariantes:
  - El scheduler no crea los candidates; solo los consume.
  - Cada candidate debe apuntar al mismo run.
  - Si `work_claims` existe, los gates evaluan la concurrencia con ese claim-set compartido.
  - Si `work_claims` no existe, se usa `claims` del candidate para compatibilidad.
  - `RequestAgent` se construye solo despues de capacidad decidida y gate allow.
```

## `SchedulableProgressSupervisionCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - supervision_input de orquesta-director.AgentProgressSupervisionInputV0
  - evidence_refs
Invariantes:
  - El scheduler no evalua progreso; consume reportes candidatos ya aportados por el caller.
  - `BuildAgentProgressSupervisionV0` construye `AssessAgentWork` y, para stalled, `AskDirector` separado.
  - `assessment_ref`, `question_id`, `agent_request_id` y `run_ref` deben venir explicitos.
  - Dedupe por `agent_assessments`, `director_questions` y `director_answered_questions`.
  - Si `assessment_ref` ya es durable pero el comando de assessment calculado
    por el director implica `stop_agent` y el agente aun no aparece en
    `stopped_agents`, puede reconstruir `AssessAgentWork` para materializar la
    parada/outbox pendiente sin interpretar proveedor, modelo ni runtime.
  - Si el agente ya aparece en `stopped_agents`, el progress candidate no
    duplica la decision y puede dejar pasar un replan followup explicito.
```

## `SchedulableReplanFollowupCandidateV0`

```text
Tipo: DTO de entrada
Estado: candidato implementado
Campos:
  - candidate_ref
  - replan_followups_input de orquesta-director.ReplanFollowupsInputV0
  - evidence_refs
Invariantes:
  - El scheduler no decide replan; consume candidates explicitos del caller.
- `BuildReplanFollowupsV0` construye `RecordReplanDecision`, `RequestCapacity`, `RequestAgent` o `AskDirector`.
- `BuildReplanFollowupsV0` tambien puede construir `CreateMicrotask` para
  split de `review_rework` si el caller aporta payloads completos.
- `replan_ref`, capacity refs, agent refs y question ids deben venir explicitos.
- Dedupe por `replan_refs`, `tasks`, `capacity_requests`, `capacity_decisions`, `agents`, `director_questions` y `director_answered_questions`.
```
