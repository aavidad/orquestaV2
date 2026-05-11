# Pruebas de contratos de funcion v0

```text
Caso: publish_function_contract_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `PublishFunctionContract` solo con fase `planificacion_microtareas` actual activa y `decision_ref` ya proyectado, emite `FunctionContractPublished`, no emite outbox y rechaza decision ausente, fase no actual o detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No crea microtareas; `CreateMicrotask` sigue siendo una transicion independiente.
```

```text
Caso: function_contract_published_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `contract_ref` en `function_contracts` sin duplicar solo si la decision citada existe, y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el detalle completo de implementacion.
```

```text
Caso: publish_function_contract_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `PublishFunctionContract` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
