# Decisiones: orquesta-orchestration-core

```text
Fecha: 2026-05-27
Decision: El core distingue fuente de uso no configurada de cuota no observable.
Motivo: T209 necesitaba que stats/web no mostrasen `not_configured` cuando una
composicion si inyecto puerto de uso pero el runtime no pudo observar cuota real.
Impacto: `BuildDirectorRunStatsWithTelemetryPortsV0` solo llama al puerto con
`IncludeAgentUsage=true`; sin puerto emite `agent_usage_source_not_configured`.
Con puerto, el core aplica el `quota_status` saneado por el adaptador
(`unknown`, `available`, `limited` o `exhausted`) y no lee runtime, logs,
proveedor, HOME, OAuth ni rutas.
Estado: aceptada.
```

```text
Fecha: 2026-06-08
Decision: El primer loop autonomo del Director vive como controlador offline por
puertos en orchestration-core.
Motivo: ya existia el ejecutor de una accion de briefing, pero faltaba una
pieza que reentrase en el Director y demostrase autonomia minima sin levantar
daemon ni proveedor real.
Impacto: `RunResidentDirectorBriefingLoopV0` pide briefing por
`ResidentDirectorBriefingSourcePortV0`, ejecuta acciones seguras con
`ExecuteDirectorBriefingActionV0` y se detiene por idle/cierre, accion externa
pendiente, falta de progreso, necesidad de Director o presupuesto operativo.
No conoce Codex, OPES, modelos, DB, HOME ni runtime real; esas decisiones quedan
en la composicion residente.
Estado: aceptada.
```

```text
Fecha: 2026-05-27
Decision: T207 se representa como directiva de aplicacion, no como runtime.
Motivo: orchestration-core coordina puertos del nucleo, pero no posee procesos,
proveedor, HOME, modelo, sesiones reales ni politica de corte.
Impacto: `BuildSessionRotationDirectiveV0` puede pedir handoff completo o
permitir relevo opt-in, pero mantiene stop_current deshabilitado y no cambia
`WaitAgentRefs`.
Estado: aceptada.
```
