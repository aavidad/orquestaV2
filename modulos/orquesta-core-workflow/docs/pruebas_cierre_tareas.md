# Pruebas de cierre de tareas v0

```text
Caso: close_task_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `CloseTask` solo con fase `revision` activa, `task_id` ya proyectado en `tasks`, `delivery_ref` ya proyectada en `deliveries` y `accepted_review_ref` ya proyectada en `accepted_reviews`; emite `TaskClosed`, no emite outbox y rechaza fase no actual, tarea ausente, entrega ausente, revision aceptada ausente o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No debe cerrar fase ni run; esas transiciones siguen separadas.
```

```text
Caso: task_closed_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `task_id` en `closed_tasks` sin duplicar solo si existen `task_id`, `delivery_ref` y `accepted_review_ref`; `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el resultado completo de entrega o revision.
```

```text
Caso: close_task_no_cierra_fase_ni_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: Tras aplicar `TaskClosed`, `current_phase` sigue siendo `revision`, la fase no queda cerrada, el run no queda cerrado y no se generan mensajes de outbox.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: El cierre de fase debe seguir pasando solo por `ClosePhase`.
```
