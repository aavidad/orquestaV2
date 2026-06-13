# Tarea técnica: auto-replan por cuota Codex agotada

Fecha: 2026-06-13

## Objetivo

Garantizar que Orquesta detecta automáticamente cuota Codex agotada y continúa
sin intervención manual cuando vuelve la capacidad.

## Estado actual

Implementado en código y probado localmente:

- `codex_usage_accounting.json` con `quota.status = exhausted` se clasifica
  como `provider_quota_exhausted`.
- El progreso resultante marca `BudgetStatus = capacity_limited`.
- Director/scheduler ya tienen ruta probada para cerrar agente y replanificar.

Prueba ejecutada:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-orchestration-core
```

Aplicado en la instancia OPES `127.0.0.1:8792` después de terminar los agentes
activos. El endpoint `/api/v0/server/status` confirmó `startup_ready`.

## Pendiente operativo

1. Ejecutar smoke real con un runtime que deje `codex_usage_accounting.json`
   en `exhausted` y proceso parado sin ACK.
2. Verificar que el supervisor emite `capacity_limited`, el director cierra el
   agente y el scheduler genera replan/retry sin intervención humana.

## Evidencias relacionadas

- Runbook:
  `docs/runbooks/incidencia_codex_quota_auto_replan_2026-06-13.md`
- Código:
  `modulos/orquesta-runtime-codex-delivery/progress_failure_context_v0.go`
- Tests:
  `modulos/orquesta-runtime-codex-delivery/progress_failure_context_v0_test.go`
  `modulos/orquesta-runtime-codex-delivery/progress_source_stopped_v0_test.go`
