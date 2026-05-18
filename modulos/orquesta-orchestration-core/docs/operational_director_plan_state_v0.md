# OperationalDirectorPlanStateV0

Este documento local describe el contrato `OperationalDirectorPlanStateV0` de
`orquesta-orchestration-core`. El primer corte ya existe en codigo; las
transiciones posteriores del ciclo operativo siguen pendientes.

## Responsabilidad

`OperationalDirectorPlanStateV0` representa el estado vivo del plan
operativo dentro de la capa de aplicacion neutral. Su trabajo es conservar una
foto compacta y reentrable del paso actual del Director Operativo:

- run y plan gobernados;
- ola/cohorte/parent task activa;
- step actual y status;
- tasks del scope activo;
- wait state enlazado;
- entregas, reviews y evidencias ya observadas;
- blockers e intentos de replan;
- razon compacta de cierre o bloqueo.

No ejecuta runtime, no decide proveedor/modelo, no valida OPES ni lee DB real.
Debe trabajar por puertos y refs opacas.

## Invariantes locales

- El scope se deriva de `wave_ref`, `cohort_ref` o `parent_task_ref`; no de todos
  los agentes del run.
- Las task refs deben existir en `WorkflowTaskStore` y seguir autorizadas por el
  run.
- El state referencia `WorkflowTaskWaitStateV0` cuando el paso activo es espera,
  pero no duplica toda la foto de espera.
- Las evidencias de tests son refs a `RequiredTestEvidenceV0`; el state no
  acepta summaries como tests ejecutados.
- Cada transicion debe tener causa: task, delivery, review, evidencia, replan,
  outbox o blocker.
- Las claves idempotentes deben ser estables por `run_ref`, `plan_ref`, step y
  refs causales, no por timestamps.
- Reentrada y replay no pueden duplicar review, tests, rework, replan ni cierre.
- Un cierre exige scope `quiescent`, outbox cero, reviews aceptadas,
  evidencias/validaciones requeridas y ninguna task abierta dentro del scope.
- El contrato no importa Codex, OPES, runtime concreto, DB, web, MCP, HOME,
  OAuth, tokens ni rutas locales.

## Frontera con stores existentes

`WorkflowTaskStore` conserva la metadata completa de tasks y linaje. El plan
state guarda `scoped_task_refs` y resumen compacto.

`WorkflowTaskWaitStateV0` conserva causa, agentes objetivo y pendientes. El plan
state guarda `wait_state_ref` y blockers derivados.

`RequiredTestEvidenceV0` conserva resultados durables de tests requeridos. El
plan state guarda refs aceptadas o blockers de ausencia/fallo.

El historial de eventos y el outbox conservan causalidad. El plan state debe
apuntar a esas refs, no convertirse en un segundo log paralelo.

## Estado implementado

1. DTO versionado, normalizacion y validacion en core.
2. Puertos de escritura y lectura/store.
3. Store en memoria para pruebas.
4. Persistencia file-based en `orquesta-state-file`.
5. Escritura inicial desde `ContinueAppDirectorV0` tras materializar
   `launch_subagents`: step activo `wait_subagents`, ola/cohorte, task refs,
   agent refs, pending agent refs, `parent_task_ref` si aplica y `wait_ref`.
6. Lectura de reentrada inicial desde `ContinueAppDirectorV0` usando
   `operational_director_plan_ref`.
7. Avance inicial `wait_subagents -> review_deliveries` cuando el wait queda
   consumido y los agentes pendientes entregaron.
8. Avance positivo `review_deliveries -> run_required_tests/replan_or_close`
   cuando las entregas del scope activo tienen cadena causal
   `DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(accepted) ->
   ReviewAccepted`. El step de review guarda `delivery_refs`,
   `review_result_refs` y `accepted_review_refs`.
9. Consumo durable de `RequiredTestEvidenceV0` desde `run_required_tests` en
   `app-director-service`: evidencias `passed` causales activan
   `replan_or_close`; evidencias `failed` bloquean con
   `required-tests-failed`.
10. Observacion negativa de `review_deliveries` cuando el historial causal trae
    `ReworkRequested` y `ReplanDecisionRecorded`. El state guarda refs de
    rework/replan, marca el step como `changes_requested` e incrementa attempts.
11. Cierre/bloqueo posterior desde `app-director-service`: si el cierre
    operacional tiene exito, el state queda `closed`; si el cerrador devuelve
    issues o el source no puede construir request, queda `blocked` con
    `closure_reason`.

Validacion focal:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service -run 'Test.*OperationalDirectorPlanState|Test.*PlanState|TestContinueAppDirectorV0.*PlanState|TestContinueRequestWithOperationalDirectorPlanStateV0|TestUpdateOperationalDirectorPlanStateAfterLoopV0'
```

## Pendiente de implementacion restante

1. Materializar runner/adaptador real para generar `RequiredTestEvidenceV0`;
   el consumo durable del step `run_required_tests` ya esta cubierto desde
   `app-director-service`.
2. Materializar runner/adaptador real y replan automatico para blockers
   negativos.
3. Agregar pruebas offline de replay/idempotencia, test faltante, blocker causal
   y cierre solo con evidencias completas.

Hasta que esos puntos existan con pruebas, `DIRECTOR-PLAN-STATE-OFFLINE` queda
parcialmente cerrado: contrato, stores, escritura inicial, reentrada, review
positiva por scope, consumo de tests durables, observacion negativa y
cierre/bloqueo del state existen; runner real, replan automatico de blockers y
replay/idempotencia siguen pendientes.
