# orquesta-run-memory

Mini-proyecto conector de memoria para `orquesta-run-control` y `orquesta-run-queue`.

Incluye:

- `RunMemoryStoreV0`, store local thread-safe;
- implementacion de `orquestaruncontrol.RunControlPortV0`;
- implementacion de `orquestarunqueue.RunQueueReaderPortV0`;
- implementacion de `orquestarunqueue.RunQueuePriorityWriterPortV0`;
- helpers de siembra en memoria para candidatos y estados.

Fuera de alcance:

- DB, red, MCP, runtime, procesos, scheduler interno y workflow core;
- persistencia en ficheros o dependencias externas;
- reloj global: el ranking sigue viviendo en `orquesta-run-queue`.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-run-memory
```
