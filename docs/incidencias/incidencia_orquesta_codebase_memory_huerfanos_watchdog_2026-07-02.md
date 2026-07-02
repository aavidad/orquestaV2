# Incidencia: codebase-memory-mcp huerfano fuera del broker

Fecha: 2026-07-02.

## Sintoma

Durante la sesion aparecio un proceso `/home/alberto/.local/bin/codebase-memory-mcp`
vivo sin consulta activa observable de Orquesta. El broker central ya exigia
lease/owner marker para los procesos que lanza Orquesta, pero no reconciliaba
procesos nacidos fuera de ese broker por sesiones Codex o MCP anteriores.

## Riesgo

Este patron puede repetir consumo de CPU por subagentes o sesiones manuales
aunque el broker central este bien configurado. Es deuda de tooling/operacion,
no de busqueda de codigo concreta.

## Cambio

- El watchdog de contexto lista procesos `codebase-memory-mcp` via `ps`.
- Compara PIDs contra owner markers de Orquesta y protege los procesos propios.
- Los procesos sin marker y con edad minima se publican como huerfanos.
- Por defecto solo observa. La parada requiere
  `ORQUESTA_CODEBASE_BROKER_WATCHDOG_STOP_ORPHANS=true`.
- La edad minima se configura con
  `ORQUESTA_CODEBASE_BROKER_WATCHDOG_ORPHAN_MIN_AGE_SECONDS`.

## Evidencia

Tests:

```bash
go test -count=1 ./cmd/orquesta-server -run 'Test(ServerCodeContextToolProcessGuard|ParseServerPSCodeContextToolProcesses|RunServerCodeContextToolWatchdogLoopAsync|ServerCodeContextToolWatchdog|ServerFileCodeContextToolOwner)'
```

## Residual

La parada automatica de huerfanos sigue siendo opt-in para no matar procesos
manuales utiles sin autorizacion operativa. El contrato canonico sigue siendo:
los agentes deben consultar Orquesta como broker central y no arrancar MCPs
propios.
