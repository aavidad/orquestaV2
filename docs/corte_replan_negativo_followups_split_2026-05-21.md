# Corte replan negativo con followups materializados - 2026-05-21

## Alcance

Este corte cierra un tramo estrecho del Director Operativo: si una review
negativa ya genero una cadena causal durable
`DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(changes_requested|rejected) -> ReworkRequested -> ReplanDecisionRecorded`,
el `OperationalDirectorPlanStateV0` puede volver a abrir `wait_subagents` en dos
casos ya materializados:

- `split_task`, cuando sus `followup_refs` ya estan reflejados como
  `WorkflowTaskV0` del run;
- `retry_task`/`replace_agent`, cuando sus `followup_refs` contienen agentes ya
  reflejados en `run.Agents`.

No interpreta texto de review, no lanza Codex, no toca OPES y no espera todos
los agentes vivos. Usa refs opacas, `WorkflowTaskStore`, `run.Tasks`,
`run.Agents` y el scope de ola/cohorte/parent task cuando existe.

## Implementado

- `operationalDirectorPlanStateAfterReviewReworkReplanV0` conserva la
  observacion durable de review negativa.
- Si el `ReplanDecisionRecorded` es `split_task`, todos los followups estan en
  `run.Tasks`, existen en `DirectorTaskStore` y comparten scope, el state:
  - marca `review_deliveries` como `changes_requested`;
  - activa `wait_subagents` de nuevo;
  - cambia `active_wave_ref`, `active_cohort_ref` y `active_parent_task_ref` al
    scope de los followups;
  - guarda `task_refs`, `agent_refs`, `pending_agent_refs` y `wait_ref` de los
    followups;
  - mantiene `ReplanAttempts` idempotente en reentradas posteriores.
- Si el `ReplanDecisionRecorded` es `retry_task` o `replace_agent` y sus
  `followup_refs` contienen un agente nuevo ya reflejado en `run.Agents`, el
  state:
  - marca `review_deliveries` como `changes_requested`;
  - activa `wait_subagents` de nuevo sin scope de ola/cohorte;
  - usa `WaitAgentRefs` explicitos para esperar solo esos agentes nuevos;
  - al consumirse la espera, vuelve a `review_deliveries` con la task original
    y los agentes nuevos, limpiando refs de la review negativa anterior.
- Si falta cualquier pieza, mantiene el comportamiento anterior: review
  negativa observada y pendiente de replan materializable, sin inventar runtime
  ni relanzar trabajo.

## Pendiente

- `retry_task`/`replace_agent` solo consumen agentes ya reflejados en el run; no
  generan capacidad/agente desde `PlanState`.
- Tests fallidos siguen bloqueando con `required-tests-failed`; el replan
  automatico de esa rama queda pendiente.
- La recursion Codex padre-hijo-nieto sigue fuera de este corte.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-app-director-service -run 'TestUpdateOperationalDirectorPlanStateAfterLoopV0Review(ChangesRequestedIdempotenteYNoReentraWaitAntiguo|ChangesRequestedAbreWaitDeSplitFollowups|ChangesRequestedAbreWaitDeRetryAgent|NegativaRegistraReworkReplan)|TestContinueRequestWithOperationalDirectorPlanStateV0NoReentraReviewChangesRequested'
```
