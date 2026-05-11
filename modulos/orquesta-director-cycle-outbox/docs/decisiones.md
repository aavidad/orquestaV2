# Decisiones: orquesta-director-cycle-outbox

## DCO-DEC-001: Recorder Separado Del Dispatcher

Decision: este modulo solo guarda y lista outbox pendiente.

Motivo: el ciclo de dispatch existente necesita dispatcher y ACK; aqui solo queremos cerrar la consistencia entre runner y siguiente tick.

Consecuencia: no hay dispatcher, ACK, reintento ni politica de entrega.

## DCO-DEC-002: Ledger Como Puerto Minimo

Decision: se define un puerto local con `SavePending` y `ListPending`.

Motivo: permite usar memoria, fichero, DB u otro storage sin que el nucleo sepa nada de la implementacion.

Consecuencia: los adaptadores convierten issues del ledger concreto al issue compacto local.

## DCO-DEC-003: Pending Refs Para Scheduler

Decision: la salida principal son `pending_outbox_refs`.

Motivo: `orquesta-director-scheduler` solo necesita saber que hay outbox pendiente para esperar y no duplicar efectos.

Consecuencia: el ensamblador superior pasa esas refs a `orquesta-director-tick-input`.
