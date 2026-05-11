# Contratos v0

## Puertos implementados

`RunMemoryStoreV0` satisface:

- `orquestaruncontrol.RunControlPortV0`;
- `orquestarunqueue.RunQueueReaderPortV0`;
- `orquestarunqueue.RunQueuePriorityWriterPortV0`.

Como `RunQueuePriorityWriterPortV0` existe en `orquesta-run-queue`, el modulo tambien satisface `orquestarunqueue.RunQueuePortV0`.

## Control de runs

Los comandos `PauseRunV0`, `ResumeRunV0`, `StopRunV0` y `CancelRunV0` mutan solo memoria local.

Leer un `run_ref` sin estado explicito devuelve
`RunControlStateNotFoundErrorV0`; no crea estado implicito en el conector. La
politica de tratarlo como `running` vive en el consumidor.

Reglas:

1. `pause` deja la run en `paused`.
2. `resume` deja la run en `running`.
3. `stop` deja la run en `stop_requested`.
4. `cancel` deja la run en `cancel_requested`.
5. `forced`, `evidence_refs` y metadatos del comando se copian al estado.
6. `CompleteRunControlV0` solo acepta `stopped` o `canceled`, normaliza el
   comando y preserva el checkpoint previo.

## Cola de runs

`ListRunSchedulingCandidatesV0` devuelve snapshots de candidatos:

- filtra por `queue_ref` si esta informado;
- filtra por `app_refs` si estan informados;
- aplica `limit` si es mayor que cero;
- no rankea ni muta el estado.

`SetRunPriorityV0` normaliza y valida el comando con helpers de `orquesta-run-queue`, y actualiza solo `priority_score`, `app_ref` si llega informado y evidencias.
