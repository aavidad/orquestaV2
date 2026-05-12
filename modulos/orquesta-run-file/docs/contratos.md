# Contratos v0

## Puertos implementados

`RunFileStoreV0` satisface:

- `orquestaruncontrol.RunControlPortV0`;
- `orquestarunqueue.RunQueuePortV0`;
- `orquestaappchange.AppChangeRecordStorePortV0`.

## Persistencia

El adaptador guarda snapshots JSON versionados y separados por responsabilidad:

- control de runs;
- cola de runs;
- solicitudes de cambio de app.

Leer un estado de control inexistente devuelve
`RunControlStateNotFoundErrorV0`, igual que el adaptador de memoria.

Las operaciones de cola no rankean ni reservan runs. `SetRunPriorityV0` crea un
candidato minimo con estado `ready` si no existia, como `orquesta-run-memory`.

Las solicitudes de cambio se guardan por clave `(run_ref, change_ref)`,
reemplazando una solicitud previa y preservando el orden de insercion.
