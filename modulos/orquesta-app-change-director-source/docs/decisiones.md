# Decisiones

## Cambio en programacion mantiene la fase actual

Decision: Si el run ya esta en `programacion` y existe una decision base, la
fuente de cambios crea contrato y microtarea directamente en la fase actual.

Motivo: un cambio de usuario a mitad de programacion no debe reiniciar todo el
flujo de brainstorming/planificacion si trae write-set y criterios concretos.
Tampoco debe esperar a que terminen todos los agentes activos: el director debe
poder incorporar trabajo incremental y dejar que el scheduler aplique claims.

Alternativas descartadas: reabrir brainstorming para cualquier cambio,
bloquear hasta que no queden agentes en vuelo, o lanzar agentes desde el
transporte web sin decision del director.

Impacto: la fuente conserva el flujo completo para fases tempranas, pero en
programacion emite `answer_director_question`, `publish_function_contract` y
`create_microtask` con refs compactas y sin conocer runtime, DB ni proveedor.

## Cambio a microtarea solo si es concreto

Decision: La fuente automatica solo crea microtarea cuando el cambio trae
write-set y criterios de aceptacion.

Motivo: Sin esos datos el sistema estaria inventando alcance. Eso ya fue una
fuente de bucles en las versiones anteriores.

Alternativas descartadas: convertir cualquier texto libre en trabajo, usar
rework de revision o meter la logica en el stack.

## Cambios de programacion declaran pruebas obligatorias

Decision: Las microtareas creadas desde `AppChangeRequestV0` siempre incluyen
`required_tests`; si el write-set parece Go se antepone `go test ./...`.

Motivo: el contrato comun del director ya no acepta programacion sin pruebas
obligatorias. Un cambio a mitad de app no puede saltarse una regla de calidad
que si se exige al bootstrap inicial.

Alternativas descartadas: dejar tests vacios en cambios, relajar el validador
para cambios, o inferir pruebas en el runtime despues de haber lanzado agentes.

## Trabajo externo como contrato de dominio

Decision: Si un cambio trae `external_work`, la fuente publica
`ApplyExternalDomainWorkV0` y crea una microtarea de dominio externo, pero no
lee la app externa ni conoce sus rutas reales.

Motivo: OPES y futuras apps de dominio deben poder pedir trabajo a Orquesta sin
que Orquesta incorpore su nucleo ni repita gestion propia de agentes dentro de
cada app.

Alternativas descartadas: integrar OPES como modulo interno de Orquesta,
mantener una rama especial por app externa o convertir cualquier texto libre en
microtarea sin contrato.

Impacto: la fuente solo consume refs compactas ya guardadas en
`AppChangeRequestV0.external_work`; las decisiones siguen validadas por el
contrato comun del director.

## Paquete documental externo suficiente

Decision: Para `draft_content_block` y trabajos documentales, la fuente declara
en `title`, `summary`, `criteria` y `required_tests` que el agente debe resolver
el contrato con un paquete de dominio suficiente, no con contexto minimo.

Motivo: OPES puede entregar temario completo, esquema, objetivo de capitulo,
posicion del bloque, vecinos, fuentes, criterios y longitud. Si esos campos
llegan por `input_fields`, la microtarea debe exigir que se usen y que no se
inventen valores ausentes.

Alternativas descartadas: esconder esa expectativa en la capa de ejecucion,
acoplar la fuente al nucleo de OPES o relajar los criterios de cierre de
trabajos documentales.

Impacto: mientras `AppChangeExternalWorkV0.input_fields` no exista, la fuente
mantiene compatibilidad y no lo lee. Cuando exista, el contrato esperado es una
lista opcional de campos `{name,value,values}` con nombres de dominio estables.

## Trabajo externo sin write-set local

Decision: Un `external_work` puede crear microtarea aunque
`allowed_write_set` este vacio. En ese caso la fuente deriva `write_set` como
scopes externos opacos, por ejemplo `external/opes/draft_content_block` y
`external/opes/opes-job-001`.

Motivo: OPES no debe mandar rutas locales de Orquesta ni fingir un write-set de
codigo para pedir un trabajo editorial. La frontera correcta es una ref externa
compacta del job/topic/chapter de OPES; Orquesta la usa como scope de
coordinacion y mantiene fuera los internals de OPES.

Alternativas descartadas: obligar a OPES a enviar rutas locales, relajar el
validador comun para permitir microtareas sin `write_set`, o meter casos
especiales de OPES en el core.

Impacto: los cambios de codigo siguen exigiendo `allowed_write_set`; solo el
trabajo externo con `external_work` obtiene scopes derivados. La microtarea
sigue pasando por contrato, criterios de aceptacion y pruebas obligatorias.

## Revision del cambio entregado

Decision: La fuente de cambios abre `revision` cuando la microtarea del cambio
esta en el run y el conjunto de entregas cubre las tareas de programacion.

Motivo: el review gate ya sabe validar ACK, ficheros reales, write-set, tests y
tamano. El hueco estaba antes del gate: un cambio entregado podia quedarse en
`programacion` si no habia una decision explicita posterior del director.

Alternativas descartadas: ejecutar review gate desde `app-change`, cerrar el
cambio al registrar la entrega, o abrir revision desde el transporte web/MCP.

Impacto: la decision nueva es solo `open_phase(revision)` con refs compactas.
No lee ficheros, runtime, DB, proveedor, modelo ni detalles de la app externa.
Si hay otro cambio listo para crear microtarea, se prioriza materializarlo antes
de abrir revision para no revisar un conjunto de tareas incompleto.
