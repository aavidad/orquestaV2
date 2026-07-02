# Incidencia: timeout local con checkpoint reciente goal-first

Fecha: 2026-07-02

## Estado

Avance acotado de BUG-073.

## Problema

Un goal OPES podia escribir `checkpoint_started.txt` y aun asi quedar
proyectado como `goal_active_timeout_backend_active` con accion inmediata de
replan. Ese diagnostico era demasiado agresivo para un backend que seguia
activo y que ya habia producido una evidencia recuperable de checkpoint.

## Cambio

`autoprogramming/status` distingue el caso:

- backend goal activo;
- estado local bloqueado por `codex_app_server_goal_active_timeout`;
- solo hay artefactos de checkpoint;
- consumo todavia por debajo del umbral de alto consumo.

En ese caso publica `active_timeout_checkpoint_recent`, severidad `info` y
accion `observe_goal_backend_wait_for_checkpoint`.

Si el consumo ya supera el umbral, conserva el bloqueo
`checkpoint_only_high_consumption`/replan estrecho.

## Pruebas

- `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRecienteNoReplanificaAun`
- `go test -count=1 ./modulos/orquesta-mcp`
