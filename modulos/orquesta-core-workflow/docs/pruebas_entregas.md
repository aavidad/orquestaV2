# Pruebas de entregas v0

```text
Caso: register_delivery_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RegisterDelivery` solo con fase
`programacion` activa, tarea ya creada y agente arrancado, emite
`DeliveryRegistered`, no emite outbox y rechaza fase no actual, tarea/agente
ausentes, agentes fallidos, agentes con parada confirmada o detalles
prohibidos. Una parada solicitada sin confirmacion aun admite entrega tardia
causal.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No solicita revision; `RequestReview` debe quedar en otro corte.
```

```text
Caso: delivery_registered_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `delivery_ref` en `deliveries` sin duplicar solo si tarea y agente citados existen, y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el detalle completo de implementacion.
```

```text
Caso: register_delivery_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `RegisterDelivery` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
