# Incidencia: server startup lifecycle puro expone stale tras readiness

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260701-091`.

## Sintoma

Un servidor podia publicar `startup_ready=true` y morir poco despues. La espera
de arranque ya cubre el caso en `cmd/orquesta-server` con
`server_exited_after_readiness`, pero faltaba una regresion focal en el modulo
puro que fijase la proyeccion durable/publica de `server_process_stale`.

## Cierre parcial

Sin tocar `cmd/orquesta-server`, se fija el contrato puro:

- `MarkServerProcessStaleStateV0` convierte un snapshot `running` +
  `startup_ready` en `status=stale`, `startup_ready=false` y
  `startup_status=server_process_stale`.
- El estado conserva causa operativa y evidencia en `last_error`,
  `startup_operational_message`, `last_error_operational_message` y
  `recent_errors`.
- `NewServerReadinessV0` y `NewServerPublicStatusV0` proyectan
  `server_process_stale`, sin dejar falso verde `running/startup_ready`.

## Residual

Sigue fuera de este cierre la limpieza completa de backends Goal huerfanos si el
HTTP desaparece y no hay owner recuperable suficiente. Ese tramo queda en el
residual operativo de `BUG-ORQ-20260701-091`/shutdown Goal.

## Evidencia

Prueba focal:

```bash
go test -count=1 ./modulos/orquesta-server -run TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0
go test -count=1 ./modulos/orquesta-server
```
