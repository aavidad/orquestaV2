# Pruebas: orquesta-app-director-service

```bash
go test -count=1 ./modulos/orquesta-app-director-service
```

Cobertura esperada:

- arranque completo con puertos inyectados;
- resultado expone `director_tasks` sin que el transporte conozca scheduler ni
  outbox;
- autonomia alta arranca el equipo de directores mediante batch dispatcher
  inyectado;
- `DeliverySource` inyectado se compone dentro del servicio y registra un
  `PhaseArtifactRegistered` sin que REST/MCP/web conozcan scheduler;
- `DirectorDecisionSource` inyectado permite aplicar una decision del director
  desde el propio servicio;
- `ExternalWaiter` inyectado permite esperar decisiones tardias del director y
  reentrar al loop gestionado para arrancar agentes derivados;
- `ContinueAppDirectorV0` permite reentrar sobre un run existente sin recrear
  AppSpec/intake y usando solo puertos inyectados;
- `ContinueAppDirectorV0` deriva `WaitAgentRefs` desde `wait_cohort_ref`,
  `wait_wave_ref` o `wait_parent_task_ref` usando `DirectorTaskStore`, sin que
  Codex interprete semantica de ola/cohorte;
- `ContinueAppDirectorV0` materializa un `OperationalDirectorPlanV0` listo,
  guarda la task, lanza la microtarea y registra `WorkflowTaskWaitStateV0` para
  la ola materializada;
- `StartAppDirectorV0` puede recibir un `OperationalDirectorPlanV0` inicial,
  tomar `run_ref` del plan si falta en la request, publicar voto/decision/fase
  y contratos funcionales, materializar la ola, lanzar el agente de tarea y
  registrar `WorkflowTaskWaitStateV0` sin pasar por Codex/OPES ni runtime real;
- `ContinueAppDirectorV0` puede reentrar desde `OperationalDirectorPlanStateV0`
  con `operational_director_plan_ref`, sin volver al scope global del run;
- el plan state puede entrar en `review_deliveries` con entregas disponibles
  del scope y procesarlas de forma incremental/streaming; el cierre del paso
  espera a que no queden entregas pendientes y el loop quede `quiescent`;
- si ese avance deja el run en `programacion`, `ContinueAppDirectorV0` abre
  `revision` por comando workflow y ejecuta un pase acotado de review sobre el
  mismo scope;
- el plan state avanza de `review_deliveries` a `run_required_tests` o
  `replan_or_close` solo si la cadena causal de eventos del scope activo queda
  aceptada; una delivery de otra ola no avanza el state y outbox pendiente
  bloquea la transicion;
- el plan state avanza de `run_required_tests` a `replan_or_close` con
  `RequiredTestEvidenceV0` `passed` causal, bloquea con `failed` y no acepta
  evidencia de otra review;
- ante `failed` de un unico task causal, el plan state genera
  `QualityGateRecorded(blocked)` + `ReplanDecisionRecorded(retry_task)`,
  reabre `programacion` si venia de `revision`, conserva idempotencia en replay
  y no emite capacidad/agente fuera del scheduler;
- un `required-tests-failed` puede reabrir `wait_subagents` si existen quality
  gate bloqueante, replan causal y followups materializados; si el followup
  aparece tarde, la reentrada con plan state persistido recompone el wait
  acotado sin volver al agente antiguo;
- el replan por tests fallidos no reabre scopes multitarea con un unico replan
  parcial;
- el plan state observa review negativa cuando hay `ReworkRequested` y
  `ReplanDecisionRecorded` causales, marcando `changes_requested`, refs de
  rework/replan y `replan_attempts`;
- el replay `state-file` no duplica refs ni eventos cuando el wait ya expiro,
  cuando review negativa ya registro `ReworkRequested`/`ReplanDecisionRecorded`
  o cuando el cierre estaba bloqueado por prerequisitos y luego se reintenta;
- el cierre operativo no se invoca si hay plan state activo en
  `review_deliveries` o `run_required_tests`;
- tras cierre operativo exitoso, el plan state queda `closed`; si hay issues de
  cierre o source insuficiente, queda `blocked` con `closure_reason`;
- politica de cierre: `crear_app_completa` normal no puede cerrarse solo con
  documentacion; `debug` permite alcance reducido;
- reentrada automatica tras decisiones del director crea microtarea y arranca
  el agente de programacion sin pasos manuales intermedios;
- una microtarea creada por decisiones del director puede reentrar con scope
  acotado, consumir entrega y review aceptada ya reflejadas, ejecutar
  `RequiredTestRunner` inyectado y cerrar task/run sin esperar agentes ajenos al
  scope;
