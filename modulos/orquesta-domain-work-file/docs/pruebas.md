# Pruebas: orquesta-domain-work-file

Comando:

```sh
go test -count=1 ./modulos/orquesta-domain-work-file
```

Cobertura:

- satisface `DomainWorkJobRecordStorePortV0`;
- ejecuta la suite compartida `orquesta-domain-work/contracttest`;
- crea jobs aceptados y escribe snapshot JSON versionado;
- replay tras reinstanciar devuelve el mismo `job_ref`;
- lista records `Request+Job` con filtros AND por campos y `external_refs`;
- filtro vacio equivale al listado de jobs en orden estable por `job_ref`;
- `limit` corta despues de ordenar y filtrar;
- lectura filtrada sobrevive a reinstanciar el adaptador;
- conflicto de idempotencia no sobrescribe el archivo;
- identidad de job usa el builder canonico, repara colisiones de `job_ref` y
  preserva replay de snapshots con fingerprint legacy por request guardado;
- request invalido no escribe estado;
- snapshot corrupto falla al abrir y no se repara en silencio;
- contexto cancelado no escribe y tampoco lista records;
- directorio relativo se rechaza;
- lecturas devuelven copias defensivas de request y job;
- integracion offline con `orquesta-document-plan-expander` y replay tras
  reinicio;
- guarda local contra DB, red, runtime, Codex, OPES, MCP, web y otros
  adaptadores concretos.
