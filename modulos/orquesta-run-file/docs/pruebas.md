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
- app-change persiste, lista por `run_ref` y reemplaza por `(run_ref,
  change_ref)` al recrear instancia.
- los snapshots contienen `schema_version` y `records`.
- la arquitectura no importa `cmd`, DB, red, runtime real ni proveedores.
