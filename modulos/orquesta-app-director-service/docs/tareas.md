# Tareas: orquesta-app-director-service

## APP-DIR-SVC-001

Objetivo: convertir `AppSpecRequestV0` en arranque de director.

Estado: hecho.

Validacion: `TestStartAppDirectorV0StartsDirectorThroughInjectedPorts`.

## APP-DIR-SVC-002

Objetivo: rechazar requests invalidas como errores publicos de factory.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ReturnsFactoryValidationIssues`.

## APP-DIR-SVC-003

Objetivo: impedir dependencias legacy, persistencia hardcodeada o adaptadores
concretos dentro del servicio.

Estado: hecho.

Validacion: `TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada`.
La guarda cubre imports directos de codigo productivo hacia DB, `cmd`,
state-file/run-file, runtime concreto, Codex, OPES, MCP, web/http y `os/exec`.

## APP-DIR-SVC-004

Objetivo: devolver el equipo de directores preparado por el intake sin exponer
detalles internos del scheduler.

Estado: hecho.

Validacion: `go test -count=1 ./modulos/orquesta-app-director-service`.

## APP-DIR-SVC-005

Objetivo: demostrar que el servicio arranca el equipo de directores con batch
dispatcher inyectado cuando la solicitud pide autonomia alta.

Estado: hecho.

Validacion: `TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch`.

## APP-DIR-SVC-006

Objetivo: consumir decisiones compactas producidas por el director sin intervencion manual.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ConsumesDirectorDecisionSource`.

## APP-DIR-SVC-007

Objetivo: reentrar automaticamente al loop tras decisiones del director y lanzar agentes de programacion para microtareas creadas.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ConsumesDecisionsAndRerunsProgrammingLoop`.

## APP-DIR-SVC-008

Objetivo: continuar un run existente sin recrear AppSpec/intake para consumir
ACKs y decisiones tardias del director.

Estado: hecho.

Validacion: `go test -count=1 ./modulos/orquesta-app-director-service` y
`TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion` desde el stack.

## APP-DIR-SVC-009

Objetivo: impedir que el director cierre una solicitud normal sin evidencias
minimas del `request_kind`.

Estado: hecho.

Validacion: `TestGuardStartAppDirectorClosurePolicyV0BloqueaAppCompletaSinCodigo`,
`TestGuardStartAppDirectorClosurePolicyV0PermiteDebugReducido` y
`TestGuardStartAppDirectorClosurePolicyV0PermiteAppCompletaConEvidencia`.

## APP-DIR-SVC-010

Objetivo: probar la politica de cierre desde `ContinueAppDirectorV0`, no solo
desde helpers unitarios.

Estado: hecho.

Validacion: `TestContinueAppDirectorV0BloqueaCierreAppCompletaNormalSinEvidencias`
y `TestContinueAppDirectorV0PermiteCierreAppCompletaNormalConEvidencias`.

## APP-DIR-SVC-011

Objetivo: usar `wait_cohort_ref`, `wait_wave_ref` y `wait_parent_task_ref` como
entrada principal para continuar runs del Director sin esperar todos los agentes
vivos.

Estado: hecho como derivacion de refs y primer estado durable de espera.

Validacion actual:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run 'Test.*WaitAgentRefs|TestExistingDirectorLoopRequestV0'
```

Actualizacion 2026-05-22: la salida positiva de esa espera ya avanza hasta
review/tests/cierre en el scope operativo aunque el run global tenga otros
agentes vivos. Cubierto por
`TestContinueAppDirectorV0DecisionPlanStateEjecutaRunnerYCierra`.

## APP-DIR-SVC-012

Objetivo: conectar el Director Operativo V1 con el camino productivo de
`StartAppDirectorV0`/`ContinueAppDirectorV0`.

Estado: hecho para `ContinueAppDirectorV0`, para decisiones del director en
`StartAppDirectorV0` y para arranque directo con `OperationalDirectorPlanV0`
inicial siempre que la composicion aporte contratos funcionales explicitos.

Alcance hecho:

1. recibir o construir `OperationalDirectorPlanV0` fuera del nucleo puro;
2. materializar `launch_subagents` con `OperationalDirectorPlanMaterializerV0`;
3. derivar waits de la ola materializada;
4. llamar al loop existente con esos `WaitAgentRefs`;
5. registrar `WorkflowTaskWaitStateV0` si hay writer disponible.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run TestContinueAppDirectorV0MaterializaOperationalDirectorPlanYEsperaOla
```

Actualizacion: `StartAppDirectorV0` crea `OperationalDirectorPlanStateV0`
reentrable cuando el director aplica decisiones que generan `WorkflowTaskV0`
operativas, con `operational_director_plan_ref` explicito o ref por defecto.
Tambien acepta un `OperationalDirectorPlanV0` completo en la request inicial:
antes de materializar abre causalmente votacion/decision, acepta la decision,
abre programacion por defecto, publica los contratos funcionales recibidos y
reusa el materializador existente. Validacion adicional:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run TestStartAppDirectorV0OperationalDirectorPlanMaterializaYEsperaScope
```

## APP-DIR-SVC-013

Objetivo: permitir que review/replan cree microtareas `split_task` desde el
servicio del Director.

Estado: hecho.

