# Contratos: orquesta-domain-work-sql

## SQLDomainWorkJobRecordStoreV0

Implementa `DomainWorkJobRecordStorePortV0` sobre `database/sql`.

Entrada:

- `*sql.DB` ya configurado por composicion externa;
- nombre de tabla opcional, por defecto `domain_work_jobs_v0`;
- `PlaceholderStyle` opcional:
  - `question`, por defecto, genera `?`;
  - `dollar` genera `$1`, `$2`, ... para composiciones tipo Postgres.
- `IsUniqueViolation` opcional: clasificador inyectado por composicion para
  reconocer errores de unicidad del driver sin importar drivers concretos.

Tabla esperada:

- `domain_ref`
- `idempotency_key`
- `job_ref`
- `correlation_id`
- `work_kind`
- `status`
- `request_json`
- `job_json`
- `fingerprint`

La tabla debe tener unicidad por `(domain_ref, idempotency_key)` y por
`job_ref`. El adaptador no crea schema ni registra drivers.

Contrato operativo para DB real:

- la DB debe proteger `(domain_ref, idempotency_key)` y `job_ref` con indices
  unicos;
- si un insert concurrente devuelve unique violation y `IsUniqueViolation` lo
  clasifica, el adaptador relee por `domain_ref + idempotency_key`;
- si el fingerprint re-leido coincide, devuelve replay estable con el mismo
  `job_ref`;
- si el fingerprint difiere, devuelve `domain_work_sql_idempotency_conflict` sin
  sobrescribir;
- si la unique violation no se puede resolver por esa clave, devuelve error
  tecnico visible.

Semantica:

- normaliza y valida `DomainWorkJobRequestV0` antes de consultar o insertar;
- request invalido devuelve `DomainWorkJobV0{status=invalid}` sin escribir;
- replay con misma clave y mismo fingerprint devuelve el mismo job;
- misma clave con fingerprint distinto devuelve
  `domain_work_sql_idempotency_conflict` sin sobrescribir;
- errores Go quedan para fallos tecnicos de SQL, contexto o serializacion;
- `ListDomainWorkJobRecordsV0` devuelve records ordenados por `job_ref` y filtra
  por contrato neutral en memoria.
- el estilo de placeholders solo cambia las consultas generadas; no cambia la
  semantica del store ni el contrato del nucleo.

El listado actual carga `request_json` y `job_json`, ordena por `job_ref` y aplica
filtros en memoria. Es suficiente para contrato y referencia offline; un conector
DB productivo grande debe empujar filtros/limites a SQL e indexar los campos y
refs necesarios.

Frontera:

- este modulo puede importar `database/sql`;
- no importa drivers concretos, OPES, Codex, runtime, MCP, web, `cmd`, `db`,
  state-file, run-file ni otros adaptadores de `domain_work`.
