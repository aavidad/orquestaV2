# Smoke automejora idle budget MEJ-104

Fecha: 2026-07-04.

Objetivo: validar en servidor residente temporal que el presupuesto diario de
automejora idle aplaza antes de lanzar trabajo cuando el cupo ya esta agotado,
y que la decision aparece en `/api/v0/server/status` y
`/api/v0/autoprogramming/status`.

## Escenario ejecutado

- State temporal sembrado con `idle_self_improvement_check` del dia actual y
  `idle_self_improvement_runs=1`.
- Configuracion temporal:
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=1`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE=1`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=1`,
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_GOAL_BUDGET=1`,
  `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`.
- Runtime aislado bajo `/tmp/orquesta-idle-budget-smoke-20260704`.
- No se lanzo ningun goal Codex; el corte ocurrio antes de planificar.

## Resultado

```text
idle_budget_smoke=ok
server_reason=budget_deferred
autoprogramming_reason=budget_deferred
diagnostics=idle_self_improvement_budget_deferred,queue_empty_or_not_visible,estado_vivo_desconocido,estado_vivo_desconocido
smoke_root=/tmp/orquesta-idle-budget-smoke-20260704
```

`server_status.json` publico:

```json
{
  "idle_self_improvement_budget": {
    "prepare": false,
    "reason": "budget_deferred",
    "requested_goals": 1,
    "max_goals_per_day": 1,
    "goals_used_today": 1
  },
  "idle_self_improvement_reason": "budget_deferred;evidence_ref=evidence-ref-idle-self-improvement-budget-v0;action=wait_for_budget_window_or_raise_declared_budget"
}
```

`autoprogramming_status.json` publico `estado=ok`,
`idle_self_improvement_budget.reason=budget_deferred` y diagnostico
`idle_self_improvement_budget_deferred`.

Verificacion de limpieza posterior: sin procesos reales `orquesta-server run`,
`codex app-server`, sesiones tmux `orquesta-goal-*` ni
`codebase-memory-mcp`.
