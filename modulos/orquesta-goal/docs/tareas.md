# Tareas

## GOAL-001 contrato neutral inicial

Estado: cerrado localmente en el primer corte goal-first.

Criterio:

- `GoalWorkSpecV0` existe con validacion estructural.
- El modulo no importa adaptadores concretos.
- Tests focales del modulo pasan.

## GOAL-002 composicion residente

Estado: pendiente.

Cablear una ruta publica opt-in para lanzar goals desde Orquesta usando un
adaptador concreto. El primer adaptador previsto es Codex Goal.

## GOAL-003 migracion del loop historico

Estado: pendiente.

Marcar que casos siguen usando `app-director-service` y cuales ya usan goal,
sin borrar el flujo historico hasta tener smokes equivalentes.
