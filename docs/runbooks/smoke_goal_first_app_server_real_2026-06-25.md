# Smoke goal-first app-server real

Fecha: 2026-06-25.

## Objetivo

Validar la ruta real no-OPES de `/nueva-app` con Codex Goal persistente:

1. Orquesta levanta un servidor temporal.
2. `/api/v0/apps/director` compila `GoalWorkSpecV0` y lanza Codex por
   `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`.
3. Codex trabaja en un proyecto temporal, no en el repo Orquesta.
4. `/api/v0/apps/director/goal/observe` lee `thread/goal/get` y `thread/read`.
5. El marcador `ORQUESTA_GOAL_RESULT_V0` o el archivo durable
   `orquesta_goal_result_v0.json` aporta artefactos/evidencias.
6. Orquesta valida cierre y deja el run `cerrada`.

## Comando

Preflight sin ejecutar Codex ni crear app:

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
./scripts/smoke_goal_first_app_server_real.sh
```

Debe devolver `smoke_goal_first_app_server_preflight=ok` si `app_server_tmux`
responde a `thread/loaded/list`. No necesita daemon externo. `app_server_proxy`
queda como compatibilidad diagnostica para investigaciones de socket ya
existente, no como smoke vigente ni ruta normal.

Smoke real, con ejecucion de Codex:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_app_server_real.sh
```

Variables utiles:

- `ORQUESTA_CODEX_COMMAND`: ruta de `codex`; por defecto resuelve `codex`.
- `ORQUESTA_CODEX_GOAL_BACKEND`: `app_server_tmux` por defecto y ruta vigente.
- `ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1`: comprueba el backend app-server
  sin arrancar Orquesta ni ejecutar una generacion.
- `ORQUESTA_CODEX_MODEL`: por defecto `gpt-5.5`.
- `ORQUESTA_CODEX_REASONING_EFFORT`: por defecto `medium`.
- `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`: por defecto `90000`.
- `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS`: preflight rapido del backend; por
  defecto `3000`.
- `ORQUESTA_CODEX_SANDBOX`: por defecto `workspace-write`. En contenedores
  remotos aislados donde Codex app-server no reconozca como escribible el
  `ORQUESTA_CODEX_PROJECT_WORKDIR` temporal bajo `/workspace/runtime`, se puede
  usar `danger-full-access` solo para este smoke opt-in, con
  `ORQUESTA_CODEX_APPROVAL_POLICY=never`, OPES vacio y artefactos bajo
  `/workspace/runtime`.
- `ORQUESTA_KEEP_SMOKE_DIR=1`: conserva el temporal para revisar salida.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR`: el script la fija a
  un directorio temporal separado para demostrar que `/nueva-app` usa
  `ORQUESTA_CODEX_PROJECT_WORKDIR` y no el workdir de automejora.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0` y
  `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`: el script los fija para no
  lanzar automejora/residente durante este smoke de `/nueva-app`.
- `ORQUESTA_GOAL_FIRST_SMOKE_POLLS` y
  `ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS`: ventana de observacion.

## Guardas

- Requiere doble confirmacion porque ejecuta Codex real y puede consumir cuota.
- Falla si `ORQUESTA_OPES_BASE_URL` u `OPES_BASE_URL` estan configuradas.
- Usa `ORQUESTA_CODEX_PROJECT_WORKDIR` temporal con contexto minimo.
- Usa un workdir temporal distinto para automejora idle; si una app aparece en
  ese directorio, la separacion de consumidores Goal ha regresado.
- No debe arrancar `request-ref-autoprogramming-*` ni agentes de automejora
  idle. Si aparecen, el smoke esta contaminado y debe cortarse.
- No detiene el daemon Codex local al terminar; puede estar compartido por el
  operador.
- Si falta el socket app-server, el daemon o la instalacion esperada por el CLI
  de Codex, Orquesta no cae al loop legacy: devuelve reason codes como
  `codex_app_server_control_socket_missing`,
  `codex_app_server_daemon_cli_missing` o
  `codex_app_server_standalone_missing`.
