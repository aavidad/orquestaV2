# Decisiones

## Metadata refs del cambio como context refs

Decision: `AppChangeRequestV0.metadata_refs` se proyecta en
`create_microtask.task.context_refs`.

Motivo: las composiciones externas pueden necesitar transportar refs opacas de
contexto hasta la tarea creada sin convertirlas en campos del core.

Impacto: la fuente solo copia refs compactas. No interpreta detalles de
composicion, runtime, proveedor ni estado local.

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

## Granularidad editorial de OPES

Decision: En OPES, `draft_content_block` no se sobreatomiza. Aunque el core
mantenga el nombre historico `create_microtask`, la tarea creada representa una
unidad editorial durable: bloque, subcapitulo o capitulo coherente segun lo
haya decidido OPES. Los trabajos `summarize_*` y `create_exam_outline` si pueden
ser pequenos porque son derivados y trazables.

Motivo: para temarios de 45-50 folios, partir la redaccion en fragmentos
minimos degrada continuidad, estructura, tono y trazabilidad. El problema no se
arregla lanzando mas agentes sobre parrafos aislados; se arregla dando a cada
agente un paquete editorial suficiente y una unidad con sentido.

Alternativas descartadas: forzar una microtarea por parrafo, pedir al agente un
tema completo de 50 folios sin paquete editorial, o dejar que Orquesta decida
la granularidad de OPES sin contexto de dominio.

Impacto: la fuente anade criterios y pruebas para validar granularidad
editorial coherente. Orquesta solo debe pedir division adicional a OPES si el
job excede contexto, trazabilidad o capacidad de revision.

## Planificacion documental como contrato propio

Decision: `plan_tema`, `plan_temario` y `plan_documento` no se tratan como
redaccion ni como trabajo generico. La fuente crea una tarea de planificacion
que debe devolver `artifact_type=document_plan` compatible con
`DomainDocumentPlanV0`.

Motivo: OPES no tiene juicio propio; Orquesta/director debe pensar el plan antes
de pedir redaccion, revision, visuales, ensamblado o exportacion. Si el plan
entra como `work_delivery` generico, no hay forma fiable de validar secciones,
entregables, criterios de calidad ni pasos posteriores.

Impacto: la tarea exige secciones, entregables, criterios y revisiones
ejecutables, y declara explicitamente que no se redacta el documento final en
esa fase. OPES conserva su dominio y Orquesta conserva el juicio de
orquestacion.

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

## Criterios compactos

Decision: La fuente compacta criterios de microtarea a un maximo de 10.

Motivo: el DTO validado del director limita listas compactas para mantener
contexto pequeno. Un job OPES puede sumar criterios estructurales de Orquesta,
criterios de dominio y criterios de usuario; emitir 11 o mas crea una decision
invalida y bloquea el run antes de lanzar el agente.

Impacto: se conservan primero los criterios estructurales generados por
Orquesta y despues los primeros criterios del solicitante hasta el limite.

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

## Visuales OPES como trabajo externo

## Readiness alineado con rails comunes

Decision: La fuente deja de mantener una lista local de vocabulario prohibido
para autoplanning. El readiness usa la politica comun de `orquesta-rails` por
campo y solo corta detalle sensible efectivo o material crudo, como
`api_key=`, `client_secret=`, `authorization: Bearer`, DSN con credenciales,
rutas privadas, `prompt=` o `transcript=`.

Motivo: `runtime`, `provider`, `model`, `db`, `sql`, `codex`, `docker` y
`home` pueden ser refs opacas, nombres de modulo o criterios operativos validos.
Bloquearlos por substring impedia cambios reales y ocultaba la causa de dominio
al director.

Impacto: los criterios se compactan, pero no sustituyen terminos de dominio ni
vocabulario operativo normal. Si aparece detalle sensible efectivo, no se crea
microtarea automatica y el director conserva la consulta pendiente.

Decision: `generate_visual_asset` publica el mismo contrato externo
`ApplyExternalDomainWorkV0`, pero la tarea declara criterios especificos de
visual pedagogico.

Motivo: OPES quiere tratar esquemas, vinetas, flujogramas e infografias como
artefactos de dominio. Si Orquesta lo tratase como bloque textual generico, el
agente no tendria criterios de seguridad SVG, accesibilidad ni trazabilidad.

Impacto: la fuente no conoce ficheros ni persistencia OPES. Solo proyecta
title, summary, acceptance criteria y required tests para que el agente entregue
`visual_asset` por el bridge de dominio externo.

## Perfiles neutrales de trabajo

Decision: La fuente marca las microtareas de cambio con `work_profile_kind`.

Motivo: el scheduler ya resuelve rol y capacidad desde perfiles neutrales. Si
la fuente deja el campo vacio, los trabajos externos se verian como
implementacion por compatibilidad legacy aunque realmente sean trabajo de
dominio.

Impacto: cambios internos usan `implementation`; cualquier `external_work` usa
`domain_work`. La fuente no decide proveedor/modelo/runtime ni interpreta
internals de OPES u otra app.

## Saneado antes de bloquear autoplaneado

Decision: La fuente sanea criterios operativos antes de decidir si un cambio
aceptado puede convertirse en microtarea automatica.

Motivo: palabras como `token economy`, `Codex`, `modelo` o `provider` pueden
aparecer en reglas operativas de usuario sin que deban cortar todo el trabajo.
El comportamiento correcto es normalizarlas a conceptos neutrales y dejar al
director/agente trabajar con refs opacas y contratos.

Impacto: se mantiene la frontera hexagonal porque la microtarea no transporta
proveedor, runtime ni secretos concretos. Tambien se evita el falso quiescent:
un cambio con write-set y criterios verificables no queda parado solo por una
palabra saneable.

## Required tests externos compactos

Decision: La fuente compacta `required_tests` antes de construir
`create_microtask`. Si una prueba externa llega como comando inline demasiado
largo, la microtarea conserva el requisito como marcador verificable:
`validar required_tests externos declarados en paquete de dominio`.

Motivo: el DTO del director exige textos operativos compactos. Un comando
Python largo procedente de OPES puede ser valido como prueba de dominio, pero
no como campo directo de una decision del director. Antes de esta decision, una
run aceptada por `external-work/run` podia bloquearse en la fuente del director
con `tasks_total=0`.

Impacto: el trabajo no se pierde ni se lanza sin validacion. El agente recibe
un requisito compacto y debe ejecutar o documentar las pruebas declaradas en el
paquete de dominio externo. La app propietaria conserva el detalle completo en
su contrato/payload.

## External work accionable sin criterios explicitos

Decision: un `external_work` con `user_intent` y `work_kind`, `job_ref` o
`work_refs` puede generar microtarea aunque `acceptance_criteria` llegue vacio.
Los criterios se derivan del contrato externo y de la clase de trabajo.

Motivo: algunas apps envian el contrato estructurado en `external_work`,
`input_fields`, `constraints` y `required_tests`. Aceptar el transporte pero
exigir criterios libres para autoplanning dejaba runs vivas, sin tarea y sin
agente.

Impacto: se mantiene la seguridad para cambios internos: si solo hay
`allowed_write_set` y faltan criterios explicitos, no se inventa microtarea.
La excepcion es exclusiva de trabajo externo accionable.
