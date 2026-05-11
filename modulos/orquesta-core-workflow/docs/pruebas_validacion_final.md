# Pruebas de validacion final v0

```text
Caso: register_final_validation_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RegisterFinalValidation` solo con fase `validacion_final` activa y `closed_task_ref` ya proyectada en `closed_tasks`; emite `FinalValidationRegistered`, no emite outbox y rechaza fase no actual, tarea cerrada ausente o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No debe cerrar fase ni run; esas transiciones siguen separadas.
```

```text
Caso: final_validation_registered_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `validation_ref` en `validations` sin duplicar solo si existe `closed_task_ref`; `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el resultado completo de validacion.
```

```text
Caso: register_final_validation_no_cierra_fase_ni_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: Tras aplicar `FinalValidationRegistered`, `current_phase` sigue siendo `validacion_final`, la fase no queda cerrada, el run no queda cerrado y no se generan mensajes de outbox.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: El cierre de fase debe seguir pasando solo por `ClosePhase` y el cierre de run por una transicion separada.
```
