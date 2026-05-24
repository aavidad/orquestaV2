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
- opcionalmente `area_aliases` para normalizar nombres seguros de area y
  `live_works` para declarar trabajos vivos con refs opacas y `write_set`.

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

El resultado tambien expone:

- `preserve_output`: la salida puede conservarse como evidencia reutilizable;
- `requires_followup`: la entrega se conserva como aceptada con rail dudoso y
  debe quedar como evidencia para revision o tarea posterior dirigida;
- `recommended_action`: `accept`, `request_followup_review`, `block_closure` o
  `request_changes`.

Si una entrega tiene ACK completado, tests obligatorios verdes y solo trae rails
dudosos de codigo (`file_too_large`, `file_outside_write_set` o destino faltante
del `write_set`), el gate devuelve `accepted=true`, conserva la salida y adjunta
`requires_followup=true` como evidencia para review/rework posterior. Asi un
arreglo util no queda bloqueado por falsos positivos, pero la evidencia no se
pierde.

Las refs `ack-pending-rail:*` se tratan igual: son evidencia de detector dudoso,
no razon suficiente para relanzar otro agente si el ACK y los tests estan
verdes.

ACK ausente, ACK no completado, tests obligatorios ausentes o tests fallidos
siguen bloqueando cierre con issues compactos y
`recommended_action=block_closure`.

## BuildAutoprogrammingProgrammableWorkV0

Funcion pura que valida `AutoprogrammingRequestV0` y construye trabajo
programable compatible con el nucleo:

- devuelve `WorkProfileV0` y `WorkflowTaskV0` por grupo;
- conserva `write_set`, `required_tests`, `worktree_ref` y `branch_ref`;
- conserva objetivo, contexto, criterios, tests y reglas compactas declaradas
  por tarea dentro de `WorkflowTaskV0`/`WorkProfileV0`;
- compacta `title`, `summary`, objetivo y criterios antes de crear
  `WorkflowTaskV0`; el contexto largo debe viajar por refs opacas, no como
  payload durable masivo;
- usa refs opacas como `ContextRefs`, sin leer repositorios ni ejecutar nada;
- para varios grupos, particiona `write_set` por area normalizada y rechaza
  rutas sin area;
- si una ruta coincide con varias areas compatibles, devuelve
  `partition.repairs` y secuencia las tareas con `depends_on`;
- si un trabajo vivo activo solapa el `write_set`, mantiene la tarea generada
  pero la marca como `postponed` mediante `depends_on` al ref vivo;
- corta refs vivos imposibles y rutas inseguras en `live_works.write_set`.
- si el core rechaza un `WorkProfileV0`/`WorkflowTaskV0`, conserva el subcampo
  causal en el issue publico para que el director pueda reparar la forma.

## BuildAutoprogrammingSelfImprovementRequestV0

Entrada pura para que un director o agente convierta un fallo observado en una
automejora programable de segundo plano.

- exige `project_ref`, resumen del fallo, worktree/rama opacas, worktree
  aislada, `suggested_write_set` propio y tests requeridos;
- devuelve `AutoprogrammingRequestV0` validada y `priority_score=10` por
  defecto, salvo prioridad explicita;
- conserva `source_run_ref`, `source_task_ref`, `context_refs` y
  `evidence_refs` como contexto opaco;
- si faltan refs o tests, no descarta la evidencia: devuelve acciones para
  completar worktree aislada, estudiar write-set o definir pruebas antes de
  preparar run.

Invariantes:

- no ejecuta agentes, no encola, no lee repositorios y no toca disco;
- no bloquea el trabajo principal que produjo la observacion;
- la preparacion real queda en el adaptador que ya usa
  `orquesta.autoprogramming.prepare_run.v0`.
