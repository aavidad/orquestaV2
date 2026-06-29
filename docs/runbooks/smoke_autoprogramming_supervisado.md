# Smoke autoprogramming supervisado

Objetivo: validar el camino opt-in minimo para autoprogramacion desatendida sin
tocar OPES. Por defecto ejecuta `codex-fake`; opcionalmente puede lanzar Codex
real con confirmacion doble y homes explicitos:

```text
request valida
  -> BuildAutoprogrammingProgrammableWorkV0
  -> PrepareAutoprogrammingRunV0 / CodexStackAutoprogrammingExecutorV0 con stack fake
  -> ruta REST de validacion si existe
  -> POST /api/v0/autoprogramming/prepare-run
  -> cola global del stack Codex
  -> Director residente del servidor con Codex fake o Codex real opt-in
  -> POST /api/v0/autoprogramming/prepare-run con el mismo payload para replay
  -> external-work/run + supervisor solo con fallback legacy explicito
  -> pruebas focales de stack fake y cierre offline segun disponibilidad
```

Este smoke no modifica el repo. Crea harnesses, servidor, state y runtime bajo
`/tmp` o bajo `ORQUESTA_SMOKE_ROOT`. Si se usa `ORQUESTA_SMOKE_ROOT` fuera de
`/tmp`, declara el prefijo permitido en `ORQUESTA_SMOKE_ALLOWED_ROOT_PREFIXES`.

## Comando

Desde la raiz del repo:

```bash
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 \
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1 \
  ./scripts/smoke_autoprogramming_supervised.sh
```

Modo Codex real legacy/residente:

```bash
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 \
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1 \
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_REAL_CODEX=1 \
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_REAL_CODEX_CONFIRMED=1 \
ORQUESTA_CODEX_COMMAND=/ruta/a/codex \
ORQUESTA_CODEX_HOME=/workspace/runtime/codex-home-real \
ORQUESTA_CODEX_CODE_HOME=/workspace/runtime/codex-home-real/.codex \
  ./scripts/smoke_autoprogramming_supervised.sh
```

Variables utiles:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-autoprogramming-supervised
ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS=15
ORQUESTA_SERVER_TICK_INTERVAL_MS=250
ORQUESTA_SMOKE_RESIDENT_POLLS=40
ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1
ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1
ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1
SMOKE_ID=manual-001
```

`ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1` es obligatorio para este
smoke porque prueba deliberadamente `prepare-run` legacy. El payload del smoke
incluye `director_execution_mode=legacy_director_loop`. Sin ese opt-in,
`prepare-run` debe responder con
`autoprogramming_legacy_director_loop_opt_in_required` y no crear `run_ref`,
`WorkflowTaskV0` ni cola legacy; con opt-in pero sin esa marca debe responder
`autoprogramming_legacy_director_mode_required`.

`ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1` es compatibilidad
historica y, desde el corte goal-first, exige tambien
`ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1` para que el servidor permita
ese fallback; el payload de external-work transporta ademas
`director_execution_mode=legacy_director_loop`. Sin esas variables, el smoke no
llama a `/api/v0/external-work/run` ni a `/api/v0/runs/supervise` y no ejecuta
el test focal legacy asociado.

## Que comprueba

- `go test -count=1 ./modulos/orquesta-autoprogramming`.
- Un programa temporal fuera del repo importa `orquesta-autoprogramming` y
  ejecuta `BuildAutoprogrammingProgrammableWorkV0` sobre una request valida,
  verificando que produce grupos, perfiles y `WorkflowTaskV0`.
- Si los tests existen, ejecuta la entrada real no-HTTP del stack:
  `PrepareAutoprogrammingRunV0` y `CodexStackAutoprogrammingExecutorV0`. Esa
  cobertura valida, para una request legacy, que se transforma en run continuable,
  `WorkflowTaskV0` persistidas, `WaitAgentRefs` y `ContinueAppDirectorRequestV0`
  usando runtime fake.
- Compila y arranca `cmd/orquesta-server` con `state_dir`, `runtime_dir`,
  `project_dir`, `CODEX_HOME` y comando Codex fake temporales.
- Configura `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux` para que el servidor
  temporal quede sano en modo goal-first aunque este smoke pruebe
  `prepare-run` legacy explicito. No usa `stdio` como backend normal.
- Llama a `POST /api/v0/autoprogramming/validate-request` si la ruta esta
  expuesta y exige `accepted=true`.
- Llama a `POST /api/v0/autoprogramming/prepare-run` en modo legacy explicito
  con `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1`,
  exige `run_ref`, `wait_agent_refs` y `continue`, y despues espera a que el
  supervisor residente del servidor tome la cola global y arranque al menos un
  agente Codex fake para ese `run_ref`. Si la request se marca `goal_ready`, el
  smoke debe esperar `goal_specs[]` sin `run_ref` y no invocar supervisor
  legacy.
- Vuelve a llamar a `POST /api/v0/autoprogramming/prepare-run` con el mismo
  payload despues del arranque residente; exige respuesta aceptada,
  `run_ref` estable, `wait_agent_refs` no vacio y `continue`, para cubrir
  idempotencia cuando el patch del stack este disponible.
- Por defecto no invoca `POST /api/v0/external-work/run` ni
  `POST /api/v0/runs/supervise`; esas rutas pertenecen al fallback legacy
  `ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1`.
- Con `ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1` y
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1`, intenta
  `POST /api/v0/external-work/run` con
  `director_execution_mode=legacy_director_loop` y un trabajo
  `autoprogramming_programmable_work`; si devuelve `run_ref`, llama a
  `POST /api/v0/runs/supervise` con limites bajos y modo legacy explicito.
