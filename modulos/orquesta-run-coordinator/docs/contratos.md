# Contratos

## Caso de uso

`CoordinateRunsTickV0(ctx, deps, command)` ejecuta un tick determinista del
runner global multiapp.

Entradas:

- `QueueRef`, `AppRefs` y `QueueLimit` se pasan a `RunQueueReaderPortV0`.
- `ExcludeRunRefs` evita ejecutar runs concretas en este tick; se devuelven
  como skips con razon `run_excluded`.
- `RankingPolicy` controla el ranking; si no trae `Now`, se usa una politica
  por defecto basada en `OccurredAt`.
- `MaxRuns` limita las ejecuciones por tick. Si llega a cero o negativo, se usa
  uno.
- `OccurredAt` y `CorrelationID` se copian a cada `RunDrainRequestV0`.
- `DrainLimits` viaja al drainer como presupuesto del ciclo. El coordinador no
  interpreta esos limites; solo evita que se pierdan entre capas.

Puertos:

- `orquesta-run-queue.RunQueueReaderPortV0`
- `orquesta-run-control.RunControlReaderPortV0`
- `RunDrainerPortV0`

Salidas:

- `Executions`: runs drenadas, con rank, outcome, `queue_status` opcional y
  evidencias.
- `Skips`: runs bloqueadas por control, con rank, razon y estado.
- `Ranked`: resumen compacto de candidatos rankeados.

`RunControlStateNotFoundErrorV0` se interpreta como estado `running`.
Si no hay `RunControlReaderPortV0`, el tick tambien considera la run ejecutable;
el control durable queda como puerto opcional para composiciones que aun no lo
inyecten. `QueueReader` y `RunDrainerPortV0` son obligatorios.

Si el `RunDrainerPortV0` devuelve `QueueStatus`, el coordinador lo propaga al
`QueueUpdater` en la rotacion posterior a la ejecucion. El coordinador no decide
ese estado: solo lo transporta desde la composicion hacia el puerto de cola.

## Reconciliacion ExternalWork/DomainWork

`ReconcileExternalWorkPublicStatusV0` es un primer corte puro e idempotente
para que una composicion no publique estados falsos de runs externas.

Fuentes causales modeladas:

- proyeccion de run;
- `WorkflowTaskStore`;
- outbox ledger;
- process registry;
- `agent_ack`/checkpoint;
- artefacto local;
- estado del job del dominio externo.

Reglas:

- un ACK completado observado gana a un estado `running/stale` y exige
  `ingest_completed_ack`;
- `projection=done` con `tasks=0` no se publica como `completed` si el job de
  dominio sigue `pending`; queda `blocked` con accion de followup/bloqueo
  causal;
- `running` solo se justifica por proceso vivo, tarea abierta, outbox pendiente
  o liveness verificable;
- si faltan fuentes causales, el estado publico queda `unknown` con
  `observe_causal_sources`.
