# Contratos

## OutboxPendingEntryV0

Entrada pendiente ya persistida por otro modulo. Sus campos son opacos para este
microproyecto salvo `message_id`, `run_id`, `target_port`, `message_type` y
`payload_version`, que deben venir informados para ser elegibles.

`correlation_id` viaja desde la entrada pendiente hasta `DispatchIntentV0` si el
ledger lo aporta. El selector no lo interpreta, pero los executors lo necesitan
para preservar trazabilidad al llamar puertos como `agent_launcher`.

## DispatchSelectionV0

Solicitud pura de seleccion. Recibe pendientes en memoria, un `target_port`
obligatorio, un `run_id` opcional y `claimed_message_ids` para impedir duplicar
una entrada ya tomada por otra decision.

## DispatchDecisionV0

Resultado del caso de uso. Puede ser:

- `ready`: contiene un unico `DispatchIntentV0`.
- `no_pending`: no hay entrada elegible y no debe ejecutarse dispatch.
- `invalid_request`: la solicitud no cumple contrato.

## DispatchBatchSelectionV0

Solicitud pura de seleccion multiple. Recibe pendientes en memoria, `target_port`
obligatorio, `run_id` opcional, `claimed_message_ids` y `max_ready`.

`max_ready` limita cuantos intents se devuelven en el lote. Si no se informa o
es menor que uno, se usa `1` para conservar compatibilidad con el flujo serial.

## DispatchBatchDecisionV0

Resultado de lote. Puede ser:

- `ready`: contiene varios `DispatchIntentV0`, todos elegibles para ejecucion
  concurrente por un adaptador operativo.
- `no_pending`: no hay entradas elegibles.
- `invalid_request`: la solicitud no cumple contrato.

El batch no ejecuta, no crea goroutines, no conoce proveedor, HOME, OAuth,
modelo ni runtime. Solo decide que intents pueden reclamarse/ejecutarse en
paralelo por un dispatcher externo.

## PlanOutboxDispatchAckClosureV0

Planificador puro para decidir que items de dispatch pueden retirarse tras
observaciones de ACK. Recibe `DispatchIntentV0` y
`OutboxDispatchAckObservationV0` en memoria. Correlaciona cada ACK por
`message_id`, `run_id` y `target_port`.

Estados por item:

- `pending`: no hay ACK correlacionado; el item no se cierra.
- `ack_failed`: hay ACK `failed`; el fallo queda expuesto como estado publico.
- `acked`: hay ACK `success`; el item puede retirarse.

El ACK `success` es idempotente para el mismo item: duplicarlo no duplica cierre
ni afecta otros intents. En un batch parcial, solo se cierra el item con ACK
correlacionado; los demas quedan pendientes o fallidos segun su propio ACK.

## PendingOutboxReaderPortV0

Puerto abstracto para listar pendientes. Este modulo declara el puerto pero no
incluye implementaciones de DB, red ni filesystem productivo.

## OutboxDispatchAckObservationPortV0

Puerto opcional para adaptadores que necesitan persistir ACK terminales de lote
con status `success` o `failed`. Mantiene el puerto historico
`OutboxDispatchAckPortV0` para ACK success y evita cerrar por arrastre items sin
ACK correlacionado.

## RunOutboxDispatchOnceV0

Caso de uso de un solo paso. Requiere `target_port` explicito y puertos
inyectados:

- `PendingOutboxReaderPortV0`: lista pendientes por `run_id` opcional y
  `target_port`.
- `OutboxDispatchClaimerPortV0`: reclama el `message_id` elegido antes de
  ejecutar.
- `OutboxDispatchExecutorPortV0`: ejecuta el dispatch contra el puerto logico.
- `OutboxDispatchAckPortV0`: registra ACK solo tras ejecucion exitosa.

Si no hay pendiente elegible devuelve status `no_pending`. Si el claim indica
que la entrada ya estaba reclamada devuelve `already_claimed` y no ejecuta. Si el
executor falla devuelve `dispatch_failed` y no registra ACK.

## OutboxDeliveryLeaseV0

Contrato puro de claim/lease para entrega de outbox. Es compatible
conceptualmente con el candidato de `orquesta-core-leases`, pero se declara
localmente para no acoplar este microproyecto al core.

Campos:

- `claim_ref`: ref opaca del claim vigente.
- `message_id`: mensaje outbox reclamado.
- `target_port`: puerto logico del dispatch.
- `claimed_by_ref`: ref opaca del dispatcher reclamante.
- `claimed_at`: instante observado externo del claim.
- `lease_until`: expiracion observada del lease.
- `attempt`: intento monotono calculado por el adaptador/puerto.

No contiene SQL, Redis, locks, runtime, proveedor, HOME, OAuth ni backend
concreto. La expiracion se decide solo con `now_observed_at` recibido como input.

## Puertos de lease y ACK por ref

Puertos declarados:

- `OutboxDeliveryLeaseClaimPortV0`: reclama por `message_id` y devuelve
  `claim_ref`.
- `OutboxDeliveryLeaseRenewalPortV0`: renueva por `claim_ref`.
- `OutboxDeliveryLeaseReleasePortV0`: libera por `claim_ref`.
- `OutboxDeliveryAckByLeasePortV0`: marca ACK por `claim_ref`, `dispatch_ref` y
  evidencias opacas.

Funciones puras de planificacion:

- `PlanOutboxDeliveryLeaseClaimV0`
- `PlanOutboxDeliveryLeaseRenewalV0`
- `PlanOutboxDeliveryLeaseReleaseV0`
- `PlanOutboxDeliveryAckByLeaseV0`

Estas funciones no persisten, no ejecutan conectores y no leen reloj real.