- Ejecuta tests focales existentes cuando estan presentes:
  `OperationalClosureSourceV0` y cierre offline de `app-director-service`. Los
  focales de `external-work/run` con stack fake y supervisor con stack fake se
  ejecutan solo con `ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1`.

## Criterio de exito

La salida debe incluir:

- `programmable_work_ok`;
- `validate_request_ok=true` si la ruta REST esta disponible;
- `prepare_run_ok=true`;
- `resident_supervisor_agents_started=1` o superior;
- `prepare_run_replay_ok=true`;
- `codex_real_executed=false`;
- `opes_touched=false`.

En modo real debe incluir `codex_real_executed=true` y `codex_command=...`.
Revalidacion local 2026-06-29: el modo fake/residente paso en
`/workspace/runtime/smokes`; el modo real quedo bloqueado antes de lanzar
agentes porque faltaban `ORQUESTA_CODEX_HOME` y `ORQUESTA_CODEX_CODE_HOME`
explicitos. Accion del operador: proporcionar homes Codex aislados y
autenticados bajo `/workspace` o `/srv/orquesta-self` y repetir el comando de
modo real.

Los mensajes `skip:` son aceptables cuando una ruta o prueba focal todavia no
existe en la composicion actual. No deben ocultar fallos de contrato puro,
payload invalido, servidor que no arranca o tests focales existentes que fallen.

## Entrada real actual

La entrada real esta disponible por composicion Go y por ruta publica opt-in del
stack Codex:

- `modulos/orquesta-app-codex-stack/PrepareAutoprogrammingRunV0`;
- `modulos/orquesta-app-codex-stack/CodexStackAutoprogrammingExecutorV0`.
- `POST /api/v0/autoprogramming/prepare-run`, expuesta por
  `orquesta.autoprogramming.prepare_run.v0` y cableada en
  `orquesta-app-codex-stack`.

El smoke la cubre con tests focales y por servidor temporal. La supervision
posterior valida el camino desatendido legacy: `prepare-run` encola el run en la
cola global del stack Codex y el supervisor residente del servidor avanza esa
cola sin recibir `run_ref` explicito por API. Tras observar agentes arrancados
por stats, repite `prepare-run` con el mismo payload y exige que el replay
idempotente conserve el `run_ref` y siga devolviendo refs causales. El camino
Goal-first queda fuera de este smoke legacy y no debe encolar run.
`/api/v0/external-work/run` queda como fallback opcional de compatibilidad para
otros trabajos externos y solo se prueba desde este smoke con
`ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1`,
`ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1` y payload legacy explicito; no
es el launcher de autoprogramacion.

## Alcance

Este smoke prepara evidencia operacional no invasiva para autoprogramacion
desatendida. El modo real cubre el arranque residente con proveedor Codex, pero
no sustituye los smokes largos de ola/recursion, replan generico de todos los
blockers ni calidad semantica de una entrega real.
