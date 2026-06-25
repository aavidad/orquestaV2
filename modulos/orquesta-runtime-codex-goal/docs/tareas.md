# Tareas

## CODEX-GOAL-001 packet y launcher por puerto

Estado: cerrado localmente en el primer corte.

## CODEX-GOAL-001B observer por puerto

Estado: cerrado localmente el 2026-06-25.

El modulo convierte `GoalObservationRequestV0` en
`CodexGoalObservationRequestV0`, llama a `CodexGoalObserverPortV0` y devuelve
`GoalWorkResultV0` validado sin aceptar cierre.

## CODEX-GOAL-002 composition root

Estado: cerrado localmente para wiring opt-in; smoke real pendiente.

`cmd/orquesta-server` puede inyectar starter y observer reales con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`. Ese backend habla con
`codex app-server proxy` contra un daemon local de Codex ya disponible, crea
thread persistente, fija `thread/goal/set`, arranca `turn/start` y observa con
`thread/goal/get`.

Sigue siendo opt-in de composicion: el modulo no conoce comando, shell, daemon,
modelo ni rutas. Sin la variable de backend no se expone launcher/observer.

## CODEX-GOAL-003 smoke opt-in

Estado: pendiente.

Ejecutar un goal temporal sobre repo de prueba y validar que Orquesta recibe
`complete`/`blocked` con evidencias.
