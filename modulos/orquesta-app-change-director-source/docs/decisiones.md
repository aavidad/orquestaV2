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
