# Decisiones: orquesta-operator-mcp

## v0 scaffold

- MCP es fachada inbound, no cerebro operativo.
- Estado, burst y consultas se delegan a conectores opacos.
- Los clientes IA reciben nombres de capacidades, refs opacas y errores publicos.
- No se publican internos de director, scheduler, runtime, capacity, persistence u observability.

## outbox pendiente v0

- El listado de pendientes es una capacidad operativa separada.
- El contrato solo devuelve refs y contadores compactos.
- La ejecucion queda delegada a un puerto opaco inyectado por el adaptador.

## adaptador MCP puro v0

- `orquesta-mcp` puede exponer las operaciones del operador como resource/tools puros.
- El servidor/transporte MCP real sigue fuera del modulo y debe entrar como conector opt-in.
- La falta de puerto se informa como error publico; no hay fallback a DB, runtime, filesystem ni scheduler interno.
- Este cierre no cambia el nucleo: solo documenta la frontera entre contratos operativos y adaptador MCP.

## transporte MCP opt-in v0

- `TransportPortV0` es una frontera de registro, no un servidor real.
- El adaptador externo decide sockets/protocolo/ciclo de vida fuera de estos modulos.
- `orquesta-operator-mcp` no conoce internals ni transportes concretos; mantiene DTOs, refs opacas y puertos publicos.
- Los handlers operativos deben fallar por error publico si falta puerto inyectado; no hay lectura directa de DB, outbox, runtime, event-store ni filesystem productivo.
