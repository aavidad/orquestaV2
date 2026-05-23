# Contratos: orquesta-autoprogramming

## AutoprogrammingRequestV0

Entrada pura para validar que una peticion de autoprogramacion es pequena,
aislada y ejecutable por adaptadores externos.

Campos obligatorios:

- `request_ref`, `project_ref`, `worktree_ref`, `branch_ref`;
- `worktree_isolated=true`;
- `tasks` con `task_ref` y `area`; opcionalmente cada tarea puede aportar
  `title`, `objective`, `context`, `context_refs`, `acceptance_criteria`,
  `required_tests` y `compact_rules` para que el worker reciba un contrato
  explicito y no tenga que inferir semantica desde `task_ref`;
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

## BuildAutoprogrammingProgrammableWorkV0

Funcion pura que valida `AutoprogrammingRequestV0` y construye trabajo
programable compatible con el nucleo:

- devuelve `WorkProfileV0` y `WorkflowTaskV0` por grupo;
- conserva `write_set`, `required_tests`, `worktree_ref` y `branch_ref`;
- conserva objetivo, contexto, criterios, tests y reglas compactas declaradas
  por tarea dentro de `WorkflowTaskV0`/`WorkProfileV0`;
- usa refs opacas como `ContextRefs`, sin leer repositorios ni ejecutar nada;
- para varios grupos, particiona `write_set` por area normalizada y rechaza
  rutas ambiguas, sin area o solapadas.
