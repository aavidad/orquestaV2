# Pruebas

Comando local:

```sh
go test -count=1 ./modulos/orquesta-run-file
```

Cobertura v0:

- `RunFileStoreV0` satisface los puertos esperados.
- run-control persiste estado terminal y conserva checkpoint al recrear
  instancia.
- run-queue persiste candidatos y prioridades al recrear instancia.
- run-queue persiste estados terminales escritos por `SetRunPriorityV0` y los
  filtra al recrear instancia.
- app-change persiste, lista por `run_ref` y reemplaza por `(run_ref,
  change_ref)` al recrear instancia.
- los snapshots contienen `schema_version` y `records`.
- el servidor puede compactar snapshots de arranque y recargar el store
  file-based para que el estado terminal no reaparezca desde memoria.
- la arquitectura no importa `cmd`, DB, red, runtime real ni proveedores.
