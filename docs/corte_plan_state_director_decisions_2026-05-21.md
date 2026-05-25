# Corte plan state para director_decisions - 2026-05-21

## Objetivo

Documentar el corte para que las `WorkflowTaskV0` creadas desde
`director_decisions` queden enlazadas con `OperationalDirectorPlanStateV0`.

El problema concreto es que hoy hay dos caminos cercanos pero no equivalentes:

- `OperationalDirectorPlanV0` listo entra por `ContinueAppDirectorV0`, se
  materializa con `OperationalDirectorPlanMaterializerV0`, guarda
  `WorkflowTaskV0` y crea el `OperationalDirectorPlanStateV0` inicial.
- `director_decisions` entra por `DirectorDecisionSource`, se aplica con
  `ApplyDirectorAgentDecisionV0` y puede crear `WorkflowTaskV0`, pero esas tasks
  no quedan necesariamente registradas como scope activo de un plan state vivo.

El corte cierra esa brecha inicial sin reabrir `WaitAgentRefs` y sin
meter detalles de Codex, OPES, filesystem, DB o runtime en el nucleo.

## Estado aplicado

Implementado en el corte del 2026-05-21:

- `orquesta-director-agent-workflow` marca las microtareas de programacion
  nacidas de decisiones del Director con
  `operational_director.task_source: director_decision`.
- `orquesta-app-director-service` detecta esas tasks desde
  `WorkflowTaskStore` y crea un `OperationalDirectorPlanStateV0` activo con
  pasos `launch_subagents`, `wait_subagents`, `review_deliveries`,
  `run_required_tests` y `replan_or_close`.
- `StartAppDirectorV0` y `ContinueAppDirectorV0` reentran con el plan state
  efectivo para acotar la espera al agente derivado de la task, sin esperar
  todos los agentes vivos del run.
- `WorkflowTaskWaitStateV0` acepta `AgentRefs` explicitos como scope durable,
  ademas de `wave_ref`, `cohort_ref` o `parent_task_ref`.
- La prueba focal
  `TestStartAppDirectorV0DecisionCreateMicrotaskRequiredTestsCreaPlanStateReentrable`
  cubre creacion de task con `RequiredTests`, persistencia del plan state y
  reentrada por `operational_director_plan_ref`.

## Direccion del corte

El enlace correcto es causal y por refs:

```text
director_decisions
  -> ApplyDirectorAgentDecisionV0
  -> WorkflowTaskV0 persistidas en WorkflowTaskStore
  -> evento/comando durable que justifica la creacion
  -> OperationalDirectorPlanStateV0 actualizado o creado
  -> wait/review/tests/replan/close por scope del plan state
```

`director_decisions` no debe convertirse en un segundo plan operativo completo.
Debe ser una fuente de decisiones del director que, cuando produce
`WorkflowTaskV0`, deja trazabilidad suficiente para que el estado vivo sepa:

- `run_ref`, `plan_ref` explicito o ref determinista de decisiones que gobierna
  la task;
- `task_ref` creada y su `agent_ref` derivable;
- `wave_ref`, `cohort_ref` o `parent_task_ref` si existen;
- paso operativo afectado: normalmente `launch_subagents` o una rama de
  `replan_or_close` que abre nueva ola;
- refs causales de evento, outbox o decision aplicada;
- tests requeridos, write-set, criterios y function contracts ya presentes en
  `WorkflowTaskV0`.

Si la decision no trae `plan_ref` pero la task esta marcada como trabajo
operativo del Director, el servicio usa un `plan_ref` determinista acotado al
run. Si tampoco hay metadata operativa suficiente, la task queda como flujo
legacy sin cierre operativo generico.

## Frontera hexagonal

La frontera queda asi:

- `orquesta-core-workflow`: valida `WorkflowTaskV0`, comandos/eventos y ausencia
  de detalles prohibidos. No sabe que existe `director_decisions.json`.
- `orquesta-director-agent-workflow`: aplica decisiones del director sobre el
  workflow y el `WorkflowTaskStore` por puertos.
- `orquesta-orchestration-core`: contiene `OperationalDirectorPlanStateV0`,
  stores/puertos neutrales, wait por scope, review/tests/cierre causal.
