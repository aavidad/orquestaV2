# Pruebas de votaciones v0

```text
Caso: request_vote_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `RequestVote` solo con fase `votacion_y_decision` actual activa, emite `VoteRequested`, no emite outbox y rechaza fase no actual o payload con detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La solicitud de voto no acepta la decision final; `AcceptDecision` queda separado.
```

```text
Caso: vote_requested_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `vote_request_id` en `votes` sin duplicar y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no papeletas completas ni razonamientos.
```

```text
Caso: request_vote_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `RequestVote` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
