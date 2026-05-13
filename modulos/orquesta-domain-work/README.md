# orquesta-domain-work

Contrato generico para trabajos de dominio externos a Orquesta.

Responsabilidad:

- describir un trabajo externo mediante refs opacas;
- transportar campos de dominio normalizados sin JSON libre;
- exigir `correlation_id`, `idempotency_key` y `requested_by`;
- modelar la creacion de un job externo por puerto;
- modelar la entrega de artefactos por puerto;
- validar que el contrato no arrastra DB, rutas, runtime, proveedor ni modelo.

Este modulo permite que OPES, programacion u otra app futura consuman la
orquestacion de agentes sin integrar su nucleo dentro de Orquesta.

Fuera de alcance:

- REST, MCP, HTTP o clientes reales;
- DB, ficheros, colas, workers o rutas internas de apps externas;
- seleccion de agentes, modelos, cuotas, leases, sesiones o tmux;
- payloads de dominio sin contrato.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-domain-work
```
