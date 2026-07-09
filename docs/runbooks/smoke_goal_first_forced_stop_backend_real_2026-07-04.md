# Smoke BUG-165 forced stop con backend vivo

Fecha: 2026-07-04.

Objetivo: validar en Codex real `app_server_tmux` la ruta que bloqueo Sueldos:
goal-first con alto consumo, backend vivo, `runs/control stop forced=true`,
estado terminal unico y `observe_goal` posterior no publicando `running`.

## Comando

Preflight sin ejecutar Codex:

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
./scripts/smoke_goal_first_forced_stop_backend_real.sh
```

Smoke real opt-in:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 \
ORQUESTA_KEEP_SMOKE_DIR=1 \
./scripts/smoke_goal_first_forced_stop_backend_real.sh
```

El wrapper activa alto consumo, backend `app_server_tmux`, observador residente,
timeout amplio de goal y modo interno `SMOKE_GOAL_FIRST_FORCED_STOP_MODE=1`.
No anade variable `ORQUESTA_*` nueva para conservar el ratchet MEJ-106.

## Resultado final

Smoke cerrado:

- `smoke_goal_first_forced_stop_backend_real=ok`
- `run_control_estado=ok`
- `run_control_status=stopped`
- `run_control_final_status=stopped`
- `run_control_goal_status_after=blocked`
- `observe_after_forced_stop_goal_status=blocked`
- `observe_after_forced_stop_closure_status=blocked`
- `observe_after_forced_stop_recommended_action=replan`
- `app_server_tmux_processes_alive=0`
- `smoke_root=/tmp/orquesta-goal-first-app-server.Sc7e7K`

Refs principales:

- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-116f51fcff09efa9aa525ee7dfd94ce7`
- `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-116f51fcff09efa9aa525ee7dfd94ce7`
- `external_goal_ref=019f2c28-d0c3-7551-9972-dcba0f3daeb2`

Evidencia durable del temporal:

- `state/run-state/control_v0.json`: `status=stopped`,
  `checkpoint_recorded=true`.
- Evidencias de control:
  `evidence-ref-codex-app-server-goal-forced-stop-requested`,
  `evidence-ref-codex-app-server-goal-forced-stop-set-blocked`,
  `evidence-ref-codex-app-server-tmux-forced-stop-requested`,
  `evidence-ref-codex-app-server-tmux-forced-stop-stopped`,
  `evidence-ref-run-control-goal-forced-stop-terminal`,
  `evidence-ref-run-control-terminal-after-goal-forced-stop`,
  `evidence-ref-mcp-run-control-checkpoint-recorded`.
- `observe_after_forced_stop_response.json`: `goal_status=blocked`,
  `closure_status=blocked`, `recommended_action=replan`.
- `state/orquesta_server_state_v0.json`: `status=stopped`,
  `shutdown_status=stopped`, `shutdown_ready=true`,
  `goal_observer_status=skipped`.
- Comprobacion posterior: sin `orquesta-server run`, sin `codex app-server` y
  sin tmux `orquesta-goal-*` vivos.

Higiene de evidencia: el temporal retenido se saneo borrando
`runtime/goal-srv/codex-home` y `bin/orquesta-server`; quedan JSON, logs,
estado y artefactos, con tamano aproximado 296 KiB.

## Fallos reproducidos antes del cierre

Intentos previos del mismo smoke real:

- Primer intento: `/api/v0/runs/control` devolvio HTTP `504`
  `run_control_timeout`; el control store quedo en `stop_requested`. El harness
  fallo ademas por `pane_pid: unbound variable`.
- Segundo intento: `/api/v0/runs/control` volvio a devolver HTTP `504`; el
  cleanup recibio `409 backend_still_running`.
- Tercer intento: el comportamiento funcional ya paso (`HTTP 200`, `stopped`,
  `blocked`, `replan`), pero el harness fallo despues por
  `socket_path: unbound variable`.

Los temporales de esos intentos se sanearon borrando `codex-home` y el binario
temporal antes de conservar solo evidencia compacta.

## Cambios cubiertos

- `scripts/smoke_goal_first_forced_stop_backend_real.sh`: wrapper opt-in del
  smoke real de forced stop con backend vivo.
- `scripts/smoke_goal_first_app_server_real.sh`: modo interno forzado, llamada
  a `/api/v0/runs/control`, validacion terminal y observe posterior.
- `modulos/orquesta-app-codex-stack/goal_first_run_control_v0.go`: fast-path
  para completar forced stop/cancel si el observer ya habia dejado el goal en
  estado terminal/rework (`blocked`, `invalid` o cierre terminal).
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_stop_v0.go`:
  puerto opcional `ShutdownForcedStopV0`.
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_tmux_v0.go`:
  forced shutdown tmux con timeout corto y cleanup con contexto fresco cuando
  el wait del pane agota plazo.

## Verificacion

Pruebas ejecutadas en este cierre:

```bash
find scripts -name '*.sh' -print0 | xargs -0 -n1 bash -n
go test -count=1 ./ -run TestEnvVarsBudget
go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|HighConsumption|ForcedStop)'
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestGoalFirstRunControl|TestCodexStackRunControl'
go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPRunControl'
go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0StopForced|TestCodexAppServerTmuxBackendV0EnsureShutdownCleanup'
go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./cmd/orquesta-server
go test -count=1 ./...
git diff --check
```

## Estado

La ruta concreta Sueldos de `BUG-ORQ-20260704-165` queda cerrada:
alto consumo, backend vivo, forced stop, estado terminal unico, observe
posterior bloqueado/replanificable y cleanup sin procesos.

No declara cerrado todo `BUG-165`: siguen como residuales separados los casos
amplios de `status/observe` lento y la coordinacion automatica completa de
shutdown/backend/checkpoint/stop/cancel/wait.

## Revalidacion Codex 2026-07-09 con status antes/despues

Smoke real ejecutado desde repo local tras ampliar el harness con snapshots de
`/api/v0/autoprogramming/status`.

Comando:

```bash
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 \
ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 \
ORQUESTA_KEEP_SMOKE_DIR=1 \
ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex \
./scripts/smoke_goal_first_forced_stop_backend_real.sh
```

Resultado:

- `smoke_goal_first_forced_stop_backend_real=ok`
- `autoprogramming_status_before_forced_stop_visible=true`
- `run_control_status=stopped`
- `run_control_final_status=stopped`
- `run_control_goal_status_after=blocked`
- `observe_after_forced_stop_goal_status=blocked`
- `observe_after_forced_stop_closure_status=blocked`
- `observe_after_forced_stop_recommended_action=replan`
- `autoprogramming_status_after_forced_stop_visible=true`
- `autoprogramming_status_after_forced_stop_not_running=true`
- `app_server_tmux_processes_alive=0`

Refs:

- `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-90afbda75e7d98ff0be34efe74150090`
- `goal_ref=goal-ref-app-director-run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-90afbda75e7d98ff0be34efe74150090`
- `external_goal_ref=019f468d-b68f-7202-ad7a-0816a63663ae`

Evidencia conservada y saneada:

- `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.5Y0PiP`
- Se elimino `runtime/goal-srv/codex-home` y el binario temporal; quedan JSON,
  logs, state y artefactos compactos (~512 KiB).
- Archivos clave: `status_before_control_response.json`,
  `run_control_response.json`, `observe_after_forced_stop_response.json`,
  `status_after_control_response.json`, `state/orquesta_server_state_v0.json`.

Higiene posterior:

- Sin `orquesta-server run` vivo del smoke.
- Sin `codex app-server` vivo para el socket del smoke.
- Sin tmux `orquesta-goal-*` vivo del smoke.
