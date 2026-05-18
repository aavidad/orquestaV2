# orquesta-domain-work-file

Conector durable file-based para crear y consultar jobs de dominio externo.

Responsabilidad:

- implementar `DomainWorkJobCreatorPortV0`;
- implementar `DomainWorkJobRecordSourcePortV0`;
- guardar jobs aceptados en un snapshot JSON versionado;
- escribir el snapshot de forma atomica;
- recuperar jobs tras reinstanciar el adaptador;
- aplicar idempotencia por `domain_ref + idempotency_key`;
- listar records `Request+Job` con filtros explicitos sin exponer filesystem al
  nucleo;
- servir como referencia durable para futuros conectores de DB, cola, REST o
  app propietaria.

Fuera de alcance:

- contratos de DB o SQL;
- red, HTTP, MCP, runtime, Codex, OPES o UI;
- planificar trabajos;
- ejecutar agentes;
- entregar artefactos al dominio externo.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-domain-work-file
```
