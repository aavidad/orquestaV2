# Decisiones

- `ChooseNextDispatchV0` no ejecuta efectos: solo devuelve `DispatchIntentV0`.
- `RunOutboxDispatchOnceV0` coordina efectos solo mediante puertos inyectados.
- `target_port` es obligatorio para evitar selecciones ambiguas entre puertos.
- `run_id` es opcional para permitir lotes por puerto cuando el llamador lo
  necesite.
- `claimed_message_ids` es la barrera minima contra duplicados en este corte.
- `correlation_id` se propaga a `DispatchIntentV0` sin interpretarlo. Motivo:
  los executors de puertos externos deben conservar trazabilidad del outbox sin
  inventar correlaciones locales.
- El ACK se emite solo despues de un executor exitoso; fallo de executor no
  registra ACK failed en este modulo.
- El cierre real de items despachados se decide aparte con
  `PlanOutboxDispatchAckClosureV0`: sin ACK correlacionado el item sigue
  pendiente; ACK failed queda como estado publico; ACK success retira solo el
  item correlacionado.
- No se importa `orquesta-director`; sus pruebas sirven solo como referencia de
  comportamiento esperado alrededor del outbox.

```text
Fecha: 2026-05-07
Decision: los leases productivos de outbox se expresan como contrato local
`OutboxDeliveryLeaseV0` y puertos opt-in por refs opacas.
Motivo: el dispatcher necesita claim, renovacion, liberacion y ACK terminal sin
acoplarse a persistence, SQL, Redis, runtime, proveedor, HOME ni OAuth.
Alternativas: importar core-leases; ampliar el claim simple existente; definir
adaptador productivo ahora.
Impacto: los adaptadores futuros podran implementar atomicidad fuera de este
modulo; aqui solo quedan invariantes puros y replayables.
Estado: aceptada para OBD-003
```

```text
Fecha: 2026-05-07
Decision: el batch de dispatch no cierra por arrastre; el cierre se planifica
por ACK correlacionado con message_id/run_id/target_port.
Motivo: seleccionar o ejecutar varios intents no prueba entrega terminal de cada
item. El replay debe poder distinguir ACK missing, ACK failed y ACK success.
Alternativas: cerrar todo el batch ante cualquier ACK success; cerrar al ejecutar
el conector; delegar la correlacion a un adaptador no probado.
Impacto: los adaptadores futuros solo podran retirar el item con ACK success
correlacionado; otros items quedan pendientes o fallidos publicamente.
Estado: aceptada para OBD-005
```

```text
Fecha: 2026-05-09
Decision: `OutboxDispatchAckObservationV0` conserva issues sanitizados en ACK failed.
Motivo: un batch de agentes no puede reducir cualquier fallo a `ack_failed`; el director necesita distinguir bloqueo de comando, fallo de proceso, falta de readiness o error de workflow sin recibir rutas, HOME, proveedor ni secretos.
Alternativas:
  - Mantener solo `ack_failed`: descartado; oculta la causa raiz y favorece bucles de parcheo.
  - Meter logs o stderr en el ACK: descartado; filtraria detalles operacionales y crecerian los contextos.
Impacto: el cierre de ACK usa los issues del executor si existen; si no, mantiene el issue generico anterior. Solo se propagan codigo, campo y message key compactos.
Estado: aceptada para OBD-006
```
