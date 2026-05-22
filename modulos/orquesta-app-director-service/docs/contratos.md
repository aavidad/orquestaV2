# Contratos: orquesta-app-director-service

## `StartAppDirectorV0`

Entrada: `StartAppDirectorRequestV0`.

- `app_spec_request`: DTO canonico de factory;
- refs opcionales: `run_ref`, `project_ref`, `correlation_id`, `requested_by`;
- espera opcional: `wait_agent_refs` explicitas o filtros
  `wait_cohort_ref`, `wait_wave_ref`, `wait_parent_task_ref`;
- plan operativo opcional: refs de contratos de funcion, fase objetivo y limite
  de items; en el primer corte la materializacion automatica del plan se usa en
  `ContinueAppDirectorV0`;
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
- `max_decision_cycles`, limite acotado para consumir decisiones y volver a ejecutar el loop sin quedar en bucle.

Salida: `StartAppDirectorResultV0`.

- `app_spec`;
- `run`;
- `director_task`;
- `director_tasks`;
- `loop_status`;
- `started_agents`;
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
  pending agent refs y `wait_ref`.

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
los pending refs ya se limpiaron al consumir el wait. `replan_or_close` y otros
steps siguen fallando de forma explicita hasta tener handler causal propio, para
no volver al scope legacy de todo el run.

Regla de avance del plan state tras el loop: si el loop queda `quiescent`, el
state pasa de `wait_subagents` a `review_deliveries` cuando todos los agentes
pendientes entregaron. Si el step activo es `review_deliveries`, outbox esta a
cero y el historial de eventos trae la cadena causal del scope activo
`DeliveryRegistered -> ReviewRequested -> ReviewResultRecorded(accepted) ->
ReviewAccepted`, el state marca review como aceptada, guarda `delivery_refs`,
`review_result_refs` y `accepted_review_refs`, y activa `run_required_tests` en
modo `programming` con tests requeridos o `replan_or_close` en el resto.
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

Regla del primer corte operativo: si `ContinueAppDirectorV0` recibe un plan
`ready`, usa `OperationalDirectorPlanMaterializerV0`, guarda las
`WorkflowTaskV0`, emite `CreateMicrotask`, deriva la primera ola/cohorte como
wait acotado, persiste `OperationalDirectorPlanStateV0` si la composicion
inyecta writer, y entra al loop progresivo con esas refs. `StartAppDirectorV0`
mantiene el bootstrap normal y no crea microtareas de producto por si mismo.
