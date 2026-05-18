# AGENTS: orquesta-domain-work-sql

Este modulo es un adaptador SQL externo para `orquesta-domain-work`.

Reglas:

- implementar `DomainWorkJobRecordStorePortV0` sin ampliar el contrato puro;
- usar `database/sql` como frontera tecnica, sin importar drivers concretos;
- no importar OPES, Codex, runtime, MCP, web, `cmd`, `db`, file adapters ni
  conectores de producto;
- no asumir SQLite, Postgres, MySQL ni otro motor como canonico;
- si un driver necesita otro estilo de placeholders, configurarlo aqui como
  dialecto del adaptador, no en `orquesta-domain-work`;
- no meter DDL sqlite-first ni migraciones automaticas en el nucleo;
- normalizar y validar antes de escribir;
- conservar request normalizado, job aceptado y fingerprint;
- aplicar idempotencia por `domain_ref + idempotency_key`;
- devolver copias defensivas;
- ejecutar `orquesta-domain-work/contracttest` en los tests del adaptador.

El wiring real debe ocurrir en composicion exterior y con configuracion opt-in.
