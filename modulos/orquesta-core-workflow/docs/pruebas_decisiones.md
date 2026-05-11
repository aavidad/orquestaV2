# Pruebas de decisiones v0

```text
Caso: accept_decision_handler_emite_evento_sin_outbox
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `HandleCommandV0` acepta `AcceptDecision` solo con fase `votacion_y_decision` actual activa y `vote_ref` ya proyectado, emite `ArchitectureDecisionAccepted`, no emite outbox y rechaza fase no actual, voto ausente o payload con detalles prohibidos.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: No abre planificacion; la apertura de fase sigue perteneciendo a `OpenPhase`.
```

```text
Caso: architecture_decision_accepted_reducer_y_replay
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `ApplyEventV0` proyecta `decision_ref` en `decisions` sin duplicar solo si el voto citado existe, y `ReplayDurableEventsV0` acepta duplicado exacto con sequence estable.
Ultima ejecucion: 2026-05-04, ok.
Riesgos: La proyeccion conserva refs compactas, no el acta completa de votacion.
```

```text
Caso: accept_decision_no_rompe_start_run
Tipo: regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow ./modulos/orquesta-core
Evidencia esperada: `StartRun` sigue produciendo `RunStarted` tras registrar `AcceptDecision` en routers de comando, evento y reducer.
Ultima ejecucion: 2026-05-04, ok.
```
