# Incidencia: smoke goal-first real bloqueado por usageLimited

Fecha: 2026-07-07.

## Resumen

Se ejecuto el smoke real minimo de Nueva App por goal-first con backend
`app_server_tmux`. Orquesta arranco, acepto el run y observo el goal, pero el
proveedor corto la ejecucion con `usageLimited` antes de producir cierre
aceptado.

## Comando

```bash
ORQUESTA_SMOKE_PARENT=/tmp \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_GOAL_FIRST_SMOKE_REQUEST_TIMEOUT_SECONDS=120 \
ORQUESTA_GOAL_FIRST_SMOKE_POLLS=90 \
ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=5 \
./scripts/smoke_goal_first_app_server_real.sh
```

## Evidencia

### Intento 1: modelo por defecto `gpt-5.5`

- `run_ref`:
  `run-spec-smoke-goal-first-req-smoke-goal-first-31fdafe9a94789b0edb2bd8cf48bf476`
- `goal_ref`:
  `goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-31fdafe9a94789b0edb2bd8cf48bf476`
- `external_goal_ref`: `019f3976-8ccd-7133-96bb-5538a4810f1f`
- `poll=1`: `goal_status=running`
- `poll=2`: `goal_status=blocked`, `run_status=bloqueada`,
  `closure_status=blocked`
- `summary`: `codex_app_server_goal_status_usageLimited`
- `recommended_action`: `inspect_goal_backend_limits`
- `closure_issues` incluye `codex_app_server_goal_provider_limited`
- Artefacto parcial:
  `project/generated-apps/smoke-goal-first/checkpoint_started.txt`

Directorio de evidencia saneado:

- `/tmp/orquesta-goal-first-app-server.0smTMv`
- Se elimino `runtime/goal-srv/codex-home` antes de documentar, para no dejar
  credenciales ni SQLite de proveedor en la evidencia retenida.

## Shutdown y limpieza

El intento de shutdown durante `assert_app_server_tmux_shutdown_ready` devolvio
`409 backend_still_running` tras el retry, pero el trap de cleanup termino sin
procesos residuales observables:

```bash
pgrep -af 'orquesta-server|codex app-server|orquesta-goal-0240b08d917c2f4f|g-0240b08d917c2f4f|orquesta-goal-first-app-server.0smTMv'
```

Resultado: sin procesos vivos; solo aparecio el propio `pgrep`.

### Intento 2: `ORQUESTA_CODEX_MODEL=gpt-5`

Comando igual al anterior, anadiendo:

```bash
ORQUESTA_CODEX_MODEL=gpt-5
ORQUESTA_CODEX_REASONING_EFFORT=medium
```

Resultado:

- `run_ref`:
  `run-spec-smoke-goal-first-req-smoke-goal-first-4959ed055bcf131f10d2a91796e8a36c`
- `goal_ref`:
  `goal-ref-app-director-run-spec-smoke-goal-first-req-smoke-goal-first-4959ed055bcf131f10d2a91796e8a36c`
- `external_goal_ref`: `019f3b5a-99ca-7531-9ece-7b5d4520aa8b`
- `poll=1`: `goal_status=running`
- `poll=2`: `goal_status=blocked`, `run_status=bloqueada`,
  `closure_status=blocked`
- `summary`: `codex_app_server_goal_status_blocked`
- `recommended_action`: `replan`
- Artefacto parcial:
  `project/generated-apps/smoke-goal-first/checkpoint_started.txt`
- Sin procesos residuales tras cleanup.

Directorio de evidencia saneado:

- `/tmp/orquesta-goal-first-app-server.dDenPb`
- Se elimino `runtime/goal-srv/codex-home`.

Durante la revision previa al smoke se corrigieron POST directos de shutdown
sin contrato vigente:

- `scripts/smoke_goal_first_app_server_real.sh`
- `scripts/smoke_codex_required_test_runner_state_file.sh`

La guarda `TestScriptsConShutdownDirectoPidenContratoShutdownV0` exige ahora
`cleanup_goal_backends`, `idempotency_key` y `requested_by=orquesta-director`.

## Lectura

El nucleo no queda validado al 100% por este smoke porque no hubo cierre
`accepted`: el proveedor entro en limite de uso. La ruta Orquesta si proyecto el
fallo como bloqueo accionable (`provider_limited`) y con checkpoint parcial.

Siguiente accion cuando haya cuota/modelo operativo:

```bash
ORQUESTA_SMOKE_PARENT=/tmp \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
./scripts/smoke_goal_first_app_server_real.sh
```
