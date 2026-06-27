# Tareas

## GOAL-001 contrato neutral inicial

Estado: cerrado localmente en el primer corte goal-first.

Criterio:

- `GoalWorkSpecV0` existe con validacion estructural.
- El modulo no importa adaptadores concretos.
- Tests focales del modulo pasan.

## GOAL-002 composicion residente

Estado: cerrado localmente para wiring opt-in; smoke real app-server stdio
cerrado el 2026-06-26.

`cmd/orquesta-server` puede inyectar launcher/observer con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy` o `app_server_stdio`;
`orquesta-app-director-service` lanza `GoalWorkSpecV0` desde
`/nueva-app`/`arrancar_director` cuando existe `AppGoalLauncher`, persiste
estado y expone observacion por `/api/v0/apps/director/goal/observe`.

Intento 2026-06-25: bloqueado antes de arrancar Orquesta porque
`codex app-server daemon start` no encuentra la instalacion standalone en
`/home/alberto/.codex/packages/standalone/current/codex`. No hay evidencia de
fallo del contrato `orquesta-goal`; falta resolver esa precondicion externa y
repetir `scripts/smoke_goal_first_app_server_real.sh`.
Actualizacion 2026-06-26: el backend `app_server_stdio` permite usar
`codex app-server --stdio` sin daemon/socket cuando el proxy no responde.
Evidencia 2026-06-26: `scripts/smoke_goal_first_app_server_real.sh` cerro una
app temporal con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_stdio`:
`goal_status=complete`, `run_status=cerrada`, `closure_status=accepted`,
`closure_accepted=true`, `artifact_refs=2` y `evidence_refs=9`.
`app_server_proxy` queda como ruta opt-in condicionada a daemon/socket
compatible; no bloquea el cierre local `app_server_stdio`.

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

## GOAL-004 lifecycle neutral reutilizable

Estado: cerrado localmente 2026-06-27.

Criterio:

- `StartGoalWorkV0` lanza por puerto, normaliza receipt y guarda
  `GoalWorkStateV0`.
- `ObserveGoalWorkV0` carga estado, observa por puerto, persiste
  `LastResult`/evidencias y valida closure solo para resultados terminales.
- El modulo sigue sin importar Codex, OPES, HTTP, filesystem, colas,
  `core-workflow` ni `app-director-service`.
- `/nueva-app`, autoprogramacion goal-first y external-work goal-first reutilizan
  el lifecycle o el constructor neutral de estado sin cambiar sus contratos
  publicos.

Evidencia local:

```bash
go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-app-director-service
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'GoalFirst|Autoprogramming|ExternalWork'
```
