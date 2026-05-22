# Corte OperationalDirectorPlanStateV0 - 2026-05-17

Este documento registra el primer corte implementado de
`OperationalDirectorPlanStateV0` y su reconciliacion posterior. Su objetivo es
fijar que representa el estado vivo del plan, que invariantes debe mantener y
que parte quedo cerrada offline/fake tras los cortes de replay durable.

## Por que existe

`OperationalDirectorPlanV0` describe la intencion inicial del Director
Operativo: olas, cohortes, pasos, dependencias, tests requeridos, write-set y
criterios. Ese plan inicial no basta para reentrada despues de esperar agentes,
revisar entregas, ejecutar tests, pedir rework o cerrar una ola.

`OperationalDirectorPlanStateV0` es la foto durable y reentrable del avance
operativo del plan. No reemplaza al plan, al `WorkflowTaskStore`, al
`WorkflowTaskWaitStateV0`, al outbox ni al historial de eventos. Los enlaza por
refs para que `ContinueAppDirectorV0` y los loops del nucleo sepan que paso
operativo esta activo, por que esta bloqueado y que evidencia causal ya cuenta.

## Que es

El estado objetivo debe responder, por refs compactas, estas preguntas:

- que `run_ref` y `plan_ref` gobierna;
- que `wave_ref`, `cohort_ref` o `parent_task_ref` esta activa;
- que paso del Director esta activo: `launch_subagents`, `wait_subagents`,
  `review_deliveries`, `run_required_tests`, `replan_or_close`, `close` o
  `govern_delegation`;
- que tasks de `WorkflowTaskStore` pertenecen al scope activo;
- que agentes, entregas, reviews, evidencias de tests, reworks y replans ya
  fueron observados para ese scope;
- que blockers impiden avanzar: entregas pendientes, review faltante, test
  faltante/fallido, outbox pendiente, presupuesto, falta de contexto, timeout o
  dependencia abierta;
- que intento/replan esta en curso y cual es la razon compacta de cierre o
  bloqueo.

Debe ser un contrato neutral de orquestacion. No contiene prompts, modelo,
proveedor, rutas locales, HOME, OAuth, detalles OPES, SQL ni filesystem de
producto.

## Frontera con piezas existentes

- `OperationalDirectorPlanV0`: sigue siendo la intencion declarada. El state no
  reescribe el plan ni inventa pasos no declarados.
- `WorkflowTaskStore`: conserva la metadata completa de tasks, linaje, ola,
  cohorte, parent/child refs, tests y write-set. El state referencia tasks, no
  duplica toda su definicion.
- `WorkflowTaskWaitStateV0`: conserva la foto durable de espera por agentes. El
  plan state puede apuntar al wait activo y registrar que el paso actual sigue
  bloqueado por espera.
- `RequiredTestEvidenceV0`: conserva resultados durables de tests requeridos.
  El plan state solo referencia evidencias vistas/aceptadas o blockers por
  ausencia/fallo.
- `OperationalDirectorClosureV0`: valida causalidad de cierre. El plan state no
  sustituye esa validacion; solo evita que la reentrada dependa de stats bonitas
  o del plan inicial.
- Outbox/eventos del workflow: siguen siendo la frontera causal hacia efectos y
  la fuente de replay. Cada avance del state debe tener causa e idempotencia.

## Invariantes

- Reentrable: una continuacion debe poder reconstruir el avance desde run,
  `WorkflowTaskStore`, `WorkflowTaskWaitStateV0`, eventos/outbox y plan state,
  sin mirar todos los agentes vivos del run.
- Scope estricto: todo avance se calcula contra `wave_ref`, `cohort_ref` o
  `parent_task_ref` activo. Entregas, ACKs o reviews fuera de scope no avanzan
  ni cierran la ola.
- Causalidad: cada cambio referencia las tasks, delivery refs, review refs,
  evidencias y comandos/outbox que lo justifican. No hay avance por summary
  textual.
