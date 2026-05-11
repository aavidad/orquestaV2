# Pruebas de brainstorming v0

```text
Caso: request_brainstorm_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RequestBrainstorm` solo con fase `brainstorming_arquitectura` actual activa, emite `BrainstormRequested`, no emite outbox y rechaza fase no actual o payload con detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: Brainstorm no arranca agentes; capacity/agent_launcher se solicitan por comandos posteriores.
```

```text
Caso: brainstorm_requested_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `brainstorm_request_id` en `brainstorms` sin duplicar y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no opciones completas de arquitectura.
```

```text
Caso: request_brainstorm_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `RequestBrainstorm` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
