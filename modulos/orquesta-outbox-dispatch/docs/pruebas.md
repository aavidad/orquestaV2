# Pruebas

Pruebas unitarias actuales:

- `TestChooseNextDispatchV0NoDuplicaEntradaReclamada`: si `message_id` ya esta
  reclamado, la decision queda en `no_pending` y no produce intent.
- `TestChooseNextDispatchV0NoDespachaSiNoHayPendiente`: sin pendientes, no hay
  dispatch.
- `TestChooseNextDispatchV0SeleccionaPrimeraPendienteElegible`: elige la primera
  entrada que coincide con `target_port` y `run_id`.
- `TestRunOutboxDispatchOnceV0SuccessExecuteAndAck`: ejecuta el conector y
  registra ACK con `target_port` explicito.
- `TestRunOutboxDispatchOnceV0ExecutorErrorDoesNotAck`: si falla el executor no
  registra ACK.
- `TestRunOutboxDispatchOnceV0NoPendingDoesNotExecute`: sin pendiente no reclama,
  no ejecuta y no registra ACK.
- `TestRunOutboxDispatchOnceV0ClaimedDoesNotDuplicate`: si el claim falla por ya
  reclamado no duplica ejecucion ni ACK.
- `TestPlanOutboxDeliveryLeaseClaimV0CreatesOpaqueLease`: crea lease puro con
  `claim_ref`, `message_id`, `target_port`, `claimed_by_ref` y expiracion
  observada.
- `TestPlanOutboxDeliveryLeaseClaimV0IdempotentByMessageAndClaimRef`: repetir el
  mismo `message_id`/`claim_ref` devuelve el lease existente.
- `TestPlanOutboxDeliveryLeaseClaimV0RejectsMessageConflict`: otro `claim_ref`
  no puede tomar un `message_id` con lease vigente.
- `TestPlanOutboxDeliveryLeaseClaimV0RejectsClaimRefConflict`: un `claim_ref` no
  puede apuntar a otro `message_id`.
- `TestPlanOutboxDeliveryLeaseClaimV0AllowsNewClaimAfterObservedExpiration`:
  otro claim puede tomar el mensaje solo cuando `now_observed_at` ya expiro el
  lease previo.
- `TestPlanOutboxDeliveryLeaseRenewalV0RejectsExpiredLeaseByObservedNow`: la
  renovacion rechaza leases expirados segun input externo.
- `TestPlanOutboxDeliveryAckByLeaseV0UsesClaimRefOnly`: el ACK usa `claim_ref` y
  refs opacas, no `message_id`.
- `TestOutboxDeliveryLeaseContractsUseOnlyOpaqueRefsV0`: los DTO de lease no
  exponen campos de infraestructura concreta.
- `TestPlanOutboxDispatchAckClosureV0AckMissingQuedaPendiente`: un intent sin
  ACK correlacionado queda pendiente.
- `TestPlanOutboxDispatchAckClosureV0AckFailedQuedaFallidoPublico`: un ACK
  `failed` produce estado publico `ack_failed`.
- `TestPlanOutboxDispatchAckClosureV0AckSuccessIdempotenteRetiraSoloEseItem`:
  ACK `success` duplicado cierra una sola vez y no retira otros items.
- `TestPlanOutboxDispatchAckClosureV0BatchParcialNoCierraOtrosItems`: un ACK
  parcial de lote no cierra los demas intents.

Comando:

```bash
go test -count=1 ./modulos/orquesta-outbox-dispatch
```
