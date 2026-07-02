# Incidencia: QA fallida estructurada en artefactos parciales goal-first

Fecha: 2026-07-02

## Estado

Avance acotado, no cierre total de BUG-072 ni BUG-075.

## Problema

Cuando un goal-first escribia artefactos recuperables pero un informe de QA
estructurado los marcaba como no publicables, la superficie publica podia quedar
en bloqueo generico (`goal_first_blocked` o `goal_first_blocked_no_artifacts`) y
sin accion focal de rework.

Ademas, tras exigir `artifact_paths` en el cierre goal-first, los cierres
external-work con receipt aceptado en ledger podian seguir bloqueados si el
resultado del backend traia `artifact_refs` pero no `artifact_paths`.

## Cambio

- El scanner de materialized refs detecta JSON de QA fallida dentro del
  `write_set` y emite `qa_failed_public_text` con evidencia opaca.
- En contexto OPES la deteccion usa la terna estructurada ya canonica
  (`extension`, `official_text`, `strict_editorial`); texto libre no se
  convierte en veto.
- `director.stats`, `autoprogramming/status` y `observe_goal` proyectan
  `qa_failed_public_text` como bloqueo recuperable con accion
  `rework_public_text`, preservando `artifact_refs`.
- El validador de receipts de external-work deriva `ArtifactPaths` desde el
  `file_ref` real persistido en el ledger aceptado; si no existe `file_ref`, no
  inventa ruta.

## Pruebas

- `TestStackGoalMaterializedRefsSourceV0DetectaQAFailedPublicTextEnContextoOPES`
- `TestMCPDirectorStatsToolExecutorV0GoalFirstProyectaQAFailedPublicTextV0`
- `TestMCPAutoprogrammingStatusExecutorV0QAFailedPublicTextPideReworkV0`
- `TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextPideRework`
- `go test -count=1 ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
