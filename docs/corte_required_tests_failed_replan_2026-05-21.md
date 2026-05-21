# Corte required-tests-failed con replan materializado - 2026-05-21

## Alcance

Este corte conecta la rama negativa de `run_required_tests` con el ciclo de
replan ya existente, sin crear una fuente nueva de replan desde evidencias de
test.

La regla es conservadora:

- si una `RequiredTestEvidenceV0` causal tiene `status=failed`, el
  `PlanState` sigue bloqueando como antes;
- si ademas existe una `QualityGateRecorded(blocked)` reflejada en el run,
  enlazada al task y a la evidencia fallida, y una
  `ReplanDecisionRecorded` cuya `source_ref` es esa quality gate, entonces el
  `PlanState` vuelve a abrir `wait_subagents` sobre los followups ya
  materializados.

## Implementado

- `run_required_tests` consume la evidencia `failed` y busca una quality gate
  bloqueante por eventos durables.
- El replan se acepta solo si la quality gate esta reflejada en
  `run.QualityGates` y el `ReplanDecisionRecorded` esta reflejado en
  `run.ReplanDecisions`.
- Para `retry_task`/`replace_agent`, el state espera solo agentes followup ya
  presentes en `run.Agents`.
- Para `split_task`, se reutiliza la misma ruta de followups `WorkflowTaskV0`
  materializados usada por review negativa.
- Mientras el `PlanState` queda activo en `wait_subagents`, el cierre generico
  no invoca `OperationalClosureSource`.
- Si falta cualquier pieza causal, el comportamiento anterior queda intacto:
  `PlanState` bloqueado con `required-tests-failed`.

## Pendiente

- Este corte no genera `RecordQualityGate` ni `RecordReplanDecision`; solo los
  consume cuando ya existen. La generacion automatica por puerto/fuente queda
  para otro tramo.
- La reentrada automatica desde `required-tests-failed` bloqueado hacia
  followups materializados tarde queda pendiente; hoy el replan solo reabre
  espera si los followups estan listos en ese mismo avance.
- Falta cubrir smokes reales de runner + quality gate + replan en el stack.
- `replan_or_close` sigue pendiente como evaluador explicito de cierre vs
  replan.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-app-director-service -run 'TestMaybeCloseOperationalDirectorV0NoCierraConPlanStatePostWaitActivo|TestUpdateOperationalDirectorPlanStateAfterLoopV0(RequiredTestsFailedSinReplanCausalPermaneceBloqueado|TestsFailedConReplanSinFollowupMaterializadoBloquea|TestsFailedConQualityGateReplanRetryAbreWait|BloqueaTestsConEvidenciaFailed)'
```
