# Incidencia: codebase-memory-mcp no debe quedar vivo sin lease

Fecha: 2026-07-02

## Sintoma

Se observaron procesos `codebase-memory-mcp` vivos y consumiendo CPU sin una
consulta activa de Orquesta. El cierre manual confirmo que eran procesos
recuperables, pero faltaba una superficie persistente que guiara la accion del
operador o del bootstrap.

## Riesgo

Un agente o subagente puede dejar un indexador/MCP externo vivo fuera del broker
central. Eso produce consumo sostenido y puede confundirse con trabajo Orquesta
activo aunque no exista lease vigente.

## Cierre aplicado

- El servidor ya tiene `serverCodeContextToolProcessGuardV0` y watchdog opt-in
  para detectar procesos `codebase-memory-mcp`, proteger PIDs con owner marker y
  parar huerfanos solo con opt-in y edad minima.
- `scripts/bootstrap_agent_tooling.sh --status` ahora publica `next_action`.
  Si detecta procesos vivos, devuelve
  `stop_orphan_codebase_memory_mcp_processes`; si detecta configuracion directa
  sin opt-in, devuelve `remove_direct_mcp_config_or_enable_explicit_opt_in`.
- La norma persistente en `~/.codex/AGENTS.md` indica parar cooperativamente los
  huerfanos o activar el watchdog configurado cuando no haya lease activo.

## Evidencia

Tests:

```bash
scripts/test_bootstrap_agent_tooling.sh
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodeContextTool(ProcessGuard|Watchdog)|TestParseServerPSCodeContextToolProcessesV0'
git diff --check
```