- errores de validacion de factory sin transporte;
- sin imports legacy, DB hardcodeada ni adaptadores concretos de producto,
  runtime real, state-file/run-file, MCP, web u OPES en codigo productivo del
  servicio.

Evidencia 2026-05-09:

- `TestStartAppDirectorV0StartsDirectorThroughInjectedPorts`;
- `TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch`;
- `TestStartAppDirectorV0ConsumesDirectorDecisionSource`;
- `TestStartAppDirectorV0IgnoresPersistedDirectorDecisionsAlreadyApplied`;
- `TestStartAppDirectorV0NoSaltaDecisionPendienteDelDirector`;
- `TestStartAppDirectorDecisionDeferredV0RetieneProgramacionHastaMicrotareas`;
- `TestStartAppDirectorV0ConsumesDecisionsAndRerunsProgrammingLoop`;
- `TestStartAppDirectorV0WaitsExternalBeforeConsumingDirectorDecisions`;
- `TestGuardStartAppDirectorClosurePolicyV0BloqueaAppCompletaSinCodigo`;
- `TestGuardStartAppDirectorClosurePolicyV0PermiteDebugReducido`;
- `TestGuardStartAppDirectorClosurePolicyV0PermiteAppCompletaConEvidencia`;
- `TestContinueAppDirectorV0BloqueaCierreAppCompletaNormalSinEvidencias`;
- `TestContinueAppDirectorV0PermiteCierreAppCompletaNormalConEvidencias`;
- `TestExistingDirectorLoopRequestV0DerivaWaitAgentRefsPorCohorte`;
- `TestExistingDirectorLoopRequestV0RegistraWaitStatePorOla`;
- `TestExistingDirectorLoopRequestV0RequiereTaskStoreConFiltroWait`;
- `TestExistingDirectorLoopRequestV0RefsExplicitosNoRequierenFiltro`;
- `TestContinueAppDirectorV0MaterializaOperationalDirectorPlanYEsperaOla`;
- `TestContinueAppDirectorV0ReentraDesdeOperationalDirectorPlanState`;
- `TestContinueAppDirectorV0PlanStateReentradaRequiereStore`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0RechazaActiveStepNoSoportado`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0ReentraReviewConScope`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0ReentraRunRequiredTestsConScope`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0RespetaWaitExplicito`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaAReviewTrasWaitConsumido`;
- `TestContinueAppDirectorV0AvanzaDeWaitAReviewAbriendoRevision`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewATestsRequeridos`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeTestsAReplanConEvidenciaPassed`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewNegativaRegistraReworkReplan`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0BloqueaTestsConEvidenciaFailed`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0RequiredTestsFailedSinReplanCausalPermaneceBloqueado`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedConReplanSinFollowupMaterializadoBloqueaHastaReentrada`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedBloqueadoReabreWaitConReplanPosterior`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedReentraConStateFile`;
- `TestOperationalDirectorPlanStateAfterRequiredTestsReplanV0NoReabreScopeMultitarea`;
- `TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0ConStateBloqueadoConservaPlanRef`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedConQualityGateReplanRetryAbreWait`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaTestsConEvidenciaDeOtraReview`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewAReplanSinTests`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewFueraDeScope`;
- `TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewConOutboxPendiente`;
- `TestMaybeCloseOperationalDirectorV0PasaRequiredTestEvidenceRefsDelPlanStateAlSource`;
- `TestMaybeCloseOperationalDirectorV0NoCierraConPlanStatePostWaitActivo`;
- `TestMaybeCloseOperationalDirectorV0CierraPlanStateTrasCierreOperativoExitoso`;
- `TestMaybeCloseOperationalDirectorV0BloqueaPlanStateConIssuesDeCierre`;
- `TestMaybeCloseOperationalDirectorV0BloqueaPlanStateSiSourceNoConstruyeRequest`;
- `TestContinueAppDirectorV0WaitExpiredStateFileRestartNoDuplicaEvidenceRefs`;
- `TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedReworkReplanNoDuplica`;
- `TestContinueRequestWithOperationalDirectorPlanStateV0StateFileRestartReintentaPrereqSinDuplicar`;
- `TestContinueAppDirectorV0DecisionPlanStateEjecutaRunnerYCierra`;
- cobertura indirecta desde `orquesta-app-codex-stack`:
  `TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion`;
- `TestStartAppDirectorV0ConsumesDirectorDeliverySource`;
- `TestComposeStartAppDirectorProviderV0IncludesReviewGateSource`;
- `TestContinueAppDirectorV0ProcessesReviewGateSource`;
- `TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy`;
- `TestStartAppDirectorV0GoalFirstBundleIncompletoNoCaeALoopLegacy`;
- `TestObserveAppDirectorGoalV0PersisteResultadoCompletoYClosure`;
- `TestObserveAppDirectorGoalV0BloqueaRunSiClosureNoAcepta`;
- `TestObserveAppDirectorGoalV0LanzaReworkGoalSiPolicyYPuertoDisponibles`;
- `TestObserveAppDirectorGoalV0BloqueaSiReworkGoalAgotaPresupuesto`;
- `TestObserveAppDirectorGoalV0BloqueaRunSiGoalTerminaInvalid`;
- `TestContinueAppDirectorV0GoalFirstContainerNoEjecutaLoopLegacySinPuertosLegacy`;
- `TestBuildContinueAppDirectorLoopRuntimeV0GoalFirstContainerCierraSinLoopLegacy`;
- `TestStartAppDirectorV0ReturnsFactoryValidationIssues`;
- `TestStartAppDirectorV0RequiresInjectedPorts`;
- `TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada`.

Evidencia 2026-06-25:

- `go test -count=1 ./modulos/orquesta-app-director-service -run 'TestObserveAppDirectorGoal|TestStartAppDirectorV0GoalFirst'`;
- `go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.

