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

## ORCH-CORE-DIR-014

Comando:

```bash
go test -count=1 ./modulos/orquesta-orchestration-core -run TestRunResidentDirectorBriefingLoopV0
```

Cobertura:

- `RunResidentDirectorBriefingLoopV0` itera
  `briefing -> ExecuteDirectorBriefingActionV0 -> briefing` hasta cierre;
- una ruta offline ejecuta un step del Director, genera outbox, despacha por
  puerto y termina en `close_or_idle`;
- no autoaplica acciones marcadas como `requires_director`;
- conserva como pendiente una accion externa sin handler;
- propaga como `external_action_pending` una accion externa cuyo handler queda
  pendiente, sin convertirla falsamente en `external_action_applied`;
- respeta `MaxActions` como presupuesto operativo del loop.

Riesgos:

- no es daemon ni servidor residente;
- no prueba proveedor real, OPES ni modelos;
- skills, fuente real de votos del consejo y seleccion de modelos siguen
  pendientes de wiring por composicion.

## ORCH-CORE-COUNCIL-001

Comando:

```bash
go test -count=1 ./modulos/orquesta-orchestration-core -run 'DecisionCouncil|WorkflowTaskProfile|AcceptDecision'
```

Cobertura:

- las tareas materializadas por el consejo conservan metadata estructurada de
  rol, `assignment_ref`, `agent_ref` y `family_ref` en criterios durables;
- el builder de `AcceptDecision` sigue neutral: recibe un resultado de voto ya
  evaluado y no conoce Codex, OPES, runtime ni fuente de artefactos;
- la composicion externa puede evaluar quorum/familias sin parsear texto libre
  de agentes.

Riesgos:

- la fuente real de votos `architecture_vote.v0` vive en la composicion/adaptador
  y no queda probada por este paquete.

## T207 Rotacion De Sesiones

```text
Caso: directiva de rotacion opt-in no corta proceso vivo
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-orchestration-core -run TestBuildSessionRotationDirectiveV0
Evidencia esperada: sin handoff completo pide completar handoff; con handoff aceptado permite relevo opt-in, conserva metricas y mantiene StopCurrentAllowed=false.
Ultima ejecucion: 2026-05-27, OK dentro de bateria transversal T207.
Riesgos: no prueba proveedor real ni relanzamiento productivo.
```
