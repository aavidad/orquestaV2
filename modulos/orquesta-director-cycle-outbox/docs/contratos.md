# Contratos: orquesta-director-cycle-outbox

## RecordDirectorCycleOutboxV0

Entrada canonica: `DirectorCycleOutboxRecordInputV0`.

Campos obligatorios:

- `ledger`: puerto inyectado.
- `run_ref`: run objetivo.

Campos opcionales:

- `messages`: outbox generada por un ciclo.
- `correlation_id`

Salida:

- `saved_count`: mensajes aceptados por el ledger.
- `pending_count`: mensajes pendientes para el run.
- `pending_outbox_refs`: refs compactas de mensajes pendientes.

Invariantes:

- No despacha mensajes.
- No marca ACK.
- No elige persistencia.
- Rechaza outbox de otro run.
- Valida `OutboxMessageV0` antes de guardarlo.
- Lista pendientes aunque `messages` venga vacio.

Puerto:

- `DirectorCycleOutboxLedgerPortV0`
  - `SavePending(ctx, messages)`
  - `ListPending(ctx, filter)`

Errores publicos:

- `director_cycle_outbox_invalido`
- `director_cycle_outbox_ledger`

## No Contratos

No forman parte de este modulo:

- dispatch real;
- reintentos;
- runtime;
- DB concreta;
- event-store;
- scheduler.
