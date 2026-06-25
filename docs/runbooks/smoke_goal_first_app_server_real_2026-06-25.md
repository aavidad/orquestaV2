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

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_app_server_real.sh
```

Variables utiles:

- `ORQUESTA_CODEX_COMMAND`: ruta de `codex`; por defecto resuelve `codex`.
- `ORQUESTA_CODEX_MODEL`: por defecto `gpt-5.5`.
- `ORQUESTA_CODEX_REASONING_EFFORT`: por defecto `medium`.
- `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`: por defecto `90000`.
- `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS`: preflight rapido del backend; por
  defecto `3000`.
- `ORQUESTA_KEEP_SMOKE_DIR=1`: conserva el temporal para revisar salida.
- `ORQUESTA_GOAL_FIRST_SMOKE_POLLS` y
  `ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS`: ventana de observacion.

## Guardas

- Requiere doble confirmacion porque ejecuta Codex real y puede consumir cuota.
- Falla si `ORQUESTA_OPES_BASE_URL` u `OPES_BASE_URL` estan configuradas.
- Usa `ORQUESTA_CODEX_PROJECT_WORKDIR` temporal con contexto minimo.
- No detiene el daemon Codex local al terminar; puede estar compartido por el
  operador.
- Si falta el socket app-server o la instalacion standalone de Codex, Orquesta
  no cae al loop legacy: devuelve reason codes como
  `codex_app_server_control_socket_missing` o
  `codex_app_server_standalone_missing`.
- El preflight se evalua al construir la composicion. Si el daemon o socket se
  levanta despues de arrancar Orquesta, reinicia el servidor para reconstruir el
  backend goal real.

## Exito

La salida debe incluir:

```text
smoke_goal_first_app_server_real=ok
artifact_refs=<n>
evidence_refs=<n>
```

El ultimo `observe_response.json` debe tener:

- `goal_status=complete`;
- `run_status=cerrada`;
- `closure_status=accepted`;
- `closure_accepted=true`;
- al menos los artefactos requeridos por el `GoalWorkSpecV0`;
- `evidence-ref-app-director-goal-first-v0` entre las evidencias observadas o
  acumuladas.

## Limites

Este smoke confirma el puente real Codex Goal para una app temporal pequena. No
cierra OPES, no prueba derivados OPES y no elimina el loop historico para
composiciones sin goal persistente.
