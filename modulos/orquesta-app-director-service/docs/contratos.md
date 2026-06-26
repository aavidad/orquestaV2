# Contratos: orquesta-app-director-service

## `StartAppDirectorV0`

Entrada: `StartAppDirectorRequestV0`.

- `app_spec_request`: DTO canonico de factory;
- refs opcionales: `run_ref`, `project_ref`, `correlation_id`, `requested_by`;
- espera opcional: `wait_agent_refs` explicitas o filtros
  `wait_cohort_ref`, `wait_wave_ref`, `wait_parent_task_ref`;
- plan operativo directo opcional: si llega `operational_director_plan`,
  `StartAppDirectorV0` exige `operational_director_function_contract_refs`,
  publica causalmente voto, decision, fase y contratos, materializa
  `launch_subagents`, deriva wait de ola/cohorte y persiste `PlanState` si la
  composicion inyecta los stores; si `run_ref` viene solo dentro del plan, el
  arranque lo usa como run del intake;
- `occurred_at`: instante operacional del arranque.

Puertos requeridos:

- `RunStorePortV0`;
- `EventSinkPortV0`;
- `OutboxLedger`;
- dispatchers de outbox ya configurados por la composicion externa.

Puertos opcionales:

- observadores de entrega, progreso, leases y replan;
- `ReviewReworkReplanSource`, fuente opcional de planes compactos para convertir
  `RequestRework` durable en replan/followups sin que el servicio conozca su origen;
- `DirectorDecisionSource`, para consumir decisiones compactas emitidas por el director;
- `DirectorTaskStore`, requerido para guardar y leer microtareas creadas por el
  director y obligatorio si la request usa filtros de espera por cohorte, ola o
  parent task;
- `WaitStateWriter`, opcional para registrar `WorkflowTaskWaitStateV0` cuando se
  resuelve un wait por metadata de task;
- `OperationalPlanStateWriter`, opcional para registrar
  `OperationalDirectorPlanStateV0` cuando un plan operativo materializa su tramo
  inicial;
- `OperationalPlanStateStore`, opcional para leer `OperationalDirectorPlanStateV0`
  cuando `ContinueAppDirectorV0` reentra por `operational_director_plan_ref`;
- `EventReader`, opcional pero necesario para que el plan state pueda avanzar
  `review_deliveries` desde el historial durable de eventos;
- bundle goal-first completo, opcional para modo goal-first:
  `GoalLauncher`, `GoalObserver`, `GoalClosureValidator` y `GoalStateStore`.
  Si se inyecta parcialmente con launcher u observer, el servicio falla de
  forma explicita y no cae al loop legacy. Si se inyecta completo, persiste el
  intake/run, compila un `GoalWorkSpecV0` neutral desde `AppSpecV0`, lanza el
  goal por el puerto y no entra en el loop legacy de agentes. Para
  `/nueva-app`, el spec incluye `required_tests` ligados al write-set generado
  y `ClosurePolicy.RequireRequiredTests=true`; un goal `complete` sin
  `required_test_results` pasados queda bloqueado y pide rework;
- `max_decision_cycles`, limite acotado para consumir decisiones y volver a ejecutar el loop sin quedar en bucle.

Salida: `StartAppDirectorResultV0`.

- `app_spec`;
- `run`;
- `director_task`;
- `director_tasks`;
- `director_execution_mode`: `goal_first` cuando el arranque usa
  `GoalWorkSpecV0` por puerto inyectado, o `legacy_director_loop` cuando usa el
  loop historico de Director/agentes;
- `loop_status`;
- `started_agents`;
- `goal_ref`, `external_goal_ref`, `goal_status` y `goal_launch_receipt` cuando
  el arranque se hizo por goal-first;
- `evidence_refs`.

Invariantes:

- no hay runtime ni DB por defecto;
- no se decide proveedor/modelo/HOME/credenciales;
- REST/MCP/web no ven scheduler, outbox ni comandos internos.
- este servicio es el punto de entrada para que Orquesta piense una app: las
  superficies publicas entregan la solicitud, pero el juicio lo aportan director
  y agentes, no el formulario ni el gateway.
- si hay fuente de decisiones del director, el servicio las aplica por el puente `orquesta-director-agent-workflow`;
- si esas decisiones generan microtareas y abren programacion, el servicio recompone el provider y relanza el loop para que el scheduler cree agentes de trabajo.
- antes de ejecutar el loop, el servicio convierte `wait_cohort_ref`,
  `wait_wave_ref` o `wait_parent_task_ref` en `WaitAgentRefs` mediante
  `WorkflowTaskWaitAgentRefsV0`; el stack Codex solo consume refs de agente y
  no interpreta semantica de cohorte/ola.
