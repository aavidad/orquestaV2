# Pruebas de revisiones v0

```text
Caso: request_review_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RequestReview` solo con fase `revision` activa y entrega ya registrada, emite `ReviewRequested`, no emite outbox y rechaza fase no actual, entrega ausente o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No ejecuta revisor; la seleccion/ejecucion pertenece a comandos de capacidad/agente o adaptadores futuros.
```

```text
Caso: review_requested_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `review_request_id` en `reviews` sin duplicar solo si la entrega citada existe, y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el resultado completo de revision.
```

```text
Caso: request_review_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `RequestReview` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```

```text
Caso: accept_review_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `AcceptReview` solo con fase `revision` activa, revision solicitada y entrega ya registradas, emite `ReviewAccepted`, no emite outbox y rechaza fase no actual, revision ausente, entrega ausente o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No cierra tarea ni fase; esas transiciones quedan separadas.
```

```text
Caso: review_accepted_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `accepted_review_ref` en `accepted_reviews` sin duplicar solo si existen `review_request_id` y `delivery_ref`, y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el resultado completo de revision.
```

```text
Caso: accept_review_no_rompe_request_review
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `RequestReview` sigue produciendo `ReviewRequested` tras registrar `AcceptReview` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
