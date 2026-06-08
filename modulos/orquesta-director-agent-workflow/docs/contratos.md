# Contratos locales

## `BuildDirectorAgentWorkflowCommandV0`

Entrada:

- `DirectorAgentDecisionV0`;
- `occurred_at`;
- refs opcionales de correlacion y solicitante.

Salida:

- `OrchestrationCommandV0` publico;
- lista de issues si la decision no es aplicable.

Comandos soportados en v0:

- `request_brainstorm` -> `RequestBrainstorm`.
- `open_phase` -> `OpenPhase`.
- `request_vote` -> `RequestVote`.
- `accept_decision` -> `AcceptDecision`.
- `publish_function_contract` -> `PublishFunctionContract`.
- `create_microtask` -> `CreateMicrotask`.
- `ask_director` -> `AskDirector`.
- `ask_user` -> `AskDirector` con `target_group=user` hasta que exista un comando publico `AskUser`.
- `request_capacity` -> `RequestCapacity`.
- `request_agent` -> `RequestAgent`.
- `request_review`, `record_review_result`, `accept_review` -> comandos publicos de revision.
- `request_rework` -> `RequestRework`.
- `record_replan_decision` -> `RecordReplanDecision`.
- `close_task` -> `CloseTask`.
- `register_final_validation` -> `RegisterFinalValidation`.
- `close_run` -> `CloseRun`.

DTOs no traducidos en v0:

- `propose_autonomous_plan_team`: no se traduce mediante
  `BuildDirectorAgentWorkflowCommandV0` porque no existe un comando unico del
  core; `ApplyDirectorAgentDecisionV0` si lo puede materializar como una serie
  causal de `CreateMicrotask`, una por `work_unit`, usando `TaskStore`.

Invariantes:

- no aplica comandos;
- no persiste;
- no lanza runtime;
- no decide proveedor, modelo, HOME ni credenciales.

## `ApplyDirectorAgentDecisionV0`

Entrada:

- decision compacta del director;
- `occurred_at`;
- `RunStore` y `EventSink` inyectados.
- `TaskStore` inyectado cuando la decision sea `create_microtask`.

Salida:

- comando aplicado;
- run actualizado;
- numero de eventos;
- issues publicos si faltan puertos o la decision no valida.

Invariantes:

- no crea stores;
- no crea runtime;
- no despacha outbox;
- aplica solo comandos publicos del workflow;
- no abre fases ni inventa decisiones: `PublishFunctionContract` requiere fase de planificacion activa y decision ya reflejada en el run.
- materializa la microtarea completa por `TaskStore` antes de guardar run/eventos; si falta, devuelve issue publico para no crear tareas imposibles de programar.
- materializa `propose_autonomous_plan_team` como microtareas ejecutables por
  `TaskStore`; un retry exacto conserva idempotencia, no duplica eventos y
  deja las tareas completas disponibles para el scheduler.
- preserva el linaje operativo neutral de `create_microtask` en
  `WorkflowTaskV0`: parent task, cohorte, ola, profundidad, fanout e hijos
  conocidos. El puente no interpreta ese linaje ni arranca runtime.
- preserva `context_refs` opacas de `create_microtask` en `WorkflowTaskV0` sin
  interpretarlas ni convertirlas en campos de producto.
- preserva `work_profile_kind` cuando el director lo declara, para que el
  scheduler resuelva rol/capacidad sin parsear texto ni conocer conectores.
- aplica `close_task` entre `accept_review` y `open_phase` hacia `validacion_final`; no cierra fase ni run por si mismo.
- el bloqueo por `request_kind` no vive aqui: lo aplica la capa de aplicacion antes de llamar al puente, para no acoplar este adaptador a politica de producto.

## `DirectorAgentDecisionSourcePortV0`

Puerto opcional para que una capa de aplicacion consuma decisiones emitidas por
un director externo.

Entrada:

- run observado;
- stats compactas opcionales construibles por `BuildDirectorAgentCompactStatsV0`;
- `occurred_at`, correlacion y solicitante.

Salida:

- lista de `DirectorAgentDecisionV0`.

Invariantes:

- el puerto no aplica comandos;
- no decide proveedor/modelo/HOME/credenciales;
- puede entregar `DirectorAgentCompactStatsV0` para evitar snapshots extensos;
- la aplicacion que lo consume sigue aplicando por `ApplyDirectorAgentDecisionV0`.

## `BuildDirectorAgentCompactStatsV0`

Entrada:

- `OrchestrationRunV0` publico observado.

Salida:

- `DirectorAgentCompactStatsV0` con schema, run, estado/fase, contadores y refs
  pendientes.

Invariantes:

- no devuelve payloads, eventos, transcripts ni datos operativos;
- no consulta stores, DB, runtime ni proveedores;
- solo resume proyecciones publicas ya compactas del run.
