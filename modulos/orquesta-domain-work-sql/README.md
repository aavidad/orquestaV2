# orquesta-domain-work-sql

Adaptador SQL generico para `DomainWorkJobRecordStorePortV0`.

Responsabilidad:

- usar una tabla SQL existente para persistir jobs de dominio;
- guardar `DomainWorkJobRequestV0` normalizado, `DomainWorkJobV0` aceptado y
  fingerprint idempotente;
- aplicar replay por `domain_ref + idempotency_key`;
- listar records `Request+Job` con filtros del contrato neutral;
- permitir placeholders SQL `question` (`?`) y `dollar` (`$1`, `$2`, ...)
  desde configuracion;
- servir como base para conectores DB concretos sin fijar driver.

Fuera de alcance:

- elegir motor DB canonico;
- registrar drivers SQLite/Postgres/MySQL;
- crear migraciones productivas;
- abrir conexiones, leer DSN o decidir pooling;
- ejecutar transacciones/upserts productivos especificos de un motor;
- actuar como store de runs, workflow tasks, outbox o estado global de Orquesta;
- ejecutar agentes;
- entregar artefactos al dominio externo;
- implementar `submit_artifact`;
- cablearse al Director o al servidor.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-domain-work-sql
```
