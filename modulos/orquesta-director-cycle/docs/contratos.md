# Contratos: orquesta-director-cycle

## ExecuteDirectorCycleStepV0

Entrada canonica: `DirectorCycleStepInputV0`.

Campos obligatorios:

- `scheduler`: puerto scheduler del runner.
- `workflow`: puerto workflow del runner.
- `outbox_ledger`: puerto ledger del recorder.
- `cycle_ref`
- `tick_ref`
- `run_ref`
- `occurred_at`
- `run`

Campos opcionales:

- candidates explicitos de work, lease, entrega, progreso y replan;
- `work_claims` compartidos para evaluar concurrencia de la ola;
- `max_commands`;
- `correlation_id`;
- `evidence_refs`.

Salida:

- estado final del runner;
- refs de outbox pendientes antes y despues;
- comandos aplicados;
- contadores de outbox guardada;
- waiting/blocking refs compactas.

Invariantes:

- No construye candidates.
- No despacha outbox.
- No registra ACK.
- No lee receipts de agentes ni archivos de runtime.
- No elige persistencia ni runtime.
- Ejecuta un unico paso; no reintenta ni hace loops.
- Si ya hay outbox pendiente, el scheduler debe devolver espera y no aplicar comandos.
- Si el runner produce outbox, se registra antes de devolver resultado.
- La consulta de outbox pendiente se hace por `OutboxLedger` inyectado y sus
  refs se pasan a `orquesta-director-tick-input` como datos.
- Si el run trae un evento durable pero el ledger inyectado no lista la outbox
  correspondiente, solo puede recuperar esa outbox cuando el candidate externo
  trae la senal explicita de recovery. El step no infiere ni construye esa
  reparacion por su cuenta.

Errores publicos:

- `director_cycle_step_invalido`
- `director_cycle_step_outbox`
- `director_cycle_step_tick_input`
- `director_cycle_step_runner`

## ExecuteDirectorCycleStepsV0

Entrada canonica: `DirectorCycleStepsInputV0`.

Campos obligatorios:

- `initial_step`: primer `DirectorCycleStepInputV0` completo.
- `max_steps`: presupuesto externo explicito, entre 1 y 50.

Campos opcionales:

- `snapshot_port`: puerto para cargar el siguiente snapshot del run cuando el
  ultimo paso queda en `commands_applied` y aun hay presupuesto;
- `correlation_id`;
- `evidence_refs`.

Salida:

- pasos ejecutados;
- ultimo resultado de step;
- historial compacto de resultados;
- status final;
- `stop_reason`: `wait_outbox`, `wait_external`, `blocked`,
  `needs_director`, `stop_quiescent`, `stop_max_steps` o `stop_error`;
- refs compactas de outbox, espera y bloqueo.

Invariantes:

- No cambia la semantica de `ExecuteDirectorCycleStepV0`: cada llamada al step
  sigue ejecutando un unico tick.
- No construye candidates, no despacha outbox y no registra ACK.
- No usa daemon, goroutines, sleeps, filesystem productivo ni runtime real.
- Solo repite cuando el step anterior queda en `commands_applied`, sin outbox
  pendiente y con presupuesto disponible.
- Para ejecutar un segundo step necesita `snapshot_port`; el snapshot debe traer
  `run`, `cycle_ref`, `tick_ref` y `occurred_at` actualizados.
- Si aparece outbox pendiente, waiting, blocked, needs_director, quiescent,
  error o `max_steps`, el coordinador devuelve el corte sin relanzar trabajo.

Errores publicos:

- `director_cycle_steps_invalido`
- `director_cycle_steps_snapshot`
- `director_cycle_steps_step`

## No Contratos

No forman parte de este modulo:

- event-store;
- candidatos generados desde backlog;
- dispatch de outbox;
- runtime real;
- seleccion de capacidad/modelo.
