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
