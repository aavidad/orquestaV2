# Incidencia 208AE: goal-first app-server ignora el routing canonico de modelo

Fecha: 2026-07-11
ID: BUG-ORQ-20260711-208AE
Estado: abierto, correccion en curso
Area: composicion Codex goal-first / economia de tokens / model routing

## Resumen

El smoke local aislado de limpieza lanzo una tarea de codigo normal mediante
`app_server_tmux` con `gpt-5.6-sol` y esfuerzo `high`, aunque la politica
canonica vigente asigna codigo normal a Terra con esfuerzo `medium`.

Tras unos dos minutos no habia diff. La base de estado aislada declaraba
`threads.tokens_used=501059`; el ledger de goals declaraba
`thread_goals.tokens_used=42819`. Son contadores de capas distintas y no deben
sumarse. El run se detuvo para evitar mas consumo.

## Evidencia retenida

Raiz aislada: `/tmp/orquesta-cleanup-goal-20260711`.

- Run: `autoprog-cleanup-deadcode-appserver-20260711`.
- Goal externo: `019f4ef9-a9c6-78c1-beed-48496d9bab29`.
- `codex-runtime-live-2/goal-srv/codex-home/config.toml` contiene
  `model = "gpt-5.6-sol"` y `model_reasoning_effort = "high"`.
- La fila del thread en `state_5.sqlite` confirma el mismo modelo/esfuerzo.
- La fila de `thread_goals` en `goals_1.sqlite` conserva el consumo propio del
  goal.
- El repositorio aislado permanecio sin cambios antes de la parada.

## Causa arquitectonica observada

La ruta de agentes Codex resuelve `ModelRouting` en `spec_resolver_v0.go`, pero
el backend goal-first residente se construye desde `runtimeConfig.Model` y
`runtimeConfig.ReasoningEffort`. `serverCodexGoalCostRoutingStarterV0` solo
rebaja el esfuerzo para documentos; no aplica el alias de modelo canonico a
los goals de codigo. Por tanto, un modelo global heredado puede contaminar la
ruta goal-first aunque las pruebas del resolver ordinario esten verdes.

## Criterio de cierre

- El backend goal-first obtiene modelo y esfuerzo de la politica canonica de
  composicion, no del modelo global heredado.
- Una tarea normal de codigo envia Terra/medium y una trivial documental envia
  Luna/low; Sol solo entra por una decision critica explicita.
- Pruebas focales inspeccionan los parametros reales enviados a `turn/start`.
- El mismo smoke completa la limpieza con el modelo esperado y atestacion
  independiente, sin consumo anomalo ni diff fuera del write-set.

