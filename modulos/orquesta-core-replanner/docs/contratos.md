# Contratos locales: orquesta-core-replanner

## Contrato candidato: ReplanProposalV0

```text
Tipo: DTO puro
Estado: candidato
Campos:
  - replan_ref
  - run_ref
  - task_ref
  - source_ref
  - reason_code
  - recommended_action: split_task | retry_task | replace_agent | escalate_capacity | ask_director | abort_task
  - capacity_request_ref opcional
  - replacement_role opcional
  - summary
  - evidence_refs opcional
Invariantes:
  - No transporta runtime, provider, modelo concreto, HOME, OAuth, DB, credenciales, prompts ni transcripts.
  - `source_ref` apunta a review, assessment, timeout, failed_agent o pregunta del director.
  - No ejecuta la replanificacion; solo propone una decision compacta.
```

## Contrato candidato: ReviewReworkSignalV0

```text
Tipo: DTO puro de entrada
Estado: candidato
Campos:
  - signal_ref
  - run_ref
  - task_ref
  - review_request_ref
  - delivery_ref
  - review_result_ref
  - review_status: changes_requested | rejected
  - reason_ref opcional
  - summary
  - evidence_refs opcional
Invariantes:
  - No acepta `accepted`; el flujo feliz sigue en core-workflow.
  - No cierra, reabre ni crea tareas por si solo.
  - `changes_requested` y `rejected` pueden recomendar `split_task` si el plan
    externo trae despues las microtareas explicitas; este contrato solo produce
    la propuesta compacta.
```

## Contrato candidato: AgentReworkSignalV0

```text
Tipo: DTO puro de entrada
Estado: candidato
Campos:
  - signal_ref
  - run_ref
  - task_ref
  - agent_request_id
  - source_ref
  - assessment_status: garbage | loop_detected | capacity_limited | timeout | needs_revision
  - summary
  - evidence_refs opcional
Invariantes:
  - No para agentes ni relanza capacidad; solo traduce evidencia a propuesta.
  - `capacity_limited` y `timeout` son veredictos validos del workflow para
    replanificar despues de `stop_agent`; no equivalen a retry automatico.
  - Si la fuente fue runtime, se guarda como ref compacta, no como log/transcript.
```

## Contrato candidato: AgentFailedReplanSignalV0

```text
Tipo: DTO puro de entrada
Estado: candidato puro implementado
Campos:
  - signal_ref
  - run_ref
  - task_ref
  - agent_request_id
  - source_ref
  - failure_reason_code
  - retryable
  - requested_action: replace_agent | ask_director
  - replacement_role opcional
  - reason_ref opcional
  - summary
  - evidence_refs opcional
Invariantes:
  - `source_ref` debe ser el `agent_request_id` fallido ya proyectado por workflow.
  - `replace_agent` exige `replacement_role` y no reutiliza implicitamente el agente fallido.
  - No acepta `retry_task`; un nuevo intento requiere decision durable y followups separados.
  - No transporta runtime, provider, modelo concreto, HOME, OAuth, DB, credenciales, prompts ni transcripts.
```

## Contrato candidato: ReplanDecisionV0

```text
Tipo: evento candidato para core-workflow
Estado: candidato puro implementado
Campos:
  - replan_ref
  - run_ref
  - task_ref
  - source_ref
  - accepted_action
  - followup_refs
  - summary
  - evidence_refs opcional
Invariantes:
  - `accepted_action` debe pertenecer al catalogo `ReplanRecommendedActionV0`.
  - `replan_ref`, `run_ref`, `task_ref`, `source_ref` y `followup_refs` son refs opacas obligatorias.
  - Debe ser idempotente por `replan_ref`.
  - Si crea nuevas tareas/agentes/capacidad, esos efectos se materializan con comandos separados o outbox.
  - No transporta DB, runtime, provider, HOME, OAuth, modelo, secretos, prompts ni transcripts.
```

## Contratos promocionados a core-workflow

```text
RecordReviewResult -> ReviewResultRecorded
Uso: registrar durablemente ReviewResultV0 antes de aceptar, pedir cambios o rechazar.
Regla: no dispara efectos; solo proyeccion compacta e idempotente.
Estado: promocionado en workflow; cubierto por RPL-005 y E2E.
```

```text
RequestRework -> ReworkRequested
Uso: abrir una intencion durable de rework desde changes_requested/rejected o trabajo basura.
Regla: no crea agente/tarea por si solo; se compone despues con capacidad/agente/director.
Estado: promocionado en workflow; cubierto por RPL-005 y E2E.
```

```text
RecordReplanDecision -> ReplanDecisionRecorded
Uso: vincular decision explicita de replan con review, assessment, timeout o respuesta del director.
Regla: evita retry automatico y bucles; cada replan debe tener decision durable.
Estado: promocionado en workflow; cubierto por RPL-005 y E2E.
```
