# orquesta-core

Responsabilidad: dominio y casos de uso de Orquesta.

Incluye:

- proyectos;
- fases;
- tareas;
- contratos de funcion;
- propuestas;
- votos;
- decisiones;
- validacion de entrega;
- cierre.

No incluye:

- DB concreta;
- Codex, Claude, Ollama o proveedor concreto;
- HTTP, CLI, MCP o UI;
- tmux, Docker o sistema operativo.

El core se conecta con el resto por puertos.

Para T198, `FunctionContractV0` y los contratos compartidos publicados por core
son fuentes canonicas de descriptors MCP. Core no importa MCP ni transporte; el
adaptador solo referencia owner, DTO/validador, freshness y errores publicos.
La reconciliacion `agent-ref-task-autoprogramming-c3678e9bc306-g01` conserva
esa frontera: core sigue sin depender de MCP, HTTP, CLI ni runtime.