Cambio: `composeStartAppDirectorProviderV0` inyecta `DirectorTaskStore` en
`ReviewReworkReplanCandidateProviderV0`.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run TestComposeStartAppDirectorProviderV0
```

## APP-DIR-SVC-014

Objetivo: convertir la salida de una espera del Director Operativo en
review/replan/cierre durable.

Estado: cerrado offline para el ciclo probado. Cerrado el camino positivo
`wait -> review_deliveries -> run_required_tests -> replan_or_close -> close`
para microtareas nacidas de decisiones del director y el consumo durable de
`run_required_tests` desde `RequiredTestEvidenceV0`.
Aniadido pase operativo de continuidad: cuando `wait_subagents` queda consumido
y el state pasa a `review_deliveries`, `ContinueAppDirectorV0` abre `revision`
si el run seguia en `programacion` y relanza un pase acotado del loop para
aplicar review gate sin llamada manual adicional.
Aniadido primer runner por puerto: si el `PlanState` entra en
`run_required_tests` sin evidencias causales ya guardadas, el servicio invoca
`RequiredTestRunnerPortV0`, persiste evidencias durables y reevalua el avance.
La observacion negativa de review con `ReworkRequested` y
`ReplanDecisionRecorded` queda probada como observacion durable del `PlanState`.
El cierre operativo ya marca el state como `closed` o `blocked` con
`closure_reason`. Existe adaptador externo opt-in en
`modulos/orquesta-runtime-required-test` y wiring desde `cmd/orquesta-server`.
El replan automatico por test fallido de una unica task causal ya queda cubierto
offline, tambien tras reinicio `state-file`, sin duplicar quality gate, replan
ni apertura de fase. El reinicio `state-file` cubre tambien wait expirado,
review negativa `changes_requested` con rework/replan ya registrado, y bloqueo
de prerequisitos de cierre hasta reentrada/cierre sin duplicar evidencias ni
eventos. Actualizacion 2026-05-23: `CODEX-WAVE-REAL` y
`CODEX-RECURSION-REAL` ya cerraron ola/cohorte Codex amplia y recursion con
proveedor real opt-in; no son backlog abierto salvo regresion demostrada. El
pendiente real vigente para este frente es OPES temporal real de
derivados/cierre hasta `generate_audio_asset -> audio_asset`, que pertenece a
la composicion/bridge OPES y no al nucleo del servicio.

Alcance esperado:

1. observar entregas de la cohorte/ola esperada;
2. pasar por review gate con evidencia causal;
3. consumir evidencias de tests requeridos ya persistidas y bloquear si fallan;
4. emitir u observar rework/replan cuando corresponda, con refs causales y
   prueba focal;
5. cerrar solo con review aceptada y pruebas/evidencias requeridas.

Validacion actual del tramo de tests requeridos:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service -run 'TestRequiredTest(Runner|Evidence)|TestInMemoryRequiredTestEvidence|TestUpdateOperationalDirectorPlanStateAfterLoopV0(AvanzaDeTestsAReplanConEvidenciaPassed|EjecutaRunnerDeTestsRequeridos|BloqueaTestsConEvidenciaFailed|NoAvanzaTestsConEvidenciaDeOtraReview)'
```

Validacion de continuidad wait -> review:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run TestContinueAppDirectorV0AvanzaDeWaitAReviewAbriendoRevision
```

Validacion de replays durables adicionales:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run 'TestContinueAppDirectorV0WaitExpiredStateFileRestartNoDuplicaEvidenceRefs|TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedReworkReplanNoDuplica|TestContinueRequestWithOperationalDirectorPlanStateV0StateFileRestartReintentaPrereqSinDuplicar'
```

## APP-DIR-SVC-015

Objetivo: dejar un mapa local del servicio para orientar cambios pequenos sin
mezclar entrada, waits, plan-state, review/tests, cierre y composicion de
providers.

Estado: hecho.

Cambio: `docs/mapa.md` documenta entradas, ciclo conceptual, grupos de ficheros
y reglas de ampliacion. `README.md` enlaza el indice local y
`docs/contratos.md` sincroniza la regla vigente de `replan_or_close`.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-app-director-service
```

Notas de test:

- para regresiones de scope, priorizar tests con `WaitAgentRefs` o
  `operational_director_plan_ref`; una espera vacia debe seguir significando
  modo legacy de run completo;
- para regresiones de cierre, probar primero `PlanState` en
  `replan_or_close/running`, outbox cero y refs causales completas antes de
  llamar a `OperationalClosureSource`;
- si aparece un blocker nuevo, anadir una prueba focal con la cadena causal que
  lo hace reparable o explicitar que queda bloqueado; no ampliar validaciones por
  texto libre ni mover pendientes reales cerrados de la matriz.

## APP-DIR-SVC-016

Objetivo: reflejar el resultado terminal de `goal-first` en la run durable del
core.

Estado: hecho.

Cambio: `ObserveAppDirectorGoalV0` exige `RunStore` y `EventSink` cuando el
goal llega a estado terminal. Si la closure es aceptada, abre
`validacion_final`, registra validacion run-level sin microtareas, abre
`cierre` y emite `CloseRun`. Si la closure no es aceptada, emite `BlockRun` con
blocker estable de goal-first. El servicio sigue sin conocer runtime, proveedor
ni DB; todo cruza por puertos.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-app-director-service -run 'TestObserveAppDirectorGoal|TestStartAppDirectorV0GoalFirst'
```
