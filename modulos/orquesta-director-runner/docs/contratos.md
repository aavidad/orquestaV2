# Contratos: orquesta-director-runner

## RunDirectorCycleV0

Entrada canonica: `DirectorCycleInputV0`.

Campos obligatorios:

- `cycle_ref`: identificador opaco del ciclo.
- `run_ref`: run objetivo.
- `scheduler_input`: tick preparado por otro modulo.
- `scheduler`: puerto inyectado que produce `DirectorSchedulerTickPlanV0`.
- `workflow`: puerto inyectado que aplica `OrchestrationCommandV0`.

Invariantes:

- `scheduler_input.run_ref` debe coincidir con `run_ref`.
- El runner no construye candidatos ni snapshots.
- Antes de invocar el scheduler descarta `progress_supervision_candidates` que no
  tengan `command_meta.run_id` y `report.run_id` iguales al `run_ref`.
- El runner no despacha outbox.
- `waiting` y `quiescent` no aplican comandos.
- `commands_ready`, `blocked` y `needs_director` pueden traer comandos durables previos y se aplican en orden.
- Si un comando produce outbox, el ciclo se detiene y devuelve `outbox_pending`.
- Por defecto `max_outbox=1`, asi que el comportamiento historico sigue siendo
  parar en el primer outbox.
- Si `max_outbox>1`, el runner puede acumular varios outbox del mismo plan antes
  de devolver `outbox_pending`; no los persiste ni despacha.
- Si un comando falla, no se aplican comandos posteriores.
- `max_commands` limita el lote; por defecto es pequeno.
- `max_outbox` limita el lote de outbox; por defecto es 1.

Puertos:

- `DirectorSchedulerPortV0`: adaptador hexagonal del scheduler.
- `WorkflowCommandPortV0`: adaptador hexagonal del workflow/event-store.

Errores publicos:

- `director_runner_cycle_invalido`: entrada, contexto o plan invalido.
- `director_runner_scheduler`: fallo del puerto scheduler.
- `director_runner_workflow`: fallo del puerto workflow.

Estados de salida:

- `quiescent`: no hay trabajo listo.
- `waiting`: existe espera contractual.
- `blocked`: hay bloqueo contractual.
- `needs_director`: hay consulta o decision humana/director.
- `commands_applied`: comandos aplicados sin outbox.
- `outbox_pending`: hay outbox que debe tratar otro conector.

Regla de estados con comandos:

- Si `blocked` trae comandos, se aplican y la salida final sigue siendo `blocked` salvo que haya outbox.
- Si `needs_director` trae comandos, se aplican y la salida final sigue siendo `needs_director` salvo que haya outbox.
- Esta regla evita repetir indefinidamente comandos de registro previo, como expiraciones de lease, antes de pedir decision.

## No Contratos

No forman parte de este modulo:

- conectores de DB;
- conectores de runtime;
- seleccion de modelos;
- OAuth/HOME/cuotas;
- preparacion de contexto de agentes;
- despacho de outbox.
