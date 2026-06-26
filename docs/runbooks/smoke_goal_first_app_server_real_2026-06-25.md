# Smoke goal-first app-server real

Fecha: 2026-06-25.

## Objetivo

Validar la ruta real no-OPES de `/nueva-app` con Codex Goal persistente:

1. Orquesta levanta un servidor temporal.
2. `/api/v0/apps/director` compila `GoalWorkSpecV0` y lanza Codex por
   `ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`.
3. Codex trabaja en un proyecto temporal, no en el repo Orquesta.
4. `/api/v0/apps/director/goal/observe` lee `thread/goal/get` y `thread/read`.
5. El marcador `ORQUESTA_GOAL_RESULT_V0` aporta artefactos/evidencias.
6. Orquesta valida cierre y deja el run `cerrada`.

## Comando

Preflight sin ejecutar Codex ni crear app:

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
./scripts/smoke_goal_first_app_server_real.sh
```

Debe devolver `smoke_goal_first_app_server_preflight=ok` si el daemon
app-server ya esta accesible. Si el CLI existe pero no hay daemon/socket,
devuelve `smoke_goal_first_app_server_preflight=blocked` con reason code
diagnostico, por ejemplo `codex_app_server_control_socket_missing`.

Smoke real, con ejecucion de Codex:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_app_server_real.sh
```

Variables utiles:

- `ORQUESTA_CODEX_COMMAND`: ruta de `codex`; por defecto resuelve `codex`.
- `ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1`: comprueba CLI/daemon app-server
  sin arrancar Orquesta ni ejecutar una generacion.
- `ORQUESTA_CODEX_MODEL`: por defecto `gpt-5.5`.
- `ORQUESTA_CODEX_REASONING_EFFORT`: por defecto `medium`.
- `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`: por defecto `90000`.
- `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS`: preflight rapido del backend; por
  defecto `3000`.
- `ORQUESTA_KEEP_SMOKE_DIR=1`: conserva el temporal para revisar salida.
- `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR`: el script la fija a
  un directorio temporal separado para demostrar que `/nueva-app` usa
  `ORQUESTA_CODEX_PROJECT_WORKDIR` y no el workdir de automejora.
- `ORQUESTA_GOAL_FIRST_SMOKE_POLLS` y
  `ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS`: ventana de observacion.

## Guardas

- Requiere doble confirmacion porque ejecuta Codex real y puede consumir cuota.
- Falla si `ORQUESTA_OPES_BASE_URL` u `OPES_BASE_URL` estan configuradas.
- Usa `ORQUESTA_CODEX_PROJECT_WORKDIR` temporal con contexto minimo.
- Usa un workdir temporal distinto para automejora idle; si una app aparece en
  ese directorio, la separacion de consumidores Goal ha regresado.
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
`codex app-server`. El preflight no-costoso detecta que aun no hay socket de
daemon en `/home/alberto/.codex/app-server-control/app-server-control.sock`;
por tanto el siguiente paso para evidencia real es arrancar el daemon con la
doble confirmacion del smoke, no relanzar el loop legacy.

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
  `ORQUESTA_GOAL_RESULT_V0` aporto resultados `passed` para los tests
  requeridos por el `GoalWorkSpecV0`;
- `evidence-ref-app-director-goal-first-v0` entre las evidencias observadas o
  acumuladas.

## Limites

Este smoke confirma el puente real Codex Goal para una app temporal pequena. No
cierra OPES, no prueba derivados OPES y no elimina el loop historico para
composiciones sin goal persistente.
