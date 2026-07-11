# Incidencia 208AF: observer bloquea irreversiblemente un goal que termina con resultado durable

Fecha: 2026-07-11
ID: BUG-ORQ-20260711-208AF
Estado: abierto
Area: autoprogramacion goal-first / observacion Codex app-server / reconciliacion terminal

## Resumen

El smoke local completo el goal externo con resultado durable, pero el observer
ya lo habia persistido como `blocked` por
`goal_active_no_checkpoint_high_consumption`. La observacion posterior recibio
HTTP 504, conservo ese bloqueo y no ingirio el resultado terminal del backend.

Esto contradice la evidencia disponible: habia checkpoint temprano material y
el goal termino `complete`. El alto consumo no debe convertir una observacion
transitoria en un terminal irreversible cuando despues existe autoridad
durable de backend.

## Evidencia retenida

Raiz del smoke: `/tmp/orquesta-cleanup-goal-20260711`.

- Run: `autoprog-cleanup-deadcode-appserver-20260711`.
- Goal externo: `019f4f10-0c7f-7421-ae18-143dc6482642`.
- Routing confirmado: Terra con esfuerzo `medium`.
- El trabajo produjo un diff de 156 lineas borradas.
- El backend completo el goal, dejo resultado durable y registro
  `202292` tokens usados.
- Cerca de 100k tokens, el observer persistio `blocked` con
  `goal_active_no_checkpoint_high_consumption`, pese a la evidencia de
  checkpoint temprano.
- El `observe` HTTP posterior devolvio 504, conservo `blocked` y no reconcilio
  ni ingirio el resultado terminal durable.

## Criterio de cierre

- El checkpoint/progreso material temprano se reconoce en la observacion.
- Alto consumo queda como aviso o replan recuperable; no como terminal
  irreversible por si solo.
- Un terminal de backend con resultado durable puede reconciliar un `blocked`
  transitorio anterior.
- Pruebas focales y el smoke aceptan el goal terminado y su resultado durable.
