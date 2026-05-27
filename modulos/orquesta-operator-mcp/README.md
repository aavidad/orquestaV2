# orquesta-operator-mcp

Responsabilidad: fachada MCP operativa para que una IA/director invoque Orquesta sin conocer internos.

Incluye:

- descriptor compacto de capacidades operativas;
- consulta de estado por ref opaca;
- invocacion de burst supervisado por ref opaca y presupuesto;
- consultas dirigidas con trazabilidad compacta;
- envelopes y errores publicos estables para clientes MCP.

No incluye DB, runtime real, red, HOME, OAuth, proveedor, modelo, secretos, prompts ni conocimiento de tablas/event-store.

El resource `orquesta.operator.operations.v0` es fuente canonica de
capabilities operativas para T198: su descriptor MCP debe referenciar
`operator_capabilities_v0.go` y publicar errores publicos/puertos por refs
opacas, no por internals del operador.
La reconciliacion `agent-ref-task-autoprogramming-c3678e9bc306-g01` conserva
esa fuente como cierre documental; no habilita transporte real ni DB/runtime.

## Arranque

```bash
go test ./modulos/orquesta-operator-mcp
```
