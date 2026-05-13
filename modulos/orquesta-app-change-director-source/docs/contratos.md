# Contratos

## `AppChangeDirectorDecisionSourceV0`

Entrada: `DirectorAgentDecisionSourceRequestV0` con el run actual y un
`AppChangeRecordSourcePortV0`.

Salida: decisiones `director_agent_decision.v0` ya validadas:

- `answer_director_question`;
- `open_phase` a `votacion_y_decision`;
- `request_vote`;
- `accept_decision`;
- `open_phase` a `planificacion_microtareas`;
- `publish_function_contract`;
- `create_microtask`;
- `open_phase` a `programacion`;
- `open_phase` a `revision` cuando la microtarea del cambio ya existe en el
  run y todas las tareas de programacion tienen entrega.

Si el cambio no tiene `allowed_write_set` o `acceptance_criteria`, la fuente no
inventa microtareas: deja la consulta pendiente para el director.

La fuente no diferencia si el `AppChangeRecordV0` procede de una solicitud
directa o de `AppChangeIntentEventV0`: solo exige store, pregunta de director
pendiente en el run, write-set y criterios.

Si `AppChangeRequestV0.external_work` esta presente, la fuente sigue sin
conocer la app externa. Solo cambia la proyeccion compacta:

- publica contrato `ApplyExternalDomainWorkV0`;
- titula la microtarea segun `work_kind` (`documentation`, `generation`,
  `review` o generico);
- exige validar el contrato externo de dominio junto a los criterios de
  aceptacion.

La fuente no ejecuta el review gate. Solo abre la fase `revision`; la
validacion de ficheros, pruebas, write-set y tamano pertenece al proveedor de
observaciones de review gate inyectado en el stack.
