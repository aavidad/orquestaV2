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

Errores publicos:

- `director_cycle_step_invalido`
- `director_cycle_step_outbox`
- `director_cycle_step_tick_input`
- `director_cycle_step_runner`

## No Contratos

No forman parte de este modulo:

- event-store;
- candidatos generados desde backlog;
- dispatch de outbox;
- runtime real;
- seleccion de capacidad/modelo.
