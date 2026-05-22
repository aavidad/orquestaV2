# Tareas: orquesta-orchestration-core

## ORCH-CORE-DIR-001: materializar launch_subagents

Estado: hecho.

`OperationalDirectorPlanMaterializerV0` toma un plan listo del Director
Operativo, selecciona items `launch_subagents`, guarda `WorkflowTaskV0` y emite
`CreateMicrotask`.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core -run TestOperationalDirectorPlanMaterializer
```

## ORCH-CORE-DIR-002: derivar waits por metadata de task

Estado: hecho como helper.

`WorkflowTaskWaitAgentRefsV0` carga tasks abiertas del run, filtra por
`cohort_ref`, `wave_ref` o `parent_task_ref` y devuelve refs de agentes
concretos.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core -run Test.*WaitAgentRefs
```

## ORCH-CORE-DIR-003: cerrar primer tramo durable del Director

Estado: hecho.

Objetivo:

1. materializar una ola `launch_subagents`;
2. persistir tasks con metadata de ola/cohorte;
3. derivar `WaitAgentRefs` desde esa metadata;
4. reentrar al loop progresivo con esos refs.

La salida esperada es `wait_external` o `quiescent` con refs acotadas a la ola,
no al run completo.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service -run 'Test.*OperationalDirector|Test.*WorkflowTaskWait|TestContinueAppDirectorV0MaterializaOperationalDirectorPlanYEsperaOla'
```

## ORCH-CORE-DIR-004: representar espera como estado con causa

Estado: hecho primer corte.

Objetivo: registrar causa de espera: cohorte en progreso, agentes pendientes,
review pendiente, timeout, presupuesto, bloqueo externo o falta de contexto.

Implementacion: `WorkflowTaskWaitStateV0` y
`BuildWorkflowTaskWaitSnapshotV0`. El estado conserva run, wait ref, filtro,
task refs, agent refs, pending agent refs, causa, intento, limites y evidencia.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core -run Test.*WorkflowTaskWait
```

## ORCH-CORE-DIR-005: materializar review/rework/replan del Director Operativo

Estado: pendiente parcial.

Objetivo: conectar pasos `review_deliveries`, `run_required_tests` y
`replan_or_close` con comandos existentes del workflow:

- `RecordReviewResult`;
- `AcceptReview`;
- `RequestRework`;
- `RecordReplanDecision`;
- `CloseTask`.

Debe conservar causalidad entre delivery, review, rework/replan y cierre.

Nota del corte 2026-05-17: el provider de review/replan ya soporta
`split_task` y guarda las nuevas `WorkflowTaskV0` si recibe un
`WorkflowTaskWriterPortV0`. La composicion de `app-director-service` ya le pasa
`DirectorTaskStore`; lo pendiente es convertirlo en tramo durable del Director
Operativo, no solo en candidate provider disponible.

## ORCH-CORE-DIR-006: evidencia durable minima de tests requeridos

Estado: hecho como contrato/store y guarda de cierre; pendiente como runner.

`RequiredTestEvidenceV0` representa un resultado durable de test requerido:
`run_ref`, `task_ref`, `test_command`, status `passed`/`failed`, delivery,
review request, review result, accepted review y refs adicionales de evidencia.
Debe traer `evidence_refs` con artefacto/salida real. `RequiredTestEvidenceReaderPortV0`
y `RequiredTestEvidenceWriterPortV0` separan consumo y escritura; el store completo
implementa ambas interfaces.

`OperationalDirectorClosureV0` lo consume al cerrar una task con
`WorkflowTaskV0.RequiredTests`: exige una evidencia `passed` por cada comando
declarado y que todas las refs causales coincidan con la entrega/review que se
esta cerrando. La `evidence_ref` debe estar solicitada por el cierre. Si falta
store, falta ref, el status es `failed` o la evidencia pertenece a otra
task/entrega/review, no cierra. El cierre tambien exige `EventSink` para no
mutar runs sin evento persistido.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file -run 'Test.*RequiredTestEvidence|TestOperationalDirectorClosureV0'
```

## ORCH-CORE-DIR-007: runner/adaptador de run_required_tests

Estado: parcial avanzado. El contrato, store, cierre, consumo desde `PlanState`
y runner puro por puerto ya existen. El runner recibe tests requeridos con refs
causales de entrega/review, ejecuta por `RequiredTestCommandExecutorPortV0` y
guarda `RequiredTestEvidenceV0` idempotente por
`RequiredTestEvidenceWriterPortV0`. Existe adaptador externo opt-in en
`modulos/orquesta-runtime-required-test`.

Objetivo: convertir el paso operativo `run_required_tests` en ejecucion o
validacion por puerto y guardar `RequiredTestEvidenceV0` idempotente. El
servicio ya puede invocar el runner inyectado si no encuentra evidencias
causales ya persistidas, reevalua el `PlanState` con las refs generadas y
bloquea si una evidencia causal llega como `failed`.

Pendiente: prueba real de programacion con ola/cohorte amplia o recursion
Codex usando el runner activado. El replan negativo automatico por test fallido
de una unica task causal ya queda cerrado offline con replay `state-file`; si
aparece un blocker distinto del runner, debe entrar como caso nuevo con prueba
propia.

No basta con que el agente escriba un summary ni con que el review result
mencione un comando. El cierre solo debe consumir evidencias durables guardadas
por `RequiredTestEvidenceWriterPortV0` y luego leidas por
`RequiredTestEvidenceReaderPortV0`.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-app-director-service -run 'TestRequiredTest(Runner|Evidence)|TestInMemoryRequiredTestEvidence|TestUpdateOperationalDirectorPlanStateAfterLoopV0(AvanzaDeTestsAReplanConEvidenciaPassed|EjecutaRunnerDeTestsRequeridos|BloqueaTestsConEvidenciaFailed|NoAvanzaTestsConEvidenciaDeOtraReview)'
```

