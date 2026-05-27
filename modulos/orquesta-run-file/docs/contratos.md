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
Si el comando trae `status` o `fairness_group_ref`, los persiste; esto permite
que una composicion marque una run cerrada en el core como terminal tambien en
la cola durable y conserve el grupo usado por la politica de fairness.
`ListRunSchedulingCandidatesV0` devuelve solo candidatos ejecutables de
scheduling; filtra estados terminales/no ejecutables antes de aplicar `limit`
para que la cola historica no bloquee trabajo nuevo.

Las solicitudes de cambio se guardan por clave `(run_ref, change_ref)`,
reemplazando una solicitud previa y preservando el orden de insercion.

`ReloadFromDiskV0` recarga los tres snapshots en memoria cuando una composicion
autorizada compacta los ficheros fuera de las mutaciones normales del store.
