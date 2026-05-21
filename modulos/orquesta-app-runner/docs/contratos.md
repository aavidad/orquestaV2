# Contratos: orquesta-app-runner

## `PrepareAppOrchestrationV0`

Entrada: `PrepareAppOrchestrationRequestV0`

- `app_spec`: `AppSpecV0` validada por factory;
- `run_ref`: opcional; si falta se deriva de `spec_id`;
- `project_ref`: opcional; si falta se deriva del slug/spec;
- `occurred_at`: requerido o tomado de `app_spec.created_at`;
- `correlation_id`, `requested_by`: refs compactas.

Salida: `AppOrchestrationPreparedV0`

- `run`: `OrchestrationRunV0` activo con fase `programacion`;
- `plan`: `AppMicrotaskPlanV0`;
- `initial_progress`: total, listas, bloqueadas, pendientes y completado;
- `candidate_provider`: provider para scheduler;
- `evidence_refs`: refs compactas.

Invariantes:

- usa comandos/eventos del workflow para crear el run;
- antes de abrir `programacion`, crea decision, contrato funcional y una
  microtarea durable por unidad del plan;
- las microtareas se construyen reutilizando `WorkflowTaskForUnitV0` del
  planner y conservan `work_profile_kind`, tests requeridos y dependencias
  causales; el runner solo reinyecta su contrato funcional global publicado;
- no construye estado a mano salvo por el reducer publico;
- no decide runtime, proveedor, modelo, HOME, credenciales ni DB;
- si la AppSpec escala a `large`, conserva el plan largo del planner y emite
  olas por dependencias hasta completar bootstrap, arquitectura, implementacion,
  integracion, documentacion y revision;
- REST/MCP/web no tienen que conocer olas, agentes ni scheduler.

Errores:

- si el planner rechaza la AppSpec, `AppRunnerIssueV0.Field` conserva el campo
  canonico cuando existe, por ejemplo `app_spec.validation.estado`;
- los adaptadores MCP/REST pueden devolver errores publicos corregibles sin
  inspeccionar el error interno del planner.

## `RunPreparedAppOrchestrationV0`

Entrada: `RunPreparedAppOrchestrationRequestV0`

- `prepared`: salida de `PrepareAppOrchestrationV0`;
- `occurred_at`: requerido para comandos del workflow;
- `use_autonomous_director_loop`: opcional; cuando es `true`, el runner delega
  el ciclo en `RunAutonomousDirectorLoopV0`;
- limites de bursts, pasos, outbox y esperas externas.

Puertos: `RunStore`, `EventSink`, `OutboxLedger`, dispatchers, y fuentes
opcionales de delivery, revision, progreso, lease, replan, espera externa y
politica autonoma de director.

Salida: `AppOrchestrationRunResultV0`

- `run`: estado final observado tras el ciclo;
- `progress`: progreso del plan calculado desde `deliveries`;
- `loop_status`: estado del loop del nucleo;
- `started_agents`, `attempts`, `external_waits`, `evidence_refs`.
- `director_loop_stats`: estadisticas opcionales del loop autonomo cuando se
  uso `RunAutonomousDirectorLoopV0`.

Invariantes:

- no arranca nada por si mismo; todo efecto sale por dispatchers inyectados;
- no sobreescribe un run ya persistido si el store lo devuelve;
- si hay `ExternalWaiter`, reintenta tras progreso externo hasta el limite;
- el loop autonomo es opt-in para no cambiar presupuestos ni concurrencia por
  sorpresa;
- las entregas validas desbloquean la siguiente ola del plan.
