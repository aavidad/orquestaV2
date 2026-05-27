# Pruebas: orquesta-orchestration-core

## ORCH-CORE-STATS-001

Comando:

```bash
go test -count=1 ./modulos/orquesta-orchestration-core
```

Cobertura:

- `BuildDirectorRunStatsWithTelemetryPortsV0` no consulta uso si
  `include_agent_usage=false`;
- sin `AgentUsageStatsProviderPortV0`, conserva stats basicas y emite
  `agent_usage_source_not_configured`;
- con observaciones saneadas, aplica uso por agente y `UsageSummary`;
- ignora observaciones de agentes fuera del run;
- el core no importa runtime Codex, web, MCP, DB, HOME ni proveedor.

## T207 Rotacion De Sesiones

```text
Caso: directiva de rotacion opt-in no corta proceso vivo
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-orchestration-core -run TestBuildSessionRotationDirectiveV0
Evidencia esperada: sin handoff completo pide completar handoff; con handoff aceptado permite relevo opt-in, conserva metricas y mantiene StopCurrentAllowed=false.
Ultima ejecucion: 2026-05-27, OK dentro de bateria transversal T207.
Riesgos: no prueba proveedor real ni relanzamiento productivo.
```