## ORCH-CORE-DIR-008: estado vivo del plan operativo

Estado: cerrado offline para el ciclo probado.

Objetivo: introducir `OperationalDirectorPlanStateV0` como foto durable y
reentrable del avance del plan. Debe registrar run, plan, step activo,
ola/cohorte o parent task activo, scope de tasks, wait state, entregas, reviews,
evidencias, blockers, intentos de replan y razon de cierre o bloqueo.

Invariantes:

- el scope sale de `wave_ref`, `cohort_ref` o `parent_task_ref`, no de todos los
  agentes vivos del run;
- el state referencia `WorkflowTaskStore`, `WorkflowTaskWaitStateV0`,
  `RequiredTestEvidenceV0`, eventos y outbox; no duplica esas verdades;
- cada transicion tiene refs causales e idempotencia estable;
- si falta entrega, review, test requerido, outbox cero, contexto o cierre de
  tasks, el blocker queda explicito;
- no hay Codex, OPES, DB concreta, runtime real ni rutas locales en el contrato.

Referencia:

```text
docs/corte_operational_director_plan_state_v0_2026-05-17.md
modulos/orquesta-orchestration-core/docs/operational_director_plan_state_v0.md
```

Implementado:

- contrato, normalizacion y validacion en core;
- store en memoria;
- store file-based en `orquesta-state-file`;
- wiring en `app-director-service`, stack Codex y servidor;
- escritura inicial tras materializar `launch_subagents`, con step activo
  `wait_subagents`, ola/cohorte, task refs, agent refs, pending agent refs y
  `wait_ref`.
- lectura inicial desde `ContinueAppDirectorV0` usando
  `operational_director_plan_ref`.
- avance de `wait_subagents` a `review_deliveries` cuando el wait queda
  consumido.
- avance por review positiva, tests requeridos, evidencia faltante/fallida,
  replan causal, followups tardios y cierre/bloqueo;
- replay con `state-file` para cierre, bloqueo de cierre y replan por tests
  fallidos sin duplicar eventos ni refs.

Pendiente operativo:

- reproducir el mismo ciclo con Codex real de ola/cohorte amplia y recursion;
- cerrar derivados OPES reales solo desde conector/adaptador de dominio.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service -run 'Test.*OperationalDirectorPlanState|Test.*PlanState|TestContinueAppDirectorV0.*PlanState|TestContinueRequestWithOperationalDirectorPlanStateV0'
```

## ORCH-CORE-DIR-009: perfiles neutrales de WorkflowTask

Estado: implementado.

Objetivo: quitar el hardcode directo de rol/capacidad en
`WorkflowTaskCandidateProviderV0` y reutilizar `WorkProfileV0`/
`WorkflowTaskV0.work_profile_kind` para programacion, estudio, refactor,
pruebas, documentacion, revision y trabajo de dominio.

Invariantes:

- el resolver vive por puerto neutral `WorkflowTaskProfileResolverPortV0`;
- el resolver por defecto mantiene compatibilidad para tareas legacy sin perfil;
- no parsea `summary`, `acceptance_criteria` ni nombres de agentes como fuente
  primaria;
- no conoce Codex, OPES, modelos, procesos reales, HOME ni credenciales;
- una composicion puede inyectar politica propia sin cambiar el scheduler.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-orchestration-core -run 'TestWorkflowTaskCandidateProviderV0.*Profile|TestWorkflowTaskFromWorkProfileV0|TestValidateWorkflowTaskV0RejectsUnknownWorkProfileKind'
```
