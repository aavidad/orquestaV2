# Incidencia: artefactos y QA pass sin receipt terminal goal-first

Fecha: 2026-07-02.

## Sintoma

Un goal puede materializar artefactos dentro de su `write_set` y dejar evidencia
estructurada de QA superada, pero no escribir `orquesta_goal_result_v0.json`,
`opes_topic_rework_delivery.json` ni otro receipt terminal. En ese estado,
`observe` o `status` podian quedarse ambiguos: hay trabajo recuperable, pero no
hay cierre causal.

## Riesgo

Es un falso bloqueo y, a la vez, un falso verde potencial: si solo se mira el
filesystem hay material; si solo se mira el receipt falta cierre. El modelo
goal-first necesita proyectar explicitamente `repair_receipt` en vez de relanzar
el trabajo o cerrar sin causalidad.

## Cambio

- El scanner de materializaciones del stack revisa solo rutas del `write_set`.
- Detecta artefactos por extension y QA pass solo en JSON estructurado, con
  lectura acotada.
- Si hay artefacto + QA pass y no hay receipt terminal, publica
  `missing_terminal_receipt_after_artifacts_pass`.
- `director.stats`, `autoprogramming/status` y `observe` transportan
  `expected_terminal_receipt_refs`, evidencias opacas y accion `repair_receipt`.

## Evidencia

Tests:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test(StackGoalMaterializedRefsSourceV0|CodexStackObserveAppDirectorGoalExecutorV0TimeoutSnapshot)'
go test -count=1 ./modulos/orquesta-mcp -run 'Test(MCPDirectorStats|AutoprogrammingStatus|ObserveAppDirectorGoal|AutoprogrammingEfficiency)'
```

## Residual

El cambio no sintetiza aun el receipt terminal; solo hace el estado accionable.
El siguiente paso es una accion `repair_receipt` que materialice el receipt
faltante con evidencia causal o pida rework si la QA estructurada no es
suficiente.