- `orquesta-app-director-service`: orquesta el puente entre decisiones aplicadas
  y actualizacion del plan state, siempre por `DirectorTaskStore`,
  `OperationalPlanStateStore/Writer`, eventos y outbox.
- adaptadores/composiciones: Codex, OPES, file-store, servidor, MCP/HTTP,
  runner local de tests y cualquier fuente concreta de `director_decisions`.

El corte debe vivir en la capa de aplicacion y composicion neutral donde ya se
inyectan puertos. El core puro solo debe recibir DTOs/refs/eventos genericos.

## Que NO se mete en core

No meter en `orquesta-core-workflow`, `orquesta-director-operativo` ni
`orquesta-orchestration-core` puro:

- nombre de archivo `director_decisions.json`, paths, HOME, runtime dirs o
  filesystem de Codex;
- modelo, proveedor, razonamiento, sandbox, approval policy o cuotas concretas;
- OPES, reglas editoriales OPES, REST OPES, colas OPES o ids internos de OPES;
- DB concreta, SQL productivo, DSN, migraciones, OAuth, tokens o secretos;
- lectura directa de artefactos de runtime para deducir plan state;
- cierre por summaries textuales o estadisticas agregadas;
- espera por todos los agentes vivos del run.

El nucleo puede conocer refs opacas, eventos durables, tasks, waits, reviews,
evidencias y blockers. Todo lo demas entra por adaptador.

## Invariantes esperadas

- Una `WorkflowTaskV0` nacida de `director_decisions` que declare metadata de
  Director Operativo debe aparecer en el `OperationalDirectorPlanStateV0` del
  mismo `run_ref` y `plan_ref`.
- El scope activo se calcula por `wave_ref`, `cohort_ref`,
  `parent_task_ref` o `WaitAgentRefs` explicitos; nunca por agentes vivos
  globales.
- Reaplicar la misma decision no duplica task, eventos ni pasos del state.
- Una decision sin metadata suficiente no crea plan state ambiguo.
- La reentrada por `operational_director_plan_ref` recupera el scope desde el
  state y sigue usando `WorkflowTaskStore` como verdad de metadata completa.
- `run_required_tests` sigue consumiendo `RequiredTestEvidenceV0` durable; no
  ejecuta tests por estar creando plan state.
- `close` sigue pasando por `OperationalDirectorClosureV0` y por una
  `OperationalClosureSource` inyectada cuando la composicion la tenga.

## Pruebas esperadas

Pruebas focales de este corte:

- `app-director-service`: una decision `create_microtask` con metadata
  operativa crea/actualiza `OperationalDirectorPlanStateV0` con `task_refs`,
  `agent_refs`, `wait_ref` y blockers de `wait_subagents`.
- `app-director-service`: una decision repetida se ignora como ya reflejada y
  no duplica el state.
- `orquesta-orchestration-core`: `WorkflowTaskWaitStateV0` acepta
  `AgentRefs` explicitos como scope.
- prueba de frontera:
  `TestNeutralOrchestrationPackagesDoNotImportProductAdapters` debe seguir
  verde.

Comandos minimos esperados para ese corte de codigo:

```bash
git diff --check
go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack -run 'Test.*DirectorDecision.*PlanState|Test.*OperationalDirectorPlanState|Test.*PlanState'
go test -count=1 ./...
```

## Pendiente verificable

- Cerrado el 2026-05-25: si aparece una nueva `WorkflowTaskV0` operativa de
  `director_decisions` y el `OperationalDirectorPlanStateV0` ya existe y sigue
  abierto, `app-director-service` la fusiona en el plan conservando `plan_ref`,
  task refs, tests requeridos y wait scope acotado por agent refs. La reentrada
  no duplica wait refs ni reescribe el state si la task ya estaba reflejada.
- Añadir pruebas especificas de persistencia file-store para este camino si se
  cambia el formato o el reload del `OperationalDirectorPlanStateV0`.
- Añadir wiring opt-in especifico en `orquesta-app-codex-stack` si una fuente
  concreta de `director_decisions` necesita transportar `plan_ref` desde
  supervisor/queue/MCP.
- Mantener `run_required_tests` como consumo de `RequiredTestEvidenceV0`
  durable; no ejecutar shell tests por el mero hecho de crear el plan state.
