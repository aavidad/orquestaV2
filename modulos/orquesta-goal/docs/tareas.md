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

## GOAL-003 migracion del loop historico

Estado: pendiente.

Marcar que casos siguen usando `app-director-service` y cuales ya usan goal,
sin borrar el flujo historico hasta tener smokes equivalentes.
