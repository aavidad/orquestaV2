# Tareas

## CODEX-GOAL-001 packet y launcher por puerto

Estado: cerrado localmente en el primer corte.

## CODEX-GOAL-001B observer por puerto

Estado: cerrado localmente el 2026-06-25.

El modulo convierte `GoalObservationRequestV0` en
`CodexGoalObservationRequestV0`, llama a `CodexGoalObserverPortV0` y devuelve
`GoalWorkResultV0` validado sin aceptar cierre.

## CODEX-GOAL-002 composition root

Estado: cerrado localmente para wiring opt-in; smoke real app-server por
`app_server_tmux` cerrado el 2026-06-26 y repetido en contenedor aislado el
2026-06-29.

`cmd/orquesta-server` puede inyectar starter y observer reales con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`. Ese backend habla con
`codex app-server`, crea thread persistente, fija `thread/goal/set`, arranca
`turn/start` y observa con `thread/goal/get`. `app_server_proxy` queda como
nombre historico/diagnostico y no es backend operativo aceptado.

Sigue siendo opt-in de composicion: el modulo no conoce comando, shell, daemon,
modelo ni rutas. Sin la variable de backend no se expone launcher/observer.

## CODEX-GOAL-003 smoke opt-in

Estado: cerrado localmente con `app_server_tmux`; `app_server_proxy` queda
fuera del camino goal-first normal y no es backend operativo aceptado.

Ejecutar un goal temporal sobre repo de prueba y validar que Orquesta recibe
`complete`/`blocked` con evidencias.

Intento 2026-06-25: ejecutado
`ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1
ORQUESTA_KEEP_SMOKE_DIR=1 ./scripts/smoke_goal_first_app_server_real.sh`.
El smoke no arranco Orquesta: fallo antes en `codex app-server daemon start`
porque falta la instalacion standalone esperada por Codex en
`/home/alberto/.codex/packages/standalone/current/codex`. Accion externa:
instalar Codex standalone con el instalador oficial indicado por la CLI y
repetir el smoke.
Actualizacion 2026-06-26: el socket manual listado como `running` no respondio
por `codex app-server proxy`; el backend local viable es `app_server_tmux`.
Ejecucion real 2026-06-26: `scripts/smoke_goal_first_app_server_real.sh` paso
con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`: Codex ejecuto el goal,
escribio resultado durable, Orquesta observo `goal_status=complete` y cerro la
run como `cerrada` con `closure_status=accepted`, `closure_accepted=true`,
`artifact_refs=2` y `evidence_refs=9`.
Repeticion 2026-06-29 en contenedor remoto aislado, sin Docker ni socket Docker:
`app_server_tmux` cerro con `goal_status=complete`, `run_status=cerrada`,
`closure_status=accepted`, `closure_accepted=true`, `artifact_refs=2` y
`evidence_refs=9`; evidencia conservada en
`/workspace/runtime/smokes/orquesta-goal-first-app-server.CD5sKB`. En ese
entorno `workspace-write` no permitio escribir en el proyecto temporal bajo
`/workspace/runtime`; la repeticion uso
`ORQUESTA_CODEX_SANDBOX=danger-full-access` como opt-in de smoke aislado con
`approval-policy=never`.

Cierre H0a 2026-07-12 en el runner Docker aislado oficial: el smoke real
`app_server_tmux` termino con `exit_code=0`,
`smoke_goal_first_app_server_real=ok`, `goal_status=complete`,
`run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`,
`artifact_refs=10` y `evidence_refs=24`. El shutdown verifico
`app_server_tmux_processes_alive=0`. Evidencia conservada en
`/workspace/runtime/smokes/orquesta-goal-first-app-server.6C05Pu` y log del
operador en `/workspace/runtime/h0a-smoke-final8-accepted-20260712.log`. El
opt-in `danger-full-access` solo se conserva con
`ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED=1`; el contenedor mantiene
usuario no-root, rootfs de solo lectura, `no-new-privileges`, `cap_drop: ALL`,
sin Docker socket y binds limitados a `/srv/orquesta-self`.
