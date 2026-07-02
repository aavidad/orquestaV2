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

El umbral ya no queda fijado solo en el MCP: la composicion `cmd/orquesta-server`
publica `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS` en
`effective_config`, lo normaliza como politica de progreso y lo inyecta en
`autoprogramming/status` y `runs/control`. El default conserva `100000` tokens.

## Pruebas

- `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRecienteNoReplanificaAun`
- `TestMCPAutoprogrammingStatusExecutorV0GoalActiveTimeoutConCheckpointRespetaUmbralConfiguradoV0`
- `TestMCPRunControlExecutorV0StopForcedRespetaUmbralConfiguradoV0`
- `TestBuildStackV0PropagaDiagnosticosAutoprogramacionABindingsV0`
- `TestServerConfigFromEnvV0PublicaUmbralCheckpointGoalConfigurableV0`
- `go test -count=1 ./modulos/orquesta-mcp`
