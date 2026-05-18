# Decisiones: orquesta-domain-work-sql

```text
Fecha: 2026-05-17
Decision: Crear `orquesta-domain-work-sql` como adaptador SQL generico y
driver-agnostic para `DomainWorkJobRecordStorePortV0`.
Motivo: `memory` y `file` fijan la semantica, pero hace falta una base para
conectores DB reales sin contaminar `orquesta-domain-work` ni elegir motor.
Alternativas: meter SQL en `orquesta-domain-work`; crear directamente un
conector SQLite/Postgres; reutilizar `orquesta-persistence`; seguir solo con
file-based.
Impacto: el adaptador usa `database/sql`, requiere tabla existente y ejecuta la
suite `contracttest`. No registra drivers, no crea schema productivo y no se
cablea al Director ni al servidor.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: soportar estilos de placeholder `question` y `dollar` en la
configuracion del adaptador SQL.
Motivo: `database/sql` no normaliza placeholders entre drivers; fijar `?`
servia para tests offline pero bloqueaba Postgres sin tocar codigo.
Alternativas: elegir un driver canonico; meter dialectos en
`orquesta-domain-work`; dejar que cada conector copie el store.
Impacto: el nucleo sigue sin saber SQL; el adaptador sigue sin registrar
drivers ni abrir DSN. La composicion externa elige el estilo adecuado al driver.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: aceptar un clasificador opcional `IsUniqueViolation` en la
configuracion del adaptador SQL.
Motivo: en DB real, dos creates concurrentes pueden pasar el pre-read y chocar en
el indice unico. El adaptador debe convertir esa carrera en replay o conflicto
contractual sin importar errores concretos de Postgres/MySQL/SQLite.
Alternativas: importar drivers para inspeccionar errores; exigir upsert canonico;
devolver siempre error tecnico.
Impacto: la composicion externa clasifica errores del driver. El adaptador relee
por `domain_ref + idempotency_key` y conserva la semantica de
`DomainWorkJobRecordStorePortV0`.
Estado: aceptada_local
```
