# Pruebas: orquesta-domain-work-sql

Comando:

```sh
go test -count=1 ./modulos/orquesta-domain-work-sql
```

Cobertura:

- satisface `DomainWorkJobRecordStorePortV0`;
- ejecuta `orquesta-domain-work/contracttest`;
- usa un driver fake local de `database/sql` en tests, sin dependencia externa;
- ejecuta el contrato con placeholders `question` y `dollar`;
- comprueba que la configuracion `dollar` genera `$1`, `$2`, ...;
- simula unique violation concurrente y verifica replay estable;
- simula unique violation con fingerprint distinto y verifica conflicto
  idempotente sin sobrescribir;
- repara colision visible de `job_ref` determinista con sufijo explicito;
- rechaza `nil *sql.DB`;
- rechaza nombres de tabla invalidos;
- rechaza estilos de placeholder desconocidos;
- no importa drivers concretos ni adaptadores de producto.

Pruebas transversales:

```sh
go test -count=1 ./modulos/orquesta-domain-work/... ./modulos/orquesta-domain-work-memory ./modulos/orquesta-domain-work-file ./modulos/orquesta-domain-work-sql
go test -count=1 . -run TestNeutralOrchestrationPackagesDoNotImportProductAdapters
```
