# orquesta-domain-work-memory

Conector in-memory generico para crear jobs de dominio externo.

Responsabilidad:

- implementar `DomainWorkJobCreatorPortV0`;
- implementar `DomainWorkJobRecordSourcePortV0`;
- normalizar y validar `DomainWorkJobRequestV0` antes de guardar;
- devolver `DomainWorkJobV0` aceptados con `job_ref` compacto;
- aplicar idempotencia por `domain_ref + idempotency_key`;
- listar records `Request+Job` con filtros explicitos;
- servir como referencia volatil de comportamiento para conectores durables.

Fuera de alcance:

- DB, filesystem, red, runtime, Codex, OPES o UI;
- planificar trabajos;
- ejecutar agentes;
- entregar artefactos al dominio externo.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-domain-work-memory
```

Para persistencia durable file-based, usar `modulos/orquesta-domain-work-file`.
