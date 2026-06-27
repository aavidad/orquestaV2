# Decisiones locales: orquesta-external-work-run

## Decisiones tomadas

```text
Fecha: 2026-05-13
Decision: Crear un caso de uso operativo separado para trabajos externos ya
definidos.
Motivo: OPES y otras apps propietarias pueden enviar un job completo; arrancar
`/api/v0/apps/director` mezcla planificacion de una app nueva con ejecucion de
trabajo externo y deja directores LLM sobrantes.
Alternativas: reutilizar request_change sobre un run creado por director;
ignorar el director sobrante en el smoke; acoplar OPES al stack. Se descartan
porque repiten el fallo de v1/v2: flujos confusos y parches sobre sintomas.
Impacto: `StartExternalWorkRunV0` crea run, abre `programacion`, registra
`AppChangeRequestV0` y encola por puerto. El director determinista de
`orquesta-app-change-director-source` planifica la microtarea en el siguiente
drain.
Estado: aceptada.
```

```text
Fecha: 2026-06-27
Decision: Anadir compilacion neutral external-work -> GoalWorkSpecV0 sin
arrancar runtime.
Motivo: con Codex Goal, Orquesta no debe forzar el loop historico para trabajos
externos ya definidos. La composicion necesita un contrato Goal-first que
mantenga reglas, write-set, tests, artefactos y cierre por evidencias sin meter
OPES ni proveedor en este modulo.
Alternativas: lanzar /runs/supervise como camino normal; meter OPES directo en
external-work-run; crear otro DTO paralelo a DomainWork. Se descartan porque
preservan el loop viejo, acoplan dominio o duplican semantica.
Impacto: `BuildExternalWorkGoalWorkSpecV0` reutiliza AppChange/DomainWork,
produce `GoalWorkSpecV0` validable con `director_kind=runtime_goal` y deja el
lanzamiento/observacion del Goal a una composicion opt-in posterior.
Estado: aceptada.
```

```text
Fecha: 2026-05-13
Decision: La cola forma parte del caso de uso.
Motivo: en servidor residente, guardar el run no basta; el supervisor global
solo avanza runs presentes en RunQueue. Dejar la cola al smoke volveria a meter
orquestacion manual fuera de Orquesta.
Alternativas: ejecutar un drain sincrono en el endpoint; pedir al cliente que
llame a run_queue/priority; depender de ticks manuales. Se descartan para no
bloquear HTTP y para mantener autonomia del servidor.
Impacto: el use case requiere `RunQueuePriorityWriterPortV0` y escribe una
prioridad inicial idempotente.
Estado: aceptada.
```
