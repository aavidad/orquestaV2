# orquesta-domain-work

Contrato generico para trabajos de dominio externos a Orquesta.

Responsabilidad:

- describir un trabajo externo mediante refs opacas;
- transportar campos de dominio normalizados sin JSON libre;
- exigir `correlation_id`, `idempotency_key` y `requested_by`;
- modelar la creacion de un job externo por puerto;
- modelar la lectura filtrada de records `Request+Job` por puerto separado;
- modelar la entrega de artefactos por puerto;
- validar que el contrato no arrastra DB, rutas, runtime, proveedor ni modelo.

Este modulo permite que OPES, programacion u otra app futura consuman la
orquestacion de agentes sin integrar su nucleo dentro de Orquesta.

Implementaciones de referencia:

- `modulos/orquesta-domain-work/contracttest`: suite de conformidad para que los
  adaptadores validen `DomainWorkJobRecordStorePortV0`.
- `modulos/orquesta-domain-work-memory`: conector in-memory neutral para
  `DomainWorkJobRecordStorePortV0`, pensado para pruebas offline y como guia
  para conectores durables de DB, cola, REST o app propietaria.
- `modulos/orquesta-domain-work-file`: conector durable file-based para
  `DomainWorkJobRecordStorePortV0`, con snapshot JSON atomico, replay tras
  reinstanciar y lectura filtrada de records persistidos.
- `modulos/orquesta-domain-work-sql`: adaptador SQL externo para
  `DomainWorkJobRecordStorePortV0`, con `*sql.DB` inyectado y sin driver
  canonico.

Regla de juicio:

- el dominio externo describe objetivo, reglas y contexto;
- Orquesta piensa mediante director/agentes;
- el dominio externo valida y ensambla con reglas deterministicas;
- si el dominio necesita planificacion inteligente, debe pedir un trabajo de
  planificacion a Orquesta, no implementarla dentro del dominio.

Fuera de alcance:

- REST, MCP, HTTP o clientes reales;
- DB, ficheros, colas, workers o rutas internas de apps externas;
- seleccion de agentes, modelos, cuotas, leases, sesiones o tmux;
- payloads de dominio sin contrato.
- planificacion documental sin contrato: usar `DomainDocumentPlanV0`
  (`PlanTemaV0`/`PlanTemarioV0`) para planes de tema, temario o documento.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-domain-work/...
```
