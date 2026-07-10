# orquesta-autonomy-program

Contrato puro para un programa padre durable, con nodos de goal/tarea y un DAG
tipado. `project_ref` y `root_ref` son obligatorios. El programa, no su primer
goal, es el agregado y el dueño del cierre.

`PrepareAutonomyProgramFrontierV0` solo emite la frontera dependiente lista:
los write-sets disjuntos salen juntos y un solape se serializa. Persistir el
resultado antes de llamar al actuador hace el replay idempotente. El puerto de
actuador no es `runs/control`.

`ClaimAutonomyProgramFrontierV0` realiza ese persist-before-act mediante CAS:
solo el scheduler que gana el snapshot recibe lanzamientos. Tras un crash,
`RecoverAutonomyProgramActionsV0` reconstruye la misma accion con el mismo
`launch_ref` o `receipt_ref`; el adaptador de actuacion debe deduplicar esas
refs porque un efecto externo exactly-once no puede garantizarse desde este
contrato puro.

`OperatorTaskV0` / `OperatorReceiptV0` representan `wait_external` y resume
tipados. Una solicitud de operador no cierra nada; solo el receipt causal
continua el lanzamiento ya persistido. Las dependencias externas declaradas se
resuelven mediante `ExternalDependencyReceiptV0` durable sin mutar el DAG. Un
cierre exige receipt causal y atestaciones de todos los tests requeridos;
`accepted` exige que sean `passed`.
