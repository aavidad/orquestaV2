# Incidencia: artifact_paths omitidos en recibo terminal goal-first

Fecha: 2026-07-02

## Estado

Avance acotado de BUG-085.

## Problema

La politica de cierre ya exige `artifact_paths`, pero un recibo terminal podia
declarar solo parte de las rutas materializadas dentro del `write_set`. Eso
dejaba artefactos reales sin trazabilidad causal en el resultado del goal.

## Cambio

- El scanner de materialized refs construye un manifest independiente de
  artefactos encontrados bajo el `write_set`.
- Si el `GoalWorkResultV0` terminal declara `artifact_paths` incompleto, emite
  `artifact_paths_omitted_materialized` con evidencias opacas por ruta omitida.
- `director.stats`, `autoprogramming/status` y `observe_goal` proyectan el
  issue con accion `repair_receipt`.

## Limites

La deteccion no sustituye enforcement de runtime: solo observa el filesystem
permitido por `write_set`. El bloqueo duro de rutas declaradas fuera de scope
sigue en `orquesta-goal`.

## Pruebas

- `TestStackGoalMaterializedRefsSourceV0DetectaArtifactPathsOmitidosEnReceiptTerminal`
- `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathsOmitidosV0`
- `TestMCPAutoprogrammingStatusExecutorV0ArtifactPathsOmitidosPideRepairReceiptV0`
- `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0ArtifactPathsOmitidosPideRepairReceipt`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
