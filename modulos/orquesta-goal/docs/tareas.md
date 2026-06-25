# Tareas

## GOAL-001 contrato neutral inicial

Estado: cerrado localmente en el primer corte goal-first.

Criterio:

- `GoalWorkSpecV0` existe con validacion estructural.
- El modulo no importa adaptadores concretos.
- Tests focales del modulo pasan.

## GOAL-002 composicion residente

Estado: cerrado localmente para wiring opt-in; smoke real pendiente.

`cmd/orquesta-server` puede inyectar launcher/observer con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`; `orquesta-app-director-service`
lanza `GoalWorkSpecV0` desde `/nueva-app`/`arrancar_director` cuando existe
`AppGoalLauncher`, persiste estado y expone observacion por
`/api/v0/apps/director/goal/observe`.

Pendiente real: smoke opt-in con daemon Codex y repo temporal hasta observacion
terminal, closure aceptada/bloqueada y cola sincronizada.

Intento 2026-06-25: bloqueado antes de arrancar Orquesta porque
`codex app-server daemon start` no encuentra la instalacion standalone en
`/home/alberto/.codex/packages/standalone/current/codex`. No hay evidencia de
fallo del contrato `orquesta-goal`; falta resolver esa precondicion externa y
repetir `scripts/smoke_goal_first_app_server_real.sh`.

## GOAL-003 migracion del loop historico

Estado: avance local 2026-06-25.

Marcar que casos siguen usando `app-director-service` y cuales ya usan goal,
sin borrar el flujo historico hasta tener smokes equivalentes.

Avance: `StartAppDirectorResultV0`, el tool/bridge
`orquesta.apps.arrancar_director.v0` y la web `/nueva-app` exponen
`director_execution_mode`. El valor `goal_first` identifica el camino
`GoalWorkSpecV0` sin loop legacy; `legacy_director_loop` identifica la ruta
historica de Director/agentes. `ObserveAppDirectorGoalV0` y
`orquesta.apps.observe_director_goal.v0` exponen tambien `goal_first` para la
observacion/cierre. Queda pendiente la matriz completa de migracion y los
smokes reales equivalentes antes de declarar legacy las rutas historicas.
