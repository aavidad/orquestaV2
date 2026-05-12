# orquesta-run-file

Adaptador durable en ficheros para:

- `orquestaruncontrol.RunControlPortV0`;
- `orquestarunqueue.RunQueuePortV0`;
- `orquestaappchange.AppChangeRecordStorePortV0`.

`RunFileStoreV0` recibe un directorio absoluto y mantiene tres snapshots JSON:

- `control_v0.json`;
- `queue_v0.json`;
- `app_change_v0.json`.

Cada mutacion toma un mutex, prepara una copia completa, escribe un temporal en
el mismo directorio y publica con `rename`.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-run-file
```