- Idempotencia: cada transicion tiene clave estable por `run_ref`, `plan_ref`,
  `step_ref` y refs causales. Replay no duplica reviews, tests, reworks ni
  cierres.
- Monotonicidad controlada: un paso aceptado no vuelve a `pending` salvo por
  replan causal que abra un nuevo intento o una nueva ola. El intento anterior
  queda trazable.
- Bloqueo explicito: si falta entrega, review, test requerido, outbox cero,
  contexto o cierre de tasks, el state registra blocker; no queda simplemente en
  "no cerrar".
- Modo respetado: `programming` exige `RequiredTestEvidenceV0` para tests
  declarados; `domain_work` usa artefactos/validadores de dominio por puerto y
  no inventa tests de programacion.
- Frontera neutral: el contrato no importa ni serializa Codex, OPES, DB
  concreta, HTTP, web, MCP, proveedor, modelo, HOME ni rutas locales.
- No sustituye stores fuente: el state puede cachear refs y resumen compacto,
  pero la verdad de tasks, wait, evidencias y eventos sigue en sus stores
  respectivos.
- No cierre prematuro: `close` solo puede quedar aceptado si el scope esta
  quiescent, outbox cero, sin tasks abiertas en scope, con reviews aceptadas y
  evidencias/validaciones requeridas.

## Forma implementada inicial

El corte de codigo introduce un DTO versionado en
`modulos/orquesta-orchestration-core/operational_director_plan_state_v0.go` con
refs compactas:

```text
OperationalDirectorPlanStateV0
  schema_version
  state_ref
  run_ref
  plan_ref
  request_ref
  project_ref
  mode
  status
  active_step_id
  active_wave_ref
  active_cohort_ref
  steps[]
  pending_agent_refs
  blocker_refs
  evidence_refs
  required_test_refs
  replan_attempts
  closure_reason
  correlation_id
  observed_at
  updated_at
```

Cada step conserva `step_id`, `kind`, `status`, `wave_ref`, `cohort_ref`,
`task_refs`, `wait_refs`, `agent_refs`, `pending_agent_refs`, refs de delivery,
review, evidencias y blockers. El store en memoria vive en core; el store
file-based vive en `orquesta-state-file` bajo
`operational_director_plan_states/`.

## Transiciones minimas

1. `launch_subagents` materializado: registra ola/cohorte activa, tasks
   materializadas y siguiente paso esperado.
2. `wait_subagents` activo: enlaza `WorkflowTaskWaitStateV0` y blockers por
   agentes pendientes.
3. `review_deliveries`: declara entregas esperadas del scope y registra review
   aceptada o rework causal.
4. `run_required_tests`: enlaza evidencias `RequiredTestEvidenceV0` por comando
   requerido o blocker por ausencia/fallo.
5. `replan_or_close`: decide nueva ola/rework o cierre con causa compacta.
6. `close`: marca ola/task/run cerrable solo si las guardas de cierre causal ya
   pasaron.

## Implementado en este corte

- DTO, normalizacion y validacion de `OperationalDirectorPlanStateV0` en
  `orquesta-orchestration-core`.
- Puertos separados de escritura y lectura/store:
  `OperationalDirectorPlanStateWriterPortV0` y
  `OperationalDirectorPlanStateStorePortV0`.
- Store en memoria para tests y store file-based mutable en `orquesta-state-file`.
- Wiring en `app-director-service`, `orquesta-app-codex-stack` y
  `cmd/orquesta-server`.
- `ContinueAppDirectorV0` guarda el estado inicial tras materializar
  `launch_subagents`: step activo `wait_subagents`, ola/cohorte activa, task
  refs, agent refs, pending agent refs, `parent_task_ref` si aplica y
  `wait_ref` determinista.
- `ContinueAppDirectorV0` puede reentrar desde `operational_director_plan_ref`
  leyendo el plan state y proyectando scope de wait sin mirar agentes vivos
  globales.
- Cuando el wait queda consumido y el loop esta `quiescent`, el servicio avanza
  el state de `wait_subagents` a `review_deliveries` si los agentes pendientes
  entregaron.
