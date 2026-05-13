# Contratos

## `ShutdownServerV0`

Entrada: `ServerShutdownCommandV0`.

Puertos:

- `RunQueueReaderPortV0` para descubrir runs candidatos;
- `RunControlReaderPortV0` para no reabrir terminales;
- `RunControlWriterPortV0` para solicitar `stop`;
- `RunControlCheckpointWriterPortV0` para registrar ACK durable de checkpoint;
- `PrepareAgentShutdownPortV0` para preparar checkpoint antes de stop no
  forzado;
- `RunSupervisorPortV0` para drenar;
- `RunStatsReaderPortV0` para verificar `agents_in_flight`.

Invariantes:

- no toca stores ni procesos directamente;
- `forced=false` prepara checkpoint por puerto y registra ACK durable antes de
  solicitar `StopRunV0`; si falta ACK, no pide stop ni ejecuta drainer;
- `forced=true` permite drenar agentes sin checkpoint previo;
- si falta checkpoint, el resultado expone `pending_checkpoint_agent_refs`,
  `checkpoint_evidence_refs` y `checkpoint_agents_pending` con refs compactas,
  sin rutas ni detalles de runtime;
- `shutdown_ready=true` solo cuando todas las runs objetivo estan terminales o
  sin agentes en vuelo segun stats compactas y sin checkpoint pendiente.
