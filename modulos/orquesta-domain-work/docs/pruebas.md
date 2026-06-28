# Pruebas: orquesta-domain-work

Comando:

```sh
go test -count=1 ./modulos/orquesta-domain-work/...
```

Cobertura:

- normalizacion de job externo sin importar adaptadores concretos;
- normalizacion y validacion de `DomainDocumentPlanV0` como
  `PlanTemaV0`/`PlanTemarioV0`;
- rechazo de planes sin secciones, sin entregables, con rangos invalidos o con
  `work_kind` que no sea de planificacion;
- deduplicacion de refs e idempotency/correlation por defecto;
- rechazo de refs no compactas;
- validacion de entrega de artefacto;
- normalizacion de filtros `DomainWorkJobRecordFilterV0`;
- puertos hexagonales de comando, lectura y entrega sin adaptadores concretos;
- politica de tests requeridos de dominio como contrato/puerto:
  `required_tests` viaja por `test_ref`, criteria refs, input refs,
  external refs y evidencias, sin banco comun ni semantica OPES;
- capacidades externas de dominio: jobs de audio derivan requisito
  `speech_synthesis`, ausencia de capacidad bloquea con
  `domain_work_external_capability_missing`, y aliases `tts`/`text_to_speech`
  se normalizan sin acoplar proveedor ni runner;
- identidad canonica de jobs con fingerprint `sha256`, `request_id` excluido,
  `required_tests` incluido y base de `job_ref` compartida por adaptadores;
- suite reusable `contracttest` para stores `DomainWorkJobRecordStorePortV0`;
- guard de arquitectura contra DB, red, runtime, filesystem y legacy.

Prueba relacionada:

```sh
go test -count=1 ./modulos/orquesta-domain-work-memory
```

Ese modulo ejecuta la suite `contracttest` sobre una implementacion neutral de
`DomainWorkJobRecordStorePortV0` y anade integracion offline con
`orquesta-document-plan-expander`.

```sh
go test -count=1 ./modulos/orquesta-domain-work-file
```

Ese adaptador ejecuta la misma suite `contracttest` y anade persistencia durable
con snapshot JSON, replay tras reinstanciar, corrupcion visible, escritura
atomica y frontera contra OPES, Codex, runtime, red y DB.

```sh
go test -count=1 ./modulos/orquesta-domain-work-sql
```

Ese adaptador ejecuta la misma suite `contracttest` sobre `database/sql` con un
driver fake local de tests. No registra drivers reales ni fija SQLite/Postgres.