- `WaitAgentRefs` explicitas se conservan y se deduplican junto con las refs
  derivadas.
- si hay `WaitStateWriter`, el servicio registra la espera como
  `WorkflowTaskWaitStateV0` con causa y pendientes.
- si hay `OperationalPlanStateWriter`, el servicio registra el plan state
  inicial con step activo `wait_subagents`, ola/cohorte, task refs, agent refs,
  pending agent refs y `wait_ref` cuando ese tramo viene de
  `ContinueAppDirectorV0`, de `StartAppDirectorV0` con plan directo o de
  decisiones ya materializadas del director.
- si hay bundle goal-first completo, Orquesta conserva la frontera neutral: el
  servicio no conoce Codex, proveedor, shell ni daemon; solo usa
  `orquesta-goal` y puertos inyectados por la composicion.

## `ContinueAppDirectorV0`

Entrada: `ContinueAppDirectorRequestV0`.

Campos principales:

- `run_ref`: run existente;
- limites de loop y decision equivalentes a start;
- `wait_agent_refs`, `wait_cohort_ref`, `wait_wave_ref` y
  `wait_parent_task_ref` para reentrar esperando solo la unidad de trabajo
  actual;
- `operational_director_plan`, function contract refs, fase objetivo y max items
  para materializar `launch_subagents` antes del loop.
- `operational_director_plan_ref`, ref compacta opcional para reentrar desde el
  estado vivo ya persistido sin reenviar el plan completo.

Regla: si se usa un filtro de espera, la composicion debe inyectar
`RunStore` y `DirectorTaskStore`; si no hay filtros, las refs explicitas no
requieren leer tasks.

Regla de reentrada por plan state: si no hay wait explicito y llega
`operational_director_plan_ref`, el servicio exige `OperationalPlanStateStore`,
carga `OperationalDirectorPlanStateV0` por `run_ref + plan_ref` y solo proyecta
los steps activos soportados. `wait_subagents` se proyecta a `wait_wave_ref`,
`wait_cohort_ref`, `wait_parent_task_ref` y refs pendientes. `review_deliveries`
y `run_required_tests` se proyectan al mismo scope usando `agent_refs`, porque
los pending refs ya se limpiaron al consumir el wait. `replan_or_close` se
maneja como puerta causal de cierre: solo consulta la fuente de cierre cuando el
step esta `running`, el loop queda `quiescent`, no hay outbox pendiente, el run
sigue activo y existen puertos de task store/fuente de cierre. Steps no
soportados siguen fallando de forma explicita para no volver al scope legacy de
todo el run.

Regla de avance del plan state tras el loop: si el loop queda `quiescent`, el
state pasa de `wait_subagents` a `review_deliveries` cuando todos los agentes
pendientes entregaron. Si el step activo es `review_deliveries`, outbox esta a
cero y el historial de eventos trae la cadena causal del scope activo
`DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(accepted) ->
ReviewAccepted`, el state marca review como aceptada, guarda `delivery_refs`,
`review_result_refs` y `accepted_review_refs`, y activa `run_required_tests` en
modo `programming` con tests requeridos o `replan_or_close` en el resto.
`run_required_tests` y `replan_or_close` conservan las mismas
`accepted_review_refs` como causalidad de cierre. Si un followup/retry aporta
una review aceptada nueva y el paso posterior es quien la porta, debe conservar
tambien `delivery_refs` y `review_result_refs`; una accepted review suelta sin
cadena causal no es valida.
Cuando un `PlanState` activo entra o reentra en `review_deliveries` y el run
todavia esta en `programacion`, el servicio abre `revision` mediante
`OpenPhase` idempotente antes del review gate. Si el cambio de state acaba de
consumir `wait_subagents`, `ContinueAppDirectorV0` ejecuta un pase acotado
adicional del loop para que el director revise entregas ya disponibles sin
esperar otra llamada manual.
Si el historial trae una review no aceptada seguida de `ReworkRequested` y
`ReplanDecisionRecorded` para la misma task/delivery/review, el codigo local del
PlanState registra la observacion negativa como `changes_requested`, guarda
`rework_request_refs` y `replan_decision_refs`, incrementa `replan_attempts` y
deja blocker `review-rework-replan-recorded`. Esta rama esta probada como
observacion durable del state y no debe confundirse con `replan_or_close`
completo ni con emision automatica de replan nuevo.
Si el step activo es `run_required_tests`, carga `RequiredTestEvidenceV0` desde
`RequiredTestEvidenceRefs` del step o, por compatibilidad del stack actual,
desde las refs de evidencia de la review aceptada; si la composicion inyecta
`RequiredTestRunner`, puede generar evidencias por puerto sin que el servicio
conozca shell/runtime concreto. Solo avanza a `replan_or_close` si cada test
requerido tiene evidencia `passed` causal para la misma task, delivery, review
request, review result y accepted review. Una evidencia `failed` bloquea el plan
con `required-tests-failed`. Para un unico task causal, el servicio emite
automaticamente `QualityGateRecorded(blocked)` y
`ReplanDecisionRecorded(retry_task)` con refs estables de capacidad/agente
futuras, sin lanzar esos followups directamente; el scheduler los convierte en
`RequestCapacity`/`RequestAgent` mediante candidates genericos. Si esos
followups ya estan materializados, el state reabre `wait_subagents` con refs
acotadas. Scopes multitarea o ambiguos siguen bloqueados de forma conservadora.
Si el step activo es `replan_or_close` y el loop queda `quiescent`, el cierre
operativo solo ejecuta comandos de cierre si el run sigue `active`. Si llega un
run no activo, el servicio no llama a la fuente de cierre ni intenta
`CloseTask`/`RegisterFinalValidation`/`CloseRun`; bloquea el plan con
`closure_reason=operational-closure-run-not-active` para que el adaptador
resuelva el bloqueo causal antes de cerrar.

