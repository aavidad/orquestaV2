# Contratos: orquesta-app-director-service

## `StartAppDirectorV0`

Entrada: `StartAppDirectorRequestV0`.

- `app_spec_request`: DTO canonico de factory;
- refs opcionales: `run_ref`, `project_ref`, `correlation_id`, `requested_by`;
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
- `DirectorTaskStore`, requerido para guardar y leer microtareas creadas por el director.
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
- si hay fuente de decisiones del director, el servicio las aplica por el puente `orquesta-director-agent-workflow`;
- si esas decisiones generan microtareas y abren programacion, el servicio recompone el provider y relanza el loop para que el scheduler cree agentes de trabajo.
