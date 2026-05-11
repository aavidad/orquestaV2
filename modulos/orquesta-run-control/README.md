# orquesta-run-control

Mini-proyecto puro para controlar el estado operativo de una run sin acoplarse al workflow core ni al scheduler interno.

Incluye:

- estado `RunControlStateV0` con `running`, `paused`, `stop_requested`, `stopped`, `cancel_requested` y `canceled`;
- contratos `RunControlReaderPortV0`, `RunControlWriterPortV0`,
  `RunControlTerminalWriterPortV0` y `RunControlPortV0`;
- comandos puros `PauseRunCommandV0`, `ResumeRunCommandV0`,
  `StopRunCommandV0`, `CancelRunCommandV0` y `CompleteRunControlCommandV0`;
- funcion pura `EvaluateRunControlV0` para decidir scheduling, dispatch, checkpoint, parada de agentes y terminalidad.
- helpers puros para normalizar la escritura terminal de `stopped/canceled`.

Fuera de alcance:

- DB, HTTP, MCP, runtime, scheduler interno y workflow core;
- adaptadores reales, transporte, almacenamiento o procesos;
- cierre material de runs o detencion fisica de agentes. Este modulo solo
  define el contrato terminal; el consumidor decide cuando es seguro invocarlo.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-run-control
```
