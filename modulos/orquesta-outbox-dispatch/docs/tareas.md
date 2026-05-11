# Tareas

- Primer corte: contratos, puerto abstracto y seleccion pura de pendientes.
- Segundo corte: `RunOutboxDispatchOnceV0` por puertos inyectados con claim,
  executor y ACK tras exito.
- Tercer corte: `OutboxDeliveryLeaseV0` local, puertos de claim/renew/release y
  ACK por `claim_ref`, con planificacion pura e invariantes de expiracion por
  `now_observed_at`.
- Cuarto corte: `ChooseDispatchBatchV0` selecciona varios intents elegibles para
  dispatch concurrente por adaptadores externos, sin ejecutar runtime ni conocer
  proveedor.
- Quinto corte: `PlanOutboxDispatchAckClosureV0` evidencia que el cierre real
  exige ACK correlacionado por item.
- Pendiente deliberado: ACK real, runtime y persistencia productiva.

```text
ID: OBD-003
Objetivo: Cerrar el contrato productivo pendiente de leases/ACK sin adaptador.
Write-set: outbox_delivery_lease_*_v0.go, outbox_delivery_lease_*_v0_test.go, docs locales
Validacion: go test -count=1 ./modulos/orquesta-outbox-dispatch
Estado: implementada en contrato y pruebas puras
Resultado: claim idempotente por message_id/claim_ref, conflictos explicitos,
renovacion/liberacion/ACK por refs opacas y expiracion por now_observed_at.
```

```text
ID: OBD-004
Objetivo: Permitir lotes de dispatch para agentes en paralelo.
Write-set: outbox_dispatch_batch_v0.go, outbox_dispatch_batch_v0_test.go, docs locales
Validacion: go test -count=1 ./modulos/orquesta-outbox-dispatch
Estado: implementada en contrato y pruebas puras
Resultado: batch ready con varios DispatchIntentV0, limite max_ready, exclusion
de message_id ya reclamado y filtro por run/target_port.
```

```text
ID: OBD-005
Objetivo: Evidenciar cierre real por ACK correlacionado por item.
Write-set: outbox_dispatch_ack_closure_v0.go,
outbox_dispatch_ack_closure_v0_test.go, docs locales
Validacion: go test -count=1 ./modulos/orquesta-outbox-dispatch
Estado: implementada en contrato y pruebas puras
Resultado: ACK missing queda pendiente, ACK failed queda fallido publico, ACK
success es idempotente por item y un batch parcial no cierra otros items.
```
