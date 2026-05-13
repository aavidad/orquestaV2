# Contratos: orquesta-autoprogramming

## AutoprogrammingRequestV0

Entrada pura para validar que una peticion de autoprogramacion es pequena,
aislada y ejecutable por adaptadores externos.

Campos obligatorios:

- `request_ref`, `project_ref`, `worktree_ref`, `branch_ref`;
- `worktree_isolated=true`;
- `tasks` con `task_ref` y `area`;
- `write_set` relativo y compacto;
- `required_tests`.

Limites por defecto:

- 3 tareas;
- 2 areas;
- 5 entradas de `write_set`.

Invariantes:

- No ejecuta agentes, tests, comandos, Git ni filesystem.
- No elige DB, runtime, proveedor, modelo ni HOME.
- No importa el nucleo de orquestacion.

## AutoprogrammingReviewGateInputV0

Entrada pura para decidir si una entrega de codigo puede aceptarse:

- ACK presente y `status=completed`;
- tests obligatorios ejecutados y pasados;
- ficheros dentro del `write_set`;
- limite de lineas por fichero, por defecto 300.

El resultado devuelve `accepted` e issues publicos. Un adaptador externo decide
como convertirlo en observaciones de review, rework o aceptacion.
