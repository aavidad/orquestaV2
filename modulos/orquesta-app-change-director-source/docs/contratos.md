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
  `draft_content_block`, `summarize_*`, `review` o generico);
- exige validar el contrato externo de dominio junto a los criterios de
  aceptacion.
- para trabajos documentales, deja explicito que el agente no recibe contexto
  minimo sino un paquete de dominio suficiente, con fuentes y longitud cuando
  el contrato las aporte.
- para `draft_content_block`, declara que la unidad es editorial y amplia:
  bloque, subcapitulo o capitulo coherente segun lo haya decidido OPES; no un
  parrafo aislado sin continuidad.
- para `summarize_block`, `summarize_chapter`, `summarize_topic` y
  `create_exam_outline`, permite granularidad pequena porque son artefactos
  derivados y trazables.

`AppChangeExternalWorkV0.input_fields` es opcional y compatible hacia atras,
con la forma `[]DomainWorkFieldV0` ya usada por `orquesta-domain-work`: objetos
`{name,value,values,value_json}` normalizados. Para
`project_ref=opes` y `work_kind=draft_content_block`, los nombres esperados son:

- `syllabus_full`: temario completo;
- `outline`: esquema;
- `chapter_objective`: objetivo de capitulo;
- `block_position`: posicion del bloque, como texto compacto o `value_json`;
- `neighbor_context`: bloques vecinos o refs vecinas;
- `source_refs`: fuentes;
- `acceptance_criteria`: criterios de dominio;
- `target_length`: longitud esperada.

La fuente no interpreta esos campos ni conoce OPES. Solo proyecta `title`,
`summary`, `criteria` y `required_tests` para que el agente valide el paquete
por el contrato externo y no invente campos de dominio.

Aunque el comando del core se llame `create_microtask`, en trabajos externos
representa una unidad durable de workflow. Para OPES no implica partir un
temario de 50 folios en trozos minimos; esa granularidad la decide OPES desde
su plan editorial. Orquesta solo debe subdividir si el job excede contexto,
trazabilidad o capacidad de revision.

La fuente no ejecuta el review gate. Solo abre la fase `revision`; la
validacion de ficheros, pruebas, write-set y tamano pertenece al proveedor de
observaciones de review gate inyectado en el stack.
