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
- Si el primer avance deja `required-tests-failed` bloqueado porque el followup
  aun no estaba reflejado, una reentrada posterior con el followup ya presente
  en el run reabre `wait_subagents` y persiste el nuevo `PlanState`.
- La reentrada tardia queda probada tambien con `orquesta-state-file`
  recreando el store entre bloqueo y reapertura.
- Mientras el `PlanState` queda activo en `wait_subagents`, el cierre generico
  no invoca `OperationalClosureSource`.
- Si falta cualquier pieza causal, el comportamiento anterior queda intacto:
  `PlanState` bloqueado con `required-tests-failed`.

## Pendiente

- Este corte no genera `RecordQualityGate` ni `RecordReplanDecision`; solo los
  consume cuando ya existen. La generacion automatica por puerto/fuente queda
  para otro tramo.
- El replan por tests fallidos en scopes multitarea queda bloqueado de forma
  conservadora hasta soportar una decision causal completa por task fallida.
- Falta prueba focal de reentrada tardia con `split_task`; la ruta reutilizada
  es la misma de followups materializados en `WorkflowTaskStore`.
- Si una llamada trae un wait scope explicito obsoleto junto al plan ref, ese
  scope explicito sigue teniendo prioridad; queda pendiente reconciliarlo con
  el `PlanState` bloqueado.
- Falta cubrir smokes reales de runner + quality gate + replan en el stack.
- `replan_or_close` sigue pendiente como evaluador explicito de cierre vs
  replan.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-app-director-service -run 'Test(EnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0ConStateBloqueadoConservaPlanRef|OperationalDirectorPlanStateAfterRequiredTestsReplanV0NoReabreScopeMultitarea|ContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailed(ReentraConStateFile|BloqueadoReabreWaitConReplanPosterior)|UpdateOperationalDirectorPlanStateAfterLoopV0(RequiredTestsFailedSinReplanCausalPermaneceBloqueado|TestsFailedConReplanSinFollowupMaterializadoBloqueaHastaReentrada|TestsFailedConQualityGateReplanRetryAbreWait|BloqueaTestsConEvidenciaFailed)|MaybeCloseOperationalDirectorV0NoCierraConPlanStatePostWaitActivo)'
```
