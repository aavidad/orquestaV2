# orquesta-director-cycle-outbox

Microproyecto para dejar registrada la outbox producida por un ciclo del director.

Responsabilidad v0:

- recibir mensajes `OutboxMessageV0` del runner;
- validar que pertenecen al run;
- guardarlos como pendientes mediante un puerto de ledger;
- listar refs pendientes para que el siguiente `DirectorSchedulerTickInputV0` espere correctamente.

Fuera de alcance:

- despachar outbox;
- registrar ACK;
- elegir DB o storage;
- arrancar runtime;
- leer event-store;
- construir candidates o snapshot del scheduler.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-cycle-outbox
```
