# orquesta-operator-mcp

Responsabilidad: fachada MCP operativa para que una IA/director invoque Orquesta sin conocer internos.

Incluye:

- descriptor compacto de capacidades operativas;
- consulta de estado por ref opaca;
- invocacion de burst supervisado por ref opaca y presupuesto;
- consultas dirigidas con trazabilidad compacta;
- envelopes y errores publicos estables para clientes MCP.

No incluye DB, runtime real, red, HOME, OAuth, proveedor, modelo, secretos, prompts ni conocimiento de tablas/event-store.

## Arranque

```bash
go test ./modulos/orquesta-operator-mcp
```
