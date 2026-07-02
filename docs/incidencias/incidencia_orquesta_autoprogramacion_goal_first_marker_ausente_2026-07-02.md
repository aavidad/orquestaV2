# Incidencia: autoprogramacion goal-first sin marker durable

Fecha: 2026-07-02

## Resumen

La ruta HTTP/app nueva goal-first persistia `GoalWorkStateV0` y
`GoalWorkRunMarkerV0`, pero la ruta de autoprogramacion goal-first solo
persistia `GoalWorkStateV0`.

Si el state se perdia, quedaba corrupto o no era visible tras restart, las
superficies de `status`, `director.stats`, shutdown y reparacion solo podian ver
un run goal-first parcialmente materializado. Esto favorecia diagnosticos
`goal_first_state_missing`, caidas al analisis legacy y ceguera operativa sobre
goals todavia vivos.

## Causa

Contrato goal-first duplicado entre rutas:

- `StartAppDirectorV0` ya escribia `State + Marker`.
- `PrepareAutoprogrammingRunV0` lanzaba el goal, guardaba state y devolvia
  resultado, pero no dejaba marker.

El marker no es un artefacto accesorio: es la evidencia durable minima para
reparar/reconstruir estado goal-first sin reabrir el loop legacy.

## Cierre aplicado

- `autoprogrammingBridgeStartSingleGoalFirstV0` guarda
  `GoalWorkRunMarkerV0` cuando existe `GoalFirstRunMarkerStore`.
- Los goals nuevos, multigoal y los fallos de launcher guardan marker junto al
  state.
- Los runs goal-first existentes con state pero sin marker reparan el marker de
  forma idempotente sin relanzar el goal.
- Si el marker configurado no puede persistirse, la preparacion no declara
  aceptado silenciosamente y devuelve
  `autoprogramming_goal_marker_save_failed`.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestPrepareAutoprogrammingRunV0(BackendGoalCompleto|GoalReadyMultiGoal|GoalReadyLaunchFailed|GoalReadyRunExistente)'
go test -count=1 ./modulos/orquesta-app-codex-stack
```

## Estado

Cerrado en codigo local. Pendiente de commit/push y sincronizacion remota junto
al siguiente lote verificado.
