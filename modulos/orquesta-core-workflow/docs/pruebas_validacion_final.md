# Pruebas de validacion final v0

```text
Caso: register_final_validation_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RegisterFinalValidation` con fase `validacion_final` activa y, si `closed_task_ref` llega informado, exige que ya este proyectado en `closed_tasks`; emite `FinalValidationRegistered`, no emite outbox y rechaza fase no actual, tarea cerrada ausente en runs con microtareas o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No debe cerrar fase ni run; esas transiciones siguen separadas.
```

```text
Caso: register_final_validation_run_level_sin_microtareas
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestHandleRegisterFinalValidationCommandV0AceptaRunSinMicrotareas
Evidencia esperada: `HandleCommandV0` acepta `RegisterFinalValidation` con `closed_task_ref` vacio solo cuando la run no tiene tareas ni tareas cerradas, proyecta `validation_ref` y conserva outbox vacio.
Ultima ejecucion: 2026-06-25, ok.
Riesgos: No debe relajar runs con microtareas; si existen tareas o `closed_tasks`, la validacion final debe seguir citando una tarea cerrada causal.
```

```text
Caso: final_validation_registered_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `validation_ref` en `validations` sin duplicar cuando existe `closed_task_ref` cerrado o cuando `closed_task_ref` esta vacio en una run sin microtareas; `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
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