- Cuando `review_deliveries` esta activo, el servicio lee `RunEventReaderPortV0`
  y avanza a `run_required_tests` o `replan_or_close` solo si todas las tasks
  del scope tienen cadena causal
  `DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(accepted) ->
  ReviewAccepted`. El step guarda `delivery_refs`, `review_result_refs` y
  `accepted_review_refs`.
- `maybeCloseOperationalDirectorV0` no invoca cierre mientras el plan state
  este activo en `review_deliveries` o `run_required_tests`.
- El drain del stack puede propagar `operational_director_plan_ref` hacia
  `ContinueAppDirectorV0`.
- Pruebas focales de core, state-file, servicio, stack y servidor.

## Estado tras cortes posteriores

`DIRECTOR-PLAN-STATE-OFFLINE` queda cerrado para el ciclo probado
offline/fake-runtime:

- `StartAppDirectorV0` puede sembrar el plan state inicial desde un
  `OperationalDirectorPlanV0` listo si hay contratos funcionales explicitos y
  stores inyectados.
- `ContinueAppDirectorV0` recupera plan state y wait state persistidos sin
  reconstruir desde agentes vivos globales.
- `wait_subagents` pasa a `review_deliveries`; el wait state se marca
  `continued`, `expired` o `cleared` segun causa.
- `review_deliveries` avanza por cadena causal aceptada, registra review
  negativa durable y reabre waits solo para followups causales ya reflejados.
- `run_required_tests` consume o genera `RequiredTestEvidenceV0` por runner
  inyectado, bloquea evidencia faltante/fallida y reentra cuando aparece
  evidencia causal.
- `replan_or_close` gobierna el cierre; outbox pendiente, source/task store
  ausentes o cierre insuficiente bloquean con causa durable y reintentan cuando
  el prerequisito vuelve.
- El replay con `state-file` cubre cierre exitoso, bloqueo de cierre, replan por
  tests fallidos y reentrada sobre plan cerrado sin duplicar eventos, evidencias
  ni refs.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'Test.*OperationalDirectorPlanState|Test.*PlanState|TestStartAppDirectorV0OperationalDirectorPlan|TestContinueAppDirectorV0.*PlanState|TestContinueAppDirectorV0CierraCicloReviewRunnerYReplanOrClose|TestContinueRequestWithOperationalDirectorPlanStateV0|TestUpdateOperationalDirectorPlanStateAfterLoopV0|TestMaybeCloseOperationalDirectorV0|TestOperationalPlanStateStoreV0|TestWaitStateStoreV0|TestStoreV0RecuperaEstadoTrasRecrearInstancia|TestBuildStackFromEnvV0UsaConectoresDurablesFileBased|TestBuildStackV0RequiresOptInAndExplicitPorts|TestBuildStackFromEnvV0RequiredTestRunnerEjecutaGoTestYPersisteEvidencia'
```

## Pendiente verificable restante

- Repetir la garantia con Codex real de ola/cohorte amplia (`CODEX-WAVE-REAL`).
- Repetir la recursion con proveedor Codex real: hijos/nietos, ACK/entregas
  vivas, review causal y cierre del arbol.
- Cerrar OPES temporal real de derivados/cierre hasta `assemble_topic`.
- Documentar como pendiente cada composicion que no tenga fuente real de
  validacion/cierre o persistencia/lectura del plan state.

## No hacer en este corte

- No reabrir P1 `WaitAgentRefs` salvo regresion demostrada.
- No meter Codex, OPES, DB real, web, MCP, rutas locales ni proveedor en el
  contrato.
- No reconstruir el plan state desde summaries, stats agregadas o agentes vivos
  globales.
- No aceptar `OperationalDirectorPlanV0` inicial como sustituto del estado vivo
  despues de materializar o reentrar.
- No extrapolar `DIRECTOR-PLAN-STATE-OFFLINE` a Codex real amplio, recursion real
  u OPES real sin smoke opt-in y evidencia guardada.
