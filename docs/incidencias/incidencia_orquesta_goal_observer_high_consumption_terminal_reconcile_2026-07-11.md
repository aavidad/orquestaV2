# Incidencia 208AF: observer bloquea irreversiblemente un goal que termina con resultado durable

Fecha: 2026-07-11
ID: BUG-ORQ-20260711-208AF
Estado: cerrado en codigo local y replay real aislado
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
- Pruebas focales y replay real ingieren el goal terminado y su resultado
  durable. La aceptacion final sigue sometida a la atestacion independiente;
  no se fuerza un verde si aparece otro fallo causal.

## Cierre 2026-07-11

Commit: `c0e728fc3` (`fix: reconcilia recibos canonicos de goals`).

- El observer futuro ya no persiste `blocked` irreversible por alto consumo:
  el corte `b0755fc5b` conserva `running` y publica aviso/replan recuperable.
- El reconciliador y el watcher leen primero el recibo canonico derivado en
  `.orquesta-runtime/goal-receipts/<goal>/`, fuera del write-set de producto.
- El recibo canonico exige fichero regular, path resuelto dentro del proyecto,
  resultado terminal valido y `goal_ref` exacto. Un canonico invalido no cae a
  un recibo legacy ni activa reparacion sintetica; un canonico valido tiene
  precedencia sobre sidecars legacy.
- `LoadTerminalGoalMaterializedResultV0` y
  `ResolveDirectorGoalMaterializedRefsV0` comparten esa precedencia.

Verificacion:

- `go test -count=1 ./modulos/orquesta-app-codex-stack`: verde.
- `go test -race -count=1 ./modulos/orquesta-app-codex-stack`: verde.
- Replay sobre copia del estado real en
  `/tmp/orquesta-208af-replay-20260711-4`: el store paso de version 12
  `blocked/goal_active_no_checkpoint_high_consumption` a version 14
  `complete`, con el summary y la evidencia del resultado durable canonico.
  El cierre quedo `blocked/requires_rework` por falta de atestacion independiente
  confiable, no por alto consumo. Es el comportamiento correcto ante el falso
  verde separado `BUG-ORQ-20260711-208AG`.
