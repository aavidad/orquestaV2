# Smoke autoprogramming supervisado sin Codex real

Objetivo: validar el camino opt-in minimo para autoprogramacion desatendida sin
ejecutar Codex real ni tocar OPES:

```text
request valida
  -> BuildAutoprogrammingProgrammableWorkV0
  -> PrepareAutoprogrammingRunV0 / CodexStackAutoprogrammingExecutorV0 con stack fake
  -> ruta REST de validacion si existe
  -> POST /api/v0/autoprogramming/prepare-run
  -> cola global del stack Codex
  -> supervisor residente del servidor con Codex fake
  -> POST /api/v0/autoprogramming/prepare-run con el mismo payload para replay
  -> external-work/run + supervisor solo con fallback legacy explicito
  -> pruebas focales de stack fake y cierre offline segun disponibilidad
```

Este smoke no modifica el repo. Crea harnesses, servidor, state, runtime y HOME
Codex falsos bajo `/tmp` o bajo `ORQUESTA_SMOKE_ROOT`.

## Comando

Desde la raiz del repo:

```bash
ORQUESTA_AUTOPROGRAMMING_SUPERVISED_SMOKE_CONFIRM=1 \
  ./scripts/smoke_autoprogramming_supervised.sh
```

Variables utiles:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-autoprogramming-supervised
ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS=15
ORQUESTA_SERVER_TICK_INTERVAL_MS=250
ORQUESTA_SMOKE_RESIDENT_POLLS=40
ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1
SMOKE_ID=manual-001
```

`ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1` es compatibilidad
historica. Sin esa variable, el smoke no llama a `/api/v0/external-work/run` ni
a `/api/v0/runs/supervise` y no ejecuta el test focal legacy asociado.

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
- Llama a `POST /api/v0/autoprogramming/validate-request` si la ruta esta
  expuesta y exige `accepted=true`.
- Llama a `POST /api/v0/autoprogramming/prepare-run` en modo legacy explicito,
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
- Con `ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1`, intenta
  `POST /api/v0/external-work/run` con un trabajo
  `autoprogramming_programmable_work`; si devuelve `run_ref`, llama a
  `POST /api/v0/runs/supervise` con limites bajos.
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
`ORQUESTA_AUTOPROGRAMMING_LEGACY_EXTERNAL_FALLBACK=1`; no es el launcher de
autoprogramacion.

## Alcance

Este smoke prepara evidencia operacional no invasiva para autoprogramacion
desatendida, pero no cierra el smoke largo con agentes Codex reales. Tampoco
prueba recursion completa, replan generico de todos los blockers ni calidad
semantica de una entrega real.