Regla goal-first: si `ObserveAppDirectorGoalV0` observa un goal terminal, debe
validar la closure con el puerto inyectado. Closure aceptada refleja el cierre
del run mediante comandos del core (`OpenPhase(validacion_final)`,
`RegisterFinalValidation` run-level, `OpenPhase(cierre)`, `CloseRun`). Closure
no aceptada refleja `BlockRun` con blocker estable. Si faltan `RunStore` o
`EventSink`, el servicio devuelve error publico de puerto ausente; no inventa
persistencia, runtime ni proveedor.

Regla del primer corte operativo: si `ContinueAppDirectorV0` recibe un plan
`ready`, usa `OperationalDirectorPlanMaterializerV0`, guarda las
`WorkflowTaskV0`, emite `CreateMicrotask`, deriva la primera ola/cohorte como
wait acotado, persiste `OperationalDirectorPlanStateV0` si la composicion
inyecta writer, y entra al loop progresivo con esas refs.
`StartAppDirectorV0` mantiene el bootstrap normal cuando no recibe plan
operativo, pero el modo plan directo ya puede hacer el mismo tramo inicial
desde un run recien creado sin meter runtime ni producto dentro del servicio.

## Ejemplos de reentrada

Reentrada por plan state vivo:

```text
run_ref=run-autoprogramacion-001
operational_director_plan_ref=operational-director-plan-autoprogramacion-001
```

El servicio carga el state por `OperationalPlanStateStore`, reusa el scope del
step activo y conserva `WaitAgentRefs` acotados. Es el camino preferido cuando
el caller no necesita reenviar el plan completo.

Reentrada por cohorte/ola:

```text
run_ref=run-autoprogramacion-001
wait_wave_ref=wave-ref-implementation-001
wait_cohort_ref=cohort-ref-docs-001
```

El servicio exige `DirectorTaskStore`, deriva agentes desde `WorkflowTaskV0` y
registra `WorkflowTaskWaitStateV0` si hay writer. No convierte la espera en
"todos los agentes vivos" del run.

Reentrada por refs explicitas:

```text
run_ref=run-legacy-001
wait_agent_refs=agent-ref-a,agent-ref-b
```

Se conserva por compatibilidad y no requiere `DirectorTaskStore`, pero no aporta
metadata de ola/cohorte ni parent/child refs. Para Director Operativo usar refs
de plan, ola, cohorte o parent task cuando existan.

Errores publicos esperables:

- filtro de wait sin `DirectorTaskStore`: falta el puerto que traduce metadata de
  task a agentes concretos;
- `operational_director_plan_ref` sin `OperationalPlanStateStore`: no hay estado
  vivo reentrable;
- plan directo sin contratos funcionales explicitos: el servicio rechaza antes
  de persistir para no materializar microtareas sin causalidad;
- cierre con `PlanState` fuera de `replan_or_close/running`, outbox pendiente o
  source/task store ausente: el state se bloquea o reintenta con razon durable.