Evidencia 2026-06-28:

- `go test -count=1 ./modulos/orquesta-app-director-service -run 'GoalFirst|ContinueAppDirectorV0GoalFirst|BuildContinueAppDirectorLoopRuntimeV0GoalFirst'`;
- `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`;
- `go test -count=1 ./...`;
- `go vet ./...`;
- `staticcheck ./...`;
- `govulncheck ./...`.

Cobertura esperada: `goal-first` no arranca loop legacy en el arranque, pero al
observar un goal terminal cierra la run si la closure es aceptada. Si la closure
necesita rework, `ReworkPolicy.PreferNewGoal` esta activo, queda presupuesto y
hay `GoalReworkLauncher`, lanza un nuevo goal causal sobre el mismo `run_ref`
sin entrar en el loop legacy; si falta ese puerto, se agota el presupuesto, el
goal devuelve estado terminal `invalid` o el cierre exige recibo de dominio,
bloquea la run.
Tambien cubre que `continue` y el builder del runtime no reentran en el loop
legacy cuando hay `GoalWorkStateV0` persistido; devuelven la senal de
observacion requerida sin exigir `OutboxLedger` ni dispatchers legacy. Cuando
el estado completo falta pero existe `GoalWorkRunMarkerV0`, devuelven estado
pendiente con `app_director_goal_first_state_missing` en vez de ejecutar el
director historico.

Evidencia real 2026-05-09:

- Harness MCP real en `/tmp/orquesta_director_driven_real_run`.
- Resultado: `estado=ok`, `loop_status=quiescent`, fase `programacion`.
- Agentes arrancados por Orquesta: director, REST y web.
- Entregas registradas: `ack-ref-agenda-rest`, `ack-ref-agenda-web`.
- Proyecto generado en `/tmp/orquesta-director-driven-20260509190441/project`.
- Validacion externa: `go test ./...` en `apps/agenda-rest`, `node --check` en
  modulos JS de `apps/agenda-web`, sin ficheros de codigo sobre 300 lineas.

## Notas para nuevos tests

Elegir el nivel minimo que pruebe la frontera:

- helpers puros para normalizacion, politica de cierre y construccion de refs;
- tests de `ContinueAppDirectorV0` cuando el comportamiento dependa de puertos,
  run durable, outbox o `OperationalDirectorPlanStateV0`;
- tests de composicion fuera de este modulo cuando haya runtime real, Codex,
  OPES, REST, MCP, state-file o servidor.

Supuestos que deben quedar visibles en la prueba:

- `WaitAgentRefs` no vacio acota pending, wait e ingesta; vacio conserva modo
  legacy de run completo;
- `OperationalClosureSource` solo se consulta con loop quiescent, outbox cero y
  `PlanState` en `replan_or_close/running`;
- `RequiredTestEvidenceV0` debe ser causal, `passed`, pedida por ref y enlazada
  a la misma delivery/review aceptada; summaries textuales no cuentan;
- `domain_work` cierra por refs opacas y validacion de dominio, no por rutas,
  URLs, DB o nombres de conectores.

Errores que merecen prueba focal antes de tocar codigo:

- una entrega o evidencia de otra ola avanza el state;
- un retry de cierre duplica `TaskClosed`, validacion, `RunClosed`, quality gate
  o replan;
- una reentrada por `operational_director_plan_ref` vuelve al scope global;
- una validacion de forma rechaza alias o refs opacas reparables en vez de dejar
  que el adaptador/director normalice.