- El preflight se evalua al construir la composicion. Si el daemon o socket se
  levanta despues de arrancar Orquesta, reinicia el servidor para reconstruir el
  backend goal real.

## Estado local 2026-06-26

En esta maquina `command -v codex` resuelve
`/home/alberto/.nvm/versions/node/v20.19.2/bin/codex` y el CLI expone
`codex app-server`. Comprobacion vigente: `app_server_tmux` arranca
`codex app-server --listen unix://<socket>` en una sesion tmux y valida
`thread/loaded/list` mediante `codex app-server proxy --sock <socket>`.
El socket manual creado con
`codex app-server --listen unix://$HOME/.codex/app-server-control/app-server-control.sock`
no respondio a `codex app-server proxy` en esta instalacion, aunque
`daemon version` lo liste como `running`; por eso el smoke real local usa
`app_server_tmux` mientras el backend proxy queda como ruta opt-in a validar
cuando exista un daemon/socket gestionado compatible.

Ejecucion real 2026-06-26 con `app_server_tmux`:
`smoke_goal_first_app_server_real=ok`, `goal_status=complete`,
`run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`,
`artifact_refs=2`, `evidence_refs=9`, `smoke_root=/tmp/orquesta-goal-first-app-server.2dZTDh`.
El goal genero una app temporal, corrigio una prueba HTTP que no podia abrir
socket local por `listen EPERM`, ejecuto `npm run verify` con resultado passed
y escribio `docs/orquesta_goal_result_v0.json` con el `test_ref` literal del
`GoalWorkSpecV0`.

## Estado contenedor remoto 2026-06-29

En el contenedor aislado `/workspace/project`, sin Docker ni `docker.sock`, el
smoke usa `go` desde `/usr/local/go/bin/go` y cache temporal bajo
`/workspace/runtime`. El primer intento con `workspace-write` lanzo
`app_server_tmux` correctamente, pero Codex app-server bloqueo escrituras dentro
del proyecto temporal aunque el usuario del contenedor podia escribir por
permisos POSIX; se conservo como evidencia causal en
`/workspace/runtime/smokes/orquesta-goal-first-app-server.ezHMFR`.

Repeticion real 2026-06-29 con `app_server_tmux`,
`ORQUESTA_CODEX_SANDBOX=danger-full-access`,
`ORQUESTA_CODEX_APPROVAL_POLICY=never` y runtime acotado a `/workspace/runtime`:
`smoke_goal_first_app_server_real=ok`, `goal_status=complete`,
`run_status=cerrada`, `closure_status=accepted`, `closure_accepted=true`,
`artifact_refs=2`, `evidence_refs=9`,
`smoke_root=/workspace/runtime/smokes/orquesta-goal-first-app-server.CD5sKB`.
Refs principales: `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-617c6ba0738874be15af06aee294d3e9`,
`goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-617c6ba0738874be15af06aee294d3e9`,
`external_goal_ref=019f131c-b884-7410-8f36-2344d0c37da3`.

## Exito

La salida debe incluir:

```text
smoke_goal_first_app_server_real=ok
artifact_refs=<n>
evidence_refs=<n>
```

El `start_response.json` debe incluir `run_ref`, `goal_ref`,
`external_goal_ref`, `goal_status` y `director_execution_mode=goal_first`.

El ultimo `observe_response.json` debe tener:

- `goal_status=complete`;
- `run_status=cerrada`;
- `closure_status=accepted`;
- `closure_accepted=true`;
- al menos los artefactos requeridos por el `GoalWorkSpecV0`;
- el cierre solo puede llegar a `accepted` si el marcador
  `ORQUESTA_GOAL_RESULT_V0` o el archivo durable
  `orquesta_goal_result_v0.json` aporto resultados `passed` para los tests
  requeridos por el `GoalWorkSpecV0`;
- `evidence-ref-app-director-goal-first-v0` entre las evidencias observadas o
  acumuladas.

## Limites

Este smoke confirma el puente real Codex Goal para una app temporal pequena. No
cierra OPES, no prueba derivados OPES y no elimina el loop historico para
composiciones sin goal persistente.
