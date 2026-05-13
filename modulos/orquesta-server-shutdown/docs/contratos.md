# Contratos

## `ShutdownServerV0`

Entrada: `ServerShutdownCommandV0`.

Puertos:

- `RunQueueReaderPortV0` para descubrir runs candidatos;
- `RunControlReaderPortV0` para no reabrir terminales;
- `RunControlWriterPortV0` para solicitar `stop`;
- `RunSupervisorPortV0` para drenar;
- `RunStatsReaderPortV0` para verificar `agents_in_flight`.

Invariantes:

- no toca stores ni procesos directamente;
- `forced=false` respeta checkpoint requerido;
- `forced=true` permite drenar agentes sin checkpoint previo;
- `shutdown_ready=true` solo cuando todas las runs objetivo estan terminales o
  sin agentes en vuelo segun stats compactas.
