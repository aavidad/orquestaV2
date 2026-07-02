# Incidencia: goal-first cerraba sin auditar rutas de artefactos contra write-set

Fecha: 2026-07-02

## Sintoma

Un goal podia declarar `complete` con `artifact_refs` opacas aunque hubiera
materializado ficheros fuera del `write_set` autorizado. El cierre neutral solo
veia refs, no rutas observadas, por lo que no podia bloquear una salida fuera de
contrato.

## Cierre aplicado

- `GoalWorkResultV0` incorpora `artifact_paths`.
- `ValidateGoalWorkResultV0` valida que esas rutas sean relativas y seguras.
- `ValidateGoalWorkClosureV0` bloquea con
  `goal_artifact_path_out_of_scope` si alguna ruta observada no es exactamente
  un scope del `write_set` ni descendiente suyo.
- El adaptador Codex Goal transporta `artifact_paths` desde el receipt.
- `cmd/orquesta-server` acepta `artifact_paths` en el marcador/JSON terminal y,
  si observa un `orquesta_goal_result_v0.json` durable, anade la ruta relativa
  del propio fichero como evidencia auditable.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./cmd/orquesta-server
```

## Residual

Este cierre audita rutas declaradas/observadas por el runtime. La deteccion
exhaustiva de todos los ficheros escritos por un proceso externo sigue siendo
responsabilidad de la composicion/runtime que tenga visibilidad del filesystem.
