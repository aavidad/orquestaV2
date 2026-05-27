# Pruebas: orquesta-domain-work-memory

Comando:

```sh
go test -count=1 ./modulos/orquesta-domain-work-memory
```

Cobertura:

- satisface `DomainWorkJobRecordStorePortV0`;
- ejecuta la suite compartida `orquesta-domain-work/contracttest`;
- crea jobs aceptados con `job_ref`, correlacion, idempotencia y refs;
- normaliza antes de guardar;
- replay idempotente devuelve el mismo `job_ref` y no duplica;
- conflicto de idempotencia devuelve `status=invalid` y conserva el job previo;
- identidad de job usa el builder canonico de `orquesta-domain-work` y repara
  colisiones de `job_ref` con sufijo explicito;
- request invalido no se guarda;
- contexto cancelado devuelve error operativo;
- replay concurrente no duplica jobs;
- lectura filtrada de records por filtros AND;
- contexto cancelado tambien se respeta al listar records;
- copias defensivas para estado visible;
- integracion offline con `orquesta-document-plan-expander`;
- guarda de arquitectura sin DB, red, filesystem, runtime ni adaptadores de
  producto.
