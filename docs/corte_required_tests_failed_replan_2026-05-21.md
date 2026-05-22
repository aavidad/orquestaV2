# Corte required-tests-failed con replan materializado - 2026-05-21

## Alcance

Este corte conecta la rama negativa de `run_required_tests` con el ciclo de
replan ya existente y deja una fuente automatica acotada para el caso de un
unico task causal.

La regla es conservadora:

- si una `RequiredTestEvidenceV0` causal tiene `status=failed` y el scope es
  un unico task, `app-director-service` registra automaticamente
  `QualityGateRecorded(blocked)` + `ReplanDecisionRecorded(retry_task)`;
- la emision solo crea la evidencia causal y refs futuras de capacidad/agente:
  `RequestCapacity` y `RequestAgent` siguen saliendo por el scheduler/outbox;
- si ademas los followups ya estan reflejados, el `PlanState` vuelve a abrir
  `wait_subagents`; si aun no lo estan, queda bloqueado con
  `required-tests-failed` y una reentrada posterior deja avanzar el scheduler.

## Implementado

- `run_required_tests` consume la evidencia `failed` y busca una quality gate
  bloqueante por eventos durables.
- Si no existe y el scope es un unico task causal, emite `OpenPhase(programacion)`
  cuando viene de `revision`, `QualityGateRecorded(blocked)` y
  `ReplanDecisionRecorded(retry_task)` con refs estables.
- El replan se acepta solo si la quality gate esta reflejada en
  `run.QualityGates` y el `ReplanDecisionRecorded` esta reflejado en
  `run.ReplanDecisions`.
- `orquesta-orchestration-core` incluye un provider generico que reconstruye
  `ReplanFollowupCandidates` desde quality gate bloqueante + replan reflejado;
  el scheduler mantiene la secuencia capacidad antes que agente.
- Para `retry_task`/`replace_agent`, el state espera solo agentes followup ya
  presentes en `run.Agents`.
- Para `split_task`, se reutiliza la misma ruta de followups `WorkflowTaskV0`
  materializados usada por review negativa.
- Si el primer avance deja `required-tests-failed` bloqueado porque el followup
  aun no estaba reflejado, una reentrada posterior con el followup ya presente
  en el run reabre `wait_subagents` y persiste el nuevo `PlanState`.
- La reentrada tardia queda probada tambien con `orquesta-state-file`
  recreando el store entre bloqueo y reapertura.
- Corte posterior 2026-05-22: si una reentrada trae un `wait_agent_refs`,
  `wait_wave_ref`, `wait_cohort_ref` o `wait_parent_task_ref` obsoleto junto a
  `operational_director_plan_ref`, el `PlanState` cargado desde store manda y
  sustituye ese scope. Si no hay store de `PlanState`, se conserva el wait
  explicito legacy.
- Mientras el `PlanState` queda activo en `wait_subagents`, el cierre generico
  no invoca `OperationalClosureSource`.
- Si falta cualquier pieza causal, el comportamiento anterior queda intacto:
  `PlanState` bloqueado con `required-tests-failed`.

## Pendiente

- El replan por tests fallidos en scopes multitarea queda bloqueado de forma
  conservadora hasta soportar una decision causal completa por task fallida.
- Falta prueba focal de reentrada tardia con `split_task`; la ruta reutilizada
  es la misma de followups materializados en `WorkflowTaskStore`.
- Falta cubrir smokes reales de runner + quality gate + replan en el stack.
- `replan_or_close` ya es puerta explicita de cierre cuando hay `PlanState`
  activo: solo el step `replan_or_close` en `running` permite invocar la fuente
  de cierre, y la ausencia de source o task store bloquea con causa durable.
  Sigue pendiente el evaluador completo de cierre vs replan para otros blockers.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-app-director-service -run 'Test(EnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0ConStateBloqueadoConservaPlanRef|OperationalDirectorPlanStateAfterRequiredTestsReplanV0NoReabreScopeMultitarea|ContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailed(ReentraConStateFile|BloqueadoReabreWaitConReplanPosterior)|UpdateOperationalDirectorPlanStateAfterLoopV0(RequiredTestsFailedSinReplanCausalPermaneceBloqueado|TestsFailedConReplanSinFollowupMaterializadoBloqueaHastaReentrada|TestsFailedConQualityGateReplanRetryAbreWait|BloqueaTestsConEvidenciaFailed)|MaybeCloseOperationalDirectorV0(NoCierraConPlanStatePostWaitActivo|NoCierraConPlanStateFueraDeReplanOrCloseRunning|BloqueaPlanStateSinClosureSourceEnReplanOrClose|BloqueaPlanStateSinTaskStoreEnReplanOrClose))'
```
