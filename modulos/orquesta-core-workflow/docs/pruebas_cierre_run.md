# Pruebas de cierre de run v0

```text
Caso: close_run_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `CloseRun` solo con fase `cierre` activa y `validation_ref` ya proyectada en `validations`; emite `RunClosed`, no emite outbox y rechaza fase no actual, validacion ausente o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No debe ejecutar conectores ni tocar runtime, DB, provider/proveedor, OAuth o HOME.
```

```text
Caso: run_closed_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `closure_ref` en `closures` sin duplicar solo si existe `validation_ref`; marca `status` del run como `cerrado`; `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el resultado completo de cierre.
```

```text
Caso: close_run_no_cierra_fase_automaticamente
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: Tras aplicar `RunClosed`, `current_phase` sigue siendo `cierre`, el estado de la fase no cambia, el run queda `cerrado` y no se generan mensajes de outbox.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La regla local mantiene `ClosePhase` como transicion separada; una implementacion futura no debe acoplar ambos cierres.
```

```text
Caso: flujo_comandos_entrega_hasta_cierre_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow
Evidencia esperada: Desde un historial previo a entrega, `RegisterDelivery`, `RequestReview`, `AcceptReview`, `CloseTask`, `RegisterFinalValidation` y `CloseRun` se encadenan por handler/reducer, no emiten outbox en el flujo de cierre y el replay durable reconstruye `status=cerrado`, fase `cierre` y refs compactas proyectadas.
Ultima ejecucion: 2026-05-05, ok.
Riesgos: Las aperturas de fase siguen siendo transiciones explicitas con `OpenPhase`; no hay cierre automatico de fase ni dispatcher de outbox.
```
