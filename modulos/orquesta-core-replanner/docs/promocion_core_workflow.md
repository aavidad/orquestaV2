# Promocion a orquesta-core-workflow

Estado: cerrada para el corte actual.

## Corte minimo aplicado

La promocion no debe mover la politica completa al workflow. `orquesta-core-workflow` solo deberia aceptar eventos compactos ya decididos por este modulo o por el director.

## Comandos/eventos promocionados

```text
RecordReviewResult -> ReviewResultRecorded
Campos minimos:
  - review_result_ref
  - run_id
  - task_id
  - review_request_id
  - delivery_ref
  - status
  - summary
  - evidence_refs
Regla:
  - status accepted permite despues AcceptReview.
  - changes_requested/rejected permite RequestRework.
```

```text
RequestRework -> ReworkRequested
Campos minimos:
  - rework_ref
  - run_id
  - task_id
  - source_ref
  - mode: revise_same_task | split_task | ask_director | replace_agent
  - summary
  - evidence_refs
Regla:
  - no crea tarea, agente, capacidad ni outbox por si solo.
```

```text
RecordReplanDecision -> ReplanDecisionRecorded
Campos minimos:
  - replan_ref
  - run_id
  - task_id
  - source_ref
  - accepted_action
  - followup_refs
  - summary
  - evidence_refs opcional
Regla:
  - cualquier retry/split/sustitucion debe quedar trazado con una decision explicita.
  - aceptar solo acciones del catalogo `ReplanRecommendedActionV0`.
  - idempotencia por `replan_ref`; replay del mismo payload no duplica efectos.
  - `followup_refs` referencia comandos/eventos/tareas posteriores, pero el registro no los crea.
```

## Criterios de promocion

- DTOs puros de `orquesta-core-replanner` con tests: cumplido.
- Sin imports a runtime, persistence, DB, reviewapp ni cmd/controlplane: cumplido.
- Sin prompts, transcripts, diffs grandes ni paths sensibles en eventos: cumplido.
- Cambios en `orquesta-core-workflow` divididos por comando/evento/test: cumplido.

## Nota de cierre

`AcceptReplan` no se promueve como comando independiente. La decision aceptada se registra con `RecordReplanDecision -> ReplanDecisionRecorded`, y cualquier capacidad/agente/tarea posterior se materializa con comandos separados o candidates explicitos del director.
