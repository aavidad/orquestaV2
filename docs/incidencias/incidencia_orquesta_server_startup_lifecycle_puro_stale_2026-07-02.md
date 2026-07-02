# Incidencia: server startup lifecycle puro expone stale tras readiness

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260701-091`.

## Sintoma

Un servidor podia publicar `startup_ready=true` y morir poco despues. La espera
de arranque ya cubre el caso en `cmd/orquesta-server` con
`server_exited_after_readiness`, pero faltaba una regresion focal en el modulo
puro que fijase la proyeccion durable/publica de `server_process_stale`.

## Cierre

Se fija el contrato puro y el cleanup acotado de la composicion `cmd`:

- `MarkServerProcessStaleStateV0` convierte un snapshot `running` +
  `startup_ready` en `status=stale`, `startup_ready=false` y
  `startup_status=server_process_stale`.
- El estado conserva causa operativa y evidencia en `last_error`,
  `startup_operational_message`, `last_error_operational_message` y
  `recent_errors`.
- `NewServerReadinessV0` y `NewServerPublicStatusV0` proyectan
  `server_process_stale`, sin dejar falso verde `running/startup_ready`.
- `waitForStateHealthyV0` devuelve `server_exited_after_readiness` y ejecuta
  cleanup del backend Goal `app_server_tmux` configurado cuando el PID del
  daemon ya no vive.
- La limpieza de sesiones sin owner marker queda limitada por la guarda ya
  existente de nombre `orquesta-goal-*` y socket propio configurado.

## Fuera de alcance

No se convierte en regla del core puro la limpieza de procesos externos no
configurados o sin ownership suficiente.

## Evidencia

Prueba focal:

```bash
go test -count=1 ./modulos/orquesta-server -run TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0
go test -count=1 ./modulos/orquesta-server
go test -count=1 ./cmd/orquesta-server -run 'TestWaitForStateHealthyV0MarcaStaleSiServidorMuereTrasReadinessV0|TestWaitForStateHealthyV0LimpiaGoalBackendConfiguradoSiMuereTrasReadinessV0|TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0'
```
